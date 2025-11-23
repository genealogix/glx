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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExamples validates all example GLX files from docs/examples
func TestExamples(t *testing.T) {
	examplesDir := "../docs/examples"

	// Check if examples directory exists
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("examples directory not found - skipping examples validation")

		return
	}

	var validFiles []string
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".glx") {
			validFiles = append(validFiles, path)
		}

		return nil
	})
	require.NoError(t, err, "failed to walk examples directory")

	if len(validFiles) == 0 {
		t.Fatal("no .glx files found in examples directory")
	}

	t.Logf("Found %d example files to validate", len(validFiles))

	for _, file := range validFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			require.NoError(t, err, "failed to read %s", file)

			// Skip reference files that contain relative paths instead of YAML
			content := strings.TrimSpace(string(data))
			if strings.HasPrefix(content, "../") || strings.HasPrefix(content, "../../../../") {
				t.Skipf("skipping reference file %s", file)

				return
			}

			doc, err := ParseYAMLFile(data)
			require.NoError(t, err, "failed to parse YAML in %s", file)

			// Validate entity structure - only check entity type keys
			entityKeys := map[string]bool{
				"persons": true, "relationships": true, "events": true, "places": true,
				"sources": true, "citations": true, "repositories": true, "assertions": true, "media": true,
			}

			for pluralKey, entities := range doc {
				// Only validate entity keys (vocabularies are ignored)
				if !entityKeys[pluralKey] {
					continue
				}

				if entityMap, ok := entities.(map[string]any); ok {
					for entityID, entityData := range entityMap {
						if entity, ok := entityData.(map[string]any); ok {
							// Check no 'id' field
							if _, hasID := entity["id"]; hasID {
								t.Errorf("%s: %s[%s] must not have 'id' field - the map key is the ID", file, pluralKey, entityID)
							}

							// Validate ID format (alphanumeric + hyphens, 1-64 chars)
							if !isValidEntityID(entityID) {
								t.Errorf("%s: %s[%s] invalid ID format (must be alphanumeric/hyphens, 1-64 chars)", file, pluralKey, entityID)
							}
						}
					}
				}
			}
		})
	}
}

// TestExamplesDirectories validates that each example directory is structured correctly
func TestExamplesDirectories(t *testing.T) {
	examplesDir := "../docs/examples"

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("examples directory not found")

		return
	}

	examples := []string{"minimal", "basic-family", "complete-family", "single-file", "participant-assertions", "temporal-properties"}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, example)
			info, err := os.Stat(examplePath)

			if os.IsNotExist(err) {
				t.Skipf("example %s not found", example)

				return
			}

			require.NoError(t, err)
			assert.True(t, info.IsDir(), "example should be a directory")

			// Check for README
			readmePath := filepath.Join(examplePath, "README.md")
			_, err = os.Stat(readmePath)
			require.NoError(t, err, "example %s should have README.md", example)
		})
	}
}

// TestExamplesCompleteFamily validates the complete-family example has all entity types
func TestExamplesCompleteFamily(t *testing.T) {
	completeFamilyDir := "../docs/examples/complete-family"

	if _, err := os.Stat(completeFamilyDir); os.IsNotExist(err) {
		t.Skip("complete-family example not found")

		return
	}

	// Expected subdirectories for complete-family
	expectedDirs := []string{
		"persons", "relationships", "events", "places",
		"sources", "citations", "repositories", "assertions",
	}

	for _, dir := range expectedDirs {
		t.Run(dir, func(t *testing.T) {
			dirPath := filepath.Join(completeFamilyDir, dir)
			info, err := os.Stat(dirPath)

			if os.IsNotExist(err) {
				t.Skipf("directory %s not found in complete-family", dir)

				return
			}

			require.NoError(t, err)
			assert.True(t, info.IsDir(), "%s should be a directory", dir)

			// Check that directory contains at least one .glx file
			files, err := os.ReadDir(dirPath)
			require.NoError(t, err)

			hasGlxFile := false
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".glx") {
					hasGlxFile = true

					break
				}
			}

			assert.True(t, hasGlxFile, "directory %s should contain at least one .glx file", dir)
		})
	}
}

// TestExamplesSingleFile validates the single-file example
func TestExamplesSingleFile(t *testing.T) {
	singleFilePath := "../docs/examples/single-file/archive.glx"

	if _, err := os.Stat(singleFilePath); os.IsNotExist(err) {
		t.Skip("single-file example not found")

		return
	}

	data, err := os.ReadFile(singleFilePath)
	require.NoError(t, err)

	doc, err := ParseYAMLFile(data)
	require.NoError(t, err)

	// Single file should contain multiple entity types
	entityTypes := []string{"persons", "relationships", "events", "places", "sources"}
	foundTypes := 0

	for _, entityType := range entityTypes {
		if _, exists := doc[entityType]; exists {
			foundTypes++
		}
	}

	assert.GreaterOrEqual(t, foundTypes, 3, "single-file example should contain at least 3 entity types")
}

// TestExamplesValidation runs full validation on each example directory
func TestExamplesValidation(t *testing.T) {
	examplesDir := "../docs/examples"

	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		t.Skip("examples directory not found")

		return
	}

	examples := []string{"minimal", "basic-family", "complete-family", "single-file", "participant-assertions", "temporal-properties"}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			examplePath := filepath.Join(examplesDir, example)

			if _, err := os.Stat(examplePath); os.IsNotExist(err) {
				t.Skipf("example %s not found", example)

				return
			}

			// Load and merge all GLX files from the archive
			archive, duplicates, err := LoadArchive(examplePath)
			require.NoError(t, err)

			// Check for duplicate IDs
			assert.Empty(t, duplicates, "example %s should not have duplicate entity IDs", example)

			// Validate the merged archive using new validation system
			result := archive.Validate()

			// Collect all errors and warnings
			var allRefIssues []string
			for _, err := range result.Errors {
				allRefIssues = append(allRefIssues, err.Message)
			}
			for _, warn := range result.Warnings {
				allRefIssues = append(allRefIssues, warn.Message)
			}
			assert.Empty(t, allRefIssues, "example %s should not have validation issues: %v", example, allRefIssues)
		})
	}
}
