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

func TestCacheReadRejectsEscapingSymlinks(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "cache file"
		if directory {
			name = "cache directory"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("GLX_CACHE", "auto")
			root := t.TempDir()
			outside := t.TempDir()
			publicPath := writePersonFile(t, root, "person-a", "Public")
			secretPath := writePersonFile(t, outside, "person-a", "Secret")
			info, err := os.Stat(publicPath)
			require.NoError(t, err)
			require.NoError(t, os.Chtimes(secretPath, info.ModTime(), info.ModTime()))
			foreign, _, err := LoadArchiveWithOptions(outside, false)
			require.NoError(t, err)
			require.NoError(t, writeCache(outside, foreign, nil))
			header, err := readCacheHeader(outside)
			require.NoError(t, err)
			require.True(t, cacheIsFresh(root, header), "identical metadata must reproduce the freshness bypass")
			before, err := os.ReadFile(cachePath(outside))
			require.NoError(t, err)
			if directory {
				createCacheTestSymlink(t, filepath.Join(outside, cacheDirName), filepath.Join(root, cacheDirName))
			} else {
				require.NoError(t, os.Mkdir(filepath.Join(root, cacheDirName), 0o755))
				createCacheTestSymlink(t, cachePath(outside), cachePath(root))
			}
			file, _, _, err := openCache(root)
			if file != nil {
				require.NoError(t, file.Close())
			}
			require.Error(t, err, "cache reader must remain inside the selected archive")
			backend, err := loadMCPArchive(root)
			require.NoError(t, err)
			assert.Contains(t, backend.archive.Persons, "person-a")
			assert.Equal(t, "Public", backend.archive.Persons["person-a"].Properties["primary_name"])
			after, err := os.ReadFile(cachePath(root))
			require.NoError(t, err)
			assert.Equal(t, before, after, "MCP fallback must not rewrite the outside cache")
		})
	}
}

func TestCacheReadAllowsContainedSymlinks(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "cache file"
		if directory {
			name = "cache directory"
		}
		t.Run(name, func(t *testing.T) {
			root := miniArchive(t, 1)
			archive, _, err := LoadArchiveWithOptions(root, false)
			require.NoError(t, err)
			require.NoError(t, writeCache(root, archive, nil))
			if directory {
				require.NoError(t, os.Rename(filepath.Join(root, cacheDirName), filepath.Join(root, ".saved-cache")))
				createCacheTestSymlink(t, ".saved-cache", filepath.Join(root, cacheDirName))
			} else {
				require.NoError(t, os.Rename(cachePath(root), filepath.Join(root, cacheDirName, "saved.bin")))
				createCacheTestSymlink(t, "saved.bin", cachePath(root))
			}
			got, _, ok := tryLoadFreshCache(root)
			require.True(t, ok)
			assert.Equal(t, archive.Persons, got.Persons)
		})
	}
}

func TestCacheReadMissingPaths(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{filepath.Join(root, "missing"), root} {
		_, _, _, err := openCache(path)
		require.ErrorIs(t, err, ErrCacheNotFound)
	}
	require.NoError(t, os.Mkdir(filepath.Join(root, cacheDirName), 0o755))
	_, _, _, err := openCache(root)
	require.ErrorIs(t, err, ErrCacheNotFound)
	regular := filepath.Join(root, "regular")
	require.NoError(t, os.WriteFile(regular, nil, 0o600))
	_, _, _, err = openCache(regular)
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrCacheNotFound)
}

func TestCacheReadContainsSymlinkReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("concurrent replacement requires POSIX atomic rename over an existing symlink")
	}

	for _, directory := range []bool{false, true} {
		name := "cache file"
		if directory {
			name = "cache directory"
		}
		t.Run(name, func(t *testing.T) {
			root := miniArchive(t, 1)
			outside := miniArchive(t, 2)
			for _, dir := range []string{root, outside} {
				archive, _, err := LoadArchiveWithOptions(dir, false)
				require.NoError(t, err)
				require.NoError(t, writeCache(dir, archive, nil))
			}
			link := cachePath(root)
			insideTarget := "saved.bin"
			outsideTarget := cachePath(outside)
			if directory {
				link = filepath.Join(root, cacheDirName)
				insideTarget = ".saved-cache"
				outsideTarget = filepath.Join(outside, cacheDirName)
			}
			require.NoError(t, os.Rename(link, filepath.Join(filepath.Dir(link), insideTarget)))
			createCacheTestSymlink(t, insideTarget, link)
			done := make(chan error, 1)
			go func() {
				for i := range 200 {
					target := insideTarget
					if i%2 == 0 {
						target = outsideTarget
					}
					if err := os.Symlink(target, link+".next"); err != nil {
						done <- err

						return
					}
					if err := os.Rename(link+".next", link); err != nil {
						done <- err

						return
					}
				}
				done <- nil
			}()
			for range 200 {
				file, _, header, err := openCache(root)
				if err == nil {
					assert.Equal(t, 1, header.Counts.Persons, "outside cache must never be opened")
					require.NoError(t, file.Close())
				}
			}
			require.NoError(t, <-done)
		})
	}
}

// Windows symlink creation may require Developer Mode or elevated privileges.
// Keep containment assertions enabled on Windows hosts that support symlinks.
func createCacheTestSymlink(t *testing.T, target, link string) {
	t.Helper()
	err := os.Symlink(target, link)
	if err != nil && runtime.GOOS == "windows" {
		t.Skipf("symlink creation unavailable on this Windows host: %v", err)
	}
	require.NoError(t, err)
}
