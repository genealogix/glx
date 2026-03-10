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
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// clusterYearRegexp extracts the first 4-digit year from a date string.
var clusterYearRegexp = regexp.MustCompile(`\b(\d{4})\b`)

// associate represents a person connected to the target through shared context.
type associate struct {
	PersonID   string            `json:"person_id"`
	PersonName string            `json:"person_name"`
	Score      int               `json:"score"`
	Links      []associateLink   `json:"links"`
}

// associateLink describes one connection between the target and an associate.
type associateLink struct {
	Type    string `json:"type"`              // "census_household", "event_coparticipant", "place_overlap"
	EventID string `json:"event_id,omitempty"`
	Label   string `json:"label"`
	Role    string `json:"role,omitempty"`
	Year    int    `json:"year,omitempty"`
}

// clusterResult holds the full output.
type clusterResult struct {
	PersonID   string      `json:"person_id"`
	PersonName string      `json:"person_name"`
	Associates []associate `json:"associates"`
}

// showCluster loads an archive and displays FAN club analysis for a person.
func showCluster(archivePath, personQuery string, filterPlace string, beforeYear, afterYear int, jsonOutput bool) error {
	archive, err := loadArchiveForCluster(archivePath)
	if err != nil {
		return err
	}

	personID, err := resolvePersonForCluster(archive, personQuery)
	if err != nil {
		return err
	}

	result := buildCluster(personID, archive, filterPlace, beforeYear, afterYear)

	if jsonOutput {
		return printClusterJSON(result)
	}

	printClusterText(result)
	return nil
}

// loadArchiveForCluster loads an archive from a path.
func loadArchiveForCluster(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// resolvePersonForCluster finds a person by exact ID or name substring.
func resolvePersonForCluster(archive *glxlib.GLXFile, query string) (string, error) {
	if person, ok := archive.Persons[query]; ok && person != nil {
		return query, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		name := extractPersonName(person)
		if strings.Contains(strings.ToLower(name), lowerQuery) {
			matches = append(matches, id)
		}
	}

	sort.Strings(matches)

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no person found matching %q", query)
	case 1:
		return matches[0], nil
	default:
		var lines []string
		for _, id := range matches {
			name := extractPersonName(archive.Persons[id])
			lines = append(lines, fmt.Sprintf("  %s  %s", id, name))
		}
		return "", fmt.Errorf("multiple persons match %q:\n%s\nUse exact person ID", query, strings.Join(lines, "\n"))
	}
}

// buildCluster performs FAN club analysis for a person.
func buildCluster(personID string, archive *glxlib.GLXFile, filterPlace string, beforeYear, afterYear int) *clusterResult {
	person := archive.Persons[personID]
	personName := "(unknown)"
	if person != nil {
		personName = extractPersonName(person)
	}

	// Build a map of associate person ID → links
	linkMap := make(map[string][]associateLink)

	// 1. Census household connections
	collectCensusLinks(personID, archive, linkMap, filterPlace, beforeYear, afterYear)

	// 2. Event co-participant connections (non-census)
	collectEventLinks(personID, archive, linkMap, filterPlace, beforeYear, afterYear)

	// 3. Place overlap connections
	collectPlaceLinks(personID, archive, linkMap, filterPlace, beforeYear, afterYear)

	// Convert to sorted associate list
	associates := buildAssociateList(linkMap, archive)

	return &clusterResult{
		PersonID:   personID,
		PersonName: personName,
		Associates: associates,
	}
}

// collectCensusLinks finds people in the same census events.
func collectCensusLinks(personID string, archive *glxlib.GLXFile, linkMap map[string][]associateLink, filterPlace string, beforeYear, afterYear int) {
	eventIDs := sortedKeys(archive.Events)
	for _, eventID := range eventIDs {
		event := archive.Events[eventID]
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}

		if !eventHasParticipant(personID, event) {
			continue
		}

		year := clusterExtractYear(string(event.Date))
		if !yearInRange(year, beforeYear, afterYear) {
			continue
		}

		if filterPlace != "" && event.PlaceID != filterPlace {
			if !placeIsDescendant(event.PlaceID, filterPlace, archive) {
				continue
			}
		}

		label := event.Title
		if label == "" {
			label = fmt.Sprintf("%d Census", year)
			if year == 0 {
				label = "Census"
			}
		}

		for _, p := range event.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			linkMap[p.Person] = append(linkMap[p.Person], associateLink{
				Type:    "census_household",
				EventID: eventID,
				Label:   label,
				Role:    p.Role,
				Year:    year,
			})
		}
	}
}

// collectEventLinks finds people co-participating in non-census events.
func collectEventLinks(personID string, archive *glxlib.GLXFile, linkMap map[string][]associateLink, filterPlace string, beforeYear, afterYear int) {
	eventIDs := sortedKeys(archive.Events)
	for _, eventID := range eventIDs {
		event := archive.Events[eventID]
		if event == nil || event.Type == glxlib.EventTypeCensus {
			continue
		}

		if !eventHasParticipant(personID, event) {
			continue
		}

		year := clusterExtractYear(string(event.Date))
		if !yearInRange(year, beforeYear, afterYear) {
			continue
		}

		if filterPlace != "" && event.PlaceID != filterPlace {
			if !placeIsDescendant(event.PlaceID, filterPlace, archive) {
				continue
			}
		}

		label := event.Title
		if label == "" {
			eventLabel := strings.ReplaceAll(event.Type, "_", " ")
			if eventLabel != "" {
				eventLabel = strings.ToUpper(eventLabel[:1]) + eventLabel[1:]
			}
			if year > 0 {
				label = fmt.Sprintf("%s (%d)", eventLabel, year)
			} else {
				label = eventLabel
			}
		}

		for _, p := range event.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			linkMap[p.Person] = append(linkMap[p.Person], associateLink{
				Type:    "event_coparticipant",
				EventID: eventID,
				Label:   label,
				Role:    p.Role,
				Year:    year,
			})
		}
	}
}

// collectPlaceLinks finds people associated with the same places in overlapping time periods.
func collectPlaceLinks(personID string, archive *glxlib.GLXFile, linkMap map[string][]associateLink, filterPlace string, beforeYear, afterYear int) {
	// Build: which places does the target person appear at (with years)?
	targetPlaces := personPlaceYears(personID, archive)
	if len(targetPlaces) == 0 {
		return
	}

	// For each other person, check place overlap
	personIDs := sortedKeys(archive.Persons)
	for _, otherID := range personIDs {
		if otherID == personID {
			continue
		}

		otherPlaces := personPlaceYears(otherID, archive)

		for placeID, targetYears := range targetPlaces {
			if filterPlace != "" && placeID != filterPlace {
				if !placeIsDescendant(placeID, filterPlace, archive) {
					continue
				}
			}

			otherYears, ok := otherPlaces[placeID]
			if !ok {
				continue
			}

			// Check for temporal overlap (within 10-year window)
			if yearsOverlap(targetYears, otherYears) {
				// Skip if already linked via a census or event at this place
				if hasEventLinkAtPlace(otherID, placeID, linkMap) {
					continue
				}

				placeName := clusterResolvePlaceName(placeID, archive)
				yearRange := formatYearRange(otherYears)

				if !yearRangeInFilter(otherYears, beforeYear, afterYear) {
					continue
				}

				linkMap[otherID] = append(linkMap[otherID], associateLink{
					Type:  "place_overlap",
					Label: fmt.Sprintf("Same place: %s (%s)", placeName, yearRange),
				})
			}
		}
	}
}

// placeYearSet maps place IDs to the years a person was associated with them.
type placeYearSet map[string][]int

// personPlaceYears collects all places and years a person is associated with via events.
func personPlaceYears(personID string, archive *glxlib.GLXFile) placeYearSet {
	result := make(placeYearSet)

	for _, event := range archive.Events {
		if event == nil || event.PlaceID == "" {
			continue
		}
		if !eventHasParticipant(personID, event) {
			continue
		}

		year := clusterExtractYear(string(event.Date))
		if year > 0 {
			result[event.PlaceID] = append(result[event.PlaceID], year)
		}
	}

	return result
}

// yearsOverlap checks if two year sets have any values within 10 years of each other.
func yearsOverlap(a, b []int) bool {
	for _, ya := range a {
		for _, yb := range b {
			diff := ya - yb
			if diff < 0 {
				diff = -diff
			}
			if diff <= 10 {
				return true
			}
		}
	}
	return false
}

// hasEventLinkAtPlace checks if an associate already has a census/event link,
// to avoid redundant place_overlap entries.
func hasEventLinkAtPlace(personID, placeID string, linkMap map[string][]associateLink) bool {
	for _, link := range linkMap[personID] {
		if link.Type == "census_household" || link.Type == "event_coparticipant" {
			return true
		}
	}
	return false
}

// buildAssociateList converts the link map to a sorted list ranked by score.
func buildAssociateList(linkMap map[string][]associateLink, archive *glxlib.GLXFile) []associate {
	associates := make([]associate, 0, len(linkMap))

	for personID, links := range linkMap {
		name := "(unknown)"
		if person, ok := archive.Persons[personID]; ok && person != nil {
			name = extractPersonName(person)
		}

		score := computeScore(links)

		associates = append(associates, associate{
			PersonID:   personID,
			PersonName: name,
			Score:      score,
			Links:      links,
		})
	}

	sort.Slice(associates, func(i, j int) bool {
		if associates[i].Score != associates[j].Score {
			return associates[i].Score > associates[j].Score
		}
		return associates[i].PersonID < associates[j].PersonID
	})

	return associates
}

// computeScore calculates connection strength. Census and event links are
// weighted higher than place-only overlaps. Multiple connections compound.
func computeScore(links []associateLink) int {
	score := 0
	for _, link := range links {
		switch link.Type {
		case "census_household":
			score += 3
		case "event_coparticipant":
			score += 2
		case "place_overlap":
			score += 1
		}
	}
	return score
}

// eventHasParticipant checks if a person participates in an event.
func eventHasParticipant(personID string, event *glxlib.Event) bool {
	for _, p := range event.Participants {
		if p.Person == personID {
			return true
		}
	}
	return false
}

// placeIsDescendant checks if placeID is a descendant of ancestorID in the place hierarchy.
func placeIsDescendant(placeID, ancestorID string, archive *glxlib.GLXFile) bool {
	visited := make(map[string]bool)
	current := placeID
	for current != "" && !visited[current] {
		visited[current] = true
		place, ok := archive.Places[current]
		if !ok || place == nil {
			return false
		}
		if place.ParentID == ancestorID {
			return true
		}
		current = place.ParentID
	}
	return false
}

// clusterExtractYear extracts the first 4-digit year from a date string.
func clusterExtractYear(dateStr string) int {
	if dateStr == "" {
		return 0
	}
	match := clusterYearRegexp.FindStringSubmatch(dateStr)
	if len(match) < 2 {
		return 0
	}
	year, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return year
}

// yearInRange checks if a year falls within the filter range.
// A zero year always passes (undated events are included).
func yearInRange(year, beforeYear, afterYear int) bool {
	if year == 0 {
		return true
	}
	if beforeYear > 0 && year >= beforeYear {
		return false
	}
	if afterYear > 0 && year <= afterYear {
		return false
	}
	return true
}

// yearRangeInFilter checks if any year in the set is within the filter range.
func yearRangeInFilter(years []int, beforeYear, afterYear int) bool {
	if beforeYear == 0 && afterYear == 0 {
		return true
	}
	for _, y := range years {
		if yearInRange(y, beforeYear, afterYear) {
			return true
		}
	}
	return false
}

func formatYearRange(years []int) string {
	if len(years) == 0 {
		return "?"
	}
	sorted := make([]int, len(years))
	copy(sorted, years)
	sort.Ints(sorted)
	if sorted[0] == sorted[len(sorted)-1] {
		return fmt.Sprintf("%d", sorted[0])
	}
	return fmt.Sprintf("%d–%d", sorted[0], sorted[len(sorted)-1])
}

func clusterResolvePlaceName(placeID string, archive *glxlib.GLXFile) string {
	if place, ok := archive.Places[placeID]; ok && place != nil {
		return place.Name
	}
	return placeID
}

// printClusterText prints the cluster in human-readable format.
func printClusterText(result *clusterResult) {
	fmt.Printf("\nAssociates of %s (%s):\n", result.PersonName, result.PersonID)

	if len(result.Associates) == 0 {
		fmt.Println("\n  No associates found.")
		return
	}

	// Group by link type
	type groupEntry struct {
		key   string
		label string
	}
	groups := []groupEntry{
		{"census_household", "Census Households"},
		{"event_coparticipant", "Shared Events"},
		{"place_overlap", "Same Place, Same Period"},
	}

	for _, group := range groups {
		var entries []string
		for _, assoc := range result.Associates {
			for _, link := range assoc.Links {
				if link.Type == group.key {
					line := fmt.Sprintf("    %s (%s)", assoc.PersonName, assoc.PersonID)
					if link.Role != "" {
						line += fmt.Sprintf(" — %s", link.Role)
					}
					line += fmt.Sprintf("  [%s]", link.Label)
					if link.Type != "place_overlap" && assoc.Score > 0 {
						line += fmt.Sprintf("  (score: %d)", assoc.Score)
					}
					entries = append(entries, line)
				}
			}
		}
		if len(entries) > 0 {
			fmt.Printf("\n  %s:\n", group.label)
			for _, e := range entries {
				fmt.Println(e)
			}
		}
	}

	fmt.Printf("\n  %d associate(s) found\n\n", len(result.Associates))
}

// printClusterJSON outputs the result as JSON.
func printClusterJSON(result *clusterResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
