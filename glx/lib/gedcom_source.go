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
		Properties: make(map[string]any),
	}

	var notes []string
	var description []string

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	exids := extractExternalIDs(sourRecord)
	if len(exids) > 0 {
		source.Properties["external_ids"] = exids
	}

	// Process subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case "TITL":
			// Title
			source.Title = sub.Value

		case "AUTH":
			// Author
			source.Authors = append(source.Authors, sub.Value)

		case "PUBL":
			// Publication facts
			source.PublicationInfo = sub.Value

		case "ABBR":
			// Abbreviation - store in notes
			notes = append(notes, "Abbreviation: "+sub.Value)

		case "REPO":
			// Repository reference
			repoID := ctx.RepositoryIDMap[sub.Value]
			if repoID != "" {
				source.RepositoryID = repoID

				// Extract call number - store in notes
				for _, repoSub := range sub.SubRecords {
					if repoSub.Tag == "CALN" {
						notes = append(notes, "Call number: "+repoSub.Value)
					}
				}
			}

		case "TEXT":
			// Full source text - add to description
			description = append(description, sub.Value)

		case "NOTE":
			// Notes
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				notes = append(notes, noteText)
			}

		case "DATA":
			// Data information - store in notes
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case "EVEN":
					// Events recorded
					notes = append(notes, "Events recorded: "+dataSub.Value)
				case "AGNC":
					// Agency
					notes = append(notes, "Agency: "+dataSub.Value)
				case "DATE":
					// Date of data
					source.Date = parseGEDCOMDate(dataSub.Value)
				}
			}

		case "OBJE":
			// Media object reference
			if sub.Value != "" {
				mediaID := ctx.MediaIDMap[sub.Value]
				if mediaID != "" {
					source.Media = append(source.Media, mediaID)
				}
			}

		case "TYPE":
			// Source type (GEDCOM 7.0)
			source.Type = mapSourceType(sub.Value)
		}
	}

	// Combine description
	if len(description) > 0 {
		source.Description = strings.Join(description, "\n")
	}

	// Combine notes
	if len(notes) > 0 {
		source.Notes = strings.Join(notes, "\n")
	}

	// Default type if not set
	if source.Type == "" {
		source.Type = inferSourceType(source.Title)
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

// inferSourceType infers source type from title
func inferSourceType(title string) string {
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
