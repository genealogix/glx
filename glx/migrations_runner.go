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
	"math"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/genealogix/glx/go-glx/glxdate"
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
	Region  string `json:"region,omitempty"`   // normalized top-level region (state, province, district), used for --pattern stops
	Label   string `json:"label"`              // what kind of observation this is (e.g. "Birth", "1860 US Census")
	Role    string `json:"role,omitempty"`     // the person's participant role, for event observations

	// Excluded marks an observation shown in the timeline but left out of
	// movement detection, with Note saying why: the person's role does not
	// put them at the place (a grantor selling land from another state, an
	// absent legatee), or a dated residence covering the date places them
	// elsewhere (#1330).
	Excluded bool   `json:"excluded,omitempty"`
	Note     string `json:"note,omitempty"`

	sortKey  string   // chronological sort key; not serialized
	location bool     // a residence value: an explicit statement of where the person lived
	visit    bool     // attendance at someone else's record: places the visitor, not their residence
	path     []string // normalized place hierarchy, outermost first, below the country/continent level
	ids      []string // place IDs of the date-scoped path, most specific first (entity places only)
}

// migrationMovement is a detected change of place between two consecutive
// dated observations. FromRegion and ToRegion name the two places at the
// highest level they differ: the states for a move from Florida to Wisconsin,
// the villages themselves for a move between two villages of one district
// (#1370).
type migrationMovement struct {
	FromRegion string `json:"from_region"`
	ToRegion   string `json:"to_region"`
	FromDate   string `json:"from_date,omitempty"` // last observation in the origin place
	ToDate     string `json:"to_date,omitempty"`   // first observation in the destination place
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
		Movements:  computeMovements(entries, archive),
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

		participation, role, ok := glxlib.EventParticipation(event, personID, archive.ParticipantRoles)
		if !ok && !isChildBirth {
			continue
		}
		label := migrationEventLabel(event)
		if isChildBirth {
			label = fmt.Sprintf("Birth of child (%s)", childName)
		}
		entry := newMigrationEntry(string(event.Date), event.PlaceID, label, archive)
		entry.Role = role
		switch {
		case ok && !participation.Present:
			entry.Excluded = true
			entry.Note = fmt.Sprintf("role %q does not imply presence", role)
		case ok && !participation.OwnRecord && isParentAtDeathRecord(event, personID):
			// A child's death or burial places the child, not the parent's
			// household: an infant buried in a war refuge or the mother's home
			// parish says nothing of where the father lived (#1370).
			entry.Excluded = true
			entry.Note = fmt.Sprintf("role %q on a %s places the deceased, not this person", role, strings.ToLower(formatEventTypeLabel(event.Type)))
		case ok && !participation.OwnRecord && !participation.DeclaredPresent && !migrationParentAtBirth(event, personID):
			// Excluded after residence periods are applied, so a dated
			// residence that already explains the appearance keeps its note.
			entry.visit = true
		}
		entries = append(entries, entry)
	}

	// Residence property values (plain string, structured map, or temporal list)
	if person := archive.Persons[personID]; person != nil && person.Properties != nil {
		entries = append(entries, residenceMigrationEntries(person.Properties[glxlib.PersonPropertyResidence], archive)...)
	}

	applyResidencePeriods(entries, archive)
	for i := range entries {
		if e := &entries[i]; e.visit && !e.Excluded {
			e.Excluded = true
			e.Note = fmt.Sprintf("attendance as %q shows a visit, not residence", e.Role)
		}
	}
	entries = dedupeMigrationEntries(entries)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].sortKey < entries[j].sortKey
	})

	return entries
}

// childBirthOf reports whether an event is the birth of one of the person's
// children, returning the child's display name. A participant other than the
// person who is in the children map and owns the record identifies the event.
func childBirthOf(event *glxlib.Event, personID string, children map[string]string) (string, bool) {
	if len(children) == 0 || !strings.EqualFold(event.Type, glxlib.EventTypeBirth) {
		return "", false
	}
	for _, p := range event.Participants {
		if p.Person == personID {
			continue
		}
		if name, ok := children[p.Person]; ok && glxlib.ClassifyParticipation(event.Type, p.Role, nil).OwnRecord {
			return name, true
		}
	}

	return "", false
}

// migrationDeathEventTypes are the event types that record where someone
// died or was laid to rest.
var migrationDeathEventTypes = map[string]bool{
	glxlib.EventTypeDeath:     true,
	glxlib.EventTypeBurial:    true,
	glxlib.EventTypeCremation: true,
	"funeral":                 true,
}

// birthRecordEventTypes are the event types that record a child's birth or
// its rite, where the parents are named as part of the record.
var birthRecordEventTypes = map[string]bool{
	glxlib.EventTypeBirth:       true,
	glxlib.EventTypeBaptism:     true,
	glxlib.EventTypeChristening: true,
}

// migrationParentRoles are the roles that name a person as the parent of an
// event's principal.
var migrationParentRoles = map[string]bool{
	glxlib.ParticipantRoleParent: true,
	"father":                     true,
	"mother":                     true,
}

// isParentRole reports whether a role names the person as a parent of the
// record's principal: a parent-child relationship role (parent,
// adoptive_parent, ...) or a register's father/mother.
func isParentRole(role string) bool {
	role = strings.ToLower(strings.TrimSpace(role))

	return glxlib.IsParentSideRole(role) || migrationParentRoles[role]
}

// isParentAtDeathRecord reports whether personID takes part in a death-type
// event only as a parent of the deceased.
func isParentAtDeathRecord(event *glxlib.Event, personID string) bool {
	if !migrationDeathEventTypes[strings.ToLower(strings.TrimSpace(event.Type))] {
		return false
	}
	parent := false
	for _, p := range event.Participants {
		if p.Person != personID {
			continue
		}
		if !migrationParentRoles[strings.ToLower(strings.TrimSpace(p.Role))] {
			return false
		}
		parent = true
	}

	return parent
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
		entry := newMigrationEntry(date, value, "Residence", archive)
		entry.location = true
		entries = append(entries, entry)
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

// residencePeriod is the bounded civil-date span a residence value covers.
type residencePeriod struct {
	date  glxdate.Date
	span  glxdate.Interval
	place *migrationEntry
	label string // the residence's date and place, for notes
}

// applyResidencePeriods lets a dated residence win over event places inside
// its period (#1330): an observation whose whole possible date span fits
// inside a residence, at a place that none of the covering residences contain
// or fall within, is kept in the timeline but excluded from movement detection. The researcher's explicit
// statement of where the person lived outranks the place of a deed signed,
// an estate settled, or a marriage witnessed on a visit elsewhere.
//
// Only bounded periods count (a single date, BET…AND, FROM…TO); an
// open-ended FROM or a BEF/AFT date says too little about where the period
// stops. Open-ended event spans remain included, and dates in different
// calendars are not compared without conversion.
func applyResidencePeriods(entries []migrationEntry, archive *glxlib.GLXFile) {
	var periods []residencePeriod
	for i := range entries {
		e := &entries[i]
		if !e.location || len(e.placePath()) == 0 {
			continue
		}
		if date, ok := boundedMigrationDate(e.Date); ok {
			periods = append(periods, residencePeriod{date: date, span: date.Timing().Outer, place: e, label: e.Date + " " + e.Place})
		}
	}
	if len(periods) == 0 {
		return
	}

	for i := range entries {
		e := &entries[i]
		if e.location || e.Excluded || len(e.placePath()) == 0 || e.sortKey == undatedMigrationSortKey {
			continue
		}
		date, ok := boundedMigrationDate(e.Date)
		if !ok {
			continue
		}
		if conflict := conflictingResidence(periods, e, date, archive); conflict != nil {
			e.Excluded = true
			e.Note = "inside residence " + conflict.label
		}
	}
}

// conflictingResidence finds a residence that contains the event's whole
// possible span, unless a matching residence preserves the agreement veto.
func conflictingResidence(periods []residencePeriod, event *migrationEntry, date glxdate.Date, archive *glxlib.GLXFile) *residencePeriod {
	span := date.Timing().Outer
	var conflict *residencePeriod
	for i := range periods {
		p := &periods[i]
		if date.Calendar() != p.date.Calendar() || date.CalendarName() != p.date.CalendarName() {
			continue
		}
		samePlace := residenceContainsObservation(p.place, event, archive)
		// Preserve the existing overlap veto when a matching residence
		// covers the first possible date, even if it does not cover the end.
		if samePlace && p.span.Start <= span.Start && span.Start < p.span.End {
			return nil
		}
		if !samePlace && p.span.Contains(span) && conflict == nil {
			conflict = p
		}
	}

	return conflict
}

// residenceContainsObservation compares both places at the observation date,
// not the (possibly much longer) residence period. Fallback to normalized names
// preserves scalar archives' successor names (e.g. Florida Territory/Florida),
// but a truncated historical label is not a complete path for comparing names.
func residenceContainsObservation(home, event *migrationEntry, archive *glxlib.GLXFile) bool {
	at := glxlib.DateString(event.Date)
	if home.PlaceID != "" && event.PlaceID != "" {
		if home.PlaceID == event.PlaceID || archive.PlaceMayHaveAncestorAt(home.PlaceID, event.PlaceID, at) ||
			archive.PlaceMayHaveAncestorAt(event.PlaceID, home.PlaceID, at) {
			return true
		}
	}
	a, aOK := migrationObservationPath(home, at, archive)
	b, bOK := migrationObservationPath(event, at, archive)

	return aOK && bOK && placesCompatible(a, b)
}

func migrationObservationPath(entry *migrationEntry, at glxlib.DateString, archive *glxlib.GLXFile) ([]string, bool) {
	if entry.PlaceID == "" {
		return entry.placePath(), len(entry.placePath()) > 0
	}
	ids := archive.PlacePathAt(entry.PlaceID, at)
	if len(ids) == 0 {
		return nil, false
	}
	root := archive.Places[ids[len(ids)-1]]
	if root == nil || root.HasTemporalParent() || root.ParentID != "" {
		return nil, false
	}
	path := migrationNamesForIDs(ids, archive.Places)

	return path, len(path) != 0
}

// boundedMigrationDate parses a date with finite, known possible bounds.
// Timing retains the written precision of approximate dates; it introduces
// no additional tolerance. Reversed and open spans cannot establish containment.
func boundedMigrationDate(date string) (glxdate.Date, bool) {
	d, err := glxlib.DateString(date).Parse()
	if err != nil || !d.Valid() {
		return d, false
	}
	bounds := d.Timing()

	return d, bounds.Known && !bounds.Reversed && bounds.Outer.Start != math.MinInt && bounds.Outer.End != math.MaxInt && bounds.Outer.Start < bounds.Outer.End
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
		at := glxlib.DateString(date)
		entry.Place = buildCanonicalPathAt(placeRef, at, archive.Places)
		entry.ids = archive.PlacePathAt(placeRef, at)
		entry.path = migrationNamesForIDs(entry.ids, archive.Places)
		entry.Region = migrationRegionAt(placeRef, at, entry.ids, archive)
	} else {
		entry.Place = placeRef
		entry.Region = regionFromFreeform(placeRef)
		entry.path = migrationFreeformPath(placeRef)
	}

	return entry
}

// placePath returns the entry's hierarchy for movement detection, falling
// back to its region alone for entries built without one.
func (e *migrationEntry) placePath() []string {
	if len(e.path) > 0 || e.Region == "" {
		return e.path
	}

	return []string{e.Region}
}

// migrationPlacePath returns a place's normalized names, outermost first,
// with the country/continent levels above the region removed, so that two
// places compare from the level regionForPlace reports downwards
// (Millbrook → [Wisconsin, Hartford Co., Millbrook]). A chain made only of
// containers keeps all of its names.
func migrationPlacePath(placeID string, places map[string]*glxlib.Place) []string {
	return migrationPlacePathAt(placeID, "", places)
}

// migrationPlacePathAt is migrationPlacePath with the hierarchy resolved as of
// date, following each place's parent at that date (see Place.ParentAt).
func migrationPlacePathAt(placeID string, date glxlib.DateString, places map[string]*glxlib.Place) []string {
	return migrationNamesForIDs((&glxlib.GLXFile{Places: places}).PlacePathAt(placeID, date), places)
}

func migrationNamesForIDs(ids []string, places map[string]*glxlib.Place) []string {
	var names, types []string
	for _, id := range ids {
		if place := places[id]; place != nil {
			if name := strings.TrimSpace(place.Name); name != "" {
				names = append(names, normalizeRegionName(name))
				types = append(types, place.Type)
			}
		}
	}
	slices.Reverse(names)
	slices.Reverse(types)

	for i := range names {
		if !migrationContainerTypes[types[i]] {
			return names[i:]
		}
	}

	return names
}

// migrationFreeformPath splits a freeform place string into its components,
// outermost first ("Millbrook, Hartford Co., WI" → [WI, Hartford Co., Millbrook]).
func migrationFreeformPath(place string) []string {
	var path []string
	for _, part := range slices.Backward(strings.Split(place, ",")) {
		if p := strings.TrimSpace(part); p != "" {
			path = append(path, normalizeRegionName(p))
		}
	}

	return path
}

// placeDivergence compares two place paths. When one place contains the other
// (the same village, or a village and its district) there is no move;
// otherwise from and to are the two places' names at the first level they
// differ.
func placeDivergence(a, b []string) (from, to string, moved bool) {
	for i := range min(len(a), len(b)) {
		if !strings.EqualFold(a[i], b[i]) {
			return a[i], b[i], true
		}
	}

	return "", "", false
}

// placesCompatible reports whether two place paths can describe the same
// residence: they are equal or one contains the other.
func placesCompatible(a, b []string) bool {
	_, _, moved := placeDivergence(a, b)

	return !moved
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

// migrationPathTruncated reports whether a date-scoped ID path stops at a
// place that still has parents: the date matched more than one jurisdiction
// above it, so its top is not a region.
func migrationPathTruncated(ids []string, places map[string]*glxlib.Place) bool {
	if len(ids) == 0 {
		return false
	}
	root := places[ids[len(ids)-1]]

	return root != nil && (root.HasTemporalParent() || root.ParentID != "")
}

// migrationRegionAt takes the region from the unambiguous path when it is
// complete. When the date leaves the path open above some place, it uses the
// region all the possible paths share, else the region of the default
// hierarchy, never the truncated path's own top.
func migrationRegionAt(placeID string, date glxlib.DateString, ids []string, archive *glxlib.GLXFile) string {
	if !migrationPathTruncated(ids, archive.Places) {
		if path := migrationNamesForIDs(ids, archive.Places); len(path) > 0 {
			return path[0]
		}

		return ""
	}
	region := ""
	for i, path := range archive.PlacePathsAt(placeID, date) {
		if migrationPathTruncated(path, archive.Places) {
			region = ""

			break
		}
		names := migrationNamesForIDs(path, archive.Places)
		if len(names) == 0 || (i > 0 && !strings.EqualFold(names[0], region)) {
			region = ""

			break
		}
		region = names[0]
	}
	if region != "" {
		return region
	}

	return regionForPlace(placeID, archive.Places)
}

// alignMigrationIDs trims two most-specific-first ID paths to a shared top
// when either was cut short at an ambiguous date, so they compare from the
// same place downward. It reports false when neither top is in the other path.
func alignMigrationIDs(a, b []string) ([]string, []string, bool) {
	if len(a) == 0 || len(b) == 0 {
		return nil, nil, false
	}
	if i := slices.Index(b, a[len(a)-1]); i >= 0 {
		return a, b[:i+1], true
	}
	if i := slices.Index(a, b[len(b)-1]); i >= 0 {
		return a[:i+1], b, true
	}

	return nil, nil, false
}

// regionForPlace walks a place's parent hierarchy and returns the normalized
// name of the topmost ancestor below the country/continent level — the
// state/region level at which migration is meaningful (e.g. "Wisconsin" for
// Millbrook → Hartford Co. → Wisconsin → United States). Falls back to the
// topmost named ancestor when the whole chain is containers.
func regionForPlace(placeID string, places map[string]*glxlib.Place) string {
	return regionForPlaceAt(placeID, "", places)
}

// regionForPlaceAt is regionForPlace for the hierarchy as it stood at date,
// so a county that moved from a territory to a state (#225) is placed in the
// region it belonged to then.
func regionForPlaceAt(placeID string, date glxlib.DateString, places map[string]*glxlib.Place) string {
	path := migrationPlacePathAt(placeID, date, places)
	if len(path) != 0 {
		return path[0]
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
// place, label, and role/exclusion semantics), which arise when the same event is reachable both
// directly and through a relationship.
func dedupeMigrationEntries(entries []migrationEntry) []migrationEntry {
	type observationKey struct {
		date, placeID, place, label, role, note string
		excluded, location                      bool
	}
	seen := make(map[observationKey]bool)
	result := make([]migrationEntry, 0, len(entries))

	for i := range entries {
		e := &entries[i]
		key := observationKey{e.Date, e.PlaceID, e.Place, e.Label, e.Role, e.Note, e.Excluded, e.location}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, *e)
	}

	return result
}

// computeMovements scans the dated observations in chronological order and
// records each change of place. Consecutive observations of the same place,
// or of a place and one containing it, collapse; undated, placeless and
// excluded observations are skipped.
func computeMovements(entries []migrationEntry, archive *glxlib.GLXFile) []migrationMovement {
	var movements []migrationMovement
	walkMigrationStops(entries, archive, func(prev, entry *migrationEntry, from, to string) {
		if prev != nil {
			movements = append(movements, migrationMovement{
				FromRegion: from,
				ToRegion:   to,
				FromDate:   prev.Date,
				ToDate:     entry.Date,
			})
		}
	}, nil)

	return movements
}

// walkMigrationStops calls onStop for the first counted observation and for
// every observation that moves the person to a place the previous counted
// observations do not cover. prev is the last observation before the move
// (nil for the first stop) and from/to name the two places where they differ.
//
// Within a stop the most specific place seen is the one compared against, so
// "Wisconsin" after "Millbrook, Wisconsin" is no move, nor is a return to
// Millbrook after it. onRefine, when provided, receives an observation that
// makes the current stop more specific without adding a movement.
func walkMigrationStops(entries []migrationEntry, archive *glxlib.GLXFile, onStop func(prev, entry *migrationEntry, from, to string), onRefine func(entry *migrationEntry)) {
	var prev, anchor *migrationEntry
	for i := range entries {
		entry := &entries[i]
		if entry.sortKey == undatedMigrationSortKey || len(entry.placePath()) == 0 || entry.Excluded {
			continue
		}
		if anchor == nil {
			onStop(nil, entry, "", "")
			anchor, prev = entry, entry

			continue
		}
		from, to, moved, refine := compareMigrationStops(anchor, entry, archive)
		switch {
		case moved:
			onStop(prev, entry, from, to)
			anchor = entry
		case refine:
			anchor = entry
			if onRefine != nil {
				onRefine(entry)
			}
		case anchor.PlaceID != "" && anchor.PlaceID == entry.PlaceID:
			// Refresh the comparison label, not the first displayed stop.
			anchor = entry
		}
		prev = entry
	}
}

// compareMigrationStops keeps the most specific known ID even across a
// coarser observation. An observation of an enclosing place applies at its
// own date: a town does not move just because a boundary changed.
func compareMigrationStops(anchor, next *migrationEntry, archive *glxlib.GLXFile) (from, to string, moved, refine bool) {
	if anchor.PlaceID != "" && next.PlaceID != "" {
		if anchor.PlaceID == next.PlaceID || archive.PlaceMayHaveAncestorAt(anchor.PlaceID, next.PlaceID, glxlib.DateString(next.Date)) {
			return "", "", false, false
		}
		if archive.PlaceMayHaveAncestorAt(next.PlaceID, anchor.PlaceID, glxlib.DateString(anchor.Date)) {
			return "", "", false, true
		}
	}
	a, b := anchor.placePath(), next.placePath()
	if len(anchor.ids) > 0 && len(next.ids) > 0 &&
		(migrationPathTruncated(anchor.ids, archive.Places) || migrationPathTruncated(next.ids, archive.Places)) {
		// Paths of unequal depth from an ambiguous date are not aligned at
		// their tops; compare them from a place both contain instead.
		if ai, bi, ok := alignMigrationIDs(anchor.ids, next.ids); ok {
			a, b = migrationNamesForIDs(ai, archive.Places), migrationNamesForIDs(bi, archive.Places)
		}
	}
	from, to, moved = placeDivergence(a, b)

	return from, to, moved, !moved && len(b) > len(a)
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
		stops := migrationStops(collectMigrationEntries(personID, archive), archive)
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

// migrationStops collapses a person's dated observations into their place
// sequence: one stop per movement computeMovements would report, keeping the
// first observation at the most specific place in each run. Its date is the
// first observation of that place, rather than an earlier enclosing region.
func migrationStops(entries []migrationEntry, archive *glxlib.GLXFile) []migrationStop {
	var stops []migrationStop
	walkMigrationStops(entries, archive, func(_, entry *migrationEntry, _, _ string) {
		stops = append(stops, migrationStop{Region: entry.Region, Place: entry.Place, Date: entry.Date})
	}, func(entry *migrationEntry) {
		stops[len(stops)-1] = migrationStop{Region: entry.Region, Place: entry.Place, Date: entry.Date}
	})

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
	fmt.Fprintln(io.MachineOut, string(data))

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
	for i := range report.Entries {
		e := &report.Entries[i]
		if n := utf8.RuneCountInString(e.Place); n > placeWidth {
			placeWidth = n
		}
	}
	if placeWidth > migrationsPlaceColumnCap {
		placeWidth = migrationsPlaceColumnCap
	}

	var undated []migrationEntry
	for i := range report.Entries {
		e := &report.Entries[i]
		if e.sortKey == undatedMigrationSortKey {
			undated = append(undated, *e)

			continue
		}
		io.Println(formatMigrationEntryLine(e, placeWidth))
	}

	if len(undated) > 0 {
		io.Println("")
		io.Println("  Undated:")
		for i := range undated {
			e := &undated[i]
			io.Println(formatMigrationEntryLine(e, placeWidth))
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

// formatMigrationEntryLine renders one timeline row. An observation left out
// of movement detection carries the reason after its label.
func formatMigrationEntryLine(e *migrationEntry, placeWidth int) string {
	label := e.Label
	if e.visit && e.Role != "" {
		label += ", as " + e.Role
	}
	line := fmt.Sprintf("  %-18s  %s  (%s)", displayDate(e.Date), padPlaceColumn(e.Place, placeWidth), label)
	if e.Excluded {
		line += " [not counted as a move: " + e.Note + "]"
	}

	return line
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

// A parent at a child's birth/baptism remains evidence of the household's
// location. Other attendance roles place the visitor, not their residence.
func migrationParentAtBirth(event *glxlib.Event, personID string) bool {
	if !birthRecordEventTypes[strings.ToLower(strings.TrimSpace(event.Type))] {
		return false
	}
	for _, participant := range event.Participants {
		if participant.Person == personID && isParentRole(participant.Role) {
			return true
		}
	}

	return false
}
