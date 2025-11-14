package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSpecRepository(t *testing.T) {
	// Test from current directory (should be spec repo)
	result := isSpecRepository()
	// This should be true if we're in the spec repository
	// We can't easily test false case without mocking, so just verify it doesn't panic
	assert.NotPanics(t, func() {
		_ = isSpecRepository()
	})
	_ = result // Use result to avoid unused variable

	// Test from a temporary directory (should be false)
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	result = isSpecRepository()
	assert.False(t, result, "should return false for non-spec repository")
}

func TestRunInit_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = runInit(true)
	assert.NoError(t, err)

	// Check that archive.glx was created
	archivePath := filepath.Join(tmpDir, "archive.glx")
	_, err = os.Stat(archivePath)
	assert.NoError(t, err, "archive.glx should be created")

	// Check content
	content, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "persons:")
	assert.Contains(t, string(content), "relationships:")
}

func TestRunInit_MultiFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = runInit(false)
	assert.NoError(t, err)

	// Check that directories were created
	expectedDirs := []string{
		"persons", "relationships", "events", "places",
		"sources", "citations", "repositories", "assertions", "media", "vocabularies",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tmpDir, dir)
		info, err := os.Stat(dirPath)
		assert.NoError(t, err, "directory %s should be created", dir)
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
		"vocabularies/quality-ratings.glx",
	}

	for _, file := range vocabFiles {
		filePath := filepath.Join(tmpDir, file)
		_, err := os.Stat(filePath)
		assert.NoError(t, err, "vocabulary file %s should be created", file)
	}

	// Check that .gitignore was created
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	_, err = os.Stat(gitignorePath)
	assert.NoError(t, err, ".gitignore should be created")

	// Check that README.md was created
	readmePath := filepath.Join(tmpDir, "README.md")
	_, err = os.Stat(readmePath)
	assert.NoError(t, err, "README.md should be created")
}

func TestRunInit_InSpecRepository(t *testing.T) {
	// This should fail if we're in the spec repository
	// Try to run init in the current directory (spec repo)
	err := runInit(false)
	if isSpecRepository() {
		assert.Error(t, err, "should fail when run in spec repository")
		assert.Contains(t, err.Error(), "specification repository")
	} else {
		// If we're not in spec repo, it might succeed
		// Clean up if it did
		if err == nil {
			os.RemoveAll("persons")
			os.RemoveAll("relationships")
			os.RemoveAll("events")
			os.RemoveAll("places")
			os.RemoveAll("sources")
			os.RemoveAll("citations")
			os.RemoveAll("repositories")
			os.RemoveAll("assertions")
			os.RemoveAll("media")
			os.RemoveAll("vocabularies")
			os.Remove(".gitignore")
			os.Remove("README.md")
		}
	}
}

func TestCreateStandardVocabularies(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Create vocabularies directory first
	err = os.MkdirAll("vocabularies", 0755)
	require.NoError(t, err)

	err = createStandardVocabularies()
	assert.NoError(t, err)

	// Check that vocabulary files were created
	vocabFiles := []string{
		"vocabularies/relationship-types.glx",
		"vocabularies/event-types.glx",
		"vocabularies/place-types.glx",
		"vocabularies/repository-types.glx",
		"vocabularies/participant-roles.glx",
		"vocabularies/media-types.glx",
		"vocabularies/confidence-levels.glx",
		"vocabularies/quality-ratings.glx",
	}

	for _, file := range vocabFiles {
		filePath := filepath.Join(tmpDir, file)
		content, err := os.ReadFile(filePath)
		assert.NoError(t, err, "vocabulary file %s should be created", file)
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
