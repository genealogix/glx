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
)

func TestShortDateYears(t *testing.T) {
	cases := map[string]string{
		"":                          "",
		"unknown":                   "",
		"1850":                      "1850",
		"1850-03-22":                "1850",
		"0044 BCE":                  "44 BCE",
		"JULIAN 1750-02-10":         "1750",
		"ABT 1765":                  "c. 1765",
		"EST 1765":                  "est. 1765",
		"CAL 1765":                  "cal. 1765",
		"INT 1765 (about 1765)":     "1765",
		"BEF 1860":                  "bef. 1860",
		"AFT 1860-05":               "aft. 1860",
		"BET 1756 AND 1774":         "1756/1774",
		"BET 1826-05-16 AND 1830":   "1826/1830",
		"BET 1826-05-16 AND 1826":   "1826",
		"FROM 1850 TO 1860":         "1850/1860",
		"FROM 1850":                 "from 1850",
		"TO 1860":                   "to 1860",
		"BET 0050 BCE AND 0030 BCE": "50 BCE/30 BCE",
	}
	for in, want := range cases {
		if got := shortDateYears(in); got != want {
			t.Errorf("shortDateYears(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatLifeSpan(t *testing.T) {
	cases := []struct {
		birth, death, want string
	}{
		{"1850", "1920-04-01", "1850–1920"},
		{"BET 1756 AND 1774", "BET 1826-05-16 AND 1830", "1756/1774 – 1826/1830"},
		{"ABT 1765", "1830", "c. 1765 – 1830"},
		{"1765", "AFT 1830", "1765 – aft. 1830"},
		{"0044 BCE", "0010", "44 BCE – 10"},
		{"BET 1756 AND 1770", "", "b. 1756/1770"},
		{"", "BEF 1860", "d. bef. 1860"},
		{"unknown", "", ""},
	}
	for _, tc := range cases {
		if got := formatLifeSpan(tc.birth, tc.death); got != tc.want {
			t.Errorf("formatLifeSpan(%q, %q) = %q, want %q", tc.birth, tc.death, got, tc.want)
		}
	}
}
