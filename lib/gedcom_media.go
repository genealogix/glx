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
func convertMedia(objeRecord *GEDCOMRecord, ctx *ConversionContext) error {
	// Panic recovery
	defer func() {
		if r := recover() {
			ctx.Logger.LogException(objeRecord.Line, "OBJE", objeRecord.XRef, "convertMedia",
				fmt.Errorf("panic: %v", r), map[string]interface{}{
					"record": objeRecord,
				})
		}
	}()

	if objeRecord.Tag != "OBJE" {
		return fmt.Errorf("expected OBJE record, got %s", objeRecord.Tag)
	}

	// Generate media ID
	mediaID := generateMediaID(ctx)
	ctx.MediaIDMap[objeRecord.XRef] = mediaID

	ctx.Logger.LogInfo(fmt.Sprintf("Converting OBJE %s -> %s", objeRecord.XRef, mediaID))

	// Create media entity
	media := &Media{
		Properties: make(map[string]interface{}),
	}

	var fileRef string
	var formatType string

	// Process subrecords
	for _, sub := range objeRecord.SubRecords {
		switch sub.Tag {
		case "FILE":
			// File reference - primary in both 5.5.1 and 7.0
			fileRef = sub.Value

			// Check for GEDCOM 7.0 MIME type as subrecord
			for _, fileSub := range sub.SubRecords {
				switch fileSub.Tag {
				case "FORM":
					// GEDCOM 5.5.1: FORM under FILE
					formatType = fileSub.Value
				case "MIME":
					// GEDCOM 7.0: MIME type
					media.Type = fileSub.Value
				case "TITL":
					// Title under FILE
					media.Title = fileSub.Value
				}
			}

		case "FORM":
			// GEDCOM 5.5.1: Format at OBJE level
			formatType = sub.Value

			// Check for MEDI subrecord
			for _, formSub := range sub.SubRecords {
				if formSub.Tag == "MEDI" {
					media.Properties["medium"] = formSub.Value
				}
			}

		case "TITL":
			// Title (can be at OBJE level or FILE level)
			if media.Title == "" {
				media.Title = sub.Value
			}

		case "CROP":
			// GEDCOM 7.0: Crop coordinates
			crop := extractCrop(sub)
			if crop != nil {
				media.Properties["crop"] = crop
			}

		case "NOTE":
			// Notes/description
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				media.Description = noteText
			}

		case "SOUR":
			// Source citations for the media
			citationID, err := createCitationFromSOUR(mediaID, sub, ctx)
			if err == nil && citationID != "" {
				citations, ok := media.Properties["citations"].([]string)
				if !ok {
					citations = []string{}
				}
				media.Properties["citations"] = append(citations, citationID)
			}
		}
	}

	// Set file reference
	media.File = fileRef

	// Infer MIME type if not set (GEDCOM 5.5.1)
	if media.Type == "" {
		if formatType != "" {
			media.Type = mapFormatToMimeType(formatType)
		} else if fileRef != "" {
			media.Type = inferMimeType(fileRef)
		}
	}

	// Default type if still not set
	if media.Type == "" {
		media.Type = "application/octet-stream"
	}

	// Store media
	ctx.GLX.Media[mediaID] = media
	ctx.Stats.MediaCreated++

	return nil
}

// convertEmbeddedMedia converts an embedded OBJE (without XRef) and returns the media ID
func convertEmbeddedMedia(objeRecord *GEDCOMRecord, ctx *ConversionContext) (string, error) {
	// Generate media ID for embedded object
	mediaID := generateMediaID(ctx)

	ctx.Logger.LogInfo(fmt.Sprintf("Converting embedded OBJE -> %s", mediaID))

	// Create media entity
	media := &Media{
		Properties: make(map[string]interface{}),
	}

	var fileRef string
	var formatType string

	// Process subrecords (same as top-level OBJE)
	for _, sub := range objeRecord.SubRecords {
		switch sub.Tag {
		case "FILE":
			fileRef = sub.Value

			for _, fileSub := range sub.SubRecords {
				switch fileSub.Tag {
				case "FORM":
					formatType = fileSub.Value
				case "MIME":
					media.Type = fileSub.Value
				case "TITL":
					media.Title = fileSub.Value
				}
			}

		case "FORM":
			formatType = sub.Value

			for _, formSub := range sub.SubRecords {
				if formSub.Tag == "MEDI" {
					media.Properties["medium"] = formSub.Value
				}
			}

		case "TITL":
			if media.Title == "" {
				media.Title = sub.Value
			}

		case "CROP":
			crop := extractCrop(sub)
			if crop != nil {
				media.Properties["crop"] = crop
			}

		case "NOTE":
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				media.Description = noteText
			}
		}
	}

	media.File = fileRef

	// Infer MIME type
	if media.Type == "" {
		if formatType != "" {
			media.Type = mapFormatToMimeType(formatType)
		} else if fileRef != "" {
			media.Type = inferMimeType(fileRef)
		}
	}

	if media.Type == "" {
		media.Type = "application/octet-stream"
	}

	// Store media
	ctx.GLX.Media[mediaID] = media
	ctx.Stats.MediaCreated++

	return mediaID, nil
}

// inferMimeType infers MIME type from file extension
func inferMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	mimeTypes := map[string]string{
		// Images
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".tif":  "image/tiff",
		".tiff": "image/tiff",
		".webp": "image/webp",
		".svg":  "image/svg+xml",

		// Audio
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".ogg":  "audio/ogg",
		".m4a":  "audio/mp4",
		".flac": "audio/flac",

		// Video
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".webm": "video/webm",

		// Documents
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".txt":  "text/plain",
		".rtf":  "application/rtf",

		// Archives
		".zip": "application/zip",
		".rar": "application/x-rar-compressed",
		".7z":  "application/x-7z-compressed",
		".tar": "application/x-tar",
		".gz":  "application/gzip",
	}

	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}

	return "application/octet-stream"
}

// mapFormatToMimeType maps GEDCOM 5.5.1 FORM values to MIME types
func mapFormatToMimeType(format string) string {
	formatLower := strings.ToLower(format)

	mapping := map[string]string{
		// Images
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"bmp":  "image/bmp",
		"tif":  "image/tiff",
		"tiff": "image/tiff",
		"pcx":  "image/x-pcx",

		// Audio
		"wav": "audio/wav",
		"mp3": "audio/mpeg",

		// Video
		"avi": "video/x-msvideo",
		"mpg": "video/mpeg",
		"mp4": "video/mp4",

		// Documents
		"pdf": "application/pdf",
		"txt": "text/plain",
	}

	if mime, ok := mapping[formatLower]; ok {
		return mime
	}

	return "application/octet-stream"
}

// extractCrop extracts GEDCOM 7.0 crop coordinates
func extractCrop(cropRecord *GEDCOMRecord) map[string]interface{} {
	crop := make(map[string]interface{})

	for _, sub := range cropRecord.SubRecords {
		switch sub.Tag {
		case "TOP":
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop["top"] = val
			}
		case "LEFT":
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop["left"] = val
			}
		case "HEIGHT":
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop["height"] = val
			}
		case "WIDTH":
			if val, err := strconv.Atoi(sub.Value); err == nil {
				crop["width"] = val
			}
		}
	}

	if len(crop) == 0 {
		return nil
	}

	return crop
}
