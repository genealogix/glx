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

// ancestorSuggestion represents a research suggestion for the ancestors command.
type ancestorSuggestion struct {
	PersonID string // which person this suggestion is about
	Category string // "gap", "census"
	Priority string // "high", "medium", ""
	Year     int    // census year (0 if not applicable)
	Message  string
}

// buildAncestorSuggestions generates research suggestions for persons in the
// ancestor tree who are missing parents. It walks up to 3 generations and
// flags gaps and key census records that could reveal parent information.
func buildAncestorSuggestions(tc *treeContext, rootPersonID string, archive *glxlib.GLXFile) []ancestorSuggestion {
	var suggestions []ancestorSuggestion
	visited := make(map[string]bool)

	collectAncestorSuggestions(tc, rootPersonID, archive, 0, 3, visited, &suggestions)

	return suggestions
}

// collectAncestorSuggestions recursively collects suggestions for persons
// missing parents in the ancestor tree.
func collectAncestorSuggestions(tc *treeContext, personID string, archive *glxlib.GLXFile, depth, maxDepth int, visited map[string]bool, suggestions *[]ancestorSuggestion) {
	if visited[personID] || depth > maxDepth {
		return
	}
	visited[personID] = true

	parents := findParents(tc, personID)

	if len(parents) == 0 {
		// No parents found — generate suggestions for this person
		person := archive.Persons[personID]
		name := "(unknown)"
		if person != nil {
			name = glxlib.PersonDisplayName(person)
		}

		*suggestions = append(*suggestions, ancestorSuggestion{
			PersonID: personID,
			Category: "gap",
			Priority: "high",
			Message:  fmt.Sprintf("%s — no parent_child relationship found", name),
		})

		// Suggest census records that could reveal parents
		if person != nil {
			censusSuggestions := suggestParentCensusRecords(personID, person, archive)
			*suggestions = append(*suggestions, censusSuggestions...)
		}
	} else {
		// Has parents — recurse into them to find gaps further up
		for _, p := range parents {
			collectAncestorSuggestions(tc, p.personID, archive, depth+1, maxDepth, visited, suggestions)
		}
	}
}

// suggestParentCensusRecords suggests census records that are particularly
// useful for finding a person's parents.
func suggestParentCensusRecords(personID string, person *glxlib.Person, archive *glxlib.GLXFile) []ancestorSuggestion {
	birthYear := glxlib.ExtractFirstYear(propertyString(person.Properties, glxlib.PersonPropertyBornOn))
	if birthYear == 0 {
		return nil
	}

	name := glxlib.PersonDisplayName(person)
	bornAt := propertyString(person.Properties, glxlib.PersonPropertyBornAt)
	placeName := ""
	if bornAt != "" {
		if place, ok := archive.Places[bornAt]; ok && place != nil {
			placeName = place.Name
		}
	}

	// Collect existing census years from events
	existingCensus := make(map[int]bool)
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				year := glxlib.ExtractFirstYear(string(event.Date))
				if year > 0 {
					existingCensus[year] = true
				}
				break
			}
		}
	}

	var suggestions []ancestorSuggestion

	// Key census years for finding parents
	for _, year := range usCensusYears {
		if year < birthYear || year > birthYear+maxLifespan {
			continue
		}
		if existingCensus[year] {
			continue
		}

		age := year - birthYear

		// Focus on censuses most useful for parent research
		var priority string
		var reason string

		switch {
		case year == 1880:
			priority = "high"
			reason = "first census to list parents' birthplaces"
		case age < 18:
			priority = "high"
			reason = "likely in parents' household"
		case year == 1850 && age >= 18 && age <= 25:
			priority = "medium"
			reason = "first census to list individual names; may show in parents' household"
		default:
			continue // only suggest high-value censuses for ancestor research
		}

		location := ""
		if placeName != "" {
			location = fmt.Sprintf(", %s", placeName)
		}

		suggestions = append(suggestions, ancestorSuggestion{
			PersonID: personID,
			Category: "census",
			Priority: priority,
			Year:     year,
			Message: fmt.Sprintf("%s — search %d census (age ~%d%s) — %s",
				name, year, age, location, reason),
		})
	}

	return suggestions
}

// printAncestorSuggestions prints research suggestions below the ancestor tree.
func printAncestorSuggestions(suggestions []ancestorSuggestion) {
	if len(suggestions) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("  Research suggestions:")

	for _, s := range suggestions {
		marker := " "
		if s.Priority == "high" {
			marker = "!"
		}

		fmt.Printf("  %s %s\n", marker, s.Message)
	}
}

// hasMissingAncestors checks whether the ancestor tree has any dead ends
// (persons with no parents) within the given generations.
func hasMissingAncestors(node *treeNode) bool {
	if len(node.Children) == 0 {
		return true
	}
	for _, child := range node.Children {
		if hasMissingAncestors(child) {
			return true
		}
	}
	return false
}

// formatSuggestionMessage formats a suggestion for display, resolving place names.
func formatSuggestionMessage(s ancestorSuggestion) string {
	var parts []string
	if s.Priority == "high" {
		parts = append(parts, "HIGH PRIORITY")
	}
	parts = append(parts, s.Message)
	return strings.Join(parts, " — ")
}
