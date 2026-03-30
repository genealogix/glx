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
	"regexp"
	"strconv"
)

// dayMonthRegexp matches day-of-month followed by a month abbreviation
// (e.g., "15 MAR"). Used to strip day values before year extraction so that
// 1–2 digit days are not mistaken for 1–2 digit years.
var dayMonthRegexp = regexp.MustCompile(`(?i)\b\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\b`)

// temporalYearRegexp matches the first 1–4 digit year in a date string.
var temporalYearRegexp = regexp.MustCompile(`\b(\d{1,4})\b`)


// validateTemporalConsistency checks for logical inconsistencies in dates
// across persons, events, and relationships. All issues are reported as
// warnings since dates are often estimates (ABT, BEF, etc.).
func (glx *GLXFile) validateTemporalConsistency(result *ValidationResult) {
	glx.validateDeathBeforeBirth(result)
	glx.validateParentChildAges(result)
	glx.validateMarriageBeforeBirth(result)
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

// ExtractPropertyYear extracts the first year (1–4 digits) from a person property.
// Handles simple string values, structured maps with a "value" key, and
// temporal lists where each entry has a "value" key.
func ExtractPropertyYear(props map[string]any, key string) int {
	raw, ok := props[key]
	if !ok {
		return 0
	}

	var dateStr string

	switch v := raw.(type) {
	case string:
		dateStr = v
	case map[string]any:
		if val, ok := v["value"]; ok {
			dateStr = fmt.Sprint(val)
		}
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if val, ok := m["value"]; ok {
					dateStr = fmt.Sprint(val)
				}
			}
		}
	}

	return ExtractFirstYear(dateStr)
}

// ExtractFirstYear extracts the first year (1–4 digits) from a date string.
// Day-of-month values (e.g., the "15" in "15 MAR 1850") are stripped first
// so they are not mistaken for years. Returns 0 if no year is found.
func ExtractFirstYear(dateStr string) int {
	if dateStr == "" {
		return 0
	}

	cleaned := dayMonthRegexp.ReplaceAllString(dateStr, "")

	match := temporalYearRegexp.FindStringSubmatch(cleaned)
	if len(match) < 2 {
		return 0
	}

	year, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return year
}
