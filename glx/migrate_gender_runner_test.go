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
	"os"
	"path/filepath"
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

	report := migrateGenderToSex(archive, &bytes.Buffer{})

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
			// Non-person subject — a custom "gender" property on an event
			// must NOT be renamed by this migration.
			"a-3": {Subject: glxlib.EntityRef{Event: "event-1"}, Property: "gender", Value: "mixed"},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.AssertionsRenamed)
	assert.Equal(t, "sex", archive.Assertions["a-1"].Property)
	assert.Equal(t, "occupation", archive.Assertions["a-2"].Property)
	assert.Equal(t, "gender", archive.Assertions["a-3"].Property)
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

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.VocabEntriesRenamed)
	require.Contains(t, archive.PersonProperties, "sex")
	assert.NotContains(t, archive.PersonProperties, "gender")
	assert.Equal(t, glxlib.VocabSexTypes, archive.PersonProperties["sex"].VocabularyType)
}

func TestMigrateGenderToSex_PreSplitGenderTypesMovedToSexTypes(t *testing.T) {
	archive := &glxlib.GLXFile{
		GenderTypes: map[string]*glxlib.VocabularyEntry{
			"male":    {Label: "Male", GEDCOM: "M"},
			"female":  {Label: "Female", GEDCOM: "F"},
			"unknown": {Label: "Unknown", GEDCOM: "U"},
			"other":   {Label: "Other", GEDCOM: "X"},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

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
		SexTypes: map[string]*glxlib.VocabularyEntry{
			"male":   {Label: "Male", GEDCOM: "M"},
			"female": {Label: "Female", GEDCOM: "F"},
		},
		GenderTypes: map[string]*glxlib.VocabularyEntry{
			"male":         {Label: "OVERWRITTEN"},
			"unknown":      {Label: "Unknown", GEDCOM: "U"},
			"intersex":     {Label: "Intersex"},
			"not_recorded": {Label: "Not Recorded"},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.VocabEntriesRenamed)
	assert.Nil(t, archive.GenderTypes)
	assert.Equal(t, "Male", archive.SexTypes["male"].Label, "existing sex_types entry should not be overwritten")
	assert.Equal(t, "Unknown", archive.SexTypes["unknown"].Label)
	assert.Equal(t, "Intersex", archive.SexTypes["intersex"].Label)
	assert.Equal(t, "Not Recorded", archive.SexTypes["not_recorded"].Label)
}

func TestMigrateGenderToSex_DoesNotMutateSharedSexTypesMap(t *testing.T) {
	// mergeStandardVocabularies assigns std.SexTypes to archive.SexTypes by
	// reference when the archive has no inlined sex_types. Two archives loaded
	// in the same process can therefore share that map. The migration on
	// archive A must not leak its custom legacy entries into the map that
	// archive B sees.
	shared := map[string]*glxlib.VocabularyEntry{
		"male":   {Label: "Male", GEDCOM: "M"},
		"female": {Label: "Female", GEDCOM: "F"},
	}
	archiveA := &glxlib.GLXFile{
		SexTypes: shared,
		GenderTypes: map[string]*glxlib.VocabularyEntry{
			"male":     {Label: "OVERWRITTEN"},
			"unknown":  {Label: "Unknown", GEDCOM: "U"},
			"intersex": {Label: "Intersex"},
		},
	}
	archiveB := &glxlib.GLXFile{SexTypes: shared}

	migrateGenderToSex(archiveA, &bytes.Buffer{})

	assert.Contains(t, archiveA.SexTypes, "intersex", "archive A should have the moved entry")
	assert.NotContains(t, shared, "intersex", "shared standard map must not be mutated")
	assert.NotContains(t, archiveB.SexTypes, "intersex", "archive B (sharing the std map) must not see archive A's entries")
	assert.Len(t, shared, 2, "shared map should still contain only its original entries")
}

func TestMigrateGenderToSex_NilWriterDoesNotPanic(t *testing.T) {
	// Pre-split archive exercises the full rename path with a nil writer.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
		},
	}

	report := migrateGenderToSex(archive, nil)
	assert.Equal(t, 1, report.PropertiesRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["sex"])
}

func TestMigrateGenderToSex_PostSplitGenderTypesUntouched(t *testing.T) {
	// Vocabulary contains "nonbinary" -> already the new identity vocabulary.
	archive := &glxlib.GLXFile{
		GenderTypes: map[string]*glxlib.VocabularyEntry{
			"male":      {Label: "Male"},
			"female":    {Label: "Female"},
			"nonbinary": {Label: "Non-binary"},
			"other":     {Label: "Other"},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 0, report.VocabEntriesRenamed)
	assert.NotNil(t, archive.GenderTypes)
	assert.Empty(t, archive.SexTypes)
}

func TestMigrateGenderToSex_SkipsWhenSexAlreadyPresent(t *testing.T) {
	// Any person with `sex` set signals the archive is post-split. Running the
	// migration on such an archive must not touch `gender` (identity) values.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male", "sex": "female"}},
			"person-2": {Properties: map[string]any{"gender": "nonbinary"}},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateGenderToSex(archive, warn)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, 0, report.AssertionsRenamed)
	assert.Equal(t, 0, report.VocabEntriesRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["gender"])
	assert.Equal(t, "female", archive.Persons["person-1"].Properties["sex"])
	assert.Equal(t, "nonbinary", archive.Persons["person-2"].Properties["gender"])
	assert.Contains(t, warn.String(), "post-split")
}

func TestMigrateGenderToSex_SkipsWhenGenderHasNonbinary(t *testing.T) {
	// `nonbinary` only appears in the post-split gender_types vocabulary, so
	// its presence (even without any `sex` values) signals the archive is
	// using `gender` as identity.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
			"person-2": {Properties: map[string]any{"gender": "nonbinary"}},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateGenderToSex(archive, warn)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["gender"])
	assert.NotContains(t, archive.Persons["person-1"].Properties, "sex")
	assert.Contains(t, warn.String(), "post-split")
}

func TestMigrateGenderToSex_SkipsWhenAssertionTargetsSex(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "sex", Value: "male"},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateGenderToSex(archive, warn)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, 0, report.AssertionsRenamed)
	assert.Contains(t, warn.String(), "post-split")
}

// TestMigrateGenderToSex_EmptySexDoesNotGatePostSplit verifies that an
// archive with a placeholder/empty `sex` value (e.g. `sex: ""`, `sex: {}`,
// `sex: []`) does NOT trip the post-split guard. The legacy data still lives
// in `gender` and the migration must proceed rather than silently skip.
func TestMigrateGenderToSex_EmptySexDoesNotGatePostSplit(t *testing.T) {
	cases := []struct {
		name   string
		sexVal any
	}{
		{"empty_string", ""},
		{"empty_map", map[string]any{}},
		{"empty_list", []any{}},
		{"temporal_empty_value", map[string]any{"value": ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := &glxlib.GLXFile{
				Persons: map[string]*glxlib.Person{
					"person-1": {Properties: map[string]any{
						"gender": "male",
						"sex":    tc.sexVal,
					}},
				},
			}

			report := migrateGenderToSex(archive, &bytes.Buffer{})

			assert.Equal(t, 1, report.PropertiesRenamed,
				"migration must run when sex is empty/placeholder")
			assert.Equal(t, "male", archive.Persons["person-1"].Properties["sex"])
			assert.NotContains(t, archive.Persons["person-1"].Properties, "gender")
		})
	}
}

func TestMigrateGenderToSex_NoOpOnAlreadyMigratedArchive(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"sex": "male"}},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, 0, report.AssertionsRenamed)
	assert.Equal(t, 0, report.VocabEntriesRenamed)
}

// TestMigrateGenderToSex_UnknownValueRoutesToSex verifies that `gender: unknown`
// in a pre-split archive becomes `sex: unknown` — `unknown` is a sex_types
// value post-split, not a gender_types value.
func TestMigrateGenderToSex_UnknownValueRoutesToSex(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "unknown"}},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.PropertiesRenamed)
	assert.Equal(t, "unknown", archive.Persons["person-1"].Properties["sex"])
	assert.NotContains(t, archive.Persons["person-1"].Properties, "gender")
}

// TestMigrateGenderToSex_IsIdempotent runs the migration twice and confirms
// the second run is a no-op (post-split gate catches the migrated archive).
func TestMigrateGenderToSex_IsIdempotent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
			"person-2": {Properties: map[string]any{"gender": "female"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "gender", Value: "male"},
		},
	}

	first := migrateGenderToSex(archive, &bytes.Buffer{})
	require.Equal(t, 2, first.PropertiesRenamed)
	require.Equal(t, 1, first.AssertionsRenamed)

	warn := &bytes.Buffer{}
	second := migrateGenderToSex(archive, warn)
	assert.Equal(t, 0, second.PropertiesRenamed)
	assert.Equal(t, 0, second.AssertionsRenamed)
	assert.Equal(t, 0, second.VocabEntriesRenamed)
	assert.Contains(t, warn.String(), "post-split")
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["sex"])
	assert.Equal(t, "female", archive.Persons["person-2"].Properties["sex"])
	assert.Equal(t, "sex", archive.Assertions["a-1"].Property)
}

// TestMigrateGenderToSex_SkipsWhenGenderHasCustomIdentity verifies that an
// archive using post-split semantics with a custom identity value (one that
// isn't the canonical `nonbinary`) is not migrated. Otherwise the rename
// would silently move identity values into `sex` and corrupt the archive.
func TestMigrateGenderToSex_SkipsWhenGenderHasCustomIdentity(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"string_two_spirit", "two-spirit"},
		{"string_fluid", "fluid"},
		{"temporal_map_custom", map[string]any{"value": "two-spirit", "date": "2020"}},
		{"temporal_list_custom", []any{
			map[string]any{"value": "male", "date": "2000"},
			map[string]any{"value": "two-spirit", "date": "2020"},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := &glxlib.GLXFile{
				Persons: map[string]*glxlib.Person{
					"person-1": {Properties: map[string]any{"gender": tc.value}},
				},
			}

			warn := &bytes.Buffer{}
			report := migrateGenderToSex(archive, warn)

			assert.Equal(t, 0, report.PropertiesRenamed,
				"custom identity values must trip the post-split guard")
			assert.Equal(t, tc.value, archive.Persons["person-1"].Properties["gender"])
			assert.NotContains(t, archive.Persons["person-1"].Properties, "sex")
			assert.Contains(t, warn.String(), "post-split")
		})
	}
}

// TestMigrateGenderToSex_SkipsWhenAssertionHasCustomIdentity covers the
// assertion-side equivalent of the above: a person-subject assertion with a
// non-legacy gender value must also be treated as a post-split signal.
func TestMigrateGenderToSex_SkipsWhenAssertionHasCustomIdentity(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"gender": "male"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:  glxlib.EntityRef{Person: "person-1"},
				Property: "gender",
				Value:    "two-spirit",
			},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateGenderToSex(archive, warn)

	assert.Equal(t, 0, report.PropertiesRenamed)
	assert.Equal(t, 0, report.AssertionsRenamed)
	assert.Equal(t, "male", archive.Persons["person-1"].Properties["gender"])
	assert.Contains(t, warn.String(), "post-split")
}

// TestMigrateGenderToSex_PreSplitVocabFullClone verifies that all optional
// VocabularyEntry fields (Category, AppliesTo, MimeType) on user-supplied
// gender_types entries survive the move into sex_types — earlier the move
// only copied Label/Description/GEDCOM and silently dropped the rest.
func TestMigrateGenderToSex_PreSplitVocabFullClone(t *testing.T) {
	original := &glxlib.VocabularyEntry{
		Label:       "Custom",
		Description: "A custom legacy entry",
		Category:    "demographic",
		AppliesTo:   []string{"persons", "events"},
		MimeType:    "text/plain",
		GEDCOM:      "X",
	}
	archive := &glxlib.GLXFile{
		GenderTypes: map[string]*glxlib.VocabularyEntry{
			"male":    {Label: "Male", GEDCOM: "M"},
			"female":  {Label: "Female", GEDCOM: "F"},
			"unknown": {Label: "Unknown", GEDCOM: "U"},
			"custom":  original,
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	require.Equal(t, 1, report.VocabEntriesRenamed)
	require.Contains(t, archive.SexTypes, "custom")
	migrated := archive.SexTypes["custom"]
	assert.Equal(t, "Custom", migrated.Label)
	assert.Equal(t, "A custom legacy entry", migrated.Description)
	assert.Equal(t, "demographic", migrated.Category)
	assert.Equal(t, []string{"persons", "events"}, migrated.AppliesTo)
	assert.Equal(t, "text/plain", migrated.MimeType)
	assert.Equal(t, "X", migrated.GEDCOM)

	// AppliesTo must be a deep copy — mutating the migrated slice must not
	// reach back into the (now-discarded but caller-shared) original.
	migrated.AppliesTo[0] = "MUTATED"
	assert.Equal(t, "persons", original.AppliesTo[0],
		"AppliesTo should be cloned, not aliased")
}

// TestMigrateGenderToSex_TemporalShapeValues exercises both the
// map-with-value and temporal-list value shapes to confirm the rename moves
// the whole value wholesale rather than only handling scalar strings.
func TestMigrateGenderToSex_TemporalShapeValues(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-map": {Properties: map[string]any{
				"gender": map[string]any{"value": "male", "date": "1900"},
			}},
			"person-list": {Properties: map[string]any{
				"gender": []any{
					map[string]any{"value": "female", "date": "1920"},
					map[string]any{"value": "male", "date": "1940"},
				},
			}},
		},
	}

	report := migrateGenderToSex(archive, &bytes.Buffer{})

	assert.Equal(t, 2, report.PropertiesRenamed)

	mapVal, ok := archive.Persons["person-map"].Properties["sex"].(map[string]any)
	require.True(t, ok, "temporal map value should be preserved")
	assert.Equal(t, "male", mapVal["value"])
	assert.Equal(t, "1900", mapVal["date"])
	assert.NotContains(t, archive.Persons["person-map"].Properties, "gender")

	listVal, ok := archive.Persons["person-list"].Properties["sex"].([]any)
	require.True(t, ok, "temporal list value should be preserved")
	require.Len(t, listVal, 2)
	assert.NotContains(t, archive.Persons["person-list"].Properties, "gender")
}

// TestMigrateArchive_SingleFileRoundTrip exercises the full archive
// load → migrate → write → reload pipeline on a single-file archive.
// In-memory unit tests can miss serialization bugs (see CLAUDE.md /
// release_smoke_test_demo_repo feedback).
func TestMigrateArchive_SingleFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")

	preSplit := `metadata:
  glx_version: "1.0"
persons:
  person-a:
    properties:
      name:
        value: "Alice"
      gender: "female"
  person-b:
    properties:
      name:
        value: "Bob"
      gender: "male"
`
	require.NoError(t, os.WriteFile(archivePath, []byte(preSplit), 0o600))

	t.Cleanup(func() { migrateRenameGenderToSex = false })
	migrateRenameGenderToSex = true
	require.NoError(t, migrateArchive(archivePath))

	written, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	writtenStr := string(written)
	assert.Contains(t, writtenStr, "sex: female")
	assert.Contains(t, writtenStr, "sex: male")
	assert.NotContains(t, writtenStr, "gender: female")
	assert.NotContains(t, writtenStr, "gender: male")

	reloaded, err := readSingleFileArchive(archivePath, false)
	require.NoError(t, err)
	require.Len(t, reloaded.Persons, 2)
	assert.Equal(t, "female", reloaded.Persons["person-a"].Properties["sex"])
	assert.Equal(t, "male", reloaded.Persons["person-b"].Properties["sex"])
	assert.NotContains(t, reloaded.Persons["person-a"].Properties, "gender")
	assert.NotContains(t, reloaded.Persons["person-b"].Properties, "gender")

	// Idempotency at the fs boundary: a second invocation must be a no-op
	// and must not corrupt the now-migrated file.
	require.NoError(t, migrateArchive(archivePath))
	second, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Contains(t, string(second), "sex: female")
	assert.Contains(t, string(second), "sex: male")
}

// TestMigrateArchive_MultiFileRoundTrip mirrors the single-file round-trip
// test against a multi-file archive. Multi-file write goes through
// safeWriteMultiFileArchive which does atomic-swap, so a bug in the rename
// logic against that path would corrupt the whole archive — worth covering
// separately from the single-file path.
func TestMigrateArchive_MultiFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.glx"),
		[]byte("metadata:\n  glx_version: \"1.0\"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-a.glx"),
		[]byte(`persons:
  person-a:
    properties:
      name:
        value: "Alice"
      gender: "female"
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-b.glx"),
		[]byte(`persons:
  person-b:
    properties:
      name:
        value: "Bob"
      gender: "male"
`), 0o600))

	t.Cleanup(func() { migrateRenameGenderToSex = false })
	migrateRenameGenderToSex = true
	require.NoError(t, migrateArchive(dir))

	personA, err := os.ReadFile(filepath.Join(dir, "persons", "person-a.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(personA), "sex: female")
	assert.NotContains(t, string(personA), "gender: female")

	personB, err := os.ReadFile(filepath.Join(dir, "persons", "person-b.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(personB), "sex: male")
	assert.NotContains(t, string(personB), "gender: male")

	// Reload via LoadArchive to verify serialized archive still parses.
	reloaded, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.Len(t, reloaded.Persons, 2)
	assert.Equal(t, "female", reloaded.Persons["person-a"].Properties["sex"])
	assert.Equal(t, "male", reloaded.Persons["person-b"].Properties["sex"])

	// Second invocation: post-split gate trips; stdout should announce the
	// skip with zero legacy remaining (benign idempotent re-run).
	require.NoError(t, migrateArchive(dir))
}
