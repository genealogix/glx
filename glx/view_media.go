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

// isImageMedia reports whether a media item should render inline as an image,
// preferring an explicit MIME type and falling back to the URI extension.
func isImageMedia(mimeType, uri string) bool {
	if mimeType != "" {
		return strings.HasPrefix(strings.ToLower(mimeType), "image/")
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
	baseDir   string            // directory media URIs are resolved relative to
	outputDir string            // site output root
	embed     bool              // base64-embed images instead of copying
	mediaDir  string            // <outputDir>/media, created lazily
	assigned  map[string]string // absolute source path -> output basename
	usedNames map[string]bool   // output basenames already taken
	copied    int
}

// resolveSiteMedia resolves every media reference in the model, copying or
// embedding files as configured. It returns the number of files copied.
func resolveSiteMedia(model *siteModel, baseDir, outputDir string, embed bool) (int, error) {
	r := &mediaResolver{
		baseDir:   baseDir,
		outputDir: outputDir,
		embed:     embed,
		mediaDir:  filepath.Join(outputDir, "media"),
		assigned:  map[string]string{},
		usedNames: map[string]bool{},
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

	srcPath := r.sourcePath(item.RawURI)
	// srcPath is built from the archive's own media URIs; a missing or
	// unreadable file is recorded as non-fatal just below.
	data, err := os.ReadFile(srcPath) // #nosec G304
	if err != nil {
		// A missing or unreadable media file is non-fatal: record it so the
		// template can show a placeholder rather than aborting the whole site.
		item.Missing = true

		return nil //nolint:nilerr // missing media is intentionally non-fatal
	}

	if r.embed && item.IsImage {
		item.Src = dataURI(item.MimeType, item.RawURI, data)

		return nil
	}

	return r.copyMedia(item, srcPath, data)
}

// copyMedia writes a media file into <outputDir>/media and sets its Src.
func (r *mediaResolver) copyMedia(item *mediaItem, srcPath string, data []byte) error {
	name := r.outputName(srcPath, item.RawURI)
	if err := os.MkdirAll(r.mediaDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create media directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(r.mediaDir, name), data, filePermissions); err != nil {
		return fmt.Errorf("failed to write media file %q: %w", name, err)
	}
	r.copied++
	// Person pages are one directory deep, so reference media via "../media/".
	item.Src = "../media/" + name

	return nil
}

// sourcePath resolves a media URI to an absolute filesystem path.
func (r *mediaResolver) sourcePath(uri string) string {
	if filepath.IsAbs(uri) {
		return filepath.Clean(uri)
	}

	return filepath.Join(r.baseDir, filepath.FromSlash(uri))
}

// outputName returns a collision-free output basename for a source file,
// reusing the same name when the same source path is referenced twice.
func (r *mediaResolver) outputName(srcPath, uri string) string {
	if name, ok := r.assigned[srcPath]; ok {
		return name
	}

	base := sanitizeFileName(filepath.Base(filepath.FromSlash(uri)))
	if base == "" {
		base = "media"
	}
	name := base
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 2; r.usedNames[name]; i++ {
		name = fmt.Sprintf("%s-%d%s", stem, i, ext)
	}
	r.usedNames[name] = true
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

// sanitizeFileName reduces an arbitrary basename to a safe, lowercase filename.
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

// personFileName returns the HTML filename for a person page.
func personFileName(personID string) string {
	name := sanitizeFileName(personID)
	if name == "" {
		name = searchKindPerson
	}

	return name + ".html"
}
