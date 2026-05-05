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
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// --- Gap Analysis ---

func TestAnalyzeGaps_MissingBirth(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "birth_event")
	if found == nil {
		t.Fatal("expected missing birth issue for person-a")
	}
	if found.Category != "gap" || found.Severity != "high" {
		t.Errorf("got category=%s severity=%s, want gap/high", found.Category, found.Severity)
	}
}

func TestAnalyzeGaps_HasBirth(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "birth_event")
	if found != nil {
		t.Error("should not flag missing birth when birth event exists")
	}
}

func TestAnalyzeGaps_MissingDeath(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "death_event")
	if found == nil {
		t.Fatal("expected missing death issue for person born in 1850")
	}
}

func TestAnalyzeGaps_MissingDeath_RecentBirth(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1990", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "death_event")
	if found != nil {
		t.Error("should not flag missing death for person born in 1990")
	}
}

func TestAnalyzeGaps_NoParents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
		Relationships: map[string]*glxlib.Relationship{},
	}

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-a", "no parents")
	if found == nil {
		t.Fatal("expected no-parents issue")
	}
}

func TestAnalyzeGaps_HasParents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
			"person-b": {Properties: map[string]any{}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-b", Role: "parent"},
					{Person: "person-a", Role: "child"},
				},
			},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-a", "no parents")
	if found != nil {
		t.Error("should not flag no-parents when parent relationship exists")
	}
}

func TestAnalyzeGaps_NoEvents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{},
	}

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-a", "no events")
	if found == nil {
		t.Fatal("expected no-events issue")
	}
}

func TestAnalyzeGaps_HasEvents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {
				Type:         "birth",
				Date:         "1850",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-a", "no events")
	if found != nil {
		t.Error("should not flag no-events when events exist")
	}
}

func TestAnalyzeGaps_NoMarriageEvent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
			"person-b": {Properties: map[string]any{}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-a", Role: "spouse"},
					{Person: "person-b", Role: "spouse"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type:         "birth",
				Date:         "1850",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-a", "no marriage event for")
	if found == nil {
		t.Fatal("expected no-marriage-event issue")
	}
	if !containsSubstring(found.Message, "person-b") {
		t.Error("expected spouse ID in message")
	}
}

func TestAnalyzeGaps_PerSpouseMarriageCheck(t *testing.T) {
	// Person with two spouses: one has marriage event, one doesn't
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary": {Properties: map[string]any{"name": "Mary"}},
			"person-dan":  {Properties: map[string]any{"name": "Daniel Lane"}},
			"person-john": {Properties: map[string]any{"name": "John Babcock"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-lane": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "spouse"},
					{Person: "person-dan", Role: "spouse"},
				},
			},
			"rel-babcock": {
				Type:       "marriage",
				StartEvent: "event-marriage-babcock",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "spouse"},
					{Person: "person-john", Role: "spouse"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-marriage-babcock": {
				Type: "marriage",
				Date: "1863-06-18",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "spouse"},
					{Person: "person-john", Role: "spouse"},
				},
			},
		},
	}

	issues := analyzeGaps(archive)

	// Should flag missing marriage event for Daniel Lane
	danIssue := findIssueByMessage(issues, "person-mary", "Daniel Lane")
	if danIssue == nil {
		t.Error("expected gap for missing Daniel Lane marriage event")
	}

	// Should NOT flag John Babcock (has marriage event)
	for _, issue := range issues {
		if issue.Person == "person-mary" && containsSubstring(issue.Message, "John Babcock") && containsSubstring(issue.Message, "no marriage event") {
			t.Error("should NOT flag John Babcock — marriage event exists")
		}
	}
}

// --- Evidence Analysis ---

func TestAnalyzeEvidence_UnsupportedAssertion(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {
				Subject: glxlib.EntityRef{Person: "person-a"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByEntity(issues, "assertion-1")
	if found == nil {
		t.Fatal("expected unsupported assertion issue")
	}
	if found.Category != "evidence" {
		t.Errorf("got category=%s, want evidence", found.Category)
	}
}

func TestAnalyzeEvidence_SupportedAssertion(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {
				Subject:   glxlib.EntityRef{Person: "person-a"},
				Citations: []string{"citation-1"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByEntity(issues, "assertion-1")
	if found != nil {
		t.Error("should not flag assertion with citation")
	}
}

func TestAnalyzeEvidence_MediaSupportedAssertion(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {
				Subject: glxlib.EntityRef{Person: "person-a"},
				Media:   []string{"media-1"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByEntity(issues, "assertion-1")
	if found != nil {
		t.Error("should not flag assertion with media evidence")
	}
}

func TestAnalyzeEvidence_OrphanedCitation(t *testing.T) {
	archive := &glxlib.GLXFile{
		Citations: map[string]*glxlib.Citation{
			"citation-1": {SourceID: "source-1"},
		},
		Assertions: map[string]*glxlib.Assertion{},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByEntity(issues, "citation-1")
	if found == nil {
		t.Fatal("expected orphaned citation issue")
	}
}

func TestAnalyzeEvidence_OrphanedSource(t *testing.T) {
	archive := &glxlib.GLXFile{
		Sources: map[string]*glxlib.Source{
			"source-1": {Title: "Test Source"},
		},
		Citations:  map[string]*glxlib.Citation{},
		Assertions: map[string]*glxlib.Assertion{},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByEntity(issues, "source-1")
	if found == nil {
		t.Fatal("expected orphaned source issue")
	}
}

func TestAnalyzeEvidence_SingleSourcePerson(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Sources: map[string]*glxlib.Source{
			"source-1": {Title: "Only Source"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject: glxlib.EntityRef{Person: "person-a"},
				Sources: []string{"source-1"},
			},
			"a-2": {
				Subject: glxlib.EntityRef{Person: "person-a"},
				Sources: []string{"source-1"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByMessage(issues, "person-a", "single source")
	if found == nil {
		t.Fatal("expected single-source person issue")
	}
}

// --- Consistency Analysis ---

func TestAnalyzeEvidence_UncitedNotes(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-uncited": {
				Subject:  glxlib.EntityRef{Person: "person-a"},
				Property: "notes_claim",
				Value:    "some value",
				Notes:    glxlib.NoteList{"County history biography noted 'one daughter married a Mr. Babcock'"},
			},
			"a-cited": {
				Subject:   glxlib.EntityRef{Event: "event-birth-a"},
				Property:  "date",
				Value:     "1832",
				Notes:     glxlib.NoteList{"Per 1880 census"},
				Citations: []string{"cit-1"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	found := findIssueByMessage(issues, "person-a", "notes reference a source")
	if found == nil {
		t.Fatal("expected uncited notes issue")
	}
	if found.Entity != "a-uncited" {
		t.Errorf("expected entity a-uncited, got %s", found.Entity)
	}
}

func TestAnalyzeEvidence_UncitedNotes_NoCitedFalsePositive(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:   glxlib.EntityRef{Event: "event-birth-a"},
				Property:  "date",
				Value:     "1832",
				Notes:     glxlib.NoteList{"Per 1880 census record"},
				Citations: []string{"cit-1"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	for _, issue := range issues {
		if containsSubstring(issue.Message, "notes reference a source") {
			t.Error("should not flag notes when citations exist")
		}
	}
}

func TestAnalyzeConsistency_BirthAfterDeath(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1920", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-a", "death year")
	if found == nil {
		t.Fatal("expected birth-after-death issue")
	}
	if found.Severity != "high" {
		t.Errorf("got severity=%s, want high", found.Severity)
	}
}

func TestAnalyzeConsistency_ValidDates(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1920", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-a", "death year")
	if found != nil {
		t.Error("should not flag valid date order")
	}
}

func TestAnalyzeConsistency_ImplausibleLifespan(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1920", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-a", "implausible lifespan")
	if found == nil {
		t.Fatal("expected implausible lifespan issue for 120-year span")
	}
}

func TestAnalyzeConsistency_ParentYoungerThanChild(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{}},
			"person-child":  {Properties: map[string]any{}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent": {Type: glxlib.EventTypeBirth, Date: "1880", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-child", "born after child")
	if found == nil {
		t.Fatal("expected parent-younger-than-child issue")
	}
}

func TestAnalyzeConsistency_EventAfterDeath(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-census": {
				Type:         "census",
				Date:         "1870",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-a", "after death")
	if found == nil {
		t.Fatal("expected event-after-death issue")
	}
}

func TestAnalyzeConsistency_BurialAfterDeath_OK(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-burial": {
				Type:         "burial",
				Date:         "1860",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-a", "after death")
	if found != nil {
		t.Error("burial after death should be allowed")
	}
}

// --- Conflict Analysis ---

func TestAnalyzeConflicts_DetectsConflicting(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary": {Properties: map[string]any{"name": "Mary Green"}},
		},
		Places: map[string]*glxlib.Place{
			"place-florida":  {Name: "Florida"},
			"place-virginia": {Name: "Virginia"},
			"place-new-york": {Name: "New York"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-mary"}, Property: "birthplace", Value: "place-florida", Confidence: "medium"},
			"a-2": {Subject: glxlib.EntityRef{Person: "person-mary"}, Property: "birthplace", Value: "place-virginia", Confidence: "medium"},
			"a-3": {Subject: glxlib.EntityRef{Person: "person-mary"}, Property: "birthplace", Value: "place-new-york", Confidence: "medium-high"},
		},
	}

	issues := analyzeConflicts(archive)
	found := findIssueByMessage(issues, "person-mary", "conflicting values")
	if found == nil {
		t.Fatal("expected conflict issue for birthplace")
	}
	if found.Severity != "high" {
		t.Errorf("expected high severity, got %s", found.Severity)
	}
	if !containsSubstring(found.Message, "3 conflicting values") {
		t.Errorf("expected 3 conflicting values in message: %s", found.Message)
	}
	if !containsSubstring(found.Message, "place-florida") {
		t.Errorf("expected place ID 'place-florida' in message: %s", found.Message)
	}
}

func TestAnalyzeConflicts_NoConflictWhenSameValue(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-a"}, Property: "birthplace", Value: "place-florida", Confidence: "medium"},
			"a-2": {Subject: glxlib.EntityRef{Person: "person-a"}, Property: "birthplace", Value: "place-florida", Confidence: "high"},
		},
	}

	issues := analyzeConflicts(archive)
	if len(issues) != 0 {
		t.Errorf("expected no conflicts when all values are the same, got %d", len(issues))
	}
}

// --- Duplicate Sibling Names ---

func TestAnalyzeConsistency_DuplicateSiblingNames(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "James Green"}},
			"person-mary-1": {Properties: map[string]any{"name": "Mary Green"}},
			"person-mary-2": {Properties: map[string]any{"name": "Mary Elizabeth Green"}},
			"person-john":   {Properties: map[string]any{"name": "John Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-mary-1", Role: "child"},
				},
			},
			"rel-2": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-mary-2", Role: "child"},
				},
			},
			"rel-3": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-john", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	issues := analyzeConsistency(archive)
	found := findIssueByMessage(issues, "person-parent", "share given name")
	if found == nil {
		t.Fatal("expected duplicate sibling name issue")
	}
	if !containsSubstring(found.Message, "Mary") {
		t.Errorf("expected capitalized 'Mary' in message: %s", found.Message)
	}
}

func TestAnalyzeConsistency_ReplacementChildNotFlagged(t *testing.T) {
	// First Mary died before second Mary was born — replacement pattern
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent"}},
			"person-mary-1": {Properties: map[string]any{"name": "Mary"}},
			"person-mary-2": {Properties: map[string]any{"name": "Mary"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-mary-1", Role: "child"}}},
			"rel-2": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-mary-2", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-mary1": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-mary-1", Role: "principal"}}},
			"event-death-mary1": {Type: glxlib.EventTypeDeath, Date: "1851", Participants: []glxlib.Participant{{Person: "person-mary-1", Role: "principal"}}},
			"event-birth-mary2": {Type: glxlib.EventTypeBirth, Date: "1853", Participants: []glxlib.Participant{{Person: "person-mary-2", Role: "principal"}}},
		},
	}

	issues := analyzeConsistency(archive)
	for _, issue := range issues {
		if containsSubstring(issue.Message, "share given name") {
			t.Error("should not flag replacement child pattern")
		}
	}
}

func TestAnalyzeConsistency_NoFalsePositiveOnUniqueNames(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent"}},
			"person-alice":  {Properties: map[string]any{"name": "Alice"}},
			"person-bob":    {Properties: map[string]any{"name": "Bob"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-alice", Role: "child"}}},
			"rel-2": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-bob", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{},
	}

	issues := analyzeConsistency(archive)
	for _, issue := range issues {
		if containsSubstring(issue.Message, "share given name") {
			t.Error("should not flag unique sibling names")
		}
	}
}

// --- Child Census Suggestions (brickwall research) ---

func TestSuggestChildCensus_PersonWithParentsNotBrickwall(t *testing.T) {
	// Mary has parents — should not be flagged as brickwall
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary":   {Properties: map[string]any{"name": "Mary Green"}},
			"person-joseph": {Properties: map[string]any{"name": "Joseph Green"}},
			"person-parent": {Properties: map[string]any{"name": "James Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-parent-joseph": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-joseph", Role: "child"},
				},
			},
			"rel-parent-mary": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-mary", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-mary":   {Type: glxlib.EventTypeBirth, Date: "ABT 1832", Participants: []glxlib.Participant{{Person: "person-mary", Role: "principal"}}},
			"event-birth-joseph": {Type: glxlib.EventTypeBirth, Date: "1835", Participants: []glxlib.Participant{{Person: "person-joseph", Role: "principal"}}},
		},
	}

	issues := suggestChildCensusRecords(archive)
	for _, issue := range issues {
		if issue.Person == "person-mary" {
			t.Error("should not suggest child census for person who has parents")
		}
	}
}

func TestSuggestChildCensus_OrphanWithNoChildren(t *testing.T) {
	// Mary has no parents and no children — no suggestions possible
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary":   {Properties: map[string]any{"name": "Mary Green"}},
			"person-joseph": {Properties: map[string]any{"name": "Joseph Green"}},
			"person-parent": {Properties: map[string]any{"name": "James Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-parent-joseph": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-joseph", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{},
	}

	issues := suggestChildCensusRecords(archive)
	for _, issue := range issues {
		if issue.Person == "person-mary" && containsSubstring(issue.Message, "Joseph") {
			t.Error("should not suggest child census when person has no children")
		}
	}
}

func TestSuggestChildCensus_BrickwallWithChildren(t *testing.T) {
	// James has no parents (brickwall) but has children Mary and Joseph
	// Should suggest searching children's 1880+ census records
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-james":  {Properties: map[string]any{"name": "James Green"}},
			"person-mary":   {Properties: map[string]any{"name": "Mary Green"}},
			"person-joseph": {Properties: map[string]any{"name": "Joseph Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-james", Role: "parent"},
					{Person: "person-mary", Role: "child"},
				},
			},
			"rel-2": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-james", Role: "parent"},
					{Person: "person-joseph", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-james":  {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-james", Role: "principal"}}},
			"event-birth-mary":   {Type: glxlib.EventTypeBirth, Date: "1832", Participants: []glxlib.Participant{{Person: "person-mary", Role: "principal"}}},
			"event-birth-joseph": {Type: glxlib.EventTypeBirth, Date: "1835", Participants: []glxlib.Participant{{Person: "person-joseph", Role: "principal"}}},
		},
	}

	issues := suggestChildCensusRecords(archive)

	found1880 := false
	for _, issue := range issues {
		if issue.Person == "person-james" && containsSubstring(issue.Message, "1880") {
			found1880 = true
		}
	}
	if !found1880 {
		t.Error("should suggest 1880 census for child of brickwall person")
	}
}

func TestSuggestChildCensus_PersonWithParentsNotFlagged(t *testing.T) {
	// person-child has parents — should never appear as the Person in suggestions
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-father": {Properties: map[string]any{"name": "Father"}},
			"person-child":  {Properties: map[string]any{"name": "Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-father", Role: "parent"}, {Person: "person-child", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-father": {Type: glxlib.EventTypeBirth, Date: "1830", Participants: []glxlib.Participant{{Person: "person-father", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := suggestChildCensusRecords(archive)
	for _, issue := range issues {
		if issue.Person == "person-child" {
			t.Error("should not flag person who has parents")
		}
	}
}

func TestSuggestChildCensus_ExistingCensusNotSuggested(t *testing.T) {
	// James is a brickwall with child Mary. Mary already has an 1880 census event.
	// Should NOT suggest 1880 for Mary.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-james": {Properties: map[string]any{"name": "James Green"}},
			"person-mary":  {Properties: map[string]any{"name": "Mary Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-james", Role: "parent"},
					{Person: "person-mary", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-james": {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-james", Role: "principal"}}},
			"event-birth-mary":  {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-mary", Role: "principal"}}},
			"event-1880-census": {
				Type: glxlib.EventTypeCensus,
				Date: "1880",
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: "subject"},
				},
			},
		},
	}

	issues := suggestChildCensusRecords(archive)
	for _, issue := range issues {
		if issue.Person == "person-james" && containsSubstring(issue.Message, "1880") && containsSubstring(issue.Message, "Mary") {
			t.Error("should NOT suggest 1880 census for Mary when event already exists")
		}
	}
}

func TestSuggestChildCensus_DeadChildNotSuggested(t *testing.T) {
	// James is brickwall, child Mary died in 1875 — should not suggest 1880+ for her
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-james": {Properties: map[string]any{"name": "James Green"}},
			"person-mary":  {Properties: map[string]any{"name": "Mary Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-james", Role: "parent"},
					{Person: "person-mary", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-james": {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-james", Role: "principal"}}},
			"event-birth-mary":  {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-mary", Role: "principal"}}},
			"event-death-mary":  {Type: glxlib.EventTypeDeath, Date: "1875", Participants: []glxlib.Participant{{Person: "person-mary", Role: "principal"}}},
		},
	}

	issues := suggestChildCensusRecords(archive)
	for _, issue := range issues {
		if issue.Person == "person-james" && containsSubstring(issue.Message, "Mary") {
			t.Errorf("should not suggest census for child who died before 1880: %s", issue.Message)
		}
	}
}

// --- Suggestion Analysis ---

func TestAnalyzeSuggestions_MissingCensus(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1890", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)
	found := findIssueByMessage(issues, "person-a", "1850 census")
	if found == nil {
		t.Fatal("expected suggestion to search 1850 census")
	}
	if found.Category != "suggestion" {
		t.Errorf("got category=%s, want suggestion", found.Category)
	}
}

func TestAnalyzeSuggestions_BEFDeathExcludesYear(t *testing.T) {
	// "BEF 1870" means died before 1870 — should NOT suggest 1870 census
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "BEF 1870", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)
	if found := findIssueByMessage(issues, "person-a", "1860 census"); found == nil {
		t.Error("expected suggestion for 1860 census (before BEF year)")
	}
	if found := findIssueByMessage(issues, "person-a", "1870 census"); found != nil {
		t.Error("should NOT suggest 1870 census (died BEF 1870)")
	}
}

func TestDeathYearUpperBound(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  int
	}{
		{"plain year", "1870", 1870},
		{"BEF prefix", "BEF 1870", 1869},
		{"bef lowercase", "bef 1870", 1869},
		{"ABT prefix", "ABT 1870", 1870},
		{"AFT prefix", "AFT 1860", 1860},
		{"empty string", "", 0},
		{"nil", nil, 0},
		{"structured map", map[string]any{"value": "BEF 1870"}, 1869},
		{"structured map plain", map[string]any{"value": "1870"}, 1870},
		{"temporal list", []any{map[string]any{"value": "BEF 1870"}}, 1869},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deathYearUpperBound(tt.input)
			if got != tt.want {
				t.Errorf("deathYearUpperBound(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestAnalyzeSuggestions_HasCensus(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-census-1850": {
				Type:         "census",
				Date:         "1850",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeSuggestions(archive)
	found := findIssueByMessage(issues, "person-a", "1850 census")
	if found != nil {
		t.Error("should not suggest 1850 census when one already exists")
	}
}

func TestAnalyzeSuggestions_CitationCoversCensus(t *testing.T) {
	// Census year covered via citation/source (no census event entity).
	// Analyze should NOT suggest searching for it.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1832", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1910", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
		Sources: map[string]*glxlib.Source{
			"source-1880-census": {
				Type:  glxlib.SourceTypeCensus,
				Title: "1880 United States Federal Census",
				Date:  "1880",
			},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-1880": {SourceID: "source-1880-census"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {
				Subject:   glxlib.EntityRef{Person: "person-a"},
				Property:  "residence",
				Value:     "some-place",
				Citations: []string{"citation-1880"},
			},
		},
	}

	issues := analyzeSuggestions(archive)
	if found := findIssueByMessage(issues, "person-a", "1880 census"); found != nil {
		t.Error("should NOT suggest 1880 census when covered by citation/source")
	}
	// But 1870 (not covered) should still be suggested
	if found := findIssueByMessage(issues, "person-a", "1870 census"); found == nil {
		t.Error("expected suggestion for 1870 census (not covered)")
	}
}

func TestAnalyzeSuggestions_CitationCoversViaTitleFallback(t *testing.T) {
	// Source.Date is empty; year is only in the title
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1832", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1910", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
		Sources: map[string]*glxlib.Source{
			"source-census": {
				Type:  glxlib.SourceTypeCensus,
				Title: "1880 United States Federal Census",
			},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-1": {SourceID: "source-census"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:   glxlib.EntityRef{Person: "person-a"},
				Property:  "residence",
				Value:     "place-x",
				Citations: []string{"cit-1"},
			},
		},
	}

	issues := analyzeSuggestions(archive)
	if found := findIssueByMessage(issues, "person-a", "1880 census"); found != nil {
		t.Error("should NOT suggest 1880 census when title mentions the year")
	}
}

func TestAnalyzeSuggestions_DirectSourceCoversCensus(t *testing.T) {
	// Census covered via direct source on assertion (no citation)
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1900", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
		Sources: map[string]*glxlib.Source{
			"src-1860": {
				Type: glxlib.SourceTypeCensus,
				Date: "1860",
			},
		},
		Citations: map[string]*glxlib.Citation{},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:  glxlib.EntityRef{Person: "person-a"},
				Property: "residence",
				Value:    "place-x",
				Sources:  []string{"src-1860"},
			},
		},
	}

	issues := analyzeSuggestions(archive)
	if found := findIssueByMessage(issues, "person-a", "1860 census"); found != nil {
		t.Error("should NOT suggest 1860 census when covered by direct source")
	}
}

func TestAnalyzeSuggestions_VitalRecords(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
		Sources:    map[string]*glxlib.Source{},
		Citations:  map[string]*glxlib.Citation{},
		Assertions: map[string]*glxlib.Assertion{},
	}

	issues := analyzeSuggestions(archive)
	found := findIssueByMessage(issues, "person-a", "vital records")
	if found == nil {
		t.Fatal("expected vital records suggestion")
	}
}

func TestAnalyzeSuggestions_MaxLifespanCap(t *testing.T) {
	// Person born 1832, no death date — should NOT suggest 1940+ census
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "ABT 1832", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	// Should suggest 1840-1930 (birth+100=1932, so 1930 is last valid)
	if found := findIssueByMessage(issues, "person-a", "1930 census"); found == nil {
		t.Error("expected suggestion for 1930 census")
	}
	if found := findIssueByMessage(issues, "person-a", "1940 census"); found != nil {
		t.Error("should NOT suggest 1940 census (beyond max lifespan)")
	}
	if found := findIssueByMessage(issues, "person-a", "1950 census"); found != nil {
		t.Error("should NOT suggest 1950 census (beyond max lifespan)")
	}
}

func TestAnalyzeSuggestions_BurialInfersDeath(t *testing.T) {
	// Person born 1832, no death event, but has burial event in 1863
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1832", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-burial": {
				Type:         "burial",
				Date:         "1863",
				Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}},
			},
		},
	}

	issues := analyzeSuggestions(archive)

	// Should suggest 1840-1860 but NOT 1870+
	if found := findIssueByMessage(issues, "person-a", "1860 census"); found == nil {
		t.Error("expected suggestion for 1860 census (before burial)")
	}
	if found := findIssueByMessage(issues, "person-a", "1870 census"); found != nil {
		t.Error("should NOT suggest 1870 census (after burial/inferred death)")
	}
}

func TestAnalyzeSuggestions_1890Note(t *testing.T) {
	// Person alive during 1890 should get a note about the destroyed census
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-a": {Type: glxlib.EventTypeBirth, Date: "1850", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
			"event-death-a": {Type: glxlib.EventTypeDeath, Date: "1920", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)
	found := findIssueByMessage(issues, "person-a", "1890 census")
	if found == nil {
		t.Fatal("expected 1890 census suggestion")
	}
	if !containsSubstring(found.Message, "destroyed") {
		t.Error("1890 census suggestion should note it was destroyed")
	}
}

// --- Suggestion Consolidation (parent + minor child same census year) ---

func TestAnalyzeSuggestions_ConsolidateParentAndMinorChild(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent Green"}},
			"person-child":  {Properties: map[string]any{"name": "Child Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-parent", Role: "parent"},
				{Person: "person-child", Role: "child"},
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-death-parent": {Type: glxlib.EventTypeDeath, Date: "1850", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1825", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	parent1830 := findIssueByMessage(issues, "person-parent", "1830 census")
	require.NotNil(t, parent1830, "expected parent to have a 1830 census suggestion")
	if !containsSubstring(parent1830.Message, "would also cover") {
		t.Errorf("expected parent 1830 message to contain consolidation note; got %q", parent1830.Message)
	}
	if !containsSubstring(parent1830.Message, "Child Green (~5)") {
		t.Errorf("expected parent 1830 message to mention child age ~5; got %q", parent1830.Message)
	}

	if child1830 := findIssueByMessage(issues, "person-child", "1830 census"); child1830 != nil {
		t.Errorf("child 1830 suggestion should be suppressed; got %q", child1830.Message)
	}
}

func TestAnalyzeSuggestions_ConsolidateAdultChildNotIncluded(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent Green"}},
			"person-child":  {Properties: map[string]any{"name": "Adult Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-parent", Role: "parent"},
				{Person: "person-child", Role: "child"},
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-death-parent": {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1828", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	parent1850 := findIssueByMessage(issues, "person-parent", "1850 census")
	require.NotNil(t, parent1850, "expected parent 1850 census suggestion")
	if containsSubstring(parent1850.Message, "would also cover") {
		t.Errorf("parent 1850 should NOT consolidate child (age 22); got %q", parent1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("expected adult child to keep independent 1850 census suggestion")
	}
}

func TestAnalyzeSuggestions_ConsolidateChildIndependentWhenParentHasCensus(t *testing.T) {
	// Parent already has the 1850 census, so produces no parent suggestion for 1850.
	// The minor child still needs its own 1850 suggestion (no parent suggestion to fold into).
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent Green"}},
			"person-child":  {Properties: map[string]any{"name": "Child Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-parent", Role: "parent"},
				{Person: "person-child", Role: "child"},
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-death-parent": {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1845", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1855", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-1850-parent": {
				Type: glxlib.EventTypeCensus, Date: "1850",
				Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}},
			},
		},
	}

	issues := analyzeSuggestions(archive)

	if parent1850 := findIssueByMessage(issues, "person-parent", "1850 census"); parent1850 != nil {
		t.Errorf("parent should NOT have 1850 suggestion when census event exists; got %q", parent1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child should keep independent 1850 census suggestion when parent already has the census")
	}
}

func TestAnalyzeSuggestions_ConsolidateMultipleMinorChildrenSorted(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent":      {Properties: map[string]any{"name": "Parent Green"}},
			"person-child-alpha": {Properties: map[string]any{"name": "Alpha Green"}},
			"person-child-beta":  {Properties: map[string]any{"name": "Beta Green"}},
			"person-child-gamma": {Properties: map[string]any{"name": "Gamma Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-a": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-child-alpha", Role: "child"}}},
			"rel-b": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-child-beta", Role: "child"}}},
			"rel-g": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-child-gamma", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent":      {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-death-parent":      {Type: glxlib.EventTypeDeath, Date: "1850", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-birth-child-alpha": {Type: glxlib.EventTypeBirth, Date: "1825", Participants: []glxlib.Participant{{Person: "person-child-alpha", Role: "principal"}}},
			"event-death-child-alpha": {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child-alpha", Role: "principal"}}},
			"event-birth-child-beta":  {Type: glxlib.EventTypeBirth, Date: "1826", Participants: []glxlib.Participant{{Person: "person-child-beta", Role: "principal"}}},
			"event-death-child-beta":  {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child-beta", Role: "principal"}}},
			"event-birth-child-gamma": {Type: glxlib.EventTypeBirth, Date: "1827", Participants: []glxlib.Participant{{Person: "person-child-gamma", Role: "principal"}}},
			"event-death-child-gamma": {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child-gamma", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	parent1830 := findIssueByMessage(issues, "person-parent", "1830 census")
	require.NotNil(t, parent1830, "expected parent 1830 census suggestion")
	want := "would also cover: Alpha Green (~5), Beta Green (~4), Gamma Green (~3)"
	if !containsSubstring(parent1830.Message, want) {
		t.Errorf("parent 1830 message missing or out-of-order children;\n got: %q\nwant substring: %q", parent1830.Message, want)
	}

	for _, childID := range []string{"person-child-alpha", "person-child-beta", "person-child-gamma"} {
		if found := findIssueByMessage(issues, childID, "1830 census"); found != nil {
			t.Errorf("child %s 1830 suggestion should be suppressed; got %q", childID, found.Message)
		}
	}
}

func TestAnalyzeSuggestions_ConsolidateChildBirthYearUnknown(t *testing.T) {
	// Child has no birth event, so neither participates in consolidation
	// nor produces independent census suggestions of its own.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {Properties: map[string]any{"name": "Parent Green"}},
			"person-child":  {Properties: map[string]any{"name": "Child Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{
				{Person: "person-parent", Role: "parent"},
				{Person: "person-child", Role: "child"},
			}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-parent": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
			"event-death-parent": {Type: glxlib.EventTypeDeath, Date: "1840", Participants: []glxlib.Participant{{Person: "person-parent", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	parent1830 := findIssueByMessage(issues, "person-parent", "1830 census")
	require.NotNil(t, parent1830, "expected parent 1830 census suggestion")
	if containsSubstring(parent1830.Message, "would also cover") {
		t.Errorf("parent 1830 should have no consolidation note when child birth year unknown; got %q", parent1830.Message)
	}
	if found := findIssueByMessage(issues, "person-child", "1830 census"); found != nil {
		t.Errorf("child without birth year should produce no census suggestion; got %q", found.Message)
	}
}

func TestAnalyzeSuggestions_ConsolidateBothParentsMissingSameYear(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-father": {Properties: map[string]any{"name": "Father Green"}},
			"person-mother": {Properties: map[string]any{"name": "Mother Green"}},
			"person-child":  {Properties: map[string]any{"name": "Child Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-f": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-father", Role: "parent"}, {Person: "person-child", Role: "child"}}},
			"rel-m": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-mother", Role: "parent"}, {Person: "person-child", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-father": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-father", Role: "principal"}}},
			"event-death-father": {Type: glxlib.EventTypeDeath, Date: "1850", Participants: []glxlib.Participant{{Person: "person-father", Role: "principal"}}},
			"event-birth-mother": {Type: glxlib.EventTypeBirth, Date: "1802", Participants: []glxlib.Participant{{Person: "person-mother", Role: "principal"}}},
			"event-death-mother": {Type: glxlib.EventTypeDeath, Date: "1850", Participants: []glxlib.Participant{{Person: "person-mother", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1825", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	father1830 := findIssueByMessage(issues, "person-father", "1830 census")
	require.NotNil(t, father1830, "expected father 1830 census suggestion")
	if !containsSubstring(father1830.Message, "Child Green (~5)") {
		t.Errorf("father 1830 message should mention child; got %q", father1830.Message)
	}

	mother1830 := findIssueByMessage(issues, "person-mother", "1830 census")
	require.NotNil(t, mother1830, "expected mother 1830 census suggestion")
	if !containsSubstring(mother1830.Message, "Child Green (~5)") {
		t.Errorf("mother 1830 message should mention child; got %q", mother1830.Message)
	}

	if found := findIssueByMessage(issues, "person-child", "1830 census"); found != nil {
		t.Errorf("child 1830 should be suppressed when either parent consolidates; got %q", found.Message)
	}
}

func TestAnalyzeSuggestions_ConsolidateStepParentNotYetMarried(t *testing.T) {
	// step_parent relationship begins at 1860 marriage event, so the
	// step-parent's 1830 census suggestion must not consolidate the child
	// (who was not yet living in their household).
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-stepfather": {Properties: map[string]any{"name": "Stepfather Green"}},
			"person-child":      {Properties: map[string]any{"name": "Child Green"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-step": {
				Type:       "step_parent",
				StartEvent: "event-marriage-1860",
				Participants: []glxlib.Participant{
					{Person: "person-stepfather", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-stepfather": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-stepfather", Role: "principal"}}},
			"event-death-stepfather": {Type: glxlib.EventTypeDeath, Date: "1870", Participants: []glxlib.Participant{{Person: "person-stepfather", Role: "principal"}}},
			"event-birth-child":      {Type: glxlib.EventTypeBirth, Date: "1825", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":      {Type: glxlib.EventTypeDeath, Date: "1832", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-marriage-1860":    {Type: glxlib.EventTypeMarriage, Date: "1860"},
		},
	}

	issues := analyzeSuggestions(archive)

	step1830 := findIssueByMessage(issues, "person-stepfather", "1830 census")
	require.NotNil(t, step1830, "expected stepfather 1830 census suggestion")
	if containsSubstring(step1830.Message, "would also cover") {
		t.Errorf("stepfather 1830 must not consolidate child before step relationship started; got %q", step1830.Message)
	}
	if child1830 := findIssueByMessage(issues, "person-child", "1830 census"); child1830 == nil {
		t.Error("child 1830 should be emitted independently when step-parent relationship had not yet started")
	}
}

func TestAnalyzeSuggestions_ConsolidateRelationshipEnded(t *testing.T) {
	// Foster parent-child relationship ended at 1840, so the foster parent's
	// 1850 census suggestion must not consolidate the (still-minor) child.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-foster": {Properties: map[string]any{"name": "Foster Parent"}},
			"person-child":  {Properties: map[string]any{"name": "Foster Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-foster": {
				Type:     "foster_parent_child",
				EndEvent: "event-foster-ended-1840",
				Participants: []glxlib.Participant{
					{Person: "person-foster", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-foster":      {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-death-foster":      {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-birth-child":       {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":       {Type: glxlib.EventTypeDeath, Date: "1855", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-foster-ended-1840": {Type: "custody_change", Date: "1840"},
		},
	}

	issues := analyzeSuggestions(archive)

	foster1850 := findIssueByMessage(issues, "person-foster", "1850 census")
	require.NotNil(t, foster1850, "expected foster 1850 census suggestion")
	if containsSubstring(foster1850.Message, "would also cover") {
		t.Errorf("foster 1850 must not consolidate child after relationship ended; got %q", foster1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently after foster relationship ended")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsViaStartedOnProperty(t *testing.T) {
	// Adoptive relationship's started_on property places the start at 1860,
	// so the 1830 suggestion for the adoptive parent must not consolidate
	// the child even though the relationship has no StartEvent.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-adoptive": {Properties: map[string]any{"name": "Adoptive Parent"}},
			"person-child":    {Properties: map[string]any{"name": "Adopted Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-adopt": {
				Type: "adoptive_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-adoptive", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
				Properties: map[string]any{"started_on": "1860"},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-adoptive": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-adoptive", Role: "principal"}}},
			"event-death-adoptive": {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-adoptive", Role: "principal"}}},
			"event-birth-child":    {Type: glxlib.EventTypeBirth, Date: "1825", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":    {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	adopt1830 := findIssueByMessage(issues, "person-adoptive", "1830 census")
	require.NotNil(t, adopt1830, "expected adoptive parent 1830 census suggestion")
	if containsSubstring(adopt1830.Message, "would also cover") {
		t.Errorf("adoptive parent 1830 must not consolidate child before adoption (started_on 1860); got %q", adopt1830.Message)
	}
	if child1830 := findIssueByMessage(issues, "person-child", "1830 census"); child1830 == nil {
		t.Error("child 1830 should be emitted independently before adoptive relationship started")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsViaEndedOnProperty(t *testing.T) {
	// Foster relationship's ended_on property places the end at 1840, so
	// the 1850 suggestion for the foster parent must not consolidate the
	// (still-minor) child even though the relationship has no EndEvent.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-foster": {Properties: map[string]any{"name": "Foster Parent"}},
			"person-child":  {Properties: map[string]any{"name": "Foster Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-foster": {
				Type: "foster_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-foster", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
				Properties: map[string]any{"ended_on": "1840"},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-foster": {Type: glxlib.EventTypeBirth, Date: "1800", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-death-foster": {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	foster1850 := findIssueByMessage(issues, "person-foster", "1850 census")
	require.NotNil(t, foster1850, "expected foster 1850 census suggestion")
	if containsSubstring(foster1850.Message, "would also cover") {
		t.Errorf("foster 1850 must not consolidate child after ended_on=1840; got %q", foster1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently after ended_on relationship boundary")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsAFTOnStart(t *testing.T) {
	// Step-parent relationship started "AFT 1850". The named year (1850) is
	// not in the active window — the relationship begins strictly after.
	// The step-parent's 1850 census must therefore not consolidate the
	// (still-minor) child.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-step":  {Properties: map[string]any{"name": "Step Parent"}},
			"person-child": {Properties: map[string]any{"name": "Step Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-step": {
				Type:       "step_parent_child",
				StartEvent: "event-step-married",
				Participants: []glxlib.Participant{
					{Person: "person-step", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-step":   {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-step", Role: "principal"}}},
			"event-death-step":   {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-step", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1900", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-step-married": {Type: "marriage", Date: "AFT 1850"},
		},
	}

	issues := analyzeSuggestions(archive)

	step1850 := findIssueByMessage(issues, "person-step", "1850 census")
	require.NotNil(t, step1850, "expected step parent 1850 census suggestion")
	if containsSubstring(step1850.Message, "would also cover") {
		t.Errorf("step parent 1850 must not consolidate child whose relationship starts AFT 1850; got %q", step1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently when step relationship starts AFT 1850")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsBEFOnEnd(t *testing.T) {
	// Foster relationship ended "BEF 1850". The named year (1850) is not in
	// the active window — the relationship had already ended. The foster
	// parent's 1850 census must therefore not consolidate the (still-minor)
	// child.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-foster": {Properties: map[string]any{"name": "Foster Parent"}},
			"person-child":  {Properties: map[string]any{"name": "Foster Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-foster": {
				Type:     "foster_parent_child",
				EndEvent: "event-foster-ended",
				Participants: []glxlib.Participant{
					{Person: "person-foster", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-foster": {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-death-foster": {Type: glxlib.EventTypeDeath, Date: "1890", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-foster-ended": {Type: "custody_change", Date: "BEF 1850"},
		},
	}

	issues := analyzeSuggestions(archive)

	foster1850 := findIssueByMessage(issues, "person-foster", "1850 census")
	require.NotNil(t, foster1850, "expected foster 1850 census suggestion")
	if containsSubstring(foster1850.Message, "would also cover") {
		t.Errorf("foster 1850 must not consolidate child whose relationship ended BEF 1850; got %q", foster1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently when foster relationship ended BEF 1850")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsAFTOnStartedOnProperty(t *testing.T) {
	// Same AFT-on-start case as the StartEvent variant, but driven by the
	// started_on property fallback. Ensures qualifier handling applies in
	// both the event and property paths.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-step":  {Properties: map[string]any{"name": "Step Parent"}},
			"person-child": {Properties: map[string]any{"name": "Step Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-step": {
				Type: "step_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-step", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
				Properties: map[string]any{"started_on": "AFT 1850"},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-step":  {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-step", Role: "principal"}}},
			"event-death-step":  {Type: glxlib.EventTypeDeath, Date: "1880", Participants: []glxlib.Participant{{Person: "person-step", Role: "principal"}}},
			"event-birth-child": {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child": {Type: glxlib.EventTypeDeath, Date: "1900", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	step1850 := findIssueByMessage(issues, "person-step", "1850 census")
	require.NotNil(t, step1850, "expected step parent 1850 census suggestion")
	if containsSubstring(step1850.Message, "would also cover") {
		t.Errorf("step parent 1850 must not consolidate when started_on=AFT 1850; got %q", step1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently when started_on AFT 1850")
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsAFTOnEnd(t *testing.T) {
	// Foster relationship ended "AFT 1840". The named year (1840) is still
	// in the active window (the relationship had not yet ended), but we
	// have no evidence about how long after — the window must therefore
	// close at 1840 conservatively, not extend open-ended into 1850. The
	// 1850 census must NOT consolidate (relationship not confirmed active
	// in 1850); the 1840 census, if both parent and child were missing it,
	// WOULD still consolidate.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-foster": {Properties: map[string]any{"name": "Foster Parent"}},
			"person-child":  {Properties: map[string]any{"name": "Foster Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-foster": {
				Type:     "foster_parent_child",
				EndEvent: "event-foster-ended",
				Participants: []glxlib.Participant{
					{Person: "person-foster", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-foster": {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-death-foster": {Type: glxlib.EventTypeDeath, Date: "1890", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1835", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-foster-ended": {Type: "custody_change", Date: "AFT 1840"},
		},
	}

	issues := analyzeSuggestions(archive)

	foster1850 := findIssueByMessage(issues, "person-foster", "1850 census")
	require.NotNil(t, foster1850, "expected foster 1850 census suggestion")
	if containsSubstring(foster1850.Message, "would also cover") {
		t.Errorf("foster 1850 must not consolidate when EndEvent=AFT 1840 (post-boundary year unconfirmed); got %q", foster1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently when foster relationship ended AFT 1840")
	}

	foster1840 := findIssueByMessage(issues, "person-foster", "1840 census")
	require.NotNil(t, foster1840, "expected foster 1840 census suggestion")
	if !containsSubstring(foster1840.Message, "would also cover") {
		t.Errorf("foster 1840 should consolidate child (named year of AFT is still active); got %q", foster1840.Message)
	}
}

func TestAnalyzeSuggestions_ConsolidateBoundsBEFOnEndedOnProperty(t *testing.T) {
	// Same BEF-on-end case as the EndEvent variant, but driven by the
	// ended_on property fallback. Ensures qualifier handling applies in
	// both the event and property paths.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-foster": {Properties: map[string]any{"name": "Foster Parent"}},
			"person-child":  {Properties: map[string]any{"name": "Foster Child"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-foster": {
				Type: "foster_parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-foster", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
				Properties: map[string]any{"ended_on": "BEF 1850"},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-foster": {Type: glxlib.EventTypeBirth, Date: "1810", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-death-foster": {Type: glxlib.EventTypeDeath, Date: "1890", Participants: []glxlib.Participant{{Person: "person-foster", Role: "principal"}}},
			"event-birth-child":  {Type: glxlib.EventTypeBirth, Date: "1840", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
			"event-death-child":  {Type: glxlib.EventTypeDeath, Date: "1860", Participants: []glxlib.Participant{{Person: "person-child", Role: "principal"}}},
		},
	}

	issues := analyzeSuggestions(archive)

	foster1850 := findIssueByMessage(issues, "person-foster", "1850 census")
	require.NotNil(t, foster1850, "expected foster 1850 census suggestion")
	if containsSubstring(foster1850.Message, "would also cover") {
		t.Errorf("foster 1850 must not consolidate when ended_on=BEF 1850; got %q", foster1850.Message)
	}
	if child1850 := findIssueByMessage(issues, "person-child", "1850 census"); child1850 == nil {
		t.Error("child 1850 should be emitted independently when ended_on BEF 1850")
	}
}

// --- Runner ---

func TestBuildSummary(t *testing.T) {
	issues := []AnalysisIssue{
		{Category: "gap"},
		{Category: "gap"},
		{Category: "evidence"},
		{Category: "consistency"},
		{Category: "suggestion"},
		{Category: "suggestion"},
		{Category: "suggestion"},
	}

	summary := buildSummary(issues)
	if summary["gap"] != 2 {
		t.Errorf("gap count = %d, want 2", summary["gap"])
	}
	if summary["evidence"] != 1 {
		t.Errorf("evidence count = %d, want 1", summary["evidence"])
	}
	if summary["consistency"] != 1 {
		t.Errorf("consistency count = %d, want 1", summary["consistency"])
	}
	if summary["suggestion"] != 3 {
		t.Errorf("suggestion count = %d, want 3", summary["suggestion"])
	}
}

func TestFilterByPerson_ExactID(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Alice"}},
			"person-b": {Properties: map[string]any{"name": "Bob"}},
		},
	}

	issues := []AnalysisIssue{
		{Person: "person-a", Message: "test-a"},
		{Person: "person-b", Message: "test-b"},
	}

	filtered := filterByPerson(issues, "person-a", archive)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(filtered))
	}
	if filtered[0].Person != "person-a" {
		t.Errorf("expected person-a, got %s", filtered[0].Person)
	}
}

func TestFilterByPerson_NameMatch(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Alice Smith"}},
			"person-b": {Properties: map[string]any{"name": "Bob Jones"}},
		},
	}

	issues := []AnalysisIssue{
		{Person: "person-a", Message: "test-a"},
		{Person: "person-b", Message: "test-b"},
	}

	filtered := filterByPerson(issues, "Alice", archive)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(filtered))
	}
	if filtered[0].Person != "person-a" {
		t.Errorf("expected person-a, got %s", filtered[0].Person)
	}
}

// --- Helpers ---

func findIssue(issues []AnalysisIssue, personID, property string) *AnalysisIssue {
	for i := range issues {
		if issues[i].Person == personID && issues[i].Property == property {
			return &issues[i]
		}
	}
	return nil
}

func findIssueByMessage(issues []AnalysisIssue, personID, msgSubstr string) *AnalysisIssue {
	for i := range issues {
		if issues[i].Person == personID && containsSubstring(issues[i].Message, msgSubstr) {
			return &issues[i]
		}
	}
	return nil
}

func findIssueByEntity(issues []AnalysisIssue, entityID string) *AnalysisIssue {
	for i := range issues {
		if issues[i].Entity == entityID {
			return &issues[i]
		}
	}
	return nil
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
