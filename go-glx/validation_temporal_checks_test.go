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
		{"800", 800},
		{"476", 476},
		{"ABT 476", 476},
		{"BET 900 AND 1000", 900},
		{"15 MAR 800", 800},
		{"5 JAN 476", 476},
		// Non-Gregorian calendar dates — year is last. Fixes #565.
		{"HEBREW 15 TSH 5765", 5765},
		{"FRENCH_R 1 VEND 0012", 12},
		{"HEBREW ABT 5765", 5765},
		{"JULIAN 1731-03-15", 1731},
		{"JULIAN ABT 1731", 1731},
		// Range formats — should return start year, not end year
		{"HEBREW BET 15 TSH 5765 AND 15 TSH 5766", 5765},
		{"FRENCH_R FROM 1 VEND 0010 TO 1 VEND 0012", 10},
	}

	for _, tt := range tests {
		got := ExtractFirstYear(tt.input)
		if got != tt.want {
			t.Errorf("ExtractFirstYear(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

