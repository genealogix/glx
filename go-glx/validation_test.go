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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEntityReferences(t *testing.T) {
	t.Run("valid references", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events:  map[string]*Event{"event-1": {Participants: []Participant{{Person: "person-1"}}}},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("missing entity reference", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{"event-1": {PlaceID: "place-nonexistent"}},
			Places: map[string]*Place{},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "events", err.SourceType)
		assert.Equal(t, "event-1", err.SourceID)
		assert.Equal(t, "PlaceID", err.SourceField)
		assert.Equal(t, "places", err.TargetType)
		assert.Equal(t, "place-nonexistent", err.TargetID)
	})

	t.Run("missing vocabulary reference", func(t *testing.T) {
		archive := &GLXFile{
			Events:     map[string]*Event{"event-1": {Type: "marriage-invalid"}},
			EventTypes: map[string]*EventType{"marriage": {Label: "Marriage"}},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "events", err.SourceType)
		assert.Equal(t, "event-1", err.SourceID)
		assert.Equal(t, "Type", err.SourceField)
		assert.Equal(t, "event_types", err.TargetType)
		assert.Equal(t, "marriage-invalid", err.TargetID)
	})
}

func TestValidatePropertyReferences(t *testing.T) {
	t.Run("valid property reference", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"birth_place": "place-1"}},
			},
			Places: map[string]*Place{"place-1": {Name: "Test Place"}},
			PersonProperties: map[string]*PropertyDefinition{
				"birth_place": {ReferenceType: "places"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid property reference", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"birth_place": "place-nonexistent"}},
			},
			Places: map[string]*Place{},
			PersonProperties: map[string]*PropertyDefinition{
				"birth_place": {ReferenceType: "places"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "persons", err.SourceType)
		assert.Equal(t, "person-1", err.SourceID)
		assert.Equal(t, "properties.birth_place", err.SourceField)
		assert.Equal(t, "places", err.TargetType)
		assert.Equal(t, "place-nonexistent", err.TargetID)
	})
}

func TestValidatePropertyWarnings(t *testing.T) {
	t.Run("unknown property", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"shoe_size": 42}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"birth_place": {ReferenceType: "places"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.shoe_size", warn.Field)
		assert.Contains(t, warn.Message, "unknown property 'shoe_size'")
	})

	t.Run("missing property vocabulary", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"shoe_size": 42}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties", warn.Field)
		assert.Contains(t, warn.Message, "no person_properties vocabulary was found")
	})

	t.Run("removed property born_on", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{DeprecatedPropertyBornOn: "1840"}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"occupation": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		ve := result.Errors[0]
		assert.Equal(t, "persons", ve.SourceType)
		assert.Equal(t, "person-1", ve.SourceID)
		assert.Equal(t, "properties.born_on", ve.SourceField)
		assert.Contains(t, ve.Message, "has been removed")
		assert.Contains(t, ve.Message, "use birth events instead")
		assert.Empty(t, result.Warnings)
	})

	t.Run("removed property died_on", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{DeprecatedPropertyDiedOn: "1910"}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"occupation": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		ve := result.Errors[0]
		assert.Equal(t, "properties.died_on", ve.SourceField)
		assert.Contains(t, ve.Message, "has been removed")
		assert.Contains(t, ve.Message, "use death events instead")
	})

	t.Run("removed property born_at", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{DeprecatedPropertyBornAt: "place-london"}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"occupation": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		ve := result.Errors[0]
		assert.Equal(t, "properties.born_at", ve.SourceField)
		assert.Contains(t, ve.Message, "has been removed")
		assert.Contains(t, ve.Message, "use birth events instead")
	})

	t.Run("removed property died_at", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{DeprecatedPropertyDiedAt: "place-london"}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"occupation": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		ve := result.Errors[0]
		assert.Equal(t, "properties.died_at", ve.SourceField)
		assert.Contains(t, ve.Message, "has been removed")
		assert.Contains(t, ve.Message, "use death events instead")
	})

	t.Run("removed property caught without vocabulary", func(t *testing.T) {
		// Archives with no PersonProperties vocabulary should still error
		// on deprecated properties, not just warn about missing vocabulary.
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{
					DeprecatedPropertyBornOn: "1850",
					"occupation":             "blacksmith",
				}},
			},
			// No PersonProperties vocabulary at all
		}
		result := archive.Validate()
		foundRemovedError := false
		for _, e := range result.Errors {
			if e.SourceField == "properties.born_on" {
				foundRemovedError = true
				assert.Contains(t, e.Message, "has been removed")
			}
		}
		assert.True(t, foundRemovedError, "should error on deprecated property even without vocabulary")
	})
}

func TestValidateNestedStructReferences(t *testing.T) {
	t.Run("event participants valid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{
						{Person: "person-1", Role: "bride"},
						{Person: "person-2", Role: "groom"},
					},
				},
			},
			ParticipantRoles: map[string]*ParticipantRole{
				"bride": {Label: "Bride"},
				"groom": {Label: "Groom"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("event participant invalid person", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{{Person: "person-nonexistent"}},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-nonexistent")
	})

	t.Run("event participant invalid role", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{{Person: "person-1", Role: "invalid-role"}},
				},
			},
			ParticipantRoles: map[string]*ParticipantRole{},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "invalid-role")
	})

	t.Run("relationship participants valid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{
						{Person: "person-1", Role: "spouse"},
						{Person: "person-2", Role: "spouse"},
					},
				},
			},
			ParticipantRoles: map[string]*ParticipantRole{
				"spouse": {Label: "Spouse"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("relationship participants invalid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{{Person: "person-missing"}},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-missing")
	})

	t.Run("assertion participant valid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {
					Subject: EntityRef{Event: "event-1"},
					Participant: &Participant{
						Person: "person-1",
						Role:   "witness",
					},
				},
			},
			Events: map[string]*Event{"event-1": {}},
			ParticipantRoles: map[string]*ParticipantRole{
				"witness": {Label: "Witness"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion participant invalid person", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{},
			Assertions: map[string]*Assertion{
				"assert-1": {
					Subject: EntityRef{Event: "event-1"},
					Participant: &Participant{
						Person: "person-nonexistent",
					},
				},
			},
			Events: map[string]*Event{"event-1": {}},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-nonexistent")
	})
}

func TestValidateSliceReferences(t *testing.T) {
	t.Run("relationship persons valid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {},
				"person-2": {},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{
						{Person: "person-1"},
						{Person: "person-2"},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("relationship persons invalid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{
						{Person: "person-1"},
						{Person: "person-missing"},
					},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-missing")
	})

	t.Run("source media valid", func(t *testing.T) {
		archive := &GLXFile{
			Media: map[string]*Media{
				"media-1": {},
				"media-2": {},
			},
			Sources: map[string]*Source{
				"source-1": {
					Media: []string{"media-1", "media-2"},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("source media invalid", func(t *testing.T) {
		archive := &GLXFile{
			Media: map[string]*Media{},
			Sources: map[string]*Source{
				"source-1": {
					Media: []string{"media-missing"},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "media-missing")
	})

	t.Run("citation media invalid", func(t *testing.T) {
		archive := &GLXFile{
			Media: map[string]*Media{"media-1": {}},
			Citations: map[string]*Citation{
				"citation-1": {
					Media: []string{"media-1", "media-missing"},
				},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "media-missing")
	})

	t.Run("assertion citations valid", func(t *testing.T) {
		archive := &GLXFile{
			Citations: map[string]*Citation{
				"citation-1": {},
				"citation-2": {},
			},
			Assertions: map[string]*Assertion{
				"assert-1": {
					Subject:   EntityRef{Person: "person-1"},
					Citations: []string{"citation-1", "citation-2"},
				},
			},
			Persons: map[string]*Person{"person-1": {}},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion sources valid", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{
				"source-1": {},
			},
			Assertions: map[string]*Assertion{
				"assert-1": {
					Subject: EntityRef{Person: "person-1"},
					Sources: []string{"source-1"},
				},
			},
			Persons: map[string]*Person{"person-1": {}},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})
}

func TestValidateMultiTypeReferences(t *testing.T) {
	t.Run("assertion subject references person", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Person: "person-1"}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references event", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{"event-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Event: "event-1"}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references relationship", func(t *testing.T) {
		archive := &GLXFile{
			Relationships: map[string]*Relationship{"rel-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Relationship: "rel-1"}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references place", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{"place-1": {Name: "Test"}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Place: "place-1"}},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject invalid person", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Person: "nonexistent"}},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "nonexistent")
	})

	t.Run("assertion subject invalid event", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Event: "nonexistent"}},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "nonexistent")
	})
}

func TestValidateRelationshipEventReferences(t *testing.T) {
	t.Run("relationship start and end events valid", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{
				"marriage": {},
				"divorce":  {},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					StartEvent: "marriage",
					EndEvent:   "divorce",
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("relationship start event invalid", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{},
			Relationships: map[string]*Relationship{
				"rel-1": {StartEvent: "event-missing"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "event-missing")
	})

	t.Run("relationship end event invalid", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{},
			Relationships: map[string]*Relationship{
				"rel-1": {EndEvent: "event-missing"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "event-missing")
	})
}

func TestValidateTemporalPropertyReferences(t *testing.T) {
	t.Run("temporal property with valid references", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {
					Properties: map[string]any{
						"residence": []any{
							map[string]any{"value": "place-1", "date": "1900"},
							map[string]any{"value": "place-2", "date": "1910"},
						},
					},
				},
			},
			Places: map[string]*Place{
				"place-1": {Name: "City A"},
				"place-2": {Name: "City B"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"residence": {ReferenceType: "places", Temporal: boolPtr(true)},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("temporal property with invalid reference", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {
					Properties: map[string]any{
						"residence": []any{
							map[string]any{"value": "place-1", "date": "1900"},
							map[string]any{"value": "place-missing", "date": "1910"},
						},
					},
				},
			},
			Places: map[string]*Place{
				"place-1": {Name: "City A"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"residence": {ReferenceType: "places", Temporal: boolPtr(true)},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		err := result.Errors[0]
		assert.Equal(t, "persons", err.SourceType)
		assert.Equal(t, "person-1", err.SourceID)
		assert.Equal(t, "properties.residence[1].value", err.SourceField)
		assert.Contains(t, err.Message, "place-missing")
	})
}

func TestValidateAllVocabularyTypes(t *testing.T) {
	t.Run("all vocabulary types valid", func(t *testing.T) {
		archive := &GLXFile{
			Events:        map[string]*Event{"event-1": {Type: "birth"}},
			Relationships: map[string]*Relationship{"rel-1": {Type: "spouse"}},
			Places:        map[string]*Place{"place-1": {Name: "Test", Type: "city"}},
			Sources:       map[string]*Source{"source-1": {Title: "Test", Type: "book"}},
			Repositories:  map[string]*Repository{"repo-1": {Name: "Test", Type: "library"}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: EntityRef{Event: "event-1"}, Confidence: "high"},
			},

			EventTypes:        map[string]*EventType{"birth": {Label: "Birth"}},
			RelationshipTypes: map[string]*RelationshipType{"spouse": {Label: "Spouse"}},
			PlaceTypes:        map[string]*PlaceType{"city": {Label: "City"}},
			SourceTypes:       map[string]*SourceType{"book": {Label: "Book"}},
			RepositoryTypes:   map[string]*RepositoryType{"library": {Label: "Library"}},
			ConfidenceLevels:  map[string]*ConfidenceLevel{"high": {Label: "High"}},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("missing vocabulary types generate errors", func(t *testing.T) {
		archive := &GLXFile{
			Events:            map[string]*Event{"event-1": {Type: "invalid"}},
			Relationships:     map[string]*Relationship{"rel-1": {Type: "invalid"}},
			Places:            map[string]*Place{"place-1": {Name: "Test", Type: "invalid"}},
			Sources:           map[string]*Source{"source-1": {Title: "Test", Type: "invalid"}},
			Repositories:      map[string]*Repository{"repo-1": {Name: "Test", Type: "invalid"}},
			EventTypes:        map[string]*EventType{},
			RelationshipTypes: map[string]*RelationshipType{},
			PlaceTypes:        map[string]*PlaceType{},
			SourceTypes:       map[string]*SourceType{},
			RepositoryTypes:   map[string]*RepositoryType{},
		}
		result := archive.Validate()
		assert.GreaterOrEqual(t, len(result.Errors), 5)
	})
}

func TestValidationCaching(t *testing.T) {
	t.Run("validation results are cached", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
		}

		result1 := archive.Validate()
		result2 := archive.Validate()

		// Should return same instance
		assert.Same(t, result1, result2)
	})

	t.Run("cache invalidation works", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
		}

		result1 := archive.Validate()
		archive.InvalidateCache()
		result2 := archive.Validate()

		// Should return different instance
		assert.NotSame(t, result1, result2)
	})
}

func TestValidatePlaceHierarchy(t *testing.T) {
	t.Run("place parent reference valid", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"country": {Name: "USA"},
				"state":   {Name: "California", ParentID: "country"},
				"city":    {Name: "San Francisco", ParentID: "state"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("place parent reference invalid", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"city": {Name: "San Francisco", ParentID: "state-missing"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "state-missing")
	})

	t.Run("place self-referencing cycle", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"place-1": {Name: "Paradox", ParentID: "place-1"},
			},
		}
		result := archive.Validate()
		var cycleErrors []ValidationError
		for _, e := range result.Errors {
			if e.SourceField == "parent" {
				cycleErrors = append(cycleErrors, e)
			}
		}
		require.Len(t, cycleErrors, 1)
		assert.Equal(t, "places", cycleErrors[0].SourceType)
		assert.Equal(t, "place-1", cycleErrors[0].SourceID)
		assert.Contains(t, cycleErrors[0].Message, "cycle detected")
	})

	t.Run("place two-node cycle", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"a": {Name: "A", ParentID: "b"},
				"b": {Name: "B", ParentID: "a"},
			},
		}
		result := archive.Validate()
		var cycleErrors []ValidationError
		for _, e := range result.Errors {
			if e.SourceField == "parent" {
				cycleErrors = append(cycleErrors, e)
			}
		}
		require.Len(t, cycleErrors, 1)
		assert.Contains(t, cycleErrors[0].Message, "cycle detected")
	})

	t.Run("place three-node cycle", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"a": {Name: "A", ParentID: "b"},
				"b": {Name: "B", ParentID: "c"},
				"c": {Name: "C", ParentID: "a"},
			},
		}
		result := archive.Validate()
		var cycleErrors []ValidationError
		for _, e := range result.Errors {
			if e.SourceField == "parent" {
				cycleErrors = append(cycleErrors, e)
			}
		}
		require.Len(t, cycleErrors, 1)
		assert.Contains(t, cycleErrors[0].Message, "cycle detected")
	})

	t.Run("chain leading into cycle reports one error", func(t *testing.T) {
		// d -> a -> b -> c -> a (d is not in the cycle, but a-b-c form one)
		archive := &GLXFile{
			Places: map[string]*Place{
				"a": {Name: "A", ParentID: "b"},
				"b": {Name: "B", ParentID: "c"},
				"c": {Name: "C", ParentID: "a"},
				"d": {Name: "D", ParentID: "a"},
			},
		}
		result := archive.Validate()
		var cycleErrors []ValidationError
		for _, e := range result.Errors {
			if e.SourceField == "parent" {
				cycleErrors = append(cycleErrors, e)
			}
		}
		require.Len(t, cycleErrors, 1)
		assert.Contains(t, cycleErrors[0].Message, "cycle detected")
	})

	t.Run("valid deep hierarchy no cycle", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{
				"country":  {Name: "USA"},
				"state":    {Name: "California", ParentID: "country"},
				"county":   {Name: "San Francisco County", ParentID: "state"},
				"city":     {Name: "San Francisco", ParentID: "county"},
				"district": {Name: "Mission District", ParentID: "city"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})
}

func TestValidateRepositoryReferences(t *testing.T) {
	t.Run("source repository valid", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{
				"source-1": {Title: "Test", RepositoryID: "repo-1"},
			},
			Repositories: map[string]*Repository{
				"repo-1": {Name: "Test Repository"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("citation repository valid", func(t *testing.T) {
		archive := &GLXFile{
			Citations: map[string]*Citation{
				"citation-1": {RepositoryID: "repo-1"},
			},
			Repositories: map[string]*Repository{
				"repo-1": {Name: "Test Repository"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("citation source valid", func(t *testing.T) {
		archive := &GLXFile{
			Citations: map[string]*Citation{
				"citation-1": {SourceID: "source-1"},
			},
			Sources: map[string]*Source{
				"source-1": {Title: "Test"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})
}

func TestValidateCitationProperties(t *testing.T) {
	t.Run("valid citation properties", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{"source-1": {Title: "Test Source"}},
			Citations: map[string]*Citation{
				"citation-1": {
					SourceID: "source-1",
					Properties: map[string]any{
						"locator":          "page 42",
						"text_from_source": "The text from the source",
					},
				},
			},
			CitationProperties: map[string]*PropertyDefinition{
				"locator":          {Label: "Locator", ValueType: "string"},
				"text_from_source": {Label: "Text From Source", ValueType: "string"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("unknown citation property", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{"source-1": {Title: "Test Source"}},
			Citations: map[string]*Citation{
				"citation-1": {
					SourceID: "source-1",
					Properties: map[string]any{
						"unknown_prop": "some value",
					},
				},
			},
			CitationProperties: map[string]*PropertyDefinition{
				"locator": {Label: "Locator", ValueType: "string"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Equal(t, "citations", result.Warnings[0].SourceType)
		assert.Equal(t, "citation-1", result.Warnings[0].SourceID)
		assert.Contains(t, result.Warnings[0].Message, "unknown property 'unknown_prop'")
	})

	t.Run("missing citation properties vocabulary", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{"source-1": {Title: "Test Source"}},
			Citations: map[string]*Citation{
				"citation-1": {
					SourceID: "source-1",
					Properties: map[string]any{
						"locator": "page 42",
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "no citation_properties vocabulary was found")
	})
}

func TestValidateMultiValueReferenceProperties(t *testing.T) {
	t.Run("valid multi-value reference array", func(t *testing.T) {
		archive := &GLXFile{
			Media: map[string]*Media{
				"media-1": {Properties: map[string]any{
					"subjects": []any{"person-1", "person-2"},
				}},
			},
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			MediaProperties: map[string]*PropertyDefinition{
				"subjects": {ReferenceType: "persons", MultiValue: boolPtr(true)},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("invalid multi-value reference in array", func(t *testing.T) {
		archive := &GLXFile{
			Media: map[string]*Media{
				"media-1": {Properties: map[string]any{
					"subjects": []any{"person-1", "person-gone"},
				}},
			},
			Persons: map[string]*Person{"person-1": {}},
			MediaProperties: map[string]*PropertyDefinition{
				"subjects": {ReferenceType: "persons", MultiValue: boolPtr(true)},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "person-gone")
		assert.Contains(t, result.Errors[0].SourceField, "subjects[1]")
	})
}

func TestValidateStructuredPropertyValue(t *testing.T) {
	t.Run("non-temporal structured value with fields", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{
					"name": map[string]any{
						"value":  "John Smith",
						"fields": map[string]any{"given": "John", "surname": "Smith"},
					},
				}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"name": {
					ValueType: "string",
					Fields: map[string]*FieldDefinition{
						"given":   {Label: "Given name"},
						"surname": {Label: "Surname"},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("non-temporal structured value with unknown field", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{
					"name": map[string]any{
						"value":  "John Smith",
						"fields": map[string]any{"given": "John", "middle": "Q"},
					},
				}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"name": {
					ValueType: "string",
					Fields: map[string]*FieldDefinition{
						"given":   {Label: "Given name"},
						"surname": {Label: "Surname"},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "unknown field")
		assert.Contains(t, result.Warnings[0].Field, "fields.middle")
	})

	t.Run("multi-value with structured objects", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{
					"external_ids": []any{
						map[string]any{"value": "ABC-123"},
						map[string]any{"value": "DEF-456"},
					},
				}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"external_ids": {ValueType: "string", MultiValue: boolPtr(true)},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})
}

// Helper function for test readability
func boolPtr(b bool) *bool {
	return &b
}

func TestValidateSuggestReferenceKey(t *testing.T) {
	t.Run("suggests underscore when hyphen used in vocabulary type", func(t *testing.T) {
		archive := &GLXFile{
			Relationships:     map[string]*Relationship{"rel-1": {Type: "parent-child"}},
			RelationshipTypes: map[string]*RelationshipType{"parent_child": {Label: "Parent-Child"}},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "did you mean 'parent_child'?")
	})

	t.Run("suggests hyphen when underscore used in entity ref", func(t *testing.T) {
		archive := &GLXFile{
			Events:     map[string]*Event{"event-1": {PlaceID: "place_leeds"}},
			Places:     map[string]*Place{"place-leeds": {Name: "Leeds"}},
			EventTypes: map[string]*EventType{},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Contains(t, result.Errors[0].Message, "did you mean 'place-leeds'?")
	})

	t.Run("no suggestion when no close match exists", func(t *testing.T) {
		archive := &GLXFile{
			Relationships:     map[string]*Relationship{"rel-1": {Type: "nonexistent"}},
			RelationshipTypes: map[string]*RelationshipType{"parent_child": {Label: "Parent-Child"}},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.NotContains(t, result.Errors[0].Message, "did you mean")
	})
}

func TestValidateParticipantProperties(t *testing.T) {
	t.Run("valid participant properties on event", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"age_at_event": 42}},
					},
				},
			},
			EventProperties: map[string]*PropertyDefinition{
				"age_at_event": {ValueType: "integer"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("unknown participant property warns", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"unknown_prop": "value"}},
					},
				},
			},
			EventProperties: map[string]*PropertyDefinition{
				"age_at_event": {ValueType: "integer"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "unknown property 'unknown_prop'")
		assert.Equal(t, "events", result.Warnings[0].SourceType)
		assert.Equal(t, "event-1 participants[0]", result.Warnings[0].SourceID)
	})

	t.Run("participant property reference validated", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Places:  map[string]*Place{},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"residence": "place-missing"}},
					},
				},
			},
			EventProperties: map[string]*PropertyDefinition{
				"residence": {ReferenceType: "places"},
			},
		}
		result := archive.Validate()
		require.Len(t, result.Errors, 1)
		assert.Equal(t, "events", result.Errors[0].SourceType)
		assert.Equal(t, "event-1 participants[0]", result.Errors[0].SourceID)
		assert.Contains(t, result.Errors[0].SourceField, "residence")
	})

	t.Run("missing event property vocab warns for participant properties", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"age_at_event": 42}},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "event_properties")
	})

	t.Run("participant properties on assertion", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events:  map[string]*Event{"event-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {
					Subject:     EntityRef{Event: "event-1"},
					Participant: &Participant{Person: "person-1", Properties: map[string]any{"unknown_prop": "val"}},
				},
			},
			EventProperties: map[string]*PropertyDefinition{
				"age_at_event": {ValueType: "integer"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "unknown property 'unknown_prop'")
		assert.Equal(t, "assertions", result.Warnings[0].SourceType)
		assert.Equal(t, "assert-1 participant", result.Warnings[0].SourceID)
	})

	t.Run("relationship participant uses relationship_properties vocab", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"custody_type": "full"}},
						{Person: "person-2"},
					},
				},
			},
			RelationshipProperties: map[string]*PropertyDefinition{
				"custody_type": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("relationship participant unknown prop warns against relationship vocab", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"unknown_prop": "val"}},
						{Person: "person-2"},
					},
				},
			},
			RelationshipProperties: map[string]*PropertyDefinition{
				"custody_type": {ValueType: "string"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0].Message, "unknown property 'unknown_prop'")
		assert.Equal(t, "relationships", result.Warnings[0].SourceType)
		assert.Equal(t, "rel-1 participants[0]", result.Warnings[0].SourceID)
	})

	t.Run("missing vocab warning not duplicated for entity and participant properties", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Events: map[string]*Event{
				"event-1": {
					Properties: map[string]any{"description": "test"},
					Participants: []Participant{
						{Person: "person-1", Properties: map[string]any{"age_at_event": 42}},
						{Person: "person-2", Properties: map[string]any{"age_at_event": 28}},
					},
				},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		// Should get exactly 1 warning for missing vocab, not 3
		vocabWarnings := 0
		for _, w := range result.Warnings {
			if w.SourceType == "events" {
				vocabWarnings++
			}
		}
		assert.Equal(t, 1, vocabWarnings, "should have exactly 1 missing-vocab warning per entity, not per participant")
	})
}

func TestValidatePropertyVocabularyValue(t *testing.T) {
	// Helper to build a GLXFile with gender_types vocabulary loaded and a person
	// whose "gender" property uses vocabulary_type: gender_types.
	makeArchive := func(genderValue any) *GLXFile {
		return &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"gender": genderValue}},
			},
			GenderTypes: map[string]*GenderType{
				"male":   {Label: "Male"},
				"female": {Label: "Female"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"gender": {Label: "Gender", VocabularyType: "gender_types"},
			},
		}
	}

	t.Run("simple string value valid", func(t *testing.T) {
		archive := makeArchive("male")
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("simple string value invalid", func(t *testing.T) {
		archive := makeArchive("nonbinary")
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender", warn.Field)
		assert.Contains(t, warn.Message, "'nonbinary' not found in gender_types")
	})

	t.Run("temporal object valid", func(t *testing.T) {
		archive := makeArchive(map[string]any{"value": "male", "date": "1990"})
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("temporal object invalid value", func(t *testing.T) {
		archive := makeArchive(map[string]any{"value": "invalid", "date": "1990"})
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender.value", warn.Field)
		assert.Contains(t, warn.Message, "'invalid' not found in gender_types")
	})

	t.Run("temporal list valid", func(t *testing.T) {
		archive := makeArchive([]any{
			map[string]any{"value": "male", "date": "FROM 1990"},
			map[string]any{"value": "female", "date": "FROM 2000"},
		})
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		assert.Empty(t, result.Warnings)
	})

	t.Run("temporal list with invalid value", func(t *testing.T) {
		archive := makeArchive([]any{
			map[string]any{"value": "male", "date": "FROM 1990"},
			map[string]any{"value": "invalid", "date": "FROM 2000"},
		})
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender[1].value", warn.Field)
		assert.Contains(t, warn.Message, "'invalid' not found in gender_types")
	})

	t.Run("multi-value list of strings", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"gender": []any{"male", "invalid"}}},
			},
			GenderTypes: map[string]*GenderType{
				"male":   {Label: "Male"},
				"female": {Label: "Female"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"gender": {Label: "Gender", VocabularyType: "gender_types", MultiValue: boolPtr(true)},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		// "male" is valid, "invalid" should produce a warning
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender[1]", warn.Field)
		assert.Contains(t, warn.Message, "'invalid' not found in gender_types")
	})

	t.Run("vocabulary not loaded", func(t *testing.T) {
		// Use a custom vocabulary type name that is NOT auto-registered in
		// buildVocabularyMaps, so the "not loaded" code path is exercised.
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"custom_field": "some_value"}},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"custom_field": {Label: "Custom Field", VocabularyType: "custom_vocab_types"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.custom_field", warn.Field)
		assert.Contains(t, warn.Message, "vocabulary 'custom_vocab_types' not loaded")
	})

	t.Run("conflicting property definition vocabulary_type and value_type", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"gender": "male"}},
			},
			GenderTypes: map[string]*GenderType{
				"male":   {Label: "Male"},
				"female": {Label: "Female"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				// Conflicting: both vocabulary_type and value_type are set.
				// Value "male" is valid in the vocabulary, so only the conflict warning fires.
				"gender": {Label: "Gender", VocabularyType: "gender_types", ValueType: "string"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender", warn.Field)
		assert.Contains(t, warn.Message, "conflicting type fields")
	})

	t.Run("conflicting property definition vocabulary_type and reference_type", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-1": {Properties: map[string]any{"gender": "male"}},
			},
			GenderTypes: map[string]*GenderType{
				"male":   {Label: "Male"},
				"female": {Label: "Female"},
			},
			PersonProperties: map[string]*PropertyDefinition{
				"gender": {Label: "Gender", VocabularyType: "gender_types", ReferenceType: "persons"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		warn := result.Warnings[0]
		assert.Equal(t, "persons", warn.SourceType)
		assert.Equal(t, "person-1", warn.SourceID)
		assert.Equal(t, "properties.gender", warn.Field)
		assert.Contains(t, warn.Message, "conflicting type fields")
	})
}
