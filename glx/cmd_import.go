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
	"strings"

	"github.com/genealogix/spec/glx/lib"
	"github.com/spf13/cobra"
)

var (
	importOutput          string
	importFormat          string
	importNoValidate      bool
	importNoVocabularies  bool
	importVerbose         bool
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
- multi: Multi-file directory structure (one file per entity)

By default, standard vocabularies are included in the output.`,
	Example: `  # Import to single file
  glx import family.ged -o family.glx

  # Import to multi-file directory
  glx import family.ged -o family-archive --format multi

  # Import without validation
  glx import family.ged -o family.glx --no-validate

  # Import without vocabularies
  glx import family.ged -o family.glx --no-vocabularies`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().StringVarP(&importOutput, "output", "o", "", "Output file or directory (required)")
	importCmd.Flags().StringVarP(&importFormat, "format", "f", "single", "Output format: single or multi")
	importCmd.Flags().BoolVar(&importNoValidate, "no-validate", false, "Skip validation before saving")
	importCmd.Flags().BoolVar(&importNoVocabularies, "no-vocabularies", false, "Don't include standard vocabularies")
	importCmd.Flags().BoolVarP(&importVerbose, "verbose", "v", false, "Verbose output")

	importCmd.MarkFlagRequired("output")
}

func runImport(cmd *cobra.Command, args []string) error {
	gedcomPath := args[0]

	// Validate format flag
	if importFormat != "single" && importFormat != "multi" {
		return fmt.Errorf("invalid format: %s (must be 'single' or 'multi')", importFormat)
	}

	// Check if GEDCOM file exists
	if _, err := os.Stat(gedcomPath); os.IsNotExist(err) {
		return fmt.Errorf("GEDCOM file not found: %s", gedcomPath)
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
	defer gedcomFile.Close()

	// Import GEDCOM from reader
	glx, _, err := lib.ImportGEDCOM(gedcomFile, "")
	if err != nil {
		return fmt.Errorf("failed to import GEDCOM: %w", err)
	}

	// Create serializer
	serializerOpts := &lib.SerializerOptions{
		IncludeVocabularies: !importNoVocabularies,
		Validate:            !importNoValidate,
		Pretty:              true,
		Indent:              "  ",
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

		if err := serializer.SerializeSingleFile(glx, importOutput); err != nil {
			return fmt.Errorf("failed to write GLX file: %w", err)
		}

		fmt.Printf("✓ Successfully imported to %s\n", importOutput)

	} else {
		// Multi-file format
		if importVerbose {
			fmt.Printf("Writing multi-file archive: %s\n", importOutput)
		}

		if err := serializer.SerializeMultiFile(glx, importOutput); err != nil {
			return fmt.Errorf("failed to write GLX archive: %w", err)
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

	if importFormat == "multi" && !importNoVocabularies {
		fmt.Println("\n  Standard vocabularies included in archive")
	}

	return nil
}
