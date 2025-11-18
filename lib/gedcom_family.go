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
func convertFamily(famRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			ctx.Logger.LogException(famRecord.Line, "FAM", famRecord.XRef, "convertFamily",
				fmt.Errorf("panic: %v", r), map[string]interface{}{
					"record": famRecord,
				})
		}
	}()

	if famRecord.Tag != "FAM" {
		return fmt.Errorf("expected FAM record, got %s", famRecord.Tag)
	}

	ctx.Logger.LogInfo(fmt.Sprintf("Converting FAM %s", famRecord.XRef))

	// Extract spouse references
	var husbandID, wifeID string
	var children []string
	var marriageRecord, divorceRecord *GEDCOMRecord

	for _, sub := range famRecord.SubRecords {
		switch sub.Tag {
		case "HUSB":
			// Husband reference
			husbandID = ctx.PersonIDMap[sub.Value]
			if husbandID == "" {
				ctx.Logger.LogWarning(famRecord.Line, "HUSB", sub.Value, "Referenced person not found")
			}

		case "WIFE":
			// Wife reference
			wifeID = ctx.PersonIDMap[sub.Value]
			if wifeID == "" {
				ctx.Logger.LogWarning(famRecord.Line, "WIFE", sub.Value, "Referenced person not found")
			}

		case "CHIL":
			// Child reference
			childID := ctx.PersonIDMap[sub.Value]
			if childID == "" {
				ctx.Logger.LogWarning(famRecord.Line, "CHIL", sub.Value, "Referenced person not found")
			} else {
				children = append(children, childID)
			}

		case "MARR":
			marriageRecord = sub

		case "DIV":
			divorceRecord = sub

		case "ENGA", "MARB", "MARC", "MARL", "MARS", "ANUL", "DIVF", "CENS", "EVEN":
			// Other family events
			if err := convertFamilyEvent(husbandID, wifeID, sub, ctx); err != nil {
				ctx.Logger.LogError(sub.Line, sub.Tag, famRecord.XRef, err)
			}
		}
	}

	// Create spousal relationship if both spouses exist
	if husbandID != "" && wifeID != "" {
		relationshipID := generateRelationshipID(ctx)

		relationship := &Relationship{
			Type:       "spousal",
			Persons:    []string{husbandID, wifeID},
			Properties: make(map[string]interface{}),
		}

		// Extract citations from FAM record itself
		citationIDs := extractCitations(relationshipID, famRecord, ctx)
		if len(citationIDs) > 0 {
			relationship.Properties["citations"] = citationIDs
		}

		ctx.GLX.Relationships[relationshipID] = relationship
		ctx.Stats.RelationshipsCreated++

		// Process marriage event if exists
		if marriageRecord != nil {
			if err := convertMarriageEvent(husbandID, wifeID, relationshipID, marriageRecord, ctx); err != nil {
				ctx.Logger.LogError(marriageRecord.Line, "MARR", famRecord.XRef, err)
			}
		}

		// Process divorce event if exists
		if divorceRecord != nil {
			if err := convertDivorceEvent(husbandID, wifeID, relationshipID, divorceRecord, ctx); err != nil {
				ctx.Logger.LogError(divorceRecord.Line, "DIV", famRecord.XRef, err)
			}
		}
	}

	// Create parent-child relationships
	parents := []string{}
	if husbandID != "" {
		parents = append(parents, husbandID)
	}
	if wifeID != "" {
		parents = append(parents, wifeID)
	}

	for _, childID := range children {
		for _, parentID := range parents {
			relationshipID := generateRelationshipID(ctx)

			relationship := &Relationship{
				Type:       "parent_child",
				Persons:    []string{parentID, childID},
				Properties: make(map[string]interface{}),
			}

			ctx.GLX.Relationships[relationshipID] = relationship
			ctx.Stats.RelationshipsCreated++
		}
	}

	return nil
}

// convertMarriageEvent converts a MARR subrecord to a marriage event
func convertMarriageEvent(husbandID, wifeID, relationshipID string, marrRecord *GEDCOMRecord, ctx *ConversionContext) error {
	eventID := generateEventID(ctx)

	event := &Event{
		Type:       "marriage",
		Properties: make(map[string]interface{}),
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

				placeID, err := buildPlaceHierarchy(hierarchy, ctx)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case "TYPE":
			// Marriage type (e.g., civil, religious)
			event.Properties["marriage_type"] = sub.Value

		case "NOTE":
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			// Citations on the event
			citationID, err := createCitationFromSOUR(eventID, sub, ctx)
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
				mediaID := ctx.MediaIDMap[sub.Value]
				if mediaID != "" {
					if event.Properties["media"] == nil {
						event.Properties["media"] = []string{}
					}
					media := event.Properties["media"].([]string)
					event.Properties["media"] = append(media, mediaID)
				}
			} else {
				// Embedded media
				mediaID, err := convertEmbeddedMedia(sub, ctx)
				if err == nil && mediaID != "" {
					if event.Properties["media"] == nil {
						event.Properties["media"] = []string{}
					}
					media := event.Properties["media"].([]string)
					event.Properties["media"] = append(media, mediaID)
				}
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     "spouse",
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     "spouse",
		})
	}
	event.Participants = participants

	// Store event
	ctx.GLX.Events[eventID] = event
	ctx.Stats.EventsCreated++

	// Link event to relationship
	relationship := ctx.GLX.Relationships[relationshipID]
	if relationship != nil {
		if relationship.Properties["marriage_event"] == nil {
			relationship.Properties["marriage_event"] = eventID
		}
	}

	return nil
}

// convertDivorceEvent converts a DIV subrecord to a divorce event
func convertDivorceEvent(husbandID, wifeID, relationshipID string, divRecord *GEDCOMRecord, ctx *ConversionContext) error {
	eventID := generateEventID(ctx)

	event := &Event{
		Type:       "divorce",
		Properties: make(map[string]interface{}),
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

				placeID, err := buildPlaceHierarchy(hierarchy, ctx)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case "NOTE":
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			citationID, err := createCitationFromSOUR(eventID, sub, ctx)
			if err == nil && citationID != "" {
				citations, ok := event.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties["citations"] = append(citations, citationID)
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     "spouse",
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     "spouse",
		})
	}
	event.Participants = participants

	// Store event
	ctx.GLX.Events[eventID] = event
	ctx.Stats.EventsCreated++

	// Link event to relationship
	relationship := ctx.GLX.Relationships[relationshipID]
	if relationship != nil {
		if relationship.Properties["divorce_event"] == nil {
			relationship.Properties["divorce_event"] = eventID
		}
	}

	return nil
}

// convertFamilyEvent converts other family events (ENGA, MARB, MARC, MARL, MARS)
func convertFamilyEvent(husbandID, wifeID string, eventRecord *GEDCOMRecord, ctx *ConversionContext) error {
	eventID := generateEventID(ctx)

	eventType := mapFamilyEventType(eventRecord.Tag)

	event := &Event{
		Type:       eventType,
		Properties: make(map[string]interface{}),
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

				placeID, err := buildPlaceHierarchy(hierarchy, ctx)
				if err == nil && placeID != "" {
					event.PlaceID = placeID
				}
			}

		case "NOTE":
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				event.Properties["notes"] = noteText
			}

		case "SOUR":
			citationID, err := createCitationFromSOUR(eventID, sub, ctx)
			if err == nil && citationID != "" {
				citations, ok := event.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				event.Properties["citations"] = append(citations, citationID)
			}
		}
	}

	// Add participants for both spouses
	var participants []EventParticipant
	if husbandID != "" {
		participants = append(participants, EventParticipant{
			PersonID: husbandID,
			Role:     "spouse",
		})
	}
	if wifeID != "" {
		participants = append(participants, EventParticipant{
			PersonID: wifeID,
			Role:     "spouse",
		})
	}
	event.Participants = participants

	// Store event
	ctx.GLX.Events[eventID] = event
	ctx.Stats.EventsCreated++

	return nil
}

// mapFamilyEventType maps GEDCOM family event tags to GLX event types
func mapFamilyEventType(tag string) string {
	mapping := map[string]string{
		"ENGA": "engagement",
		"MARB": "marriage_banns",
		"MARC": "marriage_contract",
		"MARL": "marriage_license",
		"MARS": "marriage_settlement",
		"ANUL": "annulment",
		"DIVF": "divorce_filed",
		"CENS": "census",
		"EVEN": "event",
	}

	if eventType, ok := mapping[tag]; ok {
		return eventType
	}

	return "event"
}
