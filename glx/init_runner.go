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
research_logs: {}
studies: {}
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
	// Create directory structure for a GENEALOGIX repository — every entity-type
	// directory plus vocabularies/, derived from glxlib.AllEntityTypes so new
	// entity types are scaffolded automatically.
	dirs := make([]string, 0, 1+len(glxlib.AllEntityTypes))
	dirs = append(dirs, glxlib.ArchiveDirVocabularies)
	for _, et := range glxlib.AllEntityTypes {
		dirs = append(dirs, et.String())
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
	fmt.Println("  Research: research_logs/, studies/")
	fmt.Println("Created .gitignore and README.md")
	fmt.Println("")
	fmt.Println("Each .glx file should have entity type keys at the top level.")

	return nil
}

// writeTestData writes test data to entity files.
func writeTestData(data *glxlib.GLXFile) error {
	entityMaps := []struct {
		entityType glxlib.EntityType
		source     any
	}{
		{glxlib.EntityTypePersons, data.Persons},
		{glxlib.EntityTypeRelationships, data.Relationships},
		{glxlib.EntityTypeEvents, data.Events},
		{glxlib.EntityTypePlaces, data.Places},
		{glxlib.EntityTypeSources, data.Sources},
		{glxlib.EntityTypeCitations, data.Citations},
		{glxlib.EntityTypeRepositories, data.Repositories},
		{glxlib.EntityTypeAssertions, data.Assertions},
		{glxlib.EntityTypeMedia, data.Media},
		{glxlib.EntityTypeResearchLogs, data.ResearchLogs},
		{glxlib.EntityTypeStudies, data.Studies},
	}

	for _, m := range entityMaps {
		entities, err := structToMap(m.source)
		if err != nil {
			return fmt.Errorf("marshal %s for test data: %w", m.entityType, err)
		}
		dirStr := m.entityType.String()
		for id, entity := range entities {
			if err := writeTestDataFile(dirStr, id, entity); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeTestDataFile marshals a single entity wrapped under its plural type key
// and writes it to `<dir>/<id>.glx`. Errors carry both the entity ID and the
// underlying cause so failures point straight at the offending record.
func writeTestDataFile(dir, id string, entity any) error {
	fileName := filepath.Join(dir, id+".glx")
	yamlData, err := yaml.Marshal(map[string]any{
		dir: map[string]any{id: entity},
	})
	if err != nil {
		return fmt.Errorf("marshal %s: %w", id, err)
	}
	if err := os.WriteFile(fileName, yamlData, filePermissions); err != nil {
		return fmt.Errorf("write %s: %w", fileName, err)
	}

	return nil
}

// structToMap converts a struct (typically a typed entity map like
// `map[string]*Person`) to a generic `map[string]any` via a YAML round-trip.
// Returns the surfaced error rather than panicking — callers are I/O paths
// that already propagate errors, so panicking would needlessly abort the
// process when surfacing the failure works fine.
func structToMap(v any) (map[string]any, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("yaml marshal: %w", err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	return m, nil
}
