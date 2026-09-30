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
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestConflictApproximationFlagParity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  p:
    properties: {name: Mary}
  q:
    properties: {name: Mary}
events:
  birth:
    type: birth
    participants: [{person: p, role: subject}]
assertions:
  a:
    subject: {event: birth}
    property: date
    value: ABT 1850
  b:
    subject: {event: birth}
    property: date
    value: "1853"
`), 0o600))
	cases := [][]string{{"analyze", "--check", "conflicts"}, {"proof", "p", "--question", "birth"}, {"evidence", "birth", "date"}}
	for _, args := range cases {
		base := append(append([]string{}, args...), "--archive", path, "--format", "json")
		normal := runGLX(t, dir, base...)
		require.Equal(t, 0, normal.exitCode, normal.stderr)
		require.Contains(t, normal.stdout, "conflict")
		wider := runGLX(t, dir, append(base, "--approximation-years", "3")...)
		require.Equal(t, 0, wider.exitCode, wider.stderr)
		require.NotContains(t, wider.stdout, `"verdict": "definite"`)
		require.NotContains(t, wider.stdout, `"severity": "high"`)
		invalid := runGLX(t, dir, append(base, "--approximation-years", "-1")...)
		require.NotEqual(t, 0, invalid.exitCode)
	}
	invalid := runGLX(t, dir, "merge-persons", "p", "q", "--archive", path, "--approximation-years", "-1")
	require.NotEqual(t, 0, invalid.exitCode)
}

func TestMergePersonsPreservesDatedHistoryOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  p:
    properties:
      name: Mary
      occupation: {value: miller, date: "1851"}
  q:
    properties:
      name: Mary
      occupation: {value: farmer, date: "1875"}
`), 0o600))
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	dry := runGLX(t, dir, "merge-persons", "p", "q", "--archive", path, "--dry-run")
	require.Equal(t, 0, dry.exitCode, dry.stderr)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after)
	result := runGLX(t, dir, "merge-persons", "p", "q", "--archive", path)
	require.Equal(t, 0, result.exitCode, result.stderr)
	require.NotContains(t, result.stderr, "Conflict")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "miller")
	require.Contains(t, string(data), "farmer")
	require.Contains(t, string(data), "1851")
	require.Contains(t, string(data), "1875")
}

func TestProofSingleFileTemporalParity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-mary:
    properties: {name: Mary Smith}
sources:
  source-register:
    title: Name register
assertions:
  old-name:
    subject: {person: person-mary}
    property: name
    value: Mary Smith
    date: "1850"
    confidence: high
    sources: [source-register]
  later-name:
    subject: {person: person-mary}
    property: name
    value: Mary Jones
    date: "1890"
    confidence: high
    sources: [source-register]
  undated-name:
    subject: {person: person-mary}
    property: name
    value: Mary Green
    confidence: low
    notes: Name recorded without a date.
    sources: [source-register]
`), 0o600))
	var previous string
	for _, archive := range []string{path, dir} {
		result := runGLX(t, dir, "proof", "person-mary", "--question", "identity", "--archive", archive, "--format", "json")
		require.Equal(t, 0, result.exitCode, result.stderr)
		var report struct {
			Conclusion string
			Conflicts  []any
			Undated    []struct{ Value string }
		}
		require.NoError(t, json.Unmarshal([]byte(result.stdout), &report))
		require.Empty(t, report.Conflicts)
		require.NotEqual(t, "CONFLICTED", report.Conclusion)
		require.Len(t, report.Undated, 1)
		require.Equal(t, "Mary Green", report.Undated[0].Value)
		if previous != "" {
			require.JSONEq(t, previous, result.stdout)
		}
		previous = result.stdout
		markdown := runGLX(t, dir, "proof", "person-mary", "--question", "identity", "--archive", archive, "--format", "markdown")
		require.Equal(t, 0, markdown.exitCode, markdown.stderr)
		require.Contains(t, markdown.stdout, "name = Mary Green")
		require.Contains(t, markdown.stdout, "Name recorded without a date.")
	}
}

func TestMergePersonsPreservesDropDisagreementsOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  keep:
    properties:
      name: Mary
      occupation: [{value: farmer, date: "1900"}]
  drop:
    properties:
      name: Mary
      occupation:
        - {value: miller, date: "1851"}
        - {value: driver, date: "1851"}
`), 0o600))
	result := runGLX(t, dir, "merge-persons", "keep", "drop", "--archive", path)
	require.Equal(t, 0, result.exitCode, result.stderr)
	require.NotContains(t, result.stderr, "Conflict")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, occupation := range []string{"farmer", "miller", "driver"} {
		require.Contains(t, string(data), occupation)
	}
}

// This comparison uses the SDK as an external consumer and runs the CLI binary;
// it covers loading/defaults and option forwarding as well as the shared engine.
func TestResearchSDKAndCLIParity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	data := []byte(`persons:
  p:
    properties: {name: Mary}
events:
  birth:
    type: birth
    date: "1850"
    participants: [{person: p, role: subject}]
sources:
  register: {title: Register}
assertions:
  birth-a: {subject: {event: birth}, property: date, value: ABT 1850, sources: [register], confidence: high}
  birth-b: {subject: {event: birth}, property: date, value: "1853", sources: [register], confidence: high}
  name-a: {subject: {person: p}, property: name, value: Mary Smith, date: "1850", sources: [register]}
  name-b: {subject: {person: p}, property: name, value: Mary Jones, date: "1890", sources: [register]}
  name-c: {subject: {person: p}, property: name, value: Mary Green, sources: [register]}
`)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	archive, err := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false}).DeserializeSingleFileBytes(data)
	require.NoError(t, err)
	for _, width := range []int{0, 2, 3} {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			opts := glxlib.ComparisonOptions{ApproximationYears: &width}
			proof, err := glxlib.BuildProof(archive, "p", "birth", glxlib.ProofOptions{Comparison: opts})
			require.NoError(t, err)
			want, err := json.Marshal(proof)
			require.NoError(t, err)
			actual := runGLX(t, dir, "proof", "p", "--question", "birth", "--archive", path, "--format", "json", "--approximation-years", strconv.Itoa(width))
			require.Equal(t, 0, actual.exitCode, actual.stderr)
			require.JSONEq(t, string(want), actual.stdout)
			evidence, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "name", opts)
			require.NoError(t, err)
			want, err = json.Marshal(evidence)
			require.NoError(t, err)
			actual = runGLX(t, dir, "evidence", "p", "name", "--archive", path, "--format", "json", "--approximation-years", strconv.Itoa(width))
			require.Equal(t, 0, actual.exitCode, actual.stderr)
			require.JSONEq(t, string(want), actual.stdout)
			findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{Comparison: opts})
			require.NoError(t, err)
			actual = runGLX(t, dir, "analyze", "--check", "conflicts", "--archive", path, "--format", "json", "--approximation-years", strconv.Itoa(width))
			require.Equal(t, 0, actual.exitCode, actual.stderr)
			var report struct {
				Issues []struct{ Person, Property, Severity string }
			}
			require.NoError(t, json.Unmarshal([]byte(actual.stdout), &report))
			require.Len(t, report.Issues, len(findings))
			for i, finding := range findings {
				require.Equal(t, finding.PersonID, report.Issues[i].Person)
				require.Equal(t, finding.Conflict.Property, report.Issues[i].Property)
				require.Equal(t, finding.Severity, report.Issues[i].Severity)
			}
		})
	}
}

func TestUndatedDisputeRemainsVisible(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  p:
    properties: {name: Mary}
sources:
  register: {title: Register}
assertions:
  disputed-name:
    subject: {person: p}
    property: name
    value: Mary Green
    sources: [register]
    confidence: high
    status: disputed
`), 0o600))
	proof := runGLX(t, dir, "proof", "p", "--question", "identity", "--archive", path, "--format", "json")
	require.Equal(t, 0, proof.exitCode, proof.stderr)
	var report struct {
		Conclusion, Summary string
		Conflicts           []struct {
			Verdict  string
			Definite bool
		}
	}
	require.NoError(t, json.Unmarshal([]byte(proof.stdout), &report))
	require.Equal(t, "POSSIBLE", report.Conclusion)
	require.Contains(t, report.Summary, "Known dispute")
	require.Len(t, report.Conflicts, 1)
	require.Equal(t, "known-dispute", report.Conflicts[0].Verdict)
	require.False(t, report.Conflicts[0].Definite)
	evidence := runGLX(t, dir, "evidence", "p", "name", "--archive", path)
	require.Equal(t, 0, evidence.exitCode, evidence.stderr)
	require.Contains(t, evidence.stdout, "Undated:")
	require.Contains(t, evidence.stdout, "Known dispute: Mary Green")
	require.NotContains(t, evidence.stdout, "Mary Green / Mary Green")
	analysis := runGLX(t, dir, "analyze", "--check", "conflicts", "--archive", path, "--format", "json")
	require.Equal(t, 0, analysis.exitCode, analysis.stderr)
	require.Contains(t, analysis.stdout, `"severity": "low"`)
	require.NotContains(t, analysis.stdout, `"severity": "high"`)
	for _, format := range []string{"text", "markdown"} {
		rendered := runGLX(t, dir, "proof", "p", "--question", "identity", "--archive", path, "--format", format)
		require.Equal(t, 0, rendered.exitCode, rendered.stderr)
		require.Contains(t, rendered.stdout, "dispute")
		require.Contains(t, rendered.stdout, "resolution needed")
	}
}

func TestMergePersonsDoesNotPersistComparisonDefaults(t *testing.T) {
	for _, custom := range []bool{false, true} {
		t.Run(strconv.FormatBool(custom), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			data := `persons:
  p:
    properties:
      name: Pat
      occupation: {value: miller, date: "1851"}
  q:
    properties:
      name: Pat
      occupation: {value: farmer, date: "1875"}
`
			if custom {
				data += `person_properties:
  custom_note: {label: Custom note, value_type: string}
`
			}
			require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
			result := runGLX(t, dir, "merge-persons", "p", "q", "--archive", path)
			require.Equal(t, 0, result.exitCode, result.stderr)
			require.NotContains(t, result.stderr, "Conflict")
			saved, err := os.ReadFile(path)
			require.NoError(t, err)
			archive, err := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false}).DeserializeSingleFileBytes(saved)
			require.NoError(t, err)
			require.Len(t, archive.Persons["p"].Properties["occupation"], 2, "comparison defaults must still preserve dated history")
			require.Nil(t, archive.EventTypes)
			require.Nil(t, archive.RelationshipTypes)
			require.Nil(t, archive.ParticipantRoles)
			require.Nil(t, archive.ConfidenceLevels)
			require.Nil(t, archive.EventProperties)
			if custom {
				require.Len(t, archive.PersonProperties, 1)
				require.Equal(t, "Custom note", archive.PersonProperties["custom_note"].Label)
			} else {
				require.Nil(t, archive.PersonProperties)
			}
		})
	}
}

func TestEvidenceShowsDistinctParticipants(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(`persons:
  p: {properties: {name: Alex}}
  q: {properties: {name: Alex}}
events:
  event: {type: birth}
sources:
  source: {title: Register}
citations:
  citation: {source: source}
assertions:
  parent-p: {subject: {event: event}, participant: {person: p, role: parent}, citations: [citation]}
  parent-q: {subject: {event: event}, participant: {person: q, role: parent}, citations: [citation]}
  witness-p: {subject: {event: event}, participant: {person: p, role: witness}, citations: [citation], status: disputed}
`), 0o600))
	result := runGLX(t, dir, "evidence", "event", "", "--archive", path)
	require.Equal(t, 0, result.exitCode, result.stderr)
	require.Contains(t, result.stdout, "participation and existence")
	for _, label := range []string{"Alex (p) — parent", "Alex (q) — parent", "Alex (p) — witness"} {
		require.Contains(t, result.stdout, label)
	}
	require.Contains(t, result.stdout, "3 reports across 3 values")
	require.Contains(t, result.stdout, "Known dispute: Alex (p) — witness")
	require.NotContains(t, result.stdout, "Best evidence:")
	asJSON := runGLX(t, dir, "evidence", "event", "", "--archive", path, "--format", "json")
	require.Equal(t, 0, asJSON.exitCode, asJSON.stderr)
	var report glxlib.EvidenceReport
	require.NoError(t, json.Unmarshal([]byte(asJSON.stdout), &report))
	require.Len(t, report.Groups, 3)
	require.Len(t, report.Conflicts, 1)
	require.Equal(t, "p", report.Conflicts[0].ParticipantPerson)
	require.Equal(t, "witness", report.Conflicts[0].ParticipantRole)
}
