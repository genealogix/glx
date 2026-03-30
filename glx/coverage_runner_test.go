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
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestArchiveForCoverage() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					glxlib.PersonPropertyName:   "John Smith",
					glxlib.DeprecatedPropertyBornOn: "1840",
					glxlib.DeprecatedPropertyBornAt: "place-ny",
					glxlib.DeprecatedPropertyDiedOn: "1910",
					glxlib.DeprecatedPropertyDiedAt: "place-ny",
				},
			},
			"person-jane": {
				Properties: map[string]any{
					glxlib.PersonPropertyName:   "Jane Doe",
					glxlib.DeprecatedPropertyBornOn: "1845",
				},
			},
			"person-no-dates": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Unknown Person",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type:    glxlib.EventTypeBirth,
				Date:    "1840",
				PlaceID: "place-ny",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-death": {
				Type:    glxlib.EventTypeDeath,
				Date:    "1910",
				PlaceID: "place-ny",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-census-1850": {
				Type: glxlib.EventTypeCensus,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-census-1860": {
				Type: glxlib.EventTypeCensus,
				Date: "1860",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-marriage": {
				Type: glxlib.EventTypeMarriage,
				Date: "1865",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "groom"},
					{Person: "person-jane", Role: "bride"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type:       glxlib.RelationshipTypeMarriage,
				StartEvent: "event-marriage",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "spouse"},
					{Person: "person-jane", Role: "spouse"},
				},
			},
		},
		Sources:    map[string]*glxlib.Source{},
		Citations:  map[string]*glxlib.Citation{},
		Assertions: map[string]*glxlib.Assertion{},
		Places: map[string]*glxlib.Place{
			"place-ny": {Name: "New York, NY"},
		},
	}
}

func TestBuildCoverage_BasicPerson(t *testing.T) {
	archive := newTestArchiveForCoverage()
	person := archive.Persons["person-john"]

	result := buildCoverage("person-john", person, archive)

	assert.Equal(t, "person-john", result.PersonID)
	assert.Equal(t, "John Smith", result.PersonName)
	assert.Equal(t, "1840", result.BornOn)
	assert.Equal(t, "New York, NY", result.BornAt)
	assert.Equal(t, "1910", result.DiedOn)
	assert.Greater(t, result.Expected, 0)
	assert.Greater(t, result.Found, 0)
	assert.LessOrEqual(t, result.Found, result.Expected)
}

func TestBuildCoverage_CensusRecords(t *testing.T) {
	archive := newTestArchiveForCoverage()
	person := archive.Persons["person-john"]

	result := buildCoverage("person-john", person, archive)

	// Person born 1840, died 1910 — should have census records for
	// 1840, 1850, 1860, 1870, 1880, 1890, 1900, 1910
	var censusRecords []coverageRecord
	for _, r := range result.Records {
		if r.Category == "census" {
			censusRecords = append(censusRecords, r)
		}
	}

	assert.GreaterOrEqual(t, len(censusRecords), 7, "should generate census records from 1840 to 1910")

	// 1850 and 1860 census events exist
	foundYears := make(map[string]bool)
	for _, r := range censusRecords {
		if r.Found {
			foundYears[r.Label] = true
		}
	}
	assert.True(t, len(foundYears) >= 2, "should find at least 2 census records (1850, 1860)")
}

func TestBuildCoverage_VitalRecords(t *testing.T) {
	archive := newTestArchiveForCoverage()
	person := archive.Persons["person-john"]

	result := buildCoverage("person-john", person, archive)

	var vitalRecords []coverageRecord
	for _, r := range result.Records {
		if r.Category == "vital" {
			vitalRecords = append(vitalRecords, r)
		}
	}

	// Should have birth record, death record, and marriage record
	require.GreaterOrEqual(t, len(vitalRecords), 3)

	// Birth and death should be found (events exist)
	birthFound := false
	deathFound := false
	marriageFound := false
	for _, r := range vitalRecords {
		switch {
		case r.Label == "Birth record":
			birthFound = r.Found
		case r.Label == "Death record":
			deathFound = r.Found
		case r.Category == "vital" && r.Found && r.SourceRef == "event-marriage":
			marriageFound = true
		}
	}

	assert.True(t, birthFound, "birth record should be found")
	assert.True(t, deathFound, "death record should be found")
	assert.True(t, marriageFound, "marriage record should be found")
}

func TestBuildCoverage_NoDates(t *testing.T) {
	archive := newTestArchiveForCoverage()
	person := archive.Persons["person-no-dates"]

	result := buildCoverage("person-no-dates", person, archive)

	// No census records should be generated without birth year
	var censusRecords []coverageRecord
	for _, r := range result.Records {
		if r.Category == "census" {
			censusRecords = append(censusRecords, r)
		}
	}
	assert.Empty(t, censusRecords, "no census records without birth year")
}

func TestCollectPersonEvents(t *testing.T) {
	archive := newTestArchiveForCoverage()

	events := collectPersonEvents("person-john", archive)

	// Should find birth, death, census-1850, census-1860, marriage
	assert.GreaterOrEqual(t, len(events), 5)

	eventTypes := make(map[string]bool)
	for _, e := range events {
		eventTypes[e.EventType] = true
	}
	assert.True(t, eventTypes[glxlib.EventTypeBirth])
	assert.True(t, eventTypes[glxlib.EventTypeDeath])
	assert.True(t, eventTypes[glxlib.EventTypeCensus])
	assert.True(t, eventTypes[glxlib.EventTypeMarriage])
}

func TestCollectPersonSources(t *testing.T) {
	archive := newTestArchiveForCoverage()

	// Add a source and citation with an assertion about person-john
	archive.Sources["source-1850-census"] = &glxlib.Source{
		Type:  glxlib.SourceTypeCensus,
		Title: "1850 United States Federal Census",
		Date:  "1850",
	}
	archive.Citations["citation-1850"] = &glxlib.Citation{
		SourceID: "source-1850-census",
	}
	archive.Assertions["assertion-1"] = &glxlib.Assertion{
		Subject:   glxlib.EntityRef{Person: "person-john"},
		Property:  "born_on",
		Value:     "1840",
		Citations: []string{"citation-1850"},
	}

	sources := collectPersonSources("person-john", archive)

	require.Len(t, sources, 1)
	assert.Equal(t, "citation-1850", sources[0].Ref)
	assert.Equal(t, glxlib.SourceTypeCensus, sources[0].Type)
	assert.Equal(t, 1850, sources[0].Year)
}

func TestFindCensusMatch(t *testing.T) {
	events := []personSourceInfo{
		{Ref: "event-census-1850", EventType: glxlib.EventTypeCensus, Year: 1850},
		{Ref: "event-census-1870", EventType: glxlib.EventTypeCensus, Year: 1870},
	}
	sources := []personSourceInfo{
		{Ref: "source-1860", Type: glxlib.SourceTypeCensus, Year: 1860},
	}

	assert.Equal(t, "event-census-1850", findCensusMatch(1850, sources, events))
	assert.Equal(t, "source-1860", findCensusMatch(1860, sources, events))
	assert.Equal(t, "event-census-1870", findCensusMatch(1870, sources, events))
	assert.Equal(t, "", findCensusMatch(1880, sources, events))
}

func TestCoveragePercent(t *testing.T) {
	assert.Equal(t, 0, coveragePercent(0, 0))
	assert.Equal(t, 0, coveragePercent(0, 10))
	assert.Equal(t, 50, coveragePercent(5, 10))
	assert.Equal(t, 100, coveragePercent(10, 10))
	assert.Equal(t, 33, coveragePercent(1, 3))
}

func TestBoolPriority(t *testing.T) {
	assert.Equal(t, "high", boolPriority(true, "high"))
	assert.Equal(t, "", boolPriority(false, "high"))
}

func TestHasEventType(t *testing.T) {
	events := []personSourceInfo{
		{EventType: glxlib.EventTypeBirth},
		{EventType: glxlib.EventTypeCensus},
	}
	assert.True(t, hasEventType(events, glxlib.EventTypeBirth))
	assert.True(t, hasEventType(events, glxlib.EventTypeCensus))
	assert.False(t, hasEventType(events, glxlib.EventTypeDeath))
}

func TestHasSourceType(t *testing.T) {
	sources := []personSourceInfo{
		{Type: glxlib.SourceTypeCensus, Title: "1850 Census"},
		{Type: glxlib.SourceTypeVitalRecord, Title: "Birth Certificate"},
	}
	assert.True(t, hasSourceType(sources, glxlib.SourceTypeCensus, ""))
	assert.True(t, hasSourceType(sources, glxlib.SourceTypeVitalRecord, "birth"))
	assert.False(t, hasSourceType(sources, glxlib.SourceTypeVitalRecord, "death"))
	assert.False(t, hasSourceType(sources, glxlib.SourceTypeMilitary, ""))
}

func TestFindPersonForCoverage(t *testing.T) {
	archive := newTestArchiveForCoverage()

	// Exact ID match
	id, person, err := findPersonForCoverage(archive, "person-john")
	require.NoError(t, err)
	assert.Equal(t, "person-john", id)
	assert.NotNil(t, person)

	// Name substring match
	id, person, err = findPersonForCoverage(archive, "Jane")
	require.NoError(t, err)
	assert.Equal(t, "person-jane", id)
	assert.NotNil(t, person)

	// No match
	_, _, err = findPersonForCoverage(archive, "NonExistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestBuildCoverage_MaxLifespanCap(t *testing.T) {
	// Person born 1832, no death date — should cap census records at birth+100
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-old": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Old Person",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-old": {
				Type: glxlib.EventTypeBirth,
				Date: "ABT 1832",
				Participants: []glxlib.Participant{
					{Person: "person-old", Role: "principal"},
				},
			},
		},
		Relationships:  map[string]*glxlib.Relationship{},
		Sources:        map[string]*glxlib.Source{},
		Citations:      map[string]*glxlib.Citation{},
		Assertions:     map[string]*glxlib.Assertion{},
		Places:         map[string]*glxlib.Place{},
	}

	result := buildCoverage("person-old", archive.Persons["person-old"], archive)

	var censusYears []string
	for _, r := range result.Records {
		if r.Category == "census" {
			censusYears = append(censusYears, r.Label)
		}
	}

	// 1832+100=1932, so 1940 and 1950 should not appear
	for _, label := range censusYears {
		assert.NotContains(t, label, "1940", "should not suggest 1940 census")
		assert.NotContains(t, label, "1950", "should not suggest 1950 census")
	}
	// 1930 should still appear (1932 > 1930)
	found1930 := false
	for _, label := range censusYears {
		if strings.HasPrefix(label, "1930") {
			found1930 = true
		}
	}
	assert.True(t, found1930, "should include 1930 census")
}

func TestBuildCoverage_BurialInfersDeath(t *testing.T) {
	// Person born 1832, no death event, but has burial in 1863
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-soldier": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Soldier",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-soldier": {
				Type: glxlib.EventTypeBirth,
				Date: "1832",
				Participants: []glxlib.Participant{
					{Person: "person-soldier", Role: "principal"},
				},
			},
			"event-burial": {
				Type: glxlib.EventTypeBurial,
				Date: "1863",
				Participants: []glxlib.Participant{
					{Person: "person-soldier", Role: "principal"},
				},
			},
		},
		Relationships:  map[string]*glxlib.Relationship{},
		Sources:        map[string]*glxlib.Source{},
		Citations:      map[string]*glxlib.Citation{},
		Assertions:     map[string]*glxlib.Assertion{},
		Places:         map[string]*glxlib.Place{},
	}

	result := buildCoverage("person-soldier", archive.Persons["person-soldier"], archive)

	var censusYears []string
	for _, r := range result.Records {
		if r.Category == "census" {
			censusYears = append(censusYears, r.Label)
		}
	}

	// Should include 1840-1860 but NOT 1870+
	has1860 := false
	has1870 := false
	for _, label := range censusYears {
		if len(label) >= 4 {
			if label[:4] == "1860" {
				has1860 = true
			}
			if label[:4] == "1870" {
				has1870 = true
			}
		}
	}
	assert.True(t, has1860, "should include 1860 census (before burial)")
	assert.False(t, has1870, "should NOT include 1870 census (after burial)")
}

func TestBuildCoverage_1890Note(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1890": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Person Alive 1890",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-1890": {
				Type: glxlib.EventTypeBirth,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-1890", Role: "principal"},
				},
			},
			"event-death-1890": {
				Type: glxlib.EventTypeDeath,
				Date: "1920",
				Participants: []glxlib.Participant{
					{Person: "person-1890", Role: "principal"},
				},
			},
		},
		Relationships:  map[string]*glxlib.Relationship{},
		Sources:        map[string]*glxlib.Source{},
		Citations:      map[string]*glxlib.Citation{},
		Assertions:     map[string]*glxlib.Assertion{},
		Places:         map[string]*glxlib.Place{},
	}

	result := buildCoverage("person-1890", archive.Persons["person-1890"], archive)

	for _, r := range result.Records {
		if r.Category == "census" && len(r.Label) >= 4 && r.Label[:4] == "1890" {
			assert.Contains(t, r.Description, "destroyed", "1890 census should note destruction")
			return
		}
	}
	t.Fatal("did not find 1890 census record")
}

func TestInferDeathYearFromEvents(t *testing.T) {
	events := []personSourceInfo{
		{EventType: glxlib.EventTypeBirth, Year: 1832},
		{EventType: glxlib.EventTypeBurial, Year: 1863},
	}
	assert.Equal(t, 1863, inferDeathYearFromEvents(events))

	eventsNoBurial := []personSourceInfo{
		{EventType: glxlib.EventTypeBirth, Year: 1832},
	}
	assert.Equal(t, 0, inferDeathYearFromEvents(eventsNoBurial))
}

func TestCoverageResolvePlaceName(t *testing.T) {
	archive := newTestArchiveForCoverage()

	assert.Equal(t, "New York, NY", coverageResolvePlaceName("place-ny", archive))
	assert.Equal(t, "unknown-place", coverageResolvePlaceName("unknown-place", archive))
	assert.Equal(t, "", coverageResolvePlaceName("", archive))
}

// --- State census tests ---

func TestResolveStateFromPlace_DirectState(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-wi": {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
	}
	assert.Equal(t, "Wisconsin", resolveStateFromPlace("place-wi", archive))
}

func TestResolveStateFromPlace_CityWithStateParent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-madison": {Name: "Madison", Type: glxlib.PlaceTypeCity, ParentID: "place-dane-county"},
			"place-dane-county": {Name: "Dane County", Type: glxlib.PlaceTypeCounty, ParentID: "place-wi"},
			"place-wi": {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
	}
	assert.Equal(t, "Wisconsin", resolveStateFromPlace("place-madison", archive))
}

func TestResolveStateFromPlace_EmptyRef(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{},
	}
	assert.Equal(t, "", resolveStateFromPlace("", archive))
}

func TestResolveStateFromPlace_NoState(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-county": {Name: "Dane County", Type: glxlib.PlaceTypeCounty},
		},
	}
	assert.Equal(t, "", resolveStateFromPlace("place-county", archive))
}

func TestCollectPersonStates_FromBirthplace(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-wi": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "WI Person",
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-wi": {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-wi": {
				Type:    glxlib.EventTypeBirth,
				Date:    "1850",
				PlaceID: "place-wi",
				Participants: []glxlib.Participant{
					{Person: "person-wi", Role: "principal"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}

	// Pass the birth event info so collectPersonStates can find the state
	events := collectPersonEvents("person-wi", archive)
	states := collectPersonStates(archive.Persons["person-wi"], archive, events)
	assert.Contains(t, states, "Wisconsin")
}

func TestCollectPersonStates_FromEventPlace(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.PersonPropertyName:   "Test Person",
					glxlib.DeprecatedPropertyBornOn: "1850",
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-milwaukee": {Name: "Milwaukee", Type: glxlib.PlaceTypeCity, ParentID: "place-wi"},
			"place-wi":        {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type:    glxlib.EventTypeCensus,
				Date:    "1855",
				PlaceID: "place-milwaukee",
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: "subject"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}

	events := collectPersonEvents("person-1", archive)
	states := collectPersonStates(archive.Persons["person-1"], archive, events)
	assert.Contains(t, states, "Wisconsin")
}

func TestBuildStateCensusRecords_Wisconsin(t *testing.T) {
	// Person born 1850, died 1920, connected to Wisconsin
	records := buildStateCensusRecords(1850, 1920, []string{"Wisconsin"}, nil, nil, nil)

	var labels []string
	for _, r := range records {
		labels = append(labels, r.Label)
	}

	// Wisconsin had state censuses in 1855, 1865, 1875, 1885, 1895, 1905
	// Person born 1850, died 1920 — should suggest 1855, 1865, 1875, 1885, 1895, 1905
	assert.Contains(t, labels, "1855 Wisconsin State Census (age ~5)")
	assert.Contains(t, labels, "1875 Wisconsin State Census (age ~25)")
	assert.Contains(t, labels, "1905 Wisconsin State Census (age ~55)")

	for _, r := range records {
		assert.Equal(t, "census", r.Category)
	}
}

func TestBuildStateCensusRecords_NoStateMatch(t *testing.T) {
	// Person in a state with no state censuses
	records := buildStateCensusRecords(1850, 1920, []string{"Virginia"}, nil, nil, nil)
	assert.Empty(t, records)
}

func TestBuildStateCensusRecords_MatchesExistingEvent(t *testing.T) {
	events := []personSourceInfo{
		{Ref: "event-1855-census", EventType: glxlib.EventTypeCensus, Year: 1855, Title: "1855 Wisconsin State Census"},
	}
	records := buildStateCensusRecords(1850, 1920, []string{"Wisconsin"}, nil, events, nil)

	for _, r := range records {
		if strings.Contains(r.Label, "1855") {
			assert.True(t, r.Found, "1855 state census should be marked found")
			assert.Equal(t, "event-1855-census", r.SourceRef)
			return
		}
	}
	t.Fatal("did not find 1855 state census record")
}

func TestBuildStateCensusRecords_FederalNotConfusedWithState(t *testing.T) {
	// A federal 1860 census event should NOT match Mississippi's 1860 state census
	events := []personSourceInfo{
		{Ref: "event-1860-federal", EventType: glxlib.EventTypeCensus, Year: 1860, Title: "1860 US Federal Census"},
	}
	records := buildStateCensusRecords(1850, 1920, []string{"Mississippi"}, nil, events, nil)

	for _, r := range records {
		if strings.Contains(r.Label, "1860") {
			assert.False(t, r.Found, "federal 1860 census should NOT match Mississippi state census")
			return
		}
	}
	t.Fatal("did not find 1860 Mississippi state census record")
}

func TestPlaceRefsFromProperty_String(t *testing.T) {
	refs := placeRefsFromProperty("place-wi")
	assert.ElementsMatch(t, []string{"place-wi"}, refs)
}

func TestPlaceRefsFromProperty_StructuredMap(t *testing.T) {
	refs := placeRefsFromProperty(map[string]any{"value": "place-wi"})
	assert.ElementsMatch(t, []string{"place-wi"}, refs)
}

func TestPlaceRefsFromProperty_TemporalList(t *testing.T) {
	refs := placeRefsFromProperty([]any{
		map[string]any{"value": "place-wi"},
		map[string]any{"value": "place-ny"},
	})
	assert.ElementsMatch(t, []string{"place-wi", "place-ny"}, refs)
}

func TestPlaceRefsFromProperty_Nil(t *testing.T) {
	refs := placeRefsFromProperty(nil)
	assert.Empty(t, refs)
}

func TestPlaceRefsFromProperty_EmptyString(t *testing.T) {
	refs := placeRefsFromProperty("")
	assert.Empty(t, refs)
}

func TestFindStateCensusMatch_PlaceBased(t *testing.T) {
	// Event has no state name in title but place resolves to Wisconsin
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-milwaukee": {Name: "Milwaukee", Type: glxlib.PlaceTypeCity, ParentID: "place-wi"},
			"place-wi":        {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
	}
	events := []personSourceInfo{
		{Ref: "event-1855", EventType: glxlib.EventTypeCensus, Year: 1855, Title: "1855 Census", PlaceID: "place-milwaukee"},
	}
	ref := findStateCensusMatch(1855, "Wisconsin", nil, events, archive)
	assert.Equal(t, "event-1855", ref, "should match via place resolution")
}

func TestFindStateCensusMatch_PlaceWrongState(t *testing.T) {
	// Event place resolves to New York, not Wisconsin
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-nyc": {Name: "New York City", Type: glxlib.PlaceTypeCity, ParentID: "place-ny"},
			"place-ny":  {Name: "New York", Type: glxlib.PlaceTypeState},
		},
	}
	events := []personSourceInfo{
		{Ref: "event-1855", EventType: glxlib.EventTypeCensus, Year: 1855, Title: "1855 Census", PlaceID: "place-nyc"},
	}
	ref := findStateCensusMatch(1855, "Wisconsin", nil, events, archive)
	assert.Equal(t, "", ref, "should not match when place resolves to wrong state")
}

func TestCollectPersonStates_FromBirthEvent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Test Person",
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-wi": {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-1": {
				Type:    glxlib.EventTypeBirth,
				Date:    "1850",
				PlaceID: "place-wi",
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: "principal"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}

	events := collectPersonEvents("person-1", archive)
	states := collectPersonStates(archive.Persons["person-1"], archive, events)
	assert.Contains(t, states, "Wisconsin")
}

func TestBuildCoverage_IncludesStateCensus(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-wi": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "WI Person",
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-wi": {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-wi": {
				Type:    glxlib.EventTypeBirth,
				Date:    "1850",
				PlaceID: "place-wi",
				Participants: []glxlib.Participant{
					{Person: "person-wi", Role: "principal"},
				},
			},
			"event-death-wi": {
				Type: glxlib.EventTypeDeath,
				Date: "1920",
				Participants: []glxlib.Participant{
					{Person: "person-wi", Role: "principal"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}

	result := buildCoverage("person-wi", archive.Persons["person-wi"], archive)

	hasStateCensus := false
	for _, r := range result.Records {
		if r.Category == "census" && strings.Contains(r.Label, "State Census") {
			hasStateCensus = true
			break
		}
	}
	assert.True(t, hasStateCensus, "coverage should include state census records for Wisconsin")
}

func TestBuildCensusRecords_EnhancedAnnotations(t *testing.T) {
	// Person born 1830 — check 1850 and 1880 annotations
	records := buildCensusRecords(1830, 1920, nil, nil)

	for _, r := range records {
		if strings.HasPrefix(r.Label, "1850") && !r.Found {
			assert.Contains(t, r.Description, "first census to list individual names",
				"1850 census should note it was first to list individuals")
		}
		if strings.HasPrefix(r.Label, "1880") && !r.Found {
			assert.Contains(t, r.Description, "first census to list parents' birthplaces",
				"1880 census should note parent birthplace columns")
		}
	}
}

func TestBuildCensusRecords_1850InParentsHousehold(t *testing.T) {
	// Person born 1840 — at 1850 census they're age ~10, should note "likely in parents' household"
	records := buildCensusRecords(1840, 1920, nil, nil)

	for _, r := range records {
		if strings.HasPrefix(r.Label, "1850") && !r.Found {
			assert.Contains(t, r.Description, "likely in parents' household",
				"1850 census for child age ~10 should note parents' household")
			// Should NOT have duplicate household notes
			count := strings.Count(r.Description, "household")
			assert.Equal(t, 1, count, "should only have one household mention, got: %s", r.Description)
		}
	}
}

func TestStateCensusYears(t *testing.T) {
	// Verify known state census data
	wiYears, ok := stateCensusYears["Wisconsin"]
	assert.True(t, ok, "Wisconsin should be in state census data")
	assert.Contains(t, wiYears, 1855)
	assert.Contains(t, wiYears, 1905)

	nyYears, ok := stateCensusYears["New York"]
	assert.True(t, ok, "New York should be in state census data")
	assert.Contains(t, nyYears, 1855)
	assert.Contains(t, nyYears, 1925)
}

// --- Probate priority tests ---

func TestBuildOtherRecords_ProbateHighPriority_WithFamilyAndDeath(t *testing.T) {
	// Person died with known family — probate should be HIGH priority
	archive := newTestArchiveForCoverage()
	result := buildCoverage("person-john", archive.Persons["person-john"], archive)

	for _, r := range result.Records {
		if r.Label == "Probate/will" && !r.Found {
			assert.Equal(t, "high", r.Priority, "probate should be high priority when person died with family")
			assert.Contains(t, r.Description, "heirs", "should note probate names heirs")
			return
		}
	}
	t.Fatal("did not find unfound Probate/will record")
}

func TestBuildOtherRecords_ProbateNoPriority_NoDeath(t *testing.T) {
	// Person with birth event but no death event — probate should NOT be high priority
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-alive": {Properties: map[string]any{
				glxlib.PersonPropertyName: "Living Person",
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-alive": {
				Type: glxlib.EventTypeBirth, Date: "1980",
				Participants: []glxlib.Participant{{Person: "person-alive", Role: glxlib.ParticipantRolePrincipal}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
		Places:        map[string]*glxlib.Place{},
	}

	result := buildCoverage("person-alive", archive.Persons["person-alive"], archive)

	for _, r := range result.Records {
		if r.Label == "Probate/will" {
			assert.NotEqual(t, "high", r.Priority, "probate should not be high priority without death date")
			return
		}
	}
	t.Fatal("did not find Probate/will record in coverage output")
}

func TestBuildOtherRecords_ProbateNoPriority_NoFamily(t *testing.T) {
	// Person died but has no known family — probate not elevated
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-loner": {Properties: map[string]any{
				glxlib.PersonPropertyName: "Loner Person",
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-loner": {
				Type: glxlib.EventTypeBirth, Date: "1800",
				Participants: []glxlib.Participant{{Person: "person-loner", Role: glxlib.ParticipantRolePrincipal}},
			},
			"event-death-loner": {
				Type: glxlib.EventTypeDeath, Date: "1870",
				Participants: []glxlib.Participant{{Person: "person-loner", Role: glxlib.ParticipantRolePrincipal}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
		Places:        map[string]*glxlib.Place{},
	}

	result := buildCoverage("person-loner", archive.Persons["person-loner"], archive)

	for _, r := range result.Records {
		if r.Label == "Probate/will" {
			assert.NotEqual(t, "high", r.Priority, "probate should not be high priority without family")
			return
		}
	}
	t.Fatal("did not find Probate/will record in coverage output")
}
