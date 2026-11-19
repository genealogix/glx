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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Trigger cancellation at a known in-flight checkpoint rather than racing a
// wall-clock timer against machine speed. Done and Err use the real context.
//
//nolint:containedctx // Test context implementation, not a stored request context.
type cancelQueryContext struct {
	context.Context
	cancel context.CancelFunc
	calls  atomic.Int32
	after  int32
}

func (c *cancelQueryContext) Err() error {
	if c.calls.Add(1) == c.after {
		c.cancel()
	}

	return c.Context.Err()
}

func TestMCPQueryCancellationDuringBirthIndex(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}, Events: map[string]*glxlib.Event{}}
	for i := range 6000 {
		id := fmt.Sprintf("person-%05d", i)
		archive.Persons[id] = &glxlib.Person{Properties: map[string]any{"name": fmt.Sprintf("Person %d", i)}}
		archive.Events[fmt.Sprintf("event-%05d", i)] = &glxlib.Event{Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: id, Role: glxlib.ParticipantRolePrincipal}}}
	}
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 1000}
	backend := &mcpArchiveBackend{archive: archive}
	raw := json.RawMessage(`{"entity_type":"persons","born_before":1900,"limit":1}`)
	result, err := backend.CallTool(ctx, "query", raw)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.EqualValues(t, 1000, ctx.calls.Load(), "must stop at the in-flight cancellation checkpoint, before scanning the remaining events")
	start := time.Now()
	result, err = backend.CallTool(t.Context(), "query", raw)
	require.NoError(t, err)
	require.Equal(t, 6000, result.(*mcpQueryResult).Count)
	t.Logf("6000-person birth-filter query completed in %s", time.Since(start))
}

type cancelQueryYAML struct {
	cancel context.CancelFunc
	calls  *atomic.Int32
}

//nolint:unparam // Signature required by yaml.Marshaler.
func (v cancelQueryYAML) MarshalYAML() (any, error) {
	v.calls.Add(1)
	v.cancel()

	return "value", nil
}

func TestMCPQueryCancellationDuringPageEncoding(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var calls atomic.Int32
	value := cancelQueryYAML{cancel: cancel, calls: &calls}
	backend := &mcpArchiveBackend{archive: &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"person-a": {Properties: map[string]any{"name": "Alice", "note": value}},
		"person-b": {Properties: map[string]any{"name": "Bob", "note": value}},
	}}}
	result, err := backend.CallTool(ctx, "query", json.RawMessage(`{"entity_type":"persons"}`))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.EqualValues(t, 1, calls.Load(), "cancellation must prevent encoding the rest of the page")
}

func TestQueryBirthIndexMatchesExistingLookup(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}, Events: map[string]*glxlib.Event{}}
	for i, role := range []string{"", "subject", "principal", "child", "witness", "informant"} {
		id := fmt.Sprintf("person-%d", i)
		archive.Persons[id] = &glxlib.Person{}
		for _, prefix := range []string{"a", "z"} {
			archive.Events[prefix+id] = &glxlib.Event{Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: id, Role: role}}}
		}
	}
	index, err := queryBirthIndex(t.Context(), archive)
	require.NoError(t, err)
	for id := range archive.Persons {
		_, want := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeBirth)
		require.Same(t, want, index[id], id)
	}
}

func TestContextPhasesStopAfterActiveCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	phases := 0
	err := runContextPhases(
		ctx,
		func() { phases++; cancel() },
		func() { phases++ },
	)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 1, phases)

	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	checks := 0
	findings, err := runAnalysisChecksContext(
		ctx, &glxlib.GLXFile{},
		func(*glxlib.GLXFile) []AnalysisIssue {
			checks++
			cancel()

			return []AnalysisIssue{{Category: "gap"}}
		},
		func(*glxlib.GLXFile) []AnalysisIssue {
			checks++

			return nil
		},
	)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, findings)
	require.Equal(t, 1, checks)
}

func TestMCPContextBuildersStopBetweenPhases(t *testing.T) {
	archive := explorationTestArchive()
	for name, call := range map[string]func(context.Context) (any, error){
		"summary": func(ctx context.Context) (any, error) { return buildSummaryJSONContext(ctx, archive, "person-child") },
		"cluster": func(ctx context.Context) (any, error) {
			return buildClusterContext(ctx, "person-child", archive, "", 0, 0)
		},
		"stats":      func(ctx context.Context) (any, error) { return buildStatsJSONContext(ctx, archive) },
		"validation": func(ctx context.Context) (any, error) { return buildValidationJSONContext(ctx, archive, true) },
	} {
		t.Run(name, func(t *testing.T) {
			base, cancel := context.WithCancel(t.Context())
			defer cancel()
			ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 3}
			result, err := call(ctx)
			require.ErrorIs(t, err, context.Canceled)
			require.Nil(t, result)
			require.EqualValues(t, 3, ctx.calls.Load())
		})
	}
}

func TestMCPCanceledValidationDoesNotPopulateCache(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var calls atomic.Int32
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"person-a": {Properties: map[string]any{"name": "Alice", "note": cancelQueryYAML{cancel: cancel, calls: &calls}}},
	}}
	backend := &mcpArchiveBackend{archive: archive}
	result, err := backend.CallTool(ctx, "validate", json.RawMessage(`{"report":true}`))
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.EqualValues(t, 1, calls.Load(), "cancellation must occur during serialization, not just before dispatch")
	require.Nil(t, backend.validation, "canceled validation must never populate the snapshot cache")
	delete(archive.Persons["person-a"].Properties, "note")
	_, err = backend.CallTool(t.Context(), "validate", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.NotNil(t, backend.validation)
}

func TestMCPPathCancellationDuringTraversal(t *testing.T) {
	adj := map[string][]pathEdge{"a": {{PersonID: "b"}}, "b": {{PersonID: "c"}}, "c": {{PersonID: "d"}}}
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 4}
	result, err := bfsPathContext(ctx, "a", "d", adj, 10)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
	require.EqualValues(t, 4, ctx.calls.Load())
	want := bfsPath("a", "d", adj, 10)
	got, err := bfsPathContext(t.Context(), "a", "d", adj, 10)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestMCPContextBuilderBackgroundParity(t *testing.T) {
	archive := explorationTestArchive()
	summary, err := buildSummaryJSON(archive, "person-child")
	require.NoError(t, err)
	contextualSummary, err := buildSummaryJSONContext(t.Context(), archive, "person-child")
	require.NoError(t, err)
	require.Equal(t, summary, contextualSummary)
	cluster := buildCluster("person-child", archive, "", 0, 0)
	contextualCluster, err := buildClusterContext(t.Context(), "person-child", archive, "", 0, 0)
	require.NoError(t, err)
	require.Equal(t, cluster, contextualCluster)
	for _, category := range []string{"gaps", "evidence", "consistency", "conflicts", "suggestions"} {
		plain, buildErr := buildAnalysisJSON(archive, "", category)
		require.NoError(t, buildErr)
		contextual, contextErr := buildAnalysisJSONContext(t.Context(), archive, "", category)
		require.NoError(t, contextErr)
		require.Equal(t, plain, contextual)
	}
}

func TestAnalyzeGapsCancellationDuringPersonScan(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}}
	for i := range 100 {
		archive.Persons[fmt.Sprintf("person-%03d", i)] = &glxlib.Person{}
	}
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 10}
	findings, err := analyzeGapsContext(ctx, archive)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, findings)
	require.EqualValues(t, 10, ctx.calls.Load(), "cancel after building indexes and processing several people")
}

func TestMCPPathCancellationDuringEdgeConstruction(t *testing.T) {
	archive := &glxlib.GLXFile{Relationships: map[string]*glxlib.Relationship{
		"relationship-a": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "a", Role: "parent"}, {Person: "b", Role: "child"}}},
	}}
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &cancelQueryContext{Context: base, cancel: cancel, after: 6}
	adj, err := buildPathAdjacencyContext(ctx, archive)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, adj)
	require.EqualValues(t, 6, ctx.calls.Load())
	want := buildPathAdjacency(archive)
	got, err := buildPathAdjacencyContext(t.Context(), archive)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// Compare against the independent legacy aggregators, not the Background wrappers,
// so adding a check to either path cannot silently change MCP findings.
func TestMCPAnalysisContextLegacyParity(t *testing.T) {
	populated := explorationTestArchive()
	populated.Assertions["assertion-one"].Citations = nil
	populated.Events["event-death-child"] = &glxlib.Event{
		Type: "death", Date: "1800",
		Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}},
	}
	for _, fixture := range []struct {
		name    string
		archive *glxlib.GLXFile
	}{
		{"populated", populated},
		{"empty", &glxlib.GLXFile{}},
	} {
		for _, category := range []struct {
			name   string
			legacy func(*glxlib.GLXFile) []AnalysisIssue
		}{
			{"evidence", analyzeEvidence},
			{"consistency", analyzeConsistency},
			{"suggestions", analyzeSuggestions},
		} {
			t.Run(fixture.name+"/"+category.name, func(t *testing.T) {
				want := append([]AnalysisIssue{}, category.legacy(fixture.archive)...)
				sortIssues(want)
				if fixture.name == "populated" {
					require.NotEmpty(t, want, "exercise actual findings, not only empty output")
				}
				got, err := buildAnalysisJSONContext(t.Context(), fixture.archive, "", category.name)
				require.NoError(t, err)
				require.Equal(t, want, got.Issues)
				require.Equal(t, buildSummary(want), got.Summary)
			})
		}
	}
}

func TestLoadMCPArchiveContextPreCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// Cancellation wins even before filesystem access to a missing source.
	backend, err := loadMCPArchiveContext(ctx, "missing-canceled-archive")
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, backend)
}
