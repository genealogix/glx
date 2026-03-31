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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// US federal census years.
var usCensusYears = []int{
	1790, 1800, 1810, 1820, 1830, 1840, 1850, 1860, 1870, 1880,
	1890, 1900, 1910, 1920, 1930, 1940, 1950,
}

// maxLifespan is the assumed maximum lifespan for capping census suggestions
// when no death date is known.
const maxLifespan = 100

// coverageRecord represents one expected record in the coverage checklist.
type coverageRecord struct {
	Category    string `json:"category"`
	Label       string `json:"label"`
	Found       bool   `json:"found"`
	SourceRef   string `json:"source_ref,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

// coverageResult holds the full coverage output for a person.
type coverageResult struct {
	PersonID   string           `json:"person_id"`
	PersonName string           `json:"person_name"`
	BornOn     string           `json:"born_on,omitempty"`
	BornAt     string           `json:"born_at,omitempty"`
	DiedOn     string           `json:"died_on,omitempty"`
	DiedAt     string           `json:"died_at,omitempty"`
	Records    []coverageRecord `json:"records"`
	Found      int              `json:"found"`
	Expected   int              `json:"expected"`
}

// showCoverage loads an archive and displays source coverage for a person.
func showCoverage(archivePath, personQuery string, jsonOutput bool) error {
	archive, err := loadArchiveForCoverage(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPersonForCoverage(archive, personQuery)
	if err != nil {
		return err
	}

	result := buildCoverage(personID, person, archive)

	if jsonOutput {
		return printCoverageJSON(result)
	}

	printCoverageText(result)
	return nil
}

// loadArchiveForCoverage loads an archive from a path.
func loadArchiveForCoverage(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// findPersonForCoverage finds a person by ID or name substring.
func findPersonForCoverage(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	if person, ok := archive.Persons[query]; ok && person != nil {
		return query, person, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		name := glxlib.PersonDisplayName(person)
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			matches = append(matches, id)
		}
	}

	switch len(matches) {
	case 0:
		return "", nil, fmt.Errorf("no person found matching %q", query)
	case 1:
		return matches[0], archive.Persons[matches[0]], nil
	default:
		var lines []string
		for _, id := range matches {
			name := glxlib.PersonDisplayName(archive.Persons[id])
			lines = append(lines, fmt.Sprintf("  %s  %s", id, name))
		}
		return "", nil, fmt.Errorf("multiple persons match %q:\n%s\nUse exact person ID", query, strings.Join(lines, "\n"))
	}
}

// buildCoverage generates the coverage checklist for a person.
func buildCoverage(personID string, person *glxlib.Person, archive *glxlib.GLXFile) *coverageResult {
	var bornOn, bornAt, diedOn, diedAt string
	if _, birthEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth); birthEvent != nil {
		bornOn = string(birthEvent.Date)
		bornAt = birthEvent.PlaceID
	}
	if _, deathEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeDeath); deathEvent != nil {
		diedOn = string(deathEvent.Date)
		diedAt = deathEvent.PlaceID
	}

	birthYear := glxlib.ExtractFirstYear(bornOn)
	deathYear := deathYearUpperBound(diedOn)

	// Build indexes: what sources/citations/events reference this person
	personSources := collectPersonSources(personID, archive)
	personEvents := collectPersonEvents(personID, archive)

	// Infer death year from burial event if died_on is not set
	if deathYear == 0 {
		deathYear = inferDeathYearFromEvents(personEvents)
	}

	var records []coverageRecord

	// Federal census records
	records = append(records, buildCensusRecords(birthYear, deathYear, personSources, personEvents)...)

	// State census records
	states := collectPersonStates(person, archive, personEvents)
	records = append(records, buildStateCensusRecords(birthYear, deathYear, states, personSources, personEvents, archive)...)

	// Vital records
	records = append(records, buildVitalRecords(personID, person, archive, personSources, personEvents)...)

	// Other record types — probate is high priority when person has an explicit death
	// date (not just inferred from burial) and known family
	probateHighPriority := diedOn != "" && hasFamily(personID, archive)
	records = append(records, buildOtherRecords(personSources, personEvents, probateHighPriority)...)

	found := 0
	for _, r := range records {
		if r.Found {
			found++
		}
	}

	bornAtName := coverageResolvePlaceName(bornAt, archive)
	diedAtName := coverageResolvePlaceName(diedAt, archive)

	return &coverageResult{
		PersonID:   personID,
		PersonName: glxlib.PersonDisplayName(person),
		BornOn:     bornOn,
		BornAt:     bornAtName,
		DiedOn:     diedOn,
		DiedAt:     diedAtName,
		Records:    records,
		Found:      found,
		Expected:   len(records),
	}
}

// personSourceInfo tracks a source or citation found for a person.
type personSourceInfo struct {
	Ref       string // source or citation ID
	Type      string // source type
	Title     string
	EventType string // if found via an event
	PlaceID   string // place reference (events only)
	Year      int
}

// collectPersonSources gathers all sources and citations that reference a person
// via assertions.
func collectPersonSources(personID string, archive *glxlib.GLXFile) []personSourceInfo {
	var sources []personSourceInfo
	seen := make(map[string]bool)

	// From assertions about this person
	for _, assertion := range archive.Assertions {
		if assertion == nil || assertion.Subject.ID() != personID {
			continue
		}
		for _, citID := range assertion.Citations {
			if seen[citID] {
				continue
			}
			seen[citID] = true
			cit := archive.Citations[citID]
			if cit == nil {
				continue
			}
			src := archive.Sources[cit.SourceID]
			info := personSourceInfo{Ref: citID}
			if src != nil {
				info.Type = src.Type
				info.Title = src.Title
				info.Year = glxlib.ExtractFirstYear(string(src.Date))
			}
			sources = append(sources, info)
		}
		for _, srcID := range assertion.Sources {
			if seen[srcID] {
				continue
			}
			seen[srcID] = true
			src := archive.Sources[srcID]
			if src == nil {
				continue
			}
			sources = append(sources, personSourceInfo{
				Ref:   srcID,
				Type:  src.Type,
				Title: src.Title,
				Year:  glxlib.ExtractFirstYear(string(src.Date)),
			})
		}
	}

	return sources
}

// collectPersonEvents gathers all events this person participates in.
func collectPersonEvents(personID string, archive *glxlib.GLXFile) []personSourceInfo {
	var events []personSourceInfo

	for eventID, event := range archive.Events {
		if event == nil {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				events = append(events, personSourceInfo{
					Ref:       eventID,
					EventType: event.Type,
					Year:      glxlib.ExtractFirstYear(string(event.Date)),
					Title:     event.Title,
					PlaceID:   event.PlaceID,
				})
				break
			}
		}
	}

	return events
}

// buildCensusRecords generates expected census records based on birth/death years.
func buildCensusRecords(birthYear, deathYear int, sources []personSourceInfo, events []personSourceInfo) []coverageRecord {
	if birthYear == 0 {
		return nil
	}

	// Cap at max lifespan when no death year is known
	upperBound := deathYear
	if upperBound == 0 {
		upperBound = birthYear + maxLifespan
	}

	var records []coverageRecord

	for _, year := range usCensusYears {
		if year < birthYear {
			continue
		}
		if year > upperBound {
			break
		}
		// Approximate age at this census year (may be 0 if census year == birth year)
		age := year - birthYear

		label := fmt.Sprintf("%d US Census (age ~%d)", year, age)

		rec := coverageRecord{
			Category: "census",
			Label:    label,
		}

		// 1890 census was mostly destroyed in a 1921 fire
		if year == 1890 {
			rec.Description = "mostly destroyed (1921 fire)"
		}

		// Check if we have this census
		ref := findCensusMatch(year, sources, events)
		if ref != "" {
			rec.Found = true
			rec.SourceRef = ref
		}

		// Census-specific annotations (always added, even when found)
		rec.Description = appendCensusAnnotation(rec.Description, year, age)

		// Priority annotations for missing records
		if !rec.Found {
			if year == 1880 {
				rec.Priority = "high"
			} else if age >= 14 && age <= 25 {
				rec.Priority = "high"
				// Avoid duplicating parents-household note when 1850 minor annotation already applies
				if !(year == 1850 && age < 18) {
					rec.Description = appendDescription(rec.Description, "may show in parents' household")
				}
			}
		}

		records = append(records, rec)
	}

	return records
}

// findCensusMatch checks if a census for a given year exists in sources or events.
func findCensusMatch(year int, sources []personSourceInfo, events []personSourceInfo) string {
	for _, e := range events {
		if e.EventType == glxlib.EventTypeCensus && e.Year == year {
			return e.Ref
		}
	}
	for _, s := range sources {
		if s.Type == glxlib.SourceTypeCensus && s.Year == year {
			return s.Ref
		}
		// Also check title for census year mentions
		if s.Type == glxlib.SourceTypeCensus && strings.Contains(s.Title, fmt.Sprintf("%d", year)) {
			return s.Ref
		}
	}
	return ""
}

// buildVitalRecords generates expected vital records.
func buildVitalRecords(personID string, person *glxlib.Person, archive *glxlib.GLXFile, sources []personSourceInfo, events []personSourceInfo) []coverageRecord {
	var records []coverageRecord

	// Birth record
	birthFound := hasEventType(events, glxlib.EventTypeBirth) || hasSourceType(sources, glxlib.SourceTypeVitalRecord, "birth")
	records = append(records, coverageRecord{
		Category:  "vital",
		Label:     "Birth record",
		Found:     birthFound,
		SourceRef: findEventRef(events, glxlib.EventTypeBirth),
		Priority:  boolPriority(!birthFound, "high"),
	})

	// Death record
	deathFound := hasEventType(events, glxlib.EventTypeDeath) || hasSourceType(sources, glxlib.SourceTypeVitalRecord, "death")
	records = append(records, coverageRecord{
		Category:  "vital",
		Label:     "Death record",
		Found:     deathFound,
		SourceRef: findEventRef(events, glxlib.EventTypeDeath),
		Priority:  boolPriority(!deathFound, "medium"),
	})

	// Marriage records — check relationships for spouse
	marriageRecords := buildMarriageRecords(personID, archive, events)
	records = append(records, marriageRecords...)

	return records
}

// buildMarriageRecords checks for marriage events linked to spouse relationships.
func buildMarriageRecords(personID string, archive *glxlib.GLXFile, events []personSourceInfo) []coverageRecord {
	var records []coverageRecord

	// Find spouse relationships
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if rel.Type != glxlib.RelationshipTypeMarriage && rel.Type != glxlib.RelationshipTypePartner {
			continue
		}

		var spouseID string
		isParticipant := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				isParticipant = true
			} else {
				spouseID = p.Person
			}
		}
		if !isParticipant {
			continue
		}

		spouseName := ""
		if spouse, ok := archive.Persons[spouseID]; ok && spouse != nil {
			spouseName = glxlib.PersonDisplayName(spouse)
		}
		if spouseName == "" {
			spouseName = spouseID
		}

		label := fmt.Sprintf("Marriage record — %s", spouseName)

		// Check if there's a marriage event for this relationship
		found := false
		ref := ""
		if rel.StartEvent != "" {
			if ev, ok := archive.Events[rel.StartEvent]; ok && ev != nil && ev.Type == glxlib.EventTypeMarriage {
				found = true
				ref = rel.StartEvent
			}
		}
		if !found {
			// Fall back to checking for a marriage event that involves both this person and this spouse
			for eventID, ev := range archive.Events {
				if ev == nil || ev.Type != glxlib.EventTypeMarriage {
					continue
				}
				hasPerson := false
				hasSpouse := false
				for _, ep := range ev.Participants {
					if ep.Person == personID {
						hasPerson = true
					}
					if ep.Person == spouseID {
						hasSpouse = true
					}
				}
				if hasPerson && hasSpouse {
					found = true
					ref = eventID
					break
				}
			}
		}

		rec := coverageRecord{
			Category:  "vital",
			Label:     label,
			Found:     found,
			SourceRef: ref,
			Priority:  boolPriority(!found, "medium"),
		}
		records = append(records, rec)
	}

	return records
}

// buildOtherRecords generates records for probate, land, military, church.
// When probateHighPriority is true (person died with known family), probate
// is elevated to HIGH priority because probate records name heirs.
func buildOtherRecords(sources []personSourceInfo, events []personSourceInfo, probateHighPriority bool) []coverageRecord {
	var records []coverageRecord

	// Probate/will
	probateFound := hasEventType(events, glxlib.EventTypeProbate) || hasEventType(events, glxlib.EventTypeWill) ||
		hasSourceType(sources, glxlib.SourceTypeProbate, "")
	rec := coverageRecord{
		Category:  "other",
		Label:     "Probate/will",
		Found:     probateFound,
		SourceRef: findEventRef(events, glxlib.EventTypeProbate),
	}
	if !probateFound && probateHighPriority {
		rec.Priority = "high"
		rec.Description = "often names heirs (children) and surviving spouse"
	}
	records = append(records, rec)

	// Land records
	landFound := hasSourceType(sources, glxlib.SourceTypeLand, "")
	records = append(records, coverageRecord{
		Category:  "other",
		Label:     "Land records",
		Found:     landFound,
		SourceRef: findSourceRef(sources, glxlib.SourceTypeLand),
	})

	// Military records
	militaryFound := hasSourceType(sources, glxlib.SourceTypeMilitary, "")
	records = append(records, coverageRecord{
		Category:  "other",
		Label:     "Military records",
		Found:     militaryFound,
		SourceRef: findSourceRef(sources, glxlib.SourceTypeMilitary),
	})

	// Church records
	churchFound := hasSourceType(sources, glxlib.SourceTypeChurchRegister, "") ||
		hasEventType(events, glxlib.EventTypeBaptism) || hasEventType(events, glxlib.EventTypeChristening)
	records = append(records, coverageRecord{
		Category:  "other",
		Label:     "Church records",
		Found:     churchFound,
		SourceRef: findEventRef(events, glxlib.EventTypeBaptism),
	})

	return records
}

// coverageResolvePlaceName returns the place name for a place ID, or the raw string.
func coverageResolvePlaceName(placeRef string, archive *glxlib.GLXFile) string {
	if placeRef == "" {
		return ""
	}
	if place, ok := archive.Places[placeRef]; ok && place != nil {
		return place.Name
	}
	return placeRef
}

func hasEventType(events []personSourceInfo, eventType string) bool {
	for _, e := range events {
		if e.EventType == eventType {
			return true
		}
	}
	return false
}

func hasSourceType(sources []personSourceInfo, sourceType, titleKeyword string) bool {
	for _, s := range sources {
		if s.Type == sourceType {
			if titleKeyword == "" || strings.Contains(strings.ToLower(s.Title), titleKeyword) {
				return true
			}
		}
	}
	return false
}

func findEventRef(events []personSourceInfo, eventType string) string {
	for _, e := range events {
		if e.EventType == eventType {
			return e.Ref
		}
	}
	return ""
}

func findSourceRef(sources []personSourceInfo, sourceType string) string {
	for _, s := range sources {
		if s.Type == sourceType {
			return s.Ref
		}
	}
	return ""
}

func boolPriority(condition bool, priority string) string {
	if condition {
		return priority
	}
	return ""
}

// printCoverageText prints coverage in a human-readable format.
func printCoverageText(result *coverageResult) {
	name := result.PersonName
	if name == "" {
		name = result.PersonID
	}

	fmt.Printf("Source Coverage for %s (%s)\n", name, result.PersonID)

	// Summary line
	var parts []string
	if result.BornOn != "" {
		born := "Born: " + result.BornOn
		if result.BornAt != "" {
			born += ", " + result.BornAt
		}
		parts = append(parts, born)
	}
	if result.DiedOn != "" {
		died := "Died: " + result.DiedOn
		if result.DiedAt != "" {
			died += ", " + result.DiedAt
		}
		parts = append(parts, died)
	} else {
		parts = append(parts, "Died: unknown")
	}
	if len(parts) > 0 {
		fmt.Printf("%s\n", strings.Join(parts, " | "))
	}

	// Group by category
	categories := []struct {
		key   string
		label string
	}{
		{"census", "Census Records"},
		{"vital", "Vital Records"},
		{"other", "Other Records"},
	}

	for _, cat := range categories {
		var catRecords []coverageRecord
		for _, r := range result.Records {
			if r.Category == cat.key {
				catRecords = append(catRecords, r)
			}
		}
		if len(catRecords) == 0 {
			continue
		}

		fmt.Printf("\n  %s:\n", cat.label)
		for _, r := range catRecords {
			marker := "[ ]"
			if r.Found {
				marker = "[x]"
			}

			line := fmt.Sprintf("    %s %s", marker, r.Label)

			if r.Found && r.SourceRef != "" {
				line += fmt.Sprintf(" (via %s)", r.SourceRef)
			}

			if !r.Found && r.Priority == "high" {
				line += " -- HIGH PRIORITY"
			}

			if r.Description != "" {
				line += fmt.Sprintf(" -- %s", r.Description)
			}

			fmt.Println(line)
		}
	}

	fmt.Printf("\n  Coverage: %d of %d expected records found (%d%%)\n",
		result.Found, result.Expected, coveragePercent(result.Found, result.Expected))
}

func coveragePercent(found, expected int) int {
	if expected == 0 {
		return 0
	}
	return (found * 100) / expected
}

// inferDeathYearFromEvents returns a death year inferred from burial events.
// When multiple burial events exist, returns the earliest year.
// Returns 0 if no burial event with a date is found.
func inferDeathYearFromEvents(events []personSourceInfo) int {
	earliest := 0
	for _, e := range events {
		if e.EventType == glxlib.EventTypeBurial && e.Year > 0 {
			if earliest == 0 || e.Year < earliest {
				earliest = e.Year
			}
		}
	}
	return earliest
}

// appendDescription appends text to an existing description, using "; " as separator.
func appendDescription(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + "; " + addition
}

// appendCensusAnnotation adds research-relevant notes for specific census years.
func appendCensusAnnotation(desc string, year, age int) string {
	switch year {
	case 1850:
		desc = appendDescription(desc, "first census to list individual names")
		if age < 18 {
			desc = appendDescription(desc, "likely in parents' household")
		}
	case 1880:
		desc = appendDescription(desc, "first census to list parents' birthplaces")
	}
	return desc
}

// hasFamily returns true if the person has any spouse or child relationships.
func hasFamily(personID string, archive *glxlib.GLXFile) bool {
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}

		isParticipant := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				isParticipant = true
				break
			}
		}
		if !isParticipant {
			continue
		}

		// Check for spouse/partner relationship — require spouse role to avoid
		// counting witnesses/officiants as family
		if rel.Type == glxlib.RelationshipTypeMarriage || rel.Type == glxlib.RelationshipTypePartner {
			for _, p := range rel.Participants {
				if p.Person == personID && p.Role == glxlib.ParticipantRoleSpouse {
					return true
				}
			}
		}

		// Check for parent-child where this person is the parent
		if isParentChildType(rel.Type) {
			for _, p := range rel.Participants {
				if p.Person == personID && p.Role == glxlib.ParticipantRoleParent {
					return true
				}
			}
		}
	}
	return false
}

// printCoverageJSON outputs the result as JSON.
func printCoverageJSON(result *coverageResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
