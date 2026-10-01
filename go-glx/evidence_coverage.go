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
// Direct counts only an assertion's subject pointer. Evidence follows the
// evidence chain the GLX model expects researchers to build, where facts are
// usually asserted on events and relationships rather than on persons and
// places:
//
//   - an event or relationship is covered when an assertion targets it; a
//     relationship is also covered when its start_event or end_event is
//   - a person is covered when an assertion targets them, names them as
//     participant.person, or targets an event or relationship they
//     participate in (any role: a witness on a cited marriage record is
//     evidenced by that record), or carries them as the value of a
//     person-reference property
//   - a place is covered when an assertion targets it, carries it as the
//     value of a place-reference property (an event's `place`, a person's
//     `residence`, ...), or it is the place of a covered event; and then every
//     place on a covered place's parent chain is covered too
//
// Parent-chain places count because jurisdictions (county, state, country)
// are gazetteer scaffolding that locates an evidenced place; no record is
// ever asserted against "United States" itself, and reporting it as unsourced
// would be noise.
//
// Any assertion counts, sourced or not; assertions without citations are a
// separate finding (glx validate --report lists them).
type AssertionCoverage struct {
	Direct   EntityCoverage
	Evidence EntityCoverage
}

// ComputeAssertionCoverage computes the direct and evidence coverage of an
// archive's assertions. See AssertionCoverage for the rules.
func ComputeAssertionCoverage(archive *GLXFile) AssertionCoverage {
	result := AssertionCoverage{Direct: newEntityCoverage(), Evidence: newEntityCoverage()}
	if archive == nil {
		return result
	}

	direct, evidence := result.Direct, result.Evidence

	for _, a := range archive.Assertions {
		if a == nil {
			continue
		}
		markSubject(direct, a.Subject)
		markSubject(evidence, a.Subject)

		if a.Participant != nil && a.Participant.Person != "" {
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
			if p.Person != "" {
				evidence.Persons[p.Person] = true
			}
		}
		if event.PlaceID != "" {
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
			if p.Person != "" {
				evidence.Persons[p.Person] = true
			}
		}
	}
}

// markSubject records an assertion's subject pointer.
func markSubject(coverage EntityCoverage, subject EntityRef) {
	switch {
	case subject.Person != "":
		coverage.Persons[subject.Person] = true
	case subject.Event != "":
		coverage.Events[subject.Event] = true
	case subject.Relationship != "":
		coverage.Relationships[subject.Relationship] = true
	case subject.Place != "":
		coverage.Places[subject.Place] = true
	}
}

// eventPlaceProperty is the property an assertion uses to assert an event's
// structural place field.
const eventPlaceProperty = "place"

// markAssertionValue records the entity an assertion's value names, when the
// asserted property is a place or person reference and the value resolves to
// an entity in the archive.
func markAssertionValue(coverage EntityCoverage, a *Assertion, archive *GLXFile) {
	if a.Value == "" || a.Property == "" {
		return
	}

	refType := ""
	if def := assertionPropertyDefinition(a, archive); def != nil {
		refType = def.ReferenceType
	}
	if refType == "" && a.Subject.Event != "" && a.Property == eventPlaceProperty {
		refType = EntityTypePlaces.String()
	}

	switch refType {
	case EntityTypePlaces.String():
		if _, ok := archive.Places[a.Value]; ok {
			coverage.Places[a.Value] = true
		}
	case EntityTypePersons.String():
		if _, ok := archive.Persons[a.Value]; ok {
			coverage.Persons[a.Value] = true
		}
	}
}

// assertionPropertyDefinition returns the property definition for an
// assertion's property on its subject's entity type, or nil.
func assertionPropertyDefinition(a *Assertion, archive *GLXFile) *PropertyDefinition {
	var defs map[string]*PropertyDefinition
	switch {
	case a.Subject.Person != "":
		defs = archive.PersonProperties
	case a.Subject.Event != "":
		defs = archive.EventProperties
	case a.Subject.Relationship != "":
		defs = archive.RelationshipProperties
	case a.Subject.Place != "":
		defs = archive.PlaceProperties
	}

	return defs[a.Property]
}

// markPlaceAncestors adds every ancestor of each covered place, guarding
// against parent cycles.
func markPlaceAncestors(covered map[string]bool, places map[string]*Place) {
	seeds := make([]string, 0, len(covered))
	for id := range covered {
		seeds = append(seeds, id)
	}

	for _, id := range seeds {
		visited := map[string]bool{id: true}
		place := places[id]
		for place != nil && place.ParentID != "" && !visited[place.ParentID] {
			parent := place.ParentID
			visited[parent] = true
			if _, ok := places[parent]; !ok {
				break
			}
			covered[parent] = true
			place = places[parent]
		}
	}
}
