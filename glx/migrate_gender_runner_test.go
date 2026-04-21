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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMigrateGenderToSex_PersonPropertyRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
			"person-2": {Properties: map[string]any{"gender": "female", "occupation": "teacher"}},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 2, report.PropertiesRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["sex"])
	assert.NotContains(t, archive.Persons["person-1"].Properties, "gender")
	assert.Equal(t, "female", archive.Persons["person-2"].Properties["sex"])
	assert.Equal(t, "teacher", archive.Persons["person-2"].Properties["occupation"])
}

func TestMigrateGenderToSex_AssertionRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "gender", Value: "male"},
			"a-2": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "occupation", Value: "farmer"},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 1, report.AssertionsRenamed)
	assert.Equal(t, "sex", archive.Assertions["a-1"].Property)
	assert.Equal(t, "occupation", archive.Assertions["a-2"].Property)
}

func TestMigrateGenderToSex_VocabEntryRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		PersonProperties: map[string]*glxlib.PropertyDefinition{
			"gender": {
				Label:          "Gender",
				VocabularyType: glxlib.VocabGenderTypes,
			},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 1, report.VocabEntriesRenamed)
	require.Contains(t, archive.PersonProperties, "sex")
	assert.NotContains(t, archive.PersonProperties, "gender")
	assert.Equal(t, glxlib.VocabSexTypes, archive.PersonProperties["sex"].VocabularyType)
}

func TestMigrateGenderToSex_PreSplitGenderTypesMovedToSexTypes(t *testing.T) {
	archive := &glxlib.GLXFile{
		GenderTypes: map[string]*glxlib.GenderType{
			"male":    {Label: "Male", GEDCOM: "M"},
			"female":  {Label: "Female", GEDCOM: "F"},
			"unknown": {Label: "Unknown", GEDCOM: "U"},
			"other":   {Label: "Other", GEDCOM: "X"},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 1, report.VocabEntriesRenamed)
	assert.Nil(t, archive.GenderTypes)
	require.Contains(t, archive.SexTypes, "unknown")
	assert.Equal(t, "U", archive.SexTypes["unknown"].GEDCOM)
}

func TestMigrateGenderToSex_PreSplitMergesIntoExistingSexTypes(t *testing.T) {
	// Standard sex_types is already loaded (typical multi-file load via
	// mergeStandardVocabularies). Custom legacy entries in gender_types should
	// merge into sex_types without overwriting the standard ones.
	archive := &glxlib.GLXFile{
		SexTypes: map[string]*glxlib.SexType{
			"male":   {Label: "Male", GEDCOM: "M"},
			"female": {Label: "Female", GEDCOM: "F"},
		},
		GenderTypes: map[string]*glxlib.GenderType{
			"male":         {Label: "OVERWRITTEN"},
			"unknown":      {Label: "Unknown", GEDCOM: "U"},
			"intersex":     {Label: "Intersex"},
			"not_recorded": {Label: "Not Recorded"},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 1, report.VocabEntriesRenamed)
	assert.Nil(t, archive.GenderTypes)
	assert.Equal(t, "Male", archive.SexTypes["male"].Label, "existing sex_types entry should not be overwritten")
	assert.Equal(t, "Unknown", archive.SexTypes["unknown"].Label)
	assert.Equal(t, "Intersex", archive.SexTypes["intersex"].Label)
	assert.Equal(t, "Not Recorded", archive.SexTypes["not_recorded"].Label)
}

func TestMigrateGenderToSex_NilWriterDoesNotPanic(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male", "sex": "female"}},
		},
	}

	report, err := migrateGenderToSex(archive, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, report.PropertiesRenamed)
}

func TestMigrateGenderToSex_PostSplitGenderTypesUntouched(t *testing.T) {
	// Vocabulary contains "nonbinary" -> already the new identity vocabulary.
	archive := &glxlib.GLXFile{
		GenderTypes: map[string]*glxlib.GenderType{
			"male":      {Label: "Male"},
			"female":    {Label: "Female"},
			"nonbinary": {Label: "Non-binary"},
			"other":     {Label: "Other"},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 0, report.VocabEntriesRenamed)
	assert.NotNil(t, archive.GenderTypes)
	assert.Empty(t, archive.SexTypes)
}

func TestMigrateGenderToSex_ConflictWarnsAndLeavesGender(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male", "sex": "female"}},
		},
	}

	warn := &bytes.Buffer{}
	report, err := migrateGenderToSex(archive, warn)
	require.NoError(t, err)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["gender"])
	assert.Equal(t, "female", archive.Persons["person-1"].Properties["sex"])
	assert.Contains(t, warn.String(), "person-1")
}

func TestMigrateGenderToSex_NoOpOnAlreadyMigratedArchive(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"sex": "male"}},
		},
	}

	report, err := migrateGenderToSex(archive, &bytes.Buffer{})
	require.NoError(t, err)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, 0, report.AssertionsRenamed)
	assert.Equal(t, 0, report.VocabEntriesRenamed)
}
