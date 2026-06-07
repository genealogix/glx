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
	"reflect"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Sentinel errors for the --stdin path (static, per err113: no inline dynamic errors).
var (
	// errStdinUnknownEntityType is returned when --entity-type is missing or not
	// in the allow-list (entity singulars from glxlib.AllEntityTypes plus the
	// GLXFile vocabulary collection keys).
	errStdinUnknownEntityType = errors.New("--stdin requires a valid --entity-type")
	errStdinPathArgs          = errors.New("--stdin does not take path arguments")
	// errStdinEmpty is a usage error (exit 2): nothing was piped, so the caller
	// invoked --stdin without feeding it anything. This is deliberately distinct
	// from malformed YAML (exit 1) — there content *was* piped but is unparseable,
	// which is a content failure on a par with a structural validation failure.
	errStdinEmpty = errors.New("--stdin: no YAML provided")
)

// vocabKeysOnce memoizes the GLXFile reflection done by vocabularyCollectionKeys.
var (
	vocabKeysOnce sync.Once
	vocabKeys     map[string]bool
)

// vocabularyCollectionKeys returns the set of GLXFile yaml keys whose values are
// map[string]*VocabularyEntry — every vocabulary a --stdin snippet can be
// validated against. It reflects over GLXFile so vocabularies added there are
// accepted automatically, with no parallel list to drift (the same
// source-of-truth the drift skills read).
func vocabularyCollectionKeys() map[string]bool {
	vocabKeysOnce.Do(func() {
		vocabKeys = map[string]bool{}
		want := reflect.TypeFor[map[string]*glxlib.VocabularyEntry]()
		for _, f := range reflect.VisibleFields(reflect.TypeFor[glxlib.GLXFile]()) {
			if f.Type != want {
				continue
			}
			name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
			if name != "" && name != "-" {
				vocabKeys[name] = true
			}
		}
	})

	return vocabKeys
}

// collectionForEntityType maps an --entity-type flag value to the top-level
// GLXFile collection key it lives under, or returns false if unrecognized. Two
// forms are accepted: an entity singular (person, event, …), mapped to its
// plural collection via glxlib.AllEntityTypes; and a vocabulary collection key
// (event_types, place_types, …), validated as itself so each vocabulary entry
// is checked against its own schema — e.g. a place_types entry against the
// place-type category enum rather than the event-type one.
func collectionForEntityType(flag string) (string, bool) {
	flag = strings.TrimSpace(strings.ToLower(flag))
	if flag == "" {
		return "", false
	}
	for _, et := range glxlib.AllEntityTypes {
		if et.Singular() == flag {
			return et.String(), true
		}
	}
	if vocabularyCollectionKeys()[flag] {
		return flag, true
	}

	return "", false
}

// validateStdinEntity reads one entity as YAML from in (stdin, in production)
// and structurally validates it against its entity-type schema, without any
// archive/cross-ref context. It exists so drift tooling can pipe a bare snippet
// in (issue #910) instead of the mktemp/cat/rm temp-file dance.
func validateStdinEntity(streams *IOStreams, entityType string, args []string, in io.Reader) error {
	if len(args) > 0 {
		return errStdinPathArgs
	}
	// Resolve the entity type before consuming stdin, so a typo'd --entity-type
	// fails fast instead of reading and buffering the whole stream first. The
	// resolved collection is then handed to validateSnippetInCollection so the
	// success path does not look it up a second time.
	collection, ok := collectionForEntityType(entityType)
	if !ok {
		return fmt.Errorf("%w: got %q", errStdinUnknownEntityType, entityType)
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	issues, err := validateSnippetInCollection(collection, data)
	if err != nil {
		return err
	}
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

// validateEntitySnippet is the entity-type-keyed entry point to --stdin
// validation: it maps the entity-type to its collection and delegates to
// validateSnippetInCollection. It returns the structural issues (empty == valid)
// or an error for unusable input (unknown entity-type, empty, or malformed YAML).
// validateStdinEntity resolves the collection itself (to fail fast before
// reading stdin) and calls validateSnippetInCollection directly, so it never
// re-resolves; this wrapper exists for callers — including tests — that hold
// only the type name.
func validateEntitySnippet(entityType string, data []byte) ([]string, error) {
	collection, ok := collectionForEntityType(entityType)
	if !ok {
		return nil, fmt.Errorf("%w: got %q", errStdinUnknownEntityType, entityType)
	}

	return validateSnippetInCollection(collection, data)
}

// validateSnippetInCollection is the resolution-free core: given an
// already-resolved top-level GLXFile collection key, it parses one entity from
// the YAML bytes, wraps it as a single-entity archive, and runs the structural
// validator. Empty input is a usage error (errStdinEmpty); malformed YAML is a
// content error (wrapped parse error). Splitting collection resolution out lets
// validateStdinEntity reuse the collection it already resolved for its fast-fail.
func validateSnippetInCollection(collection string, data []byte) ([]string, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errStdinEmpty
	}

	var entity any
	if err := yaml.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Wrap the bare entity under its collection so the existing whole-archive
	// structural validator (which resolves the schema $refs) can check it.
	doc := map[string]any{collection: map[string]any{"stdin": entity}}

	return ValidateGLXFileStructure(doc), nil
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
