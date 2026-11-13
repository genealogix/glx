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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func statusPerson(status any) *Person {
	return &Person{Properties: map[string]any{PersonPropertyStatus: status}}
}

func TestIsUnresolvedPlaceholder(t *testing.T) {
	tests := []struct {
		name   string
		person *Person
		want   bool
	}{
		{"nil person", nil, false},
		{"no status", &Person{}, false},
		{"hypothetical", statusPerson(PersonStatusHypothetical), true},
		{"predicted", statusPerson(PersonStatusPredicted), true},
		{"case and space insensitive", statusPerson(" Predicted "), true},
		{"structured value", statusPerson(map[string]any{"value": PersonStatusPredicted}), true},
		{"resolved", statusPerson(PersonStatusResolved), false},
		{"custom status", statusPerson("parked"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsUnresolvedPlaceholder(tt.person))
		})
	}
}

func TestPersonStatusesStandardVocabularyLoaded(t *testing.T) {
	glx := &GLXFile{}
	require.NoError(t, MergeStandardVocabularies(glx))

	assert.Len(t, glx.PersonStatuses, 3)
	assert.Contains(t, glx.PersonStatuses, PersonStatusHypothetical)
	assert.Contains(t, glx.PersonStatuses, PersonStatusPredicted)
	assert.Contains(t, glx.PersonStatuses, PersonStatusResolved)
	assert.Contains(t, glx.PersonProperties, PersonPropertyStatus)
	assert.Equal(t, VocabPersonStatuses, glx.PersonProperties[PersonPropertyStatus].VocabularyType)
	assert.Contains(t, StandardVocabularies(), "person-statuses.glx")
}

func TestPersonStatusValidation(t *testing.T) {
	validate := func(status string) []string {
		archive := &GLXFile{Persons: map[string]*Person{"person-a": statusPerson(status)}}
		require.NoError(t, MergeStandardVocabularies(archive))

		result := archive.Validate()
		warnings := make([]string, 0, len(result.Warnings))
		for _, warning := range result.Warnings {
			warnings = append(warnings, warning.Message)
		}

		return warnings
	}

	assert.Empty(t, validate(PersonStatusPredicted))
	warnings := validate("probable")
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "'probable' not found in person_statuses")
}

func TestMergePersons_PlaceholderStatusDoesNotStick(t *testing.T) {
	identified := func() *Person {
		return &Person{Properties: map[string]any{PersonPropertyName: "William Green"}}
	}
	placeholder := func(status string) *Person {
		return &Person{Properties: map[string]any{PersonPropertyName: "William Green", PersonPropertyStatus: status}}
	}
	tests := []struct {
		name       string
		keep, drop *Person
		want       string
	}{
		{"placeholder dropped into identified", identified(), placeholder(PersonStatusPredicted), ""},
		{"identified dropped into placeholder", placeholder(PersonStatusHypothetical), identified(), ""},
		{"identified with no properties dropped into placeholder", placeholder(PersonStatusPredicted), &Person{}, ""},
		{"resolved dropped into placeholder", placeholder(PersonStatusPredicted), placeholder(PersonStatusResolved), PersonStatusResolved},
		{"two placeholders stay unresolved", placeholder(PersonStatusPredicted), placeholder(PersonStatusPredicted), PersonStatusPredicted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := &GLXFile{Persons: map[string]*Person{"person-keep": tt.keep, "person-drop": tt.drop}}
			_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, PersonStatus(archive.Persons["person-keep"]))
		})
	}
}

func TestGEDCOMExport_NotesUnresolvedPlaceholder(t *testing.T) {
	export := func(status string) string {
		archive := &GLXFile{Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{PersonPropertyName: "Unknown #8", PersonPropertyStatus: status}},
		}}
		data, _, err := ExportGEDCOM(archive, GEDCOM551, nil)
		require.NoError(t, err)

		return string(data)
	}

	assert.Contains(t, export(PersonStatusPredicted), "NOTE GLX status: predicted (research placeholder")
	assert.NotContains(t, export(PersonStatusResolved), "GLX status")
}
