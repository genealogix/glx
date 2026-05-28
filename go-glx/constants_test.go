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

func TestConstants(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "RelationshipTypePossiblySamePerson",
			got:  RelationshipTypePossiblySamePerson,
			want: "possibly_same_person",
		},
		{
			name: "EventTypeBirth",
			got:  EventTypeBirth,
			want: "birth",
		},
		{
			name: "EventTypeMarriageBanns",
			got:  EventTypeMarriageBanns,
			want: "marriage_banns",
		},
		{
			name: "ParticipantRoleOfficiant",
			got:  ParticipantRoleOfficiant,
			want: "officiant",
		},
		{
			name: "ParticipantRoleEnslavedPerson",
			got:  ParticipantRoleEnslavedPerson,
			want: "enslaved_person",
		},
		{
			name: "GedcomTagBirt",
			got:  GedcomTagBirt,
			want: "BIRT",
		},
		{
			name: "GedcomTagMarr",
			got:  GedcomTagMarr,
			want: "MARR",
		},
		{
			name: "GedcomTagRole",
			got:  GedcomTagRole,
			want: "ROLE",
		},
		{
			name: "GedcomTagStae",
			got:  GedcomTagStae,
			want: "STAE",
		},
		{
			name: "VocabRelationshipTypes",
			got:  VocabRelationshipTypes,
			want: "relationship_types",
		},
		{
			name: "VocabEventTypes",
			got:  VocabEventTypes,
			want: "event_types",
		},
		{
			name: "VocabParticipantRoles",
			got:  VocabParticipantRoles,
			want: "participant_roles",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Fatalf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestEntityTypeString(t *testing.T) {
	assert.Equal(t, "persons", EntityTypePersons.String())
	assert.Equal(t, "events", EntityTypeEvents.String())
	assert.Equal(t, "research_logs", EntityTypeResearchLogs.String())
	assert.Equal(t, "studies", EntityTypeStudies.String())
}

func TestEntityTypePluralMatchesString(t *testing.T) {
	for _, et := range AllEntityTypes {
		assert.Equal(t, et.String(), et.Plural(), "Plural and String should agree for %s", et)
	}
}

func TestEntityTypeSingular(t *testing.T) {
	tests := []struct {
		et       EntityType
		singular string
	}{
		{EntityTypePersons, "person"},
		{EntityTypeRelationships, "relationship"},
		{EntityTypeEvents, "event"},
		{EntityTypePlaces, "place"},
		{EntityTypeSources, "source"},
		{EntityTypeCitations, "citation"},
		{EntityTypeRepositories, "repository"},
		{EntityTypeAssertions, "assertion"},
		{EntityTypeMedia, "media"},
		{EntityTypeResearchLogs, "research-log"},
		{EntityTypeStudies, "study"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.singular, tt.et.Singular(), "Singular(%s)", tt.et)
	}
}

func TestEntityTypeSingularReturnsEmptyForUnknown(t *testing.T) {
	assert.Empty(t, EntityType("not_a_real_type").Singular())
}

func TestEntityTypeIDPrefix(t *testing.T) {
	tests := []struct {
		et     EntityType
		prefix string
	}{
		{EntityTypePersons, "person-"},
		{EntityTypeResearchLogs, "research-log-"},
		{EntityTypeStudies, "study-"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.prefix, tt.et.IDPrefix(), "IDPrefix(%s)", tt.et)
	}
}

func TestEntityTypeIDPrefixMatchesLegacyConstants(t *testing.T) {
	// Pin that the typed-enum IDPrefix() matches the older EntityIDPrefix*
	// string constants, since both are still in use across the codebase.
	tests := []struct {
		et     EntityType
		legacy string
	}{
		{EntityTypePersons, EntityIDPrefixPerson},
		{EntityTypeRelationships, EntityIDPrefixRelationship},
		{EntityTypeEvents, EntityIDPrefixEvent},
		{EntityTypePlaces, EntityIDPrefixPlace},
		{EntityTypeSources, EntityIDPrefixSource},
		{EntityTypeCitations, EntityIDPrefixCitation},
		{EntityTypeRepositories, EntityIDPrefixRepository},
		{EntityTypeAssertions, EntityIDPrefixAssertion},
		{EntityTypeMedia, EntityIDPrefixMedia},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.legacy, tt.et.IDPrefix(),
			"IDPrefix(%s) must match EntityIDPrefix constant", tt.et)
	}
}

func TestAllEntityTypesIsExhaustive(t *testing.T) {
	// Every EntityType that appears in entityTypeSingular must also appear in
	// AllEntityTypes. If a new type is added to one but not the other,
	// consumers that iterate AllEntityTypes silently miss it.
	listed := make(map[EntityType]bool, len(AllEntityTypes))
	for _, et := range AllEntityTypes {
		listed[et] = true
	}
	for et := range entityTypeSingular {
		assert.True(t, listed[et], "EntityType %s has a Singular() entry but is missing from AllEntityTypes", et)
	}
	require.Len(t, AllEntityTypes, len(entityTypeSingular),
		"AllEntityTypes and entityTypeSingular must cover the same set")
}

func TestIsValidEntityType(t *testing.T) {
	assert.True(t, IsValidEntityType("persons"))
	assert.True(t, IsValidEntityType("research_logs"))
	assert.True(t, IsValidEntityType("studies"))
	assert.False(t, IsValidEntityType(""))
	assert.False(t, IsValidEntityType("person"))       // singular form is not valid
	assert.False(t, IsValidEntityType("vocabularies")) // not an entity type
	assert.False(t, IsValidEntityType("not_a_real_type"))
}

func TestArchiveDirVocabulariesNotAnEntityType(t *testing.T) {
	// Pin that ArchiveDirVocabularies is the literal "vocabularies" and is
	// distinct from every EntityType — vocabularies are an archive-managed
	// directory but not an entity type.
	assert.Equal(t, "vocabularies", ArchiveDirVocabularies)
	assert.False(t, IsValidEntityType(ArchiveDirVocabularies))
}
