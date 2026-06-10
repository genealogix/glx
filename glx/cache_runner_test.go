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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestBuildCacheAndStatus(t *testing.T) {
	dir := miniArchive(t, 3)

	io, out, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))
	assert.Contains(t, out.String(), "Built binary cache")
	assert.Contains(t, out.String(), "3 entities")
	require.FileExists(t, cachePath(dir))

	statusIO, statusOut, _ := TestIOStreams()
	require.NoError(t, statusCache(statusIO, dir))
	assert.Contains(t, statusOut.String(), "fresh")
	assert.Contains(t, statusOut.String(), "Persons:")
	assert.Contains(t, statusOut.String(), "3 total")
}

func TestBuildCacheNoForceIsNoOp(t *testing.T) {
	dir := miniArchive(t, 1)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))
	info, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	modBefore := info.ModTime()

	io2, out, _ := TestIOStreams()
	require.NoError(t, buildCache(io2, dir, false))
	assert.Contains(t, out.String(), "already fresh")

	info2, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	assert.Equal(t, modBefore, info2.ModTime(), "no-op build must not rewrite the cache")
}

func TestBuildCacheForceRebuilds(t *testing.T) {
	dir := miniArchive(t, 1)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))

	io2, out, _ := TestIOStreams()
	require.NoError(t, buildCache(io2, dir, true))
	assert.Contains(t, out.String(), "Built binary cache")
}

// TestBuildCacheQuietSilencesOutput verifies the cache commands honor the global
// --quiet flag: with Out bound to io.Discard, no status text is emitted but the
// cache is still built.
func TestBuildCacheQuietSilencesOutput(t *testing.T) {
	dir := miniArchive(t, 1)
	quiet := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: io.Discard}
	require.NoError(t, buildCache(quiet, dir, false))
	require.FileExists(t, cachePath(dir))
}

func TestCleanCache(t *testing.T) {
	dir := miniArchive(t, 1)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))
	require.FileExists(t, cachePath(dir))

	cleanIO, out, _ := TestIOStreams()
	require.NoError(t, cleanCache(cleanIO, dir))
	assert.Contains(t, out.String(), "Removed binary cache")
	assert.NoFileExists(t, cachePath(dir))
	// .glx directory removed when empty.
	_, err := os.Stat(cacheDir(dir))
	assert.True(t, os.IsNotExist(err), ".glx dir should be removed when empty")

	again, out2, _ := TestIOStreams()
	require.NoError(t, cleanCache(again, dir))
	assert.Contains(t, out2.String(), "No binary cache to remove")
}

func TestCleanCacheKeepsNonEmptyDir(t *testing.T) {
	dir := miniArchive(t, 1)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))
	// Drop a sibling file into .glx so the directory is not empty after cleanup.
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir(dir), "keep.txt"), []byte("x"), 0o644))

	cleanIO, _, _ := TestIOStreams()
	require.NoError(t, cleanCache(cleanIO, dir))
	assert.NoFileExists(t, cachePath(dir))
	assert.DirExists(t, cacheDir(dir), "non-empty .glx dir must be preserved")
}

func TestStatusNoCache(t *testing.T) {
	dir := miniArchive(t, 1)
	io, out, _ := TestIOStreams()
	require.NoError(t, statusCache(io, dir))
	assert.Contains(t, out.String(), "No binary cache")
}

func TestCacheCommandsRejectSingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons: {}\n"), 0o644))

	io, _, _ := TestIOStreams()
	require.ErrorIs(t, buildCache(io, file, false), ErrCacheNotDirectory)
	require.ErrorIs(t, cleanCache(io, file), ErrCacheNotDirectory)
	require.ErrorIs(t, statusCache(io, file), ErrCacheNotDirectory)
}

func TestStatusShowsGitCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := miniArchive(t, 2)
	runGitInit(t, dir)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))

	statusIO, out, _ := TestIOStreams()
	require.NoError(t, statusCache(statusIO, dir))
	assert.Contains(t, out.String(), "Git commit:")
	assert.Contains(t, out.String(), "clean work tree")
	// The short SHA is 7 hex chars; confirm one is present in the git line.
	assert.Regexp(t, `Git commit:\s+[0-9a-f]{7} `, out.String())
}

func TestWriteCacheRejectsNonDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons: {}\n"), 0o644))
	require.ErrorIs(t, writeCache(file, &glxlib.GLXFile{}, nil), ErrCacheNotDirectory)
}

func TestStatusReportsStale(t *testing.T) {
	dir := miniArchive(t, 1)
	io, _, _ := TestIOStreams()
	require.NoError(t, buildCache(io, dir, false))
	writePersonFile(t, dir, "person-new", "Newcomer")

	statusIO, out, _ := TestIOStreams()
	require.NoError(t, statusCache(statusIO, dir))
	assert.Contains(t, out.String(), "stale")
}

func TestCacheCobraRunners(t *testing.T) {
	dir := miniArchive(t, 1)
	_ = captureStdout(t, func() {
		require.NoError(t, runCacheBuild(nil, []string{dir}))
		require.NoError(t, runCacheStatus(nil, []string{dir}))
		require.NoError(t, runCacheClean(nil, []string{dir}))
	})

	assert.Equal(t, ".", cacheArgPath(nil))
	assert.Equal(t, "x", cacheArgPath([]string{"x"}))
}

func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, humanizeBytes(c.in), "humanizeBytes(%d)", c.in)
	}
}
