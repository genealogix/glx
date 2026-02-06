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

package lib

import (
	"testing"
)

// NOTE: GEDCOM import tests are now in gedcom_comprehensive_test.go
// which tests all 35 GEDCOM test files

// TestParseGEDCOMLine tests the GEDCOM line parser
func TestParseGEDCOMLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantLevel   int
		wantXRef    string
		wantTag     string
		wantValue   string
		expectError bool
	}{
		{
			name:      "level 0 with xref",
			line:      "0 @I1@ INDI",
			wantLevel: 0,
			wantXRef:  "@I1@",
			wantTag:   "INDI",
			wantValue: "",
		},
		{
			name:      "level 1 with value",
			line:      "1 NAME John /Smith/",
			wantLevel: 1,
			wantXRef:  "",
			wantTag:   "NAME",
			wantValue: "John /Smith/",
		},
		{
			name:      "level 2 with value",
			line:      "2 GIVN John",
			wantLevel: 2,
			wantXRef:  "",
			wantTag:   "GIVN",
			wantValue: "John",
		},
		{
			name:      "level 0 header",
			line:      "0 HEAD",
			wantLevel: 0,
			wantXRef:  "",
			wantTag:   "HEAD",
			wantValue: "",
		},
		{
			name:        "invalid - no level",
			line:        "INVALID",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGEDCOMLine(tt.line, 1)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if got.Level != tt.wantLevel {
				t.Errorf("Level = %d, want %d", got.Level, tt.wantLevel)
			}
			if got.XRef != tt.wantXRef {
				t.Errorf("XRef = %q, want %q", got.XRef, tt.wantXRef)
			}
			if got.Tag != tt.wantTag {
				t.Errorf("Tag = %q, want %q", got.Tag, tt.wantTag)
			}
			if got.Value != tt.wantValue {
				t.Errorf("Value = %q, want %q", got.Value, tt.wantValue)
			}
		})
	}
}

// TestBuildGEDCOMIndex verifies that the reverse GEDCOM tag lookup index
// is correctly built from loaded standard vocabularies.
func TestBuildGEDCOMIndex(t *testing.T) {
	glx := &GLXFile{}
	if err := LoadStandardVocabulariesIntoGLX(glx); err != nil {
		t.Fatalf("Failed to load vocabularies: %v", err)
	}

	index := buildGEDCOMIndex(glx)

	// Event type mappings
	eventTests := map[string]string{
		"BIRT": "birth",
		"DEAT": "death",
		"MARR": "marriage",
		"DIV":  "divorce",
		"BURI": "burial",
		"CREM": "cremation",
		"BAPM": "baptism",
		"BATM": "bat_mitzvah",
		"BASM": "bat_mitzvah", // alias
		"BARM": "bar_mitzvah",
		"CHR":  "christening",
		"GRAD": "graduation",
		"RETI": "retirement",
		"EMIG": "emigration",
		"IMMI": "immigration",
		"NATU": "naturalization",
		"PROB": "probate",
		"WILL": "will",
		"ENGA": "engagement",
		"ANUL": "annulment",
	}
	for tag, want := range eventTests {
		if got := index.EventTypes[tag]; got != want {
			t.Errorf("EventTypes[%q] = %q, want %q", tag, got, want)
		}
	}

	// Person property mappings
	personPropTests := map[string]string{
		"OCCU": "occupation",
		"RELI": "religion",
		"EDUC": "education",
		"NATI": "nationality",
		"CAST": "caste",
		"SSN":  "ssn",
		"TITL": "title",
		"EXID": "external_ids",
	}
	for tag, want := range personPropTests {
		if got := index.PersonProperties[tag]; got != want {
			t.Errorf("PersonProperties[%q] = %q, want %q", tag, got, want)
		}
	}

	// Event property mappings
	eventPropTests := map[string]string{
		"AGE":  "age_at_event",
		"CAUS": "cause",
		"TYPE": "event_subtype",
	}
	for tag, want := range eventPropTests {
		if got := index.EventProperties[tag]; got != want {
			t.Errorf("EventProperties[%q] = %q, want %q", tag, got, want)
		}
	}

	// Citation property mappings
	citationPropTests := map[string]string{
		"PAGE": "locator",
		"TEXT": "text_from_source",
		"DATE": "source_date",
	}
	for tag, want := range citationPropTests {
		if got := index.CitationProperties[tag]; got != want {
			t.Errorf("CitationProperties[%q] = %q, want %q", tag, got, want)
		}
	}

	// Source property mappings
	sourcePropTests := map[string]string{
		"ABBR": "abbreviation",
		"PUBL": "publication_info",
		"CALN": "call_number",
		"EVEN": "events_recorded",
		"AGNC": "agency",
		"EXID": "external_ids",
	}
	for tag, want := range sourcePropTests {
		if got := index.SourceProperties[tag]; got != want {
			t.Errorf("SourceProperties[%q] = %q, want %q", tag, got, want)
		}
	}

	// Repository property mappings
	repoPropTests := map[string]string{
		"PHON":  "phones",
		"EMAIL": "emails",
		"EXID":  "external_ids",
	}
	for tag, want := range repoPropTests {
		if got := index.RepositoryProperties[tag]; got != want {
			t.Errorf("RepositoryProperties[%q] = %q, want %q", tag, got, want)
		}
	}

	// Media property mappings
	mediaPropTests := map[string]string{
		"MEDI": "medium",
		"CROP": "crop",
	}
	for tag, want := range mediaPropTests {
		if got := index.MediaProperties[tag]; got != want {
			t.Errorf("MediaProperties[%q] = %q, want %q", tag, got, want)
		}
	}
}

// TestParseGEDCOMName is now in gedcom_integration_test.go with correct implementation
