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
// formatGEDCOMDate tests
// ============================================================================

func TestFormatGEDCOMDate(t *testing.T) {
	tests := []struct {
		name     string
		input    DateString
		expected string
	}{
		{
			name:     "empty date",
			input:    "",
			expected: "",
		},
		{
			name:     "year only",
			input:    "1850",
			expected: "1850",
		},
		{
			name:     "year-month",
			input:    "1850-03",
			expected: "MAR 1850",
		},
		{
			name:     "full date",
			input:    "1850-03-15",
			expected: "15 MAR 1850",
		},
		{
			name:     "ABT qualifier with full date",
			input:    "ABT 1850-03-15",
			expected: "ABT 15 MAR 1850",
		},
		{
			name:     "ABT qualifier with year",
			input:    "ABT 1850",
			expected: "ABT 1850",
		},
		{
			name:     "BEF qualifier",
			input:    "BEF 1920-01-15",
			expected: "BEF 15 JAN 1920",
		},
		{
			name:     "AFT qualifier",
			input:    "AFT 1900-12",
			expected: "AFT DEC 1900",
		},
		{
			name:     "CAL qualifier",
			input:    "CAL 1880-06-01",
			expected: "CAL 1 JUN 1880",
		},
		{
			name:     "BET range with years",
			input:    "BET 1880 AND 1890",
			expected: "BET 1880 AND 1890",
		},
		{
			name:     "BET range with full dates",
			input:    "BET 1880-01-01 AND 1890-12-31",
			expected: "BET 1 JAN 1880 AND 31 DEC 1890",
		},
		{
			name:     "FROM TO range",
			input:    "FROM 1880 TO 1890",
			expected: "FROM 1880 TO 1890",
		},
		{
			name:     "FROM TO with dates",
			input:    "FROM 1900-06 TO 1950-12",
			expected: "FROM JUN 1900 TO DEC 1950",
		},
		{
			name:     "FROM open-ended",
			input:    "FROM 1900",
			expected: "FROM 1900",
		},
		{
			name:     "all months",
			input:    "2000-01-01",
			expected: "1 JAN 2000",
		},
		{
			name:     "december",
			input:    "2000-12-25",
			expected: "25 DEC 2000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGEDCOMDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// serializeGEDCOMRecords tests
// ============================================================================

func TestSerializeGEDCOMRecords_SimpleRecord(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			Tag: "HEAD",
		},
	}

	result := string(serializeGEDCOMRecords(records))
	assert.Equal(t, "0 HEAD\n", result)
}

func TestSerializeGEDCOMRecords_RecordWithXRef(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			XRef: "@I1@",
			Tag:  "INDI",
		},
	}

	result := string(serializeGEDCOMRecords(records))
	assert.Equal(t, "0 @I1@ INDI\n", result)
}

func TestSerializeGEDCOMRecords_NestedRecords(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			XRef: "@I1@",
			Tag:  "INDI",
			SubRecords: []*GEDCOMRecord{
				{
					Tag:   "NAME",
					Value: "John /Smith/",
					SubRecords: []*GEDCOMRecord{
						{Tag: "GIVN", Value: "John"},
						{Tag: "SURN", Value: "Smith"},
					},
				},
				{Tag: "SEX", Value: "M"},
			},
		},
	}

	result := string(serializeGEDCOMRecords(records))
	expected := "0 @I1@ INDI\n1 NAME John /Smith/\n2 GIVN John\n2 SURN Smith\n1 SEX M\n"
	assert.Equal(t, expected, result)
}

func TestSerializeGEDCOMRecords_LongTextSplitting(t *testing.T) {
	// Create a value longer than 248 characters
	longValue := strings.Repeat("A", 300)

	records := []*GEDCOMRecord{
		{
			Tag:   "NOTE",
			Value: longValue,
		},
	}

	result := string(serializeGEDCOMRecords(records))
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

	// Should have at least 2 lines (first line + CONC)
	assert.GreaterOrEqual(t, len(lines), 2)

	// First line should start with "0 NOTE "
	assert.True(t, strings.HasPrefix(lines[0], "0 NOTE "))

	// Second line should be a CONC continuation
	assert.True(t, strings.HasPrefix(lines[1], "1 CONC "))
}

func TestSerializeGEDCOMRecords_NewlineInValue(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			Tag:   "NOTE",
			Value: "Line one\nLine two\nLine three",
		},
	}

	result := string(serializeGEDCOMRecords(records))
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

	assert.Len(t, lines, 3)
	assert.Equal(t, "0 NOTE Line one", lines[0])
	assert.Equal(t, "1 CONT Line two", lines[1])
	assert.Equal(t, "1 CONT Line three", lines[2])
}

func TestSerializeGEDCOMRecords_EmptyValue(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			Tag: "BIRT",
			SubRecords: []*GEDCOMRecord{
				{Tag: "DATE", Value: "1 JAN 1900"},
			},
		},
	}

	result := string(serializeGEDCOMRecords(records))
	expected := "0 BIRT\n1 DATE 1 JAN 1900\n"
	assert.Equal(t, expected, result)
}

func TestSerializeGEDCOMRecords_FullDocument(t *testing.T) {
	records := []*GEDCOMRecord{
		{
			Tag: "HEAD",
			SubRecords: []*GEDCOMRecord{
				{
					Tag: "GEDC",
					SubRecords: []*GEDCOMRecord{
						{Tag: "VERS", Value: "5.5.1"},
					},
				},
			},
		},
		{Tag: "TRLR"},
	}

	result := string(serializeGEDCOMRecords(records))
	expected := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 TRLR\n"
	assert.Equal(t, expected, result)
}

// ============================================================================
// resolvePlaceString tests
// ============================================================================

func TestResolvePlaceString_Simple(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{
				"place-1": {Name: "Springfield"},
			},
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	result := resolvePlaceString("place-1", expCtx)
	assert.Equal(t, "Springfield", result)
}

func TestResolvePlaceString_HierarchicalChain(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{
				"place-1": {Name: "Springfield", ParentID: "place-2"},
				"place-2": {Name: "Sangamon County", ParentID: "place-3"},
				"place-3": {Name: "Illinois", ParentID: "place-4"},
				"place-4": {Name: "USA"},
			},
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	result := resolvePlaceString("place-1", expCtx)
	assert.Equal(t, "Springfield, Sangamon County, Illinois, USA", result)
}

func TestResolvePlaceString_CircularReference(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{
				"place-1": {Name: "City A", ParentID: "place-2"},
				"place-2": {Name: "Region B", ParentID: "place-1"}, // circular!
			},
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	result := resolvePlaceString("place-1", expCtx)
	// Should stop at the circular reference
	assert.Equal(t, "City A, Region B", result)
	// Should have a warning
	assert.Len(t, expCtx.Stats.Warnings, 1)
	assert.Contains(t, expCtx.Stats.Warnings[0].Message, "circular")
}

func TestResolvePlaceString_NonexistentPlace(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{},
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	result := resolvePlaceString("nonexistent", expCtx)
	assert.Equal(t, "", result)
}

func TestResolvePlaceString_CacheHit(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{
				"place-1": {Name: "Springfield"},
			},
		},
		PlaceStrings: map[string]string{
			"place-1": "Cached Value",
		},
		Stats: ExportStatistics{},
	}

	result := resolvePlaceString("place-1", expCtx)
	assert.Equal(t, "Cached Value", result)
}

// ============================================================================
// buildExportIndex tests
// ============================================================================

func TestBuildExportIndex(t *testing.T) {
	glx := &GLXFile{
		EventTypes: map[string]*EventType{
			"birth":    {Label: "Birth", GEDCOM: "BIRT"},
			"death":    {Label: "Death", GEDCOM: "DEAT"},
			"marriage": {Label: "Marriage", GEDCOM: "MARR"},
			"custom":   {Label: "Custom"}, // no GEDCOM mapping
		},
		RelationshipTypes: map[string]*RelationshipType{
			"marriage": {Label: "Marriage", GEDCOM: "MARR"},
		},
		PersonProperties: map[string]*PropertyDefinition{
			"occupation": {Label: "Occupation", GEDCOM: "OCCU"},
			"gender":     {Label: "Gender"}, // no GEDCOM mapping
		},
		EventProperties:      map[string]*PropertyDefinition{},
		CitationProperties:   map[string]*PropertyDefinition{},
		SourceProperties:     map[string]*PropertyDefinition{},
		RepositoryProperties: map[string]*PropertyDefinition{},
		MediaProperties:      map[string]*PropertyDefinition{},
	}

	index := buildExportIndex(glx)

	// Event types should map GLX key -> GEDCOM tag
	assert.Equal(t, "BIRT", index.EventTypes["birth"])
	assert.Equal(t, "DEAT", index.EventTypes["death"])
	assert.Equal(t, "MARR", index.EventTypes["marriage"])
	assert.Empty(t, index.EventTypes["custom"]) // no mapping

	// Relationship types
	assert.Equal(t, "MARR", index.RelationshipTypes["marriage"])

	// Person properties
	assert.Equal(t, "OCCU", index.PersonProperties["occupation"])
	assert.Empty(t, index.PersonProperties["gender"]) // no mapping
}

func TestBuildExportIndex_FromStandardVocabularies(t *testing.T) {
	glx := &GLXFile{}
	err := LoadStandardVocabulariesIntoGLX(glx)
	require.NoError(t, err)

	index := buildExportIndex(glx)

	// Verify some standard mappings
	assert.Equal(t, "BIRT", index.EventTypes["birth"])
	assert.Equal(t, "DEAT", index.EventTypes["death"])
	assert.Equal(t, "BAPM", index.EventTypes["baptism"])
	assert.Equal(t, "BURI", index.EventTypes["burial"])
}

// ============================================================================
// exportRepository tests
// ============================================================================

func TestExportRepository_Basic(t *testing.T) {
	expCtx := &ExportContext{
		RepositoryXRefMap: map[string]string{
			"repo-1": "@R1@",
		},
		ExportIndex: &ExportIndex{
			RepositoryProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	repo := &Repository{
		Name:       "National Archives",
		Address:    "700 Pennsylvania Ave NW",
		City:       "Washington",
		State:      "DC",
		PostalCode: "20408",
		Country:    "USA",
		Website:    "https://www.archives.gov",
		Notes:      "Main facility",
		Properties: make(map[string]any),
	}

	record := exportRepository("repo-1", repo, expCtx)

	assert.Equal(t, "@R1@", record.XRef)
	assert.Equal(t, GedcomTagRepo, record.Tag)

	// Check subrecords
	var foundName, foundAddr, foundWWW, foundNote bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagName:
			foundName = true
			assert.Equal(t, "National Archives", sub.Value)
		case GedcomTagAddr:
			foundAddr = true
			// Check address subrecords
			var hasCity, hasState, hasPost, hasCountry, hasAdr1 bool
			for _, addrSub := range sub.SubRecords {
				switch addrSub.Tag {
				case GedcomTagAdr1:
					hasAdr1 = true
					assert.Equal(t, "700 Pennsylvania Ave NW", addrSub.Value)
				case GedcomTagCity:
					hasCity = true
					assert.Equal(t, "Washington", addrSub.Value)
				case GedcomTagStae:
					hasState = true
					assert.Equal(t, "DC", addrSub.Value)
				case GedcomTagPost:
					hasPost = true
					assert.Equal(t, "20408", addrSub.Value)
				case GedcomTagCtry:
					hasCountry = true
					assert.Equal(t, "USA", addrSub.Value)
				}
			}
			assert.True(t, hasAdr1, "missing ADR1")
			assert.True(t, hasCity, "missing CITY")
			assert.True(t, hasState, "missing STAE")
			assert.True(t, hasPost, "missing POST")
			assert.True(t, hasCountry, "missing CTRY")
		case GedcomTagWww:
			foundWWW = true
			assert.Equal(t, "https://www.archives.gov", sub.Value)
		case GedcomTagNote:
			foundNote = true
			assert.Equal(t, "Main facility", sub.Value)
		}
	}

	assert.True(t, foundName, "missing NAME")
	assert.True(t, foundAddr, "missing ADDR")
	assert.True(t, foundWWW, "missing WWW")
	assert.True(t, foundNote, "missing NOTE")
}

func TestExportRepository_WithPhones(t *testing.T) {
	expCtx := &ExportContext{
		RepositoryXRefMap: map[string]string{
			"repo-1": "@R1@",
		},
		ExportIndex: &ExportIndex{
			RepositoryProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	repo := &Repository{
		Name: "Test Library",
		Properties: map[string]any{
			"phones": []string{"555-1234", "555-5678"},
			"emails": []string{"info@test.com"},
		},
	}

	record := exportRepository("repo-1", repo, expCtx)

	var phoneCount, emailCount int
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagPhon:
			phoneCount++
		case GedcomTagEmail:
			emailCount++
		}
	}

	assert.Equal(t, 2, phoneCount)
	assert.Equal(t, 1, emailCount)
}

func TestExportRepository_Minimal(t *testing.T) {
	expCtx := &ExportContext{
		RepositoryXRefMap: map[string]string{
			"repo-1": "@R1@",
		},
		ExportIndex: &ExportIndex{
			RepositoryProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	repo := &Repository{
		Name:       "Simple Repo",
		Properties: make(map[string]any),
	}

	record := exportRepository("repo-1", repo, expCtx)

	assert.Equal(t, "@R1@", record.XRef)
	assert.Equal(t, GedcomTagRepo, record.Tag)

	// Should have NAME only (no ADDR since no address info)
	assert.Len(t, record.SubRecords, 1)
	assert.Equal(t, GedcomTagName, record.SubRecords[0].Tag)
	assert.Equal(t, "Simple Repo", record.SubRecords[0].Value)
}

// ============================================================================
// ExportGEDCOM end-to-end tests
// ============================================================================

func TestExportGEDCOM_NilFile(t *testing.T) {
	_, _, err := ExportGEDCOM(nil, GEDCOM551, nil)
	assert.ErrorIs(t, err, ErrGLXFileNil)
}

func TestExportGEDCOM_EmptyArchive(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Citations:     make(map[string]*Citation),
		Assertions:    make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	// Should have HEAD and TRLR
	assert.Contains(t, output, "0 HEAD\n")
	assert.Contains(t, output, "0 TRLR\n")

	// Should contain GEDCOM version
	assert.Contains(t, output, "VERS 5.5.1")

	// Should contain source system
	assert.Contains(t, output, "SOUR GLX")

	// No entities exported
	assert.Equal(t, 0, result.Statistics.PersonsExported)
	assert.Equal(t, 0, result.Statistics.RepositoriesExported)
	assert.Equal(t, "5.5.1", result.Version)
}

func TestExportGEDCOM_EmptyArchive_GEDCOM70(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Citations:     make(map[string]*Citation),
		Assertions:    make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM70, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	assert.Contains(t, output, "VERS 7.0")
	assert.Equal(t, "7.0", result.Version)

	// GEDCOM 7.0 should NOT have CHAR record
	assert.NotContains(t, output, "CHAR")
	// GEDCOM 7.0 should NOT have FORM LINEAGE-LINKED
	assert.NotContains(t, output, "LINEAGE-LINKED")
}

func TestExportGEDCOM_WithRepository(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Repositories: map[string]*Repository{
			"repo-1": {
				Name:       "National Archives",
				City:       "Washington",
				Country:    "USA",
				Properties: make(map[string]any),
			},
		},
		Media:      make(map[string]*Media),
		Citations:  make(map[string]*Citation),
		Assertions: make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)

	output := string(data)

	// Should contain the repository
	assert.Contains(t, output, "@R1@ REPO")
	assert.Contains(t, output, "NAME National Archives")
	assert.Contains(t, output, "CITY Washington")
	assert.Contains(t, output, "CTRY USA")

	assert.Equal(t, 1, result.Statistics.RepositoriesExported)
}

func TestExportGEDCOM_DeterministicXRefs(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Repositories: map[string]*Repository{
			"repo-b": {Name: "B Repo", Properties: make(map[string]any)},
			"repo-a": {Name: "A Repo", Properties: make(map[string]any)},
			"repo-c": {Name: "C Repo", Properties: make(map[string]any)},
		},
		Media:      make(map[string]*Media),
		Citations:  make(map[string]*Citation),
		Assertions: make(map[string]*Assertion),
	}

	// Run twice to verify determinism
	data1, _, err1 := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err1)

	data2, _, err2 := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err2)

	assert.Equal(t, string(data1), string(data2), "export should be deterministic")

	output := string(data1)
	// Since keys are sorted alphabetically: repo-a=@R1@, repo-b=@R2@, repo-c=@R3@
	assert.Contains(t, output, "@R1@ REPO")
	assert.Contains(t, output, "@R2@ REPO")
	assert.Contains(t, output, "@R3@ REPO")
}

// ============================================================================
// exportPlaceSubrecords tests
// ============================================================================

func TestExportPlaceSubrecords_Empty(t *testing.T) {
	expCtx := &ExportContext{
		PlaceStrings: make(map[string]string),
	}

	result := exportPlaceSubrecords("", expCtx)
	assert.Nil(t, result)
}

func TestExportPlaceSubrecords_WithCoordinates(t *testing.T) {
	lat := 40.7128
	lon := -74.0060

	expCtx := &ExportContext{
		GLX: &GLXFile{
			Places: map[string]*Place{
				"place-1": {
					Name:      "New York",
					Latitude:  &lat,
					Longitude: &lon,
				},
			},
		},
		PlaceStrings: map[string]string{
			"place-1": "New York, USA",
		},
	}

	records := exportPlaceSubrecords("place-1", expCtx)
	require.Len(t, records, 1)

	placRecord := records[0]
	assert.Equal(t, GedcomTagPlac, placRecord.Tag)
	assert.Equal(t, "New York, USA", placRecord.Value)

	// Should have MAP subrecord
	require.Len(t, placRecord.SubRecords, 1)
	mapRecord := placRecord.SubRecords[0]
	assert.Equal(t, GedcomTagMap, mapRecord.Tag)
	assert.Len(t, mapRecord.SubRecords, 2) // LATI + LONG

	// Check LATI
	assert.Equal(t, GedcomTagLati, mapRecord.SubRecords[0].Tag)
	assert.Contains(t, mapRecord.SubRecords[0].Value, "N")

	// Check LONG (negative = West)
	assert.Equal(t, GedcomTagLong, mapRecord.SubRecords[1].Tag)
	assert.Contains(t, mapRecord.SubRecords[1].Value, "W")
}

// ============================================================================
// getStringSliceProperty tests
// ============================================================================

func TestGetStringSliceProperty(t *testing.T) {
	// Test with []string
	props := map[string]any{
		"phones": []string{"555-1234", "555-5678"},
	}

	result, ok := getStringSliceProperty(props, "phones")
	assert.True(t, ok)
	assert.Equal(t, []string{"555-1234", "555-5678"}, result)

	// Test with []any
	props2 := map[string]any{
		"emails": []any{"a@test.com", "b@test.com"},
	}

	result2, ok2 := getStringSliceProperty(props2, "emails")
	assert.True(t, ok2)
	assert.Equal(t, []string{"a@test.com", "b@test.com"}, result2)

	// Test missing key
	_, ok3 := getStringSliceProperty(props, "missing")
	assert.False(t, ok3)
}

// ============================================================================
// sortedKeys tests
// ============================================================================

func TestSortedKeys(t *testing.T) {
	m := map[string]*Person{
		"c": {},
		"a": {},
		"b": {},
	}

	keys := sortedKeys(m)
	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestSortedKeys_Empty(t *testing.T) {
	m := map[string]*Person{}
	keys := sortedKeys(m)
	assert.Empty(t, keys)
}

// ============================================================================
// assignXRefIDs tests
// ============================================================================

func TestAssignXRefIDs(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-b": {},
				"person-a": {},
			},
			Sources: map[string]*Source{
				"source-1": {},
			},
			Repositories: map[string]*Repository{
				"repo-1": {},
				"repo-2": {},
			},
			Media: map[string]*Media{
				"media-1": {},
			},
		},
		PersonXRefMap:     make(map[string]string),
		SourceXRefMap:     make(map[string]string),
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
	}

	assignXRefIDs(expCtx)

	// Persons sorted: person-a -> @I1@, person-b -> @I2@
	assert.Equal(t, "@I1@", expCtx.PersonXRefMap["person-a"])
	assert.Equal(t, "@I2@", expCtx.PersonXRefMap["person-b"])

	// Sources
	assert.Equal(t, "@S1@", expCtx.SourceXRefMap["source-1"])

	// Repositories sorted: repo-1 -> @R1@, repo-2 -> @R2@
	assert.Equal(t, "@R1@", expCtx.RepositoryXRefMap["repo-1"])
	assert.Equal(t, "@R2@", expCtx.RepositoryXRefMap["repo-2"])

	// Media
	assert.Equal(t, "@O1@", expCtx.MediaXRefMap["media-1"])
}

// ============================================================================
// exportSource tests
// ============================================================================

func TestExportSource_Basic(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: map[string]string{
			"repo-1": "@R1@",
		},
		MediaXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	source := &Source{
		Title:        "Census of 1850",
		Authors:      []string{"U.S. Government"},
		RepositoryID: "repo-1",
		Properties:   make(map[string]any),
	}

	record := exportSource("source-1", source, expCtx)

	assert.Equal(t, "@S1@", record.XRef)
	assert.Equal(t, GedcomTagSour, record.Tag)

	var foundTitl, foundAuth, foundRepo bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagTitl:
			foundTitl = true
			assert.Equal(t, "Census of 1850", sub.Value)
		case GedcomTagAuth:
			foundAuth = true
			assert.Equal(t, "U.S. Government", sub.Value)
		case GedcomTagRepo:
			foundRepo = true
			assert.Equal(t, "@R1@", sub.Value)
		}
	}

	assert.True(t, foundTitl, "missing TITL")
	assert.True(t, foundAuth, "missing AUTH")
	assert.True(t, foundRepo, "missing REPO")
}

func TestExportSource_WithProperties(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: map[string]string{
			"repo-1": "@R1@",
		},
		MediaXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	source := &Source{
		Title:        "Parish Register",
		RepositoryID: "repo-1",
		Date:         "1850-06",
		Properties: map[string]any{
			"publication_info":  "Published by Archives, 2001",
			"abbreviation":     "PR",
			"call_number":      "MS-123",
			"agency":           "Church of England",
			"events_recorded":  []string{"births", "marriages", "deaths"},
		},
	}

	record := exportSource("source-1", source, expCtx)

	var foundPubl, foundAbbr, foundData bool
	var foundCaln bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagPubl:
			foundPubl = true
			assert.Equal(t, "Published by Archives, 2001", sub.Value)
		case GedcomTagAbbr:
			foundAbbr = true
			assert.Equal(t, "PR", sub.Value)
		case GedcomTagRepo:
			// Check CALN subrecord
			for _, repoSub := range sub.SubRecords {
				if repoSub.Tag == GedcomTagCaln {
					foundCaln = true
					assert.Equal(t, "MS-123", repoSub.Value)
				}
			}
		case GedcomTagData:
			foundData = true
			// Check DATA subrecords
			var hasEven, hasAgnc, hasDate bool
			var evenCount int
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case GedcomTagEven:
					hasEven = true
					evenCount++
				case GedcomTagAgnc:
					hasAgnc = true
					assert.Equal(t, "Church of England", dataSub.Value)
				case GedcomTagDate:
					hasDate = true
					assert.Equal(t, "JUN 1850", dataSub.Value)
				}
			}
			assert.True(t, hasEven, "DATA missing EVEN")
			assert.Equal(t, 3, evenCount, "expected 3 EVEN subrecords")
			assert.True(t, hasAgnc, "DATA missing AGNC")
			assert.True(t, hasDate, "DATA missing DATE")
		}
	}

	assert.True(t, foundPubl, "missing PUBL")
	assert.True(t, foundAbbr, "missing ABBR")
	assert.True(t, foundCaln, "missing CALN under REPO")
	assert.True(t, foundData, "missing DATA")
}

func TestExportSource_WithNotes(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	source := &Source{
		Title:       "Family Bible",
		Description: "Entries in the family Bible of John Smith",
		Notes:       "Handwritten entries, some water damage",
		Properties:  make(map[string]any),
	}

	record := exportSource("source-1", source, expCtx)

	var foundText, foundNote bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagText:
			foundText = true
			assert.Equal(t, "Entries in the family Bible of John Smith", sub.Value)
		case GedcomTagNote:
			foundNote = true
			assert.Equal(t, "Handwritten entries, some water damage", sub.Value)
		}
	}

	assert.True(t, foundText, "missing TEXT")
	assert.True(t, foundNote, "missing NOTE")
}

func TestExportSource_MultipleAuthors(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	source := &Source{
		Title:      "Joint Publication",
		Authors:    []string{"Alice Smith", "Bob Jones", "Carol White"},
		Properties: make(map[string]any),
	}

	record := exportSource("source-1", source, expCtx)

	var foundAuth bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagAuth {
			foundAuth = true
			assert.Equal(t, "Alice Smith; Bob Jones; Carol White", sub.Value)
		}
	}

	assert.True(t, foundAuth, "missing AUTH")
}

func TestExportSource_TypeOnlyGEDCOM70(t *testing.T) {
	// GEDCOM 5.5.1 should NOT include TYPE
	expCtx551 := &ExportContext{
		Version: GEDCOM551,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	source := &Source{
		Title:      "Test Source",
		Type:       "census",
		Properties: make(map[string]any),
	}

	record551 := exportSource("source-1", source, expCtx551)
	var foundType551 bool
	for _, sub := range record551.SubRecords {
		if sub.Tag == GedcomTagType {
			foundType551 = true
		}
	}
	assert.False(t, foundType551, "GEDCOM 5.5.1 should NOT include TYPE")

	// GEDCOM 7.0 SHOULD include TYPE
	expCtx70 := &ExportContext{
		Version: GEDCOM70,
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		RepositoryXRefMap: make(map[string]string),
		MediaXRefMap:      make(map[string]string),
		ExportIndex: &ExportIndex{
			SourceProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	record70 := exportSource("source-1", source, expCtx70)
	var foundType70 bool
	for _, sub := range record70.SubRecords {
		if sub.Tag == GedcomTagType {
			foundType70 = true
			assert.Equal(t, "census", sub.Value)
		}
	}
	assert.True(t, foundType70, "GEDCOM 7.0 should include TYPE")
}

// ============================================================================
// exportMedia tests
// ============================================================================

func TestExportMedia_Basic551(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:        "media/files/photo.jpg",
		MimeType:   MimeTypeJPEG,
		Properties: make(map[string]any),
	}

	record := exportMedia("media-1", media, expCtx)

	assert.Equal(t, "@O1@", record.XRef)
	assert.Equal(t, GedcomTagObje, record.Tag)

	// Should have FILE subrecord
	require.NotEmpty(t, record.SubRecords)
	fileRecord := record.SubRecords[0]
	assert.Equal(t, GedcomTagFile, fileRecord.Tag)
	assert.Equal(t, "media/files/photo.jpg", fileRecord.Value)

	// GEDCOM 5.5.1: FORM under FILE
	require.NotEmpty(t, fileRecord.SubRecords)
	formRecord := fileRecord.SubRecords[0]
	assert.Equal(t, GedcomTagForm, formRecord.Tag)
	assert.Equal(t, "jpg", formRecord.Value)
}

func TestExportMedia_Basic70(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM70,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:        "https://example.com/photo.png",
		MimeType:   MimeTypePNG,
		Properties: make(map[string]any),
	}

	record := exportMedia("media-1", media, expCtx)

	assert.Equal(t, "@O1@", record.XRef)
	assert.Equal(t, GedcomTagObje, record.Tag)

	// Should have FILE subrecord
	require.NotEmpty(t, record.SubRecords)
	fileRecord := record.SubRecords[0]
	assert.Equal(t, GedcomTagFile, fileRecord.Tag)
	assert.Equal(t, "https://example.com/photo.png", fileRecord.Value)

	// GEDCOM 7.0: MIME under FILE
	require.NotEmpty(t, fileRecord.SubRecords)
	mimeRecord := fileRecord.SubRecords[0]
	assert.Equal(t, GedcomTagMime, mimeRecord.Tag)
	assert.Equal(t, MimeTypePNG, mimeRecord.Value)
}

func TestExportMedia_WithTitle(t *testing.T) {
	// Test GEDCOM 5.5.1: TITL at OBJE level
	expCtx551 := &ExportContext{
		Version: GEDCOM551,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:        "media/files/wedding.jpg",
		MimeType:   MimeTypeJPEG,
		Title:      "Wedding Photo",
		Properties: make(map[string]any),
	}

	record551 := exportMedia("media-1", media, expCtx551)
	var foundTitlAtObje bool
	for _, sub := range record551.SubRecords {
		if sub.Tag == GedcomTagTitl {
			foundTitlAtObje = true
			assert.Equal(t, "Wedding Photo", sub.Value)
		}
	}
	assert.True(t, foundTitlAtObje, "5.5.1 should have TITL at OBJE level")

	// Test GEDCOM 7.0: TITL under FILE
	expCtx70 := &ExportContext{
		Version: GEDCOM70,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	record70 := exportMedia("media-1", media, expCtx70)
	// TITL should be under FILE, not at OBJE level
	var foundTitlAtObjeLvl70 bool
	for _, sub := range record70.SubRecords {
		if sub.Tag == GedcomTagTitl {
			foundTitlAtObjeLvl70 = true
		}
	}
	assert.False(t, foundTitlAtObjeLvl70, "7.0 should NOT have TITL at OBJE level")

	// Check TITL is under FILE
	fileRecord := record70.SubRecords[0]
	assert.Equal(t, GedcomTagFile, fileRecord.Tag)
	var foundTitlUnderFile bool
	for _, fileSub := range fileRecord.SubRecords {
		if fileSub.Tag == GedcomTagTitl {
			foundTitlUnderFile = true
			assert.Equal(t, "Wedding Photo", fileSub.Value)
		}
	}
	assert.True(t, foundTitlUnderFile, "7.0 should have TITL under FILE")
}

func TestExportMedia_WithMedium551(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:      "media/files/doc.pdf",
		MimeType: MimeTypePDF,
		Properties: map[string]any{
			"medium": "document",
		},
	}

	record := exportMedia("media-1", media, expCtx)

	// Should have FORM with MEDI subrecord at OBJE level
	var foundFormWithMedi bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagForm && len(sub.SubRecords) > 0 {
			for _, formSub := range sub.SubRecords {
				if formSub.Tag == GedcomTagMedi {
					foundFormWithMedi = true
					assert.Equal(t, "document", formSub.Value)
				}
			}
		}
	}
	assert.True(t, foundFormWithMedi, "5.5.1 should have FORM/MEDI for medium property")
}

func TestExportMedia_WithCrop70(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM70,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:      "https://example.com/photo.jpg",
		MimeType: MimeTypeJPEG,
		Properties: map[string]any{
			"crop": map[string]any{
				"top":    10,
				"left":   20,
				"height": 100,
				"width":  200,
			},
		},
	}

	record := exportMedia("media-1", media, expCtx)

	var foundCrop bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagCrop {
			foundCrop = true
			assert.Len(t, sub.SubRecords, 4)

			cropValues := make(map[string]string)
			for _, cropSub := range sub.SubRecords {
				cropValues[cropSub.Tag] = cropSub.Value
			}
			assert.Equal(t, "10", cropValues[GedcomTagTop])
			assert.Equal(t, "20", cropValues[GedcomTagLeft])
			assert.Equal(t, "100", cropValues[GedcomTagHeight])
			assert.Equal(t, "200", cropValues[GedcomTagWidth])
		}
	}
	assert.True(t, foundCrop, "7.0 should have CROP subrecord")
}

func TestExportMedia_WithNotes(t *testing.T) {
	expCtx := &ExportContext{
		Version: GEDCOM551,
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		ExportIndex: &ExportIndex{
			MediaProperties: make(map[string]string),
		},
		Stats: ExportStatistics{},
	}

	media := &Media{
		URI:        "media/files/photo.jpg",
		MimeType:   MimeTypeJPEG,
		Notes:      "Taken at the family reunion",
		Properties: make(map[string]any),
	}

	record := exportMedia("media-1", media, expCtx)

	var foundNote bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagNote {
			foundNote = true
			assert.Equal(t, "Taken at the family reunion", sub.Value)
		}
	}
	assert.True(t, foundNote, "missing NOTE")
}

// ============================================================================
// ExportGEDCOM end-to-end tests (sources and media)
// ============================================================================

func TestExportGEDCOM_WithSourcesAndMedia(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources: map[string]*Source{
			"source-1": {
				Title:        "Census of 1850",
				Authors:      []string{"U.S. Government"},
				RepositoryID: "repo-1",
				Description:  "Population schedules",
				Media:        []string{"media-1"},
				Properties: map[string]any{
					"publication_info": "Washington, D.C.",
				},
			},
		},
		Repositories: map[string]*Repository{
			"repo-1": {
				Name:       "National Archives",
				Properties: make(map[string]any),
			},
		},
		Media: map[string]*Media{
			"media-1": {
				URI:        "media/files/census.jpg",
				MimeType:   MimeTypeJPEG,
				Title:      "Census Page",
				Properties: make(map[string]any),
			},
		},
		Citations:  make(map[string]*Citation),
		Assertions: make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	// Should contain HEAD and TRLR
	assert.Contains(t, output, "0 HEAD\n")
	assert.Contains(t, output, "0 TRLR\n")

	// Repository
	assert.Contains(t, output, "@R1@ REPO")
	assert.Contains(t, output, "NAME National Archives")

	// Source
	assert.Contains(t, output, "@S1@ SOUR")
	assert.Contains(t, output, "TITL Census of 1850")
	assert.Contains(t, output, "AUTH U.S. Government")
	assert.Contains(t, output, "PUBL Washington, D.C.")
	assert.Contains(t, output, "TEXT Population schedules")
	assert.Contains(t, output, "REPO @R1@")
	assert.Contains(t, output, "OBJE @O1@")

	// Media
	assert.Contains(t, output, "@O1@ OBJE")
	assert.Contains(t, output, "FILE media/files/census.jpg")
	assert.Contains(t, output, "TITL Census Page")

	// Statistics
	assert.Equal(t, 1, result.Statistics.RepositoriesExported)
	assert.Equal(t, 1, result.Statistics.SourcesExported)
	assert.Equal(t, 1, result.Statistics.MediaExported)

	// Verify order: HEAD, REPO, SOUR, OBJE, TRLR
	headIdx := strings.Index(output, "0 HEAD")
	repoIdx := strings.Index(output, "@R1@ REPO")
	sourIdx := strings.Index(output, "@S1@ SOUR")
	objeIdx := strings.Index(output, "@O1@ OBJE")
	trlrIdx := strings.Index(output, "0 TRLR")

	assert.True(t, headIdx < repoIdx, "HEAD should come before REPO")
	assert.True(t, repoIdx < sourIdx, "REPO should come before SOUR")
	assert.True(t, sourIdx < objeIdx, "SOUR should come before OBJE")
	assert.True(t, objeIdx < trlrIdx, "OBJE should come before TRLR")
}

func TestExportGEDCOM_WithSourcesAndMedia_GEDCOM70(t *testing.T) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources: map[string]*Source{
			"source-1": {
				Title:      "Test Source",
				Type:       "census",
				Properties: make(map[string]any),
			},
		},
		Repositories: make(map[string]*Repository),
		Media: map[string]*Media{
			"media-1": {
				URI:      "https://example.com/photo.png",
				MimeType: MimeTypePNG,
				Title:    "Test Photo",
				Properties: map[string]any{
					"crop": map[string]any{
						"top":    5,
						"left":   10,
						"height": 50,
						"width":  80,
					},
				},
			},
		},
		Citations:  make(map[string]*Citation),
		Assertions: make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM70, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	// GEDCOM 7.0 specifics
	assert.Contains(t, output, "VERS 7.0")

	// Source with TYPE
	assert.Contains(t, output, "@S1@ SOUR")
	assert.Contains(t, output, "TYPE census")

	// Media with MIME (not FORM)
	assert.Contains(t, output, "@O1@ OBJE")
	assert.Contains(t, output, "MIME "+MimeTypePNG)

	// CROP subrecord
	assert.Contains(t, output, "CROP")
	assert.Contains(t, output, "TOP 5")
	assert.Contains(t, output, "LEFT 10")
	assert.Contains(t, output, "HEIGHT 50")
	assert.Contains(t, output, "WIDTH 80")

	// Title should be under FILE for 7.0
	// (Can't easily verify nesting in string form, but at least it should be present)
	assert.Contains(t, output, "TITL Test Photo")

	assert.Equal(t, 1, result.Statistics.SourcesExported)
	assert.Equal(t, 1, result.Statistics.MediaExported)
	assert.Equal(t, "7.0", result.Version)
}

// ============================================================================
// mimeToGEDCOMFormat tests
// ============================================================================

func TestMimeToGEDCOMFormat(t *testing.T) {
	// JPEG should map to "jpg" (shorter than "jpeg")
	assert.Equal(t, "jpg", mimeToGEDCOMFormat[MimeTypeJPEG])

	// TIFF should map to "tif" (shorter than "tiff")
	assert.Equal(t, "tif", mimeToGEDCOMFormat[MimeTypeTIFF])

	// PNG maps directly
	assert.Equal(t, "png", mimeToGEDCOMFormat[MimeTypePNG])

	// PDF maps directly
	assert.Equal(t, "pdf", mimeToGEDCOMFormat[MimeTypePDF])
}

// ============================================================================
// getStringProperty tests
// ============================================================================

func TestGetStringProperty(t *testing.T) {
	props := map[string]any{
		"key1": "value1",
		"key2": 42,
		"key3": "",
	}

	val, ok := getStringProperty(props, "key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// Non-string value
	_, ok = getStringProperty(props, "key2")
	assert.False(t, ok)

	// Empty string
	_, ok = getStringProperty(props, "key3")
	assert.False(t, ok)

	// Missing key
	_, ok = getStringProperty(props, "missing")
	assert.False(t, ok)

	// Nil map
	_, ok = getStringProperty(nil, "key1")
	assert.False(t, ok)
}

// ============================================================================
// exportPerson tests
// ============================================================================

func TestExportPerson_Basic(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap:  make(map[string]string),
		SourceXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
					"type":    "birth",
				},
			},
			"gender": "male",
		},
	}

	record := exportPerson("person-1", person, expCtx)

	assert.Equal(t, "@I1@", record.XRef)
	assert.Equal(t, GedcomTagIndi, record.Tag)

	var foundName, foundSex bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagName:
			foundName = true
			assert.Equal(t, "John /Smith/", sub.Value)
			// Check substructure
			var hasType, hasGivn, hasSurn bool
			for _, nameSub := range sub.SubRecords {
				switch nameSub.Tag {
				case GedcomTagType:
					hasType = true
					assert.Equal(t, "birth", nameSub.Value)
				case GedcomTagGivn:
					hasGivn = true
					assert.Equal(t, "John", nameSub.Value)
				case GedcomTagSurn:
					hasSurn = true
					assert.Equal(t, "Smith", nameSub.Value)
				}
			}
			assert.True(t, hasType, "missing TYPE")
			assert.True(t, hasGivn, "missing GIVN")
			assert.True(t, hasSurn, "missing SURN")
		case GedcomTagSex:
			foundSex = true
			assert.Equal(t, "M", sub.Value)
		}
	}

	assert.True(t, foundName, "missing NAME")
	assert.True(t, foundSex, "missing SEX")
}

func TestExportPerson_WithEvents(t *testing.T) {
	glxFile := &GLXFile{
		Events: map[string]*Event{
			"event-1": {
				Type: "birth",
				Date: "1850-03-15",
				Participants: []Participant{
					{Person: "person-1", Role: ParticipantRolePrincipal},
				},
			},
			"event-2": {
				Type:    "death",
				Date:    "1920-11-02",
				PlaceID: "place-1",
				Participants: []Participant{
					{Person: "person-1", Role: ParticipantRolePrincipal},
				},
				Properties: map[string]any{
					"cause": "heart failure",
				},
			},
		},
		Places: map[string]*Place{
			"place-1": {Name: "Springfield, Illinois"},
		},
	}

	expCtx := &ExportContext{
		GLX: glxFile,
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap:  make(map[string]string),
		SourceXRefMap: make(map[string]string),
		PlaceStrings: map[string]string{
			"place-1": "Springfield, Illinois",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				"birth": "BIRT",
				"death": "DEAT",
			},
			PersonProperties: make(map[string]string),
			EventProperties: map[string]string{
				"cause": "CAUS",
			},
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	// Build the person events index
	buildPersonEventsIndex(expCtx)

	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
			"gender": "male",
		},
	}

	record := exportPerson("person-1", person, expCtx)

	var foundBirt, foundDeat bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagBirt:
			foundBirt = true
			var hasDate bool
			for _, birtSub := range sub.SubRecords {
				if birtSub.Tag == GedcomTagDate {
					hasDate = true
					assert.Equal(t, "15 MAR 1850", birtSub.Value)
				}
			}
			assert.True(t, hasDate, "BIRT missing DATE")
		case GedcomTagDeat:
			foundDeat = true
			var hasDate, hasPlac, hasCaus bool
			for _, deatSub := range sub.SubRecords {
				switch deatSub.Tag {
				case GedcomTagDate:
					hasDate = true
					assert.Equal(t, "2 NOV 1920", deatSub.Value)
				case GedcomTagPlac:
					hasPlac = true
					assert.Equal(t, "Springfield, Illinois", deatSub.Value)
				case GedcomTagCaus:
					hasCaus = true
					assert.Equal(t, "heart failure", deatSub.Value)
				}
			}
			assert.True(t, hasDate, "DEAT missing DATE")
			assert.True(t, hasPlac, "DEAT missing PLAC")
			assert.True(t, hasCaus, "DEAT missing CAUS")
		}
	}

	assert.True(t, foundBirt, "missing BIRT")
	assert.True(t, foundDeat, "missing DEAT")
	assert.Equal(t, 2, expCtx.Stats.EventsProcessed)
}

func TestExportPerson_MultipleNames(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap:  make(map[string]string),
		SourceXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	person := &Person{
		Properties: map[string]any{
			"name": []any{
				map[string]any{
					"value": "Mary Johnson",
					"fields": map[string]any{
						"given":   "Mary",
						"surname": "Johnson",
						"type":    "birth",
					},
				},
				map[string]any{
					"value": "Mary Smith",
					"fields": map[string]any{
						"given":   "Mary",
						"surname": "Smith",
						"type":    "married",
					},
				},
			},
			"gender": "female",
		},
	}

	record := exportPerson("person-1", person, expCtx)

	var nameCount int
	var nameTypes []string
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagName {
			nameCount++
			for _, nameSub := range sub.SubRecords {
				if nameSub.Tag == GedcomTagType {
					nameTypes = append(nameTypes, nameSub.Value)
				}
			}
		}
	}

	assert.Equal(t, 2, nameCount, "should have 2 NAME records")
	assert.Contains(t, nameTypes, "birth")
	assert.Contains(t, nameTypes, "married")
}

func TestExportPerson_WithProperties(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap:  make(map[string]string),
		SourceXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			EventTypes: make(map[string]string),
			PersonProperties: map[string]string{
				"occupation":  "OCCU",
				"religion":    "RELI",
				"education":   "EDUC",
				"nationality": "NATI",
				"title":       "TITL",
			},
			EventProperties: make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
			"gender":      "male",
			"occupation":  "Farmer",
			"religion":    "Baptist",
			"education":   "Harvard",
			"nationality": "American",
			"title":       "Dr.",
			"born_on":     "1850-03-15", // should be skipped
			"died_on":     "1920-11-02", // should be skipped
		},
	}

	record := exportPerson("person-1", person, expCtx)

	var foundOccu, foundReli, foundEduc, foundNati, foundTitl bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagOccu:
			foundOccu = true
			assert.Equal(t, "Farmer", sub.Value)
		case GedcomTagReli:
			foundReli = true
			assert.Equal(t, "Baptist", sub.Value)
		case GedcomTagEduc:
			foundEduc = true
			assert.Equal(t, "Harvard", sub.Value)
		case GedcomTagNati:
			foundNati = true
			assert.Equal(t, "American", sub.Value)
		case GedcomTagTitl:
			foundTitl = true
			assert.Equal(t, "Dr.", sub.Value)
		}
	}

	assert.True(t, foundOccu, "missing OCCU")
	assert.True(t, foundReli, "missing RELI")
	assert.True(t, foundEduc, "missing EDUC")
	assert.True(t, foundNati, "missing NATI")
	assert.True(t, foundTitl, "missing TITL")

	// Verify born_on and died_on are NOT exported as tags
	for _, sub := range record.SubRecords {
		assert.NotEqual(t, "born_on", sub.Tag)
		assert.NotEqual(t, "died_on", sub.Tag)
	}
}

func TestExportPerson_NameFormat(t *testing.T) {
	tests := []struct {
		name     string
		nameVal  map[string]any
		expected string
	}{
		{
			name: "basic given and surname",
			nameVal: map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
			expected: "John /Smith/",
		},
		{
			name: "with surname prefix",
			nameVal: map[string]any{
				"value": "Ludwig van Beethoven",
				"fields": map[string]any{
					"given":          "Ludwig",
					"surname_prefix": "van",
					"surname":        "Beethoven",
				},
			},
			expected: "Ludwig /van Beethoven/",
		},
		{
			name: "with suffix",
			nameVal: map[string]any{
				"value": "John Smith Jr.",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
					"suffix":  "Jr.",
				},
			},
			expected: "John /Smith/ Jr.",
		},
		{
			name: "with prefix",
			nameVal: map[string]any{
				"value": "Dr. John Smith",
				"fields": map[string]any{
					"prefix":  "Dr.",
					"given":   "John",
					"surname": "Smith",
				},
			},
			// Note: NPFX is a substructure tag; the formatted NAME value
			// only uses given/surname/suffix
			expected: "John /Smith/",
		},
		{
			name: "no fields - parse from value",
			nameVal: map[string]any{
				"value": "John Smith",
			},
			expected: "John /Smith/",
		},
		{
			name: "single name no fields",
			nameVal: map[string]any{
				"value": "Madonna",
			},
			expected: "Madonna //",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := exportNameRecord(tt.nameVal, nil, &ExportContext{
				GLX:           &GLXFile{Citations: make(map[string]*Citation)},
				SourceXRefMap: make(map[string]string),
			})
			require.NotNil(t, record)
			assert.Equal(t, tt.expected, record.Value)
		})
	}
}

func TestMapGenderToSex(t *testing.T) {
	assert.Equal(t, "M", mapGenderToSex("male"))
	assert.Equal(t, "F", mapGenderToSex("female"))
	assert.Equal(t, "X", mapGenderToSex("other"))
	assert.Equal(t, "U", mapGenderToSex("unknown"))
	assert.Equal(t, "U", mapGenderToSex(""))
}

func TestBuildPersonEventsIndex(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: map[string]*Event{
				"event-1": {
					Type: "birth",
					Participants: []Participant{
						{Person: "person-1", Role: ParticipantRolePrincipal},
					},
				},
				"event-2": {
					Type: "death",
					Participants: []Participant{
						{Person: "person-1", Role: ParticipantRolePrincipal},
					},
				},
				"event-3": {
					Type: "marriage",
					Participants: []Participant{
						{Person: "person-1", Role: ParticipantRoleSpouse},
						{Person: "person-2", Role: ParticipantRoleSpouse},
					},
				},
				"event-4": {
					Type: "birth",
					Participants: []Participant{
						{Person: "person-2", Role: ParticipantRolePrincipal},
					},
				},
			},
		},
	}

	buildPersonEventsIndex(expCtx)

	// person-1 should have 2 events where they are principal
	assert.Len(t, expCtx.PersonEvents["person-1"], 2)
	assert.Contains(t, expCtx.PersonEvents["person-1"], "event-1")
	assert.Contains(t, expCtx.PersonEvents["person-1"], "event-2")

	// person-2 should have 1 event where they are principal
	assert.Len(t, expCtx.PersonEvents["person-2"], 1)
	assert.Contains(t, expCtx.PersonEvents["person-2"], "event-4")

	// Marriage event (spouse role) should NOT be in the index
	for _, events := range expCtx.PersonEvents {
		assert.NotContains(t, events, "event-3")
	}
}

func TestExportPerson_WithNotes(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap:  make(map[string]string),
		SourceXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
		},
		Notes: "Prominent local farmer",
	}

	record := exportPerson("person-1", person, expCtx)

	var foundNote bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagNote {
			foundNote = true
			assert.Equal(t, "Prominent local farmer", sub.Value)
		}
	}
	assert.True(t, foundNote, "missing NOTE")
}

// ============================================================================
// ExportGEDCOM end-to-end tests (with persons)
// ============================================================================

func TestExportGEDCOM_WithPersons(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Smith",
							"type":    "birth",
						},
					},
					"gender": "male",
				},
			},
			"person-2": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "Jane Doe",
						"fields": map[string]any{
							"given":   "Jane",
							"surname": "Doe",
						},
					},
					"gender": "female",
				},
			},
		},
		Events: map[string]*Event{
			"event-1": {
				Type:    "birth",
				Date:    "1850-03-15",
				PlaceID: "place-1",
				Participants: []Participant{
					{Person: "person-1", Role: ParticipantRolePrincipal},
				},
			},
			"event-2": {
				Type: "death",
				Date: "1920-11-02",
				Participants: []Participant{
					{Person: "person-1", Role: ParticipantRolePrincipal},
				},
			},
		},
		Relationships: make(map[string]*Relationship),
		Places: map[string]*Place{
			"place-1": {Name: "Springfield"},
		},
		Sources:      make(map[string]*Source),
		Repositories: make(map[string]*Repository),
		Media:        make(map[string]*Media),
		Citations:    make(map[string]*Citation),
		Assertions:   make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glxFile, GEDCOM551, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	// Should have HEAD and TRLR
	assert.Contains(t, output, "0 HEAD\n")
	assert.Contains(t, output, "0 TRLR\n")

	// Person 1
	assert.Contains(t, output, "@I1@ INDI")
	assert.Contains(t, output, "NAME John /Smith/")
	assert.Contains(t, output, "SEX M")
	assert.Contains(t, output, "BIRT")
	assert.Contains(t, output, "DATE 15 MAR 1850")
	assert.Contains(t, output, "PLAC Springfield")
	assert.Contains(t, output, "DEAT")
	assert.Contains(t, output, "DATE 2 NOV 1920")

	// Person 2
	assert.Contains(t, output, "@I2@ INDI")
	assert.Contains(t, output, "NAME Jane /Doe/")
	assert.Contains(t, output, "SEX F")

	// Statistics
	assert.Equal(t, 2, result.Statistics.PersonsExported)
	assert.Equal(t, 2, result.Statistics.EventsProcessed)

	// Verify order: HEAD, INDI records, TRLR
	headIdx := strings.Index(output, "0 HEAD")
	indi1Idx := strings.Index(output, "@I1@ INDI")
	indi2Idx := strings.Index(output, "@I2@ INDI")
	trlrIdx := strings.Index(output, "0 TRLR")

	assert.True(t, headIdx < indi1Idx, "HEAD should come before INDI")
	assert.True(t, indi1Idx < indi2Idx, "INDI 1 should come before INDI 2")
	assert.True(t, indi2Idx < trlrIdx, "INDI should come before TRLR")
}

func TestExportPerson_WithMediaAndSources(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events:   make(map[string]*Event),
			Citations: map[string]*Citation{
				"cit-1": {
					SourceID: "source-1",
					Properties: map[string]any{
						"locator": "Page 42",
					},
				},
			},
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		MediaXRefMap: map[string]string{
			"media-1": "@O1@",
		},
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		Stats:        ExportStatistics{},
	}

	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
			"media":     []any{"media-1"},
			"sources":   []any{"source-1"},
			"citations": []any{"cit-1"},
		},
	}

	record := exportPerson("person-1", person, expCtx)

	var foundObje, foundSourDirect, foundSourFromCit bool
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagObje:
			foundObje = true
			assert.Equal(t, "@O1@", sub.Value)
		case GedcomTagSour:
			if sub.Value == "@S1@" && len(sub.SubRecords) > 0 {
				// This is the citation-sourced SOUR with PAGE
				foundSourFromCit = true
				var foundPage bool
				for _, sourSub := range sub.SubRecords {
					if sourSub.Tag == GedcomTagPage {
						foundPage = true
						assert.Equal(t, "Page 42", sourSub.Value)
					}
				}
				assert.True(t, foundPage, "SOUR from citation missing PAGE")
			} else if sub.Value == "@S1@" {
				foundSourDirect = true
			}
		}
	}

	assert.True(t, foundObje, "missing OBJE")
	assert.True(t, foundSourDirect, "missing direct SOUR")
	assert.True(t, foundSourFromCit, "missing SOUR from citation")
}

// ============================================================================
// Event inline citation export tests
// ============================================================================

func TestExportPersonEvent_WithCitations(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Citations: map[string]*Citation{
				"cit-1": {
					SourceID: "source-1",
					Properties: map[string]any{
						"locator": "p. 42",
					},
				},
			},
		},
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				"birth": GedcomTagBirt,
			},
			EventProperties: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
	}

	event := &Event{
		Type: "birth",
		Date: "1850-03-15",
		Properties: map[string]any{
			PropertyCitations: []string{"cit-1"},
		},
	}

	record := exportPersonEvent(event, expCtx)
	require.NotNil(t, record)
	assert.Equal(t, GedcomTagBirt, record.Tag)

	// Should have DATE + SOUR subrecords
	var foundDate, foundSour bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagDate {
			foundDate = true
			assert.Equal(t, "15 MAR 1850", sub.Value)
		}
		if sub.Tag == GedcomTagSour {
			foundSour = true
			assert.Equal(t, "@S1@", sub.Value)
			// Should have PAGE subrecord
			var foundPage bool
			for _, pageSub := range sub.SubRecords {
				if pageSub.Tag == GedcomTagPage {
					foundPage = true
					assert.Equal(t, "p. 42", pageSub.Value)
				}
			}
			assert.True(t, foundPage, "SOUR subrecord missing PAGE")
		}
	}

	assert.True(t, foundDate, "missing DATE subrecord")
	assert.True(t, foundSour, "missing SOUR subrecord from citation")
}

func TestExportPersonEvent_WithDirectSources(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Citations: make(map[string]*Citation),
		},
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
			"source-2": "@S2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				"death": GedcomTagDeat,
			},
			EventProperties: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
	}

	event := &Event{
		Type: "death",
		Date: "1920",
		Properties: map[string]any{
			PropertySources: []string{"source-1", "source-2"},
		},
	}

	record := exportPersonEvent(event, expCtx)
	require.NotNil(t, record)

	sourCount := 0
	sourXRefs := []string{}
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagSour {
			sourCount++
			sourXRefs = append(sourXRefs, sub.Value)
		}
	}

	assert.Equal(t, 2, sourCount, "expected 2 SOUR subrecords")
	assert.Contains(t, sourXRefs, "@S1@")
	assert.Contains(t, sourXRefs, "@S2@")
}

func TestExportFamilyEvent_WithCitations(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: map[string]*Event{
				"event-marriage": {
					Type: "marriage",
					Date: "1875-06-12",
					Properties: map[string]any{
						PropertyCitations: []string{"cit-marr"},
					},
				},
			},
			Citations: map[string]*Citation{
				"cit-marr": {
					SourceID: "source-church",
					Properties: map[string]any{
						"locator": "Entry 234",
					},
				},
			},
		},
		SourceXRefMap: map[string]string{
			"source-church": "@S5@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:      make(map[string]string),
			EventProperties: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
	}

	record := exportFamilyEvent("event-marriage", GedcomTagMarr, expCtx)
	require.NotNil(t, record)
	assert.Equal(t, GedcomTagMarr, record.Tag)

	var foundSour bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagSour {
			foundSour = true
			assert.Equal(t, "@S5@", sub.Value)
			var foundPage bool
			for _, pageSub := range sub.SubRecords {
				if pageSub.Tag == GedcomTagPage {
					foundPage = true
					assert.Equal(t, "Entry 234", pageSub.Value)
				}
			}
			assert.True(t, foundPage, "SOUR subrecord missing PAGE")
		}
	}

	assert.True(t, foundSour, "family event missing SOUR subrecord from citation")
}

// ============================================================================
// Residence (RESI) export tests
// ============================================================================

func TestExportPerson_ResidenceExported(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-1": {
					Properties: map[string]any{
						PersonPropertyName: map[string]any{
							"value": "John Smith",
						},
						PersonPropertyResidence: []any{
							map[string]any{
								"value": "place-1",
								"date":  "1920",
							},
							"place-2",
						},
					},
				},
			},
			Events:        make(map[string]*Event),
			Relationships: make(map[string]*Relationship),
			Citations:     make(map[string]*Citation),
			Assertions:    make(map[string]*Assertion),
		},
		PersonXRefMap: map[string]string{
			"person-1": "@I1@",
		},
		SourceXRefMap: make(map[string]string),
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: map[string]string{},
			EventProperties:  make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: map[string]string{
			"place-1": "New York, USA",
			"place-2": "Boston, Massachusetts, USA",
		},
		PersonEvents:         make(map[string][]string),
		PersonSpouseFamilies: make(map[string][]string),
		PersonChildFamilies:  make(map[string][]childFamilyRef),
	}

	buildPersonPropertyAssertionsIndex(expCtx)
	record := exportPerson("person-1", expCtx.GLX.Persons["person-1"], expCtx)

	// Count RESI subrecords
	resiCount := 0
	var resiWithDate, resiWithoutDate *GEDCOMRecord
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagResi {
			resiCount++
			hasDate := false
			for _, s := range sub.SubRecords {
				if s.Tag == GedcomTagDate {
					hasDate = true
				}
			}
			if hasDate {
				resiWithDate = sub
			} else {
				resiWithoutDate = sub
			}
		}
	}

	assert.Equal(t, 2, resiCount, "expected 2 RESI records")

	// First RESI: with date and place
	require.NotNil(t, resiWithDate, "RESI with date not found")
	var foundDate, foundPlac bool
	for _, sub := range resiWithDate.SubRecords {
		if sub.Tag == GedcomTagDate {
			foundDate = true
			assert.Equal(t, "1920", sub.Value)
		}
		if sub.Tag == GedcomTagPlac {
			foundPlac = true
			assert.Equal(t, "New York, USA", sub.Value)
		}
	}
	assert.True(t, foundDate, "RESI missing DATE")
	assert.True(t, foundPlac, "RESI missing PLAC")

	// Second RESI: place only
	require.NotNil(t, resiWithoutDate, "RESI without date not found")
	foundPlac = false
	for _, sub := range resiWithoutDate.SubRecords {
		if sub.Tag == GedcomTagPlac {
			foundPlac = true
			assert.Equal(t, "Boston, Massachusetts, USA", sub.Value)
		}
	}
	assert.True(t, foundPlac, "RESI without date missing PLAC")
}

// ============================================================================
// Assertion-based citation export tests (NAME and property citations)
// ============================================================================

func TestExportPerson_NameCitationsFromAssertions(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-1": {
					Properties: map[string]any{
						PersonPropertyName: map[string]any{
							"value": "Jane Doe",
							"fields": map[string]any{
								NameFieldGiven:   "Jane",
								NameFieldSurname: "Doe",
							},
						},
					},
				},
			},
			Events:        make(map[string]*Event),
			Relationships: make(map[string]*Relationship),
			Citations: map[string]*Citation{
				"cit-name": {
					SourceID: "source-1",
					Properties: map[string]any{
						"locator": "Entry 5",
					},
				},
			},
			Assertions: map[string]*Assertion{
				"assert-name": {
					Subject:   EntityRef{Person: "person-1"},
					Property:  PersonPropertyName,
					Value:     "Jane Doe",
					Sources:   []string{"source-2"},
					Citations: []string{"cit-name"},
				},
			},
		},
		PersonXRefMap: map[string]string{"person-1": "@I1@"},
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
			"source-2": "@S2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:        make(map[string]string),
			PersonProperties:  make(map[string]string),
			EventProperties:   make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings:         make(map[string]string),
		PersonEvents:         make(map[string][]string),
		PersonSpouseFamilies: make(map[string][]string),
		PersonChildFamilies:  make(map[string][]childFamilyRef),
	}

	buildPersonPropertyAssertionsIndex(expCtx)
	record := exportPerson("person-1", expCtx.GLX.Persons["person-1"], expCtx)

	// Find the NAME subrecord
	var nameRecord *GEDCOMRecord
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagName {
			nameRecord = sub
			break
		}
	}
	require.NotNil(t, nameRecord, "NAME record not found")

	// Should have SOUR subrecords from the assertion
	var foundSourDirect, foundSourCitation bool
	for _, sub := range nameRecord.SubRecords {
		if sub.Tag == GedcomTagSour {
			if sub.Value == "@S2@" {
				foundSourDirect = true
			}
			if sub.Value == "@S1@" {
				foundSourCitation = true
				// Check PAGE
				var foundPage bool
				for _, pageSub := range sub.SubRecords {
					if pageSub.Tag == GedcomTagPage {
						foundPage = true
						assert.Equal(t, "Entry 5", pageSub.Value)
					}
				}
				assert.True(t, foundPage, "NAME SOUR from citation missing PAGE")
			}
		}
	}

	assert.True(t, foundSourDirect, "NAME missing direct SOUR from assertion")
	assert.True(t, foundSourCitation, "NAME missing SOUR from assertion citation")
}

func TestExportPerson_PropertyCitationsFromAssertions(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-1": {
					Properties: map[string]any{
						PersonPropertyName: map[string]any{
							"value": "John Smith",
						},
						"occupation": "Farmer",
					},
				},
			},
			Events:        make(map[string]*Event),
			Relationships: make(map[string]*Relationship),
			Citations:     make(map[string]*Citation),
			Assertions: map[string]*Assertion{
				"assert-occu": {
					Subject:  EntityRef{Person: "person-1"},
					Property: "occupation",
					Value:    "Farmer",
					Sources:  []string{"source-1"},
				},
			},
		},
		PersonXRefMap: map[string]string{"person-1": "@I1@"},
		SourceXRefMap: map[string]string{
			"source-1": "@S1@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:        make(map[string]string),
			PersonProperties:  map[string]string{"occupation": "OCCU"},
			EventProperties:   make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings:         make(map[string]string),
		PersonEvents:         make(map[string][]string),
		PersonSpouseFamilies: make(map[string][]string),
		PersonChildFamilies:  make(map[string][]childFamilyRef),
	}

	buildPersonPropertyAssertionsIndex(expCtx)
	record := exportPerson("person-1", expCtx.GLX.Persons["person-1"], expCtx)

	// Find the OCCU subrecord
	var occuRecord *GEDCOMRecord
	for _, sub := range record.SubRecords {
		if sub.Tag == "OCCU" {
			occuRecord = sub
			break
		}
	}
	require.NotNil(t, occuRecord, "OCCU record not found")
	assert.Equal(t, "Farmer", occuRecord.Value)

	// Should have SOUR subrecord from the assertion
	var foundSour bool
	for _, sub := range occuRecord.SubRecords {
		if sub.Tag == GedcomTagSour {
			foundSour = true
			assert.Equal(t, "@S1@", sub.Value)
		}
	}

	assert.True(t, foundSour, "OCCU missing SOUR from assertion")
}

func TestExportPerson_NoteFromProperties(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"gender": "male",
					"name": map[string]any{
						"value": "Robert Bruce",
						"fields": map[string]any{
							"given":   "Robert",
							"surname": "Bruce",
						},
					},
					"notes": "King of Scotland, fought at Bannockburn",
				},
			},
		},
		EventTypes:         make(map[string]*EventType),
		PersonProperties:   make(map[string]*PropertyDefinition),
		RelationshipTypes:  make(map[string]*RelationshipType),
		Events:             make(map[string]*Event),
		Relationships:      make(map[string]*Relationship),
		Sources:            make(map[string]*Source),
		Citations:          make(map[string]*Citation),
		Repositories:       make(map[string]*Repository),
		Media:              make(map[string]*Media),
		Assertions:         make(map[string]*Assertion),
	}

	if err := LoadStandardVocabulariesIntoGLX(glxFile); err != nil {
		t.Fatal(err)
	}

	expCtx := &ExportContext{
		GLX:                      glxFile,
		Version:                  GEDCOM551,
		Logger:                   NewImportLogger(nil),
		ExportIndex:              buildExportIndex(glxFile),
		PersonXRefMap:            map[string]string{"person-1": "@I1@"},
		SourceXRefMap:            make(map[string]string),
		RepositoryXRefMap:        make(map[string]string),
		MediaXRefMap:             make(map[string]string),
		PlaceStrings:             make(map[string]string),
		PersonEvents:             make(map[string][]string),
		PersonSpouseFamilies:     make(map[string][]string),
		PersonChildFamilies:      make(map[string][]childFamilyRef),
		PersonPropertyAssertions: make(map[string]map[string][]*Assertion),
	}
	buildPersonPropertyAssertionsIndex(expCtx)

	record := exportPerson("person-1", glxFile.Persons["person-1"], expCtx)

	var noteRecord *GEDCOMRecord
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagNote {
			noteRecord = sub
			break
		}
	}

	require.NotNil(t, noteRecord, "NOTE record should be exported from Properties['notes']")
	assert.Equal(t, "King of Scotland, fought at Bannockburn", noteRecord.Value)
}

func TestExportPersonEvent_NoteFromProperties(t *testing.T) {
	glxFile := &GLXFile{
		EventTypes:       make(map[string]*EventType),
		EventProperties:  make(map[string]*PropertyDefinition),
		PersonProperties: make(map[string]*PropertyDefinition),
		Events:           make(map[string]*Event),
		Sources:          make(map[string]*Source),
		Citations:        make(map[string]*Citation),
	}
	if err := LoadStandardVocabulariesIntoGLX(glxFile); err != nil {
		t.Fatal(err)
	}

	event := &Event{
		Type: "birth",
		Date: "1274",
		Properties: map[string]any{
			"notes": "Born at Turnberry Castle",
		},
	}

	expCtx := &ExportContext{
		GLX:           glxFile,
		ExportIndex:   buildExportIndex(glxFile),
		SourceXRefMap: make(map[string]string),
	}

	record := exportPersonEvent(event, expCtx)
	require.NotNil(t, record)
	assert.Equal(t, GedcomTagBirt, record.Tag)

	var noteRecord *GEDCOMRecord
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagNote {
			noteRecord = sub
			break
		}
	}

	require.NotNil(t, noteRecord, "NOTE should be exported from event Properties['notes']")
	assert.Equal(t, "Born at Turnberry Castle", noteRecord.Value)
}

func TestExportPerson_MultipleOccupations(t *testing.T) {
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"gender": "male",
					"name": map[string]any{
						"value":  "John Smith",
						"fields": map[string]any{"given": "John", "surname": "Smith"},
					},
					"occupation": []any{
						map[string]any{"value": "Farmer"},
						map[string]any{"value": "Blacksmith"},
						map[string]any{"value": "Mayor"},
					},
				},
			},
		},
		EventTypes:         make(map[string]*EventType),
		PersonProperties:   make(map[string]*PropertyDefinition),
		RelationshipTypes:  make(map[string]*RelationshipType),
		Events:             make(map[string]*Event),
		Relationships:      make(map[string]*Relationship),
		Sources:            make(map[string]*Source),
		Citations:          make(map[string]*Citation),
		Repositories:       make(map[string]*Repository),
		Media:              make(map[string]*Media),
		Assertions:         make(map[string]*Assertion),
	}

	if err := LoadStandardVocabulariesIntoGLX(glxFile); err != nil {
		t.Fatal(err)
	}

	expCtx := &ExportContext{
		GLX:                      glxFile,
		Version:                  GEDCOM551,
		Logger:                   NewImportLogger(nil),
		ExportIndex:              buildExportIndex(glxFile),
		PersonXRefMap:            map[string]string{"person-1": "@I1@"},
		SourceXRefMap:            make(map[string]string),
		RepositoryXRefMap:        make(map[string]string),
		MediaXRefMap:             make(map[string]string),
		PlaceStrings:             make(map[string]string),
		PersonEvents:             make(map[string][]string),
		PersonSpouseFamilies:     make(map[string][]string),
		PersonChildFamilies:      make(map[string][]childFamilyRef),
		PersonPropertyAssertions: make(map[string]map[string][]*Assertion),
	}
	buildPersonPropertyAssertionsIndex(expCtx)

	record := exportPerson("person-1", glxFile.Persons["person-1"], expCtx)

	var occuCount int
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagOccu {
			occuCount++
		}
	}
	assert.Equal(t, 3, occuCount, "Should export 3 separate OCCU records for list-valued occupation")
}

// ============================================================================
// HEAD metadata roundtrip tests
// ============================================================================

func TestBuildHEADRecord_ImportMetadataPreserved(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			ImportMetadata: &Metadata{
				Language:   "English",
				SourceFile: "my-family.ged",
				Copyright:  "© 2023 Test Author",
				Notes:      "This is a test archive.",
			},
		},
		Version: GEDCOM551,
	}

	head := buildHEADRecord(expCtx)
	require.NotNil(t, head)

	var foundLang, foundFile, foundCopr, foundNote bool
	for _, sub := range head.SubRecords {
		switch sub.Tag {
		case GedcomTagLang:
			foundLang = true
			assert.Equal(t, "English", sub.Value)
		case GedcomTagFile:
			foundFile = true
			assert.Equal(t, "my-family.ged", sub.Value)
		case GedcomTagCopr:
			foundCopr = true
			assert.Equal(t, "© 2023 Test Author", sub.Value)
		case GedcomTagNote:
			foundNote = true
			assert.Equal(t, "This is a test archive.", sub.Value)
		}
	}
	assert.True(t, foundLang, "HEAD should include LANG from ImportMetadata")
	assert.True(t, foundFile, "HEAD should include FILE from ImportMetadata")
	assert.True(t, foundCopr, "HEAD should include COPR from ImportMetadata")
	assert.True(t, foundNote, "HEAD should include NOTE from ImportMetadata")
}

func TestBuildHEADRecord_NoImportMetadata(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			ImportMetadata: nil,
		},
		Version: GEDCOM551,
	}

	head := buildHEADRecord(expCtx)
	require.NotNil(t, head)

	for _, sub := range head.SubRecords {
		assert.NotEqual(t, GedcomTagLang, sub.Tag, "HEAD should NOT include LANG without ImportMetadata")
		assert.NotEqual(t, GedcomTagFile, sub.Tag, "HEAD should NOT include FILE without ImportMetadata")
		assert.NotEqual(t, GedcomTagCopr, sub.Tag, "HEAD should NOT include COPR without ImportMetadata")
	}
}

func TestBuildHEADRecord_EmptyImportMetadataFields(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			ImportMetadata: &Metadata{
				Language: "English",
				// SourceFile, Copyright, Notes are empty
			},
		},
		Version: GEDCOM551,
	}

	head := buildHEADRecord(expCtx)
	require.NotNil(t, head)

	var foundLang bool
	for _, sub := range head.SubRecords {
		if sub.Tag == GedcomTagLang {
			foundLang = true
		}
		assert.NotEqual(t, GedcomTagFile, sub.Tag, "HEAD should NOT include empty FILE")
		assert.NotEqual(t, GedcomTagCopr, sub.Tag, "HEAD should NOT include empty COPR")
	}
	assert.True(t, foundLang, "HEAD should include non-empty LANG")
}
