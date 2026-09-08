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
	"strings"
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

// openTestRoot opens dir as an os.Root for placeholder tests and closes it
// when the test ends.
func openTestRoot(t *testing.T, dir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	return root
}

func TestResolveSymlinkPlaceholder_ResolvesTargetContent(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.glx")
	require.NoError(t, os.WriteFile(targetPath, []byte("resolved"), 0o644))

	got := resolveSymlinkPlaceholder(openTestRoot(t, dir), "link.glx", []byte("./target.glx"))

	assert.Equal(t, []byte("resolved"), got)
}

func TestResolveSymlinkPlaceholder_ResolvesNestedRelativeTarget(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "shared"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared", "p.glx"), []byte("shared"), 0o644))

	got := resolveSymlinkPlaceholder(openTestRoot(t, dir), "persons/link.glx", []byte("../shared/p.glx"))

	assert.Equal(t, []byte("shared"), got)
}

func TestResolveSymlinkPlaceholder_RejectsTargetOutsideRoot(t *testing.T) {
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.glx"), []byte("secret"), 0o644))
	dir := filepath.Join(outside, "archive")
	require.NoError(t, os.MkdirAll(dir, 0o755))

	content := "../secret.glx"
	got := resolveSymlinkPlaceholder(openTestRoot(t, dir), "link.glx", []byte(content))

	assert.Equal(t, []byte(content), got, "placeholder escaping the archive root must not be followed")
}

func TestResolveSymlinkPlaceholder_RejectsLongPlaceholderContent(t *testing.T) {
	dir := t.TempDir()
	longContent := "a/" + strings.Repeat("b", maxSymlinkPlaceholderLength)

	got := resolveSymlinkPlaceholder(openTestRoot(t, dir), "link.glx", []byte(longContent))

	assert.Equal(t, []byte(longContent), got)
}

func TestResolveSymlinkPlaceholder_RejectsInvalidCharacters(t *testing.T) {
	dir := t.TempDir()
	content := "nested/\nfile.glx"

	got := resolveSymlinkPlaceholder(openTestRoot(t, dir), "link.glx", []byte(content))

	assert.Equal(t, []byte(content), got)
}

func TestCollectGLXFilesFromDir_ReadsNestedFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "p1.glx"), []byte("persons: {}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("not glx"), 0o644))

	files, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)

	assert.Len(t, files, 1)
	assert.Equal(t, []byte("persons: {}"), files[filepath.Join("persons", "p1.glx")])
}

func TestCollectGLXFilesFromDir_MissingDir(t *testing.T) {
	_, err := collectGLXFilesFromDir(filepath.Join(t.TempDir(), "nope"))
	assert.Error(t, err)
}

func TestCollectGLXFilesFromDir_RejectsSymlinkEscapingArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.glx")
	require.NoError(t, os.WriteFile(secret, []byte("persons: {leaked: {}}"), 0o644))

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.Symlink(secret, filepath.Join(dir, "persons", "p1.glx")))

	files, err := collectGLXFilesFromDir(dir)

	require.Error(t, err, "an archive symlink pointing outside the archive must fail containment")
	assert.Contains(t, err.Error(), "p1.glx")
	for _, data := range files {
		assert.NotContains(t, string(data), "leaked", "target contents must not be read")
	}
}

func TestCollectGLXFilesFromDir_FollowsSymlinkInsideArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "shared"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "shared", "p.glx"), []byte("persons: {}"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join("..", "shared", "p.glx"), filepath.Join(dir, "persons", "link.glx")))

	files, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)

	assert.Equal(t, []byte("persons: {}"), files[filepath.Join("persons", "link.glx")])
	assert.Equal(t, []byte("persons: {}"), files[filepath.Join("shared", "p.glx")])
}
