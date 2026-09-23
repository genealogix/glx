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
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// maxLifespan is the assumed maximum lifespan for capping census suggestions
// when no death date is known.
const maxLifespan = 100

// Coverage checklist categories, as they appear in the JSON output and as the
// headings printCoverageText groups by. coverageCategoryCensus shares the
// literal value of EventTypeCensus.
const (
	coverageCategoryCensus = "census"
	categoryVital          = "vital"
	categoryOther          = "other"
)

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
	BirthDate  string           `json:"birth_date,omitempty"`
	BirthPlace string           `json:"birth_place,omitempty"`
	DeathDate  string           `json:"death_date,omitempty"`
	DeathPlace string           `json:"death_place,omitempty"`
	Records    []coverageRecord `json:"records"`
	Found      int              `json:"found"`
	Expected   int              `json:"expected"`
}

// showCoverage loads an archive and displays source coverage for a person.
func showCoverage(archivePath, personQuery, country string, jsonOutput bool) error {
	if err := applyCensusCountry(country); err != nil {
		return err
	}

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
		archive, duplicates, loadErr := LoadArchiveCached(path)
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
	var birthDate, birthPlace, deathDate, deathPlace string
	if _, birthEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth); birthEvent != nil {
		birthDate = string(birthEvent.Date)
		birthPlace = birthEvent.PlaceID
	}
	if _, deathEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeDeath); deathEvent != nil {
		deathDate = string(deathEvent.Date)
		deathPlace = deathEvent.PlaceID
	}

	birthYear := glxlib.ExtractFirstYear(birthDate)
	deathYear := deathYearUpperBound(deathDate)

	// Build indexes: what sources/citations/events reference this person, and
	// which events an assertion actually backs with a citation or source
	personSources := collectPersonSources(personID, archive)
	evidencedEvents := eventsWithEvidence(archive)
	personEvents := collectPersonEvents(personID, archive, evidencedEvents)

	// Infer death year from burial event if death_date is not set
	if deathYear == 0 {
		deathYear = inferDeathYearFromEvents(personEvents)
	}

	var records []coverageRecord

	// National census records, for the countries the person's places name
	schedules := censusSchedulesForPlaces(coveragePlaceRefs(personEvents), archive)
	records = append(records, buildCensusRecords(birthYear, deathYear, schedules, personSources, personEvents)...)

	// State census records
	states := collectPersonStates(person, archive, personEvents)
	records = append(records, buildStateCensusRecords(birthYear, deathYear, states, personSources, personEvents, archive)...)

	// Vital records
	records = append(records, buildVitalRecords(personID, archive, personSources, personEvents, evidencedEvents)...)

	// Other record types — probate is high priority when person has an explicit death
	// date (not just inferred from burial) and known family
	probateHighPriority := deathDate != "" && hasFamily(personID, archive)
	records = append(records, buildOtherRecords(personSources, personEvents, probateHighPriority)...)

	found := 0
	for _, r := range records {
		if r.Found {
			found++
		}
	}

	birthPlaceName := coverageResolvePlaceName(birthPlace, archive)
	deathPlaceName := coverageResolvePlaceName(deathPlace, archive)

	return &coverageResult{
		PersonID:   personID,
		PersonName: glxlib.PersonDisplayName(person),
		BirthDate:  birthDate,
		BirthPlace: birthPlaceName,
		DeathDate:  deathDate,
		DeathPlace: deathPlaceName,
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
	Evidenced bool // events only: an assertion about this event cites a source
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

// collectPersonEvents gathers all events this person participates in, marking
// each with whether it carries supporting evidence per the evidenced index.
func collectPersonEvents(personID string, archive *glxlib.GLXFile, evidenced map[string]bool) []personSourceInfo {
	var events []personSourceInfo

	// Sorted so a person with more than one event of a type reports the same
	// one on every run
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
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
					Evidenced: evidenced[eventID],
				})

				break
			}
		}
	}

	return events
}

// eventsWithEvidence returns the set of event IDs backed by evidence: those
// that are the subject of an assertion resolving at least one citation or
// source.
//
// An event carries no citations or sources of its own — under the GLX evidence
// model the event is a conclusion, and what supports it is the assertion
// pointing at it. An event with no such assertion is therefore an unsupported
// claim (an estimate reckoned from a relative's record, a placeholder, a
// GEDCOM import), which coverage must not count as a record found.
func eventsWithEvidence(archive *glxlib.GLXFile) map[string]bool {
	evidenced := make(map[string]bool)

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		eventID := assertion.Subject.Event
		if eventID == "" || evidenced[eventID] {
			continue
		}
		if assertionHasEvidence(assertion, archive) {
			evidenced[eventID] = true
		}
	}

	return evidenced
}

// assertionHasEvidence reports whether an assertion resolves at least one
// citation or source. A dangling reference does not count: reference
// validation already reports it as an error, and honoring it here would let a
// typo stand in for a record.
func assertionHasEvidence(assertion *glxlib.Assertion, archive *glxlib.GLXFile) bool {
	for _, citID := range assertion.Citations {
		if cit := archive.Citations[citID]; cit != nil {
			return true
		}
	}

	for _, srcID := range assertion.Sources {
		if src := archive.Sources[srcID]; src != nil {
			return true
		}
	}

	return false
}

// coveragePlaceRefs returns the place references of a person's events, for
// resolving which countries' census schedules apply to them.
func coveragePlaceRefs(events []personSourceInfo) []string {
	var refs []string
	for _, e := range events {
		if e.PlaceID != "" {
			refs = append(refs, e.PlaceID)
		}
	}

	return refs
}

// censusPrimeAgeMin and censusPrimeAgeMax bound the ages at which a missing
// census is flagged high priority: young adults move between households, so
// the record that places them is the one most worth finding.
const (
	censusPrimeAgeMin = 14
	censusPrimeAgeMax = 25
)

// buildCensusRecords generates expected census records for every supplied
// schedule, bounded by the person's birth and death years. A person with no
// applicable schedule — one whose places name a country GLX has no census
// schedule for — gets no census rows at all (#186).
func buildCensusRecords(birthYear, deathYear int, schedules []*censusSchedule, sources, events []personSourceInfo) []coverageRecord {
	if birthYear == 0 {
		return nil
	}

	// Cap at max lifespan when no death year is known
	upperBound := deathYear
	if upperBound == 0 {
		upperBound = birthYear + maxLifespan
	}

	var records []coverageRecord

	for _, schedule := range schedules {
		for _, year := range schedule.years {
			if year < birthYear {
				continue
			}
			if year > upperBound {
				break
			}
			// Approximate age at this census year (may be 0 if census year == birth year)
			age := year - birthYear
			note := schedule.notes[year]

			rec := coverageRecord{
				Category: coverageCategoryCensus,
				Label:    schedule.coverageLabel(year, age),
			}

			// Check if we have this census
			ref := findCensusMatch(year, sources, events)
			if ref != "" {
				rec.Found = true
				rec.SourceRef = ref
			}

			// Census-specific annotations (always added, even when found)
			rec.Description = appendCensusAnnotation(rec.Description, note, age)

			if !rec.Found {
				rec.Description = appendDescription(rec.Description, unevidencedNote(findUnevidencedCensus(year, events)))
			}

			// Priority annotations for missing records
			if !rec.Found {
				switch {
				case note.highPriority:
					rec.Priority = severityHigh
				case age >= censusPrimeAgeMin && age <= censusPrimeAgeMax:
					rec.Priority = severityHigh
					// Avoid duplicating the parents-household note when the
					// year's own minor annotation already said it
					if note.minorNote == "" || age >= minorAgeUnder {
						rec.Description = appendDescription(rec.Description, "may show in parents' household")
					}
				}
			}

			records = append(records, rec)
		}
	}

	return records
}

// findCensusMatch checks if a census for a given year exists in sources or events.
func findCensusMatch(year int, sources, events []personSourceInfo) string {
	for _, e := range events {
		if e.EventType == glxlib.EventTypeCensus && e.Year == year && e.Evidenced {
			return e.Ref
		}
	}
	for _, s := range sources {
		if s.Type == glxlib.SourceTypeCensus && s.Year == year {
			return s.Ref
		}
		// Also check title for census year mentions
		if s.Type == glxlib.SourceTypeCensus && strings.Contains(s.Title, strconv.Itoa(year)) {
			return s.Ref
		}
	}

	return ""
}

// findUnevidencedCensus returns the ID of a census event for the given year
// that nothing backs, or "" when the person has none.
func findUnevidencedCensus(year int, events []personSourceInfo) string {
	for _, e := range events {
		if e.EventType == glxlib.EventTypeCensus && e.Year == year && !e.Evidenced {
			return e.Ref
		}
	}

	return ""
}

// buildVitalRecords generates expected vital records.
func buildVitalRecords(personID string, archive *glxlib.GLXFile, sources, events []personSourceInfo, evidenced map[string]bool) []coverageRecord {
	var records []coverageRecord

	records = append(records,
		buildVitalRecord("Birth record", severityHigh, glxlib.EventTypeBirth, "birth", sources, events),
		buildVitalRecord("Death record", severityMedium, glxlib.EventTypeDeath, "death", sources, events),
	)

	// Marriage records — check relationships for spouse
	marriageRecords := buildMarriageRecords(personID, archive, evidenced)
	records = append(records, marriageRecords...)

	return records
}

// buildVitalRecord builds one birth-or-death checklist entry. The record counts
// as found on an event only when that event carries evidence; an event standing
// on its own is a conclusion, and counting it would inflate the score for
// exactly the people whose records are missing.
func buildVitalRecord(label, missingPriority, eventType, titleKeyword string, sources, events []personSourceInfo) coverageRecord {
	eventRef := findEvidencedEvent(events, eventType)
	found := eventRef != "" || hasSourceType(sources, glxlib.SourceTypeVitalRecord, titleKeyword)

	rec := coverageRecord{
		Category: categoryVital,
		Label:    label,
		Found:    found,
		Priority: boolPriority(!found, missingPriority),
	}
	if found {
		rec.SourceRef = eventRef
		if rec.SourceRef == "" {
			rec.SourceRef = findSourceRef(sources, glxlib.SourceTypeVitalRecord)
		}
	} else {
		rec.Description = unevidencedNote(findUnevidencedEvent(events, eventType))
	}

	return rec
}

// buildMarriageRecords checks for marriage events linked to spouse
// relationships. As with birth and death, the marriage event has to carry
// evidence before it counts as the marriage record.
func buildMarriageRecords(personID string, archive *glxlib.GLXFile, evidenced map[string]bool) []coverageRecord {
	var records []coverageRecord

	// Find spouse relationships
	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if !glxlib.IsCoupleRelationshipType(rel.Type) {
			continue
		}

		spouseID, isParticipant := spouseInRelationship(rel, personID)
		if !isParticipant {
			continue
		}

		ref := findMarriageEventID(personID, spouseID, rel, archive)
		found := ref != "" && evidenced[ref]

		rec := coverageRecord{
			Category: categoryVital,
			Label:    "Marriage record — " + spouseDisplayName(spouseID, archive),
			Found:    found,
			Priority: boolPriority(!found, severityMedium),
		}
		if found {
			rec.SourceRef = ref
		} else {
			rec.Description = unevidencedNote(ref)
		}
		records = append(records, rec)
	}

	return records
}

// spouseInRelationship returns the other participant in a couple relationship
// and whether personID is a participant in it at all.
func spouseInRelationship(rel *glxlib.Relationship, personID string) (string, bool) {
	var spouseID string
	isParticipant := false

	for _, p := range rel.Participants {
		if p.Person == personID {
			isParticipant = true
		} else {
			spouseID = p.Person
		}
	}

	return spouseID, isParticipant
}

// spouseDisplayName returns the spouse's display name, falling back to the ID
// when the archive has no person for it or the person carries no name.
func spouseDisplayName(spouseID string, archive *glxlib.GLXFile) string {
	if spouse, ok := archive.Persons[spouseID]; ok && spouse != nil {
		if name := glxlib.PersonDisplayName(spouse); name != "" {
			return name
		}
	}

	return spouseID
}

// findMarriageEventID returns the ID of the marriage event for a couple
// relationship: the relationship's start_event when it is one, otherwise a
// marriage event both spouses participate in. It returns "" when the archive
// records neither. The fallback walks event IDs in sorted order so a couple
// with more than one marriage event reports the same one on every run.
//
// It is distinct from findMarriageEvent in summary_runner.go, which answers a
// different question — the date and place rather than which event it was.
func findMarriageEventID(personID, spouseID string, rel *glxlib.Relationship, archive *glxlib.GLXFile) string {
	if rel.StartEvent != "" {
		if ev, ok := archive.Events[rel.StartEvent]; ok && ev != nil && ev.Type == glxlib.EventTypeMarriage {
			return rel.StartEvent
		}
	}

	for _, eventID := range sortedKeys(archive.Events) {
		ev := archive.Events[eventID]
		if ev == nil || ev.Type != glxlib.EventTypeMarriage {
			continue
		}
		if eventHasParticipants(ev, personID, spouseID) {
			return eventID
		}
	}

	return ""
}

// eventHasParticipants reports whether both people participate in the event.
func eventHasParticipants(event *glxlib.Event, personID, spouseID string) bool {
	hasPerson := false
	hasSpouse := false

	for _, ep := range event.Participants {
		if ep.Person == personID {
			hasPerson = true
		}
		if ep.Person == spouseID {
			hasSpouse = true
		}
	}

	return hasPerson && hasSpouse
}

// buildOtherRecords generates records for probate, land, military, church.
// When probateHighPriority is true (person died with known family), probate
// is elevated to HIGH priority because probate records name heirs.
func buildOtherRecords(sources, events []personSourceInfo, probateHighPriority bool) []coverageRecord {
	var records []coverageRecord

	// Probate/will
	probateRef := firstNonEmpty(
		findEvidencedEvent(events, glxlib.EventTypeProbate),
		findEvidencedEvent(events, glxlib.EventTypeWill),
	)
	probateFound := probateRef != "" || hasSourceType(sources, glxlib.SourceTypeProbate, "")
	rec := coverageRecord{
		Category: categoryOther,
		Label:    "Probate/will",
		Found:    probateFound,
	}
	if probateFound {
		rec.SourceRef = probateRef
		if rec.SourceRef == "" {
			rec.SourceRef = findSourceRef(sources, glxlib.SourceTypeProbate)
		}
	} else {
		if probateHighPriority {
			rec.Priority = severityHigh
			rec.Description = "often names heirs (children) and surviving spouse"
		}
		rec.Description = appendDescription(rec.Description, unevidencedNote(firstNonEmpty(
			findUnevidencedEvent(events, glxlib.EventTypeProbate),
			findUnevidencedEvent(events, glxlib.EventTypeWill),
		)))
	}
	records = append(records, rec)

	// Land records
	landFound := hasSourceType(sources, glxlib.SourceTypeLand, "")
	records = append(records, coverageRecord{
		Category:  categoryOther,
		Label:     "Land records",
		Found:     landFound,
		SourceRef: findSourceRef(sources, glxlib.SourceTypeLand),
	})

	// Military records
	militaryFound := hasSourceType(sources, glxlib.SourceTypeMilitary, "")
	records = append(records, coverageRecord{
		Category:  categoryOther,
		Label:     "Military records",
		Found:     militaryFound,
		SourceRef: findSourceRef(sources, glxlib.SourceTypeMilitary),
	})

	// Church records
	churchRef := firstNonEmpty(
		findEvidencedEvent(events, glxlib.EventTypeBaptism),
		findEvidencedEvent(events, glxlib.EventTypeChristening),
	)
	churchFound := churchRef != "" || hasSourceType(sources, glxlib.SourceTypeChurchRegister, "")
	churchRec := coverageRecord{
		Category: categoryOther,
		Label:    "Church records",
		Found:    churchFound,
	}
	if churchFound {
		churchRec.SourceRef = churchRef
		if churchRec.SourceRef == "" {
			churchRec.SourceRef = findSourceRef(sources, glxlib.SourceTypeChurchRegister)
		}
	} else {
		churchRec.Description = unevidencedNote(firstNonEmpty(
			findUnevidencedEvent(events, glxlib.EventTypeBaptism),
			findUnevidencedEvent(events, glxlib.EventTypeChristening),
		))
	}
	records = append(records, churchRec)

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

// findEvidencedEvent returns the ID of the first event of the given type that
// carries supporting evidence, or "" when the person has none.
func findEvidencedEvent(events []personSourceInfo, eventType string) string {
	return findEventRef(events, eventType, true)
}

// findUnevidencedEvent returns the ID of the first event of the given type
// recorded without any supporting evidence, or "" when the person has none.
// Coverage reports such an event so the conclusion is visible, but does not
// count it as a record found.
func findUnevidencedEvent(events []personSourceInfo, eventType string) string {
	return findEventRef(events, eventType, false)
}

func findEventRef(events []personSourceInfo, eventType string, evidenced bool) string {
	for _, e := range events {
		if e.EventType == eventType && e.Evidenced == evidenced {
			return e.Ref
		}
	}

	return ""
}

// unevidencedNote describes an event that exists but that nothing backs, so a
// reader can tell an absent record apart from one whose event is recorded as a
// conclusion only.
func unevidencedNote(ref string) string {
	if ref == "" {
		return ""
	}

	return ref + " is recorded, but no citation or source backs it"
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

// firstNonEmpty returns the first non-empty value, or "" when there is none.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
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
	if result.BirthDate != "" {
		born := "Born: " + result.BirthDate
		if result.BirthPlace != "" {
			born += ", " + result.BirthPlace
		}
		parts = append(parts, born)
	}
	if result.DeathDate != "" {
		died := "Died: " + result.DeathDate
		if result.DeathPlace != "" {
			died += ", " + result.DeathPlace
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
		{coverageCategoryCensus, "Census Records"},
		{categoryVital, "Vital Records"},
		{categoryOther, "Other Records"},
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

			if !r.Found && r.Priority == severityHigh {
				line += " -- HIGH PRIORITY"
			}

			if r.Description != "" {
				line += " -- " + r.Description
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
	if addition == "" {
		return existing
	}
	if existing == "" {
		return addition
	}

	return existing + "; " + addition
}

// appendCensusAnnotation adds a census year's research notes to a record
// description. The minor note applies only to someone who was still a child
// at that census.
func appendCensusAnnotation(desc string, note censusYearNote, age int) string {
	if note.note != "" {
		desc = appendDescription(desc, note.note)
	}
	if note.minorNote != "" && age < minorAgeUnder {
		desc = appendDescription(desc, note.minorNote)
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
		if glxlib.IsCoupleRelationshipType(rel.Type) {
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
