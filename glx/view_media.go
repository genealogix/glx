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

package main

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// mediaDirName is the site subdirectory copied media files are written to.
const mediaDirName = "media"

// imageExtensions lists file extensions rendered inline as images.
var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".svg": true, ".bmp": true, ".tif": true, ".tiff": true,
}

// buildPersonMedia gathers and classifies the media attached to a person via
// the assertions (and their citations) that reference them. This is pure — it
// resolves references and classifies items but performs no I/O. Concrete
// sources are filled in later by resolveSiteMedia.
func buildPersonMedia(personID string, archive *glxlib.GLXFile, idx *viewIndex) []*mediaItem {
	var items []*mediaItem
	seen := map[string]bool{}

	addMedia := func(mediaID string) {
		if mediaID == "" || seen[mediaID] {
			return
		}
		media, ok := archive.Media[mediaID]
		if !ok || media == nil {
			return
		}
		seen[mediaID] = true
		items = append(items, newMediaItem(mediaID, media))
	}

	for _, a := range idx.assertionsByPerson[personID] {
		for _, mediaID := range a.Media {
			addMedia(mediaID)
		}
		for _, citID := range a.Citations {
			if cit, ok := archive.Citations[citID]; ok && cit != nil {
				for _, mediaID := range cit.Media {
					addMedia(mediaID)
				}
			}
		}
	}

	return items
}

// newMediaItem classifies a media object into a presentation item.
func newMediaItem(mediaID string, media *glxlib.Media) *mediaItem {
	caption := propertyScalar(media.Properties["description"])
	if caption == "" {
		caption = media.Title
	}

	item := &mediaItem{
		MediaID:  mediaID,
		RawURI:   media.URI,
		Title:    media.Title,
		Caption:  caption,
		MimeType: media.MimeType,
		IsURL:    isHTTPURL(media.URI),
	}
	item.IsImage = isImageMedia(media.MimeType, media.URI)

	return item
}

// isImageMedia reports whether a media item should render inline as an <img>.
// It returns true when EITHER the MIME type is image/* OR the URI extension is
// a known image type. Erring toward <img> is a security choice: an item that
// is NOT classified as an image renders as a clickable link, and a browser
// (or http.FileServer) may serve that file as an active document — notably an
// SVG with a non-image MIME, whose embedded scripts would then run in the
// site's origin. Rendering via <img> instead disables SVG scripting entirely.
func isImageMedia(mimeType, uri string) bool {
	if strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return true
	}

	return imageExtensions[strings.ToLower(filepath.Ext(uri))]
}

// isHTTPURL reports whether a URI is an absolute http(s) URL.
func isHTTPURL(uri string) bool {
	lower := strings.ToLower(uri)

	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// mediaResolver copies or embeds referenced media and fills each item's Src.
// Because media only ever renders on person profile pages (which live one
// directory deep), copied-file sources are written with the "../" prefix
// baked in so templates can use Src directly.
type mediaResolver struct {
	baseDir   string                 // directory media URIs are resolved relative to
	outputDir string                 // site output root
	embed     bool                   // base64-embed images instead of copying
	mediaDir  string                 // <outputDir>/media, created lazily
	assigned  map[string]string      // absolute source path -> output basename
	usedNames map[string]bool        // output basenames already taken
	resolved  map[string]mediaResult // source path -> already-resolved result
	copied    int
}

// mediaResult caches the outcome of resolving one source file so a file shared
// by several persons is read and base64-encoded (or copied) only once.
type mediaResult struct {
	src     string
	missing bool
}

// resolveSiteMedia resolves every media reference in the model, copying or
// embedding files as configured. It returns the number of files copied.
func resolveSiteMedia(model *siteModel, baseDir, outputDir string, embed bool) (int, error) {
	r := &mediaResolver{
		baseDir:   baseDir,
		outputDir: outputDir,
		embed:     embed,
		mediaDir:  filepath.Join(outputDir, mediaDirName),
		assigned:  map[string]string{},
		usedNames: map[string]bool{},
		resolved:  map[string]mediaResult{},
	}

	for _, page := range model.Persons {
		for _, item := range page.Media {
			if err := r.resolve(item); err != nil {
				return r.copied, err
			}
		}
	}

	return r.copied, nil
}

// resolve fills in a single media item's Src (and Missing flag).
func (r *mediaResolver) resolve(item *mediaItem) error {
	// External URLs are referenced directly — never copied or embedded.
	if item.IsURL {
		item.Src = item.RawURI

		return nil
	}

	srcPath, ok := r.sourcePath(item.RawURI)
	if !ok {
		// The URI escapes the archive directory (an absolute path or ../
		// traversal). Refuse to read arbitrary local files into the generated
		// site — treat it as missing media. This is the trust boundary: only
		// files under the archive directory are ever read.
		item.Missing = true

		return nil
	}

	// A file shared by multiple persons is read/encoded/copied only once.
	if cached, hit := r.resolved[srcPath]; hit {
		item.Src, item.Missing = cached.src, cached.missing

		return nil
	}

	result, err := r.resolveFile(item, srcPath)
	if err != nil {
		return err
	}
	r.resolved[srcPath] = result
	item.Src, item.Missing = result.src, result.missing

	return nil
}

// resolveFile reads a confined source file and produces its resolved result
// (an embedded data URI, a copied-file reference, or a missing marker).
func (r *mediaResolver) resolveFile(item *mediaItem, srcPath string) (mediaResult, error) {
	// srcPath is confined to baseDir (checked by sourcePath); a missing or
	// unreadable file is non-fatal and recorded so the template shows a
	// placeholder.
	data, err := os.ReadFile(srcPath) // #nosec G304 -- confined to baseDir by sourcePath
	if err != nil {
		return mediaResult{missing: true}, nil //nolint:nilerr // missing media is non-fatal
	}

	if r.embed && item.IsImage {
		return mediaResult{src: dataURI(item.MimeType, item.RawURI, data)}, nil
	}

	name, err := r.copyMedia(srcPath, item.RawURI, data)
	if err != nil {
		return mediaResult{}, err
	}

	// Person pages are one directory deep, so reference media via "../media/".
	return mediaResult{src: "../" + mediaDirName + "/" + name}, nil
}

// copyMedia writes a media file into <outputDir>/media and returns its basename.
func (r *mediaResolver) copyMedia(srcPath, uri string, data []byte) (string, error) {
	name := r.outputName(srcPath, uri)
	if err := os.MkdirAll(r.mediaDir, dirPermissions); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(r.mediaDir, name), data, filePermissions); err != nil {
		return "", fmt.Errorf("failed to write media file %q: %w", name, err)
	}
	r.copied++

	return name, nil
}

// sourcePath resolves a media URI to a filesystem path confined to baseDir,
// returning ok=false for URIs that escape it — absolute paths or ../ traversal.
// filepath.Join neutralizes a leading "/" (the URI is always treated as
// relative to the archive), and isPathWithin rejects any ../ that climbs out.
// Mirrors copyMediaFile's guard in media_copy.go so the viewer never reads
// files outside the archive directory, even from an untrusted archive.
func (r *mediaResolver) sourcePath(uri string) (string, bool) {
	normalized := strings.ReplaceAll(uri, "\\", "/")
	srcPath := filepath.Join(r.baseDir, normalized)
	if !isPathWithin(srcPath, r.baseDir) {
		return "", false
	}

	return srcPath, true
}

// outputName returns a collision-free output basename for a source file,
// reusing the same name when the same source path is referenced twice.
func (r *mediaResolver) outputName(srcPath, uri string) string {
	if name, ok := r.assigned[srcPath]; ok {
		return name
	}

	base := safeMediaBase(filepath.Base(filepath.FromSlash(uri)))
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	name := uniqueFileName(stem, ext, r.usedNames)
	r.assigned[srcPath] = name

	return name
}

// dataURI builds a base64 data URI for embedded image content.
func dataURI(mimeType, uri string, data []byte) string {
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(uri))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// reservedFileStems are Windows device names that are invalid as filenames even
// with an extension (mirrors go-glx's windowsReservedNames). A sanitized name
// whose stem is one of these falls back to a safe default.
var reservedFileStems = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true, "com5": true,
	"com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true, "lpt5": true,
	"lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// sanitizeFileName reduces an arbitrary basename to a safe, lowercase filename,
// keeping dots so file extensions survive. Characters outside [a-z0-9-_.] become
// '-'. Returns the trimmed result (may be empty or dot-only; callers normalize).
func sanitizeFileName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}

	return strings.Trim(b.String(), "-")
}

// safeMediaBase returns a filesystem-safe media basename (extension preserved),
// falling back to "media" for empty, dot-only, leading-dot (hidden), or
// reserved-device-name results so a crafted URI cannot produce a dotfile or an
// invalid filename.
func safeMediaBase(base string) string {
	clean := strings.TrimLeft(sanitizeFileName(base), ".")
	if clean == "" || strings.Trim(clean, ".") == "" {
		return mediaDirName
	}
	ext := filepath.Ext(clean)
	stem := strings.TrimSuffix(clean, ext)
	if stem == "" || reservedFileStems[stem] {
		return mediaDirName + ext
	}

	return clean
}

// safePersonStem returns a collision-resistant, dot-free filename stem for a
// person ID, falling back to "person" for empty or reserved results. Dots are
// collapsed to '-' because a person ID carries no meaningful file extension.
func safePersonStem(personID string) string {
	clean := strings.Trim(strings.ReplaceAll(sanitizeFileName(personID), ".", "-"), "-_")
	if clean == "" || reservedFileStems[clean] {
		return searchKindPerson
	}

	return clean
}

// uniqueFileName returns stem+ext, appending -2, -3, … until it is unused, and
// records the chosen name in used.
func uniqueFileName(stem, ext string, used map[string]bool) string {
	name := stem + ext
	for i := 2; used[name]; i++ {
		name = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
	used[name] = true

	return name
}

// assignPersonFiles maps each person ID (given in sorted order for determinism)
// to a unique, collision-safe HTML filename. Distinct IDs that sanitize to the
// same stem — including case-only and punctuation-only differences the viewer's
// loader does not reject — are disambiguated with a numeric suffix so a page is
// never silently overwritten and links never resolve to the wrong person.
func assignPersonFiles(sortedIDs []string) map[string]string {
	used := map[string]bool{}
	files := make(map[string]string, len(sortedIDs))
	for _, id := range sortedIDs {
		files[id] = uniqueFileName(safePersonStem(id), ".html", used)
	}

	return files
}
