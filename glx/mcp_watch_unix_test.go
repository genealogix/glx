//go:build unix

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
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMCPWatchRejectsNamedPipes(t *testing.T) {
	dir := t.TempDir()
	writeMCPWatchSource(t, filepath.Join(dir, "person.glx"), "First")
	backend, err := newMCPWatchingBackend(t.Context(), dir)
	require.NoError(t, err)
	previous := backend.current.Load()
	pipe := filepath.Join(dir, "named-pipe.glx")
	require.NoError(t, syscall.Mkfifo(pipe, 0o600))
	for _, path := range []string{pipe, filepath.Join(dir, "pipe-link.glx")} {
		if path != pipe {
			require.NoError(t, os.Symlink(filepath.Base(pipe), path))
		}
		done := make(chan error, 1)
		go func() {
			// This direct read also covers the path after a regular-file
			// stat races with replacement by a FIFO.
			_, readErr := readMCPWatchSingleFile(path)
			done <- readErr
		}()
		select {
		case readErr := <-done:
			require.ErrorIs(t, readErr, errMCPWatchSourceType)
		case <-time.After(time.Second):
			t.Fatal("watcher blocked while opening a named pipe")
		}
	}
	changed, err := backend.poll(t.Context())
	require.ErrorIs(t, err, errMCPWatchSourceType)
	require.False(t, changed)
	require.Same(t, previous, backend.current.Load())
}

func TestMCPSingleFileSearchOnlyParent(t *testing.T) {
	dir := t.TempDir()
	selected := filepath.Join(dir, "archive.glx")
	writeMCPWatchSource(t, selected, "OriginalSyntheticMarker")
	require.NoError(t, os.Chmod(dir, 0o111))
	t.Cleanup(func() { require.NoError(t, os.Chmod(dir, 0o700)) })
	// Some privileged test environments bypass directory permissions.
	if root, err := os.OpenRoot(dir); err == nil {
		require.NoError(t, root.Close())
		t.Skip("directory read permissions are not enforced for this user")
	}
	backend, err := loadMCPArchiveContext(t.Context(), selected)
	require.NoError(t, err)
	require.Equal(t, "OriginalSyntheticMarker", backend.archive.Persons["person-one"].Properties["name"])
	watcher, err := newMCPWatchingBackend(t.Context(), selected)
	require.NoError(t, err)
	writeMCPWatchSource(t, selected, "RecoveredSyntheticMarker")
	pollMCPWatch(t, watcher, false)
	pollMCPWatch(t, watcher, true)
	require.Equal(t, "RecoveredSyntheticMarker", watcher.current.Load().archive.Persons["person-one"].Properties["name"])
}
