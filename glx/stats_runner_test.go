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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShowStats_MultiFileArchive(t *testing.T) {
	err := showStats("../docs/examples/basic-family")
	require.NoError(t, err)
}

func TestShowStats_SingleFileArchive(t *testing.T) {
	err := showStats("../docs/examples/assertion-workflow/archive.glx")
	require.NoError(t, err)
}

func TestShowStats_NonexistentPath(t *testing.T) {
	err := showStats("/nonexistent/path")
	require.Error(t, err)
}

func TestShowStats_WithAssertions(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a persons file
	personsDir := filepath.Join(tmpDir, "persons")
	require.NoError(t, os.MkdirAll(personsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(personsDir, "person-p1.glx"), []byte(`persons:
  p1:
    properties:
      primary_name: "Jane Doe"
`), 0o644))

	// Write a sources file
	sourcesDir := filepath.Join(tmpDir, "sources")
	require.NoError(t, os.MkdirAll(sourcesDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sourcesDir, "source-s1.glx"), []byte(`sources:
  s1:
    title: "Birth Certificate"
`), 0o644))

	// Write assertions with confidence levels
	assertionsDir := filepath.Join(tmpDir, "assertions")
	require.NoError(t, os.MkdirAll(assertionsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assertionsDir, "assertion-a1.glx"), []byte(`assertions:
  a1:
    subject:
      person: p1
    property: primary_name
    value: "Jane Doe"
    confidence: high
    sources:
      - s1
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(assertionsDir, "assertion-a2.glx"), []byte(`assertions:
  a2:
    subject:
      person: p1
    property: birth_date
    value: "1900-01-01"
    confidence: medium
    sources:
      - s1
`), 0o644))

	err := showStats(tmpDir)
	require.NoError(t, err)
}

func TestShowStats_OutputContent(t *testing.T) {
	// Capture stdout to verify output format
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	tmpDir := t.TempDir()

	personsDir := filepath.Join(tmpDir, "persons")
	require.NoError(t, os.MkdirAll(personsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(personsDir, "person-p1.glx"), []byte(`persons:
  p1:
    properties:
      name:
        value: "Test Person"
`), 0o644))

	assertionsDir := filepath.Join(tmpDir, "assertions")
	require.NoError(t, os.MkdirAll(assertionsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(assertionsDir, "assertion-a1.glx"), []byte(`assertions:
  a1:
    subject:
      person: p1
    property: name
    value: "Test Person"
    confidence: high
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(assertionsDir, "assertion-a2.glx"), []byte(`assertions:
  a2:
    subject:
      person: p1
    property: born_on
    value: "1900"
`), 0o644))

	require.NoError(t, showStats(tmpDir))

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout
	output := string(out)

	// Verify entity counts section
	assert.Contains(t, output, "Persons:")
	assert.Contains(t, output, "Assertions:")

	// Verify confidence distribution shows high before (unset)
	highIdx := strings.Index(output, "high")
	unsetIdx := strings.Index(output, "(unset)")
	require.True(t, highIdx >= 0, "output should contain 'high'")
	require.True(t, unsetIdx >= 0, "output should contain '(unset)'")
	assert.True(t, highIdx < unsetIdx, "(unset) should appear after high in output")

	// Verify coverage section
	assert.Contains(t, output, "Entity coverage")
	assert.Contains(t, output, "1/1")
}
