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
	EventsCreated       int
	EventsMerged        int
	PropertiesRemoved   int
	AssertionsMigrated  int
	VocabEntriesRemoved int

	// gender→sex rename counts (opt-in via --rename-gender-to-sex).
	PropertiesRenamed   int
	AssertionsRenamed   int
	VocabEntriesRenamed int

	// GenderRenameSkipped is true when the gender→sex rename was requested
	// (via --rename-gender-to-sex) but refused because the archive already
	// carries post-split data. The archive may still contain legacy `gender`
	// values that were NOT migrated — this is distinct from "no work to do".
	GenderRenameSkipped bool

	// confidence: disputed → status: disputed conversion counts (opt-in via
	// --confidence-disputed-to-status, #516). ConfidenceDisputedConverted is
	// the total number of assertions touched; ConfidenceDisputedStatusConflicts
	// is the subset where an existing non-disputed status was preserved
	// instead of being overwritten — the user must reconcile those by hand.
	ConfidenceDisputedConverted       int
	ConfidenceDisputedStatusConflicts int
}

const (
	eventFieldDate  = "date"
	eventFieldPlace = "place"
)

var deprecatedProps = []string{
	glxlib.DeprecatedPropertyBornOn,
	glxlib.DeprecatedPropertyBornAt,
	glxlib.DeprecatedPropertyDiedOn,
	glxlib.DeprecatedPropertyDiedAt,
	glxlib.DeprecatedPropertyBuriedOn,
	glxlib.DeprecatedPropertyBuriedAt,
}

// migrateVitalEventProperties converts deprecated born_on/born_at/died_on/died_at/buried_on/buried_at
// person properties into birth/death/burial events. It modifies the archive in place.
func migrateVitalEventProperties(archive *glxlib.GLXFile) (*MigrateReport, error) {
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
		buriedOn, hasBuriedOn := person.Properties[glxlib.DeprecatedPropertyBuriedOn]
		buriedAt, hasBuriedAt := person.Properties[glxlib.DeprecatedPropertyBuriedAt]

		if !hasBornOn && !hasBornAt && !hasDiedOn && !hasDiedAt && !hasBuriedOn && !hasBuriedAt {
			continue
		}

		pairs := []propertyMigration{
			{glxlib.EventTypeBirth, glxlib.DeprecatedPropertyBornOn, glxlib.DeprecatedPropertyBornAt, bornOn, hasBornOn, bornAt, hasBornAt},
			{glxlib.EventTypeDeath, glxlib.DeprecatedPropertyDiedOn, glxlib.DeprecatedPropertyDiedAt, diedOn, hasDiedOn, diedAt, hasDiedAt},
			{glxlib.EventTypeBurial, glxlib.DeprecatedPropertyBuriedOn, glxlib.DeprecatedPropertyBuriedAt, buriedOn, hasBuriedOn, buriedAt, hasBuriedAt},
		}
		for _, pm := range pairs {
			if err := migratePropertyPair(archive, personID, person, pm, report); err != nil {
				return nil, err
			}
		}
		if len(person.Properties) == 0 {
			person.Properties = nil
		}
	}

	if err := migrateOrphanedAssertions(archive, report); err != nil {
		return nil, err
	}

	archive.InvalidateCache()

	return report, nil
}

// deprecatedAssertionMapping maps deprecated property names to their event type and field.
var deprecatedAssertionMapping = map[string]struct {
	eventType string
	field     string
}{
	glxlib.DeprecatedPropertyBornOn:   {glxlib.EventTypeBirth, eventFieldDate},
	glxlib.DeprecatedPropertyBornAt:   {glxlib.EventTypeBirth, eventFieldPlace},
	glxlib.DeprecatedPropertyDiedOn:   {glxlib.EventTypeDeath, eventFieldDate},
	glxlib.DeprecatedPropertyDiedAt:   {glxlib.EventTypeDeath, eventFieldPlace},
	glxlib.DeprecatedPropertyBuriedOn: {glxlib.EventTypeBurial, eventFieldDate},
	glxlib.DeprecatedPropertyBuriedAt: {glxlib.EventTypeBurial, eventFieldPlace},
}

// migrateOrphanedAssertions catches assertions that reference deprecated property
// names but weren't processed in the first pass (e.g., the person didn't have the
// deprecated properties but assertions still reference them).
func migrateOrphanedAssertions(archive *glxlib.GLXFile, report *MigrateReport) error {
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		personID := assertion.Subject.Person
		if personID == "" {
			continue
		}

		mapping, ok := deprecatedAssertionMapping[assertion.Property]
		if !ok {
			continue
		}

		eventID, event := glxlib.FindPersonEvent(archive, personID, mapping.eventType)
		if eventID == "" {
			newID, err := glxlib.GenerateRandomID()
			if err != nil {
				return fmt.Errorf("generating event ID for orphaned assertion: %w", err)
			}
			eventID = "event-" + newID
			event = &glxlib.Event{
				Type: mapping.eventType,
				Participants: []glxlib.Participant{
					{Person: personID, Role: glxlib.ParticipantRolePrincipal},
				},
			}
			archive.Events[eventID] = event
			report.EventsCreated++
		}

		if mapping.field == eventFieldDate && event.Date == "" && assertion.Value != "" {
			event.Date = glxlib.DateString(assertion.Value)
		}
		if mapping.field == eventFieldPlace && event.PlaceID == "" && assertion.Value != "" {
			event.PlaceID = assertion.Value
		}

		assertion.Subject = glxlib.EntityRef{Event: eventID}
		assertion.Property = mapping.field
		report.AssertionsMigrated++
	}

	return nil
}

// propertyMigration describes one deprecated date/place property pair to migrate.
type propertyMigration struct {
	eventType           string
	dateProp, placeProp string
	dateVal             any
	hasDate             bool
	placeVal            any
	hasPlace            bool
}

// migratePropertyPair migrates a single date/place property pair for a person
// into the corresponding event, updating assertions and removing properties.
func migratePropertyPair(
	archive *glxlib.GLXFile,
	personID string,
	person *glxlib.Person,
	pm propertyMigration,
	report *MigrateReport,
) error {
	if !pm.hasDate && !pm.hasPlace {
		return nil
	}

	eventID, transferred, err := migrateEventProperties(
		archive, personID, pm.eventType,
		pm.dateVal, pm.hasDate, pm.placeVal, pm.hasPlace, report,
	)
	if err != nil {
		return fmt.Errorf("person %s %s: %w", personID, pm.eventType, err)
	}

	migrateAssertions(archive, personID, eventID, pm.dateProp, pm.placeProp, report)

	if transferred.date {
		delete(person.Properties, pm.dateProp)
		report.PropertiesRemoved++
	}
	if transferred.place {
		delete(person.Properties, pm.placeProp)
		report.PropertiesRemoved++
	}

	return nil
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
			assertion.Property = eventFieldDate
			report.AssertionsMigrated++
		case placeProperty:
			assertion.Subject = glxlib.EntityRef{Event: eventID}
			assertion.Property = eventFieldPlace
			report.AssertionsMigrated++
		}
	}
}
