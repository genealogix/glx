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
		return fmt.Errorf("expected INDI record, got %s", indiRecord.Tag)
	}

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			conv.Logger.LogException(
				indiRecord.Line,
				indiRecord.Tag,
				indiRecord.XRef,
				"convertIndividual",
				fmt.Errorf("panic: %v", r),
				map[string]any{
					"record": indiRecord,
				},
			)
			conv.addError(indiRecord.Line, GedcomTagIndi, fmt.Sprintf("Panic during conversion: %v", r))
		}
	}()

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

			// Store name in person properties for quick access
			if parsedName.GivenName != "" {
				person.Properties[PersonPropertyGivenName] = parsedName.GivenName
			}
			if parsedName.Surname != "" {
				person.Properties[PersonPropertyFamilyName] = parsedName.Surname
			}
			if parsedName.Prefix != "" {
				person.Properties[PersonPropertyNamePrefix] = parsedName.Prefix
			}
			if parsedName.Nickname != "" {
				person.Properties[PersonPropertyNickname] = parsedName.Nickname
			}
			if parsedName.SurnamePrefix != "" {
				person.Properties[PersonPropertySurnamePrefix] = parsedName.SurnamePrefix
			}
			if parsedName.Suffix != "" {
				person.Properties[PersonPropertyNameSuffix] = parsedName.Suffix
			}

			// Create name assertions (with evidence/citations)
			if err := createNameAssertions(personID, parsedName, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, GedcomTagName, err.Error())
			}

		case GedcomTagSex:
			// Gender mapping
			gender := mapGEDCOMSex(sub.Value)
			person.Properties[PersonPropertyGender] = gender

			// Create assertion
			createPropertyAssertion(personID, PersonPropertyGender, gender, sub, conv)

		case GedcomTagBirt, GedcomTagChr, GedcomTagDeat, GedcomTagBuri, GedcomTagCrem, GedcomTagAdop, GedcomTagBapm, GedcomTagBarm, GedcomTagBasm,
			GedcomTagBles, GedcomTagChra, GedcomTagConf, GedcomTagFcom, GedcomTagOrdn, GedcomTagNatu, GedcomTagEmig, GedcomTagImmi, GedcomTagCens,
			GedcomTagProb, GedcomTagWill, GedcomTagGrad, GedcomTagReti:
			// Convert vital/individual event
			if err := convertIndividualEvent(personID, person, sub, conv); err != nil {
				conv.addWarning(indiRecord.Line, sub.Tag, err.Error())
			}

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
			// Note: This is different from NPFX (name prefix) which is part of name formatting
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

// createNameAssertions creates assertions for name components
func createNameAssertions(personID string, name PersonName, nameRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Create citations from SOUR tags
	citationIDs := extractCitations(personID, nameRecord, conv)

	// Derive confidence
	confidence := deriveConfidence(citationIDs, conv)

	// Create assertions for each name component
	if name.GivenName != "" {
		assertionID := generateAssertionID(conv)
		conv.GLX.Assertions[assertionID] = &Assertion{
			Subject:    personID,
			Claim:      PersonPropertyGivenName,
			Value:      name.GivenName,
			Confidence: confidence,
			Citations:  citationIDs,
		}
		conv.Stats.AssertionsCreated++
	}

	if name.Surname != "" {
		assertionID := generateAssertionID(conv)
		conv.GLX.Assertions[assertionID] = &Assertion{
			Subject:    personID,
			Claim:      PersonPropertyFamilyName,
			Value:      name.Surname,
			Confidence: confidence,
			Citations:  citationIDs,
		}
		conv.Stats.AssertionsCreated++
	}

	// Store other name components as property assertions
	if name.Prefix != "" {
		createPropertyAssertion(personID, PersonPropertyNamePrefix, name.Prefix, nameRecord, conv)
	}
	if name.Nickname != "" {
		createPropertyAssertion(personID, PersonPropertyNickname, name.Nickname, nameRecord, conv)
	}
	if name.SurnamePrefix != "" {
		createPropertyAssertion(personID, PersonPropertySurnamePrefix, name.SurnamePrefix, nameRecord, conv)
	}
	if name.Suffix != "" {
		createPropertyAssertion(personID, PersonPropertyNameSuffix, name.Suffix, nameRecord, conv)
	}

	return nil
}

// convertIndividualEvent converts individual event tags to GLX events
func convertIndividualEvent(personID string, person *Person, eventRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Map GEDCOM event tag to GLX event type
	eventType := mapGEDCOMEventType(eventRecord.Tag)
	if eventType == "" {
		return fmt.Errorf("unknown event type: %s", eventRecord.Tag)
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
		return "male"
	case "F":
		return "female"
	case "U":
		return "unknown"
	case "X":
		return "other"
	default:
		return "unknown"
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
		GedcomTagCens: EventTypeCensus,
		GedcomTagProb: EventTypeProbate,
		GedcomTagWill: EventTypeWill,
		GedcomTagGrad: EventTypeGraduation,
		GedcomTagReti: EventTypeRetirement,
		GedcomTagResi: EventTypeResidence,
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

// convertResidence converts RESI to residence event or property
func convertResidence(personID string, person *Person, resiRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Check if it has date - if so, create event
	hasDate := false
	for _, sub := range resiRecord.SubRecords {
		if sub.Tag == GedcomTagDate {
			hasDate = true

			break
		}
	}

	if hasDate {
		// Create residence event
		return convertIndividualEvent(personID, person, resiRecord, conv)
	}

	// Otherwise, extract place as property
	for _, sub := range resiRecord.SubRecords {
		if sub.Tag == GedcomTagPlac {
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				placeID, _ := buildPlaceHierarchy(hierarchy, conv)
				if placeID != "" {
					createPropertyAssertion(personID, PersonPropertyResidence, placeID, resiRecord, conv)

					return nil
				}
			}
		}
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
		Subject:    personID,
		Claim:      "no_" + eventType,
		Value:      "true", // Negative assertion (NO tag from GEDCOM 7.0)
		Confidence: ConfidenceLevelHigh, // Negative assertions are typically certain
		Citations:  citationIDs,
	}
	conv.Stats.AssertionsCreated++

	return nil
}
