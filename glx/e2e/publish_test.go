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

var publishedPersonPages = []string{
	"persons/person-alice-thompson.html",
	"persons/person-mary-thompson.html",
	"persons/person-robert-thompson-jr.html",
	"persons/person-robert-thompson.html",
}

// With default flags from inside the archive, publish writes ./site and
// touches no archive file. The site directory must not break the archive:
// it is not archive content, so validate still passes with it in place.
func TestPublish_FromInsideArchiveRoot_DefaultFlags(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "publish")

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Empty(t, diff.changed)
	assert.Empty(t, diff.removed)
	for _, path := range diff.created {
		assert.True(t, strings.HasPrefix(path, "site/"), "publish wrote %s outside site/", path)
	}
	for _, page := range append([]string{"index.html", "search.html", "places/index.html", "sources/index.html"}, publishedPersonPages...) {
		assert.Contains(t, diff.created, "site/"+page)
	}
	assertArchiveValid(t, archive)
}

func TestPublish_ArchiveAndOutputFlagsFromOutside(t *testing.T) {
	archive := copyExample(t, "basic-family")
	work := t.TempDir()

	res := runGLX(t, work, "publish", "--archive", archive, "--output", "public", "--title", "The Thompson Family")

	require.Equal(t, 0, res.exitCode, res.stderr)
	index, err := os.ReadFile(filepath.Join(work, "public", "index.html"))
	require.NoError(t, err)
	assert.Contains(t, string(index), "The Thompson Family")
	assert.NoDirExists(t, filepath.Join(archive, "site"), "--output decides where the site goes")
}

func TestPublish_SingleFileArchive(t *testing.T) {
	archive := copyExample(t, "single-file")
	out := filepath.Join(t.TempDir(), "site")

	res := runGLX(t, archive, "publish", "--archive", "archive.glx", "--output", out)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(out, "index.html"))
}

// --living redacts anyone born under 100 years ago with no death recorded.
// Everyone in basic-family was born 1850–1881, so only the added person
// qualifies.
func TestPublish_LivingRedactsRecentPeople(t *testing.T) {
	archive := copyExample(t, "basic-family")
	require.Equal(t, 0, runGLX(t, archive, "add", "person", "--given", "Recent", "--surname", "Person").exitCode)
	require.Equal(t, 0, runGLX(t, archive, "add", "event", "--type", "birth", "--date", "2001-05-01", "--principal", "person-recent-person").exitCode)
	siteText := func(out string) string {
		t.Helper()
		var all strings.Builder
		for _, data := range snapshotTree(t, out) {
			all.Write(data)
		}

		return all.String()
	}

	// Control: without --living the person is published, so the assertions
	// below are about redaction rather than a page that never existed.
	plainOut := filepath.Join(t.TempDir(), "site")
	require.Equal(t, 0, runGLX(t, archive, "publish", "--output", plainOut).exitCode)
	require.Contains(t, siteText(plainOut), "Recent Person")

	out := filepath.Join(t.TempDir(), "site")
	res := runGLX(t, archive, "publish", "--output", out, "--living")

	require.Equal(t, 0, res.exitCode, res.stderr)
	redacted := siteText(out)
	assert.NotContains(t, redacted, "Recent Person", "a living person's name must not appear anywhere in the site")
	assert.NotContains(t, redacted, "2001-05-01")
	assert.Contains(t, redacted, "Robert Thompson", "people born over a century ago are published")
}

func TestPublish_RepublishOverExistingSite(t *testing.T) {
	archive := copyExample(t, "basic-family")
	require.Equal(t, 0, runGLX(t, archive, "publish").exitCode)

	res := runGLX(t, archive, "publish")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, filepath.Join(archive, "site", "index.html"))
}

func TestPublish_Errors(t *testing.T) {
	work := t.TempDir()

	assertExitWithStderr(t, runGLX(t, work, "publish", "--archive", "does-not-exist", "--output", "x"), "input path not found")
	assertExitWithStderr(t, runGLX(t, work, "publish", "extra"), `unknown command "extra" for "glx publish"`)
	entries, err := os.ReadDir(work)
	require.NoError(t, err)
	assert.Empty(t, entries, "a failed publish must not leave output behind")
}

// A date range or qualifier must survive into the one-line lifespan shown in
// the page header, family lists, and charts instead of collapsing to its first
// year (#1328).
func TestPublish_LifespanKeepsDateRanges(t *testing.T) {
	work := t.TempDir()
	archive := `persons:
  person-lewis:
    properties: {name: {value: "Lewis Little"}, sex: male}
  person-ann:
    properties: {name: {value: "Ann Little"}, sex: female}
events:
  ev-birth-lewis:
    type: birth
    date: "BET 1756 AND 1774"
    participants: [{person: person-lewis, role: principal}]
  ev-death-lewis:
    type: death
    date: "BET 1826-05-16 AND 1830"
    participants: [{person: person-lewis, role: principal}]
  ev-birth-ann:
    type: birth
    date: "ABT 1765"
    participants: [{person: person-ann, role: principal}]
relationships:
  rel-marriage:
    type: marriage
    participants: [{person: person-lewis, role: spouse}, {person: person-ann, role: spouse}]
`
	require.NoError(t, os.WriteFile(filepath.Join(work, "archive.glx"), []byte(archive), 0o644))

	res := runGLX(t, work, "publish", "--archive", "archive.glx", "--output", "site")

	require.Equal(t, 0, res.exitCode, res.stderr)
	lewis, err := os.ReadFile(filepath.Join(work, "site", "persons", "person-lewis.html"))
	require.NoError(t, err)
	assert.Contains(t, string(lewis), `<p class="lifespan">1756/1774 – 1826/1830</p>`)
	assert.Contains(t, string(lewis), "(b. c. 1765)", "the spouse's qualified birth year is kept")
	assert.NotContains(t, string(lewis), "1756–1826")
}
