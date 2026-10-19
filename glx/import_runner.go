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
	"fmt"
	"io"
	"os"
	"path/filepath"

	glxlib "github.com/genealogix/glx/go-glx"
)

// resolveOutputWriter returns the first provided non-nil writer, or os.Stdout
// when no optional writer is supplied.
func resolveOutputWriter(out ...io.Writer) io.Writer {
	if len(out) > 0 && out[0] != nil {
		return out[0]
	}

	return os.Stdout
}

// importGEDCOM imports a GEDCOM or GEDZIP file and converts it to GLX format.
// GEDZIP archives (.gdz) are detected by extension and routed through
// importGEDZIP, which extracts to a temp directory before delegating back here.
//
// Referenced media is handled by reference type. Relative FILE references are
// copied from the GEDCOM's source directory into media/files/, and inline BLOB
// data is decoded and written there. URL and absolute FILE references are left
// as-is on the media URI — neither copied nor downloaded. For single-file output
// media/files/ is a sibling directory of the output file; for multi-file output
// it lives within the archive layout.
func importGEDCOM(gedcomPath, outputPath, format string, validate, verbose bool, showFirstErrors int, out ...io.Writer) error {
	output := resolveOutputWriter(out...)

	// Validate format flag
	if format != FormatSingle && format != FormatMulti {
		return fmt.Errorf("%w: %s", ErrInvalidFormat, format)
	}

	if isGEDZIPPath(gedcomPath) {
		return importGEDZIP(gedcomPath, outputPath, format, validate, verbose, showFirstErrors, out...)
	}

	// Check if GEDCOM file exists
	if !fileExists(gedcomPath) {
		return fmt.Errorf("%w: %s", ErrGEDCOMFileNotFound, gedcomPath)
	}

	// Import GEDCOM file
	if verbose {
		_, _ = fmt.Fprintf(output, "Importing GEDCOM file: %s\n", gedcomPath)
	}

	// Open GEDCOM file
	gedcomPath = filepath.Clean(gedcomPath)
	gedcomFile, err := os.Open(gedcomPath)
	if err != nil {
		return fmt.Errorf("failed to open GEDCOM file: %w", err)
	}
	defer func() { _ = gedcomFile.Close() }()

	// Import GEDCOM from reader
	glx, result, err := glxlib.ImportGEDCOM(gedcomFile, nil)
	if err != nil {
		return fmt.Errorf("failed to import GEDCOM: %w", err)
	}

	// Determine GEDCOM source directory for resolving relative media file paths
	gedcomDir := filepath.Dir(gedcomPath)
	if err := allocateImportMediaDestinations(glx, result.MediaFiles, importMediaDir(outputPath, format), gedcomDir, nil); err != nil {
		return fmt.Errorf("reserving media destinations: %w", err)
	}

	// Serialize based on format
	if format == FormatSingle {
		return importToSingleFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, gedcomDir, output)
	}

	return importToMultiFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, gedcomDir, output)
}

// importToSingleFile writes the imported GLX data to a single file
func importToSingleFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, gedcomDir string, out io.Writer) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing single-file archive: %s\n", outputPath)
	}

	// Ensure .glx extension
	outputPath = ensureGLXExtension(outputPath)
	archiveDir := filepath.Dir(outputPath)
	if _, err := importMediaFilesDir(archiveDir); err != nil {
		return fmt.Errorf("inspecting media destination: %w", err)
	}
	if err := os.MkdirAll(archiveDir, dirPermissions); err != nil {
		return fmt.Errorf("creating archive directory: %w", err)
	}
	stageDir, err := os.MkdirTemp(archiveDir, ".glx-stage-*")
	if err != nil {
		return fmt.Errorf("creating media staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()
	if err := copyMediaFiles(SystemIOStreams(), stageDir, mediaFiles, gedcomDir, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}
	tempFile, err := os.CreateTemp(archiveDir, ".glx-single-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary archive: %w", err)
	}
	tempPath := tempFile.Name()
	_ = tempFile.Close()
	defer func() { _ = os.Remove(tempPath) }()
	if err := writeSingleFileArchive(tempPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}
	rollbackMedia, err := commitStagedMediaWithRollback(stageDir, archiveDir)
	if err != nil {
		return fmt.Errorf("failed to commit media files: %w", err)
	}
	if err := validateReusedMedia(archiveDir, mediaFiles, gedcomDir, nil); err != nil {
		rollbackMedia()

		return fmt.Errorf("validating reused media: %w", err)
	}
	rollbackNormalization, err := normalizeImportMediaDir(archiveDir)
	if err != nil {
		rollbackMedia()

		return fmt.Errorf("normalizing media directory: %w", err)
	}
	if err := robustRename(tempPath, outputPath); err != nil {
		rollbackNormalization()
		rollbackMedia()

		return fmt.Errorf("publishing archive: %w", err)
	}

	printSuccessSingleFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// importToMultiFile writes the imported GLX data to a multi-file directory
func importToMultiFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, gedcomDir string, out io.Writer) error {
	return importToMultiFileUsing(glx, outputPath, validate, verbose, showFirstErrors, mediaFiles, gedcomDir, out, writeFilesToDir)
}

// importToMultiFileUsing permits deterministic metadata-write failure tests.
func importToMultiFileUsing(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, gedcomDir string, out io.Writer, writeFiles func(string, map[string][]byte) error) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing multi-file archive: %s\n", outputPath)
	}
	if _, err := importMediaFilesDir(outputPath); err != nil {
		return fmt.Errorf("inspecting media destination: %w", err)
	}

	files, err := serializeMultiFileArchive(glx, validate)
	if err != nil {
		return formatValidationError(err, showFirstErrors)
	}
	if err := os.MkdirAll(outputPath, dirPermissions); err != nil {
		return fmt.Errorf("creating archive directory: %w", err)
	}
	stageDir, err := os.MkdirTemp(outputPath, ".glx-stage-*")
	if err != nil {
		return fmt.Errorf("creating media staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()
	if err := copyMediaFiles(SystemIOStreams(), stageDir, mediaFiles, gedcomDir, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}
	rollbackMedia, err := commitStagedMediaWithRollback(stageDir, outputPath)
	if err != nil {
		return fmt.Errorf("failed to commit media files: %w", err)
	}
	if err := validateReusedMedia(outputPath, mediaFiles, gedcomDir, nil); err != nil {
		rollbackMedia()

		return fmt.Errorf("validating reused media: %w", err)
	}
	_, err = normalizeImportMediaDir(outputPath)
	if err != nil {
		rollbackMedia()

		return fmt.Errorf("normalizing media directory: %w", err)
	}
	if err := writeFiles(outputPath, files); err != nil {
		// writeFiles can publish some entity files before a later write fails.
		// Keep the media those files may already reference.
		return err
	}

	printSuccessMultiFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}
