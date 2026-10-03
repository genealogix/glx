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
)

// writeSingleFileArchive writes content as archive.glx in a fresh directory
// and returns the directory.
func writeSingleFileArchive(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "archive.glx"), []byte(content), 0o644))

	return dir
}

// runAnalyze runs `glx analyze` against the archive in dir and returns stdout.
func runAnalyze(t *testing.T, dir string, args ...string) string {
	t.Helper()
	res := runGLX(t, dir, append([]string{"analyze", "--archive", dir}, args...)...)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	return res.stdout
}

// A citation recorded by a research-log search is not orphaned (#1323).
func TestAnalyze_ResearchLogCitationNotOrphaned(t *testing.T) {
	out := runAnalyze(t, writeSingleFileArchive(t, researchLogArchive), "--check", "evidence")
	assert.NotContains(t, out, "cit-fag-search")
	assert.NotContains(t, out, "src-study", "a study's in-scope source is referenced")
	assert.Contains(t, out, "cit-orphan", "a citation nothing refers to is still reported")
}

// A birthplace that encloses the siblings' is an info prompt, not an outlier
// (#1324).
func TestAnalyze_CoarserSiblingBirthplaceIsInfo(t *testing.T) {
	out := runAnalyze(t, writeSingleFileArchive(t, coarserBirthplaceArchive), "--check", "consistency")
	assert.NotContains(t, out, "MEDIUM person-james")
	assert.Contains(t, out, "birthplace less specific than siblings'")
}

// A dated, placed marriage license satisfies the marriage gap check, with a
// softer prompt to find the ceremony record (#1338).
func TestAnalyze_LicenseOnlyMarriageIsInfo(t *testing.T) {
	out := runAnalyze(t, writeSingleFileArchive(t, licenseOnlyArchive), "--check", "gaps")
	assert.NotContains(t, out, "no marriage event for")
	assert.Contains(t, out, "marriage to Elizabeth Starr known from a marriage license only")
}

// The 1810 census is lost for Indiana Territory: analyze does not suggest it
// and coverage does not count it as missing (#1333).
func TestAnalyzeAndCoverage_LostTerritoryCensus(t *testing.T) {
	dir := writeSingleFileArchive(t, lostCensusArchive)

	out := runAnalyze(t, dir, "--check", "suggestions")
	assert.NotContains(t, out, "search 1810 US census")
	assert.Contains(t, out, "search 1820 US census")

	res := runGLX(t, dir, "coverage", "person-lewis-little", "--archive", dir)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.NotContains(t, res.stdout, "1810 US Census")
	assert.Contains(t, res.stdout, "1820 US Census")

	res = runGLX(t, dir, "proof", "person-lewis-little", "--question", "identity", "--archive", dir)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.NotContains(t, res.stdout, "1810 US Census")
	assert.Contains(t, res.stdout, "1820 US Census")
}

func TestAncestors_LostTerritoryCensus(t *testing.T) {
	// A child in 1810 would otherwise get a parents' household suggestion.
	dir := writeSingleFileArchive(t, strings.ReplaceAll(lostCensusArchive, "1756", "1803"))
	res := runGLX(t, dir, "ancestors", "person-lewis-little", "--archive", dir)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.NotContains(t, res.stdout, "1810 US census")
	assert.Contains(t, res.stdout, "1820 US census")
}

func TestCensusLosses_FederalClassifications(t *testing.T) {
	cases := []struct {
		year      int
		name      string
		placeType string
		lost      bool
		partial   bool
	}{
		{1800, "Georgia", "state", true, false},
		{1810, "Ohio", "state", true, false},
		{1810, "Michigan Territory", "territory", true, false},
		{1810, "Michigan", "state", true, false},
		{1820, "New Jersey", "state", true, false},
		{1800, "Ohio", "state", false, true},
		{1810, "Tennessee", "state", false, true},
		{1810, "Illinois Territory", "territory", false, true},
		// Unknown federal survival keeps the search without claiming partial loss.
		{1820, "Alabama", "state", false, false},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d %s", tc.year, tc.name), func(t *testing.T) {
			// A child at the target census also exercises ancestor household notes.
			dir := writeSingleFileArchive(t, fmt.Sprintf(censusClassificationArchive, tc.name, tc.placeType, tc.year-7, tc.year))
			analyze := runAnalyze(t, dir, "--check", "suggestions")
			coverage := runGLX(t, dir, "coverage", "person-subject", "--archive", dir)
			proof := runGLX(t, dir, "proof", "person-subject", "--question", "identity", "--archive", dir)
			ancestors := runGLX(t, dir, "ancestors", "person-subject", "--archive", dir)
			for _, res := range []result{coverage, proof, ancestors} {
				require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
			}
			outputs := []struct {
				text  string
				label string
			}{
				{analyze, fmt.Sprintf("search %d US census", tc.year)},
				{coverage.stdout, fmt.Sprintf("%d US Census", tc.year)},
				{proof.stdout, fmt.Sprintf("%d US Census", tc.year)},
				{ancestors.stdout, fmt.Sprintf("%d US census", tc.year)},
			}
			for _, out := range outputs {
				if tc.lost {
					assert.NotContains(t, out.text, out.label)
				} else {
					assert.Contains(t, out.text, out.label)
				}
				if tc.partial {
					assert.Contains(t, out.text, "schedules mostly lost for "+tc.name)
				} else {
					assert.NotContains(t, out.text, "schedules mostly lost for "+tc.name)
				}
			}
			assert.Contains(t, analyze, "search 1830 US census", "surviving years remain suggested")
			assert.Contains(t, coverage.stdout, "1830 US Census")
			assert.Contains(t, proof.stdout, "1830 US Census")
		})
	}
}

const censusClassificationArchive = `places:
  place-usa: {name: United States, type: country}
  place-state: {name: %q, type: %s, parent: place-usa}
persons:
  person-subject: {properties: {name: {value: Research Subject}}}
events:
  ev-birth: {type: birth, date: "%d", place: place-state, participants: [{person: person-subject, role: principal}]}
  ev-res: {type: residence, date: "%d", place: place-state, participants: [{person: person-subject, role: principal}]}
  ev-death: {type: death, date: "1840", place: place-state, participants: [{person: person-subject, role: principal}]}
`

const researchLogArchive = `persons:
  person-lewis:
    properties: {name: {value: "Lewis Little"}, sex: male}
sources:
  src-findagrave:
    title: Find a Grave
  src-study:
    title: Study Source
citations:
  cit-fag-search:
    source: src-findagrave
    properties:
      locator: "Search: Lewis Little, d. 1816-1836, Illinois"
      text_from_source: "No memorial."
  cit-orphan:
    source: src-findagrave
studies:
  study-1:
    title: Little ONS
    sources: [src-study]
research_logs:
  rl-death:
    subject: {person: person-lewis}
    objective: "Find Lewis Little's grave"
    status: complete
    searches:
      - source: src-findagrave
        query: "Lewis Little d. 1816-1836 Illinois"
        result: not_found
        citation: cit-fag-search
    citations: [cit-fag-search]
`

const coarserBirthplaceArchive = `places:
  place-usa: {name: United States, type: country}
  place-nc: {name: North Carolina, type: state, parent: place-usa}
  place-rowan: {name: Rowan County, type: county, parent: place-nc}
persons:
  person-lewis: {properties: {name: {value: Lewis Little}, sex: male}}
  person-jasper: {properties: {name: {value: Jasper Little}}}
  person-sarah: {properties: {name: {value: Sarah Little}}}
  person-elizabeth: {properties: {name: {value: Elizabeth Little}}}
  person-james: {properties: {name: {value: James Little}}}
events:
  ev-b-jasper: {type: birth, date: "1791", place: place-rowan, participants: [{person: person-jasper, role: principal}]}
  ev-b-sarah: {type: birth, date: "1798", place: place-rowan, participants: [{person: person-sarah, role: principal}]}
  ev-b-elizabeth: {type: birth, date: "1802", place: place-rowan, participants: [{person: person-elizabeth, role: principal}]}
  ev-b-james: {type: birth, date: "ABT 1796", place: place-nc, participants: [{person: person-james, role: principal}]}
relationships:
  rel-1: {type: parent_child, participants: [{person: person-lewis, role: parent}, {person: person-jasper, role: child}]}
  rel-2: {type: parent_child, participants: [{person: person-lewis, role: parent}, {person: person-sarah, role: child}]}
  rel-3: {type: parent_child, participants: [{person: person-lewis, role: parent}, {person: person-elizabeth, role: child}]}
  rel-4: {type: parent_child, participants: [{person: person-lewis, role: parent}, {person: person-james, role: child}]}
`

const licenseOnlyArchive = `persons:
  person-adam: {properties: {name: {value: Adam Call}, sex: male}}
  person-elizabeth: {properties: {name: {value: Elizabeth Starr}, sex: female}}
places:
  place-surry: {name: Surry County}
events:
  ev-bond-1785:
    type: marriage_license
    date: "1785-12-17"
    place: place-surry
    properties: {event_subtype: marriage bond}
    participants:
      - {person: person-adam, role: groom}
      - {person: person-elizabeth, role: bride}
relationships:
  rel-marr:
    type: marriage
    start_event: ev-bond-1785
    participants:
      - {person: person-adam, role: spouse}
      - {person: person-elizabeth, role: spouse}
`

const lostCensusArchive = `places:
  place-usa: {name: United States, type: country}
  place-int: {name: Indiana Territory, type: territory, parent: place-usa}
  place-dearborn: {name: Dearborn County, type: county, parent: place-int}
  place-va: {name: Virginia, type: state, parent: place-usa}
  place-oh: {name: Ohio, type: state, parent: place-usa}
persons:
  person-lewis-little: {properties: {name: {value: Lewis Little}, sex: male}}
events:
  ev-birth: {type: birth, date: "1756", place: place-va, participants: [{person: person-lewis-little, role: principal}]}
  ev-res: {type: residence, date: "1810", place: place-dearborn, participants: [{person: person-lewis-little, role: principal}]}
  ev-res2: {type: residence, date: "1798", place: place-oh, participants: [{person: person-lewis-little, role: principal}]}
  ev-death: {type: death, date: "1830", place: place-dearborn, participants: [{person: person-lewis-little, role: principal}]}
`
