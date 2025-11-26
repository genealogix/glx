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

package lib

import (
	"fmt"
	"strings"
)

// convertIndividual converts a GEDCOM INDI record to a GLX Person
func convertIndividual(indiRecord *GEDCOMRecord, conv *ConversionContext) error {
	if indiRecord.Tag != GedcomTagIndi {
		return fmt.Errorf("%w: expected INDI, got %s", ErrUnexpectedRecordType, indiRecord.Tag)
	}

	// Generate person ID
	personID := generatePersonID(conv)
	conv.PersonIDMap[indiRecord.XRef] = personID

	conv.Logger.LogInfo(fmt.Sprintf("Converting INDI %s -> %s", indiRecord.XRef, personID))

	// Create person entity
	person := &Person{
		Properties: make(map[string]any),
	}

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	exids := extractExternalIDs(indiRecord)
	if len(exids) > 0 {
		person.Properties[PersonPropertyExternalIDs] = exids
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
				person.Properties[PersonPropertyName] = nameValue
			}

			// Create name assertion (with evidence/citations)
			if err := createNameAssertion(personID, parsedName, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagName, err.Error())
			}

		case GedcomTagSex:
			// Gender mapping
			gender := mapGEDCOMSex(sub.Value)
			person.Properties[PersonPropertyGender] = gender

			// Create assertion
			createPropertyAssertion(personID, PersonPropertyGender, gender, sub, conv)

		case GedcomTagBirt, GedcomTagChr, GedcomTagDeat, GedcomTagBuri, GedcomTagCrem, GedcomTagAdop, GedcomTagBapm, GedcomTagBarm, GedcomTagBasm,
			GedcomTagBles, GedcomTagChra, GedcomTagConf, GedcomTagFcom, GedcomTagOrdn, GedcomTagNatu, GedcomTagEmig, GedcomTagImmi,
			GedcomTagProb, GedcomTagWill, GedcomTagGrad, GedcomTagReti:
			// Convert vital/individual event
			if err := convertIndividualEvent(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, sub.Tag, err.Error())
			}

		case GedcomTagCens:
			// TODO: Census is a source/citation, not an event. See todo.md
			// For now, skip census records - they should be converted to citations
			// that support assertions about person attributes.

		case GedcomTagOccu:
			// Occupation
			if sub.Value != "" {
				person.Properties[PersonPropertyOccupation] = sub.Value
				createPropertyAssertion(personID, PersonPropertyOccupation, sub.Value, sub, conv)
			}

		case GedcomTagResi:
			// Residence - convert to event or property
			if err := convertResidence(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagResi, err.Error())
			}

		case GedcomTagReli:
			// Religion
			if sub.Value != "" {
				person.Properties[PersonPropertyReligion] = sub.Value
				createPropertyAssertion(personID, PersonPropertyReligion, sub.Value, sub, conv)
			}

		case GedcomTagEduc:
			// Education
			if sub.Value != "" {
				person.Properties[PersonPropertyEducation] = sub.Value
				createPropertyAssertion(personID, PersonPropertyEducation, sub.Value, sub, conv)
			}

		case GedcomTagNati:
			// Nationality
			if sub.Value != "" {
				person.Properties[PersonPropertyNationality] = sub.Value
				createPropertyAssertion(personID, PersonPropertyNationality, sub.Value, sub, conv)
			}

		case GedcomTagCast:
			// Caste/tribe
			if sub.Value != "" {
				person.Properties[PersonPropertyCaste] = sub.Value
				createPropertyAssertion(personID, PersonPropertyCaste, sub.Value, sub, conv)
			}

		case GedcomTagSsn:
			// Social security number
			if sub.Value != "" {
				person.Properties[PersonPropertySSN] = sub.Value
				createPropertyAssertion(personID, PersonPropertySSN, sub.Value, sub, conv)
			}

		case GedcomTagTitl:
			// Title of nobility, rank, or honor (e.g., Dr., Sir, Baron)
			if sub.Value != "" {
				person.Properties[PersonPropertyTitle] = sub.Value
				createPropertyAssertion(personID, PersonPropertyTitle, sub.Value, sub, conv)
			}

		case GedcomTagFact:
			// Generic fact - convert to property or event
			if err := convertFact(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagFact, err.Error())
			}

		case GedcomTagNote:
			// Notes
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				if notes, ok := person.Properties[PropertyNotes].(string); ok {
					person.Properties[PropertyNotes] = notes + "\n\n" + noteText
				} else {
					person.Properties[PropertyNotes] = noteText
				}
			}

		case GedcomTagSour:
			// Source citation - handled in property/event conversions
			// Citations are extracted when creating assertions

		case GedcomTagObje:
			// Media object - will be implemented when media converter is done
			conv.addWarning(indiRecord.Line, GedcomTagObje, "Media linking not yet implemented")

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
			if err := convertNegativeAssertion(personID, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagNo, err.Error())
			}

		default:
			// Administrative/reference tags - store as properties if they have values
			if sub.Value != "" && len(sub.Tag) > 0 {
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

// createNameAssertion creates a single assertion for the unified name property
func createNameAssertion(personID string, name PersonName, nameRecord *GEDCOMRecord, conv *ConversionContext) error {
	fullName := name.FormatFullName()
	if fullName == "" {
		return nil
	}

	// Create citations from SOUR tags
	citationIDs := extractCitations(personID, nameRecord, conv)

	// Create single assertion for the name
	assertionID := generateAssertionID(conv)
	conv.GLX.Assertions[assertionID] = &Assertion{
		Subject:   personID,
		Claim:     PersonPropertyName,
		Value:     fullName,
		Citations: citationIDs,
	}
	conv.Stats.AssertionsCreated++

	return nil
}

// convertIndividualEvent converts individual event tags to GLX events
func convertIndividualEvent(personID string, person *Person, eventRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Map GEDCOM event tag to GLX event type
	eventType := mapGEDCOMEventType(eventRecord.Tag)
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

	// Extract event details
	var eventDate DateString
	var eventPlace string

	for _, sub := range eventRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			eventDate = parseGEDCOMDate(sub.Value)
			if eventDate != "" {
				event.Date = eventDate
			}

		case GedcomTagPlac:
			// Parse place
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				// Extract coordinates from MAP/LATI/LONG subrecords
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon

				placeID, err := buildPlaceHierarchy(hierarchy, conv)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
					eventPlace = placeID
				}
			}

		case GedcomTagAge:
			// Age at event
			event.Properties[PropertyAgeAtEvent] = sub.Value

		case GedcomTagCaus:
			// Cause
			event.Properties[PropertyCause] = sub.Value

		case GedcomTagType:
			// Event subtype
			event.Properties[PropertyEventSubtype] = sub.Value

		case GedcomTagAddr:
			// Address - extract full address including subfields
			addr := extractAddress(sub)
			if addr != "" {
				event.Properties[PropertyAddress] = addr
			}

			// If no PLAC was provided, try to build place from ADDR subfields
			if event.PlaceID == "" && len(sub.SubRecords) > 0 {
				hierarchy := buildPlaceHierarchyFromAddress(sub)
				if hierarchy != nil {
					placeID, err := buildPlaceHierarchy(hierarchy, conv)
					if err == nil && placeID != "" {
						event.PlaceID = placeID
						eventPlace = placeID
					}
				}
			}

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties[PropertyNotes] = noteText
			}

		case GedcomTagSour:
			// Citations handled when creating participations

		case GedcomTagObje:
			// Media - not yet implemented
			conv.addWarning(eventRecord.Line, GedcomTagObje, "Media linking not yet implemented")
		}
	}

	// Add participant to event
	event.Participants = []EventParticipant{
		{
			PersonID: personID,
			Role:     ParticipantRolePrincipal,
		},
	}

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Create property assertions for born_on, died_on, etc.
	// ALSO set person properties directly for quick access
	if eventType == EventTypeBirth && eventDate != "" {
		person.Properties[PersonPropertyBornOn] = eventDate
		createPropertyAssertion(personID, PersonPropertyBornOn, eventDate, eventRecord, conv)
		if eventPlace != "" {
			person.Properties[PersonPropertyBornAt] = eventPlace
			createPropertyAssertion(personID, PersonPropertyBornAt, eventPlace, eventRecord, conv)
		}
	} else if eventType == EventTypeDeath && eventDate != "" {
		person.Properties[PersonPropertyDiedOn] = eventDate
		createPropertyAssertion(personID, PersonPropertyDiedOn, eventDate, eventRecord, conv)
		if eventPlace != "" {
			person.Properties[PersonPropertyDiedAt] = eventPlace
			createPropertyAssertion(personID, PersonPropertyDiedAt, eventPlace, eventRecord, conv)
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
		return "other"
	default:
		return GenderUnknown
	}
}

// mapGEDCOMEventType maps GEDCOM event tags to GLX event types
func mapGEDCOMEventType(tag string) string {
	mapping := map[string]string{
		GedcomTagBirt: EventTypeBirth,
		GedcomTagChr:  EventTypeChristening,
		GedcomTagDeat: EventTypeDeath,
		GedcomTagBuri: EventTypeBurial,
		GedcomTagCrem: EventTypeCremation,
		GedcomTagAdop: EventTypeAdoption,
		GedcomTagBapm: EventTypeBaptism,
		GedcomTagBarm: EventTypeBarMitzvah,
		GedcomTagBasm: EventTypeBasMitzvah,
		GedcomTagBles: EventTypeBlessing,
		GedcomTagChra: EventTypeAdultChristening,
		GedcomTagConf: EventTypeConfirmation,
		GedcomTagFcom: EventTypeFirstCommunion,
		GedcomTagOrdn: EventTypeOrdination,
		GedcomTagNatu: EventTypeNaturalization,
		GedcomTagEmig: EventTypeEmigration,
		GedcomTagImmi: EventTypeImmigration,
		GedcomTagProb: EventTypeProbate,
		GedcomTagWill: EventTypeWill,
		GedcomTagGrad: EventTypeGraduation,
		GedcomTagReti: EventTypeRetirement,
	}

	if eventType, ok := mapping[tag]; ok {
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

// convertResidence converts RESI to residence temporal property on person
func convertResidence(personID string, person *Person, resiRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Extract place and date from RESI record
	var placeID string
	var dateStr string

	for _, sub := range resiRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				placeID, _ = buildPlaceHierarchy(hierarchy, conv)
			}
		case GedcomTagDate:
			dateStr = string(parseGEDCOMDate(sub.Value))
		}
	}

	// If we have a place, create temporal property
	if placeID != "" {
		// Build temporal value with date if present
		if dateStr != "" {
			// Add as temporal property with date
			temporalValue := map[string]any{
				"value": placeID,
				"date":  dateStr,
			}
			// Append to existing residence history or create new
			if existing, ok := person.Properties[PersonPropertyResidence]; ok {
				if existingList, ok := existing.([]any); ok {
					person.Properties[PersonPropertyResidence] = append(existingList, temporalValue)
				} else {
					// Convert single value to list
					person.Properties[PersonPropertyResidence] = []any{existing, temporalValue}
				}
			} else {
				person.Properties[PersonPropertyResidence] = []any{temporalValue}
			}
		} else {
			// No date - just set the place
			person.Properties[PersonPropertyResidence] = placeID
		}

		// Create assertion for the residence
		createPropertyAssertion(personID, PersonPropertyResidence, placeID, resiRecord, conv)
	}

	return nil
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

// convertNegativeAssertion converts GEDCOM 7.0 NO tag (negative assertion)
func convertNegativeAssertion(personID string, noRecord *GEDCOMRecord, conv *ConversionContext) error {
	// NO tag indicates something did NOT happen
	eventType := mapGEDCOMEventType(noRecord.Value)

	citationIDs := extractCitations(personID, noRecord, conv)

	assertionID := generateAssertionID(conv)
	conv.GLX.Assertions[assertionID] = &Assertion{
		Subject:   personID,
		Claim:     "no_" + eventType,
		Value:     "true", // Negative assertion (NO tag from GEDCOM 7.0)
		Citations: citationIDs,
	}
	conv.Stats.AssertionsCreated++

	return nil
}
