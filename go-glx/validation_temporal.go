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
	"strings"
)

// dayMonthRegexp matches day-of-month followed by a month abbreviation
// (e.g., "15 MAR"). Used to strip day values before year extraction so that
// 1–2 digit days are not mistaken for 1–2 digit years.
var dayMonthRegexp = regexp.MustCompile(`(?i)\b\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\b`)

// temporalYearRegexp matches the first 1–4 digit year in a date string.
var temporalYearRegexp = regexp.MustCompile(`\b(\d{1,4})\b`)

// lastYearRegexp matches 1–5 digit sequences (supports Hebrew years like 5765).
var lastYearRegexp = regexp.MustCompile(`\b(\d{1,5})\b`)


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

// ExtractFirstYear extracts the first (or start) year from a date string.
// Calendar-aware: for Gregorian/Julian dates, strips DD MMM and finds the first
// number. For Hebrew/French Republican dates, finds the last number in the first
// date component (year appears after day and month in non-Gregorian formats).
// For ranges (BET...AND, FROM...TO), only the start date is considered.
// Returns 0 if no year is found.
func ExtractFirstYear(dateStr string) int {
	if dateStr == "" {
		return 0
	}

	// Strip calendar prefix and use calendar-specific extraction.
	cal, body := ExtractCalendarPrefix(DateString(dateStr))
	bodyStr := string(body)

	// For ranges, only consider the first date (start of range).
	bodyStr = extractFirstDateComponent(bodyStr)

	// For non-Gregorian calendars where the year is the LAST token
	// (e.g., "15 TSH 5765" for Hebrew, "1 VEND 0012" for French Republican),
	// extract the last number instead of the first. Fixes #565.
	if cal == CalendarHebrew || cal == CalendarFrenchR {
		return extractLastNumber(bodyStr)
	}

	// For Gregorian/Julian: strip DD MMM patterns, then find first number.
	cleaned := dayMonthRegexp.ReplaceAllString(bodyStr, "")

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

// extractFirstDateComponent returns the first date in a range expression.
// "BET 15 TSH 5765 AND 15 TSH 5766" → "BET 15 TSH 5765"
// "FROM 1900 TO 1950" → "FROM 1900"
// "ABT 1850" → "ABT 1850" (unchanged)
func extractFirstDateComponent(dateStr string) string {
	if idx := strings.Index(dateStr, " AND "); idx != -1 {
		return dateStr[:idx]
	}
	if idx := strings.Index(dateStr, " TO "); idx != -1 {
		return dateStr[:idx]
	}
	return dateStr
}

// extractLastNumber finds the last 1–5 digit sequence in a date string.
// Used for non-Gregorian calendars where the year appears after day and month.
// Supports Hebrew years >4 digits (e.g., 5765).
func extractLastNumber(dateStr string) int {
	matches := lastYearRegexp.FindAllStringSubmatch(dateStr, -1)
	if len(matches) == 0 {
		return 0
	}

	lastMatch := matches[len(matches)-1]
	year, err := strconv.Atoi(lastMatch[1])
	if err != nil {
		return 0
	}

	return year
}
