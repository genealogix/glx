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
	"slices"
)

// personFactIndex is local to one research operation. Assertions are distributed
// to their selected subjects/participants once, in assertion-ID order; vital
// events are indexed in event-ID order. No archive or global state is changed.
type personFactIndex struct {
	assertions  map[string][]proofAssertion
	vitalEvents map[string]map[string][]string
}

type researchScope struct {
	kind         string
	participants []Participant
}

func newPersonFactIndex(archive *GLXFile, personIDs []string) *personFactIndex {
	wanted := make(map[string]bool, len(personIDs))
	for _, id := range personIDs {
		wanted[id] = true
	}
	index := &personFactIndex{
		assertions:  make(map[string][]proofAssertion, len(personIDs)),
		vitalEvents: make(map[string]map[string][]string),
	}
	events := index.eventScopes(archive, wanted)
	relationships := make(map[string]researchScope)
	for id, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		participants := selectedResearchParticipants(rel.Participants, wanted)
		if len(participants) > 0 {
			relationships[id] = researchScope{kind: rel.Type, participants: participants}
		}
	}
	seen := make(map[string]bool)
	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		clear(seen)
		if wanted[a.Subject.Person] {
			personID := a.Subject.Person
			index.assertions[personID] = append(index.assertions[personID], proofAssertion{id: id, a: a, subjectID: personID})
			seen[personID] = true
		}
		event := events[a.Subject.Event]
		for _, p := range event.participants {
			if seen[p.Person] {
				continue
			}
			index.assertions[p.Person] = append(index.assertions[p.Person], proofAssertion{id: id, a: a, subjectID: a.Subject.Event, eventType: event.kind})
			seen[p.Person] = true
		}
		rel := relationships[a.Subject.Relationship]
		for _, p := range rel.participants {
			if seen[p.Person] {
				continue
			}
			index.assertions[p.Person] = append(index.assertions[p.Person], proofAssertion{id: id, a: a, subjectID: a.Subject.Relationship, relType: rel.kind, personRole: p.Role})
			seen[p.Person] = true
		}
	}

	return index
}

func (index *personFactIndex) eventScopes(archive *GLXFile, wanted map[string]bool) map[string]researchScope {
	scopes := make(map[string]researchScope)
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		participants := selectedResearchParticipants(event.Participants, wanted)
		if len(participants) > 0 {
			scopes[id] = researchScope{kind: event.Type, participants: participants}
		}
		if event.Type != EventTypeBirth && event.Type != EventTypeDeath {
			continue
		}
		seen := make(map[string]bool)
		// Check every original role: someone listed as both witness and subject is
		// a principal even when the witness entry happens to appear first.
		for _, p := range event.Participants {
			if !wanted[p.Person] || seen[p.Person] || !isVitalPrincipal(p.Role) {
				continue
			}
			seen[p.Person] = true
			if index.vitalEvents[p.Person] == nil {
				index.vitalEvents[p.Person] = make(map[string][]string)
			}
			index.vitalEvents[p.Person][event.Type] = append(index.vitalEvents[p.Person][event.Type], id)
		}
	}

	return scopes
}

func selectedResearchParticipants(participants []Participant, wanted map[string]bool) []Participant {
	var selected []Participant
	seen := make(map[string]bool)
	for _, p := range participants {
		if !wanted[p.Person] || seen[p.Person] {
			continue
		}
		seen[p.Person] = true
		// Only identity/role are needed. Preserve the first relationship role,
		// matching person-level proof collection, without retaining properties.
		selected = append(selected, Participant{Person: p.Person, Role: p.Role})
	}

	return selected
}

func isVitalPrincipal(role string) bool {
	switch role {
	case "", ParticipantRoleSubject, ParticipantRolePrincipal, ParticipantRoleChild, "deceased":
		return true
	default:
		return false
	}
}

func (index *personFactIndex) collect(archive *GLXFile, personID string) []proofAssertion {
	// Duplicate-event grouping annotates these records, so each collection owns
	// its slice even if the index is reused for the same person.
	out := slices.Clone(index.assertions[personID])

	return appendDuplicateEventFacts(out, archive, personID, index.vitalEvents[personID])
}
