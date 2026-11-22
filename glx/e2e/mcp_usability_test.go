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

// Entirely synthetic: a declaration and later index entry must stay distinct,
// and an informant's association with a death must not become their own death.
const mcpUsabilityFixture = `persons:
  person-mary:
    properties:
      name: Mary Green
  person-robin:
    properties:
      name: Robin Green
events:
  event-z-declaration:
    type: naturalization
    title: Declaration register
    date: "1880-04-05"
    properties:
      event_subtype: Declaration of intention
    participants:
      - person: person-mary
        role: principal
  event-a-final:
    type: naturalization
    title: Final admission index; original not located
    date: "1910-06-07"
    participants:
      - person: person-mary
        role: principal
  event-robin-death:
    type: death
    title: Robin's death record
    date: "1930-08-09"
    participants:
      - person: person-robin
        role: principal
      - person: person-mary
        role: informant
assertions:
  assertion-cause-a:
    subject:
      event: event-robin-death
    property: cause
    value: First reported cause
  assertion-cause-b:
    subject:
      event: event-robin-death
    property: cause
    value: Different reported cause
research_logs:
  research-log-mary:
    title: Origins investigation
    subject:
      person: person-mary
    status: in_progress
    searches:
      - result: not_found
        notes: Original register unavailable; index searched only
    notes: LOG_BODY_PLACEHOLDER
`

func TestMCPStdioCompactSummaryAndExactRetrieval(t *testing.T) {
	dir := t.TempDir()
	longNotes := strings.TrimSpace(strings.Repeat("Source caveat retained in full. ", 1000))
	fixture := strings.ReplaceAll(mcpUsabilityFixture, "LOG_BODY_PLACEHOLDER", longNotes)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(fixture), 0o600))
	before := snapshotTree(t, dir)
	session, stderr := connectMCP(t, dir, "--watch")
	args := map[string]any{"person": "person-mary"}
	compact := callMCP(t, session, "summary", args)
	var profile struct {
		ResearchLogs         []json.RawMessage `json:"research_logs"`
		ResearchLogCount     int               `json:"research_log_count"`
		ResearchLogsIncluded bool              `json:"research_logs_included"`
		ResearchLogRefs      []struct {
			ID string `json:"id"`
		} `json:"research_log_refs"`
		Narrative string `json:"narrative"`
		Timeline  []struct {
			EventID      string `json:"event_id"`
			EventType    string `json:"event_type"`
			EventSubtype string `json:"event_subtype"`
			Title        string `json:"title"`
			Label        string `json:"label"`
		} `json:"timeline"`
	}
	require.NoError(t, json.Unmarshal(compact, &profile))
	require.Equal(t, 1, profile.ResearchLogCount)
	require.False(t, profile.ResearchLogsIncluded)
	require.Empty(t, profile.ResearchLogs)
	require.Len(t, profile.ResearchLogRefs, 1)
	require.NotContains(t, string(compact), "Source caveat retained in full")
	require.Contains(t, profile.Narrative, "Declaration of intention")
	require.Contains(t, profile.Narrative, "Final admission index; original not located")
	require.Less(t, strings.Index(profile.Narrative, "1880"), strings.Index(profile.Narrative, "1910"), "proceedings must be chronological, not sorted by ID")
	require.NotContains(t, profile.Narrative, "died")
	require.Len(t, profile.Timeline, 3)
	require.Equal(t, "event-z-declaration", profile.Timeline[0].EventID)
	require.Equal(t, "naturalization", profile.Timeline[0].EventType)
	require.Equal(t, "Declaration of intention", profile.Timeline[0].EventSubtype)
	require.Equal(t, "Declaration register", profile.Timeline[0].Title)
	require.Contains(t, profile.Timeline[0].Label, "Declaration of intention")
	log := callMCP(t, session, "get_entity", map[string]any{"entity_type": "research_logs", "id": profile.ResearchLogRefs[0].ID})
	require.Contains(t, string(log), longNotes)
	require.Contains(t, string(log), "Original register unavailable; index searched only")
	event := callMCP(t, session, "get_entity", map[string]any{"entity_type": "events", "id": profile.Timeline[0].EventID})
	require.Contains(t, string(event), `"event_subtype":"Declaration of intention"`)
	full := callMCP(t, session, "summary", map[string]any{"person": "person-mary", "include_research_logs": true})
	require.Contains(t, string(full), longNotes)
	require.Contains(t, string(full), `"research_logs_included":true`)
	legacy := runGLX(t, dir, "summary", "person-mary", "--json")
	require.Zero(t, legacy.exitCode, legacy.stderr)
	require.Contains(t, legacy.stdout, longNotes, "CLI compatibility")
	for _, invalid := range []map[string]any{
		{"id": "research-log-mary"},
		{"entity_type": "research_logs"},
		{"entity_type": "research_logs", "id": 42},
		{"entity_type": "events", "id": "research-log-mary"},
		{"entity_type": "research_logs", "id": "RESEARCH-LOG-MARY"},
		{"entity_type": "events", "id": "event-z-declaration", "archive": "/etc/passwd"},
	} {
		bad, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "get_entity", Arguments: invalid})
		require.True(t, err != nil || bad.IsError)
	}
	callMCP(t, session, "get_entity", map[string]any{"entity_type": "events", "id": "event-a-final"})
	require.NoError(t, session.Close())
	require.Empty(t, stderr.String())
	require.Equal(t, before, snapshotTree(t, dir))
}

func TestMCPStdioConflictRetainsInformantContext(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(mcpUsabilityFixture), 0o600))
	session, _ := connectMCP(t, dir)
	data := callMCP(t, session, "analyze", map[string]any{"person": "person-mary", "check": "conflicts"})
	var result struct {
		Issues []struct {
			Person  string `json:"person"`
			Message string `json:"message"`
			Events  []struct {
				EventID      string   `json:"event_id"`
				PrincipalIDs []string `json:"principal_ids"`
				PersonRole   string   `json:"person_role"`
				OwnRecord    bool     `json:"own_record"`
				Title        string   `json:"title"`
			} `json:"events"`
		} `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	require.Len(t, result.Issues, 1)
	issue := result.Issues[0]
	require.Equal(t, "person-mary", issue.Person)
	require.Contains(t, issue.Message, "Death of Robin Green")
	require.Contains(t, issue.Message, "informant")
	require.Contains(t, issue.Message, "event-robin-death")
	require.Len(t, issue.Events, 1)
	require.Equal(t, "event-robin-death", issue.Events[0].EventID)
	require.Equal(t, []string{"person-robin"}, issue.Events[0].PrincipalIDs)
	require.Equal(t, "informant", issue.Events[0].PersonRole)
	require.False(t, issue.Events[0].OwnRecord)
	require.Equal(t, "Robin's death record", issue.Events[0].Title)
}
