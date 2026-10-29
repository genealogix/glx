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
	"encoding/json"
	"io/fs"
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

func readMergeJournal(t *testing.T, dir string) []inPlaceJournalEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, inPlaceJournalFile))
	require.NoError(t, err)
	var entries []inPlaceJournalEntry
	require.NoError(t, json.Unmarshal(data, &entries))

	return entries
}

func TestMergeJournalRefusesExistingGeneralWriterJournal(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	data, err := json.Marshal([]inPlaceJournalEntry{{Path: filepath.ToSlash(path), Existed: true, Data: original[path]}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, inPlaceJournalFile), data, 0o600))
	moved := false
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		moved = true

		return robustRenameIn(root, from, to)
	})
	require.ErrorIs(t, err, ErrInterruptedInPlaceWrite)
	assert.False(t, moved)
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
	journal, err := os.ReadFile(filepath.Join(dir, inPlaceJournalFile))
	require.NoError(t, err)
	assert.Equal(t, data, journal)
	assert.NoDirExists(t, dir+".bak")
}

func TestMergeJournalDurableBeforeMutationAndRemovedAfterCleanup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	keep := filepath.Join("persons", "person-keep.glx")
	drop := filepath.Join("persons", "person-drop.glx")
	planned[keep] = append([]byte("# merged\n"), original[keep]...)
	delete(planned, drop)
	expected := []inPlaceJournalEntry{
		{Path: filepath.ToSlash(drop), Existed: true, Data: original[drop]},
		{Path: filepath.ToSlash(keep), Existed: true, Data: original[keep]},
	}
	journalSynced, archiveSynced := false, false
	moved, cleanupSynced, removalSynced := false, false, false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		assert.True(t, journalSynced, "the complete journal must be synced before originals move")
		assert.True(t, archiveSynced, "the journal's directory entry must be synced before originals move")
		if !moved {
			assert.Equal(t, expected, readMergeJournal(t, dir))
			info, statErr := os.Stat(filepath.Join(dir, inPlaceJournalFile))
			require.NoError(t, statErr)
			if runtime.GOOS != "windows" {
				assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
			}
		}
		moved = true

		return robustRenameIn(root, from, to)
	}, func(file *os.File) error {
		info, statErr := file.Stat()
		require.NoError(t, statErr)
		journalInfo, journalErr := os.Stat(filepath.Join(dir, inPlaceJournalFile))
		if journalErr == nil && os.SameFile(info, journalInfo) {
			journalSynced = true
		}
		archiveInfo, statErr := os.Stat(dir)
		require.NoError(t, statErr)
		if journalSynced && os.SameFile(info, archiveInfo) {
			archiveSynced = true
			transactions, globErr := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
			require.NoError(t, globErr)
			if moved && len(transactions) == 0 && journalErr == nil {
				assert.NoDirExists(t, dir+".bak")
				cleanupSynced = true
			}
			if os.IsNotExist(journalErr) {
				assert.True(t, cleanupSynced, "recovery cleanup must be durable before the journal is removed")
				removalSynced = true
			}
		}

		return syncMergeHandle(file)
	})
	require.NoError(t, err)
	assert.True(t, moved)
	assert.True(t, removalSynced)
	assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
	assert.NoDirExists(t, dir+".bak")
}

func TestMergeJournalRemovedAfterDurableRollback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	installFailed, removalSynced := false, false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		if strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
			installFailed = true

			return fs.ErrPermission
		}

		return robustRenameIn(root, from, to)
	}, func(file *os.File) error {
		if _, statErr := os.Stat(filepath.Join(dir, inPlaceJournalFile)); installFailed && os.IsNotExist(statErr) {
			actual, readErr := collectGLXFilesFromDir(dir)
			require.NoError(t, readErr)
			assert.Equal(t, original, actual)
			assert.NoDirExists(t, dir+".bak")
			transactions, globErr := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
			require.NoError(t, globErr)
			assert.Empty(t, transactions)
			removalSynced = true
		}

		return syncMergeHandle(file)
	})
	require.ErrorIs(t, err, fs.ErrPermission)
	assert.True(t, removalSynced)
	assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
}

func TestMergeJournalPreparationFailureLeavesOriginalsAndAllowsRetry(t *testing.T) {
	for _, failure := range []string{"file_sync", "directory_sync"} {
		t.Run(failure, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "archive")
			writeMergeFixture(t, dir)
			original, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			planned := maps.Clone(original)
			path := filepath.Join("persons", "person-keep.glx")
			planned[path] = append([]byte("# merged\n"), original[path]...)
			moved, failed := false, false
			err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
				moved = true

				return robustRenameIn(root, from, to)
			}, func(file *os.File) error {
				info, statErr := file.Stat()
				require.NoError(t, statErr)
				journalInfo, journalErr := os.Stat(filepath.Join(dir, inPlaceJournalFile))
				if !failed && journalErr == nil {
					archiveInfo, statErr := os.Stat(dir)
					require.NoError(t, statErr)
					if (failure == "file_sync" && os.SameFile(info, journalInfo)) || (failure == "directory_sync" && os.SameFile(info, archiveInfo)) {
						failed = true

						return fs.ErrInvalid
					}
				}

				return syncMergeHandle(file)
			})
			require.ErrorIs(t, err, fs.ErrInvalid)
			assert.True(t, failed)
			assert.False(t, moved)
			assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
			assert.NoDirExists(t, dir+".bak")
			actual, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			assert.Equal(t, original, actual)
			require.NoError(t, installMergeFiles(dir, original, planned))
		})
	}
}

func TestMergeJournalExclusiveCreationPreservesLateCollision(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	otherJournal := []byte("another writer's recovery journal\n")
	injected, moved := false, false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		moved = true

		return robustRenameIn(root, from, to)
	}, func(file *os.File) error {
		info, statErr := file.Stat()
		require.NoError(t, statErr)
		parentInfo, statErr := os.Stat(filepath.Dir(dir))
		require.NoError(t, statErr)
		if !injected && os.SameFile(info, parentInfo) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, inPlaceJournalFile), otherJournal, 0o600))
			injected = true
		}

		return syncMergeHandle(file)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "creating merge recovery journal")
	assert.True(t, injected)
	assert.False(t, moved)
	data, err := os.ReadFile(filepath.Join(dir, inPlaceJournalFile))
	require.NoError(t, err)
	assert.Equal(t, otherJournal, data)
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
}

func TestMergeJournalCleanupPreservesReplacement(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer func() { require.NoError(t, root.Close()) }()
	finish, err := prepareMergeJournal(root, []string{"person.glx"}, map[string][]byte{"person.glx": []byte("original")}, syncMergeHandle)
	require.NoError(t, err)
	require.NoError(t, root.Rename(inPlaceJournalFile, ".previous-journal"))
	replacement := []byte("replacement journal")
	require.NoError(t, os.WriteFile(filepath.Join(dir, inPlaceJournalFile), replacement, 0o600))
	require.ErrorIs(t, finish(), errMergePreviewChanged)
	data, err := os.ReadFile(filepath.Join(dir, inPlaceJournalFile))
	require.NoError(t, err)
	assert.Equal(t, replacement, data)
}

func TestMergeJournalRemovalSyncFailureRestoresRecoverySignal(t *testing.T) {
	for _, collision := range []bool{false, true} {
		t.Run(map[bool]string{false: "restore", true: "replacement_collision"}[collision], func(t *testing.T) {
			dir := t.TempDir()
			root, err := os.OpenRoot(dir)
			require.NoError(t, err)
			defer func() { require.NoError(t, root.Close()) }()
			original := map[string][]byte{"person.glx": []byte("original")}
			failed := false
			replacement := []byte("replacement journal")
			finish, err := prepareMergeJournal(root, []string{"person.glx"}, original, func(file *os.File) error {
				if _, statErr := root.Lstat(inPlaceJournalFile); !failed && os.IsNotExist(statErr) {
					failed = true
					if collision {
						require.NoError(t, os.WriteFile(filepath.Join(dir, inPlaceJournalFile), replacement, 0o600))
					}

					return fs.ErrInvalid
				}

				return syncMergeHandle(file)
			})
			require.NoError(t, err)
			require.ErrorIs(t, finish(), fs.ErrInvalid)
			assert.True(t, failed)
			if collision {
				data, err := root.ReadFile(inPlaceJournalFile)
				require.NoError(t, err)
				assert.Equal(t, replacement, data)
			} else {
				assert.Equal(t, []inPlaceJournalEntry{{Path: "person.glx", Existed: true, Data: original["person.glx"]}}, readMergeJournal(t, dir))
			}
			require.ErrorIs(t, checkInPlaceJournal(dir), ErrInterruptedInPlaceWrite)
		})
	}
}

func TestMergeJournalRetainedWhenRollbackRenameFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		if strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
			return fs.ErrPermission
		}
		if strings.Contains(from, string(filepath.Separator)+"original"+string(filepath.Separator)) {
			return fs.ErrInvalid
		}

		return robustRenameIn(root, from, to)
	})
	require.ErrorIs(t, err, fs.ErrPermission)
	assert.Equal(t, []inPlaceJournalEntry{{Path: filepath.ToSlash(path), Existed: true, Data: original[path]}}, readMergeJournal(t, dir))
	assert.FileExists(t, filepath.Join(dir+".bak", mergeBackupMarker))
	err = writeArchiveInPlace(dir, original, &glxlib.GLXFile{}, rewriteChanged)
	require.ErrorIs(t, err, ErrInterruptedInPlaceWrite)
}

func TestMergeJournalRetainedWhenTransactionCleanupSyncFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	failed := false
	err = installMergeFilesWithSync(dir, original, planned, robustRenameIn, func(file *os.File) error {
		info, statErr := file.Stat()
		require.NoError(t, statErr)
		archiveInfo, statErr := os.Stat(dir)
		require.NoError(t, statErr)
		transactions, globErr := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
		require.NoError(t, globErr)
		if os.SameFile(info, archiveInfo) && len(transactions) == 0 {
			failed = true

			return fs.ErrInvalid
		}

		return syncMergeHandle(file)
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.True(t, failed)
	assert.Contains(t, err.Error(), "syncing completed merge transaction removal")
	assert.Equal(t, []inPlaceJournalEntry{{Path: filepath.ToSlash(path), Existed: true, Data: original[path]}}, readMergeJournal(t, dir))
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, planned, actual)
	err = writeArchiveInPlace(dir, actual, &glxlib.GLXFile{}, rewriteChanged)
	require.ErrorIs(t, err, ErrInterruptedInPlaceWrite)
}

func TestMergeJournalPrunesEmptyDeletedFileParents(t *testing.T) {
	for _, foreign := range []string{"", "note.txt", "empty-foreign-directory"} {
		name := foreign
		if name == "" {
			name = "empty"
		}
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "archive")
			path := filepath.Join("assertions", "nested", "claim.glx")
			writeSkipTestFile(t, filepath.Join(dir, path), "assertions: {}\n")
			if foreign == "note.txt" {
				writeSkipTestFile(t, filepath.Join(dir, "assertions", "nested", foreign), "foreign data")
			} else if foreign != "" {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "assertions", "nested", foreign), 0o755))
			}
			original, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			require.NoError(t, installMergeFiles(dir, original, map[string][]byte{}))
			if foreign == "" {
				assert.NoDirExists(t, filepath.Join(dir, "assertions"))
			} else {
				assert.DirExists(t, filepath.Join(dir, "assertions", "nested"))
				_, err := os.Stat(filepath.Join(dir, "assertions", "nested", foreign))
				require.NoError(t, err)
			}
			assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
		})
	}
}

func TestMergeJournalEmptyParentCleanupPreservesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	for _, component := range []string{"assertions", filepath.Join("assertions", "nested")} {
		t.Run(component, func(t *testing.T) {
			dir := t.TempDir()
			outside := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(outside, "nested"), 0o755))
			link := filepath.Join(dir, component)
			require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
			require.NoError(t, os.Symlink(outside, link))
			root, err := os.OpenRoot(dir)
			require.NoError(t, err)
			defer func() { require.NoError(t, root.Close()) }()
			path := filepath.Join("assertions", "nested", "claim.glx")
			require.NoError(t, removeEmptyMergeParents(root, []string{path}, map[string][]byte{}, syncMergeHandle))
			info, err := os.Lstat(link)
			require.NoError(t, err)
			assert.NotZero(t, info.Mode()&os.ModeSymlink)
			assert.DirExists(t, filepath.Join(outside, "nested"))
		})
	}
}

func TestMergeJournalRetainedWhenEmptyParentCleanupSyncFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	path := filepath.Join("assertions", "nested", "claim.glx")
	writeSkipTestFile(t, filepath.Join(dir, path), "assertions: {}\n")
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	failed := false
	err = installMergeFilesWithSync(dir, original, map[string][]byte{}, robustRenameIn, func(file *os.File) error {
		if _, statErr := os.Lstat(filepath.Join(dir, "assertions")); os.IsNotExist(statErr) {
			failed = true

			return fs.ErrInvalid
		}

		return syncMergeHandle(file)
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.True(t, failed)
	assert.Contains(t, err.Error(), "cleaning empty merge directories")
	assert.Equal(t, []inPlaceJournalEntry{{Path: filepath.ToSlash(path), Existed: true, Data: original[path]}}, readMergeJournal(t, dir))
	assert.FileExists(t, filepath.Join(dir+".bak", mergeBackupMarker))
	transactions, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	assert.FileExists(t, filepath.Join(transactions[0], "original", path))
	err = writeArchiveInPlace(dir, original, &glxlib.GLXFile{}, rewriteChanged)
	require.ErrorIs(t, err, ErrInterruptedInPlaceWrite)
}
