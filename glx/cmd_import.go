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
	"strings"

	"github.com/genealogix/glx/glx/lib"
	"github.com/spf13/cobra"
)

var (
	importOutput          string
	importFormat          string
	importNoValidate      bool
	importVerbose         bool
	importShowFirstErrors int
)

var importCmd = &cobra.Command{
	Use:   "import <gedcom-file>",
	Short: "Import a GEDCOM file to GLX format",
	Long: `Import a GEDCOM file and convert it to GLX format.

Supports both GEDCOM 5.5.1 and GEDCOM 7.0 formats.

The imported archive will include:
- All individuals (persons)
- All events (births, deaths, marriages, etc.)
- All relationships (parent-child, spouse, etc.)
- All places with hierarchical structure
- All sources and citations
- All repositories and media
- Evidence-based assertions

Output formats:
- single: Single YAML file (default)
- multi: Multi-file directory structure (one file per entity)`,
	Example: `  # Import to single file
  glx import family.ged -o family.glx

  # Import to multi-file directory
  glx import family.ged -o family-archive --format multi

  # Import without validation
  glx import family.ged -o family.glx --no-validate`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file or directory (required)")
	importCmd.Flags().StringVarP(&importFormat, "format", "f", "single", "Output format: single or multi")
	importCmd.Flags().BoolVar(&importNoValidate, "no-validate", false, "Skip validation before saving")
	importCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")
	importCmd.Flags().IntVar(&importShowFirstErrors, "show-first-errors", defaultShowFirstErrors, "Number of validation errors to show (0 for all)")

	_ = importCmd.MarkFlagRequired("output")
}

func runImport(_ *cobra.Command, args []string) error {
	return importGEDCOM(args[0])
}

func importGEDCOM(gedcomPath string) error {
	// Validate format flag
	if importFormat != "single" && importFormat != "multi" {
		return fmt.Errorf("%w: %s", ErrInvalidFormat, importFormat)
	}

	// Check if GEDCOM file exists
	if _, err := os.Stat(gedcomPath); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrGEDCOMFileNotFound, gedcomPath)
	}

	// Import GEDCOM file
	if importVerbose {
		fmt.Printf("Importing GEDCOM file: %s\n", gedcomPath)
	}

	// Open GEDCOM file
	gedcomFile, err := os.Open(gedcomPath)
	if err != nil {
		return fmt.Errorf("failed to open GEDCOM file: %w", err)
	}
	defer func() { _ = gedcomFile.Close() }()

	// Import GEDCOM from reader
	glx, _, err := lib.ImportGEDCOM(gedcomFile, "")
	if err != nil {
		return fmt.Errorf("failed to import GEDCOM: %w", err)
	}

	// Create serializer
	serializerOpts := &lib.SerializerOptions{
		Validate: !importNoValidate,
		Pretty:   true,
		Indent:   "  ",
	}
	serializer := lib.NewSerializer(serializerOpts)

	// Serialize based on format
	if importFormat == "single" {
		// Single-file format
		if importVerbose {
			fmt.Printf("Writing single-file archive: %s\n", importOutput)
		}

		// Ensure .glx extension
		if !strings.HasSuffix(importOutput, ".glx") {
			importOutput += ".glx"
		}

		// Serialize to bytes
		yamlBytes, err := serializer.SerializeSingleFileBytes(glx)
		if err != nil {
			return fmt.Errorf("failed to serialize GLX file: %w", formatValidationError(err, importShowFirstErrors))
		}

		// Write to file
		if err := os.WriteFile(importOutput, yamlBytes, 0o644); err != nil {
			return fmt.Errorf("failed to write GLX file: %w", err)
		}

		fmt.Printf("✓ Successfully imported to %s\n", importOutput)

	} else {
		// Multi-file format
		if importVerbose {
			fmt.Printf("Writing multi-file archive: %s\n", importOutput)
		}

		// Serialize to map of files
		files, err := serializer.SerializeMultiFileToMap(glx)
		if err != nil {
			return fmt.Errorf("failed to serialize GLX archive: %w", formatValidationError(err, importShowFirstErrors))
		}

		// Create output directory
		if err := os.MkdirAll(importOutput, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Write all files
		for relPath, content := range files {
			absPath := filepath.Join(importOutput, relPath)

			// Create parent directory
			parentDir := filepath.Dir(absPath)
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", parentDir, err)
			}

			// Write file
			if err := os.WriteFile(absPath, content, 0o644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", absPath, err)
			}
		}

		fmt.Printf("✓ Successfully imported to %s/\n", importOutput)
	}

	// Print statistics
	fmt.Println("\nImport statistics:")
	fmt.Printf("  Persons:       %d\n", len(glx.Persons))
	fmt.Printf("  Events:        %d\n", len(glx.Events))
	fmt.Printf("  Relationships: %d\n", len(glx.Relationships))
	fmt.Printf("  Places:        %d\n", len(glx.Places))
	fmt.Printf("  Sources:       %d\n", len(glx.Sources))
	fmt.Printf("  Citations:     %d\n", len(glx.Citations))
	fmt.Printf("  Repositories:  %d\n", len(glx.Repositories))
	fmt.Printf("  Media:         %d\n", len(glx.Media))
	fmt.Printf("  Assertions:    %d\n", len(glx.Assertions))

	return nil
}
