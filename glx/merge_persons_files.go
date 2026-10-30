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

	"go.yaml.in/yaml/v3"

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
	if err := materializeMergeCollections(roots, result.Changes); err != nil {
		return nil, err
	}
	if err := preserveMergeAliases(roots, result.Changes); err != nil {
		return nil, err
	}
	if err := validateMergeEditedEntities(roots, result.Changes); err != nil {
		return nil, err
	}
	_, dropParent, dropIndex, err := findMergeEntity(roots, glxlib.EntityTypePersons, dropID)
	if err != nil {
		return nil, err
	}
	fallback := cloneMergeNode(dropParent.Content[dropIndex+1])
	changedFiles := make(map[string]bool)
	references := make(map[mergeFileEntityKey][]glxlib.MergeReferenceChange)
	for _, reference := range result.ReferenceChanges {
		key := mergeFileEntityKey{reference.EntityType, reference.ID}
		references[key] = append(references[key], reference)
	}
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
		context := mergeFileNodeContext(references[mergeFileEntityKey{change.EntityType, change.ID}], keepID, dropID)
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
	if err := verifyMergeFiles(files, original, archive, directory); err != nil {
		return nil, err
	}

	return files, nil
}

// Entity discovery must see mappings rather than aliases. Copy only collection
// occurrences containing changed entities: editing the anchor's original node
// would also alter snapshots and unrelated consumers of that collection.
func materializeMergeCollections(roots map[string]*yaml.Node, changes []glxlib.EntityChange) error {
	if err := materializeMergeRoots(roots, changes); err != nil {
		return err
	}
	for i := range changes {
		for _, root := range roots {
			if err := materializeMergeCollection(roots, root, &changes[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func materializeMergeRoots(roots map[string]*yaml.Node, changes []glxlib.EntityChange) error {
	for _, root := range roots {
		if !hasMergeCollectionKey(root) {
			continue
		}
		needed, err := rootInheritsMergeChange(root, changes)
		if err != nil {
			return err
		}
		if !needed {
			continue
		}
		if err := freezeMergeInheritedSources(roots, root); err != nil {
			return err
		}
		if err := flattenMergeCollection(root); err != nil {
			return err
		}
	}

	return nil
}

// Keep unrelated root inheritance and its aliases spelled exactly as before.
func rootInheritsMergeChange(root *yaml.Node, changes []glxlib.EntityChange) (bool, error) {
	expanded, err := expandMergeAlias(root, nil, nil, make(map[*yaml.Node]bool), true)
	if err != nil {
		return false, err
	}
	if err := flattenMergeCollection(expanded); err != nil {
		return false, err
	}
	for _, change := range changes {
		if mergeMappingValue(root, string(change.EntityType)) != nil {
			continue
		}
		collection := mergeMappingValue(expanded, string(change.EntityType))
		if collection == nil {
			continue
		}
		if err := flattenMergeCollection(collection); err != nil {
			return false, err
		}
		if mergeMappingValue(collection, change.ID) != nil {
			return true, nil
		}
	}

	return false, nil
}

func materializeMergeCollection(roots map[string]*yaml.Node, root *yaml.Node, change *glxlib.EntityChange) error {
	collection := mergeMappingValue(root, string(change.EntityType))
	if collection == nil || (collection.Kind != yaml.AliasNode && !hasMergeCollectionKey(collection)) {
		return nil
	}
	expanded, err := expandMergeAlias(collection, nil, nil, make(map[*yaml.Node]bool), true)
	if err != nil {
		return err
	}
	if err := flattenMergeCollection(expanded); err != nil {
		return err
	}
	if mergeMappingValue(expanded, change.ID) == nil {
		return nil
	}
	if err := freezeMergeCollectionConsumers(roots, collection); err != nil {
		return err
	}
	// Collection mappings are mutated in place; attach occurrence comments to
	// their key so the encoder retains them after the edit.
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i+1] == collection {
			root.Content[i].HeadComment = strings.TrimSpace(root.Content[i].HeadComment + "\n" + expanded.HeadComment)
			expanded.HeadComment = ""

			break
		}
	}
	*collection = *expanded

	return nil
}

// Flattening a mapping replaces its anchored descendants too. Freeze other
// consumers before those anchors disappear; alias collections retain their source.
func freezeMergeCollectionConsumers(roots map[string]*yaml.Node, collection *yaml.Node) error {
	if collection.Kind != yaml.MappingNode {
		return nil
	}
	affected := make(map[*yaml.Node]bool)
	outgoing := make(map[*yaml.Node]bool)
	collectMergeAliases(collection, affected, outgoing)
	for _, document := range roots {
		if _, err := expandMergeAlias(document, affected, outgoing, make(map[*yaml.Node]bool), false); err != nil {
			return err
		}
	}

	return nil
}

// Only anchors physically defined in a removed << value disappear. References
// to sources defined elsewhere keep their original nodes and alias spelling.
func freezeMergeInheritedSources(roots map[string]*yaml.Node, mapping *yaml.Node) error {
	affected := make(map[*yaml.Node]bool)
	outgoing := make(map[*yaml.Node]bool)
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Tag == yamlMergeTag {
			collectMergeAliases(mapping.Content[i+1], affected, outgoing)
		}
	}
	for _, root := range roots {
		if _, err := expandMergeAlias(root, affected, outgoing, make(map[*yaml.Node]bool), false); err != nil {
			return err
		}
	}

	return nil
}

func hasMergeCollectionKey(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Tag == yamlMergeTag {
			return true
		}
	}

	return false
}

// YAML merge sources are ordered: explicit keys win, then the first inherited
// occurrence. Flatten only the copied collection so deleting an inherited
// entity cannot resurrect it or change another anchor consumer.
func flattenMergeCollection(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("%w: entity collection is not a mapping", glxlib.ErrValidationFailed)
	}
	var explicit, sources []*yaml.Node
	var mergeComments []string
	seen := make(map[string]bool)
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Tag == yamlMergeTag {
			mergeComments = append(mergeComments, key.HeadComment, key.LineComment, key.FootComment, value.HeadComment, value.LineComment, value.FootComment)
			if value.Kind == yaml.SequenceNode {
				sources = append(sources, value.Content...)
			} else {
				sources = append(sources, value)
			}
		} else {
			explicit = append(explicit, key, value)
			seen[mergeNodeKey(key)] = true
		}
	}
	for _, source := range sources {
		inherited, err := expandMergeAlias(source, nil, nil, make(map[*yaml.Node]bool), true)
		if err != nil {
			return err
		}
		if err := flattenMergeCollection(inherited); err != nil {
			return err
		}
		for i := 0; i < len(inherited.Content); i += 2 {
			key := inherited.Content[i]
			if !seen[mergeNodeKey(key)] {
				explicit = append(explicit, key, inherited.Content[i+1])
				seen[mergeNodeKey(key)] = true
			}
		}
	}
	if len(explicit) > 0 {
		comments := strings.TrimSpace(strings.Join(mergeComments, "\n"))
		if comments != "" {
			explicit[0].HeadComment = strings.TrimSpace(comments + "\n" + explicit[0].HeadComment)
		}
	}
	node.Content = explicit

	return nil
}

// Materialize aliases at the edit boundary before any nodes are moved or
// deleted. Aliases outside that boundary keep their original representation.
func preserveMergeAliases(roots map[string]*yaml.Node, changes []glxlib.EntityChange) error {
	affected := make(map[*yaml.Node]bool)
	outgoing := make(map[*yaml.Node]bool)
	edited := make(map[*yaml.Node]bool)
	for _, change := range changes {
		_, parent, index, err := findMergeEntity(roots, change.EntityType, change.ID)
		if err != nil {
			return err
		}
		for _, node := range parent.Content[index : index+2] {
			edited[node] = true
			collectMergeAliases(node, affected, outgoing)
		}
	}
	for _, root := range roots {
		markMergeAliasAncestors(root, edited, affected)
	}
	for _, root := range roots {
		if _, err := expandMergeAlias(root, affected, outgoing, make(map[*yaml.Node]bool), false); err != nil {
			return err
		}
	}

	return nil
}

func collectMergeAliases(node *yaml.Node, affected, outgoing map[*yaml.Node]bool) {
	if node.Anchor != "" {
		affected[node] = true
	}
	if node.Kind == yaml.AliasNode {
		outgoing[node] = true
	}
	for _, child := range node.Content {
		collectMergeAliases(child, affected, outgoing)
	}
}

// A collection (or another ancestor) changes when an entity is edited. Its
// aliases must retain the original collection, including any removed members.
func markMergeAliasAncestors(node *yaml.Node, edited, affected map[*yaml.Node]bool) bool {
	contains := edited[node]
	for i, child := range node.Content {
		if markMergeAliasAncestors(child, edited, affected) {
			contains = true
			if node.Kind == yaml.MappingNode && i%2 == 1 && node.Content[i-1].Anchor != "" {
				affected[node.Content[i-1]] = true
			}
		}
	}
	if contains && node.Anchor != "" {
		affected[node] = true
	}

	return contains
}

func expandMergeAlias(node *yaml.Node, affected, outgoing, stack map[*yaml.Node]bool, copied bool) (*yaml.Node, error) {
	// Copied subtrees must be self-contained: even an unchanged anchor can be
	// later in the destination file, or belong to a different source file.
	if node.Kind == yaml.AliasNode && (copied || affected[node.Alias] || outgoing[node]) {
		if node.Alias == nil || stack[node.Alias] {
			return nil, fmt.Errorf("%w: invalid or cyclic YAML alias %s", glxlib.ErrValidationFailed, node.Value)
		}
		stack[node.Alias] = true
		out, err := expandMergeAlias(node.Alias, affected, outgoing, stack, true)
		delete(stack, node.Alias)
		if err != nil {
			return nil, err
		}
		out.HeadComment, out.LineComment, out.FootComment = node.HeadComment, node.LineComment, node.FootComment
		// Block collections have no inline value on which YAML can emit a
		// line comment. Keep the alias occurrence's comment above the value.
		if (out.Kind == yaml.MappingNode || out.Kind == yaml.SequenceNode) && out.Style&yaml.FlowStyle == 0 && out.LineComment != "" {
			out.HeadComment = strings.TrimSpace(out.HeadComment + "\n" + out.LineComment)
			out.LineComment = ""
		}

		return out, nil
	}
	out := node
	if copied {
		value := *node
		out = &value
		out.Anchor = ""
		out.Content = make([]*yaml.Node, len(node.Content))
	}
	for i, child := range node.Content {
		replacement, err := expandMergeAlias(child, affected, outgoing, stack, copied)
		if err != nil {
			return nil, err
		}
		out.Content[i] = replacement
	}

	return out, nil
}

type mergeFileEntityKey struct {
	kind glxlib.EntityType
	id   string
}

func mergeFileNodeContext(references []glxlib.MergeReferenceChange, keepID, dropID string) mergeNodeContext {
	context := mergeNodeContext{keepID: keepID, dropID: dropID, personPaths: make(map[string]bool)}
	for _, reference := range references {
		if reference.TargetType == glxlib.EntityTypePersons {
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
		root := document.Content[0]
		// Keep document comments when editing the root mapping directly.
		root.HeadComment = strings.TrimSpace(document.HeadComment + "\n" + root.HeadComment)
		root.FootComment = strings.TrimSpace(root.FootComment + "\n" + document.FootComment)
		roots[path] = root
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
			if mergeNodeKey(parent.Content[index]) == id {
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

// YAML permits aliases to scalar mapping keys. Decode them for matching while
// retaining their source nodes whenever an unaffected key can remain intact.
func mergeNodeKey(node *yaml.Node) string {
	if node.Kind == yaml.ScalarNode {
		return node.Value
	}
	var key string
	if err := node.Decode(&key); err != nil {
		return ""
	}

	return key
}

func mergeMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		if mergeNodeKey(node.Content[i]) == key {
			return node.Content[i+1]
		}
	}

	return nil
}

func removeMergeMappingKey(node *yaml.Node, key string) {
	for i := 0; i < len(node.Content); i += 2 {
		if mergeNodeKey(node.Content[i]) == key {
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

const (
	yamlStringTag = "!!str"
	yamlMergeTag  = "!!merge"
)

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
			if mergeNodeKey(out.Content[i]) == "value" {
				out.Content[i+1] = rewriteMergeReferenceNode(out.Content[i+1], context)
			}
		}
	case yaml.ScalarNode:
		if out.Tag == yamlStringTag && out.Value == context.dropID {
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
		key := mergeNodeKey(desired.Content[i])
		childPath := key
		if path != "" {
			childPath = path + "." + key
		}
		out.Content[i+1] = patchMergeNode(mergeMappingValue(original, key), desired.Content[i+1], mergeMappingValue(fallback, key), childPath, context)
		// A property contributed only by drop also brings its key's comments.
		// Prefer keep's key whenever both records contain the property.
		keySource := original
		if mergeMappingValue(original, key) == nil {
			keySource = fallback
		}
		if keySource != nil && keySource.Kind == yaml.MappingNode {
			for j := 0; j < len(keySource.Content); j += 2 {
				if mergeNodeKey(keySource.Content[j]) == key {
					out.Content[i] = cloneMergeNode(keySource.Content[j])

					break
				}
			}
		}
	}
}

func patchMergeSequence(out, original, desired, fallback *yaml.Node, path string, context mergeNodeContext) {
	usedOriginal, usedFallback := make(map[*yaml.Node]bool), make(map[*yaml.Node]bool)
	for i, child := range desired.Content {
		var previous *yaml.Node
		// MergePersons preserves participant, neighbor and research-lead membership/order. A person ID
		// rewrite changes identity equality but not this entry's provenance.
		switch {
		case positionalMergeSequence(path) && original != nil && original.Kind == yaml.SequenceNode && len(original.Content) == len(desired.Content):
			previous = original.Content[i]
		case original != nil && original.Kind == yaml.SequenceNode && i < len(original.Content) && !usedOriginal[original.Content[i]] && equalMergeNodes(original.Content[i], child):
			previous = original.Content[i]
		default:
			previous = matchingMergeNode(original, child, usedOriginal)
		}
		alternative := matchingMergeNode(fallback, child, usedFallback)
		// Consume only contributing occurrences. An equal original wins
		// without using a fallback that may belong to a later unioned value.
		if previous != nil && (equalMergeNodes(previous, child) || !equalMergeNodes(alternative, child)) {
			usedOriginal[previous] = true
		}
		if alternative != nil && !equalMergeNodes(previous, child) {
			usedFallback[alternative] = true
		}
		out.Content[i] = patchMergeNode(previous, child, alternative, fmt.Sprintf("%s[%d]", path, i), context)
	}
}

// These collections retain membership and order throughout a person merge.
func positionalMergeSequence(path string) bool {
	return path == "participants" || path == "neighbors" || path == "leads"
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

func matchingMergeNode(container, desired *yaml.Node, used map[*yaml.Node]bool) *yaml.Node {
	if container == nil || used[container] {
		return nil
	}
	if container.Kind != yaml.SequenceNode {
		return container
	}
	for _, child := range container.Content {
		if !used[child] && equalMergeNodes(child, desired) {
			return child
		}
	}
	// A refined/combined structured value can still reuse its original scalar
	// spellings and comments through the recursive mapping edit.
	for _, child := range container.Content {
		if !used[child] && equalMergeNodes(mergeMappingValue(child, "value"), mergeMappingValue(desired, "value")) {
			return child
		}
	}

	return nil
}

// Unknown structural fields on an edited/deleted entity cannot be discarded
// by typed re-encoding. Refuse those edits while unrelated entities retain
// their original bytes and pre-existing structural issues.
func validateMergeEditedEntities(roots map[string]*yaml.Node, changes []glxlib.EntityChange) error {
	for _, change := range changes {
		_, parent, index, err := findMergeEntity(roots, change.EntityType, change.ID)
		if err != nil {
			return err
		}
		entity := parent.Content[index+1]
		if hasMergeCollectionKey(entity) {
			if err := flattenMergeCollection(entity); err != nil {
				return err
			}
		}
		collection := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: yamlStringTag, Value: change.ID}, cloneMergeNode(entity)}}
		fragment := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: yamlStringTag, Value: string(change.EntityType)}, collection}}
		data, err := yaml.Marshal(fragment)
		if err != nil {
			return err
		}
		issues, err := mergeStructureIssues(data)
		if err != nil {
			return err
		}
		if len(issues) > 0 {
			return fmt.Errorf("%w: cannot safely edit %s[%s] with existing structural errors: %s", glxlib.ErrValidationFailed, change.EntityType, change.ID, strings.Join(issues, "; "))
		}
	}

	return nil
}

func mergeStructureIssues(data []byte) ([]string, error) {
	document, err := ParseYAMLFile(data)
	if err != nil {
		return nil, err
	}

	return ValidateGLXFileStructure(document), nil
}

func verifyMergeStructure(files, original map[string][]byte) error {
	for path, data := range files {
		if bytes.Equal(original[path], data) {
			continue
		}
		baseline := make(map[string]int)
		if old, exists := original[path]; exists {
			issues, err := mergeStructureIssues(old)
			if err != nil {
				return err
			}
			for _, issue := range issues {
				baseline[issue]++
			}
		}
		issues, err := mergeStructureIssues(data)
		if err != nil {
			return err
		}
		for _, issue := range issues {
			if baseline[issue] == 0 {
				return fmt.Errorf("%w: %s: %s", glxlib.ErrValidationFailed, path, issue)
			}
			baseline[issue]--
		}
	}

	return nil
}

func verifyMergeFiles(files, original map[string][]byte, expected *glxlib.GLXFile, directory bool) error {
	if err := verifyMergeStructure(files, original); err != nil {
		return err
	}
	actual, err := loadMergeFiles("", files, directory)
	if err != nil {
		return err
	}
	// The SDK has already checked the detached plan against the input. Existing
	// unrelated errors must not turn a serialization check into a new merge ban.
	// Reject only validation errors introduced by the round trip.
	before, err := loadMergeFiles("", original, directory)
	if err != nil {
		return err
	}
	baseline := make(map[glxlib.ValidationError]int)
	for _, issue := range before.Validate().Errors {
		baseline[issue]++
	}
	for _, issue := range actual.Validate().Errors {
		if baseline[issue] == 0 {
			return fmt.Errorf("%w: %s", glxlib.ErrValidationFailed, issue.Message)
		}
		baseline[issue]--
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
