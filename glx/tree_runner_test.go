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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// threeGenArchive builds a small family: grandparent → parent → child.
func threeGenArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-grandpa": {Properties: map[string]any{"name": "Grandpa Smith", "born_on": "1820"}},
			"person-grandma": {Properties: map[string]any{"name": "Grandma Jones", "born_on": "1825"}},
			"person-father":  {Properties: map[string]any{"name": "Father Smith", "born_on": "1850", "died_on": "1920"}},
			"person-mother":  {Properties: map[string]any{"name": "Mother Brown", "born_on": "1855"}},
			"person-child":   {Properties: map[string]any{"name": "Child Smith", "born_on": "1880"}},
			"person-adopted": {Properties: map[string]any{"name": "Adopted Child", "born_on": "1885"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-gp-father": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-grandpa", Role: "parent"},
					{Person: "person-father", Role: "child"},
				},
			},
			"rel-gm-father": {
				Type: "biological_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-grandma", Role: "parent"},
					{Person: "person-father", Role: "child"},
				},
			},
			"rel-father-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-father", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
			"rel-mother-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-mother", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
			"rel-father-adopted": {
				Type: "adoptive_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-father", Role: "parent"},
					{Person: "person-adopted", Role: "child"},
				},
			},
		},
	}
}

func TestFindParents(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	parents := findParents(tc, "person-child")
	assert.Len(t, parents, 2)

	ids := []string{parents[0].personID, parents[1].personID}
	assert.Contains(t, ids, "person-father")
	assert.Contains(t, ids, "person-mother")
}

func TestFindParents_NoParents(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	parents := findParents(tc, "person-grandpa")
	assert.Empty(t, parents)
}

func TestFindChildren(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	children := findChildren(tc, "person-father")
	assert.Len(t, children, 2)

	ids := []string{children[0].personID, children[1].personID}
	assert.Contains(t, ids, "person-child")
	assert.Contains(t, ids, "person-adopted")
}

func TestFindChildren_NoChildren(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	children := findChildren(tc, "person-child")
	assert.Empty(t, children)
}

func TestFindChildren_AdoptiveRelType(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	children := findChildren(tc, "person-father")
	for _, c := range children {
		if c.personID == "person-adopted" {
			assert.Equal(t, "adoptive_parent_child", c.relType)
			return
		}
	}
	t.Fatal("expected to find adopted child")
}

func TestBuildAncestorTree(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	root := buildAncestorTree(tc, "person-child", 0, 0, make(map[string]bool))

	assert.Equal(t, "person-child", root.PersonID)
	assert.Equal(t, "Child Smith", root.Name)
	assert.Len(t, root.Children, 2) // father, mother

	// Find father node
	var fatherNode *treeNode
	for _, c := range root.Children {
		if c.PersonID == "person-father" {
			fatherNode = c
		}
	}
	require.NotNil(t, fatherNode)
	assert.Len(t, fatherNode.Children, 2) // grandpa, grandma
}

func TestBuildAncestorTree_MaxGen(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	root := buildAncestorTree(tc, "person-child", 1, 0, make(map[string]bool))

	assert.Equal(t, "person-child", root.PersonID)
	assert.Len(t, root.Children, 2) // father, mother

	// Parents should have no children (generation limit reached)
	for _, c := range root.Children {
		assert.Empty(t, c.Children)
	}
}

func TestBuildDescendantTree(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	root := buildDescendantTree(tc, "person-grandpa", 0, 0, make(map[string]bool))

	assert.Equal(t, "person-grandpa", root.PersonID)
	assert.Len(t, root.Children, 1) // father

	fatherNode := root.Children[0]
	assert.Equal(t, "person-father", fatherNode.PersonID)
	assert.Len(t, fatherNode.Children, 2) // child, adopted
}

func TestBuildDescendantTree_MaxGen(t *testing.T) {
	archive := threeGenArchive()
	tc := newTreeContext(archive)

	root := buildDescendantTree(tc, "person-grandpa", 1, 0, make(map[string]bool))

	assert.Len(t, root.Children, 1)
	assert.Empty(t, root.Children[0].Children) // generation limit
}

func TestBuildAncestorTree_CycleDetection(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-a", Role: "parent"},
					{Person: "person-b", Role: "child"},
				},
			},
			"rel-2": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-b", Role: "parent"},
					{Person: "person-a", Role: "child"},
				},
			},
		},
	}

	tc := newTreeContext(archive)

	// Should terminate without infinite loop
	root := buildAncestorTree(tc, "person-a", 0, 0, make(map[string]bool))
	assert.Equal(t, "person-a", root.PersonID)
}

func TestMakeTreeNode(t *testing.T) {
	archive := threeGenArchive()

	node := makeTreeNode(archive, "person-father")
	assert.Equal(t, "Father Smith", node.Name)
	assert.Equal(t, "(1850 – 1920)", node.Dates)

	node = makeTreeNode(archive, "person-grandpa")
	assert.Equal(t, "Grandpa Smith", node.Name)
	assert.Equal(t, "(b. 1820)", node.Dates)
}

func TestMakeTreeNode_UnknownPerson(t *testing.T) {
	archive := threeGenArchive()

	node := makeTreeNode(archive, "person-nonexistent")
	assert.Equal(t, "(unknown)", node.Name)
	assert.Equal(t, "", node.Dates)
}

func TestFormatRelType(t *testing.T) {
	assert.Equal(t, "biological", formatRelType("biological_parent_child"))
	assert.Equal(t, "adoptive", formatRelType("adoptive_parent_child"))
	assert.Equal(t, "foster", formatRelType("foster_parent_child"))
	assert.Equal(t, "step", formatRelType("step_parent"))
	assert.Equal(t, "parent_child", formatRelType("parent_child"))
}

func TestShowAncestors_CompleteFamily(t *testing.T) {
	err := showAncestors("../docs/examples/complete-family", "person-jane-smith-1876", 0)
	require.NoError(t, err)
}

func TestShowDescendants_CompleteFamily(t *testing.T) {
	err := showDescendants("../docs/examples/complete-family", "person-john-smith-1850", 0)
	require.NoError(t, err)
}

func TestShowAncestors_PersonNotFound(t *testing.T) {
	err := showAncestors("../docs/examples/complete-family", "person-nonexistent", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestShowDescendants_PersonNotFound(t *testing.T) {
	err := showDescendants("../docs/examples/complete-family", "person-nonexistent", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindParents_IgnoresNonParentChild(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "A"}},
			"person-b": {Properties: map[string]any{"name": "B"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-a", Role: "spouse"},
					{Person: "person-b", Role: "spouse"},
				},
			},
		},
	}

	tc := newTreeContext(archive)
	parents := findParents(tc, "person-a")
	assert.Empty(t, parents)
}

func TestFormatNodeLabel(t *testing.T) {
	node := &treeNode{
		PersonID: "person-abc",
		Name:     "John Smith",
		Dates:    "(b. 1850)",
	}
	assert.Equal(t, "John Smith  (b. 1850)  person-abc", formatNodeLabel(node))

	node.Dates = ""
	assert.Equal(t, "John Smith  person-abc", formatNodeLabel(node))
}
