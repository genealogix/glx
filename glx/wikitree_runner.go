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
	"path/filepath"
	"sort"
	"strings"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
	"gopkg.in/yaml.v3"
)

// wikiTreeEvent holds an event with its ID for sorting.
type wikiTreeEvent struct {
	ID    string
	Event *glxlib.Event
}

// wikiTreeTrackingFile is the filename used to track generation timestamps.
const wikiTreeTrackingFile = ".wikitree.yml"

// wikiTreeTracking maps person IDs to their last generation timestamp.
type wikiTreeTracking map[string]string // person ID -> RFC3339 timestamp

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

	// Record generation timestamp
	if err := recordWikiTreeGeneration(archivePath, personID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update tracking file: %v\n", err)
	}

	return nil
}

// showWikiTreeStale lists persons whose WikiTree bios may be stale.
func showWikiTreeStale(archivePath string) error {
	archive, err := loadArchiveForWikiTree(archivePath)
	if err != nil {
		return err
	}

	tracking := loadWikiTreeTracking(archivePath)
	archiveDir := resolveArchiveDir(archivePath)

	// Collect file modification times for staleness detection
	fileMtimes := collectArchiveFileMtimes(archiveDir)

	ids := sortedKeys(archive.Persons)
	hasOutput := false

	for _, personID := range ids {
		person := archive.Persons[personID]
		name := extractPersonName(person)
		genTime, generated := tracking[personID]

		if !generated {
			fmt.Printf("  %-30s  %-25s  (never generated)\n", personID, name)
			hasOutput = true

			continue
		}

		genParsed, err := time.Parse(time.RFC3339, genTime)
		if err != nil {
			fmt.Printf("  %-30s  %-25s  (invalid timestamp)\n", personID, name)
			hasOutput = true

			continue
		}

		// Check if any related file was modified after generation
		staleFiles := findStaleFiles(personID, archive, fileMtimes, genParsed)
		if len(staleFiles) > 0 {
			fmt.Printf("  %-30s  %-25s  generated %s  STALE (%d files changed)\n",
				personID, name, genParsed.Format("2006-01-02"), len(staleFiles))
			hasOutput = true
		}
	}

	if !hasOutput {
		fmt.Println("All generated biographies are up to date.")
	}

	return nil
}

// recordWikiTreeGeneration updates the tracking file with the current timestamp.
func recordWikiTreeGeneration(archivePath, personID string) error {
	archiveDir := resolveArchiveDir(archivePath)
	trackingPath := filepath.Join(archiveDir, wikiTreeTrackingFile)

	tracking := loadWikiTreeTracking(archivePath)
	tracking[personID] = time.Now().UTC().Format(time.RFC3339)

	data, err := yaml.Marshal(tracking)
	if err != nil {
		return err
	}

	return os.WriteFile(trackingPath, data, 0o644)
}

// loadWikiTreeTracking reads the tracking file from the archive directory.
func loadWikiTreeTracking(archivePath string) wikiTreeTracking {
	archiveDir := resolveArchiveDir(archivePath)
	trackingPath := filepath.Join(archiveDir, wikiTreeTrackingFile)

	data, err := os.ReadFile(trackingPath)
	if err != nil {
		return make(wikiTreeTracking)
	}

	var tracking wikiTreeTracking
	if err := yaml.Unmarshal(data, &tracking); err != nil {
		return make(wikiTreeTracking)
	}

	return tracking
}

// resolveArchiveDir returns the directory containing the archive.
func resolveArchiveDir(archivePath string) string {
	info, err := os.Stat(archivePath)
	if err != nil {
		return archivePath
	}
	if info.IsDir() {
		return archivePath
	}

	return filepath.Dir(archivePath)
}

// collectArchiveFileMtimes walks the archive directory and returns a map of
// relative file paths to their modification times.
func collectArchiveFileMtimes(archiveDir string) map[string]time.Time {
	mtimes := make(map[string]time.Time)

	_ = filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".glx" {
			return nil
		}
		rel, _ := filepath.Rel(archiveDir, path)
		mtimes[rel] = info.ModTime()

		return nil
	})

	return mtimes
}

// findStaleFiles checks if any GLX files referencing a person were modified after genTime.
func findStaleFiles(personID string, archive *glxlib.GLXFile, fileMtimes map[string]time.Time, genTime time.Time) []string {
	var stale []string

	// Check person file
	for path, mtime := range fileMtimes {
		if !mtime.After(genTime) {
			continue
		}

		// Person files
		if strings.Contains(path, personID) {
			stale = append(stale, path)

			continue
		}

		// Event files — check if person is a participant
		for eventID, event := range archive.Events {
			if !strings.Contains(path, eventID) {
				continue
			}
			for _, p := range event.Participants {
				if p.Person == personID {
					stale = append(stale, path)

					break
				}
			}
		}

		// Assertion files — check if person is the subject
		for assertionID, assertion := range archive.Assertions {
			if !strings.Contains(path, assertionID) {
				continue
			}
			if assertion.Subject.Person == personID {
				stale = append(stale, path)
			}
		}
	}

	return stale
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

// assertionIndex pre-indexes assertions for efficient lookup.
type assertionIndex struct {
	byPersonProperty map[string][]*glxlib.Assertion // "personID\x00property" -> assertions
	byEvent          map[string][]*glxlib.Assertion  // eventID -> assertions
}

// buildAssertionIndex builds an index over all assertions for efficient lookup.
func buildAssertionIndex(archive *glxlib.GLXFile) *assertionIndex {
	idx := &assertionIndex{
		byPersonProperty: make(map[string][]*glxlib.Assertion),
		byEvent:          make(map[string][]*glxlib.Assertion),
	}

	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a.Subject.Person != "" {
			key := a.Subject.Person + "\x00" + a.Property
			idx.byPersonProperty[key] = append(idx.byPersonProperty[key], a)
			// Also index with empty property for "all assertions for person"
			allKey := a.Subject.Person + "\x00"
			idx.byPersonProperty[allKey] = append(idx.byPersonProperty[allKey], a)
		}
		if a.Subject.Event != "" {
			idx.byEvent[a.Subject.Event] = append(idx.byEvent[a.Subject.Event], a)
		}
	}

	return idx
}

// generateWikiTreeBio builds the full WikiTree biography markup for a person.
func generateWikiTreeBio(personID string, person *glxlib.Person, archive *glxlib.GLXFile) string {
	var b strings.Builder
	refs := &refTracker{citations: make(map[string]int)}
	idx := buildAssertionIndex(archive)

	// == Biography ==
	b.WriteString("== Biography ==\n\n")

	// Opening sentence
	writeOpeningSentence(&b, personID, person, archive, refs, idx)

	// Census / Residence
	writeCensusSection(&b, personID, archive, refs, idx)

	// Military Service
	writeMilitarySection(&b, personID, archive, refs, idx)

	// Marriage(s)
	writeMarriageSection(&b, personID, archive, refs, idx)

	// Death and Burial
	writeDeathSection(&b, personID, person, archive, refs, idx)

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
// a person and optionally a property, using a pre-built index.
func refsForAssertions(personID, property string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) string {
	var refParts []string

	key := personID + "\x00" + property
	for _, a := range idx.byPersonProperty[key] {
		for _, citID := range a.Citations {
			refParts = append(refParts, refs.ref(citID, archive))
		}
	}

	return strings.Join(refParts, "")
}

// refsForEvent returns ref markup for citations linked to assertions about an event,
// using a pre-built index.
func refsForEvent(eventID string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) string {
	var refParts []string

	for _, a := range idx.byEvent[eventID] {
		for _, citID := range a.Citations {
			refParts = append(refParts, refs.ref(citID, archive))
		}
	}

	return strings.Join(refParts, "")
}

// writeOpeningSentence writes the birth/origin line.
func writeOpeningSentence(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
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

	birthRefs := refsForAssertions(personID, "born_on", archive, refs, idx)
	if birthRefs == "" {
		birthRefs = refsForAssertions(personID, "born_at", archive, refs, idx)
	}
	b.WriteString(birthRefs)

	// Occupation — only add pronoun + occupation if occupation exists
	occ := wikiTreePropertyValue(person.Properties, "occupation")
	if occ != "" {
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
		b.WriteString(fmt.Sprintf(" was a %s", strings.ToLower(occ)))
		occRefs := refsForAssertions(personID, "occupation", archive, refs, idx)
		b.WriteString(occRefs)
	}

	b.WriteString(".\n\n")
}

// writeCensusSection writes census event entries.
func writeCensusSection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	events := findHouseholdCensusEvents(personID, archive)
	if len(events) == 0 {
		return
	}

	b.WriteString("=== Census Records ===\n\n")

	for _, we := range events {
		event := we.Event
		date := string(event.Date)
		censusYear := extractYear(date)
		place := wikiTreePlaceName(event.PlaceID, archive)

		var line string
		switch {
		case censusYear != "" && place != "":
			line = fmt.Sprintf("In the %s census, %s was enumerated in %s", censusYear, extractPersonNameShort(personID, archive), place)
		case censusYear != "":
			line = fmt.Sprintf("In the %s census", censusYear)
		default:
			line = "A census record exists"
		}

		eventRefs := refsForEvent(we.ID, archive, refs, idx)
		// Also check for citations on assertions about this person's residence at this date
		b.WriteString(line + eventRefs + ".\n\n")

		if event.Notes != "" {
			b.WriteString(event.Notes + "\n\n")
		}
	}
}

// writeMilitarySection writes military service events.
func writeMilitarySection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	events := findPersonEventsByType(personID, "military_service", archive)
	if len(events) == 0 {
		return
	}

	b.WriteString("=== Military Service ===\n\n")

	for _, we := range events {
		event := we.Event

		// Look for assertions about military service
		serviceRefs := refsForAssertions(personID, "military_service", archive, refs, idx)
		unitRefs := refsForAssertions(personID, "military_unit", archive, refs, idx)
		rankRefs := refsForAssertions(personID, "military_rank", archive, refs, idx)
		allRefs := serviceRefs + unitRefs + rankRefs

		// If no assertion refs, try event-level refs
		if allRefs == "" {
			allRefs = refsForEvent(we.ID, archive, refs, idx)
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
func writeMarriageSection(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	events := findPersonEventsByType(personID, "marriage", archive)
	hasEvents := len(events) > 0

	// Also check for marriage relationships (inferred marriages without an event)
	type marriageRelInfo struct {
		spouseName string
		notes      string
	}
	var relMarriages []marriageRelInfo

	if !hasEvents {
		spouseIDs := findSpouseIDs(personID, archive)
		for _, spouseID := range spouseIDs {
			name := spouseID
			if sp, ok := archive.Persons[spouseID]; ok {
				name = extractPersonName(sp)
			}

			// Find the relationship notes
			notes := ""
			for _, rel := range archive.Relationships {
				relType := strings.ToLower(rel.Type)
				if relType != "marriage" && relType != "spouse" {
					continue
				}
				hasPersonID, hasSpouse := false, false
				for _, p := range rel.Participants {
					if p.Person == personID {
						hasPersonID = true
					}
					if p.Person == spouseID {
						hasSpouse = true
					}
				}
				if hasPersonID && hasSpouse && rel.Notes != "" {
					notes = rel.Notes
				}
			}

			relMarriages = append(relMarriages, marriageRelInfo{spouseName: name, notes: notes})
		}
	}

	if !hasEvents && len(relMarriages) == 0 {
		return
	}

	b.WriteString("=== Marriage ===\n\n")

	name := extractPersonNameShort(personID, archive)

	// Write marriage events
	for _, we := range events {
		event := we.Event
		date := string(event.Date)
		place := wikiTreePlaceName(event.PlaceID, archive)

		// Find the spouse in participants (look for spouse-like roles)
		spouseName := ""
		spouseRoles := map[string]bool{
			"spouse": true, "husband": true, "wife": true,
			"groom": true, "bride": true,
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				continue
			}
			role := strings.ToLower(p.Role)
			if spouseRoles[role] || role == "" {
				if sp, ok := archive.Persons[p.Person]; ok {
					spouseName = extractPersonName(sp)
				} else {
					spouseName = p.Person
				}

				break
			}
		}

		eventRefs := refsForEvent(we.ID, archive, refs, idx)

		var parts []string
		if spouseName != "" {
			parts = append(parts, fmt.Sprintf("married %s", spouseName))
		}
		if date != "" {
			parts = append(parts, narrativeDateWT(date))
		}
		if place != "" {
			parts = append(parts, fmt.Sprintf("in %s", place))
		}

		if len(parts) > 0 {
			b.WriteString(fmt.Sprintf("%s %s%s.\n\n", name, strings.Join(parts, " "), eventRefs))
		}
	}

	// Write relationship-based marriages (no event)
	for _, rm := range relMarriages {
		b.WriteString(fmt.Sprintf("%s married %s.\n\n", name, rm.spouseName))
	}
}

// writeDeathSection writes death and burial information.
func writeDeathSection(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
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
		deathRefs := refsForAssertions(personID, "died_on", archive, refs, idx)

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
			eventRefs := refsForEvent(we.ID, archive, refs, idx)
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
		burialRefs := refsForEvent(we.ID, archive, refs, idx)
		place := wikiTreePlaceName(event.PlaceID, archive)

		if place != "" {
			b.WriteString(fmt.Sprintf("%s was buried at %s%s.\n\n", name, place, burialRefs))
		}

		if event.Notes != "" {
			b.WriteString(event.Notes + "\n\n")
		}
	}
}

// childInfo holds display information for a child, used for sorting.
type childInfo struct {
	personID    string
	displayName string
	birthYear   string
}

// writeChildrenSection lists children found via parent-child relationships.
func writeChildrenSection(b *strings.Builder, personID string, archive *glxlib.GLXFile) {
	var children []childInfo

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
				childEntry := childInfo{personID: p.Person, displayName: p.Person}
				if child, ok := archive.Persons[p.Person]; ok {
					childEntry.displayName = extractPersonName(child)
					bornOn := propertyString(child.Properties, "born_on")
					childEntry.birthYear = extractYear(bornOn)
					if childEntry.birthYear != "" {
						childEntry.displayName += " (b. " + childEntry.birthYear + ")"
					}
				}
				children = append(children, childEntry)
			}
		}
	}

	if len(children) == 0 {
		return
	}

	// Sort children by birth year (unknown years sort last)
	sort.Slice(children, func(i, j int) bool {
		if children[i].birthYear == "" {
			return false
		}
		if children[j].birthYear == "" {
			return true
		}

		return children[i].birthYear < children[j].birthYear
	})

	b.WriteString("=== Children ===\n\n")
	for _, child := range children {
		b.WriteString(fmt.Sprintf("# %s\n", child.displayName))
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

// findHouseholdCensusEvents finds census events where the person or their spouse participates.
// This catches household-level census records where only the head-of-household is listed.
func findHouseholdCensusEvents(personID string, archive *glxlib.GLXFile) []wikiTreeEvent {
	// First get direct census events
	direct := findPersonEventsByType(personID, "census", archive)
	seen := make(map[string]bool)
	for _, e := range direct {
		seen[e.ID] = true
	}

	// Find spouse IDs from marriage relationships
	spouseIDs := findSpouseIDs(personID, archive)

	// Find census events where a spouse participates
	for _, spouseID := range spouseIDs {
		spouseEvents := findPersonEventsByType(spouseID, "census", archive)
		for _, e := range spouseEvents {
			if !seen[e.ID] {
				seen[e.ID] = true
				direct = append(direct, e)
			}
		}
	}

	// Re-sort by date
	sort.Slice(direct, func(i, j int) bool {
		return string(direct[i].Event.Date) < string(direct[j].Event.Date)
	})

	return direct
}

// findSpouseIDs returns person IDs of all spouses via marriage relationships.
func findSpouseIDs(personID string, archive *glxlib.GLXFile) []string {
	var spouses []string

	for _, rel := range archive.Relationships {
		relType := strings.ToLower(rel.Type)
		if relType != "marriage" && relType != "spouse" {
			continue
		}

		isMember := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				isMember = true

				break
			}
		}
		if !isMember {
			continue
		}

		for _, p := range rel.Participants {
			if p.Person != personID {
				spouses = append(spouses, p.Person)
			}
		}
	}

	return spouses
}

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

// extractYear returns just the 4-digit year from a date string.
// Handles formats like "1860-07-17", "1860", "ABT 1858", "BEF 1920".
func extractYear(date string) string {
	d := strings.TrimSpace(date)
	if d == "" {
		return ""
	}

	// Strip GLX date prefixes
	for _, prefix := range []string{"ABT ", "BEF ", "AFT ", "BET "} {
		if strings.HasPrefix(strings.ToUpper(d), prefix) {
			d = strings.TrimSpace(d[len(prefix):])

			break
		}
	}

	if len(d) >= 4 {
		return d[:4]
	}

	return d
}
