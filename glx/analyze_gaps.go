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
	childHasParents := buildChildHasParentsIndex(archive)
	spouseRels := buildSpouseRelIndex(archive)
	marriagePairs := buildMarriagePairIndex(archive)

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		name := personName(archive, id)

		issues = append(issues, checkMissingBirth(archive, id, name)...)
		issues = append(issues, checkMissingDeath(archive, id, name)...)
		if !childHasParents[id] {
			issues = append(issues, AnalysisIssue{
				Category: "gap",
				Severity: "medium",
				Person:   id,
				Message:  fmt.Sprintf("%s — no parents (no parent_child relationship as child)", name),
			})
		}
		issues = append(issues, checkNoEvents(id, name, personEvents)...)

		// Check each spouse relationship for a corresponding marriage event
		for _, sp := range spouseRels[id] {
			pairKey := marriagePairKey(id, sp.spouseID)
			if !marriagePairs[pairKey] {
				spouseName := personName(archive, sp.spouseID)
				issues = append(issues, AnalysisIssue{
					Category: "gap",
					Severity: "medium",
					Person:   id,
					Message:  fmt.Sprintf("%s — no marriage event for %s (spouse relationship exists but no date/place)", name, spouseName),
				})
			}
		}
	}

	return issues
}

// buildChildHasParentsIndex returns a set of person IDs that appear as a child
// in at least one parent-child relationship.
func buildChildHasParentsIndex(archive *glxlib.GLXFile) map[string]bool {
	index := make(map[string]bool)
	for _, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Role == glxlib.ParticipantRoleChild {
				index[p.Person] = true
			}
		}
	}
	return index
}

// spouseRef holds a spouse person ID for gap analysis.
type spouseRef struct {
	spouseID string
}

// buildSpouseRelIndex returns a map from person ID to their spouse relationships.
// Entries are sorted by relationship ID for deterministic output.
func buildSpouseRelIndex(archive *glxlib.GLXFile) map[string][]spouseRef {
	index := make(map[string][]spouseRef)
	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		if rel == nil {
			continue
		}
		if rel.Type != glxlib.RelationshipTypeMarriage && rel.Type != glxlib.RelationshipTypePartner {
			continue
		}
		for i, p := range rel.Participants {
			for j, q := range rel.Participants {
				if i != j && p.Person != "" && q.Person != "" && p.Person != q.Person {
					index[p.Person] = append(index[p.Person], spouseRef{spouseID: q.Person})
				}
			}
		}
	}
	return index
}

// buildMarriagePairIndex returns a set of (personA, personB) pairs that share a
// marriage event with a date or place. Also checks relationship start_event refs.
func buildMarriagePairIndex(archive *glxlib.GLXFile) map[string]bool {
	index := make(map[string]bool)

	// From marriage events
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeMarriage {
			continue
		}
		if event.Date == "" && event.PlaceID == "" {
			continue
		}
		for i, p := range event.Participants {
			for j, q := range event.Participants {
				if i != j && p.Person != "" && q.Person != "" {
					index[marriagePairKey(p.Person, q.Person)] = true
				}
			}
		}
	}

	// From relationship start_event refs
	for _, rel := range archive.Relationships {
		if rel == nil || rel.StartEvent == "" {
			continue
		}
		if rel.Type != glxlib.RelationshipTypeMarriage && rel.Type != glxlib.RelationshipTypePartner {
			continue
		}
		ev, ok := archive.Events[rel.StartEvent]
		if !ok || ev == nil || ev.Type != glxlib.EventTypeMarriage {
			continue
		}
		if ev.Date == "" && ev.PlaceID == "" {
			continue
		}
		for i, p := range rel.Participants {
			for j, q := range rel.Participants {
				if i != j && p.Person != "" && q.Person != "" {
					index[marriagePairKey(p.Person, q.Person)] = true
				}
			}
		}
	}

	return index
}

// marriagePairKey returns a canonical key for a pair of person IDs.
func marriagePairKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

// checkMissingBirth reports persons with no birth event (no date or place).
func checkMissingBirth(archive *glxlib.GLXFile, id, name string) []AnalysisIssue {
	_, birthEvent := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeBirth)
	if birthEvent != nil && (birthEvent.Date != "" || birthEvent.PlaceID != "") {
		return nil
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "high",
		Person:   id,
		Message:  fmt.Sprintf("%s — no birth date or place", name),
		Property: "birth_event",
	}}
}

// checkMissingDeath reports persons with a birth but no death info who are
// unlikely to still be alive (born more than 110 years ago).
func checkMissingDeath(archive *glxlib.GLXFile, id, name string) []AnalysisIssue {
	_, deathEvent := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeDeath)
	if deathEvent != nil && (deathEvent.Date != "" || deathEvent.PlaceID != "") {
		return nil
	}

	_, birthEvent := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeBirth)
	if birthEvent == nil || birthEvent.Date == "" {
		return nil
	}

	birthYear := glxlib.ExtractFirstYear(string(birthEvent.Date))
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
		Property: "death_event",
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

