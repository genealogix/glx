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
	"context"
	"fmt"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// treeNode represents a person in the ancestor/descendant tree.
type treeNode struct {
	PersonID     string      `json:"person_id"`
	Name         string      `json:"name"`
	Dates        string      `json:"dates"`        // "(1850 – 1920)" or similar
	RelType      string      `json:"rel_type"`     // relationship type connecting to parent/child
	Step         bool        `json:"step"`         // connected to parent/child through a step link
	Hypothetical bool        `json:"hypothetical"` // the connecting relationship is only a hypothesis
	Children     []*treeNode `json:"children"`
}

// treeRelPerson holds a person ID and the relationship connecting them.
type treeRelPerson struct {
	personID     string
	relType      string
	step         bool
	hypothetical bool
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
	tree, _ := newTreeContextContext(context.Background(), archive)

	return tree
}

// newTreeContextContext scans all relationships once and indexes both
// directions: child ID to parents and parent ID to children, the children in
// birth order as every other view reads them (familyLinks.childLinks). Links
// the archive has disproven are left out, and two relationships recording the
// same link are drawn once (#1506). Canceled calls return no partial context.
func newTreeContextContext(ctx context.Context, archive *glxlib.GLXFile) (*treeContext, error) {
	links, err := glxlib.ParentChildLinksContext(ctx, archive, nil)
	if err != nil {
		return nil, err
	}
	tree := &treeContext{archive: archive, parents: make(map[string][]treeRelPerson), children: make(map[string][]treeRelPerson)}
	childLinks := make(map[string][]glxlib.ParentChildLink)
	for _, link := range links {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if link.Standing == glxlib.RelationshipStandingDisproven {
			continue
		}
		tree.parents[link.ChildID] = append(tree.parents[link.ChildID], treeRelPersonFromLink(&link, link.ParentID))
		childLinks[link.ParentID] = append(childLinks[link.ParentID], link)
	}
	for id := range tree.parents {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		tree.parents[id] = dedupeRelatedPersons(tree.parents[id])
		sortRelatedPersons(tree.parents[id])
	}
	if len(childLinks) > 0 {
		order := glxlib.NewBirthOrder(archive)
		for parentID, kids := range childLinks {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			glxlib.SortByBirth(order, kids, func(l glxlib.ParentChildLink) string { return l.ChildID })
			for i := range kids {
				tree.children[parentID] = append(tree.children[parentID], treeRelPersonFromLink(&kids[i], kids[i].ChildID))
			}
			tree.children[parentID] = dedupeRelatedPersons(tree.children[parentID])
		}
	}

	return tree, ctx.Err()
}

// treeRelPersonFromLink converts a parent-child link into the tree's edge
// record for the person at the far end.
func treeRelPersonFromLink(link *glxlib.ParentChildLink, personID string) treeRelPerson {
	return treeRelPerson{
		personID:     personID,
		relType:      link.RelationshipType,
		step:         link.Step,
		hypothetical: link.Standing == glxlib.RelationshipStandingHypothetical,
	}
}

// findParents returns the parents of a person using the prebuilt index.
func findParents(tc *treeContext, personID string) []treeRelPerson {
	return tc.parents[personID]
}

// findChildren returns the children of a person using the prebuilt index.
func findChildren(tc *treeContext, personID string) []treeRelPerson {
	return tc.children[personID]
}

// showAncestors loads the archive and prints the ancestor tree for a person.
// When ancestors are missing, it appends research suggestions.
func showAncestors(archivePath, personID string, maxGen int, jsonOutput ...bool) error {
	archive, err := loadArchiveForRead(SystemIOStreams(), archivePath)
	if err != nil {
		return err
	}
	if explorationJSONEnabled(jsonOutput) {
		return printExplorationJSON(buildTreeJSON(archive, personID, maxGen, true))
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
func showDescendants(archivePath, personID string, maxGen int, jsonOutput ...bool) error {
	archive, err := loadArchiveForRead(SystemIOStreams(), archivePath)
	if err != nil {
		return err
	}
	if explorationJSONEnabled(jsonOutput) {
		return printExplorationJSON(buildTreeJSON(archive, personID, maxGen, false))
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
		child.Step = p.step
		child.Hypothetical = p.hypothetical
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
		child.Step = c.step
		child.Hypothetical = c.hypothetical
		node.Children = append(node.Children, child)
	}

	return node
}

// dedupeRelatedPersons collapses edges that link the same person in the same
// way, as two relationships recording one parent-child link do (#1506), so
// the tree draws that person once. Two edges are the same link when they share
// the step flag and their types overlap (glxlib.ParentChildTypesOverlap): the
// same type, or the generic parent_child beside a specific type. Two specific
// types (biological and adoptive) are different facts and both stay. When the
// copies differ in standing, the accepted one is kept over a hypothesis, type
// included; when they tie, the specific type is kept (see mergeRelatedPerson).
func dedupeRelatedPersons(persons []treeRelPerson) []treeRelPerson {
	type edgeKey struct {
		personID string
		step     bool
	}
	out := persons[:0]
	at := make(map[edgeKey][]int, len(persons))
	for _, p := range persons {
		key := edgeKey{p.personID, p.step}
		merged := false
		for _, i := range at[key] {
			if !glxlib.ParentChildTypesOverlap(out[i].relType, p.relType) {
				continue
			}
			out[i] = mergeRelatedPerson(out[i], p)
			merged = true

			break
		}
		if merged {
			continue
		}
		at[key] = append(at[key], len(out))
		out = append(out, p)
	}

	return out
}

// mergeRelatedPerson combines two edges for the same link. The edge with
// the stronger standing wins, type included, so a hypothetical specific type
// never shows as an accepted one beside an accepted generic link. When the
// standings tie, the specific type is kept over the generic parent_child.
func mergeRelatedPerson(kept, other treeRelPerson) treeRelPerson {
	switch {
	case kept.hypothetical && !other.hypothetical:
		return other
	case !kept.hypothetical && other.hypothetical:
		return kept
	}
	if strings.EqualFold(kept.relType, glxlib.RelationshipTypeParentChild) {
		kept.relType = other.relType
	}

	return kept
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
		termPrintln(label)
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
		switch {
		case child.Step:
			label += "  [" + formatRelType(glxlib.RelationshipTypeStepParent) + "]"
		case child.RelType != "" && child.RelType != glxlib.RelationshipTypeParentChild:
			label += fmt.Sprintf("  [%s]", formatRelType(child.RelType))
		}
		if child.Hypothetical {
			label += "  " + hypotheticalMarker
		}

		termPrintf("%s%s%s\n", prefix, connector, label)
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
