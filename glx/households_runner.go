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
	"regexp"
	"sort"
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Output formats of glx households.
const (
	householdsFormatText = "text"
	householdsFormatJSON = "json"
)

// householdMember is one participant of a reconstructed census household.
type householdMember struct {
	PersonID           string `json:"person_id"`
	Name               string `json:"name"`
	Role               string `json:"role,omitempty"`
	RelationshipToHead string `json:"relationship_to_head,omitempty"`
	Age                string `json:"age,omitempty"`
	Head               bool   `json:"head"`
	// Named is false for a member the schedule counts only as a tick mark.
	Named bool `json:"named"`

	ageYears float64 // parsed Age for sorting; -1 when unknown
	order    int     // position on the event, for a stable tie-break
}

// householdTally is one tick-mark row of a census household, with a
// rendered label for display.
type householdTally struct {
	Sex     string `json:"sex,omitempty"`
	AgeFrom *int   `json:"age_from,omitempty"`
	AgeTo   *int   `json:"age_to,omitempty"`
	Count   int    `json:"count"`
	Status  string `json:"status,omitempty"`
	Label   string `json:"label"`
}

// householdNeighbor is a nearby household recorded on the census event.
type householdNeighbor struct {
	Name     string `json:"name"`
	PersonID string `json:"person_id,omitempty"`
	Position string `json:"position,omitempty"`
	Page     string `json:"page,omitempty"`
	Line     string `json:"line,omitempty"`
}

// household is one census event viewed as a household.
type household struct {
	EventID   string              `json:"event_id"`
	Title     string              `json:"title"`
	Year      int                 `json:"year,omitempty"`
	Date      string              `json:"date,omitempty"`
	PlaceID   string              `json:"place_id,omitempty"`
	PlaceName string              `json:"place_name,omitempty"`
	Members   []householdMember   `json:"members"`
	Tally     []householdTally    `json:"tally,omitempty"`
	Neighbors []householdNeighbor `json:"neighbors,omitempty"`
}

// householdsResult is the full output of glx households.
type householdsResult struct {
	PersonID   string      `json:"person_id,omitempty"`
	PersonName string      `json:"person_name,omitempty"`
	PlaceID    string      `json:"place_id,omitempty"`
	PlaceName  string      `json:"place_name,omitempty"`
	Year       int         `json:"year,omitempty"`
	Households []household `json:"households"`
}

// householdsOptions are the filters and switches of glx households.
type householdsOptions struct {
	PersonQuery string
	PlaceID     string
	Year        int
	Neighbors   bool
	Format      string
}

// showHouseholds loads an archive and prints the census households of a
// person, or every census household at a place (#119).
func showHouseholds(io *IOStreams, archivePath string, opts householdsOptions) error {
	if opts.PersonQuery == "" && opts.PlaceID == "" {
		return ErrHouseholdsNoTarget
	}
	if opts.Format != "" && opts.Format != householdsFormatText && opts.Format != householdsFormatJSON {
		return fmt.Errorf("%w: %q", ErrHouseholdsUnknownFormat, opts.Format)
	}

	// Same loader as glx evidence: directory or single file, read-only.
	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	personID := ""
	if opts.PersonQuery != "" {
		personID, _, err = findPersonByQuery(archive, opts.PersonQuery)
		if err != nil {
			return err
		}
	}
	if opts.PlaceID != "" {
		if place, ok := archive.Places[opts.PlaceID]; !ok || place == nil {
			return fmt.Errorf("%w: %s", ErrHouseholdsPlaceNotFound, opts.PlaceID)
		}
	}

	result := buildHouseholds(archive, personID, opts)

	if opts.Format == householdsFormatJSON {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal households: %w", err)
		}
		fmt.Fprintln(io.MachineOut, string(data))

		return nil
	}

	printHouseholdsText(io, result, opts.Neighbors)

	return nil
}

// buildHouseholds groups census participants into households: each census
// event is one household. With a person, only that person's census events are
// returned; place and year narrow the set further. Households are ordered by
// year, then event ID.
func buildHouseholds(archive *glxlib.GLXFile, personID string, opts householdsOptions) *householdsResult {
	result := &householdsResult{PersonID: personID, Year: opts.Year, Households: []household{}}
	if personID != "" {
		result.PersonName = extractPersonName(archive.Persons[personID])
	}
	if opts.PlaceID != "" {
		result.PlaceID = opts.PlaceID
		result.PlaceName = clusterResolvePlaceName(opts.PlaceID, archive)
	}

	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}
		if personID != "" && !censusHasHouseholdMember(personID, event) {
			continue
		}
		year := glxlib.ExtractFirstYear(string(event.Date))
		if opts.Year != 0 && year != opts.Year {
			continue
		}
		if opts.PlaceID != "" && event.PlaceID != opts.PlaceID && !placeIsDescendant(event.PlaceID, opts.PlaceID, archive) {
			continue
		}
		result.Households = append(result.Households, buildHousehold(archive, eventID, event, year, opts.Neighbors))
	}

	sort.SliceStable(result.Households, func(i, j int) bool {
		a, b := result.Households[i], result.Households[j]
		if a.Year != b.Year {
			return a.Year < b.Year
		}

		return a.EventID < b.EventID
	})

	return result
}

// buildHousehold renders one census event as a household: members with the
// head first and the rest by age (oldest first, unknown ages last in
// schedule order), then the tally rows and, when requested, the neighbors.
func buildHousehold(archive *glxlib.GLXFile, eventID string, event *glxlib.Event, year int, withNeighbors bool) household {
	h := household{
		EventID: eventID,
		Title:   censusEventLabel(event, year),
		Year:    year,
		Date:    string(event.Date),
		PlaceID: event.PlaceID,
		Members: []householdMember{},
	}
	if event.PlaceID != "" {
		h.PlaceName = clusterResolvePlaceName(event.PlaceID, archive)
	}

	head := censusHeadIndex(event)
	for i, p := range event.Participants {
		if !isCensusHouseholdMember(p) {
			continue
		}
		age := participantStringProperty(p, glxlib.ParticipantPropertyAgeAtEvent)
		h.Members = append(h.Members, householdMember{
			PersonID:           p.Person,
			Name:               extractPersonName(archive.Persons[p.Person]),
			Role:               p.Role,
			RelationshipToHead: participantStringProperty(p, glxlib.ParticipantPropertyRelationshipToHead),
			Age:                age,
			Head:               i == head,
			Named:              !glxlib.IsUnnamedParticipant(p),
			ageYears:           householdAgeYears(age),
			order:              i,
		})
	}
	sort.SliceStable(h.Members, func(i, j int) bool {
		a, b := h.Members[i], h.Members[j]
		if a.Head != b.Head {
			return a.Head
		}
		if (a.ageYears >= 0) != (b.ageYears >= 0) {
			return a.ageYears >= 0
		}
		if a.ageYears != b.ageYears {
			return a.ageYears > b.ageYears
		}

		return a.order < b.order
	})

	if event.Household != nil {
		for i := range event.Household.Tally {
			row := &event.Household.Tally[i]
			h.Tally = append(h.Tally, householdTally{
				Sex:     row.Sex,
				AgeFrom: row.AgeFrom,
				AgeTo:   row.AgeTo,
				Count:   row.Count,
				Status:  row.Status,
				Label:   formatTallyRow(row),
			})
		}
	}

	if withNeighbors {
		for _, n := range event.Neighbors {
			name := n.Name
			if name == "" {
				name = extractPersonName(archive.Persons[n.Person])
			}
			h.Neighbors = append(h.Neighbors, householdNeighbor{
				Name:     name,
				PersonID: n.Person,
				Position: n.Position,
				Page:     n.Page,
				Line:     n.Line,
			})
		}
	}

	return h
}

// isCensusHouseholdMember uses the existing own-record classification to
// distinguish enumerated residents from enumerators and other nonresident
// roles. Vocabulary presence overrides do not change household membership.
func isCensusHouseholdMember(p glxlib.Participant) bool {
	return p.Person != "" && glxlib.ClassifyParticipation(glxlib.EventTypeCensus, p.Role, nil).OwnRecord
}

// censusHasHouseholdMember unions a person's roles, so a person with both a
// resident and a nonresident role still has this census as their own record.
func censusHasHouseholdMember(personID string, event *glxlib.Event) bool {
	participation, _, _ := glxlib.EventParticipation(event, personID, nil)

	return participation.OwnRecord
}

// censusHeadIndex returns the index of the census event's head of household
// in event.Participants, or -1 when there is no eligible resident. The head
// is the participant whose relationship_to_head is "head" or "self"; failing
// that, the first named participant in a subject role (principal/subject),
// since the head is the household's principal; failing that, the first named
// participant, as schedules list the head first.
func censusHeadIndex(event *glxlib.Event) int {
	for i, p := range event.Participants {
		if !isCensusHouseholdMember(p) {
			continue
		}
		switch strings.ToLower(participantStringProperty(p, glxlib.ParticipantPropertyRelationshipToHead)) {
		case "head", "self":
			return i
		}
	}
	firstNamed := -1
	for i, p := range event.Participants {
		if !isCensusHouseholdMember(p) || glxlib.IsUnnamedParticipant(p) {
			continue
		}
		switch p.Role {
		case glxlib.ParticipantRolePrincipal, glxlib.ParticipantRoleSubject, "":
			return i
		}
		if firstNamed < 0 {
			firstNamed = i
		}
	}

	return firstNamed
}

// participantStringProperty returns a participant property rendered as a
// string, or "" when absent. age_at_event is a string in the vocabulary, but
// hand-written archives often give a bare number.
func participantStringProperty(p glxlib.Participant, key string) string {
	v, ok := p.Properties[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}

	return fmt.Sprint(v)
}

var householdAgeUnits = regexp.MustCompile(`(?i)^(?:(\d+)y\s*)?(?:(\d+)m\s*)?(?:(\d+)d)?$`)

// householdAgeYears compares ages recorded as years, fractional years (3/12),
// or years/months/days (45y 3m, 6m). Units use approximate year lengths for
// ordering only; the recorded Age is preserved. Unrecognized ages sort last.
func householdAgeYears(age string) float64 {
	age = strings.TrimSpace(age)
	if age == "" {
		return -1
	}
	if years, err := strconv.ParseFloat(age, 64); err == nil && years >= 0 && !math.IsInf(years, 0) {
		return years
	}
	if numerator, denominator, ok := strings.Cut(age, "/"); ok {
		n, nErr := strconv.Atoi(strings.TrimSpace(numerator))
		d, dErr := strconv.Atoi(strings.TrimSpace(denominator))
		if nErr == nil && dErr == nil && n >= 0 && d > 0 {
			return float64(n) / float64(d)
		}

		return -1
	}
	parts := householdAgeUnits.FindStringSubmatch(age)
	if parts == nil {
		return -1
	}
	years := 0.0
	for i, divisor := range []float64{1, 12, 365.25} {
		if parts[i+1] == "" {
			continue
		}
		amount, err := strconv.ParseFloat(parts[i+1], 64)
		if err != nil {
			return -1
		}
		years += amount / divisor
	}

	return years
}

// formatTallyRow renders a tally row such as "2 female 10–15 (free white)",
// "1 male 45+", or "1 engaged in agriculture".
func formatTallyRow(row *glxlib.HouseholdTallyRow) string {
	parts := []string{strconv.Itoa(row.Count)}
	if row.Sex != "" {
		parts = append(parts, row.Sex)
	}
	switch {
	case row.AgeFrom != nil && row.AgeTo != nil:
		parts = append(parts, fmt.Sprintf("%d–%d", *row.AgeFrom, *row.AgeTo))
	case row.AgeFrom != nil:
		parts = append(parts, fmt.Sprintf("%d+", *row.AgeFrom))
	case row.AgeTo != nil:
		parts = append(parts, fmt.Sprintf("under %d", *row.AgeTo+1))
	}
	label := strings.Join(parts, " ")
	if row.Status != "" {
		if row.Sex == "" && row.AgeFrom == nil && row.AgeTo == nil {
			return label + " " + row.Status
		}
		label += " (" + row.Status + ")"
	}

	return label
}

// printHouseholdsText prints the households in human-readable form.
func printHouseholdsText(io *IOStreams, result *householdsResult, withNeighbors bool) {
	switch {
	case result.PersonID != "":
		io.Printf("Households for %s (%s):\n", result.PersonName, result.PersonID)
	case result.Year != 0:
		io.Printf("Census households at %s (%s) in %d:\n", result.PlaceName, result.PlaceID, result.Year)
	default:
		io.Printf("Census households at %s (%s):\n", result.PlaceName, result.PlaceID)
	}

	if len(result.Households) == 0 {
		io.Println("\n  No census households found.")

		return
	}

	for i := range result.Households {
		h := &result.Households[i]
		header := h.Title
		if h.PlaceName != "" && !strings.Contains(h.Title, h.PlaceName) {
			header += " — " + h.PlaceName
		}
		io.Printf("\n  %s  [%s]\n", header, h.EventID)
		for j := range h.Members {
			io.Printf("    %s\n", formatHouseholdMember(&h.Members[j]))
		}
		if len(h.Tally) > 0 {
			io.Println("    Tally (as enumerated):")
			for _, t := range h.Tally {
				io.Printf("      %s\n", t.Label)
			}
		}
		if withNeighbors {
			if len(h.Neighbors) == 0 {
				io.Println("    Neighbors: none recorded")

				continue
			}
			io.Println("    Neighbors:")
			for j := range h.Neighbors {
				io.Printf("      %s\n", formatHouseholdNeighbor(&h.Neighbors[j]))
			}
		}
	}
	io.Println("")
}

// formatHouseholdMember renders "Name (id) — head, age 45", marking members
// counted on the schedule without being named.
func formatHouseholdMember(m *householdMember) string {
	line := fmt.Sprintf("%s (%s)", m.Name, m.PersonID)
	var details []string
	switch {
	case m.RelationshipToHead != "":
		details = append(details, m.RelationshipToHead)
	case m.Head:
		details = append(details, "head")
	case m.Role != "":
		details = append(details, strings.ReplaceAll(m.Role, "_", " "))
	}
	if m.Age != "" {
		details = append(details, "age "+m.Age)
	}
	if !m.Named {
		details = append(details, "counted, not named")
	}
	if len(details) > 0 {
		line += " — " + strings.Join(details, ", ")
	}

	return line
}

// formatHouseholdNeighbor renders a neighbor line such as
// "Henry Jeffries (person-henry-jeffries) — previous household, page 12, line 3".
func formatHouseholdNeighbor(n *householdNeighbor) string {
	line := n.Name
	if n.PersonID != "" {
		line += " (" + n.PersonID + ")"
	}
	var details []string
	if n.Position != "" {
		details = append(details, strings.ReplaceAll(n.Position, "_", " "))
	}
	if n.Page != "" {
		details = append(details, "page "+n.Page)
	}
	if n.Line != "" {
		details = append(details, "line "+n.Line)
	}
	if len(details) > 0 {
		line += " — " + strings.Join(details, ", ")
	}

	return line
}
