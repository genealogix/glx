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

	glxlib "github.com/genealogix/glx/go-glx"
)

// importGEDCOM imports a GEDCOM file and converts it to GLX format
func importGEDCOM(gedcomPath, outputPath, format string, validate, verbose bool, showFirstErrors int) error {
	// Validate format flag
	if format != FormatSingle && format != FormatMulti {
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
	glx, result, err := glxlib.ImportGEDCOM(gedcomFile, nil)
	if err != nil {
		return fmt.Errorf("failed to import GEDCOM: %w", err)
	}

	// Determine GEDCOM source directory for resolving relative media file paths
	gedcomDir := filepath.Dir(gedcomPath)

	// Serialize based on format
	if format == FormatSingle {
		return importToSingleFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, gedcomDir)
	}

	return importToMultiFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, gedcomDir)
}

// importToSingleFile writes the imported GLX data to a single file
func importToSingleFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, gedcomDir string) error {
	if verbose {
		fmt.Printf("Writing single-file archive: %s\n", outputPath)
	}

	// Ensure .glx extension
	outputPath = ensureGLXExtension(outputPath)

	// Write archive
	if err := writeSingleFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	// Copy media files into sibling media/files/ directory
	archiveDir := filepath.Dir(outputPath)
	if err := copyMediaFiles(archiveDir, mediaFiles, gedcomDir, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	printSuccessSingleFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// importToMultiFile writes the imported GLX data to a multi-file directory
func importToMultiFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, gedcomDir string) error {
	if verbose {
		fmt.Printf("Writing multi-file archive: %s\n", outputPath)
	}

	// Write archive
	if err := writeMultiFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	// Copy media files into the archive's media/files/ directory
	if err := copyMediaFiles(outputPath, mediaFiles, gedcomDir, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	printSuccessMultiFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}
