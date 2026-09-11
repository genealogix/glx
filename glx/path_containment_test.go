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

func TestRelWithin(t *testing.T) {
	base := t.TempDir()

	rel, ok := relWithin(filepath.Join(base, "media", "a.jpg"), base)
	assert.True(t, ok)
	assert.Equal(t, filepath.Join("media", "a.jpg"), rel)

	// A sibling entry whose name merely starts with ".." is contained.
	rel, ok = relWithin(filepath.Join(base, "..hidden.glx"), base)
	assert.True(t, ok)
	assert.Equal(t, "..hidden.glx", rel)

	_, ok = relWithin(base, base)
	assert.False(t, ok, "the base directory itself is not contained")

	_, ok = relWithin(filepath.Join(base, "..", "outside.jpg"), base)
	assert.False(t, ok)

	_, ok = relWithin(filepath.Dir(base), base)
	assert.False(t, ok)
}

func TestOpenWithin_RejectsLexicalEscape(t *testing.T) {
	base := t.TempDir()

	_, err := openWithin(base, filepath.Join(base, "..", "outside.txt"))
	require.ErrorIs(t, err, ErrPathEscapesDir)

	_, err = readFileWithin(base, filepath.Dir(base))
	require.ErrorIs(t, err, ErrPathEscapesDir)
}

func TestOpenWithin_ReadsContainedFile(t *testing.T) {
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "media"), 0o755))
	target := filepath.Join(base, "media", "a.txt")
	require.NoError(t, os.WriteFile(target, []byte("hello"), 0o644))

	data, err := readFileWithin(base, target)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	f, err := openWithin(base, target)
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestOpenWithin_MissingFileIsNotExist(t *testing.T) {
	base := t.TempDir()

	_, err := openWithin(base, filepath.Join(base, "missing.txt"))
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "missing file must surface as not-exist so callers can fall back: %v", err)
}

func TestOpenWithin_RejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(secret, []byte("secret"), 0o644))

	base := t.TempDir()
	link := filepath.Join(base, "link.txt")
	require.NoError(t, os.Symlink(secret, link))

	// Lexically the link is inside base, so the old guard accepted it.
	assert.True(t, isPathWithin(link, base))

	_, err := readFileWithin(base, link)
	require.Error(t, err, "symlink whose target is outside the base directory must be refused")
	assert.False(t, os.IsNotExist(err), "escape must not be reported as a missing file: %v", err)

	_, err = openWithin(base, link)
	require.Error(t, err)
}

func TestOpenWithin_RejectsSymlinkedDirEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644))

	base := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(base, "media")))

	_, err := readFileWithin(base, filepath.Join(base, "media", "secret.txt"))
	require.Error(t, err, "a symlinked directory component escaping base must be refused")
}

func TestOpenWithin_FollowsSymlinkInsideBase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "files", "real.txt"), []byte("real"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join("files", "real.txt"), filepath.Join(base, "alias.txt")))

	data, err := readFileWithin(base, filepath.Join(base, "alias.txt"))
	require.NoError(t, err)
	assert.Equal(t, "real", string(data))
}

// TestOpenWithin_RejectsAbsoluteSymlinkEvenInsideBase pins a documented
// restriction: os.Root refuses every absolute symlink target, including one
// that resolves back inside the base, so a link created with an absolute
// path must be rewritten as a relative one. This is deliberate — resolving
// the target and re-checking containment would reopen the TOCTOU window the
// helper exists to close.
func TestOpenWithin_RejectsAbsoluteSymlinkEvenInsideBase(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	base := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(base, "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "files", "real.txt"), []byte("real"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(base, "files", "real.txt"), filepath.Join(base, "alias.txt")))

	_, err := readFileWithin(base, filepath.Join(base, "alias.txt"))
	require.Error(t, err, "absolute symlink targets are refused by os.Root even when they resolve inside the base")
}

func TestOpenWithin_SymlinkedBaseDirIsAllowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	realDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "a.txt"), []byte("a"), 0o644))
	base := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.Symlink(realDir, base))

	data, err := readFileWithin(base, filepath.Join(base, "a.txt"))
	require.NoError(t, err, "an archive directory that is itself a symlink must keep working")
	assert.Equal(t, "a", string(data))
}

func TestWalkGLXFiles_ReportsReadErrorPerFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is privilege-gated on Windows")
	}
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.glx"), []byte("leaked"), 0o644))

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.glx"), []byte("ok"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.glx"), filepath.Join(dir, "bad.glx")))

	seen := map[string]string{}
	var readErrs int
	err := walkGLXFiles(dir, func(relPath string, data []byte, readErr error) error {
		if readErr != nil {
			readErrs++
			assert.Nil(t, data)

			return nil //nolint:nilerr // the walk must continue past a contained read failure
		}
		seen[relPath] = string(data)

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, 1, readErrs)
	assert.Equal(t, map[string]string{"ok.glx": "ok"}, seen)
}

func TestWalkGLXFiles_SkipsDotDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ok.glx"), []byte("ok"), 0o644))
	for _, hidden := range []string{".git", ".glx", filepath.Join(".claude", "worktrees", "copy", "persons")} {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, hidden), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, hidden, "ok.glx"), []byte("dup"), 0o644))
	}

	seen := map[string]string{}
	err := walkGLXFiles(dir, func(relPath string, data []byte, readErr error) error {
		require.NoError(t, readErr)
		seen[relPath] = string(data)

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]string{"ok.glx": "ok"}, seen)
}

func TestWalkGLXFiles_DotPrefixedRootIsWalked(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".archive")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "a.glx"), []byte("a"), 0o644))

	var seen []string
	err := walkGLXFiles(dir, func(relPath string, _ []byte, readErr error) error {
		require.NoError(t, readErr)
		seen = append(seen, relPath)

		return nil
	})
	require.NoError(t, err)

	assert.Equal(t, []string{filepath.Join("persons", "a.glx")}, seen)
}
