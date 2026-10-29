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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportEventsArchive combines the reproductions of #1319, #1320 and #1321:
// a two-parent parent_child relationship of a remarried widow, two events of
// types with no GEDCOM tag, and events with witness, godparent and
// household-head participants, plus one event no record can carry.
const exportEventsArchive = `persons:
  person-adam:
    properties: {name: {value: "Adam Call"}, sex: male}
  person-caspar:
    properties: {name: {value: "Caspar Stoehr"}, sex: male}
  person-elizabeth:
    properties: {name: {value: "Elizabeth Starr"}, sex: female}
  person-jasper:
    properties: {name: {value: "Jasper Little"}, sex: male}
  person-lewis:
    properties: {name: {value: "Lewis Little"}, sex: male}
places:
  place-rowan:
    name: Rowan County
relationships:
  rel-marr-a-call:
    type: marriage
    participants:
      - {person: person-adam, role: spouse}
      - {person: person-elizabeth, role: spouse}
  rel-marr-b-little:
    type: marriage
    participants:
      - {person: person-lewis, role: spouse}
      - {person: person-elizabeth, role: spouse}
  rel-pc-jasper:
    type: parent_child
    participants:
      - {person: person-lewis, role: parent}
      - {person: person-elizabeth, role: parent}
      - {person: person-jasper, role: child}
events:
  ev-tax-1796:
    type: taxation
    date: "1796"
    place: place-rowan
    participants: [{person: person-lewis, role: principal}]
  ev-census-1800:
    type: census
    date: "1800-08-04"
    participants: [{person: person-lewis, role: household_head}]
  ev-will-1808:
    type: will
    date: "1808-12-19"
    participants:
      - {person: person-caspar, role: principal}
      - {person: person-lewis, role: witness}
  ev-bapt-1789:
    type: baptism
    date: "1789-10-11"
    participants:
      - {person: person-jasper, role: principal}
      - {person: person-caspar, role: godparent}
  ev-deed-witnessed:
    type: will
    date: "1810"
    participants: [{person: person-adam, role: witness}]
`

// gedcomRecord returns the level-0 record with the given XREF and the lines
// below it.
func gedcomRecord(t *testing.T, ged, xref string) string {
	t.Helper()

	start := strings.Index(ged, "0 "+xref+" ")
	require.GreaterOrEqual(t, start, 0, "no record %s in the exported file", xref)
	rest := ged[start:]
	if end := strings.Index(rest[1:], "\n0 "); end >= 0 {
		return rest[:end+1]
	}

	return rest
}

// The GEDCOM export used to file a two-parent child under its mother's first
// marriage (#1319), drop events of a type with no GEDCOM tag (#1320), and drop
// every non-principal participant (#1321), all without a warning.
func TestExport_GEDCOMParentsEventsAndAssociations(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(exportEventsArchive), 0o600))

	res := runGLX(t, dir, "export", "archive.glx", "-f", "70", "-o", "out.ged")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	data, err := os.ReadFile(filepath.Join(dir, "out.ged"))
	require.NoError(t, err)
	ged := strings.ReplaceAll(string(data), "\r\n", "\n")

	// XREFs follow sorted person IDs: adam @I1@, caspar @I2@, elizabeth @I3@,
	// jasper @I4@, lewis @I5@.
	assert.Equal(t, "0 @F1@ FAM\n1 HUSB @I1@\n1 WIFE @I3@", gedcomRecord(t, ged, "@F1@"),
		"the first marriage keeps no child")
	assert.Equal(t, "0 @F2@ FAM\n1 HUSB @I5@\n1 WIFE @I3@\n1 CHIL @I4@", gedcomRecord(t, ged, "@F2@"))

	lewis := gedcomRecord(t, ged, "@I5@")
	assert.Contains(t, lewis, "1 EVEN\n2 TYPE Taxation\n2 DATE 1796\n2 PLAC Rowan County\n")
	assert.Contains(t, lewis, "1 CENS\n2 DATE 4 AUG 1800\n")

	assert.Contains(t, gedcomRecord(t, ged, "@I2@"), "1 WILL\n2 DATE 19 DEC 1808\n2 ASSO @I5@\n3 ROLE WITN")
	assert.Contains(t, gedcomRecord(t, ged, "@I4@"), "1 BAPM\n2 DATE 11 OCT 1789\n2 ASSO @I2@\n3 ROLE GODP\n")

	// The one event no record can carry is named, not dropped silently.
	assert.Contains(t, res.stdout, "[events ev-deed-witnessed] event has no principal or household participant (roles: witness)")
	assert.NotContains(t, gedcomRecord(t, ged, "@I1@"), "WILL")
}

func TestExport_OverlappingParentFamilies(t *testing.T) {
	for _, version := range []string{"551", "70"} {
		t.Run(version, func(t *testing.T) {
			dir := t.TempDir()
			fixture, err := os.ReadFile(filepath.Join("..", "testdata", "valid", "overlapping-parent-families.glx"))
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), fixture, 0o600))
			validation := runGLX(t, dir, "validate", "archive.glx")
			require.Equal(t, 0, validation.exitCode, validation.stdout+validation.stderr)
			res := runGLX(t, dir, "export", "archive.glx", "-f", version, "-o", "out.ged")
			require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
			data, err := os.ReadFile(filepath.Join(dir, "out.ged"))
			require.NoError(t, err)
			ged := strings.ReplaceAll(string(data), "\r\n", "\n")
			assert.NotContains(t, gedcomRecord(t, ged, "@F1@"), "CHIL", "unrelated spouse must not become a parent")
			assert.Contains(t, gedcomRecord(t, ged, "@F2@"), "1 CHIL @I1@")
			assert.Contains(t, gedcomRecord(t, ged, "@F3@"), "1 CHIL @I1@")
			assert.Contains(t, gedcomRecord(t, ged, "@I1@"), "1 FAMC @F2@\n1 FAMC @F3@")
		})
	}
}

func TestExport_GEDCOM551UsesStandardAssociations(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(exportEventsArchive), 0o600))
	res := runGLX(t, dir, "export", "archive.glx", "-f", "551", "-o", "out.ged")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	data, err := os.ReadFile(filepath.Join(dir, "out.ged"))
	require.NoError(t, err)
	ged := strings.ReplaceAll(string(data), "\r\n", "\n")
	assert.Contains(t, gedcomRecord(t, ged, "@I2@"), "1 ASSO @I5@\n2 RELA Witness\n2 NOTE Event: Will (WILL; ev-will-1808); date: 1808-12-19")
	assert.Contains(t, gedcomRecord(t, ged, "@I4@"), "1 ASSO @I2@\n2 RELA Godparent\n2 NOTE Event: Baptism (BAPM; ev-bapt-1789); date: 1789-10-11")
	var root string
	for line := range strings.SplitSeq(ged, "\n") {
		if strings.HasPrefix(line, "0 ") {
			root = line
		}
		if strings.Contains(line, " ASSO ") {
			assert.True(t, strings.HasSuffix(root, " INDI"), "ASSO must belong to INDI: %s", root)
			assert.True(t, strings.HasPrefix(line, "1 ASSO "), "ASSO must be person-level: %s", line)
		}
	}
	assert.NotContains(t, ged, "_ASSO")
	assert.NotContains(t, ged, " ROLE ")
}
