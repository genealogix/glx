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

func TestParseFamilySearchARK_Accepts(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		wantNOID     string
		wantCanonURL string
	}{
		{
			name:         "full https with www",
			input:        "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
			wantNOID:     "1:1:C4H8-2DW2",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		},
		{
			name:         "full https without www",
			input:        "https://familysearch.org/ark:/61903/1:1:C4H8-2DW2",
			wantNOID:     "1:1:C4H8-2DW2",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		},
		{
			name:         "full http",
			input:        "http://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
			wantNOID:     "1:1:C4H8-2DW2",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		},
		{
			name:         "bare ark",
			input:        "ark:/61903/1:1:C4H8-2DW2",
			wantNOID:     "1:1:C4H8-2DW2",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		},
		{
			name:         "with surrounding whitespace",
			input:        "  ark:/61903/1:1:C4H8-2DW2  ",
			wantNOID:     "1:1:C4H8-2DW2",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		},
		{
			name:         "lowercase noid preserved",
			input:        "ark:/61903/1:1:abcd-efgh",
			wantNOID:     "1:1:abcd-efgh",
			wantCanonURL: "https://www.familysearch.org/ark:/61903/1:1:abcd-efgh",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFamilySearchARK(tc.input)
			if err != nil {
				t.Fatalf("ParseFamilySearchARK(%q) unexpected error: %v", tc.input, err)
			}
			if got.NOID != tc.wantNOID {
				t.Errorf("NOID: got %q, want %q", got.NOID, tc.wantNOID)
			}
			if got.CanonicalURL != tc.wantCanonURL {
				t.Errorf("CanonicalURL: got %q, want %q", got.CanonicalURL, tc.wantCanonURL)
			}
		})
	}
}

func TestParseFamilySearchARK_Rejects(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"wrong NAAN", "ark:/12345/1:1:C4H8-2DW2"},
		{"wrong domain", "https://www.ancestry.com/ark:/61903/1:1:C4H8-2DW2"},
		{"missing ark prefix", "https://www.familysearch.org/records/1:1:C4H8-2DW2"},
		{"missing noid", "ark:/61903/"},
		{"bogus chars in noid", "ark:/61903/1:1:C4H8 2DW2"},
		{"ftp scheme", "ftp://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2"},
		{"trailing slash", "ark:/61903/1:1:C4H8-2DW2/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseFamilySearchARK(tc.input); err == nil {
				t.Errorf("expected error for %q, got nil", tc.input)
			}
		})
	}
}

func TestCitationIDSlug(t *testing.T) {
	cases := []struct {
		noid string
		want string
	}{
		{"1:1:C4H8-2DW2", "c4h8-2dw2"},
		{"1:1:ABCD-EFGH", "abcd-efgh"},
		{"2:1:ZZZ-TEST", "2-1-zzz-test"},
		{"FOO", "foo"},
		{"1:1:Test-0001", "test-0001"},
	}

	for _, tc := range cases {
		t.Run(tc.noid, func(t *testing.T) {
			ark := &ARK{NOID: tc.noid}
			if got := ark.CitationIDSlug(); got != tc.want {
				t.Errorf("CitationIDSlug(%q) = %q, want %q", tc.noid, got, tc.want)
			}
		})
	}
}
