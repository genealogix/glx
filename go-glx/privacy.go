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
	"time"
)

// LivingThresholdYears is the default age threshold used by IsLivingPerson's
// heuristic: a person with no recorded end-of-life event whose most recent
// known birth year is less than this many years before `now` is assumed to be
// living for privacy-filter purposes.
const LivingThresholdYears = 100

// livingEndEventTypes are event types that establish a person is no longer
// living. A recorded event of any of these types — even with an empty date —
// is sufficient to classify the person as not living.
var livingEndEventTypes = map[string]bool{
	EventTypeDeath:     true,
	EventTypeBurial:    true,
	EventTypeCremation: true,
}

// PrivatizeResult reports what changed during a PrivatizeLiving pass. Zero
// values are valid: a no-op pass returns an empty result, not nil.
type PrivatizeResult struct {
	PersonsRedacted       int
	EventsRedacted        int
	EventsScrubbed        int
	RelationshipsRedacted int
	AssertionsDropped     int
}

// personLifeEvents pre-aggregates the information classifyLiving needs about a
// single person, so both the single-person path (personLifeEventsFor) and the
// bulk path (buildPersonLifeEventIndex) derive it in one O(events) sweep rather
// than calling FindPersonEvent once per person per event type.
//
// latestBirthYear holds the most recent (largest) parseable year across every
// birth event the person is a subject of. hasBirthYear distinguishes "no
// parseable birth year recorded" from a legitimately computed year of 0.
type personLifeEvents struct {
	hasEndEvent     bool
	hasBirthYear    bool
	latestBirthYear int
}

// IsLivingPerson reports whether the person identified by personID should be
// treated as living for privacy-filter purposes.
//
// Rules, in order:
//
//  1. If the person carries an explicit `living: true` or `living: false`
//     boolean property, that value is returned. Non-bool values (e.g. the
//     string "true") are validation errors handled elsewhere; this function
//     silently falls through to the heuristic rather than guessing.
//  2. Otherwise, if the person is the subject of any death, burial, or
//     cremation event, the person is not living — regardless of whether the
//     event's date is known.
//  3. Otherwise, if the person is the subject of any birth event with a
//     parseable year, the most recent (largest) such year is used: if it is
//     less than thresholdYears before now, the person is living. Taking the
//     most recent year is the privacy-conservative choice when birth events
//     conflict — it maximizes redaction, so a stray recent birth year cannot
//     be masked by an older or unparseable one and leak PII.
//  4. Otherwise the person is not living.
//
// Persons with no end-of-life event and no parseable birth year fall under
// rule 4 — explicit `living: true` is the escape hatch for those cases.
//
// A single call walks archive.Events once (O(events)); it does not call
// FindPersonEvent per event type.
func IsLivingPerson(archive *GLXFile, personID string, now time.Time, thresholdYears int) bool {
	if archive == nil {
		return false
	}
	person, ok := archive.Persons[personID]
	if !ok || person == nil {
		return false
	}

	return classifyLiving(person, personLifeEventsFor(archive, personID), now, thresholdYears)
}

// personLifeEventsFor walks archive.Events once and aggregates the life events
// for which personID is a subject (principal/subject/unset role): whether any
// end-of-life event names them and the most recent parseable birth year. It is
// the single-person analog of buildPersonLifeEventIndex and shares the same
// folding logic (accumulateLifeEvent), so both paths classify identically.
func personLifeEventsFor(archive *GLXFile, personID string) personLifeEvents {
	var life personLifeEvents
	for _, event := range archive.Events {
		if event == nil || !isLifeEvent(event.Type) {
			continue
		}
		for _, p := range event.Participants {
			if p.Person != personID || !isSubjectRole(p.Role) {
				continue
			}
			accumulateLifeEvent(&life, event)

			break
		}
	}

	return life
}

// classifyLiving applies the IsLivingPerson decision rule with already-
// aggregated life events. Sharing this with both the single-person and bulk
// index paths keeps the rule single-sourced.
func classifyLiving(person *Person, lifeEvents personLifeEvents, now time.Time, thresholdYears int) bool {
	if explicit, ok := person.Properties[PersonPropertyLiving].(bool); ok {
		return explicit
	}
	if lifeEvents.hasEndEvent {
		return false
	}
	if !lifeEvents.hasBirthYear {
		return false
	}

	return now.Year()-lifeEvents.latestBirthYear < thresholdYears
}

// isLifeEvent reports whether an event type feeds the living heuristic: a birth
// (for the date threshold) or an end-of-life event (death/burial/cremation).
func isLifeEvent(eventType string) bool {
	return eventType == EventTypeBirth || livingEndEventTypes[eventType]
}

// accumulateLifeEvent folds one event for which a person is a subject into that
// person's aggregate. An end-of-life event sets hasEndEvent; a birth event with
// a parseable year advances latestBirthYear to the most recent (largest) year
// seen so far. Both operations are order-independent (boolean OR and max), so
// callers need not visit events in any particular order.
func accumulateLifeEvent(life *personLifeEvents, event *Event) {
	if livingEndEventTypes[event.Type] {
		life.hasEndEvent = true

		return
	}
	if event.Type != EventTypeBirth {
		return
	}
	year := ExtractFirstYear(string(event.Date))
	if year == 0 {
		return
	}
	if !life.hasBirthYear || year > life.latestBirthYear {
		life.hasBirthYear = true
		life.latestBirthYear = year
	}
}

// buildPersonLifeEventIndex makes a single O(events) pass over archive.Events
// and returns, per person, the aggregated life events (end-of-life presence and
// most recent parseable birth year) across every event the person is a subject
// of. No sort is needed: accumulateLifeEvent is order-independent (end-event is
// a boolean OR, birth year is a max), so the bulk path produces the same
// classification as per-person IsLivingPerson calls.
func buildPersonLifeEventIndex(archive *GLXFile) map[string]personLifeEvents {
	index := make(map[string]personLifeEvents)
	for _, event := range archive.Events {
		if event == nil || !isLifeEvent(event.Type) {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" || !isSubjectRole(p.Role) {
				continue
			}
			entry := index[p.Person]
			accumulateLifeEvent(&entry, event)
			index[p.Person] = entry
		}
	}

	return index
}

// PrivatizeLiving redacts every living person in archive in place and reports
// what changed. A person is considered living per IsLivingPerson.
//
// Redaction:
//
//   - Replaces the person's Properties with exactly
//     {"name": {"value": "Living"}, "living": true} and clears their Notes.
//   - Blanks Date / PlaceID / Notes / Properties on every event whose subject
//     is a living person (Type and Participants are preserved so the GEDCOM
//     export still emits the event tag and family relationships still
//     reconstruct).
//   - For every Relationship that names any living participant: clears
//     rel.Notes and rel.Properties, scrubs Properties and Notes on the
//     participant entries naming a living person, and fully redacts the
//     relationship's StartEvent and EndEvent. This covers marriages between
//     living spouses, whose participants carry the non-subject role "spouse"
//     and would otherwise leak DATE / PLAC / NOTE through the FAM record.
//   - On every other event that names a living person as a non-subject
//     participant, scrubs Properties and Notes on those participant entries
//     so per-participant fields such as name_as_recorded do not leak.
//   - Removes every assertion whose Subject.Person or Participant.Person is a
//     living person.
func PrivatizeLiving(archive *GLXFile, now time.Time, thresholdYears int) *PrivatizeResult {
	result := &PrivatizeResult{}
	if archive == nil {
		return result
	}

	living := identifyLivingPersons(archive, now, thresholdYears)
	if len(living) == 0 {
		return result
	}

	result.PersonsRedacted = redactLivingPersonRecords(archive, living)

	fullyRedactedEvents := eventsWithLivingSubject(archive, living)
	result.RelationshipsRedacted = redactRelationshipsWithLivingParticipants(archive, living, fullyRedactedEvents)
	result.EventsRedacted, result.EventsScrubbed = redactAndScrubEvents(archive, living, fullyRedactedEvents)
	result.AssertionsDropped = dropAssertionsReferencingLivingPersons(archive, living)

	return result
}

// identifyLivingPersons returns the set of person IDs in archive that should
// be treated as living, applying IsLivingPerson's rules across all persons in
// one O(events) pass via buildPersonLifeEventIndex.
func identifyLivingPersons(archive *GLXFile, now time.Time, thresholdYears int) map[string]bool {
	lifeEventIndex := buildPersonLifeEventIndex(archive)
	living := make(map[string]bool, len(archive.Persons))
	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		if classifyLiving(person, lifeEventIndex[id], now, thresholdYears) {
			living[id] = true
		}
	}

	return living
}

// redactLivingPersonRecords replaces each living person's Properties and Notes
// with the canonical redacted form and returns the count of persons redacted.
func redactLivingPersonRecords(archive *GLXFile, living map[string]bool) int {
	count := 0
	for id := range living {
		person := archive.Persons[id]
		if person == nil {
			continue
		}
		person.Properties = map[string]any{
			PersonPropertyName: map[string]any{
				"value": "Living",
			},
			PersonPropertyLiving: true,
		}
		person.Notes = nil
		count++
	}

	return count
}

// eventsWithLivingSubject builds the initial set of events that must be fully
// redacted because a living person is the event's subject. The set is grown by
// redactRelationshipsWithLivingParticipants to also include events referenced
// as StartEvent / EndEvent of a relationship between living participants.
func eventsWithLivingSubject(archive *GLXFile, living map[string]bool) map[string]bool {
	fullyRedactedEvents := make(map[string]bool)
	for id, event := range archive.Events {
		if event != nil && eventHasLivingSubject(event, living) {
			fullyRedactedEvents[id] = true
		}
	}

	return fullyRedactedEvents
}

// redactRelationshipsWithLivingParticipants clears Notes / Properties on
// relationships that name any living participant, scrubs the living
// participants' per-entry Properties / Notes, and marks the relationship's
// StartEvent and EndEvent for full event redaction.
func redactRelationshipsWithLivingParticipants(archive *GLXFile, living, fullyRedactedEvents map[string]bool) int {
	count := 0
	for _, rel := range archive.Relationships {
		if rel == nil || !relationshipHasLivingParticipant(rel, living) {
			continue
		}
		rel.Notes = nil
		rel.Properties = nil
		scrubLivingParticipants(rel.Participants, living)
		if rel.StartEvent != "" {
			fullyRedactedEvents[rel.StartEvent] = true
		}
		if rel.EndEvent != "" {
			fullyRedactedEvents[rel.EndEvent] = true
		}
		count++
	}

	return count
}

// redactAndScrubEvents walks every event, fully blanking events flagged in
// fullyRedactedEvents and scrubbing per-participant fields on remaining events
// that name a living person in a non-subject role. Returns (fully-redacted
// count, scrubbed-only count).
func redactAndScrubEvents(archive *GLXFile, living, fullyRedactedEvents map[string]bool) (int, int) {
	redacted, scrubbed := 0, 0
	for id, event := range archive.Events {
		if event == nil {
			continue
		}
		if fullyRedactedEvents[id] {
			event.Date = ""
			event.PlaceID = ""
			event.Notes = nil
			event.Properties = nil
			scrubLivingParticipants(event.Participants, living)
			redacted++

			continue
		}
		if participantsIncludeLivingPerson(event.Participants, living) {
			scrubLivingParticipants(event.Participants, living)
			scrubbed++
		}
	}

	return redacted, scrubbed
}

// dropAssertionsReferencingLivingPersons removes every assertion whose
// Subject.Person or Participant.Person is a living person and returns the
// count dropped.
func dropAssertionsReferencingLivingPersons(archive *GLXFile, living map[string]bool) int {
	if len(archive.Assertions) == 0 {
		return 0
	}
	dropped := 0
	retained := make(map[string]*Assertion, len(archive.Assertions))
	for id, assertion := range archive.Assertions {
		if assertion != nil && assertionReferencesLivingPerson(assertion, living) {
			dropped++

			continue
		}
		retained[id] = assertion
	}
	archive.Assertions = retained

	return dropped
}

// relationshipHasLivingParticipant reports whether any participant of the
// relationship — in any role — refers to a living person.
func relationshipHasLivingParticipant(rel *Relationship, living map[string]bool) bool {
	for _, p := range rel.Participants {
		if p.Person != "" && living[p.Person] {
			return true
		}
	}

	return false
}

// participantsIncludeLivingPerson reports whether any participant in the slice
// refers to a living person, in any role.
func participantsIncludeLivingPerson(participants []Participant, living map[string]bool) bool {
	for _, p := range participants {
		if p.Person != "" && living[p.Person] {
			return true
		}
	}

	return false
}

// scrubLivingParticipants clears Properties and Notes on each participant
// entry whose Person is in the living set, preserving Person and Role so
// downstream family-graph reconstruction still works. The slice is mutated
// in place; iteration is by index because Participant is a value type.
func scrubLivingParticipants(participants []Participant, living map[string]bool) {
	for i := range participants {
		if living[participants[i].Person] {
			participants[i].Properties = nil
			participants[i].Notes = nil
		}
	}
}

// eventHasLivingSubject reports whether any participant of the event acts in a
// subject role (principal/subject/unset) and is in the living set.
func eventHasLivingSubject(event *Event, living map[string]bool) bool {
	for _, p := range event.Participants {
		if p.Person == "" {
			continue
		}
		if !isSubjectRole(p.Role) {
			continue
		}
		if living[p.Person] {
			return true
		}
	}

	return false
}

// assertionReferencesLivingPerson reports whether the assertion is about a
// living person — either by EntityRef subject or by the optional participant
// payload — and should therefore be dropped from a privatized archive.
func assertionReferencesLivingPerson(assertion *Assertion, living map[string]bool) bool {
	if assertion.Subject.Person != "" && living[assertion.Subject.Person] {
		return true
	}
	if assertion.Participant != nil && living[assertion.Participant.Person] {
		return true
	}

	return false
}
