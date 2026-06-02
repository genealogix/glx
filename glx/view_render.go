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
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

// templateRootPrefixes documents the relative path back to the site root from
// each page location, used so links resolve under both file:// and HTTP.
const (
	rootPrefixTop    = ""    // index.html, search.html
	rootPrefixNested = "../" // persons/*.html, sources/index.html, places/index.html
)

// Nav highlight keys. These match the literals compared in base.html.tmpl;
// the sources/places keys reuse the existing section-name constants.
const (
	navHome    = "home"
	navSources = searchFieldSources // "sources"
	navPlaces  = searchFieldPlaces  // "places"
	navSearch  = "search"
)

// renderContext is the data passed to every page template. The base layout and
// page templates read only the fields relevant to the page being rendered.
type renderContext struct {
	Title      string
	Generated  string
	RootPrefix string
	PageTitle  string
	Active     string // nav highlight key: "home", "sources", "places", "search"
	Stats      siteStats
	Persons    []*personPage
	Sources    []sourceRow
	Places     []placeRow
	Person     *personPage
}

// viewTemplateFuncs are the helpers available inside templates.
var viewTemplateFuncs = template.FuncMap{
	"personFile": personFileName,
	"humanize":   humanizeToken,
	"mediaURL":   mediaURL,
}

// mediaURL marks a resolved media source as trusted so html/template does not
// neutralize it. This is safe because every media Src is produced by us, not
// taken raw from the archive: it is either a relative "../media/..." path, a
// base64 "data:" URI we built from a local image, or an http(s) URL that
// passed isHTTPURL. Any other scheme (javascript:, etc.) fails the file read
// and is rendered as missing media rather than reaching this helper.
func mediaURL(src string) template.URL {
	return template.URL(src) // #nosec G203
}

// renderSite writes the complete static site (assets, search index, and all
// HTML pages) into outputDir.
func renderSite(model *siteModel, outputDir string) error {
	if err := os.MkdirAll(outputDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := writeStaticAssets(outputDir); err != nil {
		return err
	}
	if err := writeSearchIndex(model, outputDir); err != nil {
		return err
	}

	base := &renderContext{Title: model.Title, Generated: model.Generated}

	if err := renderHome(model, base, outputDir); err != nil {
		return err
	}
	if err := renderSourcesPage(model, base, outputDir); err != nil {
		return err
	}
	if err := renderPlacesPage(model, base, outputDir); err != nil {
		return err
	}
	if err := renderSearchPage(base, outputDir); err != nil {
		return err
	}

	return renderPersonPages(model, base, outputDir)
}

// renderHome writes the landing page (index.html).
func renderHome(model *siteModel, base *renderContext, outputDir string) error {
	tmpl, err := loadPageTemplate("index.html.tmpl")
	if err != nil {
		return err
	}
	ctx := *base
	ctx.RootPrefix = rootPrefixTop
	ctx.Active = navHome
	ctx.Stats = model.Stats
	ctx.Persons = model.Persons

	return writeRendered(tmpl, filepath.Join(outputDir, "index.html"), &ctx)
}

// renderSourcesPage writes sources/index.html.
func renderSourcesPage(model *siteModel, base *renderContext, outputDir string) error {
	tmpl, err := loadPageTemplate("sources.html.tmpl")
	if err != nil {
		return err
	}
	ctx := *base
	ctx.RootPrefix = rootPrefixNested
	ctx.Active = navSources
	ctx.PageTitle = "Sources"
	ctx.Sources = model.Sources

	return writeRendered(tmpl, filepath.Join(outputDir, "sources", "index.html"), &ctx)
}

// renderPlacesPage writes places/index.html.
func renderPlacesPage(model *siteModel, base *renderContext, outputDir string) error {
	tmpl, err := loadPageTemplate("places.html.tmpl")
	if err != nil {
		return err
	}
	ctx := *base
	ctx.RootPrefix = rootPrefixNested
	ctx.Active = navPlaces
	ctx.PageTitle = "Places"
	ctx.Places = model.Places

	return writeRendered(tmpl, filepath.Join(outputDir, "places", "index.html"), &ctx)
}

// renderSearchPage writes search.html.
func renderSearchPage(base *renderContext, outputDir string) error {
	tmpl, err := loadPageTemplate("search.html.tmpl")
	if err != nil {
		return err
	}
	ctx := *base
	ctx.RootPrefix = rootPrefixTop
	ctx.Active = navSearch
	ctx.PageTitle = "Search"

	return writeRendered(tmpl, filepath.Join(outputDir, "search.html"), &ctx)
}

// renderPersonPages writes one profile page per person under persons/.
func renderPersonPages(model *siteModel, base *renderContext, outputDir string) error {
	tmpl, err := loadPageTemplate("person.html.tmpl")
	if err != nil {
		return err
	}
	for _, person := range model.Persons {
		ctx := *base
		ctx.RootPrefix = rootPrefixNested
		ctx.PageTitle = person.Name
		ctx.Person = person
		dest := filepath.Join(outputDir, "persons", personFileName(person.ID))
		if err := writeRendered(tmpl, dest, &ctx); err != nil {
			return err
		}
	}

	return nil
}

// loadPageTemplate parses the base layout together with a single page template,
// yielding a template set whose "base" entry is ready to execute.
func loadPageTemplate(page string) (*template.Template, error) {
	tmpl, err := template.New("base").Funcs(viewTemplateFuncs).ParseFS(
		viewTemplatesFS,
		"view/templates/base.html.tmpl",
		"view/templates/"+page,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template %q: %w", page, err)
	}

	return tmpl, nil
}

// writeRendered executes the "base" template and writes the result to path,
// creating parent directories as needed.
func writeRendered(tmpl *template.Template, path string, ctx *renderContext) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory for %q: %w", path, err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", ctx); err != nil {
		return fmt.Errorf("failed to render %q: %w", path, err)
	}

	return os.WriteFile(path, buf.Bytes(), filePermissions)
}

// writeStaticAssets copies the embedded CSS/JS into the site output.
func writeStaticAssets(outputDir string) error {
	for name, dest := range viewStaticAssets {
		data, err := viewAssetsFS.ReadFile("view/assets/" + name)
		if err != nil {
			return fmt.Errorf("failed to read embedded asset %q: %w", name, err)
		}
		destPath := filepath.Join(outputDir, filepath.FromSlash(dest))
		if err := os.MkdirAll(filepath.Dir(destPath), dirPermissions); err != nil {
			return fmt.Errorf("failed to create asset directory for %q: %w", dest, err)
		}
		if err := os.WriteFile(destPath, data, filePermissions); err != nil {
			return fmt.Errorf("failed to write asset %q: %w", dest, err)
		}
	}

	return nil
}

// writeSearchIndex writes the client-side search index as a JS file that
// assigns window.GLX_SEARCH_INDEX. A <script> include (rather than fetch) keeps
// search working when the site is opened from disk via file://.
func writeSearchIndex(model *siteModel, outputDir string) error {
	entries := model.Search
	if entries == nil {
		entries = []searchEntry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode search index: %w", err)
	}
	jsDir := filepath.Join(outputDir, "js")
	if err := os.MkdirAll(jsDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create js directory: %w", err)
	}
	content := append([]byte("window.GLX_SEARCH_INDEX = "), data...)
	content = append(content, []byte(";\n")...)

	return os.WriteFile(filepath.Join(jsDir, "search-index.js"), content, filePermissions)
}
