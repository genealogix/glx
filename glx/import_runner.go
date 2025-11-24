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

	"github.com/genealogix/glx/glx/lib"
)

// importGEDCOM imports a GEDCOM file and converts it to GLX format
func importGEDCOM(gedcomPath, outputPath, format string, validate, verbose bool, showFirstErrors int) error {
	// Validate format flag
	if format != "single" && format != "multi" {
		return fmt.Errorf("%w: %s", ErrInvalidFormat, format)
	}

	// Check if GEDCOM file exists
	if !fileExists(gedcomPath) {
		return fmt.Errorf("%w: %s", ErrGEDCOMFileNotFound, gedcomPath)
	}

	// Import GEDCOM file
	if verbose {
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

	// Serialize based on format
	if format == "single" {
		return importToSingleFile(glx, outputPath, validate, verbose, showFirstErrors)
	}

	return importToMultiFile(glx, outputPath, validate, verbose, showFirstErrors)
}

// importToSingleFile writes the imported GLX data to a single file
func importToSingleFile(glx *lib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int) error {
	if verbose {
		fmt.Printf("Writing single-file archive: %s\n", outputPath)
	}

	// Ensure .glx extension
	outputPath = ensureGLXExtension(outputPath)

	// Write archive
	if err := writeSingleFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	printSuccessSingleFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// importToMultiFile writes the imported GLX data to a multi-file directory
func importToMultiFile(glx *lib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int) error {
	if verbose {
		fmt.Printf("Writing multi-file archive: %s\n", outputPath)
	}

	// Write archive
	if err := writeMultiFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	printSuccessMultiFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}
