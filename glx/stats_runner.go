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
		archive, duplicates, err := LoadArchiveCached(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		if len(duplicates) > 0 {
			sort.Strings(duplicates)
			fmt.Fprintf(os.Stderr, "Warning: %d duplicate entity IDs found:\n", len(duplicates))
			for _, d := range duplicates {
				fmt.Fprintf(os.Stderr, "  - %s\n", d)
			}
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
	total := 0
	for _, a := range archive.Assertions {
		if a == nil {
			continue
		}
		total++
		level := a.Confidence
		if level == "" {
			level = "(unset)"
		}
		counts[level]++
	}
	if total == 0 {
		return
	}

	// Sort levels: standard order first, then custom alphabetically, then (unset) last
	var levels []string
	standardOrder := []string{
		glxlib.ConfidenceLevelHigh,
		glxlib.ConfidenceLevelMedium,
		glxlib.ConfidenceLevelLow,
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
		pct := float64(counts[level]) / float64(total) * 100
		fmt.Printf("  %-12s %4d  (%5.1f%%)\n", level, counts[level], pct)
	}
}

// printEntityCoverage shows how many persons, events, relationships, and places
// the archive's assertions reach, in two views (#713): the direct assertion
// subjects, and the evidence coverage that follows assertions through the
// events, relationships, participants, and place values they name. See
// glxlib.AssertionCoverage for the rules.
func printEntityCoverage(archive *glxlib.GLXFile) {
	if len(archive.Assertions) == 0 {
		return
	}

	coverage := glxlib.ComputeAssertionCoverage(archive)

	fmt.Println("\nDirect assertion references (entity is an assertion's subject):")
	printEntityCoverageRows(coverage.Direct, archive)

	fmt.Println("\nEvidence coverage (subject, participant, or via an asserted event, relationship, or place):")
	printEntityCoverageRows(coverage.Evidence, archive)
}

// printEntityCoverageRows prints the four coverage rows for one view.
func printEntityCoverageRows(coverage glxlib.EntityCoverage, archive *glxlib.GLXFile) {
	printCoverageRow("Persons", countCovered(coverage.Persons, archive.Persons), len(archive.Persons))
	printCoverageRow("Events", countCovered(coverage.Events, archive.Events), len(archive.Events))
	printCoverageRow("Relationships", countCovered(coverage.Relationships, archive.Relationships), len(archive.Relationships))
	printCoverageRow("Places", countCovered(coverage.Places, archive.Places), len(archive.Places))
}

// countCovered counts the entities of one type that are in the covered set.
// An assertion pointing at a missing entity is a reference error, not
// coverage, so only IDs present in the archive count.
func countCovered[V any](covered map[string]bool, entities map[string]V) int {
	n := 0
	for id := range entities {
		if covered[id] {
			n++
		}
	}

	return n
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
