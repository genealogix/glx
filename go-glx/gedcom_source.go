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

// convertSource converts a GEDCOM SOUR record to a GLX Source
//
//nolint:gocognit,gocyclo
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
	if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[GedcomTagExid]; ok {
		extractExternalIDs(sourRecord, propertyKey, source.Properties)
	}

	// Process subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagTitl:
			// Title - may have CONT/CONC for long titles
			source.Title = extractTextWithContinuation(sub)

		case GedcomTagAuth:
			// Author - may have CONT/CONC
			source.Authors = append(source.Authors, extractTextWithContinuation(sub))

		case GedcomTagPubl:
			// Publication facts - may have CONT/CONC for long publication info
			if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[sub.Tag]; ok {
				source.Properties[propertyKey] = extractTextWithContinuation(sub)
			}

		case GedcomTagAbbr:
			// Abbreviation - unlikely to have CONT/CONC but handle it anyway
			if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[sub.Tag]; ok {
				source.Properties[propertyKey] = extractTextWithContinuation(sub)
			}

		case GedcomTagRepo:
			// Repository reference
			repoID := conv.RepositoryIDMap[sub.Value]
			if repoID != "" {
				source.RepositoryID = repoID

				// Extract call number - store in properties
				for _, repoSub := range sub.SubRecords {
					if repoSub.Tag == GedcomTagCaln {
						if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[repoSub.Tag]; ok {
							source.Properties[propertyKey] = extractTextWithContinuation(repoSub)
						}
					}
				}
			}

		case GedcomTagText:
			// Full source text - may have CONT/CONC for long text
			description = append(description, extractTextWithContinuation(sub))

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
					if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[dataSub.Tag]; ok {
						source.Properties[propertyKey] = dataSub.Value
					}
				case GedcomTagDate:
					// Date of data
					source.Date = parseGEDCOMDate(dataSub.Value)
				}
			}

		case GedcomTagObje:
			// Media object reference or embedded media
			if mediaID := resolveOBJE(sub, conv); mediaID != "" {
				source.Media = append(source.Media, mediaID)
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
		if propertyKey, ok := conv.GEDCOMIndex.SourceProperties[GedcomTagEven]; ok {
			source.Properties[propertyKey] = eventsRecorded
		}
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

// mapSourceType maps GEDCOM source type to GLX.
// Uses gedcomSourceTypeMapping from constants.go.
func mapSourceType(gedcomType string) string {
	typeLower := strings.ToLower(gedcomType)
	if mapped, ok := gedcomSourceTypeMapping[typeLower]; ok {
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
	if strings.Contains(titleLower, "population register") || strings.Contains(titleLower, "household register") {
		return SourceTypePopulationRegister
	}
	if strings.Contains(titleLower, "tax roll") || strings.Contains(titleLower, "tax record") || strings.Contains(titleLower, "tithe") {
		return SourceTypeTaxRecord
	}
	if strings.Contains(titleLower, "notarial") || strings.Contains(titleLower, "notary") {
		return SourceTypeNotarialRecord
	}

	// Default
	return SourceTypeOther
}
