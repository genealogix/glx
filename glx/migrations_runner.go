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
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	glxlib "github.com/genealogix/glx/go-glx"
)

// undatedMigrationSortKey is the sort key dateSortKey assigns to entries
// whose date is empty or unparseable; they sort last and are excluded from
// movement detection.
const undatedMigrationSortKey = "\xff"

// Output format values for the --format flag.
const (
	migrationsFormatText = "text"
	migrationsFormatJSON = "json"
)

// migrationsRelationChild is the relation label findRelatedPersons assigns
// to a person's children.
const migrationsRelationChild = "child"

// migrationsPlaceColumnCap bounds the place column width in text output so
// one very deep hierarchy path doesn't push labels off-screen.
const migrationsPlaceColumnCap = 50

// migrationEntry is a single dated place observation for a person, drawn
// from an event the person participated in, a child's birth event, or a
// residence property value.
type migrationEntry struct {
	Date    string `json:"date,omitempty"`
	PlaceID string `json:"place_id,omitempty"` // empty when the place is a freeform string, not an entity reference
	Place   string `json:"place"`              // canonical hierarchy path, or the freeform value verbatim
	Region  string `json:"region,omitempty"`   // normalized region used for movement detection
	Label   string `json:"label"`              // what kind of observation this is (e.g. "Birth", "1860 US Census")

	sortKey string // chronological sort key; not serialized
}

// migrationMovement is a detected change of region between two consecutive
// dated observations.
type migrationMovement struct {
	FromRegion string `json:"from_region"`
	ToRegion   string `json:"to_region"`
	FromDate   string `json:"from_date,omitempty"` // last observation in the origin region
	ToDate     string `json:"to_date,omitempty"`   // first observation in the destination region
}

// migrationReport is the migration timeline for one person. Family is only
// populated on the top-level report when --family is set; Relation is only
// set on the nested family reports.
type migrationReport struct {
	Person     string              `json:"person"`
	PersonName string              `json:"person_name"`
	Relation   string              `json:"relation,omitempty"`
	Entries    []migrationEntry    `json:"entries"`
	Movements  []migrationMovement `json:"movements,omitempty"`
	Family     []migrationReport   `json:"family,omitempty"`
}

// migrationStop is one stop in a person's region sequence, used for
// --pattern matching output.
type migrationStop struct {
	Region string `json:"region"`
	Place  string `json:"place"`
	Date   string `json:"date,omitempty"`
}

// migrationPatternMatch is one person whose region sequence contains the
// requested pattern, with the stops that matched each pattern term.
type migrationPatternMatch struct {
	Person     string          `json:"person"`
	PersonName string          `json:"person_name"`
	Stops      []migrationStop `json:"stops"`
}

// migrationPatternReport is the result of a --pattern search across all
// persons in the archive.
type migrationPatternReport struct {
	Pattern []string                `json:"pattern"`
	Matches []migrationPatternMatch `json:"matches"`
}

// showMigrations is the entry point for the migrations command. Exactly one
// of personQuery or pattern must be provided (validated in cli_commands.go
// before the archive is loaded, and defensively here).
func showMigrations(io *IOStreams, archivePath, personQuery, pattern string, includeFamily bool, format string) error {
	if format != "" && format != migrationsFormatText && format != migrationsFormatJSON {
		return fmt.Errorf("%w: %q", ErrMigrationsUnknownFormat, format)
	}

	// Validate the pattern before loading the archive: a malformed
	// invocation shouldn't pay for a full archive parse.
	var terms []string
	if pattern != "" {
		terms = splitMigrationPattern(pattern)
		if len(terms) < 2 {
			return fmt.Errorf("%w: %q", ErrMigrationsPatternTooShort, pattern)
		}
	}

	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	if pattern != "" {
		report := matchMigrationPattern(archive, terms)
		if format == migrationsFormatJSON {
			return printMigrationsJSON(io, report)
		}
		printMigrationPatternText(io, report)

		return nil
	}

	personID, _, err := findPersonByQuery(archive, personQuery)
	if err != nil {
		return err
	}

	report := buildMigrationReport(archive, personID, "")
	if includeFamily {
		for _, rel := range findRelatedPersons(personID, archive) {
			report.Family = append(report.Family, buildMigrationReport(archive, rel.PersonID, rel.Relation))
		}
	}

	if format == migrationsFormatJSON {
		return printMigrationsJSON(io, report)
	}
	printMigrationReportText(io, &report)
	for i := range report.Family {
		printMigrationReportText(io, &report.Family[i])
	}

	return nil
}

// buildMigrationReport collects and orders all place observations for one
// person and derives the region movements between them.
func buildMigrationReport(archive *glxlib.GLXFile, personID, relation string) migrationReport {
	entries := collectMigrationEntries(personID, archive)

	return migrationReport{
		Person:     personID,
		PersonName: extractPersonName(archive.Persons[personID]),
		Relation:   relation,
		Entries:    entries,
		Movements:  computeMovements(entries),
	}
}

// collectMigrationEntries gathers every dated place observation for a person:
// events they participated in, their children's birth events (a child's
// birthplace is evidence of the parent's residence), and residence property
// values. Entries are deduplicated and sorted chronologically, undated last.
func collectMigrationEntries(personID string, archive *glxlib.GLXFile) []migrationEntry {
	var entries []migrationEntry

	// childID -> display name, for recognizing and labeling child births
	children := make(map[string]string)
	for _, rel := range findRelatedPersons(personID, archive) {
		if rel.Relation == migrationsRelationChild {
			children[rel.PersonID] = rel.Name
		}
	}

	// Single pass over events: direct participation, plus children's births
	// the person isn't a participant in. A birth where one of the person's
	// children participates is the child's birth, never the person's own, so
	// it's labeled "Birth of child (...)" even on direct participation (e.g.
	// a `parent` role) — a bare event-type label would read as the person's
	// own birth.
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || event.PlaceID == "" {
			continue
		}
		childName, isChildBirth := childBirthOf(event, personID, children)

		switch {
		case isChildBirth:
			entries = append(entries, newMigrationEntry(
				string(event.Date), event.PlaceID, fmt.Sprintf("Birth of child (%s)", childName), archive))
		case timelineIsParticipant(personID, event):
			entries = append(entries, newMigrationEntry(string(event.Date), event.PlaceID, migrationEventLabel(event), archive))
		}
	}

	// Residence property values (plain string, structured map, or temporal list)
	if person := archive.Persons[personID]; person != nil && person.Properties != nil {
		entries = append(entries, residenceMigrationEntries(person.Properties[glxlib.PersonPropertyResidence], archive)...)
	}

	entries = dedupeMigrationEntries(entries)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].sortKey < entries[j].sortKey
	})

	return entries
}

// childBirthOf reports whether an event is the birth of one of the person's
// children, returning the child's display name. A participant other than the
// person who is in the children map identifies the event.
func childBirthOf(event *glxlib.Event, personID string, children map[string]string) (string, bool) {
	if len(children) == 0 || !strings.EqualFold(event.Type, glxlib.EventTypeBirth) {
		return "", false
	}
	for _, p := range event.Participants {
		if p.Person == personID {
			continue
		}
		if name, ok := children[p.Person]; ok {
			return name, true
		}
	}

	return "", false
}

// migrationEventLabel picks a display label for an event observation: the
// event's title when present (e.g. "1860 US Census"), otherwise its
// formatted type (e.g. "Birth").
func migrationEventLabel(event *glxlib.Event) string {
	if title := strings.TrimSpace(event.Title); title != "" {
		return title
	}

	return formatEventTypeLabel(event.Type)
}

// residenceMigrationEntries extracts dated place observations from a
// residence property value in any of its supported shapes:
//
//	residence: place-millbrook                          # plain string
//	residence: {value: place-millbrook, date: 1855}     # structured map
//	residence: [{value: ..., date: ...}, ...]           # temporal list
func residenceMigrationEntries(raw any, archive *glxlib.GLXFile) []migrationEntry {
	var entries []migrationEntry

	appendEntry := func(value, date string) {
		if value == "" {
			return
		}
		entries = append(entries, newMigrationEntry(date, value, "Residence", archive))
	}

	switch v := raw.(type) {
	case string:
		appendEntry(v, "")
	case map[string]any:
		appendEntry(stringField(v, "value"), stringField(v, "date"))
	case []any:
		for _, item := range v {
			switch it := item.(type) {
			case string:
				appendEntry(it, "")
			case map[string]any:
				appendEntry(stringField(it, "value"), stringField(it, "date"))
			}
		}
	}

	return entries
}

// stringField returns m[key] rendered as a string, or "" when absent. Dates
// in temporal property maps may be parsed by YAML as integers (date: 1855),
// so fmt.Sprint rather than a string type assertion.
func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}

	return fmt.Sprint(v)
}

// newMigrationEntry builds an entry from a place reference that may be a
// place entity ID or a freeform place string (residence values predating
// place entities).
func newMigrationEntry(date, placeRef, label string, archive *glxlib.GLXFile) migrationEntry {
	entry := migrationEntry{
		Date:    date,
		Label:   label,
		sortKey: dateSortKey(date),
	}

	if place, ok := archive.Places[placeRef]; ok && place != nil {
		entry.PlaceID = placeRef
		entry.Place = buildCanonicalPath(placeRef, archive.Places)
		entry.Region = regionForPlace(placeRef, archive.Places)
	} else {
		entry.Place = placeRef
		entry.Region = regionFromFreeform(placeRef)
	}

	return entry
}

// migrationContainerTypes are place types too large to be a meaningful
// migration region; the region is the topmost ancestor below them. The
// standard vocabulary defines country; continent is a common custom type
// (e.g. the Westeros example archive) that would otherwise swallow every
// region beneath it.
var migrationContainerTypes = map[string]bool{
	glxlib.PlaceTypeCountry: true,
	"continent":             true,
}

// regionForPlace walks a place's parent hierarchy and returns the normalized
// name of the topmost ancestor below the country/continent level — the
// state/region level at which migration is meaningful (e.g. "Wisconsin" for
// Millbrook → Hartford Co. → Wisconsin → United States). Falls back to the
// topmost named ancestor when the whole chain is containers.
func regionForPlace(placeID string, places map[string]*glxlib.Place) string {
	var names, types []string
	visited := make(map[string]bool)
	current := placeID

	for current != "" && !visited[current] {
		visited[current] = true
		place, ok := places[current]
		if !ok || place == nil {
			break
		}
		names = append(names, strings.TrimSpace(place.Name))
		types = append(types, place.Type)
		current = place.ParentID
	}

	for i, name := range slices.Backward(names) {
		if !migrationContainerTypes[types[i]] && name != "" {
			return normalizeRegionName(name)
		}
	}
	for _, name := range slices.Backward(names) {
		if name != "" {
			return normalizeRegionName(name)
		}
	}

	return ""
}

// regionFromFreeform derives a region from a freeform place string by taking
// its last comma-separated component (e.g. "Millbrook, Hartford Co., WI" →
// "WI"), matching the smallest-to-largest convention of place strings.
func regionFromFreeform(place string) string {
	parts := strings.Split(place, ",")
	for _, part := range slices.Backward(parts) {
		if p := strings.TrimSpace(part); p != "" {
			return normalizeRegionName(p)
		}
	}

	return ""
}

// normalizeRegionName strips a trailing " Territory" so that pre-statehood
// jurisdictions compare equal to their successor states ("Florida Territory"
// and "Florida" are the same region for migration purposes).
func normalizeRegionName(name string) string {
	const territorySuffix = " territory"
	if len(name) > len(territorySuffix) && strings.EqualFold(name[len(name)-len(territorySuffix):], territorySuffix) {
		return strings.TrimSpace(name[:len(name)-len(territorySuffix)])
	}

	return name
}

// dedupeMigrationEntries removes exact duplicate observations (same date,
// place, and label), which arise when the same event is reachable both
// directly and through a relationship.
func dedupeMigrationEntries(entries []migrationEntry) []migrationEntry {
	seen := make(map[string]bool)
	result := make([]migrationEntry, 0, len(entries))

	for _, e := range entries {
		key := e.Date + "\x00" + e.PlaceID + "\x00" + e.Place + "\x00" + e.Label
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, e)
	}

	return result
}

// computeMovements scans the dated observations in chronological order and
// records each change of region. Consecutive observations in the same region
// collapse; undated and region-less observations are skipped.
func computeMovements(entries []migrationEntry) []migrationMovement {
	var movements []migrationMovement
	var prev *migrationEntry

	for i := range entries {
		entry := &entries[i]
		if entry.sortKey == undatedMigrationSortKey || entry.Region == "" {
			continue
		}
		if prev != nil && !strings.EqualFold(prev.Region, entry.Region) {
			movements = append(movements, migrationMovement{
				FromRegion: prev.Region,
				ToRegion:   entry.Region,
				FromDate:   prev.Date,
				ToDate:     entry.Date,
			})
		}
		prev = entry
	}

	return movements
}

// ============================================================================
// Pattern matching
// ============================================================================

// splitMigrationPattern splits a comma-separated pattern into trimmed,
// non-empty terms.
func splitMigrationPattern(pattern string) []string {
	var terms []string
	for part := range strings.SplitSeq(pattern, ",") {
		if p := strings.TrimSpace(part); p != "" {
			terms = append(terms, p)
		}
	}

	return terms
}

// matchMigrationPattern finds every person whose chronological region
// sequence contains the pattern terms in order.
func matchMigrationPattern(archive *glxlib.GLXFile, terms []string) migrationPatternReport {
	report := migrationPatternReport{Pattern: terms}

	for _, personID := range sortedKeys(archive.Persons) {
		if archive.Persons[personID] == nil {
			continue
		}
		stops := migrationStops(collectMigrationEntries(personID, archive))
		matched := matchStopsAgainstTerms(stops, terms)
		if matched == nil {
			continue
		}
		report.Matches = append(report.Matches, migrationPatternMatch{
			Person:     personID,
			PersonName: extractPersonName(archive.Persons[personID]),
			Stops:      matched,
		})
	}

	return report
}

// migrationStops collapses a person's dated observations into their region
// sequence: one stop per consecutive run of the same region, keeping the
// first observation of each run.
func migrationStops(entries []migrationEntry) []migrationStop {
	var stops []migrationStop

	for _, e := range entries {
		if e.sortKey == undatedMigrationSortKey || e.Region == "" {
			continue
		}
		if len(stops) > 0 && strings.EqualFold(stops[len(stops)-1].Region, e.Region) {
			continue
		}
		stops = append(stops, migrationStop{Region: e.Region, Place: e.Place, Date: e.Date})
	}

	return stops
}

// matchStopsAgainstTerms greedily matches pattern terms as a subsequence of
// the stop list. Each term matches a stop when it appears (case-insensitive)
// in the stop's region or anywhere in its place hierarchy path. Returns the
// matched stops, or nil when the pattern is not contained in the sequence.
func matchStopsAgainstTerms(stops []migrationStop, terms []string) []migrationStop {
	matched := make([]migrationStop, 0, len(terms))
	next := 0

	for _, stop := range stops {
		if next >= len(terms) {
			break
		}
		if stopMatchesTerm(stop, terms[next]) {
			matched = append(matched, stop)
			next++
		}
	}

	if next < len(terms) {
		return nil
	}

	return matched
}

// stopMatchesTerm reports whether a pattern term matches a stop's region or
// place path, case-insensitively. The territory-normalized region is checked
// so "Florida" matches a stop recorded in "Florida Territory".
func stopMatchesTerm(stop migrationStop, term string) bool {
	lowerTerm := strings.ToLower(term)

	return strings.Contains(strings.ToLower(stop.Region), lowerTerm) ||
		strings.Contains(strings.ToLower(stop.Place), lowerTerm)
}

// ============================================================================
// Output
// ============================================================================

// printMigrationsJSON renders a report as indented JSON on MachineOut so
// `glx --quiet migrations ... --format json` stays scriptable.
func printMigrationsJSON(io *IOStreams, report any) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal migrations report: %w", err)
	}
	fmt.Fprintln(io.MachineOut, string(data)) //nolint:errcheck // CLI output

	return nil
}

// printMigrationReportText prints one person's migration timeline and
// detected movements.
func printMigrationReportText(io *IOStreams, report *migrationReport) {
	if report.Relation != "" {
		io.Printf("\nMigration timeline for %s (%s) — %s:\n\n", report.PersonName, report.Person, report.Relation)
	} else {
		io.Printf("\nMigration timeline for %s (%s):\n\n", report.PersonName, report.Person)
	}

	if len(report.Entries) == 0 {
		io.Println("  No place observations found.")
		io.Println("")

		return
	}

	placeWidth := 0
	for _, e := range report.Entries {
		if n := utf8.RuneCountInString(e.Place); n > placeWidth {
			placeWidth = n
		}
	}
	if placeWidth > migrationsPlaceColumnCap {
		placeWidth = migrationsPlaceColumnCap
	}

	var undated []migrationEntry
	for _, e := range report.Entries {
		if e.sortKey == undatedMigrationSortKey {
			undated = append(undated, e)

			continue
		}
		io.Printf("  %-18s  %s  (%s)\n", displayDate(e.Date), padPlaceColumn(e.Place, placeWidth), e.Label)
	}

	if len(undated) > 0 {
		io.Println("")
		io.Println("  Undated:")
		for _, e := range undated {
			io.Printf("  %-18s  %s  (%s)\n", displayDate(e.Date), padPlaceColumn(e.Place, placeWidth), e.Label)
		}
	}

	if len(report.Movements) > 0 {
		io.Println("")
		io.Println("  Movements:")
		for _, m := range report.Movements {
			io.Printf("    %s\n", formatMovement(m))
		}
	}

	io.Println("")
}

// padPlaceColumn truncates a place string to migrationsPlaceColumnCap runes
// (replacing the tail with "…") and pads it to the column width. Both
// truncation and padding count runes, not bytes — fmt's %-*s pads by byte
// length and never truncates, so a long or multibyte hierarchy path would
// break the column alignment the cap exists to protect.
func padPlaceColumn(place string, width int) string {
	runes := []rune(place)
	if len(runes) > migrationsPlaceColumnCap {
		place = string(runes[:migrationsPlaceColumnCap-1]) + "…"
		runes = []rune(place)
	}
	if pad := width - len(runes); pad > 0 {
		place += strings.Repeat(" ", pad)
	}

	return place
}

// formatMovement renders one movement as "Florida → Wisconsin (ABT 1832 – ABT 1852)".
func formatMovement(m migrationMovement) string {
	line := m.FromRegion + " → " + m.ToRegion
	from := displayDate(m.FromDate)
	to := displayDate(m.ToDate)
	if m.FromDate != "" || m.ToDate != "" {
		line += " (" + from + " – " + to + ")"
	}

	return line
}

// printMigrationPatternText prints the persons matching a migration pattern.
func printMigrationPatternText(io *IOStreams, report migrationPatternReport) {
	io.Printf("\nPeople with %s migration:\n", strings.Join(report.Pattern, " → "))

	if len(report.Matches) == 0 {
		io.Println("  (no matches found)")
		io.Println("")

		return
	}

	for _, match := range report.Matches {
		var stops []string
		for _, s := range match.Stops {
			stops = append(stops, fmt.Sprintf("%s %s", s.Region, displayDate(s.Date)))
		}
		io.Printf("  %s (%s) — %s\n", match.PersonName, match.Person, strings.Join(stops, ", "))
	}

	io.Printf("\n  %d %s\n\n", len(report.Matches), pluralize(len(report.Matches), "match", "matches"))
}
