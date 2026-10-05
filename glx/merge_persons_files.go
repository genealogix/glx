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
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Edit only changed entity nodes in their original files. This retains archive
// vocabularies, comments, custom scalar spellings and untouched file bytes.
// It also avoids the generic serializer replacing custom vocabularies with
// the starter set. Everything is reloaded and checked before approval.
func prepareMergeFiles(original map[string][]byte, archive *glxlib.GLXFile, result *glxlib.MergePersonsResult, keepID, dropID string, directory bool) (map[string][]byte, error) {
	files := maps.Clone(original)
	roots, err := parseMergeNodes(original)
	if err != nil {
		return nil, err
	}
	_, dropParent, dropIndex, err := findMergeEntity(roots, glxlib.EntityTypePersons, dropID)
	if err != nil {
		return nil, err
	}
	fallback := cloneMergeNode(dropParent.Content[dropIndex+1])
	changedFiles := make(map[string]bool)
	for _, change := range result.Changes {
		path, parent, index, err := findMergeEntity(roots, change.EntityType, change.ID)
		if err != nil {
			return nil, err
		}
		changedFiles[path] = true
		if change.Kind == glxlib.ChangeRemoved {
			parent.Content = slices.Delete(parent.Content, index, index+2)
			if len(parent.Content) == 0 {
				removeMergeMappingKey(roots[path], string(change.EntityType))
			}

			continue
		}
		entity, err := mergeFileEntity(archive, change.EntityType, change.ID)
		if err != nil {
			return nil, err
		}
		var desired yaml.Node
		if err := desired.Encode(entity); err != nil {
			return nil, err
		}
		var alternatives *yaml.Node
		if change.EntityType == glxlib.EntityTypePersons && change.ID == keepID {
			alternatives = fallback
		}
		context := mergeFileNodeContext(&change, result, keepID, dropID)
		parent.Content[index+1] = patchMergeNode(parent.Content[index+1], &desired, alternatives, "", context)
	}
	for path := range changedFiles {
		root := roots[path]
		if len(root.Content) == 0 {
			delete(files, path)

			continue
		}
		var out bytes.Buffer
		encoder := yaml.NewEncoder(&out)
		encoder.SetIndent(2)
		if err := encoder.Encode(root); err != nil {
			return nil, err
		}
		if err := encoder.Close(); err != nil {
			return nil, err
		}
		files[path] = out.Bytes()
	}
	if err := verifyMergeFiles(files, archive, directory); err != nil {
		return nil, err
	}

	return files, nil
}

func mergeFileNodeContext(change *glxlib.EntityChange, result *glxlib.MergePersonsResult, keepID, dropID string) mergeNodeContext {
	context := mergeNodeContext{keepID: keepID, dropID: dropID, personPaths: make(map[string]bool)}
	for _, reference := range result.ReferenceChanges {
		if reference.EntityType == change.EntityType && reference.ID == change.ID && reference.TargetType == glxlib.EntityTypePersons {
			path := reference.Path
			if strings.HasSuffix(path, "]") {
				path = path[:strings.LastIndex(path, "[")]
			}
			context.personPaths[path] = true
		}
	}

	return context
}

func parseMergeNodes(original map[string][]byte) (map[string]*yaml.Node, error) {
	roots := make(map[string]*yaml.Node, len(original))
	for path, data := range original {
		var document yaml.Node
		if err := yaml.Unmarshal(data, &document); err != nil {
			return nil, err
		}
		if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			return nil, glxlib.ErrValidationFailed
		}
		roots[path] = document.Content[0]
	}

	return roots, nil
}

func findMergeEntity(roots map[string]*yaml.Node, kind glxlib.EntityType, id string) (string, *yaml.Node, int, error) {
	paths := make([]string, 0, len(roots))
	for path := range roots {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	for _, path := range paths {
		parent := mergeMappingValue(roots[path], string(kind))
		if parent == nil || parent.Kind != yaml.MappingNode {
			continue
		}
		for index := 0; index < len(parent.Content); index += 2 {
			if parent.Content[index].Value == id {
				return path, parent, index, nil
			}
		}
	}

	return "", nil, 0, fmt.Errorf("%w: cannot locate %s[%s] in original files", glxlib.ErrValidationFailed, kind, id)
}

func mergeFileEntity(archive *glxlib.GLXFile, kind glxlib.EntityType, id string) (any, error) {
	root := reflect.ValueOf(archive).Elem()
	for i := range root.NumField() {
		if strings.Split(root.Type().Field(i).Tag.Get("yaml"), ",")[0] != string(kind) {
			continue
		}
		entity := root.Field(i).MapIndex(reflect.ValueOf(id))
		if entity.IsValid() {
			return entity.Interface(), nil
		}
	}

	return nil, fmt.Errorf("%w: missing planned %s[%s]", glxlib.ErrValidationFailed, kind, id)
}

func mergeMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func removeMergeMappingKey(node *yaml.Node, key string) {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = slices.Delete(node.Content, i, i+2)

			return
		}
	}
}

func cloneMergeNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	out := *node
	out.Content = make([]*yaml.Node, len(node.Content))
	for i, child := range node.Content {
		out.Content[i] = cloneMergeNode(child)
	}

	return &out
}

func equalMergeNodes(a, b *yaml.Node) bool {
	if a == nil || b == nil {
		return false
	}
	var first, second any
	if a.Decode(&first) != nil || b.Decode(&second) != nil {
		return false
	}

	return reflect.DeepEqual(first, second)
}

type mergeNodeContext struct {
	keepID, dropID string
	personPaths    map[string]bool
}

// Only the value of a declared reference occurrence is rewritten. Dates,
// fields and ordinary strings keep their original nodes and comments.
func rewriteMergeReferenceNode(node *yaml.Node, context mergeNodeContext) *yaml.Node {
	out := cloneMergeNode(node)
	if out == nil {
		return nil
	}
	switch out.Kind {
	case yaml.SequenceNode:
		for i, child := range out.Content {
			out.Content[i] = rewriteMergeReferenceNode(child, context)
		}
	case yaml.MappingNode:
		for i := 0; i < len(out.Content); i += 2 {
			if out.Content[i].Value == "value" {
				out.Content[i+1] = rewriteMergeReferenceNode(out.Content[i+1], context)
			}
		}
	case yaml.ScalarNode:
		if out.Tag == "!!str" && out.Value == context.dropID {
			out.Value = context.keepID
		}
	}

	return out
}

func patchMergeNode(original, desired, fallback *yaml.Node, path string, context mergeNodeContext) *yaml.Node {
	if context.personPaths[path] {
		fallback = contributingMergeNodes(original, fallback)
		original = rewriteMergeReferenceNode(original, context)
		fallback = rewriteMergeReferenceNode(fallback, context)
	}
	if equalMergeNodes(original, desired) {
		return cloneMergeNode(original)
	}
	if equalMergeNodes(fallback, desired) {
		return cloneMergeNode(fallback)
	}
	out := cloneMergeNode(desired)
	if original != nil {
		out.HeadComment, out.LineComment, out.FootComment = original.HeadComment, original.LineComment, original.FootComment
	}
	switch desired.Kind {
	case yaml.MappingNode:
		patchMergeMapping(out, original, desired, fallback, path, context)
	case yaml.SequenceNode:
		patchMergeSequence(out, original, desired, fallback, path, context)
	}

	return out
}

func patchMergeMapping(out, original, desired, fallback *yaml.Node, path string, context mergeNodeContext) {
	for i := 0; i < len(desired.Content); i += 2 {
		key := desired.Content[i].Value
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		out.Content[i+1] = patchMergeNode(mergeMappingValue(original, key), desired.Content[i+1], mergeMappingValue(fallback, key), childPath, context)
		if original != nil && original.Kind == yaml.MappingNode {
			for j := 0; j < len(original.Content); j += 2 {
				if original.Content[j].Value == key {
					out.Content[i] = cloneMergeNode(original.Content[j])

					break
				}
			}
		}
	}
}

func patchMergeSequence(out, original, desired, fallback *yaml.Node, path string, context mergeNodeContext) {
	for i, child := range desired.Content {
		var previous *yaml.Node
		// MergePersons preserves participant membership/order. A person ID
		// rewrite changes identity equality but not this entry's provenance.
		switch {
		case path == "participants" && original != nil && original.Kind == yaml.SequenceNode && len(original.Content) == len(desired.Content):
			previous = original.Content[i]
		case original != nil && original.Kind == yaml.SequenceNode && i < len(original.Content) && equalMergeNodes(original.Content[i], child):
			previous = original.Content[i]
		default:
			previous = matchingMergeNode(original, child)
		}
		alternative := matchingMergeNode(fallback, child)
		// Unioned values beyond keep's original sequence come from drop.
		// Rewriting B to A can make them equal to an existing keep value;
		// retain drop's occurrence comments rather than reusing keep's node.
		if original != nil && original.Kind == yaml.SequenceNode && i >= len(original.Content) && alternative != nil {
			previous = nil
		}
		out.Content[i] = patchMergeNode(previous, child, alternative, fmt.Sprintf("%s[%d]", path, i), context)
	}
}

// A drop occurrence already present in keep did not contribute to the union.
// Filter those matches before B→A makes distinct original IDs compare equal.
func contributingMergeNodes(original, fallback *yaml.Node) *yaml.Node {
	if original == nil || fallback == nil || original.Kind != yaml.SequenceNode || fallback.Kind != yaml.SequenceNode {
		return fallback
	}
	out := cloneMergeNode(fallback)
	out.Content = slices.DeleteFunc(out.Content, func(child *yaml.Node) bool {
		for _, kept := range original.Content {
			if equalMergeNodes(kept, child) {
				return true
			}
		}

		return false
	})

	return out
}

func matchingMergeNode(container, desired *yaml.Node) *yaml.Node {
	if container == nil {
		return nil
	}
	if container.Kind != yaml.SequenceNode {
		return container
	}
	for _, child := range container.Content {
		if equalMergeNodes(child, desired) {
			return child
		}
	}
	// A refined/combined structured value can still reuse its original scalar
	// spellings and comments through the recursive mapping edit.
	for _, child := range container.Content {
		if equalMergeNodes(mergeMappingValue(child, "value"), mergeMappingValue(desired, "value")) {
			return child
		}
	}

	return nil
}

func verifyMergeFiles(files map[string][]byte, expected *glxlib.GLXFile, directory bool) error {
	for path, data := range files {
		document, err := ParseYAMLFile(data)
		if err != nil {
			return err
		}
		if issues := ValidateGLXFileStructure(document); len(issues) != 0 {
			return fmt.Errorf("%w: %s: %s", glxlib.ErrValidationFailed, path, strings.Join(issues, "; "))
		}
	}
	actual, err := loadMergeFiles("", files, directory)
	if err != nil {
		return err
	}
	if validation := actual.Validate(); len(validation.Errors) != 0 {
		return fmt.Errorf("%w: %s", glxlib.ErrValidationFailed, validation.Errors[0].Message)
	}
	// Canonical YAML treats omitted and empty optional maps/slices alike and
	// normalizes SDK-only TemporalValue representations without retyping values.
	expectedBytes, err := yaml.Marshal(expected)
	if err != nil {
		return err
	}
	actualBytes, err := yaml.Marshal(actual)
	if err != nil {
		return err
	}
	if !bytes.Equal(expectedBytes, actualBytes) {
		return errMergeRoundtrip
	}

	return nil
}
