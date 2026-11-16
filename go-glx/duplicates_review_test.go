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

func TestDuplicatesReview_MultipleEventRolesAreOrderIndependent(t *testing.T) {
	for _, participants := range [][]Participant{
		{{Person: "a", Role: "witness"}, {Person: "a", Role: "parent"}, {Person: "b", Role: "witness"}},
		{{Person: "a", Role: "parent"}, {Person: "a", Role: "witness"}, {Person: "b", Role: "witness"}},
		{{Person: "a", Role: "witness"}, {Person: "b", Role: "witness"}, {Person: "b", Role: "parent"}},
	} {
		archive := &GLXFile{Events: map[string]*Event{"e": {Type: "baptism", Participants: participants}}}
		index := buildDuplicateIndex(archive)
		score, detail, hasData := scoreSharedEvents("a", "b", index)
		assert.InDelta(t, 1, score, 1e-12)
		assert.Equal(t, "1 shared", detail)
		assert.True(t, hasData)
		assert.Len(t, index.personEvents["a"], 1)
		assert.Len(t, index.personEvents["b"], 1)
	}
}

func TestDuplicatesReview_VitalPrincipalAliases(t *testing.T) {
	for _, eventType := range []string{EventTypeBirth, EventTypeBaptism, EventTypeChristening, EventTypeDeath, EventTypeBurial, EventTypeCremation} {
		role := ParticipantRoleChild
		if isDeathEventType(eventType) {
			role = participationRoleDeceased
		}
		archive := &GLXFile{Events: map[string]*Event{"vital": {Type: eventType, Date: "1700", Participants: []Participant{{Person: "p", Role: role}, {Person: "w", Role: "witness"}}}}}
		index := buildDuplicateIndex(archive)
		if isBirthEventType(eventType) {
			assert.Same(t, archive.Events["vital"], index.personBirthEvent["p"], eventType)
			assert.Nil(t, index.personBirthEvent["w"])
		} else {
			assert.Same(t, archive.Events["vital"], index.personDeathEvent["p"], eventType)
			assert.Nil(t, index.personDeathEvent["w"])
		}
	}
}

func TestDuplicatesReview_CrossArchiveRetainsAssertions(t *testing.T) {
	dest := &GLXFile{Persons: map[string]*Person{"a": duplicateTestPerson("Johann", "Roth", "male")}, Relationships: map[string]*Relationship{
		"ra": {Type: "parent_child", Participants: []Participant{{Person: "a", Role: "parent"}, {Person: "child", Role: "child"}}},
	}, Assertions: map[string]*Assertion{"dest-assertion": {Subject: EntityRef{Relationship: "ra"}, Status: "proven"}}}
	src := &GLXFile{Persons: map[string]*Person{"b": duplicateTestPerson("Johann", "Roth", "male")}, Relationships: map[string]*Relationship{
		"rb": {Type: "parent_child", Participants: []Participant{{Person: "b", Role: "parent"}, {Person: "child", Role: "child"}}},
	}, Assertions: map[string]*Assertion{"src-assertion": {Subject: EntityRef{Relationship: "rb"}, Status: "disproven"}}}
	combined := buildCombinedView(dest, src)
	assert.Same(t, dest.Assertions["dest-assertion"], combined.Assertions["dest-assertion"])
	assert.Same(t, src.Assertions["src-assertion"], combined.Assertions["src-assertion"])
	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	for _, signal := range result.Pairs[0].Signals {
		if signal.Name == "Shared relationships" {
			assert.Zero(t, signal.Score)
		}
	}
	// Maps are owned by the view; a duplicate assertion ID keeps destination precedence.
	src.Assertions["dest-assertion"] = &Assertion{Status: "disproven"}
	combined = buildCombinedView(dest, src)
	assert.Same(t, dest.Assertions["dest-assertion"], combined.Assertions["dest-assertion"])
	assert.Len(t, dest.Assertions, 1)
}

func TestDuplicatesReview_CrossArchiveRetainsPresenceVocabulary(t *testing.T) {
	no := false
	dest := &GLXFile{Persons: map[string]*Person{"a": duplicateTestPerson("Johann", "Roth", "male")}, Events: map[string]*Event{
		"death": {Type: "death", Date: "1700", Participants: []Participant{{Person: "a", Role: "principal"}}},
	}}
	src := &GLXFile{Persons: map[string]*Person{"b": duplicateTestPerson("Johann", "Roth", "male")}, Events: map[string]*Event{
		"reference": {Type: "baptism", Date: "1720", Participants: []Participant{{Person: "b", Role: "named_relative"}}},
	}, ParticipantRoles: map[string]*VocabularyEntry{"named_relative": {ImpliesPresence: &no}}}
	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Len(t, result.Pairs, 1, "a reference-only role cannot rule out the identity")
	assert.Nil(t, dest.ParticipantRoles)
	yes := true
	dest.ParticipantRoles = map[string]*VocabularyEntry{"named_relative": {ImpliesPresence: &yes}}
	combined := buildCombinedView(dest, src)
	assert.Same(t, dest.ParticipantRoles["named_relative"], combined.ParticipantRoles["named_relative"], "destination definition wins a vocabulary collision")
}

func TestDuplicatesReview_ParentReferencesAndMultipleActivityRoles(t *testing.T) {
	for _, roles := range [][]string{{"parent"}, {"adoptive_parent"}, {"parent", "witness"}, {"witness", "parent"}} {
		archive := &GLXFile{Events: map[string]*Event{
			"death":     {Type: "death", Date: "1632", Participants: []Participant{{Person: "a", Role: "principal"}}},
			"reference": {Type: "baptism", Date: "1640"},
		}}
		for _, role := range roles {
			archive.Events["reference"].Participants = append(archive.Events["reference"].Participants, Participant{Person: "b", Role: role})
		}
		conflict := duplicateLifetimeConflict("a", "b", archive, buildDuplicateIndex(archive))
		assert.Equal(t, len(roles) > 1, conflict != "", roles)
		yes := true
		archive.ParticipantRoles = map[string]*VocabularyEntry{roles[0]: {ImpliesPresence: &yes}}
		assert.NotEmpty(t, duplicateLifetimeConflict("a", "b", archive, buildDuplicateIndex(archive)), "explicit presence still matters")
	}
}

func TestDuplicatesReview_BaptismDoesNotBoundBirthFromBelow(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{
		"vital":    {Type: "baptism", Date: "1700", Participants: []Participant{{Person: "a", Role: "principal"}}},
		"activity": {Type: "baptism", Date: "1680", Participants: []Participant{{Person: "b", Role: "witness"}}},
	}}
	index := buildDuplicateIndex(archive)
	assert.Empty(t, duplicateLifetimeConflict("a", "b", archive, index), "an adult baptism can follow documented activity")
	assert.Zero(t, index.personBirthYear["a"], "a proxy must not supply a hard parent-age lower bound either")
	archive.Events["vital"].Type = "birth"
	index = buildDuplicateIndex(archive)
	assert.NotEmpty(t, duplicateLifetimeConflict("a", "b", archive, index))
}
