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

func TestCoveragePercent(t *testing.T) {
	assert.Equal(t, 0, coveragePercent(0, 0))
	assert.Equal(t, 0, coveragePercent(0, 10))
	assert.Equal(t, 50, coveragePercent(5, 10))
	assert.Equal(t, 100, coveragePercent(10, 10))
	assert.Equal(t, 33, coveragePercent(1, 3))
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

// --- State census tests ---

// --- Probate priority tests ---
