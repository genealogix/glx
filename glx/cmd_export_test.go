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

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestParseGEDCOMVersion(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		want    glxlib.GEDCOMVersion
		wantErr bool
	}{
		{name: "551 short", format: "551", want: glxlib.GEDCOM551},
		{name: "551 dotted", format: "5.5.1", want: glxlib.GEDCOM551},
		{name: "70 short", format: "70", want: glxlib.GEDCOM70},
		{name: "70 dotted", format: "7.0", want: glxlib.GEDCOM70},
		{name: "whitespace trimmed", format: "  551  ", want: glxlib.GEDCOM551},
		{name: "invalid format", format: "99", wantErr: true},
		{name: "empty string", format: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGEDCOMVersion(tt.format)
			if tt.wantErr {
				require.Error(t, err)
				require.ErrorIs(t, err, ErrInvalidExportFormat)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEnsureGEDExtension(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "no extension", path: "output", want: "output.ged"},
		{name: "has .ged", path: "output.ged", want: "output.ged"},
		{name: "has .GED uppercase", path: "output.GED", want: "output.GED"},
		{name: "has .glx", path: "output.glx", want: "output.glx.ged"},
		{name: "nested path", path: "dir/sub/file", want: "dir/sub/file.ged"},
		{name: "nested with .ged", path: "dir/sub/file.ged", want: "dir/sub/file.ged"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureGEDExtension(tt.path)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestLoadGLXArchive_FileNotFound(t *testing.T) {
	_, err := loadGLXArchive("/nonexistent/path", false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInputNotFound)
}

func TestLoadGLXArchive_SingleFile(t *testing.T) {
	// First import a GEDCOM to create a single-file GLX archive
	tmpDir := t.TempDir()
	glxPath := filepath.Join(tmpDir, "archive.glx")

	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Now load it via loadGLXArchive
	glx, err := loadGLXArchive(glxPath, false)
	require.NoError(t, err)
	require.NotNil(t, glx)
	require.NotEmpty(t, glx.Persons, "loaded archive should have persons")
}

func TestLoadGLXArchive_Directory(t *testing.T) {
	// First import a GEDCOM to create a multi-file archive
	tmpDir := t.TempDir()
	archiveDir := filepath.Join(tmpDir, "archive")

	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", archiveDir, "multi", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Now load it via loadGLXArchive
	glx, err := loadGLXArchive(archiveDir, false)
	require.NoError(t, err)
	require.NotNil(t, glx)
	require.NotEmpty(t, glx.Persons, "loaded archive should have persons")
}

func TestExportToGEDCOM_SingleFileInput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a GLX archive first
	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Export to GEDCOM
	outputPath := filepath.Join(tmpDir, "output.ged")
	err = exportToGEDCOM(glxPath, outputPath, ExportFormat551, false)
	require.NoError(t, err)

	// Verify file was created with content
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.NotEmpty(t, data, "exported GEDCOM should have content")
	require.Contains(t, string(data), "0 HEAD", "should contain GEDCOM header")
}

func TestExportToGEDCOM_MultiFileInput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a multi-file GLX archive first
	archiveDir := filepath.Join(tmpDir, "archive")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", archiveDir, "multi", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Export to GEDCOM
	outputPath := filepath.Join(tmpDir, "output.ged")
	err = exportToGEDCOM(archiveDir, outputPath, ExportFormat551, false)
	require.NoError(t, err)

	// Verify file was created
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	require.Contains(t, string(data), "0 HEAD")
}

func TestExportToGEDCOM_AddsGEDExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a GLX archive first
	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Export without .ged extension
	outputPath := filepath.Join(tmpDir, "output")
	err = exportToGEDCOM(glxPath, outputPath, ExportFormat551, false)
	require.NoError(t, err)

	// Verify .ged extension was added
	expectedPath := outputPath + ".ged"
	_, err = os.Stat(expectedPath)
	require.NoError(t, err, "should add .ged extension if missing")
}

func TestExportToGEDCOM_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()

	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	outputPath := filepath.Join(tmpDir, "output.ged")
	err = exportToGEDCOM(glxPath, outputPath, "invalid", false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInvalidExportFormat)
}

func TestExportToGEDCOM_InputNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.ged")

	err := exportToGEDCOM("/nonexistent/path", outputPath, ExportFormat551, false)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrInputNotFound)
}

func TestExportToGEDCOM_OutputDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a GLX archive first
	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	// Export to a nested directory that doesn't exist yet
	outputPath := filepath.Join(tmpDir, "nested", "dir", "output.ged")
	err = exportToGEDCOM(glxPath, outputPath, ExportFormat551, false)
	require.NoError(t, err)

	_, err = os.Stat(outputPath)
	require.NoError(t, err, "should create nested output directories")
}

func TestExportToGEDCOM_VerboseMode(t *testing.T) {
	tmpDir := t.TempDir()

	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	outputPath := filepath.Join(tmpDir, "output.ged")
	err = exportToGEDCOM(glxPath, outputPath, ExportFormat551, true)
	require.NoError(t, err, "should succeed with verbose mode")

	_, err = os.Stat(outputPath)
	require.NoError(t, err)
}

func TestExportToGEDCOM_GEDCOM70(t *testing.T) {
	tmpDir := t.TempDir()

	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	outputPath := filepath.Join(tmpDir, "output.ged")
	err = exportToGEDCOM(glxPath, outputPath, ExportFormat70, false)
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.Contains(t, string(data), "0 HEAD")
}

func TestRunExport(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a GLX archive first
	glxPath := filepath.Join(tmpDir, "archive.glx")
	err := importGEDCOM("testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged", glxPath, "single", true, false, defaultShowFirstErrors)
	require.NoError(t, err, "setup: import should succeed")

	outputPath := filepath.Join(tmpDir, "output.ged")

	exportOutput = outputPath
	exportFormat = ExportFormat551
	exportVerbose = false

	err = runExport(nil, []string{glxPath})
	require.NoError(t, err, "runExport should successfully call exportToGEDCOM")

	_, err = os.Stat(outputPath)
	require.NoError(t, err, "output file should exist")
}
