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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// ============================================================================
// Roundtrip tests: Import GEDCOM → Export GEDCOM → Re-import GEDCOM
// Verifies data preservation through the full cycle.
// ============================================================================

func TestRoundtrip_EventCitationDetailsPreserved(t *testing.T) {
	const citationA = "2 SOUR @S1@\n3 PAGE Page 42\n3 NOTE Citation A\n3 OBJE @M1@\n"
	cases := []struct {
		name      string
		refs      string
		wantNotes []string
		wantMedia []string
	}{
		{
			name:      "bare source and unpaged citation",
			refs:      "2 SOUR @S1@\n2 SOUR @S1@\n3 NOTE Citation B\n3 OBJE @M2@\n",
			wantNotes: []string{"Citation B"},
			wantMedia: []string{"media/files/second.jpg"},
		},
		{
			name:      "same page with different notes and media",
			refs:      citationA + "2 SOUR @S1@\n3 PAGE Page 42\n3 NOTE Citation B\n3 OBJE @M2@\n",
			wantNotes: []string{"Citation A", "Citation B"},
			wantMedia: []string{"media/files/first.jpg", "media/files/second.jpg"},
		},
		{
			name:      "same page with different notes",
			refs:      citationA + "2 SOUR @S1@\n3 PAGE Page 42\n3 NOTE Citation B\n3 OBJE @M1@\n",
			wantNotes: []string{"Citation A", "Citation B"},
			wantMedia: []string{"media/files/first.jpg", "media/files/first.jpg"},
		},
		{
			name:      "same page with different media",
			refs:      citationA + "2 SOUR @S1@\n3 PAGE Page 42\n3 NOTE Citation A\n3 OBJE @M2@\n",
			wantNotes: []string{"Citation A", "Citation A"},
			wantMedia: []string{"media/files/first.jpg", "media/files/second.jpg"},
		},
		{
			name:      "identical citations",
			refs:      citationA + citationA,
			wantNotes: []string{"Citation A"},
			wantMedia: []string{"media/files/first.jpg"},
		},
	}
	for _, version := range []GEDCOMVersion{GEDCOM551, GEDCOM70} {
		for _, tag := range []string{GedcomTagBirt, GedcomTagDeat, GedcomTagMarr, GedcomTagDiv} {
			for _, tc := range cases {
				t.Run(fmt.Sprintf("%d/%s/%s", version, tag, tc.name), func(t *testing.T) {
					// Use a normal import to construct the archive, including the
					// event-entity evidence on family events with no assertions.
					gedcom := `0 HEAD
1 SOUR TEST
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @S1@ SOUR
1 TITL Parish register
0 @M1@ OBJE
1 FILE first.jpg
2 FORM jpg
0 @M2@ OBJE
1 FILE second.jpg
2 FORM jpg
0 @I1@ INDI
1 NAME John /Smith/
`
					if tag == GedcomTagMarr || tag == GedcomTagDiv {
						gedcom += "1 FAMS @F1@\n0 @I2@ INDI\n1 NAME Jane /Smith/\n1 FAMS @F1@\n0 @F1@ FAM\n1 HUSB @I1@\n1 WIFE @I2@\n"
					}
					gedcom += "1 " + tag + "\n2 DATE 15 JAN 1850\n" + tc.refs + "0 TRLR\n"
					archive, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
					require.NoError(t, err)
					exported, _, err := ExportGEDCOM(archive, version, nil)
					require.NoError(t, err)
					assert.Equal(t, len(tc.wantMedia), strings.Count(string(exported), "\n3 OBJE "),
						"every distinct citation must retain its media association")

					roundtripped, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
					require.NoError(t, err)
					require.Len(t, roundtripped.Events, 1)
					var notes, mediaURIs []string
					for _, event := range roundtripped.Events {
						for _, citationID := range extractStringList(event.Properties[PropertyCitations]) {
							citation := roundtripped.Citations[citationID]
							require.NotNil(t, citation)
							notes = append(notes, citation.Notes...)
							for _, mediaID := range citation.Media {
								media := roundtripped.Media[mediaID]
								require.NotNil(t, media)
								mediaURIs = append(mediaURIs, media.URI)
							}
						}
					}
					assert.ElementsMatch(t, tc.wantNotes, notes, "citation notes must survive the round trip")
					assert.ElementsMatch(t, tc.wantMedia, mediaURIs, "citation media links must survive the round trip")
				})
			}
		}
	}
}

// TestRoundtrip_MinimalFamily tests a simple family roundtrip
func TestRoundtrip_MinimalFamily(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR PAF
2 NAME Personal Ancestral File
2 VERS 5.2
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
2 GIVN John
2 SURN Smith
1 SEX M
1 BIRT
2 DATE 15 MAR 1850
2 PLAC Springfield, Illinois, USA
1 DEAT
2 DATE 22 JUN 1920
2 PLAC Chicago, Illinois, USA
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Jones/
2 GIVN Mary
2 SURN Jones
1 SEX F
1 BIRT
2 DATE 3 APR 1855
1 FAMS @F1@
0 @I3@ INDI
1 NAME James /Smith/
2 GIVN James
2 SURN Smith
1 SEX M
1 BIRT
2 DATE 10 JAN 1880
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE 5 JUN 1875
2 PLAC Springfield, Illinois, USA
0 TRLR
`
	// Step 1: Import the original GEDCOM
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	// Verify initial import
	assert.Len(t, glx1.Persons, 3, "expected 3 persons after import")

	// Step 2: Export to GEDCOM 5.5.1
	exported, result, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err, "export failed")
	require.NotNil(t, result)

	assert.Equal(t, 3, result.Statistics.PersonsExported)
	assert.Equal(t, 1, result.Statistics.FamiliesExported)

	exportedStr := string(exported)

	// Verify exported GEDCOM has expected content
	assert.Contains(t, exportedStr, "0 HEAD")
	assert.Contains(t, exportedStr, "0 TRLR")
	assert.Contains(t, exportedStr, "INDI")
	assert.Contains(t, exportedStr, "FAM")
	assert.Contains(t, exportedStr, "MARR")

	// Step 3: Re-import the exported GEDCOM
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err, "re-import failed")

	// Step 4: Compare entity counts
	assert.Len(t, glx2.Persons, len(glx1.Persons), "person count mismatch after roundtrip")
	assert.Len(t, glx2.Relationships, len(glx1.Relationships), "relationship count mismatch after roundtrip")

	// Step 5: Verify key data preserved
	// Find John Smith in the re-imported data
	var foundJohn, foundMary, foundJames bool
	for _, person := range glx2.Persons {
		name := getPersonNameValue(person)
		switch {
		case strings.Contains(name, "John") && strings.Contains(name, "Smith"):
			foundJohn = true
			assert.Equal(t, "male", testGetPersonSex(person))
		case strings.Contains(name, "Mary") && strings.Contains(name, "Jones"):
			foundMary = true
			assert.Equal(t, "female", testGetPersonSex(person))
		case strings.Contains(name, "James") && strings.Contains(name, "Smith"):
			foundJames = true
			assert.Equal(t, "male", testGetPersonSex(person))
		}
	}
	assert.True(t, foundJohn, "John Smith not found after roundtrip")
	assert.True(t, foundMary, "Mary Jones not found after roundtrip")
	assert.True(t, foundJames, "James Smith not found after roundtrip")

	// Verify marriage events exist
	var foundMarriage bool
	for _, event := range glx2.Events {
		if event.Type == EventTypeMarriage {
			foundMarriage = true
			assert.Contains(t, string(event.Date), "1875", "marriage date should contain 1875")
		}
	}
	assert.True(t, foundMarriage, "marriage event not found after roundtrip")
}

// TestRoundtrip_FamilyNCHI verifies the FAM.NCHI (number of children) tag
// survives an import → export → re-import cycle.
func TestRoundtrip_FamilyNCHI(t *testing.T) {
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
1 NCHI 5
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err, "export failed")
	assert.Contains(t, string(exported), "1 NCHI 5", "exported GEDCOM should contain NCHI")

	glx2, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
	require.NoError(t, err, "re-import failed")

	var foundNCHI bool
	for _, rel := range glx2.Relationships {
		if rel.Type != RelationshipTypeMarriage {
			continue
		}
		if val, ok := rel.Properties[RelationshipPropertyNumberOfChildren]; ok {
			assert.Equal(t, 5, val, "NCHI value should survive roundtrip as an integer")
			foundNCHI = true
		}
	}
	assert.True(t, foundNCHI, "NCHI should be present on the relationship after roundtrip")
}

// assertNCHIRoundtrip imports a 5.5.1 FAM carrying the given NCHI value, exports
// to the target version, and re-imports — asserting the value survives intact.
func assertNCHIRoundtrip(t *testing.T, targetVersion GEDCOMVersion, nchi int) {
	t.Helper()

	gedcom := fmt.Sprintf(`0 HEAD
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
1 NCHI %d
0 TRLR
`, nchi)

	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	exported, _, err := ExportGEDCOM(glx1, targetVersion, nil)
	require.NoError(t, err, "export failed")
	assert.Contains(t, string(exported), fmt.Sprintf("1 NCHI %d", nchi), "exported GEDCOM should contain NCHI")

	glx2, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
	require.NoError(t, err, "re-import failed")

	var found bool
	for _, rel := range glx2.Relationships {
		if rel.Type != RelationshipTypeMarriage {
			continue
		}
		if val, ok := rel.Properties[RelationshipPropertyNumberOfChildren]; ok {
			assert.Equal(t, nchi, val, "NCHI value should survive roundtrip as an integer")
			found = true
		}
	}
	assert.True(t, found, "NCHI should be present on the relationship after roundtrip")
}

// TestRoundtrip_FamilyNCHI_Zero verifies NCHI 0 (a valid, distinct value)
// survives the round-trip rather than being dropped as if absent.
func TestRoundtrip_FamilyNCHI_Zero(t *testing.T) {
	assertNCHIRoundtrip(t, GEDCOM551, 0)
}

// TestRoundtrip_FamilyNCHI_GEDCOM70 verifies NCHI round-trips through GEDCOM 7.0
// export, which supports NCHI with identical semantics to 5.5.1.
func TestRoundtrip_FamilyNCHI_GEDCOM70(t *testing.T) {
	assertNCHIRoundtrip(t, GEDCOM70, 3)
}

// TestRoundtrip_GEDCOM70 tests roundtrip with GEDCOM 7.0 output
func TestRoundtrip_GEDCOM70(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Alice /Wonder/
2 GIVN Alice
2 SURN Wonder
1 SEX F
1 BIRT
2 DATE 1 JAN 1900
0 TRLR
`
	// Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Export to GEDCOM 7.0
	exported, result, err := ExportGEDCOM(glx1, GEDCOM70, nil)
	require.NoError(t, err)
	assert.Equal(t, GEDCOMVersion70, result.Version)

	exportedStr := string(exported)

	// GEDCOM 7.0 should have version 7.0 in header
	assert.Contains(t, exportedStr, "7.0")

	// GEDCOM 7.0 should NOT have CHAR or FORM in header
	// (5.5.1-specific features)
	lines := strings.Split(exportedStr, "\n")
	var inHead bool
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "0 HEAD") {
			inHead = true
		}
		if inHead && strings.HasPrefix(trimmed, "0 ") && !strings.HasPrefix(trimmed, "0 HEAD") {
			inHead = false
		}
		if inHead {
			assert.NotContains(t, trimmed, "CHAR UTF-8", "GEDCOM 7.0 should not have CHAR")
			assert.NotContains(t, trimmed, "FORM LINEAGE-LINKED", "GEDCOM 7.0 should not have FORM")
		}
	}

	// Re-import the 7.0 output
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	assert.Len(t, glx2.Persons, len(glx1.Persons))

	// Verify the person survived
	var foundAlice bool
	for _, person := range glx2.Persons {
		name := getPersonNameValue(person)
		if strings.Contains(name, "Alice") {
			foundAlice = true
		}
	}
	assert.True(t, foundAlice, "Alice not found after roundtrip")
}

// TestRoundtrip_SourcesAndRepositories tests source/repo preservation
func TestRoundtrip_SourcesAndRepositories(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @R1@ REPO
1 NAME County Records Office
1 ADDR 123 Main Street
2 CITY Springfield
2 STAE Illinois
2 CTRY USA
0 @S1@ SOUR
1 TITL Birth Records of Springfield
1 AUTH County Clerk
1 PUBL Springfield County, 1900
1 REPO @R1@
0 @I1@ INDI
1 NAME Test /Person/
2 GIVN Test
2 SURN Person
1 SEX M
0 TRLR
`
	// Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	assert.Len(t, glx1.Sources, 1, "expected 1 source")
	assert.Len(t, glx1.Repositories, 1, "expected 1 repository")

	// Export
	exported, result, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)

	assert.Equal(t, 1, result.Statistics.SourcesExported)
	assert.Equal(t, 1, result.Statistics.RepositoriesExported)

	exportedStr := string(exported)

	// Verify source fields present
	assert.Contains(t, exportedStr, "TITL Birth Records of Springfield")
	assert.Contains(t, exportedStr, "AUTH County Clerk")
	assert.Contains(t, exportedStr, "PUBL Springfield County, 1900")

	// Verify repository fields present
	assert.Contains(t, exportedStr, "NAME County Records Office")
	assert.Contains(t, exportedStr, "ADDR")

	// Verify source references repository
	assert.Contains(t, exportedStr, "REPO @R")

	// Re-import
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	assert.Len(t, glx2.Sources, len(glx1.Sources), "source count mismatch")
	assert.Len(t, glx2.Repositories, len(glx1.Repositories), "repository count mismatch")
}

// TestRoundtrip_StandardSourceTypesPreserved verifies that every standard
// source type survives a GLX -> GEDCOM 7.0 -> GLX round-trip. The exporter
// writes Source.Type verbatim as the GEDCOM TYPE value and the importer maps
// it back via mapSourceType; before mapSourceType recognized canonical GLX
// keys, types whose GEDCOM alias differs from the key (vital_record,
// church_register, tax_record, ...) or that have no alias at all (immigration,
// directory, oral_history, correspondence) downgraded to "other" (#563,
// #1005). Driven from standardSourceTypes so future vocabulary additions stay
// covered automatically.
func TestRoundtrip_StandardSourceTypesPreserved(t *testing.T) {
	sources := make(map[string]*Source, len(standardSourceTypes))
	for typ := range standardSourceTypes {
		sources["source-"+typ] = &Source{Title: "rt-source-" + typ, Type: typ}
	}
	glx1 := &GLXFile{Sources: sources}

	// TYPE is a GEDCOM 7.0-only tag, so the round-trip must use GEDCOM70.
	exported, _, err := ExportGEDCOM(glx1, GEDCOM70, nil)
	require.NoError(t, err)

	glx2, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
	require.NoError(t, err)
	require.Len(t, glx2.Sources, len(standardSourceTypes))

	// Source IDs are regenerated on import, so match by the preserved title.
	gotByTitle := make(map[string]string, len(glx2.Sources))
	for _, s := range glx2.Sources {
		gotByTitle[s.Title] = s.Type
	}
	for typ := range standardSourceTypes {
		assert.Equal(t, typ, gotByTitle["rt-source-"+typ], "source type %q must survive GLX->GEDCOM->GLX round-trip", typ)
	}
}

// TestRoundtrip_MultipleRelationshipTypes tests various relationship types
func TestRoundtrip_MultipleRelationshipTypes(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Father /Test/
2 GIVN Father
2 SURN Test
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mother /Test/
2 GIVN Mother
2 SURN Test
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Child1 /Test/
2 GIVN Child1
2 SURN Test
1 SEX M
1 FAMC @F1@
0 @I4@ INDI
1 NAME Child2 /Test/
2 GIVN Child2
2 SURN Test
1 SEX F
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
1 MARR
2 DATE 1 JAN 1950
1 DIV
2 DATE 15 MAR 1970
0 TRLR
`
	// Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	assert.Len(t, glx1.Persons, 4)

	// Export
	exported, result, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)

	assert.Equal(t, 4, result.Statistics.PersonsExported)
	assert.Equal(t, 1, result.Statistics.FamiliesExported)

	exportedStr := string(exported)

	// Verify marriage and divorce present
	assert.Contains(t, exportedStr, "MARR")
	assert.Contains(t, exportedStr, "1 JAN 1950")
	assert.Contains(t, exportedStr, "DIV")
	assert.Contains(t, exportedStr, "15 MAR 1970")

	// Verify both children referenced
	assert.Contains(t, exportedStr, "CHIL")

	// Re-import
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	assert.Len(t, glx2.Persons, len(glx1.Persons), "person count mismatch")

	// Verify divorce event preserved
	var foundDivorce bool
	for _, event := range glx2.Events {
		if event.Type == EventTypeDivorce {
			foundDivorce = true
		}
	}
	assert.True(t, foundDivorce, "divorce event not found after roundtrip")
}

// TestRoundtrip_DateFormats tests various date format preservation
func TestRoundtrip_DateFormats(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Date /Test/
2 GIVN Date
2 SURN Test
1 SEX M
1 BIRT
2 DATE 15 MAR 1850
1 DEAT
2 DATE ABT 1920
0 @I2@ INDI
1 NAME Range /Test/
2 GIVN Range
2 SURN Test
1 SEX F
1 BIRT
2 DATE BET 1 JAN 1860 AND 31 DEC 1865
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)

	exportedStr := string(exported)

	// Verify exact date preserved
	assert.Contains(t, exportedStr, "15 MAR 1850")

	// Verify approximate date preserved
	assert.Contains(t, exportedStr, "ABT 1920")

	// Verify range date preserved
	assert.Contains(t, exportedStr, "BET")
	assert.Contains(t, exportedStr, "AND")
}

// TestRoundtrip_EmptyArchive tests export of an empty archive
func TestRoundtrip_EmptyArchive(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	exported, result, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)

	assert.Equal(t, 0, result.Statistics.PersonsExported)
	assert.Equal(t, 0, result.Statistics.FamiliesExported)

	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "0 HEAD")
	assert.Contains(t, exportedStr, "0 TRLR")
}

// TestRoundtrip_NilGLXFile tests export with nil input
func TestRoundtrip_NilGLXFile(t *testing.T) {
	_, _, err := ExportGEDCOM(nil, GEDCOM551, nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrGLXFileNil)
}

// ============================================================================
// Helper functions for roundtrip tests
// ============================================================================

// getPersonNameValue extracts the name value string from a person's properties
func getPersonNameValue(person *Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}

	nameRaw, ok := person.Properties[PersonPropertyName]
	if !ok {
		return ""
	}

	switch name := nameRaw.(type) {
	case map[string]any:
		if v, ok := name["value"]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	case string:
		return name
	}

	return ""
}

// testGetPersonSex extracts the recorded sex from a person's properties.
func testGetPersonSex(person *Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}

	sexRaw, ok := person.Properties[PersonPropertySex]
	if !ok {
		return ""
	}

	if s, ok := sexRaw.(string); ok {
		return s
	}

	return ""
}

func TestRoundtrip_MarriageTypePreserved(t *testing.T) {
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
2 TYPE civil
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Verify marriage_type was imported
	var foundMarriageType bool
	for _, event := range glx1.Events {
		if event.Type == EventTypeMarriage {
			if mt, ok := event.Properties[PropertyMarriageType]; ok {
				assert.Equal(t, "civil", mt)
				foundMarriageType = true
			}
		}
	}
	assert.True(t, foundMarriageType, "marriage_type should be imported from MARR TYPE")

	// Export
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)

	// Verify TYPE appears in exported MARR
	assert.Contains(t, exportedStr, "1 MARR")
	assert.Contains(t, exportedStr, "2 TYPE civil")

	// Re-import and verify preserved
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	var foundAfterRoundtrip bool
	for _, event := range glx2.Events {
		if event.Type == EventTypeMarriage {
			if mt, ok := event.Properties[PropertyMarriageType]; ok {
				assert.Equal(t, "civil", mt)
				foundAfterRoundtrip = true
			}
		}
	}
	assert.True(t, foundAfterRoundtrip, "marriage_type should survive roundtrip")
}

func TestRoundtrip_FamilyEventTypePreserved(t *testing.T) {
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
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Verify event_subtype was imported on the family event
	var foundSubtype bool
	for _, event := range glx1.Events {
		if st, ok := event.Properties["event_subtype"]; ok && st == "separation" {
			foundSubtype = true
		}
	}
	assert.True(t, foundSubtype, "event_subtype should be imported from family EVEN TYPE")

	// Export
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)

	// Verify TYPE separation appears in exported EVEN
	assert.Contains(t, exportedStr, "TYPE separation")

	// Re-import and verify preserved
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	var foundAfterRoundtrip bool
	for _, event := range glx2.Events {
		if st, ok := event.Properties["event_subtype"]; ok && st == "separation" {
			foundAfterRoundtrip = true
		}
	}
	assert.True(t, foundAfterRoundtrip, "event_subtype should survive roundtrip")
}

func TestRoundtrip_HeadMetadataPreserved(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR PAF
2 NAME Personal Ancestral File
2 VERS 5.2
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 LANG English
1 FILE my-tree.ged
1 COPR (c) 2023 Test Author
1 NOTE This is a test archive.
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Verify metadata imported
	require.NotNil(t, glx1.ImportMetadata)
	assert.Equal(t, "English", glx1.ImportMetadata.Language)
	assert.Equal(t, "my-tree.ged", glx1.ImportMetadata.SourceFile)
	assert.Equal(t, "(c) 2023 Test Author", glx1.ImportMetadata.Copyright)
	assert.Equal(t, NoteList{"This is a test archive."}, glx1.ImportMetadata.Notes)

	// Export
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)

	// Verify metadata in exported HEAD
	assert.Contains(t, exportedStr, "1 LANG English")
	assert.Contains(t, exportedStr, "1 FILE my-tree.ged")
	assert.Contains(t, exportedStr, "1 COPR (c) 2023 Test Author")
	assert.Contains(t, exportedStr, "1 NOTE This is a test archive.")

	// Re-import and verify preserved
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	require.NotNil(t, glx2.ImportMetadata)
	assert.Equal(t, "English", glx2.ImportMetadata.Language)
	assert.Equal(t, "my-tree.ged", glx2.ImportMetadata.SourceFile)
	assert.Equal(t, "(c) 2023 Test Author", glx2.ImportMetadata.Copyright)
	assert.Equal(t, NoteList{"This is a test archive."}, glx2.ImportMetadata.Notes)
}

// TestRoundtrip_NoSpouseFamilyWithMarriageAndChild documents that a FAM record
// with a MARR event and CHIL but no HUSB/WIFE loses the marriage event on import.
// This is a known limitation: GLX requires at least one participant for relationships.
// In habsburg.ged, 148 of 149 such records are orphaned (no INDI references them).
// Only 1 (F12868) has a child — this test reproduces that case.
func TestRoundtrip_NoSpouseFamilyWithMarriageAndChild(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Jane /Doe/
1 SEX F
1 FAMC @F1@
0 @F1@ FAM
1 CHIL @I1@
1 MARR
2 DATE 11 JUN 1707
2 PLAC Swallowfield, Berkshire, England
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// The child should be imported
	require.Len(t, glx1.Persons, 1, "child should be imported")

	// Known limitation: no marriage relationship or event is created because
	// the FAM has no HUSB/WIFE. The MARR date and place are lost.
	marriageEvents := 0
	for _, event := range glx1.Events {
		if event.Type == EventTypeMarriage {
			marriageEvents++
		}
	}
	assert.Equal(t, 0, marriageEvents,
		"known gap: MARR on no-spouse FAM is dropped (no participants for relationship)")

	// The child also has no parent-child relationship since neither parent is known
	parentChildRels := 0
	for _, rel := range glx1.Relationships {
		if rel.Type == RelationshipTypeParentChild {
			parentChildRels++
		}
	}
	assert.Equal(t, 0, parentChildRels,
		"known gap: child in no-spouse FAM has no parent-child relationship")
}

// TestRoundtrip_MultiMarriageChildrenPreserved tests that children are correctly
// assigned to families when a parent has multiple marriages.
func TestRoundtrip_MultiMarriageChildrenPreserved(t *testing.T) {
	// Father has two marriages, each with 3 children.
	// All 6 children should survive the roundtrip.
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Henry /King/
1 SEX M
1 FAMS @F1@
1 FAMS @F2@
0 @I2@ INDI
1 NAME Catherine /Aragon/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Anne /Boleyn/
1 SEX F
1 FAMS @F2@
0 @I4@ INDI
1 NAME Mary /King/
1 SEX F
1 FAMC @F1@
0 @I5@ INDI
1 NAME Edward /King/
1 SEX M
1 FAMC @F1@
0 @I6@ INDI
1 NAME Elizabeth /King/
1 SEX F
1 FAMC @F1@
0 @I7@ INDI
1 NAME Henry Jr /King/
1 SEX M
1 FAMC @F2@
0 @I8@ INDI
1 NAME George /King/
1 SEX M
1 FAMC @F2@
0 @I9@ INDI
1 NAME Margaret /King/
1 SEX F
1 FAMC @F2@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I4@
1 CHIL @I5@
1 CHIL @I6@
1 MARR
2 DATE 11 JUN 1509
0 @F2@ FAM
1 HUSB @I1@
1 WIFE @I3@
1 CHIL @I7@
1 CHIL @I8@
1 CHIL @I9@
1 MARR
2 DATE 25 JAN 1533
0 TRLR
`
	// Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx1.Persons, 9)

	// Verify all 6 children have parent-child relationships
	parentChildCount := 0
	for _, rel := range glx1.Relationships {
		if isParentChildType(rel.Type) {
			parentChildCount++
		}
	}
	// Each child has 2 parent-child relationships (one per parent) = 12
	assert.Equal(t, 12, parentChildCount, "each of 6 children should have 2 parent-child rels")

	// Export to GEDCOM
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)

	// Count CHIL tags in exported GEDCOM
	chilCount := strings.Count(exportedStr, "\n1 CHIL ")
	assert.Equal(t, 6, chilCount,
		"all 6 children should appear as CHIL in exported GEDCOM; got %d", chilCount)

	// Re-import and verify children are still connected
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	parentChildCount2 := 0
	for _, rel := range glx2.Relationships {
		if isParentChildType(rel.Type) {
			parentChildCount2++
		}
	}
	assert.Equal(t, 12, parentChildCount2,
		"all 12 parent-child relationships should survive roundtrip; got %d", parentChildCount2)
}

// TestRoundtrip_DanglingChildRefsIgnored verifies that CHIL references pointing to
// non-existent INDI records are gracefully skipped. This reproduces the queen.ged
// pattern where 497 FAM CHIL refs point to persons that don't exist in the file.
func TestRoundtrip_DanglingChildRefsIgnored(t *testing.T) {
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
0 @I3@ INDI
1 NAME Real /Child/
1 SEX F
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I999@
1 MARR
2 DATE 1800
0 TRLR
`
	// @I999@ doesn't exist — import should handle gracefully
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx1.Persons, 3, "only 3 real persons should be imported")

	// Only the real child should have parent-child relationships
	parentChildCount := 0
	for _, rel := range glx1.Relationships {
		if isParentChildType(rel.Type) {
			parentChildCount++
		}
	}
	assert.Equal(t, 2, parentChildCount, "real child should have 2 parent-child rels (one per parent)")

	// Export and verify only real child appears
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)
	chilCount := strings.Count(exportedStr, "\n1 CHIL ")
	assert.Equal(t, 1, chilCount, "only real child should appear as CHIL")
}

// TestRoundtrip_MultiFamilyChild tests that a child in two families (e.g., birth
// family + step-family) appears as CHIL in both families after roundtrip.
// Reproduces the bullinger.ged pattern where 11 children in 2 FAMs each lose their
// second CHIL listing.
func TestRoundtrip_MultiFamilyChild(t *testing.T) {
	// Child has FAMC to both F1 (birth family) and F2 (step-family).
	// Both FAMs list the child as CHIL.
	gedcom := `0 HEAD
1 SOUR TEST
2 VERS 1.0
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Father /Smith/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mother /Jones/
1 SEX F
1 FAMS @F1@
1 FAMS @F2@
0 @I3@ INDI
1 NAME StepFather /Brown/
1 SEX M
1 FAMS @F2@
0 @I4@ INDI
1 NAME Child /Smith/
1 SEX M
1 FAMC @F1@
1 FAMC @F2@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I4@
1 MARR
2 DATE 1990
0 @F2@ FAM
1 HUSB @I3@
1 WIFE @I2@
1 CHIL @I4@
1 MARR
2 DATE 2000
0 TRLR
`
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	// Child should have parent-child relationships to all 3 parents
	// Father + Mother from F1, StepFather + Mother from F2 = 4
	// (Mother appears twice since child has FAMC to both families)
	parentChildCount := 0
	for _, rel := range glx1.Relationships {
		if isParentChildType(rel.Type) {
			parentChildCount++
		}
	}
	assert.Equal(t, 4, parentChildCount,
		"child should have parent-child rels to all parents from both families")

	// Export to GEDCOM
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)

	// Child should appear as CHIL in both families
	chilCount := strings.Count(exportedStr, "\n1 CHIL ")
	assert.Equal(t, 2, chilCount,
		"child should appear as CHIL in both families")

	// Re-import: all 4 parent-child relationships should survive
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)

	parentChildCount2 := 0
	for _, rel := range glx2.Relationships {
		if isParentChildType(rel.Type) {
			parentChildCount2++
		}
	}
	assert.Equal(t, 4, parentChildCount2,
		"all 4 parent-child relationships should survive roundtrip")
}

// TestRoundtrip_ResidenceEventNoPlace tests that a RESI record without a PLAC
// sub-record survives the full import → export → re-import cycle as a residence event.
// Verifies: event count, DATE, TYPE, notes, SOUR citations, and no double-export
// with the property-path exportResidenceRecords().
func TestRoundtrip_ResidenceEventNoPlace(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @S1@ SOUR
1 TITL Parish Register
0 @I1@ INDI
1 NAME Heinrich /Bullinger/
1 SEX M
1 RESI
2 DATE 1580
2 TYPE married
2 NOTE Moved after wedding
2 SOUR @S1@
3 PAGE Entry 42
1 RESI Y
0 TRLR
`
	// Step 1: Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	// Should have 2 residence events (one with DATE+TYPE+NOTE+SOUR, one bare RESI Y)
	var datedEvent, bareEvent *Event
	for _, event := range glx1.Events {
		if event.Type == EventTypeResidence {
			if event.Date != "" {
				datedEvent = event
			} else {
				bareEvent = event
			}
		}
	}
	require.NotNil(t, datedEvent, "dated residence event should exist")
	require.NotNil(t, bareEvent, "bare RESI Y event should exist")

	// Verify dated event preserved TYPE as event_subtype property
	assert.Equal(t, "married", datedEvent.Properties["event_subtype"],
		"TYPE should be stored as event_subtype property")

	// Verify notes survived import
	assert.Contains(t, datedEvent.Notes, "Moved after wedding",
		"NOTE should be preserved in event notes")

	// Verify SOUR citation was attached to event properties
	citations, _ := datedEvent.Properties["citations"].([]string)
	assert.NotEmpty(t, citations,
		"SOUR should be stored in event.Properties[citations] for export roundtrip")

	// Step 2: Export to GEDCOM 5.5.1
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err, "export failed")

	exportedStr := string(exported)

	// Verify RESI records appear — exactly 2, not duplicated by exportResidenceRecords
	resiCount := strings.Count(exportedStr, "\n1 RESI")
	assert.Equal(t, 2, resiCount,
		"exactly 2 RESI records expected (no double-export); got %d", resiCount)

	// Verify DATE and TYPE survived export
	assert.Contains(t, exportedStr, "1580", "date should survive export")
	assert.Contains(t, exportedStr, "2 TYPE married", "TYPE should survive export")

	// Verify NOTE survived export
	assert.Contains(t, exportedStr, "NOTE", "NOTE should survive export")
	assert.Contains(t, exportedStr, "Moved after wedding", "NOTE text should survive export")

	// Verify SOUR survived export
	assert.Contains(t, exportedStr, "SOUR", "SOUR should survive export")

	// Step 3: Re-import the exported GEDCOM
	glx2, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err, "re-import failed")

	// Should still have 2 residence events
	var residenceEvents2 int
	var datedEvent2 *Event
	for _, event := range glx2.Events {
		if event.Type == EventTypeResidence {
			residenceEvents2++
			if event.Date != "" {
				datedEvent2 = event
			}
		}
	}
	assert.Equal(t, 2, residenceEvents2,
		"both residence events should survive roundtrip; got %d", residenceEvents2)

	// Verify the dated event's TYPE and notes survived the full cycle
	require.NotNil(t, datedEvent2, "dated residence event should survive re-import")
	assert.Equal(t, "married", datedEvent2.Properties["event_subtype"],
		"TYPE should survive full roundtrip")
	assert.Contains(t, datedEvent2.Notes, "Moved after wedding",
		"NOTE should survive full roundtrip")
}

// TestRoundtrip_MultiplePersonNotes verifies that multiple NOTE records on a
// person survive export→re-import as separate NoteList entries.
func TestRoundtrip_MultiplePersonNotes(t *testing.T) {
	gedcom := `0 HEAD
1 SOUR TEST
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Jane /Doe/
2 GIVN Jane
2 SURN Doe
1 SEX F
1 NOTE First note about Jane
1 NOTE Second note about Jane
1 NOTE Third note about Jane
0 TRLR
`
	// Step 1: Import
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	// Find the person and verify multiple notes imported
	var person1 *Person
	for _, p := range glx1.Persons {
		if p != nil {
			person1 = p

			break
		}
	}
	require.NotNil(t, person1, "should have at least one person")
	require.Len(t, person1.Notes, 3, "should import 3 separate notes")
	assert.Equal(t, "First note about Jane", person1.Notes[0])
	assert.Equal(t, "Second note about Jane", person1.Notes[1])
	assert.Equal(t, "Third note about Jane", person1.Notes[2])

	// Step 2: Export back to GEDCOM
	exported, _, err := ExportGEDCOM(glx1, GEDCOM551, nil)
	require.NoError(t, err, "export failed")

	// Verify 3 separate NOTE records in the exported GEDCOM
	noteCount := strings.Count(string(exported), "\n1 NOTE ")
	assert.Equal(t, 3, noteCount, "should export 3 separate NOTE records")

	// Step 3: Re-import
	glx2, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
	require.NoError(t, err, "re-import failed")

	var person2 *Person
	for _, p := range glx2.Persons {
		if p != nil {
			person2 = p

			break
		}
	}
	require.NotNil(t, person2, "should have person after re-import")
	require.Len(t, person2.Notes, 3, "should preserve 3 separate notes through roundtrip")
	assert.Equal(t, "First note about Jane", person2.Notes[0])
	assert.Equal(t, "Second note about Jane", person2.Notes[1])
	assert.Equal(t, "Third note about Jane", person2.Notes[2])
}

// TestRoundtrip_ExtensionTags asserts that GEDCOM 7.0 vendor extension tags
// inside INDI and FAM records survive an import → export → re-import cycle.
// Closes #289.
func TestRoundtrip_ExtensionTags(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 _FSFTID FS-XXX-1234
1 _PROFILE FamilySearch
2 ID FS-XXX-1234
2 URL https://familysearch.org/p/FS-XXX-1234
1 FAMS @F1@
0 @I2@ INDI
1 NAME Mary /Jones/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 _FAMTAG FamilyExt
2 NOTE family-level extension note
0 TRLR
`

	// Import → re-import via export roundtrip
	glx1, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err, "first import failed")

	exported, _, err := ExportGEDCOM(glx1, GEDCOM70, nil)
	require.NoError(t, err, "export failed")

	glx2, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
	require.NoError(t, err, "re-import failed")

	// Locate John Smith in the re-imported archive
	var john *Person
	for _, p := range glx2.Persons {
		if strings.Contains(getPersonNameValue(p), "John") {
			john = p

			break
		}
	}
	require.NotNil(t, john, "John Smith missing after roundtrip")

	exts, ok := john.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok, "expected gedcom_extensions on John after roundtrip")
	require.Len(t, exts, 2, "expected both _FSFTID and _PROFILE preserved")

	// _FSFTID is value-only
	fsftid := exts[0].(map[string]any)
	assert.Equal(t, "_FSFTID", fsftid[ExtensionEntryTag])
	assert.Equal(t, "FS-XXX-1234", fsftid[ExtensionEntryValue])

	// _PROFILE has nested subrecords
	profile := exts[1].(map[string]any)
	assert.Equal(t, "_PROFILE", profile[ExtensionEntryTag])
	assert.Equal(t, "FamilySearch", profile[ExtensionEntryValue])

	subs := profile[ExtensionEntrySubrecords].([]any)
	require.Len(t, subs, 2, "expected ID and URL subrecords on _PROFILE")
	assert.Equal(t, map[string]any{ExtensionEntryTag: "ID", ExtensionEntryValue: "FS-XXX-1234"}, subs[0])
	assert.Equal(t, map[string]any{ExtensionEntryTag: "URL", ExtensionEntryValue: "https://familysearch.org/p/FS-XXX-1234"}, subs[1])

	// Locate the marriage relationship
	var rel *Relationship
	for _, r := range glx2.Relationships {
		if r.Type == RelationshipTypeMarriage {
			rel = r

			break
		}
	}
	require.NotNil(t, rel, "marriage relationship missing after roundtrip")

	famExts, ok := rel.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok, "expected gedcom_extensions on Relationship after roundtrip")
	require.Len(t, famExts, 1)

	famtag := famExts[0].(map[string]any)
	assert.Equal(t, "_FAMTAG", famtag[ExtensionEntryTag])
	assert.Equal(t, "FamilyExt", famtag[ExtensionEntryValue])

	famSubs := famtag[ExtensionEntrySubrecords].([]any)
	require.Len(t, famSubs, 1)
	assert.Equal(t, map[string]any{ExtensionEntryTag: "NOTE", ExtensionEntryValue: "family-level extension note"}, famSubs[0])
}

func quayFixture(version, citations string) string {
	return "0 HEAD\n1 GEDC\n2 VERS " + version + "\n0 @S1@ SOUR\n1 TITL Register\n0 @I1@ INDI\n1 NAME Jane /Doe/\n1 BIRT\n2 DATE 1 JAN 1900\n" + citations + "0 TRLR\n"
}

func TestCitationQUAYRoundTrip(t *testing.T) {
	for _, version := range []GEDCOMVersion{GEDCOM551, GEDCOM70} {
		versionText := "5.5.1"
		if version == GEDCOM70 {
			versionText = "7.0"
		}
		t.Run(versionText, func(t *testing.T) {
			var citations strings.Builder
			for q := range 4 {
				fmt.Fprintf(&citations, "2 SOUR @S1@\n3 PAGE p%d\n3 QUAY %d\n", q, q)
			}
			citations.WriteString("2 SOUR @S1@\n3 PAGE absent\n3 NOTE ordinary note\n")
			archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture(versionText, citations.String())), nil)
			require.NoError(t, err)
			require.NotEmpty(t, archive.Citations)
			for _, assertion := range archive.Assertions {
				assert.Empty(t, assertion.Confidence)
			}
			// Exercise the persisted YAML numeric representation, not only in-memory Go values.
			data, err := yaml.Marshal(archive)
			require.NoError(t, err)
			persisted := &GLXFile{}
			require.NoError(t, yaml.Unmarshal(data, persisted))
			exported, _, err := ExportGEDCOM(persisted, version, nil)
			require.NoError(t, err)
			assert.Equal(t, 4, strings.Count(string(exported), " QUAY "))
			roundtrip, _, err := ImportGEDCOM(strings.NewReader(string(exported)), nil)
			require.NoError(t, err)
			require.NotEmpty(t, roundtrip.Citations)
			seen := map[string]bool{}
			for _, c := range roundtrip.Citations {
				page, _ := getStringProperty(c.Properties, "locator")
				seen[page] = true
				if page == "absent" {
					_, present := c.Properties["gedcom_quay"]
					assert.False(t, present)
					assert.Contains(t, c.Notes, "ordinary note")
				} else {
					require.Len(t, page, 2)
					assert.Equal(t, int(page[1]-'0'), c.Properties["gedcom_quay"])
				}
			}
			assert.Len(t, seen, 5)
			for _, assertion := range roundtrip.Assertions {
				assert.Empty(t, assertion.Confidence)
			}
		})
	}
}

func TestCitationQUAYLegacyAndInvalid(t *testing.T) {
	for _, q := range []string{"", "4", "-1", "03", "bad"} {
		t.Run("invalid_"+q, func(t *testing.T) {
			archive, result, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 QUAY "+q+"\n")), nil)
			require.NoError(t, err)
			require.NotEmpty(t, result.Statistics.Warnings)
			require.NotEmpty(t, archive.Citations)
			for _, c := range archive.Citations {
				assert.NotContains(t, c.Properties, "gedcom_quay")
				assert.Contains(t, c.Notes, "GEDCOM QUAY: "+q)
			}
		})
	}
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 NOTE GEDCOM QUAY: 3\n")), nil)
	require.NoError(t, err)
	exported, _, err := ExportGEDCOM(archive, GEDCOM551, nil)
	require.NoError(t, err)
	assert.Contains(t, string(exported), "NOTE GEDCOM QUAY: 3")
	assert.NotContains(t, string(exported), "3 QUAY")
	// A repeated QUAY must not silently replace the first assessment.
	archive, result, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 QUAY 0\n3 QUAY 3\n")), nil)
	require.NoError(t, err)
	require.NotEmpty(t, result.Statistics.Warnings)
	for _, warning := range result.Statistics.Warnings {
		if warning.Tag == GedcomTagQuay {
			assert.Contains(t, warning.Message, "Repeated QUAY")
		}
	}
	for _, c := range archive.Citations {
		assert.Equal(t, 0, c.Properties["gedcom_quay"])
		assert.Contains(t, c.Notes, "GEDCOM QUAY: 3")
	}
}

func TestCitationQUAYExportValues(t *testing.T) {
	for i, value := range []any{0, int64(1), float64(2), uint(3), 4, -1, 2.5, "3", nil} {
		t.Run(fmt.Sprintf("%T_%v", value, value), func(t *testing.T) {
			archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 PAGE p\n")), nil)
			require.NoError(t, err)
			for _, c := range archive.Citations {
				c.Properties["gedcom_quay"] = value
			}
			output, result, err := ExportGEDCOM(archive, GEDCOM551, nil)
			require.NoError(t, err)
			if i < 4 {
				expected := []string{"0", "1", "2", "3"}[i]
				assert.Contains(t, string(output), "3 QUAY "+expected)
			} else {
				assert.NotContains(t, string(output), "3 QUAY")
				assert.NotEmpty(t, result.Statistics.Warnings)
				for _, w := range result.Statistics.Warnings {
					if w.EntityType == EntityTypeCitations {
						assert.Contains(t, archive.Citations, w.EntityID)
					}
				}
				assert.Contains(t, string(output), "NOTE GEDCOM QUAY:")
			}
		})
	}
}

func TestCitationQUAYOnlyDetailAndCustomVocabulary(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 QUAY 0\n")), nil)
	require.NoError(t, err)
	require.NotEmpty(t, archive.Citations, "zero alone is citation detail, not a bare source")
	for _, c := range archive.Citations {
		assert.Equal(t, 0, c.Properties["gedcom_quay"])
		assert.Empty(t, c.Notes)
	}
	// Export respects vocabulary mapping rather than hard-coding the property name.
	archive.CitationProperties["quality_original"] = archive.CitationProperties["gedcom_quay"]
	delete(archive.CitationProperties, "gedcom_quay")
	for _, c := range archive.Citations {
		c.Properties["quality_original"] = c.Properties["gedcom_quay"]
		delete(c.Properties, "gedcom_quay")
	}
	output, _, err := ExportGEDCOM(archive, GEDCOM70, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "3 QUAY 0")
}

func TestCitationQUAYAmbiguousMappings(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 QUAY 0\n")), nil)
	require.NoError(t, err)
	archive.CitationProperties["quality_original"] = archive.CitationProperties["gedcom_quay"]
	for _, c := range archive.Citations {
		c.Properties["quality_original"] = 3
	}
	output, result, err := ExportGEDCOM(archive, GEDCOM551, nil)
	require.NoError(t, err)
	// Import cannot map an ambiguous QUAY, so export writes none either: both
	// values become notes, and a round trip leaves them as notes.
	assert.NotContains(t, string(output), "QUAY 0")
	assert.NotContains(t, string(output), "QUAY 3")
	assert.Contains(t, string(output), "NOTE GEDCOM QUAY: 0")
	assert.Contains(t, string(output), "NOTE GEDCOM QUAY: 3")
	require.NotEmpty(t, result.Statistics.Warnings)
	roundtrip, _, err := ImportGEDCOM(strings.NewReader(string(output)), nil)
	require.NoError(t, err)
	for _, c := range roundtrip.Citations {
		assert.ElementsMatch(t, NoteList{"GEDCOM QUAY: 0", "GEDCOM QUAY: 3"}, c.Notes)
	}
	for _, w := range result.Statistics.Warnings {
		if w.EntityType == EntityTypeCitations {
			assert.Contains(t, archive.Citations, w.EntityID)
		}
	}
}

// Person-attached citations use the same serializer as event/assertion citations.
// Direct INDI.SOUR import is not currently supported; build this attachment explicitly.
func TestCitationQUAYPersonLevel(t *testing.T) {
	for _, version := range []GEDCOMVersion{GEDCOM551, GEDCOM70} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 PAGE valid\n3 QUAY 0\n3 QUAY 3\n2 SOUR @S1@\n3 PAGE invalid\n3 QUAY bad\n2 SOUR @S1@\n3 PAGE legacy\n3 NOTE GEDCOM QUAY: 2\n")), nil)
			require.NoError(t, err)
			// Import can create separate event/assertion citations; attach one per page.
			pages := map[string]bool{}
			ids := []string{}
			for _, id := range sortedKeys(archive.Citations) {
				page, _ := getStringProperty(archive.Citations[id].Properties, "locator")
				if !pages[page] {
					ids = append(ids, id)
					pages[page] = true
				}
			}
			require.Len(t, ids, 3)
			archive.Events = nil
			archive.Assertions = nil
			for _, person := range archive.Persons {
				person.Properties[PropertyCitations] = ids
			}
			output, _, err := ExportGEDCOM(archive, version, nil)
			require.NoError(t, err)
			assert.Equal(t, 3, strings.Count(string(output), "1 SOUR @"))
			assert.Equal(t, 1, strings.Count(string(output), "2 QUAY "))
			assert.Contains(t, string(output), "2 QUAY 0")
			for _, note := range []string{"3", "bad", "2"} {
				assert.Contains(t, string(output), "2 NOTE GEDCOM QUAY: "+note)
			}
			for _, page := range []string{"valid", "invalid", "legacy"} {
				assert.Contains(t, string(output), "2 PAGE "+page)
			}
		})
	}
}

func TestCitationQUAYFamilyRoundTrip(t *testing.T) {
	for _, version := range []GEDCOMVersion{GEDCOM551, GEDCOM70} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			versionText := "5.5.1"
			if version == GEDCOM70 {
				versionText = "7.0"
			}
			input := "0 HEAD\n1 GEDC\n2 VERS " + versionText + "\n0 @S1@ SOUR\n1 TITL Register\n0 @I1@ INDI\n1 NAME Jane /Doe/\n0 @F1@ FAM\n1 WIFE @I1@\n1 SOUR @S1@\n2 PAGE zero\n2 QUAY 0\n2 QUAY 3\n1 SOUR @S1@\n2 PAGE high\n2 QUAY 3\n1 SOUR @S1@\n2 PAGE legacy\n2 NOTE GEDCOM QUAY: 2\n0 TRLR\n"
			archive, _, err := ImportGEDCOM(strings.NewReader(input), nil)
			require.NoError(t, err)
			output, _, err := ExportGEDCOM(archive, version, nil)
			require.NoError(t, err)
			assert.Equal(t, 2, strings.Count(string(output), "2 QUAY "))
			roundtrip, _, err := ImportGEDCOM(strings.NewReader(string(output)), nil)
			require.NoError(t, err)
			require.Len(t, roundtrip.Citations, 3)
			for _, c := range roundtrip.Citations {
				page, _ := getStringProperty(c.Properties, "locator")
				switch page {
				case "zero":
					assert.Equal(t, 0, c.Properties["gedcom_quay"])
					assert.Contains(t, c.Notes, "GEDCOM QUAY: 3")
				case "high":
					assert.Equal(t, 3, c.Properties["gedcom_quay"])
				case "legacy":
					_, present := c.Properties["gedcom_quay"]
					assert.False(t, present)
					assert.Contains(t, c.Notes, "GEDCOM QUAY: 2")
				default:
					t.Errorf("unexpected page %q", page)
				}
			}
			require.Len(t, roundtrip.Relationships, 1)
			for _, rel := range roundtrip.Relationships {
				assert.Len(t, extractStringList(rel.Properties[PropertyCitations]), 3)
			}
		})
	}
}

func TestCitationQUAYAmbiguousImportMapping(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 QUAY 0\n")), nil)
	require.NoError(t, err)
	for _, key := range []string{"quality_copy", "quality_other"} {
		archive.CitationProperties[key] = archive.CitationProperties["gedcom_quay"]
	}
	for range 20 {
		index := buildGEDCOMIndex(archive)
		_, mapped := index.CitationProperties[GedcomTagQuay]
		require.False(t, mapped)
		conv := &ConversionContext{GLX: archive, GEDCOMIndex: index, Logger: NewImportLogger(nil), SourceIDMap: map[string]string{"@S1@": "source-1"}}
		result, err := prepareCitationFromSOUR(&GEDCOMRecord{Tag: GedcomTagSour, Value: "@S1@", SubRecords: []*GEDCOMRecord{{Tag: GedcomTagQuay, Value: "0"}}}, conv)
		require.NoError(t, err)
		citation := archive.Citations[result.CitationID]
		require.NotNil(t, citation)
		assert.Empty(t, citation.Properties)
		assert.Contains(t, citation.Notes, "GEDCOM QUAY: 0")
		require.NotEmpty(t, conv.Stats.Warnings)
		assert.Contains(t, conv.Stats.Warnings[0].Message, "no unique QUAY property")
	}
}

// Citation DATA is exported from the vocabulary's DATE/TEXT keys, the same
// keys import stores it under, so the date and transcription round-trip.
func TestCitationDataRoundTrip(t *testing.T) {
	for _, version := range []GEDCOMVersion{GEDCOM551, GEDCOM70} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 PAGE p\n3 DATA\n4 DATE 2 JAN 1850\n4 TEXT Birth entry\n")), nil)
			require.NoError(t, err)
			output, _, err := ExportGEDCOM(archive, version, nil)
			require.NoError(t, err)
			assert.Contains(t, string(output), "3 DATA\n4 DATE 2 JAN 1850\n4 TEXT Birth entry\n")
			roundtrip, _, err := ImportGEDCOM(strings.NewReader(string(output)), nil)
			require.NoError(t, err)
			require.NotEmpty(t, roundtrip.Citations)
			for _, c := range roundtrip.Citations {
				assert.Equal(t, "1850-01-02", c.Properties["source_date"])
				assert.Equal(t, "Birth entry", c.Properties["text_from_source"])
			}
		})
	}
}

// A QUAY that cannot be written follows the citation's own notes, so their
// order is unchanged by a round trip.
func TestCitationQUAYFallbackNoteFollowsCitationNotes(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 PAGE p\n3 NOTE Copy\n")), nil)
	require.NoError(t, err)
	for _, c := range archive.Citations {
		c.Properties["gedcom_quay"] = 7
	}
	output, _, err := ExportGEDCOM(archive, GEDCOM551, nil)
	require.NoError(t, err)
	assert.Contains(t, string(output), "3 NOTE Copy\n3 NOTE GEDCOM QUAY: 7\n")
}

// A vocabulary with no QUAY property keeps QUAY as a note without a warning
// per citation; only an ambiguous mapping or a bad value warns.
func TestCitationQUAYUnmappedVocabularyIsQuiet(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "2 SOUR @S1@\n3 PAGE p\n")), nil)
	require.NoError(t, err)
	delete(archive.CitationProperties, "gedcom_quay")
	conv := &ConversionContext{GLX: archive, GEDCOMIndex: buildGEDCOMIndex(archive), Logger: NewImportLogger(nil), SourceIDMap: map[string]string{"@S1@": "source-1"}}
	result, err := prepareCitationFromSOUR(&GEDCOMRecord{Tag: GedcomTagSour, Value: "@S1@", SubRecords: []*GEDCOMRecord{{Tag: GedcomTagQuay, Value: "2"}}}, conv)
	require.NoError(t, err)
	citation := archive.Citations[result.CitationID]
	require.NotNil(t, citation)
	assert.Contains(t, citation.Notes, "GEDCOM QUAY: 2")
	for _, w := range conv.Stats.Warnings {
		assert.NotContains(t, w.Message, "QUAY")
	}
}

// A GEDCOM tag mapped by several properties resolves to the first key in
// sorted order on every import, never to a map-iteration winner.
func TestGEDCOMPropertyIndexIsDeterministic(t *testing.T) {
	archive, _, err := ImportGEDCOM(strings.NewReader(quayFixture("5.5.1", "")), nil)
	require.NoError(t, err)
	for _, key := range []string{"zz_page", "aa_page", "mm_page"} {
		archive.CitationProperties[key] = archive.CitationProperties["locator"]
	}
	for range 20 {
		assert.Equal(t, "aa_page", buildGEDCOMIndex(archive).CitationProperties[GedcomTagPage])
	}
}

// Direct relationship evidence emits each identical SOUR once, as events do.
func TestEntitySourceRefsAreDeduplicated(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @S1@ SOUR\n1 TITL Register\n0 @I1@ INDI\n1 NAME Jane /Doe/\n0 @F1@ FAM\n1 WIFE @I1@\n1 SOUR @S1@\n2 PAGE p\n0 TRLR\n"
	archive, _, err := ImportGEDCOM(strings.NewReader(input), nil)
	require.NoError(t, err)
	require.Len(t, archive.Relationships, 1)
	for _, rel := range archive.Relationships {
		ids := extractStringList(rel.Properties[PropertyCitations])
		require.Len(t, ids, 1)
		rel.Properties[PropertyCitations] = []string{ids[0], ids[0]}
		rel.Properties[PropertySources] = []string{"source-1", "source-1"}
	}
	output, _, err := ExportGEDCOM(archive, GEDCOM551, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(output), "1 SOUR @S1@\n2 PAGE p\n"))
	assert.Equal(t, 2, strings.Count(string(output), "1 SOUR @S1@\n"), "one bare and one detailed reference")
}

// TestGEDCOMRoundtrip_AnnulmentEndEvent checks that a marriage ended by an
// annulment exports as ANUL and re-imports with the annulment linked as the
// relationship's end_event (#1498).
func TestGEDCOMRoundtrip_AnnulmentEndEvent(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Hans /Roth/
1 SEX M
1 FAMS @F1@
0 @I2@ INDI
1 NAME Anna /Seitz/
1 SEX F
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 7 JUL 1691
1 ANUL
2 DATE 25 JAN 1694
0 TRLR
`
	endEventType := func(archive *GLXFile) string {
		t.Helper()
		require.Len(t, archive.Relationships, 1)
		for _, rel := range archive.Relationships {
			require.NotEmpty(t, rel.EndEvent, "relationship has no end_event")
			event := archive.Events[rel.EndEvent]
			require.NotNil(t, event)

			return event.Type
		}

		return ""
	}

	imported, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	assert.Equal(t, EventTypeAnnulment, endEventType(imported))

	exported, _, err := ExportGEDCOM(imported, GEDCOM551, nil)
	require.NoError(t, err)
	exportedStr := string(exported)
	assert.Contains(t, exportedStr, "1 ANUL\n")
	assert.NotContains(t, exportedStr, "1 DIV")

	roundtripped, _, err := ImportGEDCOM(strings.NewReader(exportedStr), nil)
	require.NoError(t, err)
	assert.Equal(t, EventTypeAnnulment, endEventType(roundtripped))
}
