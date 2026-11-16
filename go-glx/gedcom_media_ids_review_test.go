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
