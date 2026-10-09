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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// newResearchTestArchive builds an archive with one brick-wall log about Mary
// Green (two leads naming John and Luther as candidates) and one log about John.
func newResearchTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary":   {Properties: map[string]any{"name": "Mary Green"}},
			"person-john":   {Properties: map[string]any{"name": "John H. Green"}},
			"person-luther": {Properties: map[string]any{"name": "Luther Green"}},
			"person-other":  {Properties: map[string]any{"name": "Other Person"}},
		},
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"research-log-mary-parents": {
				Subject:   &glxlib.EntityRef{Person: "person-mary"},
				Objective: "Identify Mary Green's parents",
				Status:    glxlib.ResearchLogStatusInProgress,
				Searches: []glxlib.Search{
					{Collection: "1850 census", Result: glxlib.SearchResultNotSearched},
					{Collection: "1840 census", Result: glxlib.SearchResultFound},
				},
				Leads: []glxlib.ResearchLead{
					{
						Description: "John H. Green of Wheeling, VA",
						Persons:     []string{"person-john"},
						Status:      glxlib.LeadStatusActive,
						NextSteps:   []string{"Search 1850 census, Ohio County, VA"},
					},
					{
						Description: "Luther Green of Springwater, NY",
						Persons:     []string{"person-luther"},
						Status:      glxlib.LeadStatusEliminated,
					},
				},
			},
			"research-log-john-origins": {
				Subject: &glxlib.EntityRef{Person: "person-john"},
				Title:   "John H. Green origins",
				Status:  glxlib.ResearchLogStatusBlocked,
			},
		},
	}
}

func TestPrintResearchSection_Subject(t *testing.T) {
	archive := newResearchTestArchive()
	output := captureStdout(t, func() {
		printResearchSection("person-mary", archive)
	})

	assert.Contains(t, output, "── Research")
	assert.Contains(t, output, "research-log-mary-parents  [in_progress]\n")
	assert.Contains(t, output, "Identify Mary Green's parents")
	assert.Contains(t, output, "1 planned search not yet performed")
	assert.Contains(t, output, "1 active, 1 eliminated")
	assert.Contains(t, output, "- John H. Green of Wheeling, VA (next: Search 1850 census, Ohio County, VA)")
	assert.NotContains(t, output, "Luther Green of Springwater", "eliminated leads are counted, not listed")
	assert.NotContains(t, output, "research-log-john-origins")
}

func TestPrintResearchSection_Candidate(t *testing.T) {
	archive := newResearchTestArchive()
	output := captureStdout(t, func() {
		printResearchSection("person-john", archive)
	})

	assert.Contains(t, output, "research-log-john-origins  [blocked]")
	assert.Contains(t, output, "John H. Green origins", "title is the fallback when there is no objective")
	assert.Contains(t, output, "research-log-mary-parents  [in_progress]  (candidate in a lead)")
	assert.Less(t, strings.Index(output, "research-log-john-origins"), strings.Index(output, "research-log-mary-parents"),
		"logs are listed in ID order")
}

func TestPrintResearchSection_NoLogs(t *testing.T) {
	archive := newResearchTestArchive()
	output := captureStdout(t, func() {
		printResearchSection("person-other", archive)
	})
	assert.Empty(t, output, "no Research header when the person has no logs")
}

func TestFormatOpenLead_Fallbacks(t *testing.T) {
	assert.Equal(t, "person-a, person-b", formatOpenLead(&glxlib.ResearchLead{Persons: []string{"person-a", "person-b"}}))
	assert.Equal(t, "(undescribed lead)", formatOpenLead(&glxlib.ResearchLead{}))
	assert.Equal(t, "X (next: Y)", formatOpenLead(&glxlib.ResearchLead{Description: "X", NextSteps: []string{"Y"}}))
}

func TestShowSummary_ResearchSectionFromExample(t *testing.T) {
	output := captureStdout(t, func() {
		require.NoError(t, showSummary("../docs/examples/complete-family", "person-mary-brown-1852"))
	})
	assert.Contains(t, output, "── Research")
	assert.Contains(t, output, "research-log-mary-brown-parents  [in_progress]")
	assert.Contains(t, output, "1 active, 1 eliminated")
}

func TestQueryResearchLogs_SubjectFilter(t *testing.T) {
	archive := newResearchTestArchive()

	tests := []struct {
		name    string
		subject string
		want    []string
		notWant []string
	}{
		{"subject ID", "person-mary", []string{"research-log-mary-parents"}, []string{"research-log-john-origins"}},
		{"lead candidate ID", "person-john", []string{"research-log-mary-parents", "research-log-john-origins"}, nil},
		{"eliminated candidate still matches", "person-luther", []string{"research-log-mary-parents"}, []string{"research-log-john-origins"}},
		{"name substring", "luther", []string{"research-log-mary-parents"}, []string{"research-log-john-origins"}},
		{"no match", "person-other", nil, []string{"research-log-mary-parents", "research-log-john-origins"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			output := captureStdout(t, func() {
				require.NoError(t, queryResearchLogs(archive, &queryOpts{Subject: tc.subject}))
			})
			for _, id := range tc.want {
				assert.Contains(t, output, id)
			}
			for _, id := range tc.notWant {
				assert.NotContains(t, output, id)
			}
			assert.Contains(t, output, "\n"+strconv.Itoa(len(tc.want))+" research logs found")
		})
	}
}

func TestQueryResearchLogs_StatusFilterAndLeadCount(t *testing.T) {
	archive := newResearchTestArchive()
	output := captureStdout(t, func() {
		require.NoError(t, queryResearchLogs(archive, &queryOpts{Status: "IN_PROGRESS"}))
	})
	assert.Contains(t, output, "research-log-mary-parents  [in_progress]  Identify Mary Green's parents  (2 leads)")
	assert.NotContains(t, output, "research-log-john-origins")
	assert.Contains(t, output, "1 research logs found")
}

func TestQueryResearchLogs_FlagsAccepted(t *testing.T) {
	require.NoError(t, validateQueryFlags("research_logs", &queryOpts{Subject: "x", Status: "open"}))
	require.Error(t, validateQueryFlags("research_logs", &queryOpts{Confidence: "high"}))
}

// Single-file archives get the research-log vocabularies from the standard set,
// like directory archives do; before #660 search_result_types and
// research_log_status_types were missing, so a self-contained file using a
// standard search result or log status failed validation.
func TestMergeStandardVocabularies_LeadStatuses(t *testing.T) {
	archive := &glxlib.GLXFile{}
	require.NoError(t, mergeStandardVocabularies(archive))
	assert.Contains(t, archive.SearchResultTypes, glxlib.SearchResultNotSearched)
	assert.Contains(t, archive.ResearchLogStatusTypes, glxlib.ResearchLogStatusInProgress)
	assert.Contains(t, archive.LeadStatuses, glxlib.LeadStatusEliminated)
}

func TestPrintResearchSection_PreservesLeadStatuses(t *testing.T) {
	archive := newResearchTestArchive()
	archive.ResearchLogs["research-log-mary-parents"].Leads = []glxlib.ResearchLead{
		{Description: "On hold", Status: "parked", NextSteps: []string{"Wait for access"}},
		{Description: "Still investigating", Status: glxlib.LeadStatusActive},
		{Description: "Awaiting records", Status: "blocked"},
		{Description: "No status yet"},
		{Description: "Ruled out", Status: glxlib.LeadStatusEliminated},
		{Description: "Proven", Status: glxlib.LeadStatusConfirmed},
	}
	output := captureStdout(t, func() {
		printResearchSection("person-mary", archive)
	})
	assert.Contains(t, output, "1 active, 1 confirmed, 1 eliminated, 1 blocked, 1 parked, 1 unspecified")
	assert.Contains(t, output, "On hold (parked; next: Wait for access)")
	assert.Contains(t, output, "Awaiting records (blocked)")
	assert.Contains(t, output, "No status yet")
	assert.NotContains(t, output, "4 active")
	assert.NotContains(t, output, "Ruled out")
	assert.NotContains(t, output, "Proven")
}
