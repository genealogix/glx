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
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImport_EmbeddedSourceDeduplication(t *testing.T) {
	// Repeated Ancestry-style facts should share a placeholder, while every
	// citation and event is still serialized and the import summary stays honest.
	for _, extension := range []string{"ged", "gdz"} {
		t.Run(extension, func(t *testing.T) {
			work := t.TempDir()
			gedcom := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @S1@ SOUR\n1 TITL Real register\n0 @I1@ INDI\n1 NAME John /Smith/\n" +
				strings.Repeat("1 BIRT\n2 DATE 1 JAN 1850\n2 SOUR\n3 PAGE Entry 1\n", 100) + "0 TRLR\n"
			data := []byte(gedcom)
			if extension == "gdz" {
				var bundle bytes.Buffer
				writer := zip.NewWriter(&bundle)
				entry, err := writer.Create("gedcom.ged")
				require.NoError(t, err)
				_, err = entry.Write(data)
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				data = bundle.Bytes()
			}
			src := filepath.Join(work, "inline."+extension)
			require.NoError(t, os.WriteFile(src, data, 0o600))

			res := runGLX(t, work, "import", src, "-o", "archive")
			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Regexp(t, `(?m)^\s*Sources:\s+2\s*$`, res.stdout)
			for collection, count := range map[string]int{"sources": 2, "events": 100} {
				files, err := filepath.Glob(filepath.Join(work, "archive", collection, "*.glx"))
				require.NoError(t, err)
				assert.Len(t, files, count, collection)
			}
			citations, err := filepath.Glob(filepath.Join(work, "archive", "citations", "*.glx"))
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(citations), 100, "every inline occurrence must retain citation detail")
			assertArchiveValid(t, filepath.Join(work, "archive"))
		})
	}
}

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
	{"7.0/media-objects/obje-1.ged", 1},
	{"7.0/media-objects/filename-1.ged", 0},
	{"7.0/extensions/extension-record.ged", 1},
	{"7.0/long-url/long-url.ged", 0},
}

// Every fixture survives import → export → re-import through the binary, in
// both GEDCOM versions, with the same people on the far side. Content the
// converter is known to lose in transit is tracked separately (#1307, #1308)
// rather than asserted here.
func TestImport_RoundTripThroughExport(t *testing.T) {
	for _, fx := range importFixtures {
		for _, format := range []string{"551", "70"} {
			t.Run(fx.rel+"/"+format, func(t *testing.T) {
				work := t.TempDir()
				require.Equal(t, 0, runGLX(t, work, "import", gedcomFixture(t, fx.rel), "-o", "first").exitCode)

				exp := runGLX(t, work, "export", "first", "-o", "out.ged", "--format", format)
				require.Equal(t, 0, exp.exitCode, exp.stderr)
				back := runGLX(t, work, "import", "out.ged", "-o", "back")
				require.Equal(t, 0, back.exitCode, back.stderr)

				persons, err := filepath.Glob(filepath.Join(work, "back", "persons", "*.glx"))
				require.NoError(t, err)
				assert.Len(t, persons, fx.persons)
				assertArchiveValid(t, filepath.Join(work, "back"))
			})
		}
	}
}

// An undocumented _LOC extension record, referenced from a PLAC and nested
// town → country, imports as a two-level place hierarchy that the birth
// event points into.
func TestImport_ExtensionRecordBecomesPlaceHierarchy(t *testing.T) {
	work := t.TempDir()

	res := runGLX(t, work, "import", gedcomFixture(t, "7.0/extensions/extension-record.ged"), "-o", "archive")

	require.Equal(t, 0, res.exitCode, res.stderr)
	places, err := filepath.Glob(filepath.Join(work, "archive", "places", "*.glx"))
	require.NoError(t, err)
	assert.Len(t, places, 2, "the town and the country")
	births := runGLX(t, filepath.Join(work, "archive"), "query", "events", "--type", "birth")
	require.Equal(t, 0, births.exitCode, births.stderr)
	assert.Contains(t, births.stdout, "1 event(s) found")
	analysis := runGLX(t, filepath.Join(work, "archive"), "places")
	require.Equal(t, 0, analysis.exitCode, analysis.stderr)
	assert.Contains(t, analysis.stdout, "Place analysis: 2 places")
}

// GEDCOM 7.0 dropped 5.5.1's 255-character line limit, so a ~800-character
// WWW must arrive whole. (Export currently drops the submitter entirely: #1307.)
func TestImport_LongLineArrivesWhole(t *testing.T) {
	work := t.TempDir()
	src := gedcomFixture(t, "7.0/long-url/long-url.ged")
	data, err := os.ReadFile(src) //nolint:gosec // test fixture path
	require.NoError(t, err)
	_, after, found := strings.Cut(string(data), "1 WWW ")
	require.True(t, found)
	url := strings.TrimSpace(strings.SplitN(after, "\n", 2)[0])
	require.Greater(t, len(url), 255, "the fixture must exceed the 5.5.1 line limit to test anything")

	res := runGLX(t, work, "import", src, "-o", "archive")

	require.Equal(t, 0, res.exitCode, res.stderr)
	meta, err := os.ReadFile(filepath.Join(work, "archive", "metadata.glx"))
	require.NoError(t, err)
	assert.Contains(t, string(meta), url, "the long URL must not be truncated or split")
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
// requiring the binaries, whatever URI shape the FILE payload takes. Each OBJE
// currently yields one entity however many FILEs it holds; #1308 tracks the
// dropped ones, and this count changes when that is fixed.
func TestImport_MediaWithoutBinaries(t *testing.T) {
	for _, tc := range []struct {
		rel   string
		media int
	}{
		{"7.0/media-objects/obje-1.ged", 2},
		{"7.0/media-objects/filename-1.ged", 1},
	} {
		t.Run(tc.rel, func(t *testing.T) {
			parent := t.TempDir()

			res := runGLX(t, parent, "import", gedcomFixture(t, tc.rel), "-o", "archive")

			require.Equal(t, 0, res.exitCode, res.stderr)
			media, err := filepath.Glob(filepath.Join(parent, "archive", "media", "*.glx"))
			require.NoError(t, err)
			assert.Len(t, media, tc.media)
			binaries, err := filepath.Glob(filepath.Join(parent, "archive", "media", "files", "*"))
			require.NoError(t, err)
			assert.Empty(t, binaries, "no binaries exist to copy, so none may be invented")
			assertArchiveValid(t, filepath.Join(parent, "archive"))
		})
	}
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
