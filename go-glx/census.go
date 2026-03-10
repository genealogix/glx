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

package glx

import (
	"fmt"
	"regexp"
	"strings"
)

// CensusTemplate represents a census record template for generating GLX entities.
type CensusTemplate struct {
	Census CensusData `yaml:"census"`
}

// CensusData holds the top-level census record data.
type CensusData struct {
	Year      int              `yaml:"year"`
	Type      string           `yaml:"type,omitempty"` // federal, state
	Date      string           `yaml:"date,omitempty"`
	Location  CensusLocation   `yaml:"location"`
	Source    CensusSourceRef  `yaml:"source"`
	Citation  CensusCitationData `yaml:"citation"`
	Household CensusHousehold  `yaml:"household"`
	FAN       *CensusFAN       `yaml:"fan,omitempty"`
}

// CensusLocation specifies the census enumeration place.
type CensusLocation struct {
	Place   string `yaml:"place,omitempty"`    // Human-readable name, matched against archive
	PlaceID string `yaml:"place_id,omitempty"` // Explicit place ID
}

// CensusSourceRef identifies or defines the census source.
type CensusSourceRef struct {
	Title        string `yaml:"title,omitempty"`         // Source title (auto-generated if omitted)
	SourceID     string `yaml:"source_id,omitempty"`     // Reuse existing source by ID
	RepositoryID string `yaml:"repository_id,omitempty"` // Optional repository reference
	CallNumber   string `yaml:"call_number,omitempty"`
	Notes        string `yaml:"notes,omitempty"`
}

// CensusCitationData holds citation-level detail for the census record.
type CensusCitationData struct {
	Locator        string   `yaml:"locator,omitempty"`
	TextFromSource string   `yaml:"text_from_source,omitempty"`
	CitationText   string   `yaml:"citation_text,omitempty"`
	URL            string   `yaml:"url,omitempty"`
	Accessed       string   `yaml:"accessed,omitempty"`
	MediaIDs       []string `yaml:"media_ids,omitempty"`
}

// CensusHousehold holds the event-level data and household members.
type CensusHousehold struct {
	Title   string                  `yaml:"title,omitempty"` // Event title (auto-generated if omitted)
	Notes   string                  `yaml:"notes,omitempty"`
	Members []CensusHouseholdMember `yaml:"members"`
}

// CensusHouseholdMember represents one person on the census schedule.
type CensusHouseholdMember struct {
	Name         string         `yaml:"name"`
	PersonID     string         `yaml:"person_id,omitempty"`   // Explicit person ID
	Role         string         `yaml:"role,omitempty"`        // Participant role (default: subject)
	Age          *int           `yaml:"age,omitempty"`         // Age at census (pointer to distinguish 0 from absent)
	Sex          string         `yaml:"sex,omitempty"`         // male, female
	Birthplace   string         `yaml:"birthplace,omitempty"`  // Free text or place name
	BirthplaceID string         `yaml:"birthplace_id,omitempty"` // Explicit place ID for birthplace
	Occupation   string         `yaml:"occupation,omitempty"`
	Notes        string         `yaml:"notes,omitempty"`
	Properties   map[string]any `yaml:"properties,omitempty"` // Additional assertions (race, education, etc.)
}

// CensusFAN holds FAN (Friends, Associates, Neighbors) notes.
type CensusFAN struct {
	Notes string `yaml:"notes,omitempty"`
}

// CensusResult holds all entities generated from a census template.
type CensusResult struct {
	Source     map[string]*Source     // 0 or 1 new source
	Citation   map[string]*Citation   // 1 new citation
	Place      map[string]*Place      // 0+ new places
	Event      map[string]*Event      // 1 census event
	Persons    map[string]*Person     // 0+ new persons
	Assertions map[string]*Assertion  // generated assertions

	SourceID     string   // Source ID used (new or existing)
	CitationID   string   // New citation ID
	EventID      string   // New event ID
	PlaceID      string   // Place ID used for the census location
	NewPersonIDs []string // IDs of newly created persons
	MatchedIDs   []string // IDs of matched existing persons
}

// BuildCensusEntities generates GLX entities from a census template.
// existing may be nil if no archive is loaded.
func BuildCensusEntities(template *CensusTemplate, existing *GLXFile) (*CensusResult, error) {
	if err := validateCensusTemplate(template); err != nil {
		return nil, err
	}

	census := &template.Census
	if existing == nil {
		existing = &GLXFile{}
	}

	result := &CensusResult{
		Source:     make(map[string]*Source),
		Citation:   make(map[string]*Citation),
		Place:      make(map[string]*Place),
		Event:      make(map[string]*Event),
		Persons:    make(map[string]*Person),
		Assertions: make(map[string]*Assertion),
	}

	// 1. Resolve or create place
	placeID, err := resolveCensusPlace(census, existing, result)
	if err != nil {
		return nil, err
	}
	result.PlaceID = placeID

	// 2. Resolve or create source
	sourceID, err := resolveCensusSource(census, existing, result)
	if err != nil {
		return nil, err
	}
	result.SourceID = sourceID

	// 3. Create citation (include household surname for uniqueness)
	surname := lastWord(census.Household.Members[0].Name)
	citationID := censusSlugIDWithHousehold("citation", census.Year, census.Location, surname)
	result.CitationID = citationID
	result.Citation[citationID] = buildCensusCitation(census, sourceID)

	// 4. Resolve persons and build participants
	participants, resolvedIDs, err := resolveCensusPersons(census, existing, result)
	if err != nil {
		return nil, err
	}

	// 5. Create census event (include household surname for uniqueness)
	eventID := censusSlugIDWithHousehold("event", census.Year, census.Location, surname)
	result.EventID = eventID
	result.Event[eventID] = buildCensusEvent(census, placeID, participants)

	// 6. Generate assertions for each member (using resolved IDs, not re-derived)
	generateCensusAssertions(census, resolvedIDs, placeID, citationID, existing, result)

	return result, nil
}

// validateCensusTemplate checks for required fields.
func validateCensusTemplate(template *CensusTemplate) error {
	if template == nil {
		return fmt.Errorf("census template is required")
	}
	c := &template.Census
	if c.Year == 0 {
		return fmt.Errorf("census.year is required")
	}
	if c.Location.Place == "" && c.Location.PlaceID == "" {
		return fmt.Errorf("census.location.place or census.location.place_id is required")
	}
	if len(c.Household.Members) == 0 {
		return fmt.Errorf("census.household.members is required (at least one member)")
	}
	for i, m := range c.Household.Members {
		if m.Name == "" {
			return fmt.Errorf("census.household.members[%d].name is required", i)
		}
	}
	return nil
}

// resolveCensusPlace resolves an existing place or creates a new one.
func resolveCensusPlace(census *CensusData, existing *GLXFile, result *CensusResult) (string, error) {
	loc := census.Location

	if loc.PlaceID != "" {
		if existing.Places != nil {
			if _, ok := existing.Places[loc.PlaceID]; ok {
				return loc.PlaceID, nil
			}
		}
		return "", fmt.Errorf("census.location.place_id %q does not exist in the loaded archive", loc.PlaceID)
	}

	// Search by name
	if existing.Places != nil {
		for id, place := range existing.Places {
			if place != nil && strings.EqualFold(place.Name, loc.Place) {
				return id, nil
			}
		}
	}

	// Create new place
	placeID := Slugify("place", loc.Place)
	result.Place[placeID] = &Place{Name: loc.Place}
	return placeID, nil
}

// resolveCensusSource resolves an existing source or creates a new one.
func resolveCensusSource(census *CensusData, existing *GLXFile, result *CensusResult) (string, error) {
	src := census.Source

	if src.SourceID != "" {
		if existing.Sources != nil {
			if _, ok := existing.Sources[src.SourceID]; ok {
				return src.SourceID, nil
			}
		}
		return "", fmt.Errorf("source_id %q not found in archive", src.SourceID)
	}

	title := src.Title
	if title == "" {
		title = fmt.Sprintf("%d U.S. Federal Census — %s", census.Year, census.Location.Place)
	}

	// Search existing sources by title
	if existing.Sources != nil {
		for id, source := range existing.Sources {
			if source != nil && strings.EqualFold(source.Title, title) && source.Type == SourceTypeCensus {
				return id, nil
			}
		}
	}

	// Create new source
	sourceID := censusSlugID("source", census.Year, census.Location)
	newSource := &Source{
		Title: title,
		Type:  SourceTypeCensus,
		Notes: src.Notes,
	}
	if src.RepositoryID != "" {
		newSource.RepositoryID = src.RepositoryID
	}
	if src.CallNumber != "" {
		newSource.Properties = map[string]any{"call_number": src.CallNumber}
	}
	result.Source[sourceID] = newSource
	return sourceID, nil
}

// buildCensusCitation creates the citation entity.
func buildCensusCitation(census *CensusData, sourceID string) *Citation {
	cit := &Citation{
		SourceID: sourceID,
	}

	props := make(map[string]any)
	if census.Citation.Locator != "" {
		props["locator"] = census.Citation.Locator
	}
	if census.Citation.TextFromSource != "" {
		props["text_from_source"] = census.Citation.TextFromSource
	}
	if census.Citation.CitationText != "" {
		// Map citation_text to text_from_source (standard property) as a fallback
		if census.Citation.TextFromSource == "" {
			props["text_from_source"] = census.Citation.CitationText
		}
	}
	if census.Citation.URL != "" {
		props["url"] = census.Citation.URL
	}
	if census.Citation.Accessed != "" {
		props["accessed"] = census.Citation.Accessed
	}
	if len(props) > 0 {
		cit.Properties = props
	}

	if len(census.Citation.MediaIDs) > 0 {
		cit.Media = census.Citation.MediaIDs
	}

	return cit
}

// resolveCensusPersons resolves each household member to an existing or new person.
// Returns participants for the event, and a parallel slice of resolved person IDs
// (one per member, in the same order as census.Household.Members).
func resolveCensusPersons(census *CensusData, existing *GLXFile, result *CensusResult) ([]Participant, []string, error) {
	var participants []Participant
	var resolvedIDs []string

	for _, member := range census.Household.Members {
		personID, isNew, err := resolveCensusPerson(member, existing, result)
		if err != nil {
			return nil, nil, err
		}

		resolvedIDs = append(resolvedIDs, personID)

		if isNew {
			result.NewPersonIDs = append(result.NewPersonIDs, personID)
		} else {
			result.MatchedIDs = append(result.MatchedIDs, personID)
		}

		role := member.Role
		if role == "" {
			role = "subject"
		}

		p := Participant{
			Person: personID,
			Role:   role,
			Notes:  member.Notes,
		}

		// Add age as participant property
		if member.Age != nil {
			p.Properties = map[string]any{"age_at_event": fmt.Sprintf("%d", *member.Age)}
		}

		participants = append(participants, p)
	}

	return participants, resolvedIDs, nil
}

// resolveCensusPerson resolves a single household member.
func resolveCensusPerson(member CensusHouseholdMember, existing *GLXFile, result *CensusResult) (string, bool, error) {
	// Explicit person ID
	if member.PersonID != "" {
		if existing.Persons != nil {
			if _, ok := existing.Persons[member.PersonID]; ok {
				return member.PersonID, false, nil
			}
		}
		// Also check newly created persons in this batch
		if _, ok := result.Persons[member.PersonID]; ok {
			return member.PersonID, false, nil
		}
		return "", false, fmt.Errorf("person_id %q not found in archive", member.PersonID)
	}

	// Search by exact name in existing archive (substring matches are
	// treated as ambiguous — require explicit person_id to disambiguate)
	if existing.Persons != nil {
		var exactMatches []string
		for id, person := range existing.Persons {
			if person == nil {
				continue
			}
			displayName := PersonDisplayName(person)
			if strings.EqualFold(displayName, member.Name) {
				exactMatches = append(exactMatches, id)
			}
		}
		if len(exactMatches) == 1 {
			return exactMatches[0], false, nil
		}
		if len(exactMatches) > 1 {
			return "", false, fmt.Errorf("ambiguous name %q matches %d persons: %s (use person_id to disambiguate)",
				member.Name, len(exactMatches), strings.Join(exactMatches, ", "))
		}
	}

	// Create new person with unique ID
	personID := uniquePersonID(Slugify("person", member.Name), existing, result)

	person := &Person{
		Properties: map[string]any{
			PersonPropertyName: member.Name,
		},
	}
	if member.Sex != "" {
		person.Properties[PersonPropertyGender] = strings.ToLower(member.Sex)
	}

	result.Persons[personID] = person
	return personID, true, nil
}

// buildCensusEvent creates the census event entity.
func buildCensusEvent(census *CensusData, placeID string, participants []Participant) *Event {
	title := census.Household.Title
	if title == "" {
		// Derive surname from first member name
		surname := lastWord(census.Household.Members[0].Name)
		title = fmt.Sprintf("%d Census — %s Household", census.Year, surname)
	}

	date := DateString(census.Date)
	if date == "" {
		date = DateString(fmt.Sprintf("%d", census.Year))
	}

	event := &Event{
		Title:        title,
		Type:         EventTypeCensus,
		Date:         date,
		PlaceID:      placeID,
		Participants: participants,
		Notes:        census.Household.Notes,
	}

	// Append FAN notes
	if census.FAN != nil && census.FAN.Notes != "" {
		if event.Notes != "" {
			event.Notes += "\n\n"
		}
		event.Notes += "FAN — " + census.FAN.Notes
	}

	return event
}

// generateCensusAssertions creates assertions for each household member.
// resolvedIDs contains the actual person ID for each member (resolved during
// person resolution), avoiding re-derivation that could mismatch existing IDs.
func generateCensusAssertions(census *CensusData, resolvedIDs []string, placeID, citationID string, existing *GLXFile, result *CensusResult) {
	yearStr := fmt.Sprintf("%d", census.Year)

	for i, member := range census.Household.Members {
		personID := resolvedIDs[i]

		slug := Slugify("", member.Name)

		// Birth year from age
		if member.Age != nil {
			birthYear := census.Year - *member.Age
			assertionID := fmt.Sprintf("assertion-%s-birth-year-%s", slug, yearStr)
			result.Assertions[assertionID] = &Assertion{
				Subject:    EntityRef{Person: personID},
				Property:   PersonPropertyBornOn,
				Value:      fmt.Sprintf("ABT %d", birthYear),
				Citations:  []string{citationID},
				Confidence: ConfidenceLevelLow,
				Notes:      fmt.Sprintf("Estimated from age %d in %d census. Census ages are frequently off by 1-2 years.", *member.Age, census.Year),
			}
		}

		// Birthplace
		birthplaceRef := member.BirthplaceID
		if birthplaceRef == "" && member.Birthplace != "" {
			// Try to find matching place in existing archive and current batch
			birthplaceRef = resolveBirthplace(member.Birthplace, existing, result)
		}
		if birthplaceRef != "" {
			assertionID := fmt.Sprintf("assertion-%s-birthplace-%s", slug, yearStr)
			result.Assertions[assertionID] = &Assertion{
				Subject:    EntityRef{Person: personID},
				Property:   PersonPropertyBornAt,
				Value:      birthplaceRef,
				Citations:  []string{citationID},
				Confidence: ConfidenceLevelMedium,
				Notes:      fmt.Sprintf("%d census lists birthplace as %q.", census.Year, member.Birthplace),
			}
		}

		// Gender
		if member.Sex != "" {
			assertionID := fmt.Sprintf("assertion-%s-gender-%s", slug, yearStr)
			result.Assertions[assertionID] = &Assertion{
				Subject:    EntityRef{Person: personID},
				Property:   PersonPropertyGender,
				Value:      strings.ToLower(member.Sex),
				Citations:  []string{citationID},
				Confidence: ConfidenceLevelHigh,
				Notes:      fmt.Sprintf("Directly stated in %d census.", census.Year),
			}
		}

		// Occupation
		if member.Occupation != "" {
			assertionID := fmt.Sprintf("assertion-%s-occupation-%s", slug, yearStr)
			result.Assertions[assertionID] = &Assertion{
				Subject:    EntityRef{Person: personID},
				Property:   "occupation",
				Value:      member.Occupation,
				Date:       DateString(yearStr),
				Citations:  []string{citationID},
				Confidence: ConfidenceLevelHigh,
				Notes:      fmt.Sprintf("Directly stated in %d census.", census.Year),
			}
		}

		// Residence
		assertionID := fmt.Sprintf("assertion-%s-residence-%s", slug, yearStr)
		result.Assertions[assertionID] = &Assertion{
			Subject:    EntityRef{Person: personID},
			Property:   PersonPropertyResidence,
			Value:      placeID,
			Date:       DateString(yearStr),
			Citations:  []string{citationID},
			Confidence: ConfidenceLevelHigh,
			Notes:      fmt.Sprintf("Enumerated in %d census.", census.Year),
		}

		// Custom properties
		for prop, val := range member.Properties {
			valStr := fmt.Sprint(val)
			propSlug := Slugify("", prop)
			assertionID := fmt.Sprintf("assertion-%s-%s-%s", slug, propSlug, yearStr)
			result.Assertions[assertionID] = &Assertion{
				Subject:    EntityRef{Person: personID},
				Property:   prop,
				Value:      valStr,
				Date:       DateString(yearStr),
				Citations:  []string{citationID},
				Confidence: ConfidenceLevelHigh,
				Notes:      fmt.Sprintf("Directly stated in %d census.", census.Year),
			}
		}
	}
}

// resolveBirthplace attempts to match a birthplace name to a place ID
// in the existing archive and current batch, or returns the name as-is.
func resolveBirthplace(name string, existing *GLXFile, result *CensusResult) string {
	// Check existing archive places
	if existing != nil && existing.Places != nil {
		for id, place := range existing.Places {
			if place != nil && strings.EqualFold(place.Name, name) {
				return id
			}
		}
	}
	// Check newly created places in current batch
	for id, place := range result.Place {
		if place != nil && strings.EqualFold(place.Name, name) {
			return id
		}
	}
	return name
}

// censusSlugID generates a deterministic entity ID from census data.
func censusSlugID(prefix string, year int, loc CensusLocation) string {
	name := loc.Place
	if name == "" {
		name = loc.PlaceID
	}
	return truncateID(fmt.Sprintf("%s-%d-census-%s", prefix, year, slugifyString(name)))
}

// censusSlugIDWithHousehold generates a deterministic entity ID that includes
// the household surname for uniqueness across multiple households at the same place/year.
func censusSlugIDWithHousehold(prefix string, year int, loc CensusLocation, surname string) string {
	name := loc.Place
	if name == "" {
		name = loc.PlaceID
	}
	return truncateID(fmt.Sprintf("%s-%d-census-%s-%s", prefix, year, slugifyString(name), slugifyString(surname)))
}

// truncateID truncates an entity ID to the 64-character maximum.
func truncateID(id string) string {
	if len(id) <= 64 {
		return id
	}
	return strings.TrimRight(id[:64], "-")
}

// uniquePersonID returns a person ID that doesn't collide with existing archive
// or current batch entries. If baseID already exists, appends an incrementing suffix.
func uniquePersonID(baseID string, existing *GLXFile, result *CensusResult) string {
	candidate := baseID
	for suffix := 2; ; suffix++ {
		existsInArchive := existing != nil && existing.Persons != nil
		if existsInArchive {
			if _, ok := existing.Persons[candidate]; ok {
				candidate = fmt.Sprintf("%s-%d", baseID, suffix)
				continue
			}
		}
		if _, ok := result.Persons[candidate]; ok {
			candidate = fmt.Sprintf("%s-%d", baseID, suffix)
			continue
		}
		return truncateID(candidate)
	}
}

// Slugify generates a deterministic entity ID from a prefix and name.
func Slugify(prefix, name string) string {
	slug := slugifyString(name)
	if prefix == "" {
		return slug
	}
	return prefix + "-" + slug
}

var slugifyNonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// slugifyString converts a name to a URL/ID-safe slug.
func slugifyString(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	slug := slugifyNonAlphanumeric.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "unknown"
	}
	return slug
}

// lastWord returns the last whitespace-delimited word in a string.
func lastWord(s string) string {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return s
	}
	return parts[len(parts)-1]
}
