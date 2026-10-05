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
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// MergeInterpretationChange reports an existing derived interpretation that
// changed. Before/After are deterministic JSON values; Aspect identifies the
// report (standing, link/<participants>, coverage, evidence/<property>, or
// proof/<question>). Link reports compare individual participants and pairs.
// Proof/coverage use default geographic options and the merge's comparison
// options. Proof reports cover persons reached by removed claims/coverage.
type MergeInterpretationChange struct {
	EntityType EntityType
	ID         string
	Aspect     string
	Before     string
	After      string
}

func reportPersonMerge(before, after *GLXFile, keepID, dropID string, comparison ComparisonOptions, result *MergePersonsResult) error {
	result.Changes = DiffArchives(before, after, "").Changes
	for i := range result.Changes {
		change := &result.Changes[i]
		if change.Kind != ChangeRemoved {
			continue
		}
		value := mergeEntityCollection(before, change.EntityType).MapIndex(reflect.ValueOf(change.ID))
		payload, err := yaml.Marshal(value.Interface())
		if err != nil {
			return fmt.Errorf("merge removal report: %w", err)
		}
		change.Fields = []FieldChange{{Path: "entity", OldValue: string(payload)}}
	}
	slices.SortFunc(result.ReferenceChanges, func(a, b MergeReferenceChange) int {
		return strings.Compare(string(a.EntityType)+"/"+a.ID+"/"+a.Path+"/"+a.Action,
			string(b.EntityType)+"/"+b.ID+"/"+b.Path+"/"+b.Action)
	})

	return reportMergeInterpretations(before, after, keepID, dropID, comparison, result)
}

func addMergeInterpretation(result *MergePersonsResult, kind EntityType, id, aspect string, oldValue, newValue any) error {
	oldJSON, err := json.Marshal(oldValue)
	if err != nil {
		return fmt.Errorf("merge interpretation report: %w", err)
	}
	newJSON, err := json.Marshal(newValue)
	if err != nil {
		return fmt.Errorf("merge interpretation report: %w", err)
	}
	if !bytes.Equal(oldJSON, newJSON) {
		result.InterpretationChanges = append(result.InterpretationChanges,
			MergeInterpretationChange{kind, id, aspect, string(oldJSON), string(newJSON)})
	}

	return nil
}

func reportMergeInterpretations(before, after *GLXFile, keepID, dropID string, comparison ComparisonOptions, result *MergePersonsResult) error {
	oldStanding, newStanding := NewRelationshipStandingIndex(before), NewRelationshipStandingIndex(after)
	changedRelationships := mergeChangedRelationships(before, after, result.Changes)
	forMergeEntities(after, after, func(entity mergeEntity) {
		if entity.kind != EntityTypeRelationships || before.Relationships[entity.id] == nil || !changedRelationships[entity.id] {
			return
		}
		// Standing is a string, so JSON encoding cannot fail.
		_ = addMergeInterpretation(result, entity.kind, entity.id, "standing", oldStanding.Relationship(entity.id), newStanding.Relationship(entity.id))
		reportMergeLinkStandings(before.Relationships[entity.id], entity.id, keepID, dropID, oldStanding, newStanding, result)
	})
	oldCoverage, newCoverage := ComputeAssertionCoverage(before), ComputeAssertionCoverage(after)
	affected := map[string]bool{}
	forMergeEntities(after, after, func(entity mergeEntity) {
		if entity.kind != EntityTypePersons && entity.kind != EntityTypeEvents && entity.kind != EntityTypeRelationships && entity.kind != EntityTypePlaces {
			return
		}
		for _, coverage := range []struct {
			name      string
			old, next EntityCoverage
		}{
			{"direct-coverage", oldCoverage.Direct, newCoverage.Direct}, {"evidence-coverage", oldCoverage.Evidence, newCoverage.Evidence},
		} {
			oldValue, nextValue := mergeCoverageSet(coverage.old, entity.kind)[entity.id], mergeCoverageSet(coverage.next, entity.kind)[entity.id]
			_ = addMergeInterpretation(result, entity.kind, entity.id, coverage.name, oldValue, nextValue)
			if oldValue != nextValue && entity.kind == EntityTypePersons {
				affected[entity.id] = true
			}
		}
	})
	if len(result.RemovedAssertions) == 0 {
		return nil
	}
	affected[keepID] = true
	if err := reportMergeEvidence(before, after, keepID, dropID, comparison, result, affected); err != nil {
		return err
	}
	if err := reportMergeProofs(before, after, dropID, comparison, result, affected); err != nil {
		return err
	}
	slices.SortFunc(result.InterpretationChanges, func(a, b MergeInterpretationChange) int {
		return strings.Compare(string(a.EntityType)+"/"+a.ID+"/"+a.Aspect, string(b.EntityType)+"/"+b.ID+"/"+b.Aspect)
	})

	return nil
}

func reportMergeEvidence(before, after *GLXFile, keepID, dropID string, comparison ComparisonOptions, result *MergePersonsResult, affected map[string]bool) error {
	seen := make(map[string]bool)
	for _, id := range result.RemovedAssertions {
		assertion := before.Assertions[id]
		collectMergeAffectedPeople(before, assertion, affected)
		subject := assertion.Subject
		if subject.Person == dropID {
			subject.Person = keepID
		}
		key := string(subject.Type()) + "/" + subject.ID() + "/" + assertion.Property
		if seen[key] || researchSubjectExists(after, subject) != nil {
			continue
		}
		seen[key] = true
		oldEvidence, err := BuildEvidenceReport(before, assertion.Subject, assertion.Property, comparison)
		if err != nil {
			return err
		}
		newEvidence, err := BuildEvidenceReport(after, subject, assertion.Property, comparison)
		if err != nil {
			return err
		}
		if err := addMergeInterpretation(result, subject.Type(), subject.ID(), "evidence/"+assertion.Property, oldEvidence, newEvidence); err != nil {
			return err
		}
	}

	return nil
}

func reportMergeProofs(before, after *GLXFile, dropID string, comparison ComparisonOptions, result *MergePersonsResult, affected map[string]bool) error {
	delete(affected, dropID)
	ids := make([]string, 0, len(affected))
	for id := range affected {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if before.Persons[id] == nil || after.Persons[id] == nil {
			continue
		}
		for _, question := range ProofQuestions() {
			oldProof, err := BuildProof(before, id, question, ProofOptions{Comparison: comparison})
			if err != nil {
				return err
			}
			newProof, err := BuildProof(after, id, question, ProofOptions{Comparison: comparison})
			if err != nil {
				return err
			}
			if err := addMergeInterpretation(result, EntityTypePersons, id, "proof/"+question, oldProof, newProof); err != nil {
				return err
			}
		}
		oldChecklist, err := BuildCoverage(before, id, CoverageOptions{})
		if err != nil {
			return err
		}
		newChecklist, err := BuildCoverage(after, id, CoverageOptions{})
		if err != nil {
			return err
		}
		if err := addMergeInterpretation(result, EntityTypePersons, id, "coverage", oldChecklist, newChecklist); err != nil {
			return err
		}
	}

	return nil
}

func mergeChangedRelationships(before, after *GLXFile, changes []EntityChange) map[string]bool {
	changed := make(map[string]bool)
	for _, change := range changes {
		if change.EntityType == EntityTypeRelationships {
			changed[change.ID] = true
		}
		if change.EntityType == EntityTypeAssertions {
			for _, archive := range []*GLXFile{before, after} {
				if assertion := archive.Assertions[change.ID]; assertion != nil && assertion.Subject.Relationship != "" {
					changed[assertion.Subject.Relationship] = true
				}
			}
		}
	}

	return changed
}

func reportMergeLinkStandings(relationship *Relationship, id, keepID, dropID string, before, after *RelationshipStandingIndex, result *MergePersonsResult) {
	people := make([]string, 0, len(relationship.Participants))
	for _, participant := range relationship.Participants {
		people = append(people, participant.Person)
	}
	slices.Sort(people)
	people = slices.Compact(people)
	compare := func(oldPeople []string) {
		newPeople := slices.Clone(oldPeople)
		for i, person := range newPeople {
			if person == dropID {
				newPeople[i] = keepID
			}
		}
		slices.Sort(newPeople)
		newPeople = slices.Compact(newPeople)
		aspect := "link/" + strings.Join(oldPeople, ",")
		if !slices.Equal(oldPeople, newPeople) {
			aspect += "→" + strings.Join(newPeople, ",")
		}
		_ = addMergeInterpretation(result, EntityTypeRelationships, id, aspect, before.Link(id, oldPeople...), after.Link(id, newPeople...))
	}
	for i, person := range people {
		compare([]string{person})
		for _, other := range people[i+1:] {
			compare([]string{person, other})
		}
	}
}

func mergeCoverageSet(coverage EntityCoverage, kind EntityType) map[string]bool {
	switch kind {
	case EntityTypePersons:

		return coverage.Persons
	case EntityTypeEvents:

		return coverage.Events
	case EntityTypeRelationships:

		return coverage.Relationships
	case EntityTypePlaces:

		return coverage.Places
	default:

		return nil
	}
}

func collectMergeAffectedPeople(archive *GLXFile, assertion *Assertion, people map[string]bool) {
	if assertion.Subject.Person != "" {
		people[assertion.Subject.Person] = true
	}
	if assertion.Participant != nil {
		people[assertion.Participant.Person] = true
	}
	var participants []Participant
	if event := archive.Events[assertion.Subject.Event]; event != nil {
		participants = event.Participants
	}
	if relationship := archive.Relationships[assertion.Subject.Relationship]; relationship != nil {
		participants = relationship.Participants
	}
	for _, participant := range participants {
		people[participant.Person] = true
	}
}
