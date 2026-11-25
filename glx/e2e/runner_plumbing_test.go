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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSingleFileArchive runs `glx init --single-file` and returns the archive
// directory and the path of its one .glx file.
func newSingleFileArchive(t *testing.T) (string, string) {
	t.Helper()

	parent := t.TempDir()
	res := runGLX(t, parent, "init", "demo", "--single-file", "--no-git")
	require.Equal(t, 0, res.exitCode, res.stderr)
	dir := filepath.Join(parent, "demo")

	return dir, filepath.Join(dir, "archive.glx")
}

// glx add, given a single-file archive either way, refuses before writing
// anything and says how to convert it, rather than failing with an OS error
// or writing a multi-file entity beside the single file (#1356).
func TestAdd_SingleFileArchiveRefusedWithoutWriting(t *testing.T) {
	dir, file := newSingleFileArchive(t)

	for name, archive := range map[string]string{"file path": file, "archive directory": dir} {
		t.Run(name, func(t *testing.T) {
			before := snapshotTree(t, dir)

			res := runGLX(t, dir, "add", "person", "--archive", archive, "--given", "Carl")

			assert.NotEqual(t, 0, res.exitCode)
			assert.Contains(t, res.stderr, "glx add does not support single-file archives yet")
			assert.Contains(t, res.stderr, "glx split")
			assert.NotContains(t, res.stderr, "not a directory")
			assertTreeUnchanged(t, before, dir)
		})
	}
}

// The refusal covers every add subcommand, including --dry-run, which must
// not report an entity it could never write.
func TestAdd_SingleFileArchiveRefusedForDryRunAndOtherTypes(t *testing.T) {
	dir, _ := newSingleFileArchive(t)
	before := snapshotTree(t, dir)

	for _, args := range [][]string{
		{"add", "person", "--archive", dir, "--given", "Carl", "--dry-run"},
		{"add", "place", "--archive", dir, "--name", "Springfield"},
		{"add", "source", "--archive", dir, "--title", "Register"},
	} {
		res := runGLX(t, dir, args...)
		assert.NotEqual(t, 0, res.exitCode, strings.Join(args, " "))
		assert.Contains(t, res.stderr, "does not support single-file archives yet", strings.Join(args, " "))
	}
	assertTreeUnchanged(t, before, dir)
}

// A multi-file archive is unaffected by the single-file check.
func TestAdd_MultiFileArchiveStillWorks(t *testing.T) {
	archive := copyExample(t, "basic-family")

	res := runGLX(t, archive, "add", "person", "--given", "Carl", "--surname", "Thompson")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "person-carl-thompson")
}

// Naming the archive as glx duplicates' positional argument, which is a
// person filter, gets a hint pointing at --archive (#719).
func TestDuplicates_ArchiveAsPositionalArgGetsHint(t *testing.T) {
	archive := copyExample(t, "basic-family")
	outside := t.TempDir()

	res := runGLX(t, filepath.Dir(archive), "duplicates", filepath.Base(archive))

	assert.NotEqual(t, 0, res.exitCode)
	assert.Contains(t, res.stderr, "not found in archive")
	assert.Contains(t, res.stderr, "Did you mean: glx duplicates --archive "+filepath.Base(archive))

	// A plain unknown person gets no hint.
	res = runGLX(t, outside, "duplicates", "person-nobody", "--archive", archive)
	assert.NotEqual(t, 0, res.exitCode)
	assert.NotContains(t, res.stderr, "Hint:")
}

// Archive-controlled text reaches the terminal with its control characters
// shown as visible escapes, in runners that print without IOStreams too
// (#1208), while JSON output keeps the archive's value.
func TestReadCommands_TerminalControlCharactersEscaped(t *testing.T) {
	parent := t.TempDir()
	res := runGLX(t, parent, "init", "a", "--no-git")
	require.Equal(t, 0, res.exitCode, res.stderr)
	archive := filepath.Join(parent, "a")

	res = runGLX(t, archive, "add", "person", "--id", "person-x", "--given", "Jane\x1b[2J", "--surname", "Doe")
	require.Equal(t, 0, res.exitCode, res.stderr)

	for _, args := range [][]string{
		{"summary", "person-x"},
		{"query", "persons"},
		{"timeline", "person-x"},
		{"vitals", "person-x"},
	} {
		res := runGLX(t, archive, args...)
		require.Equal(t, 0, res.exitCode, "%s: %s", strings.Join(args, " "), res.stderr)
		assert.NotContains(t, res.stdout, "\x1b", strings.Join(args, " "))
		assert.Contains(t, res.stdout, `Jane\x1b[2J`, strings.Join(args, " "))
	}

	res = runGLX(t, archive, "search", "Jane", "--json")
	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, `Jane\u001b[2J`, "JSON keeps the value, escaped by encoding/json")
}
