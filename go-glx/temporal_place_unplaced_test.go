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
	"testing"

	"github.com/stretchr/testify/require"
)

// unplacedDateArchive is Wayne County, Indiana Territory until statehood on
// 11 Dec 1816 and Indiana after, with Indiana as the effective default.
func unplacedDateArchive() *GLXFile {
	wayne := &Place{Name: "Wayne County", Type: PlaceTypeCounty}
	wayne.SetParentHistory([]PlaceParentPeriod{
		{Value: "territory", Date: "TO 1816-12-10"},
		{Value: "indiana", Date: "FROM 1816-12-11"},
	})

	return &GLXFile{Places: map[string]*Place{
		"us":        {Name: "United States", Type: PlaceTypeCountry},
		"territory": {Name: "Indiana Territory", Type: placeTypeTerritory, ParentID: "us"},
		"indiana":   {Name: "Indiana", Type: PlaceTypeState, ParentID: "us"},
		"wayne":     wayne,
	}}
}

// A valid date that cannot be placed on the timeline rules no period out, so
// a "may" query must not treat it as excluding every temporal ancestor.
func TestPlaceMayHaveAncestorAt_UnplacedDate(t *testing.T) {
	archive := unplacedDateArchive()
	for _, date := range []DateString{"HEBREW 10 TSH 5600", "FRENCH_R 1 VEND 0012", "1900-02-29"} {
		require.True(t, archive.PlaceMayHaveAncestorAt("wayne", "indiana", date), date)
		require.True(t, archive.PlaceMayHaveAncestorAt("wayne", "territory", date), date)
		require.True(t, archive.PlaceMayHaveAncestorAt("wayne", "us", date), date)
		require.False(t, archive.PlaceHasAncestorAt("wayne", "indiana", date), "certainty still needs a placed date: %s", date)
	}
}

// A projection for an unplaced date follows the effective defaults instead of
// stopping at the first temporal place (GEDCOM PLAC, canonical paths).
func TestPlacePathAt_UnplacedDateFollowsDefaults(t *testing.T) {
	archive := unplacedDateArchive()
	want := []string{"wayne", "indiana", "us"}
	require.Equal(t, want, archive.PlacePathAt("wayne", "FRENCH_R 1 VEND 0012"))
	require.Equal(t, want, archive.PlacePathAt("wayne", "HEBREW 10 TSH 5600"))
	require.Equal(t, []string{"wayne", "territory", "us"}, archive.PlacePathAt("wayne", "1814"))

	expCtx := &ExportContext{GLX: archive}
	require.Equal(t, "Wayne County, Indiana, United States", buildPlaceString("wayne", "FRENCH_R 1 VEND 0012", expCtx))
}

func TestParentAt_NoPlaceableEntryFallsBackToDefault(t *testing.T) {
	p := &Place{Name: "Town"}
	p.SetParentHistory([]PlaceParentPeriod{
		{Value: "place-a", Date: "FROM 1900-02-29 TO 1910"},
		{Value: "place-b", Date: "HEBREW 5660"},
	})
	require.NotEmpty(t, p.EffectiveParentID())
	require.Equal(t, p.EffectiveParentID(), p.ParentAt("1905"))
}

// An open lookup is read at its boundary, not ranked by an unbounded overlap.
func TestParentAt_OpenLookupUsesBoundary(t *testing.T) {
	wayne := unplacedDateArchive().Places["wayne"]
	require.Equal(t, "indiana", wayne.ParentAt("BEF 1820"))
	require.Equal(t, "territory", wayne.ParentAt("BEF 1815"))
	require.Equal(t, "territory", wayne.ParentAt("AFT 1810"))
	require.Equal(t, "indiana", wayne.ParentAt("AFT 1817"))

	// Nothing covers the boundary: the whole window still decides.
	town := &Place{Name: "Town"}
	town.SetParentHistory([]PlaceParentPeriod{{Value: "early", Date: "FROM 1700 TO 1750"}})
	require.Equal(t, "early", town.ParentAt("BEF 1820"))
}

// A path that was foreign at the observation date keeps the schedule from
// being called lost, even when the census year has only domestic paths.
func TestCensusSurvival_ObservationForeignAlternativeIsKept(t *testing.T) {
	archive := censusLossTestArchive()
	town := &Place{Name: "Border Town", Type: "town"}
	town.SetParentHistory([]PlaceParentPeriod{
		{Value: "foreign", Date: "FROM 1800 TO 1807"},
		{Value: "lost", Date: "FROM 1806"},
	})
	archive.Places["border"] = town
	schedule := censusSchedulesByCountry[CensusCountryUnitedStates]

	got := schedule.SurvivalAt(1810, []CensusPlaceObservation{
		{PlaceID: "border", Date: "1806"},
		{PlaceID: "lost", Date: "1815"},
	}, archive)
	require.False(t, got.Lost, "the 1806 observation may have been outside the lost jurisdiction")

	control := schedule.SurvivalAt(1810, []CensusPlaceObservation{
		{PlaceID: "border", Date: "1809"},
		{PlaceID: "lost", Date: "1815"},
	}, archive)
	require.True(t, control.Lost, "wholly domestic at both dates")
}

// The coverage index and the per-person lookup decide presence the same way,
// including for a person listed twice under different roles.
func TestCensusPlaceIndexMatchesPerPersonLookup(t *testing.T) {
	archive := censusLossTestArchive()
	archive.Events["will"] = &Event{Type: EventTypeProbate, Date: "1820", PlaceID: "survives", Participants: []Participant{
		{Person: "p", Role: "beneficiary"},
		{Person: "p", Role: "principal"},
		{Person: "q", Role: "beneficiary"},
	}}
	index := personCensusPlaceIndex(archive)
	for _, person := range []string{"p", "q"} {
		require.ElementsMatch(t, censusPlacesForPerson(archive, person), index[person], person)
	}
}
