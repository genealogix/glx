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

// An empty entity node decodes to a nil pointer. Every public path that
// validates must report it as an error instead of panicking (#1406).
func TestValidateNullEntityIsAnErrorNotAPanic(t *testing.T) {
	for _, collection := range AllEntityTypes {
		t.Run(string(collection), func(t *testing.T) {
			content := []byte(string(collection) + ":\n  my-entity-id:\n")
			want := string(collection) + "[my-entity-id]: entity is empty (null)"

			t.Run("multi-file", func(t *testing.T) {
				_, _, err := NewSerializer(nil).DeserializeMultiFileFromMap(map[string][]byte{"file.glx": content})
				assert.Contains(t, structuredErrorMessages(t, err), want)
			})

			t.Run("single-file", func(t *testing.T) {
				_, err := NewSerializer(nil).DeserializeSingleFileBytes(content)
				assert.Contains(t, structuredErrorMessages(t, err), want)
			})
		})
	}

	t.Run("direct Validate keeps checking the other entities", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"empty": nil, "alice": {}},
			Events:  map[string]*Event{"event-1": {Type: "not-a-type", Participants: []Participant{{Person: "alice"}}}},
		}
		result := archive.Validate()
		messages := errorMessages(result.Errors)
		assert.Contains(t, messages, "persons[empty]: entity is empty (null)")
		assert.Contains(t, messages, "not-a-type", "checks after the null entity still run")
		assert.Nil(t, archive.Persons["empty"], "the caller's archive is not modified")
		assert.Len(t, archive.Persons, 2)
	})
}

// IDs are unique within each entity type, ignoring case (#744, #696).
func TestValidateIDCaseCollisions(t *testing.T) {
	t.Run("same type, differing only by case", func(t *testing.T) {
		archive := &GLXFile{Persons: map[string]*Person{"Person-A": {}, "person-a": {}, "person-b": {}}}
		errs := archive.Validate().Errors
		require.Len(t, errs, 1)
		assert.Equal(t, EntityTypePersons, errs[0].SourceType)
		assert.Contains(t, errs[0].Message, `persons IDs "Person-A" and "person-a" differ only by case`)
	})

	t.Run("the same ID in two entity types is allowed", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"shared-id": {}},
			Sources: map[string]*Source{"shared-id": {Title: "A register"}},
		}
		assert.Empty(t, archive.Validate().Errors)
	})
}

// An assertion's property is checked against its subject's property
// vocabulary, with the subject's own fields allowed without an entry (#1224).
func TestValidateAssertionPropertyNames(t *testing.T) {
	newArchive := func(subject EntityRef, property string) *GLXFile {
		archive := &GLXFile{
			Persons:       map[string]*Person{"alice": {}},
			Events:        map[string]*Event{"birth-1": {Type: EventTypeBirth, Participants: []Participant{{Person: "alice"}}}},
			Relationships: map[string]*Relationship{"rel-1": {Type: RelationshipTypeAssociate, Participants: []Participant{{Person: "alice"}, {Person: "alice"}}}},
			Places:        map[string]*Place{"place-1": {Name: "Leeds"}},
			Sources:       map[string]*Source{"source-1": {Title: "Register"}},
			Assertions: map[string]*Assertion{
				"claim": {Subject: subject, Property: property, Value: "x", Sources: []string{"source-1"}},
			},
		}
		require.NoError(t, MergeStandardVocabularies(archive))

		return archive
	}
	propertyWarnings := func(archive *GLXFile) []string {
		var out []string
		for _, w := range archive.Validate().Warnings {
			if w.SourceType == EntityTypeAssertions && w.Field == "property" {
				out = append(out, w.Message)
			}
		}

		return out
	}

	cases := []struct {
		name     string
		subject  EntityRef
		property string
		warns    bool
	}{
		{"typo on a person", EntityRef{Person: "alice"}, "occupaton", true},
		{"defined person property", EntityRef{Person: "alice"}, "occupation", false},
		{"event date is structural", EntityRef{Event: "birth-1"}, "date", false},
		{"event place is structural", EntityRef{Event: "birth-1"}, "place", false},
		{"unknown event property", EntityRef{Event: "birth-1"}, "some_property", true},
		{"place parent is structural", EntityRef{Place: "place-1"}, "parent", false},
		{"relationship type is structural", EntityRef{Relationship: "rel-1"}, "type", false},
		{"unknown relationship property", EntityRef{Relationship: "rel-1"}, "strated_on", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			warnings := propertyWarnings(newArchive(tc.subject, tc.property))
			if tc.warns {
				require.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], `property "`+tc.property+`" is not defined`)
			} else {
				assert.Empty(t, warnings)
			}
		})
	}

	t.Run("no vocabulary for the subject type is not a per-assertion warning", func(t *testing.T) {
		archive := &GLXFile{
			Persons:    map[string]*Person{"alice": {}},
			Assertions: map[string]*Assertion{"claim": {Subject: EntityRef{Person: "alice"}, Property: "anything", Value: "x"}},
		}
		assert.Empty(t, propertyWarnings(archive))
	})
}

// A participant assertion's properties are checked against the vocabulary of
// its subject: relationship_properties for a relationship (#1225).
func TestValidateAssertionParticipantPropertiesFollowSubject(t *testing.T) {
	newArchive := func(subject EntityRef, property string) *GLXFile {
		return &GLXFile{
			Persons:                map[string]*Person{"alice": {}, "bob": {}},
			Events:                 map[string]*Event{"event-1": {Type: "wedding", Participants: []Participant{{Person: "alice"}}}},
			Relationships:          map[string]*Relationship{"rel-1": {Type: "union", Participants: []Participant{{Person: "alice"}, {Person: "bob"}}}},
			EventTypes:             map[string]*VocabularyEntry{"wedding": {Label: "Wedding"}},
			RelationshipTypes:      map[string]*VocabularyEntry{"union": {Label: "Union"}},
			EventProperties:        map[string]*PropertyDefinition{"event_only": {Label: "Event only"}},
			RelationshipProperties: map[string]*PropertyDefinition{"relationship_only": {Label: "Relationship only"}},
			Assertions: map[string]*Assertion{
				"claim": {Subject: subject, Participant: &Participant{Person: "alice", Properties: map[string]any{property: "x"}}},
			},
		}
	}
	unknownWarnings := func(archive *GLXFile) string {
		var b strings.Builder
		for _, w := range archive.Validate().Warnings {
			if w.SourceType == EntityTypeAssertions {
				b.WriteString(w.Message + "\n")
			}
		}

		return b.String()
	}

	assert.Empty(t, unknownWarnings(newArchive(EntityRef{Relationship: "rel-1"}, "relationship_only")))
	assert.Contains(t, unknownWarnings(newArchive(EntityRef{Relationship: "rel-1"}, "event_only")), "event_only")
	assert.Empty(t, unknownWarnings(newArchive(EntityRef{Event: "event-1"}, "event_only")))
	assert.Contains(t, unknownWarnings(newArchive(EntityRef{Event: "event-1"}, "relationship_only")), "relationship_only")
}

// A duplicate ID across files names both files (#1232).
func TestDeserializeMultiFileConflictNamesBothFiles(t *testing.T) {
	files := map[string][]byte{
		"persons/alice.glx":        []byte("persons:\n  person-alice:\n    properties:\n      primary_name: Alice\n"),
		"backup/persons/alice.glx": []byte("persons:\n  person-alice:\n    properties:\n      primary_name: Alice again\n"),
	}
	_, conflicts, err := NewSerializer(&SerializerOptions{}).DeserializeMultiFileFromMap(files)
	require.NoError(t, err)
	require.Equal(t, []string{
		"conflict persons ID: person-alice (backup/persons/alice.glx vs persons/alice.glx)",
	}, conflicts)
}

// StandardVocabularyFallback fills only what the archive leaves out (#1355).
func TestDeserializeMultiFileStandardVocabularyFallback(t *testing.T) {
	source := []byte("sources:\n  source-1:\n    title: Births\n    type: vital_record\n")

	t.Run("off: a missing vocabulary leaves the type undefined", func(t *testing.T) {
		_, _, err := NewSerializer(&SerializerOptions{Validate: true}).
			DeserializeMultiFileFromMap(map[string][]byte{"sources/source-1.glx": source})
		assert.Contains(t, structuredErrorMessages(t, err), "source_types: vital_record")
	})

	t.Run("on: the standard vocabulary supplies it", func(t *testing.T) {
		_, _, err := NewSerializer(&SerializerOptions{Validate: true, StandardVocabularyFallback: true}).
			DeserializeMultiFileFromMap(map[string][]byte{"sources/source-1.glx": source})
		require.NoError(t, err)
	})

	t.Run("on: a vocabulary the archive defines stays authoritative", func(t *testing.T) {
		_, _, err := NewSerializer(&SerializerOptions{Validate: true, StandardVocabularyFallback: true}).
			DeserializeMultiFileFromMap(map[string][]byte{
				"sources/source-1.glx":          source,
				"vocabularies/source-types.glx": []byte("source_types:\n  diary:\n    label: Diary\n"),
			})
		assert.Contains(t, structuredErrorMessages(t, err), "source_types: vital_record")
	})
}

// structuredErrorMessages returns the messages of the StructuredValidationError
// err wraps, one per line.
func structuredErrorMessages(t *testing.T, err error) string {
	t.Helper()
	var structured *StructuredValidationError
	require.ErrorAs(t, err, &structured)

	return errorMessages(structured.Errors)
}

func errorMessages(errs []ValidationError) string {
	var b strings.Builder
	for _, e := range errs {
		b.WriteString(e.Message + "\n")
	}

	return b.String()
}

func TestAssertionStructuralPropertiesMatchSpec(t *testing.T) {
	assert.ElementsMatch(t, []string{"title", "type", "place", "date"}, assertionStructuralProperties[EntityTypeEvents])
	assert.ElementsMatch(t, []string{"name", "type", "parent", "latitude", "longitude"}, assertionStructuralProperties[EntityTypePlaces])
	assert.ElementsMatch(t, []string{"type", "start_event", "end_event"}, assertionStructuralProperties[EntityTypeRelationships])
}
