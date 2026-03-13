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
	"gopkg.in/yaml.v3"
)

// censusAdd reads a census template, loads the archive, generates entities,
// and writes the new entity files to the archive directory.
func censusAdd(templatePath, archivePath string, dryRun, verbose bool) error {
	// Read and parse template
	template, err := readCensusTemplate(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Load existing archive
	archive, err := loadArchiveForCensus(archivePath)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}

	// Generate entities
	result, err := glxlib.BuildCensusEntities(template, archive)
	if err != nil {
		return fmt.Errorf("failed to generate census entities: %w", err)
	}

	// Validate cross-references before writing
	if err := validateCensusRefs(result, archive); err != nil {
		return fmt.Errorf("generated entities have invalid references: %w", err)
	}

	if verbose || dryRun {
		printCensusSummary(result)
	}

	if dryRun {
		fmt.Println("\n(dry run — no files written)")
		return nil
	}

	// Serialize and write new entities
	count, err := writeCensusEntities(archivePath, result)
	if err != nil {
		return fmt.Errorf("failed to write entities: %w", err)
	}

	fmt.Printf("Wrote %d entity files to %s\n", count, archivePath)
	return nil
}

// readCensusTemplate reads and parses a census template YAML file.
func readCensusTemplate(path string) (*glxlib.CensusTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var tpl glxlib.CensusTemplate
	if err := yaml.Unmarshal(data, &tpl); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &tpl, nil
}

// loadArchiveForCensus loads an archive from a directory path.
// Census import writes multi-file output, so single-file archives are not supported.
func loadArchiveForCensus(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("census requires an archive directory, but %s is not a directory", path)
	}

	archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
	if loadErr != nil {
		return nil, fmt.Errorf("failed to load archive: %w", loadErr)
	}
	for _, d := range duplicates {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
	}
	return archive, nil
}

// writeCensusEntities serializes the generated entities and writes them to
// the archive directory. Returns the number of files written.
func writeCensusEntities(archivePath string, result *glxlib.CensusResult) (int, error) {
	// Build a partial GLXFile from the result
	partial := &glxlib.GLXFile{
		Persons:    result.Persons,
		Events:     result.Event,
		Places:     result.Place,
		Sources:    result.Source,
		Citations:  result.Citation,
		Assertions: result.Assertions,
	}

	serializer := createSerializer(false, true, "  ")
	files, err := serializer.SerializeMultiFileToMap(partial)
	if err != nil {
		return 0, fmt.Errorf("serialization failed: %w", err)
	}

	// Filter out vocabulary and metadata files — we only want entity files
	entityFiles := make(map[string][]byte)
	for relPath, data := range files {
		if strings.HasPrefix(relPath, "vocabularies/") || strings.HasPrefix(relPath, "vocabularies\\") {
			continue
		}
		if relPath == "metadata.glx" {
			continue
		}
		entityFiles[relPath] = data
	}

	// Warn about any existing files that will be overwritten
	for relPath := range entityFiles {
		absPath := filepath.Join(archivePath, relPath)
		if _, err := os.Stat(absPath); err == nil {
			fmt.Fprintf(os.Stderr, "Warning: overwriting existing file %s\n", relPath)
		}
	}

	if err := writeFilesToDir(archivePath, entityFiles); err != nil {
		return 0, err
	}

	return len(entityFiles), nil
}

// validateCensusRefs checks that generated entity cross-references point to
// either newly created entities or entities in the existing archive.
func validateCensusRefs(result *glxlib.CensusResult, existing *glxlib.GLXFile) error {
	for id, cit := range result.Citation {
		if _, ok := result.Source[cit.SourceID]; !ok {
			if existing.Sources == nil || existing.Sources[cit.SourceID] == nil {
				return fmt.Errorf("citation %s references unknown source %s", id, cit.SourceID)
			}
		}
	}
	for id, evt := range result.Event {
		if evt.PlaceID == "" {
			continue
		}
		if _, ok := result.Place[evt.PlaceID]; !ok {
			if existing.Places == nil || existing.Places[evt.PlaceID] == nil {
				return fmt.Errorf("event %s references unknown place %s", id, evt.PlaceID)
			}
		}
	}
	for id, a := range result.Assertions {
		if a.Subject.Person != "" {
			if _, ok := result.Persons[a.Subject.Person]; !ok {
				if existing.Persons == nil || existing.Persons[a.Subject.Person] == nil {
					return fmt.Errorf("assertion %s references unknown person %s", id, a.Subject.Person)
				}
			}
		}
	}
	return nil
}

// printCensusSummary prints a summary of what was generated.
func printCensusSummary(result *glxlib.CensusResult) {
	fmt.Println("Census Import Summary")
	fmt.Println("=====================")
	fmt.Printf("  Event:      %s\n", result.EventID)
	fmt.Printf("  Source:     %s\n", result.SourceID)
	fmt.Printf("  Citation:   %s\n", result.CitationID)
	fmt.Printf("  Place:      %s\n", result.PlaceID)

	if len(result.NewPersonIDs) > 0 {
		fmt.Printf("\n  New persons (%d):\n", len(result.NewPersonIDs))
		for _, id := range result.NewPersonIDs {
			fmt.Printf("    + %s\n", id)
		}
	}

	if len(result.MatchedIDs) > 0 {
		fmt.Printf("\n  Matched existing persons (%d):\n", len(result.MatchedIDs))
		for _, id := range result.MatchedIDs {
			fmt.Printf("    = %s\n", id)
		}
	}

	fmt.Printf("\n  New places:     %d\n", len(result.Place))
	fmt.Printf("  New sources:    %d\n", len(result.Source))
	fmt.Printf("  Assertions:     %d\n", len(result.Assertions))
}
