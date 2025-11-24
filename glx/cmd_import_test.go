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
	"strings"
	"testing"

	"github.com/genealogix/glx/glx/lib"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestImportGEDCOM_SingleFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import GEDCOM to single file")

	// Verify file was created
	_, err = os.Stat(outputPath)
	require.NoError(t, err, "output file should exist")

	// Verify it's valid YAML
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var glxFile lib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err, "output should be valid YAML")

	// Verify it has content
	require.NotEmpty(t, glxFile.Persons, "should have imported persons")
}

func TestImportGEDCOM_SingleFileFormat_AddsExtension(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output") // No .glx extension

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Verify .glx extension was added
	expectedPath := outputPath + ".glx"
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "should add .glx extension if missing")
}

func TestImportGEDCOM_MultiFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "archive")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "multi", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import GEDCOM to multi-file archive")

	// Verify directory was created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "output directory should exist")
	require.True(t, info.IsDir(), "output should be a directory")

	// Verify some expected files exist
	entries, err := os.ReadDir(outputPath)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "archive directory should contain files")

	// Check for persons directory
	personsDir := filepath.Join(outputPath, "persons")
	_, err = os.Stat(personsDir)
	require.NoError(t, err, "persons directory should exist")
}

func TestImportGEDCOM_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "invalid-format", true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail with invalid format")
	require.ErrorIs(t, err, ErrInvalidFormat)
}

func TestImportGEDCOM_GEDCOMFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/does-not-exist.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail when GEDCOM file doesn't exist")
	require.ErrorIs(t, err, ErrGEDCOMFileNotFound)
}

func TestImportGEDCOM_NoValidate(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", false, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import with --no-validate")

	// Verify file was created
	_, err = os.Stat(outputPath)
	require.NoError(t, err, "output file should exist")
}

func TestImportGEDCOM_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	// Verbose mode just prints to stdout, shouldn't affect functionality
	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", true, true, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import with --verbose")

	_, err = os.Stat(outputPath)
	require.NoError(t, err, "output file should exist")
}

func TestImportGEDCOM_Shakespeare(t *testing.T) {
	// Test with a more complex GEDCOM file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "shakespeare.glx")

	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import shakespeare.ged")

	// Read and verify
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var glxFile lib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err)

	// Shakespeare file has 31 persons
	require.GreaterOrEqual(t, len(glxFile.Persons), 30, "shakespeare.ged should have at least 30 persons")
	require.NotEmpty(t, glxFile.Events, "shakespeare.ged should have events")
	require.NotEmpty(t, glxFile.Relationships, "shakespeare.ged should have relationships")
}

func TestImportGEDCOM_InvalidGEDCOMContent(t *testing.T) {
	// Create a temporary invalid GEDCOM file
	tmpDir := t.TempDir()
	gedcomPath := filepath.Join(tmpDir, "invalid.ged")
	err := os.WriteFile(gedcomPath, []byte("This is not valid GEDCOM content"), 0o644)
	require.NoError(t, err)

	outputPath := filepath.Join(tmpDir, "output.glx")

	err = importGEDCOM(gedcomPath, outputPath, "single", true, false, defaultShowFirstErrors)
	require.Error(t, err, "should fail with invalid GEDCOM content")
	require.Contains(t, err.Error(), "failed to import GEDCOM", "error should indicate GEDCOM import failure")
}

func TestImportGEDCOM_OutputDirectoryCreation(t *testing.T) {
	// Test that multi-file format creates nested directories correctly
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nested", "dir", "archive")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "multi", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should create nested output directories")

	// Verify nested directories were created
	info, err := os.Stat(outputPath)
	require.NoError(t, err, "nested output directory should exist")
	require.True(t, info.IsDir(), "output should be a directory")
}

func TestImportGEDCOM_OverwriteExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	// Create an existing file
	err := os.WriteFile(outputPath, []byte("existing content"), 0o644)
	require.NoError(t, err)

	err = importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should overwrite existing file")

	// Verify file was overwritten
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.NotContains(t, string(data), "existing content", "file should be overwritten")

	var glxFile lib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err, "overwritten file should contain valid GLX data")
}

func TestRunImport(t *testing.T) {
	// Test the runImport wrapper function
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	importOutput = outputPath
	importFormat = "single"
	importNoValidate = false
	importVerbose = false

	err := runImport(nil, []string{"testdata/gedcom/7.0/comprehensive-spec/maximal70.ged"})
	require.NoError(t, err, "runImport should successfully call importGEDCOM")

	_, err = os.Stat(outputPath)
	require.NoError(t, err, "output file should exist")
}

func TestImportGEDCOM_GEDCOM551(t *testing.T) {
	// Test importing GEDCOM 5.5.1 format
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	// Shakespeare is GEDCOM 5.5.1
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import GEDCOM 5.5.1")

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var glxFile lib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err)
	require.NotEmpty(t, glxFile.Persons, "should have imported persons from GEDCOM 5.5.1")
}

func TestImportGEDCOM_GEDCOM70(t *testing.T) {
	// Test importing GEDCOM 7.0 format
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should successfully import GEDCOM 7.0")

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var glxFile lib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err)
	require.NotEmpty(t, glxFile.Persons, "should have imported persons from GEDCOM 7.0")
}

func TestImportGEDCOM_StatisticsOutput(t *testing.T) {
	// This test verifies that the function runs successfully
	// The statistics are printed to stdout, which we're not capturing here
	// but the test ensures the code path executes without errors
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.glx")

	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", outputPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "should complete import and print statistics without error")
}

func TestImportGEDCOM_MultiFileEntityFiles(t *testing.T) {
	// Test that multi-file format creates individual entity files
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "archive")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.ged", outputPath, "multi", true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Check that person files were created
	personsDir := filepath.Join(outputPath, "persons")
	personFiles, err := os.ReadDir(personsDir)
	require.NoError(t, err)
	require.NotEmpty(t, personFiles, "should have created person files")

	// Verify at least one file has .glx extension
	hasGlxFile := false
	for _, file := range personFiles {
		if strings.HasSuffix(file.Name(), ".glx") {
			hasGlxFile = true

			break
		}
	}
	require.True(t, hasGlxFile, "person files should have .glx extension")
}
