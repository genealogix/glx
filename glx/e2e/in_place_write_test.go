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

// merge, merge-persons, and migrate write a multi-file archive in place: only
// the files holding what changed are rewritten, where they are, and a new
// entity gets a canonical file of its own (#1247). These cases use layouts the
// specification allows but the canonical writer used to flatten.

// writeFixture writes files (slash-separated relative path -> content) under
// a fresh directory and returns it.
func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relPath, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(relPath))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o755))
		require.NoError(t, os.WriteFile(abs, []byte(content), 0o644)) //nolint:gosec // test fixture
	}

	return root
}

// Before #1247, people/ was not a directory the writer managed, so the dropped
// person survived in family.glx beside a fresh persons/ copy of the kept one,
// and the custom event type was replaced by the standard vocabulary.
func TestMergePersons_HybridLayoutIsRewrittenInPlace(t *testing.T) {
	census := "# 1850 census, transcribed by hand.\nsources:\n  source-1850:\n    title: 1850 Census\n"
	archive := writeFixture(t, map[string]string{
		"people/family.glx": `persons:
  person-a:
    properties:
      name:
        value: Alice Smith
  person-b:
    properties:
      name:
        value: Alice Smith
      occupation: weaver
`,
		"sources/census/1850.glx":    census,
		"vocabularies/my-events.glx": "event_types:\n  barmitzvah:\n    label: Bar Mitzvah\n",
	})
	before := snapshotTree(t, archive)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "merge-persons", "person-a", "person-b", "--yes")

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, []string{"people/family.glx"}, diff.changed)
	assert.Empty(t, diff.created)
	assert.Empty(t, diff.removed)
	assertSameDirectory(t, dirBefore, archive)

	family, err := os.ReadFile(filepath.Join(archive, "people", "family.glx"))
	require.NoError(t, err)
	assert.NotContains(t, string(family), "person-b")
	assert.Contains(t, string(family), "weaver", "the dropped person's property is merged into the kept one")
	assertArchiveValid(t, archive)
}

// A top-level .glx file (the layout of several docs/examples archives) used to
// be refused outright. A merge now leaves it alone and adds the new entity
// beside it.
func TestMerge_IntoArchiveWithTopLevelGLXFile(t *testing.T) {
	dest := copyExample(t, "assertion-workflow")
	src := writeFixture(t, map[string]string{
		"persons/person-new.glx": "persons:\n  person-new:\n    properties:\n      name:\n        value: Newcomer\n",
	})
	before := snapshotTree(t, dest)

	res := runGLX(t, dest, "merge", src)

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, dest))
	assert.Empty(t, diff.changed, "archive.glx and the vocabularies are untouched")
	assert.Equal(t, []string{"persons/person-new.glx"}, diff.created)
	assert.Empty(t, diff.removed)
	assertArchiveValid(t, dest)
}

// migrate rewrites the top-level file that holds the legacy data, and only it.
func TestMigrate_TopLevelGLXFileIsRewrittenInPlace(t *testing.T) {
	archive := copyExample(t, "assertion-workflow")
	path := filepath.Join(archive, "archive.glx")
	data, err := os.ReadFile(path) //nolint:gosec // test fixture path
	require.NoError(t, err)
	legacy := strings.ReplaceAll(string(data), "      sex: ", "      gender: ")
	require.NotEqual(t, string(data), legacy, "the example has no sex property to rewrite")
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o644)) //nolint:gosec // test fixture
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "migrate", ".", "--rename-gender-to-sex")

	require.Equal(t, 0, res.exitCode, res.stderr)
	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, []string{"archive.glx"}, diff.changed)
	assert.Empty(t, diff.created, "no canonical entity directories appear")
	assert.Empty(t, diff.removed)
	migrated, err := os.ReadFile(path) //nolint:gosec // test fixture path
	require.NoError(t, err)
	assert.NotContains(t, string(migrated), "gender:")
	assertArchiveValid(t, archive)
}
