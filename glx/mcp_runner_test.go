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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

func TestMCPQueryPagination(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}}
	for i := range 125 {
		archive.Persons[fmt.Sprintf("person-%03d", i)] = &glxlib.Person{Properties: map[string]any{"name": "Person"}}
	}
	backend := &mcpArchiveBackend{archive: archive}
	data, err := backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons"}`))
	require.NoError(t, err)
	result, ok := data.(*mcpQueryResult)
	require.True(t, ok)
	require.Equal(t, 125, result.Count)
	require.Equal(t, 100, result.Returned)
	require.True(t, result.HasMore)
	data, err = backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons","offset":100}`))
	require.NoError(t, err)
	result, ok = data.(*mcpQueryResult)
	require.True(t, ok)
	require.Equal(t, 25, result.Returned)
	require.False(t, result.HasMore)
	require.Equal(t, "person-100", result.Entities[0].ID)
	for _, raw := range []string{`{"entity_type":"persons","offset":-1}`, `{"entity_type":"persons","limit":1001}`} {
		_, err = backend.CallTool(t.Context(), "query", json.RawMessage(raw))
		require.Error(t, err)
	}
}

func TestMCPValidationRetainsDuplicatesAndCaches(t *testing.T) {
	backend := &mcpArchiveBackend{archive: &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "P"}}}}, duplicates: []string{"duplicate person p"}}
	data, err := backend.CallTool(t.Context(), "validate", json.RawMessage(`{}`))
	require.NoError(t, err)
	result, ok := data.(*validationJSONResult)
	require.True(t, ok)
	require.False(t, result.Valid)
	require.Equal(t, backend.duplicates, result.DuplicateIDs)
	require.Nil(t, result.Report)
	require.NotNil(t, backend.validation)
	cached := backend.validation
	_, err = backend.CallTool(t.Context(), "validate", json.RawMessage(`{"report":true}`))
	require.NoError(t, err)
	require.Same(t, cached, backend.validation)
}

func TestMCPTreeBoundsPedigreeCollapse(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}, Relationships: map[string]*glxlib.Relationship{}}
	// Two shared parents at every level create exponentially many paths from
	// only forty people. The protocol must bound nodes, not only depth.
	for level := range 20 {
		for side := range 2 {
			id := fmt.Sprintf("p-%02d-%d", level, side)
			archive.Persons[id] = &glxlib.Person{Properties: map[string]any{"name": id}}
			if level == 0 {
				continue
			}
			for parentSide := range 2 {
				childID := fmt.Sprintf("p-%02d-%d", level-1, parentSide)
				archive.Relationships[id+childID] = &glxlib.Relationship{Type: "parent_child", Participants: []glxlib.Participant{{Person: id, Role: "parent"}, {Person: childID, Role: "child"}}}
			}
		}
	}
	backend := &mcpArchiveBackend{archive: archive}
	result, err := backend.tree(t.Context(), glxmcp.TreeArgs{Person: "p-00-0", Generations: 20}, true)
	require.NoError(t, err)
	require.Equal(t, mcpTreeMaxNodes, result.NodeCount)
	require.True(t, result.Truncated)
	defaultTree, err := backend.tree(t.Context(), glxmcp.TreeArgs{Person: "p-00-0"}, true)
	require.NoError(t, err)
	require.Equal(t, mcpTreeDefaultGenerations, defaultTree.Generations)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = backend.tree(ctx, glxmcp.TreeArgs{Person: "p-00-0"}, true)
	require.ErrorIs(t, err, context.Canceled)
}
