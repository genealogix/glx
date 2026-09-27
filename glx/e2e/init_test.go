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

func TestInit_NamedDirectory(t *testing.T) {
	parent := t.TempDir()

	res := runGLX(t, parent, "init", "family")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Initialized multi-file GENEALOGIX repository in family")
	archive := filepath.Join(parent, "family")
	for _, dir := range []string{"persons", "events", "relationships", "places", "sources", "citations", "repositories", "assertions", "media", "vocabularies"} {
		assert.DirExists(t, filepath.Join(archive, dir))
	}
	assert.FileExists(t, filepath.Join(archive, ".gitignore"))
	assert.FileExists(t, filepath.Join(archive, "README.md"))
	assert.FileExists(t, filepath.Join(archive, "vocabularies", "event-types.glx"))
	assertArchiveValid(t, archive)
}

// With no argument, init writes into the cwd — the directory the user's shell
// is in must be the one that becomes the archive.
func TestInit_CurrentDirectory(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "here")
	require.NoError(t, os.Mkdir(archive, 0o755))
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "init")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "in the current directory")
	assertSameDirectory(t, dirBefore, archive)
	assert.DirExists(t, filepath.Join(archive, "persons"))
	assertArchiveValid(t, archive)
}

func TestInit_SingleFile(t *testing.T) {
	parent := t.TempDir()

	res := runGLX(t, parent, "init", "family", "--single-file")

	require.Equal(t, 0, res.exitCode, res.stderr)
	archive := filepath.Join(parent, "family")
	assert.Equal(t, []string{"archive.glx"}, treePaths(t, archive), "a single-file archive is exactly one file")
	validate := runGLX(t, archive, "validate", "archive.glx")
	assert.Equal(t, 0, validate.exitCode, validate.stdout+validate.stderr)
}

func TestInit_TestDataIsValid(t *testing.T) {
	parent := t.TempDir()

	res := runGLX(t, parent, "init", "family", "--create-test-data", "3")

	require.Equal(t, 0, res.exitCode, res.stderr)
	persons, err := os.ReadDir(filepath.Join(parent, "family", "persons"))
	require.NoError(t, err)
	assert.Len(t, persons, 3)
	assertArchiveValid(t, filepath.Join(parent, "family"))
}

func TestInit_RefusesNonEmptyDirectory(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	assertExitWithStderr(t, runGLX(t, archive, "init"), "non-empty directory")
	assertTreeUnchanged(t, before, archive)
}

func TestInit_ArgumentCountIsEnforced(t *testing.T) {
	parent := t.TempDir()

	assertExitWithStderr(t, runGLX(t, parent, "init", "a", "b"), "accepts at most 1 arg(s), received 2")
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing is created when the arguments are rejected")
}
