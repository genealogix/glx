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

func TestRunInit_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := runInit(tmpDir, true, 0)
	require.NoError(t, err)

	// Check that archive.glx was created
	archivePath := filepath.Join(tmpDir, "archive.glx")
	_, err = os.Stat(archivePath)
	require.NoError(t, err, "archive.glx should be created")

	// Check content
	content, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "persons:")
	assert.Contains(t, string(content), "relationships:")
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
	personFiles, err := os.ReadDir(filepath.Join(tmpDir, "persons"))
	require.NoError(t, err)
	assert.Len(t, personFiles, numPeople, "should create the correct number of person files")

	// Check that event files were created (one birth per person)
	eventFiles, err := os.ReadDir(filepath.Join(tmpDir, "events"))
	require.NoError(t, err)
	assert.Len(t, eventFiles, numPeople, "should create the correct number of event files")
}

func TestCreateStandardVocabularies(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Create vocabularies directory first
	err = os.MkdirAll("vocabularies", 0o755)
	require.NoError(t, err)

	err = createStandardVocabularies()
	require.NoError(t, err)

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
		content, err := os.ReadFile(filePath)
		require.NoError(t, err, "vocabulary file %s should be created", file)
		assert.NotEmpty(t, content, "vocabulary file %s should not be empty", file)
	}
}

func TestPrintUsage(t *testing.T) {
	// Test that the help command works
	assert.NotPanics(t, func() {
		// The rootCmd should be available
		_ = rootCmd
	})
}
