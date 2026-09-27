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
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var gedcomIndividual = regexp.MustCompile(`(?m)^0 @[^@]+@ INDI`)

// Exporting to each GEDCOM version and importing the result back yields the
// same four people in an archive that validates. The export never touches
// the archive it reads.
func TestExport_GEDCOMRoundTrip(t *testing.T) {
	for _, tc := range []struct{ format, vers string }{
		{"551", "2 VERS 5.5.1"},
		{"70", "2 VERS 7.0"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			archive := copyExample(t, "basic-family")
			before := snapshotTree(t, archive)
			work := t.TempDir()

			res := runGLX(t, work, "export", archive, "-o", "family.ged", "--format", tc.format)

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Contains(t, res.stdout, "Persons:       4")
			assertTreeUnchanged(t, before, archive)
			ged, err := os.ReadFile(filepath.Join(work, "family.ged"))
			require.NoError(t, err)
			assert.Contains(t, string(ged), tc.vers)
			assert.Len(t, gedcomIndividual.FindAllIndex(ged, -1), 4)

			back := runGLX(t, work, "import", "family.ged", "-o", "back")
			require.Equal(t, 0, back.exitCode, back.stderr)
			persons, err := filepath.Glob(filepath.Join(work, "back", "persons", "*.glx"))
			require.NoError(t, err)
			assert.Len(t, persons, 4)
			assertArchiveValid(t, filepath.Join(work, "back"))
		})
	}
}

// The default archive argument is spelled out; export has no implicit ".".
func TestExport_FromInsideArchiveRoot(t *testing.T) {
	archive := copyExample(t, "basic-family")
	out := filepath.Join(t.TempDir(), "family.ged")

	res := runGLX(t, archive, "export", ".", "-o", out)

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.FileExists(t, out)
}

func TestExport_JSONLD(t *testing.T) {
	archive := copyExample(t, "basic-family")
	out := filepath.Join(t.TempDir(), "family.jsonld")

	res := runGLX(t, archive, "export", ".", "-o", out, "--format", "jsonld")

	require.Equal(t, 0, res.exitCode, res.stderr)
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc))
	assert.Contains(t, doc, "@context")
	assert.Contains(t, string(data), "Robert Thompson")
}

func TestExport_SingleFileArchive(t *testing.T) {
	archive := copyExample(t, "single-file")
	out := filepath.Join(t.TempDir(), "family.ged")

	res := runGLX(t, archive, "export", "archive.glx", "-o", out)

	require.Equal(t, 0, res.exitCode, res.stderr)
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.NotEmpty(t, gedcomIndividual.FindAllIndex(data, -1))
}

func TestExport_PrivatizeLiving(t *testing.T) {
	archive := copyExample(t, "basic-family")
	require.Equal(t, 0, runGLX(t, archive, "add", "person", "--given", "Recent", "--surname", "Person").exitCode)
	require.Equal(t, 0, runGLX(t, archive, "add", "event", "--type", "birth", "--date", "2001-05-01", "--principal", "person-recent-person").exitCode)
	work := t.TempDir()

	plain := runGLX(t, archive, "export", ".", "-o", filepath.Join(work, "plain.ged"))
	require.Equal(t, 0, plain.exitCode, plain.stderr)
	plainGed, err := os.ReadFile(filepath.Join(work, "plain.ged"))
	require.NoError(t, err)
	require.Contains(t, string(plainGed), "Recent", "control: the person is exported without --privatize-living")

	res := runGLX(t, archive, "export", ".", "-o", filepath.Join(work, "private.ged"), "--privatize-living")

	require.Equal(t, 0, res.exitCode, res.stderr)
	ged, err := os.ReadFile(filepath.Join(work, "private.ged"))
	require.NoError(t, err)
	assert.NotContains(t, string(ged), "Recent")
	assert.NotContains(t, string(ged), "2001")
	assert.Contains(t, string(ged), "Robert", "people born over a century ago are exported")
}

func TestExport_Errors(t *testing.T) {
	archive := copyExample(t, "basic-family")
	work := t.TempDir()

	assertExitWithStderr(t, runGLX(t, work, "export", archive), `required flag(s) "output" not set`)
	assertExitWithStderr(t, runGLX(t, work, "export", "does-not-exist", "-o", "x.ged"), "input path not found")
	assertExitWithStderr(t, runGLX(t, work, "export", archive, "-o", "x.ged", "--format", "bogus"), "use '551', '70', or 'jsonld'")
	assertExitWithStderr(t, runGLX(t, work, "export", "-o", "x.ged"), "accepts 1 arg(s), received 0")
	entries, err := os.ReadDir(work)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Empty(t, names, "a failed export must not leave output behind: %s", strings.Join(names, ", "))
}
