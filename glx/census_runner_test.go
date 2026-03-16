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

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadCensusTemplate(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "census.yaml")

	content := `census:
  year: 1860
  location:
    place: "Marion County, Florida"
  household:
    members:
      - name: "Daniel Lane"
        age: 30
        sex: male
        occupation: Farmer
        birthplace: Virginia
`

	require.NoError(t, os.WriteFile(tplPath, []byte(content), 0o644))

	tpl, err := readCensusTemplate(tplPath)
	require.NoError(t, err)

	assert.Equal(t, 1860, tpl.Census.Year)
	assert.Equal(t, "Marion County, Florida", tpl.Census.Location.Place)
	assert.Len(t, tpl.Census.Household.Members, 1)
	assert.Equal(t, "Daniel Lane", tpl.Census.Household.Members[0].Name)
	assert.Equal(t, 30, *tpl.Census.Household.Members[0].Age)
	assert.Equal(t, "male", tpl.Census.Household.Members[0].Sex)
	assert.Equal(t, "Farmer", tpl.Census.Household.Members[0].Occupation)
	assert.Equal(t, "Virginia", tpl.Census.Household.Members[0].Birthplace)
}

func TestReadCensusTemplate_FileNotFound(t *testing.T) {
	_, err := readCensusTemplate("/nonexistent/census.yaml")
	assert.Error(t, err)
}

func TestReadCensusTemplate_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	tplPath := filepath.Join(dir, "bad.yaml")
	require.NoError(t, os.WriteFile(tplPath, []byte("{{not yaml"), 0o644))

	_, err := readCensusTemplate(tplPath)
	assert.Error(t, err)
}

func TestCensusAdd_DryRun(t *testing.T) {
	// Set up a minimal archive
	archiveDir := t.TempDir()

	// Create a person file in the archive
	personDir := filepath.Join(archiveDir, "persons")
	require.NoError(t, os.MkdirAll(personDir, 0o755))

	personData := map[string]map[string]*glxlib.Person{
		"persons": {
			"person-d-lane": {
				Properties: map[string]any{
					"name": "Daniel Lane",
				},
			},
		},
	}
	personYAML, err := yaml.Marshal(personData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(personDir, "person-test.glx"), personYAML, 0o644))

	// Create template
	tplDir := t.TempDir()
	tplPath := filepath.Join(tplDir, "census.yaml")
	tplContent := `census:
  year: 1860
  location:
    place: "Marion County, Florida"
  household:
    members:
      - name: "Daniel Lane"
        person_id: person-d-lane
        age: 30
`
	require.NoError(t, os.WriteFile(tplPath, []byte(tplContent), 0o644))

	// Run with dry-run — should not write files
	err = censusAdd(tplPath, archiveDir, true, false)
	require.NoError(t, err)

	// Verify no event files were written
	eventDir := filepath.Join(archiveDir, "events")
	_, err = os.Stat(eventDir)
	assert.True(t, os.IsNotExist(err), "dry run should not create events directory")
}

func TestCensusAdd_WritesFiles(t *testing.T) {
	archiveDir := t.TempDir()

	// Create template
	tplDir := t.TempDir()
	tplPath := filepath.Join(tplDir, "census.yaml")
	tplContent := `census:
  year: 1870
  location:
    place: "Sumter County, Florida"
  household:
    members:
      - name: "William Clark"
        age: 45
        sex: male
`
	require.NoError(t, os.WriteFile(tplPath, []byte(tplContent), 0o644))

	err := censusAdd(tplPath, archiveDir, false, false)
	require.NoError(t, err)

	// Verify directories were created
	for _, dir := range []string{"events", "persons", "sources", "citations", "places", "assertions"} {
		dirPath := filepath.Join(archiveDir, dir)
		info, statErr := os.Stat(dirPath)
		assert.NoError(t, statErr, "directory %s should exist", dir)
		if statErr == nil {
			assert.True(t, info.IsDir())
		}
	}
}

func TestWriteCensusEntities_FilterVocabularies(t *testing.T) {
	archiveDir := t.TempDir()

	result := &glxlib.CensusResult{
		Source:     map[string]*glxlib.Source{},
		Citation:   map[string]*glxlib.Citation{},
		Place:      map[string]*glxlib.Place{},
		Event:      map[string]*glxlib.Event{},
		Persons:    map[string]*glxlib.Person{},
		Assertions: map[string]*glxlib.Assertion{},
	}

	result.Persons["person-test"] = &glxlib.Person{
		Properties: map[string]any{"name": "Test Person"},
	}
	result.Event["event-test"] = &glxlib.Event{
		Title: "Test",
		Type:  "census",
		Date:  "1860",
	}

	count, err := writeCensusEntities(archiveDir, result)
	require.NoError(t, err)

	// Should write entity files but not vocabulary files
	assert.Greater(t, count, 0)

	// Vocabularies dir should NOT be created
	vocabDir := filepath.Join(archiveDir, "vocabularies")
	_, err = os.Stat(vocabDir)
	assert.True(t, os.IsNotExist(err), "should not write vocabulary files")
}
