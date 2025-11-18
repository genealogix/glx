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

// convertSharedNote converts a GEDCOM 7.0 SNOTE record to shared note storage
func convertSharedNote(snoteRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if snoteRecord.Tag != "SNOTE" {
		return fmt.Errorf("expected SNOTE record, got %s", snoteRecord.Tag)
	}

	// Extract note text
	noteText := extractNoteText(snoteRecord, ctx)
	if noteText == "" {
		noteText = snoteRecord.Value
	}

	// Store in shared notes map
	ctx.SharedNotes[snoteRecord.XRef] = noteText

	ctx.Logger.LogInfo(fmt.Sprintf("Stored shared note %s (%d chars)", snoteRecord.XRef, len(noteText)))

	return nil
}

// convertExtensionSchema converts a GEDCOM 7.0 SCHMA record
func convertExtensionSchema(schmaRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if schmaRecord.Tag != "SCHMA" {
		return fmt.Errorf("expected SCHMA record, got %s", schmaRecord.Tag)
	}

	schema := make(map[string]interface{})

	for _, sub := range schmaRecord.SubRecords {
		switch sub.Tag {
		case "TAG":
			// Extension tag definition
			tagDef := make(map[string]interface{})
			tagDef["name"] = sub.Value

			for _, tagSub := range sub.SubRecords {
				switch tagSub.Tag {
				case "TYPE":
					tagDef["type"] = tagSub.Value
				case "LABL":
					tagDef["label"] = tagSub.Value
				}
			}

			if schema["tags"] == nil {
				schema["tags"] = make(map[string]interface{})
			}
			tags := schema["tags"].(map[string]interface{})
			tags[sub.Value] = tagDef
		}
	}

	// Store schema
	ctx.ExtensionSchemas[schmaRecord.XRef] = schema

	ctx.Logger.LogInfo(fmt.Sprintf("Stored extension schema %s", schmaRecord.XRef))

	return nil
}

// convertExtensionData converts extension tag data to properties
func convertExtensionData(tag string, value string, subRecords []*GEDCOMRecord, ctx *ConversionContext) map[string]interface{} {
	properties := make(map[string]interface{})

	// Extension tags start with underscore
	if len(tag) == 0 || tag[0] != '_' {
		return properties
	}

	// Store the value
	properties["extension_tag"] = tag
	if value != "" {
		properties["value"] = value
	}

	// Store subrecords if any
	if len(subRecords) > 0 {
		subData := make(map[string]interface{})
		for _, sub := range subRecords {
			subData[sub.Tag] = sub.Value
		}
		properties["subrecords"] = subData
	}

	return properties
}

// extractEventDateTime extracts combined date and time from an event record
func extractEventDateTime(eventRecord *GEDCOMRecord) string {
	var dateStr, timeStr string

	for _, sub := range eventRecord.SubRecords {
		switch sub.Tag {
		case "DATE":
			dateStr = parseGEDCOMDate(sub.Value)

			// Check for TIME subrecord under DATE (GEDCOM 7.0)
			for _, dateSub := range sub.SubRecords {
				if dateSub.Tag == "TIME" {
					timeStr = parseGEDCOMTime(dateSub.Value)
				}
			}

		case "TIME":
			// TIME at event level (less common)
			if timeStr == "" {
				timeStr = parseGEDCOMTime(sub.Value)
			}
		}
	}

	// Combine date and time if both present
	if dateStr != "" && timeStr != "" {
		return combineDateAndTime(dateStr, timeStr)
	}

	return dateStr
}

// extractPhraseValue extracts the PHRASE value from a record
// PHRASE is used in GEDCOM 7.0 to provide human-readable override for enumeration values
func extractPhraseValue(record *GEDCOMRecord) string {
	for _, sub := range record.SubRecords {
		if sub.Tag == "PHRASE" {
			return sub.Value
		}
	}
	return ""
}

// convertEventTypeWithPhrase converts event type, using PHRASE override if present
func convertEventTypeWithPhrase(eventRecord *GEDCOMRecord, defaultType string) (string, map[string]interface{}) {
	properties := make(map[string]interface{})

	// Check for PHRASE override
	phrase := extractPhraseValue(eventRecord)
	if phrase != "" {
		properties["original_phrase"] = phrase
		// Use phrase as custom event type if it differs significantly from default
		if !strings.EqualFold(phrase, defaultType) {
			properties["custom_type"] = phrase
		}
	}

	return defaultType, properties
}

// mapEnumeration maps GEDCOM 7.0 enumeration values to GLX equivalents
func mapEnumeration(tag string, value string) string {
	// GEDCOM 7.0 uses uppercase enumeration values
	// This function maps them to GLX conventions

	valueLower := strings.ToLower(value)

	switch tag {
	case "RESN":
		// Restriction notice
		mapping := map[string]string{
			"confidential": "confidential",
			"locked":       "locked",
			"privacy":      "privacy",
		}
		if mapped, ok := mapping[valueLower]; ok {
			return mapped
		}

	case "PEDI":
		// Pedigree linkage type
		mapping := map[string]string{
			"adopted":  "adopted",
			"birth":    "birth",
			"foster":   "foster",
			"sealing":  "sealing",
			"multiple": "multiple",
		}
		if mapped, ok := mapping[valueLower]; ok {
			return mapped
		}

	case "QUAY":
		// Quality of data (already handled in mapQUAYtoConfidence)
		// But return the value as-is for properties
		return value

	case "STAT":
		// Status
		mapping := map[string]string{
			"challenged":  "challenged",
			"disproven":   "disproven",
			"proven":      "proven",
			"bic":         "born_in_covenant",
			"child":       "child_sealed",
			"completed":   "completed",
			"dns":         "do_not_seal",
			"dns_can":     "dns_cancelled",
			"pre_1970":    "pre_1970",
			"stillborn":   "stillborn",
			"submitted":   "submitted",
			"uncleared":   "uncleared",
		}
		if mapped, ok := mapping[valueLower]; ok {
			return mapped
		}

	case "ROLE":
		// Role in event
		mapping := map[string]string{
			"chil":    "child",
			"clergy":  "clergy",
			"fath":    "father",
			"friend":  "friend",
			"godp":    "godparent",
			"husb":    "husband",
			"moth":    "mother",
			"multiple": "multiple",
			"nghbr":   "neighbor",
			"officiator": "officiator",
			"parent":  "parent",
			"spou":    "spouse",
			"wife":    "wife",
			"witn":    "witness",
		}
		if mapped, ok := mapping[valueLower]; ok {
			return mapped
		}

	case "MEDI":
		// Medium type
		mapping := map[string]string{
			"audio":       "audio",
			"book":        "book",
			"card":        "card",
			"electronic":  "electronic",
			"fiche":       "fiche",
			"film":        "film",
			"magazine":    "magazine",
			"manuscript":  "manuscript",
			"map":         "map",
			"newspaper":   "newspaper",
			"photo":       "photo",
			"tombstone":   "tombstone",
			"video":       "video",
		}
		if mapped, ok := mapping[valueLower]; ok {
			return mapped
		}
	}

	// Return original value if no mapping found
	return value
}

// extractRestrictionNotice extracts GEDCOM 7.0 RESN (restriction notice)
func extractRestrictionNotice(record *GEDCOMRecord) string {
	for _, sub := range record.SubRecords {
		if sub.Tag == "RESN" {
			return mapEnumeration("RESN", sub.Value)
		}
	}
	return ""
}

// extractPedigree extracts GEDCOM 7.0 PEDI (pedigree linkage) from a FAMC record
func extractPedigree(famcRecord *GEDCOMRecord) string {
	for _, sub := range famcRecord.SubRecords {
		if sub.Tag == "PEDI" {
			return mapEnumeration("PEDI", sub.Value)
		}
	}
	return ""
}

// extractStatus extracts GEDCOM 7.0 STAT (status) value
func extractStatus(record *GEDCOMRecord) string {
	for _, sub := range record.SubRecords {
		if sub.Tag == "STAT" {
			return mapEnumeration("STAT", sub.Value)
		}
	}
	return ""
}

// extractRole extracts GEDCOM 7.0 ROLE value from an event association
func extractRole(record *GEDCOMRecord) string {
	for _, sub := range record.SubRecords {
		if sub.Tag == "ROLE" {
			return mapEnumeration("ROLE", sub.Value)
		}
	}
	return ""
}
