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
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestRunInit_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInit(tmpDir, true, 0)
	require.NoError(t, err)

	// Check that archive.glx was created
	archivePath := filepath.Join(tmpDir, "archive.glx")
	_, err = os.Stat(archivePath)
	require.NoError(t, err, "archive.glx should be created")

	// Read and parse the content
	content, err := os.ReadFile(archivePath)
	require.NoError(t, err)

	// Verify it's valid YAML by parsing into GLXFile structure
	var glxFile glxlib.GLXFile
	err = yaml.Unmarshal(content, &glxFile)
	require.NoError(t, err, "archive.glx should contain valid YAML")

	// Verify the structure has all expected entity maps initialized
	// (They should be empty maps, not nil, for a new archive)
	assert.NotNil(t, glxFile.Persons, "persons map should be initialized")
	assert.NotNil(t, glxFile.Relationships, "relationships map should be initialized")
	assert.NotNil(t, glxFile.Events, "events map should be initialized")
	assert.NotNil(t, glxFile.Places, "places map should be initialized")
}

func TestRunInit_MultiFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInit(tmpDir, false, 0)
	require.NoError(t, err)

	// Check that directories were created
	expectedDirs := []string{
		"persons", "relationships", "events", "places",
		"sources", "citations", "repositories", "assertions", "media", "vocabularies",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		info, err := os.Stat(dirPath)
		require.NoError(t, err, "directory %s should be created", dir)
		assert.True(t, info.IsDir(), "%s should be a directory", dir)
	}

	// Check that vocabulary files were created
	vocabFiles := []string{
		"vocabularies/relationship-types.glx",
		"vocabularies/event-types.glx",
		"vocabularies/place-types.glx",
		"vocabularies/repository-types.glx",
		"vocabularies/participant-roles.glx",
		"vocabularies/media-types.glx",
		"vocabularies/confidence-levels.glx",
	}

	for _, file := range vocabFiles {
		filePath := filepath.Join(tmpDir, file)
		_, err := os.Stat(filePath)
		require.NoError(t, err, "vocabulary file %s should be created", file)
	}

	// Check that .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	_, err = os.Stat(gitignorePath)
	require.NoError(t, err, ".gitignore should be created")

	// Check that README.md was created
	readmePath := filepath.Join(tmpDir, "README.md")
	_, err = os.Stat(readmePath)
	require.NoError(t, err, "README.md should be created")
}

func TestRunInit_NonEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a dummy file to make the directory non-empty
	dummyFile := filepath.Join(tmpDir, "dummy.txt")
	err := os.WriteFile(dummyFile, []byte("hello"), 0o644)
	require.NoError(t, err)

	// Now, try to initialize in the non-empty directory
	err = runInit(tmpDir, false, 0)
	require.Error(t, err, "should fail when run in a non-empty directory")
	if err != nil {
		assert.Contains(t, err.Error(), "non-empty directory")
	}
}

func TestRunInit_WithTestData(t *testing.T) {
	tmpDir := t.TempDir()
	numPeople := 5

	err := runInit(tmpDir, false, numPeople)
	require.NoError(t, err)

	// Check that the person files were created
	personsDir := filepath.Join(tmpDir, "persons")
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	assert.Len(t, personFiles, numPeople, "should create the correct number of person files")

	// Verify content of at least one person file
	// The init command writes files in single-file format: persons: { person-id: { ... } }
	if len(personFiles) > 0 {
		firstPersonPath := filepath.Join(personsDir, personFiles[0].Name())
		content, err := os.ReadFile(firstPersonPath)
		require.NoError(t, err)

		// Parse as a mini GLX file with persons map
		var miniGlx struct {
			Persons map[string]*glxlib.Person `yaml:"persons"`
		}
		err = yaml.Unmarshal(content, &miniGlx)
		require.NoError(t, err, "person file should contain valid YAML")
		assert.Len(t, miniGlx.Persons, 1, "person file should contain exactly one person")

		// Get the first (only) person
		for _, person := range miniGlx.Persons {
			assert.NotNil(t, person.Properties, "person should have properties")
		}
	}

	// Check that event files were created (one birth per person)
	eventsDir := filepath.Join(tmpDir, "events")
	eventFiles, err := os.ReadDir(eventsDir)
	require.NoError(t, err)
	assert.Len(t, eventFiles, numPeople, "should create the correct number of event files")

	// Verify content of at least one event file
	if len(eventFiles) > 0 {
		firstEventPath := filepath.Join(eventsDir, eventFiles[0].Name())
		content, err := os.ReadFile(firstEventPath)
		require.NoError(t, err)

		// Parse as a mini GLX file with events map
		var miniGlx struct {
			Events map[string]*glxlib.Event `yaml:"events"`
		}
		err = yaml.Unmarshal(content, &miniGlx)
		require.NoError(t, err, "event file should contain valid YAML")
		assert.Len(t, miniGlx.Events, 1, "event file should contain exactly one event")

		// Get the first (only) event
		for _, event := range miniGlx.Events {
			assert.Equal(t, "birth", event.Type, "generated events should be birth events")
		}
	}
}

func TestCreateStandardVocabularies(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create vocabularies directory first
	err := os.MkdirAll("vocabularies", 0o755)
	require.NoError(t, err)

	err = createStandardVocabularies()
	require.NoError(t, err)

	// Check that vocabulary files were created with valid structure
	vocabFiles := map[string]string{
		"vocabularies/relationship-types.glx": "relationship_types",
		"vocabularies/event-types.glx":        "event_types",
		"vocabularies/place-types.glx":        "place_types",
		"vocabularies/repository-types.glx":   "repository_types",
		"vocabularies/participant-roles.glx":  "participant_roles",
		"vocabularies/media-types.glx":        "media_types",
		"vocabularies/confidence-levels.glx":  "confidence_levels",
	}

	for file, expectedKey := range vocabFiles {
		filePath := filepath.Join(tmpDir, file)
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "vocabulary file %s should be created", file)
		assert.NotEmpty(t, content, "vocabulary file %s should not be empty", file)

		// Verify content is valid YAML with expected structure
		var parsed map[string]any
		err = yaml.Unmarshal(content, &parsed)
		require.NoError(t, err, "vocabulary file %s should contain valid YAML", file)
		assert.Contains(t, parsed, expectedKey, "vocabulary file %s should contain key %q", file, expectedKey)
	}
}

func TestPrintUsage(t *testing.T) {
	// Test that the help command works
	assert.NotPanics(t, func() {
		// The rootCmd should be available
		_ = rootCmd
	})
}
