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
				Message:  name + " — no parents (no parent_child relationship as child)",
			})
		}
		issues = append(issues, checkNoEvents(id, name, personEvents)...)

		// Check each spouse relationship for a corresponding marriage event
		for _, sp := range spouseRels[id] {
			pairKey := marriagePairKey(id, sp.spouseID)
			evidence := marriagePairs[pairKey]
			switch {
			case evidence.ceremony:
				// A dated or placed marriage: nothing to report.
			case evidence.preliminaryType != "":
				spouseName := personName(archive, sp.spouseID)
				issues = append(issues, AnalysisIssue{
					Category: "gap",
					Severity: severityInfo,
					Person:   id,
					Message: fmt.Sprintf("%s — marriage to %s known from %s only; search for a minister's return or marriage register entry",
						name, spouseName, preliminaryMarriageRecordName(evidence.preliminaryType)),
				})
			default:
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
		if !glxlib.IsCoupleRelationshipType(rel.Type) {
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

// preliminaryMarriageTypes are the marriage-family event types that record a
// step towards a marriage rather than the ceremony itself: a license or bond,
// banns, a contract, a settlement. For colonial and early-republic marriages
// one of these is often the only surviving record, so a dated or placed one
// is evidence that the couple married, though not of the wedding (#1338).
var preliminaryMarriageTypes = map[string]bool{
	glxlib.EventTypeMarriageLicense:    true,
	glxlib.EventTypeMarriageBanns:      true,
	glxlib.EventTypeMarriageContract:   true,
	glxlib.EventTypeMarriageSettlement: true,
}

// preliminaryMarriageRecordName renders a preliminary marriage event type as
// the record it names in prose, e.g. "a marriage license".
func preliminaryMarriageRecordName(eventType string) string {
	switch eventType {
	case glxlib.EventTypeMarriageBanns:
		return "marriage banns"
	case glxlib.EventTypeMarriageContract:
		return "a marriage contract"
	case glxlib.EventTypeMarriageSettlement:
		return "a marriage settlement"
	default:
		return "a marriage license"
	}
}

// marriageEvidence records what a couple's marriage is known from.
type marriageEvidence struct {
	// ceremony is true when a dated or placed marriage event links the pair.
	ceremony bool
	// preliminaryType is the type of a dated or placed license, banns,
	// contract or settlement linking the pair, when there is one. When the
	// pair has several, the lexically smallest type is kept so the message is
	// deterministic.
	preliminaryType string
}

// recordMarriageEvidence notes that the event links every pair of the given
// participants, if the event is a dated or placed marriage-family event.
func recordMarriageEvidence(index map[string]marriageEvidence, event *glxlib.Event, participants []glxlib.Participant) {
	if event == nil || (event.Date == "" && event.PlaceID == "") {
		return
	}
	isCeremony := event.Type == glxlib.EventTypeMarriage
	if !isCeremony && !preliminaryMarriageTypes[event.Type] {
		return
	}
	for i, p := range participants {
		for j, q := range participants {
			if i == j || p.Person == "" || q.Person == "" {
				continue
			}
			key := marriagePairKey(p.Person, q.Person)
			evidence := index[key]
			if isCeremony {
				evidence.ceremony = true
			} else if evidence.preliminaryType == "" || event.Type < evidence.preliminaryType {
				evidence.preliminaryType = event.Type
			}
			index[key] = evidence
		}
	}
}

// buildMarriagePairIndex returns, for each (personA, personB) pair, what their
// marriage is known from: a shared dated or placed marriage event, or a
// license, banns, contract or settlement (#1338). Also follows couple
// relationships' start_event refs, whose participants are the relationship's.
func buildMarriagePairIndex(archive *glxlib.GLXFile) map[string]marriageEvidence {
	index := make(map[string]marriageEvidence)

	// From events the couple both participate in
	for _, event := range archive.Events {
		if event == nil {
			continue
		}
		recordMarriageEvidence(index, event, event.Participants)
	}

	// From relationship start_event refs
	for _, rel := range archive.Relationships {
		if rel == nil || rel.StartEvent == "" {
			continue
		}
		if !glxlib.IsCoupleRelationshipType(rel.Type) {
			continue
		}
		recordMarriageEvidence(index, archive.Events[rel.StartEvent], rel.Participants)
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

// checkMissingBirth reports persons with no birth event (no date or place). A
// dated or placed baptism or christening of the person satisfies it: before
// civil registration that entry is the birth evidence (#1365).
func checkMissingBirth(archive *glxlib.GLXFile, id, name string) []AnalysisIssue {
	for _, eventType := range []string{glxlib.EventTypeBirth, glxlib.EventTypeBaptism, glxlib.EventTypeChristening} {
		_, event := glxlib.FindPersonEvent(archive, id, eventType)
		if event != nil && (event.Date != "" || event.PlaceID != "") {
			return nil
		}
	}

	return []AnalysisIssue{{
		Category: "gap",
		Severity: "high",
		Person:   id,
		Message:  name + " — no birth date or place",
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
		Message:  name + " — no death date or place",
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
		Message:  name + " — no events (person participates in zero events)",
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
