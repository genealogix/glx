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
func convertSource(sourRecord *GEDCOMRecord, conv *ConversionContext) error {
	if sourRecord.Tag != GedcomTagSour {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedSourceRecord, GedcomTagSour, sourRecord.Tag)
	}

	// Generate source ID
	sourceID := generateSourceID(conv)
	conv.SourceIDMap[sourRecord.XRef] = sourceID

	conv.Logger.LogInfo(fmt.Sprintf("Converting SOUR %s -> %s", sourRecord.XRef, sourceID))

	// Create source entity with properties map
	source := &Source{
		Properties: make(map[string]any),
	}

	var notes []string
	var description []string
	var eventsRecorded []string

	// Extract external IDs (GEDCOM 7.0 EXID tags) and store in properties
	exids := extractExternalIDs(sourRecord)
	if len(exids) > 0 {
		var externalIDs []string
		for _, exid := range exids {
			if idType, ok := exid["type"]; ok {
				externalIDs = append(externalIDs, fmt.Sprintf("%s:%s", idType, exid["id"]))
			} else {
				externalIDs = append(externalIDs, exid["id"])
			}
		}
		source.Properties[SourcePropertyExternalIDs] = externalIDs
	}

	// Process subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagTitl:
			// Title
			source.Title = sub.Value

		case GedcomTagAuth:
			// Author
			source.Authors = append(source.Authors, sub.Value)

		case GedcomTagPubl:
			// Publication facts - store in properties
			source.Properties[SourcePropertyPublicationInfo] = sub.Value

		case GedcomTagAbbr:
			// Abbreviation - store in properties
			source.Properties[SourcePropertyAbbreviation] = sub.Value

		case GedcomTagRepo:
			// Repository reference
			repoID := conv.RepositoryIDMap[sub.Value]
			if repoID != "" {
				source.RepositoryID = repoID

				// Extract call number - store in properties
				for _, repoSub := range sub.SubRecords {
					if repoSub.Tag == GedcomTagCaln {
						source.Properties[SourcePropertyCallNumber] = repoSub.Value
					}
				}
			}

		case GedcomTagText:
			// Full source text - add to description
			description = append(description, sub.Value)

		case GedcomTagNote:
			// Notes
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				notes = append(notes, noteText)
			}

		case GedcomTagData:
			// Data information - store in properties
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case GedcomTagEven:
					// Events recorded - accumulate for multi-value property
					eventsRecorded = append(eventsRecorded, dataSub.Value)
				case GedcomTagAgnc:
					// Agency - store in properties
					source.Properties[SourcePropertyAgency] = dataSub.Value
				case GedcomTagDate:
					// Date of data
					source.Date = parseGEDCOMDate(dataSub.Value)
				}
			}

		case GedcomTagObje:
			// Media object reference
			if sub.Value != "" {
				mediaID := conv.MediaIDMap[sub.Value]
				if mediaID != "" {
					source.Media = append(source.Media, mediaID)
				}
			}

		case GedcomTagType:
			// Source type (GEDCOM 7.0)
			source.Type = mapSourceType(sub.Value)
		}
	}

	// Combine description
	if len(description) > 0 {
		source.Description = strings.Join(description, "\n")
	}

	// Store events recorded if any (multi-value property)
	if len(eventsRecorded) > 0 {
		source.Properties[SourcePropertyEventsRecorded] = eventsRecorded
	}

	// Clear empty properties map
	if len(source.Properties) == 0 {
		source.Properties = nil
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
	conv.GLX.Sources[sourceID] = source
	conv.Stats.SourcesCreated++

	return nil
}

// mapSourceType maps GEDCOM source type to GLX
func mapSourceType(gedcomType string) string {
	// Common GEDCOM source type values
	mapping := map[string]string{
		"book":       SourceTypeBook,
		"article":    SourceTypeBook,
		"website":    SourceTypeDatabase,
		"database":   SourceTypeDatabase,
		"census":     SourceTypeCensus,
		"vital":      SourceTypeVitalRecord,
		"church":     SourceTypeChurchRegister,
		"military":   SourceTypeMilitary,
		"newspaper":  SourceTypeNewspaper,
		"probate":    SourceTypeProbate,
		"land":       SourceTypeLand,
		"court":      SourceTypeCourt,
		"photo":      SourceTypePhotograph,
		"photograph": SourceTypePhotograph,
	}

	typeLower := strings.ToLower(gedcomType)
	if mapped, ok := mapping[typeLower]; ok {
		return mapped
	}

	return SourceTypeOther
}

// inferSourceType infers source type from title
func inferSourceType(title string) string {
	titleLower := strings.ToLower(title)

	// Check for keywords
	if strings.Contains(titleLower, "census") {
		return SourceTypeCensus
	}
	if strings.Contains(titleLower, "birth certificate") || strings.Contains(titleLower, "death certificate") {
		return SourceTypeVitalRecord
	}
	if strings.Contains(titleLower, "baptism") || strings.Contains(titleLower, "parish register") {
		return SourceTypeChurchRegister
	}
	if strings.Contains(titleLower, "military") {
		return SourceTypeMilitary
	}
	if strings.Contains(titleLower, "newspaper") {
		return SourceTypeNewspaper
	}
	if strings.Contains(titleLower, "will") || strings.Contains(titleLower, "probate") {
		return SourceTypeProbate
	}
	if strings.Contains(titleLower, "deed") || strings.Contains(titleLower, "land") {
		return SourceTypeLand
	}

	// Default
	return SourceTypeOther
}
