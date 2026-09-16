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

package glx

import (
	"archive/zip"
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

const minimalGEDCOM7ForGEDZIP = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 NAME John /Doe/\n" +
	"0 TRLR\n"

const gedcom7WithMediaForGEDZIP = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 NAME John /Doe/\n" +
	"1 OBJE @M1@\n" +
	"0 @M1@ OBJE\n" +
	"1 FILE media/photo.jpg\n" +
	"1 FORM image/jpeg\n" +
	"0 TRLR\n"

const gedcom7WithEncodedMedia = "0 HEAD\n" +
	"1 GEDC\n" +
	"2 VERS 7.0\n" +
	"0 @I1@ INDI\n" +
	"1 NAME John /Doe/\n" +
	"1 OBJE @M1@\n" +
	"0 @M1@ OBJE\n" +
	"1 FILE media/CharlotteBront%C3%AB.jpg\n" +
	"1 FORM image/jpeg\n" +
	"0 TRLR\n"

func buildMemZip(t *testing.T, entries map[string][]byte) *zip.Reader {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	return reader
}

func TestImportGEDZIP_RootGedcomStandard(t *testing.T) {
	fsys := fstest.MapFS{
		"gedcom.ged":      &fstest.MapFile{Data: []byte(gedcom7WithMediaForGEDZIP)},
		"media/photo.jpg": &fstest.MapFile{Data: []byte("jpeg-content")},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "media/photo.jpg", result.MediaFiles[0].MemberPath)

	// No fallback warning should be emitted
	for _, w := range result.Statistics.Warnings {
		require.NotEqual(t, WarningTagGEDZIP, w.Tag, "unexpected fallback warning: %s", w.Message)
	}
}

func TestImportGEDZIP_FallbackRootGedcomCaseInsensitive(t *testing.T) {
	fsys := fstest.MapFS{
		"family.GEDCOM":   &fstest.MapFile{Data: []byte(gedcom7WithMediaForGEDZIP)},
		"media/photo.jpg": &fstest.MapFile{Data: []byte("jpeg-content")},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "media/photo.jpg", result.MediaFiles[0].MemberPath)

	// Fallback warning must be present
	foundFallback := false
	for _, w := range result.Statistics.Warnings {
		if w.Tag == WarningTagGEDZIP && strings.Contains(w.Message, "using non-standard GEDCOM entry") {
			foundFallback = true

			break
		}
	}
	require.True(t, foundFallback, "expected fallback warning on ImportResult")
}

func TestImportGEDZIP_FallbackSingleWrapperDirectory(t *testing.T) {
	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @I1@ INDI\n" +
		"1 NAME Jane /Doe/\n" +
		"1 OBJE @M1@\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE images/portrait.jpg\n" +
		"1 FORM image/jpeg\n" +
		"0 TRLR\n"

	fsys := fstest.MapFS{
		"archive/tree.ged":            &fstest.MapFile{Data: []byte(gedcom)},
		"archive/images/portrait.jpg": &fstest.MapFile{Data: []byte("portrait-bytes")},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "archive/images/portrait.jpg", result.MediaFiles[0].MemberPath)

	foundFallback := false
	for _, w := range result.Statistics.Warnings {
		if w.Tag == WarningTagGEDZIP && strings.Contains(w.Message, "archive/tree.ged") {
			foundFallback = true

			break
		}
	}
	require.True(t, foundFallback, "expected fallback warning for wrapper directory GEDCOM")
}

func TestImportGEDZIP_RejectsMultipleGedcom(t *testing.T) {
	fsys := fstest.MapFS{
		"tree1.ged": &fstest.MapFile{Data: []byte(minimalGEDCOM7ForGEDZIP)},
		"tree2.ged": &fstest.MapFile{Data: []byte(minimalGEDCOM7ForGEDZIP)},
	}

	_, _, err := ImportGEDZIP(fsys, nil)
	require.ErrorIs(t, err, ErrGEDZIPMultipleGedcom)
}

func TestImportGEDZIP_RejectsNestedWrapperGedcom(t *testing.T) {
	// Nested > 1 wrapper directory level should not be discovered
	fsys := fstest.MapFS{
		"dir1/dir2/tree.ged": &fstest.MapFile{Data: []byte(minimalGEDCOM7ForGEDZIP)},
	}

	_, _, err := ImportGEDZIP(fsys, nil)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
}

func TestImportGEDZIP_RejectsMissingGedcom(t *testing.T) {
	fsys := fstest.MapFS{
		"media/photo.jpg": &fstest.MapFile{Data: []byte("jpeg-content")},
		"readme.txt":      &fstest.MapFile{Data: []byte("notes")},
	}

	_, _, err := ImportGEDZIP(fsys, nil)
	require.ErrorIs(t, err, ErrGEDZIPMissingGedcom)
}

func TestImportGEDZIP_MediaResolution_PercentDecoding(t *testing.T) {
	fsys := fstest.MapFS{
		"gedcom.ged":                &fstest.MapFile{Data: []byte(gedcom7WithEncodedMedia)},
		"media/CharlotteBrontë.jpg": &fstest.MapFile{Data: []byte("photo-bytes")},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "media/CharlotteBrontë.jpg", result.MediaFiles[0].MemberPath)
}

func TestImportGEDZIP_MediaResolution_CaseInsensitive(t *testing.T) {
	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE media/PHOTO.JPG\n" +
		"1 FORM image/jpeg\n" +
		"0 TRLR\n"

	fsys := fstest.MapFS{
		"gedcom.ged":      &fstest.MapFile{Data: []byte(gedcom)},
		"media/photo.jpg": &fstest.MapFile{Data: []byte("photo-bytes")},
	}

	_, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "media/photo.jpg", result.MediaFiles[0].MemberPath)
}

func TestImportGEDZIP_MediaResolution_UnresolvedWarns(t *testing.T) {
	fsys := fstest.MapFS{
		"gedcom.ged": &fstest.MapFile{Data: []byte(gedcom7WithMediaForGEDZIP)},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Empty(t, result.MediaFiles[0].MemberPath)

	foundUnresolved := false
	for _, w := range result.Statistics.Warnings {
		if strings.Contains(w.Message, "unresolved media file reference") {
			foundUnresolved = true

			break
		}
	}
	require.True(t, foundUnresolved, "expected unresolved media warning")
}

func TestImportGEDZIP_MediaResolution_TraversalWarns(t *testing.T) {
	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE ../escape.txt\n" +
		"1 FORM text/plain\n" +
		"0 TRLR\n"

	fsys := fstest.MapFS{
		"gedcom.ged": &fstest.MapFile{Data: []byte(gedcom)},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Empty(t, result.MediaFiles[0].MemberPath)

	foundRejected := false
	for _, w := range result.Statistics.Warnings {
		if strings.Contains(w.Message, "media file reference rejected") {
			foundRejected = true

			break
		}
	}
	require.True(t, foundRejected, "expected rejected media warning")
}

func TestImportGEDZIP_RejectsInvalidEntry_ZipSlip(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged":    []byte(minimalGEDCOM7ForGEDZIP),
		"../escape.txt": []byte("attacker payload"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_Backslash(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged":           []byte(minimalGEDCOM7ForGEDZIP),
		"subdir\\..\\evil.txt": []byte("payload"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_NUL(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7ForGEDZIP),
		"foo\x00bar": []byte("payload"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_Absolute(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged":  []byte(minimalGEDCOM7ForGEDZIP),
		"/etc/passwd": []byte("payload"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_VolumePrefix(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7ForGEDZIP),
		"C:evil.txt": []byte("payload"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsDuplicateEntry_CaseFolded(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7ForGEDZIP),
		"Gedcom.GED": []byte(minimalGEDCOM7ForGEDZIP),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPDuplicateEntry)
}

func TestImportGEDZIP_RejectsDuplicateEntry_UnicodeCaseFolded(t *testing.T) {
	// "Σ.jpg" (uppercase Greek sigma) and "ς.jpg" (lowercase Greek final sigma)
	// fold to the same Unicode rune under SimpleFold.
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged": []byte(minimalGEDCOM7ForGEDZIP),
		"Σ.jpg":      []byte("image1"),
		"ς.jpg":      []byte("image2"),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPDuplicateEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_DotSegment(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged":          []byte(minimalGEDCOM7ForGEDZIP),
		"media/../gedcom.ged": []byte(minimalGEDCOM7ForGEDZIP),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_RejectsInvalidEntry_NonCanonicalPrefix(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"./tree.ged": []byte(minimalGEDCOM7ForGEDZIP),
	})

	_, _, err := ImportGEDZIP(zr, nil)
	require.ErrorIs(t, err, ErrGEDZIPInvalidEntry)
}

func TestImportGEDZIP_MediaResolution_WrapperWithPercent(t *testing.T) {
	gedcom := "0 HEAD\n" +
		"1 GEDC\n" +
		"2 VERS 7.0\n" +
		"0 @M1@ OBJE\n" +
		"1 FILE photo%20.jpg\n" +
		"1 FORM image/jpeg\n" +
		"0 TRLR\n"

	// Wrapper directory literally named "wrap%20dir", containing "photo .jpg"
	fsys := fstest.MapFS{
		"wrap%20dir/tree.ged":   &fstest.MapFile{Data: []byte(gedcom)},
		"wrap%20dir/photo .jpg": &fstest.MapFile{Data: []byte("photo-bytes")},
	}

	glxFile, result, err := ImportGEDZIP(fsys, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "wrap%20dir/photo .jpg", result.MediaFiles[0].MemberPath)
}

func TestImportGEDZIP_ZipReaderFS(t *testing.T) {
	zr := buildMemZip(t, map[string][]byte{
		"gedcom.ged":      []byte(gedcom7WithMediaForGEDZIP),
		"media/photo.jpg": []byte("jpeg-content"),
	})

	glxFile, result, err := ImportGEDZIP(zr, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.Len(t, result.MediaFiles, 1)
	require.Equal(t, "media/photo.jpg", result.MediaFiles[0].MemberPath)
}

func TestImportGEDZIP_OfficialMinimal70(t *testing.T) {
	gdzPath := filepath.Join("..", "glx", "testdata", "gedcom", "7.0", "minimal-valid", "minimal70.gdz")
	zr, err := zip.OpenReader(gdzPath)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	glxFile, result, err := ImportGEDZIP(zr, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.NotNil(t, result)
	require.NotNil(t, glxFile.ImportMetadata)
	require.Equal(t, "7.0", glxFile.ImportMetadata.GEDCOMVersion)
}

func TestImportGEDZIP_OfficialMaximal70(t *testing.T) {
	gdzPath := filepath.Join("..", "glx", "testdata", "gedcom", "7.0", "comprehensive-spec", "maximal70.gdz")
	zr, err := zip.OpenReader(gdzPath)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	glxFile, result, err := ImportGEDZIP(zr, nil)
	require.NoError(t, err)
	require.NotNil(t, glxFile)
	require.NotNil(t, result)
	require.Len(t, glxFile.Persons, 4)
	require.NotNil(t, glxFile.ImportMetadata)
	require.Equal(t, "7.0", glxFile.ImportMetadata.GEDCOMVersion)
}
