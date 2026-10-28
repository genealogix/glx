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
	"bytes"
	"errors"
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
	return copyMediaFilesUsing(streams, archiveDir, mediaFiles, gedcomDir, func(copied, blobs, warnings int) { printImportMediaReport(streams, copied, blobs, warnings, verbose) })
}

// stageImportMedia keeps warnings immediate but defers the success count until
// both media and archive references have been published.
func stageImportMedia(archiveDir string, files []glxlib.MediaFileSource, sourceDir string, verbose bool) (func(), error) {
	streams := SystemIOStreams()
	report := func() {}
	err := copyMediaFilesUsing(streams, archiveDir, files, sourceDir, func(copied, blobs, warnings int) {
		report = func() { printImportMediaReport(streams, copied, blobs, warnings, verbose) }
	})

	return report, err
}

func copyMediaFilesUsing(streams *IOStreams, archiveDir string, mediaFiles []glxlib.MediaFileSource, gedcomDir string, report func(int, int, int)) error {
	if len(mediaFiles) == 0 {
		return nil
	}

	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	if err := os.MkdirAll(filesDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create media/files directory: %w", err)
	}

	var copyCount, blobCount, warnCount int
	reserved := glxlib.NewFilenameAllocator()

	for _, mf := range mediaFiles {
		if mf.ReuseExisting {
			continue
		}
		if !reserved.Reserve(mf.TargetFilename) {
			return fmt.Errorf("%w: duplicate media destination %s", os.ErrExist, mf.TargetFilename)
		}
		destPath := filepath.Join(filesDir, mf.TargetFilename)

		switch mf.SourceType {
		case glxlib.MediaSourceFile:
			if err := copyMediaFile(gedcomDir, mf.RelativePath, destPath); err != nil {
				if errors.Is(err, os.ErrExist) {
					return fmt.Errorf("media destination was occupied while copying %s: %w", mf.RelativePath, err)
				}
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
			if err := writeFileFrom(destPath, bytes.NewReader(decoded)); err != nil {
				if errors.Is(err, os.ErrExist) {
					return fmt.Errorf("media destination was occupied while writing BLOB %s: %w", mf.MediaID, err)
				}
				streams.Printf("Warning: could not write BLOB file %s: %v\n", destPath, err)
				warnCount++

				continue
			}
			blobCount++
		}
	}

	report(copyCount, blobCount, warnCount)

	return nil
}

func printImportMediaReport(streams *IOStreams, copyCount, blobCount, warnCount int, verbose bool) {
	if verbose || copyCount > 0 || blobCount > 0 {
		streams.Printf("  Media files: %d copied, %d blobs written", copyCount, blobCount)
		if warnCount > 0 {
			streams.Printf(", %d warnings", warnCount)
		}
		streams.Println("")
	}
}

// copyMediaFile copies a single media file from the GEDCOM source directory.
// It tries the path as-is first, then URL-decoded (for GEDCOM 7.0 percent-encoded paths).
// The source must stay inside gedcomDir: "../" traversal and absolute paths are
// rejected lexically, and a symlink inside gedcomDir that points outside it is
// refused at open time (see openWithin) rather than followed.
func copyMediaFile(gedcomDir, relativePath, destPath string) error {
	srcFile, err := openMediaSourceFile(gedcomDir, relativePath)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	return writeFileFrom(destPath, srcFile)
}

// openMediaSourceFile applies the same raw-then-decoded FILE resolution for
// both content matching and the subsequent copy.
func openMediaSourceFile(gedcomDir, relativePath string) (*os.File, error) {
	// Normalize backslashes to forward slashes for cross-platform compatibility
	// (belt-and-braces: MediaFileSource.RelativePath is already slash-normalized
	// at import, but direct callers may supply raw Windows path separators).
	normalized := strings.ReplaceAll(relativePath, "\\", "/")

	// Prevent path traversal attacks from GEDCOM FILE references
	srcPath := filepath.Join(gedcomDir, normalized)
	if !isPathWithin(srcPath, gedcomDir) {
		return nil, fmt.Errorf("path traversal detected in media reference %s: %w", relativePath, ErrPathEscapesDir)
	}
	src, err := openWithin(gedcomDir, srcPath)
	if err == nil {
		return src, nil
	}

	// Only fall back to URL-decoded path if the source file does not exist.
	// Other errors (permissions, disk full, etc.) should be returned immediately.
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("opening media file from %s: %w", srcPath, err)
	}

	// Try URL-decoded version (e.g., "CharlotteBront%C3%AB.jpg" -> "CharlotteBrontë.jpg")
	decoded, decodeErr := url.PathUnescape(normalized)
	if decodeErr != nil {
		return nil, fmt.Errorf("failed to decode media path %q: %w", normalized, decodeErr)
	}
	if decoded == normalized {
		return nil, fmt.Errorf("%w: %s", ErrMediaFileNotFound, srcPath)
	}

	decodedPath := filepath.Join(gedcomDir, decoded)
	if !isPathWithin(decodedPath, gedcomDir) {
		return nil, fmt.Errorf("path traversal detected in media reference %s: %w", relativePath, ErrPathEscapesDir)
	}
	src, err = openWithin(gedcomDir, decodedPath)
	if err == nil {
		return src, nil
	}
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", ErrMediaFileNotFound, decodedPath)
	}

	return nil, fmt.Errorf("opening media file from %s: %w", decodedPath, err)
}

// writeFileFrom streams src into a newly created file at dst, removing the
// partial file if the copy or close fails.
func writeFileFrom(dst string, src io.Reader) error {
	dst = filepath.Clean(dst)
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dstFile, src)
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
