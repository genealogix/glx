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

func TestMigrate_CreatesBirthEventFromProperties(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: "1850-03-15",
					glxlib.DeprecatedPropertyBornAt: "place-london",
					"name":                          "John Smith",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.EventsCreated)
	assert.Equal(t, 0, report.EventsMerged)
	assert.Equal(t, 2, report.PropertiesRemoved) // born_on and born_at

	// Verify the deprecated properties were removed.
	person := archive.Persons["person-1"]
	assert.NotContains(t, person.Properties, glxlib.DeprecatedPropertyBornOn)
	assert.NotContains(t, person.Properties, glxlib.DeprecatedPropertyBornAt)
	assert.Contains(t, person.Properties, "name") // non-deprecated property preserved

	// Verify a birth event was created with the correct data.
	var birthEvent *glxlib.Event
	for _, event := range archive.Events {
		if event.Type == glxlib.EventTypeBirth {
			birthEvent = event
			break
		}
	}
	require.NotNil(t, birthEvent, "birth event should be created")
	assert.Equal(t, glxlib.DateString("1850-03-15"), birthEvent.Date)
	assert.Equal(t, "place-london", birthEvent.PlaceID)
	require.Len(t, birthEvent.Participants, 1)
	assert.Equal(t, "person-1", birthEvent.Participants[0].Person)
	assert.Equal(t, glxlib.ParticipantRolePrincipal, birthEvent.Participants[0].Role)
}

func TestMigrate_CreatesDeathEventFromProperties(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyDiedOn: "1920-11-02",
					glxlib.DeprecatedPropertyDiedAt: "place-paris",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.EventsCreated)
	assert.Equal(t, 2, report.PropertiesRemoved)

	// Person properties should be nil since all were deprecated.
	assert.Nil(t, archive.Persons["person-1"].Properties)

	// Verify death event exists.
	var deathEvent *glxlib.Event
	for _, event := range archive.Events {
		if event.Type == glxlib.EventTypeDeath {
			deathEvent = event
			break
		}
	}
	require.NotNil(t, deathEvent)
	assert.Equal(t, glxlib.DateString("1920-11-02"), deathEvent.Date)
	assert.Equal(t, "place-paris", deathEvent.PlaceID)
}

func TestMigrate_MergesIntoExistingEvent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: "1850-03-15",
					glxlib.DeprecatedPropertyBornAt: "place-london",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-existing": {
				Type: glxlib.EventTypeBirth,
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: glxlib.ParticipantRolePrincipal},
				},
				// Date and PlaceID are empty, so they should be filled.
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 0, report.EventsCreated)
	assert.Equal(t, 1, report.EventsMerged)
	assert.Equal(t, 2, report.PropertiesRemoved)

	// Verify the existing event was updated.
	event := archive.Events["event-existing"]
	assert.Equal(t, glxlib.DateString("1850-03-15"), event.Date)
	assert.Equal(t, "place-london", event.PlaceID)
}

func TestMigrate_DoesNotOverwriteExistingEventData(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: "1850-03-15",
					glxlib.DeprecatedPropertyBornAt: "place-london",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-existing": {
				Type:    glxlib.EventTypeBirth,
				Date:    "1850-06-01",
				PlaceID: "place-manchester",
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: glxlib.ParticipantRolePrincipal},
				},
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 0, report.EventsCreated)
	assert.Equal(t, 0, report.EventsMerged) // nothing to merge, everything already set
	assert.Equal(t, 2, report.PropertiesRemoved)

	// Verify original event data is preserved.
	event := archive.Events["event-existing"]
	assert.Equal(t, glxlib.DateString("1850-06-01"), event.Date)
	assert.Equal(t, "place-manchester", event.PlaceID)
}

func TestMigrate_ConvertsPropertyAssertionsToEventAssertions(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: "1850-03-15",
					glxlib.DeprecatedPropertyBornAt: "place-london",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: glxlib.DeprecatedPropertyBornOn,
				Value:    "1850-03-15",
			},
			"assertion-2": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: glxlib.DeprecatedPropertyBornAt,
				Value:    "place-london",
			},
			"assertion-3": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: "name",
				Value:    "John Smith",
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 2, report.AssertionsMigrated)

	// Find the new birth event ID.
	var birthEventID string
	for id, event := range archive.Events {
		if event.Type == glxlib.EventTypeBirth {
			birthEventID = id
			break
		}
	}
	require.NotEmpty(t, birthEventID)

	// Verify assertion-1 now references the event with property "date".
	a1 := archive.Assertions["assertion-1"]
	assert.Equal(t, birthEventID, a1.Subject.Event)
	assert.Empty(t, a1.Subject.Person)
	assert.Equal(t, "date", a1.Property)

	// Verify assertion-2 now references the event with property "place".
	a2 := archive.Assertions["assertion-2"]
	assert.Equal(t, birthEventID, a2.Subject.Event)
	assert.Empty(t, a2.Subject.Person)
	assert.Equal(t, "place", a2.Property)

	// Verify assertion-3 is unchanged (non-deprecated property).
	a3 := archive.Assertions["assertion-3"]
	assert.Equal(t, "person-1", a3.Subject.Person)
	assert.Equal(t, "name", a3.Property)
}

func TestMigrate_HandlesBornAtWithoutBornOn(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornAt: "place-london",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.EventsCreated)
	assert.Equal(t, 1, report.PropertiesRemoved) // only born_at

	// Verify the event has a place but no date.
	var birthEvent *glxlib.Event
	for _, event := range archive.Events {
		if event.Type == glxlib.EventTypeBirth {
			birthEvent = event
			break
		}
	}
	require.NotNil(t, birthEvent)
	assert.Empty(t, birthEvent.Date)
	assert.Equal(t, "place-london", birthEvent.PlaceID)
}

func TestMigrate_NoDeprecatedProperties(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					"name": "John Smith",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 0, report.EventsCreated)
	assert.Equal(t, 0, report.EventsMerged)
	assert.Equal(t, 0, report.PropertiesRemoved)
	assert.Equal(t, 0, report.AssertionsMigrated)
	assert.Empty(t, archive.Events)
}

func TestMigrate_BothBirthAndDeathProperties(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: "1850",
					glxlib.DeprecatedPropertyDiedOn: "1920",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 2, report.EventsCreated) // one birth, one death
	assert.Equal(t, 2, report.PropertiesRemoved)

	birthFound := false
	deathFound := false
	for _, event := range archive.Events {
		switch event.Type {
		case glxlib.EventTypeBirth:
			birthFound = true
			assert.Equal(t, glxlib.DateString("1850"), event.Date)
		case glxlib.EventTypeDeath:
			deathFound = true
			assert.Equal(t, glxlib.DateString("1920"), event.Date)
		}
	}
	assert.True(t, birthFound, "birth event should exist")
	assert.True(t, deathFound, "death event should exist")
}

func TestMigrate_StructuredPropertyShapes(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					// Map shape: {value: "..."}
					glxlib.DeprecatedPropertyBornOn: map[string]any{"value": "1850-03-15"},
					// List shape: [{value: "..."}]
					glxlib.DeprecatedPropertyDiedOn: []any{map[string]any{"value": "1920-06-01"}},
					// Map shape for place
					glxlib.DeprecatedPropertyBornAt: map[string]any{"value": "place-leeds"},
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 2, report.EventsCreated)
	assert.Equal(t, 3, report.PropertiesRemoved)

	_, birthEvent := glxlib.FindPersonEvent(archive, "person-1", glxlib.EventTypeBirth)
	require.NotNil(t, birthEvent)
	assert.Equal(t, glxlib.DateString("1850-03-15"), birthEvent.Date)
	assert.Equal(t, "place-leeds", birthEvent.PlaceID)

	_, deathEvent := glxlib.FindPersonEvent(archive, "person-1", glxlib.EventTypeDeath)
	require.NotNil(t, deathEvent)
	assert.Equal(t, glxlib.DateString("1920-06-01"), deathEvent.Date)

	// Properties should be removed
	assert.Empty(t, archive.Persons["person-1"].Properties)
}

func TestMigrate_OrphanedAssertionCreatesEvent(t *testing.T) {
	// Second pass: person has no deprecated properties, but assertions
	// still reference deprecated property names. The migration should
	// create events and retarget the assertions.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					"name": "Jane Doe",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-orphan-born": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: glxlib.DeprecatedPropertyBornOn,
				Value:    "1860-07-04",
			},
			"assertion-orphan-died": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: glxlib.DeprecatedPropertyDiedAt,
				Value:    "place-boston",
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 2, report.EventsCreated, "should create birth and death events")
	assert.Equal(t, 2, report.AssertionsMigrated)
	assert.Equal(t, 0, report.PropertiesRemoved, "no deprecated properties to remove")

	// Birth assertion should now point to a birth event.
	a1 := archive.Assertions["assertion-orphan-born"]
	assert.NotEmpty(t, a1.Subject.Event)
	assert.Empty(t, a1.Subject.Person)
	assert.Equal(t, "date", a1.Property)

	birthEvent := archive.Events[a1.Subject.Event]
	require.NotNil(t, birthEvent)
	assert.Equal(t, glxlib.EventTypeBirth, birthEvent.Type)
	assert.Equal(t, glxlib.DateString("1860-07-04"), birthEvent.Date)

	// Death assertion should now point to a death event.
	a2 := archive.Assertions["assertion-orphan-died"]
	assert.NotEmpty(t, a2.Subject.Event)
	assert.Equal(t, "place", a2.Property)

	deathEvent := archive.Events[a2.Subject.Event]
	require.NotNil(t, deathEvent)
	assert.Equal(t, glxlib.EventTypeDeath, deathEvent.Type)
	assert.Equal(t, "place-boston", deathEvent.PlaceID)
}

func TestMigrate_UnrecognizedShapePreservesProperty(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					// Unrecognized shape — not a string, map, or list
					glxlib.DeprecatedPropertyBornOn: 12345,
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	// Property should NOT be removed since the value couldn't be transferred
	assert.Equal(t, 0, report.PropertiesRemoved)
	_, exists := archive.Persons["person-1"].Properties[glxlib.DeprecatedPropertyBornOn]
	assert.True(t, exists, "unrecognized shape should be preserved")
}

func TestMigrate_UnrecognizedShapeWithExistingEvent(t *testing.T) {
	// Edge case: person has an unrecognized property shape AND an existing
	// event with an empty date. The property must NOT be deleted since the
	// value couldn't be extracted and the event doesn't carry it either.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: 12345,
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-1": {
				Type: glxlib.EventTypeBirth,
				// Date is empty — value would be lost if property is deleted
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: glxlib.ParticipantRolePrincipal},
				},
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 0, report.PropertiesRemoved)
	_, exists := archive.Persons["person-1"].Properties[glxlib.DeprecatedPropertyBornOn]
	assert.True(t, exists, "unrecognized shape should be preserved when event date is empty")
}

func TestMigrate_UnrecognizedShapeWithPopulatedEvent(t *testing.T) {
	// When an existing event already has a date, the property is safe to
	// remove even if its shape is unrecognized — no data loss since the
	// event already carries the value.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn: 12345,
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-1": {
				Type: glxlib.EventTypeBirth,
				Date: "1850-03-15",
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: glxlib.ParticipantRolePrincipal},
				},
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.PropertiesRemoved)
	_, exists := archive.Persons["person-1"].Properties[glxlib.DeprecatedPropertyBornOn]
	assert.False(t, exists, "property safe to remove when event already has date")
}

func TestMigrate_CreatesBurialEventFromBuriedAt(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBuriedAt: "place-gates-cemetery",
					"name":                            "John C Young",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.EventsCreated)
	assert.Equal(t, 1, report.PropertiesRemoved)

	// Verify the deprecated property was removed.
	person := archive.Persons["person-1"]
	assert.NotContains(t, person.Properties, glxlib.DeprecatedPropertyBuriedAt)
	assert.Contains(t, person.Properties, "name")

	// Verify a burial event was created.
	var burialEvent *glxlib.Event
	for _, event := range archive.Events {
		if event.Type == glxlib.EventTypeBurial {
			burialEvent = event

			break
		}
	}
	require.NotNil(t, burialEvent, "burial event should be created")
	assert.Empty(t, burialEvent.Date)
	assert.Equal(t, "place-gates-cemetery", burialEvent.PlaceID)
	require.Len(t, burialEvent.Participants, 1)
	assert.Equal(t, "person-1", burialEvent.Participants[0].Person)
	assert.Equal(t, glxlib.ParticipantRolePrincipal, burialEvent.Participants[0].Role)
}

func TestMigrate_CreatesBurialEventFromBuriedOnAndAt(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBuriedOn: "1920-12-01",
					glxlib.DeprecatedPropertyBuriedAt: "place-cemetery",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.EventsCreated)
	assert.Equal(t, 2, report.PropertiesRemoved)
	assert.Nil(t, archive.Persons["person-1"].Properties)

	var burialEvent *glxlib.Event
	for _, event := range archive.Events {
		if event.Type == glxlib.EventTypeBurial {
			burialEvent = event

			break
		}
	}
	require.NotNil(t, burialEvent)
	assert.Equal(t, glxlib.DateString("1920-12-01"), burialEvent.Date)
	assert.Equal(t, "place-cemetery", burialEvent.PlaceID)
}

func TestMigrate_MergesBurialIntoExistingEvent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBuriedAt: "place-cemetery",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-existing-burial": {
				Type: glxlib.EventTypeBurial,
				Participants: []glxlib.Participant{
					{Person: "person-1", Role: glxlib.ParticipantRolePrincipal},
				},
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 0, report.EventsCreated)
	assert.Equal(t, 1, report.EventsMerged)
	assert.Equal(t, 1, report.PropertiesRemoved)

	event := archive.Events["event-existing-burial"]
	assert.Equal(t, "place-cemetery", event.PlaceID)
}

func TestMigrate_BuriedAtAssertionMigrated(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBuriedAt: "place-cemetery",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-burial": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: glxlib.DeprecatedPropertyBuriedAt,
				Value:    "place-cemetery",
			},
		},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 1, report.AssertionsMigrated)

	a := archive.Assertions["assertion-burial"]
	assert.NotEmpty(t, a.Subject.Event)
	assert.Empty(t, a.Subject.Person)
	assert.Equal(t, "place", a.Property)
}

func TestMigrate_BirthDeathAndBurialTogether(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					glxlib.DeprecatedPropertyBornOn:   "1850",
					glxlib.DeprecatedPropertyDiedOn:   "1920",
					glxlib.DeprecatedPropertyBuriedAt: "place-cemetery",
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	report, err := migrateVitalEventProperties(archive)
	require.NoError(t, err)

	assert.Equal(t, 3, report.EventsCreated)
	assert.Equal(t, 3, report.PropertiesRemoved)

	birthFound, deathFound, burialFound := false, false, false
	for _, event := range archive.Events {
		switch event.Type {
		case glxlib.EventTypeBirth:
			birthFound = true
		case glxlib.EventTypeDeath:
			deathFound = true
		case glxlib.EventTypeBurial:
			burialFound = true
			assert.Equal(t, "place-cemetery", event.PlaceID)
		}
	}
	assert.True(t, birthFound, "birth event should exist")
	assert.True(t, deathFound, "death event should exist")
	assert.True(t, burialFound, "burial event should exist")
}
