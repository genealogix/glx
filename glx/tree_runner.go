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
	"os"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// treeParentChildRelTypes is the set of relationship types that represent
// parent-child connections for tree traversal.
var treeParentChildRelTypes = map[string]bool{
	"parent_child":            true,
	"biological_parent_child": true,
	"adoptive_parent_child":   true,
	"foster_parent_child":     true,
	"step_parent":             true,
}

// treeNode represents a person in the ancestor/descendant tree.
type treeNode struct {
	PersonID string
	Name     string
	Dates    string // "(1850 – 1920)" or similar
	RelType  string // relationship type connecting to parent/child
	Children []*treeNode
}

// treeRelPerson holds a person ID and the relationship type connecting them.
type treeRelPerson struct {
	personID string
	relType  string
}

// treeContext holds the archive and prebuilt adjacency indexes for efficient
// tree traversal without repeated O(relationships) scans.
type treeContext struct {
	archive  *glxlib.GLXFile
	parents  map[string][]treeRelPerson // child ID → parents
	children map[string][]treeRelPerson // parent ID → children
}

// newTreeContext builds a treeContext with precomputed parent/child indexes.
func newTreeContext(archive *glxlib.GLXFile) *treeContext {
	return &treeContext{
		archive:  archive,
		parents:  buildParentIndex(archive),
		children: buildChildIndex(archive),
	}
}

// buildParentIndex scans all relationships once and returns a map from
// child ID to their parents.
func buildParentIndex(archive *glxlib.GLXFile) map[string][]treeRelPerson {
	index := make(map[string][]treeRelPerson)

	for _, rel := range archive.Relationships {
		if !treeParentChildRelTypes[rel.Type] {
			continue
		}

		var parentIDs []string
		var childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case "parent":
				parentIDs = append(parentIDs, p.Person)
			case "child":
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, childID := range childIDs {
			for _, parentID := range parentIDs {
				index[childID] = append(index[childID], treeRelPerson{
					personID: parentID,
					relType:  rel.Type,
				})
			}
		}
	}

	for id := range index {
		sortRelatedPersons(index[id])
	}

	return index
}

// buildChildIndex scans all relationships once and returns a map from
// parent ID to their children.
func buildChildIndex(archive *glxlib.GLXFile) map[string][]treeRelPerson {
	index := make(map[string][]treeRelPerson)

	for _, rel := range archive.Relationships {
		if !treeParentChildRelTypes[rel.Type] {
			continue
		}

		var parentIDs []string
		var childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case "parent":
				parentIDs = append(parentIDs, p.Person)
			case "child":
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, parentID := range parentIDs {
			for _, childID := range childIDs {
				index[parentID] = append(index[parentID], treeRelPerson{
					personID: childID,
					relType:  rel.Type,
				})
			}
		}
	}

	for id := range index {
		sortRelatedPersons(index[id])
	}

	return index
}

// findParents returns the parents of a person using the prebuilt index.
func findParents(tc *treeContext, personID string) []treeRelPerson {
	return tc.parents[personID]
}

// findChildren returns the children of a person using the prebuilt index.
func findChildren(tc *treeContext, personID string) []treeRelPerson {
	return tc.children[personID]
}

// loadArchiveForTree loads a GLX archive for tree operations.
func loadArchiveForTree(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}

		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// showAncestors loads the archive and prints the ancestor tree for a person.
// When ancestors are missing, it appends research suggestions.
func showAncestors(archivePath, personID string, maxGen int) error {
	archive, err := loadArchiveForTree(archivePath)
	if err != nil {
		return err
	}

	if _, ok := archive.Persons[personID]; !ok {
		return fmt.Errorf("person %q not found in archive", personID)
	}

	tc := newTreeContext(archive)
	root := buildAncestorTree(tc, personID, maxGen, 0, make(map[string]bool))
	printTree(root, "", true)

	// Show research suggestions for ancestors with missing parents.
	// buildAncestorSuggestions checks the actual parent index (not the tree)
	// so it correctly distinguishes maxGen limits from truly unknown parents.
	// printAncestorSuggestions no-ops when there are no suggestions.
	suggestions := buildAncestorSuggestions(tc, personID, archive)
	printAncestorSuggestions(suggestions)

	return nil
}

// showDescendants loads the archive and prints the descendant tree for a person.
func showDescendants(archivePath, personID string, maxGen int) error {
	archive, err := loadArchiveForTree(archivePath)
	if err != nil {
		return err
	}

	if _, ok := archive.Persons[personID]; !ok {
		return fmt.Errorf("person %q not found in archive", personID)
	}

	tc := newTreeContext(archive)
	root := buildDescendantTree(tc, personID, maxGen, 0, make(map[string]bool))
	printTree(root, "", true)

	return nil
}

// buildAncestorTree recursively builds the ancestor tree for a person.
// The visited map uses path-scoped cycle detection: entries are marked on entry
// and unmarked on return, allowing the same person to appear via multiple
// legitimate paths (pedigree collapse).
func buildAncestorTree(tc *treeContext, personID string, maxGen, depth int, visited map[string]bool) *treeNode {
	node := makeTreeNode(tc.archive, personID)

	if visited[personID] {
		return node
	}
	visited[personID] = true
	defer delete(visited, personID)

	if maxGen > 0 && depth >= maxGen {
		return node
	}

	parents := findParents(tc, personID)
	for _, p := range parents {
		child := buildAncestorTree(tc, p.personID, maxGen, depth+1, visited)
		child.RelType = p.relType
		node.Children = append(node.Children, child)
	}

	return node
}

// buildDescendantTree recursively builds the descendant tree for a person.
// The visited map uses path-scoped cycle detection: entries are marked on entry
// and unmarked on return, allowing the same person to appear via multiple
// legitimate paths (pedigree collapse).
func buildDescendantTree(tc *treeContext, personID string, maxGen, depth int, visited map[string]bool) *treeNode {
	node := makeTreeNode(tc.archive, personID)

	if visited[personID] {
		return node
	}
	visited[personID] = true
	defer delete(visited, personID)

	if maxGen > 0 && depth >= maxGen {
		return node
	}

	children := findChildren(tc, personID)
	for _, c := range children {
		child := buildDescendantTree(tc, c.personID, maxGen, depth+1, visited)
		child.RelType = c.relType
		node.Children = append(node.Children, child)
	}

	return node
}

// sortRelatedPersons sorts related persons by person ID for deterministic output.
func sortRelatedPersons(persons []treeRelPerson) {
	sort.Slice(persons, func(i, j int) bool {
		return persons[i].personID < persons[j].personID
	})
}

// makeTreeNode creates a tree node with display info for a person.
func makeTreeNode(archive *glxlib.GLXFile, personID string) *treeNode {
	person := archive.Persons[personID]

	name := "(unknown)"
	dates := ""
	if person != nil {
		name = extractPersonName(person)

		var bornOn, diedOn string
		if _, birthEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth); birthEvent != nil {
			bornOn = string(birthEvent.Date)
		}
		if _, deathEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeDeath); deathEvent != nil {
			diedOn = string(deathEvent.Date)
		}

		switch {
		case bornOn != "" && diedOn != "":
			dates = fmt.Sprintf("(%s – %s)", bornOn, diedOn)
		case bornOn != "":
			dates = fmt.Sprintf("(b. %s)", bornOn)
		case diedOn != "":
			dates = fmt.Sprintf("(d. %s)", diedOn)
		}
	}

	return &treeNode{
		PersonID: personID,
		Name:     name,
		Dates:    dates,
	}
}

// printTree prints a tree structure with box-drawing characters.
func printTree(node *treeNode, prefix string, isRoot bool) {
	if isRoot {
		label := formatNodeLabel(node)
		fmt.Println(label)
	}

	for i, child := range node.Children {
		isLast := i == len(node.Children)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		label := formatNodeLabel(child)
		if child.RelType != "" && child.RelType != "parent_child" {
			label += fmt.Sprintf("  [%s]", formatRelType(child.RelType))
		}

		fmt.Printf("%s%s%s\n", prefix, connector, label)
		printTree(child, prefix+childPrefix, false)
	}
}

// formatNodeLabel formats a tree node as "name  (dates)  id".
func formatNodeLabel(node *treeNode) string {
	var parts []string
	parts = append(parts, node.Name)
	if node.Dates != "" {
		parts = append(parts, node.Dates)
	}
	parts = append(parts, node.PersonID)

	return strings.Join(parts, "  ")
}

// formatRelType converts a relationship type to a human-readable label.
func formatRelType(relType string) string {
	switch relType {
	case "biological_parent_child":
		return "biological"
	case "adoptive_parent_child":
		return "adoptive"
	case "foster_parent_child":
		return "foster"
	case "step_parent":
		return "step"
	default:
		return relType
	}
}
