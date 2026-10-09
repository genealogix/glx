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
	"slices"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// printResearchSection lists the research logs that concern a person: logs
// whose subject is the person, and logs where the person is a candidate in one
// of the leads. For each it shows the objective, status, outstanding
// (not_searched) searches, and the leads still under investigation.
func printResearchSection(personID string, archive *glxlib.GLXFile) {
	ids := researchLogIDsForPerson(personID, archive)
	if len(ids) == 0 {
		return
	}

	fmt.Println(sectionHeader("Research"))
	for i, id := range ids {
		if i > 0 {
			fmt.Println()
		}
		printResearchLogSummary(id, personID, archive.ResearchLogs[id])
	}
	fmt.Println()
}

// researchLogIDsForPerson returns, sorted, the IDs of research logs whose
// subject is personID or that name personID as a lead candidate.
func researchLogIDsForPerson(personID string, archive *glxlib.GLXFile) []string {
	var ids []string
	for _, id := range sortedKeys(archive.ResearchLogs) {
		log := archive.ResearchLogs[id]
		if log == nil {
			continue
		}
		if slices.Contains(log.PersonIDs(), personID) {
			ids = append(ids, id)
		}
	}

	return ids
}

// printResearchLogSummary prints one research log's entry in the Research
// section.
func printResearchLogSummary(id, personID string, log *glxlib.ResearchLog) {
	header := "  " + id
	if log.Status != "" {
		header += "  [" + log.Status + "]"
	}
	if log.Subject == nil || log.Subject.Person != personID {
		header += "  (candidate in a lead)"
	}
	fmt.Println(header)

	objective := log.Objective
	if objective == "" {
		objective = log.Title
	}
	if objective != "" {
		fmt.Printf("    %-14s%s\n", "Objective:", objective)
	}

	if n := log.CountSearchResults(glxlib.SearchResultNotSearched); n > 0 {
		fmt.Printf("    %-14s%d planned %s not yet performed\n", "Outstanding:", n, pluralize(n, "search", "searches"))
	}

	if len(log.Leads) == 0 {
		return
	}
	var open []*glxlib.ResearchLead
	byStatus := map[string]int{}
	for i := range log.Leads {
		lead := &log.Leads[i]
		status := lead.Status
		if status == "" {
			status = "unspecified"
		}
		byStatus[status]++
		if lead.IsOpen() {
			open = append(open, lead)
		}
	}
	statuses := []string{glxlib.LeadStatusActive, glxlib.LeadStatusConfirmed, glxlib.LeadStatusEliminated}
	var customStatuses []string
	for status := range byStatus {
		if !slices.Contains(statuses, status) {
			customStatuses = append(customStatuses, status)
		}
	}
	slices.Sort(customStatuses)
	var counts []string
	for _, status := range append(statuses, customStatuses...) {
		if byStatus[status] > 0 {
			counts = append(counts, fmt.Sprintf("%d %s", byStatus[status], status))
		}
	}
	fmt.Printf("    %-14s%s\n", "Leads:", strings.Join(counts, ", "))
	for _, lead := range open {
		fmt.Printf("      - %s\n", formatOpenLead(lead))
	}
}

// formatOpenLead renders an open lead as "description (status; next: step)".
func formatOpenLead(lead *glxlib.ResearchLead) string {
	text := lead.Description
	if text == "" {
		text = strings.Join(lead.Persons, ", ")
	}
	if text == "" {
		text = "(undescribed lead)"
	}
	var details []string
	if lead.Status != "" && lead.Status != glxlib.LeadStatusActive {
		details = append(details, lead.Status)
	}
	if len(lead.NextSteps) > 0 {
		details = append(details, "next: "+lead.NextSteps[0])
	}
	if len(details) > 0 {
		text += " (" + strings.Join(details, "; ") + ")"
	}

	return text
}
