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

// suggestCensusNeighbors recommends cross-referencing the page neighbors
// recorded on census events (#180): a family living next door is the classic
// source of witnesses, sponsors, and in-laws. Each census event yields one
// suggestion, reported on the household's head.
//
// A neighbor linked to a person already tied to the household, by a
// relationship or a shared non-census event, is left out: that cross-reference
// has been made, and the suggestion should clear once it is. A name-only
// neighbor always counts, because nothing yet connects it to anyone.
func suggestCensusNeighbors(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue
	var connections map[string]map[string]bool

	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || event.Type != glxlib.EventTypeCensus || len(event.Neighbors) == 0 {
			continue
		}
		head := censusHeadIndex(event)
		if head < 0 {
			continue
		}
		if connections == nil {
			connections = buildPersonConnections(archive)
		}

		var labels []string
		for i := range event.Neighbors {
			neighbor := &event.Neighbors[i]
			if neighborTiedToHousehold(neighbor, event, connections) {
				continue
			}
			if label := neighborSuggestionLabel(neighbor, archive); label != "" {
				labels = append(labels, label)
			}
		}
		if len(labels) == 0 {
			continue
		}

		year := glxlib.ExtractFirstYear(string(event.Date))
		issues = append(issues, AnalysisIssue{
			Category: analysisSuggestionCategory,
			Severity: severityInfo,
			Person:   event.Participants[head].Person,
			Entity:   eventID,
			Message: fmt.Sprintf("Cross-reference page neighbors on %s: %s; check their marriage, land, and probate records for ties to this household",
				censusEventLabel(event, year), strings.Join(labels, ", ")),
		})
	}

	return issues
}

// buildPersonConnections maps each person to the people they share a
// relationship or a non-census event with. Census events are excluded because
// being enumerated on the same page is the proximity being followed up, not
// evidence of a tie.
func buildPersonConnections(archive *glxlib.GLXFile) map[string]map[string]bool {
	connections := make(map[string]map[string]bool)
	connect := func(participants []glxlib.Participant) {
		for _, a := range participants {
			for _, b := range participants {
				if a.Person == "" || b.Person == "" || a.Person == b.Person {
					continue
				}
				if connections[a.Person] == nil {
					connections[a.Person] = make(map[string]bool)
				}
				connections[a.Person][b.Person] = true
			}
		}
	}

	for _, relationship := range archive.Relationships {
		if relationship != nil {
			connect(relationship.Participants)
		}
	}
	for _, event := range archive.Events {
		if event != nil && event.Type != glxlib.EventTypeCensus {
			connect(event.Participants)
		}
	}

	return connections
}

// neighborTiedToHousehold reports whether a neighbor linked to a person is
// already connected to a household member, or is one of them.
func neighborTiedToHousehold(neighbor *glxlib.EventNeighbor, event *glxlib.Event, connections map[string]map[string]bool) bool {
	if neighbor.Person == "" {
		return false
	}
	for _, p := range event.Participants {
		if !isCensusHouseholdMember(p) {
			continue
		}
		if p.Person == neighbor.Person || connections[p.Person][neighbor.Person] {
			return true
		}
	}

	return false
}

// neighborSuggestionLabel renders a neighbor for a suggestion message: the name
// as written on the page, falling back to the linked person's name, followed by
// the position when recorded, as in "Henry Jeffries (previous household)".
func neighborSuggestionLabel(neighbor *glxlib.EventNeighbor, archive *glxlib.GLXFile) string {
	name := neighbor.Name
	if name == "" && neighbor.Person != "" {
		name = personName(archive, neighbor.Person)
	}
	if name == "" {
		return ""
	}
	if neighbor.Position == "" {
		return name
	}

	return name + " (" + strings.ReplaceAll(neighbor.Position, "_", " ") + ")"
}
