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

func TestClassifyParticipation(t *testing.T) {
	own := Participation{OwnRecord: true, Present: true}
	present := Participation{Present: true}
	mentioned := Participation{}

	tests := []struct {
		name      string
		eventType string
		role      string
		want      Participation
	}{
		{"principal", EventTypeProbate, ParticipantRolePrincipal, own},
		{"subject", EventTypeBirth, ParticipantRoleSubject, own},
		{"legacy child at birth", EventTypeBirth, ParticipantRoleChild, own},
		{"legacy child at baptism", EventTypeBaptism, ParticipantRoleChild, own},
		{"child at death", EventTypeDeath, ParticipantRoleChild, present},
		{"unset role", EventTypeDeath, "", own},
		{"role case and spacing", EventTypeBirth, " Principal ", own},
		{"bride", EventTypeMarriage, ParticipantRoleBride, own},
		{"groom", EventTypeMarriage, ParticipantRoleGroom, own},
		{"godchild at baptism", EventTypeBaptism, ParticipantRoleGodchild, own},
		{"testator", EventTypeWill, "testator", own},
		{"spouse on marriage", EventTypeMarriage, ParticipantRoleSpouse, own},
		{"spouse on death is not own", EventTypeDeath, ParticipantRoleSpouse, present},
		{"census household head", EventTypeCensus, ParticipantRoleHouseholdHead, own},
		{"census custom household role", EventTypeCensus, "daughter", own},
		{"census counted-not-named role", EventTypeCensus, "household_member", own},
		{"census enumerator", EventTypeCensus, "enumerator", present},
		{"census neighbor", EventTypeCensus, "neighbor", mentioned},
		{"witness", EventTypeProbate, ParticipantRoleWitness, present},
		{"godparent", EventTypeBaptism, ParticipantRoleGodparent, present},
		{"informant", EventTypeDeath, ParticipantRoleInformant, present},
		{"parent at birth", EventTypeBirth, ParticipantRoleParent, present},
		{"custom role defaults to present", EventTypeGeneric, "host", present},
		{"grantor", EventTypeGeneric, "grantor", mentioned},
		{"grantee", EventTypeGeneric, "grantee", mentioned},
		{"legatee", EventTypeProbate, "legatee", mentioned},
		{"heir", EventTypeProbate, "heir", mentioned},
		{"executor", EventTypeProbate, "executor", mentioned},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyParticipation(tt.eventType, tt.role, nil)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, !got.OwnRecord && !got.Present, got.Mentioned())
		})
	}
}

func TestClassifyParticipation_VocabularyOverride(t *testing.T) {
	no, yes := false, true
	roles := map[string]*VocabularyEntry{
		"absent_heir": {Label: "Absent heir", ImpliesPresence: &no},
		"grantor":     {Label: "Grantor", ImpliesPresence: &yes},
		"Visitor":     {Label: "Visitor", ImpliesPresence: &no},
		"witness":     {Label: "Witness"}, // unset keeps the default
	}

	assert.Equal(t, Participation{}, ClassifyParticipation(EventTypeProbate, "absent_heir", roles))
	assert.Equal(t, Participation{Present: true}, ClassifyParticipation(EventTypeGeneric, "grantor", roles),
		"the archive says its grantors were there")
	assert.Equal(t, Participation{}, ClassifyParticipation(EventTypeGeneric, "Visitor", roles),
		"the vocabulary key is matched as declared")
	assert.Equal(t, Participation{Present: true}, ClassifyParticipation(EventTypeMarriage, "witness", roles))

	// The flag changes presence only: a principal stays the owner of the record
	roles[ParticipantRolePrincipal] = &VocabularyEntry{Label: "Principal", ImpliesPresence: &no}
	assert.Equal(t, Participation{OwnRecord: true}, ClassifyParticipation(EventTypeProbate, ParticipantRolePrincipal, roles))
}

func TestEventParticipation(t *testing.T) {
	event := &Event{
		Type: EventTypeProbate,
		Participants: []Participant{
			{Person: "person-caspar", Role: ParticipantRolePrincipal},
			{Person: "person-lewis", Role: "legatee"},
			{Person: "person-mary", Role: "legatee"},
			{Person: "person-mary", Role: ParticipantRoleWitness},
		},
	}

	p, role, ok := EventParticipation(event, "person-caspar", nil)
	assert.True(t, ok)
	assert.Equal(t, ParticipantRolePrincipal, role)
	assert.True(t, p.OwnRecord)

	p, role, ok = EventParticipation(event, "person-lewis", nil)
	assert.True(t, ok)
	assert.Equal(t, "legatee", role)
	assert.True(t, p.Mentioned())

	p, role, ok = EventParticipation(event, "person-mary", nil)
	assert.True(t, ok)
	assert.Equal(t, ParticipantRoleWitness, role, "the most significant role is reported")
	assert.Equal(t, Participation{Present: true}, p)

	_, _, ok = EventParticipation(event, "person-nobody", nil)
	assert.False(t, ok)

	_, _, ok = EventParticipation(nil, "person-caspar", nil)
	assert.False(t, ok)
}

func TestEventParticipationUnionsIndependentAxes(t *testing.T) {
	no := false
	roles := map[string]*VocabularyEntry{"principal": {ImpliesPresence: &no}}
	for _, participants := range [][]Participant{
		{{Person: "p", Role: "principal"}, {Person: "p", Role: "witness"}, {Person: "p", Role: "grantor"}},
		{{Person: "p", Role: "grantor"}, {Person: "p", Role: "witness"}, {Person: "p", Role: "principal"}},
	} {
		participation, role, ok := EventParticipation(&Event{Type: EventTypeProbate, Participants: participants}, "p", roles)
		assert.True(t, ok)
		assert.Equal(t, Participation{OwnRecord: true, Present: true}, participation)
		assert.Equal(t, "principal", role)
	}
}

func TestAssertionPersons(t *testing.T) {
	archive := &GLXFile{
		Events: map[string]*Event{
			"ev-baptism": {
				Type: EventTypeBaptism,
				Participants: []Participant{
					{Person: "p-child", Role: ParticipantRoleChild},
					{Person: "p-godparent", Role: "godparent"},
					{Person: "p-informant", Role: "informant"},
				},
			},
			"ev-marriage": {
				Type: EventTypeMarriage,
				Participants: []Participant{
					{Person: "p-groom", Role: ParticipantRoleGroom},
					{Person: "p-bride", Role: ParticipantRoleBride},
					{Person: "p-groom", Role: ParticipantRoleWitness},
					{Person: "p-witness", Role: ParticipantRoleWitness},
				},
			},
		},
	}

	tests := []struct {
		name      string
		assertion *Assertion
		want      []string
	}{
		{"nil assertion", nil, nil},
		{"person subject", &Assertion{Subject: EntityRef{Person: "p-x"}}, []string{"p-x"}},
		{"event subject: own-record roles only", &Assertion{Subject: EntityRef{Event: "ev-baptism"}}, []string{"p-child"}},
		{"couple, each person once", &Assertion{Subject: EntityRef{Event: "ev-marriage"}}, []string{"p-groom", "p-bride"}},
		{
			"participant assertion narrows to that person",
			&Assertion{Subject: EntityRef{Event: "ev-marriage"}, Participant: &Participant{Person: "p-bride", Role: ParticipantRoleBride}},
			[]string{"p-bride"},
		},
		{
			"participant assertion about a witness",
			&Assertion{Subject: EntityRef{Event: "ev-marriage"}, Participant: &Participant{Person: "p-witness", Role: ParticipantRoleWitness}},
			nil,
		},
		{
			"witness evidence does not inherit the groom conclusion",
			&Assertion{Subject: EntityRef{Event: "ev-marriage"}, Participant: &Participant{Person: "p-groom", Role: ParticipantRoleWitness}},
			nil,
		},
		{
			"own-record evidence does not inherit the witness conclusion",
			&Assertion{Subject: EntityRef{Event: "ev-marriage"}, Participant: &Participant{Person: "p-witness", Role: ParticipantRoleGroom}},
			[]string{"p-witness"},
		},
		{
			"godparent evidence does not inherit the child conclusion",
			&Assertion{Subject: EntityRef{Event: "ev-baptism"}, Participant: &Participant{Person: "p-child", Role: ParticipantRoleGodparent}},
			nil,
		},
		{"unknown event", &Assertion{Subject: EntityRef{Event: "ev-missing"}}, nil},
		{"relationship subject", &Assertion{Subject: EntityRef{Relationship: "rel-1"}}, nil},
		{"place subject", &Assertion{Subject: EntityRef{Place: "p-x"}}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, AssertionPersons(tt.assertion, archive))
		})
	}
}
