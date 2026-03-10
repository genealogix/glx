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
	}

	for _, tt := range tests {
		got := ExtractFirstYear(tt.input)
		if got != tt.want {
			t.Errorf("ExtractFirstYear(%q) = %d, want %d", tt.input, got, tt.want)
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
			got := ExtractPropertyYear(tt.props, tt.key)
			if got != tt.want {
				t.Errorf("ExtractPropertyYear() = %d, want %d", got, tt.want)
			}
		})
	}
}
