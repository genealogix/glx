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
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genealogix/glx/glx/lib"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	initSingleFile bool
	createTestData int
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new GENEALOGIX archive in the specified directory",
	Long: `Initialize a new GENEALOGIX archive with the proper directory structure.

If a directory is provided, it will be created. If no directory is provided,
the current directory will be used (but must be empty).

By default, creates a multi-file archive with separate directories for each
entity type (persons/, events/, places/, etc.) along with standard vocabulary
files and supporting documentation.

Use --single-file to create a single archive.glx file instead.`,
	Example: `  # Initialize in a new directory
  glx init my-family-archive

  # Initialize a single-file archive in a new directory
  glx init my-family-archive --single-file

  # Initialize with test data in a new directory
  glx init my-family-archive --create-test-data 10`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInitCmd,
}

func runInitCmd(_ *cobra.Command, args []string) error {
	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	} else {
		targetDir = "."
	}
	return runInit(targetDir, initSingleFile, createTestData)
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "create a single-file archive instead of multi-file")
	initCmd.Flags().IntVarP(&createTestData, "create-test-data", "t", 0, "number of persons to generate test data for")
}

func runInit(targetDir string, singleFile bool, numTestData int) error {
	// If target is '.', check if it's empty. Otherwise, check if it exists and is not empty.
	info, err := os.Stat(targetDir)
	if err == nil { // Path exists
		if !info.IsDir() {
			return fmt.Errorf("target path '%s' exists and is not a directory", targetDir)
		}
		if err := isDirectoryEmpty(targetDir); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) { // Some other error
		return fmt.Errorf("could not stat target directory '%s': %w", targetDir, err)
	}

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
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
		// Create single-file archive template
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
		if err := os.WriteFile("archive.glx", []byte(template), 0o644); err != nil {
			return fmt.Errorf("failed to create archive.glx: %w", err)
		}

		fmt.Printf("Initialized single-file GENEALOGIX archive: archive.glx in %s\n", targetDir)
		fmt.Printf("Add entities under the appropriate type keys (persons, sources, etc.) in %s\n", targetDir)
		fmt.Printf("Entity IDs are map keys - don't include 'id' field in entities in %s\n", targetDir)
		return nil
	}

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

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create standard vocabulary files
	if err := createStandardVocabularies(); err != nil {
		return err
	}

	// Create .gitignore file
	if err := os.WriteFile(".gitignore", defaultGitignore, 0o644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}

	// Create README.md for the repository
	if err := os.WriteFile("README.md", defaultReadme, 0o644); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	if numTestData > 0 {
		fmt.Printf("Generating test data for %d persons...\n", numTestData)
		testData, err := lib.GenerateTestData(numTestData)
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
	fmt.Println("Entity IDs are map keys - don't include 'id' field in entities.")

	return nil
}

func isDirectoryEmpty(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not check directory: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Read exactly one directory entry.
	// Readdirnames will return an error if the directory is empty.
	// We expect an io.EOF error, which means it's empty.
	_, err = f.Readdirnames(1)
	if err == nil { // if err is nil, directory is not empty
		return fmt.Errorf("cannot run 'glx init' in a non-empty directory. Please create a new directory for your family archive")
	}

	return nil // Directory is empty
}

func writeTestData(data *lib.GLXFile) error {
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
			fileName := filepath.Join(dir, fmt.Sprintf("%s.glx", id))
			fileContent := map[string]any{
				dir: map[string]any{
					id: entity,
				},
			}
			yamlData, err := yaml.Marshal(fileContent)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", id, err)
			}
			if err := os.WriteFile(fileName, yamlData, 0o644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", fileName, err)
			}
		}
	}
	return nil
}

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

// createStandardVocabularies is now in vocabularies_embed.go
