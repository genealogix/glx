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
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// temporalParentArchive is the Indiana example from #225: Wayne County was
// in Indiana Territory until statehood on 11 Dec 1816, then in Indiana.
func temporalParentArchive() *glxlib.GLXFile {
	wayne := &glxlib.Place{Name: "Wayne County", Type: glxlib.PlaceTypeCounty}
	wayne.SetParentHistory([]glxlib.PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "FROM 1811 TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	})

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{glxlib.PersonPropertyName: "John Smith"}},
		},
		Places: map[string]*glxlib.Place{
			"place-usa":               {Name: "United States", Type: glxlib.PlaceTypeCountry},
			"place-indiana-territory": {Name: "Indiana Territory", ParentID: "place-usa"},
			"place-indiana":           {Name: "Indiana", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
			"place-wayne-in":          wayne,
		},
		Events: map[string]*glxlib.Event{
			"event-marriage": {
				Type: glxlib.EventTypeMarriage, Date: "1814-04-02", PlaceID: "place-wayne-in",
				Participants: []glxlib.Participant{{Person: "person-1", Role: "groom"}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}
}

func TestPlaceFullNameAt_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.Equal(t, "Wayne County, Indiana Territory, United States", placeFullNameAt("place-wayne-in", "1814", archive))
	assert.Equal(t, "Wayne County, Indiana, United States", placeFullNameAt("place-wayne-in", "1817", archive))
	assert.Equal(t, "Wayne County, Indiana, United States", placeFullName("place-wayne-in", archive), "no date: default parent")
}

func TestBuildCanonicalPathAt_TemporalParent(t *testing.T) {
	places := temporalParentArchive().Places

	assert.Equal(t, "Wayne County, Indiana Territory, United States", buildCanonicalPathAt("place-wayne-in", "1813", places))
	assert.Equal(t, "Wayne County, Indiana, United States", buildCanonicalPath("place-wayne-in", places))
}

func TestMigrationEntry_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.Equal(t, "Wayne County, Indiana Territory, United States", newMigrationEntry("1814", "place-wayne-in", "Marriage", archive).Place)
	assert.Equal(t, "Wayne County, Indiana, United States", newMigrationEntry("1830", "place-wayne-in", "Residence", archive).Place)
}

func TestMigrationPlacePathAt_TemporalParent(t *testing.T) {
	// A town moved from one county to another: the comparison path used to
	// decide whether two observations are the same place follows the dated
	// parent, as Place and Region do.
	town := &glxlib.Place{Name: "Millbrook", Type: glxlib.PlaceTypeCity}
	town.SetParentHistory([]glxlib.PlaceParentPeriod{
		{Value: "place-hartford", Date: "FROM 1840 TO 1849"},
		{Value: "place-dodge", Date: "FROM 1850"},
	})
	archive := &glxlib.GLXFile{Places: map[string]*glxlib.Place{
		"place-wi":        {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		"place-hartford":  {Name: "Hartford Co.", Type: glxlib.PlaceTypeCounty, ParentID: "place-wi"},
		"place-dodge":     {Name: "Dodge Co.", Type: glxlib.PlaceTypeCounty, ParentID: "place-wi"},
		"place-millbrook": town,
	}}

	assert.Equal(t, []string{"Wisconsin", "Hartford Co.", "Millbrook"}, newMigrationEntry("1845", "place-millbrook", "Residence", archive).path)
	assert.Equal(t, []string{"Wisconsin", "Dodge Co.", "Millbrook"}, newMigrationEntry("1860", "place-millbrook", "Residence", archive).path)
	assert.Equal(t, []string{"Wisconsin", "Dodge Co.", "Millbrook"}, migrationPlacePath("place-millbrook", archive.Places), "no date: default parent")
}

func TestEventInPlace_UndatedTemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.True(t, eventInPlace(&glxlib.Event{PlaceID: "place-wayne-in"}, "place-indiana-territory", archive))
	assert.True(t, eventInPlace(&glxlib.Event{PlaceID: "place-wayne-in"}, "place-indiana", archive))
	assert.True(t, eventInPlace(&glxlib.Event{PlaceID: "place-wayne-in"}, "place-usa", archive))
	assert.False(t, eventInPlace(&glxlib.Event{PlaceID: "place-indiana"}, "place-wayne-in", archive))
}

func TestBuildPlaceAnalysis_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()
	archive.Events = map[string]*glxlib.Event{}

	a := buildPlaceAnalysis(archive)
	assert.NotContains(t, a.Unreferenced, "place-indiana-territory", "a parent in an earlier period is referenced")
	assert.Empty(t, a.DanglingParent)

	delete(archive.Places, "place-indiana-territory")
	a = buildPlaceAnalysis(archive)
	assert.Equal(t, []string{"place-wayne-in"}, a.DanglingParent)
	assert.Equal(t, "place-indiana-territory", a.DanglingParentIDs["place-wayne-in"])
}

func TestHistoricalPlaceFilterUsesEachObservationBeforeGroupingByYear(t *testing.T) {
	a := temporalParentArchive()
	people := []glxlib.Participant{{Person: "person-1", Role: "subject"}, {Person: "other", Role: "member"}}
	a.Events = map[string]*glxlib.Event{
		"early": {Type: glxlib.EventTypeCensus, Date: "1816-12-10", PlaceID: "place-wayne-in", Participants: people},
		"late":  {Type: glxlib.EventTypeCensus, Date: "1816-12-11", PlaceID: "place-wayne-in", Participants: people},
		"neighbor": {
			Type: "marriage", Date: "1816-12-11", PlaceID: "place-wayne-in",
			Participants: []glxlib.Participant{{Person: "late-only", Role: "groom"}},
		},
	}
	a.Persons["other"] = &glxlib.Person{}
	a.Persons["late-only"] = &glxlib.Person{}
	for _, tc := range []struct {
		county, event string
		overlap       bool
	}{
		{"place-indiana-territory", "early", false}, {"place-indiana", "late", true},
	} {
		t.Run(tc.county, func(t *testing.T) {
			result := buildCluster("person-1", a, tc.county, 0, 1815)
			var events []string
			for _, associate := range result.Associates {
				for _, link := range associate.Links {
					if link.EventID != "" {
						events = append(events, link.EventID)
					}
				}
			}
			require.Equal(t, []string{tc.event}, events)
			if tc.overlap {
				require.Len(t, result.Associates, 2)
			} else {
				require.Len(t, result.Associates, 1)
			}
			h := buildHouseholds(a, "person-1", householdsOptions{PlaceID: tc.county, Year: 1816})
			require.Len(t, h.Households, 1)
			require.Equal(t, tc.event, h.Households[0].EventID)
		})
	}
}

func TestRedistrictingIsNotMovementOrAVisit(t *testing.T) {
	a := temporalParentArchive()
	a.Places["town"] = &glxlib.Place{Name: "Town", ParentID: "place-wayne-in"}
	a.Places["other"] = &glxlib.Place{Name: "Other town", ParentID: "place-indiana"}
	entry := func(date, id string) migrationEntry { return newMigrationEntry(date, id, "Observation", a) }
	before, after := entry("1814", "town"), entry("1820", "town")
	coarse := entry("1818", "place-indiana")
	run := make([]migrationEntry, 0, 4)
	run = append(run, before, coarse, after)
	require.Empty(t, computeMovements(run, a))
	stops := migrationStops(run, a)
	require.Len(t, stops, 1)
	require.Equal(t, before.Place, stops[0].Place)
	require.Nil(t, matchStopsAgainstTerms(stops, []string{"Indiana Territory", "Indiana"}))
	move := computeMovements(append(run, entry("1821", "other")), a)
	require.Len(t, move, 1)
	require.Equal(t, "1820", move[0].FromDate)

	home := entry("FROM 1813 TO 1821", "town")
	home.location = true
	observations := []migrationEntry{home, after, coarse, entry("1820", "other")}
	applyResidencePeriods(observations, a)
	require.False(t, observations[1].Excluded)
	require.False(t, observations[2].Excluded)
	require.True(t, observations[3].Excluded)
}

func TestResidenceSuccessorNameRequiresCompletePaths(t *testing.T) {
	a := &glxlib.GLXFile{Places: map[string]*glxlib.Place{
		"country":   {Name: "US", Type: glxlib.PlaceTypeCountry},
		"state":     {Name: "Florida", ParentID: "country"},
		"territory": {Name: "Florida Territory", ParentID: "country"},
		"other":     {Name: "Elsewhere", ParentID: "country"},
	}}
	home := newMigrationEntry("FROM 1840 TO 1850", "state", "Residence", a)
	home.location = true
	entries := []migrationEntry{home, newMigrationEntry("1841", "territory", "Census", a), newMigrationEntry("1841", "other", "Census", a)}
	applyResidencePeriods(entries, a)
	require.False(t, entries[1].Excluded, "static normalized territory/state naming remains compatible")
	require.True(t, entries[2].Excluded)

	// A truncated village label at an unsupported date must not match
	// an unrelated state just because its standalone name is also "Florida".
	a.Places["village"] = &glxlib.Place{Name: "Florida", ParentHistory: []glxlib.PlaceParentPeriod{
		{Value: "other", Date: "FROM 1900"},
	}}
	entries = []migrationEntry{home, newMigrationEntry("1841", "village", "Census", a)}
	applyResidencePeriods(entries, a)
	require.True(t, entries[1].Excluded)
}

// ambiguousWayneArchive has Wayne County in Indiana Territory to 10 Dec 1816
// and in Indiana from the next day, so a year-only 1816 date matches both.
func ambiguousWayneArchive() *glxlib.GLXFile {
	archive := temporalParentArchive()
	archive.Places["place-wayne-in"].SetParentHistory([]glxlib.PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	})
	archive.Places["place-town-a"] = &glxlib.Place{Name: "Town A", Type: glxlib.PlaceTypeCity, ParentID: "place-wayne-in"}
	archive.Places["place-town-b"] = &glxlib.Place{Name: "Town B", Type: glxlib.PlaceTypeCity, ParentID: "place-wayne-in"}

	return archive
}

// A path cut short at an ambiguous date is compared from a place both paths
// contain, and its top is never reported as the region.
func TestMigrationStops_AmbiguousDateAlignsPathsByID(t *testing.T) {
	archive := ambiguousWayneArchive()
	first := newMigrationEntry("1816", "place-town-a", "Residence", archive)
	second := newMigrationEntry("1820", "place-town-b", "Residence", archive)

	assert.Equal(t, "Indiana", first.Region, "region from the default hierarchy, not the truncated top")
	assert.Equal(t, "Indiana", second.Region)

	from, to, moved, _ := compareMigrationStops(&first, &second, archive)
	assert.True(t, moved)
	assert.Equal(t, "Town A", from)
	assert.Equal(t, "Town B", to)

	same := newMigrationEntry("1820", "place-wayne-in", "Residence", archive)
	_, _, moved, refine := compareMigrationStops(&same, &first, archive)
	assert.False(t, moved, "Town A is inside Wayne County")
	assert.True(t, refine)
}

// An undated birth in a county with a dated parent history may lie in the
// majority state, so it is not reported as a sibling birthplace outlier.
func TestSiblingBirthplaceOutlier_UndatedTemporalChildNotFlagged(t *testing.T) {
	archive := siblingBirthplaceArchive("person-parent", "", []struct{ id, place string }{
		{"person-child-1", "place-indiana"},
		{"person-child-2", "place-indiana"},
		{"person-child-3", "place-indiana"},
		{"person-child-4", "place-wayne-in"},
	})
	maps.Copy(archive.Places, temporalParentArchive().Places)
	archive.Events["event-birth-person-child-4"].Date = ""

	for _, issue := range checkSiblingBirthplaceOutlier(archive) {
		assert.NotEqual(t, "person-child-4", issue.Person, issue.Message)
	}
}
