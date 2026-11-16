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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func writeMCPWatchSource(t *testing.T, path, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf("persons:\n  person-one:\n    properties:\n      name: %s\n", name)), 0o600))
}

func pollMCPWatch(t *testing.T, backend *mcpWatchingBackend, wantChanged bool) {
	t.Helper()
	changed, err := backend.poll(t.Context())
	require.NoError(t, err)
	require.Equal(t, wantChanged, changed)
}

func TestMCPWatchSnapshotSwapAndValidationReset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	writeMCPWatchSource(t, path, "Before")
	originalInfo, err := os.Stat(path)
	require.NoError(t, err)
	backend, err := newMCPWatchingBackend(t.Context(), path)
	require.NoError(t, err)
	previous := backend.current.Load()
	_, err = backend.CallTool(t.Context(), "validate", json.RawMessage(`{"report":true}`))
	require.NoError(t, err)
	require.NotNil(t, previous.validation)
	// Query output marshals entity pointers later, after the backend call. Keep
	// that result across the swap to verify it still encodes the old snapshot.
	oldResult, err := backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons"}`))
	require.NoError(t, err)
	writeMCPWatchSource(t, path, "Change") // Same size and restored mtime.
	require.NoError(t, os.Chtimes(path, originalInfo.ModTime(), originalInfo.ModTime()))
	pollMCPWatch(t, backend, false)
	require.Same(t, previous, backend.current.Load())
	pollMCPWatch(t, backend, true)
	current := backend.current.Load()
	require.NotSame(t, previous, current)
	require.Nil(t, current.validation, "a new snapshot must not reuse stale validation")
	encoded, err := json.Marshal(oldResult)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "Before")
	require.NotContains(t, string(encoded), "Change")
	newResult, err := backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons"}`))
	require.NoError(t, err)
	encoded, err = json.Marshal(newResult)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "Change")
	pollMCPWatch(t, backend, false)
}

func TestMCPWatchDebounceFailureAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	writeMCPWatchSource(t, path, "First")
	backend, err := newMCPWatchingBackend(t.Context(), path)
	require.NoError(t, err)
	previous := backend.current.Load()
	writeMCPWatchSource(t, path, "Second")
	pollMCPWatch(t, backend, false)
	writeMCPWatchSource(t, path, "Third")
	pollMCPWatch(t, backend, false)
	require.Same(t, previous, backend.current.Load(), "a changing candidate must not be published")
	require.NoError(t, os.Remove(path))
	changed, err := backend.poll(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, changed)
	require.False(t, backend.hasPending)
	writeMCPWatchSource(t, path, "Third")
	pollMCPWatch(t, backend, false)
	require.Same(t, previous, backend.current.Load(), "a read failure must reset debounce")
	pollMCPWatch(t, backend, true)
	previous = backend.current.Load()
	require.NoError(t, os.WriteFile(path, []byte("persons: ["), 0o600))
	pollMCPWatch(t, backend, false)
	changed, err = backend.poll(t.Context())
	require.Error(t, err)
	require.False(t, changed)
	require.False(t, backend.hasPending)
	require.Same(t, previous, backend.current.Load(), "parse errors must retain the last successful snapshot")
	writeMCPWatchSource(t, path, "Fixed")
	pollMCPWatch(t, backend, false)
	pollMCPWatch(t, backend, true)
}

func TestMCPWatchJournalAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "person.glx")
	journal := filepath.Join(dir, inPlaceJournalFile)
	writeMCPWatchSource(t, path, "First")
	require.NoError(t, os.WriteFile(journal, []byte("[]"), 0o600))
	_, err := newMCPWatchingBackend(t.Context(), dir)
	require.ErrorIs(t, err, errMCPWatchJournal, "startup cannot load an in-progress/interrupted write")
	require.NoError(t, os.Remove(journal))
	backend, err := newMCPWatchingBackend(t.Context(), dir)
	require.NoError(t, err)
	previous := backend.current.Load()
	duplicate := filepath.Join(dir, "duplicate.glx")
	writeMCPWatchSource(t, duplicate, "Duplicate")
	pollMCPWatch(t, backend, false)
	require.NoError(t, os.WriteFile(journal, []byte("[]"), 0o600))
	changed, err := backend.poll(t.Context())
	require.ErrorIs(t, err, errMCPWatchJournal)
	require.False(t, changed)
	require.Same(t, previous, backend.current.Load())
	require.NoError(t, os.Remove(journal))
	pollMCPWatch(t, backend, false)
	pollMCPWatch(t, backend, true)
	result, err := backend.CallTool(t.Context(), "validate", json.RawMessage(`{}`))
	require.NoError(t, err)
	validation, ok := result.(*validationJSONResult)
	require.True(t, ok)
	require.False(t, validation.Valid)
	require.NotEmpty(t, validation.DuplicateIDs)
	require.NoError(t, os.Remove(duplicate))
	pollMCPWatch(t, backend, false)
	pollMCPWatch(t, backend, true)
	result, err = backend.CallTool(t.Context(), "validate", json.RawMessage(`{}`))
	require.NoError(t, err)
	validation, ok = result.(*validationJSONResult)
	require.True(t, ok)
	require.Empty(t, validation.DuplicateIDs)
}

func TestMCPWatchSourceContainmentAndType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	writeMCPWatchSource(t, path, "First")
	backend, err := newMCPWatchingBackend(t.Context(), dir)
	require.NoError(t, err)
	outside := filepath.Join(t.TempDir(), "outside.glx")
	writeMCPWatchSource(t, outside, "Private")
	link := filepath.Join(dir, "escape.glx")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	previous := backend.current.Load()
	changed, err := backend.poll(t.Context())
	require.Error(t, err)
	require.False(t, changed)
	require.Same(t, previous, backend.current.Load())
	require.NoError(t, os.Remove(link))
	fileBackend, err := newMCPWatchingBackend(t.Context(), path)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Mkdir(path, 0o700))
	changed, err = fileBackend.poll(t.Context())
	require.ErrorIs(t, err, errMCPWatchSourceType)
	require.False(t, changed)
}

func TestMCPWatchConcurrentSnapshotReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	writeMCPWatchSource(t, path, "First")
	backend, err := newMCPWatchingBackend(t.Context(), path)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var readers sync.WaitGroup
	readers.Go(func() {
		for ctx.Err() == nil {
			// One caller matches the protocol's serialization. Reload runs
			// independently, including while old results are being marshaled.
			result, callErr := backend.CallTool(ctx, "query", json.RawMessage(`{"entity_type":"persons"}`))
			if callErr != nil {
				return // Cancellation ends this reader.
			}
			if _, marshalErr := json.Marshal(result); marshalErr != nil {
				t.Errorf("encoding a snapshot: %v", marshalErr)

				return
			}
		}
	})
	for i := range 20 {
		writeMCPWatchSource(t, path, fmt.Sprintf("Version%d", i))
		pollMCPWatch(t, backend, false)
		pollMCPWatch(t, backend, true)
	}
	cancel()
	readers.Wait()
}

func TestMCPWatchCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	writeMCPWatchSource(t, path, "First")
	backend, err := newMCPWatchingBackend(t.Context(), path)
	require.NoError(t, err)
	writeMCPWatchSource(t, path, "Second")
	pollMCPWatch(t, backend, false)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	changed, err := backend.poll(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, changed)
	streams, _, stderr := TestIOStreams()
	done := make(chan struct{})
	go func() {
		defer close(done)
		backend.run(ctx, streams)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watcher did not stop after cancellation")
	}
	require.Empty(t, stderr.String())
}

func TestMCPSourceFingerprintIncludesPathsAndBoundaries(t *testing.T) {
	first := map[string][]byte{"one": []byte("two"), "three": []byte("four")}
	reordered := map[string][]byte{"three": []byte("four"), "one": []byte("two")}
	require.Equal(t, mcpSourceFingerprint(first), mcpSourceFingerprint(reordered))
	require.NotEqual(t, mcpSourceFingerprint(map[string][]byte{"a": []byte("bc")}), mcpSourceFingerprint(map[string][]byte{"ab": []byte("c")}))
}

func TestMCPWatchDiagnosticsDeduplicateAndRecover(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "archive.glx")
		writeMCPWatchSource(t, path, "First")
		backend, err := newMCPWatchingBackend(t.Context(), path)
		require.NoError(t, err)
		streams, _, stderr := TestIOStreams()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan struct{})
		go func() {
			defer close(done)
			backend.run(ctx, streams)
		}()
		synctest.Wait()
		require.NoError(t, os.Remove(path))
		time.Sleep(3 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, strings.Count(stderr.String(), "keeping previous snapshot"))
		// Recovering the original bytes must reset warning deduplication even
		// though there is no new snapshot to publish.
		writeMCPWatchSource(t, path, "First")
		time.Sleep(time.Second)
		synctest.Wait()
		require.NoError(t, os.Remove(path))
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, 2, strings.Count(stderr.String(), "keeping previous snapshot"))
		require.NoError(t, os.WriteFile(path, []byte("persons: ["), 0o600))
		time.Sleep(5 * time.Second)
		synctest.Wait()
		require.Equal(t, 3, strings.Count(stderr.String(), "keeping previous snapshot"), "pending parse retries must not repeat identical warnings")
		writeMCPWatchSource(t, path, "Recovered")
		time.Sleep(2 * time.Second)
		synctest.Wait()
		require.Contains(t, stderr.String(), "reloaded archive snapshot")
		cancel()
		<-done
	})
}

func TestMCPWatchLoadsCapturedBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.glx")
	writeMCPWatchSource(t, path, "Captured")
	source := &mcpArchiveSource{path: path}
	capture, err := source.capture(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("persons: ["), 0o600))
	backend, err := source.load(capture)
	require.NoError(t, err)
	require.Equal(t, "Captured", backend.archive.Persons["person-one"].Properties["name"])
}
