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
	"net/url"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// convertMedia converts a GEDCOM OBJE record to a GLX Media entity
func convertMedia(objeRecord *GEDCOMRecord, conv *ConversionContext) error {
	if objeRecord.Tag != GedcomTagObje {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedMediaRecord, GedcomTagObje, objeRecord.Tag)
	}

	mediaIDs := make([]string, 0, 1)
	for _, part := range splitOBJEFiles(objeRecord) {
		mediaID := generateMediaID(conv)
		mediaIDs = append(mediaIDs, mediaID)
		conv.Logger.LogInfof("Converting OBJE %s -> %s", objeRecord.XRef, mediaID)
		media := convertMediaCommon(part, mediaID, conv)

		// Warn about empty URI but preserve the entity — metadata (TITL, NOTE, FORM)
		// may still be valuable. Validation will catch the missing required URI field.
		if media.URI == "" {
			conv.Logger.LogInfof("OBJE %s has no FILE reference or BLOB data — URI will be empty", objeRecord.XRef)
			conv.addWarning(objeRecord.Line, GedcomTagObje,
				fmt.Sprintf("OBJE %s: no FILE reference or BLOB data, URI will be empty", objeRecord.XRef))
		}
		conv.GLX.Media[mediaID] = media
		conv.Stats.MediaCreated++
	}
	conv.MediaIDMap[objeRecord.XRef] = mediaIDs[0]
	if len(mediaIDs) > 1 {
		if conv.MediaFileGroups == nil {
			conv.MediaFileGroups = make(map[string][]string)
		}
		conv.MediaFileGroups[objeRecord.XRef] = mediaIDs
	}

	// Handle SOUR subrecords (only for top-level OBJE records)
	for _, sub := range objeRecord.SubRecords {
		if sub.Tag == GedcomTagSour {
			result, err := prepareCitationFromSOUR(sub, conv)
			if err != nil {
				// Error already logged in prepareCitationFromSOUR, skip
				continue
			}
			if result.CitationID != "" {
				if citation, ok := conv.GLX.Citations[result.CitationID]; ok {
					citation.Media = append(citation.Media, mediaIDs...)
				}
			}
			// Finalize private synthetic drafts only after attaching contextual
			// media, so this record cannot modify another record's shared source.
			if source := result.bareSource(conv); source != nil {
				source.Media = append(source.Media, mediaIDs...)
			}
			result.finalizeSource(conv)
		}
	}

	return nil
}

// splitOBJEFiles returns one OBJE record per FILE structure (#1308). A GEDCOM
// 7.0 OBJE may carry several FILEs, such as a scan and its transcription or
// the same photo at two resolutions, and a GLX Media entity has one uri, so
// each FILE becomes its own Media. Every copy keeps the OBJE's other
// substructures (TITL, NOTE, CROP, _OID and so on) and one FILE with its own
// FORM, MIME and TITL. An OBJE with at most one FILE is returned unchanged.
func splitOBJEFiles(obje *GEDCOMRecord) []*GEDCOMRecord {
	var files, shared []*GEDCOMRecord
	for _, sub := range obje.SubRecords {
		if sub.Tag == GedcomTagFile {
			files = append(files, sub)
		} else {
			shared = append(shared, sub)
		}
	}
	if len(files) <= 1 {
		return []*GEDCOMRecord{obje}
	}
	parts := make([]*GEDCOMRecord, 0, len(files))
	for _, file := range files {
		part := *obje
		part.SubRecords = append(slices.Clone(shared), file)
		parts = append(parts, &part)
	}

	return parts
}

// convertEmbeddedMedia converts an embedded OBJE (without XRef) and returns
// the IDs of its Media entities, one per FILE (#1308).
func convertEmbeddedMedia(objeRecord *GEDCOMRecord, conv *ConversionContext) []string {
	mediaIDs := make([]string, 0, 1)
	for _, part := range splitOBJEFiles(objeRecord) {
		mediaID := generateMediaID(conv)
		mediaIDs = append(mediaIDs, mediaID)
		conv.Logger.LogInfo("Converting embedded OBJE -> " + mediaID)
		media := convertMediaCommon(part, mediaID, conv)

		// Warn about empty URI but preserve the entity and its metadata.
		if media.URI == "" {
			conv.Logger.LogInfo("Embedded OBJE has no FILE reference or BLOB data — URI will be empty")
			conv.addWarning(objeRecord.Line, GedcomTagObje,
				"embedded OBJE: no FILE reference or BLOB data, URI will be empty")
		}
		conv.GLX.Media[mediaID] = media
		conv.Stats.MediaCreated++
	}

	return mediaIDs
}

// resolveOBJE resolves an OBJE subrecord to a media ID. It handles three cases:
//  1. XRef reference (e.g., "1 OBJE @O1@") — looks up the already-converted top-level media
//  2. VOID pointer (GEDCOM 7.0 "1 OBJE @VOID@") with subrecords — creates embedded media
//  3. Embedded OBJE (no value, has subrecords) — creates a new media entity
//
// Returns the media IDs, one per FILE of the object (#1308), or none if
// resolution failed (warnings are emitted).
func resolveOBJE(objeRecord *GEDCOMRecord, conv *ConversionContext) []string {
	isVoid := objeRecord.Value == "@VOID@"
	hasRef := objeRecord.Value != "" && !isVoid

	if hasRef {
		// Case 1: Reference to top-level OBJE (e.g., @O1@)
		if group := conv.MediaFileGroups[objeRecord.Value]; len(group) > 0 {
			return group
		}
		mediaID := conv.MediaIDMap[objeRecord.Value]
		if mediaID == "" {
			conv.addWarning(objeRecord.Line, GedcomTagObje, "Referenced media not found: "+objeRecord.Value)

			return nil
		}

		return []string{mediaID}
	}

	// Case 2 & 3: Embedded OBJE or VOID with subrecords
	if len(objeRecord.SubRecords) > 0 {
		return convertEmbeddedMedia(objeRecord, conv)
	}

	return nil
}

// handleOBJE processes an OBJE subrecord and appends the resolved media ID to
// the target properties map under PropertyMedia.
func handleOBJE(objeRecord *GEDCOMRecord, targetProps map[string]any, conv *ConversionContext) {
	for _, mediaID := range resolveOBJE(objeRecord, conv) {
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
// When a relative FILE path is found, it creates a MediaFileSource entry, rewrites
// the URI to point to media/files/<filename>, and records the source basename in the
// original_filename property. BLOB data is also captured for the CLI to write.
//
//nolint:gocognit,gocyclo // GEDCOM conversion has inherent branching complexity
func convertMediaCommon(objeRecord *GEDCOMRecord, mediaID string, conv *ConversionContext) *Media {
	media := &Media{
		Properties: make(map[string]any),
	}

	var fileRef string
	var fileOriginal string // FILE._ORIG, written by glx export (#1149)
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
				case GedcomTagOrig:
					fileOriginal = fileSub.Value
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

		case GedcomTagExid:
			if sub.Value != "" {
				appendMultiValueProperty(media.Properties, MediaPropertyExternalIDs, buildExternalIDEntry(sub))
			}

		case GedcomTagOID:
			// Ancestry's stable media object id: the one exact link back to
			// the platform, since Ancestry exports FILE empty (#944).
			if sub.Value != "" {
				appendMultiValueProperty(media.Properties, MediaPropertyExternalIDs, NewExternalIDEntry(sub.Value, ExternalIDTypeAncestry))
			}

		case GedcomTagClon:
			for _, clon := range sub.SubRecords {
				if clon.Tag == GedcomTagOID && clon.Value != "" {
					appendMultiValueProperty(media.Properties, MediaPropertyExternalIDs, NewExternalIDEntry(clon.Value, ExternalIDTypeAncestryClone))
				}
			}

		case GedcomTagBlob:
			// GEDCOM 5.5.1 BLOB data (deprecated binary embedding)
			blobText = extractTextWithContinuation(sub)
			if blobText != "" {
				media.Properties[MediaPropertyBlobSize] = len(blobText)
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

	// A FILE._ORIG written by glx export is the original filename the
	// archive recorded, and wins over any name this import would infer (#1149).
	if fileOriginal != "" {
		media.Properties[MediaPropertyOriginalFilename] = fileOriginal
	}

	// Track file sources for CLI to copy/write
	if fileRef != "" && classifyFileRef(fileRef) {
		// Relative path — track for copying and rewrite URI
		normalizedRef := normalizePathSeparators(fileRef)
		basename := filepath.Base(normalizedRef)
		// A GEDCOM 7.0 FILE payload is a URI reference, so the file it names
		// is the decoded one: the archive file, uri and original_filename all
		// use the decoded name (#1152). 5.5.1 paths keep a literal %.
		originalFilename := originalFilenameFor(basename, conv.Version)
		targetName := conv.MediaFileNames.Allocate(originalFilename)
		source := MediaFileSource{
			MediaID:           mediaID,
			SourceType:        MediaSourceFile,
			RelativePath:      normalizedRef,
			OriginalFilename:  originalFilename,
			RequestedFilename: originalFilename,
			TargetFilename:    targetName,
		}
		source.SetMediaURI(media, targetName)
		// Record the pre-rename basename: a collision rename (photo.jpg →
		// photo-2.jpg) puts only the renamed form in the URI, which would
		// otherwise lose the source filename entirely. A ref already pointing
		// into media/files/ came from a GLX export, so while its name is
		// untouched there is no "original" to record and stamping would
		// fabricate provenance on a GLX→GEDCOM→GLX round trip — but once this
		// import renames it (merging two exports can collide two canonical
		// refs), the pre-rename basename is real information and is kept. The
		// existence guard is defensive: no current tag maps to this key, but
		// a future mapping would write properties before this branch runs and
		// must win.
		if !isCanonicalMediaRef(normalizedRef) {
			source.RecordOriginalFilename(media)
		}
		// Append after stamping, so the CLI's copy knows the stamp is this
		// import's guess and may correct it (ResolveOriginalFilename).
		conv.MediaFileSources = append(conv.MediaFileSources, source)
	} else {
		// URL, absolute path, or empty — leave as-is
		media.URI = fileRef
	}

	// Track BLOB data for CLI to decode and write
	if blobText != "" {
		ext := extensionFromMimeType(media.MimeType)
		targetName := conv.MediaFileNames.Allocate("blob-" + mediaID + ext)
		conv.MediaFileSources = append(conv.MediaFileSources, MediaFileSource{
			MediaID:           mediaID,
			SourceType:        MediaSourceBlob,
			BlobData:          blobText,
			RequestedFilename: "blob-" + mediaID + ext,
			TargetFilename:    targetName,
		})
		// If no FILE ref, set URI to the blob file
		if media.URI == "" {
			media.URI = MediaFilesDir + "/" + targetName
		}
	}

	// Collect notes
	if len(notes) > 0 {
		media.Notes = NoteList(notes)
	}

	return media
}

// originalFilenameFor returns the source filename for a relative FILE ref's
// basename, which is also the name the file is given in the archive. GEDCOM
// 7.0 FILE payloads are URI references, so a percent-encoded basename names a
// file that exists only decoded (CharlotteBront%C3%AB.jpg → CharlotteBrontë.jpg);
// 5.5.1 payloads are plain paths where a literal % must survive untouched. A
// 7.0 encoding that is malformed, or whose decoded form is not a bare filename
// per isBareDecodedFilename, keeps the raw basename. This is the importer's
// best reading of the reference: a source file found under the other spelling
// corrects it once resolved (ResolveOriginalFilename, #1153).
func originalFilenameFor(basename string, version GEDCOMVersion) string {
	if version != GEDCOM70 {
		return basename
	}
	decoded, err := url.PathUnescape(basename)
	if err != nil || !isBareDecodedFilename(decoded) {
		return basename
	}

	return decoded
}

// isCanonicalMediaRef reports whether a separator-normalized FILE ref already
// points into the archive's media/files/ directory — the shape a GLX export
// produces. path.Clean catches spellings like ./media/files/x that name the
// same directory, and the prefix is compared case-insensitively so
// case-insensitive filesystems (where Media/Files/ is the same directory)
// don't defeat the check.
func isCanonicalMediaRef(normalizedRef string) bool {
	cleaned := path.Clean(normalizedRef)
	prefix := MediaFilesDir + "/"
	if len(cleaned) <= len(prefix) {
		return false
	}

	return strings.EqualFold(cleaned[:len(prefix)], prefix)
}

// isBareDecodedFilename reports whether a percent-decoded basename is still a
// bare filename: valid UTF-8, not a dot path (".", ".."), and containing no
// rune isDisallowedFilenameRune rejects. Percent-decoding untrusted GEDCOM
// 7.0 refs is the only way any of these can appear — filepath.Base has
// already run on the encoded form.
func isBareDecodedFilename(decoded string) bool {
	if decoded == "" || decoded == "." || decoded == ".." {
		return false
	}
	if !utf8.ValidString(decoded) {
		return false
	}

	return !strings.ContainsFunc(decoded, isDisallowedFilenameRune)
}

// isDisallowedFilenameRune rejects runes that make a decoded basename unsafe
// or misleading as a filename: path separators; Unicode control characters
// (C0, DEL, or C1 — a decoded %C2%80 is valid UTF-8 but still a control);
// the U+2028/U+2029 line and paragraph separators (YAML emitters treat them
// as line breaks); invisible characters (zero-width space U+200B, BOM
// U+FEFF); and the bidi embedding/override/isolate (U+202A-U+202E,
// U+2066-U+2069) and directional-mark (U+200E LRM, U+200F RLM, U+061C ALM)
// characters, which can render a filename in a misleading order (a U+202E
// override displays photo<RLO>gpj.exe as photoexe.jpg). Format characters
// that are ordinary orthography are deliberately allowed: ZWNJ (standard in
// Persian/Arabic spelling) and ZWJ (emoji sequences, several Indic scripts)
// join glyphs but never reorder or hide text.
func isDisallowedFilenameRune(r rune) bool {
	switch {
	case r == '/' || r == '\\':
		return true
	case unicode.IsControl(r):
		return true
	case r == '\u2028' || r == '\u2029':
		return true
	case r == '\u200b' || r == '\ufeff':
		return true
	case r >= '\u202a' && r <= '\u202e', r >= '\u2066' && r <= '\u2069':
		return true
	case r == '\u200e' || r == '\u200f' || r == '\u061c':
		return true
	}

	return false
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
	// UNC path (e.g., \\server\share\photo.jpg)
	if strings.HasPrefix(fileRef, `\\`) {
		return false
	}

	return true
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
