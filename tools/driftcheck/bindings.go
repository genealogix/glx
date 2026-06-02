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
	"fmt"
	"maps"
	"reflect"

	glxlib "github.com/genealogix/glx/go-glx"
)

// glxFileSchema is the top-level schema file; every entity and vocabulary
// definition is reachable from it via $ref.
const glxFileSchema = "glx-file.schema.json"

// binding pairs a Go type with the schema node it must conform to. refStr is
// resolved relative to the schema root directory and may carry a JSON pointer
// (e.g. "#/definitions/PropertyDefinition").
type binding struct {
	entity  string
	goType  reflect.Type
	refStr  string
	recurse bool
}

// entityBindings lists every Go type with a 1:1 (or canonical) schema mapping.
// Nested types (Participant, Submitter, Search, EntityRef) are reached by
// recursion from these roots, so they need no explicit entry.
func entityBindings() []binding {
	t := reflect.TypeOf

	return []binding{
		{"Person", t(glxlib.Person{}), "person.schema.json", true},
		{"Event", t(glxlib.Event{}), "event.schema.json", true},
		{"Relationship", t(glxlib.Relationship{}), "relationship.schema.json", true},
		{"Place", t(glxlib.Place{}), "place.schema.json", true},
		{"Source", t(glxlib.Source{}), "source.schema.json", true},
		{"Citation", t(glxlib.Citation{}), "citation.schema.json", true},
		{"Repository", t(glxlib.Repository{}), "repository.schema.json", true},
		{"Media", t(glxlib.Media{}), "media.schema.json", true},
		{"Assertion", t(glxlib.Assertion{}), "assertion.schema.json", true},
		{"ResearchLog", t(glxlib.ResearchLog{}), "research-log.schema.json", true},
		{"Study", t(glxlib.Study{}), "study.schema.json", true},
		{"Metadata", t(glxlib.Metadata{}), glxFileSchema + "#/properties/metadata", true},
		{
			"PropertyDefinition", t(glxlib.PropertyDefinition{}),
			"vocabularies/person-properties.schema.json#/definitions/PropertyDefinition", true,
		},
		{
			"FieldDefinition", t(glxlib.FieldDefinition{}),
			"vocabularies/person-properties.schema.json#/definitions/FieldDefinition", true,
		},
	}
}

// run executes every comparison and leaves the results in c.findings.
func (c *checker) run() {
	for _, b := range entityBindings() {
		node, loc, err := c.schemas.resolve(b.refStr, "")
		if err != nil {
			c.add(&finding{
				Severity: severityCritical, Entity: b.entity, Category: "Internal",
				File: c.typesFile, Symbol: b.entity,
				Message: fmt.Sprintf("cannot resolve schema %q: %v", b.refStr, err),
			})

			continue
		}
		node, loc = c.derefNode(b.entity, node, loc)
		if node != nil {
			c.compareStruct(b.entity, b.goType, node, loc, b.recurse)
		}
	}

	c.checkGLXFileTopLevel()
	c.checkVocabularyEntry()
}

// checkGLXFileTopLevel verifies the GLXFile container struct exposes exactly the
// top-level properties the schema declares (entity maps, vocabulary maps,
// metadata) with matching yaml tags. It does not recurse — each referenced
// entity/vocabulary is covered by its own binding.
func (c *checker) checkGLXFileTopLevel() {
	root, loc, err := c.schemas.resolve(glxFileSchema, "")
	if err != nil {
		c.add(&finding{
			Severity: severityCritical, Entity: "GLXFile", Category: "Internal",
			File: c.typesFile, Symbol: "GLXFile",
			Message: fmt.Sprintf("cannot resolve %s: %v", glxFileSchema, err),
		})

		return
	}
	c.compareStruct("GLXFile", reflect.TypeFor[glxlib.GLXFile](), root, loc, false)
}

// checkVocabularyEntry compares the unified VocabularyEntry struct (one Go type
// backing all type-vocabulary schemas since #727) against the *union* of every
// type-vocabulary definition's properties. A field is drift only if it appears
// in no vocabulary schema; a property is drift only if no Go field covers it.
func (c *checker) checkVocabularyEntry() {
	vocabType := reflect.TypeFor[glxlib.VocabularyEntry]()

	root, rootLoc, err := c.schemas.resolve(glxFileSchema, "")
	if err != nil {
		return // already reported by checkGLXFileTopLevel
	}

	glxFields := structFieldsByYAML(reflect.TypeFor[glxlib.GLXFile]())

	merged := &schemaNode{
		Type:                 "object",
		Properties:           map[string]*schemaNode{},
		AdditionalProperties: []byte("false"),
	}
	var requiredSets [][]string
	var contributed int

	for _, propName := range sortedKeys(root.Properties) {
		field, ok := glxFields.m[propName]
		if !ok || !isVocabularyEntryMap(field.Type, vocabType) {
			continue
		}
		valSchema := singleValueSchema(root.Properties[propName])
		if valSchema == nil {
			continue
		}
		def, _ := c.derefNode("VocabularyEntry", valSchema, rootLoc)
		if def == nil {
			continue
		}
		maps.Copy(merged.Properties, def.Properties)
		requiredSets = append(requiredSets, def.Required)
		contributed++
	}

	if contributed == 0 {
		c.add(&finding{
			Severity: severityMajor, Entity: "VocabularyEntry", Category: "Internal",
			File: c.typesFile, Symbol: "VocabularyEntry",
			Message: "no type-vocabulary schemas resolved for VocabularyEntry union check",
		})

		return
	}

	merged.Required = intersect(requiredSets)
	// The merged node spans many files; cite the union rather than an arbitrary
	// contributing file. None of VocabularyEntry's properties carry nested
	// $refs, so this loc is never used for further resolution.
	unionLoc := ref{file: "vocabularies/*-types.schema.json (union)"}
	c.compareStruct("VocabularyEntry", vocabType, merged, unionLoc, true)
}

// isVocabularyEntryMap reports whether t is map[string]*VocabularyEntry.
func isVocabularyEntryMap(t, vocabType reflect.Type) bool {
	return t.Kind() == reflect.Map && derefType(t.Elem()) == vocabType
}

// intersect returns the strings present in every set (the fields required by
// all contributing schemas — e.g. "label").
func intersect(sets [][]string) []string {
	if len(sets) == 0 {
		return nil
	}
	counts := map[string]int{}
	for _, s := range sets {
		for _, v := range stringSliceUnique(s) {
			counts[v]++
		}
	}
	out := make([]string, 0, len(counts))
	for v, n := range counts {
		if n == len(sets) {
			out = append(out, v)
		}
	}

	return out
}

func stringSliceUnique(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}

	return out
}
