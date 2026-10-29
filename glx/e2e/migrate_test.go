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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// legacyGenderArchive stages basic-family with every person's `sex` property
// renamed back to the pre-split `gender`, so --rename-gender-to-sex has work
// to do. The migration refuses to run when any person already uses `sex`.
func legacyGenderArchive(t *testing.T) string {
	t.Helper()
	archive := copyExample(t, "basic-family")
	persons, err := filepath.Glob(filepath.Join(archive, "persons", "*.glx"))
	require.NoError(t, err)
	require.NotEmpty(t, persons)
	for _, p := range persons {
		data, err := os.ReadFile(p) //nolint:gosec // test fixture path
		require.NoError(t, err)
		legacy := strings.ReplaceAll(string(data), "      sex: ", "      gender: ")
		require.NotEqual(t, string(data), legacy, "%s has no sex property to rewrite", p)
		require.NoError(t, os.WriteFile(p, []byte(legacy), 0o644)) //nolint:gosec // test fixture
	}

	return archive
}

func TestMigrate_FromInsideArchiveRoot(t *testing.T) {
	archive := legacyGenderArchive(t)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "migrate", ".", "--rename-gender-to-sex")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Gender→sex properties:     4")
	assertSameDirectory(t, dirBefore, archive)
	robert, err := os.ReadFile(filepath.Join(archive, "persons", "person-robert-thompson.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(robert), "sex: male")
	assert.NotContains(t, string(robert), "gender:")
	assertArchiveValid(t, archive)
}

func TestMigrate_ArchivePathFromOutside(t *testing.T) {
	archive := legacyGenderArchive(t)

	res := runGLX(t, t.TempDir(), "migrate", archive, "--rename-gender-to-sex")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assertArchiveValid(t, archive)
}

func TestMigrate_UpToDateArchiveWritesNothing(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "migrate", ".")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "already up to date")
	assertTreeUnchanged(t, before, archive)
}

func TestMigrate_SingleFileArchive(t *testing.T) {
	archive := copyExample(t, "single-file")

	res := runGLX(t, archive, "migrate", "archive.glx")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(archive, "archive.glx"))
	assert.NoDirExists(t, filepath.Join(archive, "persons"), "a single-file archive must stay a single file")
}

func TestMigrate_ArgumentCountIsEnforced(t *testing.T) {
	archive := copyExample(t, "basic-family")

	assertExitWithStderr(t, runGLX(t, archive, "migrate"), "accepts 1 arg(s), received 0")
	assertExitWithStderr(t, runGLX(t, archive, "migrate", ".", "extra"), "accepts 1 arg(s), received 2")
}

func TestMigrate_MissingArchivePathFails(t *testing.T) {
	assertExitWithStderr(t, runGLX(t, t.TempDir(), "migrate", "does-not-exist"), "cannot access path")
}
