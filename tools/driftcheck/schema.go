// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
)

var (
	errSchemaFileNotLoaded = errors.New("schema file not loaded")
	errPointerNoKey        = errors.New("json pointer container has no key")
	errPointerNotFound     = errors.New("json pointer target not found")
)

const (
	// schemaTypeObject is the JSON Schema "object" type token.
	schemaTypeObject = "object"
	// jsonFalse is the literal additionalProperties:false marker.
	jsonFalse = "false"
)

// schemaNode is the subset of JSON Schema (draft-07) the drift checker needs.
// Only structural keywords are modeled; constraint keywords (minLength,
// pattern, enum, …) are intentionally ignored because the deterministic checker
// compares *shape*, not value constraints (those are enforced at runtime by the
// two-layer validator).
type schemaNode struct {
	Type                 string                 `json:"type"`
	Ref                  string                 `json:"$ref"`
	Properties           map[string]*schemaNode `json:"properties"`
	Required             []string               `json:"required"`
	Items                *schemaNode            `json:"items"`
	PatternProperties    map[string]*schemaNode `json:"patternProperties"`
	OneOf                []*schemaNode          `json:"oneOf"`
	AnyOf                []*schemaNode          `json:"anyOf"`
	AllOf                []*schemaNode          `json:"allOf"`
	Definitions          map[string]*schemaNode `json:"definitions"`
	AdditionalProperties json.RawMessage        `json:"additionalProperties"`
}

// additionalPropsSchema returns the schema for additionalProperties when it is
// an object schema (rather than a bool). nil means additionalProperties was
// absent or a boolean.
func (n *schemaNode) additionalPropsSchema() *schemaNode {
	if len(n.AdditionalProperties) == 0 {
		return nil
	}
	// A boolean additionalProperties (`true`/`false`) is not an object schema.
	trimmed := strings.TrimSpace(string(n.AdditionalProperties))
	if trimmed == "true" || trimmed == jsonFalse {
		return nil
	}
	var sub schemaNode
	if err := json.Unmarshal(n.AdditionalProperties, &sub); err != nil {
		return nil
	}

	return &sub
}

// additionalPropsClosed reports whether additionalProperties is explicitly
// false (i.e. the object is closed and extra Go fields are real drift).
func (n *schemaNode) additionalPropsClosed() bool {
	return strings.TrimSpace(string(n.AdditionalProperties)) == jsonFalse
}

// schemaSet holds every parsed schema file, keyed by its path relative to the
// schema root directory (e.g. "person.schema.json",
// "vocabularies/event-types.schema.json"). It resolves cross-file and
// in-file ($ref: "#/definitions/X") references.
type schemaSet struct {
	roots map[string]*schemaNode
}

// newSchemaSet parses the provided files. The key of each entry is the file's
// path relative to the schema root; the value is its raw JSON bytes.
func newSchemaSet(files map[string][]byte) (*schemaSet, error) {
	set := &schemaSet{roots: make(map[string]*schemaNode, len(files))}
	for rel, data := range files {
		var node schemaNode
		if err := json.Unmarshal(data, &node); err != nil {
			return nil, fmt.Errorf("parse %s: %w", rel, err)
		}
		set.roots[rel] = &node
	}

	return set, nil
}

// ref is a fully-qualified pointer to a node: the file it lives in plus the
// JSON pointer within that file ("" for the file root).
type ref struct {
	file    string // relative path, e.g. "vocabularies/event-types.schema.json"
	pointer string // e.g. "/definitions/EventTypeDefinition", or "" for root
}

// resolve follows a $ref string interpreted relative to baseFile and returns
// the target node together with the ref describing where it lives (so further
// nested $refs resolve relative to the correct file).
func (s *schemaSet) resolve(refStr, baseFile string) (*schemaNode, ref, error) {
	filePart, pointer, _ := strings.Cut(refStr, "#")

	targetFile := baseFile
	if filePart != "" {
		// Relative to the directory of the referencing file.
		targetFile = path.Join(path.Dir(baseFile), filePart)
	}

	root, ok := s.roots[targetFile]
	if !ok {
		return nil, ref{}, fmt.Errorf("$ref %q: %w: %q", refStr, errSchemaFileNotLoaded, targetFile)
	}

	node, err := walkPointer(root, pointer)
	if err != nil {
		return nil, ref{}, fmt.Errorf("$ref %q: %w", refStr, err)
	}

	return node, ref{file: targetFile, pointer: pointer}, nil
}

// deref returns the concrete node a possibly-$ref node points at, following a
// single level of indirection. If node has no $ref it is returned unchanged
// (with the current location).
func (s *schemaSet) deref(node *schemaNode, loc ref) (*schemaNode, ref, error) {
	if node == nil || node.Ref == "" {
		return node, loc, nil
	}

	return s.resolve(node.Ref, loc.file)
}

// walkPointer resolves a JSON pointer ("/definitions/Foo",
// "/properties/bar") against root. Container tokens ("definitions",
// "properties") select the corresponding map; the following token is the key
// within it.
func walkPointer(root *schemaNode, pointer string) (*schemaNode, error) {
	if pointer == "" || pointer == "/" {
		return root, nil
	}
	tokens := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	cur := root
	for i := 0; i < len(tokens); i++ {
		container := tokens[i]
		i++
		if i >= len(tokens) {
			return nil, fmt.Errorf("pointer %q: %w: %q", pointer, errPointerNoKey, container)
		}
		key := decodePointerToken(tokens[i])
		next, ok := selectChild(cur, container, key)
		if !ok {
			return nil, fmt.Errorf("pointer %q: %w: %s/%s", pointer, errPointerNotFound, container, key)
		}
		cur = next
	}

	return cur, nil
}

// selectChild looks up key inside the named container map of node.
func selectChild(node *schemaNode, container, key string) (*schemaNode, bool) {
	switch container {
	case "definitions":
		child, ok := node.Definitions[key]

		return child, ok
	case "properties":
		child, ok := node.Properties[key]

		return child, ok
	default:
		return nil, false
	}
}

// decodePointerToken reverses RFC 6901 escaping (~1 → /, ~0 → ~).
func decodePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~1", "/")
	token = strings.ReplaceAll(token, "~0", "~")

	return token
}
