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
	"os"
	"path/filepath"
	"testing"

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
