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

const chronologyRepro = `persons:
  p: {properties: {name: Veronica}}
  q: {properties: {name: Fronica}}
sources:
  s: {title: Parish register, type: church_register}
citations:
  c: {source: s}
events:
  burial:
    type: burial
    date: JULIAN 1635
    participants: [{person: p, role: principal}]
  death:
    type: death
    date: JULIAN 1653-11-01
    participants: [{person: p, role: principal}]
assertions:
  b: {subject: {event: burial}, property: date, value: JULIAN 1635, citations: [c], confidence: high, status: proven}
  d: {subject: {event: death}, property: date, value: JULIAN 1653-11-01, citations: [c], confidence: high, status: proven}
`

func TestVitalChronology_ProofAnalyzeAndMergePreview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(chronologyRepro), 0o600))
	proof := runGLX(t, dir, "proof", "p", "--question", "death", "--archive", path)
	require.Equal(t, 0, proof.exitCode, proof.stderr)
	assert.Contains(t, proof.stdout, "CONFLICTED")
	assert.Contains(t, proof.stdout, "before death")
	analysis := runGLX(t, dir, "analyze", "--check", "consistency", "--archive", path)
	require.Equal(t, 0, analysis.exitCode, analysis.stderr)
	assert.Contains(t, analysis.stdout, "Death/burial chronology")
	// Put the burial on the other candidate: the conflict must appear in
	// the preview interpretation, while the archive remains unchanged.
	separate := strings.Replace(chronologyRepro, "participants: [{person: p, role: principal}]", "participants: [{person: q, role: principal}]", 1)
	require.NoError(t, os.WriteFile(path, []byte(separate), 0o600))
	before := snapshotTree(t, dir)
	merge := runGLX(t, dir, "merge-persons", "p", "q", "--archive", path, "--dry-run")
	require.Equal(t, 0, merge.exitCode, merge.stderr)
	assert.Contains(t, merge.stdout, "CONFLICTED")
	assert.Contains(t, merge.stdout, "Death/burial chronology")
	assertTreeUnchanged(t, before, dir)
}

func TestVitalChronology_VisitsAndBoundedDateOrdering(t *testing.T) {
	archive := `persons:
  p: {properties: {name: Elisabeth}}
  child: {properties: {name: Child}}
places:
  home: {name: Kirch-Gons, type: village}
  away: {name: Pohl-Gons, type: village}
events:
  z-birth:
    type: birth
    date: BEF 1734-09-01
    place: home
    participants: [{person: p, role: principal}]
  a-baptism:
    type: baptism
    date: 1734-09-01
    place: home
    participants: [{person: p, role: principal}]
  visit:
    type: baptism
    date: 1768-05-15
    place: away
    participants: [{person: child, role: principal}, {person: p, role: godparent}]
  death:
    type: death
    date: 1803-03-14
    place: home
    participants: [{person: p, role: principal}]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(archive), 0o600))
	timeline := runGLX(t, dir, "timeline", "p", "--archive", path, "--no-family")
	require.Equal(t, 0, timeline.exitCode, timeline.stderr)
	birth, baptism := strings.Index(timeline.stdout, "Birth"), strings.Index(timeline.stdout, "Baptism")
	require.NotEqual(t, -1, birth, timeline.stdout)
	require.NotEqual(t, -1, baptism, timeline.stdout)
	assert.Less(t, birth, baptism, "a BEF birth must precede the baptism that bounds it")
	migrations := runGLX(t, dir, "migrations", "p", "--archive", path)
	require.Equal(t, 0, migrations.exitCode, migrations.stderr)
	assert.Contains(t, migrations.stdout, "as godparent")
	assert.Contains(t, migrations.stdout, "shows a visit, not residence")
	assert.NotContains(t, migrations.stdout, "Kirch-Gons → Pohl-Gons")
	assert.NotContains(t, migrations.stdout, "Pohl-Gons → Kirch-Gons")
}
