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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Helper function to create a test single-file GLX archive
func createTestSingleFileArchive(t *testing.T, tmpDir string) string {
	t.Helper()

	// Create a GLX file with some test data
	glxFile := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {
				Properties: map[string]any{
					"primary_name": "Test Person 1",
				},
			},
			"person-2": {
				Properties: map[string]any{
					"primary_name": "Test Person 2",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {
				Type: "birth",
			},
		},
		EventTypes: map[string]*glxlib.EventType{
			"birth": {Label: "Birth"},
			"death": {Label: "Death"},
		},
	}

	// Serialize to single file
	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{
		Validate: false,
		Pretty:   true,
	})

	data, err := serializer.SerializeSingleFileBytes(glxFile)
	require.NoError(t, err)

	// Write to file
	inputPath := filepath.Join(tmpDir, "test.glx")
	err = os.WriteFile(inputPath, data, 0o644)
	require.NoError(t, err)

	return inputPath
}

func TestSplitArchive_Success(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully split archive")

	// Verify output directory was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir(), "output should be a directory")

	// Verify expected directories exist
	personsDir := filepath.Join(outputDir, "persons")
	_, err = os.Stat(personsDir)
	require.NoError(t, err, "persons directory should exist")

	eventsDir := filepath.Join(outputDir, "events")
	_, err = os.Stat(eventsDir)
	require.NoError(t, err, "events directory should exist")

	// Verify person files were created
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	require.Len(t, personFiles, 2, "should have 2 person files")

	// Verify event files were created
	eventFiles, err := os.ReadDir(eventsDir)
	require.NoError(t, err)
	require.Len(t, eventFiles, 1, "should have 1 event file")

	// Verify vocabularies directory exists
	vocabDir := filepath.Join(outputDir, "vocabularies")
	_, err = os.Stat(vocabDir)
	require.NoError(t, err, "vocabularies directory should exist")
}

func TestSplitArchive_InputFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "nonexistent.glx")
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail when input file doesn't exist")
	require.ErrorIs(t, err, ErrInputFileNotFound)
}

func TestSplitArchive_OutputDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	// Create output directory first
	err := os.MkdirAll(outputDir, 0o755)
	require.NoError(t, err)

	splitNoValidate = false
	splitVerbose = false

	err = splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail when output directory already exists")
	require.ErrorIs(t, err, ErrOutputDirectoryExists)
}

func TestSplitArchive_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid YAML file
	inputPath := filepath.Join(tmpDir, "invalid.glx")
	err := os.WriteFile(inputPath, []byte("invalid: [unclosed"), 0o644)
	require.NoError(t, err)

	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err = splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail with invalid YAML")
	require.Contains(t, err.Error(), "failed to load archive")
}

func TestSplitArchive_NoValidate(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = true // Skip validation
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully split with --no-validate")

	// Verify output was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir())
}

func TestSplitArchive_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = true // Enable verbose output

	// Verbose mode prints to stdout, shouldn't affect functionality
	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully split with --verbose")

	info, err := os.Stat(outputDir)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir())
}

func TestSplitArchive_CreatesNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "nested", "path", "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should create nested output directories")

	// Verify nested directories were created
	info, err := os.Stat(outputDir)
	require.NoError(t, err, "nested output directory should exist")
	require.True(t, info.IsDir())
}

func TestSplitArchive_PreservesEntityData(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Read one of the person files and verify data
	personsDir := filepath.Join(outputDir, "persons")
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	require.NotEmpty(t, personFiles)

	// Read the first person file
	firstPersonPath := filepath.Join(personsDir, personFiles[0].Name())
	data, err := os.ReadFile(firstPersonPath)
	require.NoError(t, err)

	// Verify it contains person data
	require.Contains(t, string(data), "primary_name")
	require.Contains(t, string(data), "Test Person")
}

func TestSplitArchive_IncludesVocabularies(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Verify vocabularies directory exists and contains files
	vocabDir := filepath.Join(outputDir, "vocabularies")
	_, err = os.Stat(vocabDir)
	require.NoError(t, err, "vocabularies directory should exist")

	vocabFiles, err := os.ReadDir(vocabDir)
	require.NoError(t, err)
	require.NotEmpty(t, vocabFiles, "vocabularies directory should contain files")
}

func TestSplitArchive_AllEntityTypes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a more comprehensive GLX file
	glxFile := &glxlib.GLXFile{
		Persons:       map[string]*glxlib.Person{"person-1": {}},
		Events:        map[string]*glxlib.Event{"event-1": {}},
		Relationships: map[string]*glxlib.Relationship{"rel-1": {}},
		Places:        map[string]*glxlib.Place{"place-1": {}},
		Sources:       map[string]*glxlib.Source{"source-1": {}},
		Citations:     map[string]*glxlib.Citation{"citation-1": {}},
		Repositories:  map[string]*glxlib.Repository{"repo-1": {}},
		Assertions: map[string]*glxlib.Assertion{"assertion-1": {
			Subject: glxlib.EntityRef{Person: "person-1"},
			Sources: []string{"source-1"},
		}},
		Media: map[string]*glxlib.Media{"media-1": {}},
	}

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{
		Validate: false,
		Pretty:   true,
	})

	data, err := serializer.SerializeSingleFileBytes(glxFile)
	require.NoError(t, err)

	inputPath := filepath.Join(tmpDir, "test.glx")
	err = os.WriteFile(inputPath, data, 0o644)
	require.NoError(t, err)

	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err = splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Verify all entity directories were created
	entityDirs := []string{
		"persons", "events", "relationships", "places",
		"sources", "citations", "repositories", "assertions", "media",
	}

	for _, dir := range entityDirs {
		dirPath := filepath.Join(outputDir, dir)
		_, err := os.Stat(dirPath)
		require.NoError(t, err, "%s directory should exist", dir)
	}
}

func TestRunSplit(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := runSplit(nil, []string{inputPath, outputDir})
	require.NoError(t, err, "runSplit should successfully call splitArchive")

	// Verify output was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir())
}

func TestSplitArchive_EmptyArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty GLX file
	glxFile := &glxlib.GLXFile{}

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{
		Validate: false,
		Pretty:   true,
	})

	data, err := serializer.SerializeSingleFileBytes(glxFile)
	require.NoError(t, err)

	inputPath := filepath.Join(tmpDir, "empty.glx")
	err = os.WriteFile(inputPath, data, 0o644)
	require.NoError(t, err)

	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err = splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully split empty archive")

	// Output directory should still be created
	info, err := os.Stat(outputDir)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir())
}

func TestSplitArchive_LargeArchive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a larger archive with multiple entities
	glxFile := &glxlib.GLXFile{
		Persons: make(map[string]*glxlib.Person),
		Events:  make(map[string]*glxlib.Event),
	}

	// Add 50 persons
	for i := range 50 {
		id := fmt.Sprintf("person-%d", i)
		glxFile.Persons[id] = &glxlib.Person{
			Properties: map[string]any{
				"primary_name": fmt.Sprintf("Person %d", i),
			},
		}
	}

	// Add 30 events with valid participants
	for i := range 30 {
		id := fmt.Sprintf("event-%d", i)
		personID := fmt.Sprintf("person-%d", i%50) // Reference one of the persons we created
		glxFile.Events[id] = &glxlib.Event{
			Type: "birth",
			Participants: []glxlib.Participant{
				{
					Person: personID,
					Role:   "principal",
				},
			},
		}
	}

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{
		Validate: false,
		Pretty:   true,
	})

	data, err := serializer.SerializeSingleFileBytes(glxFile)
	require.NoError(t, err)

	inputPath := filepath.Join(tmpDir, "large.glx")
	err = os.WriteFile(inputPath, data, 0o644)
	require.NoError(t, err)

	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = true // Skip validation for this large dataset test
	splitVerbose = false

	err = splitArchive(inputPath, outputDir, false, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Verify correct number of files
	personsDir := filepath.Join(outputDir, "persons")
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	require.Len(t, personFiles, 50, "should have 50 person files")

	eventsDir := filepath.Join(outputDir, "events")
	eventFiles, err := os.ReadDir(eventsDir)
	require.NoError(t, err)
	require.Len(t, eventFiles, 30, "should have 30 event files")
}

func TestSplitArchive_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	inputPath := createTestSingleFileArchive(t, tmpDir)
	outputDir := filepath.Join(tmpDir, "output")

	splitNoValidate = false
	splitVerbose = false

	err := splitArchive(inputPath, outputDir, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Check permissions on created directories (should be 0755)
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o755), info.Mode().Perm(), "output directory should have 0755 permissions")

	// Check permissions on created files (should be 0644)
	personsDir := filepath.Join(outputDir, "persons")
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	require.NotEmpty(t, personFiles)

	firstPersonPath := filepath.Join(personsDir, personFiles[0].Name())
	fileInfo, err := os.Stat(firstPersonPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), fileInfo.Mode().Perm(), "person files should have 0644 permissions")
}
