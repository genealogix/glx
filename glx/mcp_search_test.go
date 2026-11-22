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
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
	glxmcp "github.com/genealogix/glx/mcp"
)

func searchMCPArchive(t *testing.T, backend *mcpArchiveBackend, args glxmcp.SearchArgs) *mcpSearchResult {
	t.Helper()
	raw, err := json.Marshal(args)
	require.NoError(t, err)
	value, err := backend.CallTool(t.Context(), "search", raw)
	require.NoError(t, err)
	result, ok := value.(*mcpSearchResult)
	require.True(t, ok)

	return result
}

func TestMCPSearchRecordedTextAndExternalIdentifiers(t *testing.T) {
	const marker = "TEST-RECORD-0042"
	archive := newTestArchiveForSearch()
	archive.Citations["citation-provider"] = &glxlib.Citation{Properties: map[string]any{
		"external_id": []any{map[string]any{"value": "previous"}, map[string]any{"value": marker}},
	}}
	archive.ResearchLogs = map[string]*glxlib.ResearchLog{
		"log-provider": {Searches: []glxlib.Search{{Query: "Checked " + marker, Notes: glxlib.NoteList{marker + "; index only"}}}},
	}
	archive.Persons["person-null"] = nil
	backend := &mcpArchiveBackend{archive: archive}
	before, err := json.Marshal(archive)
	require.NoError(t, err)
	result := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "  test-record-0042  "})
	require.Equal(t, &mcpSearchResult{Matches: []mcpSearchMatch{
		{EntityType: glxlib.EntityTypeCitations, EntityID: "citation-provider", Field: "properties.external_id", Value: marker},
		{EntityType: glxlib.EntityTypeResearchLogs, EntityID: "log-provider", Field: "searches[0].query", Value: "Checked " + marker},
	}, Count: 2, Returned: 2}, result)
	require.Zero(t, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "test-record-0042", CaseSensitive: true}).Count)
	require.Equal(t, 1, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: marker, CaseSensitive: true, EntityType: " CITATIONS "}).Count)
	require.Equal(t, 1, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: marker, EntityType: "research_logs"}).Count)
	require.Zero(t, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "no such recorded text"}).Count)
	after, err := json.Marshal(archive)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after))
}

func TestMCPSearchAllCollectionsFirstFieldAndOrder(t *testing.T) {
	// Repeated IDs in different collections remain separate results. Multiple
	// matching fields in one entity retain the CLI's first field only.
	archive := &glxlib.GLXFile{
		Persons:       map[string]*glxlib.Person{"same-target": {Notes: glxlib.NoteList{"target"}}},
		Events:        map[string]*glxlib.Event{"same-target": {Title: "target"}},
		Places:        map[string]*glxlib.Place{"same-target": {Name: "target"}},
		Sources:       map[string]*glxlib.Source{"same-target": {Title: "target"}},
		Citations:     map[string]*glxlib.Citation{"same-target": {Notes: glxlib.NoteList{"target"}}},
		Repositories:  map[string]*glxlib.Repository{"same-target": {Name: "target"}},
		Assertions:    map[string]*glxlib.Assertion{"same-target": {Value: "target"}},
		Relationships: map[string]*glxlib.Relationship{"same-target": {Type: "target"}},
		Media:         map[string]*glxlib.Media{"same-target": {Title: "target"}},
		ResearchLogs:  map[string]*glxlib.ResearchLog{"same-target": {Title: "target"}},
		Studies:       map[string]*glxlib.Study{"same-target": {Title: "target"}},
	}
	backend := &mcpArchiveBackend{archive: archive}
	for range 3 {
		result := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target"})
		require.Equal(t, 11, result.Count)
		require.Len(t, result.Matches, 11)
		for i, entityType := range searchEntityTypes {
			require.Equal(t, mcpSearchMatch{EntityType: entityType.Key, EntityID: "same-target", Field: "id", Value: "same-target"}, result.Matches[i])
			filtered := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target", EntityType: entityType.Key.String()})
			require.Equal(t, 1, filtered.Count)
		}
	}
	boundary := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target", Offset: 4, Limit: 2})
	require.Equal(t, glxlib.EntityTypeCitations, boundary.Matches[0].EntityType)
	require.Equal(t, glxlib.EntityTypeRepositories, boundary.Matches[1].EntityType)
	require.True(t, boundary.HasMore)
}

func TestMCPSearchPaginationAndPreviewBounds(t *testing.T) {
	archive := &glxlib.GLXFile{Sources: map[string]*glxlib.Source{}}
	longTitle := strings.Repeat("x", 200)
	for i := range 1205 {
		archive.Sources[fmt.Sprintf("source-%04d", i)] = &glxlib.Source{Title: longTitle}
	}
	backend := &mcpArchiveBackend{archive: archive}
	for _, page := range []struct{ limit, offset, returned int }{
		{0, 0, 100},
		{1000, 0, 1000},
		{1000, 1000, 205},
		{1, 1204, 1},
		{10, 1205, 0},
		{10, math.MaxInt, 0},
	} {
		result := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "x", Limit: page.limit, Offset: page.offset})
		require.Equal(t, 1205, result.Count)
		require.Equal(t, page.offset, result.Offset)
		require.Equal(t, page.returned, result.Returned)
		require.Len(t, result.Matches, page.returned)
		require.Equal(t, page.offset < 1205-page.returned, result.HasMore)
		for i, match := range result.Matches {
			require.Equal(t, fmt.Sprintf("source-%04d", page.offset+i), match.EntityID)
			require.Equal(t, "title", match.Field)
			require.True(t, utf8.ValidString(match.Value))
			require.Equal(t, 80, utf8.RuneCountInString(match.Value))
			require.True(t, strings.HasSuffix(match.Value, "..."))
		}
		data, err := json.Marshal(result)
		require.NoError(t, err)
		if page.returned == 0 {
			require.Contains(t, string(data), `"matches":[]`)
		}
	}
	archive.Sources["source-0000"].Title = strings.Repeat("界", 200)
	unicodePage := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "界"})
	require.Equal(t, 1, unicodePage.Count)
	require.True(t, utf8.ValidString(unicodePage.Matches[0].Value))
	require.Equal(t, 80, utf8.RuneCountInString(unicodePage.Matches[0].Value))
}

func TestMCPSearchInvalidInputsAndRecovery(t *testing.T) {
	backend := &mcpArchiveBackend{archive: newTestArchiveForSearch()}
	for _, testCase := range []struct {
		args glxmcp.SearchArgs
		want error
	}{
		{glxmcp.SearchArgs{}, errSearchQueryEmpty},
		{glxmcp.SearchArgs{Query: " \n\t"}, errSearchQueryEmpty},
		{glxmcp.SearchArgs{Query: "Jane", EntityType: "person"}, errUnknownSearchType},
		{glxmcp.SearchArgs{Query: "Jane", Offset: -1}, errMCPPagination},
		{glxmcp.SearchArgs{Query: "Jane", Limit: -1}, errMCPPagination},
		{glxmcp.SearchArgs{Query: "Jane", Limit: 1001}, errMCPPagination},
		{glxmcp.SearchArgs{Query: strings.Repeat("界", 1025)}, errMCPSearchQueryLength},
	} {
		raw, err := json.Marshal(testCase.args)
		require.NoError(t, err)
		value, err := backend.CallTool(t.Context(), "search", raw)
		require.ErrorIs(t, err, testCase.want)
		require.Nil(t, value)
	}
	for _, raw := range []string{`{"query":`, `{"query":42}`, `[]`} {
		value, err := backend.CallTool(t.Context(), "search", json.RawMessage(raw))
		require.Error(t, err)
		require.Nil(t, value)
	}
	raw, err := json.Marshal(glxmcp.SearchArgs{Query: "Jane", EntityType: strings.Repeat("\x00", 300000)})
	require.NoError(t, err)
	value, err := backend.CallTool(t.Context(), "search", raw)
	require.ErrorIs(t, err, errUnknownSearchType)
	require.Nil(t, value)
	require.Less(t, len(err.Error()), 200, "invalid collections must not produce unbounded protocol errors")
	require.Zero(t, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: strings.Repeat("界", 1024)}).Count)
	require.Equal(t, 1, searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "Jane", EntityType: "persons"}).Count)
}

func TestMCPSearchByteBudgetPreservesPagingAndOpaqueReferences(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}}
	for _, suffix := range []string{"a", "b", "c", "d"} {
		archive.Persons[strings.Repeat("x", 90000)+suffix] = &glxlib.Person{Notes: glxlib.NoteList{"target"}}
	}
	backend := &mcpArchiveBackend{archive: archive}
	first := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target"})
	require.Equal(t, 4, first.Count)
	require.Equal(t, 2, first.Returned)
	require.True(t, first.HasMore)
	second := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target", Offset: first.Returned})
	require.Equal(t, 2, second.Returned)
	require.False(t, second.HasMore)
	for _, page := range []*mcpSearchResult{first, second} {
		data, err := json.Marshal(page)
		require.NoError(t, err)
		require.LessOrEqual(t, len(data), mcpSearchMaxResultBytes)
		for _, match := range page.Matches {
			require.Contains(t, archive.Persons, match.EntityID, "IDs must remain exact and retrievable")
		}
	}
	require.NotEqual(t, first.Matches[1].EntityID, second.Matches[0].EntityID)
	archive.Persons = map[string]*glxlib.Person{
		strings.Repeat("x", mcpSearchMaxResultBytes): {Notes: glxlib.NoteList{"target"}},
	}
	value, err := backend.CallTool(t.Context(), "search", json.RawMessage(`{"query":"target"}`))
	require.ErrorIs(t, err, errMCPSearchMatchSize)
	require.Nil(t, value)
	archive.Persons = map[string]*glxlib.Person{
		"person-a": {Properties: map[string]any{strings.Repeat("x", mcpSearchMaxResultBytes): "target"}},
	}
	value, err = backend.CallTool(t.Context(), "search", json.RawMessage(`{"query":"target"}`))
	require.ErrorIs(t, err, errMCPSearchMatchSize)
	require.Nil(t, value)
	archive.Persons = map[string]*glxlib.Person{
		strings.Repeat("\x00", 30000) + "a": {Notes: glxlib.NoteList{"target"}},
		strings.Repeat("\x00", 30000) + "b": {Notes: glxlib.NoteList{"target"}},
	}
	escaped := searchMCPArchive(t, backend, glxmcp.SearchArgs{Query: "target"})
	require.Equal(t, 1, escaped.Returned)
	require.True(t, escaped.HasMore)
	data, err := json.Marshal(escaped)
	require.NoError(t, err)
	require.LessOrEqual(t, len(data), mcpSearchMaxResultBytes, "budget must measure JSON escaping, not raw string bytes")
}

func TestMCPSearchCancellationDuringScans(t *testing.T) {
	for _, entityType := range []string{"persons", "events", "research_logs"} {
		t.Run(entityType, func(t *testing.T) {
			archive := &glxlib.GLXFile{
				Persons: map[string]*glxlib.Person{}, Events: map[string]*glxlib.Event{},
				ResearchLogs: map[string]*glxlib.ResearchLog{},
			}
			for i := range 100 {
				archive.Persons[fmt.Sprintf("person-%03d", i)] = &glxlib.Person{Properties: map[string]any{"name": "Person"}}
			}
			archive.Events["event-many"] = &glxlib.Event{Participants: make([]glxlib.Participant, 100)}
			archive.ResearchLogs["log-many"] = &glxlib.ResearchLog{Searches: make([]glxlib.Search, 100)}
			base, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 12}
			backend := &mcpArchiveBackend{archive: archive}
			raw, err := json.Marshal(glxmcp.SearchArgs{Query: "no-match", EntityType: entityType})
			require.NoError(t, err)
			value, err := backend.CallTool(ctx, "search", raw)
			require.ErrorIs(t, err, context.Canceled)
			require.Nil(t, value)
			require.LessOrEqual(t, ctx.calls.Load(), int32(14), "must stop at an in-flight checkpoint, including scans with no matches")
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	value, err := (&mcpArchiveBackend{}).CallTool(ctx, "search", json.RawMessage(`{"query":"anything"}`))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, value)
}

func TestSearchVisitorStopsBeforeLaterEntities(t *testing.T) {
	visited := 0
	err := visitSearchArchiveContext(t.Context(), newTestArchiveForSearch(), "person", false, "", func(searchResult) bool {
		visited++

		return false
	})
	require.NoError(t, err)
	require.Equal(t, 1, visited)
}
