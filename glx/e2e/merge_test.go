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

// The minimal example holds exactly one person, person-abc123, which
// basic-family does not have, so merging it in adds one entity.
const mergeNewPerson = "persons/person-abc123.glx"

func TestMerge_FromInsideDestination_DefaultIntoFlag(t *testing.T) {
	// cwd is the destination and --into defaults to ".": the #1192 shape.
	dest := copyExample(t, "basic-family")
	src := copyExample(t, "minimal")
	dirBefore := statDir(t, dest)
	readme, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)

	res := runGLX(t, dest, "merge", src)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "New entities:")
	assert.Contains(t, res.stdout, "1 persons")
	assertSameDirectory(t, dirBefore, dest)
	assert.FileExists(t, filepath.Join(dest, mergeNewPerson))
	after, err := os.ReadFile(filepath.Join(dest, "README.md"))
	require.NoError(t, err)
	assert.Equal(t, readme, after, "a non-archive file must survive the merge untouched")
	assertArchiveValid(t, dest)
}

func TestMerge_IntoFlagFromOutside(t *testing.T) {
	dest := copyExample(t, "basic-family")
	src := copyExample(t, "minimal")
	srcBefore := snapshotTree(t, src)

	res := runGLX(t, t.TempDir(), "merge", src, "--into", dest)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(dest, mergeNewPerson))
	assertTreeUnchanged(t, srcBefore, src)
	assertArchiveValid(t, dest)
}

func TestMerge_PreviewWritesNothing(t *testing.T) {
	dest := copyExample(t, "basic-family")
	src := copyExample(t, "minimal")
	before := snapshotTree(t, dest)

	res := runGLX(t, dest, "merge", src, "--preview")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "preview — no files written")
	assert.Contains(t, res.stdout, "cross-archive duplicates")
	assertTreeUnchanged(t, before, dest)
}

func TestMerge_ArchiveIntoItselfAddsNothing(t *testing.T) {
	dest := copyExample(t, "basic-family")
	src := copyExample(t, "basic-family")

	res := runGLX(t, dest, "merge", src)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.NotContains(t, res.stdout, "New entities:")
	assertArchiveValid(t, dest)
}

func TestMerge_ArgumentCountIsEnforced(t *testing.T) {
	dest := copyExample(t, "basic-family")

	assertExitWithStderr(t, runGLX(t, dest, "merge"), "accepts 1 arg(s), received 0")
	assertExitWithStderr(t, runGLX(t, dest, "merge", "a", "b"), "accepts 1 arg(s), received 2")
}

func TestMerge_MissingPathsFail(t *testing.T) {
	dest := copyExample(t, "basic-family")
	before := snapshotTree(t, dest)

	assertExitWithStderr(t, runGLX(t, dest, "merge", "does-not-exist"), "cannot access source")
	assertExitWithStderr(t, runGLX(t, t.TempDir(), "merge", dest, "--into", "does-not-exist"), "cannot access destination")
	assertTreeUnchanged(t, before, dest)
}
