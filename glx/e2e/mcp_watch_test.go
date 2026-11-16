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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

func watchWrite(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func watchPersons(t *testing.T, session *mcpsdk.ClientSession) int {
	t.Helper()
	var result struct {
		Counts map[string]int `json:"entity_counts"`
	}
	require.NoError(t, json.Unmarshal(callMCP(t, session, "list_entity_types", map[string]any{}), &result))

	return result.Counts["persons"]
}

func watchAwait(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("MCP watch did not publish the expected snapshot within 8 seconds")
}

func watchRetainsPersons(t *testing.T, session *mcpsdk.ClientSession, count int) {
	t.Helper()
	// Span more than two polling periods so a rejected reload is exercised.
	deadline := time.Now().Add(2500 * time.Millisecond)
	for time.Now().Before(deadline) {
		require.Equal(t, count, watchPersons(t, session))
		time.Sleep(150 * time.Millisecond)
	}
}

func TestMCPWatchDirectoryReloadRecoveryAndNoWrites(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "persons", "first.glx")
	watchWrite(t, first, "persons:\n  person-first:\n    properties:\n      name: Before\n")
	built := runGLX(t, dir, "cache", "build")
	require.Zero(t, built.exitCode, built.stderr)
	initialCache := snapshotTree(t, filepath.Join(dir, ".glx"))
	session, stderr := connectMCP(t, dir, "--watch")
	require.Equal(t, 1, watchPersons(t, session))

	// Invalid ignored files must neither poison reloads nor be rewritten.
	watchWrite(t, filepath.Join(dir, ".hidden", "bad.glx"), "persons: [")
	watchWrite(t, filepath.Join(dir, ".ignored.glx"), "persons: [")
	watchWrite(t, filepath.Join(dir, "notes.txt"), "persons: [")
	nested := filepath.Join(dir, "persons", "nested", "renamed.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(nested), 0o700))
	require.NoError(t, os.Rename(first, nested))
	watchWrite(t, nested, "persons:\n  person-first:\n    properties:\n      name: After\n")
	second := filepath.Join(dir, "persons", "nested", "second.glx")
	watchWrite(t, second, "persons:\n  person-second:\n    properties:\n      name: Second\n")
	watchAwait(t, func() bool { return watchPersons(t, session) == 2 })
	query := callMCP(t, session, "query", map[string]any{"entity_type": "persons", "name": "After"})
	require.Contains(t, string(query), "person-first", "reload must bypass the previously built cache")

	watchWrite(t, nested, "persons: [not-valid-yaml")
	watchRetainsPersons(t, session, 2)
	watchWrite(t, nested, "persons:\n  person-first:\n    properties:\n      name: Recovered\n")
	require.NoError(t, os.Remove(second))
	watchAwait(t, func() bool {
		return watchPersons(t, session) == 1 && strings.Contains(string(callMCP(t, session, "query", map[string]any{"entity_type": "persons", "name": "Recovered"})), "person-first")
	})

	watchWrite(t, filepath.Join(dir, "vocabularies", "event-types.glx"), "event_types:\n  watch_marker:\n    label: Watch marker\n")
	expected := snapshotTree(t, dir)
	watchAwait(t, func() bool {
		resource, err := session.ReadResource(t.Context(), &mcpsdk.ReadResourceParams{URI: "glx://archive/vocabularies"})
		require.NoError(t, err)

		return len(resource.Contents) == 1 && strings.Contains(resource.Contents[0].Text, "watch_marker")
	})
	require.NoError(t, session.Close())
	// Reading stderr is safe only after the command transport has stopped.
	require.NotContains(t, stderr.String(), "panic")
	require.Contains(t, stderr.String(), "MCP watch: keeping previous snapshot:")
	require.Contains(t, stderr.String(), "MCP watch: reloaded archive snapshot")
	require.Equal(t, expected, snapshotTree(t, dir), "watch must not write into the archive")
	require.Equal(t, initialCache, snapshotTree(t, filepath.Join(dir, ".glx")), "existing cache must remain byte-identical throughout every reload")
}

func TestMCPWatchSingleFileAtomicSaveAndMissingRecovery(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "archive.glx")
	one := "persons:\n  person-first:\n    properties:\n      name: First\n"
	two := one + "  person-second:\n    properties:\n      name: Second\n"
	watchWrite(t, archive, one)
	session, stderr := connectMCP(t, dir, "--watch", archive)
	require.Equal(t, 1, watchPersons(t, session))
	staged := filepath.Join(dir, ".atomic-save.tmp")
	watchWrite(t, staged, two)
	require.NoError(t, os.Rename(staged, archive))
	watchAwait(t, func() bool { return watchPersons(t, session) == 2 })

	// Editors can briefly remove the source while preparing a replacement.
	require.NoError(t, os.Rename(archive, staged))
	watchRetainsPersons(t, session, 2)
	watchWrite(t, staged, two+"  person-third:\n    properties:\n      name: Third\n")
	require.NoError(t, os.Rename(staged, archive))
	expected := snapshotTree(t, dir)
	watchAwait(t, func() bool { return watchPersons(t, session) == 3 })
	require.NoError(t, session.Close())
	require.NotContains(t, stderr.String(), "panic")
	require.Contains(t, stderr.String(), "MCP watch: keeping previous snapshot:")
	require.Contains(t, stderr.String(), "MCP watch: reloaded archive snapshot")
	require.Equal(t, expected, snapshotTree(t, dir))
	_, err := os.Stat(filepath.Join(dir, ".glx"))
	require.ErrorIs(t, err, os.ErrNotExist, "watch must not create a cache with GLX_CACHE=auto")
}
