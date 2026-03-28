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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// vitalRecord holds one row in the vitals output.
type vitalRecord struct {
	Label string
	Value string
}

// vitalsEventTypes lists event types that are considered vitals.
var vitalsEventTypes = []string{
	"birth", "christening", "baptism", "death", "burial", "cremation",
}

// showVitals loads an archive and displays vitals for a person.
func showVitals(archivePath, personQuery string) error {
	archive, err := loadArchiveForVitals(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPerson(archive, personQuery)
	if err != nil {
		return err
	}

	vitals := collectVitals(personID, person, archive)
	printVitals(personID, vitals)

	return nil
}

// loadArchiveForVitals loads an archive from a path (directory or single file).
func loadArchiveForVitals(path string) (*glxlib.GLXFile, error) {
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

// findPerson looks up a person by exact ID or by name substring match.
// Returns the person ID, the Person, or an error.
func findPerson(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	// Try exact ID match first
	if person, ok := archive.Persons[query]; ok {
		if person == nil {
			return "", nil, fmt.Errorf("person %q exists in archive but has no data", query)
		}
		return query, person, nil
	}

	// Fall back to name search (case-insensitive substring)
	lowerQuery := strings.ToLower(query)
	var matches []string

	ids := sortedKeys(archive.Persons)
	for _, id := range ids {
		person := archive.Persons[id]
		if person == nil {
			continue
		}
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

// collectVitals gathers vital records from person properties and events.
func collectVitals(personID string, person *glxlib.Person, archive *glxlib.GLXFile) []vitalRecord {
	var vitals []vitalRecord

	// Sort event keys once and reuse for all lookups
	eventIDs := sortedKeys(archive.Events)

	// Name
	name := extractPersonName(person)
	vitals = append(vitals, vitalRecord{"Name", name})

	// Sex/Gender
	gender := propertyString(person.Properties, "gender")
	if gender == "" {
		gender = propertyString(person.Properties, "sex")
	}
	vitals = append(vitals, vitalRecord{"Sex", displayOrDash(gender)})

	// Birth — check person properties first, then events
	birth := formatPropertyDatePlace(person.Properties, "born_on", "born_at", archive)
	if birth == "" {
		birth = findEventByType(personID, "birth", eventIDs, archive)
	}
	vitals = append(vitals, vitalRecord{"Birth", displayOrDash(birth)})

	// Christening/Baptism — from events
	christening := findEventByType(personID, "christening", eventIDs, archive)
	if christening == "" {
		christening = findEventByType(personID, "baptism", eventIDs, archive)
	}
	vitals = append(vitals, vitalRecord{"Christening", displayOrDash(christening)})

	// Death — check person properties first, then events
	death := formatPropertyDatePlace(person.Properties, "died_on", "died_at", archive)
	if death == "" {
		death = findEventByType(personID, "death", eventIDs, archive)
	}
	vitals = append(vitals, vitalRecord{"Death", displayOrDash(death)})

	// Burial — check person properties first, then events
	burial := formatPropertyPlace(person.Properties, "buried_at", archive)
	if burial == "" {
		burial = findEventByType(personID, "burial", eventIDs, archive)
	}
	if burial == "" {
		burial = findEventByType(personID, "cremation", eventIDs, archive)
	}
	vitals = append(vitals, vitalRecord{"Burial", displayOrDash(burial)})

	// Other life events (not already covered by vitals)
	others := findOtherEvents(personID, eventIDs, archive)
	for _, other := range others {
		vitals = append(vitals, other)
	}

	return vitals
}

// findEventByType finds the first event of a given type where the person is a participant.
// Accepts pre-sorted event IDs to avoid repeated sorting across multiple calls.
func findEventByType(personID, eventType string, eventIDs []string, archive *glxlib.GLXFile) string {
	for _, id := range eventIDs {
		event := archive.Events[id]
		if !strings.EqualFold(event.Type, eventType) {
			continue
		}
		if !isParticipant(personID, event) {
			continue
		}

		return formatEventDatePlace(event, archive)
	}

	return ""
}

// findOtherEvents returns events the person participates in that aren't standard vitals.
// Accepts pre-sorted event IDs to avoid repeated sorting.
func findOtherEvents(personID string, eventIDs []string, archive *glxlib.GLXFile) []vitalRecord {
	var others []vitalRecord

	for _, id := range eventIDs {
		event := archive.Events[id]
		if !isParticipant(personID, event) {
			continue
		}

		// Skip vitals event types (already displayed)
		eventType := strings.ToLower(event.Type)
		isVital := false
		for _, vt := range vitalsEventTypes {
			if eventType == vt {
				isVital = true

				break
			}
		}
		if isVital {
			continue
		}

		labelSource := event.Type
		if labelSource == "" {
			labelSource = id
		}
		label := strings.ToUpper(labelSource[:1]) + labelSource[1:]

		value := formatEventDatePlace(event, archive)
		if value == "" {
			value = id
		}

		others = append(others, vitalRecord{label, value})
	}

	return others
}

// isParticipant checks if a person is a participant in an event.
func isParticipant(personID string, event *glxlib.Event) bool {
	for _, p := range event.Participants {
		if p.Person == personID {
			return true
		}
	}

	return false
}

// formatEventDatePlace formats an event's date and place for display.
func formatEventDatePlace(event *glxlib.Event, archive *glxlib.GLXFile) string {
	date := formatReadableDate(string(event.Date))
	placeName := resolvePlaceName(event.PlaceID, archive)

	switch {
	case date != "" && placeName != "":
		return date + ", " + placeName
	case date != "":
		return date
	case placeName != "":
		return placeName
	default:
		return ""
	}
}

// printVitals prints the vitals in a formatted table.
func printVitals(personID string, vitals []vitalRecord) {
	// Find longest label for alignment
	maxLabel := 0
	for _, v := range vitals {
		if len(v.Label) > maxLabel {
			maxLabel = len(v.Label)
		}
	}

	fmt.Printf("\nVitals for %s:\n\n", personID)
	for _, v := range vitals {
		fmt.Printf("  %-*s  %s\n", maxLabel, v.Label, v.Value)
	}
	fmt.Println()
}
