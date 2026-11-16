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

	assert.Equal(t, map[string]string{"rel-2": "rel-1", "rel-3": "rel-1"}, DuplicateRelationships(archive))
}

func TestDuplicateRelationships_DistinctStartEvents(t *testing.T) {
	marriage := func(start string) *Relationship {
		return &Relationship{
			Type:       RelationshipTypeMarriage,
			StartEvent: start,
			Participants: []Participant{
				{Person: "p-a", Role: ParticipantRoleSpouse},
				{Person: "p-b", Role: ParticipantRoleSpouse},
			},
		}
	}
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-m1": marriage("event-m1"),
		"rel-m2": marriage("event-m2"),
		"rel-m3": marriage(""),
	}}

	// Two marriages with different start events are a remarriage; one with
	// no start event repeats the first.
	assert.Equal(t, map[string]string{"rel-m3": "rel-m1"}, DuplicateRelationships(archive))
}

func TestFindDuplicateRelationship(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
	}}
	candidate := pcRel(RelationshipTypeParentChild, "p-f", "p-c")

	assert.Equal(t, "rel-1", FindDuplicateRelationship(archive, candidate, "rel-new"))
	assert.Empty(t, FindDuplicateRelationship(archive, candidate, "rel-1"), "the entity being replaced is not its own duplicate")
	assert.Empty(t, FindDuplicateRelationship(archive, pcRel(RelationshipTypeParentChild, "p-m", "p-c"), ""))
}

func TestValidateDuplicateRelationships(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"rel-1": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
		"rel-2": pcRel(RelationshipTypeParentChild, "p-f", "p-c"),
	}}
	result := &ValidationResult{}
	archive.validateDuplicateRelationships(result)

	if assert.Len(t, result.Warnings, 1) {
		w := result.Warnings[0]
		assert.Equal(t, "rel-2", w.SourceID)
		assert.Contains(t, w.Message, "duplicates relationships[rel-1]")
	}
}
