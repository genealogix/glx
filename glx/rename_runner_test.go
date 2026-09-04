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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Hand-written fixtures with deliberately non-canonical formatting (2-space
// indent, quoted scalars, blank lines) so a byte-for-byte comparison proves a
// file was not rewritten.
const (
	renameFixtureBirths = `events:
  event-birth-robert:
    type: birth
    date: "1850-04-12"
    participants:
      - person: person-robert
        role: subject

  event-birth-mary:
    type: birth
    date: "1852-08-30"
    participants:
      - person: person-mary
        role: subject
`
	renameFixtureRobert = `persons:
  person-robert:
    properties:
      name: "Robert Thompson"
`
	renameFixtureMary = `persons:
  person-mary:
    properties:
      name: "Mary Thompson"
`
	renameFixturePlace = `places:
  place-springfield:
    name: "Springfield"
    type: city
`
)

// writeRenameFixture lays out a small multi-file archive with one
// two-entity events file, two single-entity person files, and a place file
// that nothing in the rename touches.
func writeRenameFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"events/event-births.glx":      renameFixtureBirths,
		"persons/person-robert.glx":    renameFixtureRobert,
		"persons/person-mary.glx":      renameFixtureMary,
		"places/place-springfield.glx": renameFixturePlace,
	}
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644))
	}

	return root
}

func TestRenameEntities_PreservesLayoutAndUntouchedFiles(t *testing.T) {
	root := writeRenameFixture(t)
	before, err := os.Stat(root)
	require.NoError(t, err)

	require.NoError(t, renameEntities(root, "person-robert", "person-robert-t", false))

	// #1192: the archive directory itself is never swapped out.
	after, err := os.Stat(root)
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after), "archive directory must keep its identity")

	// #1197: the two-entity file is still one file with both events.
	births, err := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(births), "event-birth-robert:")
	assert.Contains(t, string(births), "event-birth-mary:")
	assert.Contains(t, string(births), "person: person-robert-t")
	assert.NotContains(t, string(births), "person: person-robert\n")
	assert.NoFileExists(t, filepath.Join(root, "events/event-birth-robert.glx"))
	assert.NoFileExists(t, filepath.Join(root, "events/event-birth-mary.glx"))

	// Files the rename does not touch are byte-identical, formatting included.
	mary, err := os.ReadFile(filepath.Join(root, "persons/person-mary.glx"))
	require.NoError(t, err)
	assert.Equal(t, renameFixtureMary, string(mary))
	place, err := os.ReadFile(filepath.Join(root, "places/place-springfield.glx"))
	require.NoError(t, err)
	assert.Equal(t, renameFixturePlace, string(place))

	// The single-entity file named after the old ID follows the entity.
	assert.NoFileExists(t, filepath.Join(root, "persons/person-robert.glx"))
	robert, err := os.ReadFile(filepath.Join(root, "persons/person-robert-t.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(robert), "person-robert-t:")

	// No standard vocabularies are materialized as a side effect.
	assert.NoDirExists(t, filepath.Join(root, "vocabularies"))

	// The result still loads as a coherent archive.
	archive, _, err := LoadArchiveWithOptions(root, false)
	require.NoError(t, err)
	assert.True(t, archive.HasEntity("person-robert-t"))
	assert.False(t, archive.HasEntity("person-robert"))
	assert.Equal(t, "person-robert-t", archive.Events["event-birth-robert"].Participants[0].Person)
}

func TestRenameEntities_MultiEntityFileKeepsNameWhenEntityRenamed(t *testing.T) {
	// Renaming an entity that lives in a multi-entity file must not rename
	// or split that file even though the entity's own key changes.
	root := writeRenameFixture(t)

	require.NoError(t, renameEntities(root, "event-birth-robert", "event-birth-robert-1850", false))

	births, err := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(births), "event-birth-robert-1850:")
	assert.Contains(t, string(births), "event-birth-mary:")
	entries, err := os.ReadDir(filepath.Join(root, "events"))
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestRenameEntities_DryRunWritesNothing(t *testing.T) {
	root := writeRenameFixture(t)

	require.NoError(t, renameEntities(root, "person-robert", "person-robert-t", true))

	births, err := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Equal(t, renameFixtureBirths, string(births))
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
	assert.NoFileExists(t, filepath.Join(root, "persons/person-robert-t.glx"))
}

func TestRenameEntities_ErrorsWhenOldIDMissing(t *testing.T) {
	root := writeRenameFixture(t)

	err := renameEntities(root, "person-nobody", "person-someone", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenameEntities_ErrorsWhenNewIDTaken(t *testing.T) {
	root := writeRenameFixture(t)

	err := renameEntities(root, "person-robert", "person-mary", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	// Nothing was written.
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
}

func TestRenameEntities_ErrorsWhenTargetFileExists(t *testing.T) {
	// The new ID is free in the archive, but a file with that name already
	// sits in the directory (e.g. holding a differently-named entity).
	root := writeRenameFixture(t)
	stray := filepath.Join(root, "persons/person-robert-t.glx")
	require.NoError(t, os.WriteFile(stray, []byte("persons:\n  person-other:\n    properties: {}\n"), 0o644))

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "target file already exists")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
}

func TestRenameEntities_RollsBackOnWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	root := writeRenameFixture(t)

	// The plan is applied in sorted path order with creates/rewrites first:
	// events/event-births.glx is rewritten, then persons/person-robert-t.glx
	// is created. Making persons/ read-only fails the create after the
	// events rewrite has already happened, which must then be undone.
	personsDir := filepath.Join(root, "persons")
	require.NoError(t, os.Chmod(personsDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(personsDir, 0o755) })

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "all changes rolled back")

	births, err := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Equal(t, renameFixtureBirths, string(births), "rewritten file must be restored byte-for-byte")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
	assert.NoFileExists(t, filepath.Join(root, "persons/person-robert-t.glx"))
}

func TestApplyFileOps_RollbackRemovesCreatedFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}

	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "a"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "b"), 0o755))
	require.NoError(t, os.Chmod(filepath.Join(root, "b"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "b"), 0o755) })

	ops := []fileOp{
		{relPath: filepath.Join("a", "new.glx"), newData: []byte("created\n")},
		{relPath: filepath.Join("b", "blocked.glx"), newData: []byte("never\n")},
	}

	err := applyFileOps(root, ops)

	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(root, "a", "new.glx"), "created file must be removed on rollback")
}

func TestRenameEntities_SingleFileArchive(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(renameFixtureRobert+renameFixtureBirths), 0o644))

	require.NoError(t, renameEntities(path, "person-robert", "person-robert-t", false))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "person-robert-t:")
	assert.Contains(t, string(data), "person: person-robert-t")
	assert.NotContains(t, string(data), "person-robert:")
}

func TestRenameEntities_RejectsCaseOnlyCollisionWithOtherEntity(t *testing.T) {
	root := writeRenameFixture(t)

	err := renameEntities(root, "person-robert", "Person-Mary", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "case-insensitive")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
	assert.FileExists(t, filepath.Join(root, "persons/person-mary.glx"))
}

func TestRenameEntities_CaseOnlyChangeKeepsFilename(t *testing.T) {
	// person-robert -> Person-Robert maps to the same canonical filename, so
	// the file is rewritten in place rather than deleted and recreated
	// (which on a case-insensitive filesystem would delete the new file).
	root := writeRenameFixture(t)

	require.NoError(t, renameEntities(root, "person-robert", "Person-Robert", false))

	entries, err := os.ReadDir(filepath.Join(root, "persons"))
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.ElementsMatch(t, []string{"person-mary.glx", "person-robert.glx"}, names)
	data, err := os.ReadFile(filepath.Join(root, "persons/person-robert.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "Person-Robert:")
}

func TestRenameEntities_MixedCaseIDMatchesCanonicalFilename(t *testing.T) {
	// A mixed-case ID lives in its canonical lowercased filename. Renaming it
	// must move the file, not rewrite the old name in place.
	root := writeRenameFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/person-mixed.glx"),
		[]byte("persons:\n  Person-Mixed:\n    properties:\n      name: Mixed\n"), 0o644))

	require.NoError(t, renameEntities(root, "Person-Mixed", "person-other", false))

	assert.NoFileExists(t, filepath.Join(root, "persons/person-mixed.glx"))
	assert.FileExists(t, filepath.Join(root, "persons/person-other.glx"))
}

func TestRenameEntities_RejectsUnsafeNewIDBeforeWriting(t *testing.T) {
	// The entity lives in a multi-entity file, so no filename would be
	// derived for it today; the ID must still be rejected up front.
	root := writeRenameFixture(t)
	before, err := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)

	err = renameEntities(root, "event-birth-robert", "../escape", false)

	require.Error(t, err)
	after, readErr := os.ReadFile(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, readErr)
	assert.Equal(t, before, after)
}

func TestRenameEntities_RefusesToWriteThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs elevated privileges on Windows")
	}

	// events/event-births.glx is a symlink to a file outside the archive.
	// The loader follows it when reading (so the rename plan touches it);
	// the writer must refuse rather than clobber the external file.
	root := writeRenameFixture(t)
	outside := filepath.Join(t.TempDir(), "external-births.glx")
	require.NoError(t, os.WriteFile(outside, []byte(renameFixtureBirths), 0o644))
	link := filepath.Join(root, "events/event-births.glx")
	require.NoError(t, os.Remove(link))
	require.NoError(t, os.Symlink(outside, link))

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.ErrorIs(t, err, ErrRenameThroughSymlink)
	external, readErr := os.ReadFile(outside)
	require.NoError(t, readErr)
	assert.Equal(t, renameFixtureBirths, string(external), "file outside the archive must be untouched")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"), "nothing else may have been written")
	assert.NoFileExists(t, filepath.Join(root, "persons/person-robert-t.glx"))
}
