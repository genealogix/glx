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

// EntityCoverage is a set of entity IDs per entity type (persons, events,
// relationships, places) that some assertion reaches.
type EntityCoverage struct {
	Persons       map[string]bool
	Events        map[string]bool
	Relationships map[string]bool
	Places        map[string]bool
}

func newEntityCoverage() EntityCoverage {
	return EntityCoverage{
		Persons:       make(map[string]bool),
		Events:        make(map[string]bool),
		Relationships: make(map[string]bool),
		Places:        make(map[string]bool),
	}
}

// AssertionCoverage holds two views of which entities the archive's
// assertions reach.
//
// Direct counts only an assertion's subject pointer. Evidence measures
// assertion reachability through the chain the GLX model expects researchers
// to build, where facts are usually asserted on events and relationships
// rather than on persons and places:
//
//   - an event or relationship is covered when an assertion targets it; a
//     relationship is also covered when its start_event or end_event is
//   - a person is covered when an assertion targets them, names them as
//     participant.person, or targets an event or relationship they
//     participate in (any role: a witness is reached through an assertion
//     on the marriage record), or carries them as the value of a
//     person-reference property
//   - a place is covered when an assertion targets it, carries it as the
//     value of a place-reference property (an event's `place`, a person's
//     `residence`, ...), or it is the place of a covered event; and then every
//     place on a covered place's parent chain is covered too
//
// Parent-chain places count as gazetteer scaffolding that locates a reached
// place. Reaching a county also reaches its state and country ancestors,
// without requiring independent assertions about those jurisdictions.
//
// Any assertion counts, including unsourced or disproven assertions. Coverage
// percentages measure reachability, not the proportion sourced or proven;
// assertions without citations are a separate finding (glx validate --report
// lists them).
type AssertionCoverage struct {
	Direct   EntityCoverage
	Evidence EntityCoverage
}

// ComputeAssertionCoverage computes the direct and evidence coverage of an
// archive's assertions. Missing and nil entities are ignored. Standard property
// definitions are supplied without changing archive or overriding custom
// definitions, including explicit nil definitions. See AssertionCoverage for
// the rules.
func ComputeAssertionCoverage(archive *GLXFile) AssertionCoverage {
	result := AssertionCoverage{Direct: newEntityCoverage(), Evidence: newEntityCoverage()}
	if archive == nil {
		return result
	}

	prepared, err := researchArchive(archive)
	if err != nil {
		return result
	}
	archive = prepared

	direct, evidence := result.Direct, result.Evidence

	for _, a := range archive.Assertions {
		if a == nil {
			continue
		}
		markSubject(direct, a.Subject, archive)
		markSubject(evidence, a.Subject, archive)

		if a.Participant != nil && archive.Persons[a.Participant.Person] != nil {
			evidence.Persons[a.Participant.Person] = true
		}

		markAssertionValue(evidence, a, archive)
	}

	propagateThroughEvents(evidence, archive)
	propagateThroughRelationships(evidence, archive)
	markPlaceAncestors(evidence.Places, archive.Places)

	return result
}

// propagateThroughEvents covers relationships through their start and end
// events, then the participants and places of every covered event.
func propagateThroughEvents(evidence EntityCoverage, archive *GLXFile) {
	// A relationship is evidenced through the event that starts or ends it
	for id, rel := range archive.Relationships {
		if rel == nil || evidence.Relationships[id] {
			continue
		}
		if (rel.StartEvent != "" && evidence.Events[rel.StartEvent]) ||
			(rel.EndEvent != "" && evidence.Events[rel.EndEvent]) {
			evidence.Relationships[id] = true
		}
	}

	for id := range evidence.Events {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			if archive.Persons[p.Person] != nil {
				evidence.Persons[p.Person] = true
			}
		}
		if archive.Places[event.PlaceID] != nil {
			evidence.Places[event.PlaceID] = true
		}
	}
}

// propagateThroughRelationships covers the participants of every covered
// relationship.
func propagateThroughRelationships(evidence EntityCoverage, archive *GLXFile) {
	for id := range evidence.Relationships {
		rel := archive.Relationships[id]
		if rel == nil {
			continue
		}
		for _, p := range rel.Participants {
			if archive.Persons[p.Person] != nil {
				evidence.Persons[p.Person] = true
			}
		}
	}
}

// markSubject records an assertion's subject pointer.
func markSubject(coverage EntityCoverage, subject EntityRef, archive *GLXFile) {
	switch {
	case archive.Persons[subject.Person] != nil:
		coverage.Persons[subject.Person] = true
	case archive.Events[subject.Event] != nil:
		coverage.Events[subject.Event] = true
	case archive.Relationships[subject.Relationship] != nil:
		coverage.Relationships[subject.Relationship] = true
	case archive.Places[subject.Place] != nil:
		coverage.Places[subject.Place] = true
	}
}

// markAssertionValue records the entity an assertion's value names, when the
// asserted property is a place or person reference and the value resolves to
// an entity in the archive.
func markAssertionValue(coverage EntityCoverage, a *Assertion, archive *GLXFile) {
	if a.Value == "" || a.Property == "" {
		return
	}

	refType := ""
	if def := ConflictProperty(archive, a.Subject, a.Property); def != nil {
		refType = def.ReferenceType
	}

	switch refType {
	case EntityTypePlaces.String():
		if archive.Places[a.Value] != nil {
			coverage.Places[a.Value] = true
		}
	case EntityTypePersons.String():
		if archive.Persons[a.Value] != nil {
			coverage.Persons[a.Value] = true
		}
	}
}

// markPlaceAncestors adds every ancestor of each covered place, following
// parent edges of any period (#225) and guarding against parent cycles.
func markPlaceAncestors(covered map[string]bool, places map[string]*Place) {
	seeds := make([]string, 0, len(covered))
	for id := range covered {
		seeds = append(seeds, id)
	}

	for _, id := range seeds {
		visited := map[string]bool{id: true}
		queue := []string{id}
		for len(queue) > 0 {
			place := places[queue[0]]
			queue = queue[1:]
			for _, parent := range place.ParentIDs() {
				if visited[parent] {
					continue
				}
				visited[parent] = true
				if places[parent] == nil {
					continue
				}
				covered[parent] = true
				queue = append(queue, parent)
			}
		}
	}
}
