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

package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateSortKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1850-01-15", "1850-01-15"},
		{"1850-01", "1850-01"},
		{"1850", "1850"},
		{"ABT 1880", "1880"},
		{"BEF 1920-01-15", "1920-01-15"},
		{"AFT 1900", "1900"},
		{"BET 1880 AND 1890", "1880"},
		{"CAL 1855", "1855"},
		{"800", "0800"},
		{"476", "0476"},
		{"ABT 476", "0476"},
		{"BET 900 AND 1000", "0900"},
		{"15 MAR 800", "0800"},
		{"15 MAR 1850", "1850"},
		{"", "\xff"},
		{"unknown", "\xff"},
	}

	for _, tt := range tests {
		got := dateSortKey(tt.input)
		if got != tt.want {
			t.Errorf("dateSortKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatEventTypeLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"birth", "Birth"},
		{"death", "Death"},
		{"marriage", "Marriage"},
		{"census", "Census"},
		{"legal_separation", "Legal separation"},
		{"", "Event"},
	}

	for _, tt := range tests {
		got := formatEventTypeLabel(tt.input)
		if got != tt.want {
			t.Errorf("formatEventTypeLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCollectDirectEvents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type: "birth",
				Date: "1850-01-15",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-death": {
				Type: "death",
				Date: "1920-03-10",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
			"event-other-marriage": {
				Type: "marriage",
				Date: "1890-06-01",
				Participants: []glxlib.Participant{
					{Person: "person-jane", Role: "bride"},
					{Person: "person-bob", Role: "groom"},
				},
			},
		},
	}

	entries := collectDirectEvents("person-john", archive)

	if len(entries) != 2 {
		t.Fatalf("expected 2 direct events, got %d", len(entries))
	}

	// Check that none are marked as family events
	for _, e := range entries {
		if e.IsFamily {
			t.Errorf("direct event %q should not be marked as family", e.Label)
		}
	}
}

func TestCollectDirectEvents_SynthesizesFromProperties(t *testing.T) {
	// Person has birth/death event entities plus a census event
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-robert": {Properties: map[string]any{
				"name": "Robert Webb",
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-robert": {
				Type:    "birth",
				Date:    "ABT 1815",
				PlaceID: "place-virginia",
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: "principal"},
				},
			},
			"event-death-robert": {
				Type:    "death",
				Date:    "10 FEB 1863",
				PlaceID: "place-tn",
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: "principal"},
				},
			},
			"event-census-1860": {
				Type: "census",
				Date: "1860",
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-virginia": {Name: "Virginia"},
			"place-tn":       {Name: "Riverside, TN"},
		},
	}

	entries := collectDirectEvents("person-robert", archive)

	// Should have 3 entries: birth + death + census
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (birth + death + census), got %d", len(entries))
	}

	labels := map[string]bool{}
	for _, e := range entries {
		labels[e.Label] = true
	}
	if !labels["Birth"] {
		t.Error("expected Birth entry from birth event")
	}
	if !labels["Death"] {
		t.Error("expected Death entry from death event")
	}
	if !labels["Census"] {
		t.Error("expected Census event entry")
	}
}

func TestCollectDirectEvents_NoDuplicateWhenEventExists(t *testing.T) {
	// Person has both a birth event AND born_on property — should not duplicate
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{
				"name":    "John Smith",
				"born_on": "1850-01-15",
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type: "birth",
				Date: "1850-01-15",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
		},
	}

	entries := collectDirectEvents("person-john", archive)

	// Should have exactly 1 entry (the event), not a duplicate from properties
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (no duplicate), got %d", len(entries))
	}
	if entries[0].Label != "Birth" {
		t.Errorf("expected Birth, got %s", entries[0].Label)
	}
}

func TestCollectFamilyEvents_Spouse(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Brown"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "spouse"},
					{Person: "person-mary", Role: "spouse"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-mary-birth": {
				Type: "birth",
				Date: "1855",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "principal"},
				},
			},
			"event-mary-death": {
				Type: "death",
				Date: "1930-05-20",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	entries := collectFamilyEvents("person-john", archive)

	// Should find Mary's birth and death (from events)
	if len(entries) != 2 {
		t.Fatalf("expected 2 family events, got %d", len(entries))
	}

	foundBirth := false
	foundDeath := false
	for _, e := range entries {
		if !e.IsFamily {
			t.Errorf("family event %q should be marked as family", e.Label)
		}
		if e.Label == "Birth of spouse (Mary Brown)" {
			foundBirth = true
		}
		if e.Label == "Death of spouse (Mary Brown)" {
			foundDeath = true
		}
	}

	if !foundBirth {
		t.Error("expected 'Birth of spouse (Mary Brown)' entry")
	}
	if !foundDeath {
		t.Error("expected 'Death of spouse (Mary Brown)' entry")
	}
}

func TestCollectFamilyEvents_Children(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-jane": {Properties: map[string]any{"name": "Jane Smith"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-parent-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "parent"},
					{Person: "person-jane", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-jane-birth": {
				Type: "birth",
				Date: "1880-09-05",
				Participants: []glxlib.Participant{
					{Person: "person-jane", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	entries := collectFamilyEvents("person-john", archive)

	if len(entries) != 1 {
		t.Fatalf("expected 1 family event (child birth), got %d", len(entries))
	}

	if entries[0].Label != "Birth of child (Jane Smith)" {
		t.Errorf("expected label 'Birth of child (Jane Smith)', got %q", entries[0].Label)
	}
}

func TestCollectFamilyEvents_Parents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john":   {Properties: map[string]any{"name": "John Smith"}},
			"person-father": {Properties: map[string]any{"name": "William Smith"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-parent-child": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-father", Role: "parent"},
					{Person: "person-john", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-father-birth": {
				Type: "birth",
				Date: "1820-01-01",
				Participants: []glxlib.Participant{
					{Person: "person-father", Role: "subject"},
				},
			},
			"event-father-death": {
				Type: "death",
				Date: "1900-12-25",
				Participants: []glxlib.Participant{
					{Person: "person-father", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	entries := collectFamilyEvents("person-john", archive)

	// Should only include parent's death, not birth
	if len(entries) != 1 {
		t.Fatalf("expected 1 family event (parent death), got %d", len(entries))
	}

	if entries[0].Label != "Death of parent (William Smith)" {
		t.Errorf("expected label 'Death of parent (William Smith)', got %q", entries[0].Label)
	}
}

func TestDeduplicateEntries(t *testing.T) {
	direct := []timelineEntry{
		{EventID: "event-marriage", Label: "Marriage", IsFamily: false},
		{EventID: "event-birth", Label: "Birth", IsFamily: false},
	}
	family := []timelineEntry{
		{EventID: "event-marriage", Label: "Marriage of spouse (Mary)", IsFamily: true}, // Duplicate
		{EventID: "event-child-birth", Label: "Birth of child (Jane)", IsFamily: true},
	}

	result := deduplicateEntries(direct, family)

	if len(result) != 3 {
		t.Fatalf("expected 3 entries after dedup, got %d", len(result))
	}

	// The marriage should use the direct version
	for _, e := range result {
		if e.EventID == "event-marriage" && e.IsFamily {
			t.Error("duplicate event-marriage should use direct version, not family")
		}
	}
}

func TestSortTimelineEntries(t *testing.T) {
	entries := []timelineEntry{
		{Date: "1920-03-10", SortKey: dateSortKey("1920-03-10"), Label: "Death"},
		{Date: "", SortKey: dateSortKey(""), Label: "Undated"},
		{Date: "1850-01-15", SortKey: dateSortKey("1850-01-15"), Label: "Birth"},
		{Date: "ABT 1880", SortKey: dateSortKey("ABT 1880"), Label: "Marriage"},
	}

	sortTimelineEntries(entries)

	expected := []string{"Birth", "Marriage", "Death", "Undated"}
	for i, want := range expected {
		if entries[i].Label != want {
			t.Errorf("position %d: expected %q, got %q", i, want, entries[i].Label)
		}
	}
}

func TestTimelineNoFamily(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Brown", "born_on": "1855"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "spouse"},
					{Person: "person-mary", Role: "spouse"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-john-birth": {
				Type: "birth",
				Date: "1850-01-15",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{},
	}

	entries := collectTimelineEntries("person-john", archive, false)

	// Should only have John's birth, no family events
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry with --no-family, got %d", len(entries))
	}

	if entries[0].Label != "Birth" {
		t.Errorf("expected 'Birth', got %q", entries[0].Label)
	}
}

func TestTimelineEmptyArchive(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events:        map[string]*glxlib.Event{},
		Relationships: map[string]*glxlib.Relationship{},
		Places:        map[string]*glxlib.Place{},
	}

	entries := collectTimelineEntries("person-john", archive, true)

	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for person with no events, got %d", len(entries))
	}
}

func TestFindPersonForTimeline_ExactID(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}

	id, _, err := findPersonForTimeline(archive, "person-john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "person-john" {
		t.Errorf("expected person-john, got %s", id)
	}
}

func TestFindPersonForTimeline_NameSearch(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-mary": {Properties: map[string]any{"name": "Mary Brown"}},
		},
	}

	id, _, err := findPersonForTimeline(archive, "john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "person-john" {
		t.Errorf("expected person-john, got %s", id)
	}
}

func TestFindPersonForTimeline_NotFound(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}

	_, _, err := findPersonForTimeline(archive, "nobody")
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestFindPersonForTimeline_Ambiguous(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john-sr": {Properties: map[string]any{"name": "John Smith Sr"}},
			"person-john-jr": {Properties: map[string]any{"name": "John Smith Jr"}},
		},
	}

	_, _, err := findPersonForTimeline(archive, "john")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
}

func TestFindPersonForTimeline_NilEntry(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-nil": nil,
		},
	}

	_, _, err := findPersonForTimeline(archive, "person-nil")
	if err == nil {
		t.Fatal("expected error for nil person entry")
	}
	if !strings.Contains(err.Error(), "no data") {
		t.Errorf("expected 'no data' error, got: %s", err.Error())
	}
}

func TestInferRelation(t *testing.T) {
	tests := []struct {
		relType    string
		targetRole string
		otherRole  string
		want       string
	}{
		{"marriage", "spouse", "spouse", "spouse"},
		{"partner", "spouse", "spouse", "spouse"},
		{"parent_child", "parent", "child", "child"},
		{"parent_child", "child", "parent", "parent"},
		{"biological_parent_child", "parent", "child", "child"},
		{"adoptive_parent_child", "child", "parent", "parent"},
		{"foster_parent_child", "parent", "child", "child"},
		{"step_parent", "child", "parent", "parent"},
		{"guardian", "parent", "child", "child"},
		{"guardian", "child", "parent", "parent"},
		{"parent_child", "", "parent", "parent"},
		{"parent_child", "", "child", "child"},
		{"parent_child", "", "", ""},
		{"neighbor", "principal", "principal", ""},
	}

	for _, tt := range tests {
		got := inferRelation(tt.relType, tt.targetRole, tt.otherRole)
		if got != tt.want {
			t.Errorf("inferRelation(%q, %q, %q) = %q, want %q",
				tt.relType, tt.targetRole, tt.otherRole, got, tt.want)
		}
	}
}

func TestTimelineResolvePlaceName(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-leeds": {Name: "Leeds, Yorkshire, England"},
		},
	}

	if got := timelineResolvePlaceName("place-leeds", archive); got != "Leeds, Yorkshire, England" {
		t.Errorf("expected 'Leeds, Yorkshire, England', got %q", got)
	}
	if got := timelineResolvePlaceName("", archive); got != "" {
		t.Errorf("expected empty string for empty ID, got %q", got)
	}
	if got := timelineResolvePlaceName("place-unknown", archive); got != "place-unknown" {
		t.Errorf("expected raw ID for missing place, got %q", got)
	}
}

func TestPrintTimeline_FormatsISODates(t *testing.T) {
	entries := []timelineEntry{
		{Date: "1860-07-17", SortKey: "1860-07-17", Label: "Census", Detail: "Oakdale"},
		{Date: "ABT 1815", SortKey: "1815", Label: "Birth", Detail: "Virginia"},
		{Date: "", SortKey: "\xff", Label: "Undated Event"},
	}

	old := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	t.Cleanup(func() { r.Close() })
	os.Stdout = w

	printTimeline("person-test", "Test Person", entries)

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)
	output := buf.String()

	// ISO date should be formatted as readable
	assert.True(t, strings.Contains(output, "July 17, 1860"), "ISO date should render as readable: %s", output)
	// GEDCOM date should pass through
	assert.Contains(t, output, "ABT 1815")
	// Undated should show (no date)
	assert.Contains(t, output, "(no date)")
}
