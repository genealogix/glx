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
		events := conflictEventContext(archive, personID, c.Facts)
		var appearances []string
		for _, event := range events {
			if !event.OwnRecord {
				appearances = append(appearances, fmt.Sprintf("%s [%s] (%s)", event.Label, event.EventID, event.PersonRole))
			}
		}
		if len(appearances) > 0 {
			label = strings.Join(appearances, "; ") + " " + c.Property
		}
		issues = append(issues, AnalysisIssue{Subject: subject, Events: events, Category: conflictCategory, Severity: finding.Severity, Person: personID, Property: c.Property, Message: fmt.Sprintf("%s — %s%s has %d conflicting values: %s", personName(archive, personID), qualifier, label, len(c.Values), strings.Join(parts, ", "))})
	}
	sortIssues(issues)

	return issues, nil
}

// Contextual event conflicts remain useful association evidence, but must not
// read as the informant's or witness's own vital facts. Preserve the legacy
// Subject display label and add navigable event references rather than parsing it.
type analysisConflictEvent struct {
	EventID      string   `json:"event_id"`
	Title        string   `json:"title,omitempty"`
	Date         string   `json:"date,omitempty"`
	Label        string   `json:"label"`
	PrincipalIDs []string `json:"principal_ids"`
	PersonRole   string   `json:"person_role"`
	OwnRecord    bool     `json:"own_record"`
}

func conflictEventContext(archive *glxlib.GLXFile, personID string, facts []glxlib.ResearchFact) []analysisConflictEvent {
	var result []analysisConflictEvent
	seen := make(map[string]bool)
	for i := range facts {
		fact := &facts[i]
		id := fact.Subject.Event
		event := archive.Events[id]
		if event == nil || seen[id] {
			continue
		}
		seen[id] = true
		participation, role, _ := glxlib.EventParticipation(event, personID, archive.ParticipantRoles)
		principals := []string{}
		for _, participant := range event.Participants {
			if participant.Person != "" && !slices.Contains(principals, participant.Person) && glxlib.ClassifyParticipation(event.Type, participant.Role, archive.ParticipantRoles).OwnRecord {
				principals = append(principals, participant.Person)
			}
		}
		result = append(result, analysisConflictEvent{
			EventID: id, Title: event.Title, Date: string(event.Date), Label: glxlib.AppearanceLabel(event, archive),
			PrincipalIDs: principals, PersonRole: role, OwnRecord: participation.OwnRecord,
		})
	}

	return result
}
