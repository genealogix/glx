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
func convertIndividual(indiRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if indiRecord.Tag != "INDI" {
		return fmt.Errorf("expected INDI record, got %s", indiRecord.Tag)
	}

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.LogException(
				indiRecord.Line,
				indiRecord.Tag,
				indiRecord.XRef,
				"convertIndividual",
				fmt.Errorf("panic: %v", r),
				map[string]any{
					"record": indiRecord,
				},
			)
			ctx.addError(indiRecord.Line, "INDI", fmt.Sprintf("Panic during conversion: %v", r))
		}
	}()

	// Generate person ID
	personID := generatePersonID(ctx)
	ctx.PersonIDMap[indiRecord.XRef] = personID

	ctx.Logger.LogInfo(fmt.Sprintf("Converting INDI %s -> %s", indiRecord.XRef, personID))

	// Create person entity
	person := &Person{
		Properties: make(map[string]any),
	}

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	exids := extractExternalIDs(indiRecord)
	if len(exids) > 0 {
		person.Properties["external_ids"] = exids
	}

	// Process all subrecords
	for _, sub := range indiRecord.SubRecords {
		switch sub.Tag {
		case "NAME":
			// Parse name
			nameSubstructure := extractNameSubstructure(sub)
			parsedName := parseGEDCOMName(sub.Value, nameSubstructure)

			// Store name in person properties for quick access
			if parsedName.GivenName != "" {
				person.Properties["given_name"] = parsedName.GivenName
			}
			if parsedName.Surname != "" {
				person.Properties["family_name"] = parsedName.Surname
			}
			if parsedName.Prefix != "" {
				person.Properties["name_prefix"] = parsedName.Prefix
			}
			if parsedName.Nickname != "" {
				person.Properties["nickname"] = parsedName.Nickname
			}
			if parsedName.SurnamePrefix != "" {
				person.Properties["surname_prefix"] = parsedName.SurnamePrefix
			}
			if parsedName.Suffix != "" {
				person.Properties["name_suffix"] = parsedName.Suffix
			}

			// Create name assertions (with evidence/citations)
			if err := createNameAssertions(personID, parsedName, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, "NAME", err.Error())
			}

		case "SEX":
			// Gender mapping
			gender := mapGEDCOMSex(sub.Value)
			person.Properties["gender"] = gender

			// Create assertion
			if err := createPropertyAssertion(personID, "gender", gender, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, "SEX", err.Error())
			}

		case "BIRT", "CHR", "DEAT", "BURI", "CREM", "ADOP", "BAPM", "BARM", "BASM",
			"BLES", "CHRA", "CONF", "FCOM", "ORDN", "NATU", "EMIG", "IMMI", "CENS",
			"PROB", "WILL", "GRAD", "RETI":
			// Convert vital/individual event
			if err := convertIndividualEvent(personID, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, sub.Tag, err.Error())
			}

		case "OCCU":
			// Occupation
			if sub.Value != "" {
				person.Properties["occupation"] = sub.Value
				createPropertyAssertion(personID, "occupation", sub.Value, sub, ctx)
			}

		case "RESI":
			// Residence - convert to event or property
			if err := convertResidence(personID, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, "RESI", err.Error())
			}

		case "RELI":
			// Religion
			if sub.Value != "" {
				person.Properties["religion"] = sub.Value
				createPropertyAssertion(personID, "religion", sub.Value, sub, ctx)
			}

		case "EDUC":
			// Education
			if sub.Value != "" {
				person.Properties["education"] = sub.Value
				createPropertyAssertion(personID, "education", sub.Value, sub, ctx)
			}

		case "NATI":
			// Nationality
			if sub.Value != "" {
				person.Properties["nationality"] = sub.Value
				createPropertyAssertion(personID, "nationality", sub.Value, sub, ctx)
			}

		case "CAST":
			// Caste/tribe
			if sub.Value != "" {
				person.Properties["caste"] = sub.Value
				createPropertyAssertion(personID, "caste", sub.Value, sub, ctx)
			}

		case "SSN":
			// Social security number
			if sub.Value != "" {
				person.Properties["ssn"] = sub.Value
				createPropertyAssertion(personID, "ssn", sub.Value, sub, ctx)
			}

		case "FACT":
			// Generic fact - convert to property or event
			if err := convertFact(personID, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, "FACT", err.Error())
			}

		case "NOTE":
			// Notes
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				if notes, ok := person.Properties["notes"].(string); ok {
					person.Properties["notes"] = notes + "\n\n" + noteText
				} else {
					person.Properties["notes"] = noteText
				}
			}

		case "SOUR":
			// Source citation - handled in property/event conversions
			// Citations are extracted when creating assertions

		case "OBJE":
			// Media object - will be implemented when media converter is done
			ctx.addWarning(indiRecord.Line, "OBJE", "Media linking not yet implemented")

		case "FAMC":
			// Family as child - defer for family processing
			ctx.DeferredFamilyLinks = append(ctx.DeferredFamilyLinks, &FamilyLink{
				PersonID:  personID,
				FamilyRef: sub.Value,
				LinkType:  ParticipantRoleChild,
			})

		case "FAMS":
			// Family as spouse - defer for family processing
			ctx.DeferredFamilyLinks = append(ctx.DeferredFamilyLinks, &FamilyLink{
				PersonID:  personID,
				FamilyRef: sub.Value,
				LinkType:  ParticipantRoleSpouse,
			})

		case "NO":
			// Negative assertion (GEDCOM 7.0)
			if err := convertNegativeAssertion(personID, sub, ctx); err != nil {
				ctx.addWarning(indiRecord.Line, "NO", err.Error())
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
	ctx.GLX.Persons[personID] = person
	ctx.Stats.PersonsCreated++

	return nil
}

// extractNameSubstructure extracts NAME substructure fields
func extractNameSubstructure(nameRecord *GEDCOMRecord) *NameSubstructure {
	ns := &NameSubstructure{}

	for _, sub := range nameRecord.SubRecords {
		switch sub.Tag {
		case "NPFX":
			ns.NPFX = sub.Value
		case "GIVN":
			ns.GIVN = sub.Value
		case "NICK":
			ns.NICK = sub.Value
		case "SPFX":
			ns.SPFX = sub.Value
		case "SURN":
			ns.SURN = sub.Value
		case "NSFX":
			ns.NSFX = sub.Value
		}
	}

	return ns
}

// createNameAssertions creates assertions for name components
func createNameAssertions(personID string, name PersonName, nameRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Create citations from SOUR tags
	citationIDs := extractCitations(personID, nameRecord, ctx)

	// Derive confidence
	confidence := deriveConfidence(citationIDs, ctx)

	// Create assertions for each name component
	if name.GivenName != "" {
		assertionID := generateAssertionID(ctx)
		ctx.GLX.Assertions[assertionID] = &Assertion{
			Subject:    personID,
			Claim:      "given_name",
			Value:      name.GivenName,
			Confidence: confidence,
			Citations:  citationIDs,
		}
		ctx.Stats.AssertionsCreated++
	}

	if name.Surname != "" {
		assertionID := generateAssertionID(ctx)
		ctx.GLX.Assertions[assertionID] = &Assertion{
			Subject:    personID,
			Claim:      "family_name",
			Value:      name.Surname,
			Confidence: confidence,
			Citations:  citationIDs,
		}
		ctx.Stats.AssertionsCreated++
	}

	// Store other name components as property assertions
	if name.Prefix != "" {
		createPropertyAssertion(personID, "name_prefix", name.Prefix, nameRecord, ctx)
	}
	if name.Nickname != "" {
		createPropertyAssertion(personID, "nickname", name.Nickname, nameRecord, ctx)
	}
	if name.SurnamePrefix != "" {
		createPropertyAssertion(personID, "surname_prefix", name.SurnamePrefix, nameRecord, ctx)
	}
	if name.Suffix != "" {
		createPropertyAssertion(personID, "name_suffix", name.Suffix, nameRecord, ctx)
	}

	return nil
}

// convertIndividualEvent converts individual event tags to GLX events
func convertIndividualEvent(personID string, eventRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Map GEDCOM event tag to GLX event type
	eventType := mapGEDCOMEventType(eventRecord.Tag)
	if eventType == "" {
		return fmt.Errorf("unknown event type: %s", eventRecord.Tag)
	}

	// Generate event ID
	eventID := generateEventID(ctx)

	// Create event
	event := &Event{
		Type:       eventType,
		Properties: make(map[string]any),
	}

	// Extract event details
	var eventDate string
	var eventPlace string

	for _, sub := range eventRecord.SubRecords {
		switch sub.Tag {
		case "DATE":
			eventDate = parseGEDCOMDate(sub.Value)
			if eventDate != "" {
				event.Date = eventDate
			}

		case "PLAC":
			// Parse place
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				// Extract coordinates from MAP/LATI/LONG subrecords
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon

				placeID, err := buildPlaceHierarchy(hierarchy, ctx)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
					eventPlace = placeID
				}
			}

		case "AGE":
			// Age at event
			event.Properties["age_at_event"] = sub.Value

		case "CAUS":
			// Cause
			event.Properties["cause"] = sub.Value

		case "TYPE":
			// Event subtype
			event.Properties["event_subtype"] = sub.Value

		case "ADDR":
			// Address
			event.Properties["address"] = sub.Value

		case "NOTE":
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			// Citations handled when creating participations

		case "OBJE":
			// Media - not yet implemented
			ctx.addWarning(eventRecord.Line, "OBJE", "Media linking not yet implemented")
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
	ctx.GLX.Events[eventID] = event
	ctx.Stats.EventsCreated++

	// Create property assertions for born_on, died_on, etc.
	if eventType == "birth" && eventDate != "" {
		createPropertyAssertion(personID, "born_on", eventDate, eventRecord, ctx)
		if eventPlace != "" {
			createPropertyAssertion(personID, "born_at", eventPlace, eventRecord, ctx)
		}
	} else if eventType == "death" && eventDate != "" {
		createPropertyAssertion(personID, "died_on", eventDate, eventRecord, ctx)
		if eventPlace != "" {
			createPropertyAssertion(personID, "died_at", eventPlace, eventRecord, ctx)
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
		"BIRT": EventTypeBirth,
		"CHR":  EventTypeChristening,
		"DEAT": EventTypeDeath,
		"BURI": EventTypeBurial,
		"CREM": EventTypeCremation,
		"ADOP": EventTypeAdoption,
		"BAPM": EventTypeBaptism,
		"BARM": EventTypeBarMitzvah,
		"BASM": EventTypeBasMitzvah,
		"BLES": EventTypeBlessing,
		"CHRA": EventTypeAdultChristening,
		"CONF": EventTypeConfirmation,
		"FCOM": EventTypeFirstCommunion,
		"ORDN": EventTypeOrdination,
		"NATU": EventTypeNaturalization,
		"EMIG": EventTypeEmigration,
		"IMMI": EventTypeImmigration,
		"CENS": EventTypeCensus,
		"PROB": EventTypeProbate,
		"WILL": EventTypeWill,
		"GRAD": EventTypeGraduation,
		"RETI": EventTypeRetirement,
		"RESI": EventTypeResidence,
	}

	if eventType, ok := mapping[tag]; ok {
		return eventType
	}

	return strings.ToLower(tag)
}

// convertResidence converts RESI to residence event or property
func convertResidence(personID string, resiRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Check if it has date - if so, create event
	hasDate := false
	for _, sub := range resiRecord.SubRecords {
		if sub.Tag == "DATE" {
			hasDate = true
			break
		}
	}

	if hasDate {
		// Create residence event
		return convertIndividualEvent(personID, resiRecord, ctx)
	}

	// Otherwise, extract place as property
	for _, sub := range resiRecord.SubRecords {
		if sub.Tag == "PLAC" {
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				placeID, _ := buildPlaceHierarchy(hierarchy, ctx)
				if placeID != "" {
					return createPropertyAssertion(personID, "residence", placeID, resiRecord, ctx)
				}
			}
		}
	}

	return nil
}

// convertFact converts generic FACT tag
func convertFact(personID string, factRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Extract TYPE to determine what kind of fact
	factType := ""
	for _, sub := range factRecord.SubRecords {
		if sub.Tag == "TYPE" {
			factType = sub.Value
			break
		}
	}

	// If it's a recognized property type, create property assertion
	if factType != "" && factRecord.Value != "" {
		propKey := strings.ToLower(strings.ReplaceAll(factType, " ", "_"))
		return createPropertyAssertion(personID, propKey, factRecord.Value, factRecord, ctx)
	}

	// Otherwise treat as generic event if it has date/place
	hasDateOrPlace := false
	for _, sub := range factRecord.SubRecords {
		if sub.Tag == "DATE" || sub.Tag == "PLAC" {
			hasDateOrPlace = true
			break
		}
	}

	if hasDateOrPlace {
		return convertIndividualEvent(personID, factRecord, ctx)
	}

	return nil
}

// convertNegativeAssertion converts GEDCOM 7.0 NO tag (negative assertion)
func convertNegativeAssertion(personID string, noRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// NO tag indicates something did NOT happen
	eventType := mapGEDCOMEventType(noRecord.Value)

	citationIDs := extractCitations(personID, noRecord, ctx)

	assertionID := generateAssertionID(ctx)
	ctx.GLX.Assertions[assertionID] = &Assertion{
		Subject:    personID,
		Claim:      "no_" + eventType,
		Value:      "true", // Negative assertion (NO tag from GEDCOM 7.0)
		Confidence: "high", // Negative assertions are typically certain
		Citations:  citationIDs,
	}
	ctx.Stats.AssertionsCreated++

	return nil
}
