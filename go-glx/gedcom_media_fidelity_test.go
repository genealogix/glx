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
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func importGEDCOMString(t *testing.T, gedcom string) (*GLXFile, *ImportResult) {
	t.Helper()
	glx, result, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)

	return glx, result
}

// mediaURIs returns the sorted URIs of the media an ID list names.
func mediaURIs(t *testing.T, glx *GLXFile, ids []string) []string {
	t.Helper()
	uris := make([]string, 0, len(ids))
	for _, id := range ids {
		media := glx.Media[id]
		require.NotNil(t, media, id)
		uris = append(uris, media.URI)
	}
	slices.Sort(uris)

	return uris
}

// An OBJE with several FILEs becomes one Media per FILE, each keeping the
// OBJE's shared metadata and its own FILE's, and every pointer to the OBJE
// links all of them (#1308).
func TestImportGEDCOM_OneMediaPerFILE(t *testing.T) {
	glx, _ := importGEDCOMString(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n"+
		"0 @I1@ INDI\n1 NAME Jane /Doe/\n1 OBJE @O1@\n"+
		"0 @O1@ OBJE\n"+
		"1 FILE scans/letter.jpg\n2 FORM jpg\n2 TITL Scan\n"+
		"1 FILE https://example.org/letter.pdf\n2 FORM pdf\n"+
		"1 NOTE Letter to her sister\n"+
		"1 _OID 0a1b2c3d-4e5f-6789-abcd-ef0123456789\n"+
		"0 TRLR\n")

	require.Len(t, glx.Media, 2)
	require.Len(t, glx.Persons, 1)
	var ids []string
	for _, person := range glx.Persons {
		ids, _ = person.Properties[PropertyMedia].([]string)
	}
	assert.Equal(t, []string{"https://example.org/letter.pdf", "media/files/letter.jpg"}, mediaURIs(t, glx, ids),
		"the person links both files, and a remote reference stays a URL")

	for _, media := range glx.Media {
		assert.Equal(t, NoteList{"Letter to her sister"}, media.Notes, "OBJE-level notes describe every file")
		assert.NotEmpty(t, media.Properties[MediaPropertyExternalIDs], "OBJE-level ids describe every file")
		switch media.URI {
		case "media/files/letter.jpg":
			assert.Equal(t, "Scan", media.Title)
			assert.Equal(t, "image/jpeg", media.MimeType)
		default:
			assert.Empty(t, media.Title, "the scan's title belongs to its own FILE")
			assert.Equal(t, "application/pdf", media.MimeType)
		}
	}
}

// Embedded OBJEs and source and citation links follow the same rule.
func TestImportGEDCOM_EmbeddedAndCitedMultiFileOBJE(t *testing.T) {
	glx, _ := importGEDCOMString(t, "0 HEAD\n1 GEDC\n2 VERS 7.0\n"+
		"0 @S1@ SOUR\n1 TITL Letters\n1 OBJE\n2 FILE a.jpg\n3 FORM image/jpeg\n2 FILE b.jpg\n3 FORM image/jpeg\n"+
		"0 TRLR\n")
	require.Len(t, glx.Media, 2)
	for _, source := range glx.Sources {
		assert.Equal(t, []string{"media/files/a.jpg", "media/files/b.jpg"}, mediaURIs(t, glx, source.Media))
	}
}

// Ancestry's media object id and its clone origin are kept as external ids,
// as is a GEDCOM 7 EXID on the OBJE (#944).
func TestImportGEDCOM_MediaExternalIDs(t *testing.T) {
	glx, _ := importGEDCOMString(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n"+
		"0 @O1@ OBJE\n1 FILE\n2 FORM jpg\n2 TITL Some Caption\n"+
		"1 _OID 0a1b2c3d-4e5f-6789-abcd-ef0123456789\n"+
		"1 _CLON\n2 _OID 99999999-0000-1111-2222-333333333333\n"+
		"1 EXID 12345\n2 TYPE https://example.org/media\n"+
		"0 TRLR\n")
	require.Len(t, glx.Media, 1)
	for _, media := range glx.Media {
		assert.Equal(t, []any{
			NewExternalIDEntry("0a1b2c3d-4e5f-6789-abcd-ef0123456789", ExternalIDTypeAncestry),
			NewExternalIDEntry("99999999-0000-1111-2222-333333333333", ExternalIDTypeAncestryClone),
			NewExternalIDEntry("12345", "https://example.org/media"),
		}, media.Properties[MediaPropertyExternalIDs])
	}
}

// original_filename survives GLX → GEDCOM → GLX as FILE._ORIG, instead of the
// re-import recording the archive's own collision-renamed name (#1149).
func TestGEDCOMRoundTrip_KeepsOriginalFilename(t *testing.T) {
	for name, version := range map[string]GEDCOMVersion{"5.5.1": GEDCOM551, "7.0": GEDCOM70} {
		t.Run(name, func(t *testing.T) {
			archive := &GLXFile{Media: map[string]*Media{
				"media-1": {URI: "media/files/photo-2.jpg", MimeType: "image/jpeg", Properties: map[string]any{
					MediaPropertyOriginalFilename: "photo.jpg",
				}},
			}}
			exported, _ := exportGEDCOMString(t, archive, version)
			require.Contains(t, exported, "_ORIG photo.jpg")

			reimported, _ := importGEDCOMString(t, exported)
			require.Len(t, reimported.Media, 1)
			for _, media := range reimported.Media {
				assert.Equal(t, "photo.jpg", media.Properties[MediaPropertyOriginalFilename])
			}
		})
	}
}
