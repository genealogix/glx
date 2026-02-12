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
)

// convertSharedNote551 converts a GEDCOM 5.5.1 NOTE record to shared note storage
func convertSharedNote551(noteRecord *GEDCOMRecord, conv *ConversionContext) error {
	if noteRecord.Tag != GedcomTagNote {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedNoteRecord, GedcomTagNote, noteRecord.Tag)
	}

	// Extract note text (GEDCOM 5.5.1 format)
	noteText := extractNoteText(noteRecord, conv)
	if noteText == "" {
		noteText = noteRecord.Value
	}

	// Store in shared notes map (same map as GEDCOM 7.0 SNOTE)
	conv.SharedNotes[noteRecord.XRef] = noteText

	conv.Logger.LogInfo(fmt.Sprintf("Stored shared note (5.5.1) %s (%d chars)", noteRecord.XRef, len(noteText)))

	return nil
}

// convertSharedNote converts a GEDCOM 7.0 SNOTE record to shared note storage
func convertSharedNote(snoteRecord *GEDCOMRecord, conv *ConversionContext) error {
	if snoteRecord.Tag != GedcomTagSnote {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedSharedRecord, GedcomTagSnote, snoteRecord.Tag)
	}

	// Extract note text
	noteText := extractNoteText(snoteRecord, conv)
	if noteText == "" {
		noteText = snoteRecord.Value
	}

	// Store in shared notes map
	conv.SharedNotes[snoteRecord.XRef] = noteText

	conv.Logger.LogInfo(fmt.Sprintf("Stored shared note %s (%d chars)", snoteRecord.XRef, len(noteText)))

	return nil
}

// convertExtensionSchema converts a GEDCOM 7.0 SCHMA record
func convertExtensionSchema(schmaRecord *GEDCOMRecord, conv *ConversionContext) error {
	if schmaRecord.Tag != GedcomTagSchma {
		return fmt.Errorf("%w: expected SCHMA, got %s", ErrUnexpectedSchemaRecord, schmaRecord.Tag)
	}

	schema := &ExtensionSchema{
		Tag: schmaRecord.XRef,
	}

	for _, sub := range schmaRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagTag:
			// Main tag name
			if schema.Tag == "" {
				schema.Tag = sub.Value
			}
		case GedcomTagURI:
			// Schema URI
			schema.URI = sub.Value
		case GedcomTagNote:
			// Description
			schema.Description = extractNoteText(sub, conv)
		}
	}

	// Store schema
	conv.ExtensionSchemas[schmaRecord.XRef] = schema

	conv.Logger.LogInfo("Stored extension schema " + schmaRecord.XRef)

	return nil
}

// convertExtensionData converts extension tag data into properties map
func convertExtensionData(tag, value string, subRecords []*GEDCOMRecord) map[string]any {
	properties := make(map[string]any)

	// Only process extension tags (those starting with underscore)
	if len(tag) == 0 || tag[0] != '_' {
		return properties
	}

	properties["extension_tag"] = tag

	if value != "" {
		properties["value"] = value
	}

	if len(subRecords) > 0 {
		subData := make(map[string]any)
		for _, sub := range subRecords {
			subData[sub.Tag] = sub.Value
		}
		properties["subrecords"] = subData
	}

	return properties
}

// extractEventDetails extracts common event details (DATE, PLAC, NOTE, ADDR, and optionally SOUR)
// from GEDCOM subrecords and populates the event.
// If eventID is non-empty and includeCitations is true, SOUR records will be processed
// and added to event.Properties[PropertyCitations].
// Returns the place ID if one was extracted (for callers that need it for additional processing).
//
//nolint:gocognit,gocyclo
func extractEventDetails(eventID string, eventRecord *GEDCOMRecord, event *Event, conv *ConversionContext, includeCitations bool) string {
	// Ensure Properties map is initialized before writing to it
	if event.Properties == nil {
		event.Properties = make(map[string]any)
	}

	var placeID string

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

				pid := buildPlaceHierarchy(hierarchy, conv)
				if pid != "" {
					event.PlaceID = pid
					placeID = pid
				}
			}

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				event.Properties[PropertyNotes] = noteText
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
					pid := buildPlaceHierarchy(hierarchy, conv)
					if pid != "" {
						event.PlaceID = pid
						placeID = pid
					}
				}
			}

		case GedcomTagSour:
			if includeCitations && eventID != "" {
				result, err := createCitationFromSOUR(sub, conv)
				if err != nil {
					// Error already logged in createCitationFromSOUR, skip
					continue
				}
				if result.CitationID != "" {
					citations, ok := event.Properties[PropertyCitations].([]string)
					if !ok {
						citations = []string{}
					}
					event.Properties[PropertyCitations] = append(citations, result.CitationID)
				} else if result.SourceID != "" {
					sources, ok := event.Properties[PropertySources].([]string)
					if !ok {
						sources = []string{}
					}
					event.Properties[PropertySources] = append(sources, result.SourceID)
				}
			}

		case GedcomTagObje:
			handleOBJE(sub, event.Properties, conv)
		}
	}

	return placeID
}

// extractExternalIDs extracts EXID tags from a record and returns them as a slice
// EXID format (GEDCOM 7.0):
//
//	1 EXID <identifier>
//	2 TYPE <type>  (optional, e.g., "wikitree", "familysearch", "findagrave")
func extractExternalIDs(record *GEDCOMRecord) []map[string]string {
	var exids []map[string]string

	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagExid {
			exid := make(map[string]string)
			exid["id"] = sub.Value

			// Check for TYPE subrecord
			for _, exidSub := range sub.SubRecords {
				if exidSub.Tag == "TYPE" {
					exid["type"] = exidSub.Value
				}
			}

			exids = append(exids, exid)
		}
	}

	return exids
}
