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
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A symlink at a visible path into a skipped copy of the archive would read
// back every entity a second time, reproducing the duplicate-ID conflicts the
// skip exists to prevent (#1212). The skip has to govern reads, not only
// traversal.
func TestValidate_IgnoresSymlinkIntoDotDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevation on Windows")
	}
	archive := copyExample(t, "basic-family")

	clean := runGLX(t, archive, "validate", ".")
	require.Equal(t, 0, clean.exitCode, clean.stdout+clean.stderr)
	require.NotContains(t, clean.stdout, "Validated 0 files.")

	copyTreeFollowingSymlinks(t, filepath.Join(examplesDir(t), "basic-family"), filepath.Join(archive, ".worktrees", "copy"))

	entries, err := os.ReadDir(filepath.Join(archive, "persons"))
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	first := entries[0].Name()

	require.NoError(t, os.Symlink(
		filepath.Join("..", ".worktrees", "copy", "persons", first),
		filepath.Join(archive, "persons", "dup-"+first),
	))

	res := runGLX(t, archive, "validate", ".")

	assert.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Equal(t, clean.stdout, res.stdout)
	assert.NotContains(t, res.stderr, "conflict")
}

// --report used to replace validation instead of gating on it, so a CI step
// written as `glx validate . --report` stayed green on an archive that plain
// validate rejects.
func TestValidate_ReportStillValidates(t *testing.T) {
	archive := copyExample(t, "basic-family")

	ok := runGLX(t, archive, "validate", ".", "--report")
	require.Equal(t, 0, ok.exitCode, ok.stdout+ok.stderr)

	require.NoError(t, os.WriteFile(
		filepath.Join(archive, "relationships", "rel-bogus.glx"),
		[]byte("relationships:\n  rel-bogus:\n    type: parent_child\n    person1: person-nobody\n    person2: person-nobody\n"),
		0o644,
	))

	res := runGLX(t, archive, "validate", ".", "--report")

	assert.NotEqual(t, 0, res.exitCode, "an invalid archive must fail with --report as it does without it")
}

// A .glx file under a dot directory nested inside a managed entity directory
// is skipped by the loader, so the serializer never re-emits it. The safe-write
// swap must carry it across rather than delete it with the backup.
func TestMerge_PreservesDotDirectoryInsideEntityDir(t *testing.T) {
	dest := copyExample(t, "basic-family")
	src := copyExample(t, "basic-family")

	draft := filepath.Join(dest, "persons", ".drafts", "person-draft.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(draft), 0o755))
	body := []byte("persons:\n  person-draft:\n    names:\n      - given: Draft\n        surname: Person\n")
	require.NoError(t, os.WriteFile(draft, body, 0o644))

	res := runGLX(t, dest, "merge", src, "--into", ".")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	got, err := os.ReadFile(draft)
	require.NoError(t, err, "the skipped draft must survive the safe-write swap")
	assert.Equal(t, body, got)
}
