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
