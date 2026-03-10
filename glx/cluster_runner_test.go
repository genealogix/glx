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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCluster_CensusHousehold(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
			"person-bob":  {Properties: map[string]any{"name": "Bob Jones"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-1860": {
				Type: glxlib.EventTypeCensus,
				Date: "1860",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
					{Person: "person-bob", Role: "boarder"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.Len(t, result.Associates, 2)
	assert.Equal(t, "person-john", result.PersonID)
	assert.Equal(t, "John Smith", result.PersonName)

	// Both Mary and Bob should be associates via census
	ids := map[string]bool{}
	for _, a := range result.Associates {
		ids[a.PersonID] = true
		assert.Greater(t, a.Score, 0)
		require.Len(t, a.Links, 1)
		assert.Equal(t, "census_household", a.Links[0].Type)
		assert.Equal(t, "event-census-1860", a.Links[0].EventID)
		assert.Equal(t, 1860, a.Links[0].Year)
	}
	assert.True(t, ids["person-mary"])
	assert.True(t, ids["person-bob"])
}

func TestBuildCluster_EventCoparticipant(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Green"}},
		},
		Events: map[string]*glxlib.Event{
			"event-marriage": {
				Type:  "marriage",
				Date:  "1852",
				Title: "Marriage of John Smith and Mary Green",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "groom"},
					{Person: "person-mary", Role: "bride"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-mary", result.Associates[0].PersonID)
	assert.Equal(t, "event_coparticipant", result.Associates[0].Links[0].Type)
	assert.Equal(t, "Marriage of John Smith and Mary Green", result.Associates[0].Links[0].Label)
}

func TestBuildCluster_PlaceOverlap(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-bob":  {Properties: map[string]any{"name": "Bob Jones"}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-john": {
				Type:         "birth",
				Date:         "1850",
				PlaceID:      "place-ironton",
				Participants: []glxlib.Participant{{Person: "person-john", Role: "principal"}},
			},
			"event-birth-bob": {
				Type:         "birth",
				Date:         "1852",
				PlaceID:      "place-ironton",
				Participants: []glxlib.Participant{{Person: "person-bob", Role: "principal"}},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-ironton": {Name: "Ironton, Sauk Co., WI"},
		},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-bob", result.Associates[0].PersonID)
	assert.Equal(t, "place_overlap", result.Associates[0].Links[0].Type)
	assert.Contains(t, result.Associates[0].Links[0].Label, "Ironton")
}

func TestBuildCluster_NoAssociates(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: map[string]*glxlib.Event{},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	assert.Empty(t, result.Associates)
	assert.Equal(t, "John Smith", result.PersonName)
}

func TestBuildCluster_PlaceFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
			"person-bob":  {Properties: map[string]any{"name": "Bob Jones"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-wi": {
				Type:    glxlib.EventTypeCensus,
				Date:    "1860",
				PlaceID: "place-ironton",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
			"event-census-ny": {
				Type:    glxlib.EventTypeCensus,
				Date:    "1860",
				PlaceID: "place-nyc",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-bob", Role: "boarder"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-ironton": {Name: "Ironton, Sauk Co., WI"},
			"place-nyc":     {Name: "New York City, NY"},
		},
	}

	result := buildCluster("person-john", archive, "place-ironton", 0, 0)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-mary", result.Associates[0].PersonID)
}

func TestBuildCluster_BeforeYearFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
			"person-bob":  {Properties: map[string]any{"name": "Bob Jones"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-1850": {
				Type: glxlib.EventTypeCensus,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
			"event-census-1870": {
				Type: glxlib.EventTypeCensus,
				Date: "1870",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-bob", Role: "boarder"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 1860, 0)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-mary", result.Associates[0].PersonID)
}

func TestBuildCluster_AfterYearFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
			"person-bob":  {Properties: map[string]any{"name": "Bob Jones"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-1850": {
				Type: glxlib.EventTypeCensus,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
			"event-census-1870": {
				Type: glxlib.EventTypeCensus,
				Date: "1870",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-bob", Role: "boarder"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 1860)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-bob", result.Associates[0].PersonID)
}

func TestBuildCluster_MultipleLinks_CompoundScore(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-1850": {
				Type: glxlib.EventTypeCensus,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
			"event-census-1860": {
				Type: glxlib.EventTypeCensus,
				Date: "1860",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
			"event-marriage": {
				Type: "marriage",
				Date: "1848",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "groom"},
					{Person: "person-mary", Role: "bride"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.Len(t, result.Associates, 1)
	// 2 census (3 each) + 1 event (2) = 8
	assert.Equal(t, 8, result.Associates[0].Score)
	assert.Len(t, result.Associates[0].Links, 3)
}

func TestBuildCluster_ScoreRanking(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john":    {Properties: map[string]any{"name": "John Smith"}},
			"person-mary":    {Properties: map[string]any{"name": "Mary Smith"}},
			"person-bob":     {Properties: map[string]any{"name": "Bob Jones"}},
			"person-charlie": {Properties: map[string]any{"name": "Charlie Brown"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type: glxlib.EventTypeCensus,
				Date: "1860",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
					{Person: "person-bob", Role: "boarder"},
				},
			},
			"event-marriage": {
				Type: "marriage",
				Date: "1855",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "groom"},
					{Person: "person-mary", Role: "bride"},
				},
			},
			"event-birth-charlie": {
				Type:         "birth",
				Date:         "1862",
				PlaceID:      "place-ironton",
				Participants: []glxlib.Participant{{Person: "person-charlie", Role: "principal"}},
			},
			"event-birth-john": {
				Type:         "birth",
				Date:         "1830",
				PlaceID:      "place-ironton",
				Participants: []glxlib.Participant{{Person: "person-john", Role: "principal"}},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-ironton": {Name: "Ironton"},
		},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.GreaterOrEqual(t, len(result.Associates), 2)
	// Mary should be first (census=3 + event=2 = 5)
	assert.Equal(t, "person-mary", result.Associates[0].PersonID)
	assert.Equal(t, 5, result.Associates[0].Score)
	// Bob should be second (census=3)
	assert.Equal(t, "person-bob", result.Associates[1].PersonID)
	assert.Equal(t, 3, result.Associates[1].Score)
}

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name  string
		links []associateLink
		want  int
	}{
		{
			name:  "census household",
			links: []associateLink{{Type: "census_household"}},
			want:  3,
		},
		{
			name:  "event coparticipant",
			links: []associateLink{{Type: "event_coparticipant"}},
			want:  2,
		},
		{
			name:  "place overlap",
			links: []associateLink{{Type: "place_overlap"}},
			want:  1,
		},
		{
			name: "compound",
			links: []associateLink{
				{Type: "census_household"},
				{Type: "census_household"},
				{Type: "event_coparticipant"},
			},
			want: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, computeScore(tt.links))
		})
	}
}

func TestYearsOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		want bool
	}{
		{"exact match", []int{1860}, []int{1860}, true},
		{"within window", []int{1855}, []int{1860}, true},
		{"edge of window", []int{1850}, []int{1860}, true},
		{"outside window", []int{1849}, []int{1860}, false},
		{"empty a", []int{}, []int{1860}, false},
		{"empty b", []int{1860}, []int{}, false},
		{"multiple overlap", []int{1830, 1860}, []int{1850, 1870}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, yearsOverlap(tt.a, tt.b))
		})
	}
}

func TestYearInRange(t *testing.T) {
	tests := []struct {
		name                  string
		year, before, after   int
		want                  bool
	}{
		{"no filter", 1860, 0, 0, true},
		{"zero year passes", 0, 1850, 1840, true},
		{"before filter pass", 1840, 1860, 0, true},
		{"before filter fail", 1870, 1860, 0, false},
		{"after filter pass", 1870, 0, 1860, true},
		{"after filter fail", 1840, 0, 1860, false},
		{"both filters pass", 1855, 1860, 1850, true},
		{"both filters fail before", 1870, 1860, 1850, false},
		{"both filters fail after", 1840, 1860, 1850, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, yearInRange(tt.year, tt.before, tt.after))
		})
	}
}

func TestClusterExtractYear(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1860", 1860},
		{"ABT 1832", 1832},
		{"BEF 1900", 1900},
		{"12 MAR 1855", 1855},
		{"", 0},
		{"no year", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, clusterExtractYear(tt.input))
		})
	}
}

func TestFormatYearRange(t *testing.T) {
	tests := []struct {
		name  string
		years []int
		want  string
	}{
		{"empty", []int{}, "?"},
		{"single", []int{1860}, "1860"},
		{"range", []int{1855, 1870, 1860}, "1855–1870"},
		{"same year twice", []int{1860, 1860}, "1860"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatYearRange(tt.years))
		})
	}
}

func TestResolvePersonForCluster_ExactID(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}

	id, err := resolvePersonForCluster(archive, "person-john")
	require.NoError(t, err)
	assert.Equal(t, "person-john", id)
}

func TestResolvePersonForCluster_NameSearch(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Green"}},
		},
	}

	id, err := resolvePersonForCluster(archive, "john")
	require.NoError(t, err)
	assert.Equal(t, "person-john", id)
}

func TestResolvePersonForCluster_Ambiguous(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john-sr": {Properties: map[string]any{"name": "John Smith Sr."}},
			"person-john-jr": {Properties: map[string]any{"name": "John Smith Jr."}},
		},
	}

	_, err := resolvePersonForCluster(archive, "John Smith")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple persons match")
}

func TestResolvePersonForCluster_NotFound(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}

	_, err := resolvePersonForCluster(archive, "nobody")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestResolvePersonForCluster_NilPerson(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": nil,
			"person-mary": {Properties: map[string]any{"name": "Mary Green"}},
		},
	}

	// nil person should not be returned as exact ID match
	_, err := resolvePersonForCluster(archive, "person-john")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestPlaceIsDescendant(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-ironton":  {Name: "Ironton", ParentID: "place-sauk-co"},
			"place-sauk-co":  {Name: "Sauk County", ParentID: "place-wi"},
			"place-wi":       {Name: "Wisconsin", ParentID: ""},
			"place-nyc":      {Name: "New York City", ParentID: "place-ny"},
			"place-ny":       {Name: "New York", ParentID: ""},
		},
	}

	assert.True(t, placeIsDescendant("place-ironton", "place-sauk-co", archive))
	assert.True(t, placeIsDescendant("place-ironton", "place-wi", archive))
	assert.False(t, placeIsDescendant("place-ironton", "place-ny", archive))
	assert.False(t, placeIsDescendant("place-wi", "place-ironton", archive))
	assert.False(t, placeIsDescendant("nonexistent", "place-wi", archive))
}

func TestBuildCluster_PlaceDescendantFilter(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type:    glxlib.EventTypeCensus,
				Date:    "1860",
				PlaceID: "place-ironton",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-ironton": {Name: "Ironton", ParentID: "place-sauk-co"},
			"place-sauk-co": {Name: "Sauk County", ParentID: "place-wi"},
			"place-wi":      {Name: "Wisconsin"},
		},
	}

	// Filtering by parent place should include events at child places
	result := buildCluster("person-john", archive, "place-wi", 0, 0)
	require.Len(t, result.Associates, 1)
	assert.Equal(t, "person-mary", result.Associates[0].PersonID)
}

func TestClusterEventHasParticipant(t *testing.T) {
	event := &glxlib.Event{
		Participants: []glxlib.Participant{
			{Person: "person-john", Role: "head"},
			{Person: "person-mary", Role: "wife"},
		},
	}

	assert.True(t, clusterEventHasParticipant("person-john", event))
	assert.True(t, clusterEventHasParticipant("person-mary", event))
	assert.False(t, clusterEventHasParticipant("person-bob", event))
	assert.False(t, clusterEventHasParticipant("person-john", nil))
}

func TestBuildCluster_NilEventSkipped(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: map[string]*glxlib.Event{
			"event-nil": nil,
		},
		Places: map[string]*glxlib.Place{},
	}

	// Should not panic on nil event
	result := buildCluster("person-john", archive, "", 0, 0)
	assert.Empty(t, result.Associates)
}

func TestBuildCluster_CensusWithTitle(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Smith"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type:  glxlib.EventTypeCensus,
				Date:  "1860",
				Title: "1860 Census — Smith Household",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "head"},
					{Person: "person-mary", Role: "wife"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	result := buildCluster("person-john", archive, "", 0, 0)

	require.Len(t, result.Associates, 1)
	assert.Equal(t, "1860 Census — Smith Household", result.Associates[0].Links[0].Label)
}

func TestShowCluster_ArchiveNotFound(t *testing.T) {
	err := showCluster("/nonexistent/path", "person-john", "", 0, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestShowCluster_PersonNotFound(t *testing.T) {
	// Create a temp single-file archive
	dir := t.TempDir()
	archivePath := dir + "/archive.glx"
	writeTestArchive(t, archivePath, &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	})

	err := showCluster(archivePath, "nobody", "", 0, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestHasEventLinkAtPlace(t *testing.T) {
	linkMap := map[string][]associateLink{
		"person-mary": {
			{Type: "census_household", EventID: "event-1"},
		},
		"person-bob": {
			{Type: "place_overlap"},
		},
	}

	assert.True(t, hasEventLinkAtPlace("person-mary", linkMap))
	assert.False(t, hasEventLinkAtPlace("person-bob", linkMap))
	assert.False(t, hasEventLinkAtPlace("person-unknown", linkMap))
}

func writeTestArchive(t *testing.T, path string, archive *glxlib.GLXFile) {
	t.Helper()
	if err := writeSingleFileArchive(path, archive, false); err != nil {
		t.Fatalf("failed to write test archive: %v", err)
	}
}
