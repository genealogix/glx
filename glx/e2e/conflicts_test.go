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
