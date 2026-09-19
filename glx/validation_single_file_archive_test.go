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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// singleFileArchive is the shape `glx join` writes: entities and the archive's
// own vocabularies in one file. placeID is the place the birth event points at,
// so a test can dangle that reference without touching anything else.
func singleFileArchive(placeID string) string {
	return `event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
persons:
  person-robert:
    properties:
      name: "Robert Thompson"
events:
  event-birth-robert:
    type: birth
    date: "1850-04-12"
    place: ` + placeID + `
    participants:
      - person: person-robert
places:
  place-springfield:
    name: "Springfield"
    type: city
`
}

func writeArchiveFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

// The bug in #1270: the same dangling reference is a hard error in the
// multi-file form and passed green in the single-file one.
func TestValidatePaths_SingleFileArchive_ReportsDanglingReference(t *testing.T) {
	path := writeArchiveFile(t, singleFileArchive("place-DOES-NOT-EXIST"))
	streams, out, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.ErrorIs(t, err, ErrValidationFailed)
	assert.Contains(t, errOut.String(), "references non-existent places: place-DOES-NOT-EXIST")
	assert.NotContains(t, out.String(), "Cross-reference validation skipped")
}

func TestValidatePaths_SingleFileArchive_ValidArchivePasses(t *testing.T) {
	path := writeArchiveFile(t, singleFileArchive("place-springfield"))
	streams, out, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.NoError(t, err, errOut.String())
	assert.Contains(t, out.String(), "Validated 1 files.")
	assert.Contains(t, out.String(), "✅ Archive is valid.")
}

// A fragment of a multi-file archive keeps the skip: its references resolve in
// its siblings, which are not on the command line.
func TestValidatePaths_Fragment_KeepsCrossReferenceSkip(t *testing.T) {
	fragment := `events:
  event-birth-robert:
    type: birth
    place: place-springfield
    participants:
      - person: person-robert
`
	path := writeArchiveFile(t, fragment)
	streams, out, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.NoError(t, err, errOut.String())
	assert.Contains(t, out.String(), "Cross-reference validation skipped")
	assert.NotContains(t, errOut.String(), "references non-existent")
}

// The success line must not claim more than was checked: the cross-reference
// and place-hierarchy checks are filtered out of the fragment path.
func TestValidatePaths_Fragment_SuccessLineNamesWhatWasSkipped(t *testing.T) {
	path := writeArchiveFile(t, "persons:\n  person-robert:\n    properties:\n      name: Robert\n")
	streams, out, _ := TestIOStreams()

	require.NoError(t, validatePaths(streams, []string{path}))
	assert.Contains(t, out.String(), "cross-reference and place-hierarchy checks skipped")
	assert.NotContains(t, out.String(), "passed structural and semantic validation")
}

func TestIsSelfContainedArchive(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want bool
	}{
		{
			name: "join output: entities plus the archive's own vocabularies",
			yaml: singleFileArchive("place-springfield"),
			want: true,
		},
		{
			name: "init --single-file scaffold: every entity collection, no vocabulary",
			yaml: "persons: {}\nrelationships: {}\nevents: {}\nplaces: {}\nsources: {}\n" +
				"citations: {}\nrepositories: {}\nassertions: {}\nmedia: {}\n" +
				"research_logs: {}\nstudies: {}\n",
			want: true,
		},
		{
			name: "entity fragment: one collection holding one entity",
			yaml: "events:\n  event-birth-robert:\n    type: birth\n",
			want: false,
		},
		{
			name: "two entity fragments in one file, still no vocabulary of its own",
			yaml: "events:\n  event-birth-robert:\n    type: birth\npersons:\n  person-robert: {}\n",
			want: false,
		},
		{
			name: "vocabulary fragment: no entities to check references between",
			yaml: "event_types:\n  birth:\n    label: Birth\n",
			want: false,
		},
		{
			name: "property definitions count as the archive's own vocabulary",
			yaml: "person_properties:\n  name:\n    type: string\npersons:\n  person-robert: {}\n",
			want: true,
		},
		{
			name: "metadata only",
			yaml: "metadata:\n  source_system: GLX\n",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseYAMLFile([]byte(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.want, isSelfContainedArchive(doc))
		})
	}
}

// The detection keys are reflected off GLXFile and glxlib.AllEntityTypes so a
// new entity type or vocabulary is picked up with no list to update here.
func TestCollectionKeys_TrackGLXFile(t *testing.T) {
	entities := entityCollectionKeys()
	require.Len(t, entities, len(glxlib.AllEntityTypes))
	for _, et := range glxlib.AllEntityTypes {
		assert.True(t, entities[et.String()], "entity collection %s", et)
	}

	definitions := definitionCollectionKeys()
	assert.True(t, definitions["event_types"], "vocabulary collections are definitions")
	assert.True(t, definitions["person_properties"], "property definitions are definitions")
	for key := range entities {
		assert.False(t, definitions[key], "entity collection %s must not count as a definition", key)
	}
}

func TestIsSelfContainedArchiveFile_UnreadableOrUnparsable(t *testing.T) {
	dir := t.TempDir()

	notGLX := filepath.Join(dir, "archive.yaml")
	require.NoError(t, os.WriteFile(notGLX, []byte(singleFileArchive("place-springfield")), 0o644))
	assert.False(t, isSelfContainedArchiveFile(notGLX), "only .glx files are archive content")

	assert.False(t, isSelfContainedArchiveFile(filepath.Join(dir, "missing.glx")))

	broken := filepath.Join(dir, "broken.glx")
	require.NoError(t, os.WriteFile(broken, []byte("persons: [\n"), 0o644))
	assert.False(t, isSelfContainedArchiveFile(broken),
		"an unparsable file is reported by the structural pass, not routed by shape")
}

// A self-contained archive that fails schema validation is reported as a
// structural failure, with the file named — it is not waved through because the
// file was routed as a whole archive.
func TestValidatePaths_SingleFileArchive_ReportsStructuralErrors(t *testing.T) {
	// A place with no name: valid YAML, whole-archive shape, invalid schema.
	broken := `event_types:
  birth:
    label: "Birth"
persons:
  person-robert:
    properties:
      name: "Robert Thompson"
places:
  place-springfield:
    type: city
`
	path := writeArchiveFile(t, broken)
	streams, _, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.ErrorIs(t, err, ErrStructuralValidationFailed)
	assert.Contains(t, errOut.String(), "Found 1 structural errors")
	assert.Contains(t, errOut.String(), path)
}

// Media URIs in a single-file archive resolve against the directory holding the
// file, and a missing file is a warning rather than an error — the same verdict
// the directory pass gives.
func TestValidatePaths_SingleFileArchive_WarnsOnMissingMediaFile(t *testing.T) {
	archive := `media_types:
  photograph:
    label: "Photograph"
persons:
  person-robert:
    properties:
      name: "Robert Thompson"
media:
  media-portrait:
    title: "Portrait of Robert Thompson"
    type: photograph
    uri: "media/files/portrait.jpg"
`
	path := writeArchiveFile(t, archive)
	streams, out, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.NoError(t, err, errOut.String())
	assert.Contains(t, errOut.String(), "referenced file does not exist: media/files/portrait.jpg")
	assert.Contains(t, out.String(), "✅ Archive is valid.")
}

// Semantic warnings from the archive's own validation reach the output too — a
// single-file archive is not held to a narrower set of checks than a directory.
func TestValidatePaths_SingleFileArchive_ReportsSemanticWarnings(t *testing.T) {
	archive := `event_types:
  birth:
    label: "Birth"
  death:
    label: "Death"
persons:
  person-robert:
    properties:
      name: "Robert Thompson"
events:
  event-birth-robert:
    type: birth
    date: "1850-04-12"
    participants:
      - person: person-robert
        role: subject
  event-death-robert:
    type: death
    date: "1840-01-09"
    participants:
      - person: person-robert
        role: subject
`
	path := writeArchiveFile(t, archive)
	streams, out, errOut := TestIOStreams()

	err := validatePaths(streams, []string{path})

	require.NoError(t, err, errOut.String())
	assert.Contains(t, errOut.String(), "death year (1840) is before birth year (1850)")
	assert.Contains(t, out.String(), "✅ Archive is valid.")
}
