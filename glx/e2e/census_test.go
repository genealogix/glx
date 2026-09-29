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

// census1880 is the Thompson household in the 1880 census: three members
// matched to basic-family persons by ID and one boarder who is new.
const census1880 = `census:
  year: 1880
  location:
    place_id: place-springfield
  citation:
    locator: "Sheet 12, Dwelling 104"
  household:
    members:
      - name: "Robert Thompson"
        person_id: person-robert-thompson
        age: 30
        sex: male
        occupation: Clerk
      - name: "Mary Thompson"
        person_id: person-mary-thompson
        age: 27
        sex: female
      - name: "Alice Thompson"
        person_id: person-alice-thompson
        age: 2
        sex: female
      - name: "Hannah Webb"
        role: boarder
        age: 19
        sex: female
        birthplace: Ohio
`

// writeCensusTemplate writes census1880 outside the archive and returns its path.
func writeCensusTemplate(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "census-1880.yaml")
	require.NoError(t, os.WriteFile(path, []byte(census1880), 0o644))

	return path
}

func TestCensusAdd_FromInsideArchiveRoot(t *testing.T) {
	archive := copyExample(t, "basic-family")
	tpl := writeCensusTemplate(t)
	before := snapshotTree(t, archive)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "census", "add", "--from", tpl)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Wrote 20 entity files")
	assertSameDirectory(t, dirBefore, archive)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed, "census add only adds files")
	assert.Empty(t, diff.removed)
	assert.Contains(t, diff.created, "persons/person-hannah-webb.glx")
	assert.Contains(t, diff.created, "events/event-1880-census-place-springfield-thompson.glx")
	assert.Contains(t, diff.created, "citations/citation-1880-census-place-springfield-thompson.glx")
	assert.Len(t, diff.created, 20)

	// Clean, not merely valid: census output used to write age_at_event as
	// an int against a string-typed property, warning on every member.
	validate := runGLX(t, archive, "validate", ".")
	require.Equal(t, 0, validate.exitCode, validate.stdout+validate.stderr)
	assert.NotContains(t, validate.stderr, "warning")
	assert.NotContains(t, validate.stderr, "age_at_event")
}

func TestCensusAdd_ArchiveFlagFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")
	tpl := writeCensusTemplate(t)

	res := runGLX(t, t.TempDir(), "census", "add", "--from", tpl, "--archive", archive)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(archive, "persons", "person-hannah-webb.glx"))
	assertArchiveValid(t, archive)
}

func TestCensusAdd_DryRunWritesNothing(t *testing.T) {
	archive := copyExample(t, "basic-family")
	tpl := writeCensusTemplate(t)
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "census", "add", "--from", tpl, "--dry-run", "--verbose")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "+ person-hannah-webb")
	assert.Contains(t, res.stdout, "= person-robert-thompson")
	assert.Contains(t, res.stdout, "dry run")
	assertTreeUnchanged(t, before, archive)
}

func TestCensusAdd_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)
	broken := filepath.Join(t.TempDir(), "broken.yaml")
	require.NoError(t, os.WriteFile(broken, []byte(strings.Replace(census1880, "  year: 1880\n", "", 1)), 0o644))

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no template flag", []string{"add"}, `required flag(s) "from" not set`},
		{"missing template", []string{"add", "--from", "does-not-exist.yaml"}, "failed to read template"},
		{"template without year", []string{"add", "--from", broken}, "year"},
		{"stray argument", []string{"add", "extra", "--from", broken}, `unknown command "extra" for "glx census add"`},
		{"unknown subcommand", []string{"bogus"}, `unknown command "bogus" for "glx census"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"census"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestCensusAdd_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "census", "add", "--from", writeCensusTemplate(t), "--archive", "does-not-exist")

	assertExitWithStderr(t, res, "does-not-exist")
}
