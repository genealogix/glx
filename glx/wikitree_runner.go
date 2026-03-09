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
	"os"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// wikiTreeEvent holds an event with its ID for sorting.
type wikiTreeEvent struct {
	ID    string
	Event *glxlib.Event
}

// showWikiTree generates a WikiTree biography for a person.
func showWikiTree(archivePath, personQuery string) error {
	archive, err := loadArchiveForWikiTree(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPersonForWikiTree(archive, personQuery)
	if err != nil {
		return err
	}

	bio := generateWikiTreeBio(personID, person, archive)
	fmt.Print(bio)

	return nil
}

// loadArchiveForWikiTree loads an archive from a path (directory or single file).
func loadArchiveForWikiTree(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}

		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// findPersonForWikiTree looks up a person by exact ID or name substring.
func findPersonForWikiTree(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	if person, ok := archive.Persons[query]; ok {
		return query, person, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	ids := sortedKeys(archive.Persons)
	for _, id := range ids {
		person := archive.Persons[id]
		name := extractPersonName(person)
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
			name := extractPersonName(archive.Persons[id])
			lines = append(lines, fmt.Sprintf("  %s  %s", id, name))
		}

		return "", nil, fmt.Errorf("multiple persons match %q:\n%s\nUse an exact person ID to disambiguate", query, strings.Join(lines, "\n"))
	}
}

// generateWikiTreeBio builds the full WikiTree biography markup for a person.
func generateWikiTreeBio(personID string, person *glxlib.Person, archive *glxlib.GLXFile) string {
	var b strings.Builder
	refs := &refTracker{citations: make(map[string]int)}

	// == Biography ==
	b.WriteString("== Biography ==\n\n")

	// Opening sentence
	writeOpeningSentence(&b, personID, person, archive, refs)

	// Census / Residence
	writeCensusSection(&b, personID, archive, refs)

	// Military Service
	writeMilitarySection(&b, personID, archive, refs)

	// Marriage(s)
	writeMarriageSection(&b, personID, archive, refs)

	// Death and Burial
	writeDeathSection(&b, personID, person, archive, refs)

	// Children
	writeChildrenSection(&b, personID, archive)

	// == Research Notes ==
	writeResearchNotes(&b, personID, person, archive)

	// == Sources ==
	b.WriteString("== Sources ==\n")
	b.WriteString("<references />\n")
	writeSourcesList(&b, personID, archive, refs)

	return b.String()
}

// refTracker manages inline citation numbering.
type refTracker struct {
	citations map[string]int // citation ID -> first use order
	nextRef   int
}

// ref returns the WikiTree <ref> markup for a citation.
// First use emits the full citation text; subsequent uses emit a short ref.
func (r *refTracker) ref(citationID string, archive *glxlib.GLXFile) string {
	if _, seen := r.citations[citationID]; seen {
		return fmt.Sprintf(`<ref name="%s" />`, citationID)
	}

	r.nextRef++
	r.citations[citationID] = r.nextRef

	text := formatCitationText(citationID, archive)

	return fmt.Sprintf(`<ref name="%s">%s</ref>`, citationID, text)
}

// formatCitationText builds the citation text for a <ref> tag.
func formatCitationText(citationID string, archive *glxlib.GLXFile) string {
	cit, ok := archive.Citations[citationID]
	if !ok {
		return citationID
	}

	// Prefer the citation_text property (Evidence Explained format)
	citText := propertyString(cit.Properties, "citation_text")
	if citText != "" {
		return citText
	}

	// Fall back to source title + locator
	var parts []string
	if src, ok := archive.Sources[cit.SourceID]; ok {
		parts = append(parts, src.Title)
	}
	locator := propertyString(cit.Properties, "locator")
	if locator != "" {
		parts = append(parts, locator)
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}

	return citationID
}

// refsForAssertions returns ref markup for all citations on assertions matching
// a person and optionally a property.
func refsForAssertions(personID, property string, archive *glxlib.GLXFile, refs *refTracker) string {
	var refParts []string

	ids := sortedKeys(archive.Assertions)
	for _, id := range ids {
		a := archive.Assertions[id]
		if a.Subject.Person != personID {
			continue
		}
		if property != "" && a.Property != property {
			continue
		}
		for _, citID := range a.Citations {
			refParts = append(refParts, refs.ref(citID, archive))
		}
	}

	return strings.Join(refParts, "")
}

// refsForEvent returns ref markup for citations linked to assertions about an event.
func refsForEvent(eventID string, archive *glxlib.GLXFile, refs *refTracker) string {
	var refParts []string

	ids := sortedKeys(archive.Assertions)
	for _, id := range ids {
		a := archive.Assertions[id]
		if a.Subject.Event != eventID {
			continue
		}
		for _, citID := range a.Citations {
			refParts = append(refParts, refs.ref(citID, archive))
		}
	}

	return strings.Join(refParts, "")
}

// writeOpeningSentence writes the birth/origin line.
func writeOpeningSentence(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile, refs *refTracker) {
	name := extractPersonName(person)
	b.WriteString(fmt.Sprintf("'''%s'''", name))

	bornOn := propertyString(person.Properties, "born_on")
	bornAt := propertyString(person.Properties, "born_at")
	placeName := wikiTreePlaceName(bornAt, archive)

	if bornOn != "" && placeName != "" {
		b.WriteString(fmt.Sprintf(" was born %s in %s", narrativeDateWT(bornOn), placeName))
	} else if bornOn != "" {
		b.WriteString(fmt.Sprintf(" was born %s", narrativeDateWT(bornOn)))
	} else if placeName != "" {
		b.WriteString(fmt.Sprintf(" was born in %s", placeName))
	}

	birthRefs := refsForAssertions(personID, "birth_date", archive, refs)
	if birthRefs == "" {
		birthRefs = refsForAssertions(personID, "birthplace", archive, refs)
	}
	b.WriteString(birthRefs)

	// Gender
	gender := propertyString(person.Properties, "gender")
	if gender == "" {
		gender = propertyString(person.Properties, "sex")
	}
	if strings.EqualFold(gender, "male") {
		b.WriteString(". He")
	} else if strings.EqualFold(gender, "female") {
		b.WriteString(". She")
	} else {
		b.WriteString(". They")
	}

	// Occupation
	occ := wikiTreePropertyValue(person.Properties, "occupation")
	if occ != "" {
		b.WriteString(fmt.Sprintf(" was a %s", strings.ToLower(occ)))
	}

	occRefs := refsForAssertions(personID, "occupation", archive, refs)
	b.WriteString(occRefs)

	b.WriteString(".\n\n")
}

// writeCensusSection writes census event entries.
func writeCensusSection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker) {
	events := findPersonEventsByType(personID, "census", archive)
	if len(events) == 0 {
		return
	}

	b.WriteString("=== Census Records ===\n\n")

	for _, we := range events {
		event := we.Event
		date := string(event.Date)
		place := wikiTreePlaceName(event.PlaceID, archive)

		var line string
		switch {
		case date != "" && place != "":
			line = fmt.Sprintf("In the %s census, %s was enumerated in %s", date, extractPersonNameShort(personID, archive), place)
		case date != "":
			line = fmt.Sprintf("In the %s census", date)
		default:
			line = "A census record exists"
		}

		eventRefs := refsForEvent(we.ID, archive, refs)
		// Also check for citations on assertions about this person's residence at this date
		b.WriteString(line + eventRefs + ".\n\n")

		if event.Notes != "" {
			b.WriteString(event.Notes + "\n\n")
		}
	}
}

// writeMilitarySection writes military service events.
func writeMilitarySection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker) {
	events := findPersonEventsByType(personID, "military_service", archive)
	if len(events) == 0 {
		return
	}

	b.WriteString("=== Military Service ===\n\n")

	for _, we := range events {
		event := we.Event

		// Look for assertions about military service
		serviceRefs := refsForAssertions(personID, "military_service", archive, refs)
		unitRefs := refsForAssertions(personID, "military_unit", archive, refs)
		rankRefs := refsForAssertions(personID, "military_rank", archive, refs)
		allRefs := serviceRefs + unitRefs + rankRefs

		// If no assertion refs, try event-level refs
		if allRefs == "" {
			allRefs = refsForEvent(we.ID, archive, refs)
		}

		if event.Notes != "" {
			b.WriteString(event.Notes + allRefs + "\n\n")
		} else {
			date := string(event.Date)
			if date != "" {
				b.WriteString(fmt.Sprintf("Served in the military (%s)%s.\n\n", date, allRefs))
			} else {
				b.WriteString(fmt.Sprintf("Served in the military%s.\n\n", allRefs))
			}
		}
	}
}

// writeMarriageSection writes marriage events and spouse relationships.
func writeMarriageSection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker) {
	events := findPersonEventsByType(personID, "marriage", archive)
	if len(events) == 0 {
		return
	}

	b.WriteString("=== Marriage ===\n\n")

	for _, we := range events {
		event := we.Event
		date := string(event.Date)
		place := wikiTreePlaceName(event.PlaceID, archive)

		// Find the spouse in participants
		spouseName := ""
		for _, p := range event.Participants {
			if p.Person != personID {
				if sp, ok := archive.Persons[p.Person]; ok {
					spouseName = extractPersonName(sp)
				} else {
					spouseName = p.Person
				}
			}
		}

		eventRefs := refsForEvent(we.ID, archive, refs)

		var parts []string
		if spouseName != "" {
			parts = append(parts, fmt.Sprintf("married %s", spouseName))
		}
		if date != "" {
			parts = append(parts, fmt.Sprintf("on %s", narrativeDateWT(date)))
		}
		if place != "" {
			parts = append(parts, fmt.Sprintf("in %s", place))
		}

		if len(parts) > 0 {
			name := extractPersonNameShort(personID, archive)
			b.WriteString(fmt.Sprintf("%s %s%s.\n\n", name, strings.Join(parts, " "), eventRefs))
		}
	}
}

// writeDeathSection writes death and burial information.
func writeDeathSection(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile, refs *refTracker) {
	diedOn := propertyString(person.Properties, "died_on")
	diedAt := propertyString(person.Properties, "died_at")
	deathPlace := wikiTreePlaceName(diedAt, archive)

	deathEvents := findPersonEventsByType(personID, "death", archive)
	burialEvents := findPersonEventsByType(personID, "burial", archive)

	hasContent := diedOn != "" || deathPlace != "" || len(deathEvents) > 0 || len(burialEvents) > 0
	if !hasContent {
		return
	}

	b.WriteString("=== Death and Burial ===\n\n")

	name := extractPersonNameShort(personID, archive)

	// Death
	if diedOn != "" || deathPlace != "" {
		deathRefs := refsForAssertions(personID, "death_date", archive, refs)

		switch {
		case diedOn != "" && deathPlace != "":
			b.WriteString(fmt.Sprintf("%s died %s in %s%s.\n\n", name, narrativeDateWT(diedOn), deathPlace, deathRefs))
		case diedOn != "":
			b.WriteString(fmt.Sprintf("%s died %s%s.\n\n", name, narrativeDateWT(diedOn), deathRefs))
		case deathPlace != "":
			b.WriteString(fmt.Sprintf("%s died in %s%s.\n\n", name, deathPlace, deathRefs))
		}
	} else if len(deathEvents) > 0 {
		for _, we := range deathEvents {
			event := we.Event
			eventRefs := refsForEvent(we.ID, archive, refs)
			date := string(event.Date)
			place := wikiTreePlaceName(event.PlaceID, archive)

			switch {
			case date != "" && place != "":
				b.WriteString(fmt.Sprintf("%s died %s in %s%s.\n\n", name, narrativeDateWT(date), place, eventRefs))
			case date != "":
				b.WriteString(fmt.Sprintf("%s died %s%s.\n\n", name, narrativeDateWT(date), eventRefs))
			case place != "":
				b.WriteString(fmt.Sprintf("%s died in %s%s.\n\n", name, place, eventRefs))
			}
		}
	}

	// Burial
	for _, we := range burialEvents {
		event := we.Event
		burialRefs := refsForAssertions(personID, "burial_place", archive, refs)
		if burialRefs == "" {
			burialRefs = refsForEvent(we.ID, archive, refs)
		}
		place := wikiTreePlaceName(event.PlaceID, archive)

		if place != "" {
			b.WriteString(fmt.Sprintf("%s was buried at %s%s.\n\n", name, place, burialRefs))
		}

		if event.Notes != "" {
			b.WriteString(event.Notes + "\n\n")
		}
	}
}

// writeChildrenSection lists children found via parent-child relationships.
func writeChildrenSection(b *strings.Builder, personID string, archive *glxlib.GLXFile) {
	var children []string

	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		relType := strings.ToLower(rel.Type)

		isParentChild := relType == "parent_child" || relType == "biological_parent_child" ||
			relType == "adoptive_parent_child" || relType == "foster_parent_child" || relType == "step_parent"
		if !isParentChild {
			continue
		}

		isParent := false
		for _, p := range rel.Participants {
			if p.Person == personID && strings.EqualFold(p.Role, "parent") {
				isParent = true

				break
			}
		}
		if !isParent {
			continue
		}

		for _, p := range rel.Participants {
			if strings.EqualFold(p.Role, "child") {
				childName := p.Person
				if child, ok := archive.Persons[p.Person]; ok {
					childName = extractPersonName(child)
					bornOn := propertyString(child.Properties, "born_on")
					if bornOn != "" {
						childName += " (b. " + bornOn + ")"
					}
				}
				children = append(children, childName)
			}
		}
	}

	if len(children) == 0 {
		return
	}

	b.WriteString("=== Children ===\n\n")
	for _, child := range children {
		b.WriteString(fmt.Sprintf("# %s\n", child))
	}
	b.WriteString("\n")
}

// writeResearchNotes writes the Research Notes section from assertion notes and low-confidence items.
func writeResearchNotes(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile) {
	var notes []string

	// Person-level notes
	if person.Notes != "" {
		notes = append(notes, person.Notes)
	}

	// Low-confidence or noteworthy assertions
	ids := sortedKeys(archive.Assertions)
	for _, id := range ids {
		a := archive.Assertions[id]
		if a.Subject.Person != personID {
			continue
		}
		if a.Confidence != "" && a.Confidence != "high" && a.Notes != "" {
			notes = append(notes, fmt.Sprintf("%s (%s confidence): %s", a.Property, a.Confidence, a.Notes))
		}
	}

	if len(notes) == 0 {
		return
	}

	b.WriteString("== Research Notes ==\n\n")
	for _, note := range notes {
		b.WriteString(note + "\n\n")
	}
}

// writeSourcesList writes a "See also:" list of sources not already cited inline.
func writeSourcesList(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker) {
	// Collect all source IDs referenced by this person's assertions
	sourceIDs := map[string]bool{}

	ids := sortedKeys(archive.Assertions)
	for _, id := range ids {
		a := archive.Assertions[id]
		if a.Subject.Person != personID {
			continue
		}
		for _, srcID := range a.Sources {
			sourceIDs[srcID] = true
		}
		for _, citID := range a.Citations {
			if cit, ok := archive.Citations[citID]; ok {
				sourceIDs[cit.SourceID] = true
			}
		}
	}

	// Find sources not already cited via <ref>
	var uncited []string
	for srcID := range sourceIDs {
		cited := false
		for citID := range refs.citations {
			if cit, ok := archive.Citations[citID]; ok && cit.SourceID == srcID {
				cited = true

				break
			}
		}
		if !cited {
			uncited = append(uncited, srcID)
		}
	}

	if len(uncited) == 0 {
		return
	}

	sort.Strings(uncited)
	b.WriteString("See also:\n")
	for _, srcID := range uncited {
		if src, ok := archive.Sources[srcID]; ok {
			b.WriteString(fmt.Sprintf("* %s\n", src.Title))
		}
	}
}

// ============================================================================
// Helpers
// ============================================================================

// findPersonEventsByType finds all events of a given type where the person participates, sorted by date.
func findPersonEventsByType(personID, eventType string, archive *glxlib.GLXFile) []wikiTreeEvent {
	var events []wikiTreeEvent

	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]
		if !strings.EqualFold(event.Type, eventType) {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				events = append(events, wikiTreeEvent{ID: id, Event: event})

				break
			}
		}
	}

	// Sort by date
	sort.Slice(events, func(i, j int) bool {
		return string(events[i].Event.Date) < string(events[j].Event.Date)
	})

	return events
}

// extractPersonNameShort returns the given name (first name) for narrative use.
func extractPersonNameShort(personID string, archive *glxlib.GLXFile) string {
	person, ok := archive.Persons[personID]
	if !ok {
		return personID
	}
	name := extractPersonName(person)

	// Try to get just the given name from structured name fields
	raw, ok := person.Properties["name"]
	if ok {
		if list, ok := raw.([]any); ok && len(list) > 0 {
			if m, ok := list[0].(map[string]any); ok {
				if fields, ok := m["fields"].(map[string]any); ok {
					if given, ok := fields["given"].(string); ok && given != "" {
						return given
					}
				}
			}
		}
		if m, ok := raw.(map[string]any); ok {
			if fields, ok := m["fields"].(map[string]any); ok {
				if given, ok := fields["given"].(string); ok && given != "" {
					return given
				}
			}
		}
	}

	// Fall back to first word of full name
	parts := strings.Fields(name)
	if len(parts) > 0 {
		return parts[0]
	}

	return name
}

// wikiTreePlaceName resolves a place ID to its full hierarchical name.
func wikiTreePlaceName(placeID string, archive *glxlib.GLXFile) string {
	if placeID == "" {
		return ""
	}
	place, ok := archive.Places[placeID]
	if !ok {
		return placeID
	}

	return place.Name
}

// wikiTreePropertyValue extracts a display value from a property that may be
// a simple string, a temporal list, or a structured map.
func wikiTreePropertyValue(props map[string]any, key string) string {
	raw, ok := props[key]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	if m, ok := raw.(map[string]any); ok {
		if v, ok := m["value"]; ok {
			return fmt.Sprint(v)
		}
	}
	if list, ok := raw.([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if v, ok := m["value"]; ok {
				return fmt.Sprint(v)
			}
		}
	}

	return fmt.Sprint(raw)
}

// narrativeDateWT converts a GLX date string to natural English for WikiTree.
func narrativeDateWT(date string) string {
	d := strings.TrimSpace(date)
	if d == "" {
		return ""
	}

	upper := strings.ToUpper(d)

	if strings.HasPrefix(upper, "ABT ") {
		return "about " + d[4:]
	}
	if strings.HasPrefix(upper, "BEF ") {
		return "before " + d[4:]
	}
	if strings.HasPrefix(upper, "AFT ") {
		return "after " + d[4:]
	}
	if strings.HasPrefix(upper, "BET ") {
		rest := d[4:]
		if idx := strings.Index(strings.ToUpper(rest), " AND "); idx >= 0 {
			return "between " + rest[:idx] + " and " + rest[idx+5:]
		}
	}

	return "on " + d
}
