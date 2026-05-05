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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStudyRoundTrip verifies that a Study with every populated field survives
// single-file YAML serialization and deserialization unchanged.
func TestStudyRoundTrip(t *testing.T) {
	original := &GLXFile{
		Places: map[string]*Place{
			"place-pohl-goens": {Name: "Pohl-Göns"},
		},
		Sources: map[string]*Source{
			"source-pohl-goens-register": {Title: "Pohl-Göns Lutheran Register"},
		},
		Studies: map[string]*Study{
			"study-pohl-goens-ops": {
				Title:     "Pohl-Göns One Place Study",
				Type:      StudyTypeOnePlaceStudy,
				Status:    StudyStatusActive,
				DateRange: "FROM 1610 TO 1875",
				Places:    []string{"place-pohl-goens"},
				Sources:   []string{"source-pohl-goens-register"},
				Properties: map[string]any{
					"researcher": "I. Schepp",
				},
				Notes: NoteList{"Systematic review of every household 1610-1875."},
			},
		},
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(original))

	s := NewSerializer(nil)
	yamlBytes, err := s.SerializeSingleFileBytes(original)
	require.NoError(t, err)

	parsed, err := s.DeserializeSingleFileBytes(yamlBytes)
	require.NoError(t, err)

	require.Len(t, parsed.Studies, 1)
	got := parsed.Studies["study-pohl-goens-ops"]
	require.NotNil(t, got)
	assert.Equal(t, "Pohl-Göns One Place Study", got.Title)
	assert.Equal(t, StudyTypeOnePlaceStudy, got.Type)
	assert.Equal(t, StudyStatusActive, got.Status)
	assert.Equal(t, DateString("FROM 1610 TO 1875"), got.DateRange)
	assert.Equal(t, []string{"place-pohl-goens"}, got.Places)
	assert.Equal(t, []string{"source-pohl-goens-register"}, got.Sources)
	assert.Equal(t, "I. Schepp", got.Properties["researcher"])
	assert.Equal(t, []string{"Systematic review of every household 1610-1875."}, []string(got.Notes))
}

// TestStudyMultiFileRoundTrip verifies the multi-file serializer emits a study
// in studies/<id>.glx and reloads it correctly.
func TestStudyMultiFileRoundTrip(t *testing.T) {
	original := &GLXFile{
		Studies: map[string]*Study{
			"study-chiddick-ons": {
				Title:  "Chiddick One Name Study",
				Type:   StudyTypeOneNameStudy,
				Status: StudyStatusActive,
			},
		},
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(original))

	s := NewSerializer(nil)
	files, err := s.SerializeMultiFileToMap(original)
	require.NoError(t, err)

	studyPath := filepath.Join("studies", "study-chiddick-ons.glx")
	require.Contains(t, files, studyPath, "expected study file at %s", studyPath)

	parsed, _, err := s.DeserializeMultiFileFromMap(files)
	require.NoError(t, err)
	require.Len(t, parsed.Studies, 1)
	assert.Equal(t, "Chiddick One Name Study", parsed.Studies["study-chiddick-ons"].Title)
}

// TestStudyValidation covers the reflective validator's handling of Study
// reference and vocabulary fields.
func TestStudyValidation(t *testing.T) {
	t.Run("broken place reference", func(t *testing.T) {
		archive := &GLXFile{
			Studies: map[string]*Study{
				"study-1": {Title: "Test Study", Places: []string{"place-does-not-exist"}},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "studies", err.SourceType)
		assert.Equal(t, "study-1", err.SourceID)
		assert.Equal(t, "places", err.TargetType)
		assert.Equal(t, "place-does-not-exist", err.TargetID)
	})

	t.Run("broken source reference", func(t *testing.T) {
		archive := &GLXFile{
			Studies: map[string]*Study{
				"study-1": {Title: "Test Study", Sources: []string{"source-does-not-exist"}},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "sources", result.Errors[0].TargetType)
	})

	t.Run("unknown study type", func(t *testing.T) {
		archive := &GLXFile{
			Studies: map[string]*Study{
				"study-1": {Title: "Test", Type: "not_a_real_study_type"},
			},
		}
		require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "study_types", err.TargetType)
		assert.Equal(t, "not_a_real_study_type", err.TargetID)
	})

	t.Run("unknown study status", func(t *testing.T) {
		archive := &GLXFile{
			Studies: map[string]*Study{
				"study-1": {Title: "Test", Status: "not_a_real_status"},
			},
		}
		require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "study_statuses", err.TargetType)
		assert.Equal(t, "not_a_real_status", err.TargetID)
	})

	t.Run("minimal study with only title is valid", func(t *testing.T) {
		archive := &GLXFile{
			Studies: map[string]*Study{
				"study-1": {Title: "Minimal Study"},
			},
		}
		require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})
}

// TestStudyStandardVocabulariesPresent confirms study-types and study-statuses
// are included in StandardVocabularies and load into a GLXFile.
func TestStudyStandardVocabulariesPresent(t *testing.T) {
	files := StandardVocabularies()
	assert.Contains(t, files, "study-types.glx")
	assert.Contains(t, files, "study-statuses.glx")

	glx := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))

	assert.Contains(t, glx.StudyTypes, StudyTypeOnePlaceStudy)
	assert.Contains(t, glx.StudyTypes, StudyTypeOneNameStudy)
	assert.Contains(t, glx.StudyStatuses, StudyStatusActive)
	assert.Contains(t, glx.StudyStatuses, StudyStatusCompleted)
}

// TestStudyMergeReportsConflicts confirms the Merge path reports duplicate
// Study IDs the same way it does for other entity types.
func TestStudyMergeReportsConflicts(t *testing.T) {
	dest := &GLXFile{
		Studies: map[string]*Study{"study-1": {Title: "Original"}},
	}
	src := &GLXFile{
		Studies: map[string]*Study{"study-1": {Title: "Conflict"}},
	}

	conflicts, _ := dest.Merge(src)
	require.Len(t, conflicts, 1)
	assert.Contains(t, conflicts[0], "studies")
	assert.Contains(t, conflicts[0], "study-1")
}
