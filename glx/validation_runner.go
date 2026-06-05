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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
	"gopkg.in/yaml.v3"
)

// errStdinUnknownEntityType is returned when --entity-type is missing or not in
// the allow-list derived from glxlib.AllEntityTypes (plus "vocabulary-entry").
var errStdinUnknownEntityType = errors.New("--stdin requires a valid --entity-type")

// collectionForEntityType maps a singular --entity-type flag value to the
// top-level GLXFile collection key it lives under. Entity singulars are taken
// from glxlib so the allow-list stays in sync as entity types are added; the
// "vocabulary-entry" pseudo-type maps to any VocabularyEntry collection
// (their structural shape is identical) so vocab snippets can be checked too.
func collectionForEntityType(flag string) (string, bool) {
	flag = strings.TrimSpace(strings.ToLower(flag))
	if flag == "" {
		return "", false
	}
	if flag == "vocabulary-entry" {
		return glxlib.EntityTypeEvents.Singular() + "_types", true // "event_types"
	}
	for _, et := range glxlib.AllEntityTypes {
		if et.Singular() == flag {
			return et.String(), true
		}
	}
	return "", false
}

// validateStdinEntity reads one entity as YAML from stdin and structurally
// validates it against its entity-type schema, without any archive/cross-ref
// context. It exists so drift tooling can pipe a bare snippet in (issue #910)
// instead of the mktemp/cat/rm temp-file dance.
func validateStdinEntity(streams *IOStreams, entityType string, args []string) error {
	if len(args) > 0 {
		return errors.New("--stdin does not take path arguments")
	}
	collection, ok := collectionForEntityType(entityType)
	if !ok {
		return fmt.Errorf("%w: got %q", errStdinUnknownEntityType, entityType)
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return errors.New("--stdin: no YAML on stdin")
	}

	var entity any
	if err := yaml.Unmarshal(data, &entity); err != nil {
		return fmt.Errorf("parsing stdin YAML: %w", err)
	}

	// Wrap the bare entity under its collection so the existing whole-archive
	// structural validator (which resolves the schema $refs) can check it.
	doc := map[string]any{collection: map[string]any{"stdin": entity}}
	issues := ValidateGLXFileStructure(doc)
	if len(issues) > 0 {
		streams.Errorf("Found %d structural error(s) in the %s entity:\n", len(issues), entityType)
		for _, issue := range issues {
			streams.Errorf("- %s\n", issue)
		}

		return ErrStructuralValidationFailed
	}

	streams.Printf("%s entity is structurally valid.\n", entityType)

	return nil
}

// validatePaths performs comprehensive validation on the specified paths.
// Output goes to the provided IOStreams (stdout for results, stderr for errors).
//
//nolint:gocognit,gocyclo // path-validation orchestration has many branches
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
		// on the single file, filtering out cross-reference issues.
		semanticErrors, semanticWarnings := validateSingleFileSemantics(paths)

		streams.Println("⚠️  Cross-reference validation skipped (single file specified).")
		streams.Printf("%d files validated.\n", fileCount)

		if len(semanticWarnings) > 0 {
			streams.Errorf("Found %d warnings:\n", len(semanticWarnings))
			for _, warn := range semanticWarnings {
				streams.Errorf("- ⚠️  %s\n", warn)
			}
		}

		if len(semanticErrors) > 0 {
			streams.Errorf("Found %d errors:\n", len(semanticErrors))
			for _, issue := range semanticErrors {
				streams.Errorf("- ❌ %s\n", issue)
			}

			return ErrValidationFailed
		}

		streams.Println("✅ File passed structural and semantic validation (cross-references skipped).")

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

	if fileCount == 0 {
		streams.Println("No GLX files found. Validated 0 files.")
	} else {
		streams.Printf("Validated %d files.\n", fileCount)
	}
	if len(allWarnings) > 0 {
		streams.Errorf("Found %d warnings:\n", len(allWarnings))
		for _, warn := range allWarnings {
			streams.Errorf("- ⚠️  %s\n", warn)
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
// Returns errors (fatal) and warnings (informational) separately, consistent
// with directory validation behavior.
func validateSingleFileSemantics(paths []string) ([]string, []string) {
	var allErrors, allWarnings []string

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
				if isSingleFileIssue(ve.Message) {
					allErrors = append(allErrors, ve.Message)
				}
			}

			for _, warn := range result.Warnings {
				if isSingleFileIssue(warn.Message) {
					allWarnings = append(allWarnings, warn.Message)
				}
			}

			return nil
		})
	}

	return allErrors, allWarnings
}

// isSingleFileIssue returns true for validation errors/warnings that can be
// detected on a single file without the full archive context. Uses a blacklist
// approach: keep all semantic issues except known cross-archive reference errors
// and place hierarchy cycles. This ensures new go-glx checks are automatically
// included without needing whitelist updates.
func isSingleFileIssue(msg string) bool {
	lower := strings.ToLower(msg)

	// Exclude cross-entity reference errors (e.g., "references non-existent person: ...")
	if strings.Contains(lower, "references non-existent") {
		return false
	}

	// Exclude place hierarchy cycle detection (requires full archive)
	if strings.Contains(lower, "cycle detected") {
		return false
	}

	return true
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
