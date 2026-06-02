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
	"slices"
	"sort"
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Search-result kind values, recorded in the client-side search index.
const (
	searchKindPerson = "person"
	searchKindSource = "source"
	searchKindPlace  = "place"
)

// viewModelOptions controls how the presentation model is built from an archive.
type viewModelOptions struct {
	Title     string // site title shown in the header and <title>
	Generated string // pre-formatted generation date; "" omits the footer stamp
}

// siteModel is the fully-resolved, presentation-ready model for the whole
// static site. It is built once from a loaded archive (no I/O) and then handed
// to the renderer. All cross-references are resolved to display strings and
// links so the templates stay logic-free.
type siteModel struct {
	Title     string
	Generated string
	Stats     siteStats
	Persons   []*personPage
	Sources   []sourceRow
	Places    []placeRow
	Search    []searchEntry
}

// HasMedia reports whether any person page references media, used to decide
// whether the media directory needs to be created/populated.
func (s *siteModel) HasMedia() bool {
	for _, p := range s.Persons {
		if len(p.Media) > 0 {
			return true
		}
	}

	return false
}

// siteStats holds entity counts for the landing page.
type siteStats struct {
	Persons       int
	Events        int
	Places        int
	Sources       int
	Citations     int
	Relationships int
	Media         int
}

// personLink is a minimal person reference used for navigation between pages.
type personLink struct {
	ID    string
	Name  string
	Dates string // e.g. "1850–1920", "b. 1850", or "" when unknown
}

// personPage is the presentation model for a single person profile page.
type personPage struct {
	ID         string
	Name       string
	SortName   string // surname-first key for alphabetical ordering
	AltNames   []string
	Sex        string
	Gender     string
	LifeSpan   string // "1850–1920", "b. 1850", "d. 1920", or ""
	Living     bool
	BirthDate  string
	BirthPlace string
	DeathDate  string
	DeathPlace string
	Timeline   []timelineRow
	Parents    []personLink
	Spouses    []personLink
	Children   []personLink
	Siblings   []personLink
	Notes      []string
	Sources    []personSourceRef
	Media      []*mediaItem
}

// timelineRow is a single chronological entry on a person page.
type timelineRow struct {
	Date    string
	Label   string
	Detail  string
	Undated bool
}

// personSourceRef is a source/citation supporting a person, resolved to
// display strings for the "Sources" section of a profile.
type personSourceRef struct {
	Title      string
	Detail     string // citation locator or quoted text from the source
	URL        string
	Confidence string
	Status     string
}

// sourceRow is a row in the global source index.
type sourceRow struct {
	ID         string
	Title      string
	Type       string
	Authors    string
	Date       string
	Repository string
	URL        string
}

// placeRow is a row in the global place index.
type placeRow struct {
	ID         string
	Name       string
	FullName   string // hierarchical "Boston, Massachusetts, United States"
	Type       string
	HasCoords  bool
	MapURL     string // OpenStreetMap link when coordinates are present
	EventCount int
}

// searchEntry is one record in the client-side search index (search-index.json).
type searchEntry struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Kind string `json:"kind"` // "person", "source", or "place"
	Sub  string `json:"sub,omitempty"`
}

// mediaItem is a media reference attached to a person. Src/Embedded/Missing are
// filled in by the media-resolution stage (view_media.go); the model build only
// records the raw reference and classification.
type mediaItem struct {
	MediaID  string
	RawURI   string
	Title    string
	Caption  string
	MimeType string
	IsImage  bool
	IsURL    bool // RawURI is an http(s) URL — link/embed directly, never copied

	// Resolved by the media stage. Because media only renders on person
	// pages (one directory deep), copied-file sources already carry the
	// "../media/" prefix, so templates use Src verbatim.
	Src     string // "../media/foo.jpg", a data: URI, or an external URL
	Missing bool   // local file could not be found/read
}

// buildSiteModel turns a loaded archive into a fully-resolved presentation
// model. It performs no I/O — media files are only classified here and resolved
// to concrete sources later by the media stage.
func buildSiteModel(archive *glxlib.GLXFile, opts viewModelOptions) *siteModel {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "GLX Family Archive"
	}

	model := &siteModel{
		Title:     title,
		Generated: opts.Generated,
		Stats:     buildStats(archive),
	}

	idx := newViewIndex(archive)

	model.Persons = buildPersonPages(archive, idx)
	model.Sources = buildSourceRows(archive)
	model.Places = buildPlaceRows(archive, idx)
	model.Search = buildSearchIndex(model)

	return model
}

// buildStats counts entities for the landing page.
func buildStats(archive *glxlib.GLXFile) siteStats {
	return siteStats{
		Persons:       len(archive.Persons),
		Events:        len(archive.Events),
		Places:        len(archive.Places),
		Sources:       len(archive.Sources),
		Citations:     len(archive.Citations),
		Relationships: len(archive.Relationships),
		Media:         len(archive.Media),
	}
}

// viewIndex holds derived lookups computed once and shared across page builds.
type viewIndex struct {
	archive  *glxlib.GLXFile
	parents  map[string][]string // child ID -> parent IDs
	children map[string][]string // parent ID -> child IDs
	spouses  map[string][]string // person ID -> spouse IDs
	// assertionsByPerson maps a person ID to assertions that reference the
	// person as subject or as the asserted participant.
	assertionsByPerson map[string][]*glxlib.Assertion
	// eventsByPlace counts events referencing each place ID.
	eventsByPlace map[string]int
}

// newViewIndex precomputes relationship, assertion, and place lookups.
func newViewIndex(archive *glxlib.GLXFile) *viewIndex {
	idx := &viewIndex{
		archive:            archive,
		parents:            map[string][]string{},
		children:           map[string][]string{},
		spouses:            map[string][]string{},
		assertionsByPerson: map[string][]*glxlib.Assertion{},
		eventsByPlace:      map[string]int{},
	}

	for _, relID := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[relID]
		idx.indexRelationship(rel)
	}

	for _, aID := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[aID]
		idx.indexAssertion(a)
	}

	for _, evID := range sortedKeys(archive.Events) {
		ev := archive.Events[evID]
		if ev.PlaceID != "" {
			idx.eventsByPlace[ev.PlaceID]++
		}
	}

	return idx
}

// indexRelationship records parent/child and spouse edges for one relationship.
func (idx *viewIndex) indexRelationship(rel *glxlib.Relationship) {
	switch {
	case isParentChildType(rel.Type):
		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case glxlib.ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case glxlib.ParticipantRoleChild:
				childIDs = append(childIDs, p.Person)
			}
		}
		for _, c := range childIDs {
			for _, par := range parentIDs {
				idx.parents[c] = appendUnique(idx.parents[c], par)
				idx.children[par] = appendUnique(idx.children[par], c)
			}
		}
	case isMarriageType(rel.Type):
		var people []string
		for _, p := range rel.Participants {
			if p.Person != "" {
				people = append(people, p.Person)
			}
		}
		for _, a := range people {
			for _, b := range people {
				if a != b {
					idx.spouses[a] = appendUnique(idx.spouses[a], b)
				}
			}
		}
	}
}

// indexAssertion records which persons an assertion references.
func (idx *viewIndex) indexAssertion(a *glxlib.Assertion) {
	seen := map[string]bool{}
	add := func(personID string) {
		if personID == "" || seen[personID] {
			return
		}
		seen[personID] = true
		idx.assertionsByPerson[personID] = append(idx.assertionsByPerson[personID], a)
	}

	if a.Subject.Person != "" {
		add(a.Subject.Person)
	}
	if a.Participant != nil {
		add(a.Participant.Person)
	}
}

// buildPersonPages builds and alphabetically sorts every person profile page.
func buildPersonPages(archive *glxlib.GLXFile, idx *viewIndex) []*personPage {
	pages := make([]*personPage, 0, len(archive.Persons))
	for _, id := range sortedKeys(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}
		pages = append(pages, buildPersonPage(id, person, archive, idx))
	}

	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].SortName != pages[j].SortName {
			return pages[i].SortName < pages[j].SortName
		}

		return pages[i].ID < pages[j].ID
	})

	return pages
}

// buildPersonPage resolves a single person into a presentation model.
func buildPersonPage(id string, person *glxlib.Person, archive *glxlib.GLXFile, idx *viewIndex) *personPage {
	names := extractAllNames(person)
	name := extractPersonName(person) // owns the "(unnamed)" fallback
	var alt []string
	if len(names) > 1 {
		alt = names[1:]
	}

	page := &personPage{
		ID:       id,
		Name:     name,
		SortName: personSortName(name),
		AltNames: alt,
		Sex:      personSex(person),
		Gender:   displayableGenderIdentity(person),
		Living:   isLivingProperty(person),
		Notes:    append([]string(nil), person.Notes...),
	}

	birth, death := vitalEvents(id, archive)
	if birth != nil {
		page.BirthDate = displayDateOrBlank(string(birth.Date))
		page.BirthPlace = placeFullName(birth.PlaceID, archive)
	}
	if death != nil {
		page.DeathDate = displayDateOrBlank(string(death.Date))
		page.DeathPlace = placeFullName(death.PlaceID, archive)
	}
	page.LifeSpan = lifeSpan(birth, death)

	page.Timeline = buildTimelineRows(id, archive)
	page.Parents = personLinks(idx.parents[id], archive)
	page.Spouses = personLinks(idx.spouses[id], archive)
	page.Children = personLinks(idx.children[id], archive)
	page.Siblings = personLinks(siblingIDs(id, idx), archive)
	page.Sources = buildPersonSources(id, archive, idx)
	page.Media = buildPersonMedia(id, archive, idx)

	return page
}

// buildTimelineRows reuses the shared timeline collector and converts entries
// into presentation rows.
func buildTimelineRows(personID string, archive *glxlib.GLXFile) []timelineRow {
	entries := collectTimelineEntries(personID, archive, true)
	rows := make([]timelineRow, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, timelineRow{
			Date:    displayDate(e.Date),
			Label:   e.Label,
			Detail:  e.Detail,
			Undated: e.SortKey == "\xff",
		})
	}

	return rows
}

// buildPersonSources collects the sources/citations supporting a person via
// the assertions that reference them.
func buildPersonSources(personID string, archive *glxlib.GLXFile, idx *viewIndex) []personSourceRef {
	var refs []personSourceRef
	seen := map[string]bool{}

	for _, a := range idx.assertionsByPerson[personID] {
		for _, srcID := range a.Sources {
			ref, key := sourceRefFromSource(srcID, a, archive)
			if ref.Title != "" && !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
		for _, citID := range a.Citations {
			ref, key := sourceRefFromCitation(citID, a, archive)
			if ref.Title != "" && !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}

	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Title != refs[j].Title {
			return refs[i].Title < refs[j].Title
		}

		return refs[i].Detail < refs[j].Detail
	})

	return refs
}

// sourceRefFromSource builds a person source ref from a direct source reference.
func sourceRefFromSource(srcID string, a *glxlib.Assertion, archive *glxlib.GLXFile) (personSourceRef, string) {
	src, ok := archive.Sources[srcID]
	if !ok || src == nil {
		return personSourceRef{}, ""
	}

	return personSourceRef{
		Title:      src.Title,
		URL:        propertyScalar(src.Properties["url"]),
		Confidence: a.Confidence,
		Status:     a.Status,
	}, srcID + "|"
}

// sourceRefFromCitation builds a person source ref from a citation reference,
// resolving the citation's parent source and surfacing its locator/quote.
func sourceRefFromCitation(citID string, a *glxlib.Assertion, archive *glxlib.GLXFile) (personSourceRef, string) {
	cit, ok := archive.Citations[citID]
	if !ok || cit == nil {
		return personSourceRef{}, ""
	}

	title := citID
	if src, ok := archive.Sources[cit.SourceID]; ok && src != nil && src.Title != "" {
		title = src.Title
	}

	detail := propertyScalar(cit.Properties["locator"])
	if quote := propertyScalar(cit.Properties["text_from_source"]); quote != "" {
		if detail != "" {
			detail += " — "
		}
		detail += "“" + quote + "”"
	}

	url := propertyScalar(cit.Properties["url"])

	return personSourceRef{
		Title:      title,
		Detail:     detail,
		URL:        url,
		Confidence: a.Confidence,
		Status:     a.Status,
	}, citID + "|" + detail
}

// buildSourceRows builds the global source index, ordered by title then ID.
func buildSourceRows(archive *glxlib.GLXFile) []sourceRow {
	rows := make([]sourceRow, 0, len(archive.Sources))
	for _, id := range sortedKeys(archive.Sources) {
		src := archive.Sources[id]
		if src == nil {
			continue
		}
		repo := ""
		if r, ok := archive.Repositories[src.RepositoryID]; ok && r != nil {
			repo = r.Name
		}
		rows = append(rows, sourceRow{
			ID:         id,
			Title:      src.Title,
			Type:       humanizeToken(src.Type),
			Authors:    strings.Join(src.Authors, ", "),
			Date:       displayDateOrBlank(string(src.Date)),
			Repository: repo,
			URL:        propertyScalar(src.Properties["url"]),
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Title != rows[j].Title {
			return rows[i].Title < rows[j].Title
		}

		return rows[i].ID < rows[j].ID
	})

	return rows
}

// buildPlaceRows builds the global place index, ordered by full hierarchical name.
func buildPlaceRows(archive *glxlib.GLXFile, idx *viewIndex) []placeRow {
	rows := make([]placeRow, 0, len(archive.Places))
	for _, id := range sortedKeys(archive.Places) {
		place := archive.Places[id]
		if place == nil {
			continue
		}
		row := placeRow{
			ID:         id,
			Name:       place.Name,
			FullName:   placeFullName(id, archive),
			Type:       humanizeToken(place.Type),
			EventCount: idx.eventsByPlace[id],
		}
		if place.Latitude != nil && place.Longitude != nil {
			row.HasCoords = true
			row.MapURL = openStreetMapURL(*place.Latitude, *place.Longitude)
		}
		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].FullName != rows[j].FullName {
			return rows[i].FullName < rows[j].FullName
		}

		return rows[i].ID < rows[j].ID
	})

	return rows
}

// buildSearchIndex assembles the client-side search records from the model.
func buildSearchIndex(model *siteModel) []searchEntry {
	entries := make([]searchEntry, 0, len(model.Persons)+len(model.Sources)+len(model.Places))

	for _, p := range model.Persons {
		entries = append(entries, searchEntry{
			Name: p.Name,
			URL:  "persons/" + personFileName(p.ID),
			Kind: searchKindPerson,
			Sub:  p.LifeSpan,
		})
	}
	for _, s := range model.Sources {
		entries = append(entries, searchEntry{
			Name: s.Title,
			URL:  "sources/index.html#" + s.ID,
			Kind: searchKindSource,
			Sub:  s.Type,
		})
	}
	for _, pl := range model.Places {
		entries = append(entries, searchEntry{
			Name: pl.FullName,
			URL:  "places/index.html#" + pl.ID,
			Kind: searchKindPlace,
			Sub:  pl.Type,
		})
	}

	return entries
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// personLinks resolves a list of person IDs to navigation links, sorted by name.
func personLinks(ids []string, archive *glxlib.GLXFile) []personLink {
	if len(ids) == 0 {
		return nil
	}
	links := make([]personLink, 0, len(ids))
	for _, id := range ids {
		person, ok := archive.Persons[id]
		if !ok || person == nil {
			continue
		}
		birth, death := vitalEvents(id, archive)
		links = append(links, personLink{
			ID:    id,
			Name:  extractPersonName(person),
			Dates: lifeSpan(birth, death),
		})
	}

	sort.SliceStable(links, func(i, j int) bool {
		if links[i].Name != links[j].Name {
			return links[i].Name < links[j].Name
		}

		return links[i].ID < links[j].ID
	})

	return links
}

// siblingIDs returns the other children of this person's parents, excluding self.
func siblingIDs(personID string, idx *viewIndex) []string {
	var out []string
	seen := map[string]bool{personID: true}
	for _, parentID := range idx.parents[personID] {
		for _, childID := range idx.children[parentID] {
			if !seen[childID] {
				seen[childID] = true
				out = append(out, childID)
			}
		}
	}

	return out
}

// vitalEvents returns the person's birth and death events, if present. The
// first matching event of each type (in sorted ID order) is used.
func vitalEvents(personID string, archive *glxlib.GLXFile) (birth, death *glxlib.Event) {
	for _, id := range sortedKeys(archive.Events) {
		ev := archive.Events[id]
		if !timelineIsParticipant(personID, ev) {
			continue
		}
		switch strings.ToLower(ev.Type) {
		case glxlib.EventTypeBirth:
			if birth == nil {
				birth = ev
			}
		case glxlib.EventTypeDeath:
			if death == nil {
				death = ev
			}
		}
	}

	return birth, death
}

// lifeSpan renders a compact life span from birth/death events.
func lifeSpan(birth, death *glxlib.Event) string {
	var b, d string
	if birth != nil {
		b = eventYear(string(birth.Date))
	}
	if death != nil {
		d = eventYear(string(death.Date))
	}
	switch {
	case b != "" && d != "":
		return b + "–" + d
	case b != "":
		return "b. " + b
	case d != "":
		return "d. " + d
	default:
		return ""
	}
}

// eventYear extracts a 4-digit year string from a GLX date, or "" if unparseable.
func eventYear(date string) string {
	key := dateSortKey(date)
	if key == "\xff" {
		return ""
	}
	if i := strings.Index(key, "-"); i >= 0 {
		key = key[:i]
	}
	// Strip zero-padding added by dateSortKey, keeping at least one digit.
	trimmed := strings.TrimLeft(key, "0")
	if trimmed == "" {
		return "0"
	}

	return trimmed
}

// placeFullName builds a hierarchical place name by walking parent places,
// e.g. "Boston, Massachusetts, United States". Falls back to the place ID when
// the place has no name, and guards against cyclic parent references.
func placeFullName(placeID string, archive *glxlib.GLXFile) string {
	if placeID == "" {
		return ""
	}
	var parts []string
	seen := map[string]bool{}
	current := placeID
	for current != "" && !seen[current] {
		seen[current] = true
		place, ok := archive.Places[current]
		if !ok || place == nil {
			parts = append(parts, current)

			break
		}
		name := place.Name
		if name == "" {
			name = current
		}
		parts = append(parts, name)
		current = place.ParentID
	}

	return strings.Join(parts, ", ")
}

// personSortName produces a "Surname, Given" style key for alphabetical
// ordering, falling back to the full name when no surname can be inferred.
func personSortName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "￿" // unnamed sorts last
	}
	fields := strings.Fields(name)
	if len(fields) < 2 {
		return strings.ToLower(name)
	}
	surname := fields[len(fields)-1]
	rest := strings.Join(fields[:len(fields)-1], " ")

	return strings.ToLower(surname + ", " + rest)
}

// isLivingProperty reports whether the person carries a truthy `living` property.
func isLivingProperty(person *glxlib.Person) bool {
	switch v := person.Properties[glxlib.PersonPropertyLiving].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

// humanizeToken converts a vocabulary token like "vital_record" to "Vital record".
func humanizeToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	spaced := strings.ReplaceAll(token, "_", " ")

	return strings.ToUpper(spaced[:1]) + spaced[1:]
}

// displayDateOrBlank formats a date for display but returns "" (not "(no date)")
// for empty input, so index rows can omit the column cleanly.
func displayDateOrBlank(date string) string {
	if strings.TrimSpace(date) == "" {
		return ""
	}

	return displayDate(date)
}

// openStreetMapURL builds a marker link to OpenStreetMap for given coordinates.
func openStreetMapURL(lat, lng float64) string {
	latStr := strconv.FormatFloat(lat, 'f', -1, 64)
	lngStr := strconv.FormatFloat(lng, 'f', -1, 64)

	return fmt.Sprintf("https://www.openstreetmap.org/?mlat=%s&mlon=%s#map=12/%s/%s",
		latStr, lngStr, latStr, lngStr)
}

// appendUnique appends id to slice only if not already present.
func appendUnique(slice []string, id string) []string {
	if slices.Contains(slice, id) {
		return slice
	}

	return append(slice, id)
}
