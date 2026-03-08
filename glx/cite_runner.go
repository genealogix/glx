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
	"fmt"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// showCitation generates and prints a formatted citation string for the given
// citation ID. It assembles the citation from structured fields: source title,
// source type, repository name, URL, and accessed date.
func showCitation(archivePath, citationID string) error {
	archive, err := loadArchiveForQuery(archivePath)
	if err != nil {
		return err
	}

	cit, ok := archive.Citations[citationID]
	if !ok {
		return fmt.Errorf("citation %q not found in archive", citationID)
	}

	text := formatCitation(cit, archive)
	fmt.Println(text)

	return nil
}

// showAllCitations generates formatted citation strings for all citations
// in the archive.
func showAllCitations(archivePath string) error {
	archive, err := loadArchiveForQuery(archivePath)
	if err != nil {
		return err
	}

	ids := sortedKeys(archive.Citations)
	if len(ids) == 0 {
		fmt.Println("No citations found.")
		return nil
	}

	for i, id := range ids {
		cit := archive.Citations[id]
		text := formatCitation(cit, archive)
		fmt.Printf("%s:\n  %s\n", id, text)
		if i < len(ids)-1 {
			fmt.Println()
		}
	}

	return nil
}

// formatCitation assembles a citation string from structured fields.
//
// Format: "<source-title>", <source-type>, <repository> (<url> : <accessed>), <locator>.
//
// Components are omitted when the underlying data is not available.
func formatCitation(cit *glxlib.Citation, archive *glxlib.GLXFile) string {
	var parts []string

	// Source title (quoted)
	sourceTitle := resolveSourceTitle(cit.SourceID, archive)
	if sourceTitle != "" {
		parts = append(parts, fmt.Sprintf("%q", sourceTitle))
	}

	// Source type label
	sourceTypeLabel := resolveSourceType(cit.SourceID, archive)
	if sourceTypeLabel != "" {
		parts = append(parts, sourceTypeLabel)
	}

	// Repository name + URL/accessed
	repoName := resolveRepositoryName(cit, archive)
	url := citationProperty(cit, "url")
	accessed := citationProperty(cit, "accessed")

	accessClause := buildAccessClause(repoName, url, accessed)
	if accessClause != "" {
		parts = append(parts, accessClause)
	}

	result := strings.Join(parts, ", ")

	// Append locator as a trailing clause if present
	locator := citationProperty(cit, "locator")
	if locator != "" {
		result += ", " + locator
	}

	if result != "" {
		result += "."
	}

	return result
}

// resolveSourceTitle looks up the source title for a citation.
func resolveSourceTitle(sourceID string, archive *glxlib.GLXFile) string {
	if sourceID == "" {
		return ""
	}
	if src, ok := archive.Sources[sourceID]; ok {
		return src.Title
	}

	return ""
}

// resolveSourceType looks up the source type for a citation and returns
// a human-readable label.
func resolveSourceType(sourceID string, archive *glxlib.GLXFile) string {
	if sourceID == "" {
		return ""
	}

	src, ok := archive.Sources[sourceID]
	if !ok || src.Type == "" {
		return ""
	}

	return src.Type
}

// resolveRepositoryName resolves the repository name for a citation.
// Checks the citation's own RepositoryID first, then falls back to the source's.
func resolveRepositoryName(cit *glxlib.Citation, archive *glxlib.GLXFile) string {
	repoID := cit.RepositoryID
	if repoID == "" {
		if src, ok := archive.Sources[cit.SourceID]; ok {
			repoID = src.RepositoryID
		}
	}

	if repoID == "" {
		return ""
	}

	if repo, ok := archive.Repositories[repoID]; ok {
		return repo.Name
	}

	return ""
}

// buildAccessClause builds the "Repository (url : date)" portion.
func buildAccessClause(repoName, url, accessed string) string {
	if repoName == "" && url == "" && accessed == "" {
		return ""
	}

	// If we have URL or accessed date, format as "Repo (url : date)"
	if url != "" || accessed != "" {
		inner := joinNonEmpty(" : ", url, accessed)
		if repoName != "" {
			return fmt.Sprintf("%s (%s)", repoName, inner)
		}

		return fmt.Sprintf("(%s)", inner)
	}

	// Repository name only
	return repoName
}

// citationProperty extracts a string property from citation properties.
func citationProperty(cit *glxlib.Citation, key string) string {
	if cit.Properties == nil {
		return ""
	}

	raw, ok := cit.Properties[key]
	if !ok {
		return ""
	}

	if s, ok := raw.(string); ok {
		return s
	}

	return fmt.Sprint(raw)
}

// joinNonEmpty joins non-empty strings with the given separator.
func joinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}

	return strings.Join(nonEmpty, sep)
}
