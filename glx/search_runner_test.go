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
		if r.EntityType == "persons" && r.EntityID == "person-jane" {
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
		if r.EntityType == "places" && r.EntityID == "place-millbrook" {
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
		if r.EntityType == "events" && r.EntityID == "event-census" {
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
		if r.EntityType == "sources" && r.EntityID == "source-1860" {
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
		if r.EntityType == "assertions" && r.EntityID == "a-1" {
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
				if r.EntityType == "repositories" && r.EntityID == "repo-archives" && r.Field == tc.wantField {
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
		assert.Equal(t, "places", r.EntityType, "filtered results should only contain places")
	}

	// Non-place results should exist in unfiltered
	hasNonPlace := false
	for _, r := range allResults {
		if r.EntityType != "places" {
			hasNonPlace = true
		}
	}
	assert.True(t, hasNonPlace, "unfiltered results should include non-place entities")
}

func TestSearchArchive_FindsResearchLogObjectiveAndSearchFields(t *testing.T) {
	archive := &glxlib.GLXFile{
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"log-jane-parents": {
				Objective:      "Identify Jane Miller's parents in 1860 Wisconsin",
				Status:         "in_progress",
				RelatedPersons: []string{"person-jane"},
				Searches: []glxlib.ResearchLogSearch{
					{
						Repository:  "repo-fs",
						Collection:  "1860 US Census",
						SearchTerms: "Jane Miller Hartford",
						Result:      "not_found",
					},
				},
			},
		},
	}

	objectiveResults := searchArchive(archive, "1860 Wisconsin", false, "")
	hasObjective := false
	for _, r := range objectiveResults {
		if r.EntityType == "research_logs" && r.Field == "objective" {
			hasObjective = true
		}
	}
	assert.True(t, hasObjective, "should find objective text match")

	termsResults := searchArchive(archive, "Hartford", false, "")
	hasTerms := false
	for _, r := range termsResults {
		if r.EntityType == "research_logs" && r.Field == "searches[0].search_terms" {
			hasTerms = true
		}
	}
	assert.True(t, hasTerms, "should find search_terms match inside searches[0]")

	typeFilterResults := searchArchive(archive, "Identify", false, "research_logs")
	require.NotEmpty(t, typeFilterResults)
	for _, r := range typeFilterResults {
		assert.Equal(t, "research_logs", r.EntityType)
	}
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
	assert.Equal(t, "persons", results[0].EntityType)
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
	assert.Equal(t, "persons", results[0].EntityType)
	assert.Equal(t, "person-mapshaped", results[0].EntityID)
	assert.Equal(t, "properties.religion", results[0].Field)
	assert.Equal(t, "Methodist", results[0].Value)
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
