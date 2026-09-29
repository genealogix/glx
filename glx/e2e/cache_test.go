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

package e2e

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cacheFile = ".glx/cache.bin"

// The full lifecycle from inside the archive root with default arguments:
// build writes only the cache file, status reports it, a second build is a
// no-op, --force rebuilds, and clean removes exactly what build wrote.
func TestCache_LifecycleFromInsideArchiveRoot(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	none := runGLX(t, archive, "cache", "status")
	require.Equal(t, 0, none.exitCode, none.stderr)
	assert.Contains(t, none.stdout, "No binary cache")

	build := runGLX(t, archive, "cache", "build")
	require.Equal(t, 0, build.exitCode, build.stderr)
	// Printed with the OS separator (.glx\cache.bin on Windows).
	assert.Contains(t, build.stdout, "Built binary cache: "+filepath.FromSlash(cacheFile))
	assert.Contains(t, build.stdout, "19 entities")
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Equal(t, []string{cacheFile}, diff.created, "build must write only the cache")
	assert.Empty(t, diff.removed)

	status := runGLX(t, archive, "cache", "status")
	require.Equal(t, 0, status.exitCode, status.stderr)
	assert.Contains(t, status.stdout, "Status:      fresh")
	assert.Contains(t, status.stdout, "Persons:       4")

	again := runGLX(t, archive, "cache", "build")
	require.Equal(t, 0, again.exitCode, again.stderr)
	assert.Contains(t, again.stdout, "Cache already fresh")

	forced := runGLX(t, archive, "cache", "build", "--force")
	require.Equal(t, 0, forced.exitCode, forced.stderr)
	assert.Contains(t, forced.stdout, "Built binary cache")

	clean := runGLX(t, archive, "cache", "clean")
	require.Equal(t, 0, clean.exitCode, clean.stderr)
	assert.Contains(t, clean.stdout, "Removed binary cache")
	assert.NoFileExists(t, filepath.Join(archive, cacheFile))
	assertTreeUnchanged(t, before, archive)

	cleanAgain := runGLX(t, archive, "cache", "clean")
	require.Equal(t, 0, cleanAgain.exitCode, cleanAgain.stderr)
	assert.Contains(t, cleanAgain.stdout, "No binary cache to remove")
}

func TestCache_ArchivePathFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")
	outside := t.TempDir()

	build := runGLX(t, outside, "cache", "build", archive)

	require.Equal(t, 0, build.exitCode, build.stderr)
	assert.FileExists(t, filepath.Join(archive, cacheFile))
	assert.NoFileExists(t, filepath.Join(outside, cacheFile), "the cache belongs to the archive, not the cwd")
}

func TestCache_EditMakesCacheStale(t *testing.T) {
	archive := copyExample(t, "basic-family")
	require.Equal(t, 0, runGLX(t, archive, "cache", "build").exitCode)
	add := runGLX(t, archive, "add", "person", "--given", "New", "--surname", "Person")
	require.Equal(t, 0, add.exitCode, add.stderr)

	status := runGLX(t, archive, "cache", "status")

	require.Equal(t, 0, status.exitCode, status.stderr)
	assert.Contains(t, status.stdout, "stale")
}

// A stale cache must be ignored, not served: the read has to see the person
// added after the build.
func TestCache_StaleCacheIsNotServed(t *testing.T) {
	archive := copyExample(t, "basic-family")
	require.Equal(t, 0, runGLX(t, archive, "cache", "build").exitCode)
	require.Equal(t, 0, runGLX(t, archive, "add", "person", "--given", "New", "--surname", "Person").exitCode)

	res := runGLX(t, archive, "query", "persons")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "person-new-person")
	assert.Contains(t, res.stdout, "5 person(s) found")
}

func TestCache_AutoModeBuildsOnMiss(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLXWithEnv(t, []string{"GLX_CACHE=auto"}, archive, "summary", "person-robert-thompson")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Robert Thompson")
	assert.FileExists(t, filepath.Join(archive, cacheFile))
}

func TestCache_ReadsMatchWithAndWithoutCache(t *testing.T) {
	archive := copyExample(t, "basic-family")
	uncached := runGLX(t, archive, "stats")
	require.Equal(t, 0, uncached.exitCode, uncached.stderr)
	require.Equal(t, 0, runGLX(t, archive, "cache", "build").exitCode)

	cached := runGLX(t, archive, "stats")

	require.Equal(t, 0, cached.exitCode, cached.stderr)
	assert.Equal(t, uncached.stdout, cached.stdout)
}

func TestCache_Errors(t *testing.T) {
	archive := copyExample(t, "single-file")

	assertExitWithStderr(t, runGLX(t, archive, "cache", "build", "archive.glx"), "only supported for multi-file")
	assertExitWithStderr(t, runGLX(t, archive, "cache", "status", "does-not-exist"), "cannot access path")
	assertExitWithStderr(t, runGLX(t, archive, "cache", "build", "a", "b"), "accepts at most 1 arg(s), received 2")
	assertExitWithStderr(t, runGLX(t, archive, "cache", "bogus"), `unknown command "bogus" for "glx cache"`)
}
