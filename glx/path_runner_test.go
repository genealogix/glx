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
	"os"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestArchiveForPath creates a test archive with a family network:
//
//	grandparent ── parent_child ──> parent
//	parent ── parent_child ──> child
//	parent ── marriage ──> spouse
//	spouse ── parent_child ──> child
//	neighbor ── neighbor ──> parent
func newTestArchiveForPath() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-grandparent": {Properties: map[string]any{"name": "Grandparent Smith"}},
			"person-parent":      {Properties: map[string]any{"name": "Parent Smith"}},
			"person-spouse":      {Properties: map[string]any{"name": "Spouse Jones"}},
			"person-child":       {Properties: map[string]any{"name": "Child Smith"}},
			"person-neighbor":    {Properties: map[string]any{"name": "Neighbor Brown"}},
			"person-isolated":    {Properties: map[string]any{"name": "Isolated Person"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-gp-parent": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-grandparent", Role: "parent"},
					{Person: "person-parent", Role: "child"},
				},
			},
			"rel-parent-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
			"rel-marriage": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "spouse"},
					{Person: "person-spouse", Role: "spouse"},
				},
			},
			"rel-spouse-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-spouse", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
			"rel-neighbor": {
				Type: "neighbor",
				Participants: []glxlib.Participant{
					{Person: "person-neighbor", Role: "principal"},
					{Person: "person-parent", Role: "principal"},
				},
			},
		},
		Events:     map[string]*glxlib.Event{},
		Sources:    map[string]*glxlib.Source{},
		Citations:  map[string]*glxlib.Citation{},
		Assertions: map[string]*glxlib.Assertion{},
		Places:     map[string]*glxlib.Place{},
	}
}

func TestBfsPath_DirectConnection(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	path := bfsPath("person-grandparent", "person-parent", adj, 10)

	require.NotNil(t, path)
	assert.Len(t, path, 2) // grandparent -> parent
	assert.Equal(t, "person-grandparent", path[0].PersonID)
	assert.Equal(t, "person-parent", path[1].PersonID)
}

func TestBfsPath_TwoHops(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	path := bfsPath("person-grandparent", "person-child", adj, 10)

	require.NotNil(t, path)
	assert.Len(t, path, 3) // grandparent -> parent -> child
	assert.Equal(t, "person-grandparent", path[0].PersonID)
	assert.Equal(t, "person-parent", path[1].PersonID)
	assert.Equal(t, "person-child", path[2].PersonID)
}

func TestBfsPath_ViaMarriage(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	path := bfsPath("person-grandparent", "person-spouse", adj, 10)

	require.NotNil(t, path)
	assert.Len(t, path, 3) // grandparent -> parent -> spouse
	assert.Equal(t, "person-grandparent", path[0].PersonID)
	assert.Equal(t, "person-parent", path[1].PersonID)
	assert.Equal(t, "person-spouse", path[2].PersonID)
}

func TestBfsPath_NoPath(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	path := bfsPath("person-grandparent", "person-isolated", adj, 10)

	assert.Nil(t, path)
}

func TestBfsPath_MaxHopsExceeded(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	// grandparent -> parent -> child = 2 hops; max 1 should fail
	path := bfsPath("person-grandparent", "person-child", adj, 1)

	assert.Nil(t, path)
}

func TestBfsPath_MaxHopsExact(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	// grandparent -> parent -> child = 2 hops; max 2 should succeed
	path := bfsPath("person-grandparent", "person-child", adj, 2)

	require.NotNil(t, path)
	assert.Len(t, path, 3)
}

func TestBfsPath_NeighborRelationship(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	path := bfsPath("person-neighbor", "person-parent", adj, 10)

	require.NotNil(t, path)
	assert.Len(t, path, 2)
	assert.Equal(t, "person-neighbor", path[0].PersonID)
	assert.Equal(t, "person-parent", path[1].PersonID)
	assert.Equal(t, "neighbor", path[1].Edge.RelType)
}

func TestBfsPath_ShortestPath(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	// neighbor -> parent -> child (2 hops via parent_child)
	// OR neighbor -> parent -> spouse -> child (3 hops)
	// BFS should find the 2-hop path
	path := bfsPath("person-neighbor", "person-child", adj, 10)

	require.NotNil(t, path)
	assert.Len(t, path, 3) // neighbor -> parent -> child
}

func TestBuildPathAdjacency(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)

	// Grandparent should connect to parent
	edges := adj["person-grandparent"]
	assert.NotEmpty(t, edges)

	found := false
	for _, e := range edges {
		if e.PersonID == "person-parent" {
			found = true
			assert.Equal(t, "parent_child", e.RelType)
		}
	}
	assert.True(t, found, "grandparent should have edge to parent")

	// Parent should connect to grandparent, child, and spouse
	parentEdges := adj["person-parent"]
	connectedTo := make(map[string]bool)
	for _, e := range parentEdges {
		connectedTo[e.PersonID] = true
	}
	assert.True(t, connectedTo["person-grandparent"])
	assert.True(t, connectedTo["person-child"])
	assert.True(t, connectedTo["person-spouse"])
	assert.True(t, connectedTo["person-neighbor"])

	// Isolated person should have no edges
	assert.Empty(t, adj["person-isolated"])
}

func TestBuildPathResult_Found(t *testing.T) {
	archive := newTestArchiveForPath()
	adj := buildPathAdjacency(archive)
	path := bfsPath("person-grandparent", "person-child", adj, 10)

	result := buildPathResult("person-grandparent", "person-child", path, archive)

	assert.Equal(t, "person-grandparent", result.From)
	assert.Equal(t, "person-child", result.To)
	assert.Equal(t, 2, result.Hops)
	assert.Len(t, result.Path, 3)
	assert.Equal(t, "", result.Message)
	assert.Equal(t, "Grandparent Smith", result.Path[0].PersonName)
	assert.Equal(t, "Child Smith", result.Path[2].PersonName)
}

func TestBuildPathResult_NotFound(t *testing.T) {
	archive := newTestArchiveForPath()

	result := buildPathResult("person-grandparent", "person-isolated", nil, archive)

	assert.Equal(t, "No path found", result.Message)
	assert.Equal(t, 0, result.Hops)
	assert.Empty(t, result.Path)
}

func TestResolvePersonForPath_ExactID(t *testing.T) {
	archive := newTestArchiveForPath()

	id, err := resolvePersonForPath(archive, "person-parent")
	require.NoError(t, err)
	assert.Equal(t, "person-parent", id)
}

func TestResolvePersonForPath_NameSearch(t *testing.T) {
	archive := newTestArchiveForPath()

	id, err := resolvePersonForPath(archive, "Neighbor")
	require.NoError(t, err)
	assert.Equal(t, "person-neighbor", id)
}

func TestResolvePersonForPath_NotFound(t *testing.T) {
	archive := newTestArchiveForPath()

	_, err := resolvePersonForPath(archive, "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestResolvePersonForPath_Ambiguous(t *testing.T) {
	archive := newTestArchiveForPath()

	// "Smith" matches grandparent, parent, and child
	_, err := resolvePersonForPath(archive, "Smith")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple persons match")
}

func TestFormatRelLabel(t *testing.T) {
	assert.Equal(t, "child in parent child", formatRelLabel("parent_child", "child"))
	assert.Equal(t, "spouse in marriage", formatRelLabel("marriage", "spouse"))
	assert.Equal(t, "neighbor", formatRelLabel("neighbor", ""))
}

func TestPathPersonName(t *testing.T) {
	archive := newTestArchiveForPath()

	assert.Equal(t, "Parent Smith", pathPersonName(archive, "person-parent"))
	assert.Equal(t, "unknown-id", pathPersonName(archive, "unknown-id"))
}

// Integration tests using the complete-family example archive.

func TestShowPath_CompleteFamily(t *testing.T) {
	// Jane is child of John; should find 1-hop path
	output := capturePathStdout(t, func() {
		err := showPath("../docs/examples/complete-family", "person-jane-smith-1876", "person-john-smith-1850", 10, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "1 hop(s)")
	assert.Contains(t, output, "Jane Smith")
	assert.Contains(t, output, "John Smith")
}

func TestShowPath_CompleteFamily_JSON(t *testing.T) {
	output := capturePathStdout(t, func() {
		err := showPath("../docs/examples/complete-family", "person-jane-smith-1876", "person-mary-brown-1852", 10, true)
		require.NoError(t, err)
	})

	assert.True(t, strings.HasPrefix(strings.TrimSpace(output), "{"), "JSON output should start with {")
	assert.Contains(t, output, `"from"`)
	assert.Contains(t, output, `"to"`)
	assert.Contains(t, output, `"hops"`)
	assert.Contains(t, output, `"path"`)
}

func TestShowPath_CompleteFamily_ViaMarriage(t *testing.T) {
	output := capturePathStdout(t, func() {
		err := showPath("../docs/examples/complete-family", "person-john-smith-1850", "person-mary-brown-1852", 10, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "1 hop(s)")
	assert.Contains(t, output, "John Smith")
	assert.Contains(t, output, "Mary Brown")
}

func TestShowPath_PersonNotFound(t *testing.T) {
	err := showPath("../docs/examples/complete-family", "person-nonexistent", "person-john-smith-1850", 10, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestShowPath_ArchiveNotFound(t *testing.T) {
	err := showPath("/nonexistent/path", "a", "b", 10, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestShowPath_SamePerson(t *testing.T) {
	err := showPath("../docs/examples/complete-family", "person-john-smith-1850", "person-john-smith-1850", 10, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same person")
}

func TestShowPath_InvalidMaxHops(t *testing.T) {
	err := showPath("../docs/examples/complete-family", "person-john-smith-1850", "person-jane-smith-1876", 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--max-hops must be at least 1")

	err = showPath("../docs/examples/complete-family", "person-john-smith-1850", "person-jane-smith-1876", -5, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--max-hops must be at least 1")
}

// capturePathStdout redirects os.Stdout during fn execution and returns what was written.
func capturePathStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	origStdout := os.Stdout
	os.Stdout = w

	wClosed := false
	defer func() {
		if !wClosed {
			_ = w.Close()
		}
		os.Stdout = origStdout
	}()

	fn()

	require.NoError(t, w.Close())
	wClosed = true

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)

	return buf.String()
}
