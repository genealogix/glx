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
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// archiveMetadataFile is the top-level file the multi-file serializer writes the
// archive's metadata block to.
const archiveMetadataFile = "metadata.glx"

// archiveManagedTopLevel is the set of top-level names the multi-file serializer
// writes: the entity directories, vocabularies/, and metadata.glx. Validation
// uses it to recognize an archive root from inside one of its directories.
// Derived from glxlib.AllEntityTypes so new entity types are picked up
// automatically.
var archiveManagedTopLevel = func() map[string]bool {
	m := map[string]bool{
		archiveMetadataFile:           true,
		glxlib.ArchiveDirVocabularies: true,
	}
	for _, entityType := range glxlib.AllEntityTypes {
		m[entityType.String()] = true
	}

	return m
}()

// isManagedTopLevel reports whether a top-level archive entry is one the
// serializer writes: a real directory whose name matches a managed directory
// case-insensitively (PERSONS/ counts, #1246), or the metadata file by its
// exact name. A regular file that happens to be named PERSONS, a symlink named
// Persons, or a directory named Metadata.glx is not.
func isManagedTopLevel(entry fs.DirEntry) bool {
	if entry.Type().IsDir() {
		return archiveManagedTopLevel[strings.ToLower(entry.Name())] && strings.ToLower(entry.Name()) != archiveMetadataFile
	}

	return entry.Type().IsRegular() && entry.Name() == archiveMetadataFile
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

	return loadArchiveFromFiles(rootPath, files, schemaValidate)
}

// loadArchiveFromFiles builds an archive from an already-collected file map
// (relative path -> contents, as returned by collectGLXFilesFromDir). It is
// the half of LoadArchiveWithOptions that does no I/O of its own, split out so
// a caller that needs to know how many files the archive contains can read it
// off the map instead of walking the tree a second time with its own copy of
// the "what is archive content" rules — two walks that had already drifted
// apart on a symlinked archive root.
func loadArchiveFromFiles(rootPath string, files map[string][]byte, schemaValidate bool) (*glxlib.GLXFile, []string, error) {
	if schemaValidate {
		var fileErrors []fileValidationIssues
		for _, relPath := range slices.Sorted(maps.Keys(files)) {
			data := files[relPath]
			absPath := filepath.Join(rootPath, relPath)

			doc, parseErr := ParseYAMLFile(data)
			if parseErr != nil {
				fileErrors = append(fileErrors, fileValidationIssues{
					path:       absPath,
					issues:     []string{fmt.Sprintf("YAML parse error: %v", parseErr)},
					parseError: true,
				})

				continue
			}

			issues := ValidateGLXFileStructure(doc)
			if len(issues) > 0 {
				fileErrors = append(fileErrors, fileValidationIssues{path: absPath, issues: issues})
			}
		}
		if len(fileErrors) > 0 {
			return nil, nil, &archiveFilesValidationError{files: fileErrors}
		}
	}

	// Pass schemaValidate to serializer to enable referential integrity validation
	serializer := createSerializer(schemaValidate, false, "")
	glx, duplicates, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return nil, nil, err
	}
	duplicates = sanitizeDuplicateWarnings(duplicates)

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

// sanitizeDuplicateWarnings makes deserializer duplicate warnings safe to
// print. The warnings quote archive-controlled entity IDs and file names and
// exist only to be shown as diagnostics — two dozen runners write them to
// stderr, several with a bare fmt.Fprintf. Every caller of
// DeserializeMultiFileFromMap must pass its duplicates through here so no
// print site can leak a control sequence (genealogix/glx#925). The slice is
// sanitized in place and returned for convenience.
func sanitizeDuplicateWarnings(duplicates []string) []string {
	for i, d := range duplicates {
		duplicates[i] = sanitizeForTerminal(d)
	}

	return duplicates
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

// writeSingleFileArchive serializes and writes a single-file GLX archive with
// the default file permissions. Callers rewriting a file that already exists
// should use writeSingleFileArchiveWithMode instead.
func writeSingleFileArchive(path string, glx *glxlib.GLXFile, validate bool) error {
	return writeSingleFileArchiveWithMode(path, glx, validate, filePermissions)
}

// writeSingleFileArchiveWithMode is writeSingleFileArchive with an explicit
// permission mode. In-place rewrites pass the mode the file already has so a
// deliberately private archive (0600, say) does not come back world-readable
// because the write went through a fresh temp file.
func writeSingleFileArchiveWithMode(path string, glx *glxlib.GLXFile, validate bool, perm os.FileMode) error {
	serializer := createSerializer(validate, true, "  ")

	yamlBytes, err := serializer.SerializeSingleFileBytes(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize GLX file: %w", err)
	}

	if err := atomicWriteFile(path, yamlBytes, perm); err != nil {
		return fmt.Errorf("failed to write GLX file: %w", err)
	}

	return nil
}

// serializeMultiFileArchive serializes (and, when validate is set, validates) a
// multi-file GLX archive, returning the files to write without touching the
// filesystem. Split out from writeMultiFileArchive so callers with other side
// effects to commit can fail on serialization or validation *before* any of
// them happen — see the GEDZIP import path, which stages media first.
func serializeMultiFileArchive(glx *glxlib.GLXFile, validate bool) (map[string][]byte, error) {
	serializer := createSerializer(validate, true, "  ")

	files, err := serializer.SerializeMultiFileToMap(glx)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize multi-file archive: %w", err)
	}

	return files, nil
}

// writeMultiFileArchive serializes and writes a multi-file GLX archive
func writeMultiFileArchive(dirPath string, glx *glxlib.GLXFile, validate bool) error {
	files, err := serializeMultiFileArchive(glx, validate)
	if err != nil {
		return err
	}

	if err := writeFilesToDir(dirPath, files); err != nil {
		return err
	}

	return nil
}

// writePartialArchive serializes a partial GLXFile (containing only newly
// minted entities) and writes just the entity files into an existing archive
// directory — vocabulary and metadata output from the serializer is dropped so
// the existing archive's definitions are preserved. Used by commands that
// append to an archive rather than rewrite it. Returns the number of entity
// files written.
func writePartialArchive(dirPath string, partial *glxlib.GLXFile) (int, error) {
	serializer := createSerializer(false, true, "  ")

	files, err := serializer.SerializeMultiFileToMap(partial)
	if err != nil {
		return 0, fmt.Errorf("serializing partial archive: %w", err)
	}

	entityFiles := make(map[string][]byte, len(files))
	for relPath, data := range files {
		// SerializeMultiFileToMap emits forward-slash keys on every OS
		// (issue #900) — no need to check for OS-specific separators.
		if strings.HasPrefix(relPath, "vocabularies/") {
			continue
		}
		if relPath == archiveMetadataFile {
			continue
		}
		entityFiles[relPath] = data
	}

	if err := writeFilesToDir(dirPath, entityFiles); err != nil {
		return 0, err
	}

	return len(entityFiles), nil
}

// mergeStandardVocabularies loads standard vocabularies into a GLXFile,
// filling only empty maps. User-defined vocabularies are preserved.
//
//nolint:gocyclo // 21 vocab fields with identical empty-map guards; splitting hurts readability
func mergeStandardVocabularies(archive *glxlib.GLXFile) error {
	return glxlib.MergeStandardVocabularies(archive)
}
