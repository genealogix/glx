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
		archive, _, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}

		return archive, nil
	}

	archive, err := readSingleFileArchive(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load archive: %w", err)
	}

	return archive, nil
}

// printEntityCounts prints the count of each entity type.
func printEntityCounts(archive *glxlib.GLXFile) {
	fmt.Println("Entity counts:")
	fmt.Printf("  Persons:       %d\n", len(archive.Persons))
	fmt.Printf("  Events:        %d\n", len(archive.Events))
	fmt.Printf("  Relationships: %d\n", len(archive.Relationships))
	fmt.Printf("  Places:        %d\n", len(archive.Places))
	fmt.Printf("  Sources:       %d\n", len(archive.Sources))
	fmt.Printf("  Citations:     %d\n", len(archive.Citations))
	fmt.Printf("  Repositories:  %d\n", len(archive.Repositories))
	fmt.Printf("  Media:         %d\n", len(archive.Media))
	fmt.Printf("  Assertions:    %d\n", len(archive.Assertions))
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

	// Sort levels for deterministic output
	levels := make([]string, 0, len(counts))
	for level := range counts {
		levels = append(levels, level)
	}
	sort.Strings(levels)

	fmt.Println("\nAssertion confidence:")
	for _, level := range levels {
		pct := float64(counts[level]) / float64(len(archive.Assertions)) * 100
		fmt.Printf("  %-12s %4d  (%5.1f%%)\n", level, counts[level], pct)
	}
}

// printEntityCoverage shows how many persons, events, and relationships are
// referenced by at least one assertion.
func printEntityCoverage(archive *glxlib.GLXFile) {
	if len(archive.Assertions) == 0 {
		return
	}

	coveredPersons := make(map[string]struct{})
	coveredEvents := make(map[string]struct{})
	coveredRelationships := make(map[string]struct{})

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
	}

	fmt.Println("\nEntity coverage (referenced by assertions):")
	printCoverageRow("Persons", len(coveredPersons), len(archive.Persons))
	printCoverageRow("Events", len(coveredEvents), len(archive.Events))
	printCoverageRow("Relationships", len(coveredRelationships), len(archive.Relationships))
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
