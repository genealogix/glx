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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// copyMediaFiles copies or writes media files into the archive's media/files/ directory.
// gedcomDir is the source directory for resolving relative FILE paths.
// archiveDir is the root of the output archive.
// Missing source files produce warnings on stderr, not fatal errors.
func copyMediaFiles(archiveDir string, mediaFiles []glxlib.MediaFileSource, gedcomDir string, verbose bool) error {
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
			if err := copyMediaFile(gedcomDir, mf.RelativePath, destPath); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not copy media file %s: %v\n", mf.RelativePath, err)
				warnCount++

				continue
			}
			copyCount++

		case glxlib.MediaSourceBlob:
			decoded, err := decodeGEDCOMBlob(mf.BlobData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not decode BLOB for %s: %v\n", mf.MediaID, err)
				warnCount++

				continue
			}
			if err := os.WriteFile(destPath, decoded, filePermissions); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not write BLOB file %s: %v\n", destPath, err)
				warnCount++

				continue
			}
			blobCount++
		}
	}

	if verbose || copyCount > 0 || blobCount > 0 {
		fmt.Printf("  Media files: %d copied, %d blobs written", copyCount, blobCount)
		if warnCount > 0 {
			fmt.Printf(", %d warnings", warnCount)
		}
		fmt.Println()
	}

	return nil
}

// copyMediaFile copies a single media file from the GEDCOM source directory.
// It tries the path as-is first, then URL-decoded (for GEDCOM 7.0 percent-encoded paths).
func copyMediaFile(gedcomDir, relativePath, destPath string) error {
	// Normalize backslashes to forward slashes for cross-platform compatibility
	normalized := strings.ReplaceAll(relativePath, "\\", "/")

	srcPath := filepath.Join(gedcomDir, normalized)
	err := copyFile(srcPath, destPath)
	if err == nil {
		return nil
	}

	// Only fall back to URL-decoded path if the source file does not exist.
	// Other errors (permissions, disk full, etc.) should be returned immediately.
	if !os.IsNotExist(err) {
		return fmt.Errorf("copying media file from %s: %w", srcPath, err)
	}

	// Try URL-decoded version (e.g., "CharlotteBront%C3%AB.jpg" -> "CharlotteBrontë.jpg")
	decoded, decodeErr := url.PathUnescape(normalized)
	if decodeErr != nil {
		return fmt.Errorf("failed to decode media path %q: %w", normalized, decodeErr)
	}
	if decoded == normalized {
		return fmt.Errorf("%w: %s", ErrMediaFileNotFound, srcPath)
	}

	decodedPath := filepath.Join(gedcomDir, decoded)
	err = copyFile(decodedPath, destPath)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrMediaFileNotFound, decodedPath)
	}

	return fmt.Errorf("copying media file from %s: %w", decodedPath, err)
}

// copyFile copies a single file from src to dst using streaming I/O.
func copyFile(src, dst string) error {
	src = filepath.Clean(src)
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dst = filepath.Clean(dst)
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dstFile, srcFile)
	closeErr := dstFile.Close()

	if copyErr != nil {
		_ = os.Remove(dst) // best-effort cleanup of corrupted file
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst) // best-effort cleanup of truncated file
		return closeErr
	}
	return nil
}

// decodeGEDCOMBlob decodes GEDCOM 5.5.1 BLOB-encoded text to raw bytes.
// GEDCOM BLOB uses a custom encoding where each character's value minus 0x2E ('.')
// gives a 6-bit value. Groups of 4 characters encode 3 bytes.
func decodeGEDCOMBlob(blobText string) ([]byte, error) {
	// Strip whitespace and newlines
	cleaned := strings.NewReplacer("\n", "", "\r", "", " ", "").Replace(blobText)

	if len(cleaned) == 0 {
		return nil, ErrEmptyBlobData
	}

	if len(cleaned)%4 != 0 {
		return nil, fmt.Errorf("%w: %d (must be multiple of 4)", ErrInvalidBlobLength, len(cleaned))
	}

	result := make([]byte, 0, len(cleaned)*3/4)
	for i := 0; i < len(cleaned); i += 4 {
		// Validate each character is in valid GEDCOM BLOB range (0x2E '.' to 0x6D 'm')
		// This gives 6-bit values (0-63) after subtracting 0x2E
		for j := 0; j < 4; j++ {
			char := cleaned[i+j]
			if char < '.' || char > 'm' {
				return nil, fmt.Errorf("invalid BLOB character at position %d: %q (must be in range '.' to 'm')", i+j, char)
			}
		}

		b1 := cleaned[i] - '.'
		b2 := cleaned[i+1] - '.'
		b3 := cleaned[i+2] - '.'
		b4 := cleaned[i+3] - '.'

		result = append(result, (b1<<2)|(b2>>4)) //nolint:mnd // well-known base64 bit shifts
		result = append(result, (b2<<4)|(b3>>2)) //nolint:mnd // well-known base64 bit shifts
		result = append(result, (b3<<6)|b4)      //nolint:mnd // well-known base64 bit shifts
	}

	return result, nil
}
