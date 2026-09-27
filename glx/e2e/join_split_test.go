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

// join then split is a round trip: the split archive holds the same entities
// as the original, which `glx diff` reports as no changes.
func TestJoinSplit_RoundTrip(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)
	work := t.TempDir()

	join := runGLX(t, work, "join", archive, "family.glx")
	require.Equal(t, 0, join.exitCode, join.stderr)
	assert.Contains(t, join.stdout, "Successfully joined archive to family.glx")
	assertTreeUnchanged(t, before, archive)
	joined := runGLX(t, work, "validate", "family.glx")
	require.Equal(t, 0, joined.exitCode, joined.stdout+joined.stderr)

	split := runGLX(t, work, "split", "family.glx", "split")
	require.Equal(t, 0, split.exitCode, split.stderr)
	assert.Contains(t, split.stdout, "Successfully split archive to split")
	assertArchiveValid(t, filepath.Join(work, "split"))

	diff := runGLX(t, work, "diff", archive, "split")
	require.Equal(t, 0, diff.exitCode, diff.stderr)
	assert.Contains(t, diff.stdout, "No changes.")
}

// Neither command may overwrite existing output.
func TestJoinSplit_RefuseExistingOutput(t *testing.T) {
	archive := copyExample(t, "basic-family")
	work := t.TempDir()
	require.Equal(t, 0, runGLX(t, work, "join", archive, "family.glx").exitCode)
	require.Equal(t, 0, runGLX(t, work, "split", "family.glx", "split").exitCode)
	before := snapshotTree(t, work)

	assertExitWithStderr(t, runGLX(t, work, "join", archive, "family.glx"), "output file already exists")
	assertExitWithStderr(t, runGLX(t, work, "split", "family.glx", "split"), "output directory already exists")
	assertTreeUnchanged(t, before, work)
}

func TestJoinSplit_Errors(t *testing.T) {
	work := t.TempDir()

	assertExitWithStderr(t, runGLX(t, work, "join", "does-not-exist", "x.glx"), "input directory not found")
	assertExitWithStderr(t, runGLX(t, work, "split", "does-not-exist.glx", "out"), "input file not found")
	assertExitWithStderr(t, runGLX(t, work, "join", "only-one"), "accepts 2 arg(s), received 1")
	assertExitWithStderr(t, runGLX(t, work, "split", "only-one"), "accepts 2 arg(s), received 1")
	entries, err := os.ReadDir(work)
	require.NoError(t, err)
	assert.Empty(t, entries, "failed commands must not leave output behind")
}
