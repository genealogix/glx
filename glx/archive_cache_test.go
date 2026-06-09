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
	"bytes"
	"context"
	"encoding/gob"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// writePersonFile writes a single-person .glx file under dir/persons.
func writePersonFile(t *testing.T, dir, id, name string) string {
	t.Helper()
	personsDir := filepath.Join(dir, "persons")
	require.NoError(t, os.MkdirAll(personsDir, 0o755))
	path := filepath.Join(personsDir, id+".glx")
	content := "persons:\n  " + id + ":\n    properties:\n      primary_name: \"" + name + "\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

// miniArchive creates a small multi-file archive with n persons in a temp dir
// and returns the directory.
func miniArchive(t *testing.T, n int) string {
	t.Helper()
	dir := t.TempDir()
	for i := range n {
		writePersonFile(t, dir, "person-"+string(rune('a'+i)), "Person "+string(rune('A'+i)))
	}

	return dir
}

func TestCacheEnvMode(t *testing.T) {
	cases := map[string]cacheMode{
		"":         cacheModeReadOnly,
		"unset":    cacheModeReadOnly,
		"readonly": cacheModeReadOnly,
		"off":      cacheModeOff,
		"none":     cacheModeOff,
		"disabled": cacheModeOff,
		"NO":       cacheModeOff,
		"auto":     cacheModeAuto,
		"AUTO":     cacheModeAuto,
		"1":        cacheModeAuto,
		"on":       cacheModeAuto,
		"true":     cacheModeAuto,
		"write":    cacheModeAuto,
	}
	for val, want := range cases {
		t.Run(val, func(t *testing.T) {
			t.Setenv("GLX_CACHE", val)
			assert.Equal(t, want, cacheEnvMode())
		})
	}
}

func TestCountEntities(t *testing.T) {
	a := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p1": {}, "p2": {}},
		Events:  map[string]*glxlib.Event{"e1": {}},
		Studies: map[string]*glxlib.Study{"s1": {}},
	}
	c := countEntities(a)
	assert.Equal(t, 2, c.Persons)
	assert.Equal(t, 1, c.Events)
	assert.Equal(t, 1, c.Studies)
	assert.Equal(t, 4, c.Total())
}

// TestCacheRoundTripFidelity verifies a build → load round-trip reproduces an
// archive whose multi-file serialization is byte-identical to the original.
func TestCacheRoundTripFidelity(t *testing.T) {
	for _, ex := range []string{
		"../docs/examples/complete-family",
		"../docs/examples/assertion-workflow",
		"../docs/examples/basic-family",
		"../docs/examples/participant-assertions",
		"../docs/examples/temporal-properties",
		"../docs/examples/minimal",
	} {
		t.Run(filepath.Base(ex), func(t *testing.T) {
			orig, _, err := LoadArchiveWithOptions(ex, false)
			require.NoError(t, err)

			ser := glxlib.NewSerializer(&glxlib.SerializerOptions{Pretty: true, Indent: "  "})
			origFiles, err := ser.SerializeMultiFileToMap(orig)
			require.NoError(t, err)

			// In-memory gob round-trip (avoids writing into the example dirs).
			var buf bytes.Buffer
			require.NoError(t, gob.NewEncoder(&buf).Encode(orig))
			var got glxlib.GLXFile
			require.NoError(t, gob.NewDecoder(&buf).Decode(&got))

			gotFiles, err := ser.SerializeMultiFileToMap(&got)
			require.NoError(t, err)

			require.Len(t, gotFiles, len(origFiles), "file count")
			for name, want := range origFiles {
				assert.Truef(t, bytes.Equal(want, gotFiles[name]),
					"file %s differs after gob round-trip", name)
			}
		})
	}
}

// TestCacheRoundTripExoticProperties exercises property value types the YAML
// decoder can produce (nested map, list, timestamp, large uint, null) through a
// full writeCache → loadCache cycle.
func TestCacheRoundTripExoticProperties(t *testing.T) {
	dir := t.TempDir()
	personsDir := filepath.Join(dir, "persons")
	require.NoError(t, os.MkdirAll(personsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(personsDir, "person-x.glx"), []byte(`persons:
  person-x:
    properties:
      primary_name: "Exotic Tester"
      tags: [alpha, beta]
      nested:
        a: 1
        b: hello
      big: 18446744073709551615
      nothing: null
      external_ids:
        - type: familysearch
          value: ABC-123
`), 0o644))

	orig, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, orig, nil))

	got, header, err := loadCache(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, header.Counts.Persons)

	ser := glxlib.NewSerializer(&glxlib.SerializerOptions{Pretty: true, Indent: "  "})
	origFiles, err := ser.SerializeMultiFileToMap(orig)
	require.NoError(t, err)
	gotFiles, err := ser.SerializeMultiFileToMap(got)
	require.NoError(t, err)
	for name, want := range origFiles {
		assert.Truef(t, bytes.Equal(want, gotFiles[name]), "file %s differs", name)
	}
}

func TestCacheMagicAndVersionGuards(t *testing.T) {
	dir := miniArchive(t, 1)

	// No cache yet.
	_, err := readCacheHeader(dir)
	require.ErrorIs(t, err, ErrCacheNotFound)

	// Garbage file is rejected by the magic check.
	require.NoError(t, os.MkdirAll(cacheDir(dir), 0o755))
	require.NoError(t, os.WriteFile(cachePath(dir), []byte("not a glx cache at all"), 0o644))
	_, err = readCacheHeader(dir)
	require.ErrorIs(t, err, ErrNotCacheFile)

	// A truncated-but-magic file is also rejected (header decode fails).
	require.NoError(t, os.WriteFile(cachePath(dir), append(cacheMagic[:], 0x01, 0x02), 0o644))
	_, err = readCacheHeader(dir)
	require.ErrorIs(t, err, ErrNotCacheFile)
}

func TestCacheVersionMismatch(t *testing.T) {
	dir := miniArchive(t, 1)
	arch, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)

	// Hand-write a cache with a bumped format version.
	header := CacheHeader{Version: cacheFormatVersion + 1, GLXVersion: version}
	var buf bytes.Buffer
	buf.Write(cacheMagic[:])
	enc := gob.NewEncoder(&buf)
	require.NoError(t, enc.Encode(&header))
	require.NoError(t, enc.Encode(arch))
	require.NoError(t, os.MkdirAll(cacheDir(dir), 0o755))
	require.NoError(t, os.WriteFile(cachePath(dir), buf.Bytes(), 0o644))

	_, err = readCacheHeader(dir)
	require.ErrorIs(t, err, ErrCacheVersion)

	// And the transparent loader simply ignores it.
	_, _, ok := tryLoadFreshCache(dir)
	assert.False(t, ok)
}

func TestCacheStalenessFSFingerprint(t *testing.T) {
	dir := miniArchive(t, 2)
	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	header, err := readCacheHeader(dir)
	require.NoError(t, err)
	assert.True(t, cacheIsFresh(dir, header), "freshly built cache should be fresh")

	// Adding a file changes the file set → stale.
	writePersonFile(t, dir, "person-z", "Zelda")
	assert.False(t, cacheIsFresh(dir, header), "added file should invalidate cache")

	// Rebuild → fresh again.
	arch2, dups2, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch2, dups2))
	header2, err := readCacheHeader(dir)
	require.NoError(t, err)
	assert.True(t, cacheIsFresh(dir, header2))

	// Touching a file's mtime invalidates too.
	future := time.Now().Add(2 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(dir, "persons", "person-a.glx"), future, future))
	assert.False(t, cacheIsFresh(dir, header2), "mtime change should invalidate cache")
}

// TestLoadArchiveCachedUsesCache proves the cache (not the YAML) supplies the
// result when the fingerprint matches: it overwrites a person file with
// different content of identical length and restores the original mtime, so the
// fingerprint is unchanged while the YAML on disk differs from the cache.
func TestLoadArchiveCachedUsesCache(t *testing.T) {
	t.Setenv("GLX_CACHE", "")
	dir := t.TempDir()
	path := writePersonFile(t, dir, "person-a", "Alice") // 5-char name
	info, err := os.Stat(path)
	require.NoError(t, err)
	origMtime := info.ModTime()

	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	// Overwrite with a same-length name and restore mtime+size → same fingerprint.
	require.NoError(t, os.WriteFile(path,
		[]byte("persons:\n  person-a:\n    properties:\n      primary_name: \"Bobby\"\n"), 0o644))
	require.NoError(t, os.Chtimes(path, origMtime, origMtime))

	cached, _, err := LoadArchiveCached(dir)
	require.NoError(t, err)
	require.NotNil(t, cached.Persons["person-a"])
	assert.Equal(t, "Alice", cached.Persons["person-a"].Properties["primary_name"],
		"fresh cache should win over changed-but-same-fingerprint YAML")

	// With caching off, the on-disk YAML wins.
	t.Setenv("GLX_CACHE", "off")
	fresh, _, err := LoadArchiveCached(dir)
	require.NoError(t, err)
	assert.Equal(t, "Bobby", fresh.Persons["person-a"].Properties["primary_name"])
}

// TestLoadArchiveCachedStaleIgnored proves a stale cache is ignored and the
// current YAML is returned.
func TestLoadArchiveCachedStaleIgnored(t *testing.T) {
	t.Setenv("GLX_CACHE", "")
	dir := miniArchive(t, 1)
	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	// Add a person → cache is stale.
	writePersonFile(t, dir, "person-new", "Newcomer")

	got, _, err := LoadArchiveCached(dir)
	require.NoError(t, err)
	assert.Len(t, got.Persons, 2, "stale cache must be ignored; YAML has 2 persons")
}

func TestLoadArchiveCachedAutoWriteAndOff(t *testing.T) {
	dir := miniArchive(t, 1)

	// Default mode: a miss does NOT write a cache.
	t.Setenv("GLX_CACHE", "")
	_, _, err := LoadArchiveCached(dir)
	require.NoError(t, err)
	_, statErr := os.Stat(cachePath(dir))
	assert.True(t, os.IsNotExist(statErr), "read-only mode must not write a cache")

	// Auto mode: a miss writes the cache.
	t.Setenv("GLX_CACHE", "auto")
	_, _, err = LoadArchiveCached(dir)
	require.NoError(t, err)
	_, err = os.Stat(cachePath(dir))
	require.NoError(t, err, "auto mode should build the cache on miss")

	// Off mode: never reads the cache even if present. Stale-ify by adding a file;
	// off mode reads YAML regardless.
	writePersonFile(t, dir, "person-extra", "Extra")
	t.Setenv("GLX_CACHE", "off")
	got, _, err := LoadArchiveCached(dir)
	require.NoError(t, err)
	assert.Len(t, got.Persons, 2)
}

func TestCacheDuplicatesPreserved(t *testing.T) {
	dir := miniArchive(t, 1)
	dups := []string{"duplicate entity ID: person-a"}
	arch, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	_, gotDups, ok := tryLoadFreshCache(dir)
	require.True(t, ok)
	assert.Equal(t, dups, gotDups)
}

// TestCacheGitMetadataAndFreshness verifies that the cache header records the
// git HEAD commit and clean state (read in-process via go-git, against a repo
// created by the real git binary), and that staleness is decided solely by the
// filesystem fingerprint — the recorded git metadata never makes a changed
// archive look fresh. The git binary is used only for fixture setup, so the
// test self-skips when it is unavailable.
func TestCacheGitMetadataAndFreshness(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := miniArchive(t, 2)
	runGitInit(t, dir)

	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	header, err := readCacheHeader(dir)
	require.NoError(t, err)
	require.NotEmpty(t, header.GitCommitSHA, "HEAD SHA should be recorded via go-git")
	require.True(t, header.GitClean, "clean tree should be recorded via go-git")
	assert.True(t, cacheIsFresh(dir, header), "matching fingerprint should be fresh")

	// The git metadata is informational only: a fingerprint mismatch must
	// invalidate the cache even though the recorded clean HEAD still matches.
	stale := *header
	stale.FSFingerprint = strings.Repeat("0", 64)
	assert.False(t, cacheIsFresh(dir, &stale),
		"fingerprint is the sole authority; a recorded clean git HEAD must not override it")

	// Modify a tracked file without committing → fingerprint changes → stale.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-a.glx"),
		[]byte("persons:\n  person-a:\n    properties:\n      primary_name: \"Changed Name\"\n"), 0o644))
	assert.False(t, cacheIsFresh(dir, header), "dirty tracked file should invalidate cache")
}

// TestCacheGitHeadSHAUnbornHead checks that a repository with no commits (an
// unborn HEAD) yields an empty SHA rather than an error or panic.
func TestCacheGitHeadSHAUnbornHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := miniArchive(t, 1)
	cmd := exec.CommandContext(context.Background(), "git", "-C", dir, "init")
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git init: %s", out)

	assert.Empty(t, gitHeadSHA(dir), "unborn HEAD should yield an empty SHA")
}

// TestCacheGitOpTimeoutFallback shrinks the go-git deadline so the in-process
// lookups give up, confirming they fall back to the safe "git unavailable"
// result (empty SHA / ok=false) rather than blocking the command.
func TestCacheGitOpTimeoutFallback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := miniArchive(t, 1)
	runGitInit(t, dir)

	orig := gitOpTimeout
	gitOpTimeout = time.Nanosecond
	t.Cleanup(func() { gitOpTimeout = orig })

	assert.Empty(t, gitHeadSHA(dir), "timed-out HEAD lookup should yield an empty SHA")
	clean, ok := gitWorkingTreeClean(dir)
	assert.False(t, ok, "timed-out status should report not-ok")
	assert.False(t, clean)
}

// TestCacheRejectsOversizedFile shrinks the decode ceiling below any real cache
// so the size guard fires, confirming an oversized (potentially hostile) cache
// is reported unusable and the transparent loader falls back rather than
// gob-decoding it.
func TestCacheRejectsOversizedFile(t *testing.T) {
	dir := miniArchive(t, 2)
	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	orig := maxCacheFileBytes
	maxCacheFileBytes = 8 // below any real cache (the magic prefix alone is 8 bytes)
	t.Cleanup(func() { maxCacheFileBytes = orig })

	_, err = readCacheHeader(dir)
	require.ErrorIs(t, err, ErrCacheTooLarge, "oversized cache must be rejected before decoding")

	_, _, ok := tryLoadFreshCache(dir)
	assert.False(t, ok, "oversized cache must not be treated as a hit")
}

// TestCacheGitOpsOutsideRepo checks that the git helpers report "unavailable"
// (empty SHA / ok=false) when the archive is not inside a git repository, so
// callers fall back to the filesystem fingerprint. No git binary is needed —
// go-git simply finds no repository.
func TestCacheGitOpsOutsideRepo(t *testing.T) {
	dir := miniArchive(t, 1) // a plain directory, not a git repo

	assert.Empty(t, gitHeadSHA(dir), "no git repo -> empty SHA")
	clean, ok := gitWorkingTreeClean(dir)
	assert.False(t, ok, "no git repo -> not ok")
	assert.False(t, clean)
}

// TestCacheIsFreshRejectsForeignGLXVersion confirms a cache written by a
// different glx version is treated as stale up front (its gob layout may not
// match the current structs), independent of the filesystem fingerprint.
func TestCacheIsFreshRejectsForeignGLXVersion(t *testing.T) {
	dir := miniArchive(t, 2)
	arch, dups, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	require.NoError(t, writeCache(dir, arch, dups))

	header, err := readCacheHeader(dir)
	require.NoError(t, err)
	require.True(t, cacheIsFresh(dir, header), "freshly built cache should be fresh")

	// An unwalkable archive root (computeFSFingerprint error) is treated as
	// stale, never fresh.
	assert.False(t, cacheIsFresh(filepath.Join(t.TempDir(), "does-not-exist"), header),
		"an unwalkable archive root must not be reported fresh")

	header.GLXVersion += "-other"
	assert.False(t, cacheIsFresh(dir, header),
		"a cache written by a different glx version must be treated as stale")
}

func runGitInit(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"add", "-A"},
		{"commit", "-m", "init"},
	} {
		cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		require.NoErrorf(t, err, "git %v: %s", args, out)
	}
}
