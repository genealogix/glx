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
	"strings"
	"testing"
)

func TestExtractFirstYear(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"1850", 1850},
		{"1850-01-15", 1850},
		{"ABT 1880", 1880},
		{"BEF 1920-01-15", 1920},
		{"AFT 1900", 1900},
		{"BET 1880 AND 1890", 1880},
		{"", 0},
		{"unknown", 0},
		{"800", 800},
		{"476", 476},
		{"ABT 476", 476},
		{"BET 900 AND 1000", 900},
		{"15 MAR 800", 800},
		{"5 JAN 476", 476},
	}

	for _, tt := range tests {
		got := extractFirstYear(tt.input)
		if got != tt.want {
			t.Errorf("extractFirstYear(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestExtractPropertyYear(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]any
		key   string
		want  int
	}{
		{
			name:  "simple string",
			props: map[string]any{"born_on": "1850"},
			key:   "born_on",
			want:  1850,
		},
		{
			name:  "structured map",
			props: map[string]any{"born_on": map[string]any{"value": "ABT 1832"}},
			key:   "born_on",
			want:  1832,
		},
		{
			name: "temporal list",
			props: map[string]any{"born_on": []any{
				map[string]any{"value": "1850-03-15"},
			}},
			key:  "born_on",
			want: 1850,
		},
		{
			name:  "missing property",
			props: map[string]any{},
			key:   "born_on",
			want:  0,
		},
		{
			name:  "empty string",
			props: map[string]any{"born_on": ""},
			key:   "born_on",
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPropertyYear(tt.props, tt.key)
			if got != tt.want {
				t.Errorf("extractPropertyYear() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidateDeathBeforeBirth(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-ok": {
				Properties: map[string]any{
					"born_on": "1850",
					"died_on": "1920",
				},
			},
			"person-bad": {
				Properties: map[string]any{
					"born_on": "1920",
					"died_on": "1850",
				},
			},
			"person-no-dates": {
				Properties: map[string]any{},
			},
			"person-birth-only": {
				Properties: map[string]any{
					"born_on": "1860",
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateDeathBeforeBirth(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}

	w := result.Warnings[0]
	if w.SourceID != "person-bad" {
		t.Errorf("expected warning for person-bad, got %s", w.SourceID)
	}
	if !strings.Contains(w.Message, "1850") || !strings.Contains(w.Message, "1920") {
		t.Errorf("warning should mention both years, got: %s", w.Message)
	}
}

func TestValidateDeathBeforeBirth_SameYear(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-same-year": {
				Properties: map[string]any{
					"born_on": "1850",
					"died_on": "1850",
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateDeathBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for same year, got %d", len(result.Warnings))
	}
}

func TestValidateParentChildAges(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-parent": {Properties: map[string]any{"born_on": "1820"}},
			"person-child":  {Properties: map[string]any{"born_on": "1850"}},
		},
		Relationships: map[string]*Relationship{
			"rel-ok": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateParentChildAges(result)

	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for valid parent-child, got %d", len(result.Warnings))
	}
}

func TestValidateParentChildAges_ParentYoungerThanChild(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-parent": {Properties: map[string]any{"born_on": "1860"}},
			"person-child":  {Properties: map[string]any{"born_on": "1850"}},
		},
		Relationships: map[string]*Relationship{
			"rel-bad": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateParentChildAges(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}

	w := result.Warnings[0]
	if !strings.Contains(w.Message, "person-parent") || !strings.Contains(w.Message, "person-child") {
		t.Errorf("warning should mention both IDs, got: %s", w.Message)
	}
}

func TestValidateParentChildAges_AdoptiveRelationship(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-parent": {Properties: map[string]any{"born_on": "1860"}},
			"person-child":  {Properties: map[string]any{"born_on": "1850"}},
		},
		Relationships: map[string]*Relationship{
			"rel-bad": {
				Type: "adoptive_parent_child",
				Participants: []Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateParentChildAges(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning for adoptive parent-child too, got %d", len(result.Warnings))
	}
}

func TestValidateParentChildAges_SkipsNonParentChild(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"born_on": "1860"}},
			"person-b": {Properties: map[string]any{"born_on": "1850"}},
		},
		Relationships: map[string]*Relationship{
			"rel-marriage": {
				Type: "marriage",
				Participants: []Participant{
					{Person: "person-a", Role: "spouse"},
					{Person: "person-b", Role: "spouse"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateParentChildAges(result)

	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for marriage, got %d", len(result.Warnings))
	}
}

func TestValidateMarriageBeforeBirth(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-groom": {Properties: map[string]any{"born_on": "1850"}},
			"person-bride": {Properties: map[string]any{"born_on": "1855"}},
		},
		Events: map[string]*Event{
			"event-ok": {
				Type: "marriage",
				Date: "1875",
				Participants: []Participant{
					{Person: "person-groom", Role: "groom"},
					{Person: "person-bride", Role: "bride"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateMarriageBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for valid marriage, got %d", len(result.Warnings))
	}
}

func TestValidateMarriageBeforeBirth_MarriageBeforeParticipantBorn(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-groom": {Properties: map[string]any{"born_on": "1900"}},
			"person-bride": {Properties: map[string]any{"born_on": "1855"}},
		},
		Events: map[string]*Event{
			"event-bad": {
				Type: "marriage",
				Date: "1875",
				Participants: []Participant{
					{Person: "person-groom", Role: "groom"},
					{Person: "person-bride", Role: "bride"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateMarriageBeforeBirth(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}

	w := result.Warnings[0]
	if !strings.Contains(w.Message, "person-groom") {
		t.Errorf("warning should mention person-groom, got: %s", w.Message)
	}
}

func TestValidateMarriageBeforeBirth_SkipsNonMarriage(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"born_on": "1900"}},
		},
		Events: map[string]*Event{
			"event-census": {
				Type: "census",
				Date: "1860",
				Participants: []Participant{
					{Person: "person-a", Role: "subject"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateMarriageBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for non-marriage event, got %d", len(result.Warnings))
	}
}

func TestValidateTemporalConsistency_Integration(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{
				"born_on": "1920",
				"died_on": "1850",
			}},
			"person-b":      {Properties: map[string]any{"born_on": "1860"}},
			"person-parent": {Properties: map[string]any{"born_on": "1870"}},
			"person-child":  {Properties: map[string]any{"born_on": "1850"}},
		},
		Events: map[string]*Event{
			"event-marriage": {
				Type: "marriage",
				Date: "1840",
				Participants: []Participant{
					{Person: "person-a", Role: "groom"},
					{Person: "person-b", Role: "bride"},
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-pc": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
		},
	}

	result := &ValidationResult{}
	glxFile.validateTemporalConsistency(result)

	// Should find 3 warnings:
	// 1. person-a: death (1850) before birth (1920)
	// 2. rel-pc: parent (1870) born after child (1850)
	// 3. event-marriage: marriage (1840) before person-a birth (1920)
	// Note: person-b born 1860, marriage 1840 → that's also before birth = 4th warning
	if len(result.Warnings) < 3 {
		t.Errorf("expected at least 3 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}
