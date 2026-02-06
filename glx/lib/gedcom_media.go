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
	"path/filepath"
	"strconv"
	"strings"
)

// convertMedia converts a GEDCOM OBJE record to a GLX Media entity
func convertMedia(objeRecord *GEDCOMRecord, conv *ConversionContext) error {
	if objeRecord.Tag != GedcomTagObje {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedMediaRecord, GedcomTagObje, objeRecord.Tag)
	}

	// Generate media ID
	mediaID := generateMediaID(conv)
	conv.MediaIDMap[objeRecord.XRef] = mediaID

	conv.Logger.LogInfo(fmt.Sprintf("Converting OBJE %s -> %s", objeRecord.XRef, mediaID))

	// Convert using common logic
	media := convertMediaCommon(objeRecord, conv)

	// Handle SOUR subrecords (only for top-level OBJE records)
	for _, sub := range objeRecord.SubRecords {
		if sub.Tag == GedcomTagSour {
			citationID, err := createCitationFromSOUR(mediaID, sub, conv)
			if err != nil {
				// Error already logged in createCitationFromSOUR, skip this citation
				continue
			}
			if citationID != "" {
				if citation, ok := conv.GLX.Citations[citationID]; ok {
					citation.Media = append(citation.Media, mediaID)
				}
			}
		}
	}

	// Store media
	conv.GLX.Media[mediaID] = media
	conv.Stats.MediaCreated++

	return nil
}

// convertEmbeddedMedia converts an embedded OBJE (without XRef) and returns the media ID
func convertEmbeddedMedia(objeRecord *GEDCOMRecord, conv *ConversionContext) (string, error) {
	// Generate media ID for embedded object
	mediaID := generateMediaID(conv)

	conv.Logger.LogInfo("Converting embedded OBJE -> " + mediaID)

	// Convert using common logic
	media := convertMediaCommon(objeRecord, conv)

	// Store media
	conv.GLX.Media[mediaID] = media
	conv.Stats.MediaCreated++

	return mediaID, nil
}

// convertMediaCommon contains the shared logic for converting GEDCOM OBJE records to GLX Media entities.
// It processes FILE, FORM, TITL, CROP, and NOTE subrecords.
func convertMediaCommon(objeRecord *GEDCOMRecord, conv *ConversionContext) *Media {
	media := &Media{
		Properties: make(map[string]any),
	}

	var fileRef string
	var formatType string
	var notes []string

	// Process subrecords
	for _, sub := range objeRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagFile:
			// File reference - primary in both 5.5.1 and 7.0
			fileRef = sub.Value

			// Check for GEDCOM 7.0 MIME type as subrecord
			for _, fileSub := range sub.SubRecords {
				switch fileSub.Tag {
				case GedcomTagForm:
					// GEDCOM 5.5.1: FORM under FILE
					formatType = fileSub.Value
				case GedcomTagMime:
					// GEDCOM 7.0: MIME type
					media.MimeType = fileSub.Value
				case GedcomTagTitl:
					// Title under FILE
					media.Title = fileSub.Value
				}
			}

		case GedcomTagForm:
			// GEDCOM 5.5.1: Format at OBJE level
			formatType = sub.Value

			// Check for MEDI subrecord - store as property
			for _, formSub := range sub.SubRecords {
				if formSub.Tag == GedcomTagMedi {
					if propertyKey, ok := conv.GEDCOMIndex.MediaProperties[formSub.Tag]; ok {
						media.Properties[propertyKey] = formSub.Value
					}
				}
			}

		case GedcomTagTitl:
			// Title (can be at OBJE level or FILE level)
			if media.Title == "" {
				media.Title = sub.Value
			}

		case GedcomTagCrop:
			// GEDCOM 7.0: Crop coordinates - store as structured property
			crop := extractCrop(sub)
			if crop != nil {
				if propertyKey, ok := conv.GEDCOMIndex.MediaProperties[sub.Tag]; ok {
					media.Properties[propertyKey] = crop
				}
			}

		case GedcomTagNote:
			// Notes/description
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				notes = append(notes, noteText)
			}
		}
	}

	// Set file reference as URI
	media.URI = fileRef

	// Infer MIME type if not set (GEDCOM 5.5.1)
	if media.MimeType == "" {
		if formatType != "" {
			media.MimeType = mapFormatToMimeType(formatType)
		} else if fileRef != "" {
			media.MimeType = inferMimeType(fileRef)
		}
	}

	// Default type if still not set
	if media.MimeType == "" {
		media.MimeType = "application/octet-stream"
	}

	// Combine notes
	if len(notes) > 0 {
		media.Notes = strings.Join(notes, "\n")
	}

	// Clean up empty Properties map so it doesn't appear in YAML
	if len(media.Properties) == 0 {
		media.Properties = nil
	}

	return media
}

// inferMimeType infers MIME type from file extension.
// Uses mimeTypeByExtension from constants.go.
func inferMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mime, ok := mimeTypeByExtension[ext]; ok {
		return mime
	}

	return MimeTypeOctetStream
}

// mapFormatToMimeType maps GEDCOM 5.5.1 FORM values to MIME types.
// Uses mimeTypeByFormat from constants.go.
func mapFormatToMimeType(format string) string {
	formatLower := strings.ToLower(format)
	if mime, ok := mimeTypeByFormat[formatLower]; ok {
		return mime
	}

	return MimeTypeOctetStream
}

// extractCrop extracts GEDCOM 7.0 crop coordinates
func extractCrop(cropRecord *GEDCOMRecord) map[string]any {
	crop := make(map[string]any)

	for _, sub := range cropRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagTop:
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop[CropKeyTop] = val
			}
		case GedcomTagLeft:
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop[CropKeyLeft] = val
			}
		case GedcomTagHeight:
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop[CropKeyHeight] = val
			}
		case GedcomTagWidth:
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop[CropKeyWidth] = val
			}
		}
	}

	if len(crop) == 0 {
		return nil
	}

	return crop
}
