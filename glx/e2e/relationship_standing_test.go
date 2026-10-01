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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// littleFamilyGLX is the Lewis Little family from #1325, #1326, #1327, #1331
// and #208, as a single-file archive: a guardian/ward and godparent/godchild
// relationship, a step-parent, and three candidate fathers of whom two are
// disproven.
const littleFamilyGLX = `participant_roles:
  guardian:
    label: Guardian
    description: Person appointed guardian of a minor
    applies_to: [event, relationship]
  ward:
    label: Ward
    description: Minor placed under a guardian
    applies_to: [event, relationship]
persons:
  person-lewis: {properties: {name: {value: "Lewis Little"}, sex: male}}
  person-rachel: {properties: {name: {value: "Rachel Call"}, sex: female}}
  person-peggy: {properties: {name: {value: "Peggy Call"}, sex: female}}
  person-adam: {properties: {name: {value: "Adam Little"}, sex: male}}
  person-adam-call: {properties: {name: {value: "Adam Call"}, sex: male}}
  person-starr: {properties: {name: {value: "Elizabeth Starr"}, sex: female}}
  person-nancy: {properties: {name: {value: "Nancy Little"}, sex: female}}
  person-daniel: {properties: {name: {value: "Johannes Daniel Little"}, sex: male}}
  person-jacob: {properties: {name: {value: "Jacob Little"}, sex: male}}
  person-john: {properties: {name: {value: "John Little"}, sex: male}}
events:
  ev-death-rachel:
    type: death
    date: "1843-12"
    participants: [{person: person-rachel, role: principal}]
sources:
  src-1800-census: {title: 1800 census}
relationships:
  rel-guardian:
    type: guardian
    participants:
      - {person: person-lewis, role: guardian}
      - {person: person-rachel, role: ward}
  rel-godparent:
    type: godparent
    participants:
      - {person: person-lewis, role: godparent}
      - {person: person-adam, role: godchild}
  rel-pc-call:
    type: parent_child
    participants:
      - {person: person-adam-call, role: parent}
      - {person: person-starr, role: parent}
      - {person: person-rachel, role: child}
      - {person: person-peggy, role: child}
  rel-step:
    type: step_parent
    participants:
      - {person: person-lewis, role: parent}
      - {person: person-rachel, role: child}
      - {person: person-peggy, role: child}
  rel-pc-nancy:
    type: parent_child
    participants:
      - {person: person-lewis, role: parent}
      - {person: person-starr, role: parent}
      - {person: person-nancy, role: child}
  rel-pc-daniel:
    type: parent_child
    participants: [{person: person-daniel, role: parent}, {person: person-lewis, role: child}]
  rel-pc-jacob:
    type: parent_child
    participants: [{person: person-jacob, role: parent}, {person: person-lewis, role: child}]
  rel-pc-john:
    type: parent_child
    participants: [{person: person-john, role: parent}, {person: person-lewis, role: child}]
assertions:
  a-daniel:
    subject: {relationship: rel-pc-daniel}
    sources: [src-1800-census]
    confidence: low
    status: unresearched
  a-jacob:
    subject: {relationship: rel-pc-jacob}
    sources: [src-1800-census]
    confidence: high
    status: disproven
  a-john:
    subject: {relationship: rel-pc-john}
    sources: [src-1800-census]
    confidence: medium
    status: disproven
`

// writeLittleFamily writes littleFamilyGLX to a temp dir and returns the dir
// and the archive path.
func writeLittleFamily(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "little.glx")
	require.NoError(t, os.WriteFile(path, []byte(littleFamilyGLX), 0o644))

	return dir, path
}

// summary labels the other person by role (#1326), separates step-relatives
// (#1331), marks hypothetical parents and sets disproven ones aside (#208).
func TestRelationshipStanding_Summary(t *testing.T) {
	dir, path := writeLittleFamily(t)

	lewis := runGLX(t, dir, "summary", "person-lewis", "--archive", path)
	require.Equal(t, 0, lewis.exitCode, lewis.stdout+lewis.stderr)
	assert.Contains(t, lewis.stdout, "Godchild:         Adam Little")
	assert.Contains(t, lewis.stdout, "Ward:             Rachel Call")
	assert.Contains(t, lewis.stdout, "Father:           Johannes Daniel Little  (?)")
	assert.Contains(t, lewis.stdout, "Excluded:         Jacob Little, John Little  (disproven parent)")
	assert.NotContains(t, lewis.stdout, "Father:           Jacob Little")
	assert.Contains(t, lewis.stdout, "He had one child: Nancy, and two stepdaughters, Peggy and Rachel Call.")

	rachel := runGLX(t, dir, "summary", "person-rachel", "--archive", path)
	require.Equal(t, 0, rachel.exitCode, rachel.stdout+rachel.stderr)
	assert.Contains(t, rachel.stdout, "Stepfather:       Lewis Little")
	assert.NotContains(t, rachel.stdout, "Father:           Lewis Little")
	assert.Contains(t, rachel.stdout, "Siblings:         Peggy Call")
	assert.Contains(t, rachel.stdout, "Half-siblings:    Nancy Little")
	assert.Contains(t, rachel.stdout, "Guardian:         Lewis Little")
}

// The ancestor tree drops disproven fathers and marks the hypothetical one
// (#208); the descendant tree keeps marking stepchildren.
func TestRelationshipStanding_Trees(t *testing.T) {
	dir, path := writeLittleFamily(t)

	anc := runGLX(t, dir, "ancestors", "person-lewis", "--archive", path)
	require.Equal(t, 0, anc.exitCode, anc.stdout+anc.stderr)
	assert.Contains(t, anc.stdout, "Johannes Daniel Little  person-daniel  (?)")
	assert.NotContains(t, anc.stdout, "Jacob Little")
	assert.NotContains(t, anc.stdout, "John Little")

	desc := runGLX(t, dir, "descendants", "person-lewis", "--archive", path)
	require.Equal(t, 0, desc.exitCode, desc.stdout+desc.stderr)
	assert.Contains(t, desc.stdout, "Rachel Call  (d. 1843-12)  person-rachel  [step]")
}

// A guardian's timeline shows the ward's death as such (#1325).
func TestRelationshipStanding_TimelineWard(t *testing.T) {
	dir, path := writeLittleFamily(t)

	res := runGLX(t, dir, "timeline", "person-lewis", "--archive", path)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Contains(t, res.stdout, "Death of ward (Rachel Call)")
	assert.NotContains(t, res.stdout, "Death of parent")
}

// proof lists disproven candidates as excluded alternatives, not parents
// (#1327).
func TestRelationshipStanding_ProofParentage(t *testing.T) {
	dir, path := writeLittleFamily(t)

	res := runGLX(t, dir, "proof", "person-lewis", "--question", "parentage", "--archive", path)
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Contains(t, res.stdout, "Alternatives Excluded:")
	assert.Contains(t, res.stdout, "x Jacob Little (rel-pc-jacob) -- all assertions disproven")
	assert.Contains(t, res.stdout, "x John Little (rel-pc-john) -- all assertions disproven")
	assert.Contains(t, res.stdout, "Parents identified: Johannes Daniel Little.")
	assert.Contains(t, res.stdout, "Conclusion: POSSIBLE")
}
