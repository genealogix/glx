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
				"name": "Jane Miller", "born_at": "place-millbrook",
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
		Repositories:  map[string]*glxlib.Repository{},
		Media:         map[string]*glxlib.Media{},
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
	err := showSearch(".", "", false, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestShowSearch_InvalidType(t *testing.T) {
	err := showSearch(".", "test", false, "invalid_type")
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
		err := showSearch(archivePath, "Millbrook", false, "places")
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Places")
	assert.NotContains(t, output, "Persons")
}
