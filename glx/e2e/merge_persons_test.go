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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	mergeKeepID = "person-robert-thompson"
	mergeDropID = "person-robert-dup"
)

// archiveWithDuplicate stages basic-family plus a second record for Robert
// Thompson, created through the CLI, and returns the archive path.
func archiveWithDuplicate(t *testing.T) string {
	t.Helper()
	archive := copyExample(t, "basic-family")
	res := runGLX(t, archive, "add", "person", "--id", mergeDropID,
		"--given", "Robert", "--surname", "Thompson", "--sex", "male", "--note", "second record")
	require.Equal(t, 0, res.exitCode, res.stderr)
	require.FileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))

	return archive
}

func TestMergePersons_FromInsideArchiveRoot_DefaultArchiveFlag(t *testing.T) {
	archive := archiveWithDuplicate(t)
	dirBefore := statDir(t, archive)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "-y")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Merging "+mergeKeepID+" ← "+mergeDropID)
	assertSameDirectory(t, dirBefore, archive)
	assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
	kept, err := os.ReadFile(filepath.Join(archive, "persons", mergeKeepID+".glx"))
	require.NoError(t, err)
	assert.Contains(t, string(kept), "second record", "the dropped person's note is appended by default")
	assertArchiveValid(t, archive)
}

func TestMergePersons_ArchiveFlagFromOutside(t *testing.T) {
	archive := archiveWithDuplicate(t)

	res := runGLX(t, t.TempDir(), "merge-persons", mergeKeepID, mergeDropID, "--archive", archive, "--yes")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
	assertArchiveValid(t, archive)
}

func TestMergePersons_DryRunWritesNothing(t *testing.T) {
	archive := archiveWithDuplicate(t)
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "Merging "+mergeKeepID)
	assertTreeUnchanged(t, before, archive)
}

func TestMergePersons_ResolvedIdentityCandidate(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprintf("reverse=%t", reverse), func(t *testing.T) {
			archive := archiveWithDuplicate(t)
			first, second := mergeKeepID, mergeDropID
			if reverse {
				first, second = second, first
			}
			relPath := filepath.Join(archive, "relationships", "rel-resolved.glx")
			require.NoError(t, os.WriteFile(relPath, []byte(fmt.Sprintf(`relationships:
  rel-resolved:
    type: possibly_same_person
    participants:
      - person: %s
      - person: %s
`, first, second)), 0o644))
			assertArchiveValid(t, archive)
			before := snapshotTree(t, archive)

			preview := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--dry-run")
			require.Equal(t, 0, preview.exitCode, preview.stderr)
			assert.Contains(t, preview.stdout, "Relationship removed: rel-resolved (possibly_same_person)")
			assertTreeUnchanged(t, before, archive)

			merged := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "-y")
			require.Equal(t, 0, merged.exitCode, merged.stderr)
			assert.Contains(t, merged.stdout, "Relationship removed: rel-resolved (possibly_same_person)")
			assert.NoFileExists(t, relPath)
			assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
			assertArchiveValid(t, archive)

			after := snapshotTree(t, archive)
			// Compare all entity collections as well as preserved file content.
			serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{})
			expected, conflicts, err := serializer.DeserializeMultiFileFromMap(before)
			require.NoError(t, err)
			require.Empty(t, conflicts)
			actual, conflicts, err := serializer.DeserializeMultiFileFromMap(after)
			require.NoError(t, err)
			require.Empty(t, conflicts)
			delete(expected.Persons, mergeDropID)
			delete(expected.Relationships, "rel-resolved")
			expected.Persons[mergeKeepID].Notes = append(expected.Persons[mergeKeepID].Notes, "second record")
			assert.Equal(t, expected.Persons, actual.Persons)
			assert.Equal(t, expected.Relationships, actual.Relationships)
			assert.Equal(t, expected.Events, actual.Events)
			assert.Equal(t, expected.Places, actual.Places)
			assert.Equal(t, expected.Sources, actual.Sources)
			assert.Equal(t, expected.Citations, actual.Citations)
			assert.Equal(t, expected.Repositories, actual.Repositories)
			assert.Equal(t, expected.Assertions, actual.Assertions)
			assert.Equal(t, expected.Media, actual.Media)
			assert.Equal(t, expected.ResearchLogs, actual.ResearchLogs)
			assert.Equal(t, expected.Studies, actual.Studies)
			assert.Equal(t, string(before["README.md"]), string(after["README.md"]))
		})
	}
}

func TestMergePersons_NotesStrategyPreferKeep(t *testing.T) {
	archive := archiveWithDuplicate(t)

	res := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--notes-strategy", "prefer-keep", "-y")

	require.Equal(t, 0, res.exitCode, res.stderr)
	kept, err := os.ReadFile(filepath.Join(archive, "persons", mergeKeepID+".glx"))
	require.NoError(t, err)
	assert.NotContains(t, string(kept), "second record")
}

func TestMergePersons_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := archiveWithDuplicate(t)
	before := snapshotTree(t, archive)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown keep", []string{"person-nobody", mergeDropID}, `keep-id "person-nobody"`},
		{"unknown drop", []string{mergeKeepID, "person-nobody"}, `drop-id "person-nobody"`},
		{"self merge", []string{mergeKeepID, mergeKeepID}, "cannot merge person with itself"},
		{"one arg", []string{mergeKeepID}, "accepts 2 arg(s), received 1"},
		{"three args", []string{mergeKeepID, mergeDropID, "extra"}, "accepts 2 arg(s), received 3"},
		{"conflicting keep flags", []string{mergeKeepID, mergeDropID, "--keep-newest", "--keep-oldest"}, "mutually exclusive"},
		{"bad notes strategy", []string{mergeKeepID, mergeDropID, "--notes-strategy", "bogus"}, "invalid notes strategy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"merge-persons"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestMergePersons_MissingArchivePathFails(t *testing.T) {
	res := runGLX(t, t.TempDir(), "merge-persons", mergeKeepID, mergeDropID, "--archive", "does-not-exist")

	assertExitWithStderr(t, res, "cannot access path")
}

func TestMergePersons_NoninteractiveRequiresExplicitYes(t *testing.T) {
	archive := archiveWithDuplicate(t)
	before := snapshotTree(t, archive)
	for _, flags := range [][]string{nil, {"--yes=false"}} {
		args := append([]string{"merge-persons", mergeKeepID, mergeDropID}, flags...)
		result := runGLX(t, archive, args...)
		assertExitWithStderr(t, result, "noninteractive merge requires --yes")
		assert.Contains(t, result.stdout, "Merging "+mergeKeepID)
		assertTreeUnchanged(t, before, archive)
	}
	preview := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "--dry-run", "-y")
	require.Equal(t, 0, preview.exitCode, preview.stderr)
	assertTreeUnchanged(t, before, archive)
	applied := runGLX(t, archive, "merge-persons", mergeKeepID, mergeDropID, "-y")
	require.Equal(t, 0, applied.exitCode, applied.stderr)
	assert.NoFileExists(t, filepath.Join(archive, "persons", mergeDropID+".glx"))
}

func TestMergePersonsResolvedEvidenceCleanupPreviewMatchesAppliedFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	data := []byte(`persons:
  keep: {}
  drop: {}
  other:
    properties:
      identity_case: [resolved, unrelated]
person_properties:
  identity_case: {label: Identity case, reference_type: relationships, multi_value: true}
relationship_types:
  associate: {label: Associate}
  possibly_same_person: {label: Possibly same person}
confidence_levels:
  high: {label: High confidence}
relationships:
  resolved:
    type: possibly_same_person
    participants: [{person: drop}, {person: keep}]
  unrelated:
    type: associate
    participants: [{person: keep}, {person: other}]
sources:
  register: {title: Original register}
citations:
  record: {source: register}
media:
  scan: {uri: scan.jpg, source: register}
assertions:
  rejected:
    subject: {relationship: resolved}
    status: disproven
    confidence: high
    citations: [record]
    sources: [register]
    media: [scan]
    notes: Original rejected claim
  accepted:
    subject: {person: drop}
    property: name
    value: Original name
    status: proven
    citations: [record]
research_logs:
  research:
    title: Original negative research
    subject: {relationship: resolved}
    conclusions: No identification found
    citations: [record]
`)
	require.NoError(t, os.WriteFile(path, data, 0o600))
	assertArchiveValid(t, dir)
	before := snapshotTree(t, dir)
	preview := runGLX(t, dir, "merge-persons", "keep", "drop", "--dry-run", "-y")
	require.Equal(t, 0, preview.exitCode, preview.stderr)
	for _, text := range []string{"Assertion removed: rejected", "status: disproven", "subject", "properties.identity_case[0]"} {
		assert.Contains(t, preview.stdout, text)
	}
	assertTreeUnchanged(t, before, dir)
	applied := runGLX(t, dir, "merge-persons", "keep", "drop", "-y")
	require.Equal(t, 0, applied.exitCode, applied.stderr)
	assert.Equal(t, preview.stdout, applied.stdout+"(dry run — no files written)\n")
	saved, err := os.ReadFile(path)
	require.NoError(t, err)
	archive, err := glxlib.NewSerializer(&glxlib.SerializerOptions{}).DeserializeSingleFileBytes(saved)
	require.NoError(t, err)
	assert.NotContains(t, archive.Assertions, "rejected")
	assert.Equal(t, "keep", archive.Assertions["accepted"].Subject.Person)
	assert.Equal(t, "proven", archive.Assertions["accepted"].Status)
	assert.Nil(t, archive.ResearchLogs["research"].Subject)
	assert.Equal(t, "No identification found", archive.ResearchLogs["research"].Conclusions)
	assert.Equal(t, []any{"unrelated"}, archive.Persons["other"].Properties["identity_case"])
	assert.Contains(t, archive.Sources, "register")
	assert.Contains(t, archive.Citations, "record")
	assert.Contains(t, archive.Media, "scan")
	assert.Equal(t, "relationships", archive.PersonProperties["identity_case"].ReferenceType)
	assertArchiveValid(t, dir)
}
