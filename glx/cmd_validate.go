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
		checked     int
		hadError    bool
		reportLines []string
		allWarnings []string
		allErrors   []string
	)

	// Load vocabularies for validation
	vocabs, err := LoadArchiveVocabularies(".")
	if err != nil {
		allErrors = append(allErrors, fmt.Sprintf("Failed to load vocabularies: %v", err))
		hadError = true
	}

	// First Pass: Discover all files and perform initial validation
	allGLXFiles := discoverGLXFiles(paths, &allErrors)
	if len(allGLXFiles) == 0 && !hadError {
		return errors.New("no .glx files found to validate")
	}

	for _, filePath := range allGLXFiles {
		checked++
		issues := validateGLXFileFromPath(filePath, vocabs)
		if len(issues) > 0 {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(filePath, issues)...)
		} else {
			reportLines = append(reportLines, fmt.Sprintf("✓ %s", filePath))
		}
	}

	// Collect all entity IDs for cross-reference validation
	allEntities, duplicates, err := CollectAllEntities(".")
	if err != nil {
		allErrors = append(allErrors, fmt.Sprintf("Failed to collect entities: %v", err))
		hadError = true
	}
	if len(duplicates) > 0 {
		hadError = true
		allErrors = append(allErrors, duplicates...)
	}

	// Second pass: validate all cross-references if the first pass was clean
	if !hadError {
		refErrors, refWarnings := ValidateRepositoryReferences(".", allEntities, vocabs)
		allErrors = append(allErrors, refErrors...)
		allWarnings = append(allWarnings, refWarnings...)
	}

	// Format and print the final report
	printValidationReport(reportLines, allWarnings, allErrors)

	if len(allErrors) > 0 || hadError {
		return errors.New("validation failed")
	}

	fmt.Printf("\nValidated %d file(s) with %d warning(s).\n", checked, len(allWarnings))
	return nil
}

func discoverGLXFiles(paths []string, allErrors *[]string) []string {
	var allGLXFiles []string
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			*allErrors = append(*allErrors, fmt.Sprintf("stat error for %s: %v", root, err))
			continue
		}

		if info.IsDir() {
			filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					*allErrors = append(*allErrors, fmt.Sprintf("walk error for %s: %v", path, walkErr))
					return nil
				}
				if !d.IsDir() {
					ext := filepath.Ext(d.Name())
					if ext == ".glx" || ext == ".yaml" || ext == ".yml" {
						allGLXFiles = append(allGLXFiles, path)
					}
				}
				return nil
			})
		} else {
			ext := filepath.Ext(root)
			if ext == ".glx" || ext == ".yaml" || ext == ".yml" {
				allGLXFiles = append(allGLXFiles, root)
			} else {
				*allErrors = append(*allErrors, fmt.Sprintf("%s is not a .glx/.yaml file", root))
			}
		}
	}
	return allGLXFiles
}

func printValidationReport(reportLines, allWarnings, allErrors []string) {
	for _, line := range reportLines {
		fmt.Println(line)
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

func validateGLXFileFromPath(path string, vocabs *ArchiveVocabularies) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("read error: %v", err)}
	}

	doc, err := ParseYAMLFile(data)
	if err != nil {
		return []string{fmt.Sprintf("YAML parse error: %v", err)}
	}

	return ValidateGLXFile(path, doc, vocabs)
}

func formatValidationIssues(path string, issues []string) []string {
	lines := []string{fmt.Sprintf("✗ %s", path)}
	for _, issue := range issues {
		lines = append(lines, fmt.Sprintf("  - %s", issue))
	}
	return lines
}
