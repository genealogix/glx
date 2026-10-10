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

package glx

import (
	"slices"
	"strconv"
	"strings"
)

// maxLifespan is the assumed maximum lifespan for capping census suggestions
// when no death date is known.
const maxLifespan = 100

// Coverage checklist categories, as they appear in the JSON output and as the
// headings printCoverageText groups by. coverageCategoryCensus shares the
// literal value of EventTypeCensus.
const (
	coverageCategoryCensus = "census"
	coverageCategoryVital  = "vital"
	coverageCategoryOther  = "other"
)

// CoverageRecord represents one expected record in the coverage checklist.
type CoverageRecord struct {
	Category    string `json:"category"`
	Label       string `json:"label"`
	Found       bool   `json:"found"`
	SourceRef   string `json:"source_ref,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Description string `json:"description,omitempty"`
}

// CoverageResult holds the full coverage output for a person.
type CoverageResult struct {
	PersonID   string           `json:"person_id"`
	PersonName string           `json:"person_name"`
	BirthDate  string           `json:"birth_date,omitempty"`
	BirthPlace string           `json:"birth_place,omitempty"`
	DeathDate  string           `json:"death_date,omitempty"`
	DeathPlace string           `json:"death_place,omitempty"`
	Records    []CoverageRecord `json:"records"`
	Found      int              `json:"found"`
	Expected   int              `json:"expected"`
	// AppearsIn lists other people's records the person participates in.
	AppearsIn []CoverageAppearance `json:"appears_in,omitempty"`
}

// CoverageAppearance is one event in which the person appears in someone
// else's record.
type CoverageAppearance struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type,omitempty"`
	Role      string `json:"role,omitempty"`
	Date      string `json:"date,omitempty"`
	Label     string `json:"label"` // whose record it is, e.g. "Probate of Caspar Stoehr"
}

// buildCoverage generates the coverage checklist for a person.
func buildCoverage(personID string, person *Person, archive *GLXFile, fallback string) *CoverageResult {
	var birthDate, birthPlace, deathDate, deathPlace string
	if birthEvent := findCoveragePersonEvent(archive, personID, EventTypeBirth); birthEvent != nil {
		birthDate = string(birthEvent.Date)
		birthPlace = birthEvent.PlaceID
	}
	if deathEvent := findCoveragePersonEvent(archive, personID, EventTypeDeath); deathEvent != nil {
		deathDate = string(deathEvent.Date)
		deathPlace = deathEvent.PlaceID
	}

	birthYear := ExtractFirstYear(birthDate)
	deathYear := deathYearUpperBound(deathDate)

	// Build indexes: what sources/citations/events reference this person,
	// which events an assertion firmly backs with a citation, source or media,
	// and which are backed only by low-confidence or speculative assertions
	personSources := collectPersonSources(personID, archive)
	evidencedEvents := eventsWithEvidence(archive)
	estimatedEvents := eventsWithOnlyWeakEvidence(archive, evidencedEvents)
	participations := collectPersonEvents(personID, archive, evidencedEvents, estimatedEvents)
	personEvents := filterPersonEvents(participations, func(e personSourceInfo) bool { return e.OwnRecord })
	presentEvents := filterPersonEvents(participations, func(e personSourceInfo) bool { return e.Present })

	// Infer death year from burial event if death_date is not set
	if deathYear == 0 {
		deathYear = inferDeathYearFromEvents(personEvents)
	}

	// National census records, for the countries the person's places name
	schedules := censusSchedulesForDatedPlaces(coveragePlaceRefs(presentEvents), archive, fallback)
	records := buildCensusRecords(birthYear, deathYear, schedules, personSources, personEvents, presentEvents, archive)

	// State census records
	states := collectPersonStates(archive, presentEvents)
	records = append(records, buildStateCensusRecords(birthYear, deathYear, states, personSources, personEvents, archive)...)

	// Vital records
	records = append(records, buildVitalRecords(personID, archive, personSources, personEvents, evidencedEvents, estimatedEvents)...)

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

	return &CoverageResult{
		PersonID:   personID,
		PersonName: PersonDisplayName(person),
		BirthDate:  birthDate,
		BirthPlace: birthPlaceName,
		DeathDate:  deathDate,
		DeathPlace: deathPlaceName,
		Records:    records,
		Found:      found,
		Expected:   len(records),
		AppearsIn:  buildAppearances(participations, archive),
	}
}

// personSourceInfo tracks a source or citation found for a person.
type personSourceInfo struct {
	Ref        string // source or citation ID
	Type       string // source type
	Title      string
	EventType  string     // if found via an event
	PersonRole string     // this person's role in that event
	PlaceID    string     // place reference (events only)
	Date       DateString // event date (events only); resolves temporal place parents
	Year       int
	Evidenced  bool // events only: a firm assertion resolves a citation, source or media
	Estimated  bool // events only: only hypothetical assertions resolve any evidence
	OwnRecord  bool // events only: the event is the person's own record
	Present    bool // events only: the role implies presence at the event's place
}

// collectPersonSources gathers all sources and citations that reference a person
// via assertions: those about the person, and those about an event that is the
// person's own record (AssertionPersons). A church register cited for a burial
// date is the decedent's church record, not only the burial's (#1211).
func collectPersonSources(personID string, archive *GLXFile) []personSourceInfo {
	var sources []personSourceInfo
	seen := make(map[string]bool)

	for _, assertionID := range sortedKeys(archive.Assertions) {
		assertion := archive.Assertions[assertionID]
		if !slices.Contains(AssertionPersons(assertion, archive), personID) {
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
				info.Year = ExtractFirstYear(string(src.Date))
			}
			sources = append(sources, info)
		}
		for _, srcID := range sourcesWithMedia(assertion, archive) {
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
				Year:  ExtractFirstYear(string(src.Date)),
			})
		}
	}

	return sources
}

// sourcesWithMedia returns a detached list of directly linked sources and the
// source IDs reached through media. Missing media/source references are ignored
// when constructing personSourceInfo, just like directly linked missing sources.
func sourcesWithMedia(assertion *Assertion, archive *GLXFile) []string {
	ids := slices.Clone(assertion.Sources)
	for _, id := range assertion.Media {
		if media := archive.Media[id]; media != nil && media.Source != "" {
			ids = append(ids, media.Source)
		}
	}

	return ids
}

// collectPersonEvents gathers all events this person participates in, marking
// each with whether it carries supporting evidence per the evidenced index (or
// only the weak evidence of the estimated index) and with what the person's
// role makes of it (ClassifyParticipation).
func collectPersonEvents(personID string, archive *GLXFile, evidenced, estimated map[string]bool) []personSourceInfo {
	var events []personSourceInfo

	// Sorted so a person with more than one event of a type reports the same
	// one on every run
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		participation, role, ok := EventParticipation(event, personID, archive.ParticipantRoles)
		if !ok {
			continue
		}
		events = append(events, personSourceInfo{
			Ref:        eventID,
			EventType:  event.Type,
			Year:       ExtractFirstYear(string(event.Date)),
			Title:      event.Title,
			PlaceID:    event.PlaceID,
			Date:       event.Date,
			Evidenced:  evidenced[eventID],
			Estimated:  estimated[eventID],
			PersonRole: role,
			OwnRecord:  participation.OwnRecord,
			Present:    participation.Present,
		})
	}

	return events
}

// filterPersonEvents returns the events that satisfy keep.
func filterPersonEvents(events []personSourceInfo, keep func(personSourceInfo) bool) []personSourceInfo {
	var kept []personSourceInfo
	for _, e := range events {
		if keep(e) {
			kept = append(kept, e)
		}
	}

	return kept
}

// buildAppearances lists the events in which the person takes part without
// the event being their own record, labeled by whose record it is.
func buildAppearances(events []personSourceInfo, archive *GLXFile) []CoverageAppearance {
	var appearances []CoverageAppearance
	for _, e := range events {
		if e.OwnRecord {
			continue
		}
		event := archive.Events[e.Ref]
		appearances = append(appearances, CoverageAppearance{
			EventID:   e.Ref,
			EventType: e.EventType,
			Role:      e.PersonRole,
			Date:      string(event.Date),
			Label:     appearanceLabel(event, archive),
		})
	}

	return appearances
}

// appearanceLabel names whose record an event is: "Probate of Caspar
// Stoehr" from the participants whose own record it is, else the event's
// title, else its type.
func appearanceLabel(event *Event, archive *GLXFile) string {
	typeLabel := coverageEventTypeLabel(event.Type)

	var names []string
	for _, p := range event.Participants {
		if p.Person == "" || !ClassifyParticipation(event.Type, p.Role, archive.ParticipantRoles).OwnRecord {
			continue
		}
		name := p.Person
		if person := archive.Persons[p.Person]; person != nil {
			if display := PersonDisplayName(person); display != "" {
				name = display
			}
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}

	switch {
	case len(names) > 0:
		return typeLabel + " of " + strings.Join(names, " and ")
	case strings.TrimSpace(event.Title) != "":
		return strings.TrimSpace(event.Title)
	default:
		return typeLabel
	}
}

// coverageDatedPlaces returns the dated places of a person's events, for
// working out which state or territory they were in at a census (#1333).
func coverageDatedPlaces(events []personSourceInfo) []CensusDatedPlace {
	var places []CensusDatedPlace
	for _, e := range events {
		if e.PlaceID != "" && e.Year != 0 {
			places = append(places, CensusDatedPlace{Year: e.Year, PlaceID: e.PlaceID})
		}
	}

	return places
}

// coveragePlaceRefs returns the place references of a person's events, for
// resolving which countries' census schedules apply to them.
func coveragePlaceRefs(events []personSourceInfo) []datedPlaceRef {
	var refs []datedPlaceRef
	for _, e := range events {
		if e.PlaceID != "" {
			refs = append(refs, datedPlaceRef{placeID: e.PlaceID, date: e.Date})
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

// eventsWithEvidence returns the set of event IDs backed by evidence: those
// that are the subject of an assertion resolving at least one citation or
// source or media object, and that the assertion does not itself mark as a
// hypothesis (AssertionIsHypothetical) or as disproven.
//
// An event carries no citations or sources of its own — under the GLX evidence
// model the event is a conclusion, and what supports it is the assertion
// pointing at it. An event with no such assertion is therefore an unsupported
// claim (an estimate reckoned from a relative's record, a placeholder, a
// GEDCOM import), which coverage must not count as a record found. So is an
// event whose only support is a low-confidence or speculative assertion: an
// estimated birth reckoned from a marriage entry cites that marriage, not a
// birth record (#1369).
func eventsWithEvidence(archive *GLXFile) map[string]bool {
	evidenced := make(map[string]bool)

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		eventID := assertion.Subject.Event
		if eventID == "" || evidenced[eventID] {
			continue
		}
		if assertionIsFirm(assertion) && assertionHasEvidence(assertion, archive) {
			evidenced[eventID] = true
		}
	}

	return evidenced
}

// eventsWithOnlyWeakEvidence returns the events, outside evidenced, that a
// hypothetical (low-confidence, speculative, unresearched or disputed)
// assertion backs with a citation, source or media object. Coverage names them
// as estimates rather than counting them as records found.
func eventsWithOnlyWeakEvidence(archive *GLXFile, evidenced map[string]bool) map[string]bool {
	estimated := make(map[string]bool)

	for _, assertion := range archive.Assertions {
		if assertion == nil {
			continue
		}
		eventID := assertion.Subject.Event
		if eventID == "" || evidenced[eventID] || estimated[eventID] {
			continue
		}
		if AssertionIsHypothetical(assertion) && assertionHasEvidence(assertion, archive) {
			estimated[eventID] = true
		}
	}

	return estimated
}

// assertionIsFirm reports whether an assertion states a conclusion: it is
// neither disproven nor recorded as a hypothesis.
func assertionIsFirm(assertion *Assertion) bool {
	return !AssertionIsDisproven(assertion) && !AssertionIsHypothetical(assertion)
}

// assertionHasEvidence reports whether an assertion resolves at least one
// citation, source, or media object. A dangling reference does not count: reference
// validation already reports it as an error, and honoring it here would let a
// typo stand in for a record.
func assertionHasEvidence(assertion *Assertion, archive *GLXFile) bool {
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
	for _, mediaID := range assertion.Media {
		if archive.Media[mediaID] != nil {
			return true
		}
	}

	return false
}

// buildCensusRecords generates expected census records for every supplied
// schedule, bounded by the person's birth and death years. A person with no
// applicable schedule — one whose places name a country GLX has no census
// schedule for — gets no census rows at all (#186). A census lost in every
// bracketing jurisdiction is omitted unless found, so it does not count as
// missing (#1333). A nil archive skips the jurisdiction survival check.
func buildCensusRecords(birthYear, deathYear int, schedules []*CensusSchedule, sources, events, presentEvents []personSourceInfo, archive *GLXFile) []CoverageRecord {
	if birthYear == 0 {
		return nil
	}

	// Cap at max lifespan when no death year is known
	upperBound := deathYear
	if upperBound == 0 {
		upperBound = birthYear + maxLifespan
	}

	var records []CoverageRecord
	places := coverageDatedPlaces(presentEvents)

	for _, schedule := range schedules {
		for _, year := range schedule.Years {
			if year < birthYear {
				continue
			}
			if year > upperBound {
				break
			}
			survival := schedule.Survival(year, places, archive)
			// Approximate age at this census year (may be 0 if census year == birth year)
			age := year - birthYear
			note := schedule.Notes[year]

			rec := CoverageRecord{
				Category: coverageCategoryCensus,
				Label:    schedule.CoverageLabel(year, age),
			}

			// Check if we have this census
			ref := findCensusMatch(year, sources, events)
			if ref != "" {
				rec.Found = true
				rec.SourceRef = ref
			}

			if survival.Lost && !rec.Found {
				continue
			}

			// Census-specific annotations (always added, even when found)
			rec.Description = appendCensusAnnotation(rec.Description, note, age)
			rec.Description = appendDescription(rec.Description, survival.Note)

			// An unevidenced census event is named, not counted
			if !rec.Found {
				rec.Description = appendDescription(rec.Description, unevidencedEventNote(events, findUnevidencedCensus(year, events)))
			}

			setCensusPriority(&rec, note, age)

			records = append(records, rec)
		}
	}

	return records
}

// setCensusPriority annotates a missing census record.
func setCensusPriority(rec *CoverageRecord, note CensusYearNote, age int) {
	if rec.Found {
		return
	}
	switch {
	case note.HighPriority:
		rec.Priority = severityHigh
	case age >= censusPrimeAgeMin && age <= censusPrimeAgeMax:
		rec.Priority = severityHigh
		if note.MinorNote == "" || age >= minorAgeUnder {
			rec.Description = appendDescription(rec.Description, "may show in parents' household")
		}
	}
}

// findCensusMatch checks if a census for a given year exists in sources or events.
func findCensusMatch(year int, sources, events []personSourceInfo) string {
	for _, e := range events {
		if e.EventType == EventTypeCensus && e.Year == year && e.Evidenced {
			return e.Ref
		}
	}
	for _, s := range sources {
		if s.Type == SourceTypeCensus && s.Year == year {
			return s.Ref
		}
		// Also check title for census year mentions
		if s.Type == SourceTypeCensus && strings.Contains(s.Title, strconv.Itoa(year)) {
			return s.Ref
		}
	}

	return ""
}

// findUnevidencedCensus returns the ID of a census event for the given year
// that nothing backs, or "" when the person has none.
func findUnevidencedCensus(year int, events []personSourceInfo) string {
	for _, e := range events {
		if e.EventType == EventTypeCensus && e.Year == year && !e.Evidenced {
			return e.Ref
		}
	}

	return ""
}

// buildVitalRecords generates expected vital records.
func buildVitalRecords(personID string, archive *GLXFile, sources, events []personSourceInfo, evidenced, estimated map[string]bool) []CoverageRecord {
	marriageRecords := buildMarriageRecords(personID, archive, evidenced, estimated)
	records := make([]CoverageRecord, 0, 2+len(marriageRecords))

	records = append(records,
		buildVitalRecord("Birth record", severityHigh, EventTypeBirth, "birth", sources, events),
		buildVitalRecord("Death record", severityMedium, EventTypeDeath, "death", sources, events),
	)

	// Marriage records — check relationships for spouse
	records = append(records, marriageRecords...)

	return records
}

// buildVitalRecord builds one birth-or-death checklist entry. The record counts
// as found on an event only when that event carries evidence; an event standing
// on its own is a conclusion, and counting it would inflate the score for
// exactly the people whose records are missing.
func buildVitalRecord(label, missingPriority, eventType, titleKeyword string, sources, events []personSourceInfo) CoverageRecord {
	eventRef := findEvidencedEvent(events, eventType)
	sourceRef := findMatchingSourceRef(sources, SourceTypeVitalRecord, titleKeyword)
	found := eventRef != "" || sourceRef != ""

	rec := CoverageRecord{
		Category: coverageCategoryVital,
		Label:    label,
		Found:    found,
		Priority: boolPriority(!found, missingPriority),
	}
	if found {
		rec.SourceRef = eventRef
		if rec.SourceRef == "" {
			rec.SourceRef = sourceRef
		}
	} else {
		rec.Description = unevidencedEventNote(events, findUnevidencedEvent(events, eventType))
	}

	return rec
}

// buildMarriageRecords checks for marriage events linked to spouse
// relationships. As with birth and death, the marriage event has to carry
// evidence before it counts as the marriage record.
func buildMarriageRecords(personID string, archive *GLXFile, evidenced, estimated map[string]bool) []CoverageRecord {
	var records []CoverageRecord

	// Find spouse relationships
	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel == nil {
			continue
		}
		if !IsCoupleRelationshipType(rel.Type) {
			continue
		}

		spouseID, isParticipant := spouseInRelationship(rel, personID)
		if !isParticipant {
			continue
		}

		ref := findMarriageEventID(personID, spouseID, rel, archive)
		found := ref != "" && evidenced[ref]

		rec := CoverageRecord{
			Category: coverageCategoryVital,
			Label:    "Marriage record — " + spouseDisplayName(spouseID, archive),
			Found:    found,
			Priority: boolPriority(!found, severityMedium),
		}
		if found {
			rec.SourceRef = ref
		} else {
			rec.Description = unevidencedNote(ref, estimated[ref])
		}
		records = append(records, rec)
	}

	return records
}

// spouseInRelationship returns the other participant in a couple relationship
// and whether personID is a participant in it at all.
func spouseInRelationship(rel *Relationship, personID string) (string, bool) {
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
func spouseDisplayName(spouseID string, archive *GLXFile) string {
	if spouse, ok := archive.Persons[spouseID]; ok && spouse != nil {
		if name := PersonDisplayName(spouse); name != "" {
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
func findMarriageEventID(personID, spouseID string, rel *Relationship, archive *GLXFile) string {
	if rel.StartEvent != "" {
		if ev, ok := archive.Events[rel.StartEvent]; ok && ev != nil && ev.Type == EventTypeMarriage {
			return rel.StartEvent
		}
	}

	for _, eventID := range sortedKeys(archive.Events) {
		ev := archive.Events[eventID]
		if ev == nil || ev.Type != EventTypeMarriage {
			continue
		}
		personParticipation, _, hasPerson := EventParticipation(ev, personID, archive.ParticipantRoles)
		spouseParticipation, _, hasSpouse := EventParticipation(ev, spouseID, archive.ParticipantRoles)
		if hasPerson && hasSpouse && personParticipation.OwnRecord && spouseParticipation.OwnRecord {
			return eventID
		}
	}

	return ""
}

// buildOtherRecords generates records for probate, land, military, church.
// When probateHighPriority is true (person died with known family), probate
// is elevated to HIGH priority because probate records name heirs.
func buildOtherRecords(sources, events []personSourceInfo, probateHighPriority bool) []CoverageRecord {
	const otherRecordCount = 4
	records := make([]CoverageRecord, 0, otherRecordCount)

	// Probate/will
	probateRef := firstNonEmpty(
		findEvidencedEvent(events, EventTypeProbate),
		findEvidencedEvent(events, EventTypeWill),
	)
	probateFound := probateRef != "" || hasSourceType(sources, SourceTypeProbate, "")
	rec := CoverageRecord{
		Category: coverageCategoryOther,
		Label:    "Probate/will",
		Found:    probateFound,
	}
	if probateFound {
		rec.SourceRef = probateRef
		if rec.SourceRef == "" {
			rec.SourceRef = findSourceRef(sources, SourceTypeProbate)
		}
	} else {
		if probateHighPriority {
			rec.Priority = severityHigh
			rec.Description = "often names heirs (children) and surviving spouse"
		}
		rec.Description = appendDescription(rec.Description, unevidencedEventNote(events, firstNonEmpty(
			findUnevidencedEvent(events, EventTypeProbate),
			findUnevidencedEvent(events, EventTypeWill),
		)))
	}
	records = append(records, rec)

	// Land records
	landFound := hasSourceType(sources, SourceTypeLand, "")
	records = append(records, CoverageRecord{
		Category:  coverageCategoryOther,
		Label:     "Land records",
		Found:     landFound,
		SourceRef: findSourceRef(sources, SourceTypeLand),
	})

	// Military records
	militaryFound := hasSourceType(sources, SourceTypeMilitary, "")
	records = append(records, CoverageRecord{
		Category:  coverageCategoryOther,
		Label:     "Military records",
		Found:     militaryFound,
		SourceRef: findSourceRef(sources, SourceTypeMilitary),
	})

	// Church records
	churchRef := firstNonEmpty(
		findEvidencedEvent(events, EventTypeBaptism),
		findEvidencedEvent(events, EventTypeChristening),
	)
	churchFound := churchRef != "" || hasSourceType(sources, SourceTypeChurchRegister, "")
	churchRec := CoverageRecord{
		Category: coverageCategoryOther,
		Label:    "Church records",
		Found:    churchFound,
	}
	if churchFound {
		churchRec.SourceRef = churchRef
		if churchRec.SourceRef == "" {
			churchRec.SourceRef = findSourceRef(sources, SourceTypeChurchRegister)
		}
	} else {
		churchRec.Description = unevidencedEventNote(events, firstNonEmpty(
			findUnevidencedEvent(events, EventTypeBaptism),
			findUnevidencedEvent(events, EventTypeChristening),
		))
	}
	records = append(records, churchRec)

	return records
}

// coverageResolvePlaceName returns the place name for a place ID, or the raw string.
func coverageResolvePlaceName(placeRef string, archive *GLXFile) string {
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

// unevidencedNote describes an event that exists but that nothing firmly backs,
// so a reader can tell an absent record apart from one whose event is recorded
// as a conclusion only, or as an estimate whose only evidence is a
// low-confidence or speculative assertion.
func unevidencedNote(ref string, estimated bool) string {
	if ref == "" {
		return ""
	}
	if estimated {
		return ref + " is recorded only as an estimate: its evidence is low-confidence or speculative"
	}

	return ref + " is recorded, but no citation, source or media backs it"
}

// unevidencedEventNote is unevidencedNote for one of the person's events,
// looked up by ID in events.
func unevidencedEventNote(events []personSourceInfo, ref string) string {
	for _, e := range events {
		if e.Ref == ref {
			return unevidencedNote(ref, e.Estimated)
		}
	}

	return unevidencedNote(ref, false)
}

func hasSourceType(sources []personSourceInfo, sourceType, titleKeyword string) bool {
	_, found := matchingSource(sources, sourceType, titleKeyword)

	return found
}

func findMatchingSourceRef(sources []personSourceInfo, sourceType, titleKeyword string) string {
	source, _ := matchingSource(sources, sourceType, titleKeyword)

	return source.Ref
}

func matchingSource(sources []personSourceInfo, sourceType, titleKeyword string) (personSourceInfo, bool) {
	for _, source := range sources {
		if source.Type == sourceType && (titleKeyword == "" || strings.Contains(strings.ToLower(source.Title), titleKeyword)) {
			return source, true
		}
	}

	return personSourceInfo{}, false
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
	return findMatchingSourceRef(sources, sourceType, "")
}

func boolPriority(condition bool, priority string) string {
	if condition {
		return priority
	}

	return ""
}

// inferDeathYearFromEvents returns a death year inferred from burial events.
// When multiple burial events exist, returns the earliest year.
// Returns 0 if no burial event with a date is found.
func inferDeathYearFromEvents(events []personSourceInfo) int {
	earliest := 0
	for _, e := range events {
		if e.EventType == EventTypeBurial && e.Year > 0 {
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
func appendCensusAnnotation(desc string, note CensusYearNote, age int) string {
	if note.Note != "" {
		desc = appendDescription(desc, note.Note)
	}
	if note.MinorNote != "" && age < minorAgeUnder {
		desc = appendDescription(desc, note.MinorNote)
	}

	return desc
}

// hasFamily returns true if the person has any spouse or child relationships.
func hasFamily(personID string, archive *GLXFile) bool {
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
		if IsCoupleRelationshipType(rel.Type) {
			for _, p := range rel.Participants {
				if p.Person == personID && p.Role == ParticipantRoleSpouse {
					return true
				}
			}
		}

		// Check for parent-child where this person is the parent
		if researchIsParentChildType(rel.Type) {
			for _, p := range rel.Participants {
				if p.Person == personID && p.Role == ParticipantRoleParent {
					return true
				}
			}
		}
	}

	return false
}

// findCoveragePersonEvent applies own-record semantics to the coverage lifetime.
func findCoveragePersonEvent(archive *GLXFile, personID, eventType string) *Event {
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || event.Type != eventType {
			continue
		}
		participation, _, ok := EventParticipation(event, personID, archive.ParticipantRoles)
		if ok && participation.OwnRecord {
			return event
		}
	}

	return nil
}

func coverageEventTypeLabel(eventType string) string {
	if eventType == "" {
		return "Event"
	}
	label := strings.ReplaceAll(eventType, "_", " ")

	return strings.ToUpper(label[:1]) + label[1:]
}
