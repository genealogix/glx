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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func preserveTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"primary_name": "Alice"}},
		},
	}
}

// A dot entry inside media/files/ must not break the safe-write swap. The
// dot-entry preservation walk used to run first and recreate media/files/ in
// the destination to hold .keep, so the whole-directory rename of media/files/
// that followed failed on an existing target and the write errored out.
func TestSafeWrite_PreservesDotEntryUnderMediaFiles(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, "media", "files", ".keep"), "")
	writeSkipTestFile(t, filepath.Join(archiveDir, "media", "files", "portrait.jpg"), "jpeg bytes")

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, filepath.Join(archiveDir, "media", "files", ".keep"))
	assert.FileExists(t, filepath.Join(archiveDir, "media", "files", "portrait.jpg"))
	assert.FileExists(t, filepath.Join(archiveDir, "persons", "person-1.glx"))
}

// Dot-prefixed entries nested inside a managed directory are skipped by the
// loader, so the serializer never re-emits them; the swap must carry them over.
func TestSafeWrite_PreservesNestedDotEntries(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	draft := filepath.Join(archiveDir, "persons", ".drafts", "draft.glx")
	sidecar := filepath.Join(archiveDir, "persons", "._person-1.glx")
	writeSkipTestFile(t, draft, "persons: {}")
	writeSkipTestFile(t, sidecar, "not glx")

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, draft)
	assert.FileExists(t, sidecar)
	assert.FileExists(t, filepath.Join(archiveDir, "persons", "person-1.glx"))
}

func TestFirstSkippedEntry(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "person-1.glx"), "persons: {}")

	found, err := firstSkippedEntry(dir)
	require.NoError(t, err)
	assert.Empty(t, found)

	writeSkipTestFile(t, filepath.Join(dir, "persons", ".drafts", "x.glx"), "persons: {}")

	found, err = firstSkippedEntry(dir)
	require.NoError(t, err)
	assert.Equal(t, "persons/.drafts", found)
}

// The loader follows an ordinary symlink and reads its target, so the cache
// fingerprint has to describe the target too. Recording the link's own
// metadata let edits to the target leave the fingerprint unchanged and the
// cache serve stale entities.
func TestComputeFSFingerprint_FollowsSymlinkTarget(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "data", "current.txt")
	writeSkipTestFile(t, target, "persons: {}")
	link := filepath.Join(root, "persons", "current.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(filepath.Join("..", "data", "current.txt"), link))

	before, err := computeFSFingerprint(root)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(target, []byte("persons: {}\n# edited through the target\n"), 0o644))

	after, err := computeFSFingerprint(root)
	require.NoError(t, err)
	assert.NotEqual(t, before, after, "editing a symlink's target must change the fingerprint")
}

func TestValidateSingleFileSemantics_SkipsDotEntries(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "good.glx"), "persons: {}\n")
	hidden := filepath.Join(dir, "persons", ".hidden.glx")
	writeSkipTestFile(t, hidden, "persons: [\n")
	writeSkipTestFile(t, filepath.Join(dir, "persons", ".drafts", "bad.glx"), "persons: [\n")

	errs, _ := validateSingleFileSemantics([]string{dir})
	assert.Empty(t, errs, "dot-prefixed entries discovered by the walk are skipped")

	errs, _ = validateSingleFileSemantics([]string{hidden})
	assert.NotEmpty(t, errs, "a dot-prefixed file named explicitly is still validated")
}

func TestValidateMediaFileExistence_DotComponentWarns(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, ".stash", "x.jpg"), "jpeg")
	archive := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{
			"media-1": {URI: ".stash/x.jpg"},
		},
	}

	warnings := validateMediaFileExistence(archive, root)

	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "dot-prefixed path")
	assert.Contains(t, warnings[0], "media-1")
}

func TestValidateAndReport(t *testing.T) {
	t.Run("rejects more than one path", func(t *testing.T) {
		streams, _, _ := newTestStreams()
		err := validateAndReport(streams, []string{"a", "b"})
		assert.ErrorIs(t, err, errReportTooManyArgs)
	})

	t.Run("validation gates the report", func(t *testing.T) {
		dir := t.TempDir()
		writeSkipTestFile(t, filepath.Join(dir, "persons", "bad.glx"), "persons: [\n")
		streams, _, _ := newTestStreams()
		err := validateAndReport(streams, []string{dir})
		assert.Error(t, err, "an archive that fails validation must not produce a report")
	})
}

// Top-level dot entries (.gitignore, .git/) are carried across the swap by the
// non-archive-file preservation that runs first, so the dot-entry walk finds
// them already present in the destination and must leave them alone.
func TestSafeWrite_ToleratesDotEntriesAlreadyCarriedAtTopLevel(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".gitignore"), "*.bak\n")
	writeSkipTestFile(t, filepath.Join(archiveDir, ".git", "HEAD"), "ref: refs/heads/main\n")

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, filepath.Join(archiveDir, ".gitignore"))
	assert.FileExists(t, filepath.Join(archiveDir, ".git", "HEAD"))
}

// A leftover backup from a failed run that still holds a nested dot entry is
// unrecovered data; the next safe write must refuse rather than delete it.
func TestSafeWrite_RefusesStaleBackupWithNestedDotEntry(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	stale := archiveDir + ".bak"
	writeSkipTestFile(t, filepath.Join(stale, "persons", "person-1.glx"), "persons: {}\n")
	writeSkipTestFile(t, filepath.Join(stale, "persons", ".drafts", "draft.glx"), "persons: {}\n")

	err := safeWriteMultiFileArchive(archiveDir, preserveTestArchive())

	require.ErrorIs(t, err, ErrStaleBackupForeignFile)
	assert.FileExists(t, filepath.Join(stale, "persons", ".drafts", "draft.glx"), "the stale backup must be left for the user to inspect")
}

func TestCollectGLXFiles_DanglingSymlinkIsAnError(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}\n")
	require.NoError(t, os.Symlink("missing-target.glx", filepath.Join(dir, "persons", "dangling.glx")))

	_, err := collectGLXFilesFromDir(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "dangling.glx")
}

func TestValidatePaths_UnreadableEntryIsStructuralFailure(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}\n")
	require.NoError(t, os.Symlink("missing-target.glx", filepath.Join(dir, "persons", "dangling.glx")))
	streams, _, _ := newTestStreams()

	err := validatePaths(streams, []string{dir})

	assert.ErrorIs(t, err, ErrStructuralValidationFailed)
}

func TestValidateAndReport_ValidArchiveProducesReport(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "person-1.glx"), "persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	streams, _, _ := newTestStreams()

	assert.NoError(t, validateAndReport(streams, []string{dir}))
}

func TestValidateSingleFileSemantics_IgnoresNonGLXFiles(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "good.glx"), "persons: {}\n")
	writeSkipTestFile(t, filepath.Join(dir, "persons", "notes.txt"), "not: [yaml\n")

	errs, _ := validateSingleFileSemantics([]string{dir})

	assert.Empty(t, errs)
}

func TestComputeFSFingerprint_DanglingSymlinkFallsBack(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "persons", "a.glx"), "persons: {}\n")
	require.NoError(t, os.Symlink("missing-target.glx", filepath.Join(root, "persons", "dangling.glx")))

	_, err := computeFSFingerprint(root)

	assert.NoError(t, err, "a dangling link falls back to the entry's own metadata instead of failing the walk")
}

// Walk errors inside the backup tree must surface as errors rather than be
// swallowed: an unreadable directory means the preservation walks cannot
// prove what they would be deleting.
func TestSafeWrite_UnreadableBackupDirectoryIsAnError(t *testing.T) {
	if runtime.GOOS == goosWindows || os.Geteuid() == 0 {
		t.Skip("relies on permission bits being enforced")
	}
	unreadable := func(t *testing.T, dir string) {
		t.Helper()
		require.NoError(t, os.Chmod(dir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	}

	t.Run("stale backup", func(t *testing.T) {
		archiveDir := filepath.Join(t.TempDir(), "archive")
		require.NoError(t, os.MkdirAll(archiveDir, 0o755))
		require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
		stale := archiveDir + ".bak"
		writeSkipTestFile(t, filepath.Join(stale, "persons", "person-1.glx"), "persons: {}\n")
		unreadable(t, filepath.Join(stale, "persons"))

		err := safeWriteMultiFileArchive(archiveDir, preserveTestArchive())

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrStaleBackupForeignFile, "an inspection failure is not the same as a foreign file")
	})

	t.Run("dot-entry walk", func(t *testing.T) {
		archiveDir := filepath.Join(t.TempDir(), "archive")
		require.NoError(t, os.MkdirAll(archiveDir, 0o755))
		require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
		locked := filepath.Join(archiveDir, "persons", "locked")
		writeSkipTestFile(t, filepath.Join(locked, "note.txt"), "x")
		unreadable(t, locked)
		t.Cleanup(func() { _ = os.Chmod(filepath.Join(archiveDir+".bak", "persons", "locked"), 0o755) })

		err := safeWriteMultiFileArchive(archiveDir, preserveTestArchive())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "preserving")
	})
}

// A visible symlink whose target resolves under a dot-prefixed directory is
// skipped by the loader (symlinkTargetIsExcluded) and never re-emitted, so
// the swap must carry it across like a dot-named entry.
func TestSafeWrite_PreservesSymlinkIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".worktrees", "copy", "persons", "alias.glx"), "persons: {}\n")
	link := filepath.Join(archiveDir, "persons", "alias.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy", "persons", "alias.glx"), link))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	info, err := os.Lstat(link)
	require.NoError(t, err, "the excluded symlink must survive the safe write")
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
	assert.NoDirExists(t, archiveDir+".bak")
}

// The loader matches the .glx extension case-insensitively and never filtered
// directory names, so PERSONS/person-1.glx loads. The managed-directory check
// on safe write has to agree, or PERSONS/ is preserved as foreign next to a
// freshly written persons/ and every entity comes back twice.
func TestSafeWrite_UppercaseManagedDirectoryIsReplacedNotDuplicated(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	writeSkipTestFile(t, filepath.Join(archiveDir, "PERSONS", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")

	loaded, _, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	require.Len(t, loaded.Persons, 1)

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, loaded))

	entries, err := os.ReadDir(archiveDir)
	require.NoError(t, err)
	var personDirs []string
	for _, e := range entries {
		if strings.EqualFold(e.Name(), "persons") {
			personDirs = append(personDirs, e.Name())
		}
	}
	assert.Len(t, personDirs, 1, "exactly one persons directory after the write, got %v", personDirs)

	reloaded, duplicates, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	assert.Empty(t, duplicates)
	assert.Len(t, reloaded.Persons, 1)
}

// The fingerprint must cover exactly what the loader reads: edits behind a
// symlink the loader excludes must not invalidate the cache, while retargeting
// a link or a target appearing or disappearing must.
func TestComputeFSFingerprint_SymlinkIdentity(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}

	t.Run("edits behind an excluded symlink do not change it", func(t *testing.T) {
		root := t.TempDir()
		writeSkipTestFile(t, filepath.Join(root, "persons", "a.glx"), "persons: {}\n")
		excludedTarget := filepath.Join(root, ".worktrees", "copy", "persons", "b.glx")
		writeSkipTestFile(t, excludedTarget, "persons: {}\n")
		require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy", "persons", "b.glx"), filepath.Join(root, "persons", "b.glx")))

		before, err := computeFSFingerprint(root)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(excludedTarget, []byte("persons: {}\n# edited copy\n"), 0o644))
		after, err := computeFSFingerprint(root)
		require.NoError(t, err)

		assert.Equal(t, before, after)
	})

	t.Run("retargeting a link changes it even when the targets match", func(t *testing.T) {
		root := t.TempDir()
		one := filepath.Join(root, "data", "one.txt")
		two := filepath.Join(root, "data", "two.txt")
		writeSkipTestFile(t, one, "persons: {}\n")
		writeSkipTestFile(t, two, "persons: {}\n")
		require.NoError(t, os.Chtimes(two, time.Unix(1_700_000_000, 0), time.Unix(1_700_000_000, 0)))
		require.NoError(t, os.Chtimes(one, time.Unix(1_700_000_000, 0), time.Unix(1_700_000_000, 0)))
		link := filepath.Join(root, "persons", "current.glx")
		require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
		require.NoError(t, os.Symlink(filepath.Join("..", "data", "one.txt"), link))

		before, err := computeFSFingerprint(root)
		require.NoError(t, err)
		require.NoError(t, os.Remove(link))
		require.NoError(t, os.Symlink(filepath.Join("..", "data", "two.txt"), link))
		after, err := computeFSFingerprint(root)
		require.NoError(t, err)

		assert.NotEqual(t, before, after)
	})

	t.Run("a target disappearing changes it", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "data", "current.txt")
		writeSkipTestFile(t, target, "persons: {}\n")
		link := filepath.Join(root, "persons", "current.glx")
		require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
		require.NoError(t, os.Symlink(filepath.Join("..", "data", "current.txt"), link))

		before, err := computeFSFingerprint(root)
		require.NoError(t, err)
		require.NoError(t, os.Remove(target))
		after, err := computeFSFingerprint(root)
		require.NoError(t, err)

		assert.NotEqual(t, before, after)
	})
}

// The loader resolves symlink chains; a visible link that reaches a dot
// directory through another visible link is skipped on read and must survive
// the swap the same way a direct link does.
func TestSafeWrite_PreservesSymlinkChainIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".worktrees", "copy", "persons", "alias.glx"), "persons: {}\n")
	hop := filepath.Join(archiveDir, "persons", "hop.glx")
	dup := filepath.Join(archiveDir, "persons", "dup.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy", "persons", "alias.glx"), hop))
	require.NoError(t, os.Symlink("hop.glx", dup))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	for _, link := range []string{hop, dup} {
		info, err := os.Lstat(link)
		require.NoError(t, err, "%s must survive the safe write", link)
		assert.NotZero(t, info.Mode()&os.ModeSymlink)
	}
	assert.NoDirExists(t, archiveDir+".bak")
}

// When a skipped entry sits at the same path as a file the writer produced,
// moving on would let the backup — and the entry — be deleted. The write must
// fail and keep the backup.
func TestSafeWrite_PreservedEntryCollisionRetainsBackup(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	// The archive's person-1 lives in a differently named file, so the
	// serializer will emit persons/person-1.glx — the path the link occupies.
	writeSkipTestFile(t, filepath.Join(archiveDir, "persons", "alice.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(archiveDir, ".drafts", "person-1.glx"), "persons: {}\n")
	link := filepath.Join(archiveDir, "persons", "person-1.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".drafts", "person-1.glx"), link))

	loaded, _, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	err = safeWriteMultiFileArchive(archiveDir, loaded)

	require.ErrorIs(t, err, ErrPreservedEntryCollision)
	info, lerr := os.Lstat(filepath.Join(archiveDir+".bak", "persons", "person-1.glx"))
	require.NoError(t, lerr, "the backup with the skipped link must be retained")
	assert.NotZero(t, info.Mode()&os.ModeSymlink)
}

func TestSafeWrite_RefusesStaleBackupWithSkippedSymlink(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	stale := archiveDir + ".bak"
	writeSkipTestFile(t, filepath.Join(stale, "persons", "person-1.glx"), "persons: {}\n")
	require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy", "alias.glx"), filepath.Join(stale, "persons", "alias.glx")))

	err := safeWriteMultiFileArchive(archiveDir, preserveTestArchive())

	require.ErrorIs(t, err, ErrStaleBackupForeignFile)
	_, lerr := os.Lstat(filepath.Join(stale, "persons", "alias.glx"))
	assert.NoError(t, lerr, "the stale backup must be left for the user to inspect")
}

// Managed directories are recognized case-insensitively, so the binaries of an
// archive laid out as MEDIA/files/ have to be found the same way or a safe
// write deletes them with the backup.
func TestSafeWrite_PreservesMediaBinariesFromCaseVariantDirectory(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	writeSkipTestFile(t, filepath.Join(archiveDir, "PERSONS", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(archiveDir, "MEDIA", "files", "portrait.jpg"), "jpeg bytes")

	loaded, _, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, loaded))

	dirs, derr := mediaFilesDirsIn(archiveDir)
	require.NoError(t, derr)
	require.Len(t, dirs, 1, "exactly one media/files must exist after the write")
	found := dirs[0]
	assert.FileExists(t, filepath.Join(found, "portrait.jpg"))
	assert.NoDirExists(t, archiveDir+".bak")
}

func TestPlaceholderTarget(t *testing.T) {
	target, ok := placeholderTarget("persons/alias.glx", []byte("../.worktrees/copy/persons/p.glx\n"))
	require.True(t, ok)
	assert.Equal(t, ".worktrees/copy/persons/p.glx", target)
	assert.True(t, pathHasDotComponent(target))

	_, ok = placeholderTarget("persons/alias.glx", []byte("persons:\n  person-1: {}\n"))
	assert.False(t, ok, "GLX content is not a placeholder")

	_, ok = placeholderTarget("link.glx", []byte("../secret.glx"))
	assert.False(t, ok, "a target climbing out of the root is not acted on")
}

func TestCollectGLXFiles_SymlinkOutsideRootIsAnError(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	outside := t.TempDir()
	writeSkipTestFile(t, filepath.Join(outside, "secret.glx"), "persons: {}\n")
	root := filepath.Join(outside, "archive")
	writeSkipTestFile(t, filepath.Join(root, "persons", "a.glx"), "persons: {}\n")
	require.NoError(t, os.Symlink(filepath.Join("..", "..", "secret.glx"), filepath.Join(root, "persons", "escape.glx")))

	_, err := collectGLXFilesFromDir(root)

	require.Error(t, err, "a link escaping the archive root must be refused, not read")
	assert.Contains(t, err.Error(), "escape.glx")
}

// Classification must finish before anything moves: z.glx reaches the dot
// directory only through a.glx, and a.glx sorts first.
func TestSafeWrite_PreservesSymlinkChainWhoseHopSortsFirst(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".drafts", "person.glx"), "persons: {}\n")
	a := filepath.Join(archiveDir, "persons", "a.glx")
	z := filepath.Join(archiveDir, "persons", "z.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".drafts", "person.glx"), a))
	require.NoError(t, os.Symlink("a.glx", z))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	for _, link := range []string{a, z} {
		_, err := os.Lstat(link)
		require.NoError(t, err, "%s must survive the safe write", filepath.Base(link))
	}
	assert.NoDirExists(t, archiveDir+".bak")
}

// A managed top-level name (metadata.glx) that is a skipped symlink is not
// carried over by restoreForeignEntries, so a fresh metadata.glx from the
// writer is a genuine collision and must keep the backup.
func TestSafeWrite_ManagedTopLevelCollisionRetainsBackup(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	writeSkipTestFile(t, filepath.Join(archiveDir, "persons", "person-1.glx"),
		"metadata:\n  source_system: test\npersons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(archiveDir, ".drafts", "metadata.glx"), "metadata: {}\n")
	require.NoError(t, os.Symlink(filepath.Join(".drafts", "metadata.glx"), filepath.Join(archiveDir, "metadata.glx")))

	loaded, _, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	if loaded.ImportMetadata == nil {
		t.Skip("metadata is not loaded from an entity file in this build; collision cannot be staged")
	}
	err = safeWriteMultiFileArchive(archiveDir, loaded)

	require.ErrorIs(t, err, ErrPreservedEntryCollision)
	_, lerr := os.Lstat(filepath.Join(archiveDir+".bak", "metadata.glx"))
	assert.NoError(t, lerr, "the backup with the skipped link must be retained")
}

// fsCaseSensitive reports whether dir lives on a case-sensitive filesystem.
func fsCaseSensitive(t *testing.T, dir string) bool {
	t.Helper()
	probe := filepath.Join(dir, "CaseProbe")
	require.NoError(t, os.WriteFile(probe, []byte("x"), 0o644))
	_, err := os.Stat(filepath.Join(dir, "caseprobe"))

	return err != nil
}

func TestSafeWrite_RefusesAmbiguousCaseVariantMediaDirectories(t *testing.T) {
	base := t.TempDir()
	if !fsCaseSensitive(t, base) {
		t.Skip("needs a case-sensitive filesystem to hold media/ and MEDIA/ side by side")
	}
	archiveDir := filepath.Join(base, "archive")
	writeSkipTestFile(t, filepath.Join(archiveDir, "persons", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(archiveDir, "media", "files", "a.jpg"), "a")
	writeSkipTestFile(t, filepath.Join(archiveDir, "MEDIA", "files", "b.jpg"), "b")

	loaded, _, err := LoadArchive(archiveDir)
	require.NoError(t, err)
	err = safeWriteMultiFileArchive(archiveDir, loaded)

	require.ErrorIs(t, err, ErrAmbiguousMediaFilesDirs)
	assert.FileExists(t, filepath.Join(archiveDir+".bak", "MEDIA", "files", "b.jpg"), "neither tree may be deleted")
}

// dirEntryFor returns the fs.DirEntry for name inside dir, as a walk would
// present it.
func dirEntryFor(t *testing.T, dir, name string) fs.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		if e.Name() == name {
			return e
		}
	}
	t.Fatalf("%s not found in %s", name, dir)

	return nil
}

// The Windows placeholder branches cannot run for real on this host, so they
// are exercised with the platform made explicit.
func TestWindowsPlaceholderBranches(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, ".drafts", "p.glx"), "persons: {}\n")
	writeSkipTestFile(t, filepath.Join(root, "persons", "alias.glx"), "../.drafts/p.glx\n")
	writeSkipTestFile(t, filepath.Join(root, "persons", "real.glx"), "persons:\n  person-1: {}\n")
	personsDir := filepath.Join(root, "persons")
	alias := dirEntryFor(t, personsDir, "alias.glx")
	content := dirEntryFor(t, personsDir, "real.glx")

	t.Run("loaderSkips recognizes a placeholder into a dot directory", func(t *testing.T) {
		assert.True(t, loaderSkipsOn(goosWindows, root, root, filepath.Join("persons", "alias.glx"), alias))
		assert.False(t, loaderSkipsOn(goosWindows, root, root, filepath.Join("persons", "real.glx"), content), "ordinary content is not a placeholder")
		assert.False(t, loaderSkipsOn("linux", root, root, filepath.Join("persons", "alias.glx"), alias), "placeholders are a Windows representation only")
	})

	t.Run("fingerprint excludes a placeholder into a dot directory", func(t *testing.T) {
		assert.True(t, placeholderExcludedOn(goosWindows, filepath.Join(personsDir, "alias.glx"), "persons/alias.glx", alias))
		assert.False(t, placeholderExcludedOn(goosWindows, filepath.Join(personsDir, "real.glx"), "persons/real.glx", content))
		assert.False(t, placeholderExcludedOn("linux", filepath.Join(personsDir, "alias.glx"), "persons/alias.glx", alias))
	})
}

// A symlinked intermediate directory hides the dot path from a plain Lstat of
// the whole target; the loader's full resolution sees it, so the swap must
// too.
func TestSafeWrite_PreservesLinkThroughSymlinkedDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".worktrees", "copy", "p.glx"), "persons: {}\n")
	shared := filepath.Join(archiveDir, "persons", "shared")
	alias := filepath.Join(archiveDir, "persons", "alias.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy"), shared))
	require.NoError(t, os.Symlink(filepath.Join("shared", "p.glx"), alias))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	for _, link := range []string{shared, alias} {
		_, err := os.Lstat(link)
		require.NoError(t, err, "%s must survive the safe write", filepath.Base(link))
	}
	assert.NoDirExists(t, archiveDir+".bak")
}

// The loader accepts an absolute target that resolves inside the archive; a
// link written that way into a dot directory is skipped on read and must be
// carried across the swap like a relative one.
func TestSafeWrite_PreservesAbsoluteSymlinkIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	target := filepath.Join(archiveDir, ".worktrees", "copy", "p.glx")
	writeSkipTestFile(t, target, "persons: {}\n")
	alias := filepath.Join(archiveDir, "persons", "alias.glx")
	require.NoError(t, os.Symlink(target, alias))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	_, err := os.Lstat(alias)
	require.NoError(t, err, "the absolute link into the dot directory must survive")
	assert.NoDirExists(t, archiveDir+".bak")
}

// Case folding applies to real managed directories only. A regular file that
// happens to be named PERSONS, or a symlink named Persons, is foreign data
// the loader never read and must be preserved, not treated as replaceable.
func TestSafeWrite_ForeignEntriesNamedLikeManagedDirsArePreserved(t *testing.T) {
	base := t.TempDir()
	if !fsCaseSensitive(t, base) {
		t.Skip("needs a case-sensitive filesystem to hold persons/ and a PERSONS file side by side")
	}
	archiveDir := filepath.Join(base, "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, "PERSONS"), "not an archive directory\n")
	require.NoError(t, os.Symlink("persons", filepath.Join(archiveDir, "Events")))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, filepath.Join(archiveDir, "PERSONS"))
	_, err := os.Lstat(filepath.Join(archiveDir, "Events"))
	assert.NoError(t, err, "a symlink named like a managed directory is foreign and must survive")
}

// A case-variant media directory that is a symlink must not be followed: the
// rename would otherwise move data from outside the archive.
func TestSafeWrite_DoesNotFollowSymlinkedMediaDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	base := t.TempDir()
	external := filepath.Join(base, "assets", "files", "portrait.jpg")
	writeSkipTestFile(t, external, "jpeg bytes")
	archiveDir := filepath.Join(base, "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	require.NoError(t, os.Symlink(filepath.Join("..", "assets"), filepath.Join(archiveDir, "MEDIA")))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, external, "data behind the symlink must stay where it was")
	_, err := os.Lstat(filepath.Join(archiveDir, "MEDIA"))
	assert.NoError(t, err, "the symlink itself is foreign and is preserved")
}

// An unreadable media directory in a stale backup is an inspection failure,
// not evidence that the backup is empty; cleanup must fail closed.
func TestSafeWrite_UnreadableMediaInStaleBackupIsRefused(t *testing.T) {
	if runtime.GOOS == goosWindows || os.Geteuid() == 0 {
		t.Skip("relies on permission bits being enforced")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	stale := archiveDir + ".bak"
	writeSkipTestFile(t, filepath.Join(stale, "media", "files", "portrait.jpg"), "jpeg bytes")
	mediaDir := filepath.Join(stale, "media")
	require.NoError(t, os.Chmod(mediaDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(mediaDir, 0o755) })

	err := safeWriteMultiFileArchive(archiveDir, preserveTestArchive())

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrStaleBackupForeignFile, "an inspection failure is reported as such")
	require.NoError(t, os.Chmod(mediaDir, 0o755))
	assert.FileExists(t, filepath.Join(stale, "media", "files", "portrait.jpg"), "the backup must be left intact")
}

// A skipped link's chain can pass through a top-level foreign entry that
// restoreForeignEntries carries across before the nested entries are looked
// at. Classification has to happen on the intact backup, or the nested link
// resolves no further than the missing hop and is deleted with the backup.
func TestSafeWrite_PreservesLinkThroughTopLevelForeignLink(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, ".drafts", "x.glx"), "persons: {}\n")
	alias := filepath.Join(archiveDir, "alias.glx")
	require.NoError(t, os.Symlink(filepath.Join(".drafts", "x.glx"), alias))
	z := filepath.Join(archiveDir, "persons", "z.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", "alias.glx"), z))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	for _, link := range []string{alias, z} {
		info, err := os.Lstat(link)
		require.NoError(t, err, "%s must survive the safe write", link)
		assert.NotZero(t, info.Mode()&os.ModeSymlink)
	}
	assert.NoDirExists(t, archiveDir+".bak")
}

// The loader reads a .glx file at the archive top level, but the multi-file
// writer has no place for it: preserving it as foreign duplicates its entities
// beside the freshly written entity directories, and dropping it deletes a
// hand-made file. Until #1247 settles which the user wants, the write is
// refused before anything is moved — in any letter case, since the loader's
// extension test is case-insensitive.
func TestSafeWrite_RefusesTopLevelGLXFile(t *testing.T) {
	for _, name := range []string{"family.glx", "FAMILY.GLX", "metadata.GLX"} {
		t.Run(name, func(t *testing.T) {
			archiveDir := filepath.Join(t.TempDir(), "archive")
			writeSkipTestFile(t, filepath.Join(archiveDir, name),
				"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")

			loaded, _, err := LoadArchive(archiveDir)
			require.NoError(t, err)
			require.Len(t, loaded.Persons, 1, "the loader reads the top-level file")

			err = safeWriteMultiFileArchive(archiveDir, loaded)

			require.ErrorIs(t, err, ErrTopLevelGLXFile)
			assert.FileExists(t, filepath.Join(archiveDir, name), "the archive is left untouched")
			assert.NoDirExists(t, filepath.Join(archiveDir, "persons"))
			assert.NoDirExists(t, archiveDir+".bak")
		})
	}
}

// metadata.glx is the one top-level file the writer owns, so an archive that
// has one is written normally.
func TestSafeWrite_AllowsTopLevelMetadataFile(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	archive := preserveTestArchive()
	archive.ImportMetadata = &glxlib.Metadata{SourceSystem: "test"}
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, archive))
	require.FileExists(t, filepath.Join(archiveDir, archiveMetadataFile))

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, archive))

	assert.FileExists(t, filepath.Join(archiveDir, archiveMetadataFile))
	assert.NoDirExists(t, archiveDir+".bak")
}
