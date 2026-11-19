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

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A predicted person (#191) validates, is tracked by analyze instead of being
// reported as full of gaps, and can be found with query and summary.
func TestPersonStatus_ValidateAnalyzeQuerySummary(t *testing.T) {
	archive := copyExample(t, "complete-family")
	placeholder := []byte(`persons:
  person-unknown-ggp-8:
    properties:
      name:
        value: "Unknown Great-Grandparent 8"
      status: predicted
    notes: "Predicted from DNA cluster 8."
`)
	require.NoError(t, os.WriteFile(filepath.Join(archive, "persons", "person-unknown-ggp-8.glx"), placeholder, 0o600))
	assertArchiveValid(t, archive)

	a := runGLX(t, archive, "analyze")
	require.Equal(t, 0, a.exitCode, a.stdout+a.stderr)
	assert.Contains(t, a.stdout, "Unknown Great-Grandparent 8 — predicted placeholder, not yet identified")
	assert.Contains(t, a.stdout, "Research placeholders: 1 open, 0 resolved")
	assert.NotContains(t, a.stdout, "Unknown Great-Grandparent 8 — no ")

	q := runGLX(t, archive, "query", "persons", "--status", "predicted")
	require.Equal(t, 0, q.exitCode, q.stdout+q.stderr)
	assert.Contains(t, q.stdout, "person-unknown-ggp-8")
	assert.Contains(t, q.stdout, "[predicted]")
	assert.Contains(t, q.stdout, "1 person(s) found")

	s := runGLX(t, archive, "summary", "person-unknown-ggp-8")
	require.Equal(t, 0, s.exitCode, s.stdout+s.stderr)
	assert.Regexp(t, `Status:\s+predicted`, s.stdout)
}

// Resolving a placeholder by merging it (#191) must leave no unresolved status on
// the survivor on disk, whichever side is kept.
func TestPersonStatus_MergeResolvesPlaceholder(t *testing.T) {
	tests := []struct {
		name       string
		keep, drop string
	}{
		{"keep identified", "person-john-smith-1850", "person-unknown-ggp-8"},
		{"keep placeholder", "person-unknown-ggp-8", "person-john-smith-1850"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := copyExample(t, "complete-family")
			placeholder := []byte(`persons:
  person-unknown-ggp-8:
    properties:
      status: predicted
    notes: "Predicted from DNA cluster 8."
`)
			require.NoError(t, os.WriteFile(filepath.Join(archive, "persons", "person-unknown-ggp-8.glx"), placeholder, 0o600))

			m := runGLX(t, archive, "merge-persons", tt.keep, tt.drop, "--yes")
			require.Equal(t, 0, m.exitCode, m.stdout+m.stderr)
			assertArchiveValid(t, archive)

			q := runGLX(t, archive, "query", "persons", "--status", "predicted")
			require.Equal(t, 0, q.exitCode, q.stdout+q.stderr)
			assert.Contains(t, q.stdout, "0 person(s) found")
		})
	}
}
