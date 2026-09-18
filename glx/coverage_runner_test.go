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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func newTestArchiveForCoverage() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "John Smith",
				},
			},
			"person-jane": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Jane Doe",
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
			"event-birth-jane": {
				Type: glxlib.EventTypeBirth,
				Date: "1845",
				Participants: []glxlib.Participant{
					{Person: "person-jane", Role: "subject"},
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
		// John's events are each backed by an assertion citing a record, so they
		// count as records found. Jane's birth event deliberately is not: it
		// stands alone, as a conclusion nothing supports.
		Sources: map[string]*glxlib.Source{
			"source-town-records": {
				Type:  glxlib.SourceTypeVitalRecord,
				Title: "Town of Greenfield vital records",
			},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-town-records": {SourceID: "source-town-records"},
		},
		Assertions: assertionsEvidencing(
			"event-birth", "event-death", "event-census-1850", "event-census-1860", "event-marriage",
		),
		Places: map[string]*glxlib.Place{
			"place-ny": {Name: "New York, NY"},
		},
	}
}

// assertionsEvidencing returns one assertion per event ID, each citing
// citation-town-records, which is what makes the event count as evidenced.
func assertionsEvidencing(eventIDs ...string) map[string]*glxlib.Assertion {
	assertions := make(map[string]*glxlib.Assertion, len(eventIDs))
	for _, eventID := range eventIDs {
		assertions["assertion-"+eventID] = &glxlib.Assertion{
			Subject:   glxlib.EntityRef{Event: eventID},
			Citations: []string{"citation-town-records"},
		}
	}

	return assertions
}

func TestBuildCoverage_BasicPerson(t *testing.T) {
	archive := newTestArchiveForCoverage()
	person := archive.Persons["person-john"]

	result := buildCoverage("person-john", person, archive)

	assert.Equal(t, "person-john", result.PersonID)
	assert.Equal(t, "John Smith", result.PersonName)
	assert.Equal(t, "1840", result.BirthDate)
	assert.Equal(t, "New York, NY", result.BirthPlace)
	assert.Equal(t, "1910", result.DeathDate)
	assert.Positive(t, result.Expected)
	assert.Positive(t, result.Found)
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
	assert.GreaterOrEqual(t, len(foundYears), 2, "should find at least 2 census records (1850, 1860)")
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

	events := collectPersonEvents("person-john", archive, eventsWithEvidence(archive))

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
		Property:  "name",
		Value:     "John Smith",
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
		{Ref: "event-census-1850", EventType: glxlib.EventTypeCensus, Year: 1850, Evidenced: true},
		{Ref: "event-census-1870", EventType: glxlib.EventTypeCensus, Year: 1870, Evidenced: true},
		{Ref: "event-census-1900", EventType: glxlib.EventTypeCensus, Year: 1900},
	}
	sources := []personSourceInfo{
		{Ref: "source-1860", Type: glxlib.SourceTypeCensus, Year: 1860},
	}

	assert.Equal(t, "event-census-1850", findCensusMatch(1850, sources, events))
	assert.Equal(t, "source-1860", findCensusMatch(1860, sources, events))
	assert.Equal(t, "event-census-1870", findCensusMatch(1870, sources, events))
	assert.Empty(t, findCensusMatch(1880, sources, events))
	assert.Empty(t, findCensusMatch(1900, sources, events),
		"a census event nothing backs is not a census record found")
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
	assert.Empty(t, boolPriority(false, "high"))
}

func TestFindEvidencedEvent(t *testing.T) {
	events := []personSourceInfo{
		{Ref: "event-birth", EventType: glxlib.EventTypeBirth, Evidenced: true},
		{Ref: "event-census", EventType: glxlib.EventTypeCensus},
	}

	assert.Equal(t, "event-birth", findEvidencedEvent(events, glxlib.EventTypeBirth))
	assert.Empty(t, findEvidencedEvent(events, glxlib.EventTypeCensus),
		"an event nothing backs is not an evidenced event")
	assert.Empty(t, findEvidencedEvent(events, glxlib.EventTypeDeath))

	assert.Equal(t, "event-census", findUnevidencedEvent(events, glxlib.EventTypeCensus))
	assert.Empty(t, findUnevidencedEvent(events, glxlib.EventTypeBirth))
	assert.Empty(t, findUnevidencedEvent(events, glxlib.EventTypeDeath))
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
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
		Places:        map[string]*glxlib.Place{},
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
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
		Places:        map[string]*glxlib.Place{},
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
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
		Places:        map[string]*glxlib.Place{},
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
	assert.Empty(t, coverageResolvePlaceName("", archive))
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
			"place-madison":     {Name: "Madison", Type: glxlib.PlaceTypeCity, ParentID: "place-dane-county"},
			"place-dane-county": {Name: "Dane County", Type: glxlib.PlaceTypeCounty, ParentID: "place-wi"},
			"place-wi":          {Name: "Wisconsin", Type: glxlib.PlaceTypeState},
		},
	}
	assert.Equal(t, "Wisconsin", resolveStateFromPlace("place-madison", archive))
}

func TestResolveStateFromPlace_EmptyRef(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{},
	}
	assert.Empty(t, resolveStateFromPlace("", archive))
}

func TestResolveStateFromPlace_NoState(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-county": {Name: "Dane County", Type: glxlib.PlaceTypeCounty},
		},
	}
	assert.Empty(t, resolveStateFromPlace("place-county", archive))
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
	events := collectPersonEvents("person-wi", archive, eventsWithEvidence(archive))
	states := collectPersonStates(archive.Persons["person-wi"], archive, events)
	assert.Contains(t, states, "Wisconsin")
}

func TestCollectPersonStates_FromEventPlace(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Test Person",
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

	events := collectPersonEvents("person-1", archive, eventsWithEvidence(archive))
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
		{Ref: "event-1855-census", EventType: glxlib.EventTypeCensus, Year: 1855, Title: "1855 Wisconsin State Census", Evidenced: true},
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
		{Ref: "event-1855", EventType: glxlib.EventTypeCensus, Year: 1855, Title: "1855 Census", PlaceID: "place-milwaukee", Evidenced: true},
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
	assert.Empty(t, ref, "should not match when place resolves to wrong state")
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

	events := collectPersonEvents("person-1", archive, eventsWithEvidence(archive))
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

func TestCoverageResult_JSONKeys(t *testing.T) {
	// Regression test: ensure JSON output uses the renamed keys, not the
	// deprecated born_on/born_at/died_on/died_at names.
	result := coverageResult{
		PersonID:   "person-1",
		PersonName: "Test Person",
		BirthDate:  "1840",
		BirthPlace: "place-ny",
		DeathDate:  "1910",
		DeathPlace: "place-ca",
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)
	jsonStr := string(data)

	assert.Contains(t, jsonStr, `"birth_date"`)
	assert.Contains(t, jsonStr, `"birth_place"`)
	assert.Contains(t, jsonStr, `"death_date"`)
	assert.Contains(t, jsonStr, `"death_place"`)
	assert.NotContains(t, jsonStr, `"born_on"`)
	assert.NotContains(t, jsonStr, `"born_at"`)
	assert.NotContains(t, jsonStr, `"died_on"`)
	assert.NotContains(t, jsonStr, `"died_at"`)
}

// archiveWithUnevidencedBirth reproduces the archive from #1267: a death
// recorded from a parish register, and a birth that is an explicit estimate
// reckoned back from the age band in that death entry. No birth record exists.
func archiveWithUnevidencedBirth() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-x": {
				Properties: map[string]any{
					glxlib.PersonPropertyName: "Michael David Hollnagel",
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-liepen": {Name: "Liepen"},
		},
		Events: map[string]*glxlib.Event{
			"event-death": {
				Type:    glxlib.EventTypeDeath,
				Date:    "1807-08-31",
				PlaceID: "place-liepen",
				Participants: []glxlib.Participant{
					{Person: "person-x", Role: "subject"},
				},
			},
			"event-birth": {
				Type: glxlib.EventTypeBirth,
				Date: "BET 1737 AND 1747",
				Participants: []glxlib.Participant{
					{Person: "person-x", Role: "subject"},
				},
			},
		},
		Sources: map[string]*glxlib.Source{
			"source-parish-register": {
				Type:  glxlib.SourceTypeChurchRegister,
				Title: "Liepen parish register, burials",
			},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-death-entry": {SourceID: "source-parish-register"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-death": {
				Subject:   glxlib.EntityRef{Event: "event-death"},
				Citations: []string{"citation-death-entry"},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
	}
}

func coverageRecordByLabel(t *testing.T, result *coverageResult, label string) coverageRecord {
	t.Helper()

	for _, r := range result.Records {
		if r.Label == label {
			return r
		}
	}
	t.Fatalf("no %q record in coverage output", label)

	return coverageRecord{}
}

func TestBuildCoverage_UnevidencedEventIsNotARecordFound(t *testing.T) {
	archive := archiveWithUnevidencedBirth()

	result := buildCoverage("person-x", archive.Persons["person-x"], archive)

	birth := coverageRecordByLabel(t, result, "Birth record")
	assert.False(t, birth.Found,
		"a birth event nothing backs is a conclusion, not a birth record found")
	assert.Equal(t, "high", birth.Priority, "the missing birth record is still worth chasing")
	assert.Empty(t, birth.SourceRef,
		"source_ref must not name the unevidenced event, which a caller would read as evidence")
	assert.Contains(t, birth.Description, "event-birth",
		"the unsourced birth event is still reported, so it is not silently dropped")

	death := coverageRecordByLabel(t, result, "Death record")
	assert.True(t, death.Found, "the death event is cited to the parish register")
	assert.Equal(t, "event-death", death.SourceRef)
}

func TestBuildCoverage_EvidenceOnEventSubjectAssertionCounts(t *testing.T) {
	// The shape the example archives in this repo use: the record is cited on
	// an assertion whose subject is the birth event.
	archive := archiveWithUnevidencedBirth()
	archive.Sources["source-birth-register"] = &glxlib.Source{
		Type:  glxlib.SourceTypeVitalRecord,
		Title: "Liepen parish register, baptisms",
	}
	archive.Citations["citation-baptism-entry"] = &glxlib.Citation{SourceID: "source-birth-register"}
	archive.Assertions["assertion-birth"] = &glxlib.Assertion{
		Subject:   glxlib.EntityRef{Event: "event-birth"},
		Citations: []string{"citation-baptism-entry"},
	}

	result := buildCoverage("person-x", archive.Persons["person-x"], archive)

	birth := coverageRecordByLabel(t, result, "Birth record")
	assert.True(t, birth.Found, "the birth event is now cited to a record")
	assert.Equal(t, "event-birth", birth.SourceRef)
	assert.Empty(t, birth.Priority)
	assert.Empty(t, birth.Description)
}

func TestBuildCoverage_UnevidencedEventDoesNotCountTowardScore(t *testing.T) {
	archive := archiveWithUnevidencedBirth()
	withoutBirthEvidence := buildCoverage("person-x", archive.Persons["person-x"], archive)

	archive.Assertions["assertion-birth"] = &glxlib.Assertion{
		Subject: glxlib.EntityRef{Event: "event-birth"},
		Sources: []string{"source-parish-register"},
	}
	withBirthEvidence := buildCoverage("person-x", archive.Persons["person-x"], archive)

	assert.Equal(t, withoutBirthEvidence.Expected, withBirthEvidence.Expected,
		"evidence changes what is found, not what is expected")
	assert.Equal(t, withoutBirthEvidence.Found+1, withBirthEvidence.Found,
		"citing the birth event is what moves the score, not recording the event")
}

func TestBuildCoverage_UnevidencedMarriageEventIsNotARecordFound(t *testing.T) {
	archive := newTestArchiveForCoverage()
	delete(archive.Assertions, "assertion-event-marriage")

	result := buildCoverage("person-john", archive.Persons["person-john"], archive)

	marriage := coverageRecordByLabel(t, result, "Marriage record — Jane Doe")
	assert.False(t, marriage.Found, "a marriage event nothing backs is not the marriage record")
	assert.Empty(t, marriage.SourceRef)
	assert.Contains(t, marriage.Description, "event-marriage")
}

func TestBuildCoverage_UnevidencedCensusEventIsNotARecordFound(t *testing.T) {
	archive := newTestArchiveForCoverage()
	delete(archive.Assertions, "assertion-event-census-1850")

	result := buildCoverage("person-john", archive.Persons["person-john"], archive)

	census1850 := coverageRecordByLabel(t, result, "1850 US Census (age ~10)")
	assert.False(t, census1850.Found, "a census event nothing backs is not the census record")
	assert.Contains(t, census1850.Description, "event-census-1850")
	assert.Contains(t, census1850.Description, "first census to list individual names",
		"the year annotation is kept alongside the note")
}

func TestEventsWithEvidence(t *testing.T) {
	archive := &glxlib.GLXFile{
		Sources: map[string]*glxlib.Source{
			"source-1": {Type: glxlib.SourceTypeVitalRecord, Title: "Birth register"},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-1": {SourceID: "source-1"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-cited": {
				Subject:   glxlib.EntityRef{Event: "event-cited"},
				Citations: []string{"citation-1"},
			},
			"assertion-sourced": {
				Subject: glxlib.EntityRef{Event: "event-sourced"},
				Sources: []string{"source-1"},
			},
			"assertion-bare": {
				Subject:  glxlib.EntityRef{Event: "event-bare"},
				Property: "date",
				Value:    "1840",
			},
			"assertion-dangling": {
				Subject:   glxlib.EntityRef{Event: "event-dangling"},
				Citations: []string{"citation-missing"},
				Sources:   []string{"source-missing"},
			},
			"assertion-about-person": {
				Subject:   glxlib.EntityRef{Person: "person-1"},
				Citations: []string{"citation-1"},
			},
		},
	}

	evidenced := eventsWithEvidence(archive)

	assert.True(t, evidenced["event-cited"], "a resolved citation is evidence")
	assert.True(t, evidenced["event-sourced"], "a resolved source is evidence")
	assert.False(t, evidenced["event-bare"], "an assertion citing nothing is not evidence")
	assert.False(t, evidenced["event-dangling"],
		"a dangling reference is a validation error, not evidence")
	assert.False(t, evidenced["person-1"], "a person-subject assertion evidences no event")
	assert.Len(t, evidenced, 2)
}

func TestEventsWithEvidence_MixedAssertionsOnOneEvent(t *testing.T) {
	// One event carrying both an uncited assertion and a cited one is evidenced:
	// the order the assertion map is walked in must not decide the answer.
	archive := &glxlib.GLXFile{
		Sources: map[string]*glxlib.Source{
			"source-1": {Type: glxlib.SourceTypeVitalRecord},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-bare": {
				Subject:  glxlib.EntityRef{Event: "event-1"},
				Property: "date",
				Value:    "1840",
			},
			"assertion-sourced": {
				Subject: glxlib.EntityRef{Event: "event-1"},
				Sources: []string{"source-1"},
			},
		},
	}

	for range 20 {
		assert.True(t, eventsWithEvidence(archive)["event-1"])
	}
}

func TestBuildCoverage_JSONReportsUnevidencedEventAsNotFound(t *testing.T) {
	archive := archiveWithUnevidencedBirth()

	result := buildCoverage("person-x", archive.Persons["person-x"], archive)

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded struct {
		Records []struct {
			Label     string `json:"label"`
			Found     bool   `json:"found"`
			SourceRef string `json:"source_ref"`
		} `json:"records"`
	}
	require.NoError(t, json.Unmarshal(data, &decoded))

	for _, r := range decoded.Records {
		if r.Label == "Birth record" {
			assert.False(t, r.Found, "a caller reading found must not be told a missing record exists")
			assert.Empty(t, r.SourceRef)

			return
		}
	}
	t.Fatal("no Birth record in JSON output")
}
