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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roleContextWarnings returns the applies_to warnings from validating archive.
func roleContextWarnings(t *testing.T, archive *GLXFile) []ValidationWarning {
	t.Helper()
	result := archive.Validate()
	require.Empty(t, result.Errors)

	var out []ValidationWarning
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "applies_to") {
			out = append(out, w)
		}
	}

	return out
}

func roleContextArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{"person-a": {}, "person-b": {}},
		ParticipantRoles: map[string]*VocabularyEntry{
			"principal": {Label: "Principal", AppliesTo: []string{RoleContextEvent}},
			"sibling":   {Label: "Sibling", AppliesTo: []string{RoleContextRelationship}},
			"guardian":  {Label: "Guardian", AppliesTo: []string{RoleContextEvent, RoleContextRelationship}},
			"anything":  {Label: "Custom role without applies_to"},
		},
	}
}

func TestValidateParticipantRoleContexts(t *testing.T) {
	t.Run("event-only role on a relationship warns", func(t *testing.T) {
		archive := roleContextArchive()
		archive.Relationships = map[string]*Relationship{
			"rel-1": {Participants: []Participant{
				{Person: "person-a", Role: "principal"},
				{Person: "person-b", Role: "sibling"},
			}},
		}

		warnings := roleContextWarnings(t, archive)
		require.Len(t, warnings, 1)
		assert.Equal(t, EntityTypeRelationships, warnings[0].SourceType)
		assert.Equal(t, "rel-1", warnings[0].SourceID)
		assert.Equal(t, "participants[0].role", warnings[0].Field)
		assert.Contains(t, warnings[0].Message, "role 'principal' is used on a relationship, but its applies_to is [event]")
	})

	t.Run("relationship-only role on an event warns", func(t *testing.T) {
		archive := roleContextArchive()
		archive.Events = map[string]*Event{
			"event-1": {Participants: []Participant{
				{Person: "person-a", Role: "principal"},
				{Person: "person-b", Role: "sibling"},
			}},
		}

		warnings := roleContextWarnings(t, archive)
		require.Len(t, warnings, 1)
		assert.Equal(t, EntityTypeEvents, warnings[0].SourceType)
		assert.Equal(t, "participants[1].role", warnings[0].Field)
		assert.Contains(t, warnings[0].Message, "used on an event")
	})

	t.Run("roles valid in both contexts, omitted applies_to and no role do not warn", func(t *testing.T) {
		archive := roleContextArchive()
		archive.Events = map[string]*Event{
			"event-1": {Participants: []Participant{
				{Person: "person-a", Role: "guardian"},
				{Person: "person-b", Role: "anything"},
				{Person: "person-b"},
			}},
		}
		archive.Relationships = map[string]*Relationship{
			"rel-1": {Participants: []Participant{
				{Person: "person-a", Role: "guardian"},
				{Person: "person-b", Role: "anything"},
			}},
		}

		assert.Empty(t, roleContextWarnings(t, archive))
	})

	t.Run("assertion participant takes its subject's context", func(t *testing.T) {
		archive := roleContextArchive()
		archive.Events = map[string]*Event{
			"event-1": {Participants: []Participant{{Person: "person-a", Role: "principal"}}},
		}
		archive.Relationships = map[string]*Relationship{
			"rel-1": {Participants: []Participant{
				{Person: "person-a", Role: "sibling"},
				{Person: "person-b", Role: "sibling"},
			}},
		}
		archive.Assertions = map[string]*Assertion{
			"assertion-on-event": {
				Subject:     EntityRef{Event: "event-1"},
				Participant: &Participant{Person: "person-b", Role: "sibling"},
			},
			"assertion-on-rel": {
				Subject:     EntityRef{Relationship: "rel-1"},
				Participant: &Participant{Person: "person-b", Role: "sibling"},
			},
		}

		warnings := roleContextWarnings(t, archive)
		require.Len(t, warnings, 1)
		assert.Equal(t, EntityTypeAssertions, warnings[0].SourceType)
		assert.Equal(t, "assertion-on-event", warnings[0].SourceID)
		assert.Equal(t, "participant.role", warnings[0].Field)
	})

	t.Run("unknown role is an error, not an applies_to warning", func(t *testing.T) {
		archive := roleContextArchive()
		archive.Events = map[string]*Event{
			"event-1": {Participants: []Participant{{Person: "person-a", Role: "no_such_role"}}},
		}

		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		for _, w := range result.Warnings {
			assert.NotContains(t, w.Message, "applies_to")
		}
	})
}

// TestStandardParticipantRolesDeclareAppliesTo pins the decisions made for
// #499 and #526: every standard role declares applies_to, and roles that glx
// itself or common records place on events (GEDCOM import writes spouse on
// marriage events and parent/child from ASSO ROLE; a census names its head)
// accept the event context.
func TestStandardParticipantRolesDeclareAppliesTo(t *testing.T) {
	archive := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

	for key, entry := range archive.ParticipantRoles {
		assert.NotEmpty(t, entry.AppliesTo, "standard role %q must declare applies_to", key)
	}

	both := []string{RoleContextEvent, RoleContextRelationship}
	want := map[string][]string{
		ParticipantRoleSpouse:             both,
		ParticipantRoleParent:             both,
		ParticipantRoleChild:              both,
		ParticipantRoleGuardian:           both,
		ParticipantRoleWard:               both,
		ParticipantRoleHouseholdHead:      both,
		ParticipantRoleBoarder:            both,
		ParticipantRoleGodparent:          both,
		ParticipantRoleWitness:            {RoleContextEvent},
		ParticipantRoleSubject:            {RoleContextEvent},
		ParticipantRoleEnumerator:         {RoleContextEvent},
		ParticipantRoleAttendingPhysician: {RoleContextEvent},
		ParticipantRoleRegistrar:          {RoleContextEvent},
		ParticipantRoleBondsman:           {RoleContextEvent},
		ParticipantRoleTestator:           {RoleContextEvent},
		ParticipantRoleExecutor:           {RoleContextEvent},
		ParticipantRoleLegatee:            {RoleContextEvent},
		ParticipantRoleBeneficiary:        {RoleContextEvent},
		ParticipantRoleGrantor:            {RoleContextEvent},
		ParticipantRoleGrantee:            {RoleContextEvent},
		ParticipantRoleAdjoiningOwner:     {RoleContextEvent},
		ParticipantRoleSibling:            {RoleContextRelationship},
		ParticipantRoleFosterParent:       {RoleContextRelationship},
		ParticipantRoleFosterChild:        {RoleContextRelationship},
		ParticipantRoleStepParent:         {RoleContextRelationship},
		ParticipantRoleStepChild:          {RoleContextRelationship},
	}
	for role, contexts := range want {
		entry, ok := archive.ParticipantRoles[role]
		if assert.True(t, ok, "standard vocabulary must define role %q", role) {
			assert.ElementsMatch(t, contexts, entry.AppliesTo, "applies_to of %q", role)
		}
	}
}

func TestStandardVocabulariesLandTerritorySearchResults(t *testing.T) {
	archive := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

	assert.Contains(t, archive.EventTypes, EventTypeLandTransaction)
	assert.Equal(t, "legal", archive.EventTypes[EventTypeLandTransaction].Category)
	assert.Empty(t, archive.EventTypes[EventTypeLandTransaction].GEDCOM, "land_transaction has no GEDCOM tag; export uses EVEN + TYPE")
	assert.Contains(t, archive.PlaceTypes, PlaceTypeTerritory)
	assert.Contains(t, archive.SearchResultTypes, SearchResultUnavailable)
	assert.Contains(t, archive.SearchResultTypes, SearchResultRequiresVisit)
}
