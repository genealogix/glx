package main

import "testing"

func TestCollectionForEntityType(t *testing.T) {
	cases := []struct {
		flag string
		want string
		ok   bool
	}{
		{"person", "persons", true},
		{"event", "events", true},
		{"media", "media", true},
		{"relationship", "relationships", true},
		{"research-log", "research_logs", true},
		{"study", "studies", true},
		// vocabulary-entry must map to the SINGULAR-derived "event_types",
		// not the plural "events_types" — regression guard.
		{"vocabulary-entry", "event_types", true},
		// case/space tolerant
		{"  Person ", "persons", true},
		// rejects
		{"nonsense", "", false},
		{"", "", false},
		{"persons", "", false}, // plural form is not a valid --entity-type
	}
	for _, c := range cases {
		got, ok := collectionForEntityType(c.flag)
		if ok != c.ok || got != c.want {
			t.Errorf("collectionForEntityType(%q) = (%q, %v), want (%q, %v)",
				c.flag, got, ok, c.want, c.ok)
		}
	}
}
