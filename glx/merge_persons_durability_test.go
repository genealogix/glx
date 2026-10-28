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
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePersonsRetainsRecoveryWhenInstallationSyncFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	delete(planned, filepath.Join("persons", "person-drop.glx"))
	installed := false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		err := robustRenameIn(root, from, to)
		if err == nil && strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
			installed = true
		}

		return err
	}, func(file *os.File) error {
		if installed {
			return fs.ErrInvalid
		}

		return nil
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.FileExists(t, filepath.Join(dir+".bak", swapInProgressMarker))
	backup, err := os.ReadFile(filepath.Join(dir+".bak", path))
	require.NoError(t, err)
	assert.Equal(t, original[path], backup)
	transactions, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
	require.NoError(t, err)
	require.Len(t, transactions, 1)
	moved, err := os.ReadFile(filepath.Join(transactions[0], "original", path))
	require.NoError(t, err)
	assert.Equal(t, original[path], moved)
}

func TestMergePersonsSyncsStagedFilesBeforeMovingOriginals(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	moved := false
	stagedSynced := false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		moved = true

		return robustRenameIn(root, from, to)
	}, func(file *os.File) error {
		if strings.Contains(file.Name(), string(filepath.Separator)+"new"+string(filepath.Separator)) {
			assert.False(t, moved)
			stagedSynced = true

			return fs.ErrInvalid
		}

		return nil
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.True(t, stagedSynced)
	assert.False(t, moved)
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
}

func TestMergePersonsSyncsRecoveryAndRenamesBeforeCleanup(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	synced := make(map[string]bool)
	installed := false
	archiveSyncedAfterInstall := false
	backupRemovalSynced := false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		if !installed {
			assert.True(t, synced[filepath.Clean(dir+".bak")], "backup entries must persist before originals move")
			assert.True(t, synced[filepath.Dir(dir)], "the sibling backup name must persist")
		}
		err := robustRenameIn(root, from, to)
		if err == nil && strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
			installed = true
		}

		return err
	}, func(file *os.File) error {
		name := ""
		info, statErr := file.Stat()
		require.NoError(t, statErr)
		for _, expected := range []string{dir + ".bak", filepath.Dir(dir), dir} {
			candidate, statErr := os.Stat(expected)
			if statErr == nil && os.SameFile(info, candidate) {
				name = expected
				synced[name] = true
			}
		}
		if installed && name == filepath.Dir(dir) {
			assert.NoDirExists(t, dir+".bak")
			backupRemovalSynced = true
		} else if installed {
			assert.FileExists(t, filepath.Join(dir+".bak", swapInProgressMarker))
			if name == dir {
				archiveSyncedAfterInstall = true
			}
		}

		return syncMergeHandle(file)
	})
	require.NoError(t, err)
	assert.True(t, archiveSyncedAfterInstall)
	assert.True(t, backupRemovalSynced)
	assert.NoDirExists(t, dir+".bak")
	transactions, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
	require.NoError(t, err)
	assert.Empty(t, transactions)
}

func TestMergePersonsRetainsBackupWhenRollbackSyncFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	installFailed := false
	err = installMergeFilesWithSync(dir, original, planned, func(root *os.Root, from, to string) error {
		if strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator)) {
			installFailed = true

			return fs.ErrPermission
		}

		return robustRenameIn(root, from, to)
	}, func(file *os.File) error {
		if installFailed {
			return fs.ErrInvalid
		}

		return syncMergeHandle(file)
	})
	require.ErrorIs(t, err, fs.ErrPermission)
	assert.FileExists(t, filepath.Join(dir+".bak", swapInProgressMarker))
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, original, actual)
}

func TestMergePersonsReportsBackupRemovalSyncFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	observed := false
	err = installMergeFilesWithSync(dir, original, planned, robustRenameIn, func(file *os.File) error {
		info, statErr := file.Stat()
		require.NoError(t, statErr)
		parentInfo, statErr := os.Stat(filepath.Dir(dir))
		require.NoError(t, statErr)
		_, backupErr := os.Stat(dir + ".bak")
		if os.SameFile(info, parentInfo) && os.IsNotExist(backupErr) {
			observed = true

			return fs.ErrInvalid
		}

		return nil
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	assert.True(t, observed)
	assert.Contains(t, err.Error(), "syncing completed merge backup removal")
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	assert.Equal(t, planned, actual, "installed data remains intact when cleanup durability is uncertain")
}

func TestMergePersonsBackupSyncFailureDoesNotBlockRetry(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	failed := false
	err = installMergeFilesWithSync(dir, original, planned, robustRenameIn, func(file *os.File) error {
		if strings.Contains(file.Name(), ".bak") {
			failed = true

			return fs.ErrInvalid
		}

		return nil
	})
	require.ErrorIs(t, err, fs.ErrInvalid)
	require.True(t, failed)
	require.NoDirExists(t, dir+".bak")
	actual, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	require.Equal(t, original, actual)
	require.NoError(t, installMergeFilesWithSync(dir, original, planned, robustRenameIn, func(*os.File) error { return nil }))
}
