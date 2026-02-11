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

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// runInit initializes a new GLX archive in the specified directory
func runInit(targetDir string, singleFile bool, numTestData int) error {
	// If target is '.', check if it's empty. Otherwise, check if it exists and is not empty.
	info, err := os.Stat(targetDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not stat target directory '%s': %w", targetDir, err)
		}
		// Path doesn't exist, will be created below
	} else {
		// Path exists
		if !info.IsDir() {
			return fmt.Errorf("%w: %s", ErrTargetNotDirectory, targetDir)
		}
		if err := isDirectoryEmpty(targetDir); err != nil {
			return err
		}
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(targetDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

	// Change into the target directory to perform initialization
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not get current directory: %w", err)
	}
	if err := os.Chdir(targetDir); err != nil {
		return fmt.Errorf("failed to change into directory %s: %w", targetDir, err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	if singleFile {
		return createSingleFileArchive(targetDir)
	}

	return createMultiFileArchive(targetDir, numTestData)
}

// createSingleFileArchive creates a single-file GLX archive template
func createSingleFileArchive(targetDir string) error {
	template := `# GENEALOGIX Family Archive
# Single-file format

persons: {}
relationships: {}
events: {}
places: {}
sources: {}
citations: {}
repositories: {}
assertions: {}
media: {}
`
	if err := os.WriteFile("archive.glx", []byte(template), filePermissions); err != nil {
		return fmt.Errorf("failed to create archive.glx: %w", err)
	}

	fmt.Printf("Initialized single-file GENEALOGIX archive: archive.glx in %s\n", targetDir)
	fmt.Printf("Add entities under the appropriate type keys (persons, sources, etc.) in %s\n", targetDir)

	return nil
}

// createMultiFileArchive creates a multi-file GLX archive directory structure
func createMultiFileArchive(targetDir string, numTestData int) error {
	// Create directory structure for a GENEALOGIX repository
	dirs := []string{
		"persons",
		"relationships",
		"events",
		"places",
		"sources",
		"citations",
		"repositories",
		"assertions",
		"media",
		"vocabularies",
	}

	if err := createDirectoryStructure(dirs); err != nil {
		return err
	}

	// Create standard vocabulary files
	if err := createStandardVocabularies(); err != nil {
		return err
	}

	// Create .gitignore file
	if err := os.WriteFile(".gitignore", defaultGitignore, filePermissions); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README.md for the repository
	if err := os.WriteFile("README.md", defaultReadme, filePermissions); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	if numTestData > 0 {
		fmt.Printf("Generating test data for %d persons...\n", numTestData)
		testData, err := glxlib.GenerateTestData(numTestData)
		if err != nil {
			return fmt.Errorf("failed to generate test data: %w", err)
		}
		if err := writeTestData(testData); err != nil {
			return fmt.Errorf("failed to write test data: %w", err)
		}
		fmt.Println("Test data generated successfully.")
	}

	if targetDir == "." {
		targetDir = "the current directory"
	}
	fmt.Printf("Initialized multi-file GENEALOGIX repository in %s\n", targetDir)
	fmt.Println("Created directories:")
	fmt.Println("  Core: persons/, relationships/, events/, places/")
	fmt.Println("  Evidence: sources/, citations/, repositories/, assertions/")
	fmt.Println("  Media: media/")
	fmt.Println("Created .gitignore and README.md")
	fmt.Println("")
	fmt.Println("Each .glx file should have entity type keys at the top level.")

	return nil
}

// writeTestData writes test data to entity files
func writeTestData(data *glxlib.GLXFile) error {
	entityTypes := map[string]map[string]any{
		"persons":       mustMarshal(data.Persons),
		"relationships": mustMarshal(data.Relationships),
		"events":        mustMarshal(data.Events),
		"places":        mustMarshal(data.Places),
		"sources":       mustMarshal(data.Sources),
		"citations":     mustMarshal(data.Citations),
		"repositories":  mustMarshal(data.Repositories),
		"assertions":    mustMarshal(data.Assertions),
		"media":         mustMarshal(data.Media),
	}

	for dir, entities := range entityTypes {
		for id, entity := range entities {
			fileName := filepath.Join(dir, id+".glx")
			fileContent := map[string]any{
				dir: map[string]any{
					id: entity,
				},
			}
			yamlData, err := yaml.Marshal(fileContent)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", id, err)
			}
			if err := os.WriteFile(fileName, yamlData, filePermissions); err != nil {
				return fmt.Errorf("failed to write file %s: %w", fileName, err)
			}
		}
	}

	return nil
}

// mustMarshal converts a struct to map[string]any
func mustMarshal(v any) map[string]any {
	// A bit of a hack to convert struct to map[string]any
	// for easy file writing.
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		panic(err)
	}

	return m
}
