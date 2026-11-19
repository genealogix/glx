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
	"testing"

	"github.com/stretchr/testify/assert"
)

func pcRel(relType, parent, child string) *Relationship {
	return &Relationship{
		Type: relType,
		Participants: []Participant{
			{Person: parent, Role: ParticipantRoleParent},
			{Person: child, Role: ParticipantRoleChild},
		},
	}
}

func marriageRel(start, end string) *Relationship {
	return &Relationship{
		Type:       RelationshipTypeMarriage,
		StartEvent: start,
		EndEvent:   end,
		Participants: []Participant{
			{Person: "p-a", Role: ParticipantRoleSpouse},
			{Person: "p-b", Role: ParticipantRoleSpouse},
		},
	}
}

func TestDuplicateRelationships(t *testing.T) {
	reordered := &Relationship{
		Type: "Parent_Child",
		Participants: []Participant{
			{Person: "p-c", Role: "Child"},
			{Person: "p-f", Role: "parent"},
		},
	}
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1":       pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-2":       pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-3":       reordered,
		"rel-bio":     pcRel(RelationshipTypeBiologicalParentChild, "p-f", "p-c"),
		"rel-swapped": pcRel(RelationshipTypeParentChild, "p-c", "p-f"),
		"rel-other":   pcRel(RelationshipTypeParentChild, "p-m", "p-c"),
	}}

	assert.Equal(t, map[string][]string{"rel-2": {"rel-1"}, "rel-3": {"rel-1"}}, DuplicateRelationships(archive))
}

func TestDuplicateRelationships_DistinctStartEvents(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-m1": marriageRel("event-m1", ""),
		"rel-m2": marriageRel("event-m2", ""),
	}}
	assert.Empty(t, DuplicateRelationships(archive), "two marriages with different start events are a remarriage")

	archive.Relationships["rel-m3"] = marriageRel("event-m1", "")
	assert.Equal(t, map[string][]string{"rel-m3": {"rel-m1"}}, DuplicateRelationships(archive))
}

func TestDuplicateRelationships_DistinctEndEvents(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-m1": marriageRel("", "event-divorce"),
		"rel-m2": marriageRel("", "event-death"),
	}}

	assert.Empty(t, DuplicateRelationships(archive), "two unions that ended differently are two relationships")
}

// TestDuplicateRelationships_UndatedBesideRemarriage checks that one undated
// marriage beside two dated ones is reported as matching both, and that the
// dated ones are not reported as copies of it (#1506 review).
func TestDuplicateRelationships_UndatedBesideRemarriage(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-a":  marriageRel("", ""),
		"rel-m1": marriageRel("event-m1", ""),
		"rel-m2": marriageRel("event-m2", ""),
	}}

	assert.Equal(t, map[string][]string{"rel-a": {"rel-m1", "rel-m2"}}, DuplicateRelationships(archive))

	result := &ValidationResult{}
	archive.validateDuplicateRelationships(result)
	if assert.Len(t, result.Warnings, 1) {
		assert.Equal(t, "rel-a", result.Warnings[0].SourceID)
		assert.Contains(t, result.Warnings[0].Message, "relationships[rel-m1] or relationships[rel-m2]")
	}
}

func TestDuplicateRelationships_DisprovenLeftOut(t *testing.T) {
	archive := &GLXFile{
		Relationships: map[string]*Relationship{
			"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
			"rel-2": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		},
		Assertions: map[string]*Assertion{
			"a-1": {Subject: EntityRef{Relationship: "rel-1"}, Status: "disproven"},
		},
	}

	assert.Empty(t, DuplicateRelationships(archive))

	delete(archive.Relationships, "rel-2")
	assert.Empty(t, FindDuplicateRelationship(archive, pcRel(RelationshipTypeParentChild, "p-f", "p-c"), "rel-new"),
		"a disproven alternative does not block recording the link")
}

func TestFindDuplicateRelationship(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
	}}
	candidate := pcRel(RelationshipTypeParentChild, "p-f", "p-c")

	assert.Equal(t, []string{"rel-1"}, FindDuplicateRelationship(archive, candidate, "rel-new"))
	assert.Equal(t, []string{"rel-1"}, FindDuplicateRelationship(archive, candidate, "rel-0"),
		"a candidate whose ID sorts first still finds the existing copy")
	assert.Empty(t, FindDuplicateRelationship(archive, candidate, "rel-1"), "the entity being replaced is not its own duplicate")
	assert.Empty(t, FindDuplicateRelationship(archive, pcRel(RelationshipTypeParentChild, "p-m", "p-c"), ""))
}

func TestFindDuplicateRelationship_Remarriage(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-a":  marriageRel("", ""),
		"rel-m1": marriageRel("event-m1", ""),
	}}

	assert.Empty(t, FindDuplicateRelationship(archive, marriageRel("event-m2", ""), "rel-m2"),
		"a second dated marriage is not a copy of the undated one that matches the first")
	// A copy of the first marriage matches both it and the undated one, which
	// agree with each other; the earliest by ID is named.
	assert.Equal(t, []string{"rel-a"}, FindDuplicateRelationship(archive, marriageRel("event-m1", ""), "rel-z"))
}

func TestRedundantParentChildLinks(t *testing.T) {
	twoParents := &Relationship{
		Type: RelationshipTypeParentChild,
		Participants: []Participant{
			{Person: "p-f", Role: ParticipantRoleParent},
			{Person: "p-m", Role: ParticipantRoleParent},
			{Person: "p-c", Role: ParticipantRoleChild},
		},
	}
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-family":  twoParents,
		"rel-father":  pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-bio":     pcRel(RelationshipTypeBiologicalParentChild, "p-f", "p-d"),
		"rel-generic": pcRel(RelationshipTypeParentChild, "p-f", "p-d"),
		"rel-adopt":   pcRel(RelationshipTypeAdoptiveParentChild, "p-m", "p-d"),
		"rel-bio-m":   pcRel(RelationshipTypeBiologicalParentChild, "p-m", "p-d"),
	}}

	assert.Equal(t, map[string][]RecordedParentChildLink{
		"rel-father":  {{ParentID: "p-f", ChildID: "p-c", RecordedIn: "rel-family"}},
		"rel-generic": {{ParentID: "p-f", ChildID: "p-d", RecordedIn: "rel-bio"}},
	}, RedundantParentChildLinks(archive), "biological and adoptive links to one parent are different facts")
}

func TestRedundantParentChildLinks_SkipsWholeDuplicates(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-2": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
	}}

	assert.Empty(t, RedundantParentChildLinks(archive), "a whole duplicate is reported once, by DuplicateRelationships")
}

func TestFindRecordedParentChildLinks(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-family": {
			Type: RelationshipTypeParentChild,
			Participants: []Participant{
				{Person: "p-f", Role: ParticipantRoleParent},
				{Person: "p-m", Role: ParticipantRoleParent},
				{Person: "p-c", Role: ParticipantRoleChild},
			},
		},
		"rel-generic": pcRel(RelationshipTypeParentChild, "p-f", "p-d"),
	}}

	recorded, total := FindRecordedParentChildLinks(archive, pcRel(RelationshipTypeParentChild, "p-f", "p-c"), "rel-new")
	assert.Equal(t, 1, total)
	assert.Equal(t, []RecordedParentChildLink{{ParentID: "p-f", ChildID: "p-c", RecordedIn: "rel-family"}}, recorded)

	recorded, _ = FindRecordedParentChildLinks(archive, pcRel(RelationshipTypeBiologicalParentChild, "p-f", "p-d"), "rel-new")
	assert.Empty(t, recorded, "a specific type is not covered by the generic one; the generic one is the redundant copy")

	recorded, total = FindRecordedParentChildLinks(archive, marriageRel("", ""), "rel-new")
	assert.Empty(t, recorded)
	assert.Zero(t, total)
}

func TestValidateDuplicateRelationships(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-2": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-3": pcRel(RelationshipTypeBiologicalParentChild, "p-f", "p-d"),
		"rel-4": pcRel(RelationshipTypeParentChild, "p-f", "p-d"),
	}}
	result := &ValidationResult{}
	archive.validateDuplicateRelationships(result)

	got := map[string]string{}
	for _, w := range result.Warnings {
		got[w.SourceID] = w.Message
	}
	if assert.Len(t, got, 2) {
		assert.Contains(t, got["rel-2"], "duplicates relationships[rel-1]")
		assert.Contains(t, got["rel-4"], "the link from parent p-f to child p-d is already recorded by relationships[rel-3]")
	}
}
