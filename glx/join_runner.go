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
)

// joinArchive converts a multi-file GLX archive to single-file format
func joinArchive(inputDir, outputPath string, validate, verbose bool, showFirstErrors int) error {
	// Check if input directory exists
	if !dirExists(inputDir) {
		return fmt.Errorf("%w: %s", ErrInputDirectoryNotFound, inputDir)
	}

	// Check if output file exists
	if fileExists(outputPath) {
		return fmt.Errorf("%w: %s", ErrOutputFileExists, outputPath)
	}

	// Ensure .glx extension
	outputPath = ensureGLXExtension(outputPath)

	// Load multi-file archive
	if verbose {
		fmt.Printf("Loading multi-file archive: %s\n", inputDir)
	}

	glx, duplicates, err := LoadArchiveWithOptions(inputDir, validate)
	if err != nil {
		return err
	}

	if len(duplicates) > 0 {
		fmt.Printf("Warning: found %d duplicate entity IDs\n", len(duplicates))
		for _, dup := range duplicates {
			fmt.Printf("  - %s\n", dup)
		}
	}

	if verbose {
		printVerboseArchiveStatistics(glx, "Archive loaded successfully")
	}

	// Serialize to single-file format
	if verbose {
		fmt.Printf("\nWriting single-file archive: %s\n", outputPath)
	}

	// Write archive (no validation on save since we already validated on load)
	if err := writeSingleFileArchive(outputPath, glx, false); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	printSuccessSingleFile("joined archive", outputPath)

	return nil
}
