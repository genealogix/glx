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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMigrateConfidenceDisputedToStatus_CleanConversion(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:    glxlib.EntityRef{Person: "person-1"},
				Property:   "name",
				Value:      "Mary Smith",
				Confidence: "disputed",
			},
		},
	}

	var warnBuf bytes.Buffer
	report := migrateConfidenceDisputed(archive, &warnBuf)

	assert.Equal(t, 1, report.ConfidenceDisputedConverted)
	assert.Equal(t, 0, report.ConfidenceDisputedStatusConflicts)
	assert.Empty(t, archive.Assertions["a-1"].Confidence)
	assert.Equal(t, "disputed", archive.Assertions["a-1"].Status)
	assert.Empty(t, warnBuf.String(), "no warnings expected for clean conversion")
}

func TestMigrateConfidenceDisputedToStatus_PreservesNonDisputedConfidence(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-high":   {Subject: glxlib.EntityRef{Person: "person-1"}, Confidence: "high"},
			"a-medium": {Subject: glxlib.EntityRef{Person: "person-1"}, Confidence: "medium"},
			"a-low":    {Subject: glxlib.EntityRef{Person: "person-1"}, Confidence: "low"},
			"a-empty":  {Subject: glxlib.EntityRef{Person: "person-1"}},
		},
	}

	report := migrateConfidenceDisputed(archive, &bytes.Buffer{})

	assert.Equal(t, 0, report.ConfidenceDisputedConverted)
	assert.Equal(t, "high", archive.Assertions["a-high"].Confidence)
	assert.Equal(t, "medium", archive.Assertions["a-medium"].Confidence)
	assert.Equal(t, "low", archive.Assertions["a-low"].Confidence)
	assert.Empty(t, archive.Assertions["a-empty"].Confidence)
	for id, a := range archive.Assertions {
		assert.Empty(t, a.Status, "status should remain empty for %s", id)
	}
}

func TestMigrateConfidenceDisputedToStatus_StatusConflictPreservesExistingStatus(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-proven": {
				Subject:    glxlib.EntityRef{Person: "person-1"},
				Confidence: "disputed",
				Status:     "proven",
			},
			"a-speculative": {
				Subject:    glxlib.EntityRef{Person: "person-2"},
				Confidence: "disputed",
				Status:     "speculative",
			},
		},
	}

	var warnBuf bytes.Buffer
	report := migrateConfidenceDisputed(archive, &warnBuf)

	assert.Equal(t, 2, report.ConfidenceDisputedConverted)
	assert.Equal(t, 2, report.ConfidenceDisputedStatusConflicts)
	assert.Empty(t, archive.Assertions["a-proven"].Confidence, "confidence should be cleared")
	assert.Equal(t, "proven", archive.Assertions["a-proven"].Status, "existing status preserved")
	assert.Empty(t, archive.Assertions["a-speculative"].Confidence)
	assert.Equal(t, "speculative", archive.Assertions["a-speculative"].Status)

	warnings := warnBuf.String()
	assert.Contains(t, warnings, "a-proven")
	assert.Contains(t, warnings, "proven")
	assert.Contains(t, warnings, "a-speculative")
	assert.Contains(t, warnings, "speculative")
	require.Equal(t, 2, strings.Count(warnings, "Warning:"), "one warning per conflict")
}

func TestMigrateConfidenceDisputedToStatus_IdempotentWhenStatusAlreadyDisputed(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:    glxlib.EntityRef{Person: "person-1"},
				Confidence: "disputed",
				Status:     "disputed",
			},
		},
	}

	var warnBuf bytes.Buffer
	report := migrateConfidenceDisputed(archive, &warnBuf)

	assert.Equal(t, 1, report.ConfidenceDisputedConverted)
	assert.Equal(t, 0, report.ConfidenceDisputedStatusConflicts, "idempotent re-run is not a conflict")
	assert.Empty(t, archive.Assertions["a-1"].Confidence)
	assert.Equal(t, "disputed", archive.Assertions["a-1"].Status)
	assert.Empty(t, warnBuf.String(), "no warning when status already records disputed")

	// A second run with the same archive must be a complete no-op.
	report = migrateConfidenceDisputed(archive, &warnBuf)
	assert.Equal(t, 0, report.ConfidenceDisputedConverted, "post-migration archive yields no work")
}

func TestMigrateConfidenceDisputedToStatus_NoOpOnEmptyArchive(t *testing.T) {
	report := migrateConfidenceDisputed(&glxlib.GLXFile{}, &bytes.Buffer{})
	assert.Equal(t, 0, report.ConfidenceDisputedConverted)
	assert.Equal(t, 0, report.ConfidenceDisputedStatusConflicts)
}

func TestMigrateConfidenceDisputedToStatus_NilWarnOutSafe(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-1": {
				Subject:    glxlib.EntityRef{Person: "person-1"},
				Confidence: "disputed",
				Status:     "proven",
			},
		},
	}

	report := migrateConfidenceDisputed(archive, nil)
	assert.Equal(t, 1, report.ConfidenceDisputedConverted)
	assert.Equal(t, 1, report.ConfidenceDisputedStatusConflicts)
}

// TestMigrateArchive_ConfidenceDisputedToStatus_SingleFileRoundTrip exercises
// the full load -> migrate -> write -> reload pipeline against the new
// --confidence-disputed-to-status flag, mirroring the gender-rename round-trip
// test. The fixture includes a clean conversion, a status conflict (to drive
// the conflict-print branch in migrateArchive), and an already-disputed
// assertion (idempotent path).
func TestMigrateArchive_ConfidenceDisputedToStatus_SingleFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")

	preMigration := `metadata:
  glx_version: "1.0"
assertions:
  a-clean:
    subject:
      person: person-1
    property: name
    value: "Mary Smith"
    confidence: disputed
  a-conflict:
    subject:
      person: person-2
    property: birth_date
    value: "1850"
    confidence: disputed
    status: proven
  a-already-disputed:
    subject:
      person: person-3
    property: death_date
    value: "1900"
    confidence: disputed
    status: disputed
`
	require.NoError(t, os.WriteFile(archivePath, []byte(preMigration), 0o600))

	t.Cleanup(func() { migrateConfidenceDisputedToStatus = false })
	migrateConfidenceDisputedToStatus = true
	require.NoError(t, migrateArchive(archivePath))

	written, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	writtenStr := string(written)
	assert.NotContains(t, writtenStr, "confidence: disputed",
		"all confidence: disputed entries should be cleared post-migration")

	reloaded, err := readSingleFileArchive(archivePath, false)
	require.NoError(t, err)

	require.NotNil(t, reloaded.Assertions["a-clean"])
	assert.Empty(t, reloaded.Assertions["a-clean"].Confidence)
	assert.Equal(t, "disputed", reloaded.Assertions["a-clean"].Status)

	require.NotNil(t, reloaded.Assertions["a-conflict"])
	assert.Empty(t, reloaded.Assertions["a-conflict"].Confidence)
	assert.Equal(t, "proven", reloaded.Assertions["a-conflict"].Status,
		"existing non-disputed status must be preserved")

	require.NotNil(t, reloaded.Assertions["a-already-disputed"])
	assert.Empty(t, reloaded.Assertions["a-already-disputed"].Confidence)
	assert.Equal(t, "disputed", reloaded.Assertions["a-already-disputed"].Status)

	// Idempotency at the fs boundary: a second invocation must not corrupt
	// the now-migrated file.
	require.NoError(t, migrateArchive(archivePath))
	second, err := os.ReadFile(archivePath)
	require.NoError(t, err)
	assert.NotContains(t, string(second), "confidence: disputed")
	assert.Contains(t, string(second), "status: disputed")
	assert.Contains(t, string(second), "status: proven")
}
