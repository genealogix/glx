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

func TestMergePersonsCompletedRenameErrorRollsBack(t *testing.T) {
	for _, phase := range []string{"backup", "install", "rollback"} {
		t.Run(phase, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "archive")
			writeMergeFixture(t, dir)
			original, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			planned := maps.Clone(original)
			path := filepath.Join("persons", "person-keep.glx")
			planned[path] = append([]byte("# merged\n"), original[path]...)
			delete(planned, filepath.Join("persons", "person-drop.glx"))
			failed := false
			rollbackFailed := false
			err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
				isInstall := strings.Contains(from, string(filepath.Separator)+"new"+string(filepath.Separator))
				if phase == "rollback" && !failed && isInstall {
					failed = true

					return fs.ErrPermission
				}
				renameErr := robustRenameIn(root, from, to)
				if renameErr != nil {
					return renameErr
				}
				isBackup := !strings.HasPrefix(from, ".glx-merge-") && strings.Contains(to, string(filepath.Separator)+"original"+string(filepath.Separator))
				isRollback := strings.Contains(from, string(filepath.Separator)+"original"+string(filepath.Separator))
				if (!failed && ((phase == "backup" && isBackup) || (phase == "install" && isInstall))) || (phase == "rollback" && isRollback && !rollbackFailed) {
					failed = true
					rollbackFailed = true

					return fs.ErrPermission
				}

				return nil
			})
			require.ErrorIs(t, err, fs.ErrPermission)
			require.True(t, failed)
			if phase == "rollback" {
				require.True(t, rollbackFailed)
			}
			actual, err := collectGLXFilesFromDir(dir)
			require.NoError(t, err)
			assert.Equal(t, original, actual, "a completed rename reported as failure must not lose original files")
			assert.NoDirExists(t, dir+".bak")
			assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
			stages, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
			require.NoError(t, err)
			assert.Empty(t, stages)
		})
	}
}

func TestMergePersonsUncertainRenameRetainsRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	failed := false
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		renameErr := robustRenameIn(root, from, to)
		if renameErr != nil || failed {
			return renameErr
		}
		failed = true
		// Hide the moved inode under a different name, making the attempted rename's
		// source/destination pair insufficient to prove what happened.
		require.NoError(t, root.Rename(to, to+".uncertain"))
		require.NoError(t, root.WriteFile(to, []byte("replacement"), 0o600))

		return fs.ErrInvalid
	})
	require.ErrorIs(t, err, errMergeRenameUncertain)
	require.ErrorIs(t, err, fs.ErrInvalid)
	require.True(t, failed)
	require.FileExists(t, filepath.Join(dir, inPlaceJournalFile))
	require.ErrorIs(t, checkInPlaceJournal(dir), ErrInterruptedInPlaceWrite)
	backup, err := collectGLXFilesFromDir(dir + ".bak")
	require.NoError(t, err)
	assert.Equal(t, original[path], backup[path])
	stages, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
	require.NoError(t, err)
	assert.Len(t, stages, 1)
}

func TestMergePersonsMissingRenameSourceRetainsRecovery(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "archive")
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	path := filepath.Join("persons", "person-keep.glx")
	planned[path] = append([]byte("# merged\n"), original[path]...)
	dropPath := filepath.Join("persons", "person-drop.glx")
	delete(planned, dropPath)
	failed := false
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		renameErr := robustRenameIn(root, from, to)
		if renameErr != nil || failed {
			return renameErr
		}
		failed = true
		// An independent writer changes a file that rollback has not moved yet.
		// Restoring our tracked inode is insufficient to prove complete recovery.
		require.NoError(t, root.Remove(path))

		return nil
	})
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ErrorIs(t, err, errMergePreviewChanged)
	require.Contains(t, err.Error(), "recovery data retained")
	require.True(t, failed)
	require.FileExists(t, filepath.Join(dir, inPlaceJournalFile))
	backup, err := collectGLXFilesFromDir(dir + ".bak")
	require.NoError(t, err)
	assert.Equal(t, original[path], backup[path])
	assert.Equal(t, original[dropPath], backup[dropPath])
	stages, err := filepath.Glob(filepath.Join(dir, ".glx-merge-*"))
	require.NoError(t, err)
	assert.Len(t, stages, 1)
}
