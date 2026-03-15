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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffArchives_Integration_AddedEntities(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	// Old archive: empty (just a persons dir with no files)
	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "persons"), 0o755))

	// New archive: one person
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "persons"), 0o755))
	personYAML := `persons:
  person-john:
    properties:
      name: John Smith
      born_on: "1850"
`
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "persons", "person-john.glx"), []byte(personYAML), 0o644))

	err := diffArchives(oldDir, newDir, "", false, false, false)
	assert.NoError(t, err)
}

func TestDiffArchives_Integration_ShortOutput(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "persons"), 0o755))

	personYAML := `persons:
  person-john:
    properties:
      name: John Smith
`
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "persons", "person-john.glx"), []byte(personYAML), 0o644))

	err := diffArchives(oldDir, newDir, "", false, true, false)
	assert.NoError(t, err)
}

func TestDiffArchives_Integration_JSONOutput(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "persons"), 0o755))

	personYAML := `persons:
  person-john:
    properties:
      name: John Smith
`
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "persons", "person-john.glx"), []byte(personYAML), 0o644))

	err := diffArchives(oldDir, newDir, "", false, false, true)
	assert.NoError(t, err)
}

func TestDiffArchives_Integration_VerboseOutput(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	// Old archive: person with one property
	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "persons"), 0o755))
	oldYAML := `persons:
  person-mary:
    properties:
      name: Jane Webb
      born_on: "ABT 1832"
`
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "persons", "person-mary.glx"), []byte(oldYAML), 0o644))

	// New archive: person with updated property
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "persons"), 0o755))
	newYAML := `persons:
  person-mary:
    properties:
      name: Jane Webb
      born_on: "1832"
      died_on: "1905"
`
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "persons", "person-mary.glx"), []byte(newYAML), 0o644))

	err := diffArchives(oldDir, newDir, "", true, false, false)
	assert.NoError(t, err)
}

func TestDiffArchives_Integration_NoChanges(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))

	personYAML := `persons:
  person-john:
    properties:
      name: John Smith
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "person-john.glx"), []byte(personYAML), 0o644))

	// Same directory as both old and new
	err := diffArchives(dir, dir, "", false, false, false)
	assert.NoError(t, err)
}

func TestDiffArchives_Integration_PersonFilter(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(oldDir, "persons"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(newDir, "persons"), 0o755))

	// Two new persons
	for _, id := range []string{"person-mary", "person-john"} {
		yaml := "persons:\n  " + id + ":\n    properties:\n      name: \"Test\"\n"
		require.NoError(t, os.WriteFile(filepath.Join(newDir, "persons", id+".glx"), []byte(yaml), 0o644))
	}

	err := diffArchives(oldDir, newDir, "person-mary", false, false, false)
	assert.NoError(t, err)
}

func TestLoadArchiveForDiff_InvalidPath(t *testing.T) {
	_, err := loadArchiveForDiff("/nonexistent/path")
	assert.Error(t, err)
}
