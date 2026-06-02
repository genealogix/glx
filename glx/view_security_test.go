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

// TestRenderSite_EscapesArchiveContentAcrossPages locks in html/template
// auto-escaping for archive-controlled text on the person, index, sources, and
// places pages, guarding against a future template.HTML/JS regression.
func TestRenderSite_EscapesArchiveContentAcrossPages(t *testing.T) {
	const payload = `<script>alert(1)</script>`
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-evil": {
				Properties: map[string]any{"name": payload},
				Notes:      glxlib.NoteList{payload},
			},
		},
		Sources: map[string]*glxlib.Source{
			"source-evil": {Title: payload, Type: "census"},
		},
		Places: map[string]*glxlib.Place{
			"place-evil": {Name: payload, Type: "city"},
		},
	}
	model := buildSiteModel(archive, viewModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	pages := []string{
		"index.html",
		"sources/index.html",
		"places/index.html",
		filepath.Join("persons", model.Persons[0].File),
	}
	for _, rel := range pages {
		body := readFile(t, filepath.Join(out, filepath.FromSlash(rel)))
		if strings.Contains(body, payload) {
			t.Errorf("%s contains unescaped payload", rel)
		}
	}
	// Spot-check the person page actually escaped it (not merely omitted).
	person := readFile(t, filepath.Join(out, "persons", model.Persons[0].File))
	if !strings.Contains(person, "&lt;script&gt;") {
		t.Error("person page did not HTML-escape the script payload")
	}
}

// TestSearchIndexScriptSafety ensures a name containing </script> stays escaped
// in the JS-loaded search index (encoding/json SetEscapeHTML default).
func TestSearchIndexScriptSafety(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-x": {Properties: map[string]any{"name": `</script><img onerror=alert(1)>`}},
		},
	}
	model := buildSiteModel(archive, viewModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	idx := readFile(t, filepath.Join(out, "js", "search-index.js"))
	// encoding/json (SetEscapeHTML default) escapes < > & to \uXXXX, so no
	// literal angle bracket survives — a name can never break out of the
	// enclosing <script> tag.
	for _, bad := range []string{"</script>", "<", ">"} {
		if strings.Contains(idx, bad) {
			t.Errorf("search-index.js contains literal %q; HTML chars must be \\u-escaped", bad)
		}
	}
	// Sanity: the escaped form is present (proving the name was emitted, not dropped).
	lower := strings.ToLower(idx)
	if !strings.Contains(lower, "u003c") {
		t.Error("expected the name's < to appear unicode-escaped in the search index")
	}
}

// TestSVGMediaRendersAsImage verifies an SVG with a non-image MIME renders via
// <img> (scripting disabled) rather than a clickable link a browser would open
// as an active document.
func TestSVGMediaRendersAsImage(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "drawing.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))

	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-x": {Properties: map[string]any{"name": "X"}},
		},
		Media: map[string]*glxlib.Media{
			"media-svg": {URI: "drawing.svg", MimeType: "application/xml"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-x": {Subject: glxlib.EntityRef{Person: "person-x"}, Media: []string{"media-svg"}},
		},
	}
	model := buildSiteModel(archive, viewModelOptions{})
	out := t.TempDir()
	if _, err := resolveSiteMedia(model, base, out, false); err != nil {
		t.Fatalf("resolveSiteMedia: %v", err)
	}
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	page := readFile(t, filepath.Join(out, "persons", model.Persons[0].File))
	if !strings.Contains(page, `<img src="../media/drawing.svg"`) {
		t.Error("SVG should render via <img>, not a clickable link")
	}
	if strings.Contains(page, `class="media-file"`) {
		t.Error("SVG must not render as an openable media-file link")
	}
}

// TestRenderSite_PersonFileCollisions verifies that distinct person IDs which
// sanitize to the same stem get distinct pages on disk and distinct index links
// (no silent overwrite, no cross-linking).
func TestRenderSite_PersonFileCollisions(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"Person-A": {Properties: map[string]any{"name": "Alpha"}},
			"person-a": {Properties: map[string]any{"name": "Beta"}},
		},
	}
	model := buildSiteModel(archive, viewModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	if len(model.Persons) != 2 {
		t.Fatalf("expected 2 persons, got %d", len(model.Persons))
	}
	a, b := model.Persons[0].File, model.Persons[1].File
	if a == b {
		t.Fatalf("colliding IDs got the same filename %q", a)
	}
	for _, f := range []string{a, b} {
		if _, err := os.Stat(filepath.Join(out, "persons", f)); err != nil {
			t.Errorf("person page %q not written: %v", f, err)
		}
	}
	index := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(index, "persons/"+a) || !strings.Contains(index, "persons/"+b) {
		t.Error("index.html must link to both colliding persons distinctly")
	}
}
