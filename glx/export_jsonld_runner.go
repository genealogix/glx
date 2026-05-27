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

const jsonLDExtension = ".jsonld"

// exportToJSONLD loads a GLX archive and exports it to a JSON-LD document
// aligned with the Schema.org context shipped in
// specification/jsonld/glx-context.jsonld.
func exportToJSONLD(inputPath, outputPath string, verbose, privatizeLiving bool) error {
	glx, err := loadGLXArchive(inputPath, verbose)
	if err != nil {
		return err
	}

	applyExportPrivacy(glx, privatizeLiving, verbose)

	var logWriter *os.File
	if verbose {
		logWriter = os.Stderr
		fmt.Println("Exporting to JSON-LD format")
	}

	data, result, err := glxlib.ExportJSONLD(glx, logWriter)
	if err != nil {
		return fmt.Errorf("failed to export JSON-LD: %w", err)
	}

	outputPath = ensureJSONLDExtension(outputPath)

	outputDir := filepath.Dir(outputPath)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, dirPermissions); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, data, filePermissions); err != nil {
		return fmt.Errorf("failed to write JSON-LD file: %w", err)
	}

	printJSONLDExportSuccess(outputPath, result)

	return nil
}

// ensureJSONLDExtension adds .jsonld to outputPath if it does not already end
// in one (case-insensitive).
func ensureJSONLDExtension(path string) string {
	if !strings.HasSuffix(strings.ToLower(path), jsonLDExtension) {
		return path + jsonLDExtension
	}

	return path
}

func printJSONLDExportSuccess(outputPath string, result *glxlib.ExportResult) {
	fmt.Printf("✓ Successfully exported to %s (JSON-LD %s)\n", outputPath, result.Version)
	fmt.Println("\nExport statistics:")
	fmt.Printf("  Persons:        %d\n", result.Statistics.PersonsExported)
	fmt.Printf("  Events:         %d\n", result.Statistics.EventsProcessed)
	fmt.Printf("  Places:         %d\n", result.Statistics.PlacesResolved)
	fmt.Printf("  Relationships:  %d\n", result.Statistics.RelationshipsExported)
	fmt.Printf("  Sources:        %d\n", result.Statistics.SourcesExported)
	fmt.Printf("  Citations:      %d\n", result.Statistics.CitationsExported)
	fmt.Printf("  Repositories:   %d\n", result.Statistics.RepositoriesExported)
	fmt.Printf("  Media:          %d\n", result.Statistics.MediaExported)
	fmt.Printf("  Assertions:     %d\n", result.Statistics.AssertionsExported)

	if len(result.Statistics.Warnings) > 0 {
		fmt.Printf("\nWarnings (%d):\n", len(result.Statistics.Warnings))
		for _, w := range result.Statistics.Warnings {
			fmt.Printf("  [%s %s] %s\n", w.EntityType, w.EntityID, w.Message)
		}
	}
}
