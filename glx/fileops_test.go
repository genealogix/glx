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

func TestAtomicWriteFile_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.glx")

	require.NoError(t, atomicWriteFile(path, []byte("content"), 0o644))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))
}

func TestAtomicWriteFile_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.glx")

	// Write initial content
	require.NoError(t, atomicWriteFile(path, []byte("original"), 0o644))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))

	// Overwrite atomically
	require.NoError(t, atomicWriteFile(path, []byte("updated"), 0o644))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "updated", string(data))
}

func TestAtomicWriteFile_Permissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support POSIX file permission granularity")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "perms.glx")

	require.NoError(t, atomicWriteFile(path, []byte("data"), 0o644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm())
}

func TestAtomicWriteFile_NoTempFileLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.glx")

	require.NoError(t, atomicWriteFile(path, []byte("data"), 0o644))

	// Only the target file should exist, no .glx-tmp-* leftovers
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "clean.glx", entries[0].Name())
}

func TestAtomicWriteFile_InvalidDir(t *testing.T) {
	err := atomicWriteFile("/nonexistent/dir/file.glx", []byte("data"), 0o644)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "creating temp file")
}
