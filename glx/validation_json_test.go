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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestBuildValidationJSON(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{"place-a": {}},
		Events: map[string]*glxlib.Event{"event-a": {Type: "birth", PlaceID: "missing"}},
	}
	result, err := buildValidationJSON(archive, true)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.NotEmpty(t, result.StructuralErrors)
	require.NotEmpty(t, result.Errors)
	require.NotNil(t, result.Report)
	require.Contains(t, result.Scope, "unknown fields discarded")
	require.Nil(t, archive.EventTypes, "validation must not populate the caller's vocabulary maps")
	_, err = buildValidationJSON(nil, false)
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
}

func TestValidateJSONPathRawFields(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons:\n  person-a:\n    properties:\n      name: Alice\n    unknown_field: preserve-this-error\nevent_types: {}\n"), 0o600))
	result, err := validateJSONPath(file, false)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.NotEmpty(t, result.StructuralErrors)
	require.Contains(t, result.Scope, "source_files")
	require.Nil(t, result.Report)
	require.NoError(t, os.WriteFile(file, []byte("persons:\n  person-a:\n    properties:\n      name: Alice\n"), 0o600))
	_, err = validateJSONPath(file, false)
	require.ErrorIs(t, err, errJSONValidationScope)
	require.ErrorIs(t, showValidationJSON([]string{"a", "b"}, false), errJSONValidationScope)
	require.ErrorIs(t, showValidationJSON([]string{"-"}, false), errJSONValidationScope)
}

func TestValidateJSONPathParseError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons: [not-yaml"), 0o600))
	result, err := validateJSONPath(file, false)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.NotEmpty(t, result.StructuralErrors)
}

func TestValidateJSONPathSingleFileReferences(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {Properties: map[string]any{"name": "Alice"}}},
		Events:  map[string]*glxlib.Event{"event-a": {Type: "birth", Date: "1900", Participants: []glxlib.Participant{{Person: "person-a", Role: "principal"}}}},
	}
	require.NoError(t, glxlib.MergeStandardVocabularies(archive))
	data, err := createSerializer(false, false, "").SerializeSingleFileBytes(archive)
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(file, data, 0o600))
	result, err := validateJSONPath(file, true)
	require.NoError(t, err)
	require.True(t, result.Valid, "%+v", result)
	require.Empty(t, result.StructuralErrors)
	require.Empty(t, result.Errors)
	require.Equal(t, 0, result.Report.TotalAssertions)
	require.Equal(t, []string{"person-a"}, result.Report.UnbackedPersons)
	require.Equal(t, []string{"event-a"}, result.Report.UnbackedEvents)

	archive.Events["event-a"].Participants[0].Person = "person-missing"
	data, err = createSerializer(false, false, "").SerializeSingleFileBytes(archive)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, data, 0o600))
	result, err = validateJSONPath(file, false)
	require.NoError(t, err)
	require.False(t, result.Valid)
	require.NotEmpty(t, result.Errors)
}

func TestValidateJSONPathDuplicates(t *testing.T) {
	dir := t.TempDir()
	data := []byte("persons:\n  person-z:\n    properties:\n      name: Zelda\n  person-a:\n    properties:\n      name: Alice\n")
	for _, name := range []string{"z.glx", "a.glx"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o600))
	}
	first, err := validateJSONPath(dir, false)
	require.NoError(t, err)
	require.False(t, first.Valid)
	require.Len(t, first.DuplicateIDs, 2)
	for range 10 {
		result, runErr := validateJSONPath(dir, false)
		require.NoError(t, runErr)
		require.False(t, result.Valid)
		require.Equal(t, first.DuplicateIDs, result.DuplicateIDs)
	}
}

func TestValidationJSONConfidenceReportParity(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {}, "person-unbacked": {}},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-z": {Subject: glxlib.EntityRef{Person: "person-a"}, Confidence: "high", Property: "name", Value: "Alice"},
			"assertion-a": {Subject: glxlib.EntityRef{Person: "person-a"}, Property: "name", Value: "Alice"},
		},
	}
	result, err := buildValidationJSON(archive, true)
	require.NoError(t, err)
	expected := buildConfidenceReport(archive)
	require.Equal(t, expected.TotalAssertions, result.Report.TotalAssertions)
	require.Equal(t, expected.ByConfidence, result.Report.ByConfidence)
	require.Equal(t, expected.NoConfidence, result.Report.NoConfidence)
	require.Equal(t, expected.NoCitations, result.Report.NoCitations)
	require.Equal(t, []string{"person-unbacked"}, result.Report.UnbackedPersons)
	require.Equal(t, "assertion-a", result.Report.NoCitations[0].ID)
	require.Equal(t, "assertion-z", result.Report.NoCitations[1].ID)
	empty, err := buildValidationJSON(&glxlib.GLXFile{}, true)
	require.NoError(t, err)
	require.NotNil(t, empty.Report.NoConfidence)
	require.NotNil(t, empty.Report.NoCitations)
	require.NotNil(t, empty.Report.UnbackedPersons)
	require.NotNil(t, empty.Report.UnbackedEvents)
	require.NotNil(t, empty.Report.UnbackedRelations)
	require.NotNil(t, empty.Report.ConfidenceOrder)
}

func TestValidateJSONPathRejectsNonGLXFiles(t *testing.T) {
	// This is a self-contained, structurally parseable archive with a dangling
	// reference. A filtered-out input must never become a valid empty archive.
	data := []byte("persons:\n  person-a:\n    properties:\n      name: Alice\nevents:\n  event-a:\n    type: birth\n    date: \"1900\"\n    place: place-missing\nevent_types: {}\n")
	for _, name := range []string{"archive.txt", "archive", "archive.yaml", "archive.yml"} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), name)
			require.NoError(t, os.WriteFile(file, data, 0o600))
			result, err := validateJSONPath(file, true)
			require.ErrorIs(t, err, ErrNothingToValidate)
			require.Nil(t, result, "unsupported inputs must not produce a success result")
		})
	}
}
