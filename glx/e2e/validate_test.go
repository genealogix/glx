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
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// The reproduction in #1322: an archive with more than ten errors. Validate
// used to print a different ten on every run with no way to see the rest;
// --show-first-errors 0 now lists every one, in the same order each time.
// Each deed uses an event type and a role that no vocabulary defines, so it
// carries exactly two errors.
func TestValidate_ShowFirstErrors(t *testing.T) {
	archive := copyExample(t, "basic-family")
	var deeds strings.Builder
	deeds.WriteString("events:\n")
	for i := range 12 {
		fmt.Fprintf(&deeds, "  ev-deed-%02d:\n    type: not_a_standard_type\n    date: \"1831\"\n"+
			"    participants:\n      - person: person-robert-thompson\n        role: not_a_standard_role\n", i)
	}
	require.NoError(t, os.WriteFile(filepath.Join(archive, "events", "deeds.glx"), []byte(deeds.String()), 0o644))

	first := runGLX(t, archive, "validate")
	assert.Equal(t, 1, first.exitCode, first.stdout+first.stderr)
	assert.Contains(t, first.stderr, "Validation failed: 24 error(s)")
	assert.NotContains(t, first.stderr, "Error loading archive")
	assert.Contains(t, first.stderr, "... and 14 more errors (use --show-first-errors 0 to list all)")

	again := runGLX(t, archive, "validate")
	assert.Equal(t, first.stderr, again.stderr, "the truncated list must be the same on every run")

	all := runGLX(t, archive, "validate", "--show-first-errors", "0")
	assert.Equal(t, 1, all.exitCode, all.stdout+all.stderr)
	assert.Equal(t, 24, strings.Count(all.stderr, "\n  - "))
	assert.NotContains(t, all.stderr, "more errors")
}

// A participant role used in a context its applies_to excludes is a warning,
// not an error (#499): the archive still validates, and the warning names the
// role and the context.
func TestValidate_WarnsOnRoleOutsideAppliesTo(t *testing.T) {
	archive := copyExample(t, "basic-family")
	rel := `relationships:
  rel-possibly-same:
    type: possibly_same_person
    participants:
      - person: person-alice-thompson
        role: subject
      - person: person-mary-thompson
        role: subject
`
	require.NoError(t, os.WriteFile(filepath.Join(archive, "relationships", "rel-possibly-same.glx"), []byte(rel), 0o644))

	res := runGLX(t, archive, "validate", ".")

	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Contains(t, res.stderr, "relationships[rel-possibly-same].participants[0].role: role 'subject' is used on a relationship, but its applies_to is [event]")
	assert.Contains(t, res.stdout, "Archive is valid")
}
