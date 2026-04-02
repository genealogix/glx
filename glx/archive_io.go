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

// safeWriteMultiFileArchive writes a multi-file archive to a temporary directory
// first, then swaps it into place. This prevents archive destruction if the write
// fails partway through (e.g., power loss, disk full, signal).
func safeWriteMultiFileArchive(destPath string, archive *glxlib.GLXFile) error {
	// Resolve to absolute path so rename and cwd containment checks work
	// reliably for relative paths like ".". mergeArchives does the same.
	absPath, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}
	destPath = absPath

	// On Windows, a directory cannot be renamed while it is any process's cwd.
	// If our cwd is inside (or equal to) destPath, temporarily move to the
	// parent directory so the rename operations succeed.
	parentDir := filepath.Dir(destPath)
	if cwd, err := os.Getwd(); err == nil {
		if absCwd, err2 := filepath.Abs(cwd); err2 == nil {
			if rel, err3 := filepath.Rel(destPath, absCwd); err3 == nil {
				// cwd is inside destPath if rel is "." or does not start with "..".
				if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))) {
					if err := os.Chdir(parentDir); err != nil {
						return fmt.Errorf("changing to parent directory: %w", err)
					}
					defer os.Chdir(cwd) //nolint:errcheck // best-effort restore
				}
			}
		}
	}

	// Create temp dir next to the destination (same filesystem for rename)
	tmpDir, err := os.MkdirTemp(parentDir, ".glx-tmp-")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}

	// Clean up temp dir on failure
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	// Write the archive to temp
	if err := writeMultiFileArchive(tmpDir, archive, false); err != nil {
		return fmt.Errorf("writing to temp directory: %w", err)
	}

	// Create backup of the original
	backupDir := destPath + ".bak"
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("removing stale backup %s: %w", backupDir, err)
	}
	if err := os.Rename(destPath, backupDir); err != nil {
		return fmt.Errorf("backing up original: %w", err)
	}

	// Move temp into place
	if err := os.Rename(tmpDir, destPath); err != nil {
		// Restore backup on failure
		_ = os.Rename(backupDir, destPath) // best-effort restore
		return fmt.Errorf("moving archive into place: %w", err)
	}

	// Clean up backup
	_ = os.RemoveAll(backupDir)
	success = true
	return nil
}

// LoadArchive loads all GLX files from a directory with schema validation.
// This is the primary entry point for the validate command.
func LoadArchive(rootPath string) (*glxlib.GLXFile, []string, error) {
	return LoadArchiveWithOptions(rootPath, true)
}

// LoadArchiveWithOptions loads all GLX files from a directory into a single GLXFile.
// When schemaValidate is true, each file is validated against the GLX JSON schema
// before deserialization. Deserialization is delegated to DeserializeMultiFileFromMap.
func LoadArchiveWithOptions(rootPath string, schemaValidate bool) (*glxlib.GLXFile, []string, error) {
	files, err := collectGLXFilesFromDir(rootPath)
	if err != nil {
		return nil, nil, err
	}

	if schemaValidate {
		var allErrors []string
		for relPath, data := range files {
			absPath := filepath.Join(rootPath, relPath)

			doc, parseErr := ParseYAMLFile(data)
			if parseErr != nil {
				allErrors = append(allErrors, fmt.Sprintf("%s: YAML parse error: %v", absPath, parseErr))

				continue
			}

			issues := ValidateGLXFileStructure(doc)
			if len(issues) > 0 {
				allErrors = append(allErrors, fmt.Sprintf("%s:\n  - %s", absPath, strings.Join(issues, "\n  - ")))
			}
		}
		if len(allErrors) > 0 {
			return nil, nil, fmt.Errorf("%w:\n\n%s", ErrMultipleFilesFailed, strings.Join(allErrors, "\n\n"))
		}
	}

	// Pass schemaValidate to serializer to enable referential integrity validation
	serializer := createSerializer(schemaValidate, false, "")
	glx, duplicates, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return nil, nil, err
	}

	// Load standard vocabularies as defaults for any vocabulary maps not
	// already defined by the archive. This enables property reference
	// validation (e.g., born_at with reference_type: places) without
	// overwriting user-defined vocabularies.
	if err := mergeStandardVocabularies(glx); err != nil {
		return nil, nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}
	// Invalidate cached validation results from deserialization, which ran
	// before vocabularies were loaded and would miss property reference checks.
	glx.InvalidateCache()

	return glx, duplicates, nil
}

// createSerializer creates a new serializer with the specified options
func createSerializer(validate, pretty bool, indent string) *glxlib.DefaultSerializer {
	opts := &glxlib.SerializerOptions{
		Validate: validate,
		Pretty:   pretty,
		Indent:   indent,
	}

	return glxlib.NewSerializer(opts)
}

// readSingleFileArchive reads and deserializes a single-file GLX archive
func readSingleFileArchive(path string, validate bool) (*glxlib.GLXFile, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	serializer := createSerializer(validate, false, "")
	glx, err := serializer.DeserializeSingleFileBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to load archive: %w", err)
	}

	return glx, nil
}

// writeSingleFileArchive serializes and writes a single-file GLX archive
func writeSingleFileArchive(path string, glx *glxlib.GLXFile, validate bool) error {
	serializer := createSerializer(validate, true, "  ")

	yamlBytes, err := serializer.SerializeSingleFileBytes(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize GLX file: %w", err)
	}

	if err := atomicWriteFile(path, yamlBytes, filePermissions); err != nil {
		return fmt.Errorf("failed to write GLX file: %w", err)
	}

	return nil
}

// writeMultiFileArchive serializes and writes a multi-file GLX archive
func writeMultiFileArchive(dirPath string, glx *glxlib.GLXFile, validate bool) error {
	serializer := createSerializer(validate, true, "  ")

	files, err := serializer.SerializeMultiFileToMap(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize multi-file archive: %w", err)
	}

	if err := writeFilesToDir(dirPath, files); err != nil {
		return err
	}

	return nil
}

// mergeStandardVocabularies loads standard vocabularies into a GLXFile,
// filling only empty maps. User-defined vocabularies are preserved.
func mergeStandardVocabularies(glx *glxlib.GLXFile) error {
	std := &glxlib.GLXFile{}
	if err := glxlib.LoadStandardVocabulariesIntoGLX(std); err != nil {
		return err
	}

	if len(glx.EventTypes) == 0 {
		glx.EventTypes = std.EventTypes
	}
	if len(glx.RelationshipTypes) == 0 {
		glx.RelationshipTypes = std.RelationshipTypes
	}
	if len(glx.PlaceTypes) == 0 {
		glx.PlaceTypes = std.PlaceTypes
	}
	if len(glx.SourceTypes) == 0 {
		glx.SourceTypes = std.SourceTypes
	}
	if len(glx.RepositoryTypes) == 0 {
		glx.RepositoryTypes = std.RepositoryTypes
	}
	if len(glx.ParticipantRoles) == 0 {
		glx.ParticipantRoles = std.ParticipantRoles
	}
	if len(glx.MediaTypes) == 0 {
		glx.MediaTypes = std.MediaTypes
	}
	if len(glx.ConfidenceLevels) == 0 {
		glx.ConfidenceLevels = std.ConfidenceLevels
	}
	if len(glx.GenderTypes) == 0 {
		glx.GenderTypes = std.GenderTypes
	}
	if len(glx.PersonProperties) == 0 {
		glx.PersonProperties = std.PersonProperties
	}
	if len(glx.EventProperties) == 0 {
		glx.EventProperties = std.EventProperties
	}
	if len(glx.RelationshipProperties) == 0 {
		glx.RelationshipProperties = std.RelationshipProperties
	}
	if len(glx.PlaceProperties) == 0 {
		glx.PlaceProperties = std.PlaceProperties
	}
	if len(glx.MediaProperties) == 0 {
		glx.MediaProperties = std.MediaProperties
	}
	if len(glx.RepositoryProperties) == 0 {
		glx.RepositoryProperties = std.RepositoryProperties
	}
	if len(glx.CitationProperties) == 0 {
		glx.CitationProperties = std.CitationProperties
	}
	if len(glx.SourceProperties) == 0 {
		glx.SourceProperties = std.SourceProperties
	}

	return nil
}
