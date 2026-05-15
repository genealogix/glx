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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

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

	err := importGEDCOM(gdz, outDir, FormatMulti, true, false, defaultShowFirstErrors)
	require.NoError(t, err)

	require.True(t, dirExists(outDir), "archive directory should exist")
	require.True(t, dirExists(filepath.Join(outDir, "persons")), "persons directory should exist")
}

func TestImportGEDZIP_SingleFile(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	outPath := filepath.Join(t.TempDir(), "out.glx")

	err := importGEDCOM(gdz, outPath, FormatSingle, true, false, defaultShowFirstErrors)
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

func TestImportGEDZIP_MissingGedcomEntry(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"other.ged": []byte(minimalGEDCOM7),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
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

func TestImportGEDZIP_RejectsDotSegmentDuplicateGedcom(t *testing.T) {
	// gedcom.ged and media/../gedcom.ged are two distinct ZIP entry names
	// that path.Clean folds to the same destination. Without the
	// destination-keyed dedup, the second write would silently overwrite
	// the first, hijacking the GEDCOM after hasGedcomEntry already approved
	// the archive on the original (uncleaned) name.
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged":          []byte(minimalGEDCOM7),
		"media/../gedcom.ged": []byte(minimalGEDCOM7),
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.ErrorIs(t, err, ErrGEDZIPDuplicateEntry)
}

func TestImportGEDZIP_VerboseEmitsExtractionMessage(t *testing.T) {
	gdz := buildGEDZIP(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7),
	})
	outDir := filepath.Join(t.TempDir(), "archive")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	origStdout := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	importErr := importGEDCOM(gdz, outDir, FormatMulti, true, true, defaultShowFirstErrors)
	require.NoError(t, w.Close())
	out, readErr := io.ReadAll(r)
	require.NoError(t, readErr)

	require.NoError(t, importErr)
	require.Contains(t, string(out), "Extracting GEDZIP archive")
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

func TestImportGEDZIP_MkdirAllFailsWhenFileOccupiesDirectoryPath(t *testing.T) {
	// Write "foo" as a regular file, then "foo/bar.txt". When extracting the
	// second entry, MkdirAll("foo/") fails because "foo" is already a regular
	// file — exercising the directory-creation error branch.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Body: []byte(minimalGEDCOM7)},
		{Name: "foo", Body: []byte("regular-file-content")},
		{Name: "foo/bar.txt", Body: []byte("nested-content")},
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating directory")
}

func TestImportGEDZIP_OpenFileFailsWhenDirectoryOccupiesFilePath(t *testing.T) {
	// Write "foo/bar.txt" first (which creates "foo" as a directory), then
	// write "foo" as a regular file. The second entry's OpenFile fails with
	// EISDIR — exercising the destination-open error branch in writeZipEntry
	// and the error-propagation branch in extractGEDZIP.
	gdz := buildGEDZIPOrdered(t, []gedzipTestEntry{
		{Name: "gedcom.ged", Body: []byte(minimalGEDCOM7)},
		{Name: "foo/bar.txt", Body: []byte("nested-content")},
		{Name: "foo", Body: []byte("conflicts-with-dir")},
	})

	err := importGEDCOM(gdz, filepath.Join(t.TempDir(), "archive"), FormatMulti, true, false, defaultShowFirstErrors)
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
