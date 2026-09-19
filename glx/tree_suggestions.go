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
			Message:  name + " — no parent_child relationship found",
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

	// Key census years for finding parents, on the schedules of the countries
	// this person's places name (#186)
	for _, schedule := range censusSchedulesForPerson(archive, personID) {
		suggestions = append(suggestions,
			scheduleParentCensusSuggestions(schedule, personID, name, placeName, birthYear, existingCensus)...)
	}

	return suggestions
}

// scheduleParentCensusSuggestions returns the census years on one schedule
// that are most worth searching to identify a person's parents: the year that
// first recorded parents' birthplaces, the years the person was still a child
// and so enumerated in the parents' household, and the year that first named
// every individual.
func scheduleParentCensusSuggestions(
	schedule *censusSchedule,
	personID, name, placeName string,
	birthYear int,
	existingCensus map[int]bool,
) []ancestorSuggestion {
	var suggestions []ancestorSuggestion

	for _, year := range schedule.years {
		if year < birthYear || year > birthYear+maxLifespan {
			continue
		}
		if existingCensus[year] {
			continue
		}

		age := year - birthYear
		note := schedule.notes[year]

		// Focus on censuses most useful for parent research
		var priority string
		var reason string

		switch {
		case note.highPriority:
			priority = "high"
			reason = note.note
		case age < minorAgeUnder:
			priority = "high"
			reason = "likely in parents' household"
		case note.minorNote != "" && age >= minorAgeUnder && age <= censusPrimeAgeMax:
			priority = "medium"
			reason = note.note + "; may show in parents' household"
		default:
			continue // only suggest high-value censuses for ancestor research
		}

		location := ""
		if placeName != "" {
			location = ", " + placeName
		}

		suggestions = append(suggestions, ancestorSuggestion{
			PersonID: personID,
			Category: "census",
			Priority: priority,
			Year:     year,
			Message: fmt.Sprintf("%s — search %s (age ~%d%s) — %s",
				name, schedule.suggestionLabel(year), age, location, reason),
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
