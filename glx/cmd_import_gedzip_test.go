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
