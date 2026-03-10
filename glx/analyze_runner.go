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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// AnalysisIssue represents a single finding from the analysis engine.
type AnalysisIssue struct {
	Category string `json:"category"` // "gap", "evidence", "consistency", "suggestion"
	Severity string `json:"severity"` // "high", "medium", "low", "info"
	Person   string `json:"person,omitempty"`
	Entity   string `json:"entity,omitempty"`
	Message  string `json:"message"`
	Property string `json:"property,omitempty"`
}

// AnalysisResult holds all findings from the analysis engine.
type AnalysisResult struct {
	Summary map[string]int  `json:"summary"`
	Issues  []AnalysisIssue `json:"issues"`
}

// loadArchiveForAnalyze loads a GLX archive for analysis.
func loadArchiveForAnalyze(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
		if loadErr != nil {
			return nil, loadErr
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

// showAnalysis runs the analysis engine and prints results.
func showAnalysis(archivePath, personFilter, checkFilter, format string) error {
	archive, err := loadArchiveForAnalyze(archivePath)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}

	var issues []AnalysisIssue

	checks := map[string]func(*glxlib.GLXFile) []AnalysisIssue{
		"gaps":        analyzeGaps,
		"evidence":    analyzeEvidence,
		"consistency": analyzeConsistency,
		"suggestions": analyzeSuggestions,
	}

	// Accept singular aliases ("gap" → "gaps", "suggestion" → "suggestions")
	singularToPlural := map[string]string{
		"gap":        "gaps",
		"suggestion": "suggestions",
	}

	if checkFilter != "" {
		if mapped, ok := singularToPlural[checkFilter]; ok {
			checkFilter = mapped
		}
		fn, ok := checks[checkFilter]
		if !ok {
			return fmt.Errorf("unknown check category: %q (valid: gaps, evidence, consistency, suggestions)", checkFilter)
		}
		issues = fn(archive)
	} else {
		for _, category := range []string{"gaps", "evidence", "consistency", "suggestions"} {
			issues = append(issues, checks[category](archive)...)
		}
	}

	// Filter by person if specified
	if personFilter != "" {
		issues = filterByPerson(issues, personFilter, archive)
	}

	// Sort by severity (high → medium → low → info) then by person/entity for stability
	sortIssues(issues)

	result := AnalysisResult{
		Summary: buildSummary(issues),
		Issues:  issues,
	}

	if format == "json" {
		return printAnalysisJSON(result)
	}

	printAnalysisTerminal(result)
	return nil
}

// filterByPerson filters issues to only those matching a person ID or name.
func filterByPerson(issues []AnalysisIssue, query string, archive *glxlib.GLXFile) []AnalysisIssue {
	// Check if query is an exact person ID
	if _, ok := archive.Persons[query]; ok {
		var filtered []AnalysisIssue
		for _, issue := range issues {
			if issue.Person == query {
				filtered = append(filtered, issue)
			}
		}
		return filtered
	}

	// Try name match
	lowerQuery := strings.ToLower(query)
	matchingIDs := make(map[string]bool)
	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		name := glxlib.PersonDisplayName(person)
		if containsFold(name, lowerQuery) {
			matchingIDs[id] = true
		}
	}

	var filtered []AnalysisIssue
	for _, issue := range issues {
		if matchingIDs[issue.Person] {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// buildSummary counts issues by category.
func buildSummary(issues []AnalysisIssue) map[string]int {
	summary := map[string]int{
		"gap":         0,
		"evidence":    0,
		"consistency": 0,
		"suggestion":  0,
	}
	for _, issue := range issues {
		summary[issue.Category]++
	}
	return summary
}

// printAnalysisJSON outputs analysis results as JSON.
func printAnalysisJSON(result AnalysisResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printAnalysisTerminal outputs analysis results to the terminal.
func printAnalysisTerminal(result AnalysisResult) {
	total := len(result.Issues)
	if total == 0 {
		fmt.Println("No issues found — archive looks good!")
		return
	}

	fmt.Printf("=== Research Gap Analysis: %d issues found ===\n", total)

	categories := []struct {
		key   string
		label string
	}{
		{"gap", "EVIDENCE GAPS"},
		{"evidence", "EVIDENCE QUALITY"},
		{"consistency", "CONSISTENCY"},
		{"suggestion", "SUGGESTIONS"},
	}

	for _, cat := range categories {
		catIssues := filterCategory(result.Issues, cat.key)
		if len(catIssues) == 0 {
			continue
		}

		fmt.Printf("\n%s (%d)\n", cat.label, len(catIssues))
		for _, issue := range catIssues {
			printIssue(issue)
		}
	}
	fmt.Println()
}

// filterCategory returns issues matching a category.
func filterCategory(issues []AnalysisIssue, category string) []AnalysisIssue {
	var filtered []AnalysisIssue
	for _, issue := range issues {
		if issue.Category == category {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// printIssue prints a single analysis issue.
func printIssue(issue AnalysisIssue) {
	ref := issue.Person
	if ref == "" {
		ref = issue.Entity
	}
	if ref == "" {
		ref = "(archive)"
	}

	switch issue.Category {
	case "suggestion":
		fmt.Printf("  →   %-30s %s\n", ref, issue.Message)
	default:
		sev := strings.ToUpper(issue.Severity)
		fmt.Printf("  %-4s %-30s %s\n", sev, ref, issue.Message)
	}
}

// severityRank maps severity strings to numeric rank for sorting (lower = more severe).
var severityRank = map[string]int{
	"high":   0,
	"medium": 1,
	"low":    2,
	"info":   3,
}

// sortIssues sorts issues by severity (high first), then by person/entity ID.
func sortIssues(issues []AnalysisIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		ri, okI := severityRank[issues[i].Severity]
		if !okI {
			ri = len(severityRank)
		}
		rj, okJ := severityRank[issues[j].Severity]
		if !okJ {
			rj = len(severityRank)
		}
		if ri != rj {
			return ri < rj
		}
		pi, pj := issues[i].Person+issues[i].Entity, issues[j].Person+issues[j].Entity
		return pi < pj
	})
}

// personName returns the display name for a person ID, or the ID itself.
func personName(archive *glxlib.GLXFile, personID string) string {
	if person, ok := archive.Persons[personID]; ok && person != nil {
		name := glxlib.PersonDisplayName(person)
		if name != "" {
			return name
		}
	}
	return personID
}

// sortedPersonIDs returns person IDs in sorted order.
func sortedPersonIDs(persons map[string]*glxlib.Person) []string {
	ids := make([]string, 0, len(persons))
	for id := range persons {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
