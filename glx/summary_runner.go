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
	"sort"
	"strings"
	"unicode/utf8"

	glxlib "github.com/genealogix/glx/go-glx"
)

// nameVariant holds a single name variant with its type classification.
type nameVariant struct {
	Value    string
	NameType string // e.g., "birth", "married", "as_recorded"
}

// spouseInfo holds details about a spouse relationship.
type spouseInfo struct {
	PersonID      string
	PersonName    string
	RelType       string
	MarriageDate  string
	MarriagePlace string
}

// otherRelInfo holds a non-family relationship for display.
type otherRelInfo struct {
	RelType   string
	OtherName string
	OtherID   string
}

// parentChildRelTypes maps relationship types that represent parent-child connections.
var parentChildRelTypes = map[string]bool{
	"parent_child":            true,
	"biological_parent_child": true,
	"adoptive_parent_child":   true,
	"foster_parent_child":     true,
	"step_parent":             true,
	"sibling":                 true,
}

// marriageRelTypes maps relationship types that represent spouse/partner connections.
var marriageRelTypes = map[string]bool{
	"marriage": true,
	"partner":  true,
}

// summarySkippedEventTypes are event types excluded from the life events section.
var summarySkippedEventTypes = map[string]bool{
	"birth": true, "christening": true, "baptism": true,
	"death": true, "burial": true, "cremation": true,
	"marriage": true,
}

// summarySkippedProperties are person properties shown in other sections.
var summarySkippedProperties = map[string]bool{
	"name": true, "primary_name": true,
	"gender": true, "sex": true,
}

// loadArchiveForSummary loads an archive from a path (directory or single file).
func loadArchiveForSummary(path string) (*glxlib.GLXFile, error) {
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

// showSummary loads an archive and displays a comprehensive person profile.
func showSummary(archivePath, personQuery string) error {
	archive, err := loadArchiveForSummary(archivePath)
	if err != nil {
		return err
	}

	personID, person, err := findPersonByQuery(archive, personQuery)
	if err != nil {
		return err
	}

	fmt.Printf("=== %s ===\n\n", personID)

	printIdentitySection(person)
	printVitalEventsSection(personID, person, archive)
	printLifeEventsSection(personID, person, archive)
	printFamilySection(personID, archive)
	printOtherRelationshipsSection(personID, archive)
	printLifeHistorySection(personID, person, archive)

	return nil
}

// ============================================================================
// Person lookup
// ============================================================================

// findPersonByQuery looks up a person by exact ID or by name substring match.
func findPersonByQuery(archive *glxlib.GLXFile, query string) (string, *glxlib.Person, error) {
	if person, ok := archive.Persons[query]; ok {
		if person == nil {
			return "", nil, fmt.Errorf("person %q exists in archive but has no data", query)
		}
		return query, person, nil
	}

	lowerQuery := strings.ToLower(query)
	var matches []string

	for id, person := range archive.Persons {
		names := extractAllNameValues(person)
		for _, name := range names {
			if strings.Contains(strings.ToLower(name), lowerQuery) {
				matches = append(matches, id)

				break
			}
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

		return "", nil, fmt.Errorf("multiple persons match %q:\n%s\nUse an exact person ID to disambiguate",
			query, strings.Join(lines, "\n"))
	}
}

// ============================================================================
// Name extraction helpers
// ============================================================================

// extractAllNameVariants returns all name variants from a person's properties.
func extractAllNameVariants(person *glxlib.Person) []nameVariant {
	raw, ok := person.Properties["name"]
	if !ok {
		raw, ok = person.Properties["primary_name"]
	}
	if !ok {
		return nil
	}

	if s, ok := raw.(string); ok {
		return []nameVariant{{Value: s}}
	}

	if m, ok := raw.(map[string]any); ok {
		return []nameVariant{nameVariantFromMap(m)}
	}

	if list, ok := raw.([]any); ok {
		var variants []nameVariant
		for _, item := range list {
			if m, ok := item.(map[string]any); ok {
				variants = append(variants, nameVariantFromMap(m))
			}
		}

		return variants
	}

	return nil
}

// nameVariantFromMap extracts a nameVariant from a structured map.
func nameVariantFromMap(m map[string]any) nameVariant {
	entry := nameVariant{}
	if v, ok := m["value"]; ok {
		entry.Value = fmt.Sprint(v)
	}
	if fields, ok := m["fields"].(map[string]any); ok {
		if t, ok := fields["type"]; ok {
			entry.NameType = fmt.Sprint(t)
		}
	}

	return entry
}

// extractAllNameValues returns just the name strings (no type info).
func extractAllNameValues(person *glxlib.Person) []string {
	variants := extractAllNameVariants(person)
	if len(variants) == 0 {
		return []string{extractPersonName(person)}
	}

	seen := map[string]bool{}
	var names []string
	for _, v := range variants {
		if v.Value != "" && !seen[v.Value] {
			names = append(names, v.Value)
			seen[v.Value] = true
		}
	}

	return names
}

// formatNameType converts a name type code to a display label.
func formatNameType(t string) string {
	switch strings.ToLower(t) {
	case "birth":
		return "Birth Name"
	case "married":
		return "Married Name"
	case "maiden":
		return "Maiden Name"
	case "as_recorded":
		return "As Recorded"
	case "aka":
		return "Also Known As"
	case "nickname":
		return "Nickname"
	case "anglicized":
		return "Anglicized"
	case "professional":
		return "Professional"
	default:
		return snakeCaseToTitle(t)
	}
}

// ============================================================================
// Section printers
// ============================================================================

// printIdentitySection prints name, sex, and alternate names.
func printIdentitySection(person *glxlib.Person) {
	name := extractPersonName(person)
	fmt.Printf("  %-18s%s\n", "Name:", name)

	gender := propertyString(person.Properties, "gender")
	if gender == "" {
		gender = propertyString(person.Properties, "sex")
	}
	fmt.Printf("  %-18s%s\n", "Sex:", displayOrDash(gender))

	variants := extractAllNameVariants(person)
	if len(variants) > 1 {
		fmt.Printf("  Alternate Names:\n")

		// Group by type
		grouped := map[string][]string{}
		var typeOrder []string
		for i, v := range variants {
			if i == 0 {
				continue // skip primary name
			}
			key := v.NameType
			if key == "" {
				key = "(untyped)"
			}
			if _, exists := grouped[key]; !exists {
				typeOrder = append(typeOrder, key)
			}
			grouped[key] = append(grouped[key], v.Value)
		}

		for _, key := range typeOrder {
			label := formatNameType(key)
			values := grouped[key]
			fmt.Printf("    %-16s%s\n", label+":", strings.Join(values, ", "))
		}
	}

	fmt.Println()
}

// printVitalEventsSection prints birth, christening, death, burial.
func printVitalEventsSection(personID string, person *glxlib.Person, archive *glxlib.GLXFile) {
	fmt.Println(sectionHeader("Vital Events"))

	// Birth: from events
	birth := findEventForPerson(personID, "birth", archive)
	fmt.Printf("  %-18s%s\n", "Birth:", displayOrDash(birth))

	// Christening/Baptism: from events
	christening := findEventForPerson(personID, "christening", archive)
	if christening == "" {
		christening = findEventForPerson(personID, "baptism", archive)
	}
	fmt.Printf("  %-18s%s\n", "Christening:", displayOrDash(christening))

	// Death: from events
	death := findEventForPerson(personID, "death", archive)
	fmt.Printf("  %-18s%s\n", "Death:", displayOrDash(death))

	// Burial/Cremation: events only
	burial := findEventForPerson(personID, "burial", archive)
	if burial == "" {
		burial = findEventForPerson(personID, "cremation", archive)
	}
	fmt.Printf("  %-18s%s\n", "Burial:", displayOrDash(burial))

	fmt.Println()
}

// printLifeEventsSection prints non-vital events and temporal person properties.
func printLifeEventsSection(personID string, person *glxlib.Person, archive *glxlib.GLXFile) {
	fmt.Println(sectionHeader("Life Events"))

	hasContent := false

	// Non-vital, non-marriage events where person participates
	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		if summarySkippedEventTypes[strings.ToLower(event.Type)] {
			continue
		}
		if !isPersonParticipant(personID, event) {
			continue
		}

		label := snakeCaseToTitle(event.Type) + ":"
		detail := formatSummaryEventDatePlace(event, archive)
		fmt.Printf("  %-18s%s\n", label, detail)
		hasContent = true
	}

	// Temporal person properties (occupation, residence, religion, etc.)
	if person.Properties != nil {
		var propKeys []string
		for k := range person.Properties {
			if !summarySkippedProperties[k] {
				propKeys = append(propKeys, k)
			}
		}
		sort.Strings(propKeys)

		for _, key := range propKeys {
			raw := person.Properties[key]
			label := snakeCaseToTitle(key)

			if printTemporalProperty(label, raw) {
				hasContent = true
			}
		}
	}

	if !hasContent {
		fmt.Println("  (none)")
	}

	fmt.Println()
}

// printTemporalProperty prints a person property value, handling temporal lists.
// Returns true if anything was printed.
func printTemporalProperty(label string, raw any) bool {
	if raw == nil {
		return false
	}

	// Temporal list
	if list, ok := raw.([]any); ok {
		for _, item := range list {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value := ""
			if v, exists := m["value"]; exists {
				value = fmt.Sprint(v)
			}
			date := ""
			if d, exists := m["date"]; exists {
				date = fmt.Sprint(d)
			}
			display := value
			if date != "" {
				display += " (" + date + ")"
			}
			fmt.Printf("  %-18s%s\n", label+":", display)
		}

		return len(list) > 0
	}

	// Structured map with "value" key
	if m, ok := raw.(map[string]any); ok {
		if v, exists := m["value"]; exists {
			fmt.Printf("  %-18s%s\n", label+":", fmt.Sprint(v))

			return true
		}
	}

	// Simple string
	if s, ok := raw.(string); ok {
		fmt.Printf("  %-18s%s\n", label+":", s)

		return true
	}

	return false
}

// printFamilySection prints spouses, parents, and siblings.
func printFamilySection(personID string, archive *glxlib.GLXFile) {
	fmt.Println(sectionHeader("Family"))

	// Spouses
	spouses := findSpouses(personID, archive)
	if len(spouses) == 0 {
		fmt.Printf("  %-18s%s\n", "Spouse:", "(none)")
	}
	for _, sp := range spouses {
		detail := sp.PersonName
		var parts []string
		if sp.MarriageDate != "" {
			parts = append(parts, sp.MarriageDate)
		}
		if sp.MarriagePlace != "" {
			parts = append(parts, sp.MarriagePlace)
		}
		if len(parts) > 0 {
			detail += "  (m. " + strings.Join(parts, ", ") + ")"
		}
		fmt.Printf("  %-18s%s\n", "Spouse:", detail)
	}

	// Parents
	parentIDs := findParentIDs(personID, archive)
	if len(parentIDs) == 0 {
		fmt.Printf("  %-18s%s\n", "Father:", "(unknown)")
		fmt.Printf("  %-18s%s\n", "Mother:", "(unknown)")
	} else {
		for _, pid := range parentIDs {
			parent, ok := archive.Persons[pid]
			label := "Parent"
			name := pid
			if ok {
				name = extractPersonName(parent)
				gender := strings.ToLower(propertyString(parent.Properties, "gender"))
				if gender == "male" {
					label = "Father"
				} else if gender == "female" {
					label = "Mother"
				}
			}
			fmt.Printf("  %-18s%s\n", label+":", name)
		}
	}

	// Siblings
	siblingIDs := findSiblingIDs(personID, parentIDs, archive)
	if len(siblingIDs) == 0 {
		fmt.Printf("  %-18s%s\n", "Siblings:", "(none found)")
	} else {
		var sibNames []string
		for _, sid := range siblingIDs {
			if sib, ok := archive.Persons[sid]; ok && sib != nil {
				sibNames = append(sibNames, extractPersonName(sib))
			} else {
				sibNames = append(sibNames, sid)
			}
		}
		fmt.Printf("  %-18s%s\n", "Siblings:", strings.Join(sibNames, ", "))
	}

	fmt.Println()
}

// printOtherRelationshipsSection prints non-family relationships.
func printOtherRelationshipsSection(personID string, archive *glxlib.GLXFile) {
	rels := findOtherRelationships(personID, archive)
	if len(rels) == 0 {
		return
	}

	fmt.Println(sectionHeader("Relationships"))
	for _, r := range rels {
		label := snakeCaseToTitle(r.RelType)
		fmt.Printf("  %-18s%s\n", label+":", r.OtherName)
	}
	fmt.Println()
}

// printLifeHistorySection prints an auto-generated biographical narrative.
func printLifeHistorySection(personID string, person *glxlib.Person, archive *glxlib.GLXFile) {
	history := generateLifeHistory(personID, person, archive)
	if history == "" {
		return
	}

	fmt.Println(sectionHeader("Life History"))
	fmt.Printf("  %s\n\n", history)
}

// ============================================================================
// Relationship finders
// ============================================================================

// findSpouses finds spouse/partner relationships for a person.
func findSpouses(personID string, archive *glxlib.GLXFile) []spouseInfo {
	var spouses []spouseInfo

	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		if !marriageRelTypes[strings.ToLower(rel.Type)] {
			continue
		}

		if !hasParticipant(personID, rel.Participants) {
			continue
		}

		for _, p := range rel.Participants {
			if p.Person == personID {
				continue
			}

			info := spouseInfo{
				PersonID: p.Person,
				RelType:  rel.Type,
			}

			if sp, ok := archive.Persons[p.Person]; ok && sp != nil {
				info.PersonName = extractPersonName(sp)
			} else {
				info.PersonName = p.Person
			}

			// Get marriage date/place from start_event
			if rel.StartEvent != "" {
				if ev, ok := archive.Events[rel.StartEvent]; ok && ev != nil {
					info.MarriageDate = string(ev.Date)
					info.MarriagePlace = resolvePlaceName(ev.PlaceID, archive)
				}
			}

			// If no start_event, search for a marriage event with both participants
			if info.MarriageDate == "" {
				info.MarriageDate, info.MarriagePlace = findMarriageEvent(personID, p.Person, archive)
			}

			spouses = append(spouses, info)
		}
	}

	// Sort spouses chronologically by full date (not just year).
	// Uses dateSortKey which handles ISO dates, prefixed dates, and
	// sorts undated ("\xff") after all dated entries.
	sort.SliceStable(spouses, func(i, j int) bool {
		ki := dateSortKey(spouses[i].MarriageDate)
		kj := dateSortKey(spouses[j].MarriageDate)
		return ki < kj
	})

	return spouses
}

// findMarriageEvent searches for a marriage event involving both persons.
func findMarriageEvent(personA, personB string, archive *glxlib.GLXFile) (date, place string) {
	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		ev := archive.Events[id]
		if ev == nil {
			continue
		}
		if !strings.EqualFold(ev.Type, "marriage") {
			continue
		}
		hasA, hasB := false, false
		for _, p := range ev.Participants {
			if p.Person == personA {
				hasA = true
			}
			if p.Person == personB {
				hasB = true
			}
		}
		if hasA && hasB {
			return string(ev.Date), resolvePlaceName(ev.PlaceID, archive)
		}
	}

	return "", ""
}

// findParentIDs finds parent person IDs for a given person.
func findParentIDs(personID string, archive *glxlib.GLXFile) []string {
	var parents []string
	seen := map[string]bool{}

	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		if !parentChildRelTypes[strings.ToLower(rel.Type)] {
			continue
		}

		isChild := false
		for _, p := range rel.Participants {
			if p.Person == personID && strings.EqualFold(p.Role, "child") {
				isChild = true

				break
			}
		}
		if !isChild {
			continue
		}

		for _, p := range rel.Participants {
			if strings.EqualFold(p.Role, "parent") && !seen[p.Person] {
				parents = append(parents, p.Person)
				seen[p.Person] = true
			}
		}
	}

	return parents
}

// findChildIDs finds child person IDs for a given person, sorted by birth year.
func findChildIDs(personID string, archive *glxlib.GLXFile) []string {
	children := map[string]bool{}

	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		if rel == nil || !parentChildRelTypes[strings.ToLower(rel.Type)] {
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
			if strings.EqualFold(p.Role, "child") && !children[p.Person] {
				children[p.Person] = true
			}
		}
	}

	result := make([]string, 0, len(children))
	for id := range children {
		result = append(result, id)
	}

	// Sort by birth year, then by ID for stability
	sort.Slice(result, func(i, j int) bool {
		yi := birthYear(archive, result[i])
		yj := birthYear(archive, result[j])
		if yi != yj {
			if yi == 0 {
				return false
			}
			if yj == 0 {
				return true
			}
			return yi < yj
		}
		return result[i] < result[j]
	})

	return result
}

// birthYear returns the birth year for a person by looking up their birth event.
// Returns 0 if no birth event is found.
func birthYear(archive *glxlib.GLXFile, personID string) int {
	_, event := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth)
	if event == nil {
		return 0
	}
	return glxlib.ExtractFirstYear(string(event.Date))
}

// findSiblingIDs finds siblings by looking for other children of the same parents.
func findSiblingIDs(personID string, parentIDs []string, archive *glxlib.GLXFile) []string {
	siblings := map[string]bool{}

	// Infer siblings from shared parent-child relationships
	if len(parentIDs) > 0 {
		parentSet := map[string]bool{}
		for _, pid := range parentIDs {
			parentSet[pid] = true
		}

		ids := sortedKeys(archive.Relationships)
		for _, relID := range ids {
			rel := archive.Relationships[relID]
			if !parentChildRelTypes[strings.ToLower(rel.Type)] || strings.EqualFold(rel.Type, "sibling") {
				continue
			}

			hasKnownParent := false
			for _, p := range rel.Participants {
				if strings.EqualFold(p.Role, "parent") && parentSet[p.Person] {
					hasKnownParent = true

					break
				}
			}
			if !hasKnownParent {
				continue
			}

			for _, p := range rel.Participants {
				if p.Person != personID && strings.EqualFold(p.Role, "child") {
					siblings[p.Person] = true
				}
			}
		}
	}

	// Also include explicit sibling relationships
	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		if !strings.EqualFold(rel.Type, "sibling") {
			continue
		}
		if !hasParticipant(personID, rel.Participants) {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person != personID {
				siblings[p.Person] = true
			}
		}
	}

	result := make([]string, 0, len(siblings))
	for id := range siblings {
		result = append(result, id)
	}
	sort.Strings(result)

	return result
}

// findOtherRelationships finds non-family relationships involving a person.
func findOtherRelationships(personID string, archive *glxlib.GLXFile) []otherRelInfo {
	var rels []otherRelInfo

	ids := sortedKeys(archive.Relationships)
	for _, relID := range ids {
		rel := archive.Relationships[relID]
		relType := strings.ToLower(rel.Type)

		if parentChildRelTypes[relType] || marriageRelTypes[relType] {
			continue
		}

		if !hasParticipant(personID, rel.Participants) {
			continue
		}

		for _, p := range rel.Participants {
			if p.Person == personID {
				continue
			}
			name := p.Person
			if sp, ok := archive.Persons[p.Person]; ok && sp != nil {
				name = extractPersonName(sp)
			}
			rels = append(rels, otherRelInfo{
				RelType:   rel.Type,
				OtherName: name,
				OtherID:   p.Person,
			})
		}
	}

	return rels
}

// ============================================================================
// Event and place helpers
// ============================================================================

// findEventForPerson finds the first event of a given type where the person participates.
func findEventForPerson(personID, eventType string, archive *glxlib.GLXFile) string {
	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		if !strings.EqualFold(event.Type, eventType) {
			continue
		}
		if isPersonParticipant(personID, event) {
			return formatSummaryEventDatePlace(event, archive)
		}
	}

	return ""
}

// isPersonParticipant checks if a person participates in an event.
func isPersonParticipant(personID string, event *glxlib.Event) bool {
	for _, p := range event.Participants {
		if p.Person == personID {
			return true
		}
	}

	return false
}

// hasParticipant checks if a person is among participants.
func hasParticipant(personID string, participants []glxlib.Participant) bool {
	for _, p := range participants {
		if p.Person == personID {
			return true
		}
	}

	return false
}

// formatSummaryEventDatePlace formats an event's date and place for display.
func formatSummaryEventDatePlace(event *glxlib.Event, archive *glxlib.GLXFile) string {
	date := formatReadableDate(string(event.Date))
	place := resolvePlaceName(event.PlaceID, archive)

	switch {
	case date != "" && place != "":
		return date + ", " + place
	case date != "":
		return date
	case place != "":
		return place
	default:
		return "(no details)"
	}
}

// formatPropertyDatePlace combines a date property and a place property.
func formatPropertyDatePlace(props map[string]any, dateKey, placeKey string, archive *glxlib.GLXFile) string {
	date := formatReadableDate(propertyString(props, dateKey))
	placeID := propertyString(props, placeKey)
	place := resolvePlaceName(placeID, archive)

	switch {
	case date != "" && place != "":
		return date + ", " + place
	case date != "":
		return date
	case place != "":
		return place
	default:
		return ""
	}
}

// formatPropertyPlace returns a resolved place name from a person property.
// Handles plain string values ("place-id") and map values ({"value": "place-id"}).
func formatPropertyPlace(props map[string]any, placeKey string, archive *glxlib.GLXFile) string {
	raw, ok := props[placeKey]
	if !ok {
		return ""
	}
	var placeID string
	switch v := raw.(type) {
	case string:
		placeID = v
	case map[string]any:
		if val, ok := v["value"].(string); ok {
			placeID = val
		}
	}

	return resolvePlaceName(placeID, archive)
}

// resolvePlaceName looks up a place ID and returns its name.
func resolvePlaceName(placeID string, archive *glxlib.GLXFile) string {
	if placeID == "" {
		return ""
	}
	if place, ok := archive.Places[placeID]; ok && place != nil {
		return place.Name
	}

	return placeID
}

// ============================================================================
// Life history narrative
// ============================================================================

// generateLifeHistory creates a brief biographical paragraph from archive data.
func generateLifeHistory(personID string, person *glxlib.Person, archive *glxlib.GLXFile) string {
	var sentences []string

	name := extractPersonName(person)
	subject, possessive := pronounFor(person)

	// Birth
	birthDate, birthPlace := findEventDatePlace(personID, "birth", archive)
	if birthDate != "" || birthPlace != "" {
		s := name + " was born"
		if birthDate != "" {
			s += " " + narrativeDate(birthDate)
		}
		if birthPlace != "" {
			s += " in " + birthPlace
		}
		sentences = append(sentences, s+".")
	}

	// Parents
	parentIDs := findParentIDs(personID, archive)
	if len(parentIDs) > 0 {
		var parentNames []string
		for _, pid := range parentIDs {
			if p, ok := archive.Persons[pid]; ok && p != nil {
				parentNames = append(parentNames, extractPersonName(p))
			}
		}
		if len(parentNames) > 0 {
			s := subject + " was the child of " + joinNames(parentNames)
			sentences = append(sentences, s+".")
		}
	}

	// Marriages
	spouses := findSpouses(personID, archive)
	for _, sp := range spouses {
		s := subject + " married " + sp.PersonName
		if sp.MarriageDate != "" {
			s += " " + narrativeDate(sp.MarriageDate)
		}
		if sp.MarriagePlace != "" {
			s += " in " + sp.MarriagePlace
		}
		sentences = append(sentences, s+".")
	}

	// Children
	childIDs := findChildIDs(personID, archive)
	if len(childIDs) > 0 {
		var childNames []string
		for _, cid := range childIDs {
			if child, ok := archive.Persons[cid]; ok && child != nil {
				name := extractPersonName(child)
				// Use given name only for brevity
				if parts := strings.Fields(name); len(parts) > 0 {
					childNames = append(childNames, parts[0])
				} else {
					childNames = append(childNames, name)
				}
			}
		}
		if len(childNames) > 0 {
			count := numberWord(len(childNames))
			childWord := "children"
			if len(childNames) == 1 {
				childWord = "child"
			}
			s := fmt.Sprintf("%s had %s %s: %s", subject, count, childWord, joinNames(childNames))
			sentences = append(sentences, s+".")
		}
	}

	// Notable events (first of each type)
	notableTypes := []string{"immigration", "naturalization", "military_service"}
	for _, evType := range notableTypes {
		evDate, evPlace := findEventDatePlace(personID, evType, archive)
		if evDate == "" && evPlace == "" {
			continue
		}
		label := strings.ToLower(snakeCaseToTitle(evType))
		s := possessive + " " + label + " was recorded"
		if evDate != "" {
			s += " " + narrativeDate(evDate)
		}
		if evPlace != "" {
			s += " in " + evPlace
		}
		// Capitalize the first letter of possessive
		s = strings.ToUpper(s[:1]) + s[1:]
		sentences = append(sentences, s+".")
	}

	// Death
	deathDate, deathPlace := findEventDatePlace(personID, "death", archive)
	if deathDate != "" || deathPlace != "" {
		s := subject + " died"
		if deathDate != "" {
			s += " " + narrativeDate(deathDate)
		}
		if deathPlace != "" {
			s += " in " + deathPlace
		}
		sentences = append(sentences, s+".")
	}

	return strings.Join(sentences, " ")
}

// findEventDatePlace returns the date and place for the first event of a given type.
func findEventDatePlace(personID, eventType string, archive *glxlib.GLXFile) (string, string) {
	ids := sortedKeys(archive.Events)
	for _, id := range ids {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		if !strings.EqualFold(event.Type, eventType) {
			continue
		}
		if isPersonParticipant(personID, event) {
			return string(event.Date), resolvePlaceName(event.PlaceID, archive)
		}
	}

	return "", ""
}

// narrativeDate converts a GLX date string to narrative form.
func narrativeDate(date string) string {
	trimmed := strings.TrimSpace(date)
	upper := strings.ToUpper(trimmed)

	switch {
	case strings.HasPrefix(upper, "ABT "):
		return "about " + formatReadableDate(trimmed[4:])
	case strings.HasPrefix(upper, "BEF "):
		return "before " + formatReadableDate(trimmed[4:])
	case strings.HasPrefix(upper, "AFT "):
		return "after " + formatReadableDate(trimmed[4:])
	case strings.HasPrefix(upper, "BET "):
		rest := trimmed[4:]
		if idx := strings.Index(strings.ToUpper(rest), " AND "); idx >= 0 {
			from := formatReadableDate(strings.TrimSpace(rest[:idx]))
			to := formatReadableDate(strings.TrimSpace(rest[idx+5:]))
			return "between " + from + " and " + to
		}
		return "between " + rest
	default:
		readable := formatReadableDate(trimmed)
		if isFullDate(trimmed) {
			return "on " + readable
		}
		return "in " + readable
	}
}

// pronounFor returns subject ("He"/"She"/"They") and possessive ("his"/"her"/"their")
// pronouns based on the person's gender property.
func pronounFor(person *glxlib.Person) (subject, possessive string) {
	gender := strings.ToLower(propertyString(person.Properties, "gender"))
	if gender == "" {
		gender = strings.ToLower(propertyString(person.Properties, "sex"))
	}

	switch gender {
	case "male":
		return "He", "his"
	case "female":
		return "She", "her"
	default:
		return "They", "their"
	}
}

// numberWord returns the English word for small numbers, or the numeral for larger ones.
func numberWord(n int) string {
	words := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
	if n >= 0 && n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%d", n)
}

// joinNames joins names with commas and "and" before the last.
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}

// ============================================================================
// Display helpers
// ============================================================================

// sectionHeader returns a formatted section header line.
func sectionHeader(title string) string {
	const width = 50
	prefix := "── " + title + " "
	remaining := width - utf8.RuneCountInString(prefix)
	if remaining < 2 {
		remaining = 2
	}

	return prefix + strings.Repeat("─", remaining)
}

// displayOrDash returns "—" for empty strings.
func displayOrDash(s string) string {
	if s == "" {
		return "—"
	}

	return s
}

// snakeCaseToTitle converts "snake_case" to "Title Case".
func snakeCaseToTitle(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}

	return strings.Join(parts, " ")
}
