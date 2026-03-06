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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"EXID": "external_ids",
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

// TestConvertCensus_IndividualWithDate tests minimal CENS with just DATE.
// Should create a synthetic census source + citation but no residence property.
func TestConvertCensus_IndividualWithDate(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Smith/
1 CENS
2 DATE 1800
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Person should be created
	assert.Equal(t, 1, result.Statistics.PersonsCreated)

	// A synthetic census source should be created
	var censusSources []*Source
	for _, s := range glx.Sources {
		if s.Type == SourceTypeCensus {
			censusSources = append(censusSources, s)
		}
	}
	assert.Len(t, censusSources, 1, "Should create one synthetic census source")
	assert.Equal(t, "Census of 1800", censusSources[0].Title)
	assert.Equal(t, DateString("1800"), censusSources[0].Date)

	// No citation needed - bare source reference has no citation-level detail
	assert.Empty(t, glx.Citations, "Should not create meaningless citations")

	// No residence property (no PLAC in CENS)
	for _, p := range glx.Persons {
		_, hasResidence := p.Properties[PersonPropertyResidence]
		assert.False(t, hasResidence, "Should not set residence without PLAC")
	}

	// No assertions (no property to assert without PLAC)
	assert.Empty(t, glx.Assertions, "Should not create assertions without PLAC data")
}

// TestConvertCensus_IndividualWithDateAndPlace tests CENS with DATE + PLAC.
// Should create synthetic source + citation + residence temporal property + assertion.
func TestConvertCensus_IndividualWithDateAndPlace(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Jane /Doe/
1 CENS
2 DATE 1850
2 PLAC London, England
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.Statistics.PersonsCreated)

	// Synthetic census source
	var censusSources []*Source
	for _, s := range glx.Sources {
		if s.Type == SourceTypeCensus {
			censusSources = append(censusSources, s)
		}
	}
	assert.Len(t, censusSources, 1)
	assert.Equal(t, "Census of 1850", censusSources[0].Title)

	// No citation needed - bare source reference has no citation-level detail
	assert.Empty(t, glx.Citations, "Should not create meaningless citations")

	// Places created from PLAC hierarchy
	assert.GreaterOrEqual(t, len(glx.Places), 1, "Should create place from PLAC")

	// Residence temporal property set on person
	for _, p := range glx.Persons {
		res, hasResidence := p.Properties[PersonPropertyResidence]
		assert.True(t, hasResidence, "Should set residence from PLAC")

		// Should be a temporal value (list with date)
		resList, ok := res.([]any)
		assert.True(t, ok, "Residence should be a temporal list")
		assert.Len(t, resList, 1)

		temporal, ok := resList[0].(map[string]any)
		assert.True(t, ok, "Temporal entry should be a map")
		assert.Equal(t, "1850", temporal["date"])
		assert.NotEmpty(t, temporal["value"], "Should have place ID")
	}

	// Assertion for residence with source reference (not citation)
	assert.GreaterOrEqual(t, len(glx.Assertions), 1, "Should create residence assertion")
	var residenceAssertions []*Assertion
	for _, a := range glx.Assertions {
		if a.Property == PersonPropertyResidence {
			residenceAssertions = append(residenceAssertions, a)
		}
	}
	assert.Len(t, residenceAssertions, 1)
	assert.NotEmpty(t, residenceAssertions[0].Sources, "Assertion should have source references")
}

// TestConvertCensus_IndividualWithSOUR tests CENS with existing SOUR sub-records.
// Should use the existing source instead of creating a synthetic one.
func TestConvertCensus_IndividualWithSOUR(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @S1@ SOUR
1 TITL 1900 US Federal Census
0 @I1@ INDI
1 NAME Bob /Jones/
1 CENS
2 DATE 1900
2 PLAC Ohio, United States
2 SOUR @S1@
3 PAGE Sheet 12, Line 45
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Only the pre-existing source should exist (no synthetic source created)
	// Note: the pre-existing source gets Type inferred as "census" from its title
	assert.Len(t, glx.Sources, 1, "Should only have the pre-existing source, no synthetic source")

	// Citation should reference the pre-existing source
	assert.GreaterOrEqual(t, len(glx.Citations), 1)

	// Residence should be set
	for _, p := range glx.Persons {
		_, hasResidence := p.Properties[PersonPropertyResidence]
		assert.True(t, hasResidence, "Should set residence from PLAC")
	}
}

// TestConvertCensus_IndividualWithTYPE tests CENS with TYPE for source title.
func TestConvertCensus_IndividualWithTYPE(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Mary /Williams/
1 CENS
2 TYPE 1900 Census
2 DATE 1900
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, _, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)

	// Synthetic source should use TYPE as title
	var censusSources []*Source
	for _, s := range glx.Sources {
		if s.Type == SourceTypeCensus {
			censusSources = append(censusSources, s)
		}
	}
	assert.Len(t, censusSources, 1)
	assert.Equal(t, "1900 Census", censusSources[0].Title, "Should use TYPE value as source title")
}

// TestConvertCensus_FamilyBothSpouses tests family-level CENS applied to both spouses.
func TestConvertCensus_FamilyBothSpouses(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Husband /Test/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Wife /Test/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CENS
2 DATE 1901
2 PLAC New York, United States
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 2, result.Statistics.PersonsCreated)

	// Both persons should have residence set
	personsWithResidence := 0
	for _, p := range glx.Persons {
		if _, has := p.Properties[PersonPropertyResidence]; has {
			personsWithResidence++
		}
	}
	assert.Equal(t, 2, personsWithResidence, "Both spouses should have residence from family CENS")

	// Should have assertions for both spouses
	residenceAssertions := 0
	for _, a := range glx.Assertions {
		if a.Property == PersonPropertyResidence {
			residenceAssertions++
		}
	}
	assert.Equal(t, 2, residenceAssertions, "Both spouses should have residence assertions")

	// Only ONE synthetic source + citation should be created for the one CENS record
	var censusSources []*Source
	for _, s := range glx.Sources {
		if s.Type == SourceTypeCensus {
			censusSources = append(censusSources, s)
		}
	}
	assert.Len(t, censusSources, 1, "Family CENS should create only one synthetic source")
	assert.Empty(t, glx.Citations, "Should not create meaningless citations")
}

// TestConvertResidence_PlaceWithoutDateAppendsToExisting tests that RESI with PLAC but no DATE
// appends to existing residence list instead of overwriting it (issue #14).
func TestConvertResidence_PlaceWithoutDateAppendsToExisting(t *testing.T) {
	// Person has two RESI records: first with DATE+PLAC (temporal), second with only PLAC.
	// The second should append, not overwrite the first.
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Alice /Test/
1 RESI
2 DATE 1900
2 PLAC London, England
1 RESI
2 PLAC Paris, France
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, _, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)

	for _, p := range glx.Persons {
		res, hasResidence := p.Properties[PersonPropertyResidence]
		require.True(t, hasResidence, "Person should have residence property")

		// Should be a list with both entries preserved
		resList, ok := res.([]any)
		require.True(t, ok, "Residence should be a list, got %T", res)
		assert.Len(t, resList, 2, "Both residence entries should be preserved, not overwritten")
	}
}

// TestConvertResidence_TwoUndatedAppendsToList tests that two consecutive RESI with PLAC
// but no DATE both get preserved (covers the non-list append branch).
func TestConvertResidence_TwoUndatedAppendsToList(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Carol /Test/
1 RESI
2 PLAC London, England
1 RESI
2 PLAC Paris, France
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, _, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)

	for _, p := range glx.Persons {
		res, hasResidence := p.Properties[PersonPropertyResidence]
		require.True(t, hasResidence, "Person should have residence property")

		resList, ok := res.([]any)
		require.True(t, ok, "Residence should be a list when multiple undated entries exist, got %T", res)
		assert.Len(t, resList, 2, "Both undated residence entries should be preserved")
	}
}

// TestConvertCensus_PlaceWithoutDateAppendsToExisting tests that CENS with PLAC but no DATE
// appends to existing residence list instead of overwriting it (issue #14).
func TestConvertCensus_PlaceWithoutDateAppendsToExisting(t *testing.T) {
	// Person has a RESI with DATE+PLAC, then a CENS with only PLAC (no DATE).
	// The CENS should append, not overwrite the existing RESI.
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Bob /Test/
1 RESI
2 DATE 1900
2 PLAC London, England
1 CENS
2 PLAC Manchester, England
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, _, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)

	for _, p := range glx.Persons {
		res, hasResidence := p.Properties[PersonPropertyResidence]
		require.True(t, hasResidence, "Person should have residence property")

		// Should be a list with both entries preserved
		resList, ok := res.([]any)
		require.True(t, ok, "Residence should be a list, got %T", res)
		assert.Len(t, resList, 2, "Both residence entries should be preserved, not overwritten")
	}
}
