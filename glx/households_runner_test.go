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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// newHouseholdsArchive builds an archive with an 1820 head-only census (James
// named; his wife Elizabeth and daughter Mary counted as tick marks; a tally;
// two page neighbors) and an 1860 named census of the Webbs (no explicit
// head, ages out of order) at a child place of the county.
func newHouseholdsArchive() *glxlib.GLXFile {
	unnamed := map[string]any{glxlib.ParticipantPropertyNamed: false}

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-james":   {Properties: map[string]any{"name": "James Little"}},
			"person-eliz":    {Properties: map[string]any{"name": "Elizabeth Starr"}},
			"person-mary":    {Properties: map[string]any{"name": "Mary Little"}},
			"person-henry":   {Properties: map[string]any{"name": "Henry Jeffries"}},
			"person-robert":  {Properties: map[string]any{"name": "Robert Webb"}},
			"person-jane":    {Properties: map[string]any{"name": "Jane Webb"}},
			"person-emma":    {Properties: map[string]any{"name": "Emma Webb"}},
			"person-charles": {Properties: map[string]any{"name": "Charles Webb"}},
		},
		Places: map[string]*glxlib.Place{
			"place-us":        {Name: "United States"},
			"place-wythe":     {Name: "Wythe County, Virginia", ParentID: "place-us"},
			"place-millbrook": {Name: "Millbrook", ParentID: "place-wythe"},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-mary": {
				Type: glxlib.EventTypeBirth, Date: "1806", PlaceID: "place-wythe",
				Participants: []glxlib.Participant{{Person: "person-mary", Role: glxlib.ParticipantRolePrincipal}},
			},
			"event-census-1820": {
				Title: "1820 Census — Little Household", Type: glxlib.EventTypeCensus, Date: "1820", PlaceID: "place-wythe",
				Participants: []glxlib.Participant{
					{Person: "person-eliz", Role: glxlib.ParticipantRoleHouseholdMember, Properties: unnamed},
					{Person: "person-james", Role: glxlib.ParticipantRolePrincipal},
					{Person: "person-mary", Role: glxlib.ParticipantRoleHouseholdMember, Properties: unnamed},
				},
				Household: &glxlib.Household{Tally: []glxlib.HouseholdTallyRow{
					{Sex: "male", AgeFrom: new(45), Count: 1, Status: "free white"},
					{Sex: "female", AgeFrom: new(10), AgeTo: new(15), Count: 2},
					{Sex: "female", AgeTo: new(9), Count: 1},
					{Count: 1, Status: "engaged in agriculture"},
				}},
				Neighbors: []glxlib.EventNeighbor{
					{Name: "Henry Jeffries", Person: "person-henry", Position: "previous_household", Page: "12", Line: "3"},
					{Name: "James M Clark", Position: "next_household"},
				},
			},
			"event-census-1860": {
				Type: glxlib.EventTypeCensus, Date: "1860", PlaceID: "place-millbrook",
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: glxlib.ParticipantRoleSubject, Properties: map[string]any{"age_at_event": "45"}},
					{Person: "person-charles", Role: glxlib.ParticipantRoleSubject, Properties: map[string]any{"age_at_event": "2"}},
					{Person: "person-jane", Role: glxlib.ParticipantRoleSubject, Properties: map[string]any{"age_at_event": "28", glxlib.ParticipantPropertyRelationshipToHead: "wife"}},
					{Person: "person-emma", Role: glxlib.ParticipantRoleSubject, Properties: map[string]any{"age_at_event": 8}},
				},
			},
		},
	}
}

func memberIDs(h *household) []string {
	ids := make([]string, 0, len(h.Members))
	for _, m := range h.Members {
		ids = append(ids, m.PersonID)
	}

	return ids
}

func TestBuildHouseholds_PersonHeadOnlyCensus(t *testing.T) {
	archive := newHouseholdsArchive()

	result := buildHouseholds(archive, "person-mary", householdsOptions{Neighbors: true})

	assert.Equal(t, "Mary Little", result.PersonName)
	require.Len(t, result.Households, 1)
	h := result.Households[0]
	assert.Equal(t, "event-census-1820", h.EventID)
	assert.Equal(t, 1820, h.Year)
	assert.Equal(t, "Wythe County, Virginia", h.PlaceName)
	// Head first even though listed second; unnamed members keep event order.
	assert.Equal(t, []string{"person-james", "person-eliz", "person-mary"}, memberIDs(&h))
	assert.True(t, h.Members[0].Head)
	assert.True(t, h.Members[0].Named)
	assert.False(t, h.Members[1].Named)

	labels := make([]string, 0, len(h.Tally))
	for _, row := range h.Tally {
		labels = append(labels, row.Label)
	}
	assert.Equal(t, []string{"1 male 45+ (free white)", "2 female 10–15", "1 female under 10", "1 engaged in agriculture"}, labels)

	require.Len(t, h.Neighbors, 2)
	assert.Equal(t, "person-henry", h.Neighbors[0].PersonID)
	assert.Equal(t, "next_household", h.Neighbors[1].Position)
}

func TestBuildHouseholds_HeadFirstThenAge(t *testing.T) {
	archive := newHouseholdsArchive()

	result := buildHouseholds(archive, "person-jane", householdsOptions{})

	require.Len(t, result.Households, 1)
	h := result.Households[0]
	// No relationship_to_head: head marks Robert (first subject); the rest by
	// age, oldest first (Emma's age is a bare YAML number).
	assert.Equal(t, []string{"person-robert", "person-jane", "person-emma", "person-charles"}, memberIDs(&h))
	assert.Equal(t, "wife", h.Members[1].RelationshipToHead)
	assert.Equal(t, "8", h.Members[2].Age)
	assert.Nil(t, h.Neighbors, "neighbors are only collected with --neighbors")
}

func TestBuildHouseholds_RelationshipToHeadWins(t *testing.T) {
	archive := newHouseholdsArchive()
	ev := archive.Events["event-census-1860"]
	ev.Participants[3].Properties[glxlib.ParticipantPropertyRelationshipToHead] = "Head"

	h := buildHouseholds(archive, "person-emma", householdsOptions{}).Households[0]

	assert.Equal(t, "person-emma", h.Members[0].PersonID)
	assert.True(t, h.Members[0].Head)
}

func TestBuildHouseholds_PlaceAndYearFilters(t *testing.T) {
	archive := newHouseholdsArchive()

	all := buildHouseholds(archive, "", householdsOptions{PlaceID: "place-us"})
	require.Len(t, all.Households, 2, "descendant places are included")
	assert.Equal(t, "event-census-1820", all.Households[0].EventID, "ordered by year")

	only1860 := buildHouseholds(archive, "", householdsOptions{PlaceID: "place-wythe", Year: 1860})
	require.Len(t, only1860.Households, 1)
	assert.Equal(t, "event-census-1860", only1860.Households[0].EventID)

	none := buildHouseholds(archive, "person-mary", householdsOptions{Year: 1860})
	assert.Empty(t, none.Households)
}

func writeHouseholdsArchive(t *testing.T) string {
	t.Helper()
	archive := newHouseholdsArchive()
	data, err := glxlib.NewSerializer(&glxlib.SerializerOptions{Pretty: true, Indent: "  "}).SerializeSingleFileBytes(archive)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return path
}

func TestShowHouseholds_TextAndJSON(t *testing.T) {
	path := writeHouseholdsArchive(t)

	io, out, _ := TestIOStreams()
	require.NoError(t, showHouseholds(io, path, householdsOptions{PersonQuery: "Mary Little", Neighbors: true}))
	text := out.String()
	assert.Contains(t, text, "Households for Mary Little (person-mary):")
	assert.Contains(t, text, "James Little (person-james) — head")
	assert.Contains(t, text, "Elizabeth Starr (person-eliz) — household member, counted, not named")
	assert.Contains(t, text, "2 female 10–15")
	assert.Contains(t, text, "Henry Jeffries (person-henry) — previous household, page 12, line 3")

	io, out, _ = TestIOStreams()
	require.NoError(t, showHouseholds(io, path, householdsOptions{PlaceID: "place-wythe", Year: 1820, Format: "json"}))
	var result householdsResult
	require.NoError(t, json.Unmarshal(out.Bytes(), &result))
	require.Len(t, result.Households, 1)
	assert.Equal(t, 1820, result.Year)
	assert.Len(t, result.Households[0].Tally, 4)
}

func TestShowHouseholds_Errors(t *testing.T) {
	path := writeHouseholdsArchive(t)
	io, _, _ := TestIOStreams()

	require.ErrorIs(t, showHouseholds(io, path, householdsOptions{}), ErrHouseholdsNoTarget)
	require.ErrorIs(t, showHouseholds(io, path, householdsOptions{PersonQuery: "person-mary", Format: "xml"}), ErrHouseholdsUnknownFormat)
	require.ErrorIs(t, showHouseholds(io, path, householdsOptions{PlaceID: "place-nowhere"}), ErrHouseholdsPlaceNotFound)
	require.ErrorIs(t, showHouseholds(io, path, householdsOptions{PersonQuery: "Nobody"}), ErrNoPersonMatch)
}

func TestFormatTallyRow(t *testing.T) {
	tests := []struct {
		row  glxlib.HouseholdTallyRow
		want string
	}{
		{glxlib.HouseholdTallyRow{Sex: "male", AgeFrom: new(16), AgeTo: new(25), Count: 1}, "1 male 16–25"},
		{glxlib.HouseholdTallyRow{Count: 3, Status: "enslaved"}, "3 enslaved"},
		{glxlib.HouseholdTallyRow{Sex: "female", AgeTo: new(9), Count: 2, Status: "free colored"}, "2 female under 10 (free colored)"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatTallyRow(&tt.row))
	}
}

func TestBuildCluster_CensusNeighbors(t *testing.T) {
	archive := newHouseholdsArchive()

	fromHead := buildCluster("person-james", archive, "", 0, 0)
	var henry *associate
	for i := range fromHead.Associates {
		if fromHead.Associates[i].PersonID == "person-henry" {
			henry = &fromHead.Associates[i]
		}
	}
	require.NotNil(t, henry, "a neighbor linked to a person is an associate")
	require.Len(t, henry.Links, 1)
	assert.Equal(t, "census_neighbor", henry.Links[0].Type)
	assert.Equal(t, "neighbor (previous household)", henry.Links[0].Role)
	assert.Equal(t, 2, henry.Score)

	// The reverse direction links the neighbor to the household's head, with
	// the position seen from the neighbor's side.
	fromNeighbor := buildCluster("person-henry", archive, "", 0, 0)
	require.Len(t, fromNeighbor.Associates, 1)
	assert.Equal(t, "person-james", fromNeighbor.Associates[0].PersonID)
	assert.Equal(t, "neighbor (next household)", fromNeighbor.Associates[0].Links[0].Role)
}

// A person counted in a head-only census household (named: false) is found in
// that census: analyze does not suggest searching for it and coverage ticks
// the row (#1332).
func TestUnnamedHouseholdMemberCountsAsCensusFound(t *testing.T) {
	setCensusFallback(t, countryUnitedStates)
	archive := newHouseholdsArchive()
	archive.Sources = map[string]*glxlib.Source{"source-1820": {Title: "1820 Census", Type: glxlib.SourceTypeCensus}}
	archive.Citations = map[string]*glxlib.Citation{"citation-1820": {SourceID: "source-1820"}}
	archive.Assertions = map[string]*glxlib.Assertion{
		"assertion-1820": {Subject: glxlib.EntityRef{Event: "event-census-1820"}, Property: "date", Value: "1820", Citations: []string{"citation-1820"}},
	}

	maryIssues := 0
	for _, issue := range suggestCensusSearches(archive) {
		if issue.Person == "person-mary" {
			maryIssues++
			assert.NotContains(t, issue.Message, "1820", "1820 is satisfied by the head's household event")
		}
	}
	require.Positive(t, maryIssues, "other census years are still suggested, so the check above is not vacuous")

	result := buildCoverage("person-mary", archive.Persons["person-mary"], archive)
	found := false
	for _, r := range result.Records {
		if r.Category == "census" && strings.HasPrefix(r.Label, "1820") {
			found = r.Found
		}
	}
	assert.True(t, found, "coverage ticks the 1820 census for an unnamed household member")
}
