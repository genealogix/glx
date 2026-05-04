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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResearchLogRoundTrip exercises the full serialize → deserialize cycle for
// a research log entry containing every field (single-file format).
func TestResearchLogRoundTrip(t *testing.T) {
	original := &GLXFile{
		Persons: map[string]*Person{
			"person-jane-webb": {Properties: map[string]any{"name": "Jane Webb"}},
		},
		Repositories: map[string]*Repository{
			"repo-familysearch": {Name: "FamilySearch"},
		},
		Sources: map[string]*Source{
			"source-1860-census": {Title: "1860 U.S. Census"},
		},
		Citations: map[string]*Citation{
			"citation-1860-webb": {SourceID: "source-1860-census"},
		},
		ResearchLogs: map[string]*ResearchLog{
			"log-1860-jane-search": {
				Objective:  "Locate Jane Webb in 1860 census",
				Date:       "2026-03-09",
				Researcher: "Isaac Schepp",
				Status:     ResearchStatusComplete,
				Searches: []ResearchLogSearch{
					{
						Repository:  "repo-familysearch",
						Collection:  "United States, Census, 1860",
						SearchTerms: "Jane Webb, Wisconsin",
						Result:      SearchResultFound,
						Date:        "2026-03-09",
						Citation:    "citation-1860-webb",
						Notes:       NoteList{"Found as Jane, age 28"},
					},
					{
						Repository:  "repo-familysearch",
						Collection:  "United States, Census, 1850",
						SearchTerms: "Jane Webb, Hartford County",
						Result:      SearchResultNotFound,
						Notes:       NoteList{"Negative evidence: not in 1850"},
					},
				},
				Conclusions:    "Located in 1860; absent from 1850 (consistent with later migration).",
				RelatedPersons: []string{"person-jane-webb"},
				Notes:          NoteList{"Closes 1860-census research arc"},
			},
		},
	}

	// Standard vocabularies must be loaded so the validator can resolve
	// search_result_types / research_status_types references.
	require.NoError(t, LoadStandardVocabulariesIntoGLX(original))

	s := NewSerializer(nil)

	yamlBytes, err := s.SerializeSingleFileBytes(original)
	require.NoError(t, err)

	loaded, err := s.DeserializeSingleFileBytes(yamlBytes)
	require.NoError(t, err)

	require.Len(t, loaded.ResearchLogs, 1)

	got := loaded.ResearchLogs["log-1860-jane-search"]
	require.NotNil(t, got)

	assert.Equal(t, "Locate Jane Webb in 1860 census", got.Objective)
	assert.Equal(t, DateString("2026-03-09"), got.Date)
	assert.Equal(t, "Isaac Schepp", got.Researcher)
	assert.Equal(t, ResearchStatusComplete, got.Status)
	assert.Equal(t, "Located in 1860; absent from 1850 (consistent with later migration).", got.Conclusions)
	assert.Equal(t, []string{"person-jane-webb"}, got.RelatedPersons)

	require.Len(t, got.Searches, 2)
	assert.Equal(t, SearchResultFound, got.Searches[0].Result)
	assert.Equal(t, "citation-1860-webb", got.Searches[0].Citation)
	assert.Equal(t, SearchResultNotFound, got.Searches[1].Result)
	assert.Empty(t, got.Searches[1].Citation, "not_found searches typically have no citation")
}

// TestResearchLogMultiFileRoundTrip verifies the entity gets its own directory
// in multi-file serialization.
func TestResearchLogMultiFileRoundTrip(t *testing.T) {
	original := &GLXFile{
		ResearchLogs: map[string]*ResearchLog{
			"log-find-marriage": {
				Objective: "Locate marriage record",
				Searches: []ResearchLogSearch{
					{Result: SearchResultNotSearched, Collection: "Wisconsin Marriage Index"},
				},
			},
		},
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(original))

	s := NewSerializer(nil)
	files, err := s.SerializeMultiFileToMap(original)
	require.NoError(t, err)

	// filepath.Join uses the OS separator, so normalize before comparing.
	var logFile string
	for path := range files {
		if strings.HasPrefix(filepath.ToSlash(path), "research_logs/") {
			logFile = path

			break
		}
	}
	require.NotEmpty(t, logFile, "expected a file under research_logs/ in multi-file output, got paths: %v", files)

	// Empty/zero-valued optional fields should be omitted from on-disk YAML.
	// Guards against someone removing `omitempty` from a future log field.
	body := string(files[logFile])
	for _, omitted := range []string{"status:", "researcher:", "date:", "conclusions:", "related_persons:"} {
		assert.NotContains(t, body, omitted, "minimal research log unexpectedly serialized %q", omitted)
	}

	loaded, conflicts, err := s.DeserializeMultiFileFromMap(files)
	require.NoError(t, err)
	assert.Empty(t, conflicts)
	require.Len(t, loaded.ResearchLogs, 1)
	assert.Equal(t, "Locate marriage record", loaded.ResearchLogs["log-find-marriage"].Objective)
}

// TestResearchLogReferenceValidation verifies that broken references inside a
// research log are surfaced as validation errors. Both missing entity refs
// (repository, citation, person) and unknown vocabulary values (status, result)
// are reported as hard errors via the refType-tag walker — matching how
// Assertion.Confidence and other refType-tagged vocabulary fields behave.
func TestResearchLogReferenceValidation(t *testing.T) {
	g := &GLXFile{
		ResearchLogs: map[string]*ResearchLog{
			"log-bad-refs": {
				Objective: "Test broken refs",
				Status:    "made-up-status",
				Searches: []ResearchLogSearch{
					{
						Repository: "repo-nonexistent",
						Result:     "made-up-result",
						Citation:   "citation-nope",
					},
				},
				RelatedPersons: []string{"person-ghost"},
			},
		},
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(g))

	result := g.Validate()

	require.NotEmpty(t, result.Errors, "expected reference errors")
	errMsgs := strings.Join(collectErrorMessages(result.Errors), "\n")
	assert.Contains(t, errMsgs, "repo-nonexistent")
	assert.Contains(t, errMsgs, "citation-nope")
	assert.Contains(t, errMsgs, "person-ghost")
	assert.Contains(t, errMsgs, "made-up-status")
	assert.Contains(t, errMsgs, "made-up-result")
}

func collectErrorMessages(errs []ValidationError) []string {
	out := make([]string, len(errs))
	for i, e := range errs {
		out[i] = e.Message
	}

	return out
}
