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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// deathYearUpperBound returns the effective upper bound year for census
// suggestions from a death date property value. Handles string, structured
// map ({value: "BEF 1870"}), and temporal list ([{value: "BEF 1870"}]) shapes.
// For "BEF <year>" dates, the year is decremented by 1 since the person
// died before that year. Calendar prefixes (e.g. "JULIAN BEF 1870") are
// stripped before the qualifier check.
func deathYearUpperBound(raw any) int {
	dateStr := extractDateString(raw)
	year := glxlib.ExtractFirstYear(dateStr)
	if year > 0 && strings.HasPrefix(dateStringWithoutCalendarPrefix(dateStr), "BEF ") {
		year--
	}
	return year
}

// deathYearFromEvent returns the death year upper bound from a person's death
// event. For "BEF <year>" dates the year is decremented by 1. Calendar
// prefixes (e.g. "JULIAN BEF 1870") are stripped before the qualifier check.
func deathYearFromEvent(archive *glxlib.GLXFile, personID string) int {
	_, event := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeDeath)
	if event == nil || event.Date == "" {
		return 0
	}
	dateStr := string(event.Date)
	year := glxlib.ExtractFirstYear(dateStr)
	if year > 0 && strings.HasPrefix(dateStringWithoutCalendarPrefix(dateStr), "BEF ") {
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

// minorAgeUnder is the cutoff for treating a child as living in the parent's
// household for census-suggestion consolidation. Children younger than this
// age at the census year normally appear on the same record as the parent,
// so one search covers both.
const minorAgeUnder = 18

// suggestCensusSearches recommends census years to search for persons who
// were alive during a census year but have no census event or citation for
// that year. When both a parent and a minor child are missing the same year,
// the parent's suggestion lists the children covered and the children's
// independent suggestions for that year are suppressed — minors live in the
// parent's household and would appear on the same record (#161).
func suggestCensusSearches(archive *glxlib.GLXFile) []AnalysisIssue {
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

	addCensusYearFromSources(archive, personCensusYears)
	personBurialYear := buildBurialYearIndex(archive)

	plans := buildCensusSuggestionPlans(archive, personCensusYears, personBurialYear)
	parentExtras, suppressed := consolidateParentChildCensus(archive, plans)

	return emitCensusSuggestions(archive, plans, parentExtras, suppressed)
}

// censusSuggestionPlan is the missing-year set computed for one person.
type censusSuggestionPlan struct {
	birthYear int
	missing   []int
}

// coveredChild records a minor child whose census suggestion is rolled up
// into a parent's suggestion for the same year.
type coveredChild struct {
	id  string
	age int
}

// buildCensusSuggestionPlans returns, for every person with a known birth
// year, the set of US federal census years between birth and death (capped at
// maxLifespan when death is unknown) that are not already represented by a
// census event or census source for that person.
func buildCensusSuggestionPlans(
	archive *glxlib.GLXFile,
	personCensusYears map[string]map[int]bool,
	personBurialYear map[string]int,
) map[string]*censusSuggestionPlan {
	plans := make(map[string]*censusSuggestionPlan)
	for _, id := range sortedPersonIDs(archive.Persons) {
		if archive.Persons[id] == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		if birthYear == 0 {
			continue
		}

		deathYear := deathYearFromEvent(archive, id)
		if deathYear == 0 {
			deathYear = personBurialYear[id]
		}
		upperBound := deathYear
		if upperBound == 0 {
			upperBound = birthYear + maxLifespan
		}

		existing := personCensusYears[id]
		var missing []int
		for _, censusYear := range usFederalCensusYears {
			if censusYear < birthYear || censusYear > upperBound || existing[censusYear] {
				continue
			}
			missing = append(missing, censusYear)
		}
		if len(missing) > 0 {
			plans[id] = &censusSuggestionPlan{birthYear: birthYear, missing: missing}
		}
	}

	return plans
}

// parentChildBound is the time window of one parent-child relationship
// record. A zero value on either side means that boundary is unknown
// and the consolidation pass treats it as open-ended.
type parentChildBound struct {
	startYear int
	endYear   int
}

// consolidateParentChildCensus finds (parent, year) pairs whose minor children
// also need that year, returning the children to mention under each parent's
// suggestion and the (childID, year) pairs whose independent suggestions
// should be suppressed. A relationship that is bounded by start_event /
// end_event (or the started_on / ended_on properties) only consolidates for
// census years where the relationship was active — a step-parent who married
// in 1860 cannot subsume their stepchild's 1830 census, and a foster
// relationship that ended in 1845 cannot subsume the child's 1850 census.
func consolidateParentChildCensus(
	archive *glxlib.GLXFile,
	plans map[string]*censusSuggestionPlan,
) (parentExtras map[string]map[int][]coveredChild, suppressed map[string]map[int]bool) {
	parentChildBounds := buildParentChildBoundsIndex(archive)
	parentExtras = make(map[string]map[int][]coveredChild)
	suppressed = make(map[string]map[int]bool)

	for _, parentID := range sortedPersonIDs(archive.Persons) {
		parentPlan := plans[parentID]
		if parentPlan == nil {
			continue
		}
		childMap := parentChildBounds[parentID]
		if len(childMap) == 0 {
			continue
		}
		parentMissing := yearSet(parentPlan.missing)

		childIDs := make([]string, 0, len(childMap))
		for cid := range childMap {
			childIDs = append(childIDs, cid)
		}
		sort.Strings(childIDs)

		for _, childID := range childIDs {
			recordConsolidatedChild(plans[childID], childID, parentID, parentMissing, childMap[childID], parentExtras, suppressed)
		}
	}

	return parentExtras, suppressed
}

// yearSet turns a list of years into a presence map for cheap membership tests.
func yearSet(years []int) map[int]bool {
	set := make(map[int]bool, len(years))
	for _, y := range years {
		set[y] = true
	}

	return set
}

// buildParentChildBoundsIndex returns parentID → childID → relationship
// bounds for every parent-child relationship in the archive. A pair may
// have multiple bound records when more than one relationship (e.g.
// biological + step) connects the same two people; consolidation treats
// the union of these windows as the active range.
func buildParentChildBoundsIndex(archive *glxlib.GLXFile) map[string]map[string][]parentChildBound {
	index := make(map[string]map[string][]parentChildBound)
	for _, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		startYear, endYear := relationshipYearBounds(archive, rel)
		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			if p.Person == "" {
				continue
			}
			switch p.Role {
			case glxlib.ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case glxlib.ParticipantRoleChild:
				childIDs = append(childIDs, p.Person)
			}
		}
		bound := parentChildBound{startYear: startYear, endYear: endYear}
		for _, parentID := range parentIDs {
			if index[parentID] == nil {
				index[parentID] = make(map[string][]parentChildBound)
			}
			for _, childID := range childIDs {
				index[parentID][childID] = append(index[parentID][childID], bound)
			}
		}
	}

	return index
}

// boundaryKind distinguishes whether an extracted relationship-bound year
// represents the active-from (start) side or the active-through (end) side
// of the active window. The kind controls how BEF / AFT qualifiers are
// translated into the inclusive boundary year used for membership tests.
type boundaryKind int

const (
	boundaryStart boundaryKind = iota
	boundaryEnd
)

// extractRelationshipBoundaryYear returns an inclusive boundary year for the
// given side of a relationship's active window. Qualified dates are
// interpreted relative to the side so that the boundary year itself is only
// reported as "active" when the qualifier supports it: AFT on a start shifts
// to year+1 (the relationship was not yet active in the named year); BEF on
// an end shifts to year-1 (the relationship had already ended in the named
// year). The other two combinations are treated conservatively at the named
// year: BEF on a start (we know the relationship had started by then, even
// though it may have been active earlier — keep the window closed below the
// named year); AFT on an end (we know the relationship was still active in
// the named year, but we have no evidence about how much later — keep the
// window closed above the named year, so we never falsely consolidate post-
// boundary years we cannot confirm). Unqualified dates and approximations
// (ABT / EST / CIRCA) use the extracted year directly.
func extractRelationshipBoundaryYear(dateStr string, kind boundaryKind) int {
	year := glxlib.ExtractFirstYear(dateStr)
	if year == 0 {
		return 0
	}

	upper := dateStringWithoutCalendarPrefix(dateStr)
	switch {
	case strings.HasPrefix(upper, "AFT "):
		if kind == boundaryStart {
			return year + 1
		}

		return year
	case strings.HasPrefix(upper, "BEF "):
		if kind == boundaryEnd {
			return year - 1
		}

		return year
	}

	return year
}

// dateStringWithoutCalendarPrefix returns the upper-cased body of a GLX date
// string with any leading calendar prefix removed (e.g. "JULIAN AFT 1731" →
// "AFT 1731"). It centralizes the prefix-stripping step that all of the
// qualifier-aware date helpers in this file rely on so they recognize BEF /
// AFT regardless of the calendar in which the date is expressed.
func dateStringWithoutCalendarPrefix(dateStr string) string {
	_, body := glxlib.ExtractCalendarPrefix(glxlib.DateString(strings.TrimSpace(dateStr)))

	return strings.ToUpper(string(body))
}

// relationshipYearBounds extracts a relationship's start and end years from
// its StartEvent / EndEvent (preferred) and falls back to the started_on /
// ended_on temporal properties. BEF and AFT qualifiers are translated by
// extractRelationshipBoundaryYear so that the named year is only reported as
// "active" when the qualifier actually places the relationship there.
func relationshipYearBounds(archive *glxlib.GLXFile, rel *glxlib.Relationship) (start, end int) {
	if rel.StartEvent != "" {
		if e, ok := archive.Events[rel.StartEvent]; ok && e != nil {
			start = extractRelationshipBoundaryYear(string(e.Date), boundaryStart)
		}
	}
	if rel.EndEvent != "" {
		if e, ok := archive.Events[rel.EndEvent]; ok && e != nil {
			end = extractRelationshipBoundaryYear(string(e.Date), boundaryEnd)
		}
	}
	if start == 0 {
		if v, ok := rel.Properties["started_on"]; ok {
			start = extractRelationshipBoundaryYear(extractDateString(v), boundaryStart)
		}
	}
	if end == 0 {
		if v, ok := rel.Properties["ended_on"]; ok {
			end = extractRelationshipBoundaryYear(extractDateString(v), boundaryEnd)
		}
	}

	return start, end
}

// wasActiveInYear reports whether any of the supplied bound records covers
// the given year. A zero start or end is treated as open on that side.
func wasActiveInYear(bounds []parentChildBound, year int) bool {
	for _, b := range bounds {
		if (b.startYear == 0 || year >= b.startYear) && (b.endYear == 0 || year <= b.endYear) {
			return true
		}
	}

	return false
}

// recordConsolidatedChild appends (childID, age) to parentExtras and marks
// (childID, year) as suppressed for every year the child is missing where
// the parent is also missing, the parent-child relationship was active,
// and the child was a minor (age < minorAgeUnder).
func recordConsolidatedChild(
	childPlan *censusSuggestionPlan,
	childID, parentID string,
	parentMissing map[int]bool,
	bounds []parentChildBound,
	parentExtras map[string]map[int][]coveredChild,
	suppressed map[string]map[int]bool,
) {
	if childPlan == nil {
		return
	}
	for _, year := range childPlan.missing {
		if !parentMissing[year] {
			continue
		}
		if !wasActiveInYear(bounds, year) {
			continue
		}
		age := year - childPlan.birthYear
		if age < 0 || age >= minorAgeUnder {
			continue
		}
		if parentExtras[parentID] == nil {
			parentExtras[parentID] = make(map[int][]coveredChild)
		}
		parentExtras[parentID][year] = append(parentExtras[parentID][year], coveredChild{id: childID, age: age})
		if suppressed[childID] == nil {
			suppressed[childID] = make(map[int]bool)
		}
		suppressed[childID][year] = true
	}
}

// emitCensusSuggestions turns the per-person plans into AnalysisIssues,
// skipping pairs suppressed by parent consolidation and appending a
// "would also cover" annotation to parent suggestions that subsume children.
func emitCensusSuggestions(
	archive *glxlib.GLXFile,
	plans map[string]*censusSuggestionPlan,
	parentExtras map[string]map[int][]coveredChild,
	suppressed map[string]map[int]bool,
) []AnalysisIssue {
	var issues []AnalysisIssue
	for _, id := range sortedPersonIDs(archive.Persons) {
		plan := plans[id]
		if plan == nil {
			continue
		}
		name := personName(archive, id)
		for _, year := range plan.missing {
			if suppressed[id][year] {
				continue
			}
			note := fmt.Sprintf("%s — search %d census (alive, no census event)", name, year)
			if year == 1890 {
				note += " — mostly destroyed (1921 fire)"
			}
			if extras := parentExtras[id][year]; len(extras) > 0 {
				parts := make([]string, 0, len(extras))
				for _, c := range extras {
					parts = append(parts, fmt.Sprintf("%s (~%d)", personName(archive, c.id), c.age))
				}
				note += " — would also cover: " + strings.Join(parts, ", ")
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
