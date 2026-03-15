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

	"github.com/stretchr/testify/assert"
)

func TestGenerateEventTitle_IndividualWithDateAndName(t *testing.T) {
	title := GenerateEventTitle("birth", []string{"Robert Webb"}, "1815")
	assert.Equal(t, "Birth of Robert Webb (1815)", title)
}

func TestGenerateEventTitle_IndividualWithNameOnly(t *testing.T) {
	title := GenerateEventTitle("death", []string{"Jane Miller"}, "")
	assert.Equal(t, "Death of Jane Miller", title)
}

func TestGenerateEventTitle_IndividualWithDateOnly(t *testing.T) {
	title := GenerateEventTitle("burial", []string{""}, "1863")
	assert.Equal(t, "Burial (1863)", title)
}

func TestGenerateEventTitle_IndividualNoInfo(t *testing.T) {
	title := GenerateEventTitle("christening", nil, "")
	assert.Equal(t, "Christening", title)
}

func TestGenerateEventTitle_CoupleWithDate(t *testing.T) {
	title := GenerateEventTitle("marriage", []string{"Robert Webb", "Jane Miller"}, "ABT 1850")
	assert.Equal(t, "Marriage of Robert Webb and Jane Miller (1850)", title)
}

func TestGenerateEventTitle_CoupleNoDate(t *testing.T) {
	title := GenerateEventTitle("divorce", []string{"John Smith", "Jane Doe"}, "")
	assert.Equal(t, "Divorce of John Smith and Jane Doe", title)
}

func TestGenerateEventTitle_CoupleOneSpouseEmpty(t *testing.T) {
	title := GenerateEventTitle("marriage", []string{"Robert Webb", ""}, "1850")
	assert.Equal(t, "Marriage of Robert Webb (1850)", title)
}

func TestGenerateEventTitle_SnakeCaseFallback(t *testing.T) {
	title := GenerateEventTitle("military_service", []string{"Robert Webb"}, "1862")
	assert.Equal(t, "Military Service of Robert Webb (1862)", title)
}

func TestGenerateEventTitle_AllEventTypes(t *testing.T) {
	tests := []struct {
		eventType string
		wantLabel string
	}{
		{"birth", "Birth"},
		{"death", "Death"},
		{"marriage", "Marriage"},
		{"divorce", "Divorce"},
		{"burial", "Burial"},
		{"cremation", "Cremation"},
		{"baptism", "Baptism"},
		{"christening", "Christening"},
		{"confirmation", "Confirmation"},
		{"bar_mitzvah", "Bar Mitzvah"},
		{"bat_mitzvah", "Bat Mitzvah"},
		{"immigration", "Immigration"},
		{"emigration", "Emigration"},
		{"naturalization", "Naturalization"},
		{"census", "Census"},
		{"probate", "Probate"},
		{"will", "Will"},
		{"graduation", "Graduation"},
		{"retirement", "Retirement"},
		{"adoption", "Adoption"},
		{"engagement", "Engagement"},
		{"annulment", "Annulment"},
		{"residence", "Residence"},
		{"event", "Event"},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			title := GenerateEventTitle(tt.eventType, nil, "")
			assert.Equal(t, tt.wantLabel, title)
		})
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1850", "1850"},
		{"ABT 1850", "1850"},
		{"BEF 1920", "1920"},
		{"AFT 1880", "1880"},
		{"BET 1850 AND 1860", "1850"},
		{"15 MAR 1850", "1850"},
		{"800", "800"},
		{"476", "476"},
		{"ABT 476", "476"},
		{"BET 900 AND 1000", "900"},
		{"15 MAR 800", "800"},
		{"", ""},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, extractYear(DateString(tt.input)))
		})
	}
}

func TestPersonDisplayName(t *testing.T) {
	tests := []struct {
		name   string
		person *Person
		want   string
	}{
		{
			name:   "nil person",
			person: nil,
			want:   "",
		},
		{
			name:   "simple string name",
			person: &Person{Properties: map[string]any{"name": "John Smith"}},
			want:   "John Smith",
		},
		{
			name: "structured map name",
			person: &Person{Properties: map[string]any{
				"name": map[string]any{"value": "Jane Miller"},
			}},
			want: "Jane Miller",
		},
		{
			name: "temporal list name",
			person: &Person{Properties: map[string]any{
				"name": []any{
					map[string]any{"value": "Jane Webb"},
					map[string]any{"value": "Jane Miller"},
				},
			}},
			want: "Jane Webb",
		},
		{
			name:   "primary_name fallback",
			person: &Person{Properties: map[string]any{"primary_name": "Bob Clark"}},
			want:   "Bob Clark",
		},
		{
			name:   "no name",
			person: &Person{Properties: map[string]any{}},
			want:   "",
		},
		{
			name:   "nil properties",
			person: &Person{},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PersonDisplayName(tt.person))
		})
	}
}
