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
			if version.version == GEDCOM70 {
				assert.Contains(t, output, "2 TYPE https://www.ancestry.com/")
				assert.Contains(t, output, "2 TYPE urn:glx:external-id:ancestry_clone")
			}
			// Import reverses the 7.0 authority URIs, so both dialects keep
			// GLX's own ancestry/ancestry_clone names.
			for _, media := range roundTrip.Media {
				assert.Equal(t, []any{NewExternalIDEntry("object-123", "ancestry"), NewExternalIDEntry("clone-456", "ancestry_clone")}, media.Properties["external_ids"])
			}
		}
	}
}

func TestGEDCOMReview_AuthorityNamesSurviveRepeatedRoundTrips(t *testing.T) {
	archive := newEventExportArchive(t, map[string][2]string{"p": {"Child One", "male"}})
	identifiers := []any{NewExternalIDEntry("Schepp-190", "wikitree"), NewExternalIDEntry("P1", "local registry"), NewExternalIDEntry("x", "https://example.org/id/")}
	archive.Persons["p"].Properties["external_ids"] = identifiers
	current := archive
	for _, version := range []GEDCOMVersion{GEDCOM70, GEDCOM551, GEDCOM70} {
		output, _ := exportGEDCOMString(t, current, version)
		imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
		require.NoError(t, err)
		require.Len(t, imported.Persons, 1)
		for _, person := range imported.Persons {
			assert.Equal(t, identifiers, person.Properties["external_ids"], version)
		}
		current = imported
	}
}

func TestGEDCOMReview_ExternalIDsExportUnderVocabularyKey(t *testing.T) {
	archive := newEventExportArchive(t, nil)
	archive.SourceProperties["ids"] = &PropertyDefinition{Label: "IDs", ValueType: "string", MultiValue: new(true), GEDCOM: GedcomTagExid}
	delete(archive.SourceProperties, "external_ids")
	archive.Sources["s"] = &Source{Title: "Register", Properties: map[string]any{"ids": []any{"abc"}}}
	output, _ := exportGEDCOMString(t, archive, GEDCOM70)
	assert.Contains(t, output, "1 EXID abc")
}

func TestGEDCOMReview_FilelessMediaRestoresTitleAndType(t *testing.T) {
	for _, version := range gedcomVersions {
		archive := newEventExportArchive(t, nil)
		archive.Media = map[string]*Media{"m": {Title: "Caption", MimeType: "image/jpeg", Notes: NoteList{"Kept note"}}}
		output, _ := exportGEDCOMString(t, archive, version.version)
		imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
		require.NoError(t, err)
		require.Len(t, imported.Media, 1)
		for _, media := range imported.Media {
			assert.Equal(t, "Caption", media.Title, version.name)
			assert.Equal(t, "image/jpeg", media.MimeType, version.name)
			assert.Equal(t, NoteList{"Kept note"}, media.Notes, version.name)
		}
	}
}

// convertWithoutPersonEXID imports gedcom through a vocabulary whose person
// properties map nothing to EXID.
func convertWithoutPersonEXID(t *testing.T, gedcom string) *GLXFile {
	t.Helper()
	logger := NewImportLogger(nil)
	records, version, _, err := parseGEDCOM(strings.NewReader(gedcom), logger)
	require.NoError(t, err)
	glx := &GLXFile{Persons: map[string]*Person{}, Events: map[string]*Event{}, Relationships: map[string]*Relationship{}, Places: map[string]*Place{}, Sources: map[string]*Source{}, Repositories: map[string]*Repository{}, Media: map[string]*Media{}, Citations: map[string]*Citation{}, Assertions: map[string]*Assertion{}}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))
	index := buildGEDCOMIndex(glx)
	delete(index.PersonProperties, GedcomTagExid)
	conv := &ConversionContext{
		GLX: glx, Version: version, Logger: logger, GEDCOMIndex: index,
		PersonIDMap: map[string]string{}, FamilyIDMap: map[string]string{}, SourceIDMap: map[string]string{}, RepositoryIDMap: map[string]string{},
		MediaIDMap: map[string]string{}, PlaceIDMap: map[string]string{}, RepositoryNameMap: map[string]string{}, FamilyParentsMap: map[string][]string{},
		SharedNotes: map[string]string{}, ExtensionSchemas: map[string]*ExtensionSchema{}, MediaFileNames: NewFilenameAllocator(),
	}
	require.NoError(t, conv.Convert(records))

	return glx
}

func TestGEDCOMReview_PersonREFNKeptWithoutEXIDMapping(t *testing.T) {
	glx := convertWithoutPersonEXID(t, "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 REFN 12345\n0 TRLR\n")
	require.Len(t, glx.Persons, 1)
	for _, person := range glx.Persons {
		assert.Equal(t, "12345", person.Properties["refn"])
	}
}

func TestGEDCOMReview_NotePointerWithTrailingSpace(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @N1@ NOTE Shared text\n0 @I1@ INDI\n1 NAME Child /One/\n1 NOTE @N1@ \n0 TRLR\n"
	archive, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	for _, person := range archive.Persons {
		assert.Equal(t, NoteList{"Shared text"}, person.Notes)
	}
}

func TestGEDCOMReview_CONCSplitKeepsSpacesInTrimmedTags(t *testing.T) {
	title := strings.Repeat("a", 247) + " " + strings.Repeat("b", 50)
	archive := newEventExportArchive(t, nil)
	archive.Sources["s"] = &Source{Title: title}
	output, _ := exportGEDCOMString(t, archive, GEDCOM551)
	assert.Contains(t, output, " CONC ")
	for line := range strings.SplitSeq(output, "\n") {
		assert.False(t, strings.HasSuffix(line, " ") || strings.Contains(line, "CONC  "), "split next to a space: %q", line)
	}
	imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
	require.NoError(t, err)
	for _, source := range imported.Sources {
		assert.Equal(t, title, source.Title)
	}
}
