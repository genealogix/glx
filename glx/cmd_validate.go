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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate GLX files and cross-references",
	Long: `Validate GENEALOGIX (.glx) files for correctness and integrity.

Performs comprehensive validation including:
- YAML syntax correctness
- Required fields presence
- Entity ID format validation
- Cross-reference integrity
- Duplicate ID detection
- Vocabulary validation (if vocabularies/ exists)

When validating directories, automatically checks all .glx, .yaml, and .yml files
and validates cross-references between entities.`,
	Example: `  # Validate current directory
  glx validate

  # Validate specific directory
  glx validate persons/

  # Validate multiple paths
  glx validate persons/ events/ places/

  # Validate single file
  glx validate archive.glx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runValidate(args)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func runValidate(args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var (
		allWarnings []string
		allErrors   []string
		foundFiles  bool
	)

	// For each path, load archive and validate
	for _, rootPath := range paths {
		info, err := os.Stat(rootPath)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("stat error for %s: %v", rootPath, err))
			continue
		}

		var archivePath string
		if info.IsDir() {
			archivePath = rootPath
			// Check if directory contains any GLX files
			hasGLXFiles := false
			filepath.WalkDir(archivePath, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				ext := filepath.Ext(d.Name())
				if ext == ".glx" || ext == ".yaml" || ext == ".yml" {
					hasGLXFiles = true
					return filepath.SkipAll // Stop walking once we find one
				}
				return nil
			})
			if !hasGLXFiles {
				allErrors = append(allErrors, fmt.Sprintf("no .glx files found in %s", rootPath))
				continue
			}
		} else {
			// If it's a file, check extension
			ext := filepath.Ext(rootPath)
			if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
				allErrors = append(allErrors, fmt.Sprintf("%s is not a .glx/.yaml file", rootPath))
				continue
			}
			archivePath = filepath.Dir(rootPath)
			foundFiles = true
		}

		// Load and merge all GLX files from the archive
		archive, duplicates, err := LoadArchive(archivePath)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("Failed to load archive from %s: %v", archivePath, err))
			continue
		}

		// Check if archive has any content
		hasContent := len(archive.Persons) > 0 || len(archive.Relationships) > 0 || len(archive.Events) > 0 ||
			len(archive.Places) > 0 || len(archive.Sources) > 0 || len(archive.Citations) > 0 ||
			len(archive.Repositories) > 0 || len(archive.Assertions) > 0 || len(archive.Media) > 0
		if hasContent {
			foundFiles = true
		}

		// Report duplicate IDs as errors
		if len(duplicates) > 0 {
			allErrors = append(allErrors, duplicates...)
		}

		// Validate the merged archive
		refErrors, refWarnings := ValidateArchive(archive, archivePath)
		allErrors = append(allErrors, refErrors...)
		allWarnings = append(allWarnings, refWarnings...)
	}

	// Check if we found any files to validate
	if !foundFiles && len(allErrors) == 0 {
		return errors.New("no .glx files found to validate")
	}

	// Format and print the final report
	printValidationReport(nil, allWarnings, allErrors)

	if len(allErrors) > 0 {
		return errors.New("validation failed")
	}

	fmt.Printf("\nValidation passed with %d warning(s).\n", len(allWarnings))
	return nil
}

func printValidationReport(reportLines []string, allWarnings, allErrors []string) {
	if reportLines != nil {
		for _, line := range reportLines {
			fmt.Println(line)
		}
	}

	if len(allErrors) > 0 {
		fmt.Println("\nValidation Errors:")
		for _, issue := range allErrors {
			fmt.Printf("  ✗ %s\n", issue)
		}
	}

	if len(allWarnings) > 0 {
		fmt.Println("\nValidation Warnings:")
		for _, warn := range allWarnings {
			fmt.Printf("  - %s\n", warn)
		}
	}
}
