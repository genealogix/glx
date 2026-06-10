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

func TestGenerateEventTitle_IndividualWithName(t *testing.T) {
	title := GenerateEventTitle("birth", []string{"Robert Webb"})
	assert.Equal(t, "Birth of Robert Webb", title)
}

func TestGenerateEventTitle_IndividualWithNameOnly(t *testing.T) {
	title := GenerateEventTitle("death", []string{"Jane Miller"})
	assert.Equal(t, "Death of Jane Miller", title)
}

func TestGenerateEventTitle_IndividualNoName(t *testing.T) {
	title := GenerateEventTitle("burial", []string{""})
	assert.Equal(t, "Burial", title)
}

func TestGenerateEventTitle_IndividualNoInfo(t *testing.T) {
	title := GenerateEventTitle("christening", nil)
	assert.Equal(t, "Christening", title)
}

func TestGenerateEventTitle_Couple(t *testing.T) {
	title := GenerateEventTitle("marriage", []string{"Robert Webb", "Jane Miller"})
	assert.Equal(t, "Marriage of Robert Webb and Jane Miller", title)
}

func TestGenerateEventTitle_CoupleDivorce(t *testing.T) {
	title := GenerateEventTitle("divorce", []string{"John Smith", "Jane Doe"})
	assert.Equal(t, "Divorce of John Smith and Jane Doe", title)
}

func TestGenerateEventTitle_CoupleOneSpouseEmpty(t *testing.T) {
	title := GenerateEventTitle("marriage", []string{"Robert Webb", ""})
	assert.Equal(t, "Marriage of Robert Webb", title)
}

func TestGenerateEventTitle_SnakeCaseFallback(t *testing.T) {
	title := GenerateEventTitle("military_service", []string{"Robert Webb"})
	assert.Equal(t, "Military Service of Robert Webb", title)
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
			title := GenerateEventTitle(tt.eventType, nil)
			assert.Equal(t, tt.wantLabel, title)
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

func TestStripStaleEventTitleYear(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		eventType string
		names     []string
		want      string
	}{
		// Stale suffixes that must be stripped.
		{
			name:      "redundant year on individual event",
			title:     "Birth of John Smith (1850)",
			eventType: "birth",
			names:     []string{"John Smith"},
			want:      "Birth of John Smith",
		},
		{
			name:      "buggy day-as-year suffix",
			title:     "Birth of John Smith (15)",
			eventType: "birth",
			names:     []string{"John Smith"},
			want:      "Birth of John Smith",
		},
		{
			name:      "couple event",
			title:     "Marriage of Robert Webb and Jane Miller (1850)",
			eventType: "marriage",
			names:     []string{"Robert Webb", "Jane Miller"},
			want:      "Marriage of Robert Webb and Jane Miller",
		},
		{
			name:      "bare label with no participant names",
			title:     "Burial (1863)",
			eventType: "burial",
			names:     nil,
			want:      "Burial",
		},
		{
			name:      "bare label even when names are known",
			title:     "Burial (1863)",
			eventType: "burial",
			names:     []string{"John Smith"},
			want:      "Burial",
		},
		{
			name:      "snake_case fallback label",
			title:     "Military Service of Robert Webb (1850)",
			eventType: "military_service",
			names:     []string{"Robert Webb"},
			want:      "Military Service of Robert Webb",
		},
		{
			name:      "empty names are skipped like GenerateEventTitle does",
			title:     "Marriage of Jane Miller (1850)",
			eventType: "marriage",
			names:     []string{"", "Jane Miller"},
			want:      "Marriage of Jane Miller",
		},
		// Titles that must be left untouched.
		{
			name:      "no trailing parenthetical",
			title:     "1860 Census — Webb Household",
			eventType: "census",
			names:     []string{"John Smith"},
			want:      "1860 Census — Webb Household",
		},
		{
			name:      "custom title with trailing year fails the gate",
			title:     "Family reunion (1955)",
			eventType: "event",
			names:     []string{"John Smith"},
			want:      "Family reunion (1955)",
		},
		{
			name:      "non-digit parenthetical",
			title:     "Census (Webb Household)",
			eventType: "census",
			names:     []string{"John Smith"},
			want:      "Census (Webb Household)",
		},
		{
			name:      "more than four digits",
			title:     "Reunion (10543)",
			eventType: "event",
			names:     nil,
			want:      "Reunion (10543)",
		},
		{
			name:      "participant name mismatch fails the gate",
			title:     "Birth of John Smith (1850)",
			eventType: "birth",
			names:     []string{"Jane Doe"},
			want:      "Birth of John Smith (1850)",
		},
		{
			name:      "event type mismatch fails the gate",
			title:     "Birth of John Smith (1850)",
			eventType: "death",
			names:     []string{"John Smith"},
			want:      "Birth of John Smith (1850)",
		},
		{
			name:      "of-shape with no participant names fails the gate",
			title:     "Birth of John Smith (1850)",
			eventType: "birth",
			names:     nil,
			want:      "Birth of John Smith (1850)",
		},
		{
			name:      "no space before the parenthetical",
			title:     "Birth of John Smith(1850)",
			eventType: "birth",
			names:     []string{"John Smith"},
			want:      "Birth of John Smith(1850)",
		},
		{
			name:      "empty title",
			title:     "",
			eventType: "birth",
			names:     []string{"John Smith"},
			want:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripStaleEventTitleYear(tt.title, tt.eventType, tt.names)
			assert.Equal(t, tt.want, got)
			// Stripping is idempotent: a cleaned title no longer matches the gate.
			assert.Equal(t, got, StripStaleEventTitleYear(got, tt.eventType, tt.names))
		})
	}
}
