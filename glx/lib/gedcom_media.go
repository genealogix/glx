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
		return fmt.Errorf("%w: expected OBJE, got %s", ErrUnexpectedMediaRecord, objeRecord.Tag)
	}

	// Generate media ID
	mediaID := generateMediaID(conv)
	conv.MediaIDMap[objeRecord.XRef] = mediaID

	conv.Logger.LogInfo(fmt.Sprintf("Converting OBJE %s -> %s", objeRecord.XRef, mediaID))

	// Create media entity
	media := &Media{}

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

			// Check for MEDI subrecord (store in notes)
			for _, formSub := range sub.SubRecords {
				if formSub.Tag == GedcomTagMedi {
					notes = append(notes, "Medium: "+formSub.Value)
				}
			}

		case GedcomTagTitl:
			// Title (can be at OBJE level or FILE level)
			if media.Title == "" {
				media.Title = sub.Value
			}

		case GedcomTagCrop:
			// GEDCOM 7.0: Crop coordinates stored in notes (should be a field - see todo.md)
			crop := extractCrop(sub)
			if crop != nil {
				notes = append(notes, fmt.Sprintf("Crop: %+v", crop))
			}

		case GedcomTagNote:
			// Notes/description
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				notes = append(notes, noteText)
			}

		case GedcomTagSour:
			// Source citations stored in notes (should be a field - see todo.md)
			citationID, err := createCitationFromSOUR(mediaID, sub, conv)
			if err == nil && citationID != "" {
				notes = append(notes, "Citation: "+citationID)
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

	// Create media entity
	media := &Media{}

	var fileRef string
	var formatType string
	var notes []string

	// Process subrecords (same as top-level OBJE)
	for _, sub := range objeRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagFile:
			fileRef = sub.Value

			for _, fileSub := range sub.SubRecords {
				switch fileSub.Tag {
				case GedcomTagForm:
					formatType = fileSub.Value
				case GedcomTagMime:
					media.MimeType = fileSub.Value
				case GedcomTagTitl:
					media.Title = fileSub.Value
				}
			}

		case GedcomTagForm:
			formatType = sub.Value

			for _, formSub := range sub.SubRecords {
				if formSub.Tag == GedcomTagMedi {
					notes = append(notes, "Medium: "+formSub.Value)
				}
			}

		case GedcomTagTitl:
			if media.Title == "" {
				media.Title = sub.Value
			}

		case GedcomTagCrop:
			// Crop coordinates stored in notes (should be a field - see todo.md)
			crop := extractCrop(sub)
			if crop != nil {
				notes = append(notes, fmt.Sprintf("Crop: %+v", crop))
			}

		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				notes = append(notes, noteText)
			}
		}
	}

	media.URI = fileRef

	// Infer MIME type
	if media.MimeType == "" {
		if formatType != "" {
			media.MimeType = mapFormatToMimeType(formatType)
		} else if fileRef != "" {
			media.MimeType = inferMimeType(fileRef)
		}
	}

	if media.MimeType == "" {
		media.MimeType = "application/octet-stream"
	}

	// Combine notes
	if len(notes) > 0 {
		media.Notes = strings.Join(notes, "\n")
	}

	// Store media
	conv.GLX.Media[mediaID] = media
	conv.Stats.MediaCreated++

	return mediaID, nil
}

// inferMimeType infers MIME type from file extension
func inferMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeTypes := map[string]string{
		// Images
		".jpg":  MimeTypeJPEG,
		".jpeg": MimeTypeJPEG,
		".png":  MimeTypePNG,
		".gif":  MimeTypeGIF,
		".bmp":  MimeTypeBMP,
		".tif":  MimeTypeTIFF,
		".tiff": MimeTypeTIFF,
		".webp": MimeTypeWEBP,
		".svg":  "image/svg+xml",

		// Audio
		".mp3":  MimeTypeMP3,
		".wav":  MimeTypeWAV,
		".ogg":  MimeTypeOGG,
		".m4a":  MimeTypeM4A,
		".flac": MimeTypeFLAC,

		// Video
		".mp4":  MimeTypeMP4,
		".avi":  MimeTypeAVI,
		".mov":  MimeTypeMOV,
		".wmv":  MimeTypeWMV,
		".flv":  MimeTypeFLV,
		".webm": MimeTypeWEBM,

		// Documents
		".pdf":  MimeTypePDF,
		".doc":  MimeTypeDOC,
		".docx": MimeTypeDOCX,
		".txt":  MimeTypeTXT,
		".rtf":  MimeTypeRTF,

		// Archives
		".zip": MimeTypeZIP,
		".rar": MimeTypeRAR,
		".7z":  MimeType7Z,
		".tar": MimeTypeTAR,
		".gz":  MimeTypeGZIP,
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	return MimeTypeOctetStream
}

// mapFormatToMimeType maps GEDCOM 5.5.1 FORM values to MIME types
func mapFormatToMimeType(format string) string {
	formatLower := strings.ToLower(format)

	mapping := map[string]string{
		// Images
		"jpg":  MimeTypeJPEG,
		"jpeg": MimeTypeJPEG,
		"png":  MimeTypePNG,
		"gif":  MimeTypeGIF,
		"bmp":  MimeTypeBMP,
		"tif":  MimeTypeTIFF,
		"tiff": MimeTypeTIFF,
		"pcx":  MimeTypePCX,

		// Audio
		"wav": MimeTypeWAV,
		"mp3": MimeTypeMP3,

		// Video
		"avi": MimeTypeAVI,
		"mpg": "video/mpeg",
		"mp4": MimeTypeMP4,

		// Documents
		"pdf": MimeTypePDF,
		"txt": MimeTypeTXT,
	}

	if mime, ok := mapping[formatLower]; ok {
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
