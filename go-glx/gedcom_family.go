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

// convertFamily converts a GEDCOM FAM record to GLX Relationships and Events
//
//nolint:gocognit,gocyclo
func convertFamily(famRecord *GEDCOMRecord, conv *ConversionContext) error {
	if famRecord.Tag != GedcomTagFam {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedRecordType, GedcomTagFam, famRecord.Tag)
	}

	conv.Logger.LogInfo("Converting FAM " + famRecord.XRef)

	// Two-pass processing: first extract spouse IDs, then process events.
	// GEDCOM does not guarantee tag order, so event tags may appear before
	// HUSB/WIFE tags. Collecting everything first prevents empty spouse IDs.
	var husbandID, wifeID string
	var marriageRecords []*GEDCOMRecord
	var divorceRecord *GEDCOMRecord
	var censusRecords []*GEDCOMRecord
	var familyEventRecords []*GEDCOMRecord
	var familyResiRecords []*GEDCOMRecord
	var objeRecords []*GEDCOMRecord
	var noteTexts []string

	for _, sub := range famRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				noteTexts = append(noteTexts, noteText)
			}
		case GedcomTagHusb:
			husbandID = conv.PersonIDMap[sub.Value]
			if husbandID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagHusb, sub.Value, "Referenced person not found")
			}

		case GedcomTagWife:
			wifeID = conv.PersonIDMap[sub.Value]
			if wifeID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagWife, sub.Value, "Referenced person not found")
			}

		case GedcomTagChil:
			childID := conv.PersonIDMap[sub.Value]
			if childID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagChil, sub.Value, "Referenced person not found")
			}

		case GedcomTagMarr:
			marriageRecords = append(marriageRecords, sub)
		case GedcomTagDiv:
			divorceRecord = sub
		case GedcomTagCens:
			censusRecords = append(censusRecords, sub)
		case GedcomTagResi:
			familyResiRecords = append(familyResiRecords, sub)
		case GedcomTagEnga, GedcomTagMarb, GedcomTagMarc, GedcomTagMarl, GedcomTagMars, GedcomTagAnul, GedcomTagDivf, GedcomTagEven:
			familyEventRecords = append(familyEventRecords, sub)
		case GedcomTagObje:
			objeRecords = append(objeRecords, sub)
		default:
			if isExtensionTag(sub.Tag) {
				conv.addWarning(sub.Line, sub.Tag, "Extension tag not stored")
			}
		}
	}

	// Second pass: process deferred family events now that spouse IDs are known
	// Skip if both spouse IDs are empty — events require at least one participant
	if husbandID != "" || wifeID != "" {
		for _, censRec := range censusRecords {
			convertFamilyCensus(husbandID, wifeID, censRec, conv)
		}
		for _, resiRec := range familyResiRecords {
			convertFamilyResidence(husbandID, wifeID, resiRec, conv)
		}
		for _, eventRec := range familyEventRecords {
			convertFamilyEvent(husbandID, wifeID, eventRec, conv)
		}
	}

	// Create spousal relationship if at least one spouse exists
	if husbandID != "" || wifeID != "" {
		relationshipID := generateRelationshipID(conv)

		var participants []Participant
		if husbandID != "" {
			participants = append(participants, Participant{Person: husbandID, Role: ParticipantRoleSpouse})
		}
		if wifeID != "" {
			participants = append(participants, Participant{Person: wifeID, Role: ParticipantRoleSpouse})
		}

		relationship := &Relationship{
			Type:         RelationshipTypeMarriage,
			Participants: participants,
			Properties:   make(map[string]any),
		}

		// Extract evidence from FAM record itself
		refs := extractEvidence(famRecord, conv)
		if len(refs.SourceIDs) > 0 {
			relationship.Properties[PropertySources] = refs.SourceIDs
		}
		if len(refs.CitationIDs) > 0 {
			relationship.Properties[PropertyCitations] = refs.CitationIDs
		}

		// Attach FAM-level NOTEs to the relationship
		if len(noteTexts) > 0 {
			relationship.Notes = strings.Join(noteTexts, "\n\n")
		}

		// Resolve FAM-level OBJE references
		for _, obje := range objeRecords {
			handleOBJE(obje, relationship.Properties, conv)
		}

		conv.GLX.Relationships[relationshipID] = relationship
		conv.Stats.RelationshipsCreated++

		// Process marriage events — first becomes StartEvent, rest become family events
		for i, marrRec := range marriageRecords {
			if i == 0 {
				convertMarriageEvent(husbandID, wifeID, relationshipID, marrRec, conv)
			} else {
				convertFamilyEvent(husbandID, wifeID, marrRec, conv)
			}
		}

		// Process divorce event if exists
		if divorceRecord != nil {
			convertDivorceEvent(husbandID, wifeID, relationshipID, divorceRecord, conv)
		}
	}

	// Store family mapping for parent-child relationship creation later
	// We need to defer this because PEDI (pedigree type) information
	// is on the FAMC tag in INDI records, not in the FAM record
	parents := []string{}
	if husbandID != "" {
		parents = append(parents, husbandID)
	}
	if wifeID != "" {
		parents = append(parents, wifeID)
	}

	// Store family parents for later lookup
	conv.FamilyParentsMap[famRecord.XRef] = parents

	return nil
}

// convertMarriageEvent converts a MARR subrecord to a marriage event
func convertMarriageEvent(husbandID, wifeID, relationshipID string, marrRecord *GEDCOMRecord, conv *ConversionContext) {
	eventID := generateEventID(conv)

	event := &Event{
		Type:       EventTypeMarriage,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	extractEventDetails(eventID, marrRecord, event, conv, true)

	// Process marriage-specific tags
	for _, sub := range marrRecord.SubRecords {
		if sub.Tag == GedcomTagType {
			// Marriage type (e.g., civil, religious)
			event.Properties[PropertyMarriageType] = sub.Value
		}
	}

	// Add participants for both spouses
	var participants []Participant
	if husbandID != "" {
		participants = append(participants, Participant{
			Person: husbandID,
			Role:   ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, Participant{
			Person: wifeID,
			Role:   ParticipantRoleSpouse,
		})
	}
	event.Participants = participants

	// Add ASSO participants (witnesses, officiants, etc.)
	appendASSOParticipants(event, marrRecord, conv)

	// Generate event title from spouse names
	var marrNames []string
	if husbandID != "" {
		marrNames = append(marrNames, PersonDisplayName(conv.GLX.Persons[husbandID]))
	}
	if wifeID != "" {
		marrNames = append(marrNames, PersonDisplayName(conv.GLX.Persons[wifeID]))
	}
	event.Title = GenerateEventTitle(EventTypeMarriage, marrNames, event.Date)

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Link marriage event as the relationship's start_event
	relationship := conv.GLX.Relationships[relationshipID]
	if relationship != nil && relationship.StartEvent == "" {
		relationship.StartEvent = eventID
	}
}

// convertDivorceEvent converts a DIV subrecord to a divorce event
func convertDivorceEvent(husbandID, wifeID, relationshipID string, divRecord *GEDCOMRecord, conv *ConversionContext) {
	eventID := generateEventID(conv)

	event := &Event{
		Type:       EventTypeDivorce,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	extractEventDetails(eventID, divRecord, event, conv, true)

	// Add participants for both spouses
	var participants []Participant
	if husbandID != "" {
		participants = append(participants, Participant{
			Person: husbandID,
			Role:   ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, Participant{
			Person: wifeID,
			Role:   ParticipantRoleSpouse,
		})
	}
	event.Participants = participants

	// Add ASSO participants (witnesses, officiants, etc.)
	appendASSOParticipants(event, divRecord, conv)

	// Generate event title from spouse names
	var divNames []string
	if husbandID != "" {
		divNames = append(divNames, PersonDisplayName(conv.GLX.Persons[husbandID]))
	}
	if wifeID != "" {
		divNames = append(divNames, PersonDisplayName(conv.GLX.Persons[wifeID]))
	}
	event.Title = GenerateEventTitle(EventTypeDivorce, divNames, event.Date)

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Link divorce event as the relationship's end_event
	relationship := conv.GLX.Relationships[relationshipID]
	if relationship != nil && relationship.EndEvent == "" {
		relationship.EndEvent = eventID
	}
}

// convertFamilyEvent converts other family events (ENGA, MARB, MARC, MARL, MARS)
func convertFamilyEvent(husbandID, wifeID string, eventRecord *GEDCOMRecord, conv *ConversionContext) {
	eventID := generateEventID(conv)

	eventType := mapFamilyEventType(eventRecord.Tag, conv.GEDCOMIndex)

	event := &Event{
		Type:       eventType,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	extractEventDetails(eventID, eventRecord, event, conv, true)

	// Extract TYPE as event_subtype (same as individual events)
	for _, sub := range eventRecord.SubRecords {
		if sub.Tag == GedcomTagType {
			if propertyKey, ok := conv.GEDCOMIndex.EventProperties[sub.Tag]; ok {
				event.Properties[propertyKey] = sub.Value
			}
			break
		}
	}

	// Add participants for both spouses
	var participants []Participant
	if husbandID != "" {
		participants = append(participants, Participant{
			Person: husbandID,
			Role:   ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, Participant{
			Person: wifeID,
			Role:   ParticipantRoleSpouse,
		})
	}
	event.Participants = participants

	// Add ASSO participants (witnesses, officiants, etc.)
	appendASSOParticipants(event, eventRecord, conv)

	// Generate event title from spouse names
	var names []string
	if husbandID != "" {
		names = append(names, PersonDisplayName(conv.GLX.Persons[husbandID]))
	}
	if wifeID != "" {
		names = append(names, PersonDisplayName(conv.GLX.Persons[wifeID]))
	}
	event.Title = GenerateEventTitle(eventType, names, event.Date)

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++
}

// convertFamilyResidence applies a family-level RESI record to both spouses.
// Evidence is extracted once to avoid creating duplicate citations.
func convertFamilyResidence(husbandID, wifeID string, resiRecord *GEDCOMRecord, conv *ConversionContext) {
	// Extract residence data and evidence once
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

	if placeID == "" {
		return
	}

	refs := extractEvidence(resiRecord, conv)

	// Apply to each spouse
	for _, spouseID := range []string{husbandID, wifeID} {
		if spouseID == "" {
			continue
		}
		person, ok := conv.GLX.Persons[spouseID]
		if !ok {
			continue
		}

		if dateStr != "" {
			appendResidence(person, map[string]any{
				"value": placeID,
				"date":  dateStr,
			})
		} else {
			appendResidence(person, placeID)
		}

		createPropertyAssertionWithEvidence(spouseID, PersonPropertyResidence, placeID, refs, conv)
	}
}

// convertFamilyCensus applies a family-level CENS record to both spouses.
// Source/citation creation happens once; the resulting data is applied to each spouse.
func convertFamilyCensus(husbandID, wifeID string, censRecord *GEDCOMRecord, conv *ConversionContext) {
	// Extract census data once (creates source/citation once)
	firstSpouseID := husbandID
	if firstSpouseID == "" {
		firstSpouseID = wifeID
	}
	if firstSpouseID == "" {
		return
	}
	data := extractCensusData(censRecord, conv)

	// Apply to each spouse
	for _, spouseID := range []string{husbandID, wifeID} {
		if spouseID == "" {
			continue
		}
		person, ok := conv.GLX.Persons[spouseID]
		if !ok {
			continue
		}
		applyCensusData(spouseID, person, data, conv)
	}
}

// mapFamilyEventType maps GEDCOM family event tags to GLX event types using the vocabulary index.
func mapFamilyEventType(tag string, gedcomIndex *GEDCOMIndex) string {
	if eventType, ok := gedcomIndex.EventTypes[tag]; ok {
		return eventType
	}

	return EventTypeGeneric
}
