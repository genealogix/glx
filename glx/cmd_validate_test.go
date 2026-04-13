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

func TestRunValidate_SingleValidFile(t *testing.T) {
	// Test validating a single valid GLX file (structure only, no cross-references)
	t.Chdir("../docs/examples/basic-family")
	streams, out, _ := TestIOStreams()

	err := validatePaths(streams, []string{"persons/person-father.glx"})
	require.NoError(t, err, "should successfully validate a valid GLX file")
	require.Contains(t, out.String(), "Cross-reference validation skipped")
	require.Contains(t, out.String(), "File structure is valid")
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

func TestRunValidate_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	streams, _, _ := TestIOStreams()
	err := validatePaths(streams, []string{"does-not-exist"})
	require.NoError(t, err, "non-existent path results in 0 files validated")
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
	streams, _, _ := TestIOStreams()

	err := validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "empty directory should validate successfully")
}

func TestRunValidate_OnlyNonGLXFiles(t *testing.T) {
	tmpDir := t.TempDir()

	txtFile := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtFile, []byte("This is not a GLX file"), 0o644)
	require.NoError(t, err)

	streams, _, _ := TestIOStreams()
	err = validatePaths(streams, []string{tmpDir})
	require.NoError(t, err, "directory with no GLX files should validate successfully")
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
