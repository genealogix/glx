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

// convertSource converts a GEDCOM SOUR record to a GLX Source
func convertSource(sourRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if sourRecord.Tag != "SOUR" {
		return fmt.Errorf("expected SOUR record, got %s", sourRecord.Tag)
	}

	// Generate source ID
	sourceID := generateSourceID(ctx)
	ctx.SourceIDMap[sourRecord.XRef] = sourceID

	ctx.Logger.LogInfo(fmt.Sprintf("Converting SOUR %s -> %s", sourRecord.XRef, sourceID))

	// Create source entity
	source := &Source{
		Properties: make(map[string]interface{}),
	}

	// Process subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case "TITL":
			// Title
			source.Title = sub.Value

		case "AUTH":
			// Author
			source.Properties["author"] = sub.Value

		case "PUBL":
			// Publication facts
			source.Properties["publication_info"] = sub.Value

		case "ABBR":
			// Abbreviation
			source.Properties["abbreviation"] = sub.Value

		case "REPO":
			// Repository reference
			repoID := ctx.RepositoryIDMap[sub.Value]
			if repoID != "" {
				source.Repository = repoID

				// Extract call number
				for _, repoSub := range sub.SubRecords {
					if repoSub.Tag == "CALN" {
						source.Properties["call_number"] = repoSub.Value
					}
				}
			}

		case "TEXT":
			// Full source text
			source.Properties["source_text"] = sub.Value

		case "NOTE":
			// Notes
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				source.Properties["notes"] = noteText
			}

		case "DATA":
			// Data information
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case "EVEN":
					// Events recorded
					source.Properties["events_recorded"] = dataSub.Value
				case "AGNC":
					// Agency
					source.Properties["agency"] = dataSub.Value
				}
			}

		case "OBJE":
			// Media object - not yet implemented
			ctx.addWarning(sourRecord.Line, "OBJE", "Media linking not yet implemented")

		case "TYPE":
			// Source type (GEDCOM 7.0)
			source.Type = mapSourceType(sub.Value)
		}
	}

	// Default type if not set
	if source.Type == "" {
		source.Type = inferSourceType(source.Title, source.Properties)
	}

	// Store source
	ctx.GLX.Sources[sourceID] = source
	ctx.Stats.SourcesCreated++

	return nil
}

// mapSourceType maps GEDCOM source type to GLX
func mapSourceType(gedcomType string) string {
	// Common GEDCOM source type values
	mapping := map[string]string{
		"book":       "book",
		"article":    "book",
		"website":    "database",
		"database":   "database",
		"census":     "census",
		"vital":      "vital_record",
		"church":     "church_register",
		"military":   "military",
		"newspaper":  "newspaper",
		"probate":    "probate",
		"land":       "land",
		"court":      "court",
		"photo":      "photograph",
		"photograph": "photograph",
	}

	typeLower := strings.ToLower(gedcomType)
	if mapped, ok := mapping[typeLower]; ok {
		return mapped
	}

	return "other"
}

// inferSourceType infers source type from title and properties
func inferSourceType(title string, properties map[string]interface{}) string {
	titleLower := strings.ToLower(title)

	// Check for keywords
	if strings.Contains(titleLower, "census") {
		return "census"
	}
	if strings.Contains(titleLower, "birth certificate") || strings.Contains(titleLower, "death certificate") {
		return "vital_record"
	}
	if strings.Contains(titleLower, "baptism") || strings.Contains(titleLower, "parish register") {
		return "church_register"
	}
	if strings.Contains(titleLower, "military") {
		return "military"
	}
	if strings.Contains(titleLower, "newspaper") {
		return "newspaper"
	}
	if strings.Contains(titleLower, "will") || strings.Contains(titleLower, "probate") {
		return "probate"
	}
	if strings.Contains(titleLower, "deed") || strings.Contains(titleLower, "land") {
		return "land"
	}

	// Default
	return "other"
}
