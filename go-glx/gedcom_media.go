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

	conv.Logger.LogInfof("Converting OBJE %s -> %s", objeRecord.XRef, mediaID)

	// Convert using common logic
	media := convertMediaCommon(objeRecord, mediaID, conv)

	// Handle SOUR subrecords (only for top-level OBJE records)
	for _, sub := range objeRecord.SubRecords {
		if sub.Tag == GedcomTagSour {
			result, err := createCitationFromSOUR(sub, conv)
			if err != nil {
				// Error already logged in createCitationFromSOUR, skip
				continue
			}
			if result.CitationID != "" {
				if citation, ok := conv.GLX.Citations[result.CitationID]; ok {
					citation.Media = append(citation.Media, mediaID)
				}
			}
			// For bare source references on media, link the media to the source
			// (the source already exists, media can reference it via its own media field)
			if result.SourceID != "" {
				if source, ok := conv.GLX.Sources[result.SourceID]; ok {
					source.Media = append(source.Media, mediaID)
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
func convertEmbeddedMedia(objeRecord *GEDCOMRecord, conv *ConversionContext) string {
	// Generate media ID for embedded object
	mediaID := generateMediaID(conv)

	conv.Logger.LogInfo("Converting embedded OBJE -> " + mediaID)

	// Convert using common logic
	media := convertMediaCommon(objeRecord, mediaID, conv)

	// Store media
	conv.GLX.Media[mediaID] = media
	conv.Stats.MediaCreated++

	return mediaID
}

// resolveOBJE resolves an OBJE subrecord to a media ID. It handles three cases:
//  1. XRef reference (e.g., "1 OBJE @O1@") — looks up the already-converted top-level media
//  2. VOID pointer (GEDCOM 7.0 "1 OBJE @VOID@") with subrecords — creates embedded media
//  3. Embedded OBJE (no value, has subrecords) — creates a new media entity
//
// Returns the media ID, or empty string if resolution failed (warnings are emitted).
func resolveOBJE(objeRecord *GEDCOMRecord, conv *ConversionContext) string {
	isVoid := objeRecord.Value == "@VOID@"
	hasRef := objeRecord.Value != "" && !isVoid

	if hasRef {
		// Case 1: Reference to top-level OBJE (e.g., @O1@)
		mediaID := conv.MediaIDMap[objeRecord.Value]
		if mediaID == "" {
			conv.addWarning(objeRecord.Line, GedcomTagObje, "Referenced media not found: "+objeRecord.Value)
		}

		return mediaID
	}

	// Case 2 & 3: Embedded OBJE or VOID with subrecords
	if len(objeRecord.SubRecords) > 0 {
		return convertEmbeddedMedia(objeRecord, conv)
	}

	return ""
}

// handleOBJE processes an OBJE subrecord and appends the resolved media ID to
// the target properties map under PropertyMedia.
func handleOBJE(objeRecord *GEDCOMRecord, targetProps map[string]any, conv *ConversionContext) {
	if mediaID := resolveOBJE(objeRecord, conv); mediaID != "" {
		appendMediaID(targetProps, mediaID)
	}
}

// appendMediaID appends a media ID to the PropertyMedia list in a properties map.
func appendMediaID(props map[string]any, mediaID string) {
	switch v := props[PropertyMedia].(type) {
	case []string:
		props[PropertyMedia] = append(v, mediaID)
	case []any:
		list := make([]string, 0, len(v)+1)
		for _, item := range v {
			if s, ok := item.(string); ok {
				list = append(list, s)
			}
		}
		props[PropertyMedia] = append(list, mediaID)
	default:
		props[PropertyMedia] = []string{mediaID}
	}
}

// convertMediaCommon contains the shared logic for converting GEDCOM OBJE records to GLX Media entities.
// It processes FILE, FORM, TITL, CROP, NOTE, and BLOB subrecords.
// When a relative FILE path is found, it creates a MediaFileSource entry and rewrites
// the URI to point to media/files/<filename>. BLOB data is also captured for the CLI to write.
//
//nolint:gocognit,gocyclo
func convertMediaCommon(objeRecord *GEDCOMRecord, mediaID string, conv *ConversionContext) *Media {
	media := &Media{
		Properties: make(map[string]any),
	}

	var fileRef string
	var formatType string
	var notes []string
	var blobText string

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

		case GedcomTagBlob:
			// GEDCOM 5.5.1 BLOB data (deprecated binary embedding)
			blobText = extractTextWithContinuation(sub)
			if blobText != "" {
				media.Properties["blob_size"] = len(blobText)
			}
		}
	}

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
		media.MimeType = MimeTypeOctetStream
	}

	// Track file sources for CLI to copy/write
	if fileRef != "" && classifyFileRef(fileRef) {
		// Relative path — track for copying and rewrite URI
		basename := filepath.Base(normalizePathSeparators(fileRef))
		targetName := deduplicateFilename(basename, conv.MediaFileNames)
		conv.MediaFileSources = append(conv.MediaFileSources, MediaFileSource{
			MediaID:        mediaID,
			SourceType:     MediaSourceFile,
			RelativePath:   fileRef,
			TargetFilename: targetName,
		})
		media.URI = MediaFilesDir + "/" + targetName
	} else {
		// URL, absolute path, or empty — leave as-is
		media.URI = fileRef
	}

	// Track BLOB data for CLI to decode and write
	if blobText != "" {
		ext := extensionFromMimeType(media.MimeType)
		targetName := deduplicateFilename("blob-"+mediaID+ext, conv.MediaFileNames)
		conv.MediaFileSources = append(conv.MediaFileSources, MediaFileSource{
			MediaID:        mediaID,
			SourceType:     MediaSourceBlob,
			BlobData:       blobText,
			TargetFilename: targetName,
		})
		// If no FILE ref, set URI to the blob file
		if media.URI == "" {
			media.URI = MediaFilesDir + "/" + targetName
		}
	}

	// Combine notes
	if len(notes) > 0 {
		media.Notes = strings.Join(notes, "\n")
	}

	return media
}

// classifyFileRef determines if a GEDCOM FILE reference is a relative path
// that should be copied into the archive. Returns true for relative paths.
func classifyFileRef(fileRef string) bool {
	if fileRef == "" {
		return false
	}
	// URL schemes: http://, https://, ftp://, mailto:, etc.
	if strings.Contains(fileRef, "://") || strings.HasPrefix(fileRef, "mailto:") {
		return false
	}
	// Absolute Unix path
	if strings.HasPrefix(fileRef, "/") {
		return false
	}
	// Absolute Windows path (e.g., C:\, D:/)
	if len(fileRef) >= 2 && fileRef[1] == ':' {
		return false
	}

	return true
}

// deduplicateFilename returns a unique filename for media/files/.
// If "photo.jpg" is already used, returns "photo-2.jpg", etc.
func deduplicateFilename(basename string, usedNames map[string]int) string {
	if _, exists := usedNames[basename]; !exists {
		usedNames[basename] = 1

		return basename
	}
	usedNames[basename]++
	ext := filepath.Ext(basename)
	name := strings.TrimSuffix(basename, ext)
	deduped := fmt.Sprintf("%s-%d%s", name, usedNames[basename], ext)
	usedNames[deduped] = 1

	return deduped
}

// normalizePathSeparators converts backslashes to forward slashes.
func normalizePathSeparators(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// preferredExtension maps MIME types to their canonical file extension.
// This avoids non-deterministic results from iterating mimeTypeByExtension
// where multiple extensions map to the same MIME type (.jpg/.jpeg, .tif/.tiff).
var preferredExtension = map[string]string{
	MimeTypeJPEG: ".jpg",
	MimeTypeTIFF: ".tif",
}

// extensionFromMimeType returns a file extension (with dot) for a MIME type.
func extensionFromMimeType(mimeType string) string {
	if ext, ok := preferredExtension[mimeType]; ok {
		return ext
	}
	for ext, mime := range mimeTypeByExtension {
		if mime == mimeType {
			return ext
		}
	}

	return ".bin"
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
