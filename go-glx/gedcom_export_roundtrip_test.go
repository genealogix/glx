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

// ============================================================================
// Roundtrip tests: Import GEDCOM → Export GEDCOM → Re-import GEDCOM
// Verifies data preservation through the full cycle.
// ============================================================================

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
	assert.Equal(t, len(glx1.Persons), len(glx2.Persons), "person count mismatch after roundtrip")
	assert.Equal(t, len(glx1.Relationships), len(glx2.Relationships), "relationship count mismatch after roundtrip")

	// Step 5: Verify key data preserved
	// Find John Smith in the re-imported data
	var foundJohn, foundMary, foundJames bool
	for _, person := range glx2.Persons {
		name := getPersonNameValue(person)
		switch {
		case strings.Contains(name, "John") && strings.Contains(name, "Smith"):
			foundJohn = true
			assert.Equal(t, "male", testGetPersonGender(person))
		case strings.Contains(name, "Mary") && strings.Contains(name, "Jones"):
			foundMary = true
			assert.Equal(t, "female", testGetPersonGender(person))
		case strings.Contains(name, "James") && strings.Contains(name, "Smith"):
			foundJames = true
			assert.Equal(t, "male", testGetPersonGender(person))
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

	assert.Equal(t, len(glx1.Persons), len(glx2.Persons))

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

	assert.Equal(t, len(glx1.Sources), len(glx2.Sources), "source count mismatch")
	assert.Equal(t, len(glx1.Repositories), len(glx2.Repositories), "repository count mismatch")
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

	assert.Equal(t, len(glx1.Persons), len(glx2.Persons), "person count mismatch")

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

// testGetPersonGender extracts the gender from a person's properties
func testGetPersonGender(person *Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}

	genderRaw, ok := person.Properties[PersonPropertyGender]
	if !ok {
		return ""
	}

	if g, ok := genderRaw.(string); ok {
		return g
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
	assert.Equal(t, "This is a test archive.", glx1.ImportMetadata.Notes)

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
	assert.Equal(t, "This is a test archive.", glx2.ImportMetadata.Notes)
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
