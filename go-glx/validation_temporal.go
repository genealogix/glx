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
	"fmt"
)

// validateTemporalConsistency checks for logical inconsistencies in dates
// across persons, events, and relationships. All issues are reported as
// warnings since dates are often estimates (ABT, BEF, etc.).
func (glx *GLXFile) validateTemporalConsistency(result *ValidationResult) {
	glx.validateDeathBeforeBirth(result)
	glx.validateParentChildAges(result)
	glx.validateMarriageBeforeBirth(result)
	glx.validateRelationshipEventOrder(result)
	glx.validateRelationshipBoundarySources(result)
}

// extractEventYear finds a person's event of the given type and returns the
// year from its date, or 0 if no event is found.
func extractEventYear(archive *GLXFile, personID, eventType string) int {
	_, event := FindPersonEvent(archive, personID, eventType)
	if event == nil {
		return 0
	}

	return ExtractFirstYear(string(event.Date))
}

// validateDeathBeforeBirth checks that no person has a death date earlier than
// their birth date.
func (glx *GLXFile) validateDeathBeforeBirth(result *ValidationResult) {
	for id, person := range glx.Persons {
		if person == nil {
			continue
		}
		birthYear := extractEventYear(glx, id, EventTypeBirth)
		deathYear := extractEventYear(glx, id, EventTypeDeath)

		if birthYear == 0 || deathYear == 0 {
			continue
		}

		if deathYear < birthYear {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypePersons,
				SourceID:   id,
				Field:      "death_event",
				Message: fmt.Sprintf("%s[%s]: death year (%d) is before birth year (%d)",
					EntityTypePersons, id, deathYear, birthYear),
			})
		}
	}
}

// validateParentChildAges checks that parents are born before their children.
func (glx *GLXFile) validateParentChildAges(result *ValidationResult) {
	for relID, rel := range glx.Relationships {
		if rel == nil {
			continue
		}
		if !isParentChildRelType(rel.Type) {
			continue
		}

		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case ParticipantRoleChild:
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, parentID := range parentIDs {
			parent, ok := glx.Persons[parentID]
			if !ok || parent == nil {
				continue
			}
			parentBirth := extractEventYear(glx, parentID, EventTypeBirth)
			if parentBirth == 0 {
				continue
			}

			for _, childID := range childIDs {
				child, ok := glx.Persons[childID]
				if !ok || child == nil {
					continue
				}
				childBirth := extractEventYear(glx, childID, EventTypeBirth)
				if childBirth == 0 {
					continue
				}

				if parentBirth > childBirth {
					result.Warnings = append(result.Warnings, ValidationWarning{
						SourceType: EntityTypeRelationships,
						SourceID:   relID,
						Field:      "participants",
						Message: fmt.Sprintf("%s[%s]: parent %s (born %d) is born after child %s (born %d)",
							EntityTypeRelationships, relID, parentID, parentBirth, childID, childBirth),
					})
				}
			}
		}
	}
}

// validateMarriageBeforeBirth checks that marriage events do not occur before
// any participant's birth.
func (glx *GLXFile) validateMarriageBeforeBirth(result *ValidationResult) {
	for eventID, event := range glx.Events {
		if event == nil {
			continue
		}
		if event.Type != EventTypeMarriage {
			continue
		}

		eventYear := ExtractFirstYear(string(event.Date))
		if eventYear == 0 {
			continue
		}

		for _, p := range event.Participants {
			person, ok := glx.Persons[p.Person]
			if !ok || person == nil {
				continue
			}

			birthYear := extractEventYear(glx, p.Person, EventTypeBirth)
			if birthYear == 0 {
				continue
			}

			if eventYear < birthYear {
				result.Warnings = append(result.Warnings, ValidationWarning{
					SourceType: EntityTypeEvents,
					SourceID:   eventID,
					Field:      "date",
					Message: fmt.Sprintf("%s[%s]: marriage year (%d) is before participant %s birth year (%d)",
						EntityTypeEvents, eventID, eventYear, p.Person, birthYear),
				})
			}
		}
	}
}

// validateRelationshipEventOrder checks that a relationship's start event does
// not occur after its end event. Relationships where either event reference is
// missing, unresolvable, or has no parseable year are skipped.
func (glx *GLXFile) validateRelationshipEventOrder(result *ValidationResult) {
	for relID, rel := range glx.Relationships {
		if rel == nil {
			continue
		}
		if rel.StartEvent == "" || rel.EndEvent == "" {
			continue
		}

		startYear := resolveEventYear(glx, rel.StartEvent)
		endYear := resolveEventYear(glx, rel.EndEvent)
		if startYear == 0 || endYear == 0 {
			continue
		}

		if endYear < startYear {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeRelationships,
				SourceID:   relID,
				Field:      "end_event",
				Message: fmt.Sprintf("%s[%s]: end event %s year (%d) is before start event %s year (%d)",
					EntityTypeRelationships, relID, rel.EndEvent, endYear, rel.StartEvent, startYear),
			})
		}
	}
}

// validateRelationshipBoundarySources checks that a relationship records each
// of its boundaries once: either as an event reference (`start_event` /
// `end_event`) or as a date property (`started_on` / `ended_on`), not both.
// Setting both is redundant because tooling reads the event, and the two can
// drift apart. Boundaries where the event reference is unresolvable or where
// either date has no parseable year are skipped — a dangling reference is
// already reported as an error by reference validation.
func (glx *GLXFile) validateRelationshipBoundarySources(result *ValidationResult) {
	for relID, rel := range glx.Relationships {
		if rel == nil {
			continue
		}

		glx.checkBoundarySource(relID, rel.StartEvent, RelationshipPropertyStartedOn, "start_event", rel, result)
		glx.checkBoundarySource(relID, rel.EndEvent, RelationshipPropertyEndedOn, "end_event", rel, result)
	}
}

// checkBoundarySource warns when one relationship boundary is recorded both as
// an event reference and as a date property.
func (glx *GLXFile) checkBoundarySource(
	relID, eventID, propName, eventField string,
	rel *Relationship,
	result *ValidationResult,
) {
	if eventID == "" {
		return
	}

	propValue, ok := rel.Properties[propName]
	if !ok {
		return
	}

	propYear := ExtractFirstYear(propertyDateString(propValue))
	eventYear := resolveEventYear(glx, eventID)
	if propYear == 0 || eventYear == 0 {
		return
	}

	field := "properties." + propName

	var message string
	if propYear == eventYear {
		message = fmt.Sprintf(
			"%s[%s]: %s %s and %s both record the same boundary; the event is authoritative — if the property is the more precise date, move it onto the event, then remove %s",
			EntityTypeRelationships, relID, eventField, eventID, field, field,
		)
	} else {
		message = fmt.Sprintf(
			"%s[%s]: %s %s (%d) and %s (%d) record different years; the event is authoritative — reconcile the dates, then remove %s",
			EntityTypeRelationships, relID, eventField, eventID, eventYear, field, propYear, field,
		)
	}

	result.Warnings = append(result.Warnings, ValidationWarning{
		SourceType: EntityTypeRelationships,
		SourceID:   relID,
		Field:      field,
		Message:    message,
	})
}

// propertyDateString extracts a date string from a property value that may be
// a plain string, a structured {value, ...} object, or a temporal list of
// {value, date} objects (or plain strings). The first usable
// value wins; an empty string is returned when no value can be extracted.
func propertyDateString(propValue any) string {
	switch v := propValue.(type) {
	case string:
		return v
	case map[string]any:
		if value, isString := v["value"].(string); isString {
			return value
		}
	case []any:
		for _, item := range v {
			switch entry := item.(type) {
			case string:
				if entry != "" {
					return entry
				}
			case map[string]any:
				if value, isString := entry["value"].(string); isString && value != "" {
					return value
				}
			}
		}
	}

	return ""
}

// resolveEventYear resolves an event reference to the year of its date, or 0
// if the event does not exist or its date has no parseable year.
func resolveEventYear(archive *GLXFile, eventID string) int {
	event, ok := archive.Events[eventID]
	if !ok || event == nil {
		return 0
	}

	return ExtractFirstYear(string(event.Date))
}

// isParentChildRelType returns true for relationship types that model a
// parent-child connection.
func isParentChildRelType(relType string) bool {
	switch relType {
	case RelationshipTypeParentChild, RelationshipTypeBiologicalParentChild, RelationshipTypeAdoptiveParentChild,
		RelationshipTypeFosterParentChild, RelationshipTypeStepParent:
		return true
	}

	return false
}

// ExtractFirstYear extracts the first (or start) year from a date string.
// It is a convenience wrapper over glxdate.Parse: for ranges only the start
// date is considered (for an open-start "TO 1950" the end year, the only
// one present), a BCE year is negative, non-Gregorian bodies take the year
// that follows the day and month ("HEBREW 15 TSH 5765" → 5765), and
// raw-preserved Gregorian bodies prefer a 4-digit token so a day of month
// is never reported as the year ("1 JANUARY 1900" → 1900, not 1). Returns 0
// if no year is found.
func ExtractFirstYear(dateStr string) int {
	return DateString(dateStr).Year()
}
