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
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMCPGetEntityEveryCollectionAndFidelity(t *testing.T) {
	archive := explorationTestArchive()
	archive.ResearchLogs["research-one"].Notes = glxlib.NoteList{"Unlocated record; do not treat the lead as proved."}
	archive.ResearchLogs["research-one"].Properties = map[string]any{"custom_field": "Preserve custom provenance"}
	archive.ResearchLogs["research-one"].Searches = []glxlib.Search{{SourceID: "source-census", CitationID: "citation-one", Result: "not_found", Notes: glxlib.NoteList{"Index checked; original unavailable"}}}
	backend := &mcpArchiveBackend{archive: archive}
	before, err := json.Marshal(archive)
	require.NoError(t, err)
	for _, entityType := range glxlib.AllEntityTypes {
		t.Run(string(entityType), func(t *testing.T) {
			page, queryErr := buildQueryJSON(archive, entityType, &queryOpts{})
			require.NoError(t, queryErr)
			require.NotEmpty(t, page.Entities)
			for _, entity := range page.Entities {
				raw, encodeErr := json.Marshal(map[string]any{"entity_type": entityType, "id": entity.ID})
				require.NoError(t, encodeErr)
				value, callErr := backend.CallTool(t.Context(), "get_entity", raw)
				require.NoError(t, callErr)
				result, ok := value.(*mcpEntityResult)
				require.True(t, ok)
				require.Equal(t, entityType, result.EntityType)
				require.Equal(t, entity.ID, result.ID)
				want, marshalErr := marshalArchiveJSON(entity.Entity)
				require.NoError(t, marshalErr)
				require.JSONEq(t, string(want), string(result.Entity))
			}
		})
	}
	after, err := json.Marshal(archive)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
}

func TestMCPGetEntityRejectsInexactAndWrongCollectionIDs(t *testing.T) {
	archive := explorationTestArchive()
	// IDs are scoped by the explicit collection even in hand-edited archives.
	archive.Events["person-child"] = &glxlib.Event{Type: "naturalization", Title: "Distinct entity"}
	archive.Events["event-nil"] = nil
	backend := &mcpArchiveBackend{archive: archive}
	for _, raw := range []string{
		`{}`, `{"entity_type":"events","id":""}`, `{"entity_type":"events","id":"  "}`,
		`{"entity_type":"event","id":"person-child"}`, `{"entity_type":"../events","id":"person-child"}`,
		`{"entity_type":"events","id":"event-nil"}`, `{"entity_type":"events","id":"event-missing"}`,
		`{"entity_type":"events","id":"event-birth"}`, `{"entity_type":"events","id":"EVENT-BIRTH-CHILD"}`,
		`{"entity_type":"events","id":" event-birth-child"}`, `{"entity_type":"sources","id":"event-birth-child"}`,
		`{"entity_type":"persons","id":"Child Smith"}`, `{"entity_type":"events","id":"/etc/passwd"}`,
	} {
		_, err := backend.CallTool(t.Context(), "get_entity", json.RawMessage(raw))
		require.Error(t, err, raw)
	}
	for _, entityType := range []string{"events", "persons"} {
		result, err := backend.CallTool(t.Context(), "get_entity", json.RawMessage(`{"entity_type":"`+entityType+`","id":"person-child"}`))
		require.NoError(t, err)
		require.EqualValues(t, entityType, result.(*mcpEntityResult).EntityType)
	}
	_, err := backend.CallTool(t.Context(), "stats", json.RawMessage(`{}`))
	require.NoError(t, err)
}

func TestMCPGetEntityCancellationDuringEncoding(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var calls atomic.Int32
	archive := explorationTestArchive()
	archive.ResearchLogs["research-one"].Properties = map[string]any{"custom": cancelQueryYAML{cancel: cancel, calls: &calls}}
	backend := &mcpArchiveBackend{archive: archive}
	result, err := backend.CallTool(ctx, "get_entity", json.RawMessage(`{"entity_type":"research_logs","id":"research-one"}`))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.EqualValues(t, 1, calls.Load())
}

func TestMCPSummaryCompactLogsAndExplicitCompatibility(t *testing.T) {
	archive := explorationTestArchive()
	longText := strings.Repeat("Detailed source notes and uncertainty. ", 10000)
	log := archive.ResearchLogs["research-one"]
	log.Title, log.Date = "Parents investigation", "2020-02-03"
	log.Notes, log.Conclusions = glxlib.NoteList{longText}, longText
	log.Searches = []glxlib.Search{{Result: "not_found", Notes: glxlib.NoteList{longText}}}
	archive.ResearchLogs["research-candidate"] = &glxlib.ResearchLog{Title: "Candidate investigation", Leads: []glxlib.ResearchLead{{Persons: []string{"person-child"}}}}
	archive.ResearchLogs["research-unrelated"] = &glxlib.ResearchLog{Title: "Unrelated"}
	archive.ResearchLogs["research-nil"] = nil
	backend := &mcpArchiveBackend{archive: archive}
	value, err := backend.CallTool(t.Context(), "summary", json.RawMessage(`{"person":"person-child"}`))
	require.NoError(t, err)
	compact := value.(*summaryJSONResult)
	require.Empty(t, compact.ResearchLogs)
	require.False(t, compact.ResearchLogsIncluded)
	require.Equal(t, 2, compact.ResearchLogCount)
	require.Equal(t, []summaryResearchLogRef{
		{ID: "research-candidate", Title: "Candidate investigation", LeadCount: 1},
		{ID: "research-one", Title: "Parents investigation", Status: "in_progress", Date: "2020-02-03", IsSubject: true, SearchCount: 1},
	}, compact.ResearchLogRefs)
	data, err := json.Marshal(compact)
	require.NoError(t, err)
	require.Less(t, len(data), 20000, "routine profiles must not grow with log bodies")
	require.NotContains(t, string(data), "Detailed source notes")
	require.Contains(t, string(data), `"research_logs":[]`)
	fullValue, err := backend.CallTool(t.Context(), "summary", json.RawMessage(`{"person":"person-child","include_research_logs":true}`))
	require.NoError(t, err)
	full := fullValue.(*summaryJSONResult)
	require.True(t, full.ResearchLogsIncluded)
	require.Len(t, full.ResearchLogs, 2)
	require.Equal(t, compact.ResearchLogRefs, full.ResearchLogRefs)
	legacy, err := buildSummaryJSON(archive, "person-child")
	require.NoError(t, err)
	require.Equal(t, legacy, full, "CLI JSON retains complete logs")
	encodedLog, err := json.Marshal(full.ResearchLogs[1])
	require.NoError(t, err)
	require.Contains(t, string(encodedLog), longText)
	for range 2 {
		repeated, callErr := backend.CallTool(t.Context(), "summary", json.RawMessage(`{"person":"person-child"}`))
		require.NoError(t, callErr)
		require.Equal(t, compact, repeated)
	}
	empty, err := backend.CallTool(t.Context(), "summary", json.RawMessage(`{"person":"person-father"}`))
	require.NoError(t, err)
	require.Zero(t, empty.(*summaryJSONResult).ResearchLogCount)
	require.Empty(t, empty.(*summaryJSONResult).ResearchLogRefs)
}

func TestTimelineFamilyEventRetainsSubtype(t *testing.T) {
	archive := explorationTestArchive()
	event := archive.Events["event-birth-child"]
	event.Title = "Birth register entry"
	event.Properties = map[string]any{"event_subtype": "Delayed registration"}
	entries := relatedPersonTimelineEvents(relatedPerson{PersonID: "person-child", Name: "Child Smith", Relation: relationChild}, archive)
	require.Len(t, entries, 1)
	require.Equal(t, "event-birth-child", entries[0].EventID)
	require.Equal(t, "birth", entries[0].EventType)
	require.Equal(t, "Delayed registration", entries[0].EventSubtype)
	require.Equal(t, "Birth register entry", entries[0].Title)
	require.True(t, entries[0].IsFamily)
}

func TestNotableNarrativeExcludesWitnessProceedings(t *testing.T) {
	archive := explorationTestArchive()
	archive.Events["event-own"] = &glxlib.Event{Type: "naturalization", Date: "1900", Properties: map[string]any{"event_subtype": "Declaration"}, Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}}
	archive.Events["event-witness"] = &glxlib.Event{Type: "naturalization", Date: "1910", Properties: map[string]any{"event_subtype": "Other person's admission"}, Participants: []glxlib.Participant{{Person: "person-child", Role: "witness"}, {Person: "person-father", Role: "principal"}}}
	narrative := generateLifeHistory("person-child", archive.Persons["person-child"], archive)
	require.Contains(t, narrative, "Declaration")
	require.NotContains(t, narrative, "Other person's admission")
}
