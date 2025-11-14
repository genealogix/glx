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

	"github.com/genealogix/spec/lib"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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
	)

	// Load vocabularies for validation
	vocabs, err := LoadArchiveVocabularies(".")
	if err != nil {
		reportLines = append(reportLines, fmt.Sprintf("✗ Failed to load vocabularies: %v", err))
		hadError = true
	}

	// Collect all GLX files to validate
	var allGLXFiles []string
	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			reportLines = append(reportLines, fmt.Sprintf("✗ %s (stat error: %v)", root, err))
			hadError = true
			continue
		}

		if info.IsDir() {
			err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					reportLines = append(reportLines, fmt.Sprintf("✗ %s (walk error: %v)", path, walkErr))
					hadError = true
					return nil
				}
				if d.IsDir() {
					return nil
				}
				ext := filepath.Ext(d.Name())
				if ext == ".glx" || ext == ".yaml" || ext == ".yml" {
					allGLXFiles = append(allGLXFiles, path)
				}
				return nil
			})
			if err != nil {
				reportLines = append(reportLines, fmt.Sprintf("✗ %s (walk error: %v)", root, err))
				hadError = true
			}
		} else {
			ext := filepath.Ext(root)
			if ext == ".glx" || ext == ".yaml" || ext == ".yml" {
				allGLXFiles = append(allGLXFiles, root)
			} else {
				reportLines = append(reportLines, fmt.Sprintf("✗ %s is not a .glx/.yaml file", root))
				hadError = true
			}
		}
	}

	if len(allGLXFiles) == 0 {
		return errors.New("no .glx files found to validate")
	}

	// First pass: validate each file against JSON schema
	mergedGLXFile := &lib.GLXFile{
		Persons:          make(map[string]*lib.Person),
		Relationships:    make(map[string]*lib.Relationship),
		Events:           make(map[string]*lib.Event),
		Places:           make(map[string]*lib.Place),
		Sources:          make(map[string]*lib.Source),
		Citations:        make(map[string]*lib.Citation),
		Repositories:     make(map[string]*lib.Repository),
		Assertions:       make(map[string]*lib.Assertion),
		Media:            make(map[string]*lib.Media),
		EventTypes:       make(map[string]*lib.EventType),
		ParticipantRoles: make(map[string]*lib.ParticipantRole),
		ConfidenceLevels: make(map[string]*lib.ConfidenceLevel),
		RelationshipTypes: make(map[string]*lib.RelationshipType),
		PlaceTypes:       make(map[string]*lib.PlaceType),
		SourceTypes:      make(map[string]*lib.SourceType),
		RepositoryTypes:  make(map[string]*lib.RepositoryType),
		MediaTypes:       make(map[string]*lib.MediaType),
		QualityRatings:   make(map[string]*lib.QualityRating),
	}

	for _, filePath := range allGLXFiles {
		checked++
		// Validate against JSON schema
		issues := validateGLXFileFromPath(filePath, vocabs)
		if len(issues) > 0 {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(filePath, issues)...)
			continue // Skip parsing if schema validation failed
		}

		// Parse into struct
		data, err := os.ReadFile(filePath)
		if err != nil {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(filePath, []string{fmt.Sprintf("read error: %v", err)})...)
			continue
		}

		var glxFile lib.GLXFile
		if err := yaml.Unmarshal(data, &glxFile); err != nil {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(filePath, []string{fmt.Sprintf("YAML parse error: %v", err)})...)
			continue
		}

		// Merge into merged GLXFile (fail fast on duplicates)
		duplicates := mergedGLXFile.Merge(&glxFile)
		if len(duplicates) > 0 {
			hadError = true
			reportLines = append(reportLines, formatValidationIssues(filePath, duplicates)...)
		} else {
			reportLines = append(reportLines, fmt.Sprintf("✓ %s", filePath))
		}
	}

	// Second pass: validate all references using struct-based validation
	if !hadError || len(mergedGLXFile.Persons) > 0 || len(mergedGLXFile.Relationships) > 0 {
		refIssues := ValidateReferencesWithStructs(mergedGLXFile)
		if len(refIssues) > 0 {
			hadError = true
			reportLines = append(reportLines, "")
			reportLines = append(reportLines, "Cross-reference issues:")
			for _, issue := range refIssues {
				reportLines = append(reportLines, fmt.Sprintf("  ✗ %s", issue))
			}
		}
	}

	// Print all results
	for _, line := range reportLines {
		fmt.Println(line)
	}

	if hadError {
		return errors.New("validation failed")
	}

	fmt.Printf("\nValidated %d file(s)\n", checked)
	return nil
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
