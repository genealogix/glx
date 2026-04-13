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
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// validatePaths performs comprehensive validation on the specified paths.
// Output goes to the provided IOStreams (stdout for results, stderr for errors).
//
//nolint:gocognit,gocyclo
func validatePaths(streams *IOStreams, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Determine archive root and validation mode
	var archiveRoot string
	var shouldValidateCrossRefs bool

	if len(paths) == 1 {
		if info, err := os.Stat(paths[0]); err == nil {
			if info.IsDir() {
				archiveRoot = paths[0]
				shouldValidateCrossRefs = true
			}
		}
	} else if len(paths) == 0 {
		archiveRoot = "."
		shouldValidateCrossRefs = true
	} else {
		if info, err := os.Stat(paths[0]); err == nil {
			if info.IsDir() {
				archiveRoot = paths[0]
			} else {
				archiveRoot = filepath.Dir(paths[0])
			}
			shouldValidateCrossRefs = true
		}
	}

	// Single file: structural validation + semantic checks (no cross-references)
	if !shouldValidateCrossRefs {
		fileCount, structErrors := validateSingleFilePaths(paths)
		if len(structErrors) > 0 {
			streams.Errorf("Found %d structural errors in %d files:\n", len(structErrors), fileCount)
			for _, err := range structErrors {
				streams.Errorf("- %s\n", err)
			}

			return ErrStructuralValidationFailed
		}

		// Run semantic validation (deprecated properties, date formats, etc.)
		// on the single file, filtering out cross-reference errors.
		semanticErrors := validateSingleFileSemantics(paths)

		streams.Println("⚠️  Cross-reference validation skipped (single file specified).")
		streams.Printf("Validated %d file(s).\n", fileCount)

		if len(semanticErrors) > 0 {
			streams.Errorf("Found %d errors:\n", len(semanticErrors))
			for _, err := range semanticErrors {
				streams.Errorf("- ❌ %s\n", err)
			}

			return ErrValidationFailed
		}

		streams.Println("✅ File structure is valid.")

		return nil
	}

	// Directory: single-pass load with schema validation + cross-reference checks.
	// LoadArchiveWithOptions(true) reads each file once, runs JSON schema validation,
	// then deserializes into Go structs — avoiding the previous double file-read.
	fileCount := countGLXFiles(archiveRoot)

	archive, duplicates, err := LoadArchiveWithOptions(archiveRoot, true)
	if err != nil {
		formatted := formatValidationError(err, defaultShowFirstErrors)
		streams.Errorf("Error loading archive: %v\n", formatted)

		return ErrStructuralValidationFailed
	}

	var allErrors, allWarnings []string

	if len(duplicates) > 0 {
		allErrors = append(allErrors, duplicates...)
	}

	result := archive.Validate()

	for _, warn := range result.Warnings {
		allWarnings = append(allWarnings, warn.Message)
	}
	for _, err := range result.Errors {
		allErrors = append(allErrors, err.Message)
	}

	// Check media file existence on disk
	allWarnings = append(allWarnings, validateMediaFileExistence(archive, archiveRoot)...)

	streams.Printf("Validated %d files.\n", fileCount)
	if len(allWarnings) > 0 {
		streams.Printf("Found %d warnings:\n", len(allWarnings))
		for _, warn := range allWarnings {
			streams.Printf("- ⚠️  %s\n", warn)
		}
	}

	if len(allErrors) > 0 {
		streams.Errorf("Found %d errors:\n", len(allErrors))
		for _, err := range allErrors {
			streams.Errorf("- ❌ %s\n", err)
		}

		return ErrValidationFailed
	}

	streams.Println("✅ Archive is valid.")

	return nil
}

// validateSingleFilePaths runs structural validation on individual files
// (used when a single file is specified, not a directory).
func validateSingleFilePaths(paths []string) (int, []string) {
	var allErrors []string
	var fileCount int

	for _, path := range paths {
		_ = filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !isGLXFile(d.Name()) {
				return nil
			}

			fileCount++
			filePath = filepath.Clean(filePath)
			data, err := os.ReadFile(filePath)
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error reading %s: %v", filePath, err))

				return nil
			}

			doc, err := ParseYAMLFile(data)
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error parsing YAML in %s: %v", filePath, err))

				return nil
			}

			issues := ValidateGLXFileStructure(doc)
			for _, issue := range issues {
				allErrors = append(allErrors, fmt.Sprintf("Error in %s: %s", filePath, issue))
			}

			return nil
		})
	}

	return fileCount, allErrors
}

// validateSingleFileSemantics runs semantic validation (deprecated properties,
// date formats, property types) on single files. Cross-reference errors are
// filtered out since we don't have the full archive context.
func validateSingleFileSemantics(paths []string) []string {
	var allErrors []string

	for _, path := range paths {
		_ = filepath.WalkDir(path, func(filePath string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !isGLXFile(d.Name()) {
				return nil
			}

			archive, loadErr := readSingleFileArchive(filePath, false)
			if loadErr != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error loading %s for semantic validation: %v", filePath, loadErr))

				return nil
			}

			if mergeErr := mergeStandardVocabularies(archive); mergeErr != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error loading vocabularies for %s: %v", filePath, mergeErr))

				return nil
			}

			archive.InvalidateCache()
			result := archive.Validate()

			for _, ve := range result.Errors {
				// Keep only errors that work on single files without cross-references
				if isSingleFileError(ve.Message) {
					allErrors = append(allErrors, ve.Message)
				}
			}

			return nil
		})
	}

	return allErrors
}

// isSingleFileError returns true for validation errors that can be detected on a
// single file without the full archive context. This is a whitelist approach —
// only known single-file-compatible error patterns are included.
func isSingleFileError(msg string) bool {
	return strings.Contains(msg, "has been removed") || // deprecated properties
		strings.Contains(msg, "invalid date format") || // date format errors
		strings.Contains(msg, "unknown property") // property type errors
}

// countGLXFiles counts .glx files in a directory without reading them.
func countGLXFiles(root string) int {
	var count int
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && isGLXFile(d.Name()) {
			count++
		}

		return nil
	})

	return count
}

// validateMediaFileExistence checks that media entities with local relative URIs
// point to files that actually exist on disk. Returns warnings for missing files.
func validateMediaFileExistence(archive *glxlib.GLXFile, archiveRoot string) []string {
	var warnings []string
	for mediaID, media := range archive.Media {
		if !isLocalMediaURI(media.URI) {
			continue
		}
		filePath := filepath.Join(archiveRoot, media.URI)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			warnings = append(warnings, fmt.Sprintf(
				"media[%s]: referenced file does not exist: %s", mediaID, media.URI))
		}
	}

	return warnings
}

// isLocalMediaURI returns true if a URI is a local relative path (not a URL,
// absolute path, or empty string) that should exist on disk.
func isLocalMediaURI(uri string) bool {
	if uri == "" {
		return false
	}
	if strings.Contains(uri, "://") || strings.HasPrefix(uri, "mailto:") {
		return false
	}
	if strings.HasPrefix(uri, "/") {
		return false
	}
	if len(uri) >= 2 && uri[1] == ':' {
		return false
	}

	return true
}
