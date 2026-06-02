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
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

const exampleArchive = "../docs/examples/assertion-workflow/archive.glx"

func TestRenderSite_WritesAllFiles(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{Title: "Smith Family"})
	out := t.TempDir()

	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	wantFiles := []string{
		"index.html",
		"search.html",
		"sources/index.html",
		"places/index.html",
		"css/style.css",
		"js/search.js",
		"js/theme.js",
		"js/search-index.js",
		"persons/person-john-smith.html",
		"persons/person-jane-doe.html",
	}
	for _, rel := range wantFiles {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected file %q: %v", rel, err)
		}
	}

	index := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(index, "Smith Family") {
		t.Error("index.html missing site title")
	}
	if !strings.Contains(index, "persons/person-john-smith.html") {
		t.Error("index.html missing link to John's profile")
	}

	john := readFile(t, filepath.Join(out, "persons", "person-john-smith.html"))
	for _, want := range []string{"John Smith", "1850–1920", "Boston, Massachusetts, United States", "1880 US Census", "Jane Doe"} {
		if !strings.Contains(john, want) {
			t.Errorf("John's profile missing %q", want)
		}
	}

	searchIdx := readFile(t, filepath.Join(out, "js", "search-index.js"))
	if !strings.HasPrefix(searchIdx, "window.GLX_SEARCH_INDEX =") {
		t.Error("search-index.js missing global assignment")
	}
}

func TestRenderSite_EscapesHTML(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-evil": {Properties: map[string]any{"name": "<script>alert(1)</script>"}},
		},
	}
	model := buildSiteModel(archive, viewModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	page := readFile(t, filepath.Join(out, "persons", "person-evil.html"))
	if strings.Contains(page, "<script>alert(1)</script>") {
		t.Error("person name was not HTML-escaped")
	}
	if !strings.Contains(page, "&lt;script&gt;") {
		t.Error("expected escaped script tag in output")
	}
}

func TestResolveSiteMedia_CopyMode(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "portrait.jpg"), []byte("\xff\xd8\xff\xe0JFIF-fake"))

	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})
	out := t.TempDir()

	copied, err := resolveSiteMedia(model, base, out, false)
	if err != nil {
		t.Fatalf("resolveSiteMedia: %v", err)
	}
	if copied != 1 {
		t.Fatalf("copied = %d, want 1", copied)
	}

	item := johnMedia(t, model)
	if item.Src != "../media/portrait.jpg" {
		t.Errorf("Src = %q, want ../media/portrait.jpg", item.Src)
	}
	if _, err := os.Stat(filepath.Join(out, "media", "portrait.jpg")); err != nil {
		t.Errorf("copied media file missing: %v", err)
	}
}

func TestResolveSiteMedia_EmbedMode(t *testing.T) {
	base := t.TempDir()
	writeFile(t, filepath.Join(base, "portrait.jpg"), []byte("fake-image-bytes"))

	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})
	out := t.TempDir()

	copied, err := resolveSiteMedia(model, base, out, true)
	if err != nil {
		t.Fatalf("resolveSiteMedia: %v", err)
	}
	if copied != 0 {
		t.Errorf("copied = %d, want 0 in embed mode", copied)
	}

	item := johnMedia(t, model)
	if !strings.HasPrefix(item.Src, "data:image/jpeg;base64,") {
		t.Errorf("embedded Src = %q, want data: URI", item.Src)
	}
	if _, err := os.Stat(filepath.Join(out, "media")); !os.IsNotExist(err) {
		t.Error("embed mode should not create a media directory")
	}

	// The data URI must survive template rendering (html/template would
	// otherwise neutralize a data: src to #ZgotmplZ without mediaURL).
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}
	page := readFile(t, filepath.Join(out, "persons", "person-john-smith.html"))
	if !strings.Contains(page, "data:image/jpeg;base64,") {
		t.Error("embedded data URI did not survive rendering")
	}
	if strings.Contains(page, "#ZgotmplZ") {
		t.Error("media src was neutralized by html/template sanitization")
	}
}

func TestResolveSiteMedia_MissingFile(t *testing.T) {
	base := t.TempDir() // portrait.jpg deliberately absent
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})

	if _, err := resolveSiteMedia(model, base, t.TempDir(), false); err != nil {
		t.Fatalf("resolveSiteMedia: %v", err)
	}

	item := johnMedia(t, model)
	if !item.Missing {
		t.Error("expected Missing=true for absent media file")
	}
}

func TestGenerateView_EndToEnd(t *testing.T) {
	requireExample(t)
	out := filepath.Join(t.TempDir(), "site")
	streams, outBuf, _ := TestIOStreams()

	err := generateView(streams, viewOptions{
		ArchivePath: exampleArchive,
		OutputDir:   out,
		Title:       "Chen Family",
	})
	if err != nil {
		t.Fatalf("generateView: %v", err)
	}

	if !strings.Contains(outBuf.String(), "Generated static site") {
		t.Errorf("summary not printed: %q", outBuf.String())
	}

	index := readFile(t, filepath.Join(out, "index.html"))
	if !strings.Contains(index, "Chen Family") || !strings.Contains(index, "Robert Chen") {
		t.Error("index.html missing expected content")
	}
	if _, err := os.Stat(filepath.Join(out, "persons", "person-robert-chen.html")); err != nil {
		t.Errorf("Robert's profile not generated: %v", err)
	}
}

func TestGenerateView_Living(t *testing.T) {
	requireExample(t)
	out := filepath.Join(t.TempDir(), "site")
	streams, outBuf, _ := TestIOStreams()

	err := generateView(streams, viewOptions{
		ArchivePath: exampleArchive,
		OutputDir:   out,
		Living:      true,
	})
	if err != nil {
		t.Fatalf("generateView: %v", err)
	}

	if !strings.Contains(outBuf.String(), "Privatized") {
		t.Errorf("expected privatization notice, got %q", outBuf.String())
	}

	index := readFile(t, filepath.Join(out, "index.html"))
	if strings.Contains(index, "Robert Chen") {
		t.Error("living person's real name leaked into output")
	}
	if !strings.Contains(index, "Living") {
		t.Error("expected redacted 'Living' label in output")
	}
}

func TestSiteHandler(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.html"), []byte("<h1>hello</h1>"))

	server := httptest.NewServer(siteHandler(dir))
	defer server.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/index.html", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

// --- helpers ---

func johnMedia(t *testing.T, model *siteModel) *mediaItem {
	t.Helper()
	for _, p := range model.Persons {
		if p.ID == "person-john-smith" {
			if len(p.Media) == 0 {
				t.Fatal("John has no media")
			}

			return p.Media[0]
		}
	}
	t.Fatal("John page not found")

	return nil
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // test-controlled path
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}

	return string(data)
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func requireExample(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(exampleArchive); err != nil {
		t.Skipf("example archive not available: %v", err)
	}
}
