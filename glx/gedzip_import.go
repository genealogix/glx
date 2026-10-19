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
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
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

	limitedBundle := &sizeLimitedFS{inner: zr}
	glx, result, err := glxlib.ImportGEDZIP(limitedBundle, nil)
	if err != nil {
		// sizeLimitedFile already names the offending entry and the limit, so
		// the error is returned as-is rather than re-wrapped against a guessed
		// entry name (the GEDCOM need not be "gedcom.ged" — see the fallback
		// discovery in glxlib.ImportGEDZIP).
		return err
	}

	// Surface any non-standard GEDCOM entry or other import warnings
	for _, w := range result.Statistics.Warnings {
		if w.Tag == glxlib.WarningTagGEDZIP {
			_, _ = fmt.Fprintf(output, "Warning: %s\n", w.Message)
		}
	}
	if err := allocateImportMediaDestinations(glx, result.MediaFiles, importMediaDir(outputPath, format), "", limitedBundle); err != nil {
		return fmt.Errorf("reserving media destinations: %w", err)
	}

	if format == FormatSingle {
		return importGEDZIPToSingleFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, limitedBundle, output)
	}

	return importGEDZIPToMultiFile(glx, outputPath, validate, verbose, showFirstErrors, result.MediaFiles, limitedBundle, output)
}

// sizeLimitedFS wraps an fs.FS to enforce maxGEDZIPEntryBytes on any opened file.
type sizeLimitedFS struct {
	inner fs.FS
}

func (s *sizeLimitedFS) Unwrap() fs.FS {
	return s.inner
}

func (s *sizeLimitedFS) Open(name string) (fs.File, error) {
	f, err := s.inner.Open(name)
	if err != nil {
		return nil, err
	}

	return &sizeLimitedFile{
		File: f,
		name: name,
		r:    &entrySizeLimitReader{r: f, remaining: maxGEDZIPEntryBytes + 1},
	}, nil
}

type sizeLimitedFile struct {
	fs.File
	name string
	r    io.Reader
}

// Read enforces the per-entry decompressed size limit. The limit error is
// annotated here, where the entry name is known, so callers upstack do not have
// to guess which entry blew the budget.
func (s *sizeLimitedFile) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	if errors.Is(err, ErrGEDZIPEntryTooLarge) {
		return n, fmt.Errorf("extracting zip entry %q: %w (limit %d bytes)", s.name, ErrGEDZIPEntryTooLarge, maxGEDZIPEntryBytes)
	}

	return n, err
}

// importGEDZIPToSingleFile writes the imported GLX data to a single file and copies media
func importGEDZIPToSingleFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, out io.Writer) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing single-file archive: %s\n", outputPath)
	}

	outputPath = ensureGLXExtension(outputPath)
	archiveDir := filepath.Dir(outputPath)
	if _, err := importMediaFilesDir(archiveDir); err != nil {
		return fmt.Errorf("inspecting media destination: %w", err)
	}

	stageDir, err := os.MkdirTemp(archiveDir, ".glx-stage-*")
	if err != nil {
		return fmt.Errorf("creating staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()

	streams := SystemIOStreams()
	copyCount, blobCount, warnCount, err := stageMediaFilesFromFS(streams, stageDir, mediaFiles, bundle)
	if err != nil {
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

	// Media is committed before the archive is published so that any media
	// failure leaves an existing target archive untouched.
	rollbackMedia, err := commitStagedMediaWithRollback(stageDir, archiveDir)
	if err != nil {
		return fmt.Errorf("failed to commit media files: %w", err)
	}
	if err := validateReusedMedia(archiveDir, mediaFiles, "", bundle); err != nil {
		rollbackMedia()

		return fmt.Errorf("validating reused media: %w", err)
	}
	rollbackNormalization, err := normalizeImportMediaDir(archiveDir)
	if err != nil {
		rollbackMedia()

		return fmt.Errorf("normalizing media directory: %w", err)
	}

	if verbose || copyCount > 0 || blobCount > 0 {
		streams.Printf("  Media files: %d copied, %d blobs written", copyCount, blobCount)
		if warnCount > 0 {
			streams.Printf(", %d warnings", warnCount)
		}
		streams.Println("")
	}

	if err := robustRename(tempPath, outputPath); err != nil {
		rollbackNormalization()
		rollbackMedia()

		return fmt.Errorf("writing archive to %s: %w", outputPath, err)
	}

	printSuccessSingleFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// importGEDZIPToMultiFile writes the imported GLX data to a multi-file directory and copies media
func importGEDZIPToMultiFile(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, out io.Writer) error {
	return importGEDZIPToMultiFileUsing(glx, outputPath, validate, verbose, showFirstErrors, mediaFiles, bundle, out, writeFilesToDir)
}

// importGEDZIPToMultiFileUsing permits deterministic metadata-write failure tests.
func importGEDZIPToMultiFileUsing(glx *glxlib.GLXFile, outputPath string, validate, verbose bool, showFirstErrors int, mediaFiles []glxlib.MediaFileSource, bundle fs.FS, out io.Writer, writeFiles func(string, map[string][]byte) error) error {
	if verbose {
		_, _ = fmt.Fprintf(out, "Writing multi-file archive: %s\n", outputPath)
	}
	if _, err := importMediaFilesDir(outputPath); err != nil {
		return fmt.Errorf("inspecting media destination: %w", err)
	}

	// Serialize and validate up front, before anything touches the filesystem.
	// Media has to commit before the entity files are written (see below), so
	// the failure mode to rule out is a validation or serialization error
	// surfacing after media is already in place: get it out of the way while
	// the import is still side-effect free.
	files, err := serializeMultiFileArchive(glx, validate)
	if err != nil {
		return formatValidationError(err, showFirstErrors)
	}

	// Stage inside the archive directory itself: commitStagedMedia resolves
	// every path through an os.Root scoped to the archive, which only reaches
	// paths beneath it. The same filesystem allows exclusive hard-link commits.
	if err := os.MkdirAll(outputPath, dirPermissions); err != nil {
		return fmt.Errorf("creating directory for %s: %w", outputPath, err)
	}

	stageDir, mkdirErr := os.MkdirTemp(outputPath, ".glx-stage-*")
	if mkdirErr != nil {
		return fmt.Errorf("creating staging directory: %w", mkdirErr)
	}
	defer func() { _ = os.RemoveAll(stageDir) }()

	streams := SystemIOStreams()
	copyCount, blobCount, warnCount, err := stageMediaFilesFromFS(streams, stageDir, mediaFiles, bundle)
	if err != nil {
		return fmt.Errorf("failed to copy media files: %w", err)
	}

	// Media is committed before the archive is written, matching the
	// single-file path: a media failure then leaves an existing target archive
	// untouched instead of replacing it and only afterwards discovering that
	// the media URIs it now references cannot be satisfied. writeFilesToDir
	// only creates and overwrites the entity files it manages, so media/files/
	// survives the archive write.
	rollbackMedia, err := commitStagedMediaWithRollback(stageDir, outputPath)
	if err != nil {
		return fmt.Errorf("failed to commit media files: %w", err)
	}
	if err := validateReusedMedia(outputPath, mediaFiles, "", bundle); err != nil {
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

	if verbose || copyCount > 0 || blobCount > 0 {
		streams.Printf("  Media files: %d copied, %d blobs written", copyCount, blobCount)
		if warnCount > 0 {
			streams.Printf(", %d warnings", warnCount)
		}
		streams.Println("")
	}

	printSuccessMultiFile("imported", outputPath)
	printArchiveStatistics(glx)

	return nil
}

// stageMediaFilesFromFS copies media files into a temporary staging directory.
func stageMediaFilesFromFS(streams *IOStreams, stageDir string, mediaFiles []glxlib.MediaFileSource, bundle fs.FS) (int, int, int, error) {
	if len(mediaFiles) == 0 {
		return 0, 0, 0, nil
	}

	filesDir := filepath.Join(stageDir, glxlib.MediaFilesDir)
	if err := os.MkdirAll(filesDir, dirPermissions); err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create staging media/files directory: %w", err)
	}

	var copyCount, blobCount, warnCount int
	reserved := glxlib.NewFilenameAllocator()

	for _, mf := range mediaFiles {
		if mf.ReuseExisting {
			continue
		}
		if !reserved.Reserve(mf.TargetFilename) {
			return 0, 0, 0, fmt.Errorf("%w: duplicate staged media destination %s", os.ErrExist, mf.TargetFilename)
		}
		destPath := filepath.Join(filesDir, mf.TargetFilename)

		switch mf.SourceType {
		case glxlib.MediaSourceFile:
			if mf.MemberPath == "" {
				streams.Printf("Warning: could not copy media file %s: unresolved or invalid reference\n", mf.RelativePath)
				warnCount++

				continue
			}

			if err := copyMemberFile(bundle, mf.MemberPath, destPath); err != nil {
				if errors.Is(err, errGEDZIPNonRegularEntry) {
					// Not fatal, but the archive's media URI now points at a
					// file that was never written — say so rather than leaving
					// the user with a silently dangling reference.
					streams.Printf("Warning: could not copy media file %s: %v\n", mf.RelativePath, err)
					warnCount++

					continue
				}

				return 0, 0, 0, err
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
				streams.Printf("Warning: could not write BLOB file %s: %v\n", destPath, err)
				warnCount++

				continue
			}
			blobCount++
		}
	}

	return copyCount, blobCount, warnCount, nil
}

// commitStagedMedia moves the media staged under stageDir into targetDir's
// media/files/ directory.
//
// stageDir must live inside targetDir. Every path is then resolved through an
// os.Root scoped to targetDir, so no symlinked component can redirect a write
// outside the archive — not the leaf media/files/<name>, and not a "media" or
// "media/files" directory that a bare os.MkdirAll would happily follow. Each
// file lands by an exclusive hard link (or exclusive copy when links are not
// supported), so even a destination created after allocation is never replaced.
func commitStagedMedia(stageDir, targetDir string) error {
	_, err := commitStagedMediaWithRollback(stageDir, targetDir)

	return err
}

func commitStagedMediaWithRollback(stageDir, targetDir string) (func(), error) {
	return commitStagedMediaUsing(stageDir, targetDir, commitStagedFile)
}

type committedMediaFile struct {
	name string
	info os.FileInfo
}

// commitStagedMediaUsing permits deterministic failure injection in rollback tests.
func commitStagedMediaUsing(stageDir, targetDir string, publish func(*os.Root, string, string) (os.FileInfo, error)) (rollback func(), commitErr error) {
	stagedFilesDir := filepath.Join(stageDir, glxlib.MediaFilesDir)
	entries, err := os.ReadDir(stagedFilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return func() {}, nil
		}

		return nil, fmt.Errorf("reading staged media directory: %w", err)
	}

	stageRel, ok := relWithin(stageDir, targetDir)
	if !ok {
		return nil, fmt.Errorf("%w: staging directory %s is not inside %s", ErrPathEscapesDir, stageDir, targetDir)
	}
	targetFilesDir, err := importMediaFilesDir(targetDir)
	if err != nil {
		return nil, err
	}
	targetRel, err := filepath.Rel(targetDir, targetFilesDir)
	if err != nil {
		return nil, fmt.Errorf("resolving media destination: %w", err)
	}
	targetRel = filepath.ToSlash(targetRel)

	root, err := os.OpenRoot(targetDir)
	if err != nil {
		return nil, fmt.Errorf("opening archive directory: %w", err)
	}
	defer func() { _ = root.Close() }()

	if err := root.MkdirAll(targetRel, dirPermissions); err != nil {
		return nil, fmt.Errorf("creating media/files directory: %w", err)
	}
	targetFiles, err := root.Open(targetRel)
	if err != nil {
		return nil, fmt.Errorf("opening media/files directory: %w", err)
	}
	existing, readErr := targetFiles.Readdirnames(-1)
	closeErr := targetFiles.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading media/files directory: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing media/files directory: %w", closeErr)
	}
	reserved := glxlib.NewFilenameAllocator(existing...)
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, fmt.Errorf("%w: staged media %s is not a regular file", os.ErrInvalid, entry.Name())
		}
		if !reserved.Reserve(entry.Name()) {
			return nil, fmt.Errorf("%w: media destination %s", os.ErrExist, entry.Name())
		}
	}

	stagedRel := path.Join(filepath.ToSlash(stageRel), glxlib.MediaFilesDir)
	committed := make([]committedMediaFile, 0, len(entries))
	defer func() {
		if commitErr != nil {
			rollbackCommittedMedia(root, committed)
		}
	}()
	for _, entry := range entries {
		src := path.Join(stagedRel, entry.Name())
		dst := path.Join(targetRel, entry.Name())
		info, err := publish(root, src, dst)
		if err != nil {
			return nil, fmt.Errorf("committing media file %s: %w", entry.Name(), err)
		}
		committed = append(committed, committedMediaFile{dst, info})
	}

	return func() {
		root, err := os.OpenRoot(targetDir)
		if err != nil {
			return
		}
		defer func() { _ = root.Close() }()
		rollbackCommittedMedia(root, committed)
	}, nil
}

func rollbackCommittedMedia(root *os.Root, files []committedMediaFile) {
	for _, file := range files {
		current, err := root.Lstat(file.name)
		if err == nil && current.Mode().IsRegular() && os.SameFile(current, file.info) {
			_ = root.Remove(file.name)
		}
	}
}

// commitStagedFile publishes a staged regular file without replacing an
// existing destination. Both paths stay confined to root. Hard links preserve
// atomic visibility; exclusive copying is a fallback for filesystems without
// hard-link support.
func commitStagedFile(root *os.Root, src, dst string) (os.FileInfo, error) {
	stagedInfo, err := root.Lstat(src)
	if err != nil {
		return nil, err
	}
	if !stagedInfo.Mode().IsRegular() {
		return nil, os.ErrInvalid
	}
	if err := root.Link(src, dst); err == nil {
		return stagedInfo, nil
	} else if errors.Is(err, os.ErrExist) {
		return nil, err
	}

	input, err := root.Open(src)
	if err != nil {
		return nil, err
	}
	defer func() { _ = input.Close() }()
	output, err := root.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
	if err != nil {
		return nil, err
	}
	createdInfo, statErr := output.Stat()
	if statErr != nil {
		_ = output.Close()

		return nil, statErr
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		current, err := root.Lstat(dst)
		if err == nil && os.SameFile(current, createdInfo) {
			_ = root.Remove(dst)
		}

		return nil, errors.Join(copyErr, closeErr)
	}

	return createdInfo, nil
}

// errGEDZIPNonRegularEntry marks a bundle member that is a symlink or a
// directory rather than a regular file. Symlinks are refused (not followed) to
// prevent a zip-symlink-slip redirecting the copy outside the archive.
var errGEDZIPNonRegularEntry = errors.New("bundle entry is not a regular file")

func copyMemberFile(bundle fs.FS, memberPath, destPath string) error {
	src, err := bundle.Open(memberPath)
	if err != nil {
		if errors.Is(err, zip.ErrAlgorithm) {
			return fmt.Errorf("%w: %q: %w", ErrGEDZIPUnsupportedAlgorithm, memberPath, err)
		}

		return fmt.Errorf("opening zip entry %q: %w", memberPath, err)
	}
	defer func() { _ = src.Close() }()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat zip entry %q: %w", memberPath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return fmt.Errorf("%w: %q", errGEDZIPNonRegularEntry, memberPath)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), dirPermissions); err != nil {
		return fmt.Errorf("creating directory for %q: %w", memberPath, err)
	}

	return writeStreamEntry(src, memberPath, destPath)
}

// writeStreamEntry copies one bundle member to destPath, rejecting any member
// whose decompressed size exceeds maxGEDZIPEntryBytes. On a copy or close
// failure, or when that size limit is exceeded, the partially written
// destination is removed so the next caller cannot observe a truncated or
// oversized file.
func writeStreamEntry(src io.Reader, name, destPath string) error {
	dst, err := os.OpenFile(filepath.Clean(destPath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
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
