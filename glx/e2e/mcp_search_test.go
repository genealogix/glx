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

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// Synthetic identifiers and notes exercise cross-collection discovery without
// incorporating private research data into public regression fixtures.
const mcpSearchFixture = `sources:
  source-example:
    title: Example register
citations:
  citation-example:
    source: source-example
    properties:
      external_id: TEST-RECORD-0042
    notes: Index entry only; original image not examined
research_logs:
  research-example:
    title: Example investigation
    searches:
      - query: Checked TEST-RECORD-0042
        result: found
        citation: citation-example
        notes: Reference located; interpretation remains uncertain
`

func TestMCPStdioSearchParityPaginationErrorsAndNoWrites(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(mcpSearchFixture), 0o600))
	before := snapshotTree(t, dir)
	session, stderr := connectMCP(t, dir)
	var result struct {
		Matches []struct {
			EntityType string `json:"entity_type"`
			EntityID   string `json:"entity_id"`
			Field      string `json:"field"`
			Value      string `json:"value"`
		} `json:"matches"`
		Count    int  `json:"count"`
		Offset   int  `json:"offset"`
		Returned int  `json:"returned"`
		HasMore  bool `json:"has_more"`
	}
	args := map[string]any{"query": "  test-record-0042  "}
	data := callMCP(t, session, "search", args)
	require.NoError(t, json.Unmarshal(data, &result))
	require.Equal(t, 2, result.Count)
	require.Equal(t, 2, result.Returned)
	require.False(t, result.HasMore)
	require.Len(t, result.Matches, 2)
	require.Equal(t, "citation-example", result.Matches[0].EntityID)
	require.Equal(t, "properties.external_id", result.Matches[0].Field)
	require.Equal(t, "research-example", result.Matches[1].EntityID)
	require.Equal(t, "searches[0].query", result.Matches[1].Field)
	cli := runGLX(t, dir, "search", "test-record-0042", "--json")
	require.Zero(t, cli.exitCode, cli.stderr)
	matches, err := json.Marshal(result.Matches)
	require.NoError(t, err)
	require.JSONEq(t, cli.stdout, string(matches), "retain the existing CLI's matching, deduplication and order")
	for _, match := range result.Matches {
		entity := callMCP(t, session, "get_entity", map[string]any{"entity_type": match.EntityType, "id": match.EntityID})
		require.Contains(t, string(entity), "TEST-RECORD-0042")
	}
	require.NoError(t, json.Unmarshal(callMCP(t, session, "search", map[string]any{"query": "TEST-RECORD-0042", "limit": 1}), &result))
	require.Equal(t, 2, result.Count)
	require.Equal(t, 1, result.Returned)
	require.True(t, result.HasMore)
	require.NoError(t, json.Unmarshal(callMCP(t, session, "search", map[string]any{"query": "TEST-RECORD-0042", "limit": 1, "offset": 1}), &result))
	require.Equal(t, 1, result.Offset)
	require.Equal(t, "research-example", result.Matches[0].EntityID)
	require.False(t, result.HasMore)
	for _, input := range []map[string]any{
		{"query": "missing"},
		{"query": "test-record-0042", "case_sensitive": true},
		{"query": "TEST-RECORD-0042", "offset": 100},
	} {
		require.NoError(t, json.Unmarshal(callMCP(t, session, "search", input), &result))
		require.Empty(t, result.Matches)
		require.Zero(t, result.Returned)
		require.False(t, result.HasMore)
	}
	filtered := callMCP(t, session, "search", map[string]any{"query": "TEST-RECORD-0042", "entity_type": "citations"})
	require.NoError(t, json.Unmarshal(filtered, &result))
	require.Equal(t, 1, result.Count)
	require.Equal(t, "citation-example", result.Matches[0].EntityID)
	for _, invalid := range []map[string]any{
		{},
		{"query": 42},
		{"query": nil},
		{"query": " "},
		{"query": "record", "entity_type": "person"},
		{"query": "record", "limit": -1},
		{"query": "record", "limit": 1001},
		{"query": "record", "offset": -1},
		{"query": "record", "case_sensitive": "yes"},
		{"query": "record", "archive": "/etc/passwd"},
		{"query": strings.Repeat("界", 1025)},
	} {
		bad, callErr := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "search", Arguments: invalid})
		require.True(t, callErr != nil || bad.IsError)
	}
	bad, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "search", Arguments: map[string]any{
		"query": "record", "entity_type": strings.Repeat("x", 300000),
	}})
	require.NoError(t, err)
	require.True(t, bad.IsError)
	errorData, err := json.Marshal(bad)
	require.NoError(t, err)
	require.Less(t, len(errorData), 1000, "unknown collection errors must not echo arbitrarily large input")
	callMCP(t, session, "search", args)
	require.NoError(t, session.Close())
	require.Empty(t, stderr.String())
	require.Equal(t, before, snapshotTree(t, dir), "search and CLI parity reads must not create or rewrite archive/cache files")
}
