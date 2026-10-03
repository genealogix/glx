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
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roleArchive is the minimal shape of #1329 and #1330: Lewis is a witness at
// his father-in-law's probate and a grantor of a deed in North Carolina,
// both while his dated residence places him in Indiana. Every fact is
// asserted on an event, never on a person or place (#713).
const roleArchive = `participant_roles:
  principal: {label: Principal}
  witness: {label: Witness}
  grantor: {label: Grantor}
places:
  place-us: {name: United States, type: country}
  place-nc: {name: North Carolina, type: state, parent: place-us}
  place-in: {name: Indiana Territory, type: state, parent: place-us}
  place-rowan: {name: Rowan County, type: county, parent: place-nc}
  place-dearborn: {name: Dearborn County, type: county, parent: place-in}
persons:
  person-lewis:
    properties:
      name: {value: Lewis Little}
      residence:
        - {value: place-rowan, date: "FROM 1790 TO 1808"}
        - {value: place-dearborn, date: "FROM 1809 TO 1810"}
  person-caspar: {properties: {name: {value: Caspar Stoehr}}}
sources:
  src-estate-files: {title: "North Carolina, Estate Files"}
events:
  ev-birth-lewis:
    type: birth
    date: "1765"
    place: place-rowan
    participants: [{person: person-lewis, role: principal}]
  ev-will-father-in-law:
    type: probate
    date: "1810-04-03"
    place: place-rowan
    participants:
      - {person: person-caspar, role: principal}
      - {person: person-lewis, role: witness}
  ev-deed-1810:
    type: event
    title: Deed
    date: "1810-10-19"
    place: place-rowan
    participants: [{person: person-lewis, role: grantor}]
assertions:
  a-will-date:
    subject: {event: ev-will-father-in-law}
    property: date
    value: "1810-04-03"
    sources: [src-estate-files]
    confidence: high
  a-birth-place:
    subject: {event: ev-birth-lewis}
    property: place
    value: place-rowan
    sources: [src-estate-files]
    confidence: high
`

func writeRoleArchive(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(roleArchive), 0o600))

	return dir
}

// TestCoverage_WitnessProbateIsNotOwnRecord covers #1329.
func TestCoverage_WitnessProbateIsNotOwnRecord(t *testing.T) {
	dir := writeRoleArchive(t)

	res := runGLX(t, dir, "coverage", "person-lewis", "--archive", "archive.glx")
	require.Equal(t, 0, res.exitCode, res.stderr)

	assert.Contains(t, res.stdout, "[ ] Probate/will")
	assert.NotContains(t, res.stdout, "via ev-will-father-in-law")
	assert.Contains(t, res.stdout, "Probate of Caspar Stoehr (witness) -- ev-will-father-in-law")
}

// TestMigrations_RemoteDeedIsNotAMove covers #1330.
func TestMigrations_RemoteDeedIsNotAMove(t *testing.T) {
	dir := writeRoleArchive(t)

	res := runGLX(t, dir, "migrations", "person-lewis", "--archive", "archive.glx")
	require.Equal(t, 0, res.exitCode, res.stderr)

	assert.Contains(t, res.stdout, `(Deed) [not counted as a move: role "grantor" does not imply presence]`)
	assert.Contains(t, res.stdout, "[not counted as a move: inside residence FROM 1809 TO 1810")
	assert.Contains(t, res.stdout, "North Carolina → Indiana")
	assert.NotContains(t, res.stdout, "Indiana → North Carolina")
}

// TestStatsAndReport_EvidenceCoverage covers #713.
func TestStatsAndReport_EvidenceCoverage(t *testing.T) {
	dir := writeRoleArchive(t)

	stats := runGLX(t, dir, "stats", "archive.glx")
	require.Equal(t, 0, stats.exitCode, stats.stderr)
	assert.Contains(t, stats.stdout, "Direct assertion references")
	assert.Contains(t, stats.stdout, "Evidence coverage")
	assert.Contains(t, stats.stdout, "Persons         2/2  (100.0%)")
	assert.Contains(t, stats.stdout, "Places          3/5  (60.0%)")

	report := runGLX(t, dir, "validate", "archive.glx", "--report")
	require.Equal(t, 0, report.exitCode, report.stderr)
	assert.NotContains(t, report.stdout, "Persons (")
	assert.Contains(t, report.stdout, "Events (1): ev-deed-1810")
}

func TestStatsAndValidation_NullAssertions(t *testing.T) {
	dir := t.TempDir()
	data := "assertions:\n  assertion-null: null\n  assertion-valid:\n    subject: {person: p}\n    confidence: high\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(data), 0o600))
	stats := runGLX(t, dir, "stats", "archive.glx")
	require.Equal(t, 0, stats.exitCode, stats.stderr)
	assert.Contains(t, stats.stdout, "100.0%")
	report := runGLX(t, dir, "validate", "archive.glx", "--report")
	require.Equal(t, 1, report.exitCode, report.stderr)
	assert.Contains(t, report.stderr, "got null, want object")
	assert.NotContains(t, report.stderr, "panic:")
}

// Coverage and the gaps in proof must use the same role-aware SDK result for
// single-file and directory loading, including custom presence overrides.
func TestRoleAwareCoverageSDKCLIParity(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmtBool(override), func(t *testing.T) {
			data := roleArchive
			if override {
				data = strings.Replace(data, "principal: {label: Principal}", "principal: {label: Principal, implies_presence: false}", 1)
			}
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
			archive, err := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false}).DeserializeSingleFileBytes([]byte(data))
			require.NoError(t, err)
			coverage, err := glxlib.BuildCoverage(archive, "person-lewis", glxlib.CoverageOptions{})
			require.NoError(t, err)
			want, err := json.Marshal(coverage)
			require.NoError(t, err)
			proof, err := glxlib.BuildProof(archive, "person-lewis", glxlib.QuestionBirth, glxlib.ProofOptions{})
			require.NoError(t, err)
			proofJSON, err := json.Marshal(proof)
			require.NoError(t, err)
			for _, archivePath := range []string{path, dir} {
				actual := runGLX(t, dir, "coverage", "person-lewis", "--archive", archivePath, "--json")
				require.Equal(t, 0, actual.exitCode, actual.stderr)
				require.JSONEq(t, string(want), actual.stdout)
				actual = runGLX(t, dir, "proof", "person-lewis", "--question", "birth", "--archive", archivePath, "--format", "json")
				require.Equal(t, 0, actual.exitCode, actual.stderr)
				require.JSONEq(t, string(proofJSON), actual.stdout)
			}
		})
	}
}

func fmtBool(value bool) string {
	if value {
		return "override"
	}

	return "defaults"
}
