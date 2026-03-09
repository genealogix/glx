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
	"os/exec"
	"path/filepath"
	"runtime"
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
// If outputPath is non-empty, writes to that file; otherwise prints to stdout.
func showWikiTree(archivePath, personQuery, outputPath string) error {
	archive, err := loadArchiveForWikiTree(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPersonForWikiTree(archive, personQuery)
	if err != nil {
		return err
	}

	bio := generateWikiTreeBio(personID, person, archive)

	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(bio), filePermissions); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		absPath, _ := filepath.Abs(outputPath)
		fmt.Fprintf(os.Stderr, "Biography written to %s\n", absPath)
	}

	// Copy to clipboard
	if err := copyToClipboard(bio); err == nil {
		fmt.Fprintln(os.Stderr, "Biography copied to clipboard.")
	}

	if outputPath == "" {
		fmt.Print(bio)
	}

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

	// Pre-index: person -> related entity IDs (events, assertions)
	relatedIDs := buildPersonEntityIndex(archive)

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
		staleFiles := findStaleFiles(personID, relatedIDs[personID], fileMtimes, genParsed)
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

	return os.WriteFile(trackingPath, data, filePermissions)
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
		fmt.Fprintf(os.Stderr, "Warning: corrupted tracking file %s: %v\n", trackingPath, err)

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
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot access %s: %v\n", path, err)

			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".glx" {
			return nil
		}
		rel, relErr := filepath.Rel(archiveDir, path)
		if relErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot compute relative path for %s: %v\n", path, relErr)

			return nil
		}
		mtimes[rel] = info.ModTime()

		return nil
	})

	return mtimes
}

// buildPersonEntityIndex creates a map of person ID -> set of related entity IDs
// (the person's own ID, plus event and assertion IDs that reference them).
func buildPersonEntityIndex(archive *glxlib.GLXFile) map[string]map[string]bool {
	idx := make(map[string]map[string]bool)

	// Initialize with person IDs themselves
	for personID := range archive.Persons {
		idx[personID] = map[string]bool{personID: true}
	}

	// Index events by participant
	for eventID, event := range archive.Events {
		for _, p := range event.Participants {
			if _, ok := idx[p.Person]; ok {
				idx[p.Person][eventID] = true
			}
		}
	}

	// Index assertions by subject
	for assertionID, assertion := range archive.Assertions {
		if pid := assertion.Subject.Person; pid != "" {
			if _, ok := idx[pid]; ok {
				idx[pid][assertionID] = true
			}
		}
	}

	return idx
}

// findStaleFiles checks if any GLX files referencing a person were modified after genTime.
// entityIDs is the pre-indexed set of entity IDs related to the person.
func findStaleFiles(personID string, entityIDs map[string]bool, fileMtimes map[string]time.Time, genTime time.Time) []string {
	var stale []string

	for path, mtime := range fileMtimes {
		if !mtime.After(genTime) {
			continue
		}

		for entityID := range entityIDs {
			if strings.Contains(path, entityID) {
				stale = append(stale, path)

				break
			}
		}
	}

	return stale
}

// copyToClipboard copies text to the system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}

	cmd.Stdin = strings.NewReader(text)

	return cmd.Run()
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

	// Flowing life narrative: birth, residence, occupation, death
	writeLifeNarrative(&b, personID, person, archive, refs, idx)

	// Military Service (subsection if applicable)
	writeMilitarySection(&b, personID, archive, refs, idx)

	// Marriage and Family
	writeMarriageAndFamily(&b, personID, archive, refs, idx)

	// == Research Notes ==
	writeResearchNotes(&b, personID, person, archive)

	// == Sources ==
	b.WriteString("== Sources ==\n")
	b.WriteString("<references />\n")
	writeSourcesList(&b, personID, archive, refs)

	return sanitizeBioText(b.String())
}

// sanitizeBioText replaces non-ASCII characters that cause encoding issues
// on Windows terminals and clipboards.
func sanitizeBioText(text string) string {
	text = strings.ReplaceAll(text, "\u2014", "--")  // em-dash
	text = strings.ReplaceAll(text, "\u2013", "-")   // en-dash
	text = strings.ReplaceAll(text, "\u2018", "'")   // left single quote
	text = strings.ReplaceAll(text, "\u2019", "'")   // right single quote
	text = strings.ReplaceAll(text, "\u201c", "\"")  // left double quote
	text = strings.ReplaceAll(text, "\u201d", "\"")  // right double quote

	return text
}

// refTracker manages inline citation numbering.
type refTracker struct {
	citations map[string]int // citation ID -> first use order
	nextRef   int
}

// ref returns the WikiTree <ref> markup for a citation.
// First use emits the full citation text; subsequent uses emit a short ref.
func (r *refTracker) ref(citationID string, archive *glxlib.GLXFile) string {
	name := shortRefName(citationID)

	if _, seen := r.citations[citationID]; seen {
		return fmt.Sprintf(`<ref name="%s" />`, name)
	}

	r.nextRef++
	r.citations[citationID] = r.nextRef

	text := formatCitationText(citationID, archive)

	return fmt.Sprintf(`<ref name="%s">%s</ref>`, name, text)
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

// shortRefName converts a citation ID to a shorter WikiTree ref name.
// WikiTree convention: short, lowercase, descriptive.
func shortRefName(citationID string) string {
	return strings.TrimPrefix(citationID, "citation-")
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

// refsForMultipleProperties collects unique citation IDs across multiple assertion
// properties for a person, then emits refs without duplicates.
func refsForMultipleProperties(personID string, properties []string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) string {
	seen := make(map[string]bool)
	var refParts []string

	for _, prop := range properties {
		key := personID + "\x00" + prop
		for _, a := range idx.byPersonProperty[key] {
			for _, citID := range a.Citations {
				if !seen[citID] {
					seen[citID] = true
					refParts = append(refParts, refs.ref(citID, archive))
				}
			}
		}
	}

	return strings.Join(refParts, "")
}

// writeLifeNarrative writes a flowing biographical narrative weaving together
// birth, marriage, settlement, census, occupation, and death into connected prose.
func writeLifeNarrative(b *strings.Builder, personID string, person *glxlib.Person, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	name := extractPersonName(person)
	shortName := extractPersonNameShort(personID, archive)
	pronoun := pronounFor(person.Properties)

	// --- Birth ---
	b.WriteString(fmt.Sprintf("'''%s'''", name))

	bornOn := propertyString(person.Properties, "born_on")
	bornAt := propertyString(person.Properties, "born_at")
	birthPlace := wikiTreePlaceName(bornAt, archive)

	hasBirth := bornOn != "" || birthPlace != ""
	if bornOn != "" && birthPlace != "" {
		b.WriteString(fmt.Sprintf(" was born %s in %s", narrativeDateWT(bornOn), birthPlace))
	} else if bornOn != "" {
		b.WriteString(fmt.Sprintf(" was born %s", narrativeDateWT(bornOn)))
	} else if birthPlace != "" {
		b.WriteString(fmt.Sprintf(" was born in %s", birthPlace))
	}

	birthRefs := refsForMultipleProperties(personID, []string{"born_on", "born_at"}, archive, refs, idx)
	if hasBirth {
		b.WriteString(".")
		b.WriteString(birthRefs)
	} else {
		b.WriteString(".")
	}

	// --- Gather narrative data ---
	spouseIDs := findSpouseIDs(personID, archive)
	censusEvents := findHouseholdCensusEvents(personID, archive)

	// --- First marriage + settlement ---
	if len(spouseIDs) > 0 {
		firstSpouseID := spouseIDs[0]
		spouseName := firstSpouseID
		if sp, ok := archive.Persons[firstSpouseID]; ok {
			spouseName = extractPersonName(sp)
		}
		desc := spouseDescription(firstSpouseID, archive)

		if desc != "" {
			b.WriteString(fmt.Sprintf(" %s married %s, %s", pronoun, spouseName, desc))
		} else {
			b.WriteString(fmt.Sprintf(" %s married %s", pronoun, spouseName))
		}

		// Weave census into settlement narrative
		if len(censusEvents) > 0 {
			writeCensusWithSettlement(b, birthPlace, censusEvents, archive, refs, idx)
		} else {
			b.WriteString(".")
		}

		// Later marriages
		for i := 1; i < len(spouseIDs); i++ {
			laterName := spouseIDs[i]
			if sp, ok := archive.Persons[laterName]; ok {
				laterName = extractPersonName(sp)
			}
			b.WriteString(fmt.Sprintf(" %s later married %s.", shortName, laterName))
		}
	} else if len(censusEvents) > 0 {
		// No marriage — standalone census narrative
		writeCensusStandalone(b, pronoun, censusEvents, archive, refs, idx)
	}

	// Occupation (if not already implied by spouse description)
	occ := wikiTreePropertyValue(person.Properties, "occupation")
	if occ != "" {
		b.WriteString(fmt.Sprintf(" %s was a %s.", pronoun, strings.ToLower(occ)))
		b.WriteString(refsForAssertions(personID, "occupation", archive, refs, idx))
	}

	b.WriteString("\n\n")

	// --- Death and Burial ---
	writeDeathNarrative(b, personID, person, shortName, archive, refs, idx)
}

// writeCensusWithSettlement writes census data woven into a marriage/settlement sentence.
// Continues from "married [spouse]" — adds settlement location and census refs.
func writeCensusWithSettlement(b *strings.Builder, birthPlace string, events []wikiTreeEvent, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	firstPlace := wikiTreePlaceName(events[0].Event.PlaceID, archive)

	// Check if all census events share the same place
	allSamePlace := true
	for _, we := range events[1:] {
		if wikiTreePlaceName(we.Event.PlaceID, archive) != firstPlace {
			allSamePlace = false

			break
		}
	}

	if allSamePlace && firstPlace != "" {
		// "...and the family settled in [place], where they appear in the [year] census<ref> and the [year] census.<ref>"
		if firstPlace != birthPlace {
			b.WriteString(fmt.Sprintf(", and the family settled in %s", firstPlace))
		}
		b.WriteString(", where they appear in ")

		for i, we := range events {
			year := extractYear(string(we.Event.Date))
			ref := censusEventRef(we, archive, refs, idx)

			if i > 0 {
				b.WriteString(" and ")
			}
			b.WriteString(fmt.Sprintf("the %s census", year))

			if i == len(events)-1 {
				b.WriteString(".")
				b.WriteString(ref)
			} else {
				b.WriteString(ref)
			}
		}
	} else {
		// Different places — list as separate sentences
		b.WriteString(".")

		for _, we := range events {
			year := extractYear(string(we.Event.Date))
			place := wikiTreePlaceName(we.Event.PlaceID, archive)
			ref := censusEventRef(we, archive, refs, idx)

			if year != "" && place != "" {
				b.WriteString(fmt.Sprintf(" By %s, the family was living in %s.", year, place))
			} else if year != "" {
				b.WriteString(fmt.Sprintf(" They appeared in the %s census.", year))
			}
			b.WriteString(ref)
		}
	}
}

// writeCensusStandalone writes census data when there's no marriage to weave it with.
func writeCensusStandalone(b *strings.Builder, pronoun string, events []wikiTreeEvent, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	prevPlace := ""

	for _, we := range events {
		year := extractYear(string(we.Event.Date))
		place := wikiTreePlaceName(we.Event.PlaceID, archive)
		ref := censusEventRef(we, archive, refs, idx)

		switch {
		case place != "" && place == prevPlace && year != "":
			b.WriteString(fmt.Sprintf(" %s was still residing there in %s.", pronoun, year))
		case place != "" && year != "":
			b.WriteString(fmt.Sprintf(" By %s, %s was living in %s.", year, strings.ToLower(pronoun), place))
		case place != "":
			b.WriteString(fmt.Sprintf(" %s was living in %s.", pronoun, place))
		case year != "":
			b.WriteString(fmt.Sprintf(" %s appeared in the %s census.", pronoun, year))
		}
		b.WriteString(ref)

		if place != "" {
			prevPlace = place
		}
	}
}

// censusEventRef returns a ref tag for a census event, using assertion refs or
// falling back to the first paragraph of event notes.
func censusEventRef(we wikiTreeEvent, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) string {
	eventRefs := refsForEvent(we.ID, archive, refs, idx)
	if eventRefs == "" && we.Event.Notes != "" {
		noteText := firstNoteParagraph(we.Event.Notes)
		if noteText != "" {
			return fmt.Sprintf("<ref>%s</ref>", noteText)
		}
	}

	return eventRefs
}

// spouseDescription returns a brief description of a spouse, e.g. "a Virginia-born farmer".
func spouseDescription(spouseID string, archive *glxlib.GLXFile) string {
	spouse, ok := archive.Persons[spouseID]
	if !ok {
		return ""
	}

	var parts []string

	bornAt := propertyString(spouse.Properties, "born_at")
	if region := birthRegion(bornAt, archive); region != "" {
		parts = append(parts, region+"-born")
	}

	occ := wikiTreePropertyValue(spouse.Properties, "occupation")
	if occ != "" {
		parts = append(parts, strings.ToLower(occ))
	}

	if len(parts) == 0 {
		return ""
	}

	return "a " + strings.Join(parts, " ")
}

// birthRegion extracts the first component of a place name (e.g. "Virginia" from "Virginia, United States").
func birthRegion(placeID string, archive *glxlib.GLXFile) string {
	if placeID == "" {
		return ""
	}
	place, ok := archive.Places[placeID]
	if !ok {
		return ""
	}
	parts := strings.SplitN(place.Name, ",", 2)

	return strings.TrimSpace(parts[0])
}

// pronounFor returns "He", "She", or "They" based on person properties.
func pronounFor(props map[string]any) string {
	gender := propertyString(props, "gender")
	if gender == "" {
		gender = propertyString(props, "sex")
	}

	if strings.EqualFold(gender, "male") {
		return "He"
	}
	if strings.EqualFold(gender, "female") {
		return "She"
	}

	return "They"
}

// writeDeathNarrative writes death and burial as narrative prose (no subsection header).
func writeDeathNarrative(b *strings.Builder, personID string, person *glxlib.Person, shortName string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	diedOn := propertyString(person.Properties, "died_on")
	diedAt := propertyString(person.Properties, "died_at")
	deathPlace := wikiTreePlaceName(diedAt, archive)
	deathEvents := findPersonEventsByType(personID, "death", archive)
	burialEvents := findPersonEventsByType(personID, "burial", archive)

	if diedOn == "" && deathPlace == "" && len(deathEvents) == 0 && len(burialEvents) == 0 {
		return
	}

	if diedOn != "" || deathPlace != "" {
		deathRefs := refsForMultipleProperties(personID, []string{"died_on", "died_at"}, archive, refs, idx)
		switch {
		case diedOn != "" && deathPlace != "":
			b.WriteString(fmt.Sprintf("%s died %s in %s.%s", shortName, narrativeDateWT(diedOn), deathPlace, deathRefs))
		case diedOn != "":
			b.WriteString(fmt.Sprintf("%s died %s.%s", shortName, narrativeDateWT(diedOn), deathRefs))
		case deathPlace != "":
			b.WriteString(fmt.Sprintf("%s died in %s.%s", shortName, deathPlace, deathRefs))
		}
	} else if len(deathEvents) > 0 {
		we := deathEvents[0]
		eventRefs := refsForEvent(we.ID, archive, refs, idx)
		date := string(we.Event.Date)
		place := wikiTreePlaceName(we.Event.PlaceID, archive)
		switch {
		case date != "" && place != "":
			b.WriteString(fmt.Sprintf("%s died %s in %s.%s", shortName, narrativeDateWT(date), place, eventRefs))
		case date != "":
			b.WriteString(fmt.Sprintf("%s died %s.%s", shortName, narrativeDateWT(date), eventRefs))
		case place != "":
			b.WriteString(fmt.Sprintf("%s died in %s.%s", shortName, place, eventRefs))
		}
	}

	for _, we := range burialEvents {
		burialRefs := refsForEvent(we.ID, archive, refs, idx)
		place := wikiTreePlaceName(we.Event.PlaceID, archive)
		if place != "" {
			b.WriteString(fmt.Sprintf(" %s was buried at %s.%s", shortName, place, burialRefs))
		}
	}

	b.WriteString("\n\n")
}

// isFANContinuation returns true if a paragraph looks like a continuation of
// FAN neighbor data (e.g. "Previous page:", "Current page:", "Next page:", "Same page:").
func isFANContinuation(s string) bool {
	lower := strings.ToLower(s)

	return strings.HasPrefix(lower, "previous page") ||
		strings.HasPrefix(lower, "current page") ||
		strings.HasPrefix(lower, "next page") ||
		strings.HasPrefix(lower, "same page")
}

// firstNoteParagraph returns the first non-empty, non-FAN paragraph from event notes.
func firstNoteParagraph(notes string) string {
	for _, para := range strings.Split(notes, "\n\n") {
		trimmed := strings.TrimSpace(para)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "FAN") || isFANContinuation(trimmed) {
			continue
		}

		return trimmed
	}

	return ""
}

// humanPropertyName converts a GLX property key to a human-readable label.
func humanPropertyName(prop string) string {
	names := map[string]string{
		"born_on":    "Birth date",
		"born_at":    "Birthplace",
		"died_on":    "Death date",
		"died_at":    "Death place",
		"married":    "Marriage",
		"occupation": "Occupation",
	}
	if name, ok := names[prop]; ok {
		return name
	}

	return strings.ReplaceAll(prop, "_", " ")
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
			notes := strings.TrimRight(event.Notes, " \n")
			if !strings.HasSuffix(notes, ".") {
				notes += "."
			}
			b.WriteString(notes + allRefs + "\n\n")
		} else {
			date := string(event.Date)
			if date != "" {
				b.WriteString(fmt.Sprintf("Served in the military (%s).%s\n\n", date, allRefs))
			} else {
				b.WriteString(fmt.Sprintf("Served in the military.%s\n\n", allRefs))
			}
		}
	}
}

// writeMarriageAndFamily writes the Marriage and Family subsection.
// Lists marriages with dates and context, then children with vital details.
func writeMarriageAndFamily(b *strings.Builder, personID string, archive *glxlib.GLXFile, refs *refTracker, idx *assertionIndex) {
	spouseIDs := findSpouseIDs(personID, archive)
	children := findChildren(personID, archive)

	if len(spouseIDs) == 0 && len(children) == 0 {
		return
	}

	b.WriteString("=== Marriage and Family ===\n\n")

	// List marriages with dates and context
	for _, spouseID := range spouseIDs {
		writeMarriageEntry(b, personID, spouseID, archive)
	}

	// Children list
	if len(children) > 0 {
		fullName := extractPersonName(archive.Persons[personID])
		b.WriteString(fmt.Sprintf("Children of %s:\n", fullName))

		for _, child := range children {
			b.WriteString(fmt.Sprintf("# %s\n", child.displayName))
		}

		b.WriteString("\n")
	}
}

// writeMarriageEntry writes a single marriage with date, spouse description, and end context.
func writeMarriageEntry(b *strings.Builder, personID, spouseID string, archive *glxlib.GLXFile) {
	// Get subject's short name for pronoun reference
	subjectShort := personID
	if _, ok := archive.Persons[personID]; ok {
		subjectShort = extractPersonNameShort(personID, archive)
	}

	spouseName := spouseID
	if sp, ok := archive.Persons[spouseID]; ok {
		spouseName = extractPersonName(sp)
	}

	marriageDate, marriagePlace := findMarriageDate(personID, spouseID, archive)

	// Build marriage sentence: "Mary married Daniel Lane about 1850-1851."
	sentence := subjectShort + " married " + spouseName
	if marriageDate != "" && marriagePlace != "" {
		sentence += fmt.Sprintf(" %s in %s", narrativeDateWT(marriageDate), marriagePlace)
	} else if marriageDate != "" {
		sentence += " " + narrativeDateWT(marriageDate)
	} else if marriagePlace != "" {
		sentence += " in " + marriagePlace
	}
	sentence += "."

	// Check if spouse died — append as separate sentence
	spouseDeathInfo := personDeathSummary(spouseID, archive)
	if spouseDeathInfo != "" {
		spouseShort := extractPersonNameShort(spouseID, archive)
		sentence += " " + spouseShort + " " + spouseDeathInfo + "."
	}

	b.WriteString(sentence + "\n\n")
}

// findMarriageDate returns the date and place of a marriage between two people,
// checking the relationship's start_event first, then the relationship notes.
func findMarriageDate(personID, spouseID string, archive *glxlib.GLXFile) (string, string) {
	for _, rel := range archive.Relationships {
		relType := strings.ToLower(rel.Type)
		if relType != "marriage" && relType != "spouse" {
			continue
		}

		hasPerson := false
		hasSpouse := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				hasPerson = true
			}
			if p.Person == spouseID {
				hasSpouse = true
			}
		}
		if !hasPerson || !hasSpouse {
			continue
		}

		// Check start_event for formal marriage date
		if rel.StartEvent != "" {
			if event, ok := archive.Events[rel.StartEvent]; ok {
				date := string(event.Date)
				place := wikiTreePlaceName(event.PlaceID, archive)

				return date, place
			}
		}

		// Extract approximate date from notes (e.g., "~1850-1851")
		if rel.Notes != "" {
			if date := extractApproxDateFromNotes(rel.Notes); date != "" {
				return date, ""
			}
		}

		return "", ""
	}

	return "", ""
}

// extractApproxDateFromNotes extracts approximate date patterns from relationship notes.
// Looks for "~YYYY", "~YYYY-YYYY", "Likely married ~YYYY" patterns.
func extractApproxDateFromNotes(notes string) string {
	// Look for "~YYYY-YYYY" or "~YYYY"
	for _, word := range strings.Fields(notes) {
		word = strings.TrimRight(word, ".,;:")
		if strings.HasPrefix(word, "~") {
			return "ABT " + strings.TrimPrefix(word, "~")
		}
	}

	return ""
}

// personDeathSummary returns a brief death summary for narrative use, e.g. "died on 1863-02-10".
func personDeathSummary(personID string, archive *glxlib.GLXFile) string {
	person, ok := archive.Persons[personID]
	if !ok {
		return ""
	}

	diedOn := propertyString(person.Properties, "died_on")
	diedAt := propertyString(person.Properties, "died_at")
	deathPlace := wikiTreePlaceName(diedAt, archive)

	if diedOn != "" && deathPlace != "" {
		return fmt.Sprintf("died %s in %s", narrativeDateWT(diedOn), deathPlace)
	}
	if diedOn != "" {
		return "died " + narrativeDateWT(diedOn)
	}

	// Check death events
	deathEvents := findPersonEventsByType(personID, "death", archive)
	if len(deathEvents) > 0 {
		date := string(deathEvents[0].Event.Date)
		place := wikiTreePlaceName(deathEvents[0].Event.PlaceID, archive)
		if date != "" && place != "" {
			return fmt.Sprintf("died %s in %s", narrativeDateWT(date), place)
		}
		if date != "" {
			return "died " + narrativeDateWT(date)
		}
	}

	// Check burial events as fallback (implies death)
	burialEvents := findPersonEventsByType(personID, "burial", archive)
	if len(burialEvents) > 0 {
		place := wikiTreePlaceName(burialEvents[0].Event.PlaceID, archive)
		if place != "" {
			return "was buried at " + place
		}
	}

	return ""
}


// childInfo holds display information for a child, used for sorting.
type childInfo struct {
	personID    string
	displayName string
	birthYear   string
}

// findChildren returns sorted children for a person via parent-child relationships.
// Includes birth year, death year, and married name where available.
func findChildren(personID string, archive *glxlib.GLXFile) []childInfo {
	seen := make(map[string]bool)
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
			if strings.EqualFold(p.Role, "child") && !seen[p.Person] {
				seen[p.Person] = true
				childEntry := childInfo{personID: p.Person, displayName: p.Person}
				if child, ok := archive.Persons[p.Person]; ok {
					childEntry.displayName = extractPersonName(child)
					bornOn := propertyString(child.Properties, "born_on")
					childEntry.birthYear = extractYear(bornOn)
					childEntry.displayName += childVitalAnnotation(p.Person, childEntry.birthYear, archive)
				}
				children = append(children, childEntry)
			}
		}
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

	return children
}

// childVitalAnnotation builds a parenthetical annotation for a child listing.
// e.g. " (b. 1855; d. 1920)" or " (b. 1855; m. Thomas Clark)" or " (b. 1855)"
func childVitalAnnotation(childID, birthYear string, archive *glxlib.GLXFile) string {
	var parts []string

	if birthYear != "" {
		parts = append(parts, "b. "+birthYear)
	}

	// Check for death year
	if child, ok := archive.Persons[childID]; ok {
		diedOn := propertyString(child.Properties, "died_on")
		if deathYear := extractYear(diedOn); deathYear != "" {
			parts = append(parts, "d. "+deathYear)
		} else {
			// Check death events
			deathEvents := findPersonEventsByType(childID, "death", archive)
			if len(deathEvents) > 0 {
				if dy := extractYear(string(deathEvents[0].Event.Date)); dy != "" {
					parts = append(parts, "d. "+dy)
				}
			}
		}
	}

	// Check for spouse (married name)
	spouseName := findFirstSpouseName(childID, archive)
	if spouseName != "" {
		parts = append(parts, "m. "+spouseName)
	}

	if len(parts) == 0 {
		return ""
	}

	return " (" + strings.Join(parts, "; ") + ")"
}

// findFirstSpouseName returns the name of the chronologically first spouse of a person.
// Uses marriage event dates when available, falls back to sorted relationship ID order.
func findFirstSpouseName(personID string, archive *glxlib.GLXFile) string {
	type marriageInfo struct {
		spouseID string
		date     string // for sorting
	}

	var marriages []marriageInfo

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		relType := strings.ToLower(rel.Type)
		if relType != "marriage" && relType != "spouse" {
			continue
		}

		hasPerson := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				hasPerson = true

				break
			}
		}
		if !hasPerson {
			continue
		}

		for _, p := range rel.Participants {
			if p.Person != personID {
				date := ""
				if rel.StartEvent != "" {
					if event, ok := archive.Events[rel.StartEvent]; ok {
						date = string(event.Date)
					}
				}
				marriages = append(marriages, marriageInfo{spouseID: p.Person, date: date})
			}
		}
	}

	if len(marriages) == 0 {
		return ""
	}

	// Sort by date (marriages with dates first, then alphabetical)
	sort.Slice(marriages, func(i, j int) bool {
		if marriages[i].date == "" && marriages[j].date == "" {
			return marriages[i].spouseID < marriages[j].spouseID
		}
		if marriages[i].date == "" {
			return false
		}
		if marriages[j].date == "" {
			return true
		}

		return marriages[i].date < marriages[j].date
	})

	if sp, ok := archive.Persons[marriages[0].spouseID]; ok {
		return extractPersonName(sp)
	}

	return marriages[0].spouseID
}

// writeResearchNotes writes the Research Notes section from assertion notes and low-confidence items.
// Deduplicates notes that cover the same topic from different sources.
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
			propName := humanPropertyName(a.Property)
			notes = append(notes, fmt.Sprintf("%s: %s (%s confidence)", propName, a.Notes, a.Confidence))
		}
	}

	// Collect relationship notes, but skip if already covered by an assertion or person note.
	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		if rel.Notes == "" {
			continue
		}
		hasPerson := false
		for _, p := range rel.Participants {
			if p.Person == personID {
				hasPerson = true

				break
			}
		}
		if !hasPerson {
			continue
		}
		if !isRedundantNote(rel.Notes, notes) {
			notes = append(notes, rel.Notes)
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

// isRedundantNote checks whether a candidate note is substantially covered by
// any already-collected note. Two notes are considered redundant if they share
// evidence references (year + record type patterns like "1897 marriage record",
// "1860 census") on the same topic.
func isRedundantNote(candidate string, existing []string) bool {
	candidateRefs := extractEvidenceRefs(candidate)
	if len(candidateRefs) == 0 {
		return false
	}

	for _, note := range existing {
		noteRefs := extractEvidenceRefs(note)
		shared := 0
		for ref := range candidateRefs {
			if noteRefs[ref] {
				shared++
			}
		}
		// If any evidence reference is shared, the notes cover the same ground
		if shared > 0 {
			return true
		}
	}

	return false
}

// extractEvidenceRefs extracts evidence reference identifiers from text.
// Looks for patterns like "1897 marriage record", "1860 census", "1855 census"
// that identify the specific record being cited.
func extractEvidenceRefs(text string) map[string]bool {
	refs := make(map[string]bool)
	lower := strings.ToLower(text)
	words := strings.Fields(lower)

	for i, word := range words {
		// Look for 4-digit years followed by a record type keyword
		if len(word) >= 4 && isYear(word[:4]) && i+1 < len(words) {
			nextWord := strings.TrimRight(words[i+1], ".,;:)")
			switch nextWord {
			case "census", "marriage", "death", "burial", "birth",
				"baptism", "probate", "will", "deed", "land":
				refs[word[:4]+" "+nextWord] = true
			case "u", "us", "federal", "state":
				// "1860 u.s. census" or "1860 federal census"
				for j := i + 2; j < len(words) && j <= i+3; j++ {
					kw := strings.TrimRight(words[j], ".,;:)")
					if kw == "census" || kw == "marriage" {
						refs[word[:4]+" "+kw] = true

						break
					}
				}
			}
		}

		// Also catch "marriage record of [name]" pattern
		if word == "marriage" && i+2 < len(words) && words[i+1] == "record" {
			// Find the year nearby (within previous 3 words)
			for j := max(0, i-3); j < i; j++ {
				w := strings.TrimRight(words[j], ".,;:)")
				if len(w) >= 4 && isYear(w[:4]) {
					refs[w[:4]+" marriage"] = true
				}
			}
		}
	}

	return refs
}

// isYear checks if a string looks like a year (1000-2099).
func isYear(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return s >= "1000" && s <= "2099"
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

// findSpouseIDs returns person IDs of all spouses via marriage relationships,
// in deterministic order (sorted by relationship ID).
func findSpouseIDs(personID string, archive *glxlib.GLXFile) []string {
	relIDs := make([]string, 0, len(archive.Relationships))
	for id := range archive.Relationships {
		relIDs = append(relIDs, id)
	}
	sort.Strings(relIDs)

	var spouses []string

	for _, relID := range relIDs {
		rel := archive.Relationships[relID]
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
