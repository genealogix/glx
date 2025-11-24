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

// splitArchive converts a single-file GLX archive to multi-file format
func splitArchive(inputPath, outputDir string, validate, verbose bool, showFirstErrors int) error {
	// Check if input file exists
	if !fileExists(inputPath) {
		return fmt.Errorf("%w: %s", ErrInputFileNotFound, inputPath)
	}

	// Check if output directory exists
	if fileExists(outputDir) {
		return fmt.Errorf("%w: %s", ErrOutputDirectoryExists, outputDir)
	}

	// Load single-file archive
	if verbose {
		fmt.Printf("Loading archive: %s\n", inputPath)
	}

	glx, err := readSingleFileArchive(inputPath, validate)
	if err != nil {
		return err
	}

	if verbose {
		printVerboseArchiveStatistics(glx, "Archive loaded successfully")
	}

	// Serialize to multi-file format
	if verbose {
		fmt.Printf("\nWriting multi-file archive: %s\n", outputDir)
	}

	// Write archive (no validation on save since we already validated on load)
	if err := writeMultiFileArchive(outputDir, glx, false); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	printSuccessMultiFile("split archive", outputDir)

	return nil
}
