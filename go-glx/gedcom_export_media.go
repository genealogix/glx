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
	"sort"
)

// mimeToGEDCOMFormat maps MIME types back to GEDCOM 5.5.1 FORM values.
// Built by inverting mimeTypeByFormat, preferring shorter format names.
var mimeToGEDCOMFormat = buildMimeToGEDCOMFormat()

func buildMimeToGEDCOMFormat() map[string]string {
	result := make(map[string]string)

	// Sort keys for deterministic behavior when multiple keys map to the same MIME type
	keys := make([]string, 0, len(mimeTypeByFormat))
	for k := range mimeTypeByFormat {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := mimeTypeByFormat[k]
		// Prefer shorter format name (e.g., "jpg" over "jpeg", "tif" over "tiff")
		if existing, ok := result[v]; !ok || len(k) < len(existing) {
			result[v] = k
		}
	}

	return result
}

// exportMedia converts a GLX Media to a GEDCOM OBJE record.
func exportMedia(mediaID string, media *Media, expCtx *ExportContext) *GEDCOMRecord {
	xref := expCtx.MediaXRefMap[mediaID]

	record := &GEDCOMRecord{
		XRef:       xref,
		Tag:        GedcomTagObje,
		SubRecords: []*GEDCOMRecord{},
	}

	// FILE subrecord with format/MIME information
	if media.URI != "" {
		fileRecord := &GEDCOMRecord{
			Tag:        GedcomTagFile,
			Value:      media.URI,
			SubRecords: []*GEDCOMRecord{},
		}

		if expCtx.Version == GEDCOM70 {
			// GEDCOM 7.0: MIME under FILE
			if media.MimeType != "" {
				fileRecord.SubRecords = append(fileRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagMime,
					Value: media.MimeType,
				})
			}

			// GEDCOM 7.0: TITL under FILE
			if media.Title != "" {
				fileRecord.SubRecords = append(fileRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagTitl,
					Value: media.Title,
				})
			}
		} else {
			// GEDCOM 5.5.1: FORM under FILE (convert MIME to format string)
			if media.MimeType != "" {
				formValue := mimeToGEDCOMFormat[media.MimeType]
				if formValue != "" {
					fileRecord.SubRecords = append(fileRecord.SubRecords, &GEDCOMRecord{
						Tag:   GedcomTagForm,
						Value: formValue,
					})
				}
			}
		}

		record.SubRecords = append(record.SubRecords, fileRecord)
	}

	// GEDCOM 5.5.1: TITL at OBJE level (not under FILE)
	if expCtx.Version == GEDCOM551 && media.Title != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagTitl,
			Value: media.Title,
		})
	}

	// GEDCOM 5.5.1: FORM/MEDI from medium property
	if expCtx.Version == GEDCOM551 {
		if medium, ok := getStringProperty(media.Properties, "medium"); ok {
			formRecord := &GEDCOMRecord{
				Tag: GedcomTagForm,
				SubRecords: []*GEDCOMRecord{
					{
						Tag:   GedcomTagMedi,
						Value: medium,
					},
				},
			}
			record.SubRecords = append(record.SubRecords, formRecord)
		}
	}

	// GEDCOM 7.0: CROP from crop property
	if expCtx.Version == GEDCOM70 {
		if cropVal, ok := media.Properties["crop"]; ok {
			cropRecord := buildCropRecord(cropVal)
			if cropRecord != nil {
				record.SubRecords = append(record.SubRecords, cropRecord)
			}
		}
	}

	// NOTE
	if media.Notes != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: media.Notes,
		})
	}

	return record
}

// buildCropRecord constructs a CROP record from crop property data.
// The crop value can be map[string]any (from runtime) or map[string]int.
func buildCropRecord(cropVal any) *GEDCOMRecord {
	cropMap, ok := cropVal.(map[string]any)
	if !ok {
		return nil
	}

	cropRecord := &GEDCOMRecord{
		Tag:        GedcomTagCrop,
		SubRecords: []*GEDCOMRecord{},
	}

	// Add crop coordinates in a deterministic order
	cropFields := []struct {
		key string
		tag string
	}{
		{CropKeyTop, GedcomTagTop},
		{CropKeyLeft, GedcomTagLeft},
		{CropKeyHeight, GedcomTagHeight},
		{CropKeyWidth, GedcomTagWidth},
	}

	for _, field := range cropFields {
		if val, exists := cropMap[field.key]; exists {
			valStr := formatCropValue(val)
			if valStr != "" {
				cropRecord.SubRecords = append(cropRecord.SubRecords, &GEDCOMRecord{
					Tag:   field.tag,
					Value: valStr,
				})
			}
		}
	}

	if len(cropRecord.SubRecords) == 0 {
		return nil
	}

	return cropRecord
}

// formatCropValue converts a crop coordinate value to a string.
// Handles int, float64, and string types.
func formatCropValue(val any) string {
	switch v := val.(type) {
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%d", int(v))
	case string:
		return v
	default:
		return ""
	}
}
