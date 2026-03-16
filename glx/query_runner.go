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
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// queryOpts holds all filter options for the query command.
type queryOpts struct {
	Archive    string
	Name       string
	BornBefore int
	BornAfter  int
	Type       string
	Before     int
	After      int
	Confidence string
	Status     string
	Source     string
	Citation   string
	Birthplace string
}

// queryEntityTypes lists the entity types supported by the query command.
var queryEntityTypes = []string{
	"persons", "events", "assertions", "sources",
	"relationships", "places", "citations",
	"repositories", "media",
}

// validateQueryFlags checks that the given filter flags are applicable to the
// entity type and returns an error for any unsupported combination.
func validateQueryFlags(entityType string, opts queryOpts) error {
	type check struct {
		flag  string
		value bool
	}

	checks := []check{
		{"--name", opts.Name != ""},
		{"--born-before", opts.BornBefore != 0},
		{"--born-after", opts.BornAfter != 0},
		{"--type", opts.Type != ""},
		{"--before", opts.Before != 0},
		{"--after", opts.After != 0},
		{"--confidence", opts.Confidence != ""},
		{"--status", opts.Status != ""},
		{"--source", opts.Source != ""},
		{"--citation", opts.Citation != ""},
		{"--birthplace", opts.Birthplace != ""},
	}

	// Map each entity type to its supported flags.
	supported := map[string]map[string]bool{
		"persons":       {"--name": true, "--born-before": true, "--born-after": true, "--birthplace": true},
		"events":        {"--type": true, "--before": true, "--after": true},
		"assertions":    {"--confidence": true, "--status": true, "--source": true, "--citation": true},
		"sources":       {"--name": true, "--type": true},
		"relationships": {"--type": true},
		"places":        {"--name": true},
		"repositories":  {"--name": true},
		"citations":     {},
		"media":         {},
	}

	allowed := supported[entityType]
	for _, c := range checks {
		if c.value && !allowed[c.flag] {
			return fmt.Errorf("flag %s is not supported for entity type %q", c.flag, entityType)
		}
	}

	return nil
}

// queryEntities validates the entity type, loads the archive, and dispatches.
func queryEntities(entityType string, opts queryOpts) error {
	if !slices.Contains(queryEntityTypes, entityType) {
		return fmt.Errorf("unknown entity type: %s", entityType)
	}

	if err := validateQueryFlags(entityType, opts); err != nil {
		return err
	}

	archive, err := loadArchiveForQuery(opts.Archive)
	if err != nil {
		return err
	}

	// Pre-lowercase filter strings for case-insensitive comparisons
	opts.Name = strings.ToLower(opts.Name)

	switch entityType {
	case "persons":
		return queryPersons(archive, opts)
	case "events":
		return queryEvents(archive, opts)
	case "assertions":
		return queryAssertions(archive, opts)
	case "sources":
		return querySources(archive, opts)
	case "relationships":
		return queryRelationships(archive, opts)
	case "places":
		return queryPlaces(archive, opts)
	case "citations":
		return queryCitations(archive)
	case "repositories":
		return queryRepositories(archive, opts)
	case "media":
		return queryMedia(archive)
	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}
}

// loadArchiveForQuery loads an archive from a path (directory or single file).
func loadArchiveForQuery(path string) (*glxlib.GLXFile, error) {
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

// queryPersons filters and displays persons.
func queryPersons(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Persons)
	var count int

	for _, id := range ids {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		allNames := extractAllNames(person)

		if opts.Name != "" {
			matched := false
			for _, n := range allNames {
				if containsFold(n, opts.Name) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		if opts.Birthplace != "" {
			if !personMatchesBirthplace(person, opts.Birthplace, archive) {
				continue
			}
		}
		if opts.BornBefore > 0 || opts.BornAfter > 0 {
			year := extractPropertyYear(person.Properties, "born_on")
			if opts.BornBefore > 0 && (year == 0 || year >= opts.BornBefore) {
				continue
			}
			if opts.BornAfter > 0 && (year == 0 || year <= opts.BornAfter) {
				continue
			}
		}

		name := "(unnamed)"
		if len(allNames) > 0 {
			name = allNames[0]
		}
		bornOn := propertyString(person.Properties, "born_on")
		diedOn := propertyString(person.Properties, "died_on")

		detail := name
		switch {
		case bornOn != "" && diedOn != "":
			detail += fmt.Sprintf("  (%s – %s)", bornOn, diedOn)
		case bornOn != "":
			detail += fmt.Sprintf("  (b. %s)", bornOn)
		case diedOn != "":
			detail += fmt.Sprintf("  (d. %s)", diedOn)
		}

		// Show alternate names if present (deduplicated, excluding primary)
		seen := map[string]bool{name: true}
		var akaNames []string
		for _, n := range allNames {
			if n != "" && !seen[n] {
				seen[n] = true
				akaNames = append(akaNames, n)
			}
		}
		if len(akaNames) > 0 {
			detail += "  aka: " + strings.Join(akaNames, ", ")
		}

		fmt.Printf("  %s  %s\n", id, detail)
		count++
	}

	fmt.Printf("\n%d person(s) found\n", count)

	return nil
}

// queryEvents filters and displays events.
func queryEvents(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Events)
	var count int

	for _, id := range ids {
		event := archive.Events[id]

		if opts.Type != "" && !strings.EqualFold(event.Type, opts.Type) {
			continue
		}
		dateYear := extractDateYear(string(event.Date))
		if opts.Before > 0 && (dateYear == 0 || dateYear >= opts.Before) {
			continue
		}
		if opts.After > 0 && (dateYear == 0 || dateYear <= opts.After) {
			continue
		}

		date := string(event.Date)
		if date == "" {
			date = "(no date)"
		}
		fmt.Printf("  %s  %s  %s\n", id, event.Type, date)
		count++
	}

	fmt.Printf("\n%d event(s) found\n", count)

	return nil
}

// queryAssertions filters and displays assertions.
func queryAssertions(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Assertions)
	var count int

	for _, id := range ids {
		a := archive.Assertions[id]

		if opts.Confidence != "" && !strings.EqualFold(a.Confidence, opts.Confidence) {
			continue
		}
		if opts.Status != "" && !strings.EqualFold(a.Status, opts.Status) {
			continue
		}
		if opts.Source != "" && !assertionReferencesSource(a, archive, opts.Source) {
			continue
		}
		if opts.Citation != "" && !slices.Contains(a.Citations, opts.Citation) {
			continue
		}

		subject := a.Subject.ID()
		subjectType := a.Subject.Type()
		detail := subjectType + ":" + subject
		if a.Property != "" {
			detail += "  " + a.Property + "=" + a.Value
		} else if a.Participant != nil {
			detail += "  participant:" + a.Participant.Person
		}
		if a.Confidence != "" {
			detail += "  [" + a.Confidence + "]"
		}
		fmt.Printf("  %s  %s\n", id, detail)
		count++
	}

	fmt.Printf("\n%d assertion(s) found\n", count)

	return nil
}

// querySources filters and displays sources.
func querySources(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Sources)
	var count int

	for _, id := range ids {
		source := archive.Sources[id]

		if opts.Name != "" && !containsFold(source.Title, opts.Name) {
			continue
		}
		if opts.Type != "" && !strings.EqualFold(source.Type, opts.Type) {
			continue
		}

		fmt.Printf("  %s  %s\n", id, source.Title)
		count++
	}

	fmt.Printf("\n%d source(s) found\n", count)

	return nil
}

// queryRelationships filters and displays relationships.
func queryRelationships(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Relationships)
	var count int

	for _, id := range ids {
		rel := archive.Relationships[id]

		if opts.Type != "" && !strings.EqualFold(rel.Type, opts.Type) {
			continue
		}

		var participants []string
		for _, p := range rel.Participants {
			participants = append(participants, p.Person)
		}
		fmt.Printf("  %s  %s  [%s]\n", id, rel.Type, strings.Join(participants, ", "))
		count++
	}

	fmt.Printf("\n%d relationship(s) found\n", count)

	return nil
}

// queryPlaces filters and displays places.
func queryPlaces(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Places)
	var count int

	for _, id := range ids {
		place := archive.Places[id]

		if opts.Name != "" && !containsFold(place.Name, opts.Name) {
			continue
		}

		placeType := place.Type
		if placeType != "" {
			placeType = "  (" + placeType + ")"
		}
		fmt.Printf("  %s  %s%s\n", id, place.Name, placeType)
		count++
	}

	fmt.Printf("\n%d place(s) found\n", count)

	return nil
}

// queryCitations lists all citations.
func queryCitations(archive *glxlib.GLXFile) error {
	ids := sortedKeys(archive.Citations)

	for _, id := range ids {
		cit := archive.Citations[id]
		fmt.Printf("  %s  source:%s\n", id, cit.SourceID)
	}

	fmt.Printf("\n%d citation(s) found\n", len(ids))

	return nil
}

// queryRepositories filters and displays repositories.
func queryRepositories(archive *glxlib.GLXFile, opts queryOpts) error {
	ids := sortedKeys(archive.Repositories)
	var count int

	for _, id := range ids {
		repo := archive.Repositories[id]

		if opts.Name != "" && !containsFold(repo.Name, opts.Name) {
			continue
		}

		fmt.Printf("  %s  %s\n", id, repo.Name)
		count++
	}

	fmt.Printf("\n%d repository(ies) found\n", count)

	return nil
}

// queryMedia lists all media.
func queryMedia(archive *glxlib.GLXFile) error {
	ids := sortedKeys(archive.Media)

	for _, id := range ids {
		m := archive.Media[id]
		title := m.Title
		if title == "" {
			title = m.URI
		}
		fmt.Printf("  %s  %s\n", id, title)
	}

	fmt.Printf("\n%d media found\n", len(ids))

	return nil
}

// assertionReferencesSource checks if an assertion references a source, either
// directly via its Sources list or indirectly via a citation whose SourceID matches.
func assertionReferencesSource(a *glxlib.Assertion, archive *glxlib.GLXFile, sourceID string) bool {
	// Check direct source references
	if slices.Contains(a.Sources, sourceID) {
		return true
	}

	// Check indirect references via citations
	for _, citID := range a.Citations {
		if cit, ok := archive.Citations[citID]; ok && cit.SourceID == sourceID {
			return true
		}
	}

	return false
}

// ============================================================================
// Helper functions
// ============================================================================

// extractPersonName extracts a display name from person properties.
// Handles simple strings, structured maps, and temporal lists.
// Delegates to extractAllNames and returns the first entry.
func extractPersonName(person *glxlib.Person) string {
	names := extractAllNames(person)
	if len(names) == 0 {
		return "(unnamed)"
	}

	return names[0]
}

// extractAllNames returns all name variants for a person.
// Handles simple strings, structured maps, and temporal lists.
// Falls back to primary_name if name is missing or yields no usable entries.
func extractAllNames(person *glxlib.Person) []string {
	if names := extractNamesFromProperty(person.Properties, "name"); len(names) > 0 {
		return names
	}
	return extractNamesFromProperty(person.Properties, "primary_name")
}

// extractNamesFromProperty extracts name strings from a property value.
func extractNamesFromProperty(props map[string]any, key string) []string {
	raw, ok := props[key]
	if !ok {
		return nil
	}

	// Simple string value
	if s, ok := raw.(string); ok {
		if s == "" {
			return nil
		}
		return []string{s}
	}

	// Structured: map with "value" key (single name entry)
	if m, ok := raw.(map[string]any); ok {
		if v, ok := m["value"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return []string{s}
			}
		}
		return nil
	}

	// Temporal list: []any where each entry has a "value" key
	if list, ok := raw.([]any); ok {
		var names []string
		for _, entry := range list {
			if m, ok := entry.(map[string]any); ok {
				if v, ok := m["value"]; ok {
					if s, ok := v.(string); ok && s != "" {
						names = append(names, s)
					}
				}
			}
		}
		return names
	}

	return nil
}

// propertyString extracts a simple string value from properties.
func propertyString(props map[string]any, key string) string {
	raw, ok := props[key]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}

	return fmt.Sprint(raw)
}

// extractPropertyYear extracts the year from a date-valued property.
func extractPropertyYear(props map[string]any, key string) int {
	s := propertyString(props, key)
	if s == "" {
		return 0
	}

	return extractDateYear(s)
}

// queryDayMonthRegexp matches day-of-month followed by a month abbreviation
// (e.g., "15 MAR"). Used to strip day values before year extraction so that
// 1–2 digit days are not mistaken for 1–2 digit years.
var queryDayMonthRegexp = regexp.MustCompile(`(?i)\b\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\b`)

// yearRegexp matches the first 1–4 digit year in a date string.
var yearRegexp = regexp.MustCompile(`\b(\d{1,4})\b`)

// extractDateYear extracts the first year (1–4 digits) from a date string.
// Handles formats like "1850", "1850-01-15", "ABT 1850", "BET 1880 AND 1890",
// "800", "ABT 476". Day-of-month values are stripped first.
func extractDateYear(dateStr string) int {
	cleaned := queryDayMonthRegexp.ReplaceAllString(dateStr, "")

	match := yearRegexp.FindStringSubmatch(cleaned)
	if len(match) < 2 {
		return 0
	}

	year, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return year
}

// containsFold checks if s contains lowerSubstr (case-insensitive).
// lowerSubstr must already be in lowercase; callers should pre-lowercase the needle.
// personMatchesBirthplace checks if a person's born_at property matches the query.
// Handles string, structured map ({value: ...}), and temporal list property shapes.
// Matches against both the place ID and resolved place name (case-insensitive substring).
func personMatchesBirthplace(person *glxlib.Person, query string, archive *glxlib.GLXFile) bool {
	if person == nil || person.Properties == nil {
		return false
	}
	raw := person.Properties["born_at"]
	if raw == nil {
		return false
	}
	lowerQuery := strings.ToLower(query)

	// Extract place ref(s) from the property value
	var refs []string
	switch v := raw.(type) {
	case string:
		refs = append(refs, v)
	case map[string]any:
		if val, ok := v["value"].(string); ok {
			refs = append(refs, val)
		}
	case []any:
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				if val, ok := m["value"].(string); ok {
					refs = append(refs, val)
				}
			} else if s, ok := item.(string); ok {
				refs = append(refs, s)
			}
		}
	}

	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if containsFold(ref, lowerQuery) {
			return true
		}
		if place, ok := archive.Places[ref]; ok && place != nil {
			if containsFold(place.Name, lowerQuery) {
				return true
			}
		}
	}
	return false
}

// containsFold returns true if s contains lowerSubstr (case-insensitive).
func containsFold(s, lowerSubstr string) bool {
	return strings.Contains(strings.ToLower(s), lowerSubstr)
}

// sortedKeys returns map keys sorted alphabetically.
func sortedKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}
