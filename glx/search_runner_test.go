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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func newTestArchiveForSearch() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane": {Properties: map[string]any{
				"name": "Jane Miller", "occupation": "farmer",
			}},
			"person-john": {Properties: map[string]any{
				"name": "John Smith",
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type:    "census",
				Title:   "1860 Census — Millbrook",
				Date:    "1860",
				PlaceID: "place-millbrook",
				Participants: []glxlib.Participant{
					{Person: "person-jane", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-millbrook": {Name: "Millbrook, Hartford County, Wisconsin"},
		},
		Sources: map[string]*glxlib.Source{
			"source-1860": {Title: "1860 Federal Census — Millbrook", Type: "census"},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-1860": {SourceID: "source-1860"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:  glxlib.EntityRef{Person: "person-jane"},
				Property: "born_at",
				Value:    "place-millbrook",
				Notes:    glxlib.NoteList{"Born in Millbrook area"},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Repositories: map[string]*glxlib.Repository{
			"repo-archives": {
				Name:       "Hartford County Archives",
				State:      "Wisconsin",
				PostalCode: "54321",
				Country:    "USA",
			},
		},
		Media: map[string]*glxlib.Media{},
	}
}

func TestSearchArchive_FindsPersonName(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "Miller", false, "")

	hasPersonMatch := false
	for _, r := range results {
		if r.EntityType == glxlib.EntityTypePersons && r.EntityID == "person-jane" {
			hasPersonMatch = true
		}
	}
	assert.True(t, hasPersonMatch, "should find 'Miller' in person name")
}

func TestSearchArchive_CaseInsensitive(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "millbrook", false, "")

	require.NotEmpty(t, results, "case-insensitive search should find 'Millbrook'")
}

func TestSearchArchive_CaseSensitive(t *testing.T) {
	archive := newTestArchiveForSearch()

	// "MILLBROOK" (uppercase) should NOT match "Millbrook" in case-sensitive mode
	results := searchArchive(archive, "MILLBROOK", true, "")
	assert.Empty(t, results, "case-sensitive search for 'MILLBROOK' should not match 'Millbrook'")
}

func TestSearchArchive_FindsPlaceName(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "Hartford", false, "")

	hasPlaceMatch := false
	for _, r := range results {
		if r.EntityType == glxlib.EntityTypePlaces && r.EntityID == "place-millbrook" {
			hasPlaceMatch = true
		}
	}
	assert.True(t, hasPlaceMatch, "should find 'Hartford' in place name")
}

func TestSearchArchive_FindsEventTitle(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "1860 Census", false, "")

	hasEventMatch := false
	for _, r := range results {
		if r.EntityType == glxlib.EntityTypeEvents && r.EntityID == "event-census" {
			hasEventMatch = true
		}
	}
	assert.True(t, hasEventMatch, "should find '1860 Census' in event title")
}

func TestSearchArchive_FindsSourceTitle(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "Federal Census", false, "")

	hasSourceMatch := false
	for _, r := range results {
		if r.EntityType == glxlib.EntityTypeSources && r.EntityID == "source-1860" {
			hasSourceMatch = true
		}
	}
	assert.True(t, hasSourceMatch, "should find 'Federal Census' in source title")
}

func TestSearchArchive_FindsAssertionNotes(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "Millbrook area", false, "")

	hasAssertionMatch := false
	for _, r := range results {
		if r.EntityType == glxlib.EntityTypeAssertions && r.EntityID == "a-1" {
			hasAssertionMatch = true
		}
	}
	assert.True(t, hasAssertionMatch, "should find 'Millbrook area' in assertion notes")
}

// Regression guard: searchArchive must match repo.State, repo.PostalCode, and repo.Country.
func TestSearchArchive_FindsRepositoryAddressFields(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantField string
	}{
		{"state", "Wisconsin", "state"},
		{"postal_code", "54321", "postal_code"},
		{"country", "USA", "country"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := newTestArchiveForSearch()
			results := searchArchive(archive, tc.query, false, "")

			found := false
			for _, r := range results {
				if r.EntityType == glxlib.EntityTypeRepositories && r.EntityID == "repo-archives" && r.Field == tc.wantField {
					found = true

					break
				}
			}
			assert.True(t, found, "should find %q in repository %s field", tc.query, tc.wantField)
		})
	}
}

func TestSearchArchive_NoMatches(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "XYZ_NONEXISTENT", false, "")

	assert.Empty(t, results, "should return no matches for nonexistent term")
}

func TestSearchArchive_MatchesEntityID(t *testing.T) {
	archive := newTestArchiveForSearch()
	results := searchArchive(archive, "person-jane", false, "")

	require.NotEmpty(t, results, "should match entity IDs")
}

func TestShowSearch_EmptyQuery(t *testing.T) {
	err := showSearch(".", "", false, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestShowSearch_InvalidType(t *testing.T) {
	err := showSearch(".", "test", false, "invalid_type", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestSearchArchive_TypeFilter(t *testing.T) {
	archive := newTestArchiveForSearch()

	// "Millbrook" appears in persons, events, places, sources, assertions
	allResults := searchArchive(archive, "Millbrook", false, "")
	require.NotEmpty(t, allResults)

	// Filter to just places via searchArchive typeFilter
	placesOnly := searchArchive(archive, "Millbrook", false, "places")
	require.NotEmpty(t, placesOnly, "should have place matches")
	for _, r := range placesOnly {
		assert.Equal(t, glxlib.EntityTypePlaces, r.EntityType, "filtered results should only contain places")
	}

	// Non-place results should exist in unfiltered
	hasNonPlace := false
	for _, r := range allResults {
		if r.EntityType != glxlib.EntityTypePlaces {
			hasNonPlace = true
		}
	}
	assert.True(t, hasNonPlace, "unfiltered results should include non-place entities")
}

func TestShowSearch_TypeFilterOutput(t *testing.T) {
	// Write a temporary single-file archive
	archiveContent := `persons:
  person-test:
    properties:
      name: "Jane Millbrook"
places:
  place-mill:
    name: "Millbrook"
`
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(archivePath, []byte(archiveContent), 0o644))

	// Search with --type=places should only show places
	output := captureStdout(t, func() {
		err := showSearch(archivePath, "Millbrook", false, "places", false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Places")
	assert.NotContains(t, output, "Persons")
}

func TestSearchArchive_FindsTemporalListPropertyValue(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-temporal": {Properties: map[string]any{
				"occupation": []any{
					map[string]any{"value": "farmer", "date": 1850},
					map[string]any{"value": "blacksmith", "date": 1860},
				},
			}},
		},
	}

	results := searchArchive(archive, "blacksmith", false, "")

	require.Len(t, results, 1, "should find one match for 'blacksmith' in temporal-list property")
	assert.Equal(t, glxlib.EntityTypePersons, results[0].EntityType)
	assert.Equal(t, "person-temporal", results[0].EntityID)
	assert.Equal(t, "properties.occupation", results[0].Field)
	assert.Equal(t, "blacksmith", results[0].Value)
}

func TestSearchArchive_FindsMapShapedPropertyValue(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mapshaped": {Properties: map[string]any{
				"religion": map[string]any{"value": "Methodist", "date": 1850},
			}},
		},
	}

	results := searchArchive(archive, "Methodist", false, "")

	require.Len(t, results, 1, "should find one match for 'Methodist' in map-shaped property")
	assert.Equal(t, glxlib.EntityTypePersons, results[0].EntityType)
	assert.Equal(t, "person-mapshaped", results[0].EntityID)
	assert.Equal(t, "properties.religion", results[0].Field)
	assert.Equal(t, "Methodist", results[0].Value)
}

func TestSearchArchive_FindsResearchLogFields(t *testing.T) {
	archive := &glxlib.GLXFile{
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"research-log-1": {
				Title:       "Smith family birth records",
				Researcher:  "I. Schepp",
				Objective:   "Locate baptism for William Smith b. 1812",
				Status:      "in_progress",
				Date:        "2024-03",
				Conclusions: "Inconclusive after parish-register sweep",
				Notes:       glxlib.NoteList{"Cross-reference Hartford registry"},
				Citations:   []string{"cit-smith-baptism"},
				Searches: []glxlib.Search{
					{
						RepositoryID: "repo-hartford",
						SourceID:     "source-parish-register",
						Collection:   "Baptisms 1810-1820",
						Query:        "Smith William",
						Result:       "not_found",
						CitationID:   "cit-smith-baptism",
						Date:         "2024-03-15",
						Notes:        glxlib.NoteList{"checked all 1812 entries"},
					},
				},
				Properties: map[string]any{"priority": "high"},
			},
		},
	}

	// Top-level scalar fields.
	for _, q := range []string{"Smith family birth records", "Schepp", "Locate baptism", "in_progress", "Inconclusive", "Hartford registry"} {
		results := searchArchive(archive, q, false, "")
		require.NotEmpty(t, results, "should match %q", q)
		assert.Equal(t, glxlib.EntityTypeResearchLogs, results[0].EntityType)
	}

	// Citation slice.
	results := searchArchive(archive, "cit-smith-baptism", false, "")
	require.NotEmpty(t, results)

	// Embedded Search fields use searches[i]. prefix.
	results = searchArchive(archive, "Smith William", false, "")
	require.NotEmpty(t, results)
	var foundQuery bool
	for _, r := range results {
		if r.Field == "searches[0].query" {
			foundQuery = true

			break
		}
	}
	assert.True(t, foundQuery, "expected searches[0].query match, got %+v", results)

	// Properties.
	results = searchArchive(archive, "high", false, "")
	require.NotEmpty(t, results)
	assert.Contains(t, results[0].Field, "properties.priority")
}

func TestSearchArchive_FindsStudyFields(t *testing.T) {
	archive := &glxlib.GLXFile{
		Studies: map[string]*glxlib.Study{
			"study-yorkshire-ops": {
				Title:      "Yorkshire One Place Study",
				Type:       "one_place_study",
				Status:     "active",
				DateRange:  "FROM 1750 TO 1900",
				Places:     []string{"place-yorkshire"},
				Sources:    []string{"source-yorkshire-register"},
				Notes:      glxlib.NoteList{"Coverage: 1750-1900"},
				Properties: map[string]any{"surname_variants": "Smyth, Smith"},
			},
		},
	}

	for _, q := range []string{"Yorkshire One Place", "one_place_study", "active", "FROM 1750", "Coverage", "place-yorkshire", "source-yorkshire-register", "Smyth"} {
		results := searchArchive(archive, q, false, "")
		require.NotEmpty(t, results, "should match %q", q)
		assert.Equal(t, glxlib.EntityTypeStudies, results[0].EntityType)
	}
}

func TestSearchSearchEntry_EveryFieldMatches(t *testing.T) {
	// Per-field coverage of the embedded Search struct: a permissive matchFn
	// triggers every if-branch so the field-path constants stay covered.
	archive := &glxlib.GLXFile{
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"research-log-coverage": {
				Searches: []glxlib.Search{
					{
						RepositoryID: "repo-x",
						SourceID:     "source-x",
						Collection:   "Collection X",
						Query:        "Query X",
						Date:         "2024-01",
						Result:       "found",
						CitationID:   "cit-x",
						Notes:        glxlib.NoteList{"Notes X"},
					},
				},
			},
		},
	}

	matchAll := func(_ string) bool { return true }
	results := searchResearchLogs(archive, matchAll)

	// Map field path → expected, so any future Search field that's added must
	// also get a path here or the test fails informatively.
	got := make(map[string]string)
	for _, r := range results {
		if strings.HasPrefix(r.Field, "searches[0].") {
			got[r.Field] = r.Value
		}
	}
	wantFields := []string{
		"searches[0].repository",
		"searches[0].source",
		"searches[0].collection",
		"searches[0].query",
		"searches[0].result",
		"searches[0].citation",
		"searches[0].date",
		"searches[0].notes",
	}
	for _, f := range wantFields {
		_, ok := got[f]
		assert.Truef(t, ok, "expected match for field %s, got %+v", f, got)
	}
}

func TestSearchArchive_TypeFilterIncludesResearchLogsAndStudies(t *testing.T) {
	archive := &glxlib.GLXFile{
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"research-log-1": {Title: "Common term", Status: "open"},
		},
		Studies: map[string]*glxlib.Study{
			"study-1": {Title: "Common term", Type: "one_name_study", Status: "active"},
		},
	}

	logsOnly := searchArchive(archive, "Common", false, "research_logs")
	require.NotEmpty(t, logsOnly, "should match research_logs with type filter")
	for _, r := range logsOnly {
		assert.Equal(t, glxlib.EntityTypeResearchLogs, r.EntityType)
	}

	studiesOnly := searchArchive(archive, "Common", false, "studies")
	require.NotEmpty(t, studiesOnly, "should match studies with type filter")
	for _, r := range studiesOnly {
		assert.Equal(t, glxlib.EntityTypeStudies, r.EntityType)
	}
}

func TestSearchArchive_DeterministicOrdering(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-multi": {Properties: map[string]any{
				"alpha": map[string]any{"value": "smith-target"},
				"beta":  map[string]any{"value": "jones-target"},
			}},
		},
	}

	first := searchArchive(archive, "target", false, "")
	require.Len(t, first, 2, "both map-shaped properties should match")

	for i := range 50 {
		got := searchArchive(archive, "target", false, "")
		assert.Equal(t, first, got, "search results must be identical across runs (run %d)", i)
	}
}

func TestSearchArchive_SkipsNonExtractableProperty(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-int": {Properties: map[string]any{
				"age": 42,
			}},
		},
	}

	results := searchArchive(archive, "42", false, "")

	assert.Empty(t, results)
}
