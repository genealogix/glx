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

// exportSource converts a GLX Source to a GEDCOM SOUR record.
func exportSource(sourceID string, source *Source, expCtx *ExportContext) *GEDCOMRecord {
	xref := expCtx.SourceXRefMap[sourceID]

	record := &GEDCOMRecord{
		XRef:       xref,
		Tag:        GedcomTagSour,
		SubRecords: []*GEDCOMRecord{},
	}

	// TITL
	if source.Title != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagTitl,
			Value: source.Title,
		})
	}

	// AUTH - join multiple authors with "; "
	if len(source.Authors) > 0 {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagAuth,
			Value: strings.Join(source.Authors, "; "),
		})
	}

	// PUBL from publication_info property
	if pubInfo, ok := getStringProperty(source.Properties, "publication_info"); ok {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagPubl,
			Value: pubInfo,
		})
	}

	// ABBR from abbreviation property
	if abbr, ok := getStringProperty(source.Properties, "abbreviation"); ok {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagAbbr,
			Value: abbr,
		})
	}

	// REPO with optional CALN
	if source.RepositoryID != "" {
		repoXRef := expCtx.RepositoryXRefMap[source.RepositoryID]
		if repoXRef != "" {
			repoRecord := &GEDCOMRecord{
				Tag:        GedcomTagRepo,
				Value:      repoXRef,
				SubRecords: []*GEDCOMRecord{},
			}

			// CALN from call_number property
			if callNum, ok := getStringProperty(source.Properties, "call_number"); ok {
				repoRecord.SubRecords = append(repoRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagCaln,
					Value: callNum,
				})
			}

			record.SubRecords = append(record.SubRecords, repoRecord)
		} else {
			expCtx.addExportWarning(EntityTypeSources, sourceID,
				fmt.Sprintf("repository %s has no XREF mapping", source.RepositoryID))
		}
	}

	// DATA subrecord (contains DATE, AGNC, EVEN)
	dataRecord := buildSourceDataRecord(source)
	if dataRecord != nil {
		record.SubRecords = append(record.SubRecords, dataRecord)
	}

	// TEXT from Description
	if source.Description != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagText,
			Value: source.Description,
		})
	}

	// OBJE references for media
	for _, mediaID := range source.Media {
		mediaXRef := expCtx.MediaXRefMap[mediaID]
		if mediaXRef != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagObje,
				Value: mediaXRef,
			})
		} else {
			expCtx.addExportWarning(EntityTypeSources, sourceID,
				fmt.Sprintf("media %s has no XREF mapping", mediaID))
		}
	}

	// NOTE
	if source.Notes != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: source.Notes,
		})
	}

	// TYPE (GEDCOM 7.0 only)
	if expCtx.Version == GEDCOM70 && source.Type != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagType,
			Value: source.Type,
		})
	}

	return record
}

// buildSourceDataRecord builds a DATA subrecord from source date, agency,
// and events_recorded properties. Returns nil if there is no data to include.
func buildSourceDataRecord(source *Source) *GEDCOMRecord {
	dataRecord := &GEDCOMRecord{
		Tag:        GedcomTagData,
		SubRecords: []*GEDCOMRecord{},
	}

	// EVEN from events_recorded property
	if eventsRecorded, ok := source.Properties["events_recorded"]; ok {
		switch v := eventsRecorded.(type) {
		case []string:
			for _, event := range v {
				dataRecord.SubRecords = append(dataRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagEven,
					Value: event,
				})
			}
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					dataRecord.SubRecords = append(dataRecord.SubRecords, &GEDCOMRecord{
						Tag:   GedcomTagEven,
						Value: s,
					})
				}
			}
		case string:
			dataRecord.SubRecords = append(dataRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagEven,
				Value: v,
			})
		}
	}

	// AGNC from agency property
	if agency, ok := getStringProperty(source.Properties, "agency"); ok {
		dataRecord.SubRecords = append(dataRecord.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagAgnc,
			Value: agency,
		})
	}

	// DATE from source.Date
	if source.Date != "" {
		gedcomDate := formatGEDCOMDate(source.Date)
		if gedcomDate != "" {
			dataRecord.SubRecords = append(dataRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagDate,
				Value: gedcomDate,
			})
		}
	}

	if len(dataRecord.SubRecords) == 0 {
		return nil
	}

	return dataRecord
}

// getStringProperty extracts a string value from a properties map.
func getStringProperty(props map[string]any, key string) (string, bool) {
	if props == nil {
		return "", false
	}

	val, ok := props[key]
	if !ok {
		return "", false
	}

	s, ok := val.(string)
	if !ok || s == "" {
		return "", false
	}

	return s, true
}

