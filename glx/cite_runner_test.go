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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullCiteArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Repositories: map[string]*glxlib.Repository{
			"repo-fs": {Name: "FamilySearch"},
			"repo-nara": {Name: "NARA"},
		},
		Sources: map[string]*glxlib.Source{
			"source-marriages": {
				Title:        "Wisconsin, Marriages, 1836-1930",
				Type:         "vital_record",
				RepositoryID: "repo-fs",
			},
			"source-census": {
				Title:        "1860 U.S. Federal Census",
				Type:         "census",
				RepositoryID: "repo-nara",
			},
			"source-bare": {
				Title: "Bare Source",
			},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-full": {
				SourceID: "source-marriages",
				Properties: map[string]any{
					"url":      "https://example.com/record/123",
					"accessed": "2024-02-29",
					"locator":  "it 1 cn 02350",
				},
			},
			"cit-no-url": {
				SourceID: "source-census",
				Properties: map[string]any{
					"locator": "Page 104",
				},
			},
			"cit-bare": {
				SourceID:   "source-bare",
				Properties: map[string]any{},
			},
			"cit-with-own-repo": {
				SourceID:     "source-bare",
				RepositoryID: "repo-fs",
				Properties: map[string]any{
					"url": "https://example.com",
				},
			},
		},
	}
}

func TestFormatCitation_Full(t *testing.T) {
	archive := fullCiteArchive()
	cit := archive.Citations["cit-full"]

	result := formatCitation(cit, archive)
	assert.Contains(t, result, `"Wisconsin, Marriages, 1836-1930"`)
	assert.Contains(t, result, "vital_record")
	assert.Contains(t, result, "FamilySearch")
	assert.Contains(t, result, "https://example.com/record/123")
	assert.Contains(t, result, "2024-02-29")
	assert.Contains(t, result, "it 1 cn 02350")
	assert.True(t, result[len(result)-1] == '.', "should end with period")
}

func TestFormatCitation_NoURL(t *testing.T) {
	archive := fullCiteArchive()
	cit := archive.Citations["cit-no-url"]

	result := formatCitation(cit, archive)
	assert.Contains(t, result, `"1860 U.S. Federal Census"`)
	assert.Contains(t, result, "census")
	assert.Contains(t, result, "NARA")
	assert.Contains(t, result, "Page 104")
	assert.NotContains(t, result, "(")
}

func TestFormatCitation_Bare(t *testing.T) {
	archive := fullCiteArchive()
	cit := archive.Citations["cit-bare"]

	result := formatCitation(cit, archive)
	assert.Contains(t, result, `"Bare Source"`)
	assert.NotContains(t, result, "vital_record")
	assert.NotContains(t, result, "FamilySearch")
}

func TestFormatCitation_CitationOwnRepo(t *testing.T) {
	archive := fullCiteArchive()
	cit := archive.Citations["cit-with-own-repo"]

	result := formatCitation(cit, archive)
	// Citation's own repo should take precedence
	assert.Contains(t, result, "FamilySearch")
	assert.Contains(t, result, "https://example.com")
}

func TestResolveSourceTitle(t *testing.T) {
	archive := fullCiteArchive()

	assert.Equal(t, "Wisconsin, Marriages, 1836-1930", resolveSourceTitle("source-marriages", archive))
	assert.Equal(t, "", resolveSourceTitle("nonexistent", archive))
	assert.Equal(t, "", resolveSourceTitle("", archive))
}

func TestResolveSourceType(t *testing.T) {
	archive := fullCiteArchive()

	assert.Equal(t, "vital_record", resolveSourceType("source-marriages", archive))
	assert.Equal(t, "", resolveSourceType("source-bare", archive))
	assert.Equal(t, "", resolveSourceType("nonexistent", archive))
}

func TestResolveRepositoryName(t *testing.T) {
	archive := fullCiteArchive()

	// Via source's repo
	cit := archive.Citations["cit-full"]
	assert.Equal(t, "FamilySearch", resolveRepositoryName(cit, archive))

	// Via citation's own repo (overrides source)
	cit = archive.Citations["cit-with-own-repo"]
	assert.Equal(t, "FamilySearch", resolveRepositoryName(cit, archive))

	// No repo
	cit = archive.Citations["cit-bare"]
	assert.Equal(t, "", resolveRepositoryName(cit, archive))
}

func TestBuildAccessClause(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		url      string
		accessed string
		want     string
	}{
		{"full", "FamilySearch", "https://example.com", "2024-01-01", "FamilySearch (https://example.com : 2024-01-01)"},
		{"repo+url", "FamilySearch", "https://example.com", "", "FamilySearch (https://example.com)"},
		{"repo+accessed", "FamilySearch", "", "2024-01-01", "FamilySearch (2024-01-01)"},
		{"url only", "", "https://example.com", "", "(https://example.com)"},
		{"repo only", "FamilySearch", "", "", "FamilySearch"},
		{"empty", "", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildAccessClause(tt.repo, tt.url, tt.accessed)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCitationProperty(t *testing.T) {
	cit := &glxlib.Citation{
		Properties: map[string]any{
			"url":     "https://example.com",
			"locator": "Page 5",
			"number":  42,
		},
	}

	assert.Equal(t, "https://example.com", citationProperty(cit, "url"))
	assert.Equal(t, "Page 5", citationProperty(cit, "locator"))
	assert.Equal(t, "42", citationProperty(cit, "number"))
	assert.Equal(t, "", citationProperty(cit, "missing"))
}

func TestCitationProperty_NilProperties(t *testing.T) {
	cit := &glxlib.Citation{}
	assert.Equal(t, "", citationProperty(cit, "url"))
}

func TestJoinNonEmpty(t *testing.T) {
	assert.Equal(t, "a : b", joinNonEmpty(" : ", "a", "b"))
	assert.Equal(t, "a", joinNonEmpty(" : ", "a", ""))
	assert.Equal(t, "b", joinNonEmpty(" : ", "", "b"))
	assert.Equal(t, "", joinNonEmpty(" : ", "", ""))
}

func TestShowCitation_CompleteFamily(t *testing.T) {
	err := showCitation("../docs/examples/complete-family", "citation-john-birth")
	require.NoError(t, err)
}

func TestShowCitation_NotFound(t *testing.T) {
	err := showCitation("../docs/examples/complete-family", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestShowAllCitations_CompleteFamily(t *testing.T) {
	err := showAllCitations("../docs/examples/complete-family")
	require.NoError(t, err)
}
