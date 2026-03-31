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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// timelineEntry represents a single row in the timeline output.
type timelineEntry struct {
	Date     string // Raw date string for display
	SortKey  string // Normalized string for chronological sorting
	Label    string // e.g., "Birth", "Marriage", "Birth of child (Jane)"
	Detail   string // Place or other context
	EventID  string // For deduplication
	IsFamily bool   // Whether this came from relationship traversal
}

// familyRelationshipTypes defines which relationship types count as "family"
// for the purpose of timeline event collection.
var familyRelationshipTypes = map[string]bool{
	"marriage":                  true,
	"partner":                   true,
	"parent_child":              true,
	"biological_parent_child":   true,
	"adoptive_parent_child":     true,
	"foster_parent_child":       true,
	"step_parent":               true,
	"guardian":                   true,
}

// showTimeline loads an archive and displays a chronological timeline for a person.
func showTimeline(archivePath, personQuery string, includeFamily bool) error {
	archive, err := loadArchiveForTimeline(archivePath)
	if err != nil {
		return err
	}

	personID, _, err := findPersonForTimeline(archive, personQuery)
	if err != nil {
		return err
	}

	entries := collectTimelineEntries(personID, archive, includeFamily)
	printTimeline(personID, extractPersonName(archive.Persons[personID]), entries)

	return nil
}

// loadArchiveForTimeline loads an archive from a path (directory or single file).
func loadArchiveForTimeline(path string) (*glxlib.GLXFile, error) {
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

// findPersonForTimeline looks up a person by exact ID or name substring.
func findPersonForTimeline(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	// Exact ID match
	if person, ok := archive.Persons[query]; ok {
		if person == nil {
			return "", nil, fmt.Errorf("person %q exists in archive but has no data", query)
		}
		return query, person, nil
	}

	// Name substring search (case-insensitive)
	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		name := extractPersonName(person)
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			matches = append(matches, id)
		}
	}

	sort.Strings(matches)

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

// collectTimelineEntries gathers all timeline events for a person.
func collectTimelineEntries(personID string, archive *glxlib.GLXFile, includeFamily bool) []timelineEntry {
	direct := collectDirectEvents(personID, archive)

	var family []timelineEntry
	if includeFamily {
		family = collectFamilyEvents(personID, archive)
	}

	entries := deduplicateEntries(direct, family)
	sortTimelineEntries(entries)

	return entries
}

// collectDirectEvents finds all events where the person is a participant.
func collectDirectEvents(personID string, archive *glxlib.GLXFile) []timelineEntry {
	var entries []timelineEntry

	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]

		if !timelineIsParticipant(personID, event) {
			continue
		}

		date := string(event.Date)
		label := formatEventTypeLabel(event.Type)
		detail := timelineResolvePlaceName(event.PlaceID, archive)

		entries = append(entries, timelineEntry{
			Date:     date,
			SortKey:  dateSortKey(date),
			Label:    label,
			Detail:   detail,
			EventID:  id,
			IsFamily: false,
		})
	}

	return entries
}

// collectFamilyEvents finds events for related persons via relationship traversal.
func collectFamilyEvents(personID string, archive *glxlib.GLXFile) []timelineEntry {
	var entries []timelineEntry

	related := findRelatedPersons(personID, archive)
	for _, rel := range related {
		relEntries := relatedPersonTimelineEvents(rel, archive)
		entries = append(entries, relEntries...)
	}

	return entries
}

// relatedPerson describes a person related to the target through a relationship.
type relatedPerson struct {
	PersonID string // The related person's ID
	Name     string // Display name
	Relation string // "spouse", "child", "parent"
}

// findRelatedPersons traverses relationships to find family members.
func findRelatedPersons(personID string, archive *glxlib.GLXFile) []relatedPerson {
	var related []relatedPerson
	seen := make(map[string]bool) // Avoid duplicate related persons

	relIDs := sortedKeys(archive.Relationships)
	for _, relID := range relIDs {
		rel := archive.Relationships[relID]
		if !familyRelationshipTypes[rel.Type] {
			continue
		}

		// Check if target person is a participant
		targetIdx := -1
		for i, p := range rel.Participants {
			if p.Person == personID {
				targetIdx = i

				break
			}
		}
		if targetIdx < 0 {
			continue
		}

		targetRole := rel.Participants[targetIdx].Role

		// Determine relation type based on relationship type and target's role
		for i, p := range rel.Participants {
			if i == targetIdx || p.Person == "" {
				continue
			}
			if seen[p.Person] {
				continue
			}

			relation := inferRelation(rel.Type, targetRole, p.Role)
			if relation == "" {
				continue
			}

			name := "(unknown)"
			if person, ok := archive.Persons[p.Person]; ok && person != nil {
				name = extractPersonName(person)
			}

			seen[p.Person] = true
			related = append(related, relatedPerson{
				PersonID: p.Person,
				Name:     name,
				Relation: relation,
			})
		}
	}

	return related
}

// inferRelation determines the family relationship label based on relationship type and roles.
func inferRelation(relType, targetRole, otherRole string) string {
	switch {
	case isMarriageType(relType):
		return "spouse"
	case isParentChildType(relType):
		switch {
		case targetRole == "parent":
			return "child"
		case targetRole == "child":
			return "parent"
		case otherRole == "parent":
			return "parent"
		case otherRole == "child":
			return "child"
		default:
			// Ambiguous — skip
			return ""
		}
	case relType == "guardian":
		// In standard archives, guardians use the "parent" participant role.
		if targetRole == "parent" {
			return "child"
		}

		return "parent"
	default:
		return ""
	}
}

func isMarriageType(relType string) bool {
	return relType == "marriage" || relType == "partner"
}

func isParentChildType(relType string) bool {
	switch relType {
	case "parent_child", "biological_parent_child", "adoptive_parent_child",
		"foster_parent_child", "step_parent":
		return true
	}

	return false
}

// relatedPersonTimelineEvents selects which events to include for a related person.
func relatedPersonTimelineEvents(rel relatedPerson, archive *glxlib.GLXFile) []timelineEntry {
	var entries []timelineEntry

	// Determine which event types to include based on the relation
	var includeTypes map[string]string // event type -> label template
	switch rel.Relation {
	case "spouse":
		includeTypes = map[string]string{
			"birth": "Birth of spouse (%s)",
			"death": "Death of spouse (%s)",
		}
	case "child":
		includeTypes = map[string]string{
			"birth": "Birth of child (%s)",
			"death": "Death of child (%s)",
		}
	case "parent":
		includeTypes = map[string]string{
			"death": "Death of parent (%s)",
		}
	default:
		return nil
	}

	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]
		if !timelineIsParticipant(rel.PersonID, event) {
			continue
		}

		eventType := strings.ToLower(event.Type)
		labelTemplate, ok := includeTypes[eventType]
		if !ok {
			continue
		}

		date := string(event.Date)
		label := fmt.Sprintf(labelTemplate, rel.Name)
		detail := timelineResolvePlaceName(event.PlaceID, archive)

		entries = append(entries, timelineEntry{
			Date:     date,
			SortKey:  dateSortKey(date),
			Label:    label,
			Detail:   detail,
			EventID:  id,
			IsFamily: true,
		})
	}

	return entries
}

// deduplicateEntries merges direct and family entries, preferring direct over family
// when both reference the same event.
func deduplicateEntries(direct, family []timelineEntry) []timelineEntry {
	seen := make(map[string]bool)
	result := make([]timelineEntry, 0, len(direct)+len(family))

	for _, e := range direct {
		if e.EventID != "" {
			seen[e.EventID] = true
		}
		result = append(result, e)
	}

	for _, e := range family {
		if e.EventID != "" && seen[e.EventID] {
			continue
		}
		result = append(result, e)
	}

	return result
}

// sortTimelineEntries sorts entries chronologically, with undated entries last.
func sortTimelineEntries(entries []timelineEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].SortKey < entries[j].SortKey
	})
}

// timelineDayMonthRegexp matches day-of-month followed by a month abbreviation
// (e.g., "15 MAR"). Used to strip day values before year extraction.
var timelineDayMonthRegexp = regexp.MustCompile(`(?i)\b\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\b`)

// dateSortKeyRegexp matches a 1–4 digit year optionally followed by -MM and -DD,
// using word boundaries to avoid partial matches inside longer digit sequences.
var dateSortKeyRegexp = regexp.MustCompile(`\b(\d{1,4}(?:-\d{2}(?:-\d{2})?)?)\b`)

// dateSortKey extracts a sortable date string from a GLX date value.
// Strips qualifiers like ABT, BEF, AFT, BET...AND and day-of-month values.
// Years are zero-padded to 4 digits for correct chronological sorting.
// Returns "\xff" for empty/unparseable dates (sorts last).
func dateSortKey(dateStr string) string {
	if dateStr == "" {
		return "\xff"
	}

	cleaned := timelineDayMonthRegexp.ReplaceAllString(dateStr, "")

	match := dateSortKeyRegexp.FindString(cleaned)
	if match == "" {
		return "\xff"
	}

	// Zero-pad the year portion to 4 digits for proper string sorting.
	if idx := strings.Index(match, "-"); idx >= 0 {
		year := match[:idx]
		rest := match[idx:]
		for len(year) < 4 {
			year = "0" + year
		}
		return year + rest
	}
	for len(match) < 4 {
		match = "0" + match
	}
	return match
}

// formatEventTypeLabel converts an event type string to a display label.
func formatEventTypeLabel(eventType string) string {
	if eventType == "" {
		return "Event"
	}

	// Capitalize first letter, replace underscores with spaces
	label := strings.ReplaceAll(eventType, "_", " ")

	return strings.ToUpper(label[:1]) + label[1:]
}

// timelineIsParticipant checks if a person is a participant in an event.
func timelineIsParticipant(personID string, event *glxlib.Event) bool {
	for _, p := range event.Participants {
		if p.Person == personID {
			return true
		}
	}

	return false
}

// timelineResolvePlaceName looks up a place ID and returns its display name.
func timelineResolvePlaceName(placeID string, archive *glxlib.GLXFile) string {
	if placeID == "" {
		return ""
	}
	if place, ok := archive.Places[placeID]; ok {
		return place.Name
	}

	return placeID
}

// printTimeline prints the timeline in a formatted table.
func printTimeline(personID, personName string, entries []timelineEntry) {
	fmt.Printf("\nTimeline for %s (%s):\n\n", personID, personName)

	if len(entries) == 0 {
		fmt.Println("  No events found.")
		fmt.Println()

		return
	}

	// Find longest label for alignment
	maxLabel := 0
	for _, e := range entries {
		if len(e.Label) > maxLabel {
			maxLabel = len(e.Label)
		}
	}

	// Cap label width at a reasonable max
	if maxLabel > 40 {
		maxLabel = 40
	}

	var undated []timelineEntry

	for _, e := range entries {
		if e.SortKey == "\xff" {
			undated = append(undated, e)

			continue
		}

		date := displayDate(e.Date)

		if e.Detail != "" {
			fmt.Printf("  %-18s  %-*s  %s\n", date, maxLabel, e.Label, e.Detail)
		} else {
			fmt.Printf("  %-18s  %-*s\n", date, maxLabel, e.Label)
		}
	}

	if len(undated) > 0 {
		fmt.Println()
		fmt.Println("  Undated:")
		for _, e := range undated {
			date := displayDate(e.Date)
			if e.Detail != "" {
				fmt.Printf("  %-18s  %-*s  %s\n", date, maxLabel, e.Label, e.Detail)
			} else {
				fmt.Printf("  %-18s  %-*s\n", date, maxLabel, e.Label)
			}
		}
	}

	fmt.Printf("\n  %d event(s) shown\n\n", len(entries))
}
