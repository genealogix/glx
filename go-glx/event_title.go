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
	"strconv"
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
// For individual events: "Birth of John Smith (1850)"
// For couple events: "Marriage of John Smith and Jane Doe (1850)"
// Falls back to just the event type label if no names are available.
func GenerateEventTitle(eventType string, personNames []string, date DateString) string {
	label := eventTypeLabel(eventType)
	names := filterNonEmpty(personNames)
	dateYear := extractYear(date)

	switch {
	case len(names) == 0 && dateYear != "":
		return fmt.Sprintf("%s (%s)", label, dateYear)
	case len(names) == 0:
		return label
	case len(names) == 1 && dateYear != "":
		return fmt.Sprintf("%s of %s (%s)", label, names[0], dateYear)
	case len(names) == 1:
		return fmt.Sprintf("%s of %s", label, names[0])
	case dateYear != "":
		return fmt.Sprintf("%s of %s and %s (%s)", label, names[0], names[1], dateYear)
	default:
		return fmt.Sprintf("%s of %s and %s", label, names[0], names[1])
	}
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

// extractYear extracts the start year from a DateString as a string.
// Calendar-aware: handles Gregorian, Julian, Hebrew, and French Republican formats.
// Delegates to ExtractFirstYear for the actual extraction logic.
func extractYear(date DateString) string {
	s := string(date)
	if s == "" {
		return ""
	}

	// Use calendar-aware extraction (delegates to ExtractFirstYear logic)
	year := ExtractFirstYear(s)
	if year == 0 {
		return ""
	}
	return strconv.Itoa(year)
}
