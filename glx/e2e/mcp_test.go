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
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func connectMCP(t *testing.T, dir string, args ...string) (*mcpsdk.ClientSession, *bytes.Buffer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(t.Context()), 30*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(ctx, glxBinary, append([]string{"--quiet", "mcp", "serve"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(envWithout(os.Environ(), "GLX_CACHE"), "GLX_CACHE=auto")
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "glx-e2e", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, session.Close()) })

	return session, stderr
}

func callMCP(t *testing.T, session *mcpsdk.ClientSession, name string, args any) json.RawMessage {
	t.Helper()
	result, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err, name)
	require.False(t, result.IsError, "%s: %+v", name, result.Content)
	require.NotNil(t, result.StructuredContent, name)
	data, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	require.True(t, json.Valid(data))

	return data
}

func TestMCPRealStdioAllToolsAndNoWrites(t *testing.T) {
	dir := copyExample(t, "complete-family")
	before := snapshotTree(t, dir)
	session, stderr := connectMCP(t, dir)
	catalog, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, catalog.Tools, 16)
	const person = "person-jane-smith-1876"
	calls := map[string]any{
		"search":     map[string]any{"query": "Jane"},
		"query":      map[string]any{"entity_type": "persons", "name": "Jane"},
		"get_entity": map[string]any{"entity_type": "persons", "id": person},
		"vitals":     map[string]any{"person": person}, "summary": map[string]any{"person": person},
		"timeline": map[string]any{"person": person}, "ancestors": map[string]any{"person": person, "generations": 3},
		"descendants": map[string]any{"person": "person-john-smith-1850", "generations": 3},
		"path":        map[string]any{"from": person, "to": "person-john-smith-1850"},
		"cite":        map[string]any{"all": true}, "analyze": map[string]any{"person": person, "check": "gaps"},
		"stats": map[string]any{}, "places": map[string]any{}, "cluster": map[string]any{"person": person},
		"validate": map[string]any{"report": true}, "list_entity_types": map[string]any{},
	}
	for name, args := range calls {
		callMCP(t, session, name, args)
	}
	for _, collection := range []string{"persons", "events", "relationships", "places", "sources", "citations", "repositories", "media", "assertions", "research_logs", "studies"} {
		callMCP(t, session, "query", map[string]any{"entity_type": collection})
	}
	for _, check := range []string{"gaps", "evidence", "consistency", "conflicts", "suggestions"} {
		callMCP(t, session, "analyze", map[string]any{"check": check})
	}
	for _, uri := range []string{"glx://archive/stats", "glx://archive/vocabularies"} {
		resource, readErr := session.ReadResource(t.Context(), &mcpsdk.ReadResourceParams{URI: uri})
		require.NoError(t, readErr)
		require.True(t, json.Valid([]byte(resource.Contents[0].Text)))
	}
	_, err = session.GetPrompt(t.Context(), &mcpsdk.GetPromptParams{Name: "genealogy-research-assistant"})
	require.NoError(t, err)
	bad, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "vitals", Arguments: map[string]any{"person": "missing-person"}})
	require.NoError(t, err)
	require.True(t, bad.IsError)
	callMCP(t, session, "stats", map[string]any{})
	require.NoError(t, session.Close())
	require.Empty(t, stderr.String())
	require.Equal(t, before, snapshotTree(t, dir), "even GLX_CACHE=auto must not write during MCP serving")
}

func TestMCPSnapshotAndSingleFile(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(archive, []byte("persons:\n  person-first:\n    properties:\n      name: First\n"), 0o600))
	session, _ := connectMCP(t, dir, archive)
	initial := callMCP(t, session, "list_entity_types", map[string]any{})
	initialSearch := callMCP(t, session, "search", map[string]any{"query": "Second"})
	require.NoError(t, os.WriteFile(archive, []byte("persons:\n  person-first:\n    properties:\n      name: First\n  person-second:\n    properties:\n      name: Second\n"), 0o600))
	require.JSONEq(t, string(initial), string(callMCP(t, session, "list_entity_types", map[string]any{})))
	require.JSONEq(t, string(initialSearch), string(callMCP(t, session, "search", map[string]any{"query": "Second"})))
	require.NoError(t, session.Close())
	restarted, _ := connectMCP(t, dir, archive)
	var result struct {
		Counts map[string]int `json:"entity_counts"`
	}
	require.NoError(t, json.Unmarshal(callMCP(t, restarted, "list_entity_types", map[string]any{}), &result))
	require.Equal(t, 2, result.Counts["persons"])
	require.Contains(t, string(callMCP(t, restarted, "search", map[string]any{"query": "Second"})), `"count":1`)
}

func TestMCPUsesExistingCacheWithoutChangingIt(t *testing.T) {
	dir := copyExample(t, "complete-family")
	built := runGLX(t, dir, "cache", "build")
	require.Zero(t, built.exitCode, built.stderr)
	before := snapshotTree(t, dir)
	session, _ := connectMCP(t, dir)
	callMCP(t, session, "stats", map[string]any{})
	require.NoError(t, session.Close())
	require.Equal(t, before, snapshotTree(t, dir))
}

func TestResearchCommandsJSON(t *testing.T) {
	dir := copyExample(t, "complete-family")
	for _, args := range [][]string{
		{"query", "persons", "--json"},
		{"vitals", "Jane", "--json"},
		{"timeline", "Jane", "--json"},
		{"summary", "Jane", "--json"},
		{"ancestors", "person-jane-smith-1876", "--json"},
		{"descendants", "person-john-smith-1850", "--json"},
		{"cite", "--json"},
		{"analyze", "--json"},
		{"stats", "--json"},
		{"places", "--json"},
		{"validate", "--json"},
		{"validate", "--report", "--json"},
	} {
		t.Run(args[0]+args[len(args)-2], func(t *testing.T) {
			result := runGLX(t, dir, append([]string{"--quiet"}, args...)...)
			require.Zero(t, result.exitCode, result.stderr)
			require.True(t, json.Valid([]byte(result.stdout)), result.stdout)
		})
	}
}

func TestMCPInvalidEntitiesAndDuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "first.glx"), []byte("persons:\n  person-first:\n    properties:\n      name: First\nevents:\n  event-null: null\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "duplicate.glx"), []byte("persons:\n  person-first:\n    properties:\n      name: Duplicate\n"), 0o600))
	session, _ := connectMCP(t, dir)
	for _, name := range []string{"vitals", "timeline", "summary", "cluster"} {
		// Invalid data may produce a tool error but must not terminate stdio.
		_, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: name, Arguments: map[string]any{"person": "person-first"}})
		require.NoError(t, err, name)
	}
	var result struct {
		Valid      bool     `json:"valid"`
		Errors     []any    `json:"errors"`
		Duplicates []string `json:"duplicate_ids"`
	}
	require.NoError(t, json.Unmarshal(callMCP(t, session, "validate", map[string]any{}), &result))
	require.False(t, result.Valid)
	require.NotEmpty(t, result.Errors)
	require.NotEmpty(t, result.Duplicates)
	callMCP(t, session, "stats", map[string]any{})
}

func TestMCPPathAlternateAndAmbiguousNames(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "archive.glx")
	data := `persons:
  person-alpha:
    properties:
      name:
        - value: Alpha Original
        - value: Historical Alias
        - value: Shared Name
  person-beta:
    properties:
      name: Shared Name
  person-target:
    properties:
      name:
        - value: Target Original
        - value: Destination Alias
relationships:
  relationship-alpha:
    type: parent_child
    participants:
      - person: person-alpha
        role: parent
      - person: person-target
        role: child
  relationship-beta:
    type: parent_child
    participants:
      - person: person-beta
        role: parent
      - person: person-target
        role: child
`
	require.NoError(t, os.WriteFile(archive, []byte(data), 0o600))
	session, _ := connectMCP(t, dir, archive)
	cases := []struct {
		name, from, to, wantFrom, wantTo string
		ambiguous                        bool
	}{
		{name: "alternate source", from: "Historical Alias", to: "person-target", wantFrom: "person-alpha", wantTo: "person-target"},
		{name: "alternate destination", from: "person-alpha", to: "Destination Alias", wantFrom: "person-alpha", wantTo: "person-target"},
		{name: "both alternate", from: "Historical Alias", to: "Destination Alias", wantFrom: "person-alpha", wantTo: "person-target"},
		{name: "ambiguous source", from: "Shared Name", to: "person-target", ambiguous: true},
		{name: "ambiguous destination", from: "person-target", to: "Shared Name", ambiguous: true},
		{name: "exact ID disambiguates", from: "person-beta", to: "person-target", wantFrom: "person-beta", wantTo: "person-target"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "path", Arguments: map[string]any{"from": tc.from, "to": tc.to}})
			require.NoError(t, err)
			if tc.ambiguous {
				require.True(t, result.IsError, "an alternate-name collision must not silently select a person")
				content, marshalErr := json.Marshal(result.Content)
				require.NoError(t, marshalErr)
				require.Contains(t, string(content), "multiple persons match")
				require.Contains(t, string(content), "person-alpha")
				require.Contains(t, string(content), "person-beta")

				return
			}
			require.False(t, result.IsError, "%+v", result.Content)
			content, err := json.Marshal(result.StructuredContent)
			require.NoError(t, err)
			var path struct {
				From string `json:"from"`
				To   string `json:"to"`
				Hops int    `json:"hops"`
			}
			require.NoError(t, json.Unmarshal(content, &path))
			require.Equal(t, tc.wantFrom, path.From)
			require.Equal(t, tc.wantTo, path.To)
			require.Equal(t, 1, path.Hops)
		})
	}
	callMCP(t, session, "stats", map[string]any{})
}
