//go:build unix

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
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMergePersonsPreservesModeUnderRestrictiveUmask(t *testing.T) {
	const childEnv = "GLX_MERGE_UMASK_TEST_CHILD"
	if os.Getenv(childEnv) != "1" {
		// Umask is process-wide. Keep the change away from other test writers.
		command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMergePersonsPreservesModeUnderRestrictiveUmask$")
		command.Env = append(os.Environ(), childEnv+"=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))

		return
	}
	dir := t.TempDir()
	writeMergeFixture(t, dir)
	interruptedDir := t.TempDir()
	writeMergeFixture(t, interruptedDir)
	modes := map[string]os.FileMode{
		filepath.Join("persons", "person-keep.glx"):  0o644,
		filepath.Join("events", "event-baptism.glx"): 0o640,
	}
	for _, archiveDir := range []string{dir, interruptedDir} {
		for path, mode := range modes {
			require.NoError(t, os.Chmod(filepath.Join(archiveDir, path), mode))
		}
	}
	original, err := collectGLXFilesFromDir(interruptedDir)
	require.NoError(t, err)
	planned := maps.Clone(original)
	delete(planned, filepath.Join("persons", "person-drop.glx"))
	for path := range modes {
		planned[path] = append([]byte{}, original[path]...)
		planned[path] = append(planned[path], []byte("\n# merged\n")...)
	}
	previous := syscall.Umask(0o077)
	defer syscall.Umask(previous)
	streams, _, _ := TestIOStreams()
	require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, dir,
		"person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
	for path, mode := range modes {
		info, err := os.Stat(filepath.Join(dir, path))
		require.NoError(t, err)
		assert.Equal(t, mode, info.Mode().Perm(), path)
	}
	// A failed install and rollback must retain recovery copies with the
	// original modes as well as original bytes.
	err = installMergeFilesWithRename(interruptedDir, original, planned, func(root *os.Root, from, to string) error {
		if strings.HasPrefix(from, ".glx-merge-") {
			return os.ErrPermission
		}

		return robustRenameIn(root, from, to)
	})
	require.ErrorIs(t, err, os.ErrPermission)
	for path, mode := range modes {
		backupPath := filepath.Join(interruptedDir+".bak", path)
		info, err := os.Stat(backupPath)
		require.NoError(t, err)
		assert.Equal(t, mode, info.Mode().Perm(), backupPath)
		data, err := os.ReadFile(backupPath)
		require.NoError(t, err)
		assert.Equal(t, original[path], data)
	}
}

func TestMergePersonsBackupRemainsPrivateUnderPermissiveUmask(t *testing.T) {
	const childEnv = "GLX_MERGE_PRIVATE_BACKUP_TEST_CHILD"
	if os.Getenv(childEnv) != "1" {
		command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestMergePersonsBackupRemainsPrivateUnderPermissiveUmask$")
		command.Env = append(os.Environ(), childEnv+"=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))

		return
	}
	previous := syscall.Umask(0)
	defer syscall.Umask(previous)
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o755))
	dir := filepath.Join(parent, "private-archive")
	require.NoError(t, os.Mkdir(dir, 0o700))
	writeMergeFixture(t, dir)
	original, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	path := filepath.Join("persons", "person-keep.glx")
	require.NoError(t, os.Chmod(filepath.Join(dir, path), 0o644))
	planned := maps.Clone(original)
	planned[path] = append([]byte("# merged\n"), original[path]...)
	var activeMode os.FileMode
	err = installMergeFilesWithRename(dir, original, planned, func(root *os.Root, from, to string) error {
		info, statErr := os.Stat(dir + ".bak")
		require.NoError(t, statErr)
		activeMode = info.Mode().Perm()
		if strings.HasPrefix(from, ".glx-merge-") {
			return os.ErrPermission // Retain the backup after installation and rollback fail.
		}

		return robustRenameIn(root, from, to)
	})
	require.ErrorIs(t, err, os.ErrPermission)
	assert.Equal(t, os.FileMode(0o700), activeMode)
	backup, err := os.Stat(dir + ".bak")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), backup.Mode().Perm())
	file, err := os.Stat(filepath.Join(dir+".bak", path))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), file.Mode().Perm())
	data, err := os.ReadFile(filepath.Join(dir+".bak", path))
	require.NoError(t, err)
	assert.Equal(t, original[path], data)
}
