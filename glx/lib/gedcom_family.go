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
)

// convertFamily converts a GEDCOM FAM record to GLX Relationships and Events
func convertFamily(famRecord *GEDCOMRecord, conv *ConversionContext) error {
	if famRecord.Tag != GedcomTagFam {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedRecordType, GedcomTagFam, famRecord.Tag)
	}

	conv.Logger.LogInfo("Converting FAM " + famRecord.XRef)

	// Extract spouse references
	var husbandID, wifeID string
	var marriageRecord, divorceRecord *GEDCOMRecord

	for _, sub := range famRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagHusb:
			// Husband reference
			husbandID = conv.PersonIDMap[sub.Value]
			if husbandID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagHusb, sub.Value, "Referenced person not found")
			}

		case GedcomTagWife:
			// Wife reference
			wifeID = conv.PersonIDMap[sub.Value]
			if wifeID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagWife, sub.Value, "Referenced person not found")
			}

		case GedcomTagChil:
			// Child reference - validation only, parent-child relationships are
			// created when processing INDI records (which contain PEDI information)
			childID := conv.PersonIDMap[sub.Value]
			if childID == "" {
				conv.Logger.LogWarning(famRecord.Line, GedcomTagChil, sub.Value, "Referenced person not found")
			}

		case GedcomTagMarr:
			marriageRecord = sub

		case GedcomTagDiv:
			divorceRecord = sub

		case GedcomTagCens:
			// Census - apply to both spouses as citations/temporal properties
			if err := convertFamilyCensus(husbandID, wifeID, sub, conv); err != nil {
				conv.Logger.LogError(sub.Line, sub.Tag, famRecord.XRef, err)
			}

		case GedcomTagEnga, GedcomTagMarb, GedcomTagMarc, GedcomTagMarl, GedcomTagMars, GedcomTagAnul, GedcomTagDivf, GedcomTagEven:
			// Other family events
			if err := convertFamilyEvent(husbandID, wifeID, sub, conv); err != nil {
				conv.Logger.LogError(sub.Line, sub.Tag, famRecord.XRef, err)
			}
		}
	}

	// Create spousal relationship if both spouses exist
	if husbandID != "" && wifeID != "" {
		relationshipID := generateRelationshipID(conv)

		relationship := &Relationship{
			Type: RelationshipTypeMarriage,
			Participants: []Participant{
				{Person: husbandID, Role: ParticipantRoleSpouse},
				{Person: wifeID, Role: ParticipantRoleSpouse},
			},
			Properties: make(map[string]any),
		}

		// Extract citations from FAM record itself
		citationIDs := extractCitations(relationshipID, famRecord, conv)
		if len(citationIDs) > 0 {
			relationship.Properties[PropertyCitations] = citationIDs
		}

		conv.GLX.Relationships[relationshipID] = relationship
		conv.Stats.RelationshipsCreated++

		// Process marriage event if exists
		if marriageRecord != nil {
			if err := convertMarriageEvent(husbandID, wifeID, relationshipID, marriageRecord, conv); err != nil {
				conv.Logger.LogError(marriageRecord.Line, GedcomTagMarr, famRecord.XRef, err)
			}
		}

		// Process divorce event if exists
		if divorceRecord != nil {
			if err := convertDivorceEvent(husbandID, wifeID, relationshipID, divorceRecord, conv); err != nil {
				conv.Logger.LogError(divorceRecord.Line, GedcomTagDiv, famRecord.XRef, err)
			}
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
func convertMarriageEvent(husbandID, wifeID, relationshipID string, marrRecord *GEDCOMRecord, conv *ConversionContext) error {
	eventID := generateEventID(conv)

	event := &Event{
		Type:       EventTypeMarriage,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	extractEventDetails(eventID, marrRecord, event, conv, true)

	// Process marriage-specific tags
	for _, sub := range marrRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagType:
			// Marriage type (e.g., civil, religious)
			event.Properties[PropertyMarriageType] = sub.Value

		case GedcomTagObje:
			// Media attached to event
			if sub.Value != "" {
				mediaID := conv.MediaIDMap[sub.Value]
				if mediaID != "" {
					if event.Properties[PropertyMedia] == nil {
						event.Properties[PropertyMedia] = []string{}
					}
					media := event.Properties[PropertyMedia].([]string)
					event.Properties[PropertyMedia] = append(media, mediaID)
				}
			} else {
				// Embedded media
				mediaID, err := convertEmbeddedMedia(sub, conv)
				if err != nil {
					conv.Logger.LogWarning(sub.Line, GedcomTagObje, "", "Failed to convert embedded media: "+err.Error())

					continue
				}
				if mediaID != "" {
					if event.Properties[PropertyMedia] == nil {
						event.Properties[PropertyMedia] = []string{}
					}
					media := event.Properties[PropertyMedia].([]string)
					event.Properties[PropertyMedia] = append(media, mediaID)
				}
			}
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

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Link event to relationship
	relationship := conv.GLX.Relationships[relationshipID]
	if relationship != nil {
		if relationship.Properties[PropertyMarriageEvent] == nil {
			relationship.Properties[PropertyMarriageEvent] = eventID
		}
	}

	return nil
}

// convertDivorceEvent converts a DIV subrecord to a divorce event
func convertDivorceEvent(husbandID, wifeID, relationshipID string, divRecord *GEDCOMRecord, conv *ConversionContext) error {
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

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	// Link event to relationship
	relationship := conv.GLX.Relationships[relationshipID]
	if relationship != nil {
		if relationship.Properties[PropertyDivorceEvent] == nil {
			relationship.Properties[PropertyDivorceEvent] = eventID
		}
	}

	return nil
}

// convertFamilyEvent converts other family events (ENGA, MARB, MARC, MARL, MARS)
func convertFamilyEvent(husbandID, wifeID string, eventRecord *GEDCOMRecord, conv *ConversionContext) error {
	eventID := generateEventID(conv)

	eventType := mapFamilyEventType(eventRecord.Tag, conv.GEDCOMIndex)

	event := &Event{
		Type:       eventType,
		Properties: make(map[string]any),
	}

	// Extract common event details (DATE, PLAC, NOTE, ADDR, SOUR)
	extractEventDetails(eventID, eventRecord, event, conv, true)

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

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	return nil
}

// convertFamilyCensus applies a family-level CENS record to both spouses.
// Source/citation creation happens once; the resulting data is applied to each spouse.
func convertFamilyCensus(husbandID, wifeID string, censRecord *GEDCOMRecord, conv *ConversionContext) error {
	// Extract census data once (creates source/citation once)
	firstSpouseID := husbandID
	if firstSpouseID == "" {
		firstSpouseID = wifeID
	}
	if firstSpouseID == "" {
		return nil
	}
	data := extractCensusData(firstSpouseID, censRecord, conv)

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

	return nil
}

// mapFamilyEventType maps GEDCOM family event tags to GLX event types using the vocabulary index.
func mapFamilyEventType(tag string, gedcomIndex *GEDCOMIndex) string {
	if eventType, ok := gedcomIndex.EventTypes[tag]; ok {
		return eventType
	}

	return EventTypeGeneric
}
