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
	"testing"

	"github.com/stretchr/testify/require"
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
