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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunValidate_SingleValidFile(t *testing.T) {
	// Test validating a single valid GLX file (structure only, no cross-references)
	t.Chdir("../docs/examples/basic-family")
	streams, out, _ := TestIOStreams()

	err := validatePaths(streams, []string{"persons/person-robert-thompson.glx"})
	require.NoError(t, err, "should successfully validate a valid GLX file")
	require.Contains(t, out.String(), "Cross-reference validation skipped")
	require.Contains(t, out.String(), "passed structural and semantic validation")
}

func TestRunValidate_ValidDirectory(t *testing.T) {
	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"../docs/examples/basic-family"})
	require.NoError(t, err, "should successfully validate a valid directory")
}

func TestRunValidate_CurrentDirectory(t *testing.T) {
	t.Chdir("../docs/examples/basic-family")
	streams, _, _ := TestIOStreams()

	err := validatePaths(streams, []string{})
	require.NoError(t, err, "should successfully validate current directory when no args provided")
}

func TestRunValidate_MultiplePaths(t *testing.T) {
	t.Chdir("../docs/examples/basic-family")
	streams, _, _ := TestIOStreams()

	err := validatePaths(streams, []string{"persons", "relationships"})
	require.NoError(t, err, "should successfully validate multiple valid paths")
}

func TestRunValidate_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.glx")
	err := os.WriteFile(invalidFile, []byte("persons:\n  person-1:\n    invalid: [unclosed"), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.Error(t, err, "should fail on invalid YAML syntax")
}

func TestRunValidate_StructuralErrors(t *testing.T) {
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "bad-structure.glx")
	err := os.WriteFile(invalidFile, []byte(`persons:
  "person with spaces":
    properties:
      primary_name: "Test"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.Error(t, err, "should fail on structural validation errors")
}

func TestRunValidate_DuplicateIDs(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.glx")
	err := os.WriteFile(file1, []byte(`persons:
  person-duplicate:
    properties:
      primary_name: "Person One"
`), 0o644)
	require.NoError(t, err)

	file2 := filepath.Join(tmpDir, "file2.glx")
	err = os.WriteFile(file2, []byte(`persons:
  person-duplicate:
    properties:
      primary_name: "Person Two"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.Error(t, err, "should detect duplicate entity IDs across files")
}

func TestRunValidate_BrokenReferences(t *testing.T) {
	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"testdata/invalid/broken-references"})
	require.Error(t, err, "should fail when cross-references are broken")
}

func TestRunValidate_PlaceCoordsHalfSet(t *testing.T) {
	// A Place may carry both latitude and longitude or neither; setting one
	// without the other is meaningless and must be rejected by the schema's
	// dependencies clause (see specification/schema/v1/place.schema.json).
	// The fixture contains two places — one for each half-set direction —
	// so a single validation pass covers both.
	streams, _, errOut := TestIOStreams()
	err := validatePaths(streams, []string{"testdata/invalid/place-coords-half-set"})
	require.Error(t, err, "half-set coordinates should be rejected")
	require.Contains(t, errOut.String(), "latitude",
		"error should name latitude (for the longitude-only place)")
	require.Contains(t, errOut.String(), "longitude",
		"error should name longitude (for the latitude-only place)")
}

func TestRunValidate_PlaceCoordsBothSet(t *testing.T) {
	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"testdata/valid/place-coords-both-set"})
	require.NoError(t, err, "place with both latitude and longitude should pass validation")
}

func TestRunValidate_PlaceCoordsNeitherSet(t *testing.T) {
	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"testdata/valid/place-coords-neither-set"})
	require.NoError(t, err, "place with neither latitude nor longitude should pass validation")
}

func TestRunValidate_RemovedProperty(t *testing.T) {
	tmpDir := t.TempDir()

	personFile := filepath.Join(tmpDir, "person.glx")
	err := os.WriteFile(personFile, []byte(`persons:
  person-test:
    properties:
      born_at: "place-nonexistent"
`), 0o644)
	require.NoError(t, err)

	streams, _, errOut := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})

	require.Error(t, err, "should fail when person has removed born_at property")
	require.Contains(t, errOut.String(), "has been removed",
		"error should mention that property has been removed")
	require.Contains(t, errOut.String(), "use birth events instead",
		"error should mention the migration path")
}

func TestRunValidate_SingleFileDeprecatedProperty(t *testing.T) {
	// Single-file validation should catch deprecated properties (not just directory mode)
	tmpDir := t.TempDir()
	personFile := filepath.Join(tmpDir, "person.glx")
	err := os.WriteFile(personFile, []byte(`persons:
  person-test:
    properties:
      name:
        value: "Test Person"
      born_on: "1850"
`), 0o644)
	require.NoError(t, err)

	streams, _, errOut := TestIOStreams()

	// Validate the single file (not the directory)
	err = validatePaths(streams, []string{personFile})

	require.ErrorIs(t, err, ErrValidationFailed, "single-file validation should return ErrValidationFailed")
	require.Contains(t, errOut.String(), "has been removed",
		"error should mention that born_on has been removed")
}

func TestRunValidate_SingleFileInvalidDateFormat(t *testing.T) {
	// Single-file validation should catch date format warnings
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.glx")
	err := os.WriteFile(eventFile, []byte(`events:
  event-test:
    type: birth
    date: "January 15, 1850"
    participants:
      - person: person-test
        role: principal
`), 0o644)
	require.NoError(t, err)

	streams, out, _ := TestIOStreams()
	err = validatePaths(streams, []string{eventFile})

	require.NoError(t, err,
		"invalid date format is a warning, not an error — should not fail validation")
	require.Contains(t, out.String(), "should be in format",
		"warning output should mention expected date format")
}

func TestRunValidate_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	streams, out, _ := TestIOStreams()
	err := validatePaths(streams, []string{"does-not-exist"})
	require.NoError(t, err, "non-existent path results in 0 files validated")
	require.Contains(t, out.String(), "0 files validated",
		"should report that no files were validated for a non-existent path")
}

func TestRunValidate_MixedValidAndInvalidFiles(t *testing.T) {
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.glx")
	err := os.WriteFile(validFile, []byte(`persons:
  person-test:
    properties:
      primary_name: "Test Person"
`), 0o644)
	require.NoError(t, err)

	invalidFile := filepath.Join(tmpDir, "invalid.glx")
	err = os.WriteFile(invalidFile, []byte(`persons:
  "person@invalid!":
    properties:
      primary_name: "Invalid"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.Error(t, err, "should fail when any file in directory has errors")
}

func TestRunValidate_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	streams, out, _ := TestIOStreams()

	err := validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "empty directory should validate successfully")
	require.Contains(t, out.String(), "Validated 0 file", "should report that no GLX files were validated")
}

func TestRunValidate_OnlyNonGLXFiles(t *testing.T) {
	tmpDir := t.TempDir()

	txtFile := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtFile, []byte("This is not a GLX file"), 0o644)
	require.NoError(t, err)

	streams, out, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "directory with no GLX files should validate successfully")
	require.Contains(t, out.String(), "No GLX files", "should report that zero GLX files were processed")
}

func TestRunValidate_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	personsDir := filepath.Join(tmpDir, "persons")
	err := os.MkdirAll(personsDir, 0o755)
	require.NoError(t, err)

	personFile := filepath.Join(personsDir, "person.glx")
	err = os.WriteFile(personFile, []byte(`persons:
  person-nested:
    properties:
      primary_name: "Nested Person"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "should successfully validate nested directory structures")
}

func TestRunValidate_WithVocabularies(t *testing.T) {
	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"../docs/examples/complete-family"})
	require.NoError(t, err, "should successfully validate archive with vocabularies")
}

func TestRunValidate_MediaFileMissing(t *testing.T) {
	tmpDir := t.TempDir()

	mediaFile := filepath.Join(tmpDir, "media.glx")
	err := os.WriteFile(mediaFile, []byte(`media:
  media-photo:
    uri: "media/files/nonexistent.jpg"
    mime_type: "image/jpeg"
    title: "Missing Photo"
`), 0o644)
	require.NoError(t, err)

	streams, out, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})

	require.NoError(t, err, "missing media file should produce warning, not error")
	require.Contains(t, out.String(), "media[media-photo]: referenced file does not exist: media/files/nonexistent.jpg",
		"should produce warning about missing media file")
}

func TestRunValidate_MediaFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	filesDir := filepath.Join(tmpDir, "media", "files")
	err := os.MkdirAll(filesDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("fake jpeg"), 0o644)
	require.NoError(t, err)

	mediaFile := filepath.Join(tmpDir, "media.glx")
	err = os.WriteFile(mediaFile, []byte(`media:
  media-photo:
    uri: "media/files/photo.jpg"
    mime_type: "image/jpeg"
    title: "Existing Photo"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "existing media file should not produce warnings")
}

func TestRunValidate_MediaExternalURLSkipped(t *testing.T) {
	tmpDir := t.TempDir()

	mediaFile := filepath.Join(tmpDir, "media.glx")
	err := os.WriteFile(mediaFile, []byte(`media:
  media-online:
    uri: "https://example.com/photo.jpg"
    mime_type: "image/jpeg"
    title: "Online Photo"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "external URL should not trigger file existence check")
}

func TestRunValidate_YAMLAndYMLExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	yamlFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(yamlFile, []byte(`persons:
  person-yaml:
    properties:
      primary_name: "YAML Person"
`), 0o644)
	require.NoError(t, err)

	ymlFile := filepath.Join(tmpDir, "test.yml")
	err = os.WriteFile(ymlFile, []byte(`persons:
  person-yml:
    properties:
      primary_name: "YML Person"
`), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "should successfully validate .yaml and .yml files")
}

func TestRunValidate_RespectsQuietFlag(t *testing.T) {
	// Use TestIOStreams() and rebind only Out to io.Discard, matching the
	// pattern documented in iostreams.go ("dedicated --quiet tests rebind Out
	// to io.Discard and assert separately on MachineOut"). Constructing the
	// streams ourselves avoids mutating the package-global quietOutput, which
	// SystemIOStreams reads without synchronization. SystemIOStreams's own
	// response to quietOutput is covered end-to-end by TestSystemIOStreams_Quiet
	// in iostreams_test.go.
	//
	// The test asserts the full silencing contract for --quiet: validatePaths
	// must (a) succeed on a known-good file, (b) write nothing to streams.Out
	// (trivially, since Out is io.Discard), (c) write nothing to
	// streams.MachineOut (diagnostic output must not be misrouted to the
	// machine-consumable stream), and (d) write nothing directly to os.Stdout
	// (must not bypass the IOStreams abstraction).
	streams, machineBuf, _ := TestIOStreams()
	streams.Out = io.Discard

	t.Chdir("../docs/examples/basic-family")

	var validateErr error
	stdout := captureStdout(t, func() {
		validateErr = validatePaths(streams, []string{"persons/person-robert-thompson.glx"})
	})

	require.NoError(t, validateErr, "validation of a known-good file should succeed when Out is io.Discard")
	require.Empty(t, stdout, "validatePaths must not write directly to os.Stdout, bypassing streams.Out")
	require.Empty(t, machineBuf.String(), "validatePaths must not write diagnostic output to streams.MachineOut")
}
