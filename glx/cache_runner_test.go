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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCacheAndStatus(t *testing.T) {
	dir := miniArchive(t, 3)

	var buildErr error
	out := captureStdout(t, func() { buildErr = buildCache(dir, false) })
	require.NoError(t, buildErr)
	assert.Contains(t, out, "Built binary cache")
	assert.Contains(t, out, "3 entities")
	require.FileExists(t, cachePath(dir))

	var statusErr error
	statusOut := captureStdout(t, func() { statusErr = statusCache(dir) })
	require.NoError(t, statusErr)
	assert.Contains(t, statusOut, "fresh")
	assert.Contains(t, statusOut, "Persons:")
	assert.Contains(t, statusOut, "3 total")
}

func TestBuildCacheNoForceIsNoOp(t *testing.T) {
	dir := miniArchive(t, 1)
	require.NoError(t, buildCache(dir, false))
	info, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	modBefore := info.ModTime()

	out := captureStdout(t, func() { require.NoError(t, buildCache(dir, false)) })
	assert.Contains(t, out, "already fresh")

	info2, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	assert.Equal(t, modBefore, info2.ModTime(), "no-op build must not rewrite the cache")
}

func TestBuildCacheForceRebuilds(t *testing.T) {
	dir := miniArchive(t, 1)
	require.NoError(t, buildCache(dir, false))

	out := captureStdout(t, func() { require.NoError(t, buildCache(dir, true)) })
	assert.Contains(t, out, "Built binary cache")
}

func TestCleanCache(t *testing.T) {
	dir := miniArchive(t, 1)
	require.NoError(t, buildCache(dir, false))
	require.FileExists(t, cachePath(dir))

	out := captureStdout(t, func() { require.NoError(t, cleanCache(dir)) })
	assert.Contains(t, out, "Removed binary cache")
	assert.NoFileExists(t, cachePath(dir))
	// .glx directory removed when empty.
	_, err := os.Stat(cacheDir(dir))
	assert.True(t, os.IsNotExist(err), ".glx dir should be removed when empty")

	out2 := captureStdout(t, func() { require.NoError(t, cleanCache(dir)) })
	assert.Contains(t, out2, "No binary cache to remove")
}

func TestCleanCacheKeepsNonEmptyDir(t *testing.T) {
	dir := miniArchive(t, 1)
	require.NoError(t, buildCache(dir, false))
	// Drop a sibling file into .glx so the directory is not empty after cleanup.
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir(dir), "keep.txt"), []byte("x"), 0o644))

	require.NoError(t, cleanCache(dir))
	assert.NoFileExists(t, cachePath(dir))
	assert.DirExists(t, cacheDir(dir), "non-empty .glx dir must be preserved")
}

func TestStatusNoCache(t *testing.T) {
	dir := miniArchive(t, 1)
	out := captureStdout(t, func() { require.NoError(t, statusCache(dir)) })
	assert.Contains(t, out, "No binary cache")
}

func TestCacheCommandsRejectSingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(file, []byte("persons: {}\n"), 0o644))

	require.ErrorIs(t, buildCache(file, false), ErrCacheNotDirectory)
	require.ErrorIs(t, cleanCache(file), ErrCacheNotDirectory)
	require.ErrorIs(t, statusCache(file), ErrCacheNotDirectory)
}

func TestStatusReportsStale(t *testing.T) {
	dir := miniArchive(t, 1)
	require.NoError(t, buildCache(dir, false))
	writePersonFile(t, dir, "person-new", "Newcomer")

	out := captureStdout(t, func() { require.NoError(t, statusCache(dir)) })
	assert.Contains(t, out, "stale")
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
