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
	"fmt"
	"sort"

	glxlib "github.com/genealogix/glx/go-glx"
)

// MigrateReport summarizes the changes made by a migration run.
type MigrateReport struct {
	EventsCreated      int
	EventsMerged       int
	PropertiesRemoved  int
	AssertionsMigrated int
	VocabEntriesRemoved int
}

var deprecatedProps = []string{
	glxlib.DeprecatedPropertyBornOn,
	glxlib.DeprecatedPropertyBornAt,
	glxlib.DeprecatedPropertyDiedOn,
	glxlib.DeprecatedPropertyDiedAt,
}

// migrateBirthDeathProperties converts deprecated born_on/born_at/died_on/died_at
// person properties into birth/death events. It modifies the archive in place.
func migrateBirthDeathProperties(archive *glxlib.GLXFile) (*MigrateReport, error) {
	report := &MigrateReport{}

	if archive.Events == nil {
		archive.Events = make(map[string]*glxlib.Event)
	}

	// Remove deprecated entries from person property vocabulary.
	if archive.PersonProperties != nil {
		for _, prop := range deprecatedProps {
			if _, exists := archive.PersonProperties[prop]; exists {
				delete(archive.PersonProperties, prop)
				report.VocabEntriesRemoved++
			}
		}
	}

	// Sort person IDs for deterministic output order.
	personIDs := make([]string, 0, len(archive.Persons))
	for id := range archive.Persons {
		personIDs = append(personIDs, id)
	}
	sort.Strings(personIDs)

	for _, personID := range personIDs {
		person := archive.Persons[personID]
		if person == nil || len(person.Properties) == 0 {
			continue
		}

		bornOn, hasBornOn := person.Properties[glxlib.DeprecatedPropertyBornOn]
		bornAt, hasBornAt := person.Properties[glxlib.DeprecatedPropertyBornAt]
		diedOn, hasDiedOn := person.Properties[glxlib.DeprecatedPropertyDiedOn]
		diedAt, hasDiedAt := person.Properties[glxlib.DeprecatedPropertyDiedAt]

		if !hasBornOn && !hasBornAt && !hasDiedOn && !hasDiedAt {
			continue
		}

		// Handle birth properties.
		if hasBornOn || hasBornAt {
			birthEventID, transferred, err := migrateEventProperties(
				archive, personID, glxlib.EventTypeBirth,
				bornOn, hasBornOn, bornAt, hasBornAt, report,
			)
			if err != nil {
				return nil, fmt.Errorf("person %s birth: %w", personID, err)
			}
			migrateAssertions(archive, personID, birthEventID,
				glxlib.DeprecatedPropertyBornOn, glxlib.DeprecatedPropertyBornAt, report)
			// Only delete properties whose values were successfully transferred.
			if transferred.date {
				delete(person.Properties, glxlib.DeprecatedPropertyBornOn)
				report.PropertiesRemoved++
			}
			if transferred.place {
				delete(person.Properties, glxlib.DeprecatedPropertyBornAt)
				report.PropertiesRemoved++
			}
		}

		// Handle death properties.
		if hasDiedOn || hasDiedAt {
			deathEventID, transferred, err := migrateEventProperties(
				archive, personID, glxlib.EventTypeDeath,
				diedOn, hasDiedOn, diedAt, hasDiedAt, report,
			)
			if err != nil {
				return nil, fmt.Errorf("person %s death: %w", personID, err)
			}
			migrateAssertions(archive, personID, deathEventID,
				glxlib.DeprecatedPropertyDiedOn, glxlib.DeprecatedPropertyDiedAt, report)
			if transferred.date {
				delete(person.Properties, glxlib.DeprecatedPropertyDiedOn)
				report.PropertiesRemoved++
			}
			if transferred.place {
				delete(person.Properties, glxlib.DeprecatedPropertyDiedAt)
				report.PropertiesRemoved++
			}
		}
		if len(person.Properties) == 0 {
			person.Properties = nil
		}
	}

	// Second pass: catch any remaining assertions that reference deprecated
	// property names but weren't processed above (e.g., the person didn't have
	// the deprecated properties but assertions still reference them).
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		personID := assertion.Subject.Person
		if personID == "" {
			continue
		}

		var eventType string
		var newProp string
		switch assertion.Property {
		case glxlib.DeprecatedPropertyBornOn:
			eventType, newProp = glxlib.EventTypeBirth, "date"
		case glxlib.DeprecatedPropertyBornAt:
			eventType, newProp = glxlib.EventTypeBirth, "place"
		case glxlib.DeprecatedPropertyDiedOn:
			eventType, newProp = glxlib.EventTypeDeath, "date"
		case glxlib.DeprecatedPropertyDiedAt:
			eventType, newProp = glxlib.EventTypeDeath, "place"
		default:
			continue
		}

		eventID, event := glxlib.FindPersonEvent(archive, personID, eventType)
		if eventID == "" {
			// Create an event so the assertion has a valid target,
			// populating date/place from the assertion value.
			newID, err := glxlib.GenerateRandomID()
			if err != nil {
				return nil, fmt.Errorf("generating event ID for orphaned assertion: %w", err)
			}
			eventID = "event-" + newID
			event = &glxlib.Event{
				Type: eventType,
				Participants: []glxlib.Participant{
					{Person: personID, Role: glxlib.ParticipantRolePrincipal},
				},
			}
			archive.Events[eventID] = event
			report.EventsCreated++
		}
		// Populate event fields from assertion value if empty.
		if newProp == "date" && event.Date == "" && assertion.Value != "" {
			event.Date = glxlib.DateString(assertion.Value)
		}
		if newProp == "place" && event.PlaceID == "" && assertion.Value != "" {
			event.PlaceID = assertion.Value
		}
		assertion.Subject = glxlib.EntityRef{Event: eventID}
		assertion.Property = newProp
		report.AssertionsMigrated++
	}

	archive.InvalidateCache()

	return report, nil
}

// extractPropertyString extracts a string value from a property that may be
// a plain string, a structured map with a "value" key, or a temporal list
// where the first entry has a "value" key.
func extractPropertyString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["value"].(string); ok {
			return s
		}
	case []any:
		if len(v) > 0 {
			// List of maps: [{value: "1850"}]
			if m, ok := v[0].(map[string]any); ok {
				if s, ok := m["value"].(string); ok {
					return s
				}
			}
			// List of plain strings: ["1850"]
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// transferResult tracks which property values were successfully transferred.
type transferResult struct {
	date  bool
	place bool
}

// migrateEventProperties creates or merges a birth/death event for a person.
// Returns the event ID, which values were transferred, and any error.
func migrateEventProperties(
	archive *glxlib.GLXFile,
	personID, eventType string,
	dateVal any, hasDate bool,
	placeVal any, hasPlace bool,
	report *MigrateReport,
) (string, transferResult, error) {
	var transferred transferResult

	dateStr := ""
	if hasDate {
		dateStr = extractPropertyString(dateVal)
		transferred.date = dateStr != ""
	}
	placeStr := ""
	if hasPlace {
		placeStr = extractPropertyString(placeVal)
		transferred.place = placeStr != ""
	}

	eventID, existing := glxlib.FindPersonEvent(archive, personID, eventType)

	if existing != nil {
		// Merge: fill in missing fields only.
		merged := false
		if dateStr != "" && existing.Date == "" {
			existing.Date = glxlib.DateString(dateStr)
			merged = true
		}
		if placeStr != "" && existing.PlaceID == "" {
			existing.PlaceID = placeStr
			merged = true
		}
		if merged {
			report.EventsMerged++
		}
		// Safe to remove the property if either: the value was extracted and
		// written, or the event already had data for that field. If the value
		// had an unrecognized shape (extractPropertyString returned ""), only
		// remove if the event already carries data — otherwise preserve the
		// property to avoid silent data loss.
		if hasDate {
			transferred.date = dateStr != "" || existing.Date != ""
		}
		if hasPlace {
			transferred.place = placeStr != "" || existing.PlaceID != ""
		}
		return eventID, transferred, nil
	}

	// Only create a new event if at least one value was extracted.
	if dateStr == "" && placeStr == "" {
		return "", transferred, nil
	}

	newID, err := glxlib.GenerateRandomID()
	if err != nil {
		return "", transferred, fmt.Errorf("generating event ID: %w", err)
	}
	eventID = "event-" + newID

	event := &glxlib.Event{
		Type: eventType,
		Participants: []glxlib.Participant{
			{Person: personID, Role: glxlib.ParticipantRolePrincipal},
		},
	}
	if dateStr != "" {
		event.Date = glxlib.DateString(dateStr)
	}
	if placeStr != "" {
		event.PlaceID = placeStr
	}

	archive.Events[eventID] = event
	report.EventsCreated++

	return eventID, transferred, nil
}

// migrateAssertions converts assertions that reference deprecated person properties
// to reference the corresponding event instead.
func migrateAssertions(
	archive *glxlib.GLXFile,
	personID, eventID string,
	dateProperty, placeProperty string,
	report *MigrateReport,
) {
	if eventID == "" {
		return
	}
	for _, assertion := range archive.Assertions {
		if assertion == nil || assertion.Subject.Person != personID {
			continue
		}

		switch assertion.Property {
		case dateProperty:
			assertion.Subject = glxlib.EntityRef{Event: eventID}
			assertion.Property = "date"
			report.AssertionsMigrated++
		case placeProperty:
			assertion.Subject = glxlib.EntityRef{Event: eventID}
			assertion.Property = "place"
			report.AssertionsMigrated++
		}
	}
}
