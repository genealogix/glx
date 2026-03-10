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

// usFederalCensusYears lists U.S. Federal Census years.
var usFederalCensusYears = []int{
	1790, 1800, 1810, 1820, 1830, 1840, 1850, 1860, 1870,
	1880, 1890, 1900, 1910, 1920, 1930, 1940, 1950,
}

// analyzeSuggestions generates research recommendations based on archive data.
func analyzeSuggestions(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	issues = append(issues, suggestCensusSearches(archive)...)
	issues = append(issues, suggestVitalRecords(archive)...)

	return issues
}

// suggestCensusSearches recommends census years to search for persons who were
// alive during a census year but have no census event for that year.
func suggestCensusSearches(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build index of census years each person already has
	personCensusYears := make(map[string]map[int]bool)
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}
		year := glxlib.ExtractFirstYear(string(event.Date))
		if year == 0 {
			continue
		}
		for _, p := range event.Participants {
			if personCensusYears[p.Person] == nil {
				personCensusYears[p.Person] = make(map[int]bool)
			}
			personCensusYears[p.Person][year] = true
		}
	}

	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := glxlib.ExtractPropertyYear(person.Properties, "born_on")
		if birthYear == 0 {
			continue
		}

		deathYear := glxlib.ExtractPropertyYear(person.Properties, "died_on")

		name := personName(archive, id)
		existing := personCensusYears[id]

		var missing []int
		for _, censusYear := range usFederalCensusYears {
			if censusYear < birthYear {
				continue
			}
			if deathYear > 0 && censusYear > deathYear {
				continue
			}
			if existing[censusYear] {
				continue
			}
			missing = append(missing, censusYear)
		}

		for _, year := range missing {
			issues = append(issues, AnalysisIssue{
				Category: "suggestion",
				Severity: "info",
				Person:   id,
				Message:  fmt.Sprintf("%s — search %d census (alive, no census event)", name, year),
			})
		}
	}

	return issues
}

// suggestVitalRecords recommends searching for vital records when a person has
// approximate birth/death dates but no vital_record source type is cited.
func suggestVitalRecords(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build set of persons who have a vital_record source
	personsWithVitals := make(map[string]bool)

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		personID := assertion.Subject.Person
		if personID == "" {
			continue
		}

		hasVitalSource := false

		for _, sourceID := range assertion.Sources {
			source := archive.Sources[sourceID]
			if source != nil && source.Type == glxlib.SourceTypeVitalRecord {
				hasVitalSource = true
				break
			}
		}

		if !hasVitalSource {
			for _, citID := range assertion.Citations {
				cit := archive.Citations[citID]
				if cit == nil {
					continue
				}
				source := archive.Sources[cit.SourceID]
				if source != nil && source.Type == glxlib.SourceTypeVitalRecord {
					hasVitalSource = true
					break
				}
			}
		}

		if hasVitalSource {
			personsWithVitals[personID] = true
		}
	}

	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		if personsWithVitals[id] {
			continue
		}

		person := archive.Persons[id]
		if person == nil {
			continue
		}

		bornOn := propertyString(person.Properties, "born_on")
		diedOn := propertyString(person.Properties, "died_on")
		if bornOn == "" && diedOn == "" {
			continue
		}

		name := personName(archive, id)
		issues = append(issues, AnalysisIssue{
			Category: "suggestion",
			Severity: "info",
			Person:   id,
			Message:  fmt.Sprintf("%s — search vital records (dates exist but no vital record source)", name),
		})
	}

	return issues
}

