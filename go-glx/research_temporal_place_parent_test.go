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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indianaTerritoryArchive is the Indiana example from #225: Wayne County was
// in Indiana Territory until statehood on 11 Dec 1816, then in Indiana.
func indianaTerritoryArchive() *GLXFile {
	wayne := &Place{Name: "Wayne County", Type: PlaceTypeCounty}
	wayne.SetParentHistory([]PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "FROM 1811 TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	})

	return &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{PersonPropertyName: "John Smith"}},
		},
		Places: map[string]*Place{
			"place-usa":               {Name: "United States", Type: PlaceTypeCountry},
			"place-indiana-territory": {Name: "Indiana Territory", ParentID: "place-usa"},
			"place-indiana":           {Name: "Indiana", Type: PlaceTypeState, ParentID: "place-usa"},
			"place-wayne-in":          wayne,
		},
		Events: map[string]*Event{
			"event-marriage": {
				Type: EventTypeMarriage, Date: "1814-04-02", PlaceID: "place-wayne-in",
				Participants: []Participant{{Person: "person-1", Role: "groom"}},
			},
		},
		Relationships: map[string]*Relationship{},
		Sources:       map[string]*Source{},
		Citations:     map[string]*Citation{},
		Assertions:    map[string]*Assertion{},
	}
}

func TestResolveStateFromPlaceAt_TemporalParent(t *testing.T) {
	archive := indianaTerritoryArchive()

	assert.Empty(t, resolveStateFromPlaceAt("place-wayne-in", "1814", archive), "a territory is not a state")
	assert.Equal(t, "Indiana", resolveStateFromPlaceAt("place-wayne-in", "1850", archive))
	assert.Equal(t, "Indiana", resolveStateFromPlace("place-wayne-in", archive))

	// The person's only event is the territorial marriage, so no state census applies.
	events := collectPersonEvents("person-1", archive, eventsWithEvidence(archive))
	assert.Empty(t, collectPersonStates(archive, events))
}

func TestResolveCountryFromPlaceAt_TemporalParent(t *testing.T) {
	strasbourg := &Place{Name: "Strasbourg", Type: PlaceTypeCity}
	strasbourg.SetParentHistory([]PlaceParentPeriod{
		{Value: "place-france", Date: "TO 1871-05-09"},
		{Value: "place-germany", Date: "FROM 1871-05-10 TO 1918-11-10"},
		{Value: "place-france", Date: "FROM 1918-11-11"},
	})
	archive := &GLXFile{Places: map[string]*Place{
		"place-france":     {Name: "France", Type: PlaceTypeCountry},
		"place-germany":    {Name: "Germany", Type: PlaceTypeCountry},
		"place-strasbourg": strasbourg,
	}}

	assert.Equal(t, "France", resolveCountryFromPlaceAt("place-strasbourg", "1850", archive))
	assert.Equal(t, "Germany", resolveCountryFromPlaceAt("place-strasbourg", "1900", archive))
	assert.Equal(t, "France", resolveCountryFromPlaceAt("place-strasbourg", "1950", archive))
	assert.Equal(t, "France", resolveCountryFromPlace("place-strasbourg", archive), "no date: default (latest) parent")
}

func TestCensusSchedulesForPerson_UsesEventDates(t *testing.T) {
	// A village moves into the US only in the fictional hierarchy below; the
	// schedule must follow the event's date.
	village := &Place{Name: "Village", Type: PlaceTypeCity}
	village.SetParentHistory([]PlaceParentPeriod{
		{Value: "place-usa", Date: "FROM 1900"},
		{Value: "place-mecklenburg", Date: "TO 1899"},
	})
	archive := &GLXFile{
		Places: map[string]*Place{
			"place-usa":         {Name: "United States", Type: PlaceTypeCountry},
			"place-mecklenburg": {Name: "Mecklenburg-Strelitz", Type: PlaceTypeCountry},
			"place-village":     village,
		},
		Events: map[string]*Event{
			"event-birth": {
				Type: EventTypeBirth, Date: "1850", PlaceID: "place-village",
				Participants: []Participant{{Person: "person-1", Role: "subject"}},
			},
		},
	}

	// Date-aware: in 1850 the village was in Mecklenburg-Strelitz, which has
	// no schedule, so nothing is suggested.
	assert.Empty(t, censusSchedulesForPerson(archive, "person-1", CensusCountryUnitedStates))

	// The undated lookup follows the default (latest) parent into the US.
	schedules := censusSchedulesForPlaces([]string{"place-village"}, archive, CensusCountryUnitedStates)
	require.Len(t, schedules, 1)
	assert.Equal(t, CensusCountryUnitedStates, schedules[0].Country)
}

func TestPlaceAncestor_TemporalParent(t *testing.T) {
	places := indianaTerritoryArchive().Places

	assert.True(t, placeAncestor("place-indiana-territory", "place-wayne-in", places))
	assert.True(t, placeAncestor("place-indiana", "place-wayne-in", places))
	assert.True(t, placeAncestor("place-wayne-in", "place-wayne-in", places))
	assert.False(t, placeAncestor("place-wayne-in", "place-indiana", places))
}

func TestMarkPlaceAncestors_TemporalParent(t *testing.T) {
	covered := map[string]bool{"place-wayne-in": true}
	markPlaceAncestors(covered, indianaTerritoryArchive().Places)

	assert.Equal(t, map[string]bool{
		"place-wayne-in": true, "place-indiana-territory": true, "place-indiana": true, "place-usa": true,
	}, covered)
}
