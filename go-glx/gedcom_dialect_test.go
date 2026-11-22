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

func TestGEDCOMDialect_ExternalIDsAgeAndLongNotes(t *testing.T) {
	for _, version := range gedcomVersions {
		t.Run(version.name, func(t *testing.T) {
			archive := newEventExportArchive(t, map[string][2]string{"p": {"Catharina Schepp", "female"}})
			archive.Persons["p"].Properties["external_ids"] = []any{
				NewExternalIDEntry("Schepp-190", "wikitree"),
				NewExternalIDEntry("P123", "local registry"),
				NewExternalIDEntry("42", "https://example.org/people/"),
				"untyped-123",
			}
			note := strings.Repeat("ü word ", 100) + "\nSecond line\n" + strings.Repeat("終", 300)
			archive.Persons["p"].Notes = NoteList{note}
			archive.Events["e"] = &Event{Type: "death", Date: "1803-03-14", Participants: []Participant{
				{Person: "p", Role: "principal", Properties: map[string]any{"age_at_event": "68y 6m"}},
			}}
			gedcom, _ := exportGEDCOMString(t, archive, version.version)
			assert.Contains(t, gedcom, "2 AGE 68y 6m")
			if version.version == GEDCOM551 {
				assert.NotContains(t, gedcom, " EXID ")
				assert.Contains(t, gedcom, "1 REFN Schepp-190\n2 TYPE wikitree")
				assert.Contains(t, gedcom, " CONC ")
			} else {
				assert.Contains(t, gedcom, "1 EXID Schepp-190\n2 TYPE https://www.wikitree.com/wiki/")
				assert.Contains(t, gedcom, "2 TYPE urn:glx:external-id:local%20registry")
				assert.NotContains(t, gedcom, " CONC ")
			}
			imported, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
			require.NoError(t, err)
			require.Len(t, imported.Persons, 1)
			person := imported.Persons["person-1"]
			assert.Equal(t, NoteList{note}, person.Notes)
			require.Len(t, person.Properties["external_ids"], 4)
			assert.NotContains(t, person.Properties, "exid")
			for _, event := range imported.Events {
				assert.Equal(t, "68y 6m", event.Participants[0].Properties["age_at_event"])
			}
		})
	}
}

func TestGEDCOMDialect_MediaTypes(t *testing.T) {
	for _, format := range []string{"image/jpeg", "application/pdf", "image/x-custom", "jpg"} {
		t.Run(format, func(t *testing.T) {
			gedcom := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM " + format + "\n0 TRLR\n"
			archive, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
			require.NoError(t, err)
			expected := format
			if format == "jpg" {
				expected = "image/jpeg"
			}
			require.Len(t, archive.Media, 1)
			for _, media := range archive.Media {
				assert.Equal(t, expected, media.MimeType)
			}
			output, _ := exportGEDCOMString(t, archive, GEDCOM70)
			assert.Contains(t, output, "2 FORM "+expected)
			assert.NotContains(t, output, " MIME ")
		})
	}
}

func TestGEDCOM551_AssociationRoundTrip(t *testing.T) {
	archive := newAssociationArchive(t)
	for _, version := range gedcomVersions {
		t.Run(version.name, func(t *testing.T) {
			gedcom, _ := exportGEDCOMString(t, archive, version.version)
			imported, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
			require.NoError(t, err)
			for _, person := range imported.Persons {
				assert.NotContains(t, person.Properties, "asso")
			}
			for _, original := range archive.Events {
				if original.Type == EventTypeCensus {
					continue
				} // CENS is currently imported as residence properties.
				var found *Event
				for _, event := range imported.Events {
					if event.Type == original.Type && event.Date == original.Date {
						found = event

						break
					}
				}
				require.NotNil(t, found, original.Type)
				assert.Len(t, found.Participants, len(original.Participants), original.Type)
				for _, participant := range original.Participants {
					var actual *Participant
					for i := range found.Participants {
						other := &found.Participants[i]
						if PersonDisplayName(imported.Persons[other.Person]) == PersonDisplayName(archive.Persons[participant.Person]) && other.Role == participant.Role {
							actual = other

							break
						}
					}
					require.NotNil(t, actual, participant)
					for _, note := range participant.Notes {
						assert.Contains(t, actual.Notes, note)
					}
				}
			}
		})
	}
}

func TestGEDCOM551_UnscopedAssociationRetainsPersonAndRelation(t *testing.T) {
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 ASSO @I2@\n2 RELA Godfather\n2 NOTE The original note\n0 @I2@ INDI\n1 NAME Sponsor /Two/\n0 TRLR\n"
	archive, result, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	require.Len(t, archive.Relationships, 1)
	for _, relation := range archive.Relationships {
		assert.Equal(t, "associate", relation.Type)
		assert.Equal(t, "person-1", relation.Participants[0].Person)
		assert.Equal(t, "person-2", relation.Participants[1].Person)
		assert.Contains(t, relation.Notes, "GEDCOM RELA: Godfather")
		assert.Contains(t, relation.Notes, "The original note")
	}
	assert.NotEmpty(t, result.Statistics.Warnings)
}

func TestGEDCOMDialect_FilelessMediaReportsLimitsAndKeepsMetadata(t *testing.T) {
	for _, version := range gedcomVersions {
		t.Run(version.name, func(t *testing.T) {
			archive := newEventExportArchive(t, nil)
			archive.Media = map[string]*Media{}
			archive.Media["m"] = &Media{Title: "Some Caption", MimeType: "image/jpeg", Properties: map[string]any{"external_ids": []any{NewExternalIDEntry("123", "ancestry")}}}
			output, result := exportGEDCOMString(t, archive, version.version)
			assert.NotContains(t, output, "1 FILE")
			assert.Contains(t, output, "Some Caption")
			assert.Contains(t, output, "Media type: image/jpeg")
			if version.version == GEDCOM70 {
				assert.Contains(t, output, "EXID 123")
			} else {
				assert.Contains(t, output, "REFN 123")
			}
			require.NotEmpty(t, result.Statistics.Warnings)
			assert.Contains(t, result.Statistics.Warnings[0].Message, "no URI")
		})
	}
}

func TestGEDCOM551_AssociationFamilyAndAmbiguousContext(t *testing.T) {
	archive := newEventExportArchive(t, map[string][2]string{"a": {"Anna One", "female"}, "b": {"Bob Two", "male"}, "w": {"Witness Three", "female"}})
	archive.Relationships["marriage"] = &Relationship{Type: "marriage", StartEvent: "wedding", Participants: []Participant{{Person: "a", Role: "spouse"}, {Person: "b", Role: "spouse"}}}
	archive.Events["wedding"] = &Event{Type: "marriage", Date: "1800-01-01", Participants: []Participant{{Person: "a", Role: "bride"}, {Person: "b", Role: "groom"}, {Person: "w", Role: "witness", Notes: NoteList{"Signed the register"}}}}
	gedcom, _ := exportGEDCOMString(t, archive, GEDCOM551)
	imported, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
	require.NoError(t, err)
	for _, event := range imported.Events {
		if event.Type == "marriage" {
			require.Len(t, event.Participants, 3, "the witness projected under both spouses must be restored once")
			assert.Contains(t, event.Participants[2].Notes, "Signed the register")
		}
	}
	ambiguous := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 BAPM\n2 DATE 1800\n1 BAPM\n2 DATE 1800\n1 ASSO @I2@\n2 RELA Godparent\n2 NOTE Event: Baptism (BAPM; original-id); date: 1800\n0 @I2@ INDI\n1 NAME Sponsor /Two/\n0 TRLR\n"
	imported, result, err := ImportGEDCOM(strings.NewReader(ambiguous), nil)
	require.NoError(t, err)
	require.Len(t, imported.Relationships, 1, "ambiguous context must not select an event arbitrarily")
	for _, event := range imported.Events {
		assert.Len(t, event.Participants, 1)
	}
	assert.NotEmpty(t, result.Statistics.Warnings)
}

func TestGEDCOM551_NeutralAssociationRoleComesFromEventRoleNote(t *testing.T) {
	for _, relation := range []string{"Participant", "Godparent"} {
		gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 BAPM\n2 DATE 1800\n1 ASSO @I2@\n2 RELA " + relation + "\n2 NOTE Event role: Informant\n2 NOTE Original note\n2 NOTE Event: Baptism (BAPM; e); date: 1800\n0 @I2@ INDI\n1 NAME Associate /Two/\n0 TRLR\n"
		archive, _, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
		require.NoError(t, err)
		for _, event := range archive.Events {
			require.Len(t, event.Participants, 2)
			associate := event.Participants[1]
			assert.Contains(t, associate.Notes, "Original note")
			if relation == "Participant" {
				assert.Equal(t, "informant", associate.Role)
				assert.Equal(t, NoteList{"Original note"}, associate.Notes)
			} else {
				assert.Equal(t, "godparent", associate.Role, "a real RELA takes precedence over the context note")
			}
		}
	}
}

func TestGEDCOM551_ContextPunctuationKeepsParticipants(t *testing.T) {
	for _, placeName := range []string{"Town; County", "Town; subjects: Parish; place: Original", "Town \"quoted\"; County", "St. Mary (Parish; Old)"} {
		archive := newEventExportArchive(t, map[string][2]string{"p": {"Child; date: Original", "male"}, "w": {"Witness One", "female"}})
		archive.Places["place"] = &Place{Name: placeName}
		archive.Events["e"] = &Event{Type: "baptism", Date: "1800", PlaceID: "place", Participants: []Participant{{Person: "p", Role: "principal"}, {Person: "w", Role: "godparent"}}}
		output, _ := exportGEDCOMString(t, archive, GEDCOM551)
		imported, _, err := ImportGEDCOM(strings.NewReader(output), nil)
		require.NoError(t, err)
		require.Len(t, imported.Events, 1)
		for _, event := range imported.Events {
			assert.Len(t, event.Participants, 2, placeName)
		}
		assert.Empty(t, imported.Relationships, "a unique event should retain its participant rather than create an associate relationship")
	}
}

func TestGEDCOM551_LegacyContextWithUnlabelledSemicolons(t *testing.T) {
	input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 BAPM\n2 DATE 1800\n2 PLAC Town; County\n1 ASSO @I2@\n2 RELA Godparent\n2 NOTE Event: Baptism (BAPM; original); date: 1800; place: Town; County; subjects: Child; extra text\n0 @I2@ INDI\n1 NAME Sponsor /Two/\n0 TRLR\n"
	archive, _, err := ImportGEDCOM(strings.NewReader(input), nil)
	require.NoError(t, err)
	require.Len(t, archive.Events, 1)
	for _, event := range archive.Events {
		assert.Len(t, event.Participants, 2)
	}
	assert.Empty(t, archive.Relationships)
}

func TestGEDCOM551_InvalidEncodedContextPreservesAssociation(t *testing.T) {
	for _, value := range []string{"{invalid", "null", "[]", `{"date":1}`} {
		note := "Event: Baptism (BAPM; original); context: " + value
		input := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME Child /One/\n1 BAPM\n1 ASSO @I2@\n2 RELA Godparent\n2 NOTE " + note + "\n0 @I2@ INDI\n1 NAME Sponsor /Two/\n0 TRLR\n"
		archive, _, err := ImportGEDCOM(strings.NewReader(input), nil)
		require.NoError(t, err)
		require.Len(t, archive.Events, 1)
		for _, event := range archive.Events {
			assert.Len(t, event.Participants, 1)
		}
		require.Len(t, archive.Relationships, 1)
		for _, relationship := range archive.Relationships {
			assert.Contains(t, relationship.Notes, note)
		}
	}
}

func TestGEDCOM551_AssociationOnUnconvertedHostIsSkipped(t *testing.T) {
	archive := newEventExportArchive(t, map[string][2]string{"w": {"Witness One", "female"}})
	conv := &ConversionContext{GLX: archive, PersonIDMap: map[string]string{"@I2@": "w"}, Logger: NewImportLogger(nil)}
	records := []*GEDCOMRecord{{XRef: "@I1@", Tag: GedcomTagIndi, SubRecords: []*GEDCOMRecord{{Tag: GedcomTagAsso, Value: "@I2@"}}}}
	importIndividualAssociations(records, conv)
	assert.Empty(t, archive.Relationships)
	assert.NotEmpty(t, conv.Stats.Warnings)
}
