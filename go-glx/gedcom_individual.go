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
	"strings"
)

// convertIndividual converts a GEDCOM INDI record to a GLX Person
//
//nolint:gocognit,gocyclo
func convertIndividual(indiRecord *GEDCOMRecord, conv *ConversionContext) error {
	if indiRecord.Tag != GedcomTagIndi {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedRecordType, GedcomTagIndi, indiRecord.Tag)
	}

	// Generate person ID
	personID := generatePersonID(conv)
	conv.PersonIDMap[indiRecord.XRef] = personID

	conv.Logger.LogInfof("Converting INDI %s -> %s", indiRecord.XRef, personID)

	// Create person entity
	person := &Person{
		Properties: make(map[string]any),
	}

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	if propertyKey, ok := conv.GEDCOMIndex.PersonProperties[GedcomTagExid]; ok {
		extractExternalIDs(indiRecord, propertyKey, person.Properties)
	}

	// Process all subrecords
	for _, sub := range indiRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagName:
			// Parse name
			nameSubstructure := extractNameSubstructure(sub)
			parsedName := parseGEDCOMName(sub.Value, nameSubstructure)

			// Store unified name property with optional fields
			fullName := parsedName.FormatFullName()
			if fullName != "" {
				nameValue := map[string]any{
					"value": fullName,
				}
				// Only add fields if explicitly present in GEDCOM substructure
				// We don't infer fields from parsing the name string
				fields := nameSubstructure.ToFields()
				if fields != nil {
					nameValue["fields"] = fields
				}
				appendNameProperty(person, nameValue)
			}

			// Create name assertion (with evidence/citations)
			createNameAssertion(personID, parsedName, sub, conv)

		case GedcomTagSex:
			// Gender mapping
			gender := mapGEDCOMSex(sub.Value)
			person.Properties[PersonPropertyGender] = gender

			// Create assertion
			createPropertyAssertion(personID, PersonPropertyGender, gender, sub, conv)

		case GedcomTagBirt, GedcomTagChr, GedcomTagDeat, GedcomTagBuri, GedcomTagCrem, GedcomTagAdop, GedcomTagBapm, GedcomTagBarm, GedcomTagBatm, GedcomTagBasm,
			GedcomTagBles, GedcomTagChra, GedcomTagConf, GedcomTagFcom, GedcomTagOrdn, GedcomTagNatu, GedcomTagEmig, GedcomTagImmi,
			GedcomTagProb, GedcomTagWill, GedcomTagGrad, GedcomTagReti, GedcomTagEven:
			// Convert vital/individual event
			if err := convertIndividualEvent(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, sub.Tag, err.Error())
			}

		case GedcomTagCens:
			convertCensus(personID, person, sub, conv)

		case GedcomTagOccu, GedcomTagReli, GedcomTagEduc, GedcomTagNati, GedcomTagCast, GedcomTagSsn:
			// Simple person properties - resolved via vocabulary index
			handlePersonPropertyTag(personID, person, sub.Tag, sub, conv)

		case GedcomTagResi:
			// Residence - convert to event or property
			convertResidence(personID, person, sub, conv)

		case GedcomTagTitl:
			// Title of nobility, rank, or honor (e.g., Dr., Sir, Baron)
			// May have CONT/CONC for long titles - needs extractTextWithContinuation
			// May have DATE sub-records (e.g., habsburg file) - preserve as temporal list
			titleText := extractTextWithContinuation(sub)
			if titleText != "" {
				if propertyKey, ok := conv.GEDCOMIndex.PersonProperties[sub.Tag]; ok {
					convertPropertyWithDate(personID, person, propertyKey, titleText, sub, conv)
				}
			}

		case GedcomTagFact:
			// Generic fact - convert to property or event
			if err := convertFact(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagFact, err.Error())
			}

		case GedcomTagNote:
			// Notes — store in struct field, not Properties
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				if person.Notes != "" {
					person.Notes += "\n\n" + noteText
				} else {
					person.Notes = noteText
				}
			}

		case GedcomTagSour:
			// Source citation - handled in property/event conversions
			// Citations are extracted when creating assertions

		case GedcomTagObje:
			handleOBJE(sub, person.Properties, conv)

		case GedcomTagFamc:
			// Family as child - defer for family processing
			// Extract PEDI (pedigree linkage) if present
			pedigreeType := ""
			for _, pediSub := range sub.SubRecords {
				if pediSub.Tag == GedcomTagPedi {
					pedigreeType = strings.ToLower(pediSub.Value)

					break
				}
			}
			conv.DeferredFamilyLinks = append(conv.DeferredFamilyLinks, &FamilyLink{
				PersonID:     personID,
				FamilyRef:    sub.Value,
				LinkType:     ParticipantRoleChild,
				PedigreeType: pedigreeType,
			})

		case GedcomTagFams:
			// Family as spouse - defer for family processing
			conv.DeferredFamilyLinks = append(conv.DeferredFamilyLinks, &FamilyLink{
				PersonID:  personID,
				FamilyRef: sub.Value,
				LinkType:  ParticipantRoleSpouse,
			})

		case GedcomTagNo:
			// Negative assertion (GEDCOM 7.0)
			convertNegativeAssertion(personID, sub, conv)

		default:
			if isExtensionTag(sub.Tag) {
				conv.addWarning(sub.Line, sub.Tag, "Extension tag not stored")
			} else if sub.Value != "" && len(sub.Tag) > 0 {
				// Administrative/reference tags - store as properties if they have values
				propKey := strings.ToLower(sub.Tag)
				person.Properties[propKey] = sub.Value
			}
		}
	}

	// Store person
	conv.GLX.Persons[personID] = person
	conv.Stats.PersonsCreated++

	return nil
}

// extractNameSubstructure extracts NAME substructure fields
func extractNameSubstructure(nameRecord *GEDCOMRecord) *NameSubstructure {
	ns := &NameSubstructure{}

	for _, sub := range nameRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagType:
			ns.TYPE = strings.ToLower(sub.Value)
		case GedcomTagNpfx:
			ns.NPFX = sub.Value
		case GedcomTagGivn:
			ns.GIVN = sub.Value
		case GedcomTagNick:
			ns.NICK = sub.Value
		case GedcomTagSpfx:
			ns.SPFX = sub.Value
		case GedcomTagSurn:
			ns.SURN = sub.Value
		case GedcomTagNsfx:
			ns.NSFX = sub.Value
		}
	}

	return ns
}

// createNameAssertion creates an assertion for the name property, but only if there are citations.
// Assertions without citations are not meaningful - the name is already stored on the person entity.
func createNameAssertion(personID string, name PersonName, nameRecord *GEDCOMRecord, conv *ConversionContext) {
	fullName := name.FormatFullName()
	if fullName == "" {
		return
	}

	// Extract evidence from SOUR tags
	refs := extractEvidence(nameRecord, conv)

	// Only create assertion if there is evidence to back it up
	if !refs.hasEvidence() {
		return
	}

	// Create single assertion for the name
	assertionID := generateAssertionID(conv)
	conv.GLX.Assertions[assertionID] = &Assertion{
		Subject:   EntityRef{Person: personID},
		Property:  PersonPropertyName,
		Value:     fullName,
		Sources:   refs.SourceIDs,
		Citations: refs.CitationIDs,
	}
	conv.Stats.AssertionsCreated++
}

// convertIndividualEvent converts individual event tags to GLX events
func convertIndividualEvent(personID string, person *Person, eventRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Map GEDCOM event tag to GLX event type
	eventType := mapGEDCOMEventType(eventRecord.Tag, conv.GEDCOMIndex)
	if eventType == "" {
		return fmt.Errorf("%w: %s", ErrUnknownEventType, eventRecord.Tag)
	}

	// Generate event ID
	eventID := generateEventID(conv)

	// Create event
	event := &Event{
		Type:       eventType,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	eventPlace := extractEventDetails(eventID, eventRecord, event, conv, true)
	eventDate := event.Date

	// Process individual-specific tags
	for _, sub := range eventRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagAge:
			// Age at event
			if propertyKey, ok := conv.GEDCOMIndex.EventProperties[sub.Tag]; ok {
				event.Properties[propertyKey] = sub.Value
			}

		case GedcomTagCaus:
			// Cause
			if propertyKey, ok := conv.GEDCOMIndex.EventProperties[sub.Tag]; ok {
				event.Properties[propertyKey] = sub.Value
			}

		case GedcomTagType:
			// Event subtype
			if propertyKey, ok := conv.GEDCOMIndex.EventProperties[sub.Tag]; ok {
				event.Properties[propertyKey] = sub.Value
			}
		}
	}

	// Add participant to event
	event.Participants = []Participant{
		{
			Person: personID,
			Role:   ParticipantRolePrincipal,
		},
	}

	// Generate event title
	event.Title = GenerateEventTitle(eventType, []string{PersonDisplayName(person)}, event.Date)

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Create event assertions for birth/death date and place
	if (eventType == EventTypeBirth || eventType == EventTypeDeath) && eventDate != "" {
		createEventAssertion(eventID, "date", eventDate, eventRecord, conv)
		if eventPlace != "" {
			createEventAssertion(eventID, "place", eventPlace, eventRecord, conv)
		}
	}

	return nil
}

// mapGEDCOMSex maps GEDCOM sex values to GLX gender
func mapGEDCOMSex(sex string) string {
	switch strings.ToUpper(sex) {
	case "M":
		return GenderMale
	case "F":
		return GenderFemale
	case "U":
		return GenderUnknown
	case "X":
		return GenderOther
	default:
		// Preserve unrecognized values as lowercase so the data is not lost.
		// Validation will warn about unknown gender types. Fixes #520.
		// Note: GEDCOM export maps unknown values to SEX U; true roundtrip
		// requires a custom gender_types vocabulary entry with a gedcom: field.
		trimmed := strings.ToLower(strings.TrimSpace(sex))
		if trimmed == "" {
			return GenderUnknown
		}
		return trimmed
	}
}

// convertPropertyWithDate stores a person property, using a temporal list item
// with a date field when the GEDCOM record has a DATE sub-record.
func convertPropertyWithDate(personID string, person *Person, propertyKey, value string, record *GEDCOMRecord, conv *ConversionContext) {
	var dateStr, placeStr string
	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			dateStr = string(parseGEDCOMDate(sub.Value))
		case GedcomTagPlac:
			placeStr = sub.Value
		}
	}

	itemMap := map[string]any{"value": value}
	if dateStr != "" {
		itemMap["date"] = dateStr
	}
	if placeStr != "" {
		itemMap["place"] = placeStr
	}
	appendTemporalProperty(person, propertyKey, itemMap)
	createPropertyAssertion(personID, propertyKey, value, record, conv)
}

// appendTemporalProperty appends an item to a person property, creating a temporal
// list when multiple values exist.
func appendTemporalProperty(person *Person, propertyKey string, item any) {
	if existing, exists := person.Properties[propertyKey]; exists {
		switch v := existing.(type) {
		case []any:
			person.Properties[propertyKey] = append(v, item)
		default:
			person.Properties[propertyKey] = []any{v, item}
		}
	} else {
		person.Properties[propertyKey] = item
	}
}

// handlePersonPropertyTag processes a GEDCOM tag that maps to a simple person property
// via the vocabulary index. Returns true if the tag was handled.
func handlePersonPropertyTag(personID string, person *Person, tag string, record *GEDCOMRecord, conv *ConversionContext) bool {
	propertyKey, ok := conv.GEDCOMIndex.PersonProperties[tag]
	if !ok {
		return false
	}

	value := record.Value
	if value == "" {
		// Some GEDCOM files store the value in PLAC sub-record (e.g., habsburg OCCU)
		for _, sub := range record.SubRecords {
			if sub.Tag == GedcomTagPlac && sub.Value != "" {
				value = sub.Value
				break
			}
		}
		if value == "" {
			return true
		}
	}

	// Append to list if property already exists (e.g., multiple OCCU)
	// Wrap in {value: ...} objects for temporal list compatibility
	if existing, exists := person.Properties[propertyKey]; exists {
		newItem := map[string]any{"value": value}
		switch v := existing.(type) {
		case []any:
			person.Properties[propertyKey] = append(v, newItem)
		case string:
			// Convert existing simple string to temporal list
			person.Properties[propertyKey] = []any{
				map[string]any{"value": v},
				newItem,
			}
		default:
			// Existing is already a map or other type, wrap in list
			person.Properties[propertyKey] = []any{v, newItem}
		}
	} else {
		person.Properties[propertyKey] = value
	}
	createPropertyAssertion(personID, propertyKey, value, record, conv)

	// Handle OBJE subrecords on person property tags (e.g., OCCU with linked media)
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagObje {
			handleOBJE(sub, person.Properties, conv)
		}
	}

	return true
}

// mapGEDCOMEventType maps GEDCOM event tags to GLX event types using the vocabulary index.
func mapGEDCOMEventType(tag string, gedcomIndex *GEDCOMIndex) string {
	if eventType, ok := gedcomIndex.EventTypes[tag]; ok {
		return eventType
	}

	return strings.ToLower(tag)
}

// buildPlaceHierarchyFromAddress builds a place hierarchy from ADDR subfields
// when no PLAC field is provided. Returns nil if insufficient data.
func buildPlaceHierarchyFromAddress(addrRecord *GEDCOMRecord) *PlaceHierarchy {
	var city, state, country string

	for _, sub := range addrRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagAdr2:
			// Often contains city/locality name
			if city == "" && sub.Value != "" {
				city = sub.Value
			}
		case GedcomTagCity:
			// Explicit city field overrides ADR2
			if sub.Value != "" {
				city = sub.Value
			}
		case GedcomTagStae:
			state = sub.Value
		case GedcomTagCtry:
			country = sub.Value
		}
	}

	// Build hierarchy from specific to general
	var components []string
	if city != "" {
		components = append(components, city)
	}
	if state != "" {
		components = append(components, state)
	}
	if country != "" {
		components = append(components, country)
	}

	// Need at least one component to create a place
	if len(components) == 0 {
		return nil
	}

	return &PlaceHierarchy{
		Components: components,
	}
}

// appendNameProperty appends a name value to a person's name property.
// The name property is temporal, so multiple names (birth name, married name, etc.)
// are stored as a list. A single name is stored as a scalar map.
func appendNameProperty(person *Person, nameValue map[string]any) {
	existing, exists := person.Properties[PersonPropertyName]
	if !exists {
		person.Properties[PersonPropertyName] = nameValue
		return
	}
	// Convert existing scalar to list and append
	if existingList, ok := existing.([]any); ok {
		person.Properties[PersonPropertyName] = append(existingList, nameValue)
	} else {
		person.Properties[PersonPropertyName] = []any{existing, nameValue}
	}
}

// appendResidence appends a residence value to a person's residence property.
// The value may be a temporal map (with date) or a bare place ID string.
// If the property already exists, it is converted to/appended to a list.
// If the property does not exist, a bare value is stored directly (scalar for
// a single undated entry, single-element list for a dated entry).
func appendResidence(person *Person, value any) {
	existing, exists := person.Properties[PersonPropertyResidence]
	if !exists {
		// Temporal entries (maps with date) always start as a list
		if _, isMap := value.(map[string]any); isMap {
			person.Properties[PersonPropertyResidence] = []any{value}
		} else {
			person.Properties[PersonPropertyResidence] = value
		}
		return
	}
	if existingList, ok := existing.([]any); ok {
		person.Properties[PersonPropertyResidence] = append(existingList, value)
	} else {
		person.Properties[PersonPropertyResidence] = []any{existing, value}
	}
}

// convertResidence converts RESI to residence temporal property on person
func convertResidence(personID string, person *Person, resiRecord *GEDCOMRecord, conv *ConversionContext) {
	var placeID string
	var dateStr string

	for _, sub := range resiRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				placeID = buildPlaceHierarchy(hierarchy, conv)
			}
		case GedcomTagDate:
			dateStr = string(parseGEDCOMDate(sub.Value))
		}
	}

	if placeID != "" {
		if dateStr != "" {
			appendResidence(person, map[string]any{
				"value": placeID,
				"date":  dateStr,
			})
		} else {
			appendResidence(person, placeID)
		}

		createPropertyAssertion(personID, PersonPropertyResidence, placeID, resiRecord, conv)
	}
}

// censusData holds extracted data from a GEDCOM CENS record.
// Shared across individual and family census conversion so that
// source/citation creation happens once per CENS record.
type censusData struct {
	dateStr  string
	placeID  string
	evidence evidenceRefs
	mediaIDs []string
}

// convertCensus converts a GEDCOM CENS record to GLX citations and temporal properties.
// Census records are treated as evidence sources, not events. Each CENS record produces:
//   - A synthetic census Source + Citation (when no SOUR sub-records exist)
//   - OR uses existing citations from SOUR sub-records
//   - A temporal residence property (when PLAC is present)
//   - An assertion for residence backed by citations (when PLAC and citations exist)
func convertCensus(personID string, person *Person, censRecord *GEDCOMRecord, conv *ConversionContext) {
	data := extractCensusData(censRecord, conv)
	applyCensusData(personID, person, data, conv)
}

// extractCensusData extracts all data from a CENS record and creates source/citation entities.
// This is separated from applyCensusData so that family-level CENS can extract once and apply
// to both spouses without creating duplicate sources.
//
//nolint:gocognit,gocyclo
func extractCensusData(censRecord *GEDCOMRecord, conv *ConversionContext) censusData {
	var dateStr string
	var placeID string
	var censusType string
	var noteText string
	var mediaIDs []string

	for _, sub := range censRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			dateStr = string(parseGEDCOMDate(sub.Value))
		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon
				pid := buildPlaceHierarchy(hierarchy, conv)
				if pid != "" {
					placeID = pid
				}
			}
		case GedcomTagAddr:
			// If no PLAC was provided, try to build place from ADDR subfields
			if placeID == "" && len(sub.SubRecords) > 0 {
				hierarchy := buildPlaceHierarchyFromAddress(sub)
				if hierarchy != nil {
					pid := buildPlaceHierarchy(hierarchy, conv)
					if pid != "" {
						placeID = pid
					}
				}
			}
		case GedcomTagType:
			censusType = sub.Value
		case GedcomTagNote:
			noteText = extractNoteText(sub, conv)
		case GedcomTagSour:
			// Handled by extractEvidence below
		case GedcomTagObje:
			if mediaID := resolveOBJE(sub, conv); mediaID != "" {
				mediaIDs = append(mediaIDs, mediaID)
			}
		default:
			if sub.Tag != "" {
				conv.addWarning(sub.Line, sub.Tag, "Unhandled CENS sub-tag")
			}
		}
	}

	// Extract evidence from any SOUR sub-records
	refs := extractEvidence(censRecord, conv)

	// When SOUR sub-records exist but we also have a NOTE, attach the note
	// to existing citations so it's not silently lost (#30).
	// If only bare source references exist (no citations), attach to sources.
	if refs.hasEvidence() && noteText != "" {
		if len(refs.CitationIDs) > 0 {
			for _, citID := range refs.CitationIDs {
				if cit, ok := conv.GLX.Citations[citID]; ok {
					if cit.Notes == "" {
						cit.Notes = noteText
					} else {
						cit.Notes += "\n\n" + noteText
					}
				}
			}
		} else if len(refs.SourceIDs) > 0 {
			for _, srcID := range refs.SourceIDs {
				if src, ok := conv.GLX.Sources[srcID]; ok {
					if src.Notes == "" {
						src.Notes = noteText
					} else {
						src.Notes += "\n\n" + noteText
					}
				}
			}
		}
	}

	// If no SOUR sub-records, create a synthetic census source
	if !refs.hasEvidence() {
		// Build source title from TYPE or DATE
		title := censusType
		if title == "" && dateStr != "" {
			title = "Census of " + dateStr
		}
		if title == "" {
			title = "Census"
		}

		// Create synthetic source
		sourceID := generateSourceID(conv)
		source := &Source{
			Title: title,
			Type:  SourceTypeCensus,
		}
		if dateStr != "" {
			source.Date = DateString(dateStr)
		}
		conv.GLX.Sources[sourceID] = source
		conv.Stats.SourcesCreated++

		// If there's a note, create a citation to hold it; otherwise just reference the source
		if noteText != "" {
			citationID := generateCitationID(conv)
			citation := &Citation{
				SourceID: sourceID,
				Notes:    noteText,
			}
			conv.GLX.Citations[citationID] = citation
			conv.Stats.CitationsCreated++

			refs.CitationIDs = []string{citationID}
		} else {
			refs.SourceIDs = []string{sourceID}
		}
	}

	return censusData{
		dateStr:  dateStr,
		placeID:  placeID,
		evidence: refs,
		mediaIDs: mediaIDs,
	}
}

// applyCensusData applies extracted census data to a person: sets temporal
// residence property and creates assertions backed by citations.
func applyCensusData(personID string, person *Person, data censusData, conv *ConversionContext) {
	// Attach media to census citations and sources
	if len(data.mediaIDs) > 0 {
		for _, citID := range data.evidence.CitationIDs {
			if cit, ok := conv.GLX.Citations[citID]; ok {
				cit.Media = append(cit.Media, data.mediaIDs...)
			}
		}
		for _, srcID := range data.evidence.SourceIDs {
			if src, ok := conv.GLX.Sources[srcID]; ok {
				src.Media = append(src.Media, data.mediaIDs...)
			}
		}
	}

	if data.placeID == "" {
		return
	}

	if data.dateStr != "" {
		appendResidence(person, map[string]any{
			"value": data.placeID,
			"date":  data.dateStr,
		})
	} else {
		appendResidence(person, data.placeID)
	}

	// Create assertion for residence backed by citations
	createPropertyAssertionWithEvidence(personID, PersonPropertyResidence, data.placeID, data.evidence, conv)
}

// convertFact converts generic FACT tag
func convertFact(personID string, person *Person, factRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Extract TYPE to determine what kind of fact
	factType := ""
	for _, sub := range factRecord.SubRecords {
		if sub.Tag == GedcomTagType {
			factType = sub.Value

			break
		}
	}

	// If it's a recognized property type, create property assertion
	if factType != "" && factRecord.Value != "" {
		propKey := strings.ToLower(strings.ReplaceAll(factType, " ", "_"))
		createPropertyAssertion(personID, propKey, factRecord.Value, factRecord, conv)

		return nil
	}

	// Otherwise treat as generic event if it has date/place
	hasDateOrPlace := false
	for _, sub := range factRecord.SubRecords {
		if sub.Tag == GedcomTagDate || sub.Tag == GedcomTagPlac {
			hasDateOrPlace = true

			break
		}
	}

	if hasDateOrPlace {
		return convertIndividualEvent(personID, person, factRecord, conv)
	}

	return nil
}

// convertNegativeAssertion converts GEDCOM 7.0 NO tag (negative assertion).
// Only creates an assertion if there are citations to back up the property.
func convertNegativeAssertion(personID string, noRecord *GEDCOMRecord, conv *ConversionContext) {
	// NO tag indicates something did NOT happen
	eventType := mapGEDCOMEventType(noRecord.Value, conv.GEDCOMIndex)

	refs := extractEvidence(noRecord, conv)

	// Only create assertion if there is evidence to back it up
	if !refs.hasEvidence() {
		return
	}

	assertionID := generateAssertionID(conv)
	conv.GLX.Assertions[assertionID] = &Assertion{
		Subject:   EntityRef{Person: personID},
		Property:  "no_" + eventType,
		Value:     "true", // Negative assertion (NO tag from GEDCOM 7.0)
		Sources:   refs.SourceIDs,
		Citations: refs.CitationIDs,
	}
	conv.Stats.AssertionsCreated++
}
