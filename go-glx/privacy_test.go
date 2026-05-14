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
	"time"
)

// fixedNow is the deterministic clock used by every test in this file so that
// the "born less than N years ago" heuristic produces stable results year over
// year.
var fixedNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestIsLivingPerson(t *testing.T) {
	tests := []struct {
		name    string
		archive *GLXFile
		want    bool
	}{
		{
			name: "explicit living true",
			archive: &GLXFile{
				Persons: map[string]*Person{
					"p": {Properties: map[string]any{PersonPropertyLiving: true}},
				},
			},
			want: true,
		},
		{
			name: "explicit living false beats heuristic",
			archive: &GLXFile{
				Persons: map[string]*Person{
					"p": {Properties: map[string]any{PersonPropertyLiving: false}},
				},
				Events: map[string]*Event{
					"e": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "non-bool living property falls through to heuristic",
			archive: &GLXFile{
				Persons: map[string]*Person{
					"p": {Properties: map[string]any{PersonPropertyLiving: "true"}},
				},
				Events: map[string]*Event{
					"e": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "death event with date is sufficient",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"d": {
						Type: EventTypeDeath,
						Date: "1920",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "death event with empty date is sufficient",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"d": {
						Type: EventTypeDeath,
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
					"b": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "burial without death event is sufficient",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"bu": {
						Type: EventTypeBurial,
						Date: "1920",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
					"b": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "cremation without death event is sufficient",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"c": {
						Type: EventTypeCremation,
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
					"b": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "born within threshold is living",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"b": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "born exactly threshold years ago is not living",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"b": {
						Type: EventTypeBirth,
						Date: "1926",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "born long ago is not living",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"b": {
						Type: EventTypeBirth,
						Date: "1850",
						Participants: []Participant{
							{Person: "p", Role: ParticipantRolePrincipal},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "no birth date and no end event is not living",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
			},
			want: false,
		},
		{
			name: "birth as witness does not count",
			archive: &GLXFile{
				Persons: map[string]*Person{"p": {}},
				Events: map[string]*Event{
					"b": {
						Type: EventTypeBirth,
						Date: "2010",
						Participants: []Participant{
							{Person: "subject", Role: ParticipantRolePrincipal},
							{Person: "p", Role: ParticipantRoleWitness},
						},
					},
				},
			},
			want: false,
		},
		{
			name:    "nil archive",
			archive: nil,
			want:    false,
		},
		{
			name: "unknown person id",
			archive: &GLXFile{
				Persons: map[string]*Person{},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLivingPerson(tt.archive, "p", fixedNow, LivingThresholdYears)
			if got != tt.want {
				t.Errorf("IsLivingPerson = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrivatizeLiving_RedactsLivingPerson(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"alive": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value":  "Jane Doe",
						"fields": map[string]any{"given": "Jane", "surname": "Doe"},
					},
					PersonPropertyOccupation: "engineer",
					PersonPropertyResidence:  "place-x",
					"ssn":                    "123-45-6789",
				},
				Notes: NoteList{"private note"},
			},
			"dead": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{"value": "John Old"},
				},
			},
		},
		Events: map[string]*Event{
			"birth-alive": {
				Type:    EventTypeBirth,
				Date:    "2010-04-15",
				PlaceID: "place-x",
				Notes:   NoteList{"hospital note"},
				Properties: map[string]any{
					"description": "Born at General Hospital",
				},
				Participants: []Participant{
					{Person: "alive", Role: ParticipantRolePrincipal},
				},
			},
			"death-dead": {
				Type:    EventTypeDeath,
				Date:    "1920-06-01",
				PlaceID: "place-y",
				Participants: []Participant{
					{Person: "dead", Role: ParticipantRolePrincipal},
				},
			},
		},
		Assertions: map[string]*Assertion{
			"a1": {
				Subject:  EntityRef{Person: "alive"},
				Property: PersonPropertyOccupation,
				Value:    "engineer",
				Sources:  []string{"source-x"},
			},
			"a2": {
				Subject:  EntityRef{Person: "dead"},
				Property: PersonPropertyName,
				Value:    "John Old",
			},
		},
	}

	result := PrivatizeLiving(archive, fixedNow, LivingThresholdYears)
	if result.PersonsRedacted != 1 {
		t.Errorf("PrivatizeLiving PersonsRedacted = %d, want 1", result.PersonsRedacted)
	}
	if result.EventsRedacted != 1 {
		t.Errorf("PrivatizeLiving EventsRedacted = %d, want 1", result.EventsRedacted)
	}
	if result.AssertionsDropped != 1 {
		t.Errorf("PrivatizeLiving AssertionsDropped = %d, want 1", result.AssertionsDropped)
	}

	alive := archive.Persons["alive"]
	if len(alive.Properties) != 2 {
		t.Errorf("alive person should have exactly 2 properties (name, living); got %d: %v", len(alive.Properties), alive.Properties)
	}
	nameProp, ok := alive.Properties[PersonPropertyName].(map[string]any)
	if !ok || nameProp["value"] != "Living" {
		t.Errorf("alive person name should be redacted to 'Living'; got %v", alive.Properties[PersonPropertyName])
	}
	if living, _ := alive.Properties[PersonPropertyLiving].(bool); !living {
		t.Errorf("alive person living flag should be true; got %v", alive.Properties[PersonPropertyLiving])
	}
	if alive.Notes != nil {
		t.Errorf("alive person notes should be cleared; got %v", alive.Notes)
	}

	dead := archive.Persons["dead"]
	if dead.Properties[PersonPropertyName].(map[string]any)["value"] != "John Old" {
		t.Errorf("dead person should be untouched; got %v", dead.Properties[PersonPropertyName])
	}

	birthAlive := archive.Events["birth-alive"]
	if birthAlive.Date != "" || birthAlive.PlaceID != "" || birthAlive.Notes != nil || birthAlive.Properties != nil {
		t.Errorf("living person's birth event should be redacted; got date=%q place=%q notes=%v props=%v",
			birthAlive.Date, birthAlive.PlaceID, birthAlive.Notes, birthAlive.Properties)
	}
	if birthAlive.Type != EventTypeBirth {
		t.Errorf("event type should be preserved; got %q", birthAlive.Type)
	}
	if len(birthAlive.Participants) != 1 {
		t.Errorf("event participants should be preserved; got %v", birthAlive.Participants)
	}

	deathDead := archive.Events["death-dead"]
	if deathDead.Date != "1920-06-01" || deathDead.PlaceID != "place-y" {
		t.Errorf("dead person's death event should be untouched; got date=%q place=%q",
			deathDead.Date, deathDead.PlaceID)
	}

	if _, ok := archive.Assertions["a1"]; ok {
		t.Errorf("assertion about living person should be dropped")
	}
	if _, ok := archive.Assertions["a2"]; !ok {
		t.Errorf("assertion about dead person should be retained")
	}
}

func TestPrivatizeLiving_AssertionWithLivingParticipant(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"alive": {Properties: map[string]any{PersonPropertyLiving: true}},
			"dead":  {},
		},
		Assertions: map[string]*Assertion{
			"a1": {
				Subject:     EntityRef{Event: "marriage-event"},
				Participant: &Participant{Person: "alive", Role: ParticipantRoleSpouse},
			},
			"a2": {
				Subject:     EntityRef{Event: "marriage-event"},
				Participant: &Participant{Person: "dead", Role: ParticipantRoleSpouse},
			},
		},
	}

	PrivatizeLiving(archive, fixedNow, LivingThresholdYears)

	if _, ok := archive.Assertions["a1"]; ok {
		t.Errorf("assertion whose participant is a living person should be dropped")
	}
	if _, ok := archive.Assertions["a2"]; !ok {
		t.Errorf("assertion whose participant is a dead person should be retained")
	}
}

func TestPrivatizeLiving_NoLivingPersons(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"dead1": {Properties: map[string]any{PersonPropertyName: map[string]any{"value": "Old One"}}},
			"dead2": {Properties: map[string]any{PersonPropertyName: map[string]any{"value": "Old Two"}}},
		},
	}

	result := PrivatizeLiving(archive, fixedNow, LivingThresholdYears)
	if result.PersonsRedacted != 0 || result.EventsRedacted != 0 ||
		result.EventsScrubbed != 0 || result.RelationshipsRedacted != 0 ||
		result.AssertionsDropped != 0 {
		t.Errorf("expected empty result, got %+v", result)
	}
	if archive.Persons["dead1"].Properties[PersonPropertyName].(map[string]any)["value"] != "Old One" {
		t.Errorf("dead1 should be untouched")
	}
}

func TestPrivatizeLiving_NilArchive(t *testing.T) {
	got := PrivatizeLiving(nil, fixedNow, LivingThresholdYears)
	if got == nil || got.PersonsRedacted != 0 || got.EventsRedacted != 0 ||
		got.EventsScrubbed != 0 || got.RelationshipsRedacted != 0 ||
		got.AssertionsDropped != 0 {
		t.Errorf("PrivatizeLiving(nil) = %+v, want empty result", got)
	}
}

// TestPrivatizeLiving_MarriageBetweenLivingSpouses covers the case where two
// living spouses are joined by a marriage relationship and event. Spouse role
// is not a subject role, so the marriage event would otherwise leak DATE /
// PLAC / NOTE / Properties through the FAM record. The relationship pass must
// fully redact both the event and the relationship payload.
func TestPrivatizeLiving_MarriageBetweenLivingSpouses(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"alice": {Properties: map[string]any{PersonPropertyLiving: true}},
			"bob":   {Properties: map[string]any{PersonPropertyLiving: true}},
		},
		Events: map[string]*Event{
			"marriage": {
				Type:    EventTypeMarriage,
				Date:    "2018-06-15",
				PlaceID: "place-las-vegas",
				Notes:   NoteList{"Eloped while parents were on vacation"},
				Properties: map[string]any{
					"description": "Marriage of Alice and Bob",
				},
				Participants: []Participant{
					{
						Person:     "alice",
						Role:       ParticipantRoleSpouse,
						Properties: map[string]any{"name_as_recorded": map[string]any{"value": "Alice Currentperson"}},
						Notes:      NoteList{"alice's note"},
					},
					{
						Person:     "bob",
						Role:       ParticipantRoleSpouse,
						Properties: map[string]any{"name_as_recorded": map[string]any{"value": "Bob Currentperson"}},
					},
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-marriage": {
				Type:       RelationshipTypeMarriage,
				StartEvent: "marriage",
				Participants: []Participant{
					{Person: "alice", Role: ParticipantRoleSpouse, Notes: NoteList{"a's role note"}},
					{Person: "bob", Role: ParticipantRoleSpouse},
				},
				Notes:      NoteList{"Met at Stanford 2018; via Jane Doe"},
				Properties: map[string]any{"marriage_type": "civil"},
			},
		},
	}

	result := PrivatizeLiving(archive, fixedNow, LivingThresholdYears)
	if result.RelationshipsRedacted != 1 {
		t.Errorf("RelationshipsRedacted = %d, want 1", result.RelationshipsRedacted)
	}
	if result.EventsRedacted != 1 {
		t.Errorf("EventsRedacted = %d, want 1", result.EventsRedacted)
	}

	rel := archive.Relationships["rel-marriage"]
	if rel.Notes != nil {
		t.Errorf("relationship notes should be cleared; got %v", rel.Notes)
	}
	if rel.Properties != nil {
		t.Errorf("relationship properties should be cleared; got %v", rel.Properties)
	}
	for i, p := range rel.Participants {
		if p.Properties != nil || p.Notes != nil {
			t.Errorf("relationship participant %d (%s) should have nil Properties/Notes; got props=%v notes=%v",
				i, p.Person, p.Properties, p.Notes)
		}
		if p.Person == "" || p.Role == "" {
			t.Errorf("relationship participant %d should preserve Person and Role; got %+v", i, p)
		}
	}

	ev := archive.Events["marriage"]
	if ev.Date != "" || ev.PlaceID != "" || ev.Notes != nil || ev.Properties != nil {
		t.Errorf("marriage event should be fully redacted; got date=%q place=%q notes=%v props=%v",
			ev.Date, ev.PlaceID, ev.Notes, ev.Properties)
	}
	for i, p := range ev.Participants {
		if p.Properties != nil || p.Notes != nil {
			t.Errorf("event participant %d (%s) should have nil Properties/Notes; got props=%v notes=%v",
				i, p.Person, p.Properties, p.Notes)
		}
	}
}

// TestPrivatizeLiving_ScrubsLivingNonSubjectParticipants covers a dead
// person's event that names a living person in a non-subject role (witness,
// informant). The event itself is not redacted, but the living-witness's
// participant-level Properties and Notes must be scrubbed so per-participant
// fields like name_as_recorded do not leak.
func TestPrivatizeLiving_ScrubsLivingNonSubjectParticipants(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"alive": {Properties: map[string]any{PersonPropertyLiving: true}},
			"dead":  {},
		},
		Events: map[string]*Event{
			"death-dead": {
				Type:    EventTypeDeath,
				Date:    "1920-06-01",
				PlaceID: "place-y",
				Participants: []Participant{
					{Person: "dead", Role: ParticipantRolePrincipal},
					{
						Person:     "alive",
						Role:       ParticipantRoleWitness,
						Properties: map[string]any{"name_as_recorded": map[string]any{"value": "Alive Witness"}},
						Notes:      NoteList{"signed the death certificate"},
					},
				},
			},
		},
	}

	result := PrivatizeLiving(archive, fixedNow, LivingThresholdYears)
	if result.EventsRedacted != 0 {
		t.Errorf("dead person's event should not be fully redacted; EventsRedacted = %d", result.EventsRedacted)
	}
	if result.EventsScrubbed != 1 {
		t.Errorf("event should be scrubbed for living non-subject participant; EventsScrubbed = %d", result.EventsScrubbed)
	}

	ev := archive.Events["death-dead"]
	if ev.Date != "1920-06-01" || ev.PlaceID != "place-y" {
		t.Errorf("dead person's event top-level fields should be preserved; got date=%q place=%q", ev.Date, ev.PlaceID)
	}
	for _, p := range ev.Participants {
		if p.Person == "alive" {
			if p.Properties != nil || p.Notes != nil {
				t.Errorf("living witness participant should be scrubbed; got props=%v notes=%v",
					p.Properties, p.Notes)
			}
		}
		if p.Person == "dead" {
			if p.Role != ParticipantRolePrincipal {
				t.Errorf("dead principal should be preserved; got %+v", p)
			}
		}
	}
}
