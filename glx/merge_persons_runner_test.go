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
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

	err := mergePersons(dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true)
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

	err := mergePersons(dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, false)
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
relationships:
  rel-resolved:
    type: possibly_same_person
    participants:
      - person: person-drop
      - person: person-keep
`), 0o644))

	err := mergePersons(path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true)
	require.NoError(t, err)

	loaded, err := readSingleFileArchive(path, false)
	require.NoError(t, err)
	assert.Contains(t, loaded.Persons, "person-keep")
	assert.NotContains(t, loaded.Persons, "person-drop")
	assert.Equal(t, "person-keep", loaded.Events["event-1"].Participants[0].Person)
	assert.Empty(t, loaded.Relationships, "the single-file archive must not retain a self-link")
}

func TestMergePersonsRunner_MissingArchive(t *testing.T) {
	err := mergePersons(filepath.Join(t.TempDir(), "nope"), "a", "b",
		glxlib.MergePersonsOptions{}, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestMergePersonsRunner_MissingPerson(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)

	err := mergePersons(dir, "person-keep", "person-nonexistent",
		glxlib.MergePersonsOptions{}, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMergePersonsConfirmationAndDryRun(t *testing.T) {
	cases := []struct {
		name, answer                                 string
		interactive, yes, dryRun, applied, wantError bool
	}{
		{"yes answer", "y\n", true, false, false, true, false},
		{"full uppercase answer", " YES \n", true, false, false, true, false},
		{"no", "n\n", true, false, false, false, false},
		{"blank", "\n", true, false, false, false, false},
		{"EOF", "", true, false, false, false, false},
		{"partial answer at EOF", "y", true, false, false, false, false},
		{"unrecognized", "maybe\n", true, false, false, false, false},
		{"noninteractive", "y\n", false, false, false, false, true},
		{"explicit yes", "", false, true, false, true, false},
		{"dry run", "", false, false, true, false, false},
		{"dry run and yes", "", false, true, true, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeMergeFixture(t, dir)
			before, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			streams, out, errOut := TestIOStreams()
			err = mergePersonsWithIO(streams, strings.NewReader(tc.answer), tc.interactive, dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, tc.dryRun, tc.yes)
			if tc.wantError {
				require.ErrorIs(t, err, errMergeConfirmationRequired)
			} else {
				require.NoError(t, err)
			}
			assert.Contains(t, out.String(), "Merging person-keep ← person-drop")
			assert.Contains(t, out.String(), "Recovery requires a recorded pre-merge version")
			if tc.interactive && !tc.yes && !tc.dryRun {
				assert.Contains(t, errOut.String(), "Apply this merge? [y/N]")
			} else {
				assert.NotContains(t, errOut.String(), "[y/N]")
			}
			if tc.applied {
				assert.NoFileExists(t, filepath.Join(dir, "persons", "person-drop.glx"))
			} else {
				after, err := collectGLXFilesFromDir(dir)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			}
		})
	}
}

type mergeTestReader func([]byte) (int, error)

func (read mergeTestReader) Read(data []byte) (int, error) { return read(data) }

func TestMergePersonsPreviewChangeRefusesWrite(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)
	path := filepath.Join(dir, "persons", "person-keep.glx")
	original, err := os.ReadFile(path)
	require.NoError(t, err)
	changed := bytes.ReplaceAll(original, []byte("Hans Juncker"), []byte("Concurrent edit"))
	streams, _, _ := TestIOStreams()
	input := mergeTestReader(func(data []byte) (int, error) {
		require.NoError(t, os.WriteFile(path, changed, 0o644))

		return copy(data, "y\n"), nil
	})
	err = mergePersonsWithIO(streams, input, true, dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, false)
	require.ErrorIs(t, err, errMergePreviewChanged)
	assert.FileExists(t, filepath.Join(dir, "persons", "person-drop.glx"))
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, changed, actual)
}

func TestMergePersonsYesDoesNotBypassValidation(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)
	path := filepath.Join(dir, "persons", "person-keep.glx")
	original, err := os.ReadFile(path)
	require.NoError(t, err)
	invalid := bytes.Replace(original, []byte("      sex:"), []byte("      born_on: 1750\n      sex:"), 1)
	require.NoError(t, os.WriteFile(path, invalid, 0o644))
	before, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	streams, _, errOut := TestIOStreams()
	input := mergeTestReader(func([]byte) (int, error) {
		t.Fatal("--yes must not read stdin")

		return 0, io.EOF
	})
	err = mergePersonsWithIO(streams, input, false, dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "removed")
	assert.Empty(t, errOut.String())
	after, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestMergePersonsSingleFilePreservesMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte("persons:\n  person-keep:\n    notes: keep\n  person-drop:\n    notes: drop\n"), 0o600))
	originalInfo, err := os.Stat(path)
	require.NoError(t, err)
	streams, _, _ := TestIOStreams()
	require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, originalInfo.Mode().Perm(), info.Mode().Perm())
}

func TestMergePersonsPreservesCustomLayoutAndUnrelatedYAML(t *testing.T) {
	for _, layout := range []string{"archive.glx", "family/history.glx"} {
		t.Run(layout, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, filepath.FromSlash(layout))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep:
    notes: keep
  person-drop:
    notes: drop
events:
  e:
    type: birth
    participants:
      - person: person-drop # preserve participant comment
        role: subject
        properties:
          age: 0x20 # preserve scalar spelling
relationships:
  resolved:
    type: possibly_same_person
    participants:
      - person: person-drop
      - person: person-keep
`), 0o600))
			originalInfo, err := os.Stat(path)
			require.NoError(t, err)
			foreign := filepath.Join(filepath.Dir(path), "README.txt")
			require.NoError(t, os.WriteFile(foreign, []byte("user research"), 0o600))
			untouched := filepath.Join(dir, "unrelated.glx")
			unchanged := []byte("sources:\n  source-original:\n    type: book\n    title: 'Original source' # retain bytes\n")
			require.NoError(t, os.WriteFile(untouched, unchanged, 0o600))
			streams, _, _ := TestIOStreams()
			require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
			archive, _, err := LoadArchiveWithOptions(dir, false)
			require.NoError(t, err)
			assert.NotContains(t, archive.Persons, "person-drop")
			assert.Empty(t, archive.Relationships)
			assert.Equal(t, "person-keep", archive.Events["e"].Participants[0].Person)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Contains(t, string(data), "person: person-keep # preserve participant comment")
			assert.Contains(t, string(data), "age: 0x20 # preserve scalar spelling")
			data, err = os.ReadFile(untouched)
			require.NoError(t, err)
			assert.Equal(t, unchanged, data)
			data, err = os.ReadFile(foreign)
			require.NoError(t, err)
			assert.Equal(t, "user research", string(data))
			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, originalInfo.Mode().Perm(), info.Mode().Perm())
			assert.NoDirExists(t, dir+".bak")
		})
	}
}

func TestMergePersonsInstallFailureRollsBackOriginalFiles(t *testing.T) {
	dir := t.TempDir()
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	delete(planned, filepath.FromSlash("persons/person-drop.glx"))
	planned[filepath.FromSlash("persons/person-keep.glx")] = []byte("persons:\n  person-keep:\n    notes: merged\n")
	failed := false
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		if !failed && strings.Contains(from, ".glx-merge-") {
			failed = true

			return os.ErrPermission
		}

		return robustRenameIn(root, from, to)
	})
	require.ErrorIs(t, err, os.ErrPermission)
	require.True(t, failed, "failure must happen after original files have moved to backup")
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
	assert.NoDirExists(t, dir+".bak")
}

func TestMergePersonsInstallRejectsDirectorySymlinkRace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	dir := t.TempDir()
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	planned[filepath.FromSlash("persons/person-keep.glx")] = []byte("persons:\n  person-keep:\n    notes: merged\n")
	delete(planned, filepath.FromSlash("persons/person-drop.glx"))
	outside := t.TempDir()
	untouched := []byte("outside archive")
	outsideFile := filepath.Join(outside, "person-keep.glx")
	require.NoError(t, os.WriteFile(outsideFile, untouched, 0o600))
	replaced := false
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		if !replaced && strings.Contains(from, filepath.Join("new", "persons")) {
			replaced = true
			require.NoError(t, os.Rename(filepath.Join(dir, "persons"), filepath.Join(dir, "persons-original-directory")))
			require.NoError(t, os.Symlink(outside, filepath.Join(dir, "persons")))
		}

		return robustRenameIn(root, from, to)
	})
	require.Error(t, err)
	require.True(t, replaced, "race must happen after all original files have moved")
	data, err := os.ReadFile(outsideFile)
	require.NoError(t, err)
	assert.Equal(t, untouched, data, "installation and rollback must stay inside the archive")
	assert.NoFileExists(t, filepath.Join(outside, "person-drop.glx"))
	backup, err := collectGLXFilesFromDir(dir + ".bak")
	require.NoError(t, err)
	assert.Equal(t, original[filepath.FromSlash("persons/person-keep.glx")], backup[filepath.FromSlash("persons/person-keep.glx")])
	assert.Equal(t, original[filepath.FromSlash("persons/person-drop.glx")], backup[filepath.FromSlash("persons/person-drop.glx")])
	require.ErrorIs(t, removeStaleBackup(dir+".bak"), ErrInterruptedSwap)
}

func TestMergePersonsStaleBackupPreservesRecoveryData(t *testing.T) {
	for _, path := range []string{swapInProgressMarker, "README.txt", "persons/.drafts/record.glx", "persons/research.txt", "media/files/record.glx"} {
		t.Run(path, func(t *testing.T) {
			parentDir := t.TempDir()
			name := "archive.bak"
			file := filepath.Join(parentDir, name, filepath.FromSlash(path))
			require.NoError(t, os.MkdirAll(filepath.Dir(file), 0o755))
			require.NoError(t, os.WriteFile(file, []byte("recovery data"), 0o600))
			parent, err := os.OpenRoot(parentDir)
			require.NoError(t, err)
			defer func() { _ = parent.Close() }()
			require.Error(t, removeStaleMergeBackup(parent, name))
			data, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, "recovery data", string(data))
		})
	}
}

func TestMergePersonsStaleBackupCleanupStaysInOpenedParent(t *testing.T) {
	container := t.TempDir()
	parentDir := filepath.Join(container, "parent")
	backupFile := filepath.Join("archive.bak", "persons", "person.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(parentDir, backupFile)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, backupFile), []byte("stale GLX"), 0o600))
	parent, err := os.OpenRoot(parentDir)
	require.NoError(t, err)
	defer func() { _ = parent.Close() }()
	moved := filepath.Join(container, "moved-parent")
	require.NoError(t, os.Rename(parentDir, moved))
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(parentDir, backupFile)), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(parentDir, backupFile), []byte("replacement tree"), 0o600))
	require.NoError(t, removeStaleMergeBackup(parent, "archive.bak"))
	assert.NoDirExists(t, filepath.Join(moved, "archive.bak"))
	data, err := os.ReadFile(filepath.Join(parentDir, backupFile))
	require.NoError(t, err)
	assert.Equal(t, "replacement tree", string(data))
}

func TestMergePersonsParticipantCommentsRemainWithTheirOriginalEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep: {}
  person-drop: {}
events:
  event-birth:
    type: birth
    participants:
      - person: person-keep # first participant
        role: subject
        properties:
          age: 32
      - person: person-drop # second participant
        role: subject
        properties:
          age: 0x20 # second scalar
`), 0o600))
	streams, _, _ := TestIOStreams()
	require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "person: person-keep # first participant")
	assert.Contains(t, string(data), "person: person-keep # second participant")
	assert.Contains(t, string(data), "age: 0x20 # second scalar")
}

func TestMergePersonsTypedReferenceListsKeepOccurrenceComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep: {}
  person-drop: {}
  person-third:
    properties:
      mentors:
        - person-keep # first reference
        - person-drop # second reference
person_properties:
  mentors:
    label: Mentors
    reference_type: persons
    multi_value: true
`), 0o600))
	streams, _, _ := TestIOStreams()
	require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "person-keep # first reference")
	assert.Contains(t, string(data), "person-keep # second reference")
}

func TestMergePersonsUnionedReferenceListsRetainBothPersonsComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep:
    properties:
      mentors:
        - person-keep # keep occurrence
  person-drop:
    properties:
      mentors:
        - person-keep # already present in keep
        - person-drop # drop occurrence
person_properties:
  mentors:
    label: Mentors
    reference_type: persons
    multi_value: true
`), 0o600))
	streams, _, _ := TestIOStreams()
	require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, path, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "person-keep # keep occurrence")
	assert.Contains(t, string(data), "person-keep # drop occurrence")
}
