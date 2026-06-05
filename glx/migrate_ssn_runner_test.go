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

func TestMigrateSsnToNationalID_PersonPropertyRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"ssn": "123-45-6789"}},
			"person-2": {Properties: map[string]any{"ssn": "987-65-4321", "occupation": "teacher"}},
		},
	}

	report := migrateSsnToNationalID(archive, &bytes.Buffer{})

	assert.Equal(t, 2, report.SsnPropertiesRenamed)
	assert.Equal(t, "123-45-6789", archive.Persons["person-1"].Properties["national_id"])
	assert.NotContains(t, archive.Persons["person-1"].Properties, "ssn")
	assert.Equal(t, "987-65-4321", archive.Persons["person-2"].Properties["national_id"])
	assert.Equal(t, "teacher", archive.Persons["person-2"].Properties["occupation"])
}

func TestMigrateSsnToNationalID_AssertionRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"ssn": "123-45-6789"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "ssn", Value: "123-45-6789"},
			"a-2": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "occupation", Value: "farmer"},
			// Non-person subject — a custom "ssn" property on an event must NOT
			// be renamed by this migration.
			"a-3": {Subject: glxlib.EntityRef{Event: "event-1"}, Property: "ssn", Value: "x"},
		},
	}

	report := migrateSsnToNationalID(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.SsnAssertionsRenamed)
	assert.Equal(t, "national_id", archive.Assertions["a-1"].Property)
	assert.Equal(t, "occupation", archive.Assertions["a-2"].Property)
	assert.Equal(t, "ssn", archive.Assertions["a-3"].Property)
}

func TestMigrateSsnToNationalID_VocabEntryRename(t *testing.T) {
	archive := &glxlib.GLXFile{
		PersonProperties: map[string]*glxlib.PropertyDefinition{
			"ssn": {Label: "Social Security Number", GEDCOM: "SSN"},
		},
	}

	report := migrateSsnToNationalID(archive, &bytes.Buffer{})

	assert.Equal(t, 1, report.SsnVocabEntriesRenamed)
	require.Contains(t, archive.PersonProperties, "national_id")
	assert.NotContains(t, archive.PersonProperties, "ssn")
	// The definition is moved wholesale — the GEDCOM mapping is preserved.
	assert.Equal(t, "SSN", archive.PersonProperties["national_id"].GEDCOM)
}

func TestMigrateSsnToNationalID_DoesNotOverwriteExistingNationalID(t *testing.T) {
	// A person carrying BOTH keys signals a manual conflict: the legacy `ssn`
	// property must be left in place rather than clobbering `national_id`, AND
	// that person's `ssn` assertions must NOT be repointed to `national_id`
	// (otherwise legacy evidence is misattributed and the kept property is
	// inconsistent with its assertions). person-clean carries only `ssn` and is
	// fully migrated in the same run, proving the skip is targeted at the
	// conflicted person rather than disabling assertion renaming globally.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-conflict": {Properties: map[string]any{
				"ssn":         "OLD",
				"national_id": "KEEP",
			}},
			"person-clean": {Properties: map[string]any{"ssn": "123-45-6789"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-conflict": {Subject: glxlib.EntityRef{Person: "person-conflict"}, Property: "ssn", Value: "OLD"},
			"a-clean":    {Subject: glxlib.EntityRef{Person: "person-clean"}, Property: "ssn", Value: "123-45-6789"},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateSsnToNationalID(archive, warn)

	// Conflicted person: property AND assertion both left as `ssn`.
	assert.Equal(t, "OLD", archive.Persons["person-conflict"].Properties["ssn"])
	assert.Equal(t, "KEEP", archive.Persons["person-conflict"].Properties["national_id"])
	assert.Equal(t, "ssn", archive.Assertions["a-conflict"].Property,
		"conflicted person's ssn assertion must NOT be repointed to national_id")

	// Clean person: property and assertion both migrated.
	assert.Equal(t, "123-45-6789", archive.Persons["person-clean"].Properties["national_id"])
	assert.NotContains(t, archive.Persons["person-clean"].Properties, "ssn")
	assert.Equal(t, "national_id", archive.Assertions["a-clean"].Property)

	// Only the clean person's property and assertion are counted.
	assert.Equal(t, 1, report.SsnPropertiesRenamed)
	assert.Equal(t, 1, report.SsnAssertionsRenamed)
	assert.Contains(t, warn.String(), "person-conflict")
	assert.Contains(t, warn.String(), "national_id")
}

func TestMigrateSsnToNationalID_VocabDoesNotOverwriteExistingNationalID(t *testing.T) {
	archive := &glxlib.GLXFile{
		PersonProperties: map[string]*glxlib.PropertyDefinition{
			"ssn":         {Label: "Social Security Number"},
			"national_id": {Label: "National Identification Number"},
		},
	}

	warn := &bytes.Buffer{}
	report := migrateSsnToNationalID(archive, warn)

	assert.Equal(t, 0, report.SsnVocabEntriesRenamed)
	require.Contains(t, archive.PersonProperties, "ssn")
	assert.Equal(t, "National Identification Number", archive.PersonProperties["national_id"].Label)
	assert.Contains(t, warn.String(), "national_id")
}

func TestMigrateSsnToNationalID_NilWriterDoesNotPanic(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"ssn": "123-45-6789"}},
		},
	}

	report := migrateSsnToNationalID(archive, nil)
	assert.Equal(t, 1, report.SsnPropertiesRenamed)
	assert.Equal(t, "123-45-6789", archive.Persons["person-1"].Properties["national_id"])
}

func TestMigrateSsnToNationalID_NoOpOnAlreadyMigratedArchive(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"national_id": "123-45-6789"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "national_id", Value: "123-45-6789"},
		},
	}

	report := migrateSsnToNationalID(archive, &bytes.Buffer{})

	assert.Equal(t, 0, report.SsnPropertiesRenamed)
	assert.Equal(t, 0, report.SsnAssertionsRenamed)
	assert.Equal(t, 0, report.SsnVocabEntriesRenamed)
}

// TestMigrateSsnToNationalID_TemporalShapeValues confirms the rename moves the
// whole value wholesale (map and list shapes), not just scalar strings — even
// though `ssn` is not declared temporal, a hand-authored archive could use any
// shape and the migration must not drop data.
func TestMigrateSsnToNationalID_TemporalShapeValues(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-map": {Properties: map[string]any{
				"ssn": map[string]any{"value": "123-45-6789", "date": "1990"},
			}},
			"person-list": {Properties: map[string]any{
				"ssn": []any{
					map[string]any{"value": "123-45-6789", "date": "1990"},
				},
			}},
		},
	}

	report := migrateSsnToNationalID(archive, &bytes.Buffer{})

	assert.Equal(t, 2, report.SsnPropertiesRenamed)
	mapVal, ok := archive.Persons["person-map"].Properties["national_id"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "123-45-6789", mapVal["value"])
	assert.NotContains(t, archive.Persons["person-map"].Properties, "ssn")
	listVal, ok := archive.Persons["person-list"].Properties["national_id"].([]any)
	require.True(t, ok)
	require.Len(t, listVal, 1)
	assert.NotContains(t, archive.Persons["person-list"].Properties, "ssn")
}

// TestMigrateSsnToNationalID_IsIdempotent runs the migration twice and confirms
// the second run is a no-op.
func TestMigrateSsnToNationalID_IsIdempotent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"ssn": "123-45-6789"}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {Subject: glxlib.EntityRef{Person: "person-1"}, Property: "ssn", Value: "123-45-6789"},
		},
	}

	first := migrateSsnToNationalID(archive, &bytes.Buffer{})
	require.Equal(t, 1, first.SsnPropertiesRenamed)
	require.Equal(t, 1, first.SsnAssertionsRenamed)

	second := migrateSsnToNationalID(archive, &bytes.Buffer{})
	assert.Equal(t, 0, second.SsnPropertiesRenamed)
	assert.Equal(t, 0, second.SsnAssertionsRenamed)
	assert.Equal(t, 0, second.SsnVocabEntriesRenamed)
	assert.Equal(t, "123-45-6789", archive.Persons["person-1"].Properties["national_id"])
	assert.Equal(t, "national_id", archive.Assertions["a-1"].Property)
}

// TestMigrateArchive_SsnSingleFileRoundTrip exercises the full archive
// load → migrate → write → reload pipeline on a single-file archive.
func TestMigrateArchive_SsnSingleFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")

	preRename := `metadata:
  glx_version: "1.0"
persons:
  person-a:
    properties:
      name:
        value: "Alice"
      ssn: "123-45-6789"
`
	require.NoError(t, os.WriteFile(archivePath, []byte(preRename), 0o600))

	t.Cleanup(func() { migrateRenameSsnToNationalID = false })
	migrateRenameSsnToNationalID = true
	require.NoError(t, migrateArchive(archivePath))

	written, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	writtenStr := string(written)
	assert.Contains(t, writtenStr, "national_id")
	assert.Contains(t, writtenStr, "123-45-6789")
	assert.NotContains(t, writtenStr, "ssn")

	reloaded, err := readSingleFileArchive(archivePath, false)
	require.NoError(t, err)
	require.Len(t, reloaded.Persons, 1)
	assert.Equal(t, "123-45-6789", reloaded.Persons["person-a"].Properties["national_id"])
	assert.NotContains(t, reloaded.Persons["person-a"].Properties, "ssn")

	// Idempotency at the fs boundary: a second invocation must be a no-op.
	require.NoError(t, migrateArchive(archivePath))
	second, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Contains(t, string(second), "national_id")
	assert.NotContains(t, string(second), "ssn")
}

// TestMigrateArchive_SsnMultiFileRoundTrip mirrors the round-trip test against a
// multi-file archive. The person data rename happens in memory; the standard
// vocabulary file is regenerated from the embedded (post-#532) definitions on
// write, so both the data and the on-disk vocabulary end up using national_id.
func TestMigrateArchive_SsnMultiFileRoundTrip(t *testing.T) {
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
      ssn: "123-45-6789"
`), 0o600))

	t.Cleanup(func() { migrateRenameSsnToNationalID = false })
	migrateRenameSsnToNationalID = true
	require.NoError(t, migrateArchive(dir))

	personA, err := os.ReadFile(filepath.Join(dir, "persons", "person-a.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(personA), "national_id")
	assert.NotContains(t, string(personA), "ssn")

	// The regenerated standard vocabulary must define national_id, not ssn.
	vocab, err := os.ReadFile(filepath.Join(dir, "vocabularies", "person-properties.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(vocab), "national_id")

	reloaded, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.Len(t, reloaded.Persons, 1)
	assert.Equal(t, "123-45-6789", reloaded.Persons["person-a"].Properties["national_id"])
	assert.NotContains(t, reloaded.Persons["person-a"].Properties, "ssn")

	// Second invocation is a benign no-op.
	require.NoError(t, migrateArchive(dir))
}
