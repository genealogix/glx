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

	"github.com/genealogix/glx/glx/lib"
	"gopkg.in/yaml.v3"
)

// LoadArchive loads and merges all GLX files from a directory into a single GLXFile struct
// Moved from validator.go to centralize all archive I/O operations
func LoadArchive(rootPath string) (*lib.GLXFile, []string, error) {
	merged := &lib.GLXFile{
		Persons:       make(map[string]*lib.Person),
		Relationships: make(map[string]*lib.Relationship),
		Events:        make(map[string]*lib.Event),
		Places:        make(map[string]*lib.Place),
		Sources:       make(map[string]*lib.Source),
		Citations:     make(map[string]*lib.Citation),
		Repositories:  make(map[string]*lib.Repository),
		Assertions:    make(map[string]*lib.Assertion),
		Media:         make(map[string]*lib.Media),

		EventTypes:        make(map[string]*lib.EventType),
		ParticipantRoles:  make(map[string]*lib.ParticipantRole),
		ConfidenceLevels:  make(map[string]*lib.ConfidenceLevel),
		RelationshipTypes: make(map[string]*lib.RelationshipType),
		PlaceTypes:        make(map[string]*lib.PlaceType),
		SourceTypes:       make(map[string]*lib.SourceType),
		RepositoryTypes:   make(map[string]*lib.RepositoryType),
		MediaTypes:        make(map[string]*lib.MediaType),

		PersonProperties:       make(map[string]*lib.PropertyDefinition),
		EventProperties:        make(map[string]*lib.PropertyDefinition),
		RelationshipProperties: make(map[string]*lib.PropertyDefinition),
		PlaceProperties:        make(map[string]*lib.PropertyDefinition),
	}

	var allDuplicates []string
	var allErrors []string

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // I/O errors are fatal
		}
		if d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != FileExtGLX && ext != FileExtYAML && ext != FileExtYML {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err // I/O errors are fatal
		}

		// YAML parsing
		doc, err := ParseYAMLFile(data)
		if err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: YAML parse error: %v", path, err))

			return nil // Continue to next file
		}

		// Structural validation against master schema
		issues := ValidateGLXFileStructure(doc)
		if len(issues) > 0 {
			allErrors = append(allErrors, fmt.Sprintf("%s:\n  - %s", path, strings.Join(issues, "\n  - ")))

			return nil // Continue to next file
		}

		var glxFile lib.GLXFile
		err = yaml.Unmarshal(data, &glxFile)
		if err != nil {
			// This should not happen if parsing and structural validation passed
			allErrors = append(allErrors, fmt.Sprintf("%s: unmarshal error: %v", path, err))

			return nil // Continue to next file
		}

		duplicates := merged.Merge(&glxFile)
		allDuplicates = append(allDuplicates, duplicates...)

		return nil
	})
	// If WalkDir itself failed (I/O error), return that
	if err != nil {
		return nil, nil, err
	}

	// If any files had validation/parse errors, return them all
	if len(allErrors) > 0 {
		return nil, nil, fmt.Errorf("%w:\n\n%s", ErrMultipleFilesFailed, strings.Join(allErrors, "\n\n"))
	}

	return merged, allDuplicates, nil
}

// createSerializer creates a new serializer with the specified options
func createSerializer(validate, pretty bool, indent string) *lib.DefaultSerializer {
	opts := &lib.SerializerOptions{
		Validate: validate,
		Pretty:   pretty,
		Indent:   indent,
	}

	return lib.NewSerializer(opts)
}

// readSingleFileArchive reads and deserializes a single-file GLX archive
func readSingleFileArchive(path string, validate bool) (*lib.GLXFile, error) {
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
func writeSingleFileArchive(path string, glx *lib.GLXFile, validate bool) error {
	serializer := createSerializer(validate, true, "  ")

	yamlBytes, err := serializer.SerializeSingleFileBytes(glx)
	if err != nil {
		return fmt.Errorf("failed to serialize GLX file: %w", err)
	}

	if err := os.WriteFile(path, yamlBytes, 0o644); err != nil {
		return fmt.Errorf("failed to write GLX file: %w", err)
	}

	return nil
}

// readMultiFileArchive reads and deserializes a multi-file GLX archive
func readMultiFileArchive(dirPath string, validate bool) (*lib.GLXFile, error) {
	files, err := collectFilesFromDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	serializer := createSerializer(validate, false, "")
	glx, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return nil, fmt.Errorf("failed to load multi-file archive: %w", err)
	}

	return glx, nil
}

// writeMultiFileArchive serializes and writes a multi-file GLX archive
func writeMultiFileArchive(dirPath string, glx *lib.GLXFile, validate bool) error {
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
