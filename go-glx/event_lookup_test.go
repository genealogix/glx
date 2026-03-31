package glx

import "testing"

func TestFindPersonEvent(t *testing.T) {
	archive := &GLXFile{
		Events: map[string]*Event{
			"event-birth-alice": {
				Type: EventTypeBirth,
				Date: "1850-03-15",
				Participants: []Participant{
					{Person: "person-alice", Role: ParticipantRolePrincipal},
				},
			},
			"event-birth-bob": {
				Type: EventTypeBirth,
				Date: "1855-07-20",
				Participants: []Participant{
					{Person: "person-bob", Role: ParticipantRolePrincipal},
					{Person: "person-alice", Role: ParticipantRoleWitness},
				},
			},
			"event-death-alice": {
				Type:    EventTypeDeath,
				Date:    "1920-11-01",
				PlaceID: "place-london",
				Participants: []Participant{
					{Person: "person-alice", Role: ParticipantRolePrincipal},
				},
			},
		},
	}

	t.Run("finds birth event for principal", func(t *testing.T) {
		id, event := FindPersonEvent(archive, "person-alice", EventTypeBirth)
		if event == nil {
			t.Fatal("expected to find birth event for alice")
		}
		if id != "event-birth-alice" {
			t.Errorf("got id %q, want %q", id, "event-birth-alice")
		}
		if string(event.Date) != "1850-03-15" {
			t.Errorf("got date %q, want %q", event.Date, "1850-03-15")
		}
	})

	t.Run("does not match witness role", func(t *testing.T) {
		id, event := FindPersonEvent(archive, "person-alice", EventTypeBirth)
		if event == nil {
			t.Fatal("expected to find a birth event for alice")
		}
		if id != "event-birth-alice" {
			t.Errorf("got id %q, want %q (should skip event where alice is witness)", id, "event-birth-alice")
		}
	})

	t.Run("finds death event", func(t *testing.T) {
		id, event := FindPersonEvent(archive, "person-alice", EventTypeDeath)
		if event == nil {
			t.Fatal("expected to find death event for alice")
		}
		if id != "event-death-alice" {
			t.Errorf("got id %q, want %q", id, "event-death-alice")
		}
		if event.PlaceID != "place-london" {
			t.Errorf("got place %q, want %q", event.PlaceID, "place-london")
		}
	})

	t.Run("returns nil for missing person", func(t *testing.T) {
		id, event := FindPersonEvent(archive, "person-unknown", EventTypeBirth)
		if event != nil {
			t.Errorf("expected nil, got event %q", id)
		}
	})

	t.Run("returns nil for missing event type", func(t *testing.T) {
		id, event := FindPersonEvent(archive, "person-bob", EventTypeDeath)
		if event != nil {
			t.Errorf("expected nil, got event %q", id)
		}
	})

	t.Run("matches empty role as subject", func(t *testing.T) {
		archiveWithEmptyRole := &GLXFile{
			Events: map[string]*Event{
				"event-birth-carol": {
					Type: EventTypeBirth,
					Date: "1860-01-01",
					Participants: []Participant{
						{Person: "person-carol"},
					},
				},
			},
		}
		_, event := FindPersonEvent(archiveWithEmptyRole, "person-carol", EventTypeBirth)
		if event == nil {
			t.Fatal("expected to find event with empty role")
		}
	})
}

func TestFindPersonEvent_Determinism(t *testing.T) {
	archive := &GLXFile{
		Events: map[string]*Event{
			"event-z-birth": {
				Type: EventTypeBirth,
				Date: "1850-01-01",
				Participants: []Participant{
					{Person: "person-x", Role: ParticipantRolePrincipal},
				},
			},
			"event-a-birth": {
				Type: EventTypeBirth,
				Date: "1850-06-15",
				Participants: []Participant{
					{Person: "person-x", Role: ParticipantRolePrincipal},
				},
			},
			"event-m-birth": {
				Type: EventTypeBirth,
				Date: "1850-03-10",
				Participants: []Participant{
					{Person: "person-x", Role: ParticipantRolePrincipal},
				},
			},
		},
	}

	// Must always return the lowest-sorted ID regardless of map iteration order.
	for range 100 {
		id, event := FindPersonEvent(archive, "person-x", EventTypeBirth)
		if event == nil {
			t.Fatal("expected to find birth event")
		}
		if id != "event-a-birth" {
			t.Fatalf("got id %q, want %q (must be deterministic)", id, "event-a-birth")
		}
	}
}

func TestIsSubjectRole(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{ParticipantRolePrincipal, true},
		{"subject", true},
		{"", true},
		{ParticipantRoleWitness, false},
		{ParticipantRoleInformant, false},
		{ParticipantRoleParent, false},
		{ParticipantRoleOfficiant, false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := isSubjectRole(tt.role); got != tt.want {
				t.Errorf("isSubjectRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}
