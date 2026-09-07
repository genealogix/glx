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
// Missing source files produce warnings on stdout (suppressible via --quiet),
// not fatal errors.
func copyMediaFiles(streams *IOStreams, archiveDir string, mediaFiles []glxlib.MediaFileSource, gedcomDir string, verbose bool) error {
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
				streams.Printf("Warning: could not copy media file %s: %v\n", mf.RelativePath, err)
				warnCount++

				continue
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

// isPathWithin checks whether child is contained within the parent directory.
// Uses filepath.Rel to handle edge cases like parent being "." or "/".
func isPathWithin(child, parent string) bool {
	absChild, err := filepath.Abs(filepath.Clean(child))
	if err != nil {
		return false
	}
	absParent, err := filepath.Abs(filepath.Clean(parent))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return false
	}

	return !strings.HasPrefix(rel, "..") && rel != "."
}

// copyMediaFile copies a single media file from the GEDCOM source directory.
// It tries the path as-is first, then URL-decoded (for GEDCOM 7.0 percent-encoded paths).
func copyMediaFile(gedcomDir, relativePath, destPath string) error {
	// Normalize backslashes to forward slashes for cross-platform compatibility
	normalized := strings.ReplaceAll(relativePath, "\\", "/")

	// Prevent path traversal attacks from GEDCOM FILE references
	srcPath := filepath.Join(gedcomDir, normalized)
	if !isPathWithin(srcPath, gedcomDir) {
		return fmt.Errorf("path traversal detected in media reference: %s", relativePath)
	}
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
	if !isPathWithin(decodedPath, gedcomDir) {
		return fmt.Errorf("path traversal detected in media reference: %s", relativePath)
	}
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
