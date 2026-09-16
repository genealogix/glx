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
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	glxlib "github.com/genealogix/glx/go-glx"
)

// maxGEDZIPEntries is a security limit on per-archive entry count to mitigate
// inode/syscall exhaustion DoS from archives containing huge numbers of tiny
// (including zero-byte) entries. The 100k threshold is intentionally high
// enough for very large legitimate genealogy datasets, while still bounding
// extraction work. Declared as var (not const) so tests can lower the cap
// without building a 100k-entry fixture.
var maxGEDZIPEntries = 100_000

// maxGEDZIPEntryBytes is a security limit on the decompressed size of a single
// archive entry, mitigating decompression-bomb DoS where a tiny compressed
// entry expands to exhaust disk during extraction. 512 MiB comfortably exceeds
// any legitimate single GEDCOM or media file while still bounding extraction
// work. Declared as var (not const) so tests can lower the cap without writing
// a multi-hundred-MiB fixture.
var maxGEDZIPEntryBytes int64 = 512 << 20

// importGEDZIP opens a .gdz archive and delegates to glxlib.ImportGEDZIP via
// Go's fs.FS abstraction without extracting the entire archive to a temporary
// directory.
func importGEDZIP(gedzipPath, outputPath, format string, validate, verbose bool, showFirstErrors int, out ...io.Writer) error {
	output := resolveOutputWriter(out...)

	if verbose {
		_, _ = fmt.Fprintf(output, "Extracting GEDZIP archive: %s\n", gedzipPath)
	}

	zr, err := zip.OpenReader(filepath.Clean(gedzipPath))
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return fmt.Errorf("%w: %s: %w", ErrGEDCOMFileNotFound, gedzipPath, err)
		case errors.Is(err, zip.ErrInsecurePath):
			return fmt.Errorf("%w: %s: %w", ErrGEDZIPInvalidEntry, gedzipPath, err)
		case errors.Is(err, zip.ErrFormat), errors.Is(err, zip.ErrChecksum):
			return fmt.Errorf("%w: %s: %w", ErrGEDZIPNotValidArchive, gedzipPath, err)
		default:
			return fmt.Errorf("opening gedzip archive %s: %w", gedzipPath, err)
		}
	}
	defer func() { _ = zr.Close() }()

	if len(zr.File) > maxGEDZIPEntries {
		return fmt.Errorf("%w: %d entries (limit %d)", ErrGEDZIPTooManyEntries, len(zr.File), maxGEDZIPEntries)
	}

	glx, result, err := glxlib.ImportGEDZIP(zr, nil)
	if err != nil {
		return err
	}

	// Surface any non-standard GEDCOM entry or other import warnings
	if result != nil {
		for _, w := range result.Statistics.Warnings {
			if w.Tag == "GEDZIP" {
				_, _ = fmt.Fprintf(output, "Warning: %s\n", w.Message)
			}
		}
	}

	if format == FormatSingle {
		return importGEDZIPToSingleFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, zr, output)
	}

	return importGEDZIPToMultiFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, zr, output)
}

// importGEDZIPToSingleFile writes the imported GLX data to a single file and copies media
func importGEDZIPToSingleFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, out io.Writer) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing single-file archive: %s\n", outputPath)
	}

	outputPath = ensureGLXExtension(outputPath)

	if err := writeSingleFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	archiveDir := filepath.Dir(outputPath)
	if err := copyMediaFilesFromFS(SystemIOStreams(), archiveDir, mediaFiles, bundle, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	printSuccessSingleFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// importGEDZIPToMultiFile writes the imported GLX data to a multi-file directory and copies media
func importGEDZIPToMultiFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, out io.Writer) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing multi-file archive: %s\n", outputPath)
	}

	if err := writeMultiFileArchive(outputPath, glx, validate); err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	if err := copyMediaFilesFromFS(SystemIOStreams(), outputPath, mediaFiles, bundle, verbose); err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	printSuccessMultiFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// copyMediaFilesFromFS copies media files from an fs.FS bundle into the archive's media/files/ directory.
func copyMediaFilesFromFS(streams *IOStreams, archiveDir string, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, verbose bool) error {
	if len(mediaFiles) == 0 {
		return nil
	}

	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	if err := os.MkdirAll(filesDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create media/files directory: %w", err)
	}

	var copyCount, blobCount, warnCount int

	for _, mf := range mediaFiles {
		destPath := filepath.Join(filesDir, mf.TargetFilename)

		switch mf.SourceType {
		case glxlib.MediaSourceFile:
			if mf.MemberPath == "" {
				streams.Printf("Warning: could not copy media file %s: unresolved or invalid reference\n", mf.RelativePath)
				warnCount++

				continue
			}

			if err := copyMemberFile(bundle, mf.MemberPath, destPath); err != nil {
				return err
			}
			copyCount++

		case glxlib.MediaSourceBlob:
			decoded, err := glxlib.DecodeGEDCOMBlob(mf.BlobData)
			if err != nil {
				streams.Printf("Warning: could not decode BLOB for %s: %v\n", mf.MediaID, err)
				warnCount++

				continue
			}
			if err := os.WriteFile(destPath, decoded, filePermissions); err != nil {
				streams.Printf("Warning: could not write BLOB file %s: %v\n", destPath, err)
				warnCount++

				continue
			}
			blobCount++
		}
	}

	if verbose || copyCount > 0 || blobCount > 0 {
		streams.Printf("  Media files: %d copied, %d blobs written", copyCount, blobCount)
		if warnCount > 0 {
			streams.Printf(", %d warnings", warnCount)
		}
		streams.Println("")
	}

	return nil
}

var errGEDZIPSymlinkEntry = errors.New("skipping symlink entry")

func copyMemberFile(bundle fs.FS, memberPath, destPath string) error {
	src, err := bundle.Open(memberPath)
	if err != nil {
		if errors.Is(err, zip.ErrAlgorithm) {
			return fmt.Errorf("%w: %q: %w", ErrGEDZIPUnsupportedAlgorithm, memberPath, err)
		}

		return fmt.Errorf("opening zip entry %q: %w", memberPath, err)
	}
	defer func() { _ = src.Close() }()

	var zipFiles []*zip.File
	if zr, ok := bundle.(*zip.Reader); ok {
		zipFiles = zr.File
	} else if zrc, ok := bundle.(*zip.ReadCloser); ok {
		zipFiles = zrc.File
	}
	for _, f := range zipFiles {
		if f.Name == memberPath {
			if f.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%w: %q", errGEDZIPSymlinkEntry, memberPath)
			}

			break
		}
	}

	if err := os.MkdirAll(filepath.Dir(destPath), dirPermissions); err != nil {
		return fmt.Errorf("creating directory for %q: %w", memberPath, err)
	}

	return writeStreamEntry(src, memberPath, destPath)
}

// writeZipEntry copies one ZIP entry to destPath, rejecting any entry whose
// decompressed size exceeds maxGEDZIPEntryBytes. On a copy or close failure, or
// when that size limit is exceeded, the partially written destination is
// removed so the next caller cannot observe a truncated or oversized file.
func writeZipEntry(f *zip.File, destPath string) error {
	src, err := f.Open()
	if err != nil {
		if errors.Is(err, zip.ErrAlgorithm) {
			return fmt.Errorf("%w: %q: %w", ErrGEDZIPUnsupportedAlgorithm, f.Name, err)
		}

		return fmt.Errorf("opening zip entry %q: %w", f.Name, err)
	}
	defer func() { _ = src.Close() }()

	if err := os.MkdirAll(filepath.Dir(destPath), dirPermissions); err != nil {
		return fmt.Errorf("creating directory for %q: %w", f.Name, err)
	}

	return writeStreamEntry(src, f.Name, destPath)
}

func writeStreamEntry(src io.Reader, name, destPath string) error {
	dst, err := os.OpenFile(filepath.Clean(destPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return fmt.Errorf("creating destination file for %q: %w", name, err)
	}

	limitedSrc := &entrySizeLimitReader{r: src, remaining: maxGEDZIPEntryBytes + 1}
	_, copyErr := io.Copy(dst, limitedSrc)
	closeErr := dst.Close()

	if copyErr != nil {
		_ = os.Remove(destPath)
		if errors.Is(copyErr, ErrGEDZIPEntryTooLarge) {
			return fmt.Errorf("extracting zip entry %q: %w (limit %d bytes)", name, ErrGEDZIPEntryTooLarge, maxGEDZIPEntryBytes)
		}

		return fmt.Errorf("extracting zip entry %q: %w", name, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(destPath)

		return fmt.Errorf("closing destination file for %q: %w", name, closeErr)
	}

	return nil
}

// entrySizeLimitReader wraps an io.Reader and returns ErrGEDZIPEntryTooLarge
// once it has read `remaining` bytes from r (and on every Read thereafter).
// To accept entries of up to N bytes, callers initialize remaining to N+1: the
// sentinel then fires only after the (N+1)th byte, so an entry of exactly N
// bytes passes through unchanged.
type entrySizeLimitReader struct {
	r         io.Reader
	remaining int64
}

func (l *entrySizeLimitReader) Read(p []byte) (int, error) {
	if l.remaining <= 0 {
		return 0, ErrGEDZIPEntryTooLarge
	}

	if int64(len(p)) > l.remaining {
		p = p[:l.remaining]
	}

	n, err := l.r.Read(p)
	l.remaining -= int64(n)

	if l.remaining == 0 {
		// Budget exhausted — see the type doc for the N+1 convention.
		return n, ErrGEDZIPEntryTooLarge
	}

	return n, err
}
