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
	"strings"
)

func comparisonOptions(options []ComparisonOptions) ComparisonOptions {
	if len(options) > 0 {
		return options[0]
	}

	return ComparisonOptions{}
}

func verdictRank(v Verdict) int {
	switch v {
	case VerdictDefinite:
		return 3
	case VerdictPossible:
		return 2
	case VerdictDisputed:
		return 1
	default:
		return 0
	}
}

// Non-property assertions cannot enter a property/value comparison, but an
// explicit dispute about participation or existence still needs resolution.
// Let the shared engine classify the status, just as for a singleton property.
func nonPropertyProofDispute(pa *proofAssertion, archive *GLXFile, opts ComparisonOptions) (ConflictGroup, bool) {
	evaluation := evaluateFacts([]FactValue{AssertionFact(pa.a)}, nil, archive.Places, opts)
	comparisons := evaluation.Comparisons
	if len(comparisons) == 0 || !comparisons[0].IsConflict() {
		return ConflictGroup{}, false
	}
	subject := describeProofSubject(pa, archive)
	if subject == "" {
		subject = pa.subjectID
	}
	property, value := "existence", "existence asserted"
	if participant := pa.a.Participant; participant != nil {
		property = "participation"
		value = personName(archive, participant.Person)
		if participant.Role != "" {
			value += " (" + participant.Role + ")"
		}
	}

	return ConflictGroup{
		Facts: []ResearchFact{researchFact(pa, archive)}, Evaluation: evaluation,
		Subject: subject, Property: property,
		Verdict: comparisons[0].Verdict, Definite: comparisons[0].Definite,
		Values: []ConflictValue{{Value: value, Confidence: pa.a.Confidence, Status: pa.a.Status}},
	}, true
}

// appendDuplicateEventFacts compares repeated birth/death events for the person
// in the principal role. Parents and witnesses do not acquire the child's birth.
// A structural field is used only when no assertion records that field, so a
// denormalized conclusion never revives a researcher's disproven claim.
func appendDuplicateEventFacts(out []proofAssertion, archive *GLXFile, personID string) []proofAssertion {
	groups := duplicateVitalEventIDs(archive, personID)
	for _, eventType := range []string{EventTypeBirth, EventTypeDeath} {
		ids := groups[eventType]
		if len(ids) < 2 {
			continue
		}
		for _, id := range ids {
			seen := make(map[string]bool)
			key := personID + ":" + eventType
			for i := range out {
				if out[i].a.Subject.Event == id {
					out[i].factKey = key
					seen[out[i].a.Property] = true
				}
			}
			ev := archive.Events[id]
			for _, field := range []struct{ property, value string }{{conflictDateType, string(ev.Date)}, {eventFieldPlace, ev.PlaceID}} {
				if field.value == "" || seen[field.property] {
					continue
				}
				a := &Assertion{Subject: EntityRef{Event: id}, Property: field.property, Value: field.value}
				out = append(out, proofAssertion{id: id + ":" + field.property, a: a, subjectID: id, eventType: eventType, factKey: key})
			}
		}
	}

	return out
}

func factDisplay(value string, subject EntityRef, property string, archive *GLXFile) string {
	display := resolveAssertionValue(value, property, subject, archive)
	def := ConflictProperty(archive, subject, property)
	if def != nil && def.ReferenceType == "places" {
		if p := archive.Places[value]; p != nil {
			for id, other := range archive.Places {
				if id != value && other != nil && strings.EqualFold(other.Name, p.Name) {
					return display + " (" + value + ")"
				}
			}
		}
	}

	return display
}

func duplicateVitalEventIDs(archive *GLXFile, personID string) map[string][]string {
	groups := make(map[string][]string)
	for _, id := range sortedKeys(archive.Events) {
		ev := archive.Events[id]
		if ev == nil || (ev.Type != EventTypeBirth && ev.Type != EventTypeDeath) {
			continue
		}
		for _, p := range ev.Participants {
			if p.Person == personID && (p.Role == "subject" || p.Role == "principal" || p.Role == "" || p.Role == ParticipantRoleChild || p.Role == "deceased") {
				groups[ev.Type] = append(groups[ev.Type], id)

				break
			}
		}
	}

	return groups
}

// researchConfidenceRank returns a numeric rank for confidence levels (lower = higher confidence).
func researchConfidenceRank(c string) int {
	switch strings.ToLower(c) {
	case "high":
		return 0
	case "medium-high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}

// personName returns the display name for a person ID, or the ID itself.
func personName(archive *GLXFile, personID string) string {
	if person, ok := archive.Persons[personID]; ok && person != nil {
		name := PersonDisplayName(person)
		if name != "" {
			return name
		}
	}

	return personID
}

// hasParticipant checks if a person is among participants.
func hasParticipant(personID string, participants []Participant) bool {
	for _, p := range participants {
		if p.Person == personID {
			return true
		}
	}

	return false
}

// resolvePlaceName looks up a place ID and returns its name.
func resolvePlaceName(placeID string, archive *GLXFile) string {
	if placeID == "" {
		return ""
	}
	if place, ok := archive.Places[placeID]; ok && place != nil {
		return place.Name
	}

	return placeID
}

// resolveSourceTitle looks up the source title for a citation.
func resolveSourceTitle(sourceID string, archive *GLXFile) string {
	if sourceID == "" {
		return ""
	}
	if src, ok := archive.Sources[sourceID]; ok {
		return src.Title
	}

	return ""
}

// citationProperty extracts a string property from citation properties.
func citationProperty(cit *Citation, key string) string {
	return propertyString(cit.Properties, key)
}

// deathYearUpperBound returns the effective upper bound year for census
// suggestions from a death date property value. Handles string, structured
// map ({value: "BEF 1870"}), and temporal list ([{value: "BEF 1870"}]) shapes.
// For "BEF <year>" dates, the year is decremented by 1 since the person
// died before that year. Calendar prefixes (e.g. "JULIAN BEF 1870") are
// stripped before the qualifier check.
func deathYearUpperBound(raw any) int {
	dateStr := extractDateString(raw)
	year := ExtractFirstYear(dateStr)
	if year > 0 && strings.HasPrefix(dateStringWithoutCalendarPrefix(dateStr), "BEF ") {
		year--
	}

	return year
}

// extractDateString extracts the date string from a property value,
// handling string, structured map, and temporal list shapes.
func extractDateString(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		if val, ok := v["value"]; ok {
			return fmt.Sprint(val)
		}
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if val, ok := m["value"]; ok {
					return fmt.Sprint(val)
				}
			}
		}
	}

	return ""
}

// placeRefProperties is the set form of placeRefPropertyKeys for quick lookup.
var placeRefProperties = map[string]bool{
	"buried_at": true, "residence": true,
}

// collectPlaceRefsFromProperty extracts place IDs from a property value,
// handling string, structured map ({value: ...}), and temporal list shapes.
func collectPlaceRefsFromProperty(raw any, referenced map[string]struct{}) {
	switch v := raw.(type) {
	case string:
		if v != "" {
			referenced[v] = struct{}{}
		}
	case map[string]any:
		if val, ok := v["value"].(string); ok && val != "" {
			referenced[val] = struct{}{}
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if val, ok := m["value"].(string); ok && val != "" {
					referenced[val] = struct{}{}
				}
			} else if s, ok := item.(string); ok && s != "" {
				referenced[s] = struct{}{}
			}
		}
	}
}

const (
	statusDisputed  = "disputed"
	severityHigh    = "high"
	severityMedium  = "medium"
	eventFieldPlace = "place"
)

const minorAgeUnder = 18

// dateStringWithoutCalendarPrefix returns the upper-cased body of a GLX date
// string with any leading calendar prefix removed (e.g. "JULIAN AFT 1731" →
// "AFT 1731"). It centralizes the prefix-stripping step that all of the
// qualifier-aware date helpers in this file rely on so they recognize BEF /
// AFT regardless of the calendar in which the date is expressed.
func dateStringWithoutCalendarPrefix(dateStr string) string {
	_, body := ExtractCalendarPrefix(DateString(strings.TrimSpace(dateStr)))

	return strings.ToUpper(string(body))
}

// propertyString extracts a simple string value from properties.
func propertyString(props map[string]any, key string) string {
	raw, ok := props[key]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}

	return fmt.Sprint(raw)
}

func equalStatus(a, b string) bool { return strings.EqualFold(a, b) }

func researchIsParentChildType(relType string) bool {
	switch relType {
	case RelationshipTypeParentChild, RelationshipTypeBiologicalParentChild,
		RelationshipTypeAdoptiveParentChild, RelationshipTypeFosterParentChild,
		RelationshipTypeStepParent:
		return true
	}

	return false
}
