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
	"sort"

	glxlib "github.com/genealogix/glx/go-glx"
)

// printEntityCounts prints the count of each entity type, reusing the shared helper.
func printEntityCounts(archive *glxlib.GLXFile) {
	printVerboseArchiveStatistics(archive, "Entity counts:")
}

// showStats loads a GLX archive and prints summary statistics.
func showStats(path string) error {
	archive, err := loadArchiveForStats(path)
	if err != nil {
		return err
	}

	printEntityCounts(archive)
	printConfidenceDistribution(archive)
	printEntityCoverage(archive)

	return nil
}

// loadArchiveForStats loads an archive from a path (directory or single file).
func loadArchiveForStats(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		if len(duplicates) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: %d duplicate entity IDs found\n", len(duplicates))
		}

		return archive, nil
	}

	archive, err := readSingleFileArchive(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load archive: %w", err)
	}

	return archive, nil
}

// printConfidenceDistribution prints a breakdown of assertions by confidence level.
func printConfidenceDistribution(archive *glxlib.GLXFile) {
	if len(archive.Assertions) == 0 {
		return
	}

	counts := make(map[string]int)
	for _, a := range archive.Assertions {
		level := a.Confidence
		if level == "" {
			level = "(unset)"
		}
		counts[level]++
	}

	// Sort levels: standard order first, then custom alphabetically, then (unset) last
	var levels []string
	standardOrder := []string{
		glxlib.ConfidenceLevelHigh,
		glxlib.ConfidenceLevelMedium,
		glxlib.ConfidenceLevelLow,
		glxlib.ConfidenceLevelDisputed,
	}
	seen := make(map[string]bool)
	for _, level := range standardOrder {
		if counts[level] > 0 {
			levels = append(levels, level)
			seen[level] = true
		}
	}
	var custom []string
	for level := range counts {
		if !seen[level] && level != "(unset)" {
			custom = append(custom, level)
		}
	}
	sort.Strings(custom)
	levels = append(levels, custom...)
	if counts["(unset)"] > 0 {
		levels = append(levels, "(unset)")
	}

	fmt.Println("\nAssertion confidence:")
	for _, level := range levels {
		pct := float64(counts[level]) / float64(len(archive.Assertions)) * 100
		fmt.Printf("  %-12s %4d  (%5.1f%%)\n", level, counts[level], pct)
	}
}

// printEntityCoverage shows how many persons, events, relationships, and places
// are referenced by at least one assertion.
func printEntityCoverage(archive *glxlib.GLXFile) {
	if len(archive.Assertions) == 0 {
		return
	}

	coveredPersons := make(map[string]struct{})
	coveredEvents := make(map[string]struct{})
	coveredRelationships := make(map[string]struct{})
	coveredPlaces := make(map[string]struct{})

	for _, a := range archive.Assertions {
		if a.Subject.Person != "" {
			coveredPersons[a.Subject.Person] = struct{}{}
		}
		if a.Subject.Event != "" {
			coveredEvents[a.Subject.Event] = struct{}{}
		}
		if a.Subject.Relationship != "" {
			coveredRelationships[a.Subject.Relationship] = struct{}{}
		}
		if a.Subject.Place != "" {
			coveredPlaces[a.Subject.Place] = struct{}{}
		}
	}

	fmt.Println("\nEntity coverage (referenced by assertions):")
	printCoverageRow("Persons", len(coveredPersons), len(archive.Persons))
	printCoverageRow("Events", len(coveredEvents), len(archive.Events))
	printCoverageRow("Relationships", len(coveredRelationships), len(archive.Relationships))
	printCoverageRow("Places", len(coveredPlaces), len(archive.Places))
}

// printCoverageRow prints a single coverage line with percentage.
func printCoverageRow(label string, covered, total int) {
	if total == 0 {
		fmt.Printf("  %-15s  -\n", label)

		return
	}

	pct := float64(covered) / float64(total) * 100
	fmt.Printf("  %-15s %d/%d  (%.1f%%)\n", label, covered, total, pct)
}
