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

const (
	mergeKeepID = "person-robert-thompson"
	mergeDropID = "person-robert-dup"
)

// archiveWithDuplicate stages basic-family plus a second record for Robert
// Thompson, created through the CLI, and returns the archive path.
func archiveWithDuplicate(t *testing.T) string {
	t.Helper()
	archive := copyExample(t, "basic-family")
	res := runGLX(t, archive, "add", "person", "--id", mergeDropID,
		"--given", "Robert", "--surname", "Thompson", "--sex", "male", "--note", "second record")
	require.Equal(t, 0, res.exitCode, res.stderr)
	require.FileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))

	return archive
}

func TestMergePersons_FromInsideArchiveRoot_DefaultArchiveFlag(t *testing.T) {
	archive := archiveWithDuplicate(t)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Merging "+mergeKeepID+" ← "+mergeDropID)
	assertSameDirectory(t, dirBefore, archive)
	assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
	kept, err := os.ReadFile(filepath.Join(archive, "persons", mergeKeepID+".glx"))
	require.NoError(t, err)
	assert.Contains(t, string(kept), "second record", "the dropped person's note is appended by default")
	assertArchiveValid(t, archive)
}

func TestMergePersons_ArchiveFlagFromOutside(t *testing.T) {
	archive := archiveWithDuplicate(t)

	res := runGLX(t, t.TempDir(), "merge-persons", mergeKeepID, mergeDropID, "--archive", archive)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
	assertArchiveValid(t, archive)
}

func TestMergePersons_DryRunWritesNothing(t *testing.T) {
	archive := archiveWithDuplicate(t)
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Merging "+mergeKeepID)
	assertTreeUnchanged(t, before, archive)
}

func TestMergePersons_NotesStrategyPreferKeep(t *testing.T) {
	archive := archiveWithDuplicate(t)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--notes-strategy", "prefer-keep")

	require.Equal(t, 0, res.exitCode, res.stderr)
	kept, err := os.ReadFile(filepath.Join(archive, "persons", mergeKeepID+".glx"))
	require.NoError(t, err)
	assert.NotContains(t, string(kept), "second record")
}

func TestMergePersons_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := archiveWithDuplicate(t)
	before := snapshotTree(t, archive)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown keep", []string{"person-nobody", mergeDropID}, `keep-id "person-nobody"`},
		{"unknown drop", []string{mergeKeepID, "person-nobody"}, `drop-id "person-nobody"`},
		{"self merge", []string{mergeKeepID, mergeKeepID}, "cannot merge person with itself"},
		{"one arg", []string{mergeKeepID}, "accepts 2 arg(s), received 1"},
		{"three args", []string{mergeKeepID, mergeDropID, "extra"}, "accepts 2 arg(s), received 3"},
		{"conflicting keep flags", []string{mergeKeepID, mergeDropID, "--keep-newest", "--keep-oldest"}, "mutually exclusive"},
		{"bad notes strategy", []string{mergeKeepID, mergeDropID, "--notes-strategy", "bogus"}, "invalid notes strategy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"merge-persons"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestMergePersons_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "merge-persons", mergeKeepID, mergeDropID, "--archive", "does-not-exist")

	assertExitWithStderr(t, res, "cannot access path")
}
