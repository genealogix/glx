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
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestResearchLogRoundTrip(t *testing.T) {
	yamlData := `research_logs:
  research-log-1860-census-jane-webb:
    title: 1860 census - locate Jane Webb
    subject:
      person: person-jane-webb
    objective: Confirm Jane Webb birthplace
    status: complete
    date: "2026-03-09"
    researcher: Isaac Schepp
    searches:
      - repository: repository-familysearch
        collection: United States, Census, 1860
        query: Jane Webb, Hartford County
        result: found
        citation: citation-1860-census-webb
      - repository: repository-familysearch
        collection: United States, Census, 1860
        query: Jane Miller, Hartford County
        result: not_found
        notes: No matches under maiden name
    citations:
      - citation-1860-census-webb
    conclusions: Jane Webb located, born Florida confirmed.
`

	var glx GLXFile
	err := yaml.Unmarshal([]byte(yamlData), &glx)
	require.NoError(t, err)

	require.Len(t, glx.ResearchLogs, 1)
	log, ok := glx.ResearchLogs["research-log-1860-census-jane-webb"]
	require.True(t, ok)
	assert.Equal(t, "1860 census - locate Jane Webb", log.Title)
	require.NotNil(t, log.Subject)
	assert.Equal(t, "person-jane-webb", log.Subject.Person)
	assert.Equal(t, "complete", log.Status)
	assert.Equal(t, "Isaac Schepp", log.Researcher)
	require.Len(t, log.Searches, 2)
	assert.Equal(t, "found", log.Searches[0].Result)
	assert.Equal(t, "citation-1860-census-webb", log.Searches[0].CitationID)
	assert.Equal(t, "not_found", log.Searches[1].Result)
	assert.Equal(t, NoteList{"No matches under maiden name"}, log.Searches[1].Notes)
	assert.Equal(t, []string{"citation-1860-census-webb"}, log.Citations)

	// Re-serialize and verify the result re-parses identically.
	out, err := yaml.Marshal(&glx)
	require.NoError(t, err)

	var glx2 GLXFile
	require.NoError(t, yaml.Unmarshal(out, &glx2))

	log2, ok := glx2.ResearchLogs["research-log-1860-census-jane-webb"]
	require.True(t, ok)
	assert.Equal(t, log.Title, log2.Title)
	assert.Equal(t, log.Status, log2.Status)
	require.NotNil(t, log2.Subject)
	assert.Equal(t, log.Subject.Person, log2.Subject.Person)
	require.Len(t, log2.Searches, 2)
	assert.Equal(t, log.Searches[0].Result, log2.Searches[0].Result)
	assert.Equal(t, log.Searches[1].Notes, log2.Searches[1].Notes)
}

func TestResearchLogMultiFileSerialize(t *testing.T) {
	glx := &GLXFile{
		ResearchLogs: map[string]*ResearchLog{
			"research-log-1": {
				Title:    "Test log",
				Status:   "open",
				Searches: []Search{{Collection: "Test collection", Result: "not_found"}},
			},
		},
	}

	// Validation off — vocabularies aren't loaded for this minimal fixture.
	s := NewSerializer(&SerializerOptions{Validate: false})
	files, err := s.SerializeMultiFileToMap(glx)
	require.NoError(t, err)

	// Multi-file serialization writes per-entity files. Path uses filepath.Join,
	// which differs by platform; assert by suffix to stay portable.
	var found bool
	for path := range files {
		if strings.HasSuffix(path, "research-log-1.glx") &&
			(strings.Contains(path, "research_logs/") || strings.Contains(path, "research_logs\\")) {
			found = true

			break
		}
	}
	assert.True(t, found, "expected research_logs/research-log-1.glx in output, got files: %v",
		slices.Sorted(maps.Keys(files)))
}

func TestResearchLogValidationReferences(t *testing.T) {
	t.Run("subject references existing person", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Subject: &EntityRef{Person: "person-1"},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("subject references non-existent person", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Subject: &EntityRef{Person: "person-missing"},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-missing")
	})

	t.Run("search references existing repository and source", func(t *testing.T) {
		archive := &GLXFile{
			Repositories: map[string]*Repository{"repo-1": {Name: "Test Repo"}},
			Sources:      map[string]*Source{"source-1": {Title: "Test Source"}},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Searches: []Search{
						{RepositoryID: "repo-1", SourceID: "source-1"},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("search references non-existent repository", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Searches: []Search{{RepositoryID: "repo-missing"}},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "repo-missing")
	})

	t.Run("search citation reference checked", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Searches: []Search{{CitationID: "citation-missing"}},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "citation-missing")
	})

	t.Run("log-level citations reference checked", func(t *testing.T) {
		archive := &GLXFile{
			Citations: map[string]*Citation{"cite-1": {SourceID: "source-1"}},
			Sources:   map[string]*Source{"source-1": {Title: "Test"}},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Citations: []string{"cite-1", "cite-missing"},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "cite-missing")
	})
}

func TestResearchLogValidationVocabulary(t *testing.T) {
	t.Run("status from loaded vocabulary accepted", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogStatusTypes: map[string]*VocabularyEntry{
				"complete": {Label: "Complete"},
			},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Status: "complete"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("status not in vocabulary rejected", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogStatusTypes: map[string]*VocabularyEntry{
				"complete": {Label: "Complete"},
			},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Status: "made-up-status"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "made-up-status")
	})

	t.Run("search result from loaded vocabulary accepted", func(t *testing.T) {
		archive := &GLXFile{
			SearchResultTypes: map[string]*VocabularyEntry{
				"not_found": {Label: "Not Found"},
			},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Searches: []Search{{Result: "not_found"}}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("search result not in vocabulary rejected", func(t *testing.T) {
		archive := &GLXFile{
			SearchResultTypes: map[string]*VocabularyEntry{
				"found": {Label: "Found"},
			},
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Searches: []Search{{Result: "made-up-result"}}},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "made-up-result")
	})
}

func TestResearchLogStandardVocabulariesLoaded(t *testing.T) {
	glx := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))

	assert.Len(t, glx.SearchResultTypes, 5, "expected 5 standard search result types")
	assert.Contains(t, glx.SearchResultTypes, SearchResultFound)
	assert.Contains(t, glx.SearchResultTypes, SearchResultNotFound)
	assert.Contains(t, glx.SearchResultTypes, SearchResultInconclusive)
	assert.Contains(t, glx.SearchResultTypes, SearchResultPartial)
	assert.Contains(t, glx.SearchResultTypes, SearchResultNotSearched)

	assert.Len(t, glx.ResearchLogStatusTypes, 4, "expected 4 standard research log statuses")
	assert.Contains(t, glx.ResearchLogStatusTypes, ResearchLogStatusOpen)
	assert.Contains(t, glx.ResearchLogStatusTypes, ResearchLogStatusInProgress)
	assert.Contains(t, glx.ResearchLogStatusTypes, ResearchLogStatusComplete)
	assert.Contains(t, glx.ResearchLogStatusTypes, ResearchLogStatusBlocked)
}

func TestResearchLogDateFormatValidation(t *testing.T) {
	t.Run("malformed log date emits warning", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Date: "not-a-date"},
			},
		}
		result := archive.Validate()
		var dateWarning bool
		for _, w := range result.Warnings {
			if w.SourceType == EntityTypeResearchLogs && w.Field == "date" {
				dateWarning = true

				break
			}
		}
		assert.True(t, dateWarning, "expected date-format warning on ResearchLog, got %v", result.Warnings)
	})

	t.Run("malformed search date emits warning", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {Searches: []Search{{Date: "not-a-date"}}},
			},
		}
		result := archive.Validate()
		var dateWarning bool
		for _, w := range result.Warnings {
			if w.SourceType == EntityTypeResearchLogs &&
				strings.HasPrefix(w.Field, "searches[") &&
				strings.HasSuffix(w.Field, "].date") {
				dateWarning = true

				break
			}
		}
		assert.True(t, dateWarning, "expected date-format warning on Search.Date, got %v", result.Warnings)
	})

	t.Run("valid dates produce no date warnings", func(t *testing.T) {
		archive := &GLXFile{
			ResearchLogs: map[string]*ResearchLog{
				"log-1": {
					Date:     "2026-03-09",
					Searches: []Search{{Date: "2026-03-10"}},
				},
			},
		}
		result := archive.Validate()
		for _, w := range result.Warnings {
			assert.NotEqual(t, "date", w.Field,
				"unexpected date warning for valid date: %v", w)
		}
	})
}

func TestResearchLogMergeConflictDetection(t *testing.T) {
	g1 := &GLXFile{
		ResearchLogs: map[string]*ResearchLog{
			"log-1": {Title: "First"},
		},
	}
	g2 := &GLXFile{
		ResearchLogs: map[string]*ResearchLog{
			"log-1": {Title: "Second"},
			"log-2": {Title: "New"},
		},
	}

	conflicts, _ := g1.Merge(g2)
	require.Len(t, conflicts, 1)
	assert.Contains(t, conflicts[0], "research_logs")
	assert.Contains(t, conflicts[0], "log-1")
	assert.Len(t, g1.ResearchLogs, 2)
	assert.Contains(t, g1.ResearchLogs, "log-2")
}
