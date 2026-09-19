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

// export was the only archive-taking command with a required positional, so
// the obvious invocation from inside an archive failed on argument count
// before any export ran (#1274). The archive now defaults to ".", matching
// stats / serve / validate / places.
func TestExport_DefaultsArchiveToCurrentDirectory(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "export", "-f", "70", "-o", "out.ged")

	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	data, err := os.ReadFile(filepath.Join(archive, "out.ged"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "0 HEAD")
	assert.Contains(t, string(data), "2 VERS 7.0")
}

// The default must not change what an explicit archive argument does: every
// invocation that worked before keeps working, and both spellings produce
// byte-identical output.
func TestExport_ExplicitArchiveMatchesDefault(t *testing.T) {
	archive := copyExample(t, "basic-family")

	implicit := runGLX(t, archive, "export", "-f", "70", "-o", "implicit.ged")
	require.Equal(t, 0, implicit.exitCode, implicit.stdout+implicit.stderr)

	explicit := runGLX(t, archive, "export", ".", "-f", "70", "-o", "explicit.ged")
	require.Equal(t, 0, explicit.exitCode, explicit.stdout+explicit.stderr)

	fromImplicit, err := os.ReadFile(filepath.Join(archive, "implicit.ged"))
	require.NoError(t, err)
	fromExplicit, err := os.ReadFile(filepath.Join(archive, "explicit.ged"))
	require.NoError(t, err)
	assert.Equal(t, string(fromImplicit), string(fromExplicit))
}

// A named archive one directory up still resolves relative to the cwd.
func TestExport_NamedArchiveFromParentDirectory(t *testing.T) {
	archive := copyExample(t, "basic-family")
	parent := filepath.Dir(archive)

	res := runGLX(t, parent, "export", filepath.Base(archive), "-f", "551", "-o", "family.ged")

	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.FileExists(t, filepath.Join(parent, "family.ged"))
}

// --output is still required, and its absence is still reported as a missing
// flag rather than silently writing somewhere.
func TestExport_StillRequiresOutputFlag(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "export", "-f", "70")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, `required flag(s) "output" not set`)
}

// Passing a second positional is a mistake worth reporting, not an argument
// the command quietly ignores.
func TestExport_RejectsSecondPositional(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "export", ".", "extra", "-o", "out.ged")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "accepts at most 1 arg(s), received 2")
	assert.NoFileExists(t, filepath.Join(archive, "out.ged"))
}

// join and split keep both positionals — an output path cannot default — but
// cobra's "accepts 2 arg(s), received 0" never said which argument was owed
// (#1274).
func TestJoinSplit_NameTheMissingPositional(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"join with none", []string{"join"}, "glx join: missing required argument(s): <input-directory> <output-file>"},
		{"join with one", []string{"join", "family-archive"}, "glx join: missing required argument(s): <output-file>"},
		{"split with none", []string{"split"}, "glx split: missing required argument(s): <input-file> <output-directory>"},
		{"split with one", []string{"split", "family.glx"}, "glx split: missing required argument(s): <output-directory>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := runGLX(t, dir, tt.args...)

			assert.NotEqual(t, 0, res.exitCode)
			assert.Contains(t, res.stderr, tt.want)
		})
	}
}

// Too many positionals are named too, so the count in the message can be
// matched against the argument list the user typed.
func TestJoin_NamesPositionalsWhenTooMany(t *testing.T) {
	dir := t.TempDir()

	res := runGLX(t, dir, "join", "a", "b", "c")

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "glx join: too many arguments: accepts 2 (<input-directory> <output-file>), received 3")
}
