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
)

// validatePaths performs comprehensive validation on the specified paths
func validatePaths(args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var allErrors, allWarnings []string
	var fileCount int

	// First pass: structural validation of all files
	for _, path := range paths {
		err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			if !isGLXFile(d.Name()) {
				return nil
			}

			fileCount++
			data, err := os.ReadFile(filePath)
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error reading %s: %v", filePath, err))

				return nil // Continue to next file
			}

			doc, err := ParseYAMLFile(data)
			if err != nil {
				allErrors = append(allErrors, fmt.Sprintf("Error parsing YAML in %s: %v", filePath, err))

				return nil // Continue
			}

			issues := ValidateGLXFileStructure(doc)
			if len(issues) > 0 {
				for _, issue := range issues {
					allErrors = append(allErrors, fmt.Sprintf("Error in %s: %s", filePath, issue))
				}
			}

			return nil
		})
		if err != nil {
			// This would be an error from WalkDir itself, not a validation error
			fmt.Fprintf(os.Stderr, "Error walking directory %s: %v\n", path, err)
		}
	}

	if len(allErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Found %d structural errors in %d files:\n", len(allErrors), fileCount)
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "- %s\n", err)
		}

		return ErrStructuralValidationFailed
	}

	// Second pass: load and cross-reference validation
	// We assume a single archive root for simplicity here. A more robust implementation
	// might handle multiple disconnected roots.
	archiveRoot := "."
	if len(paths) == 1 {
		info, err := os.Stat(paths[0])
		if err == nil && info.IsDir() {
			archiveRoot = paths[0]
		}
	}

	archive, duplicates, err := LoadArchive(archiveRoot)
	if err != nil {
		// This error comes from LoadArchive if a file fails validation during load
		fmt.Fprintf(os.Stderr, "Error loading archive: %v\n", err)

		return err
	}

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

	fmt.Printf("Validated %d files.\n", fileCount)
	if len(allWarnings) > 0 {
		fmt.Printf("Found %d warnings:\n", len(allWarnings))
		for _, warn := range allWarnings {
			fmt.Printf("- ⚠️  %s\n", warn)
		}
	}

	if len(allErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Found %d errors:\n", len(allErrors))
		for _, err := range allErrors {
			fmt.Fprintf(os.Stderr, "- ❌ %s\n", err)
		}

		return ErrValidationFailed
	}

	fmt.Println("✅ Archive is valid.")

	return nil
}
