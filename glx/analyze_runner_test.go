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
	found := findIssue(issues, "person-a", "born_on")
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "born_on")
	if found != nil {
		t.Error("should not flag missing birth when born_on is set")
	}
}

func TestAnalyzeGaps_MissingDeath(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "died_on")
	if found == nil {
		t.Fatal("expected missing death issue for person born in 1850")
	}
}

func TestAnalyzeGaps_MissingDeath_RecentBirth(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"born_on": "1990"}},
		},
	}

	issues := analyzeGaps(archive)
	found := findIssue(issues, "person-a", "died_on")
	if found != nil {
		t.Error("should not flag missing death for person born in 1990")
	}
}

func TestAnalyzeGaps_NoParents(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
			"person-b": {Properties: map[string]any{"born_on": "1820"}},
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
			"person-b": {Properties: map[string]any{"born_on": "1855"}},
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
			"person-mary":   {Properties: map[string]any{"name": "Mary", "born_on": "1832"}},
			"person-dan":    {Properties: map[string]any{"name": "Daniel Lane", "born_on": "1830"}},
			"person-john":   {Properties: map[string]any{"name": "John Babcock", "born_on": "1825"}},
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
				Notes:    "County history biography noted 'one daughter married a Mr. Babcock'",
			},
			"a-cited": {
				Subject:   glxlib.EntityRef{Person: "person-a"},
				Property:  "born_on",
				Value:     "1832",
				Notes:     "Per 1880 census",
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
				Subject:   glxlib.EntityRef{Person: "person-a"},
				Property:  "born_on",
				Value:     "1832",
				Notes:     "Per 1880 census record",
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
			"person-a": {Properties: map[string]any{
				"born_on": "1920",
				"died_on": "1850",
			}},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1850",
				"died_on": "1920",
			}},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1800",
				"died_on": "1920",
			}},
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
			"person-parent": {Properties: map[string]any{"born_on": "1880"}},
			"person-child":  {Properties: map[string]any{"born_on": "1850"}},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1800",
				"died_on": "1860",
			}},
		},
		Events: map[string]*glxlib.Event{
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
			"person-a": {Properties: map[string]any{
				"born_on": "1800",
				"died_on": "1860",
			}},
		},
		Events: map[string]*glxlib.Event{
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
			"person-mary-1": {Properties: map[string]any{"name": "Mary", "born_on": "1850", "died_on": "1851"}},
			"person-mary-2": {Properties: map[string]any{"name": "Mary", "born_on": "1853"}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-mary-1", Role: "child"}}},
			"rel-2": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-parent", Role: "parent"}, {Person: "person-mary-2", Role: "child"}}},
		},
		Events: map[string]*glxlib.Event{},
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

// --- Suggestion Analysis ---

func TestAnalyzeSuggestions_MissingCensus(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{
				"born_on": "1840",
				"died_on": "1890",
			}},
		},
		Events: map[string]*glxlib.Event{},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1840",
				"died_on": "BEF 1870",
			}},
		},
		Events: map[string]*glxlib.Event{},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1840",
				"died_on": "1860",
			}},
		},
		Events: map[string]*glxlib.Event{
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
			"person-a": {Properties: map[string]any{
				"born_on": "1832",
				"died_on": "1910",
			}},
		},
		Events:  map[string]*glxlib.Event{},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1832",
				"died_on": "1910",
			}},
		},
		Events: map[string]*glxlib.Event{},
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
			"person-a": {Properties: map[string]any{
				"born_on": "1840",
				"died_on": "1900",
			}},
		},
		Events: map[string]*glxlib.Event{},
		Sources: map[string]*glxlib.Source{
			"src-1860": {
				Type: glxlib.SourceTypeCensus,
				Date: "1860",
			},
		},
		Citations:  map[string]*glxlib.Citation{},
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
			"person-a": {Properties: map[string]any{"born_on": "1850"}},
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
			"person-a": {Properties: map[string]any{
				"born_on": "ABT 1832",
			}},
		},
		Events: map[string]*glxlib.Event{},
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
	// Person born 1832, no died_on, but has burial event in 1863
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{
				"born_on": "1832",
			}},
		},
		Events: map[string]*glxlib.Event{
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
			"person-a": {Properties: map[string]any{
				"born_on": "1850",
				"died_on": "1920",
			}},
		},
		Events: map[string]*glxlib.Event{},
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
