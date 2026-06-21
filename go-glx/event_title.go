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
	"strings"
)

// eventTypeLabels maps GLX event types to human-readable labels for title generation.
var eventTypeLabels = map[string]string{
	EventTypeBirth:              "Birth",
	EventTypeDeath:              "Death",
	EventTypeBurial:             "Burial",
	EventTypeChristening:        "Christening",
	EventTypeAdultChristening:   "Adult Christening",
	EventTypeBaptism:            "Baptism",
	EventTypeConfirmation:       "Confirmation",
	EventTypeGraduation:         "Graduation",
	EventTypeRetirement:         "Retirement",
	EventTypeCremation:          "Cremation",
	EventTypeAdoption:           "Adoption",
	EventTypeProbate:            "Probate",
	EventTypeWill:               "Will",
	EventTypeImmigration:        "Immigration",
	EventTypeEmigration:         "Emigration",
	EventTypeNaturalization:     "Naturalization",
	EventTypeCensus:             "Census",
	EventTypeResidence:          "Residence",
	EventTypeOccupation:         "Occupation",
	EventTypeEducation:          "Education",
	EventTypeMarriage:           "Marriage",
	EventTypeDivorce:            "Divorce",
	EventTypeAnnulment:          "Annulment",
	EventTypeEngagement:         "Engagement",
	EventTypeDivorceFiled:       "Divorce Filed",
	EventTypeMarriageBanns:      "Marriage Banns",
	EventTypeMarriageContract:   "Marriage Contract",
	EventTypeMarriageLicense:    "Marriage License",
	EventTypeMarriageSettlement: "Marriage Settlement",
	EventTypeLegalSeparation:    "Legal Separation",
	EventTypeBarMitzvah:         "Bar Mitzvah",
	EventTypeBatMitzvah:         "Bat Mitzvah",
	EventTypeBlessing:           "Blessing",
	EventTypeFirstCommunion:     "First Communion",
	EventTypeOrdination:         "Ordination",
	EventTypeTaxation:           "Taxation",
	EventTypeVoterRegistration:  "Voter Registration",
	EventTypeGeneric:            "Event",
}

// GenerateEventTitle builds a human-readable title for an event.
// For individual events: "Birth of John Smith"
// For couple events: "Marriage of John Smith and Jane Doe"
// Falls back to just the event type label if no names are available.
//
// The event's date is intentionally omitted: it is already carried by the
// event's `date` field, so repeating it in the title would be redundant.
func GenerateEventTitle(eventType string, personNames []string) string {
	label := eventTypeLabel(eventType)
	names := filterNonEmpty(personNames)

	switch len(names) {
	case 0:
		return label
	case 1:
		return fmt.Sprintf("%s of %s", label, names[0])
	default:
		return fmt.Sprintf("%s of %s and %s", label, names[0], names[1])
	}
}

// staleTitleYearSuffix matches the trailing " (YEAR)" suffix that pre-#1026
// GEDCOM imports appended to auto-generated event titles. It matches 1–4
// ASCII digits because the suffix was derived by ExtractFirstYear, which
// never emitted more — this also covers its buggy day-as-year output
// (e.g. "(15)", see #1025).
var staleTitleYearSuffix = regexp.MustCompile(` \([0-9]{1,4}\)$`)

// StripStaleEventTitleYear removes the stale " (YEAR)" suffix that GEDCOM
// imports before #1026 appended to auto-generated event titles, returning the
// title unchanged when no stale suffix is recognized.
//
// A hand-authored title must never be altered, so a bare "strip trailing
// digits in parentheses" is not enough. The suffix is removed only when the
// remaining body is byte-for-byte identical to the one title
// GenerateEventTitle would produce for this event from its type and full
// participant-name list — the exact shapes import generated ("Burial",
// "Birth of John Smith", "Marriage of Robert Webb and Jane Miller"). Titles
// like "Family reunion (1955)" or "Census (Webb Household)" fail that gate
// and are preserved.
//
// Deliberately, no partial-name shape is accepted: for an event whose
// participants are Robert Webb and Jane Miller, "Marriage of Robert Webb
// (1850)" is NOT stripped — it doesn't match the auto title for the event's
// current participant set, so it is treated as hand-authored. A stale title
// whose participant set changed after import (a spouse added, a person
// renamed) is likewise left alone; leaving a stale suffix in place is benign,
// while stripping a hand-authored title is not.
//
// participantNames are the display names of the event's principal/spouse
// participants in stored order — the names import titled the event with.
//
// Known residual: a user who hand-typed a title identical to the auto-shape
// including the year is indistinguishable from a stale auto-title and will be
// stripped; the result equals the current canonical title and the year
// remains in the event's date field.
func StripStaleEventTitleYear(title, eventType string, participantNames []string) string {
	loc := staleTitleYearSuffix.FindStringIndex(title)
	if loc == nil {
		return title
	}
	body := title[:loc[0]]

	names := filterNonEmpty(participantNames)
	if len(names) > 2 {
		names = names[:2] // GenerateEventTitle only ever used the first two
	}
	if body == GenerateEventTitle(eventType, names) {
		return body
	}

	return title
}

// PersonDisplayName extracts a display name from a Person's properties.
// Returns an empty string if no name is found.
func PersonDisplayName(person *Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}
	raw, ok := person.Properties[PersonPropertyName]
	if !ok {
		raw, ok = person.Properties["primary_name"]
	}
	if !ok {
		return ""
	}

	if s, ok := raw.(string); ok {
		return s
	}
	if m, ok := raw.(map[string]any); ok {
		if v, ok := m["value"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	if list, ok := raw.([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if v, ok := m["value"]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
	}
	return ""
}

func eventTypeLabel(eventType string) string {
	if label, ok := eventTypeLabels[eventType]; ok {
		return label
	}
	// Convert snake_case to Title Case as fallback
	words := strings.Split(eventType, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func filterNonEmpty(ss []string) []string {
	var result []string
	for _, s := range ss {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
