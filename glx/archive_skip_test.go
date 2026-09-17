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
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// collectRelPaths returns the archive's file set as a sorted slice of relative
// paths, which is what every enumerator ultimately agrees or disagrees about.
func collectRelPaths(t *testing.T, dir string) []string {
	t.Helper()

	files, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)

	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, filepath.ToSlash(rel))
	}
	sort.Strings(paths)

	return paths
}

func writeSkipTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestCollectGLXFiles_SkipsDotDirectories(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".git", "objects", "a.glx"), "persons: {}")

	assert.Equal(t, []string{"persons/a.glx"}, collectRelPaths(t, dir))
}

// Dot-prefixed files are editor and filesystem droppings: AppleDouble sidecars
// (._foo.glx), Emacs lock symlinks (.#foo.glx). Reading them failed the whole
// archive load.
func TestCollectGLXFiles_SkipsDotPrefixedFiles(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, "persons", "._a.glx"), "\x00\x05\x16\x07not yaml")
	writeSkipTestFile(t, filepath.Join(dir, "persons", ".glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".backup.glx"), "persons: {}")

	assert.Equal(t, []string{"persons/a.glx"}, collectRelPaths(t, dir))
}

// An archive whose own root is dot-named is still a valid archive: the skip
// applies to entries found inside the root, never to the root itself.
func TestCollectGLXFiles_DotPrefixedRootIsWalked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".archive")
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")

	assert.Equal(t, []string{"persons/a.glx"}, collectRelPaths(t, dir))
}

// The extension match is case-insensitive: on a case-insensitive filesystem
// PERSON.GLX is the same file the archive refers to as person.glx, and
// treating it as invisible silently drops entities from every command.
func TestCollectGLXFiles_UppercaseExtension(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "A.GLX"), "persons: {}")

	assert.Equal(t, []string{"persons/A.GLX"}, collectRelPaths(t, dir))
}

// The skip has to govern reads, not just traversal. A symlink at a visible
// path pointing into a skipped directory would otherwise read back exactly the
// duplicate entities the skip exists to exclude (#1212).
func TestCollectGLXFiles_SkipsSymlinkIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "a.glx"), "persons: {}")

	require.NoError(t, os.Symlink(
		filepath.Join("..", ".worktrees", "copy", "persons", "a.glx"),
		filepath.Join(dir, "persons", "dup.glx"),
	))

	assert.Equal(t, []string{"persons/a.glx"}, collectRelPaths(t, dir))
}

// A chain of symlinks ending inside a dot directory is the same case; the
// target is resolved all the way before it is judged.
func TestCollectGLXFiles_SkipsChainedSymlinkIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "a.glx"), "persons: {}")

	require.NoError(t, os.Symlink(
		filepath.Join("..", ".worktrees", "copy", "persons", "a.glx"),
		filepath.Join(dir, "persons", "hop.glx"),
	))
	require.NoError(t, os.Symlink("hop.glx", filepath.Join(dir, "persons", "dup.glx")))

	assert.Equal(t, []string{"persons/a.glx"}, collectRelPaths(t, dir))
}

// A symlink between two ordinary archive paths is still followed: the rule is
// about dot-prefixed entries, not about symlinks.
func TestCollectGLXFiles_FollowsOrdinarySymlink(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	require.NoError(t, os.Symlink("a.glx", filepath.Join(dir, "persons", "b.glx")))

	assert.Equal(t, []string{"persons/a.glx", "persons/b.glx"}, collectRelPaths(t, dir))
}

// The cache fingerprint must cover exactly the loader's file set. When it
// covered more, an edit inside a worktree copy invalidated a cache whose
// contents could not have changed.
func TestComputeFSFingerprint_IgnoresDotDirectoryEdits(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "a.glx"), "persons: {}")
	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "a.glx"), "persons: {}")

	before, err := computeFSFingerprint(dir)
	require.NoError(t, err)

	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "a.glx"), "persons: {changed: true}")
	writeSkipTestFile(t, filepath.Join(dir, ".worktrees", "copy", "persons", "b.glx"), "persons: {}")

	after, err := computeFSFingerprint(dir)
	require.NoError(t, err)

	assert.Equal(t, before, after)
}

// filepath.WalkDir lstats its root, so an unresolved symlinked root walked
// zero files and hashed the empty string — a fingerprint that matched forever,
// leaving the cache "fresh" no matter how the archive changed.
func TestComputeFSFingerprint_SymlinkedRoot(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	base := t.TempDir()
	target := filepath.Join(base, "archive")
	link := filepath.Join(base, "link")
	writeSkipTestFile(t, filepath.Join(target, "persons", "a.glx"), "persons: {}")
	require.NoError(t, os.Symlink(target, link))

	viaTarget, err := computeFSFingerprint(target)
	require.NoError(t, err)
	viaLink, err := computeFSFingerprint(link)
	require.NoError(t, err)

	assert.Equal(t, viaTarget, viaLink)

	writeSkipTestFile(t, filepath.Join(target, "persons", "b.glx"), "persons: {}")

	afterLink, err := computeFSFingerprint(link)
	require.NoError(t, err)
	assert.NotEqual(t, viaLink, afterLink, "a change through the real path must invalidate the symlinked root's cache")
}

func TestPathHasDotComponent(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"persons/a.glx", false},
		{"./persons/a.glx", false},
		{".worktrees/copy/persons/a.glx", true},
		{"persons/.drafts/a.glx", true},
		{"persons/._a.glx", true},
		{"a.glx", false},
	} {
		assert.Equal(t, tc.want, pathHasDotComponent(tc.path), tc.path)
	}
}
