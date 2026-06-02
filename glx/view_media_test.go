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
	"os"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// personMediaItem returns the first media item on the given person's page.
func personMediaItem(t *testing.T, model *siteModel, personID string) *mediaItem {
	t.Helper()
	for _, p := range model.Persons {
		if p.ID == personID {
			if len(p.Media) == 0 {
				t.Fatalf("person %q has no media", personID)
			}

			return p.Media[0]
		}
	}
	t.Fatalf("person %q not found", personID)

	return nil
}

func TestDataURI(t *testing.T) {
	if got := dataURI("image/png", "x.png", []byte("data")); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("explicit mime: %q", got)
	}
	if got := dataURI("", "photo.png", []byte("data")); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Errorf("mime from extension: %q", got)
	}
	if got := dataURI("", "noext", []byte("data")); !strings.HasPrefix(got, "data:application/octet-stream;base64,") {
		t.Errorf("octet-stream fallback: %q", got)
	}
}

func TestIsImageMedia(t *testing.T) {
	cases := []struct {
		mime, uri string
		want      bool
	}{
		{"image/jpeg", "weird.bin", true},        // image mime -> image
		{"application/pdf", "report.pdf", false}, // neither image
		{"", "scan.PNG", true},                   // extension, case-insensitive
		{"", "doc.pdf", false},
		{"", "noext", false},
		// Security: an SVG with a non-image MIME must still classify as an
		// image so it renders via <img> (scripting disabled) rather than as a
		// clickable link a browser would open as an active document.
		{"application/xml", "drawing.svg", true},
		{"text/html", "evil.svg", true},
	}
	for _, tc := range cases {
		if got := isImageMedia(tc.mime, tc.uri); got != tc.want {
			t.Errorf("isImageMedia(%q,%q) = %v, want %v", tc.mime, tc.uri, got, tc.want)
		}
	}
}

func TestSourcePath(t *testing.T) {
	base := t.TempDir()
	r := &mediaResolver{baseDir: base}

	// A legitimate relative path resolves within the archive directory.
	got, ok := r.sourcePath("files/photo.jpg")
	if !ok {
		t.Fatal("legitimate relative path was rejected")
	}
	if want := filepath.Join(base, "files", "photo.jpg"); got != want {
		t.Errorf("sourcePath = %q, want %q", got, want)
	}

	// ../ traversal that climbs out of baseDir is rejected.
	if _, ok := r.sourcePath("../../../etc/passwd"); ok {
		t.Error("expected ../ traversal to be rejected")
	}

	// An absolute URI is confined under baseDir (never read from the root),
	// so it never resolves to the original absolute location.
	abs := filepath.Join(t.TempDir(), "secret.txt")
	if got, ok := r.sourcePath(abs); ok && got == abs {
		t.Errorf("absolute URI resolved to its raw location %q (escaped baseDir)", got)
	}
}

func TestResolveSiteMedia_RejectsTraversal(t *testing.T) {
	base := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	writeFile(t, secret, []byte("TOP SECRET"))

	rel, err := filepath.Rel(base, secret)
	if err != nil {
		t.Fatalf("rel: %v", err)
	}

	// Both a ../ traversal URI and an absolute URI must be refused so the
	// secret outside the archive is never read into the generated site.
	for _, uri := range []string{rel, secret} {
		archive := &glxlib.GLXFile{
			Persons: map[string]*glxlib.Person{
				"person-x": {Properties: map[string]any{"name": "X"}},
			},
			Media: map[string]*glxlib.Media{
				"media-evil": {URI: uri, MimeType: "text/plain"},
			},
			Assertions: map[string]*glxlib.Assertion{
				"assertion-x": {Subject: glxlib.EntityRef{Person: "person-x"}, Media: []string{"media-evil"}},
			},
		}
		model := buildSiteModel(archive, viewModelOptions{})
		out := t.TempDir()
		copied, err := resolveSiteMedia(model, base, out, false)
		if err != nil {
			t.Fatalf("uri %q: resolveSiteMedia: %v", uri, err)
		}
		if copied != 0 {
			t.Errorf("uri %q: expected 0 files copied, got %d", uri, copied)
		}

		item := personMediaItem(t, model, "person-x")
		if !item.Missing {
			t.Errorf("uri %q: expected Missing=true (escape blocked), got Src=%q", uri, item.Src)
		}
		if entries, _ := os.ReadDir(filepath.Join(out, "media")); len(entries) > 0 {
			t.Errorf("uri %q: media dir must stay empty, found %d entries", uri, len(entries))
		}
	}
}

func TestSafePersonStem(t *testing.T) {
	cases := map[string]string{
		"person-john-smith": "person-john-smith",
		"Weird ID!":         "weird-id",
		"---":               "person", // sanitizes to empty -> fallback
		"...":               "person", // dot-only -> fallback
		"con":               "person", // reserved device name -> fallback
		"世界":                "person", // all non-ASCII -> fallback
	}
	for in, want := range cases {
		if got := safePersonStem(in); got != want {
			t.Errorf("safePersonStem(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAssignPersonFiles_Collisions(t *testing.T) {
	// Distinct IDs that collapse to the same stem must get distinct filenames.
	ids := []string{"Person-A", "person-a", "person/a", "世界", "界世"}
	files := assignPersonFiles(ids)

	seen := map[string]bool{}
	for _, id := range ids {
		f := files[id]
		if f == "" {
			t.Errorf("no file assigned for %q", id)
		}
		if seen[f] {
			t.Errorf("filename %q assigned to more than one person", f)
		}
		seen[f] = true
		if !strings.HasSuffix(f, ".html") {
			t.Errorf("filename %q missing .html suffix", f)
		}
	}
	if len(seen) != len(ids) {
		t.Errorf("got %d unique filenames for %d ids", len(seen), len(ids))
	}
}

func TestSafeMediaBase(t *testing.T) {
	cases := map[string]string{
		"photo.jpg":  "photo.jpg",
		"Scan 1.PNG": "scan-1.png",
		"..":         "media",
		".":          "media",
		".hidden":    "hidden",
		"con.jpg":    "media.jpg", // reserved stem -> fallback, extension kept
	}
	for in, want := range cases {
		if got := safeMediaBase(in); got != want {
			t.Errorf("safeMediaBase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOutputName_CollisionSuffix(t *testing.T) {
	r := &mediaResolver{assigned: map[string]string{}, usedNames: map[string]bool{}}

	first := r.outputName("/a/photo.jpg", "photo.jpg")
	second := r.outputName("/b/photo.jpg", "photo.jpg") // same basename, different source
	reuse := r.outputName("/a/photo.jpg", "photo.jpg")  // same source path -> same name

	if first != "photo.jpg" {
		t.Errorf("first = %q, want photo.jpg", first)
	}
	if second != "photo-2.jpg" {
		t.Errorf("second = %q, want photo-2.jpg", second)
	}
	if reuse != first {
		t.Errorf("reuse = %q, want %q", reuse, first)
	}
}

func TestAppendUnique(t *testing.T) {
	out := appendUnique(nil, "a")
	out = appendUnique(out, "a") // duplicate -> ignored
	out = appendUnique(out, "b")
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Errorf("appendUnique result = %v, want [a b]", out)
	}
}
