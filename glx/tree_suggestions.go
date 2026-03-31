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
// A census year index is precomputed once to avoid repeated archive scans.
func buildAncestorSuggestions(tc *treeContext, rootPersonID string, archive *glxlib.GLXFile) []ancestorSuggestion {
	censusIndex := buildPersonCensusIndex(archive)

	var suggestions []ancestorSuggestion
	visited := make(map[string]bool)

	collectAncestorSuggestions(tc, rootPersonID, archive, censusIndex, 0, 3, visited, &suggestions)

	return suggestions
}

// personCensusIndex maps person IDs to their known census years.
type personCensusIndex map[string]map[int]bool

// buildPersonCensusIndex scans all events once to build a map of person → census years.
func buildPersonCensusIndex(archive *glxlib.GLXFile) personCensusIndex {
	index := make(personCensusIndex)
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}
		year := glxlib.ExtractFirstYear(string(event.Date))
		if year == 0 {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" {
				continue
			}
			if index[p.Person] == nil {
				index[p.Person] = make(map[int]bool)
			}
			index[p.Person][year] = true
		}
	}
	return index
}

// collectAncestorSuggestions recursively collects suggestions for persons
// missing parents in the ancestor tree.
func collectAncestorSuggestions(tc *treeContext, personID string, archive *glxlib.GLXFile, censusIndex personCensusIndex, depth, maxDepth int, visited map[string]bool, suggestions *[]ancestorSuggestion) {
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
			censusSuggestions := suggestParentCensusRecords(personID, person, archive, censusIndex)
			*suggestions = append(*suggestions, censusSuggestions...)
		}
	} else {
		// Has parents — recurse into them to find gaps further up
		for _, p := range parents {
			collectAncestorSuggestions(tc, p.personID, archive, censusIndex, depth+1, maxDepth, visited, suggestions)
		}
	}
}

// suggestParentCensusRecords suggests census records that are particularly
// useful for finding a person's parents. Uses the person's birth event to
// determine birth year and birthplace.
func suggestParentCensusRecords(personID string, person *glxlib.Person, archive *glxlib.GLXFile, censusIndex personCensusIndex) []ancestorSuggestion {
	_, birthEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth)
	if birthEvent == nil || birthEvent.Date == "" {
		return nil
	}
	birthYear := glxlib.ExtractFirstYear(string(birthEvent.Date))
	if birthYear == 0 {
		return nil
	}

	name := glxlib.PersonDisplayName(person)

	// Resolve birthplace from event PlaceID
	var placeName string
	if birthEvent.PlaceID != "" {
		if place, ok := archive.Places[birthEvent.PlaceID]; ok && place != nil {
			placeName = place.Name
		}
	}

	existingCensus := censusIndex[personID]

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
