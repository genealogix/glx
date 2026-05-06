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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// confidenceReport generates a confidence summary report for a GLX archive.
func confidenceReport(archivePath string) error {
	archive, err := loadArchiveForReport(archivePath)
	if err != nil {
		return err
	}

	report := buildConfidenceReport(archive)
	printConfidenceReport(report)

	return nil
}

// loadArchiveForReport loads a GLX archive from a path (directory or single file).
func loadArchiveForReport(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, err
		}
		if len(duplicates) > 0 {
			fmt.Fprintf(os.Stderr, "Warning: %d duplicate entity IDs found:\n", len(duplicates))
			for _, d := range duplicates {
				fmt.Fprintf(os.Stderr, "  - %s\n", d)
			}
		}

		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// reportData holds the analyzed confidence report data.
type reportData struct {
	TotalAssertions    int
	ByConfidence       map[string]int            // confidence level -> count
	NoConfidence       []assertionSummary        // assertions with no confidence set
	NoCitations        []assertionSummary        // assertions with no citations
	UnbackedPersons    []string                  // person IDs with no assertions
	UnbackedEvents     []string                  // event IDs with no assertions
	UnbackedRelations  []string                  // relationship IDs with no assertions
	ConfidenceOrder    []string                  // ordered confidence levels for display
}

// assertionSummary is a short description of an assertion for display.
type assertionSummary struct {
	ID          string
	SubjectType string
	SubjectID   string
	Property    string
}

// buildConfidenceReport analyzes assertions in the archive and returns report data.
func buildConfidenceReport(archive *glxlib.GLXFile) reportData {
	report := reportData{
		ByConfidence: make(map[string]int),
	}
	assertedSubjects := make(map[string]map[string]bool)

	for id, assertion := range archive.Assertions {
		report.TotalAssertions++

		// Track confidence levels
		conf := assertion.Confidence
		if conf == "" {
			conf = "(unset)"
			report.NoConfidence = append(report.NoConfidence, summarizeAssertion(id, assertion))
		}
		report.ByConfidence[conf]++

		// Track assertions with no citations
		if len(assertion.Citations) == 0 {
			report.NoCitations = append(report.NoCitations, summarizeAssertion(id, assertion))
		}

		// Track which entities have assertions
		subjectType := assertion.Subject.Type()
		subjectID := assertion.Subject.ID()
		if subjectType != "" && subjectID != "" {
			if assertedSubjects[subjectType] == nil {
				assertedSubjects[subjectType] = make(map[string]bool)
			}
			assertedSubjects[subjectType][subjectID] = true
		}
	}

	// Find entities with no assertions
	report.UnbackedPersons = findUnbacked(archive.Persons, assertedSubjects[glxlib.EntityTypePersons])
	report.UnbackedEvents = findUnbacked(archive.Events, assertedSubjects[glxlib.EntityTypeEvents])
	report.UnbackedRelations = findUnbacked(archive.Relationships, assertedSubjects[glxlib.EntityTypeRelationships])

	// Build ordered confidence levels: known levels first, then custom, then (unset)
	report.ConfidenceOrder = buildConfidenceOrder(report.ByConfidence)

	// Sort assertion lists for deterministic output
	sort.Slice(report.NoConfidence, func(i, j int) bool { return report.NoConfidence[i].ID < report.NoConfidence[j].ID })
	sort.Slice(report.NoCitations, func(i, j int) bool { return report.NoCitations[i].ID < report.NoCitations[j].ID })
	sort.Strings(report.UnbackedPersons)
	sort.Strings(report.UnbackedEvents)
	sort.Strings(report.UnbackedRelations)

	return report
}

func summarizeAssertion(id string, a *glxlib.Assertion) assertionSummary {
	s := assertionSummary{
		ID:          id,
		SubjectType: a.Subject.Type(),
		SubjectID:   a.Subject.ID(),
		Property:    a.Property,
	}
	if s.Property == "" && a.Participant != nil {
		s.Property = fmt.Sprintf("participant(%s)", a.Participant.Person)
	}

	return s
}

// findUnbacked returns IDs of entities that have no assertions referencing them.
func findUnbacked[V any](entities map[string]V, asserted map[string]bool) []string {
	var unbacked []string
	for id := range entities {
		if !asserted[id] {
			unbacked = append(unbacked, id)
		}
	}

	return unbacked
}

// buildConfidenceOrder returns confidence levels in display order:
// high, medium, low, then any custom levels alphabetically, then (unset).
func buildConfidenceOrder(counts map[string]int) []string {
	knownOrder := []string{
		glxlib.ConfidenceLevelHigh,
		glxlib.ConfidenceLevelMedium,
		glxlib.ConfidenceLevelLow,
	}

	var order []string
	seen := make(map[string]bool)

	for _, level := range knownOrder {
		if counts[level] > 0 {
			order = append(order, level)
			seen[level] = true
		}
	}

	// Custom levels alphabetically
	var custom []string
	for level := range counts {
		if !seen[level] && level != "(unset)" {
			custom = append(custom, level)
		}
	}
	sort.Strings(custom)
	order = append(order, custom...)

	// (unset) last
	if counts["(unset)"] > 0 {
		order = append(order, "(unset)")
	}

	return order
}

// printConfidenceReport outputs the report to stdout.
func printConfidenceReport(report reportData) {
	fmt.Println("Confidence Summary Report")
	fmt.Println(strings.Repeat("=", 40))

	if report.TotalAssertions == 0 {
		fmt.Println("\nNo assertions found in this archive.")
	} else {
		// Confidence breakdown
		fmt.Printf("\nAssertions: %d total\n\n", report.TotalAssertions)
		fmt.Println("  Confidence Level    Count")
		fmt.Println("  " + strings.Repeat("-", 30))
		for _, level := range report.ConfidenceOrder {
			fmt.Printf("  %-20s %d\n", level, report.ByConfidence[level])
		}

		// Assertions without confidence
		if len(report.NoConfidence) > 0 {
			fmt.Printf("\nAssertions without confidence level (%d):\n", len(report.NoConfidence))
			for _, a := range report.NoConfidence {
				desc := formatAssertionDesc(a)
				fmt.Printf("  - %s: %s\n", a.ID, desc)
			}
		}

		// Assertions without citations
		if len(report.NoCitations) > 0 {
			fmt.Printf("\nAssertions without citations (%d):\n", len(report.NoCitations))
			for _, a := range report.NoCitations {
				desc := formatAssertionDesc(a)
				fmt.Printf("  - %s: %s\n", a.ID, desc)
			}
		}
	}

	// Unbacked entities
	hasUnbacked := len(report.UnbackedPersons) > 0 || len(report.UnbackedEvents) > 0 || len(report.UnbackedRelations) > 0
	if hasUnbacked {
		fmt.Println("\nEntities with no assertions:")
		printUnbackedList("Persons", report.UnbackedPersons)
		printUnbackedList("Events", report.UnbackedEvents)
		printUnbackedList("Relationships", report.UnbackedRelations)
	}
}

func formatAssertionDesc(a assertionSummary) string {
	if a.Property != "" {
		return fmt.Sprintf("%s[%s].%s", a.SubjectType, a.SubjectID, a.Property)
	}

	return fmt.Sprintf("%s[%s] (existential)", a.SubjectType, a.SubjectID)
}

func printUnbackedList(label string, ids []string) {
	if len(ids) == 0 {
		return
	}
	fmt.Printf("  %s (%d): %s\n", label, len(ids), strings.Join(ids, ", "))
}
