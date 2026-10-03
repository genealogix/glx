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
	"github.com/stretchr/testify/require"
)

func TestStandingFromAssertions(t *testing.T) {
	a := func(confidence, status string) *Assertion {
		return &Assertion{Confidence: confidence, Status: status}
	}
	tests := []struct {
		name       string
		assertions []*Assertion
		want       RelationshipStanding
	}{
		{"no assertions", nil, RelationshipStandingAccepted},
		{"high confidence", []*Assertion{a("high", "")}, RelationshipStandingAccepted},
		{"medium confidence", []*Assertion{a("medium", "")}, RelationshipStandingAccepted},
		{"low confidence", []*Assertion{a("low", "")}, RelationshipStandingHypothetical},
		{"legacy tentative", []*Assertion{a("tentative", "")}, RelationshipStandingHypothetical},
		{"high but speculative", []*Assertion{a("high", "speculative")}, RelationshipStandingHypothetical},
		{"unresearched", []*Assertion{a("", "unresearched")}, RelationshipStandingHypothetical},
		{"disputed status", []*Assertion{a("medium", "disputed")}, RelationshipStandingHypothetical},
		{"low but proven", []*Assertion{a("low", "proven")}, RelationshipStandingAccepted},
		{"case-insensitive", []*Assertion{a("LOW", "")}, RelationshipStandingHypothetical},
		{"only disproven", []*Assertion{a("high", "disproven")}, RelationshipStandingDisproven},
		{"all disproven", []*Assertion{a("high", "disproven"), a("medium", "Disproven")}, RelationshipStandingDisproven},
		{"disproven plus high survivor", []*Assertion{a("high", "disproven"), a("high", "")}, RelationshipStandingAccepted},
		{"disproven plus low survivor", []*Assertion{a("high", "disproven"), a("low", "")}, RelationshipStandingHypothetical},
		{"any hypothetical survivor", []*Assertion{a("high", ""), a("low", "")}, RelationshipStandingHypothetical},
		{"proven survivor wins", []*Assertion{a("low", "speculative"), a("high", "proven")}, RelationshipStandingAccepted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StandingFromAssertions(tt.assertions))
		})
	}
}

// standingArchive has a two-parent relationship in which one parent is
// disproven by a participant assertion, a relationship disproven as a whole,
// one with only a disproven property assertion, and a step relationship.
func standingArchive() *GLXFile {
	return &GLXFile{
		Relationships: map[string]*Relationship{
			"rel-two": {Type: RelationshipTypeParentChild, Participants: []Participant{
				{Person: "dad", Role: ParticipantRoleParent},
				{Person: "mum", Role: ParticipantRoleParent},
				{Person: "kid", Role: ParticipantRoleChild},
			}},
			"rel-disproven": {Type: RelationshipTypeParentChild, Participants: []Participant{
				{Person: "other", Role: ParticipantRoleParent},
				{Person: "kid", Role: ParticipantRoleChild},
			}},
			"rel-property": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{
				{Person: "gran", Role: ParticipantRoleParent},
				{Person: "mum", Role: ParticipantRoleChild},
			}},
			"rel-step": {Type: RelationshipTypeStepParent, Participants: []Participant{
				{Person: "step", Role: ParticipantRoleParent},
				{Person: "kid", Role: ParticipantRoleChild},
			}},
			"rel-step-roles": {Type: RelationshipTypeParentChild, Participants: []Participant{
				{Person: "step2", Role: "step_parent"},
				{Person: "kid", Role: "step_child"},
			}},
			"rel-marriage": {Type: RelationshipTypeMarriage, Participants: []Participant{
				{Person: "dad", Role: ParticipantRoleSpouse},
				{Person: "mum", Role: ParticipantRoleSpouse},
			}},
		},
		Assertions: map[string]*Assertion{
			"a-dad-disproven": {
				Subject:     EntityRef{Relationship: "rel-two"},
				Participant: &Participant{Person: "dad", Role: ParticipantRoleParent},
				Status:      "disproven",
			},
			"a-rel-disproven": {Subject: EntityRef{Relationship: "rel-disproven"}, Status: "disproven"},
			"a-property-disproven": {
				Subject: EntityRef{Relationship: "rel-property"}, Property: "start_date", Value: "1800", Status: "disproven",
			},
			"a-step-low": {Subject: EntityRef{Relationship: "rel-step"}, Confidence: "low"},
		},
	}
}

func TestRelationshipStandingIndex(t *testing.T) {
	idx := NewRelationshipStandingIndex(standingArchive())

	// A participant assertion disproves only that participant's link.
	assert.Equal(t, RelationshipStandingDisproven, idx.Link("rel-two", "dad", "kid"))
	assert.Equal(t, RelationshipStandingAccepted, idx.Link("rel-two", "mum", "kid"))
	assert.Equal(t, RelationshipStandingDisproven, idx.Relationship("rel-two"))

	assert.Equal(t, RelationshipStandingDisproven, idx.Relationship("rel-disproven"))
	// A property assertion says nothing about whether the link exists.
	assert.Equal(t, RelationshipStandingAccepted, idx.Relationship("rel-property"))
	assert.Equal(t, RelationshipStandingHypothetical, idx.Link("rel-step", "step", "kid"))
	assert.Equal(t, RelationshipStandingAccepted, idx.Relationship("rel-unknown"))

	var nilIdx *RelationshipStandingIndex
	assert.Equal(t, RelationshipStandingAccepted, nilIdx.Link("rel-two", "dad"))
	assert.Equal(t, RelationshipStandingAccepted, NewRelationshipStandingIndex(nil).Relationship("x"))
}

func TestParentChildLinks(t *testing.T) {
	links := ParentChildLinks(standingArchive(), nil)

	type key struct{ rel, parent, child string }
	got := map[key]ParentChildLink{}
	for _, l := range links {
		got[key{l.RelationshipID, l.ParentID, l.ChildID}] = l
	}
	require.Len(t, got, 6, "marriage is not a parent-child link")

	assert.Equal(t, RelationshipStandingDisproven, got[key{"rel-two", "dad", "kid"}].Standing)
	assert.Equal(t, RelationshipStandingAccepted, got[key{"rel-two", "mum", "kid"}].Standing)
	assert.False(t, got[key{"rel-two", "mum", "kid"}].Step)
	assert.True(t, got[key{"rel-step", "step", "kid"}].Step, "step_parent type")
	assert.Equal(t, RelationshipStandingHypothetical, got[key{"rel-step", "step", "kid"}].Standing)
	assert.True(t, got[key{"rel-step-roles", "step2", "kid"}].Step, "step_parent/step_child roles")

	// Deterministic relationship-ID order.
	assert.Equal(t, "rel-disproven", links[0].RelationshipID)
}

func TestParentChildRoleHelpers(t *testing.T) {
	for _, role := range []string{"parent", "Adoptive_Parent", "step_parent", "foster_parent"} {
		assert.True(t, IsParentSideRole(role), role)
		assert.False(t, IsChildSideRole(role), role)
	}
	for _, role := range []string{"child", "adopted_child", "step_child", "foster_child"} {
		assert.True(t, IsChildSideRole(role), role)
	}
	assert.False(t, IsParentSideRole("guardian"))
	assert.True(t, IsStepRole("STEP_CHILD"))
	assert.False(t, IsStepRole("child"))
	assert.True(t, IsParentChildRelationshipType("Step_Parent"))
	assert.False(t, IsParentChildRelationshipType(RelationshipTypeGuardian))
}
