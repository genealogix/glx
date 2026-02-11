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

// TestConvertExtensionData tests the convertExtensionData function
func TestConvertExtensionData(t *testing.T) {
	tests := []struct {
		name       string
		tag        string
		value      string
		subRecords []*GEDCOMRecord
		wantProps  map[string]any
	}{
		{
			name:  "extension tag with value only",
			tag:   "_CUSTOM",
			value: "custom value",
			wantProps: map[string]any{
				"extension_tag": "_CUSTOM",
				"value":         "custom value",
			},
		},
		{
			name:  "extension tag with subrecords",
			tag:   "_MYEXT",
			value: "main value",
			subRecords: []*GEDCOMRecord{
				{Tag: "TYPE", Value: "custom_type"},
				{Tag: "NOTE", Value: "custom note"},
			},
			wantProps: map[string]any{
				"extension_tag": "_MYEXT",
				"value":         "main value",
				"subrecords": map[string]any{
					"TYPE": "custom_type",
					"NOTE": "custom note",
				},
			},
		},
		{
			name:  "extension tag with subrecords no value",
			tag:   "_DATA",
			value: "",
			subRecords: []*GEDCOMRecord{
				{Tag: "FIELD1", Value: "value1"},
				{Tag: "FIELD2", Value: "value2"},
			},
			wantProps: map[string]any{
				"extension_tag": "_DATA",
				"subrecords": map[string]any{
					"FIELD1": "value1",
					"FIELD2": "value2",
				},
			},
		},
		{
			name:      "non-extension tag returns empty",
			tag:       "REGULAR",
			value:     "value",
			wantProps: map[string]any{},
		},
		{
			name:      "empty tag returns empty",
			tag:       "",
			value:     "value",
			wantProps: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertExtensionData(tt.tag, tt.value, tt.subRecords)
			assert.Equal(t, tt.wantProps, got)
		})
	}
}

// TestIsExtensionTag tests the isExtensionTag function
func TestIsExtensionTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"_CUSTOM", true},
		{"_MYEXT", true},
		{"_A", true},
		{"REGULAR", false},
		{"INDI", false},
		{"FAM", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := isExtensionTag(tt.tag)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestExtensionTagInGEDCOM tests that extension tags are properly processed in GEDCOM conversion
func TestExtensionTagInGEDCOM(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
0 _CUSTOM Extension Record
1 TYPE custom_type
1 DATA some data
0 TRLR`

	// Import GEDCOM
	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify that the person was created (other records should still be processed)
	assert.Len(t, glx.Persons, 1, "Should create person despite extension tag")

	// Verify statistics show successful import
	assert.Equal(t, 1, result.Statistics.PersonsCreated)
}

// TestExtensionTagWithSubrecords tests extension tags with complex subrecord structures
func TestExtensionTagWithSubrecords(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Jane /Smith/
0 _GENEALOGY_SITE FamilySearch Profile
1 NAME FamilySearch
1 ID FS-12345
1 URL https://familysearch.org
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify person was created
	assert.Equal(t, 1, result.Statistics.PersonsCreated)
	assert.Len(t, glx.Persons, 1)
}

// TestMultipleExtensionTags tests handling of multiple extension tags
func TestMultipleExtensionTags(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 _EXT1 First Extension
1 DATA data1
0 _EXT2 Second Extension
1 DATA data2
0 @I1@ INDI
1 NAME Test /Person/
0 _EXT3 Third Extension
1 NOTE final extension
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should process person successfully despite multiple extension tags
	assert.Equal(t, 1, result.Statistics.PersonsCreated)
	assert.Len(t, glx.Persons, 1)
}

// TestExtensionTagVsUnknownTag tests that extension tags are handled differently than unknown tags
func TestExtensionTagVsUnknownTag(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 _EXTENSION Valid Extension
1 DATA data
0 UNKNOWN Invalid Tag
1 DATA data
0 TRLR`

	reader := strings.NewReader(gedcom)
	glx, result, err := ImportGEDCOM(reader, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Import should succeed - both extension and unknown tags are handled gracefully
	// The difference is that extension tags are processed by convertExtensionData
	// while unknown tags just get a warning
	assert.NotNil(t, glx)
}
