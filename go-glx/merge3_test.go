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
	"reflect"
	"testing"
)

// =============================================================================
// Top-level invariants
// =============================================================================

func TestThreeWayMerge_NilInputsAreTreatedAsEmpty(t *testing.T) {
	merged, conflicts := ThreeWayMerge(nil, nil, nil)
	if merged == nil {
		t.Fatalf("merged file should never be nil")
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
	if merged.Persons != nil {
		t.Errorf("merged.Persons should be nil for empty inputs, got %v", merged.Persons)
	}
}

// TestThreeWayMerge_NilPointerInEntityMap pins down behavior on the
// nil-entity-pointer edge: a map of `{"p1": nil}` should be treated like an
// absent entry, not crash the merger. The earlier accessor-based design
// guarded this defensively at every field access; the simplified design
// guards it once per mergeOne* (post-DeepEqual nil → zero substitution).
// This test exists so a future refactor can't quietly un-do that.
func TestThreeWayMerge_NilPointerInEntityMap(t *testing.T) {
	base := &GLXFile{Persons: map[string]*Person{"p1": nil}}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	theirs := &GLXFile{Persons: map[string]*Person{"p1": nil}}

	// Must not panic. Exact merge semantics on this edge aren't part of the
	// contract; we just assert it terminates with a non-nil merged file.
	merged, _ := ThreeWayMerge(base, ours, theirs)
	if merged == nil {
		t.Fatalf("merged file should never be nil")
	}
}

func TestHasUnresolvedConflict(t *testing.T) {
	cases := []struct {
		name string
		in   []Merge3Conflict
		want bool
	}{
		{"nil", nil, false},
		{"empty", []Merge3Conflict{}, false},
		{"all auto-resolved", []Merge3Conflict{{AutoResolved: true}, {AutoResolved: true}}, false},
		{"one unresolved", []Merge3Conflict{{AutoResolved: true}, {AutoResolved: false}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasUnresolvedConflict(c.in); got != c.want {
				t.Errorf("HasUnresolvedConflict() = %v, want %v", got, c.want)
			}
		})
	}
}

// =============================================================================
// Entity add/remove cases
// =============================================================================

func TestThreeWayMerge_OneSidedAdd_Ours(t *testing.T) {
	base := &GLXFile{}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	theirs := &GLXFile{}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	if got, ok := merged.Persons["p1"]; !ok || got.Properties["name"] != "Alice" {
		t.Errorf("expected p1.name=Alice in merged, got %+v", merged.Persons)
	}
}

func TestThreeWayMerge_OneSidedAdd_Theirs(t *testing.T) {
	base := &GLXFile{}
	ours := &GLXFile{}
	theirs := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	if _, ok := merged.Persons["p1"]; !ok {
		t.Errorf("expected p1 in merged from theirs, got %v", merged.Persons)
	}
}

func TestThreeWayMerge_ConcurrentIdenticalAdd(t *testing.T) {
	base := &GLXFile{}
	add := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}

	merged, conflicts := ThreeWayMerge(base, add, add)
	if len(conflicts) != 0 {
		t.Fatalf("identical concurrent add must not conflict, got %v", conflicts)
	}
	if _, ok := merged.Persons["p1"]; !ok {
		t.Errorf("expected p1 in merged, got %v", merged.Persons)
	}
}

func TestThreeWayMerge_ConcurrentDivergingAdd(t *testing.T) {
	base := &GLXFile{}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	theirs := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Bob"}},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("expected unresolved conflict for diverging concurrent add, got %v", conflicts)
	}
}

func TestThreeWayMerge_OneSidedDelete_OtherUnchanged(t *testing.T) {
	p := &Person{Properties: map[string]any{"name": "Alice"}}
	base := &GLXFile{Persons: map[string]*Person{"p1": p}}
	ours := &GLXFile{Persons: map[string]*Person{"p1": p}}
	theirs := &GLXFile{}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("delete with no concurrent change must not conflict, got %v", conflicts)
	}
	if _, ok := merged.Persons["p1"]; ok {
		t.Errorf("expected p1 deleted in merged, got %v", merged.Persons)
	}
}

func TestThreeWayMerge_DeleteVsModify_Conflict(t *testing.T) {
	base := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1850"}},
	}}
	theirs := &GLXFile{}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("delete-vs-modify must conflict, got %v", conflicts)
	}
}

// =============================================================================
// Person property merge
// =============================================================================

func TestThreeWayMerge_PropertyChangedOnOursOnly(t *testing.T) {
	base := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1850"}},
	}}
	theirs := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", conflicts)
	}
	if got := merged.Persons["p1"].Properties["birthDate"]; got != "1850" {
		t.Errorf("expected birthDate=1850 from ours, got %v", got)
	}
}

func TestThreeWayMerge_PropertyChangedIdenticallyOnBothSides(t *testing.T) {
	base := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	side := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1850"}},
	}}

	_, conflicts := ThreeWayMerge(base, side, side)
	if len(conflicts) != 0 {
		t.Fatalf("identical change on both sides must not conflict, got %v", conflicts)
	}
}

func TestThreeWayMerge_PropertyChangedDifferently_Conflict(t *testing.T) {
	base := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1850"}},
	}}
	ours := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1851"}},
	}}
	theirs := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice", "birthDate": "1852"}},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging property edits must conflict, got %v", conflicts)
	}
	// Path should point at the property.
	found := false
	for _, c := range conflicts {
		if c.Path == "persons[p1].properties.birthDate" {
			found = true

			break
		}
	}
	if !found {
		t.Errorf("expected conflict path persons[p1].properties.birthDate, got %v", conflicts)
	}
}

// =============================================================================
// Assertion: value + confidence coupling
// =============================================================================

func TestThreeWayMerge_AssertionAdditiveCitations(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Citations: []string{"cit-a"}},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Citations: []string{"cit-a", "cit-b"}},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Citations: []string{"cit-a", "cit-c"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("additive citation merge must not conflict, got %v", conflicts)
	}
	got := merged.Assertions["a1"].Citations
	want := []string{"cit-a", "cit-b", "cit-c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("citations: got %v, want %v", got, want)
	}
}

func TestThreeWayMerge_AssertionAdditiveNotes(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Notes: NoteList{"original"}},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Notes: NoteList{"original", "ours-added"}},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Notes: NoteList{"original", "theirs-added"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("additive notes must not conflict, got %v", conflicts)
	}
	got := merged.Assertions["a1"].Notes
	want := NoteList{"original", "ours-added", "theirs-added"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("notes: got %v, want %v", got, want)
	}
}

func TestThreeWayMerge_AssertionConfidenceWins_OursHigher(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelLow},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-12", Confidence: ConfidenceLevelHigh},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-15", Confidence: ConfidenceLevelMedium},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if HasUnresolvedConflict(conflicts) {
		t.Fatalf("expected confidence-resolved (no unresolved), got %v", conflicts)
	}
	if got := merged.Assertions["a1"].Value; got != "1850-04-12" {
		t.Errorf("expected ours value (high confidence) to win, got %q", got)
	}
	if got := merged.Assertions["a1"].Confidence; got != ConfidenceLevelHigh {
		t.Errorf("expected confidence=high, got %q", got)
	}
	// One auto-resolved record should be present.
	if len(conflicts) == 0 || !conflicts[0].AutoResolved || conflicts[0].Resolution == "" {
		t.Errorf("expected one auto-resolved record with non-empty Resolution, got %v", conflicts)
	}
}

func TestThreeWayMerge_AssertionConfidenceWins_TheirsHigher(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelLow},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-12", Confidence: ConfidenceLevelMedium},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-15", Confidence: ConfidenceLevelHigh},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if HasUnresolvedConflict(conflicts) {
		t.Fatalf("expected confidence-resolved (no unresolved), got %v", conflicts)
	}
	if got := merged.Assertions["a1"].Value; got != "1850-04-15" {
		t.Errorf("expected theirs value (high confidence) to win, got %q", got)
	}
}

func TestThreeWayMerge_AssertionEqualConfidence_Conflict(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelMedium},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-12", Confidence: ConfidenceLevelMedium},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-15", Confidence: ConfidenceLevelMedium},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("equal-confidence diverging values must remain unresolved, got %v", conflicts)
	}
}

func TestThreeWayMerge_AssertionUnknownConfidence_Conflict(t *testing.T) {
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: "custom-rank-unknown"},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-12", Confidence: "custom-rank-unknown"},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850-04-15", Confidence: ConfidenceLevelHigh},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	// Even though theirs has known high confidence, ours has unknown — so neither
	// side wins on rank, and it remains unresolved. This guards against silent
	// data loss on custom confidence vocabularies the driver can't compare.
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("custom (unknown) confidence on one side must not auto-resolve, got %v", conflicts)
	}
}

func TestThreeWayMerge_AssertionConfidenceChangedAloneOnOursOnly(t *testing.T) {
	// Neither side changes Value; ours upgrades confidence; theirs unchanged.
	// Expectation: ours wins by standard 3-way scalar.
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelLow},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelHigh},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Value: "1850", Confidence: ConfidenceLevelLow},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("one-sided confidence change must not conflict, got %v", conflicts)
	}
	if got := merged.Assertions["a1"].Confidence; got != ConfidenceLevelHigh {
		t.Errorf("expected confidence=high (from ours), got %q", got)
	}
}

// =============================================================================
// Reference set merge primitive
// =============================================================================

func TestMerge3StringSet(t *testing.T) {
	cases := []struct {
		name               string
		base, ours, theirs []string
		want               []string
	}{
		{
			"nil base, ours adds, theirs adds disjoint",
			nil,
			[]string{"a", "b"},
			[]string{"c"},
			[]string{"a", "b", "c"},
		},
		{
			"both sides add the same entry",
			nil,
			[]string{"a"},
			[]string{"a"},
			[]string{"a"},
		},
		{
			"theirs adds, ours unchanged",
			[]string{"a"},
			[]string{"a"},
			[]string{"a", "b"},
			[]string{"a", "b"},
		},
		{
			"both sides remove different base entries",
			[]string{"a", "b", "c"},
			[]string{"a", "b"},
			[]string{"a", "c"},
			[]string{"a"},
		},
		{
			"one side removes, other adds",
			[]string{"a", "b"},
			[]string{"a"},
			[]string{"a", "b", "c"},
			[]string{"a", "c"},
		},
		{
			"all empty",
			nil, nil, nil, nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := merge3StringSet(c.base, c.ours, c.theirs)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// =============================================================================
// Place: pointer-to-scalar handling
// =============================================================================

func TestThreeWayMerge_PlaceLatLonOneSidedAdd(t *testing.T) {
	base := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield"},
	}}
	ours := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield", Latitude: new(39.7817)},
	}}
	theirs := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield"},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("one-sided lat add must not conflict, got %v", conflicts)
	}
	if got := merged.Places["pl1"].Latitude; got == nil || *got != 39.7817 {
		t.Errorf("expected lat=39.7817 from ours, got %v", got)
	}
}

func TestThreeWayMerge_PlaceLatConflict(t *testing.T) {
	base := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield", Latitude: new(0.0)},
	}}
	ours := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield", Latitude: new(39.7817)},
	}}
	theirs := &GLXFile{Places: map[string]*Place{
		"pl1": {Name: "Springfield", Latitude: new(40.0)},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging lat must conflict, got %v", conflicts)
	}
}

// =============================================================================
// Per-entity-type field merges (coverage for mergeOne* functions beyond
// Person, Place, Assertion which are already exercised above)
// =============================================================================

func TestThreeWayMerge_Event_FieldLevelMerge(t *testing.T) {
	// Title changes on ours only → ours wins.
	// PlaceID changes identically on both sides → no conflict.
	// Type changes differently on both sides → conflict, tagged with
	// EntityType=events so the runner can resolve assertion context.
	base := &GLXFile{Events: map[string]*Event{
		"ev1": {Title: "", Type: "birth", PlaceID: "pl-old"},
	}}
	ours := &GLXFile{Events: map[string]*Event{
		"ev1": {Title: "Birth of John", Type: "baptism", PlaceID: "pl-new"},
	}}
	theirs := &GLXFile{Events: map[string]*Event{
		"ev1": {Title: "", Type: "death", PlaceID: "pl-new"},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Fatalf("expected unresolved conflict on Type, got %v", conflicts)
	}
	if merged.Events["ev1"].Title != "Birth of John" {
		t.Errorf("expected Title=Birth of John (ours), got %q", merged.Events["ev1"].Title)
	}
	if merged.Events["ev1"].PlaceID != "pl-new" {
		t.Errorf("expected PlaceID=pl-new (both sides agreed), got %q", merged.Events["ev1"].PlaceID)
	}
	// Conflict should be tagged with the entity type — pin this so
	// the runner can route assertion context correctly (issue from PR review).
	if !hasConflictTaggedAs(conflicts, EntityTypeEvents, "ev1") {
		t.Errorf("expected at least one conflict tagged EntityType=events, EntityID=ev1, got %+v", conflicts)
	}
}

func TestThreeWayMerge_Relationship_FieldLevelMerge(t *testing.T) {
	base := &GLXFile{Relationships: map[string]*Relationship{
		"r1": {Type: "spouse", Participants: []Participant{{Person: "p1"}, {Person: "p2"}}},
	}}
	ours := &GLXFile{Relationships: map[string]*Relationship{
		"r1": {Type: "spouse", Participants: []Participant{{Person: "p1"}, {Person: "p2"}}, StartEvent: "ev-marriage"},
	}}
	theirs := &GLXFile{Relationships: map[string]*Relationship{
		"r1": {Type: "spouse", Participants: []Participant{{Person: "p1"}, {Person: "p2"}}, EndEvent: "ev-divorce"},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("disjoint event additions should not conflict, got %v", conflicts)
	}
	r := merged.Relationships["r1"]
	if r.StartEvent != "ev-marriage" || r.EndEvent != "ev-divorce" {
		t.Errorf("expected both StartEvent and EndEvent set, got %+v", r)
	}
}

func TestThreeWayMerge_Source_AuthorsAndMedia(t *testing.T) {
	// Authors is ordered (treated opaquely): one-sided edit takes.
	// Media is reference-list (additive set): both sides' adds union.
	base := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, J."}, Media: []string{"m-base"}},
	}}
	ours := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, J.", "Doe, A."}, Media: []string{"m-base", "m-ours"}},
	}}
	theirs := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, J."}, Media: []string{"m-base", "m-theirs"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("expected clean merge, got %v", conflicts)
	}
	s := merged.Sources["s1"]
	if len(s.Authors) != 2 || s.Authors[1] != "Doe, A." {
		t.Errorf("expected ours' Authors to win, got %v", s.Authors)
	}
	if !reflect.DeepEqual(s.Media, []string{"m-base", "m-ours", "m-theirs"}) {
		t.Errorf("Media should union, got %v", s.Media)
	}
}

func TestThreeWayMerge_Source_AuthorsConflict(t *testing.T) {
	base := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, J."}},
	}}
	ours := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, John"}},
	}}
	theirs := &GLXFile{Sources: map[string]*Source{
		"s1": {Title: "Census 1850", Authors: []string{"Smith, Jane"}},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging ordered author lists must conflict, got %v", conflicts)
	}
}

func TestThreeWayMerge_Citation_FieldLevelMerge(t *testing.T) {
	base := &GLXFile{Citations: map[string]*Citation{
		"c1": {SourceID: "s1", RepositoryID: "r1"},
	}}
	ours := &GLXFile{Citations: map[string]*Citation{
		"c1": {SourceID: "s1", RepositoryID: "r-better", Media: []string{"m-a"}},
	}}
	theirs := &GLXFile{Citations: map[string]*Citation{
		"c1": {SourceID: "s1", RepositoryID: "r1", Media: []string{"m-b"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("expected clean merge, got %v", conflicts)
	}
	c := merged.Citations["c1"]
	if c.RepositoryID != "r-better" {
		t.Errorf("expected RepositoryID=r-better from ours, got %q", c.RepositoryID)
	}
	if !reflect.DeepEqual(c.Media, []string{"m-a", "m-b"}) {
		t.Errorf("Media should union, got %v", c.Media)
	}
}

func TestThreeWayMerge_Repository_FieldLevelMerge(t *testing.T) {
	base := &GLXFile{Repositories: map[string]*Repository{
		"r1": {Name: "State Archives", City: "Old City"},
	}}
	ours := &GLXFile{Repositories: map[string]*Repository{
		"r1": {Name: "State Archives", City: "New City", Country: "USA"},
	}}
	theirs := &GLXFile{Repositories: map[string]*Repository{
		"r1": {Name: "State Archives", City: "Old City", Website: "https://example.org"},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("disjoint Repository edits should not conflict, got %v", conflicts)
	}
	r := merged.Repositories["r1"]
	if r.City != "New City" || r.Country != "USA" || r.Website != "https://example.org" {
		t.Errorf("expected all three additive fields preserved, got %+v", r)
	}
}

func TestThreeWayMerge_Media_FieldLevelMerge(t *testing.T) {
	base := &GLXFile{Media: map[string]*Media{
		"m1": {URI: "/a.jpg", Title: "Old title"},
	}}
	ours := &GLXFile{Media: map[string]*Media{
		"m1": {URI: "/a.jpg", Title: "New title"},
	}}
	theirs := &GLXFile{Media: map[string]*Media{
		"m1": {URI: "/a.jpg", Title: "Old title", Description: "Family photo"},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("disjoint Media edits should not conflict, got %v", conflicts)
	}
	m := merged.Media["m1"]
	if m.Title != "New title" || m.Description != "Family photo" {
		t.Errorf("expected both edits preserved, got %+v", m)
	}
}

// hasConflictTaggedAs reports whether the slice contains a conflict whose
// EntityType and EntityID match the given values. Used by per-entity tests to
// confirm conflicts produced by field-level helpers are tagged by the
// orchestrator (see tagConflicts in merge3.go).
func hasConflictTaggedAs(conflicts []Merge3Conflict, entityType, entityID string) bool {
	for i := range conflicts {
		if conflicts[i].EntityType == entityType && conflicts[i].EntityID == entityID {
			return true
		}
	}

	return false
}

// =============================================================================
// Nil-entity-pointer-in-map: Copilot-flagged edge from PR #906
// =============================================================================

func TestThreeWayMerge_NilEntityPointer_TreatedAsAbsent(t *testing.T) {
	// `ours` has `p1: nil` (e.g. from an odd YAML serialization).
	// Expectation: treated as if `p1` weren't there, so the merge proceeds
	// as base + theirs (no concurrent delete-vs-modify confusion), and the
	// merged output does NOT include `p1: null`.
	base := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}
	ours := &GLXFile{Persons: map[string]*Person{"p1": nil}}
	theirs := &GLXFile{Persons: map[string]*Person{
		"p1": {Properties: map[string]any{"name": "Alice"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	// ours' nil counts as "deleted", theirs unchanged from base → p1 should
	// be dropped from the merged output (delete on one side, no change on
	// the other side).
	if _, present := merged.Persons["p1"]; present {
		t.Errorf("p1 should be dropped (nil-as-absent + unchanged-theirs), got %+v", merged.Persons)
	}
	if len(conflicts) != 0 {
		t.Errorf("clean delete-vs-unchanged should not conflict, got %v", conflicts)
	}
}

// =============================================================================
// Field-level helpers: cover the diverging branches that mergeOne* tests
// don't otherwise exercise.
// =============================================================================

func TestThreeWayMerge_AssertionSubjectDivergence_Conflict(t *testing.T) {
	// EntityRef Subject is opaque-3-way. Both sides switch the subject to
	// different entity types → unresolved conflict.
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Event: "e-base"}, Property: "date", Value: "1850"},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Person: "p-john"}, Property: "date", Value: "1850"},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Event: "e-better"}, Property: "date", Value: "1850"},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging Subject must conflict, got %v", conflicts)
	}
}

func TestThreeWayMerge_Event_ParticipantsConflict(t *testing.T) {
	// []Participant is treated opaquely. Both sides change the list
	// differently → conflict.
	base := &GLXFile{Events: map[string]*Event{
		"ev1": {Type: "marriage", Participants: []Participant{{Person: "p1"}}},
	}}
	ours := &GLXFile{Events: map[string]*Event{
		"ev1": {Type: "marriage", Participants: []Participant{{Person: "p1"}, {Person: "p2"}}},
	}}
	theirs := &GLXFile{Events: map[string]*Event{
		"ev1": {Type: "marriage", Participants: []Participant{{Person: "p1"}, {Person: "p3"}}},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging Participants must conflict, got %v", conflicts)
	}
}

func TestThreeWayMerge_Assertion_ParticipantPtrChange(t *testing.T) {
	// *Participant on Assertion, only ours sets it (from nil base) → take ours.
	base := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Event: "e1"}},
	}}
	ours := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Event: "e1"}, Participant: &Participant{Person: "p1", Role: "primary"}},
	}}
	theirs := &GLXFile{Assertions: map[string]*Assertion{
		"a1": {Subject: EntityRef{Event: "e1"}},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("one-sided Participant add must not conflict, got %v", conflicts)
	}
	if merged.Assertions["a1"].Participant == nil || merged.Assertions["a1"].Participant.Person != "p1" {
		t.Errorf("expected Participant from ours, got %+v", merged.Assertions["a1"].Participant)
	}
}

// =============================================================================
// Opaque (vocabulary) merge: all-three-present diverging case
// =============================================================================

func TestThreeWayMerge_Vocabulary_BothSidesDivergeConflict(t *testing.T) {
	// All three present, both sides change a vocabulary entry differently
	// → opaqueAllPresent emits a conflict.
	r1 := 1
	r2 := 2
	r3 := 3

	base := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium": {Label: "Medium", Rank: &r1},
	}}
	ours := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium": {Label: "Medium-Ours", Rank: &r2},
	}}
	theirs := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium": {Label: "Medium-Theirs", Rank: &r3},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging vocab definitions must conflict, got %v", conflicts)
	}
}

func TestThreeWayMerge_Vocabulary_DeleteVsModify(t *testing.T) {
	// Ours deletes a vocab entry; theirs modifies it. Conflict.
	r1 := 1
	r2 := 2
	base := &GLXFile{EventTypes: map[string]*VocabularyEntry{
		"baptism": {Label: "Baptism", Rank: &r1},
	}}
	ours := &GLXFile{}
	theirs := &GLXFile{EventTypes: map[string]*VocabularyEntry{
		"baptism": {Label: "Christening", Rank: &r2},
	}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("delete-vs-modify on vocab must conflict, got %v", conflicts)
	}
}

// =============================================================================
// Top-level metadata
// =============================================================================

func TestThreeWayMerge_MetadataDivergesOnBothSides_Conflict(t *testing.T) {
	base := &GLXFile{ImportMetadata: &Metadata{Copyright: ""}}
	ours := &GLXFile{ImportMetadata: &Metadata{Copyright: "(c) Alice"}}
	theirs := &GLXFile{ImportMetadata: &Metadata{Copyright: "(c) Bob"}}

	_, conflicts := ThreeWayMerge(base, ours, theirs)
	if !HasUnresolvedConflict(conflicts) {
		t.Errorf("diverging metadata must conflict, got %v", conflicts)
	}
}

// =============================================================================
// Vocabulary opaque merge
// =============================================================================

func TestThreeWayMerge_VocabularyOneSidedChange(t *testing.T) {
	rank2 := 2
	rank3 := 3

	base := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium": {Label: "Medium", Rank: &rank2},
	}}
	ours := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium":      {Label: "Medium", Rank: &rank2},
		"medium-high": {Label: "Medium-High", Rank: &rank3},
	}}
	theirs := &GLXFile{ConfidenceLevels: map[string]*VocabularyEntry{
		"medium": {Label: "Medium", Rank: &rank2},
	}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	if len(conflicts) != 0 {
		t.Fatalf("one-sided vocab add must not conflict, got %v", conflicts)
	}
	if _, ok := merged.ConfidenceLevels["medium-high"]; !ok {
		t.Errorf("expected medium-high added from ours, got %v", merged.ConfidenceLevels)
	}
}
