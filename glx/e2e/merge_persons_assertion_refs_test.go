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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMergePersons_AssertionPropertyReferences(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			data := `persons:
  keep: {}
  drop: {}
  third:
    properties:
      supporting_claims: [identity, retained]
      sole_claim: identity
source_types:
  book: {label: Book}
sources:
  source: {type: book, title: Register}
relationships:
  candidate:
    type: possibly_same_person
    participants: [{person: keep}, {person: drop}]
assertions:
  identity:
    subject: {relationship: candidate}
    sources: [source]
    notes: Identity claim
  dependent-retained:
    subject: {person: third}
    property: sole_claim
    value: retained
    sources: [source]
  retained:
    subject: {person: keep}
    sources: [source]
    notes: Retained claim
person_properties:
  supporting_claims:
    label: Supporting claims
    reference_type: assertions
    multi_value: true
  sole_claim:
    label: Sole claim
    reference_type: assertions
`
			// Every entity property vocabulary uses its own schema.
			var definitions strings.Builder
			for _, entity := range []string{"event", "relationship", "place", "source", "citation", "repository", "media"} {
				definitions.WriteString(entity + "_properties:\n  supporting_claim:\n    label: Supporting claim\n    reference_type: assertions\n")
			}
			data += definitions.String()
			require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
			validation := runGLX(t, dir, "validate", path)
			require.Equal(t, 0, validation.exitCode, validation.stdout+validation.stderr)
			archive := path
			if directory {
				archive = dir
			}
			merged := runGLX(t, dir, "merge-persons", "keep", "drop", "--archive", archive, "--yes")
			require.Equal(t, 0, merged.exitCode, merged.stdout+merged.stderr)
			output, err := os.ReadFile(path)
			require.NoError(t, err)
			serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{})
			actual, err := serializer.DeserializeSingleFileBytes(output)
			require.NoError(t, err)
			assert.Equal(t, []any{"retained"}, actual.Persons["third"].Properties["supporting_claims"])
			assert.Equal(t, "retained", actual.Assertions["dependent-retained"].Value)
			assert.NotContains(t, string(output), "sole_claim: identity")
			assert.NotContains(t, string(output), "  identity:")
			assert.Contains(t, string(output), "  retained:")
			validation = runGLX(t, dir, "validate", archive)
			require.Equal(t, 0, validation.exitCode, validation.stdout+validation.stderr)
		})
	}
}

func TestMergePersons_AssertionValueCascade(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			data := `persons:
  keep: {}
  drop: {}
  third:
    properties:
      supporting_claims: [second-hop, retained]
source_types:
  book: {label: Book}
sources:
  source: {type: book, title: Register}
relationship_types:
  possibly_same_person:
    label: Possibly same person
relationships:
  candidate:
    type: possibly_same_person
    participants: [{person: keep}, {person: drop}]
assertions:
  identity:
    subject: {relationship: candidate}
    sources: [source]
    notes: Identity claim
  first-hop:
    subject: {person: third}
    property: sole_claim
    value: identity
    sources: [source]
  second-hop:
    subject: {person: third}
    property: sole_claim
    value: first-hop
    sources: [source]
  dependent-retained:
    subject: {person: third}
    property: sole_claim
    value: retained
    sources: [source]
  retained:
    subject: {person: keep}
    sources: [source]
    notes: Retained claim
person_properties:
  supporting_claims:
    label: Supporting claims
    reference_type: assertions
    multi_value: true
  sole_claim:
    label: Sole claim
    reference_type: assertions
research_logs:
  investigation:
    objective: Follow surviving evidence
    leads:
      - description: Preserve the lead and surviving claims
        assertions: [identity, first-hop, second-hop, retained, dependent-retained]
`
			require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
			archive := path
			if directory {
				archive = dir
			}
			validation := runGLX(t, dir, "validate", archive)
			require.Equal(t, 0, validation.exitCode, validation.stdout+validation.stderr)
			before := snapshotTree(t, dir)
			preview := runGLX(t, dir, "merge-persons", "keep", "drop", "--archive", archive, "--dry-run", "--yes")
			require.Equal(t, 0, preview.exitCode, preview.stdout+preview.stderr)
			assertTreeUnchanged(t, before, dir)
			for _, id := range []string{"identity", "first-hop", "second-hop"} {
				assert.Contains(t, preview.stdout, "Assertion removed: "+id)
			}
			merged := runGLX(t, dir, "merge-persons", "keep", "drop", "--archive", archive, "--yes")
			require.Equal(t, 0, merged.exitCode, merged.stdout+merged.stderr)
			output, err := os.ReadFile(path)
			require.NoError(t, err)
			actual, err := glxlib.NewSerializer(&glxlib.SerializerOptions{}).DeserializeSingleFileBytes(output)
			require.NoError(t, err)
			require.Len(t, actual.Assertions, 2)
			assert.Equal(t, "retained", actual.Assertions["dependent-retained"].Value)
			assert.Equal(t, []any{"retained"}, actual.Persons["third"].Properties["supporting_claims"])
			require.Len(t, actual.ResearchLogs["investigation"].Leads, 1)
			assert.Equal(t, []string{"retained", "dependent-retained"}, actual.ResearchLogs["investigation"].Leads[0].Assertions)
			validation = runGLX(t, dir, "validate", archive)
			require.Equal(t, 0, validation.exitCode, validation.stdout+validation.stderr)
		})
	}
}
