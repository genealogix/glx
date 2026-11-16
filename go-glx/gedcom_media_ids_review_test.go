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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGEDCOMReview_MediaIdentifiersRoundTrip(t *testing.T) {
	for _, version := range gedcomVersions {
		for _, uri := range []string{"media/files/photo.jpg", ""} {
			t.Run(version.name+uri, func(t *testing.T) {
				archive := newEventExportArchive(t, nil)
				identifiers := []any{NewExternalIDEntry("typed-123", "https://example.org/media/"), "bare-456"}
				archive.Media = map[string]*Media{"m": {URI: uri, Title: "Caption", MimeType: "image/jpeg", Properties: map[string]any{"external_ids": identifiers}}}
				output, _ := exportGEDCOMString(t, archive, version.version)
				imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
				require.NoError(t, err)
				require.Len(t, imported.Media, 1)
				for _, media := range imported.Media {
					assert.Equal(t, identifiers, media.Properties["external_ids"])
				}
				require.NotNil(t, imported.MediaProperties["external_ids"])
				assert.True(t, *imported.MediaProperties["external_ids"].MultiValue)
			})
		}
	}
}

func TestGEDCOMReview_EmbeddedMediaIdentifiersImport(t *testing.T) {
	for _, tag := range []string{"EXID", "REFN"} {
		gedcom := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME Child /One/\n1 OBJE\n2 FILE media/files/photo.jpg\n3 FORM image/jpeg\n2 " + tag + " abc\n3 TYPE https://example.org/media/\n0 TRLR\n"
		archive, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
		require.NoError(t, err)
		require.Len(t, archive.Media, 1)
		for _, media := range archive.Media {
			assert.Equal(t, []any{NewExternalIDEntry("abc", "https://example.org/media/")}, media.Properties["external_ids"])
		}
	}
}

func TestGEDCOMReview_PersonSourceRepositoryIdentifiersRoundTrip(t *testing.T) {
	for _, version := range gedcomVersions {
		archive := newEventExportArchive(t, map[string][2]string{"p": {"Child One", "male"}})
		identifiers := []any{NewExternalIDEntry("typed-123", "https://example.org/id/"), "bare-456"}
		archive.Persons["p"].Properties["external_ids"] = identifiers
		archive.Sources["s"] = &Source{Title: "Register", Properties: map[string]any{"external_ids": identifiers}}
		archive.Repositories = map[string]*Repository{"r": {Name: "Archive", Properties: map[string]any{"external_ids": identifiers}}}
		output, _ := exportGEDCOMString(t, archive, version.version)
		imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
		require.NoError(t, err)
		require.Len(t, imported.Persons, 1)
		require.Len(t, imported.Sources, 1)
		require.Len(t, imported.Repositories, 1)
		for _, person := range imported.Persons {
			assert.Equal(t, identifiers, person.Properties["external_ids"], version.name)
			assert.NotContains(t, person.Properties, "refn")
		}
		for _, source := range imported.Sources {
			assert.Equal(t, identifiers, source.Properties["external_ids"], version.name)
		}
		for _, repository := range imported.Repositories {
			assert.Equal(t, identifiers, repository.Properties["external_ids"], version.name)
		}
	}
}

func TestGEDCOMReview_AncestryMediaIdentifiers(t *testing.T) {
	for _, object := range []string{
		"0 @O1@ OBJE\n1 FILE\n1 _OID object-123\n1 _CLON\n2 _OID clone-456\n",
		"0 @I1@ INDI\n1 NAME Child /One/\n1 OBJE\n2 FILE\n2 _OID object-123\n2 _CLON\n3 _OID clone-456\n",
	} {
		input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n" + object + "0 TRLR\n"
		archive, _, err := ImportGEDCOM(strings.NewReader(input), nil)
		require.NoError(t, err)
		require.Len(t, archive.Media, 1)
		for _, media := range archive.Media {
			assert.Empty(t, media.URI)
			assert.Equal(t, []any{NewExternalIDEntry("object-123", "ancestry"), NewExternalIDEntry("clone-456", "ancestry_clone")}, media.Properties["external_ids"])
		}
		for _, version := range gedcomVersions {
			output, _ := exportGEDCOMString(t, archive, version.version)
			roundTrip, _, importErr := ImportGEDCOM(strings.NewReader(output), nil)
			require.NoError(t, importErr)
			require.Len(t, roundTrip.Media, 1)
			authority, cloneAuthority := "ancestry", "ancestry_clone"
			if version.version == GEDCOM70 {
				authority, cloneAuthority = "https://www.ancestry.com/", "urn:glx:external-id:ancestry_clone"
			}
			for _, media := range roundTrip.Media {
				assert.Equal(t, []any{NewExternalIDEntry("object-123", authority), NewExternalIDEntry("clone-456", cloneAuthority)}, media.Properties["external_ids"])
			}
		}
	}
}
