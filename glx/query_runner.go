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
	}

	// Map each entity type to its supported flags.
	supported := map[string]map[string]bool{
		"persons":       {"--name": true, "--born-before": true, "--born-after": true},
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

		if opts.Name != "" && !nameMatches(person, opts.Name) {
			continue
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

		name := extractPersonName(person)
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
// Handles both simple string values and structured name objects.
func extractPersonName(person *glxlib.Person) string {
	raw, ok := person.Properties["name"]
	if !ok {
		raw, ok = person.Properties["primary_name"]
	}
	if !ok {
		return "(unnamed)"
	}

	// Simple string value
	if s, ok := raw.(string); ok {
		return s
	}

	// Structured: map with "value" key
	if m, ok := raw.(map[string]any); ok {
		if v, ok := m["value"]; ok {
			return fmt.Sprint(v)
		}
	}

	// Temporal list: []any where each entry has a "value" key
	if list, ok := raw.([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			if v, ok := m["value"]; ok {
				return fmt.Sprint(v)
			}
		}
	}

	return "(unnamed)"
}

// nameMatches checks if a person's name contains the query string (case-insensitive).
func nameMatches(person *glxlib.Person, query string) bool {
	name := extractPersonName(person)

	return containsFold(name, query)
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

// yearRegexp matches the first 4-digit year in a date string.
var yearRegexp = regexp.MustCompile(`\b(\d{4})\b`)

// extractDateYear extracts the first 4-digit year from a date string.
// Handles formats like "1850", "1850-01-15", "ABT 1850", "BET 1880 AND 1890".
func extractDateYear(dateStr string) int {
	match := yearRegexp.FindStringSubmatch(dateStr)
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
