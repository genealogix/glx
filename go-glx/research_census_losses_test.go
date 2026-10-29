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

package glx

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func censusLossTestArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{"p": {Properties: map[string]any{"name": "Research subject"}}},
		Places: map[string]*Place{
			"us":       {Name: "United States", Type: PlaceTypeCountry},
			"lost":     {Name: "Indiana Territory", Type: placeTypeTerritory, ParentID: "us"},
			"survives": {Name: "North Carolina", Type: PlaceTypeState, ParentID: "us"},
			"partial":  {Name: "Tennessee", Type: PlaceTypeState, ParentID: "us"},
			"foreign":  {Name: "Ireland", Type: PlaceTypeCountry},
		},
		Events: map[string]*Event{
			"birth":     {Type: EventTypeBirth, Date: "1770", PlaceID: "survives", Participants: []Participant{{Person: "p", Role: "principal"}}},
			"residence": {Type: EventTypeResidence, Date: "1810", PlaceID: "lost", Participants: []Participant{{Person: "p"}}},
			"death":     {Type: EventTypeDeath, Date: "1830", PlaceID: "lost", Participants: []Participant{{Person: "p", Role: "principal"}}},
		},
	}
}

func TestCensusSurvival_ConservativeBrackets(t *testing.T) {
	archive := censusLossTestArchive()
	schedule := censusSchedulesByCountry[CensusCountryUnitedStates]
	cases := []struct {
		name   string
		places []CensusDatedPlace
		lost   bool
		note   string
	}{
		{"total on both sides", []CensusDatedPlace{{1805, "lost"}, {1815, "lost"}}, true, ""},
		{"mixed sides", []CensusDatedPlace{{1805, "lost"}, {1815, "survives"}}, false, "schedules lost"},
		{"same year conflicting locations", []CensusDatedPlace{{1810, "lost"}, {1810, "survives"}}, false, "schedules lost"},
		{"unknown state on a bracket", []CensusDatedPlace{{1805, "lost"}, {1815, "us"}}, false, "schedules lost"},
		{"missing place on a bracket", []CensusDatedPlace{{1805, "lost"}, {1815, "missing"}}, false, "schedules lost"},
		{"partial on a bracket", []CensusDatedPlace{{1805, "lost"}, {1815, "partial"}}, false, "schedules mostly lost"},
		{"foreign does not replace domestic bracket", []CensusDatedPlace{{1805, "lost"}, {1810, "foreign"}}, true, ""},
		{"no places", nil, false, ""},
		{"no date", []CensusDatedPlace{{0, "lost"}}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := schedule.Survival(1810, tc.places, archive)
			require.Equal(t, tc.lost, got.Lost)
			if tc.note == "" {
				require.Empty(t, got.Note)
			} else {
				require.Contains(t, got.Note, tc.note)
			}
		})
	}
	require.Equal(t, CensusYearSurvival{}, schedule.Survival(1810, []CensusDatedPlace{{1810, "lost"}}, nil))
}

func TestCensusLosses_PublicCoveragePreservesEvidenceAndDenominator(t *testing.T) {
	for _, tc := range []struct {
		year int
		name string
	}{
		{1810, "Indiana Territory"},
		{1800, "Georgia"},
		{1810, "Ohio"},
		{1810, "Michigan Territory"},
		{1820, "New Jersey"},
	} {
		t.Run(strconv.Itoa(tc.year)+" "+tc.name, func(t *testing.T) {
			testCensusLossCoverage(t, tc.year, tc.name)
		})
	}
}

func testCensusLossCoverage(t *testing.T, year int, jurisdiction string) {
	t.Helper()
	archive := censusLossTestArchive()
	archive.Places["lost"].Name = jurisdiction
	archive.Places["lost"].Type = PlaceTypeState
	if strings.HasSuffix(jurisdiction, " Territory") {
		archive.Places["lost"].Type = placeTypeTerritory
	}
	archive.Events["residence"].Date = DateString(strconv.Itoa(year))
	label := strconv.Itoa(year) + " US Census"
	baseline, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	for _, rec := range baseline.Records {
		require.NotContains(t, rec.Label, label)
	}
	require.Len(t, baseline.Records, baseline.Expected)
	require.Zero(t, baseline.Found)

	archive.Sources = map[string]*Source{"census": {Type: SourceTypeCensus, Title: label, Date: DateString(strconv.Itoa(year))}}
	archive.Citations = map[string]*Citation{"search": {SourceID: "census"}}
	archive.ResearchLogs = map[string]*ResearchLog{"log": {
		Subject:   &EntityRef{Person: "p"},
		Searches:  []Search{{SourceID: "census", CitationID: "search", Query: "Research subject", Result: "not_found"}},
		Citations: []string{"search"},
	}}
	logged, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	require.Equal(t, baseline, logged, "negative searches are provenance, not found records")
	proof, err := BuildProof(archive, "p", QuestionIdentity, ProofOptions{})
	require.NoError(t, err)
	require.Len(t, proof.Searches, 1)
	require.Equal(t, "census", proof.Searches[0].Source)
	require.Equal(t, "not_found", proof.Searches[0].Result)
	for _, gap := range proof.Gaps {
		require.NotContains(t, gap.Label, label)
	}
	require.Equal(t, "search", archive.ResearchLogs["log"].Searches[0].CitationID)

	archive.Media = map[string]*Media{"image": {Source: "census"}}
	archive.Assertions = map[string]*Assertion{"found": {Subject: EntityRef{Person: "p"}, Media: []string{"image"}}}
	found, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	wantExpected := baseline.Expected + 1
	require.Equal(t, wantExpected, found.Expected)
	require.Equal(t, 1, found.Found)
	var census *CoverageRecord
	for i := range found.Records {
		if strings.HasPrefix(found.Records[i].Label, label) {
			census = &found.Records[i]
		}
	}
	require.NotNil(t, census)
	require.True(t, census.Found)
	require.Equal(t, "census", census.SourceRef)
	require.Equal(t, "not_found", archive.ResearchLogs["log"].Searches[0].Result)
}

func TestCensusLosses_OwnedSnapshotsAndDatedPlaces(t *testing.T) {
	archive := censusLossTestArchive()
	opts := CoverageOptions{CensusCountry: CensusCountryUnitedStates}
	schedules, err := CensusSchedulesForPerson(archive, "p", opts)
	require.NoError(t, err)
	places, err := DatedPlacesForPerson(archive, "p")
	require.NoError(t, err)
	require.Len(t, places, 3)
	index, err := PersonDatedPlaceIndex(archive)
	require.NoError(t, err)
	require.Equal(t, places, index["p"])
	index["p"][0].PlaceID = "changed"
	require.NotEqual(t, "changed", places[0].PlaceID)
	delete(schedules[0].losses[1810], jurisdictionKey("Indiana Territory"))
	next, err := CensusSchedulesForPerson(archive, "p", opts)
	require.NoError(t, err)
	require.True(t, next[0].Survival(1810, places, archive).Lost)
	require.Equal(t, "survives", archive.Events["birth"].PlaceID)
	_, err = PersonDatedPlaceIndex(nil)
	require.ErrorIs(t, err, ErrNilArchive)
	_, err = DatedPlacesForPerson(archive, "missing")
	require.ErrorIs(t, err, ErrPersonNotFound)
}
