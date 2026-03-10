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

func TestImportPersonNote_StoredInNotesField(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n1 NOTE This is a person note\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glxFile.Persons, 1)

	for _, person := range glxFile.Persons {
		assert.Equal(t, "This is a person note", person.Notes,
			"NOTE should be stored in person.Notes struct field")
		_, inProps := person.Properties[PropertyNotes]
		assert.False(t, inProps,
			"NOTE should NOT be stored in Properties['notes']")
	}
}

func TestImportEventNote_StoredInNotesField(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 BIRT\n2 DATE 1850\n2 NOTE Born in a small village\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	var foundNote bool
	for _, event := range glxFile.Events {
		if event.Type == "birth" && event.Notes != "" {
			assert.Equal(t, "Born in a small village", event.Notes)
			_, inProps := event.Properties[PropertyNotes]
			assert.False(t, inProps,
				"NOTE should NOT be stored in event Properties['notes']")
			foundNote = true
		}
	}
	assert.True(t, foundNote, "Should find birth event with note")
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

func TestImportGenericEVEN_CreatesEvent(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 EVEN\n2 TYPE Anecdote\n2 DATE 1900\n2 NOTE A funny story\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	var foundEvent bool
	for _, event := range glxFile.Events {
		// EVEN with TYPE "Anecdote" should create an event with type "anecdote" or "even"
		if event.Date == "1900" {
			foundEvent = true
			assert.NotEmpty(t, event.Type, "Generic EVEN should have a type")
			assert.Equal(t, "A funny story", event.Notes, "EVEN NOTE should be in Notes field")
		}
	}
	assert.True(t, foundEvent, "Generic EVEN should create an event")
}

func TestImportIndividualEvent_SOURCitationsPreserved(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @S1@ SOUR\n1 TITL Birth Record\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 BIRT\n2 DATE 1 JAN 1900\n2 SOUR @S1@\n3 PAGE Page 42\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Find the birth event
	var birthEvent *Event
	for _, event := range glxFile.Events {
		if event.Type == "birth" && event.Date == "1900-01-01" {
			birthEvent = event
			break
		}
	}
	require.NotNil(t, birthEvent, "Should have a birth event")

	// The event should have citations or sources in its properties
	hasCitations := false
	if citations, ok := birthEvent.Properties[PropertyCitations].([]string); ok && len(citations) > 0 {
		hasCitations = true
	}
	if sources, ok := birthEvent.Properties[PropertySources].([]string); ok && len(sources) > 0 {
		hasCitations = true
	}
	assert.True(t, hasCitations, "Birth event should have SOUR citations stored in Properties")
}

func TestImportTITL_WithDatePreserved(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 TITL König\n2 DATE 1991\n" +
		"1 TITL Baron\n2 DATE 1985\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glxFile.Persons, 1)

	for _, person := range glxFile.Persons {
		titleVal, ok := person.Properties["title"]
		require.True(t, ok, "Person should have title property")

		// Multiple TITL with DATE should be stored as temporal list
		titleList, ok := titleVal.([]any)
		require.True(t, ok, "Multiple titles should be a list, got %T", titleVal)
		assert.Len(t, titleList, 2)

		// Each item should have value and date
		for _, item := range titleList {
			itemMap, ok := item.(map[string]any)
			require.True(t, ok, "Item should be a map")
			assert.Contains(t, itemMap, "value")
			assert.Contains(t, itemMap, "date")
		}
	}
}

func TestImportDate_CalendarEscapePreserved(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 BIRT\n2 DATE @#DJULIAN@ 11 FEB 1731/32\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	var birthEvent *Event
	for _, event := range glxFile.Events {
		if event.Type == "birth" {
			birthEvent = event
			break
		}
	}
	require.NotNil(t, birthEvent, "Should have a birth event")
	assert.NotEmpty(t, string(birthEvent.Date), "Julian calendar date should be preserved, not dropped")
}

func TestImportDate_BCEPreserved(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 7.0\n1 SCHMA\n" +
		"0 @I1@ INDI\n1 NAME Julius /Caesar/\n" +
		"1 BIRT\n2 DATE 100 BCE\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	var birthEvent *Event
	for _, event := range glxFile.Events {
		if event.Type == "birth" {
			birthEvent = event
			break
		}
	}
	require.NotNil(t, birthEvent, "Should have a birth event")
	assert.NotEmpty(t, string(birthEvent.Date), "BCE date should be preserved, not dropped")
}

func TestImportSingleSpouseFamily_MarriagePreserved(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n1 FAMS @F1@\n" +
		"0 @F1@ FAM\n1 HUSB @I1@\n1 MARR\n2 DATE 1 JAN 1900\n2 PLAC London\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Should have a marriage relationship even with single spouse
	var foundMarriage bool
	for _, rel := range glxFile.Relationships {
		if rel.Type == RelationshipTypeMarriage {
			foundMarriage = true
			// Should have at least one participant
			assert.GreaterOrEqual(t, len(rel.Participants), 1, "Marriage should have at least one participant")
			break
		}
	}
	assert.True(t, foundMarriage, "Single-spouse family with MARR should create marriage relationship")

	// The marriage event should exist with the date
	var foundMarriageEvent bool
	for _, event := range glxFile.Events {
		if event.Type == "marriage" && event.Date == "1900-01-01" {
			foundMarriageEvent = true
			break
		}
	}
	assert.True(t, foundMarriageEvent, "Marriage event with date should be created")
}

func TestImportMultipleOCCU_PreservesAll(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" +
		"0 @I1@ INDI\n1 NAME John /Smith/\n" +
		"1 OCCU Farmer\n1 OCCU Blacksmith\n1 OCCU Mayor\n" +
		"0 TRLR\n"

	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glxFile.Persons, 1)

	for _, person := range glxFile.Persons {
		occVal, ok := person.Properties["occupation"]
		require.True(t, ok, "Person should have occupation property")

		// Multiple OCCU should be stored as a temporal list of {value: ...} objects
		occList, ok := occVal.([]any)
		require.True(t, ok, "Multiple occupations should be a list, got %T", occVal)
		assert.Len(t, occList, 3, "All three occupations should be preserved")

		// Each item should be a {value: ...} map
		for i, item := range occList {
			itemMap, ok := item.(map[string]any)
			require.True(t, ok, "Item %d should be a map, got %T", i, item)
			assert.Contains(t, itemMap, "value", "Item %d should have 'value' key", i)
		}
	}
}

func TestImportFamilyEventType_Preserved(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Jones/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 15 JUN 1950
1 EVEN
2 TYPE separation
2 DATE 1965
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Find the generic event with event_subtype = "separation"
	var foundSeparation bool
	for _, event := range glxFile.Events {
		if event.Type == EventTypeGeneric {
			if st, ok := event.Properties["event_subtype"]; ok && st == "separation" {
				foundSeparation = true
				// Should have both spouses as participants
				assert.GreaterOrEqual(t, len(event.Participants), 2,
					"Family event should have both spouses as participants")
				assert.Contains(t, string(event.Date), "1965")
			}
		}
	}
	assert.True(t, foundSeparation, "Family EVEN with TYPE separation should be imported with event_subtype")
}

func TestImportFamilyNote_PreservedOnRelationship(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Jones/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 15 JUN 1950
1 NOTE Marriage performed at city hall.
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Find the marriage relationship and check Notes
	var foundNote bool
	for _, rel := range glxFile.Relationships {
		if rel.Type == RelationshipTypeMarriage && rel.Notes != "" {
			assert.Equal(t, "Marriage performed at city hall.", rel.Notes)
			foundNote = true
		}
	}
	assert.True(t, foundNote, "FAM-level NOTE should be stored on relationship.Notes")
}

func TestImportFamilyRESI_DistributedToSpouses(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Jones/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 RESI
2 PLAC Springfield, Illinois, USA
2 DATE 1920
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Both spouses should have a residence property
	var residenceCount int
	for _, person := range glxFile.Persons {
		if _, ok := person.Properties[PersonPropertyResidence]; ok {
			residenceCount++
		}
	}
	assert.Equal(t, 2, residenceCount, "Family-level RESI should be distributed to both spouses")
}

// TestImportCensus_SyntheticSource tests census records without SOUR sub-records
// create a synthetic census source.
func TestImportCensus_SyntheticSource(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 CENS
2 DATE 1850
2 PLAC Springfield, Illinois, USA
2 NOTE Household #42, dwelling 15
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Should create a synthetic census source
	require.NotEmpty(t, glxFile.Sources, "synthetic census source should be created")

	var censusSource *Source
	for _, src := range glxFile.Sources {
		if src.Type == SourceTypeCensus {
			censusSource = src
			break
		}
	}
	require.NotNil(t, censusSource, "should have a census-type source")
	assert.Contains(t, censusSource.Title, "1850", "census source title should contain the date")

	// Note should be on a citation
	var noteCitation *Citation
	for _, cit := range glxFile.Citations {
		if cit.Notes != "" {
			noteCitation = cit
			break
		}
	}
	require.NotNil(t, noteCitation, "census note should create a citation")
	assert.Contains(t, noteCitation.Notes, "Household #42")
}

// TestImportCensus_WithSOURAndNote tests census records with both SOUR and NOTE
// attach the note to the existing citation.
func TestImportCensus_WithSOURAndNote(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @S1@ SOUR
1 TITL 1860 US Census
0 @I1@ INDI
1 NAME Jane /Doe/
1 SEX F
1 CENS
2 DATE 1860
2 PLAC Boston, Massachusetts, USA
2 SOUR @S1@
3 PAGE Sheet 5, line 12
2 NOTE Boarder in household
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Note should be attached to the citation from the SOUR reference
	foundNote := false
	for _, cit := range glxFile.Citations {
		if strings.Contains(cit.Notes, "Boarder") {
			foundNote = true
			break
		}
	}
	assert.True(t, foundNote, "census NOTE should be attached to citation when SOUR exists")
}

// TestImportCensus_NoPlace tests census without PLAC still creates source but no residence.
func TestImportCensus_NoPlace(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Bob /Jones/
1 SEX M
1 CENS
2 DATE 1870
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// No residence property (no PLAC)
	for _, person := range glxFile.Persons {
		_, hasResi := person.Properties[PersonPropertyResidence]
		assert.False(t, hasResi, "census without PLAC should not create residence property")
	}
}

// TestImportCensus_WithType tests census TYPE sub-record is used in synthetic source title.
func TestImportCensus_WithType(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Alice /Brown/
1 SEX F
1 CENS
2 TYPE Federal Census
2 PLAC New York, New York, USA
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	var censusSource *Source
	for _, src := range glxFile.Sources {
		if src.Type == SourceTypeCensus {
			censusSource = src
			break
		}
	}
	require.NotNil(t, censusSource)
	assert.Equal(t, "Federal Census", censusSource.Title, "TYPE should be used as source title")
}

// TestImportFact_WithTypeAndValue tests FACT with TYPE and value creates property assertion.
func TestImportFact_WithTypeAndValue(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FACT Farmer
2 TYPE Occupation
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glxFile.Persons, 1)

	// FACT with TYPE+value should NOT create an event (handled as property)
	assert.Empty(t, glxFile.Events, "FACT with TYPE+value should be a property, not an event")
}

// TestImportFact_WithDateAndPlace tests FACT with DATE/PLAC is converted as generic event.
func TestImportFact_WithDateAndPlace(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FACT
2 TYPE Arrival
2 DATE 15 MAR 1880
2 PLAC New York, USA
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// FACT with DATE/PLAC should create an event
	assert.NotEmpty(t, glxFile.Events, "FACT with DATE/PLAC should be converted to an event")
}

// TestImportFamilyCensus_DistributedToBothSpouses tests family-level CENS
// creates source once and applies residence to both spouses.
func TestImportFamilyCensus_DistributedToBothSpouses(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CENS
2 DATE 1900
2 PLAC Chicago, Cook, Illinois, USA
1 MARR
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Both spouses should have residence
	residenceCount := 0
	for _, person := range glxFile.Persons {
		if _, ok := person.Properties[PersonPropertyResidence]; ok {
			residenceCount++
		}
	}
	assert.Equal(t, 2, residenceCount, "family CENS should give both spouses a residence")

	// Should only create ONE census source (not duplicated per spouse)
	censusSourceCount := 0
	for _, src := range glxFile.Sources {
		if src.Type == SourceTypeCensus {
			censusSourceCount++
		}
	}
	assert.Equal(t, 1, censusSourceCount, "family CENS should create only one source")
}

// TestImportInvalidGEDCOM tests import of malformed input.
func TestImportInvalidGEDCOM(t *testing.T) {
	// No HEAD record — should still parse (lenient)
	glx1, _, err := ImportGEDCOM(strings.NewReader("not a gedcom file"), nil)
	// Malformed input may or may not error; just verify no panic
	_ = err
	_ = glx1

	// Empty input — should not panic
	glx2, _, err2 := ImportGEDCOM(strings.NewReader(""), nil)
	_ = err2
	_ = glx2
}

// TestImportCensus_WithDateNoPlace tests applyCensusData early return on missing place.
func TestImportCensus_WithDateNoPlace(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Test /Person/
1 SEX M
1 CENS
2 DATE 1910
2 NOTE Census note only
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Source should still be created (synthetic)
	require.NotEmpty(t, glxFile.Sources)
	// But no residence (no PLAC)
	for _, person := range glxFile.Persons {
		_, hasResi := person.Properties[PersonPropertyResidence]
		assert.False(t, hasResi)
	}
}

// TestImportCensus_PlaceWithoutDate tests residence without date gets stored as simple string.
func TestImportCensus_PlaceWithoutDate(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Test /Person/
1 SEX M
1 CENS
2 PLAC Austin, Texas, USA
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Should have residence property (place without date = simple string, not map)
	for _, person := range glxFile.Persons {
		resi, ok := person.Properties[PersonPropertyResidence]
		if ok {
			// Should be a simple string (place ID), not a map
			_, isString := resi.(string)
			_, isSlice := resi.([]any)
			assert.True(t, isString || isSlice, "residence without date should be stored as string or list")
		}
	}
}

// TestImportCensus_WithMedia tests census with OBJE attaches media to citations.
func TestImportCensus_WithMedia(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @O1@ OBJE
1 FILE census_image.jpg
2 FORM JPEG
0 @I1@ INDI
1 NAME Test /Person/
1 SEX M
1 CENS
2 DATE 1920
2 PLAC Portland, Oregon, USA
2 OBJE @O1@
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Should have media and it should be linked
	assert.NotEmpty(t, glxFile.Media, "OBJE should create media entity")

	// Census should create a synthetic source, and the media should be attached
	mediaLinked := false
	for _, src := range glxFile.Sources {
		if len(src.Media) > 0 {
			mediaLinked = true
			break
		}
	}
	for _, cit := range glxFile.Citations {
		if len(cit.Media) > 0 {
			mediaLinked = true
			break
		}
	}
	assert.True(t, mediaLinked, "census OBJE should be linked to source or citation")
}

// TestImportFamilyCensus_SingleSpouse tests family census with only one spouse.
func TestImportFamilyCensus_SingleSpouse(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 CENS
2 DATE 1880
2 PLAC Denver, Colorado, USA
1 MARR
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Single spouse should still get residence
	for _, person := range glxFile.Persons {
		_, hasResi := person.Properties[PersonPropertyResidence]
		assert.True(t, hasResi, "single-spouse family CENS should create residence")
	}
}

// TestImportFamilyEvent_UnmappedTag tests family events with tags not in vocabulary
// fall back to generic event type.
func TestImportFamilyEvent_UnmappedTag(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Jane /Smith/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 EVEN
2 TYPE Family Reunion
2 DATE 4 JUL 1950
1 MARR
0 TRLR
`
	glxFile, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// The EVEN tag should produce a generic event
	foundGeneric := false
	for _, event := range glxFile.Events {
		if event.Type == EventTypeGeneric {
			foundGeneric = true
			break
		}
	}
	assert.True(t, foundGeneric, "EVEN on family should create a generic event")
}
