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
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// unsupportedZipMethod is a compression method id registered by tests with a
// passthrough compressor but no decompressor, so an archive built with it
// parses fine but fails entry decompression with zip.ErrAlgorithm. It is well
// outside the range archive/zip ships compressors for (0=Store, 8=Deflate).
const unsupportedZipMethod uint16 = 0xABCD

var registerUnsupportedZipCompressorOnce sync.Once

// registerUnsupportedZipCompressor installs a passthrough compressor for
// unsupportedZipMethod so tests can build a fixture whose readback fails with
// zip.ErrAlgorithm. archive/zip exposes no Unregister API; the registration is
// idempotent and confined to a method id no other code path uses.
func registerUnsupportedZipCompressor(t *testing.T) {
	t.Helper()
	registerUnsupportedZipCompressorOnce.Do(func() {
		zip.RegisterCompressor(unsupportedZipMethod, func(w io.Writer) (io.WriteCloser, error) {
			return passthroughWriteCloser{Writer: w}, nil
		})
	})
}

type passthroughWriteCloser struct{ io.Writer }

func (passthroughWriteCloser) Close() error { return nil }

type repeatedByteReader struct{ b byte }

func (r repeatedByteReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.b
	}

	return len(p), nil
}

// minimalGEDCOM7 is a self-contained GEDCOM 7.0 fixture used by tests that
// only need to assert "import succeeded and a person came through".
const minimalGEDCOM7 = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME John /Smith/
0 TRLR
`

// gedcom7WithMedia is a GEDCOM 7.0 fixture that references a single media file
// at media/photo.jpg, used by tests that exercise the media-extraction path.
const gedcom7WithMedia = `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Jane /Doe/
1 OBJE @O1@
0 @O1@ OBJE
1 FILE media/photo.jpg
2 FORM image/jpeg
0 TRLR
`

// buildGEDZIP writes the given entries (entry name → bytes) into a .gdz file
// in t.TempDir() and returns the absolute path. Entry names use forward
// slashes per the ZIP and GEDZIP specifications.
func buildGEDZIP(t *testing.T, entries map[string][]byte) string {
	t.Helper()

	gdzPath := filepath.Join(t.TempDir(), "fixture.gdz")
	f, err := os.Create(filepath.Clean(gdzPath))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err, "creating zip entry %s", name)
		_, err = w.Write(content)
		require.NoError(t, err, "writing zip entry %s", name)
	}
	require.NoError(t, zw.Close())

	return gdzPath
}

// gedzipTestEntry describes one entry written into a test fixture archive.
// Mode==0 means a regular file; set os.ModeDir for a directory entry,
// os.ModeSymlink for a symlink (Body holds the link target).
type gedzipTestEntry struct {
	Name string
	Body []byte
	Mode os.FileMode
}

// buildGEDZIPOrdered is the slice-based sibling of buildGEDZIP. Tests that
// depend on the order entries are written into the archive (file-vs-directory
// collisions on the destination filesystem) must use this helper because the
// map-keyed buildGEDZIP iterates in nondeterministic order.
func buildGEDZIPOrdered(t *testing.T, entries []gedzipTestEntry) string {
	t.Helper()

	gdzPath := filepath.Join(t.TempDir(), "fixture.gdz")
	f, err := os.Create(filepath.Clean(gdzPath))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.Name, Method: zip.Deflate}
		if e.Mode != 0 {
			h.SetMode(e.Mode)
		}
		w, err := zw.CreateHeader(h)
		require.NoError(t, err, "creating zip entry %q", e.Name)
		_, err = w.Write(e.Body)
		require.NoError(t, err, "writing zip entry %q", e.Name)
	}
	require.NoError(t, zw.Close())

	return gdzPath
}

func TestImportGEDZIP_MultiFile(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	err := importGEDZIP(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	require.True(t, dirExists(outDir), "archive directory should exist")
	require.True(t, dirExists(filepath.Join(outDir, "persons")), "persons directory should exist")
}

func TestImportGEDZIP_SingleFile(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	outPath := filepath.Join(t.TempDir(), "out.glx")

	err := importGEDZIP(gdz, outPath, FormatSingle, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Clean(outPath))
	require.NoError(t, err)

	var glx glxlib.GLXFile
	require.NoError(t, yaml.Unmarshal(data, &glx))
	require.NotEmpty(t, glx.Persons, "imported archive should contain at least one person")
}

func TestImportGEDZIP_CopiesBundledMedia(t *testing.T) {
	wantBytes := []byte("not-actually-a-jpeg-but-distinguishable")
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":      []byte(gedcom7WithMedia),
		"media/photo.jpg": wantBytes,
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	err := importGEDCOM(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	mediaPath := filepath.Join(outDir, "media", "files", "photo.jpg")
	gotBytes, err := os.ReadFile(filepath.Clean(mediaPath))
	require.NoError(t, err, "media file should be copied into archive/media/files/")
	require.Equal(t, wantBytes, gotBytes, "media file contents should match the source bytes")
}

func TestImportGEDZIP_StampsOriginalFilename(t *testing.T) {
	// Bundled media imported from a GEDZIP records the zip entry's basename in
	// properties.original_filename (#1121). The entry is referenced from
	// gedcom.ged by its relative path, so this flows through the same
	// convertMediaCommon path as plain GEDCOM imports.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":      []byte(gedcom7WithMedia),
		"media/photo.jpg": []byte("jpeg-bytes"),
	})
	outPath := filepath.Join(t.TempDir(), "out.glx")

	err := importGEDCOM(gdz, outPath, FormatSingle, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Clean(outPath))
	require.NoError(t, err)

	var glx glxlib.GLXFile
	require.NoError(t, yaml.Unmarshal(data, &glx))
	require.Len(t, glx.Media, 1)
	for _, media := range glx.Media {
		require.Equal(t, "media/files/photo.jpg", media.URI)
		require.Equal(t, "photo.jpg", media.Properties[glxlib.MediaPropertyOriginalFilename])
	}
}

func TestImportGEDZIP_MissingGedcomEntry(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"readme.txt": []byte("no gedcom file here"),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
}

func TestImportGEDZIP_FallbackNonStandardGedcom(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"family_tree.ged": []byte(minimalGEDCOM7),
	})
	var out bytes.Buffer
	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors, &out)
	require.NoError(t, err)
	require.Contains(t, out.String(), "using non-standard GEDCOM entry")
}

func TestImportGEDZIP_FallbackSingleWrapperDir(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"wrapper/gedcom.ged": []byte(minimalGEDCOM7),
	})
	var out bytes.Buffer
	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors, &out)
	require.NoError(t, err)
	require.Contains(t, out.String(), "using non-standard GEDCOM entry")
}

func TestImportGEDZIP_MultipleGedcomEntries(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"tree1.ged": []byte(minimalGEDCOM7),
		"tree2.ged": []byte(minimalGEDCOM7),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPMultipleGedcom)
}

func TestImportGEDZIP_RejectsZipSlip(t *testing.T) {
	// Note: the GEDCOM entry is also present so the missing-gedcom check passes
	// and we exercise the per-entry path validation.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":    []byte(minimalGEDCOM7),
		"../escape.txt": []byte("attacker payload"),
	})

	outDir := filepath.Join(t.TempDir(), "archive")
	err := importGEDCOM(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)

	// Confirm the attacker payload did not land anywhere outside the destination
	escaped := filepath.Join(filepath.Dir(outDir), "escape.txt")
	_, statErr := os.Stat(escaped)
	require.True(t, os.IsNotExist(statErr), "escape file must not exist outside the archive: %s", escaped)
}

func TestImportGEDZIP_RejectsAbsoluteEntryPath(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":  []byte(minimalGEDCOM7),
		"/etc/passwd": []byte("root:x:0:0::/root:/bin/sh"),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_UppercaseExtension(t *testing.T) {
	// buildGEDZIP always names the file fixture.gdz; rename to .GDZ to confirm
	// the extension match is case-insensitive.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	upper := strings.TrimSuffix(gdz, FileExtGEDZIP) + ".GDZ"
	require.NoError(t, os.Rename(gdz, upper))

	err := importGEDCOM(upper, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)
}

func TestImportGEDZIP_NotAValidArchive(t *testing.T) {
	notAZip := filepath.Join(t.TempDir(), "garbage.gdz")
	require.NoError(t, os.WriteFile(notAZip, []byte("definitely not a zip file"), filePermissions))

	err := importGEDCOM(notAZip, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPNotValidArchive)
}

func TestImportGEDZIP_UnsupportedCompressionMethod(t *testing.T) {
	// Build a gdz whose gedcom.ged entry uses a compression method with no
	// registered decompressor. The archive parses fine (so the OpenReader
	// arm is not taken), but (*zip.File).Open returns zip.ErrAlgorithm
	// during extraction. That must surface as ErrGEDZIPUnsupportedAlgorithm,
	// not the corrupt-archive sentinel.
	registerUnsupportedZipCompressor(t)

	gdzPath := filepath.Join(t.TempDir(), "fixture.gdz")
	f, err := os.Create(filepath.Clean(gdzPath))
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "gedcom.ged", Method: unsupportedZipMethod})
	require.NoError(t, err)
	_, err = w.Write([]byte(minimalGEDCOM7))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	err = importGEDCOM(gdzPath, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPUnsupportedAlgorithm)
}

func TestImportGEDZIP_FileNotFound(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.gdz")
	err := importGEDCOM(missing, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDCOMFileNotFound)
}

func TestImportGEDZIP_RejectsBackslashInEntryName(t *testing.T) {
	// On Windows the final isPathWithin guard catches this; the explicit
	// reject closes the gap on every platform and matches the ZIP spec
	// (APPNOTE 4.4.17.1 mandates forward slashes only).
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":         []byte(minimalGEDCOM7),
		`subdir\..\evil.txt`: []byte("payload"),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsNULByteInEntryName(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
		"foo\x00bar": []byte("payload"),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsCaseFoldedDuplicateGedcom(t *testing.T) {
	// Without the dedup check, an attacker could ship gedcom.ged (benign)
	// alongside Gedcom.GED (malicious); on case-insensitive filesystems the
	// second write silently overwrites the first, hijacking the GEDCOM the
	// importer parses after hasGedcomEntry has already approved the archive.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
		"Gedcom.GED": []byte(minimalGEDCOM7),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPDuplicateEntry)
}

func TestImportGEDZIP_RejectsDotSegmentInvalidGedcom(t *testing.T) {
	// gedcom.ged and media/../gedcom.ged are two distinct ZIP entry names.
	// media/../gedcom.ged violates fs.ValidPath and contains a non-canonical
	// dot segment, so it must be rejected as an invalid entry.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":          []byte(minimalGEDCOM7),
		"media/../gedcom.ged": []byte(minimalGEDCOM7),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsOversizedGedcomEntry(t *testing.T) {
	orig := maxGEDZIPEntryBytes
	maxGEDZIPEntryBytes = 32
	t.Cleanup(func() { maxGEDZIPEntryBytes = orig })

	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7), // > 32 bytes
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPEntryTooLarge)
}

func TestImportGEDZIP_FailedMediaDoesNotOverwriteExistingArchive(t *testing.T) {
	orig := maxGEDZIPEntryBytes
	maxGEDZIPEntryBytes = 64
	t.Cleanup(func() { maxGEDZIPEntryBytes = orig })

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "target.glx")
	originalContent := []byte("original valid glx content")
	require.NoError(t, os.WriteFile(outPath, originalContent, filePermissions))

	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE huge.jpg\n" +
		"1 FORM image/jpeg\n" +
		"0 TRLR\n"

	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(gedcom),
		"huge.jpg":   bytes.Repeat([]byte("A"), 128), // exceeds maxGEDZIPEntryBytes (64)
	})

	err := importGEDCOM(gdz, outPath, FormatSingle, true, false, defaultShowFirstErrors)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrGEDZIPEntryTooLarge)

	// Verify target.glx was NOT overwritten
	content, readErr := os.ReadFile(outPath)
	require.NoError(t, readErr)
	require.Equal(t, originalContent, content, "existing archive must not be modified if media copy fails")
}

func TestImportGEDZIP_VerboseEmitsExtractionMessage(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	var out bytes.Buffer
	importErr := importGEDCOM(gdz, outDir, FormatMulti, true, true, defaultShowFirstErrors, &out)

	require.NoError(t, importErr)
	require.Contains(t, out.String(), "Extracting GEDZIP archive")
}

func TestImportGEDZIP_SkipsDirectoryEntry(t *testing.T) {
	// A directory entry (name ending "/" with ModeDir set) must be skipped
	// during extraction so its zero-byte body isn't written as a regular file
	// over a path our later entries might rely on as a directory.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "media/", Mode: os.ModeDir | dirPermissions},
		{Name: "gedcom.ged", Body: []byte(minimalGEDCOM7)},
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	err := importGEDCOM(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)
}

func TestImportGEDZIP_SkipsSymlinkEntry(t *testing.T) {
	// A symlink entry must be skipped so a later entry's write cannot follow
	// the link outside the destination directory (zip-symlink-slip).
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Body: []byte(minimalGEDCOM7)},
		{Name: "link-to-passwd", Body: []byte("/etc/passwd"), Mode: os.ModeSymlink | 0o777},
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	err := importGEDCOM(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// The symlink entry must not have been materialized anywhere under the
	// destination — neither as a link nor as a regular file holding the
	// link-target string.
	_, statErr := os.Lstat(filepath.Join(outDir, "media", "files", "link-to-passwd"))
	require.True(t, os.IsNotExist(statErr), "symlink entry must not be extracted")
}

func TestImportGEDZIP_RejectsArchiveExceedingEntryLimit(t *testing.T) {
	orig := maxGEDZIPEntries
	maxGEDZIPEntries = 2
	t.Cleanup(func() { maxGEDZIPEntries = orig })

	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":  []byte(minimalGEDCOM7),
		"media/a.jpg": []byte("a"),
		"media/b.jpg": []byte("b"),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPTooManyEntries)
}

func TestCopyMemberFile_RejectsEntryExceedingDecompressedLimit(t *testing.T) {
	// Lower the cap so this writes ~1 KiB instead of 512 MiB; the limit is a
	// var precisely so this regression stays cheap and CI-stable.
	orig := maxGEDZIPEntryBytes
	maxGEDZIPEntryBytes = 1024
	t.Cleanup(func() { maxGEDZIPEntryBytes = orig })

	zipPath := filepath.Join(t.TempDir(), "oversized.gdz")
	f, err := os.Create(filepath.Clean(zipPath))
	require.NoError(t, err)

	zw := zip.NewWriter(f)
	w, err := zw.Create("media/huge.bin")
	require.NoError(t, err)
	_, err = io.CopyN(w, repeatedByteReader{b: 'A'}, maxGEDZIPEntryBytes+1)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	zr, err := zip.OpenReader(filepath.Clean(zipPath))
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()
	require.Len(t, zr.File, 1)

	destPath := filepath.Join(t.TempDir(), "huge.bin")
	err = copyMemberFile(zr, "media/huge.bin", destPath)
	require.ErrorIs(t, err, ErrGEDZIPEntryTooLarge)

	_, statErr := os.Stat(destPath)
	require.True(t, os.IsNotExist(statErr), "oversized extracted file should be removed")
}

func TestCopyMemberFile_AcceptsEntryAtExactLimit(t *testing.T) {
	// An entry whose decompressed size equals exactly maxGEDZIPEntryBytes
	// must be accepted (not treated as oversized).
	orig := maxGEDZIPEntryBytes
	maxGEDZIPEntryBytes = 1024
	t.Cleanup(func() { maxGEDZIPEntryBytes = orig })

	zipPath := filepath.Join(t.TempDir(), "exact.gdz")
	f, err := os.Create(filepath.Clean(zipPath))
	require.NoError(t, err)

	zw := zip.NewWriter(f)
	w, err := zw.Create("media/exact.bin")
	require.NoError(t, err)
	_, err = io.CopyN(w, repeatedByteReader{b: 'B'}, maxGEDZIPEntryBytes)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	zr, err := zip.OpenReader(filepath.Clean(zipPath))
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()
	require.Len(t, zr.File, 1)

	destPath := filepath.Join(t.TempDir(), "exact.bin")
	err = copyMemberFile(zr, "media/exact.bin", destPath)
	require.NoError(t, err, "entry at exactly the size limit should be accepted")

	info, statErr := os.Stat(destPath)
	require.NoError(t, statErr, "extracted file must exist")
	require.Equal(t, maxGEDZIPEntryBytes, info.Size())
}

func TestEntrySizeLimitReader(t *testing.T) {
	t.Run("under limit passes through", func(t *testing.T) {
		data := []byte("hello")
		r := &entrySizeLimitReader{r: strings.NewReader(string(data)), remaining: 10}
		got, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, data, got)
	})

	t.Run("exactly at limit passes through", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 8)
		r := &entrySizeLimitReader{r: bytes.NewReader(data), remaining: int64(len(data)) + 1}
		got, err := io.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, data, got)
	})

	t.Run("over limit returns ErrGEDZIPEntryTooLarge", func(t *testing.T) {
		data := bytes.Repeat([]byte("y"), 12)
		r := &entrySizeLimitReader{r: bytes.NewReader(data), remaining: 10 + 1}
		_, err := io.ReadAll(r)
		require.ErrorIs(t, err, ErrGEDZIPEntryTooLarge)
	})

	t.Run("second read on exhausted limit returns ErrGEDZIPEntryTooLarge", func(t *testing.T) {
		r := &entrySizeLimitReader{r: strings.NewReader(""), remaining: 0}
		n, err := r.Read(make([]byte, 4))
		require.Equal(t, 0, n)
		require.ErrorIs(t, err, ErrGEDZIPEntryTooLarge)
	})
}

func TestImportGEDZIP_MkdirAllFailsWhenFileOccupiesDirectoryPath(t *testing.T) {
	// When destPath's parent directory is occupied by a regular file, MkdirAll
	// fails — exercising the directory-creation error branch.
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file-not-dir")
	require.NoError(t, os.WriteFile(filePath, []byte("content"), filePermissions))

	gdz := buildGEDZIP(t, map[string][]byte{"test.txt": []byte("data")})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	destPath := filepath.Join(filePath, "sub", "test.txt")
	err = copyMemberFile(zr, "test.txt", destPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating directory")
}

func TestImportGEDZIP_OpenFileFailsWhenDirectoryOccupiesFilePath(t *testing.T) {
	// When destPath is occupied by an existing directory, OpenFile fails with
	// EISDIR — exercising the destination-open error branch in writeStreamEntry.
	dirPath := filepath.Join(t.TempDir(), "existing-dir")
	require.NoError(t, os.MkdirAll(dirPath, dirPermissions))

	gdz := buildGEDZIP(t, map[string][]byte{"test.txt": []byte("data")})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	err = copyMemberFile(zr, "test.txt", dirPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating destination file")
}

func TestImportGEDZIP_OpenReaderDefaultErrorPath(t *testing.T) {
	// A path with an embedded NUL byte is rejected by os.Open with EINVAL,
	// which is neither IsNotExist nor any of the typed zip.ErrXxx errors —
	// exercising the default arm of the OpenReader error switch.
	err := importGEDCOM("foo\x00bar.gdz", filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening gedzip archive")
}

func TestImportGEDZIP_RejectsGedcomEntryAsDirectory(t *testing.T) {
	// A malicious archive can name a directory entry "gedcom.ged" (no
	// trailing slash, ModeDir set via external attrs) so a raw name match
	// passes but extractGEDZIP skips the entry, leaving the tempdir with
	// no gedcom.ged for the inner importer. hasGedcomEntry must reject
	// this shape at the same ErrGEDZIPMissingGedcom boundary.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Mode: os.ModeDir | dirPermissions},
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
}

func TestImportGEDZIP_RejectsGedcomEntryAsSymlink(t *testing.T) {
	// A symlink entry named gedcom.ged is skipped during extraction (to
	// prevent zip-symlink-slip), so the tempdir would have no gedcom.ged
	// file. Treat it as a missing gedcom rather than letting the inner
	// importer fail with a confusing ErrGEDCOMFileNotFound on the tempdir.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Body: []byte("/etc/passwd"), Mode: os.ModeSymlink | 0o777},
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
}

func TestImportGEDZIP_OfficialMinimal70(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "minimal.glx")

	err := importGEDCOM("testdata/gedcom/7.0/minimal-valid/minimal70.gdz", outputPath, FormatSingle, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var glxFile glxlib.GLXFile
	err = yaml.Unmarshal(data, &glxFile)
	require.NoError(t, err)
	require.NotNil(t, glxFile.ImportMetadata)
	require.Equal(t, "7.0", glxFile.ImportMetadata.GEDCOMVersion)
}

func TestImportGEDZIP_OfficialMaximal70(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "maximal-archive")

	err := importGEDCOM("testdata/gedcom/7.0/comprehensive-spec/maximal70.gdz", outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	// Verify imported persons
	personDir := filepath.Join(outDir, "persons")
	entries, err := os.ReadDir(personDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries, "should have imported person files")

	// Verify media directory and copied media files
	mediaFilesDir := filepath.Join(outDir, "media", "files")
	require.DirExists(t, mediaFilesDir)

	// Check original.mp3 was extracted
	mp3Path := filepath.Join(mediaFilesDir, "original.mp3")
	require.FileExists(t, mp3Path, "original.mp3 should be extracted from maximal70.gdz")
}

func TestCopyFileExclusive_RefusesToFollowDestinationSymlink(t *testing.T) {
	// commitStagedMedia falls back to a copy when the rename fails. If
	// media/files/<name> is an existing symlink pointing outside the archive,
	// a plain os.Create would follow it and overwrite the target; O_EXCL must
	// refuse instead.
	tmpDir := t.TempDir()

	outside := filepath.Join(tmpDir, "outside.txt")
	originalContent := []byte("must not be overwritten")
	require.NoError(t, os.WriteFile(outside, originalContent, filePermissions))

	src := filepath.Join(tmpDir, "staged.jpg")
	require.NoError(t, os.WriteFile(src, []byte("staged bytes"), filePermissions))

	dst := filepath.Join(tmpDir, "photo.jpg")
	require.NoError(t, os.Symlink(outside, dst))

	err := copyFileExclusive(src, dst)
	require.Error(t, err, "copy through an existing symlink must be refused")

	content, readErr := os.ReadFile(outside)
	require.NoError(t, readErr)
	require.Equal(t, originalContent, content, "symlink target must not be written through")
}

func TestCommitStagedMedia_UnlinksSymlinkBeforeCopyFallback(t *testing.T) {
	// End-to-end for the fallback path: even with a hostile symlink already at
	// the destination, the committed media file must be a regular file holding
	// the staged bytes, and the symlink's target must be untouched.
	tmpDir := t.TempDir()

	outside := filepath.Join(tmpDir, "outside.txt")
	originalContent := []byte("must not be overwritten")
	require.NoError(t, os.WriteFile(outside, originalContent, filePermissions))

	stageDir := filepath.Join(tmpDir, "stage")
	stagedFiles := filepath.Join(stageDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(stagedFiles, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(stagedFiles, "photo.jpg"), []byte("staged bytes"), filePermissions))

	targetDir := filepath.Join(tmpDir, "archive")
	targetFiles := filepath.Join(targetDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(targetFiles, dirPermissions))
	require.NoError(t, os.Symlink(outside, filepath.Join(targetFiles, "photo.jpg")))

	require.NoError(t, commitStagedMedia(stageDir, targetDir))

	committed := filepath.Join(targetFiles, "photo.jpg")
	info, err := os.Lstat(committed)
	require.NoError(t, err)
	require.Zero(t, info.Mode()&os.ModeSymlink, "committed media must be a regular file, not a symlink")

	got, err := os.ReadFile(committed)
	require.NoError(t, err)
	require.Equal(t, []byte("staged bytes"), got)

	content, err := os.ReadFile(outside)
	require.NoError(t, err)
	require.Equal(t, originalContent, content, "symlink target must not be written through")
}

// stagingStreams returns IOStreams whose Out is captured, for asserting on the
// warnings stageMediaFilesFromFS emits.
func stagingStreams(buf *bytes.Buffer) *IOStreams {
	return &IOStreams{Out: buf, MachineOut: io.Discard, ErrOut: buf}
}

func TestStageMediaFilesFromFS_WarnsOnUnresolvedReference(t *testing.T) {
	// A FILE ref the library could not bind to a bundle member arrives with an
	// empty MemberPath. That must warn rather than abort the import, and must
	// not produce a file.
	gdz := buildGEDZIP(t, map[string][]byte{"gedcom.ged": []byte(minimalGEDCOM7)})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	var out bytes.Buffer
	stageDir := t.TempDir()
	copyCount, blobCount, warnCount, err := stageMediaFilesFromFS(stagingStreams(&out), stageDir, []glxlib.MediaFileSource{{
		MediaID:        "M1",
		SourceType:     glxlib.MediaSourceFile,
		RelativePath:   "media/missing.jpg",
		TargetFilename: "missing.jpg",
	}}, zr)

	require.NoError(t, err)
	require.Zero(t, copyCount)
	require.Zero(t, blobCount)
	require.Equal(t, 1, warnCount)
	require.Contains(t, out.String(), "unresolved or invalid reference")
	require.NoFileExists(t, filepath.Join(stageDir, glxlib.MediaFilesDir, "missing.jpg"))
}

func TestStageMediaFilesFromFS_WarnsOnSymlinkMember(t *testing.T) {
	// A media member that is a symlink is refused (never followed), but the
	// import continues — with a warning, so the dangling media URI is visible.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Body: []byte(minimalGEDCOM7)},
		{Name: "media/photo.jpg", Body: []byte("/etc/passwd"), Mode: os.ModeSymlink | 0o777},
	})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	var out bytes.Buffer
	stageDir := t.TempDir()
	copyCount, _, warnCount, err := stageMediaFilesFromFS(stagingStreams(&out), stageDir, []glxlib.MediaFileSource{{
		MediaID:        "M1",
		SourceType:     glxlib.MediaSourceFile,
		RelativePath:   "media/photo.jpg",
		MemberPath:     "media/photo.jpg",
		TargetFilename: "photo.jpg",
	}}, zr)

	require.NoError(t, err)
	require.Zero(t, copyCount)
	require.Equal(t, 1, warnCount)
	require.Contains(t, out.String(), "could not copy media file")
	require.NoFileExists(t, filepath.Join(stageDir, glxlib.MediaFilesDir, "photo.jpg"))
}

func TestStageMediaFilesFromFS_WritesBlobAndWarnsOnBadBlob(t *testing.T) {
	// GEDCOM 5.5.1 BLOB members are decoded and written locally — they never
	// touch the bundle — so a GEDZIP carrying a 5.5.1 GEDCOM still gets its
	// inline media. A malformed BLOB warns instead of aborting.
	gdz := buildGEDZIP(t, map[string][]byte{"gedcom.ged": []byte(minimalGEDCOM7)})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	var out bytes.Buffer
	stageDir := t.TempDir()
	copyCount, blobCount, warnCount, err := stageMediaFilesFromFS(stagingStreams(&out), stageDir, []glxlib.MediaFileSource{
		{MediaID: "M1", SourceType: glxlib.MediaSourceBlob, BlobData: ".HM.......k.1..F", TargetFilename: "blob-M1.bin"},
		{MediaID: "M2", SourceType: glxlib.MediaSourceBlob, BlobData: "!!!not-a-blob!!!", TargetFilename: "blob-M2.bin"},
	}, zr)

	require.NoError(t, err)
	require.Zero(t, copyCount)
	require.Equal(t, 1, blobCount)
	require.Equal(t, 1, warnCount)
	require.Contains(t, out.String(), "could not decode BLOB")
	require.FileExists(t, filepath.Join(stageDir, glxlib.MediaFilesDir, "blob-M1.bin"))
	require.NoFileExists(t, filepath.Join(stageDir, glxlib.MediaFilesDir, "blob-M2.bin"))
}

func TestCopyMemberFile_ReportsMissingMember(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{"gedcom.ged": []byte(minimalGEDCOM7)})
	zr, err := zip.OpenReader(gdz)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	err = copyMemberFile(zr, "media/absent.jpg", filepath.Join(t.TempDir(), "absent.jpg"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening zip entry")
}

func TestCopyMemberFile_MapsUnsupportedCompression(t *testing.T) {
	registerUnsupportedZipCompressor(t)

	zipPath := filepath.Join(t.TempDir(), "unsupported.gdz")
	f, err := os.Create(filepath.Clean(zipPath))
	require.NoError(t, err)
	zw := zip.NewWriter(f)
	w, err := zw.CreateHeader(&zip.FileHeader{Name: "media/photo.jpg", Method: unsupportedZipMethod})
	require.NoError(t, err)
	_, err = w.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	err = copyMemberFile(zr, "media/photo.jpg", filepath.Join(t.TempDir(), "photo.jpg"))
	require.ErrorIs(t, err, ErrGEDZIPUnsupportedAlgorithm)
}

func TestCommitStagedMedia_NoStagedDirectoryIsNoOp(t *testing.T) {
	stageDir := t.TempDir()
	targetDir := filepath.Join(t.TempDir(), "archive")

	require.NoError(t, commitStagedMedia(stageDir, targetDir))
	require.NoDirExists(t, filepath.Join(targetDir, glxlib.MediaFilesDir))
}

func TestCommitStagedMedia_FailsWhenDestinationIsNonEmptyDirectory(t *testing.T) {
	// A directory sitting where a media file belongs defeats both the rename
	// and the unlink, so the commit must report the failure rather than
	// silently dropping the file.
	tmpDir := t.TempDir()

	stageDir := filepath.Join(tmpDir, "stage")
	stagedFiles := filepath.Join(stageDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(stagedFiles, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(stagedFiles, "photo.jpg"), []byte("staged"), filePermissions))

	targetDir := filepath.Join(tmpDir, "archive")
	blocking := filepath.Join(targetDir, glxlib.MediaFilesDir, "photo.jpg")
	require.NoError(t, os.MkdirAll(blocking, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(blocking, "occupant"), []byte("x"), filePermissions))

	err := commitStagedMedia(stageDir, targetDir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "committing media file photo.jpg")
}

func TestCopyFileExclusive_CopiesToFreshDestination(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.bin")
	payload := []byte("staged bytes")
	require.NoError(t, os.WriteFile(src, payload, filePermissions))

	dst := filepath.Join(tmpDir, "dst.bin")
	require.NoError(t, copyFileExclusive(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestCopyFileExclusive_ReportsMissingSource(t *testing.T) {
	tmpDir := t.TempDir()
	err := copyFileExclusive(filepath.Join(tmpDir, "absent.bin"), filepath.Join(tmpDir, "dst.bin"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

func TestImportGEDZIP_VerboseSingleFileWithMedia(t *testing.T) {
	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @I1@ INDI\n" +
		"1 NAME John /Doe/\n" +
		"1 OBJE @M1@\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE media/photo.jpg\n" +
		"1 FORM image/jpeg\n" +
		"0 TRLR\n"

	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":      []byte(gedcom),
		"media/photo.jpg": []byte("jpeg-content"),
	})

	outPath := filepath.Join(t.TempDir(), "archive.glx")
	var out bytes.Buffer
	err := importGEDCOM(gdz, outPath, FormatSingle, true, true, defaultShowFirstErrors, &out)
	require.NoError(t, err)

	require.Contains(t, out.String(), "Extracting GEDZIP archive")
	require.Contains(t, out.String(), "Writing single-file archive")
	require.FileExists(t, outPath)

	// Media lands in a sibling media/files/ directory for single-file output.
	copied, err := os.ReadFile(filepath.Join(filepath.Dir(outPath), glxlib.MediaFilesDir, "photo.jpg"))
	require.NoError(t, err)
	require.Equal(t, []byte("jpeg-content"), copied)
}
