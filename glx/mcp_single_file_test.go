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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPSingleFileRejectsSymlinkStartup(t *testing.T) {
	for _, targetKind := range []string{"absolute", "relative", "sibling"} {
		t.Run(targetKind, func(t *testing.T) {
			dir := t.TempDir()
			selected := filepath.Join(dir, "selected", "archive.glx")
			require.NoError(t, os.MkdirAll(filepath.Dir(selected), 0o700))
			outside := filepath.Join(dir, "outside.glx")
			if targetKind == "sibling" {
				outside = filepath.Join(filepath.Dir(selected), "sibling.glx")
			}
			writeMCPWatchSource(t, outside, "OutsideSyntheticMarker")
			target := outside
			switch targetKind {
			case "relative":
				target = filepath.Join("..", "outside.glx")
			case "sibling":
				target = "sibling.glx"
			}
			if err := os.Symlink(target, selected); err != nil {
				t.Skipf("symlinks not available: %v", err)
			}
			for _, watch := range []bool{false, true} {
				t.Run(fmt.Sprintf("watch=%v", watch), func(t *testing.T) {
					var err error
					if watch {
						var backend *mcpWatchingBackend
						backend, err = newMCPWatchingBackend(t.Context(), selected)
						if backend != nil {
							t.Logf("watch startup loaded %v", backend.current.Load().archive.Persons["person-one"].Properties["name"])
						}
					} else {
						var backend *mcpArchiveBackend
						backend, err = loadMCPArchiveContext(t.Context(), selected)
						if backend != nil {
							t.Logf("default startup loaded %v", backend.archive.Persons["person-one"].Properties["name"])
						}
					}
					require.Error(t, err, "watch=%v must reject a symlink to external synthetic data", watch)
				})
			}
		})
	}
}

func TestMCPWatchSingleFileRejectsSymlinkAndRecovers(t *testing.T) {
	for _, targetKind := range []string{"absolute", "relative", "sibling"} {
		t.Run(targetKind, func(t *testing.T) { testMCPWatchSingleFileSymlinkRecovery(t, targetKind) })
	}
}

func testMCPWatchSingleFileSymlinkRecovery(t *testing.T, targetKind string) {
	t.Helper()
	dir := t.TempDir()
	selected := filepath.Join(dir, "selected", "archive.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(selected), 0o700))
	writeMCPWatchSource(t, selected, "OriginalSyntheticMarker")
	outside := filepath.Join(dir, "outside.glx")
	if targetKind == "sibling" {
		outside = filepath.Join(filepath.Dir(selected), "sibling.glx")
	}
	writeMCPWatchSource(t, outside, "OutsideSyntheticMarker")
	outsideBefore, err := os.ReadFile(outside)
	require.NoError(t, err)
	backend, err := newMCPWatchingBackend(t.Context(), selected)
	require.NoError(t, err)
	previous := backend.current.Load()
	link := filepath.Join(filepath.Dir(selected), ".replacement")
	target := outside
	switch targetKind {
	case "relative":
		target = filepath.Join("..", "outside.glx")
	case "sibling":
		target = "sibling.glx"
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	require.NoError(t, os.Rename(link, selected))
	firstChanged, firstErr := backend.poll(t.Context())
	secondChanged, secondErr := backend.poll(t.Context())
	t.Logf("after replacement polls: first changed=%v err=%v; second changed=%v err=%v; published=%v", firstChanged, firstErr, secondChanged, secondErr, backend.current.Load().archive.Persons["person-one"].Properties["name"])
	require.Error(t, firstErr)
	require.Error(t, secondErr)
	require.False(t, firstChanged)
	require.False(t, secondChanged)
	require.Same(t, previous, backend.current.Load())
	require.False(t, backend.hasPending)
	result, err := backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons"}`))
	require.NoError(t, err)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "OriginalSyntheticMarker")
	require.NotContains(t, string(encoded), "OutsideSyntheticMarker")
	// An ordinary editor-style atomic save must recover without reconnecting.
	writeMCPWatchSource(t, link, "RecoveredSyntheticMarker")
	require.NoError(t, os.Rename(link, selected))
	pollMCPWatch(t, backend, false)
	pollMCPWatch(t, backend, true)
	require.Equal(t, "RecoveredSyntheticMarker", backend.current.Load().archive.Persons["person-one"].Properties["name"])
	outsideAfter, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, outsideBefore, outsideAfter)
}

func TestMCPSingleFileRejectsReplacedDescriptor(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%v", symlink), func(t *testing.T) {
			dir := t.TempDir()
			selected := filepath.Join(dir, "archive.glx")
			writeMCPWatchSource(t, selected, "OriginalSyntheticMarker")
			root, err := os.OpenRoot(dir)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, root.Close()) })
			// Capture the same no-follow identity as production, then force
			// replacement in the validation/open gap without timing a race.
			expected, err := root.Lstat("archive.glx")
			require.NoError(t, err)
			replacement := filepath.Join(dir, ".replacement")
			if symlink {
				writeMCPWatchSource(t, filepath.Join(dir, "sibling.glx"), "OutsideSyntheticMarker")
				if err := os.Symlink("sibling.glx", replacement); err != nil {
					t.Skipf("symlinks not available: %v", err)
				}
			} else {
				writeMCPWatchSource(t, replacement, "OutsideSyntheticMarker")
			}
			require.NoError(t, os.Rename(replacement, selected))
			file, err := root.OpenFile("archive.glx", os.O_RDONLY|nonblockOpenFlag, 0)
			require.NoError(t, err)
			data, err := readMCPWatchRegularFile(file, expected)
			require.ErrorIs(t, err, errMCPWatchFileChanged)
			require.Nil(t, data, "a substituted descriptor must be rejected before reading its contents")
		})
	}
}

func TestMCPSingleFileAllowsSymlinkedParent(t *testing.T) {
	realDir := t.TempDir()
	writeMCPWatchSource(t, filepath.Join(realDir, "archive.glx"), "OriginalSyntheticMarker")
	alias := filepath.Join(t.TempDir(), "parent-alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	selected := filepath.Join(alias, "archive.glx")
	backend, err := loadMCPArchiveContext(t.Context(), selected)
	require.NoError(t, err)
	require.Equal(t, "OriginalSyntheticMarker", backend.archive.Persons["person-one"].Properties["name"])
	watcher, err := newMCPWatchingBackend(t.Context(), selected)
	require.NoError(t, err)
	staged := filepath.Join(realDir, ".replacement")
	writeMCPWatchSource(t, staged, "RecoveredSyntheticMarker")
	require.NoError(t, os.Rename(staged, filepath.Join(realDir, "archive.glx")))
	pollMCPWatch(t, watcher, false)
	pollMCPWatch(t, watcher, true)
	require.Equal(t, "RecoveredSyntheticMarker", watcher.current.Load().archive.Persons["person-one"].Properties["name"])
}

func TestMCPSingleFileOpenRejectsReplacementSymlink(t *testing.T) {
	dir := t.TempDir()
	selected := filepath.Join(dir, "archive.glx")
	writeMCPWatchSource(t, selected, "OriginalSyntheticMarker")
	// Model the caller's successful regular-file stat before substitution.
	before, err := os.Stat(selected)
	require.NoError(t, err)
	require.True(t, before.Mode().IsRegular())
	writeMCPWatchSource(t, filepath.Join(dir, "sibling.glx"), "OutsideSyntheticMarker")
	link := filepath.Join(dir, ".replacement")
	if err := os.Symlink("sibling.glx", link); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	require.NoError(t, os.Rename(link, selected))
	file, expected, openErr := openMCPWatchSingleFile(selected)
	if openErr == nil {
		// Windows opens the reparse point itself; Unix refuses the open.
		data, readErr := readMCPWatchRegularFile(file, expected)
		require.ErrorIs(t, readErr, errMCPWatchSourceType)
		require.Nil(t, data)
	} else {
		require.ErrorIs(t, openErr, errMCPWatchSourceType)
	}
}
