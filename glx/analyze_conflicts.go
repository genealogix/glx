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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	conflictCategory           = "conflict"
	analysisGapCategory        = "gap"
	analysisSuggestionCategory = "suggestion"
	severityLow                = "low"
	conflictsCheck             = "conflicts"
)

// analyzeConflicts reports shared-engine conflicts on each person's facts.
func analyzeConflictsWithOptions(archive *glxlib.GLXFile, opts glxlib.ComparisonOptions) ([]AnalysisIssue, error) {
	findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{Comparison: opts})
	if err != nil {
		return nil, err
	}
	var issues []AnalysisIssue
	for i := range findings {
		finding := &findings[i]
		personID, c := finding.PersonID, &finding.Conflict
		qualifier := ""
		if c.Verdict == glxlib.VerdictPossible {
			qualifier = "possible — check: "
		}
		if c.Verdict == glxlib.VerdictDisputed {
			qualifier = "known dispute: "
		}
		var parts []string
		for _, v := range c.Values {
			entry := v.Value
			if v.Confidence != "" {
				entry += " [" + v.Confidence + "]"
			}
			parts = append(parts, entry)
		}
		subject, label := c.Subject, c.Property
		if subject == glxlib.EntityTypePersons.String()+":"+personID {
			subject = ""
		}
		if subject != "" {
			label = subject + " " + label
		}
		issues = append(issues, AnalysisIssue{Subject: subject, Category: conflictCategory, Severity: finding.Severity, Person: personID, Property: c.Property, Message: fmt.Sprintf("%s — %s%s has %d conflicting values: %s", personName(archive, personID), qualifier, label, len(c.Values), strings.Join(parts, ", "))})
	}
	sortIssues(issues)

	return issues, nil
}

// confidenceRank returns a numeric rank for confidence levels (lower = higher confidence).
func confidenceRank(c string) int {
	switch strings.ToLower(c) {
	case "high":
		return 0
	case "medium-high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
