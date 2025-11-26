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
		return fmt.Errorf("%w: expected FAM, got %s", ErrUnexpectedRecordType, famRecord.Tag)
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
			Type:    RelationshipTypeMarriage,
			Persons: []string{husbandID, wifeID},
			Participants: []RelationshipParticipant{
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

	// Extract event details
	for _, sub := range marrRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			event.Date = parseGEDCOMDate(sub.Value)

		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				// Extract coordinates from MAP/LATI/LONG subrecords
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon

				placeID, err := buildPlaceHierarchy(hierarchy, conv)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case GedcomTagType:
			// Marriage type (e.g., civil, religious)
			event.Properties[PropertyMarriageType] = sub.Value

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties[PropertyNotes] = noteText
			}

		case GedcomTagSour:
			// Citations on the event
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties[PropertyCitations].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties[PropertyCitations] = append(citations, citationID)
			}

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
				if err == nil && mediaID != "" {
					if event.Properties[PropertyMedia] == nil {
						event.Properties[PropertyMedia] = []string{}
					}
					media := event.Properties[PropertyMedia].([]string)
					event.Properties[PropertyMedia] = append(media, mediaID)
				}
			}

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
					}
				}
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     ParticipantRoleSpouse,
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

	// Extract event details
	for _, sub := range divRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			event.Date = parseGEDCOMDate(sub.Value)

		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				// Extract coordinates from MAP/LATI/LONG subrecords
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon

				placeID, err := buildPlaceHierarchy(hierarchy, conv)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties[PropertyNotes] = noteText
			}

		case GedcomTagSour:
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties[PropertyCitations].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties[PropertyCitations] = append(citations, citationID)
			}

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
					}
				}
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     ParticipantRoleSpouse,
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

	eventType := mapFamilyEventType(eventRecord.Tag)

	event := &Event{
		Type:       eventType,
		Properties: make(map[string]any),
	}

	// Extract event details
	for _, sub := range eventRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			event.Date = parseGEDCOMDate(sub.Value)

		case GedcomTagPlac:
			hierarchy := parseGEDCOMPlace(sub.Value)
			if hierarchy != nil {
				// Extract coordinates from MAP/LATI/LONG subrecords
				lat, lon := extractPlaceCoordinates(sub)
				hierarchy.Latitude = lat
				hierarchy.Longitude = lon

				placeID, err := buildPlaceHierarchy(hierarchy, conv)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties[PropertyNotes] = noteText
			}

		case GedcomTagSour:
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties[PropertyCitations].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties[PropertyCitations] = append(citations, citationID)
			}

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
					}
				}
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     ParticipantRoleSpouse,
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     ParticipantRoleSpouse,
		})
	}
	event.Participants = participants

	// Store event
	conv.GLX.Events[eventID] = event
	conv.Stats.EventsCreated++

	return nil
}

// mapFamilyEventType maps GEDCOM family event tags to GLX event types
func mapFamilyEventType(tag string) string {
	mapping := map[string]string{
		GedcomTagEnga: EventTypeEngagement,
		GedcomTagMarb: EventTypeMarriageBanns,
		GedcomTagMarc: EventTypeMarriageContract,
		GedcomTagMarl: EventTypeMarriageLicense,
		GedcomTagMars: EventTypeMarriageSettlement,
		GedcomTagAnul: EventTypeAnnulment,
		GedcomTagDivf: EventTypeDivorceFiled,
		GedcomTagEven: EventTypeGeneric,
	}

	if eventType, ok := mapping[tag]; ok {
		return eventType
	}

	return EventTypeGeneric
}
