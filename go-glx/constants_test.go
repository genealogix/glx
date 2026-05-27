package glx

import "testing"

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
