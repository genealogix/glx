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

// A target that does not hold an archive must fail, not pass having checked
// nothing: a typo in a CI step's path would otherwise stay green forever.
func TestValidate_TargetsWithNothingToValidateFail(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(work, "empty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(work, "notes.txt"), []byte("not a GLX file\n"), 0o644))
	archive := copyExample(t, "basic-family")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing path", []string{"validate", "archvie"}, "cannot access path"},
		{"missing path among valid ones", []string{"validate", filepath.Join(archive, "persons"), filepath.Join(archive, "evnets")}, "cannot access path"},
		{"empty directory", []string{"validate", "empty"}, "No GLX files found"},
		{"non-GLX file", []string{"validate", "notes.txt"}, "No GLX files found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runGLX(t, work, tc.args...)

			assertExitWithStderr(t, res, tc.want)
			assert.NotContains(t, res.stdout, "valid", "a failed target must not print a pass")
		})
	}

	// Bare `glx validate` in a directory that is not an archive fails too.
	assertExitWithStderr(t, runGLX(t, filepath.Join(work, "empty"), "validate"), "No GLX files found")
}
