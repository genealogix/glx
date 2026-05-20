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

// TestRecordToExtensionEntry verifies the recursive shape produced for an
// extension GEDCOMRecord, including duplicate child tags and 2-level nesting
// — both of which the old flat map[string]string subrecord shape collapsed.
func TestRecordToExtensionEntry(t *testing.T) {
	tests := []struct {
		name   string
		record *GEDCOMRecord
		want   map[string]any
	}{
		{
			name:   "tag with value only",
			record: &GEDCOMRecord{Tag: "_CUSTOM", Value: "custom value"},
			want: map[string]any{
				ExtensionEntryTag:   "_CUSTOM",
				ExtensionEntryValue: "custom value",
			},
		},
		{
			name: "tag with subrecords",
			record: &GEDCOMRecord{
				Tag:   "_MYEXT",
				Value: "main value",
				SubRecords: []*GEDCOMRecord{
					{Tag: "TYPE", Value: "custom_type"},
					{Tag: "NOTE", Value: "custom note"},
				},
			},
			want: map[string]any{
				ExtensionEntryTag:   "_MYEXT",
				ExtensionEntryValue: "main value",
				ExtensionEntrySubrecords: []any{
					map[string]any{ExtensionEntryTag: "TYPE", ExtensionEntryValue: "custom_type"},
					map[string]any{ExtensionEntryTag: "NOTE", ExtensionEntryValue: "custom note"},
				},
			},
		},
		{
			name: "duplicate child tags preserved",
			record: &GEDCOMRecord{
				Tag: "_DATA",
				SubRecords: []*GEDCOMRecord{
					{Tag: "FIELD", Value: "first"},
					{Tag: "FIELD", Value: "second"},
				},
			},
			want: map[string]any{
				ExtensionEntryTag: "_DATA",
				ExtensionEntrySubrecords: []any{
					map[string]any{ExtensionEntryTag: "FIELD", ExtensionEntryValue: "first"},
					map[string]any{ExtensionEntryTag: "FIELD", ExtensionEntryValue: "second"},
				},
			},
		},
		{
			name: "two-level nesting preserved",
			record: &GEDCOMRecord{
				Tag:   "_PROFILE",
				Value: "FamilySearch",
				SubRecords: []*GEDCOMRecord{
					{
						Tag:   "ID",
						Value: "FS-12345",
						SubRecords: []*GEDCOMRecord{
							{Tag: "URL", Value: "https://familysearch.org/p/FS-12345"},
						},
					},
				},
			},
			want: map[string]any{
				ExtensionEntryTag:   "_PROFILE",
				ExtensionEntryValue: "FamilySearch",
				ExtensionEntrySubrecords: []any{
					map[string]any{
						ExtensionEntryTag:   "ID",
						ExtensionEntryValue: "FS-12345",
						ExtensionEntrySubrecords: []any{
							map[string]any{ExtensionEntryTag: "URL", ExtensionEntryValue: "https://familysearch.org/p/FS-12345"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordToExtensionEntry(tt.record)
			assert.Equal(t, tt.want, got)
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

// TestExtensionTagOnPerson asserts that an extension subrecord inside an
// INDI record is preserved on the Person's gedcom_extensions list.
func TestExtensionTagOnPerson(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME John /Doe/
1 _CUSTOM Extension Value
2 TYPE custom_type
2 DATA some data
0 TRLR`

	glx, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx.Persons, 1)

	person := firstPerson(glx)
	require.NotNil(t, person)

	exts, ok := person.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok, "expected gedcom_extensions list on Person")
	require.Len(t, exts, 1)

	entry, ok := exts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "_CUSTOM", entry[ExtensionEntryTag])
	assert.Equal(t, "Extension Value", entry[ExtensionEntryValue])

	subs, ok := entry[ExtensionEntrySubrecords].([]any)
	require.True(t, ok)
	require.Len(t, subs, 2)
	assert.Equal(t, map[string]any{ExtensionEntryTag: "TYPE", ExtensionEntryValue: "custom_type"}, subs[0])
	assert.Equal(t, map[string]any{ExtensionEntryTag: "DATA", ExtensionEntryValue: "some data"}, subs[1])
}

// TestExtensionTagOnPersonNested asserts that subrecord nesting under an
// extension tag is preserved (FamilySearch profile-style structure).
func TestExtensionTagOnPersonNested(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Jane /Smith/
1 _GENEALOGY_SITE FamilySearch Profile
2 NAME FamilySearch
2 ID FS-12345
2 URL https://familysearch.org
0 TRLR`

	glx, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx.Persons, 1)

	person := firstPerson(glx)
	require.NotNil(t, person)

	exts, ok := person.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok)
	require.Len(t, exts, 1)

	entry := exts[0].(map[string]any)
	assert.Equal(t, "_GENEALOGY_SITE", entry[ExtensionEntryTag])
	assert.Equal(t, "FamilySearch Profile", entry[ExtensionEntryValue])

	subs := entry[ExtensionEntrySubrecords].([]any)
	require.Len(t, subs, 3)
	assert.Equal(t, map[string]any{ExtensionEntryTag: "NAME", ExtensionEntryValue: "FamilySearch"}, subs[0])
	assert.Equal(t, map[string]any{ExtensionEntryTag: "ID", ExtensionEntryValue: "FS-12345"}, subs[1])
	assert.Equal(t, map[string]any{ExtensionEntryTag: "URL", ExtensionEntryValue: "https://familysearch.org"}, subs[2])
}

// TestMultipleExtensionTagsOnPerson asserts that multiple distinct extension
// tags on the same INDI all land in the same gedcom_extensions list, in
// source order.
func TestMultipleExtensionTagsOnPerson(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
0 @I1@ INDI
1 NAME Test /Person/
1 _EXT1 First
1 _EXT2 Second
1 _EXT3 Third
0 TRLR`

	glx, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx.Persons, 1)

	person := firstPerson(glx)
	require.NotNil(t, person)

	exts, ok := person.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok)
	require.Len(t, exts, 3)

	tags := make([]string, 0, len(exts))
	for _, ext := range exts {
		tags = append(tags, ext.(map[string]any)[ExtensionEntryTag].(string))
	}
	assert.Equal(t, []string{"_EXT1", "_EXT2", "_EXT3"}, tags)
}

// TestExtensionTagOnFamily asserts that an extension subrecord inside a FAM
// record is preserved on the marriage Relationship's gedcom_extensions list.
func TestExtensionTagOnFamily(t *testing.T) {
	gedcom := `0 HEAD
1 GEDC
2 VERS 5.5.1
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
1 _FAMTAG Family Extension
2 NOTE attached note
0 TRLR`

	glx, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, glx.Relationships, 1)

	rel := firstRelationship(glx)
	require.NotNil(t, rel)
	require.Equal(t, RelationshipTypeMarriage, rel.Type)

	exts, ok := rel.Properties[PropertyGEDCOMExtensions].([]any)
	require.True(t, ok, "expected gedcom_extensions list on Relationship")
	require.Len(t, exts, 1)

	entry := exts[0].(map[string]any)
	assert.Equal(t, "_FAMTAG", entry[ExtensionEntryTag])
	assert.Equal(t, "Family Extension", entry[ExtensionEntryValue])

	subs := entry[ExtensionEntrySubrecords].([]any)
	require.Len(t, subs, 1)
	assert.Equal(t, map[string]any{ExtensionEntryTag: "NOTE", ExtensionEntryValue: "attached note"}, subs[0])
}
