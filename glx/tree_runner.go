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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// parentChildRelTypes is the set of relationship types that represent
// parent-child connections for tree traversal.
var parentChildRelTypes = map[string]bool{
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
	Dates    string // "(b. 1850 – d. 1920)" or similar
	RelType  string // relationship type connecting to parent/child
	Children []*treeNode
}

// showAncestors loads the archive and prints the ancestor tree for a person.
func showAncestors(archivePath, personID string, maxGen int) error {
	archive, err := loadArchiveForQuery(archivePath)
	if err != nil {
		return err
	}

	if _, ok := archive.Persons[personID]; !ok {
		return fmt.Errorf("person %q not found in archive", personID)
	}

	root := buildAncestorTree(archive, personID, maxGen, 0, make(map[string]bool))
	printTree(root, "", true)

	return nil
}

// showDescendants loads the archive and prints the descendant tree for a person.
func showDescendants(archivePath, personID string, maxGen int) error {
	archive, err := loadArchiveForQuery(archivePath)
	if err != nil {
		return err
	}

	if _, ok := archive.Persons[personID]; !ok {
		return fmt.Errorf("person %q not found in archive", personID)
	}

	root := buildDescendantTree(archive, personID, maxGen, 0, make(map[string]bool))
	printTree(root, "", true)

	return nil
}

// buildAncestorTree recursively builds the ancestor tree for a person.
func buildAncestorTree(archive *glxlib.GLXFile, personID string, maxGen, depth int, visited map[string]bool) *treeNode {
	node := makeTreeNode(archive, personID)

	if visited[personID] {
		return node
	}
	visited[personID] = true

	if maxGen > 0 && depth >= maxGen {
		return node
	}

	parents := findParents(archive, personID)
	for _, p := range parents {
		child := buildAncestorTree(archive, p.personID, maxGen, depth+1, visited)
		child.RelType = p.relType
		node.Children = append(node.Children, child)
	}

	return node
}

// buildDescendantTree recursively builds the descendant tree for a person.
func buildDescendantTree(archive *glxlib.GLXFile, personID string, maxGen, depth int, visited map[string]bool) *treeNode {
	node := makeTreeNode(archive, personID)

	if visited[personID] {
		return node
	}
	visited[personID] = true

	if maxGen > 0 && depth >= maxGen {
		return node
	}

	children := findChildren(archive, personID)
	for _, c := range children {
		child := buildDescendantTree(archive, c.personID, maxGen, depth+1, visited)
		child.RelType = c.relType
		node.Children = append(node.Children, child)
	}

	return node
}

// relatedPerson holds a person ID and the relationship type connecting them.
type relatedPerson struct {
	personID string
	relType  string
}

// findParents finds all parents of a person by scanning relationships.
func findParents(archive *glxlib.GLXFile, personID string) []relatedPerson {
	var parents []relatedPerson

	for _, rel := range archive.Relationships {
		if !parentChildRelTypes[rel.Type] {
			continue
		}

		isChild := false
		for _, p := range rel.Participants {
			if p.Person == personID && p.Role == "child" {
				isChild = true
				break
			}
		}

		if !isChild {
			continue
		}

		for _, p := range rel.Participants {
			if p.Role == "parent" {
				parents = append(parents, relatedPerson{personID: p.Person, relType: rel.Type})
			}
		}
	}

	sortRelatedPersons(parents)

	return parents
}

// findChildren finds all children of a person by scanning relationships.
func findChildren(archive *glxlib.GLXFile, personID string) []relatedPerson {
	var children []relatedPerson

	for _, rel := range archive.Relationships {
		if !parentChildRelTypes[rel.Type] {
			continue
		}

		isParent := false
		for _, p := range rel.Participants {
			if p.Person == personID && p.Role == "parent" {
				isParent = true
				break
			}
		}

		if !isParent {
			continue
		}

		for _, p := range rel.Participants {
			if p.Role == "child" {
				children = append(children, relatedPerson{personID: p.Person, relType: rel.Type})
			}
		}
	}

	sortRelatedPersons(children)

	return children
}

// sortRelatedPersons sorts related persons by person ID for deterministic output.
func sortRelatedPersons(persons []relatedPerson) {
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
		bornOn := propertyString(person.Properties, "born_on")
		diedOn := propertyString(person.Properties, "died_on")

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
	labels := map[string]string{
		"biological_parent_child": "biological",
		"adoptive_parent_child":   "adoptive",
		"foster_parent_child":     "foster",
		"step_parent":             "step",
	}
	if label, ok := labels[relType]; ok {
		return label
	}

	return relType
}
