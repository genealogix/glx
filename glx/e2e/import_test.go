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

// gedcomFixture returns the path of a file under glx/testdata/gedcom.
func gedcomFixture(t *testing.T, rel string) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	path := filepath.Join(wd, "..", "testdata", "gedcom", filepath.FromSlash(rel))
	require.FileExists(t, path)

	return path
}

// Fixtures that import to an archive `glx validate` accepts, with their
// person counts. Files known to import invalid (#1243) are deliberately not
// here.
var importFixtures = []struct {
	rel     string
	persons int
}{
	{"5.5.1/kennedy-family/kennedy.ged", 70},
	{"5.5.1/famous-people/bronte.ged", 14},
	{"7.0/remarriage/remarriage1.ged", 3},
	{"7.0/remarriage/remarriage2.ged", 3},
	{"7.0/minimal-valid/minimal70.gdz", 0},
}

func TestImport_MultiFileArchivesValidate(t *testing.T) {
	for _, fx := range importFixtures {
		t.Run(fx.rel, func(t *testing.T) {
			parent := t.TempDir()

			res := runGLX(t, parent, "import", gedcomFixture(t, fx.rel), "-o", "archive")

			require.Equal(t, 0, res.exitCode, res.stderr)
			archive := filepath.Join(parent, "archive")
			persons, err := filepath.Glob(filepath.Join(archive, "persons", "*.glx"))
			require.NoError(t, err)
			assert.Len(t, persons, fx.persons)
			assertArchiveValid(t, archive)
		})
	}
}

// Both remarriage encodings describe the same three people; the second puts
// the couple's remarriage in a separate family, so it yields one more
// relationship.
func TestImport_RemarriageEncodingsAgreeOnPeople(t *testing.T) {
	parent := t.TempDir()
	one := runGLX(t, parent, "import", gedcomFixture(t, "7.0/remarriage/remarriage1.ged"), "-o", "one")
	require.Equal(t, 0, one.exitCode, one.stderr)
	two := runGLX(t, parent, "import", gedcomFixture(t, "7.0/remarriage/remarriage2.ged"), "-o", "two")
	require.Equal(t, 0, two.exitCode, two.stderr)

	names := func(dir string) []string {
		t.Helper()
		files, err := filepath.Glob(filepath.Join(parent, dir, "persons", "*.glx"))
		require.NoError(t, err)
		out := make([]string, 0, len(files))
		for _, f := range files {
			out = append(out, filepath.Base(f))
		}

		return out
	}
	assert.Equal(t, names("one"), names("two"))
	rels := func(dir string) int {
		t.Helper()
		files, err := filepath.Glob(filepath.Join(parent, dir, "relationships", "*.glx"))
		require.NoError(t, err)

		return len(files)
	}
	assert.Equal(t, rels("one")+1, rels("two"))
}

func TestImport_SingleFileFormat(t *testing.T) {
	parent := t.TempDir()

	res := runGLX(t, parent, "import", gedcomFixture(t, "5.5.1/famous-people/bronte.ged"), "-o", "bronte.glx", "--format", "single")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Equal(t, []string{"bronte.glx"}, treePaths(t, parent))
	validate := runGLX(t, parent, "validate", "bronte.glx")
	assert.Equal(t, 0, validate.exitCode, validate.stdout+validate.stderr)
}

// Media records whose files do not exist import as Media entities without
// requiring the binaries.
func TestImport_MediaWithoutBinaries(t *testing.T) {
	parent := t.TempDir()

	res := runGLX(t, parent, "import", gedcomFixture(t, "7.0/media-objects/obje-1.ged"), "-o", "archive")

	require.Equal(t, 0, res.exitCode, res.stderr)
	media, err := filepath.Glob(filepath.Join(parent, "archive", "media", "*.glx"))
	require.NoError(t, err)
	assert.NotEmpty(t, media)
	assertArchiveValid(t, filepath.Join(parent, "archive"))
}

func TestImport_Errors(t *testing.T) {
	parent := t.TempDir()
	ged := gedcomFixture(t, "5.5.1/famous-people/bronte.ged")

	assertExitWithStderr(t, runGLX(t, parent, "import", ged), `required flag(s) "output" not set`)
	assertExitWithStderr(t, runGLX(t, parent, "import", "does-not-exist.ged", "-o", "x"), "GEDCOM file not found")
	assertExitWithStderr(t, runGLX(t, parent, "import", ged, "-o", "x", "--format", "bogus"), "invalid format")
	assertExitWithStderr(t, runGLX(t, parent, "import", "-o", "x"), "accepts 1 arg(s), received 0")
	entries, err := os.ReadDir(parent)
	require.NoError(t, err)
	assert.Empty(t, entries, "a failed import must not leave output behind")
}
