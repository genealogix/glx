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
	"sort"

	glxlib "github.com/genealogix/glx/go-glx"
)

// analyzeEvidence checks for unsupported or weakly supported claims.
func analyzeEvidence(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	issues = append(issues, checkUnsupportedAssertions(archive)...)
	issues = append(issues, checkSingleSourcePersons(archive)...)
	issues = append(issues, checkOrphanedCitations(archive)...)
	issues = append(issues, checkOrphanedSources(archive)...)

	return issues
}

// checkUnsupportedAssertions finds assertions with no citations and no sources.
func checkUnsupportedAssertions(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	ids := sortedAssertionIDs(archive.Assertions)
	for _, id := range ids {
		assertion := archive.Assertions[id]
		if assertion == nil {
			continue
		}

		if len(assertion.Citations) > 0 || len(assertion.Sources) > 0 || len(assertion.Media) > 0 {
			continue
		}

		issues = append(issues, AnalysisIssue{
			Category: "evidence",
			Severity: "medium",
			Entity:   id,
			Person:   assertion.Subject.Person,
			Message:  "Assertion without citation or source",
		})
	}

	return issues
}

// checkSingleSourcePersons finds persons where all assertions cite the same
// single source.
func checkSingleSourcePersons(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build per-person source sets from assertions
	personSources := make(map[string]map[string]bool) // person ID → set of source IDs
	personAssertionCount := make(map[string]int)

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		personID := assertion.Subject.Person
		if personID == "" {
			continue
		}

		personAssertionCount[personID]++

		if personSources[personID] == nil {
			personSources[personID] = make(map[string]bool)
		}

		for _, sourceID := range assertion.Sources {
			personSources[personID][sourceID] = true
		}

		// Also count sources via citations
		for _, citationID := range assertion.Citations {
			cit, ok := archive.Citations[citationID]
			if !ok || cit == nil {
				continue
			}
			if cit.SourceID != "" {
				personSources[personID][cit.SourceID] = true
			}
		}
	}

	var issues []AnalysisIssue

	for _, personID := range sortedPersonIDs(archive.Persons) {
		sources := personSources[personID]
		count := personAssertionCount[personID]

		if count < 2 || len(sources) != 1 {
			continue
		}

		var sourceID string
		for s := range sources {
			sourceID = s
		}

		name := personName(archive, personID)
		issues = append(issues, AnalysisIssue{
			Category: "evidence",
			Severity: "medium",
			Person:   personID,
			Message:  fmt.Sprintf("%s — single source: all %d assertions cite %s", name, count, sourceID),
		})
	}

	return issues
}

// checkOrphanedCitations finds citations not referenced by any assertion.
func checkOrphanedCitations(archive *glxlib.GLXFile) []AnalysisIssue {
	if len(archive.Citations) == 0 {
		return nil
	}

	referenced := make(map[string]bool)
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		for _, citID := range assertion.Citations {
			referenced[citID] = true
		}
	}

	var issues []AnalysisIssue
	ids := sortedCitationIDs(archive.Citations)
	for _, id := range ids {
		if referenced[id] {
			continue
		}

		issues = append(issues, AnalysisIssue{
			Category: "evidence",
			Severity: "info",
			Entity:   id,
			Message:  "Orphaned citation — not referenced by any assertion",
		})
	}

	return issues
}

// checkOrphanedSources finds sources not referenced by any citation or assertion.
func checkOrphanedSources(archive *glxlib.GLXFile) []AnalysisIssue {
	if len(archive.Sources) == 0 {
		return nil
	}

	referenced := make(map[string]bool)

	// Sources referenced by citations
	for _, cit := range archive.Citations {
		if cit == nil {
			continue
		}
		if cit.SourceID != "" {
			referenced[cit.SourceID] = true
		}
	}

	// Sources referenced directly by assertions
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		for _, sourceID := range assertion.Sources {
			referenced[sourceID] = true
		}
	}

	var issues []AnalysisIssue
	ids := sortedSourceIDs(archive.Sources)
	for _, id := range ids {
		if referenced[id] {
			continue
		}

		issues = append(issues, AnalysisIssue{
			Category: "evidence",
			Severity: "info",
			Entity:   id,
			Message:  "Orphaned source — not referenced by any citation or assertion",
		})
	}

	return issues
}

// sortedAssertionIDs returns assertion IDs in sorted order.
func sortedAssertionIDs(assertions map[string]*glxlib.Assertion) []string {
	ids := make([]string, 0, len(assertions))
	for id := range assertions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// sortedCitationIDs returns citation IDs in sorted order.
func sortedCitationIDs(citations map[string]*glxlib.Citation) []string {
	ids := make([]string, 0, len(citations))
	for id := range citations {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// sortedSourceIDs returns source IDs in sorted order.
func sortedSourceIDs(sources map[string]*glxlib.Source) []string {
	ids := make([]string, 0, len(sources))
	for id := range sources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
