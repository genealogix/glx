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
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

var errJSONValidationScope = errors.New("JSON validation requires one complete archive directory or self-contained archive file; fragments, multiple paths and stdin are not supported")

type validationJSONResult struct {
	Valid            bool                       `json:"valid"`
	StructuralErrors []string                   `json:"structural_errors"`
	Errors           []glxlib.ValidationError   `json:"errors"`
	Warnings         []glxlib.ValidationWarning `json:"warnings"`
	DuplicateIDs     []string                   `json:"duplicate_ids"`
	Report           *reportData                `json:"report,omitempty"`
	Scope            string                     `json:"scope"`
}

func emptyValidationJSON(scope string) *validationJSONResult {
	return &validationJSONResult{StructuralErrors: []string{}, Errors: []glxlib.ValidationError{}, Warnings: []glxlib.ValidationWarning{}, DuplicateIDs: []string{}, Scope: scope}
}

// buildValidationJSON checks a loaded typed snapshot, not its original files.
// Unknown source fields discarded during loading cannot be reconstructed.
func buildValidationJSON(archive *glxlib.GLXFile, report bool) (*validationJSONResult, error) {
	return buildValidationJSONContext(context.Background(), archive, report)
}

func buildValidationJSONContext(ctx context.Context, archive *glxlib.GLXFile, report bool) (*validationJSONResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	view := *archive
	if err := glxlib.MergeStandardVocabularies(&view); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := yaml.Marshal(&view)
	if err != nil {
		return nil, fmt.Errorf("serialize validation snapshot: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	doc, err := ParseYAMLFile(data)
	if err != nil {
		return nil, fmt.Errorf("parse validation snapshot: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := emptyValidationJSON("typed_snapshot; unknown fields discarded at load time are not checked")
	result.StructuralErrors = append(result.StructuralErrors, ValidateGLXFileStructure(doc)...)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	findings := view.Validate()
	result.Errors = append(result.Errors, findings.Errors...)
	result.Warnings = append(result.Warnings, findings.Warnings...)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if report {
		result.Report = buildConfidenceJSONReport(&view)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result.Valid = len(result.StructuralErrors) == 0 && len(result.Errors) == 0

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func showValidationJSON(args []string, report bool) error {
	if len(args) > 1 || (len(args) == 1 && args[0] == "-") {
		return errJSONValidationScope
	}
	path := "."
	if len(args) == 1 {
		path = args[0]
	}
	result, err := validateJSONPath(path, report)
	if err != nil {
		return err
	}
	if err := printArchiveAnalysisJSON(result); err != nil {
		return err
	}
	if !result.Valid {
		return ErrValidationWithErrors
	}

	return nil
}

func validateJSONPath(path string, report bool) (*validationJSONResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("access archive: %w", err)
	}
	var files map[string][]byte
	if info.IsDir() {
		if root, rootErr := commonArchiveRoot([]string{path}); rootErr == nil && !sameDirectory(root, path) {
			return nil, errJSONValidationScope
		}
		files = make(map[string][]byte)
		err = walkGLXFilesUnderReading(path, ".", readMCPWatchFile, func(name string, data []byte, readErr error) error {
			if readErr != nil {
				return fmt.Errorf("read validation source %s: %w", name, readErr)
			}
			files[name] = data

			return nil
		})
	} else {
		// Match text validation's file selection. The multi-file loader skips
		// unsupported extensions, which must not turn this explicit input
		// into a successfully validated empty archive.
		if !isGLXFile(path) {
			return nil, fmt.Errorf("%w: %s is not a .glx file", ErrNothingToValidate, path)
		}
		if !info.Mode().IsRegular() {
			return nil, errMCPWatchSourceType
		}
		var data []byte
		data, err = readValidationJSONFile(path)
		files = map[string][]byte{path: data}
	}
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, ErrNothingToValidate
	}
	result := emptyValidationJSON("source_files_and_archive_references")
	result.StructuralErrors, err = validateJSONSourceFiles(files, info.IsDir())
	if err != nil {
		return nil, err
	}
	archive, duplicates, loadErr := loadArchiveFromFiles(path, files, false)
	if loadErr != nil {
		result.StructuralErrors = append(result.StructuralErrors, fmt.Sprintf("load archive: %v", loadErr))

		return result, nil
	}
	typed, err := buildValidationJSON(archive, report)
	if err != nil {
		return nil, err
	}
	result.Errors, result.Warnings, result.Report = typed.Errors, typed.Warnings, typed.Report
	result.DuplicateIDs = append(result.DuplicateIDs, duplicates...)
	sort.Strings(result.DuplicateIDs)
	result.Valid = len(result.StructuralErrors) == 0 && len(result.Errors) == 0 && len(result.DuplicateIDs) == 0

	return result, nil
}

// readValidationJSONFile preserves explicit-path symlink support while rejecting
// special files before reading. Nonblocking open also closes the Unix FIFO
// replacement race; the descriptor check does not rely on the earlier stat.
func readValidationJSONFile(path string) ([]byte, error) {
	// #nosec G304 -- explicit CLI input; MCP validation never reads paths.
	file, err := os.OpenFile(path, os.O_RDONLY|mcpWatchNonblock, 0)
	if err != nil {
		return nil, err
	}

	return readMCPWatchRegularFile(file, nil)
}

func validateJSONSourceFiles(files map[string][]byte, directory bool) ([]string, error) {
	issues := []string{}
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		doc, parseErr := ParseYAMLFile(files[name])
		if parseErr != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", name, parseErr))

			continue
		}
		if !directory && !isSelfContainedArchive(doc) {
			return nil, errJSONValidationScope
		}
		for _, issue := range ValidateGLXFileStructure(doc) {
			issues = append(issues, fmt.Sprintf("%s: %s", name, issue))
		}
	}

	return issues, nil
}

// buildConfidenceJSONReport preserves the CLI confidence report's semantics,
// with empty collections encoded as arrays rather than null.
func buildConfidenceJSONReport(archive *glxlib.GLXFile) *reportData {
	report := buildConfidenceReport(archive)
	if report.NoConfidence == nil {
		report.NoConfidence = []assertionSummary{}
	}
	if report.NoCitations == nil {
		report.NoCitations = []assertionSummary{}
	}
	if report.UnbackedPersons == nil {
		report.UnbackedPersons = []string{}
	}
	if report.UnbackedEvents == nil {
		report.UnbackedEvents = []string{}
	}
	if report.UnbackedRelations == nil {
		report.UnbackedRelations = []string{}
	}
	if report.ConfidenceOrder == nil {
		report.ConfidenceOrder = []string{}
	}

	return &report
}
