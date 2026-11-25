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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPStdioSingleFileSymlinkStartup(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(dir, "outside.glx")
	watchWrite(t, outside, "persons:\n  person-outside:\n    properties:\n      name: OutsideSyntheticMarker\n")
	selected := filepath.Join(dir, "selected", "archive.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(selected), 0o700))
	if err := os.Symlink(outside, selected); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	before := snapshotTree(t, dir)
	for _, args := range [][]string{
		{"mcp", "serve", selected},
		{"mcp", "serve", "--watch", selected},
	} {
		result := runGLX(t, dir, args...)
		require.NotZero(t, result.exitCode)
		require.Contains(t, result.stderr, "regular file")
		require.Empty(t, result.stdout)
		require.NotContains(t, result.stderr, "OutsideSyntheticMarker")
	}
	require.Equal(t, before, snapshotTree(t, dir))
}

func TestMCPStdioSingleFileSymlinkRetentionAndRecovery(t *testing.T) {
	dir := t.TempDir()
	selected := filepath.Join(dir, "selected", "archive.glx")
	one := "persons:\n  person-first:\n    properties:\n      name: OriginalSyntheticMarker\n"
	recovered := "persons:\n  person-first:\n    properties:\n      name: RecoveredSyntheticMarker\n"
	outside := filepath.Join(dir, "outside.glx")
	outsideData := "persons:\n  person-outside:\n    properties:\n      name: OutsideSyntheticMarker\n  person-other:\n    properties:\n      name: OtherSyntheticMarker\n"
	watchWrite(t, selected, one)
	watchWrite(t, outside, outsideData)
	link := filepath.Join(filepath.Dir(selected), ".replacement")
	if err := os.Symlink(filepath.Join("..", "outside.glx"), link); err != nil {
		t.Skipf("symlinks not available: %v", err)
	}
	session, stderr := connectMCP(t, dir, "--watch", selected)
	require.Equal(t, 1, watchPersons(t, session))
	original := callMCP(t, session, "query", map[string]any{"entity_type": "persons"})
	require.Contains(t, string(original), "OriginalSyntheticMarker")
	require.NoError(t, os.Rename(link, selected))
	watchRetainsPersons(t, session, 1) // Spans multiple real polling periods.
	require.JSONEq(t, string(original), string(callMCP(t, session, "query", map[string]any{"entity_type": "persons"})))
	watchWrite(t, link, recovered)
	require.NoError(t, os.Rename(link, selected))
	expected := snapshotTree(t, dir)
	watchAwait(t, func() bool {
		data := callMCP(t, session, "search", map[string]any{"query": "RecoveredSyntheticMarker"})

		return strings.Contains(string(data), "person-first")
	})
	require.NoError(t, session.Close())
	require.Contains(t, stderr.String(), "MCP watch: keeping previous snapshot:")
	require.Contains(t, stderr.String(), "MCP watch: reloaded archive snapshot")
	require.NotContains(t, stderr.String(), "OutsideSyntheticMarker")
	require.Equal(t, expected, snapshotTree(t, dir), "MCP must not write archive/cache files")
	outsideAfter, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, outsideData, string(outsideAfter))
}
