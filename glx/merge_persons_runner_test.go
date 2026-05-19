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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// writeMergeFixture creates a minimal multi-file archive at dir containing
// two persons (one to keep, one to drop) and one event whose participant
// is the drop person.
func writeMergeFixture(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "events"), 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-keep.glx"),
		[]byte(`persons:
  person-keep:
    properties:
      name:
        value: "Hans Juncker"
        fields:
          given: "Hans"
          surname: "Juncker"
      sex: "male"
    notes: "primary record"
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-drop.glx"),
		[]byte(`persons:
  person-drop:
    properties:
      name:
        value: "Hans Jungk"
        fields:
          given: "Hans"
          surname: "Jungk"
      occupation: "blacksmith"
    notes: "from spelling-variant record"
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "events", "event-baptism.glx"),
		[]byte(`events:
  event-baptism:
    type: baptism
    date: "1750-04-12"
    participants:
      - person: person-drop
        role: subject
`), 0o644))
}

func TestMergePersonsRunner_DiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)

	err := mergePersons(dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false)
	require.NoError(t, err)

	// Drop file should be gone
	_, err = os.Stat(filepath.Join(dir, "persons", "person-drop.glx"))
	assert.True(t, os.IsNotExist(err), "drop person file should be removed")

	// Reload archive and verify
	archive, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)

	assert.Contains(t, archive.Persons, "person-keep")
	assert.NotContains(t, archive.Persons, "person-drop")

	// Event participant rewritten
	require.Contains(t, archive.Events, "event-baptism")
	require.Len(t, archive.Events["event-baptism"].Participants, 1)
	assert.Equal(t, "person-keep", archive.Events["event-baptism"].Participants[0].Person)

	// Drop's occupation merged in (keep didn't have one)
	assert.Equal(t, "blacksmith", archive.Persons["person-keep"].Properties["occupation"])

	// Notes appended
	assert.Equal(t, glxlib.NoteList{"primary record", "from spelling-variant record"},
		archive.Persons["person-keep"].Notes)
}

func TestMergePersonsRunner_DryRun(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)

	err := mergePersons(dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true)
	require.NoError(t, err)

	// Both person files should still exist
	_, err = os.Stat(filepath.Join(dir, "persons", "person-keep.glx"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "persons", "person-drop.glx"))
	require.NoError(t, err, "drop file should NOT be removed in dry-run")

	// Reload — drop should still be present, event participant unchanged
	archive, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	assert.Contains(t, archive.Persons, "person-drop")
	assert.Equal(t, "person-drop", archive.Events["event-baptism"].Participants[0].Person)
}

func TestMergePersonsRunner_SingleFileArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep:
    properties:
      name: "Keep"
  person-drop:
    properties:
      name: "Drop"
events:
  event-1:
    type: birth
    date: "1800"
    participants:
      - person: person-drop
        role: subject
`), 0o644))

	err := mergePersons(path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false)
	require.NoError(t, err)

	loaded, err := readSingleFileArchive(path, false)
	require.NoError(t, err)
	assert.Contains(t, loaded.Persons, "person-keep")
	assert.NotContains(t, loaded.Persons, "person-drop")
	assert.Equal(t, "person-keep", loaded.Events["event-1"].Participants[0].Person)
}

func TestMergePersonsRunner_MissingArchive(t *testing.T) {
	err := mergePersons(filepath.Join(t.TempDir(), "nope"), "a", "b",
		glxlib.MergePersonsOptions{}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestMergePersonsRunner_MissingPerson(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)

	err := mergePersons(dir, "person-keep", "person-nonexistent",
		glxlib.MergePersonsOptions{}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
