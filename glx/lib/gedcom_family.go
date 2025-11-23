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
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			conv.Logger.LogException(famRecord.Line, "FAM", famRecord.XRef, "convertFamily",
				fmt.Errorf("panic: %v", r), map[string]any{
					"record": famRecord,
				})
		}
	}()

	if famRecord.Tag != "FAM" {
		return fmt.Errorf("expected FAM record, got %s", famRecord.Tag)
	}

	conv.Logger.LogInfo(fmt.Sprintf("Converting FAM %s", famRecord.XRef))

	// Extract spouse references
	var husbandID, wifeID string
	var marriageRecord, divorceRecord *GEDCOMRecord

	for _, sub := range famRecord.SubRecords {
		switch sub.Tag {
		case "HUSB":
			// Husband reference
			husbandID = conv.PersonIDMap[sub.Value]
			if husbandID == "" {
				conv.Logger.LogWarning(famRecord.Line, "HUSB", sub.Value, "Referenced person not found")
			}

		case "WIFE":
			// Wife reference
			wifeID = conv.PersonIDMap[sub.Value]
			if wifeID == "" {
				conv.Logger.LogWarning(famRecord.Line, "WIFE", sub.Value, "Referenced person not found")
			}

		case "CHIL":
			// Child reference - validation only, parent-child relationships are
			// created when processing INDI records (which contain PEDI information)
			childID := conv.PersonIDMap[sub.Value]
			if childID == "" {
				conv.Logger.LogWarning(famRecord.Line, "CHIL", sub.Value, "Referenced person not found")
			}

		case "MARR":
			marriageRecord = sub

		case "DIV":
			divorceRecord = sub

		case "ENGA", "MARB", "MARC", "MARL", "MARS", "ANUL", "DIVF", "CENS", "EVEN":
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
			Type:       RelationshipTypeMarriage,
			Persons:    []string{husbandID, wifeID},
			Properties: make(map[string]any),
		}

		// Extract citations from FAM record itself
		citationIDs := extractCitations(relationshipID, famRecord, conv)
		if len(citationIDs) > 0 {
			relationship.Properties["citations"] = citationIDs
		}

		conv.GLX.Relationships[relationshipID] = relationship
		conv.Stats.RelationshipsCreated++

		// Process marriage event if exists
		if marriageRecord != nil {
			if err := convertMarriageEvent(husbandID, wifeID, relationshipID, marriageRecord, conv); err != nil {
				conv.Logger.LogError(marriageRecord.Line, "MARR", famRecord.XRef, err)
			}
		}

		// Process divorce event if exists
		if divorceRecord != nil {
			if err := convertDivorceEvent(husbandID, wifeID, relationshipID, divorceRecord, conv); err != nil {
				conv.Logger.LogError(divorceRecord.Line, "DIV", famRecord.XRef, err)
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
		case "DATE":
			event.Date = parseGEDCOMDate(sub.Value)

		case "PLAC":
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

		case "TYPE":
			// Marriage type (e.g., civil, religious)
			event.Properties["marriage_type"] = sub.Value

		case "NOTE":
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			// Citations on the event
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties["citations"] = append(citations, citationID)
			}

		case "OBJE":
			// Media attached to event
			if sub.Value != "" {
				mediaID := conv.MediaIDMap[sub.Value]
				if mediaID != "" {
					if event.Properties["media"] == nil {
						event.Properties["media"] = []string{}
					}
					media := event.Properties["media"].([]string)
					event.Properties["media"] = append(media, mediaID)
				}
			} else {
				// Embedded media
				mediaID, err := convertEmbeddedMedia(sub, conv)
				if err == nil && mediaID != "" {
					if event.Properties["media"] == nil {
						event.Properties["media"] = []string{}
					}
					media := event.Properties["media"].([]string)
					event.Properties["media"] = append(media, mediaID)
				}
			}

		case "ADDR":
			// Address - extract full address including subfields
			addr := extractAddress(sub)
			if addr != "" {
				event.Properties["address"] = addr
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
		if relationship.Properties["marriage_event"] == nil {
			relationship.Properties["marriage_event"] = eventID
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
		case "DATE":
			event.Date = parseGEDCOMDate(sub.Value)

		case "PLAC":
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

		case "NOTE":
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties["citations"] = append(citations, citationID)
			}

		case "ADDR":
			// Address - extract full address including subfields
			addr := extractAddress(sub)
			if addr != "" {
				event.Properties["address"] = addr
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
		if relationship.Properties["divorce_event"] == nil {
			relationship.Properties["divorce_event"] = eventID
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
		case "DATE":
			event.Date = parseGEDCOMDate(sub.Value)

		case "PLAC":
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

		case "NOTE":
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			citationID, err := createCitationFromSOUR(eventID, sub, conv)
			if err == nil && citationID != "" {
				citations, ok := event.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties["citations"] = append(citations, citationID)
			}

		case "ADDR":
			// Address - extract full address including subfields
			addr := extractAddress(sub)
			if addr != "" {
				event.Properties["address"] = addr
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
		"ENGA": EventTypeEngagement,
		"MARB": EventTypeMarriageBanns,
		"MARC": EventTypeMarriageContract,
		"MARL": EventTypeMarriageLicense,
		"MARS": EventTypeMarriageSettlement,
		"ANUL": EventTypeAnnulment,
		"DIVF": EventTypeDivorceFiled,
		"CENS": EventTypeCensus,
		"EVEN": EventTypeGeneric,
	}

	if eventType, ok := mapping[tag]; ok {
		return eventType
	}

	return EventTypeGeneric
}
