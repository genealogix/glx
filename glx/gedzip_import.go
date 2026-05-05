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
	"os"
	"path"
	"path/filepath"
	"strings"
)

// gedzipGedcomEntry is the canonical name of the GEDCOM file inside a GEDZIP
// archive, as defined by the GEDZIP specification.
const gedzipGedcomEntry = "gedcom.ged"

// maxGEDZIPEntries caps the per-archive entry count to prevent inode/syscall
// DoS from archives with millions of zero-byte entries. The largest plausible
// genealogy archive (a 100k-person tree with several media items per person)
// would still fit comfortably.
const maxGEDZIPEntries = 100_000

// importGEDZIP extracts a .gdz archive into a temporary directory and delegates
// to the existing GEDCOM import pipeline. The temp directory is removed when
// the function returns, regardless of outcome.
func importGEDZIP(gedzipPath, outputPath, format string, validate, verbose bool, showFirstErrors int) error {
	if verbose {
		fmt.Printf("Extracting GEDZIP archive: %s\n", gedzipPath)
	}

	zr, err := zip.OpenReader(filepath.Clean(gedzipPath))
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return fmt.Errorf("%w: %s: %w", ErrGEDCOMFileNotFound, gedzipPath, err)
		case errors.Is(err, zip.ErrFormat),
			errors.Is(err, zip.ErrAlgorithm),
			errors.Is(err, zip.ErrChecksum),
			errors.Is(err, zip.ErrInsecurePath):
			return fmt.Errorf("%w: %s: %w", ErrGEDZIPNotValidArchive, gedzipPath, err)
		default:
			return fmt.Errorf("opening gedzip archive %s: %w", gedzipPath, err)
		}
	}
	defer func() { _ = zr.Close() }()

	if len(zr.File) > maxGEDZIPEntries {
		return fmt.Errorf("%w: %d entries (limit %d)", ErrGEDZIPTooManyEntries, len(zr.File), maxGEDZIPEntries)
	}

	if !hasGedcomEntry(zr.File) {
		return fmt.Errorf("%w: %s", ErrGEDZIPMissingGedcom, gedzipPath)
	}

	tempDir, err := os.MkdirTemp("", "glx-gedzip-*")
	if err != nil {
		return fmt.Errorf("creating temp directory for GEDZIP extraction: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if err := extractGEDZIP(zr.File, tempDir); err != nil {
		return err
	}

	return importGEDCOM(filepath.Join(tempDir, gedzipGedcomEntry), outputPath, format, validate, verbose, showFirstErrors)
}

func hasGedcomEntry(files []*zip.File) bool {
	for _, f := range files {
		if f.Name == gedzipGedcomEntry {
			return true
		}
	}

	return false
}

// extractGEDZIP writes each file entry into destDir, skipping directory and
// symlink entries (the latter to prevent zip-symlink-slip attacks where a
// symlink target could redirect a later entry's write outside destDir).
// Duplicate entries — defined as two entries whose cleaned, case-folded
// destination paths collide — are rejected. This catches both case-only
// variants ("gedcom.ged" vs "Gedcom.GED") that overwrite on Windows NTFS and
// the default macOS APFS configuration, AND dot-segment variants
// ("gedcom.ged" vs "media/../gedcom.ged") that path.Clean folds to the same
// destination. Without this, an attacker could hijack gedcom.ged after
// hasGedcomEntry has already approved the archive.
func extractGEDZIP(files []*zip.File, destDir string) error {
	seen := make(map[string]struct{}, len(files))
	for _, f := range files {
		dest, err := safeExtractPath(destDir, f.Name)
		if err != nil {
			return err
		}

		key := strings.ToLower(dest)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("%w: %q", ErrGEDZIPDuplicateEntry, f.Name)
		}
		seen[key] = struct{}{}

		if f.FileInfo().IsDir() {
			continue
		}

		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), dirPermissions); err != nil {
			return fmt.Errorf("creating directory for %q: %w", f.Name, err)
		}

		if err := writeZipEntry(f, dest); err != nil {
			return err
		}
	}

	return nil
}

// safeExtractPath rejects ZIP entry names that could escape destDir during
// extraction. The layered checks guard distinct attack surfaces: forward and
// backslash absolute prefixes (path.IsAbs only sees the spec-mandated forward
// slash form), Windows volume prefixes (e.g. "C:\\"), and any cleaned path that
// still resolves to "..". The final isPathWithin check catches cases the
// per-prefix checks miss after platform-specific path joining.
func safeExtractPath(destDir, entryName string) (string, error) {
	if entryName == "" {
		return "", fmt.Errorf("%w: empty entry name", ErrGEDZIPInvalidEntry)
	}

	// ZIP entries use forward slashes per APPNOTE 4.4.17.1; backslashes and
	// embedded NULs are not legal and cause platform-specific anomalies.
	if strings.ContainsRune(entryName, 0) {
		return "", fmt.Errorf("%w: NUL byte in entry name %q", ErrGEDZIPInvalidEntry, entryName)
	}
	if strings.Contains(entryName, `\`) {
		return "", fmt.Errorf("%w: backslash in entry name %q", ErrGEDZIPInvalidEntry, entryName)
	}

	if path.IsAbs(entryName) || strings.HasPrefix(entryName, "/") {
		return "", fmt.Errorf("%w: absolute path %q", ErrGEDZIPInvalidEntry, entryName)
	}
	if filepath.VolumeName(filepath.FromSlash(entryName)) != "" {
		return "", fmt.Errorf("%w: volume-prefixed path %q", ErrGEDZIPInvalidEntry, entryName)
	}

	cleaned := path.Clean(entryName)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("%w: path traversal in %q", ErrGEDZIPInvalidEntry, entryName)
	}

	dest := filepath.Join(destDir, filepath.FromSlash(cleaned))
	if !isPathWithin(dest, destDir) {
		return "", fmt.Errorf("%w: %q escapes destination", ErrGEDZIPInvalidEntry, entryName)
	}

	return dest, nil
}

// writeZipEntry copies one ZIP entry to destPath. On any copy or close failure
// the partially written destination is removed so the next caller cannot
// observe a truncated file.
func writeZipEntry(f *zip.File, destPath string) error {
	src, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip entry %q: %w", f.Name, err)
	}
	defer func() { _ = src.Close() }()

	dst, err := os.OpenFile(filepath.Clean(destPath), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return fmt.Errorf("creating destination file for %q: %w", f.Name, err)
	}

	// #nosec G110 -- decompressed-size cap is intentionally deferred; tracked in #775
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()

	if copyErr != nil {
		_ = os.Remove(destPath)

		return fmt.Errorf("extracting zip entry %q: %w", f.Name, copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(destPath)

		return fmt.Errorf("closing destination file for %q: %w", f.Name, closeErr)
	}

	return nil
}
