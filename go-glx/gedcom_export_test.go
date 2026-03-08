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
