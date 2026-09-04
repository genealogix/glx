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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The basic-family example has a two-event file (events/event-births.glx),
// which is what #1197 was reported against. Renaming Robert touches his
// person file, that events file, the marriage event, and three relationships.
const (
	renameOldID = "person-robert-thompson"
	renameNewID = "person-robert-t"
)

var renameExpectedChanged = []string{
	"events/event-births.glx",
	"events/event-marriage.glx",
	"relationships/rel-marriage.glx",
	"relationships/rel-parent-alice.glx",
	"relationships/rel-parent-robert-jr.glx",
}

func TestRename_FromInsideArchiveRoot_DefaultArchiveFlag(t *testing.T) {
	// The exact shape of #1192: cwd is the archive root, no --archive flag.
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)
	dirBefore, err := os.Stat(archive)
	require.NoError(t, err)

	res := runGLX(t, archive, "rename", renameOldID, renameNewID)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Renaming "+renameOldID+" → "+renameNewID+" (persons)")
	assert.Contains(t, res.stdout, "Updated 6 reference(s) in 7 file(s)")
	assert.Empty(t, res.stderr)

	// The directory the user's shell is sitting in is still the archive.
	dirAfter, err := os.Stat(archive)
	require.NoError(t, err)
	assert.True(t, os.SameFile(dirBefore, dirAfter), "archive directory was replaced (#1192)")

	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, renameExpectedChanged, diff.changed)
	assert.Equal(t, []string{"persons/" + renameNewID + ".glx"}, diff.created)
	assert.Equal(t, []string{"persons/" + renameOldID + ".glx"}, diff.removed)

	// The two-event file is still one file holding both events (#1197).
	births, err := os.ReadFile(filepath.Join(archive, "events/event-births.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(births), "event-birth-robert:")
	assert.Contains(t, string(births), "event-birth-mary:")

	// And the archive is still valid according to the CLI itself.
	validate := runGLX(t, archive, "validate", ".")
	assert.Equal(t, 0, validate.exitCode, validate.stdout+validate.stderr)
}

func TestRename_ArchiveFlagFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, t.TempDir(), "rename", renameOldID, renameNewID, "--archive", archive)

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, renameExpectedChanged, diff.changed)
	assert.Equal(t, []string{"persons/" + renameNewID + ".glx"}, diff.created)
	assert.Equal(t, []string{"persons/" + renameOldID + ".glx"}, diff.removed)
}

func TestRename_ShortArchiveFlag(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, t.TempDir(), "rename", renameOldID, renameNewID, "-a", archive)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(archive, "persons", renameNewID+".glx"))
}

func TestRename_DryRunWritesNothing(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "rename", renameOldID, renameNewID, "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Renaming "+renameOldID)
	assert.Contains(t, res.stdout, "dry run")
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Empty(t, diff.created)
	assert.Empty(t, diff.removed)
}

func TestRename_UnknownIDFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "rename", "person-nobody", "person-someone")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "not found")
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Empty(t, diff.created)
	assert.Empty(t, diff.removed)
}

func TestRename_TakenIDFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "rename", renameOldID, "person-mary-thompson")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "already exists")
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Empty(t, diff.created)
	assert.Empty(t, diff.removed)
}

func TestRename_ArgumentCountIsEnforced(t *testing.T) {
	archive := copyExample(t, "basic-family")

	one := runGLX(t, archive, "rename", renameOldID)
	assert.NotEqual(t, 0, one.exitCode)
	assert.Contains(t, one.stderr, "accepts 2 arg(s)")

	three := runGLX(t, archive, "rename", renameOldID, renameNewID, "extra")
	assert.NotEqual(t, 0, three.exitCode)
	assert.Contains(t, three.stderr, "accepts 2 arg(s)")
}

func TestRename_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "rename", renameOldID, renameNewID, "--archive", "does-not-exist")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "cannot access path")
}

func TestRename_SingleFileArchive(t *testing.T) {
	archive := copyExample(t, "single-file")
	file := filepath.Join(archive, "archive.glx")
	data, err := os.ReadFile(file)
	require.NoError(t, err)
	oldID, newID := "person-a1b2c3d4", "person-e2e-renamed"
	require.Contains(t, string(data), oldID+":")

	res := runGLX(t, archive, "rename", oldID, newID, "--archive", "archive.glx")

	require.Equal(t, 0, res.exitCode, res.stderr)
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(after), newID+":")
	assert.NotContains(t, string(after), oldID+":")
	assert.FileExists(t, file, "single-file archive must stay a single file")
	entries, err := os.ReadDir(archive)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, e.IsDir() && e.Name() == "persons", "single-file archive must not be split")
	}
}

func TestRename_HelpMentionsFlags(t *testing.T) {
	res := runGLX(t, t.TempDir(), "rename", "--help")

	require.Equal(t, 0, res.exitCode)
	assert.Contains(t, res.stdout, "--archive")
	assert.Contains(t, res.stdout, "--dry-run")
	assert.Contains(t, res.stdout, "rename <old-id> <new-id>")
}
