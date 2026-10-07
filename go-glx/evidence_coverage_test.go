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
)

// evidenceCoverageArchive is the evidence-first shape from #713: facts are
// asserted on events and relationships, never directly on persons or places.
func evidenceCoverageArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-child":   {},
			"person-father":  {},
			"person-mother":  {},
			"person-witness": {},
			"person-sponsor": {},
			"person-alone":   {},
		},
		Places: map[string]*Place{
			"place-country": {Name: "Prussia", Type: PlaceTypeCountry},
			"place-county":  {Name: "Kreis", ParentID: "place-country"},
			"place-village": {Name: "Liepen", ParentID: "place-county"},
			"place-church":  {Name: "St. Marien"},
			"place-cycle-a": {Name: "A", ParentID: "place-cycle-b"},
			"place-cycle-b": {Name: "B", ParentID: "place-cycle-a"},
			"place-unused":  {Name: "Elsewhere"},
		},
		Events: map[string]*Event{
			"event-christening": {
				Type:    EventTypeChristening,
				PlaceID: "place-church",
				Participants: []Participant{
					{Person: "person-child", Role: ParticipantRolePrincipal},
					{Person: "person-witness", Role: ParticipantRoleWitness},
				},
			},
			"event-marriage": {
				Type: EventTypeMarriage,
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleGroom},
					{Person: "person-mother", Role: ParticipantRoleBride},
				},
			},
			"event-unasserted": {Type: EventTypeBurial, PlaceID: "place-unused"},
		},
		Relationships: map[string]*Relationship{
			"rel-couple": {
				Type:       RelationshipTypeMarriage,
				StartEvent: "event-marriage",
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleSpouse},
					{Person: "person-mother", Role: ParticipantRoleSpouse},
				},
			},
			"rel-unasserted": {Type: RelationshipTypeParentChild},
		},
		Assertions: map[string]*Assertion{
			"a-christening-date": {Subject: EntityRef{Event: "event-christening"}, Property: "date", Value: "1850"},
			"a-marriage-place":   {Subject: EntityRef{Event: "event-marriage"}, Property: "place", Value: "place-village"},
			"a-sponsor": {
				Subject:     EntityRef{Event: "event-christening"},
				Participant: &Participant{Person: "person-sponsor", Role: ParticipantRoleGodparent},
			},
			"a-cycle": {Subject: EntityRef{Place: "place-cycle-a"}, Property: "name", Value: "A"},
		},
	}
}

func TestComputeAssertionCoverage_Direct(t *testing.T) {
	coverage := ComputeAssertionCoverage(evidenceCoverageArchive())

	assert.Empty(t, coverage.Direct.Persons, "no assertion targets a person")
	assert.Equal(t, map[string]bool{"event-christening": true, "event-marriage": true}, coverage.Direct.Events)
	assert.Empty(t, coverage.Direct.Relationships)
	assert.Equal(t, map[string]bool{"place-cycle-a": true}, coverage.Direct.Places)
}

func TestComputeAssertionCoverage_Evidence(t *testing.T) {
	coverage := ComputeAssertionCoverage(evidenceCoverageArchive())
	evidence := coverage.Evidence

	// Persons: participants of asserted events (any role), participant.person
	assert.Equal(t, map[string]bool{
		"person-child":   true,
		"person-witness": true,
		"person-father":  true,
		"person-mother":  true,
		"person-sponsor": true,
	}, evidence.Persons)

	assert.Equal(t, map[string]bool{"event-christening": true, "event-marriage": true}, evidence.Events)

	// The couple is evidenced through its asserted start event
	assert.Equal(t, map[string]bool{"rel-couple": true}, evidence.Relationships)

	// Places: an asserted event's place, an assertion value naming a place,
	// that place's parent chain, and a direct subject (cycle-safe)
	assert.Equal(t, map[string]bool{
		"place-church":  true,
		"place-village": true,
		"place-county":  true,
		"place-country": true,
		"place-cycle-a": true,
		"place-cycle-b": true,
	}, evidence.Places)
}

func TestComputeAssertionCoverage_PropertyReferenceValues(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"person-a": {}, "person-b": {}},
		Places:  map[string]*Place{"place-home": {Name: "Home"}},
		PersonProperties: map[string]*PropertyDefinition{
			"residence": {ReferenceType: "places"},
			"godfather": {ReferenceType: "persons"},
			"note":      {ValueType: "string"},
		},
		Assertions: map[string]*Assertion{
			"a-res":  {Subject: EntityRef{Person: "person-a"}, Property: "residence", Value: "place-home"},
			"a-god":  {Subject: EntityRef{Person: "person-a"}, Property: "godfather", Value: "person-b"},
			"a-note": {Subject: EntityRef{Person: "person-a"}, Property: "note", Value: "place-elsewhere"},
		},
	}

	evidence := ComputeAssertionCoverage(archive).Evidence
	assert.Equal(t, map[string]bool{"place-home": true}, evidence.Places)
	assert.Equal(t, map[string]bool{"person-a": true, "person-b": true}, evidence.Persons)
}

func TestComputeAssertionCoverage_Empty(t *testing.T) {
	coverage := ComputeAssertionCoverage(nil)
	assert.Empty(t, coverage.Evidence.Persons)

	coverage = ComputeAssertionCoverage(&GLXFile{Assertions: map[string]*Assertion{"a-nil": nil}})
	assert.Empty(t, coverage.Direct.Events)
}
