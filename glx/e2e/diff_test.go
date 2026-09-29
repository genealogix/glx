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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var parentLinkLine = regexp.MustCompile(`(?m)^    parent: place-united-states\r?\n`)

// diffPair returns two copies of basic-family: old untouched, and new with a
// person added, a person's file edited, and a place removed.
func diffPair(t *testing.T) (string, string) {
	t.Helper()
	oldDir := copyExample(t, "basic-family")
	newDir := copyExample(t, "basic-family")
	res := runGLX(t, newDir, "add", "person", "--given", "Zed", "--surname", "New")
	require.Equal(t, 0, res.exitCode, res.stderr)
	alice := filepath.Join(newDir, "persons", "person-alice-thompson.glx")
	data, err := os.ReadFile(alice)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(alice, append(data, []byte("    notes: \"Edited for the diff test\"\n")...), 0o644))
	// Nothing references place-united-states except its child's parent link,
	// so drop that link too to keep the new side valid.
	require.NoError(t, os.Remove(filepath.Join(newDir, "places", "place-united-states.glx")))
	illinois := filepath.Join(newDir, "places", "place-illinois.glx")
	ill, err := os.ReadFile(illinois)
	require.NoError(t, err)
	// A Windows checkout may carry CRLF line endings.
	unlinked := parentLinkLine.ReplaceAllString(string(ill), "")
	require.NotEqual(t, string(ill), unlinked)
	require.NoError(t, os.WriteFile(illinois, []byte(unlinked), 0o644))
	assertArchiveValid(t, newDir)

	return oldDir, newDir
}

func TestDiff_IdenticalArchives(t *testing.T) {
	a := copyExample(t, "basic-family")
	b := copyExample(t, "basic-family")

	res := runGLX(t, t.TempDir(), "diff", a, b)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "No changes.")
}

func TestDiff_ReportsAddedModifiedRemoved(t *testing.T) {
	oldDir, newDir := diffPair(t)
	oldBefore, newBefore := snapshotTree(t, oldDir), snapshotTree(t, newDir)

	res := runGLX(t, t.TempDir(), "diff", oldDir, newDir)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "+ person-zed-new")
	assert.Contains(t, res.stdout, "person-alice-thompson")
	assert.Contains(t, res.stdout, "- place-united-states")
	assert.Contains(t, res.stdout, "+1 added")
	assert.Contains(t, res.stdout, "-1 removed")
	assertTreeUnchanged(t, oldBefore, oldDir)
	assertTreeUnchanged(t, newBefore, newDir)

	short := runGLX(t, t.TempDir(), "diff", oldDir, newDir, "--short")
	require.Equal(t, 0, short.exitCode, short.stderr)
	assert.Regexp(t, `^\+1 ~\d+ -1`, short.stdout)
}

func TestDiff_JSON(t *testing.T) {
	oldDir, newDir := diffPair(t)

	res := runGLX(t, t.TempDir(), "diff", oldDir, newDir, "--json")

	require.Equal(t, 0, res.exitCode, res.stderr)
	var doc struct {
		Changes []struct {
			Kind       string `json:"kind"`
			EntityType string `json:"entity_type"`
			ID         string `json:"id"`
		} `json:"changes"`
		Stats struct {
			Added   int `json:"added"`
			Removed int `json:"removed"`
		} `json:"stats"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.stdout), &doc), res.stdout)
	assert.Equal(t, 1, doc.Stats.Added)
	assert.Equal(t, 1, doc.Stats.Removed)
	kinds := map[string]string{}
	for _, c := range doc.Changes {
		kinds[c.ID] = c.Kind
	}
	assert.Equal(t, "added", kinds["person-zed-new"])
	assert.Equal(t, "removed", kinds["place-united-states"])
}

func TestDiff_PersonFilter(t *testing.T) {
	oldDir, newDir := diffPair(t)

	res := runGLX(t, t.TempDir(), "diff", oldDir, newDir, "--person", "person-zed-new")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "person-zed-new")
	assert.NotContains(t, res.stdout, "place-united-states")
}

func TestDiff_Errors(t *testing.T) {
	a := copyExample(t, "basic-family")
	work := t.TempDir()

	assertExitWithStderr(t, runGLX(t, work, "diff", a), "accepts 2 arg(s), received 1")
	assertExitWithStderr(t, runGLX(t, work, "diff", "does-not-exist", a), "loading does-not-exist")
	assertExitWithStderr(t, runGLX(t, work, "diff", a, "does-not-exist"), "loading does-not-exist")
}
