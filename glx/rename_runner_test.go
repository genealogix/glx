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
	"io"
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

// --- error paths -----------------------------------------------------------

func TestRenameEntities_MissingArchivePath(t *testing.T) {
	err := renameEntities(filepath.Join(t.TempDir(), "nope"), "person-a", "person-b", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestRenameEntities_InvalidYAMLInArchive(t *testing.T) {
	root := writeRenameFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/broken.glx"), []byte("persons: [unclosed\n"), 0o644))

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load archive")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
}

func TestRenameEntities_WarnsOnDuplicateIDs(t *testing.T) {
	// Two files defining the same entity: the loader reports a duplicate
	// warning on stderr but the rename still proceeds.
	root := writeRenameFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/person-mary-again.glx"), []byte(renameFixtureMary), 0o644))

	stderr := captureStderr(t, func() {
		require.NoError(t, renameEntities(root, "person-robert", "person-robert-t", false))
	})

	assert.Contains(t, stderr, "Warning:")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert-t.glx"))
}

func TestRenameEntities_SingleFileInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte("persons: [unclosed\n"), 0o644))

	err := renameEntities(path, "person-a", "person-b", false)

	require.Error(t, err)
}

func TestRenameEntities_SingleFileUnknownID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(renameFixtureRobert), 0o644))

	err := renameEntities(path, "person-nobody", "person-b", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenameEntities_SingleFileDryRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(renameFixtureRobert), 0o644))

	require.NoError(t, renameEntities(path, "person-robert", "person-robert-t", true))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, renameFixtureRobert, string(data))
}

func TestRenameEntities_UnsafeOldIDIsRewrittenInPlace(t *testing.T) {
	// An entity whose ID cannot be a filename (legacy data) has no canonical
	// file to move; the containing file is rewritten in place.
	root := writeRenameFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/odd.glx"),
		[]byte("persons:\n  \"bad:id\":\n    properties:\n      name: Odd\n"), 0o644))

	require.NoError(t, renameEntities(root, "bad:id", "person-fixed", false))

	data, err := os.ReadFile(filepath.Join(root, "persons/odd.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "person-fixed:")
	assert.NoFileExists(t, filepath.Join(root, "persons/person-fixed.glx"))
}

func TestFileNamedAfterEntity(t *testing.T) {
	assert.True(t, fileNamedAfterEntity("persons/person-a.glx", "person-a"))
	assert.True(t, fileNamedAfterEntity("persons/Person-A.glx", "person-a"), "case-insensitive")
	assert.True(t, fileNamedAfterEntity("persons/person-a.glx", "Person-A"), "canonical name is lowercased")
	assert.False(t, fileNamedAfterEntity("persons/person-b.glx", "person-a"))
	assert.False(t, fileNamedAfterEntity("persons/person-a.yaml", "person-a"))
	assert.False(t, fileNamedAfterEntity("persons/x.glx", "bad:id"), "unsafe ID has no canonical filename")
}

func TestPreflightFileOps_RejectsEscapingPath(t *testing.T) {
	root := t.TempDir()

	err := preflightFileOps(root, []fileOp{{relPath: filepath.Join("..", "outside.glx"), newData: []byte("x")}})

	require.ErrorIs(t, err, ErrRenamePathEscapesArchive)
}

func TestPreflightFileOps_RejectsMissingSourceFile(t *testing.T) {
	root := t.TempDir()

	err := preflightFileOps(root, []fileOp{{relPath: "gone.glx", oldData: []byte("old"), newData: []byte("new")}})

	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestPreflightFileOps_RejectsCreateOntoExistingFile(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "taken.glx"), []byte("x"), 0o644))

	err := preflightFileOps(root, []fileOp{{relPath: "taken.glx", newData: []byte("new")}})

	require.ErrorIs(t, err, ErrRenameTargetFileExists)
}

func TestPreflightFileOps_ReportsUnexpectedStatError(t *testing.T) {
	// A path component that is a regular file makes Lstat fail with
	// ENOTDIR rather than ENOENT.
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644))

	err := preflightFileOps(root, []fileOp{{relPath: filepath.Join("file", "x.glx"), newData: []byte("new")}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "checking")
}

func TestExecuteFileOp_RemoveFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "ro")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x.glx"), []byte("x"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := executeFileOp(root, &fileOp{relPath: filepath.Join("ro", "x.glx"), oldData: []byte("x")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove")
}

func TestExecuteFileOp_MkdirFailure(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644))

	err := executeFileOp(root, &fileOp{relPath: filepath.Join("file", "x.glx"), newData: []byte("new")})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestRollbackFileOps_MissingCreatedFileIsFine(t *testing.T) {
	root := t.TempDir()

	err := rollbackFileOps(root, []fileOp{{relPath: "never-made.glx", newData: []byte("x")}})

	require.NoError(t, err)
}

func TestRollbackFileOps_ReportsRestoreFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "ro")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "x.glx"), []byte("new"), 0o644))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := rollbackFileOps(root, []fileOp{{relPath: filepath.Join("ro", "x.glx"), oldData: []byte("old"), newData: []byte("new")}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "restore")
}

// captureStderr runs fn with os.Stderr redirected to a pipe and returns what
// was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()
	require.NoError(t, w.Close())
	data, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(data)
}

func TestRenameEntities_UnreadableFileInArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	root := writeRenameFixture(t)
	locked := filepath.Join(root, "places/place-springfield.glx")
	require.NoError(t, os.Chmod(locked, 0o000))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o644) })

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load archive")
	assert.FileExists(t, filepath.Join(root, "persons/person-robert.glx"))
}

func TestRenameEntities_SingleFileAcceptsIDsThatAreNotFilenames(t *testing.T) {
	// Single-file archives never derive per-entity filenames, so an ID
	// that could not be a filename is still a valid rename target there.
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(renameFixtureRobert), 0o644))

	require.NoError(t, renameEntities(path, "person-robert", "legacy:robert", false))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "legacy:robert")
}

func TestRenameEntities_PreservesFileModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not preserved on Windows")
	}
	root := writeRenameFixture(t)
	require.NoError(t, os.Chmod(filepath.Join(root, "events/event-births.glx"), 0o600))
	require.NoError(t, os.Chmod(filepath.Join(root, "persons/person-robert.glx"), 0o600))

	require.NoError(t, renameEntities(root, "person-robert", "person-robert-t", false))

	rewritten, err := os.Stat(filepath.Join(root, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), rewritten.Mode().Perm(), "rewritten file keeps its mode")
	moved, err := os.Stat(filepath.Join(root, "persons/person-robert-t.glx"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), moved.Mode().Perm(), "moved file inherits the source mode")
}

func TestRenameEntities_RollbackPreservesFileModes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission bits are not enforced on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := writeRenameFixture(t)
	private := filepath.Join(root, "events/event-births.glx")
	require.NoError(t, os.Chmod(private, 0o600))
	personsDir := filepath.Join(root, "persons")
	require.NoError(t, os.Chmod(personsDir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(personsDir, 0o755) })

	err := renameEntities(root, "person-robert", "person-robert-t", false)

	require.Error(t, err)
	info, statErr := os.Stat(private)
	require.NoError(t, statErr)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "rolled-back file keeps its mode")
}

func TestRenameEntities_SameIDUnderTwoTypesRenamesOnlySelected(t *testing.T) {
	// persons/shared-id.glx and events/shared-id.glx both define "shared-id".
	// RenameEntity resolves it as a person (canonical type order), so only
	// the person file moves; the event keeps its ID and file.
	root := writeRenameFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/shared-id.glx"),
		[]byte("persons:\n  shared-id:\n    properties:\n      name: Shared\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "events"), 0o755))
	eventFile := filepath.Join(root, "events/shared-id.glx")
	eventData := "events:\n  shared-id:\n    type: birth\n    date: \"1900-01-01\"\n"
	require.NoError(t, os.WriteFile(eventFile, []byte(eventData), 0o644))

	require.NoError(t, renameEntities(root, "shared-id", "person-shared", false))

	assert.FileExists(t, filepath.Join(root, "persons/person-shared.glx"))
	assert.NoFileExists(t, filepath.Join(root, "persons/shared-id.glx"))
	after, err := os.ReadFile(eventFile)
	require.NoError(t, err)
	assert.Equal(t, eventData, string(after), "event file with the same ID must be untouched")
}
