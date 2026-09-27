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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	linkARK        = "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2"
	linkCitation   = "citations/citation-familysearch-c4h8-2dw2.glx"
	linkRepository = "repositories/repository-familysearch.glx"
)

func TestLink_FromInsideArchiveRoot_ExistingSource(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "link", linkARK, "--source", "source-sangamon-births", "--locator", "Entry 9")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Created citation citation-familysearch-c4h8-2dw2")
	assertSameDirectory(t, dirBefore, archive)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Equal(t, []string{linkCitation, linkRepository}, diff.created)
	assert.Empty(t, diff.removed)

	citation, err := os.ReadFile(filepath.Join(archive, linkCitation))
	require.NoError(t, err)
	assert.Contains(t, string(citation), "source: source-sangamon-births")
	assert.Contains(t, string(citation), "locator: Entry 9")
	assert.Contains(t, string(citation), "url: "+linkARK)
	assertArchiveValid(t, archive)
}

func TestLink_CreateSourceFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, t.TempDir(), "link", "ark:/61903/1:1:C4H8-2DW2",
		"--archive", archive, "--create-source", "FS Index", "--text", "Robert Thompson")

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, []string{linkCitation, linkRepository, "sources/source-fs-index.glx"}, diff.created)
	assertArchiveValid(t, archive)
}

func TestLink_DryRunWritesNothing(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "link", linkARK, "--source", "source-sangamon-births", "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "citation-familysearch-c4h8-2dw2 (new)")
	assert.Contains(t, res.stdout, "dry run")
	assertTreeUnchanged(t, before, archive)
}

// The accessed date is the researcher's calendar day. Kiritimati (UTC+14) and
// Etc/GMT+12 (UTC-12) are 26 hours apart, so their dates always differ: if
// the CLI stamped UTC's date, at most one of these could pass.
func TestLink_AccessedDateIsLocal(t *testing.T) {
	for _, zone := range []string{"Pacific/Kiritimati", "Etc/GMT+12"} {
		t.Run(zone, func(t *testing.T) {
			loc, err := time.LoadLocation(zone)
			if err != nil {
				t.Skipf("time zone database lacks %s: %v", zone, err)
			}
			archive := copyExample(t, "basic-family")

			dayBefore := time.Now().In(loc).Format(time.DateOnly)
			res := runGLXWithEnv(t, []string{"TZ=" + zone}, archive, "link", linkARK, "--source", "source-sangamon-births")
			dayAfter := time.Now().In(loc).Format(time.DateOnly)

			require.Equal(t, 0, res.exitCode, res.stderr)
			citation, err := os.ReadFile(filepath.Join(archive, linkCitation))
			require.NoError(t, err)
			// The two days differ only when the run straddles local midnight.
			got := string(citation)
			assert.True(t,
				strings.Contains(got, `accessed: "`+dayBefore+`"`) || strings.Contains(got, `accessed: "`+dayAfter+`"`),
				"accessed date is not today in %s (%s):\n%s", zone, dayBefore, got)
		})
	}
}

func TestLink_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"not an ARK", []string{"not-an-ark", "--source", "source-sangamon-births"}, "not a valid FamilySearch ARK"},
		{"no source flag", []string{linkARK}, "exactly one of --source or --create-source is required"},
		{"both source flags", []string{linkARK, "--source", "source-sangamon-births", "--create-source", "X"}, "mutually exclusive"},
		{"unknown source", []string{linkARK, "--source", "source-nope"}, "--source not found in archive: source-nope"},
		{"no ARK", []string{"--source", "source-sangamon-births"}, "accepts 1 arg(s), received 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"link"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestLink_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "link", linkARK, "--source", "source-x", "--archive", "does-not-exist")

	assertExitWithStderr(t, res, "does-not-exist")
}
