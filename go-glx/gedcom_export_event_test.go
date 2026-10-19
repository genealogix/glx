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
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEventExportArchive returns a GLXFile with the standard vocabularies
// loaded and the given persons (ID -> name, sex).
func newEventExportArchive(t *testing.T, persons map[string][2]string) *GLXFile {
	t.Helper()

	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))
	for id, p := range persons {
		glx.Persons[id] = &Person{Properties: map[string]any{
			PersonPropertyName: map[string]any{"value": p[0]},
			PersonPropertySex:  p[1],
		}}
	}

	return glx
}

// exportGEDCOMString exports glx and returns the GEDCOM text with LF line
// endings, plus the export result.
func exportGEDCOMString(t *testing.T, glx *GLXFile, version GEDCOMVersion) (string, *ExportResult) {
	t.Helper()

	data, result, err := ExportGEDCOM(glx, version, nil)
	require.NoError(t, err)

	return strings.ReplaceAll(string(data), "\r\n", "\n"), result
}

// gedcomSubstructures returns every level-1 structure whose line starts with
// "1 <tag>" inside the level-0 record with the given XREF, each as its line
// plus the deeper lines below it.
func gedcomSubstructures(ged, xref, tag string) []string {
	var (
		blocks  []string
		current []string
		inRec   bool
	)
	flush := func() {
		if current != nil {
			blocks = append(blocks, strings.Join(current, "\n"))
			current = nil
		}
	}
	for line := range strings.SplitSeq(ged, "\n") {
		switch {
		case strings.HasPrefix(line, "0 "):
			flush()
			inRec = strings.HasPrefix(line, "0 "+xref+" ")
		case inRec && strings.HasPrefix(line, "1 "):
			flush()
			if line == "1 "+tag || strings.HasPrefix(line, "1 "+tag+" ") {
				current = []string{line}
			}
		case current != nil:
			current = append(current, line)
		}
	}
	flush()

	return blocks
}

func exportWarningMessages(result *ExportResult, entityID string) []string {
	var messages []string
	for _, w := range result.Statistics.Warnings {
		if w.EntityID == entityID {
			messages = append(messages, w.Message)
		}
	}

	return messages
}

var gedcomVersions = []struct {
	name    string
	version GEDCOMVersion
}{
	{"551", GEDCOM551},
	{"70", GEDCOM70},
}

// An event whose type has no GEDCOM tag (taxation, voter_registration) used
// to be dropped from the export without a warning. It now exports as EVEN with
// a TYPE naming the event type, keeping its date, place, notes and sources
// (#1320).
func TestExportGEDCOM_UnmappedEventTypeExportsAsEVEN(t *testing.T) {
	for _, tc := range gedcomVersions {
		t.Run(tc.name, func(t *testing.T) {
			glx := newEventExportArchive(t, map[string][2]string{
				"person-lewis": {"Lewis Little", "male"},
			})
			glx.Places["place-rowan"] = &Place{Name: "Rowan County"}
			glx.Sources["source-tax"] = &Source{Title: "Rowan County Tax Lists"}
			glx.Events["ev-tax-1796"] = &Event{
				Type:         "taxation",
				Date:         "1796",
				PlaceID:      "place-rowan",
				Notes:        NoteList{"Assessed for 200 acres"},
				Properties:   map[string]any{PropertySources: []any{"source-tax"}},
				Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
			}
			glx.Events["ev-vote-1809"] = &Event{
				Type:         "voter_registration",
				Date:         "1809-04-03",
				Properties:   map[string]any{"event_subtype": "Election return"},
				Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
			}

			ged, result := exportGEDCOMString(t, glx, tc.version)

			evens := gedcomSubstructures(ged, "@I1@", "EVEN")
			require.Len(t, evens, 2, ged)
			assert.Equal(t, strings.Join([]string{
				"1 EVEN",
				"2 TYPE Taxation",
				"2 DATE 1796",
				"2 PLAC Rowan County",
				"2 NOTE Assessed for 200 acres",
				"2 SOUR @S1@",
			}, "\n"), evens[0])
			assert.Equal(t, strings.Join([]string{
				"1 EVEN",
				"2 TYPE Voter Registration: Election return",
				"2 DATE 3 APR 1809",
			}, "\n"), evens[1], "the event subtype rides in the single TYPE")
			assert.Empty(t, result.Statistics.Warnings)
		})
	}
}

// An archive-defined event type with no gedcom mapping exports under its own
// label.
func TestExportGEDCOM_ArchiveDefinedEventTypeExportsAsEVEN(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-lewis": {"Lewis Little", "male"},
	})
	glx.EventTypes["land_transaction"] = &VocabularyEntry{Label: "Land Transaction"}
	glx.Events["ev-deed"] = &Event{
		Type:         "land_transaction",
		Date:         "1802",
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
	}

	ged, _ := exportGEDCOMString(t, glx, GEDCOM70)

	evens := gedcomSubstructures(ged, "@I1@", "EVEN")
	require.Len(t, evens, 1, ged)
	assert.Contains(t, evens[0], "2 TYPE Land Transaction")
}

// A generic event of the vocabulary's own `event` type keeps exporting its
// subtype as the TYPE, unchanged.
func TestExportGEDCOM_GenericEventTypeKeepsSubtypeAsTYPE(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-lewis": {"Lewis Little", "male"},
	})
	glx.Events["ev-generic"] = &Event{
		Type:         EventTypeGeneric,
		Properties:   map[string]any{"event_subtype": "Militia muster"},
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
	}

	ged, _ := exportGEDCOMString(t, glx, GEDCOM70)

	assert.Equal(t, []string{"1 EVEN\n2 TYPE Militia muster"}, gedcomSubstructures(ged, "@I1@", "EVEN"))
}

// A couple's event of an unmapped type (legal_separation) is a FAM-level EVEN.
func TestExportGEDCOM_UnmappedFamilyEventExportsAsFamEVEN(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-h": {"John Smith", "male"},
		"person-w": {"Mary Jones", "female"},
	})
	glx.Relationships["rel-marriage"] = &Relationship{
		Type: RelationshipTypeMarriage,
		Participants: []Participant{
			{Person: "person-h", Role: ParticipantRoleSpouse},
			{Person: "person-w", Role: ParticipantRoleSpouse},
		},
	}
	glx.Events["ev-separation"] = &Event{
		Type: "legal_separation",
		Date: "1890",
		Participants: []Participant{
			{Person: "person-h", Role: ParticipantRoleSpouse},
			{Person: "person-w", Role: ParticipantRoleSpouse},
		},
	}

	ged, result := exportGEDCOMString(t, glx, GEDCOM70)

	assert.Equal(t, []string{"1 EVEN\n2 DATE 1890\n2 TYPE Legal Separation"},
		gedcomSubstructures(ged, "@F1@", "EVEN"))
	assert.Empty(t, result.Statistics.Warnings)
}

// Exporting an unmapped event as EVEN + TYPE and importing it back restores
// its event type and subtype (#1320).
func TestRoundtrip_UnmappedEventType(t *testing.T) {
	for _, tc := range gedcomVersions {
		t.Run(tc.name, func(t *testing.T) {
			glx := newEventExportArchive(t, map[string][2]string{
				"person-lewis": {"Lewis Little", "male"},
			})
			glx.Events["ev-tax"] = &Event{
				Type:         "taxation",
				Date:         "1796",
				Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
			}
			glx.Events["ev-vote"] = &Event{
				Type:         "voter_registration",
				Date:         "1809",
				Properties:   map[string]any{"event_subtype": "Election return"},
				Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
			}

			data, _, err := ExportGEDCOM(glx, tc.version, nil)
			require.NoError(t, err)
			imported, _, err := ImportGEDCOM(strings.NewReader(string(data)), nil)
			require.NoError(t, err)

			byType := make(map[string]*Event)
			for _, event := range imported.Events {
				byType[event.Type] = event
			}
			require.Contains(t, byType, "taxation")
			assert.NotContains(t, byType["taxation"].Properties, "event_subtype")
			require.Contains(t, byType, "voter_registration")
			assert.Equal(t, "Election return", byType["voter_registration"].Properties["event_subtype"])
			assert.NotContains(t, byType, EventTypeGeneric)
		})
	}
}

// An EVEN whose TYPE names no vocabulary type still imports as a generic
// event with that subtype.
func TestImportGEDCOM_EVENWithFreeTextTypeStaysGeneric(t *testing.T) {
	ged := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME A /B/\n" +
		"1 EVEN\n2 TYPE Militia muster\n1 EVEN\n2 TYPE Note: not a vocabulary label\n0 TRLR\n"

	imported, _, err := ImportGEDCOM(strings.NewReader(ged), nil)
	require.NoError(t, err)

	subtypes := make([]string, 0, len(imported.Events))
	for _, event := range imported.Events {
		assert.Equal(t, EventTypeGeneric, event.Type)
		subtypes = append(subtypes, event.Properties["event_subtype"].(string))
	}
	assert.ElementsMatch(t, []string{"Militia muster", "Note: not a vocabulary label"}, subtypes)
}

func TestImportGEDCOM_ASSOPhrasePreservesDeclaredRole(t *testing.T) {
	for _, role := range []string{GedcomRoleNghbr, GedcomRoleFriend, GedcomRoleMultiple} {
		t.Run(role, func(t *testing.T) {
			ged := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @I1@ INDI\n1 NAME A //\n" +
				"1 BIRT\n2 ASSO @I2@\n3 ROLE " + role + "\n4 PHRASE Informant\n" +
				"0 @I2@ INDI\n1 NAME B //\n0 TRLR\n"
			imported, _, err := ImportGEDCOM(strings.NewReader(ged), nil)
			require.NoError(t, err)
			require.Len(t, imported.Events, 1)
			for _, event := range imported.Events {
				require.Len(t, event.Participants, 2)
				associate := event.Participants[1]
				assert.Equal(t, ParticipantRoleWitness, associate.Role)
				assert.Equal(t, NoteList{"GEDCOM ROLE: " + role, "Role: Informant"}, associate.Notes)
			}
		})
	}
}

// newAssociationArchive is the #1321 reproduction: a census entered with only
// a household head, a will with a witness, and a baptism with a godparent and
// an informant.
func newAssociationArchive(t *testing.T) *GLXFile {
	t.Helper()

	glx := newEventExportArchive(t, map[string][2]string{
		"person-adam":   {"Adam Little", "male"},
		"person-caspar": {"Caspar Stoehr", "male"},
		"person-lewis":  {"Lewis Little", "male"},
	})
	glx.Events["ev-census-1800"] = &Event{
		Type:         EventTypeCensus,
		Date:         "1800-08-04",
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRoleHouseholdHead}},
	}
	glx.Events["ev-will-1808"] = &Event{
		Type: "will",
		Date: "1808-12-19",
		Participants: []Participant{
			{Person: "person-caspar", Role: ParticipantRolePrincipal},
			{Person: "person-lewis", Role: ParticipantRoleWitness, Notes: NoteList{"Signed with his mark"}},
		},
	}
	glx.Events["ev-bapt-1789"] = &Event{
		Type: "baptism",
		Date: "1789-10-11",
		Participants: []Participant{
			{Person: "person-adam", Role: ParticipantRolePrincipal},
			{Person: "person-lewis", Role: ParticipantRoleGodparent},
			{Person: "person-caspar", Role: ParticipantRoleInformant},
		},
	}

	return glx
}

// Non-principal participants used to be dropped. They now become ASSO with
// ROLE under the event on the principal's record (#1321).
func TestExportGEDCOM_EventParticipantsExportAsASSO70(t *testing.T) {
	ged, result := exportGEDCOMString(t, newAssociationArchive(t), GEDCOM70)

	// XREFs follow sorted person IDs: adam @I1@, caspar @I2@, lewis @I3@.
	assert.Equal(t, []string{strings.Join([]string{
		"1 BAPM",
		"2 DATE 11 OCT 1789",
		"2 ASSO @I3@",
		"3 ROLE GODP",
		"2 ASSO @I2@",
		"3 ROLE OTHER",
		"4 PHRASE Informant",
	}, "\n")}, gedcomSubstructures(ged, "@I1@", "BAPM"))

	assert.Equal(t, []string{strings.Join([]string{
		"1 WILL",
		"2 DATE 19 DEC 1808",
		"2 ASSO @I3@",
		"3 ROLE WITN",
		"3 NOTE Signed with his mark",
	}, "\n")}, gedcomSubstructures(ged, "@I2@", "WILL"))

	// The census with only a household head is written under the head.
	assert.Equal(t, []string{"1 CENS\n2 DATE 4 AUG 1800"}, gedcomSubstructures(ged, "@I3@", "CENS"))
	assert.Empty(t, gedcomSubstructures(ged, "@I3@", "WILL"), "a witness is not given the event as their own")
	assert.Empty(t, result.Statistics.Warnings)
}

// GEDCOM 5.5.1 associations belong to INDI, with event context in notes.
func TestExportGEDCOM_EventParticipantsExportAsASSO551(t *testing.T) {
	ged, _ := exportGEDCOMString(t, newAssociationArchive(t), GEDCOM551)

	assert.Equal(t, []string{"1 BAPM\n2 DATE 11 OCT 1789"}, gedcomSubstructures(ged, "@I1@", "BAPM"))
	assert.ElementsMatch(t, []string{
		"1 ASSO @I3@\n2 RELA Godparent\n2 NOTE Event: Baptism (BAPM; ev-bapt-1789); date: 1789-10-11; subjects: Adam Little",
		"1 ASSO @I2@\n2 RELA Informant\n2 NOTE Event: Baptism (BAPM; ev-bapt-1789); date: 1789-10-11; subjects: Adam Little",
	}, gedcomSubstructures(ged, "@I1@", GedcomTagAsso))
	assert.Equal(t, []string{
		"1 ASSO @I3@\n2 RELA Witness\n2 NOTE Signed with his mark\n2 NOTE Event: Will (WILL; ev-will-1808); date: 1808-12-19; subjects: Caspar Stoehr",
	}, gedcomSubstructures(ged, "@I2@", GedcomTagAsso))
	assert.Empty(t, gedcomSubstructures(ged, "@I3@", "WILL"))
	assertGEDCOM551AssociationPlacement(t, ged)
}

// The permissive GLX importer cannot verify standard placement. Check the
// exported hierarchy against 5.5.1 INDI/ASSOCIATION_STRUCTURE instead.
func assertGEDCOM551AssociationPlacement(t *testing.T, ged string) {
	t.Helper()

	records, version, _, err := parseGEDCOM(strings.NewReader(ged), NewImportLogger(nil))
	require.NoError(t, err)
	require.Equal(t, GEDCOM551, version)
	var visit func(*GEDCOMRecord, string, int)
	visit = func(record *GEDCOMRecord, rootTag string, depth int) {
		assert.NotEqual(t, "_ASSO", record.Tag)
		assert.NotEqual(t, GedcomTagRole, record.Tag)
		if record.Tag == GedcomTagAsso {
			assert.Equal(t, GedcomTagIndi, rootTag, "ASSO must belong to INDI")
			assert.Equal(t, 1, depth, "ASSO must be directly under INDI")
			var relations []string
			for _, sub := range record.SubRecords {
				if sub.Tag == GedcomTagRela {
					assert.Empty(t, sub.SubRecords, "RELA has no CONT/CONC children in 5.5.1")
					relations = append(relations, sub.Value)
				}
			}
			require.Len(t, relations, 1)
			assert.NotEmpty(t, relations[0])
			assert.LessOrEqual(t, utf8.RuneCountInString(relations[0]), 25)
		}
		for _, sub := range record.SubRecords {
			visit(sub, rootTag, depth+1)
		}
	}
	for _, record := range records {
		visit(record, record.Tag, 0)
	}
}

func TestExportGEDCOM_FamilyAssociations551UseEachKnownSpouse(t *testing.T) {
	for _, tc := range []struct{ name, eventType, link, tag string }{
		{"start", EventTypeMarriage, "start", GedcomTagMarr},
		{"end", EventTypeDivorce, "end", GedcomTagDiv},
		{"other", "legal_separation", "", GedcomTagEven},
		{"single-spouse", EventTypeMarriage, "start", GedcomTagMarr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			glx := newEventExportArchive(t, map[string][2]string{
				"person-a": {"A One", "male"}, "person-b": {"B One", "female"},
				"person-w": {"Witness One", "unknown"},
			})
			glx.Places["place-town"] = &Place{Name: "Town"}
			spouses := []Participant{{Person: "person-a", Role: ParticipantRoleSpouse}}
			owners := []string{"@I1@"}
			if tc.name != "single-spouse" {
				spouses = append(spouses, Participant{Person: "person-b", Role: ParticipantRoleSpouse})
				owners = append(owners, "@I2@")
			}
			glx.Relationships["rel-marriage"] = &Relationship{Type: RelationshipTypeMarriage, Participants: spouses}
			switch tc.link {
			case "start":
				glx.Relationships["rel-marriage"].StartEvent = "ev-family"
			case "end":
				glx.Relationships["rel-marriage"].EndEvent = "ev-family"
			}
			glx.Events["ev-family"] = &Event{
				Type: tc.eventType, Date: "1900", PlaceID: "place-town",
				Participants: append(append([]Participant(nil), spouses...), Participant{
					Person: "person-w", Role: ParticipantRoleWitness, Notes: NoteList{"Signed register"},
				}),
			}
			require.Empty(t, glx.Validate().Errors)
			ged, _ := exportGEDCOMString(t, glx, GEDCOM551)
			for _, owner := range owners {
				associations := gedcomSubstructures(ged, owner, GedcomTagAsso)
				require.Len(t, associations, 1)
				assert.Contains(t, associations[0], "1 ASSO @I3@\n2 RELA Witness\n2 NOTE Signed register")
				assert.Contains(t, associations[0], "("+tc.tag+"; ev-family); date: 1900; place: Town; subjects: A One")
			}
			assert.NotContains(t, strings.Join(gedcomSubstructures(ged, "@F1@", tc.tag), "\n"), "ASSO")
			assert.Empty(t, gedcomSubstructures(ged, "@I3@", GedcomTagAsso))
			assertGEDCOM551AssociationPlacement(t, ged)
		})
	}
}

func TestExportGEDCOM_Associations551KeepDistinctEventsAndNotes(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-a": {"A One", "unknown"}, "person-b": {"B One", "unknown"},
		"person-w": {"Witness One", "unknown"},
	})
	glx.ParticipantRoles["custom_role"] = &VocabularyEntry{Label: "Archive-defined role with a long name", AppliesTo: []string{RoleContextEvent}}
	for _, eventID := range []string{"ev-a", "ev-b"} {
		glx.Events[eventID] = &Event{
			Type: "taxation", Date: "1900",
			Participants: []Participant{
				{Person: "person-a", Role: ParticipantRolePrincipal},
				{Person: "person-b", Role: ParticipantRolePrincipal},
				{Person: "person-w", Role: "custom_role", Notes: NoteList{"First detail"}},
				{Person: "person-w", Role: "custom_role", Notes: NoteList{"Second detail"}},
			},
		}
	}
	require.Empty(t, glx.Validate().Errors)
	require.Empty(t, glx.Validate().Warnings)
	ged, _ := exportGEDCOMString(t, glx, GEDCOM551)
	for _, owner := range []string{"@I1@", "@I2@"} {
		associations := gedcomSubstructures(ged, owner, GedcomTagAsso)
		require.Len(t, associations, 4)
		text := strings.Join(associations, "\n")
		assert.Equal(t, 4, strings.Count(text, "2 RELA Participant"))
		assert.Equal(t, 4, strings.Count(text, "2 NOTE Event role: Archive-defined role with a long name"))
		assert.Equal(t, 2, strings.Count(text, "2 NOTE First detail"))
		assert.Equal(t, 2, strings.Count(text, "2 NOTE Second detail"))
		assert.Equal(t, 2, strings.Count(text, "; ev-a)"))
		assert.Equal(t, 2, strings.Count(text, "; ev-b)"))
	}
	assertGEDCOM551AssociationPlacement(t, ged)
}

func TestExportGEDCOM_Associations551MultilineRoleUsesNote(t *testing.T) {
	for name, separator := range map[string]string{"LF": "\n", "CR": "\r", "CRLF": "\r\n"} {
		t.Run(name, func(t *testing.T) {
			glx := newEventExportArchive(t, map[string][2]string{
				"person-a": {"A One", "unknown"}, "person-w": {"Witness One", "unknown"},
			})
			glx.ParticipantRoles["parish_clerk"] = &VocabularyEntry{Label: "Parish" + separator + "Clerk", AppliesTo: []string{RoleContextEvent}}
			glx.Events["ev-tax"] = &Event{
				Type: "taxation",
				Participants: []Participant{
					{Person: "person-a", Role: ParticipantRolePrincipal},
					{Person: "person-w", Role: "parish_clerk"},
				},
			}
			require.Empty(t, glx.Validate().Errors)
			require.Empty(t, glx.Validate().Warnings)
			ged, _ := exportGEDCOMString(t, glx, GEDCOM551)
			associations := gedcomSubstructures(ged, "@I1@", GedcomTagAsso)
			require.Len(t, associations, 1)
			assert.Contains(t, associations[0], "2 RELA Participant\n2 NOTE Event role: Parish\n3 CONT Clerk")
			assertGEDCOM551AssociationPlacement(t, ged)
		})
	}
}

// Parent and spouse roles narrow to FATH/MOTH and HUSB/WIFE by recorded sex,
// and a `gedcom:` value on the participant-roles vocabulary entry overrides
// the built-in table (#524).
func TestGEDCOMAssociationRole(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-m": {"M", "male"},
		"person-f": {"F", "female"},
		"person-u": {"U", ""},
	})
	glx.ParticipantRoles["clergyman"] = &VocabularyEntry{Label: "Clergyman", GEDCOM: "clergy"}
	glx.ParticipantRoles["bondsman"] = &VocabularyEntry{Label: "Bondsman", GEDCOM: "BOND"}
	expCtx := &ExportContext{GLX: glx}

	tests := []struct {
		person, role, want string
	}{
		{"person-m", ParticipantRoleParent, "FATH"},
		{"person-f", ParticipantRoleParent, "MOTH"},
		{"person-u", ParticipantRoleParent, "PARENT"},
		{"person-m", ParticipantRoleSpouse, "HUSB"},
		{"person-f", ParticipantRoleSpouse, "WIFE"},
		{"person-u", ParticipantRoleSpouse, "SPOU"},
		{"person-u", ParticipantRoleWitness, "WITN"},
		{"person-u", ParticipantRoleOfficiant, "OFFICIATOR"},
		{"person-u", ParticipantRoleGodparent, "GODP"},
		{"person-u", ParticipantRoleChild, "CHIL"},
		{"person-u", ParticipantRoleInformant, "OTHER"},
		{"person-u", "clergyman", "CLERGY"},
		{"person-u", "bondsman", "OTHER"}, // not a ROLE enumeration value
	}
	for _, tc := range tests {
		got := gedcomAssociationRole(Participant{Person: tc.person, Role: tc.role}, expCtx)
		assert.Equal(t, tc.want, got, "%s as %s", tc.person, tc.role)
	}
}

// Every household participant of a census with no principal carries it.
func TestExportGEDCOM_HouseholdEventWrittenUnderEachHouseholdMember(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-head":    {"Lewis Little", "male"},
		"person-boarder": {"John Doe", "male"},
	})
	glx.Events["ev-census"] = &Event{
		Type: EventTypeCensus,
		Date: "1800",
		Participants: []Participant{
			{Person: "person-head", Role: ParticipantRoleHouseholdHead},
			{Person: "person-boarder", Role: ParticipantRoleBoarder},
		},
	}

	ged, _ := exportGEDCOMString(t, glx, GEDCOM70)

	// person-boarder @I1@, person-head @I2@
	assert.Equal(t, []string{"1 CENS\n2 DATE 1800"}, gedcomSubstructures(ged, "@I1@", "CENS"))
	assert.Equal(t, []string{"1 CENS\n2 DATE 1800"}, gedcomSubstructures(ged, "@I2@", "CENS"))
}

// A census with a principal keeps the household head as an association.
func TestExportGEDCOM_HouseholdHeadBesidePrincipalIsASSO(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-head":  {"Lewis Little", "male"},
		"person-child": {"Adam Little", "male"},
	})
	glx.Events["ev-census"] = &Event{
		Type: EventTypeCensus,
		Date: "1800",
		Participants: []Participant{
			{Person: "person-child", Role: ParticipantRolePrincipal},
			{Person: "person-head", Role: ParticipantRoleHouseholdHead},
		},
	}

	ged, _ := exportGEDCOMString(t, glx, GEDCOM70)

	// person-child @I1@, person-head @I2@
	assert.Equal(t, []string{"1 CENS\n2 DATE 1800\n2 ASSO @I2@\n3 ROLE OTHER\n4 PHRASE Household Head"},
		gedcomSubstructures(ged, "@I1@", "CENS"))
	assert.Empty(t, gedcomSubstructures(ged, "@I2@", "CENS"))
}

// Witnesses of a marriage are ASSO under the FAM's MARR; the spouses are not.
func TestExportGEDCOM_FamilyEventWitnessExportsAsASSO(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-h": {"John Smith", "male"},
		"person-w": {"Mary Jones", "female"},
		"person-x": {"Peter Witness", "male"},
	})
	glx.Events["ev-marr"] = &Event{
		Type: "marriage",
		Date: "1875",
		Participants: []Participant{
			{Person: "person-h", Role: ParticipantRoleSpouse},
			{Person: "person-w", Role: ParticipantRoleSpouse},
			{Person: "person-x", Role: ParticipantRoleWitness},
		},
	}
	glx.Relationships["rel-marriage"] = &Relationship{
		Type:       RelationshipTypeMarriage,
		StartEvent: "ev-marr",
		Participants: []Participant{
			{Person: "person-h", Role: ParticipantRoleSpouse},
			{Person: "person-w", Role: ParticipantRoleSpouse},
		},
	}

	ged, result := exportGEDCOMString(t, glx, GEDCOM70)

	assert.Equal(t, []string{"1 MARR\n2 DATE 1875\n2 ASSO @I3@\n3 ROLE WITN"},
		gedcomSubstructures(ged, "@F1@", "MARR"))
	assert.Empty(t, result.Statistics.Warnings)
}

// Events that no INDI or FAM record can carry are named in the export
// warnings rather than dropped silently (#1320, #1321).
func TestExportGEDCOM_ReportsEventsNotExported(t *testing.T) {
	glx := newEventExportArchive(t, map[string][2]string{
		"person-lewis": {"Lewis Little", "male"},
	})
	glx.Events["ev-witness-only"] = &Event{
		Type:         "will",
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRoleWitness}},
	}
	glx.Events["ev-no-participants"] = &Event{Type: "birth"}
	glx.Events["ev-unknown-type"] = &Event{
		Type:         "not_in_vocabulary",
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
	}
	glx.Events["ev-exported"] = &Event{
		Type:         "birth",
		Participants: []Participant{{Person: "person-lewis", Role: ParticipantRolePrincipal}},
	}

	_, result := exportGEDCOMString(t, glx, GEDCOM70)

	assert.Equal(t, []string{"event has no principal or household participant (roles: witness) " +
		"and is not a family event of an exported FAM; event not exported"},
		exportWarningMessages(result, "ev-witness-only"))
	assert.Equal(t, []string{"event has no person participants; event not exported"},
		exportWarningMessages(result, "ev-no-participants"))
	assert.Equal(t, []string{`event type "not_in_vocabulary" is not in the event_types vocabulary; event not exported`},
		exportWarningMessages(result, "ev-unknown-type"))
	assert.Empty(t, exportWarningMessages(result, "ev-exported"))
}

// GEDCOM 7.0's structured event associations restore participant roles.
// 5.5.1 projects associations to people and keeps event context in notes;
// its standard structures cannot provide a lossless event-participant roundtrip.
func TestRoundtrip_EventAssociations70(t *testing.T) {
	data, _, err := ExportGEDCOM(newAssociationArchive(t), GEDCOM70, nil)
	require.NoError(t, err)
	imported, _, err := ImportGEDCOM(strings.NewReader(string(data)), nil)
	require.NoError(t, err)

	roles := make(map[string][]string) // event type -> sorted "name:role"
	for _, event := range imported.Events {
		for _, p := range event.Participants {
			roles[event.Type] = append(roles[event.Type],
				PersonDisplayName(imported.Persons[p.Person])+":"+p.Role)
		}
	}
	assert.ElementsMatch(t, []string{
		"Adam Little:principal", "Lewis Little:godparent", "Caspar Stoehr:informant",
	}, roles["baptism"])
	assert.ElementsMatch(t, []string{
		"Caspar Stoehr:principal", "Lewis Little:witness",
	}, roles["will"])

	for _, event := range imported.Events {
		for _, p := range event.Participants {
			assert.NotContains(t, strings.Join(p.Notes, "\n"), "GEDCOM ",
				"a vocabulary role must not fall back to a note")
		}
	}
}

// The #1282 correction must survive the new generic-event and association
// paths: a bare source cannot suppress a detailed no-PAGE citation, and
// same-page citations with different notes/media remain distinct.
func TestExportGEDCOM_EventEvidenceKeepsDistinctDetailsWithAssociations(t *testing.T) {
	for _, tc := range gedcomVersions {
		t.Run(tc.name, func(t *testing.T) {
			glx := newEventExportArchive(t, map[string][2]string{
				"person-a": {"A One", "male"}, "person-b": {"B One", "female"},
				"person-w": {"Witness One", "unknown"},
			})
			glx.Sources["source-register"] = &Source{Title: "Register"}
			glx.Places["place-town"] = &Place{Name: "Town"}
			glx.Media = map[string]*Media{
				"media-date":  {URI: "date.jpg", MimeType: "image/jpeg"},
				"media-place": {URI: "place.jpg", MimeType: "image/jpeg"},
			}
			glx.Citations = map[string]*Citation{
				"citation-date": {
					SourceID: "source-register", Properties: map[string]any{"locator": "p. 4"},
					Notes: NoteList{"Date transcription"}, Media: []string{"media-date"},
				},
				"citation-place": {
					SourceID: "source-register", Properties: map[string]any{"locator": "p. 4"},
					Notes: NoteList{"Place transcription"}, Media: []string{"media-place"},
				},
				"citation-unpaged": {SourceID: "source-register", Notes: NoteList{"Unpaged detail"}},
			}
			glx.Assertions = make(map[string]*Assertion)
			for _, eventID := range []string{"event-tax", "event-marriage"} {
				eventType := "taxation"
				participants := []Participant{
					{Person: "person-a", Role: ParticipantRolePrincipal},
					{Person: "person-w", Role: ParticipantRoleWitness},
				}
				if eventID == "event-marriage" {
					eventType = EventTypeMarriage
					participants = []Participant{
						{Person: "person-a", Role: ParticipantRoleSpouse},
						{Person: "person-b", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleWitness},
					}
				}
				glx.Events[eventID] = &Event{
					Type: eventType, Date: "1900", Participants: participants,
					Properties: map[string]any{PropertySources: []string{"source-register"}, PropertyCitations: []string{"citation-date"}},
				}
				glx.Assertions[eventID+"-date"] = &Assertion{
					Subject: EntityRef{Event: eventID}, Property: "date", Value: "1900",
					Sources: []string{"source-register"}, Citations: []string{"citation-date", "citation-place", "citation-unpaged"},
				}
				glx.Assertions[eventID+"-place"] = &Assertion{
					Subject: EntityRef{Event: eventID}, Property: "place", Value: "place-town",
					Citations: []string{"citation-date"},
				}
			}
			glx.Relationships["rel-marriage"] = marriageOf("person-a", "person-b")
			glx.Relationships["rel-marriage"].StartEvent = "event-marriage"
			require.Empty(t, glx.Validate().Errors)
			ged, _ := exportGEDCOMString(t, glx, tc.version)
			for _, structure := range []struct{ xref, tag string }{{"@I1@", GedcomTagEven}, {"@F1@", GedcomTagMarr}} {
				blocks := gedcomSubstructures(ged, structure.xref, structure.tag)
				require.Len(t, blocks, 1)
				block := blocks[0]
				assert.Equal(t, 4, strings.Count(block, "2 SOUR @S1@"))
				assert.Equal(t, 2, strings.Count(block, "3 PAGE p. 4"))
				assert.Contains(t, block, "3 NOTE Date transcription\n3 OBJE @O1@")
				assert.Contains(t, block, "3 NOTE Place transcription\n3 OBJE @O2@")
				assert.Contains(t, block, "2 SOUR @S1@\n3 NOTE Unpaged detail")
			}
		})
	}
}
