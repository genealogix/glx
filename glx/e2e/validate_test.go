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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A git worktree checked out under a dot-prefixed directory is a full copy of the
// archive inside the archive (#1212). Every entity would collide with itself
// if the walk entered it.
func TestValidate_IgnoresArchiveCopyUnderDotDirectory(t *testing.T) {
	archive := copyExample(t, "basic-family")

	clean := runGLX(t, archive, "validate", ".")
	require.Equal(t, 0, clean.exitCode, clean.stdout+clean.stderr)
	// Without this the comparison below is vacuous: if the walk found nothing
	// in either run, both print the same "No GLX files found" line and the
	// equality assertion passes while validate is doing nothing at all.
	require.NotContains(t, clean.stdout, "Validated 0 files.")
	require.Contains(t, clean.stdout, "Validated ")

	copyTreeFollowingSymlinks(t, filepath.Join(examplesDir(t), "basic-family"), filepath.Join(archive, ".worktrees", "copy"))

	res := runGLX(t, archive, "validate", ".")

	assert.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Equal(t, clean.stdout, res.stdout)
	assert.NotContains(t, res.stderr, "conflict")
}

// The reproduction in #1270: join an archive to single-file form, break one
// reference, and the corruption that is a hard error in the multi-file form
// used to pass green in the joined one.
func TestValidate_JoinedSingleFileArchive_CatchesDanglingReference(t *testing.T) {
	archive := copyExample(t, "basic-family")
	workDir := filepath.Dir(archive)

	joined := filepath.Join(workDir, "joined.glx")
	join := runGLX(t, workDir, "join", archive, joined)
	require.Equal(t, 0, join.exitCode, join.stdout+join.stderr)

	ok := runGLX(t, workDir, "validate", joined)
	require.Equal(t, 0, ok.exitCode, ok.stdout+ok.stderr)
	require.Contains(t, ok.stdout, "✅ Archive is valid.")
	require.NotContains(t, ok.stdout, "Cross-reference validation skipped")

	content, err := os.ReadFile(joined)
	require.NoError(t, err)
	broken := filepath.Join(workDir, "broken.glx")
	require.NoError(t, os.WriteFile(broken,
		bytes.ReplaceAll(content, []byte("place: place-springfield"), []byte("place: place-DOES-NOT-EXIST")),
		0o644))

	res := runGLX(t, workDir, "validate", broken)

	assert.Equal(t, 1, res.exitCode, res.stdout+res.stderr)
	assert.Contains(t, res.stderr, "references non-existent places: place-DOES-NOT-EXIST")
}

// An entity file of a multi-file archive is a fragment: its references resolve
// in its siblings, so validating it alone must not report them as dangling.
func TestValidate_EntityFragment_KeepsCrossReferenceSkip(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "validate", filepath.Join("events", "event-births.glx"))

	assert.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Contains(t, res.stdout, "Cross-reference validation skipped")
	assert.NotContains(t, res.stderr, "references non-existent")
}
