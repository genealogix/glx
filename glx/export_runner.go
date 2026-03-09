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

	glxlib "github.com/genealogix/glx/go-glx"
)

// GEDCOM version format constants for the --format flag
const (
	ExportFormat551 = "551"
	ExportFormat70  = "70"
)

// exportToGEDCOM loads a GLX archive and exports it to GEDCOM format
func exportToGEDCOM(inputPath, outputPath, format string, verbose bool) error {
	// Parse GEDCOM version
	version, err := parseGEDCOMVersion(format)
	if err != nil {
		return err
	}

	// Determine if input is single-file or multi-file archive
	glx, err := loadGLXArchive(inputPath, verbose)
	if err != nil {
		return err
	}

	// Set up log writer for verbose mode
	var logWriter *os.File
	if verbose {
		logWriter = os.Stderr
	}

	// Export to GEDCOM
	if verbose {
		fmt.Printf("Exporting to GEDCOM %s format\n", format)
	}

	data, result, err := glxlib.ExportGEDCOM(glx, version, logWriter)
	if err != nil {
		return fmt.Errorf("failed to export GEDCOM: %w", err)
	}

	// Ensure .ged extension
	outputPath = ensureGEDExtension(outputPath)

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, dirPermissions); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Write GEDCOM file
	if err := os.WriteFile(outputPath, data, filePermissions); err != nil {
		return fmt.Errorf("failed to write GEDCOM file: %w", err)
	}

	printExportSuccess(outputPath, result)

	return nil
}

// parseGEDCOMVersion converts a format string to a GEDCOMVersion enum
func parseGEDCOMVersion(format string) (glxlib.GEDCOMVersion, error) {
	switch strings.TrimSpace(format) {
	case ExportFormat551, "5.5.1":
		return glxlib.GEDCOM551, nil
	case ExportFormat70, "7.0":
		return glxlib.GEDCOM70, nil
	default:
		return glxlib.GEDCOMUnknown, fmt.Errorf("%w: %s (use '551' or '70')", ErrInvalidExportFormat, format)
	}
}

// loadGLXArchive loads a GLX archive from either a single file or multi-file directory
func loadGLXArchive(inputPath string, verbose bool) (*glxlib.GLXFile, error) {
	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrInputNotFound, inputPath)
		}
		return nil, fmt.Errorf("failed to access input path: %w", err)
	}

	if info.IsDir() {
		// Multi-file archive
		if verbose {
			fmt.Printf("Loading multi-file archive: %s\n", inputPath)
		}

		glx, _, err := LoadArchiveWithOptions(inputPath, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}

		if verbose {
			printVerboseArchiveStatistics(glx, "Archive loaded successfully")
		}

		return glx, nil
	}

	// Single-file archive
	if verbose {
		fmt.Printf("Loading single-file archive: %s\n", inputPath)
	}

	glx, err := readSingleFileArchive(inputPath, false)
	if err != nil {
		return nil, err
	}

	if verbose {
		printVerboseArchiveStatistics(glx, "Archive loaded successfully")
	}

	return glx, nil
}

// ensureGEDExtension adds .ged extension if not present
func ensureGEDExtension(path string) string {
	if !strings.HasSuffix(strings.ToLower(path), ".ged") {
		return path + ".ged"
	}

	return path
}

// printExportSuccess prints export completion message with statistics
func printExportSuccess(outputPath string, result *glxlib.ExportResult) {
	fmt.Printf("✓ Successfully exported to %s (GEDCOM %s)\n", outputPath, result.Version)
	fmt.Println("\nExport statistics:")
	fmt.Printf("  Persons:       %d\n", result.Statistics.PersonsExported)
	fmt.Printf("  Families:      %d\n", result.Statistics.FamiliesExported)
	fmt.Printf("  Sources:       %d\n", result.Statistics.SourcesExported)
	fmt.Printf("  Repositories:  %d\n", result.Statistics.RepositoriesExported)
	fmt.Printf("  Media:         %d\n", result.Statistics.MediaExported)

	if len(result.Statistics.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Statistics.Warnings))
		for _, w := range result.Statistics.Warnings {
			fmt.Printf("  [%s %s] %s\n", w.EntityType, w.EntityID, w.Message)
		}
	}
}
