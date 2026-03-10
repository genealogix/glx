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
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

// analyzeGaps detects missing data that should be findable for each person.
func analyzeGaps(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	personEvents := buildPersonEventIndex(archive)

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		name := personName(archive, id)

		issues = append(issues, checkMissingBirth(id, name, person)...)
		issues = append(issues, checkMissingDeath(id, name, person)...)
		issues = append(issues, checkNoParents(id, name, archive)...)
		issues = append(issues, checkNoEvents(id, name, personEvents)...)
		issues = append(issues, checkNoMarriageEvent(id, name, archive)...)
	}

	return issues
}

// checkMissingBirth reports persons with no birth date or place.
func checkMissingBirth(id, name string, person *glxlib.Person) []AnalysisIssue {
	bornOn := propertyString(person.Properties, "born_on")
	bornAt := propertyString(person.Properties, "born_at")
	if bornOn != "" || bornAt != "" {
		return nil
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "high",
		Person:   id,
		Message:  fmt.Sprintf("%s — no birth date or place", name),
		Property: "born_on",
	}}
}

// checkMissingDeath reports persons with a birth but no death info who are
// unlikely to still be alive (born more than 110 years ago).
func checkMissingDeath(id, name string, person *glxlib.Person) []AnalysisIssue {
	diedOn := propertyString(person.Properties, "died_on")
	diedAt := propertyString(person.Properties, "died_at")
	if diedOn != "" || diedAt != "" {
		return nil
	}

	bornOn := propertyString(person.Properties, "born_on")
	if bornOn == "" {
		return nil
	}

	birthYear := glxlib.ExtractFirstYear(bornOn)
	cutoff := time.Now().Year() - 110
	if birthYear == 0 || birthYear > cutoff {
		// Unknown birth year or could still be alive — skip.
		return nil
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "high",
		Person:   id,
		Message:  fmt.Sprintf("%s — no death date or place", name),
		Property: "died_on",
	}}
}

// checkNoParents reports persons with no parent_child relationship as child.
func checkNoParents(id, name string, archive *glxlib.GLXFile) []AnalysisIssue {
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if !isParentChildType(rel.Type) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == id && p.Role == glxlib.ParticipantRoleChild {
				return nil
			}
		}
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "medium",
		Person:   id,
		Message:  fmt.Sprintf("%s — no parents (no parent_child relationship as child)", name),
	}}
}

// checkNoEvents reports persons who participate in zero events.
func checkNoEvents(id, name string, personEvents map[string]int) []AnalysisIssue {
	if personEvents[id] > 0 {
		return nil
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "high",
		Person:   id,
		Message:  fmt.Sprintf("%s — no events (person participates in zero events)", name),
	}}
}

// checkNoMarriageEvent reports persons who have a spouse relationship but
// no marriage event with a date/place.
func checkNoMarriageEvent(id, name string, archive *glxlib.GLXFile) []AnalysisIssue {
	hasSpouse := false
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if rel.Type != glxlib.RelationshipTypeMarriage && rel.Type != glxlib.RelationshipTypePartner {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == id {
				hasSpouse = true
				break
			}
		}
		if hasSpouse {
			break
		}
	}

	if !hasSpouse {
		return nil
	}

	// Check if there's a marriage event for this person
	for _, event := range archive.Events {
		if event == nil {
			continue
		}
		if event.Type != glxlib.EventTypeMarriage {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == id && (event.Date != "" || event.PlaceID != "") {
				return nil
			}
		}
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "medium",
		Person:   id,
		Message:  fmt.Sprintf("%s — no marriage event (spouse relationship exists but no date/place)", name),
	}}
}

// buildPersonEventIndex counts how many events each person participates in.
func buildPersonEventIndex(archive *glxlib.GLXFile) map[string]int {
	counts := make(map[string]int)
	for _, event := range archive.Events {
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			counts[p.Person]++
		}
	}
	return counts
}

