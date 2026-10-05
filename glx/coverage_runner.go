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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Coverage checklist categories, as they appear in the JSON output and as the
// headings printCoverageText groups by. coverageCategoryCensus shares the
// literal value of EventTypeCensus.
const (
	coverageCategoryCensus = "census"
	coverageCategoryVital  = "vital"
	coverageCategoryOther  = "other"
)

// showCoverage loads an archive and displays source coverage for a person.
func showCoverage(archivePath, personQuery, country string, jsonOutput bool) error {
	if err := applyCensusCountry(country); err != nil {
		return err
	}

	archive, err := loadArchiveForCoverage(archivePath)
	if err != nil {
		return err
	}

	personID, _, err := findPersonForCoverage(archive, personQuery)
	if err != nil {
		return err
	}

	result, err := glxlib.BuildCoverage(archive, personID, glxlib.CoverageOptions{CensusCountry: censusCountryFallback})
	if err != nil {
		return err
	}

	if jsonOutput {
		return printCoverageJSON(result)
	}

	printCoverageText(result)

	return nil
}

// loadArchiveForCoverage loads an archive from a path.
func loadArchiveForCoverage(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveCached(path)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}

		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// findPersonForCoverage finds a person by ID or name substring.
func findPersonForCoverage(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	if person, ok := archive.Persons[query]; ok && person != nil {
		return query, person, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		name := glxlib.PersonDisplayName(person)
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("no person found matching %q", query)
	case 1:
		return matches[0], archive.Persons[matches[0]], nil
	default:
		var lines []string
		for _, id := range matches {
			name := glxlib.PersonDisplayName(archive.Persons[id])
			lines = append(lines, fmt.Sprintf("  %s  %s", id, name))
		}

		return "", nil, fmt.Errorf("multiple persons match %q:\n%s\nUse exact person ID", query, strings.Join(lines, "\n"))
	}
}

// printCoverageText prints coverage in a human-readable format.
func printCoverageText(result *coverageResult) {
	name := result.PersonName
	if name == "" {
		name = result.PersonID
	}

	fmt.Printf("Source Coverage for %s (%s)\n", name, result.PersonID)

	// Summary line
	var parts []string
	if result.BirthDate != "" {
		born := "Born: " + result.BirthDate
		if result.BirthPlace != "" {
			born += ", " + result.BirthPlace
		}
		parts = append(parts, born)
	}
	if result.DeathDate != "" {
		died := "Died: " + result.DeathDate
		if result.DeathPlace != "" {
			died += ", " + result.DeathPlace
		}
		parts = append(parts, died)
	} else {
		parts = append(parts, "Died: unknown")
	}
	if len(parts) > 0 {
		fmt.Printf("%s\n", strings.Join(parts, " | "))
	}

	// Group by category
	categories := []struct {
		key   string
		label string
	}{
		{coverageCategoryCensus, "Census Records"},
		{coverageCategoryVital, "Vital Records"},
		{coverageCategoryOther, "Other Records"},
	}

	for _, cat := range categories {
		var catRecords []coverageRecord
		for _, r := range result.Records {
			if r.Category == cat.key {
				catRecords = append(catRecords, r)
			}
		}
		if len(catRecords) == 0 {
			continue
		}

		fmt.Printf("\n  %s:\n", cat.label)
		for _, r := range catRecords {
			marker := "[ ]"
			if r.Found {
				marker = "[x]"
			}

			line := fmt.Sprintf("    %s %s", marker, r.Label)

			if r.Found && r.SourceRef != "" {
				line += fmt.Sprintf(" (via %s)", r.SourceRef)
			}

			if !r.Found && r.Priority == "high" {
				line += " -- HIGH PRIORITY"
			}

			if r.Description != "" {
				line += " -- " + r.Description
			}

			fmt.Println(line)
		}
	}

	fmt.Printf("\n  Coverage: %d of %d expected records found (%d%%)\n",
		result.Found, result.Expected, coveragePercent(result.Found, result.Expected))
	printCoverageAppearances(result.AppearsIn)
}

// printCoverageAppearances lists the other people's records the person
// appears in, which the checklist above does not count.
func printCoverageAppearances(appearances []coverageAppearance) {
	if len(appearances) == 0 {
		return
	}

	fmt.Println("\n  Appears in (other people's records, not counted above):")
	for _, a := range appearances {
		line := "    " + a.Label
		if a.Role != "" {
			line += " (" + a.Role + ")"
		}
		line += " -- " + a.EventID
		if a.Date != "" {
			line += ", " + a.Date
		}
		fmt.Println(line)
	}
}

func coveragePercent(found, expected int) int {
	if expected == 0 {
		return 0
	}

	return (found * 100) / expected
}

// printCoverageJSON outputs the result as JSON.
func printCoverageJSON(result *coverageResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))

	return nil
}

type (
	coverageRecord     = glxlib.CoverageRecord
	coverageResult     = glxlib.CoverageResult
	coverageAppearance = glxlib.CoverageAppearance
)
