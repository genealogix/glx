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

// deathYearUpperBound returns the effective upper bound year for census
// suggestions from a death date property value. Handles string, structured
// map ({value: "BEF 1870"}), and temporal list ([{value: "BEF 1870"}]) shapes.
// For "BEF <year>" dates, the year is decremented by 1 since the person
// died before that year.
func deathYearUpperBound(raw any) int {
	dateStr := extractDateString(raw)
	year := glxlib.ExtractFirstYear(dateStr)
	if year > 0 && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dateStr)), "BEF ") {
		year--
	}
	return year
}

// deathYearFromEvent returns the death year upper bound from a person's death
// event. For "BEF <year>" dates the year is decremented by 1.
func deathYearFromEvent(archive *glxlib.GLXFile, personID string) int {
	_, event := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeDeath)
	if event == nil || event.Date == "" {
		return 0
	}
	dateStr := string(event.Date)
	year := glxlib.ExtractFirstYear(dateStr)
	if year > 0 && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dateStr)), "BEF ") {
		year--
	}
	return year
}

// extractDateString extracts the date string from a property value,
// handling string, structured map, and temporal list shapes.
func extractDateString(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	case map[string]any:
		if val, ok := v["value"]; ok {
			return fmt.Sprint(val)
		}
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if val, ok := m["value"]; ok {
					return fmt.Sprint(val)
				}
			}
		}
	}
	return ""
}

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
	issues = append(issues, suggestChildCensusRecords(archive)...)

	return issues
}

// suggestCensusSearches recommends census years to search for persons who were
// alive during a census year but have no census event or citation for that year.
func suggestCensusSearches(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build index of census years each person already has — from events
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

	// Also index census years from citations/sources via assertions,
	// matching the detection logic used by `glx coverage`.
	addCensusYearFromSources(archive, personCensusYears)

	// Build index of burial event years per person for death inference
	personBurialYear := buildBurialYearIndex(archive)

	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		if birthYear == 0 {
			continue
		}

		deathYear := deathYearFromEvent(archive, id)

		// Infer death from burial event if no death event
		if deathYear == 0 {
			deathYear = personBurialYear[id]
		}

		// Cap at max lifespan when no death year is known
		upperBound := deathYear
		if upperBound == 0 {
			upperBound = birthYear + maxLifespan
		}

		name := personName(archive, id)
		existing := personCensusYears[id]

		var missing []int
		for _, censusYear := range usFederalCensusYears {
			if censusYear < birthYear {
				continue
			}
			if censusYear > upperBound {
				continue
			}
			if existing[censusYear] {
				continue
			}
			missing = append(missing, censusYear)
		}

		for _, year := range missing {
			note := fmt.Sprintf("%s — search %d census (alive, no census event)", name, year)
			if year == 1890 {
				note += " — mostly destroyed (1921 fire)"
			}
			issues = append(issues, AnalysisIssue{
				Category: "suggestion",
				Severity: "info",
				Person:   id,
				Message:  note,
			})
		}
	}

	return issues
}

// buildBurialYearIndex returns a map of person ID to the earliest burial event year.
// Only principal participants are considered (witnesses/officiants are excluded).
func buildBurialYearIndex(archive *glxlib.GLXFile) map[string]int {
	index := make(map[string]int)
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeBurial {
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
			if p.Role != "" && p.Role != "principal" && p.Role != "subject" {
				continue
			}
			if existing, ok := index[p.Person]; !ok || year < existing {
				index[p.Person] = year
			}
		}
	}
	return index
}

// addCensusYearFromSources indexes census years from citations and sources
// referenced by assertions, so that persons documented only via citations
// (not full census events) are not flagged as missing.
func addCensusYearFromSources(archive *glxlib.GLXFile, personCensusYears map[string]map[int]bool) {
	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		personID := assertion.Subject.Person
		if personID == "" {
			continue
		}

		// Check citations → sources
		for _, citID := range assertion.Citations {
			cit := archive.Citations[citID]
			if cit == nil {
				continue
			}
			indexCensusSource(archive.Sources[cit.SourceID], personID, personCensusYears)
		}

		// Check direct sources
		for _, srcID := range assertion.Sources {
			indexCensusSource(archive.Sources[srcID], personID, personCensusYears)
		}
	}
}

// indexCensusSource indexes census years from a source for a person.
// Checks the source date first, then matches any census year mentioned
// in the title (aligning with findCensusMatch in coverage_runner.go).
func indexCensusSource(src *glxlib.Source, personID string, personCensusYears map[string]map[int]bool) {
	if src == nil || src.Type != glxlib.SourceTypeCensus {
		return
	}

	// Try source date first
	year := glxlib.ExtractFirstYear(string(src.Date))
	if year > 0 {
		if personCensusYears[personID] == nil {
			personCensusYears[personID] = make(map[int]bool)
		}
		personCensusYears[personID][year] = true
		return
	}

	// Fall back to matching any census year in the title
	for _, censusYear := range usFederalCensusYears {
		if strings.Contains(src.Title, fmt.Sprintf("%d", censusYear)) {
			if personCensusYears[personID] == nil {
				personCensusYears[personID] = make(map[int]bool)
			}
			personCensusYears[personID][censusYear] = true
		}
	}
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

		_, birthEvent := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeBirth)
		_, deathEvent := glxlib.FindPersonEvent(archive, id, glxlib.EventTypeDeath)
		hasBirthDate := birthEvent != nil && birthEvent.Date != ""
		hasDeathDate := deathEvent != nil && deathEvent.Date != ""
		if !hasBirthDate && !hasDeathDate {
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

