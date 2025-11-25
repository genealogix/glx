package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateEntityReferences(t *testing.T) {
	t.Run("valid references", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events:  map[string]*Event{"event-1": {Participants: []EventParticipant{{PersonID: "person-1"}}}},
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
		assert.Contains(t, warn.Message, "no persons_properties vocabulary was found")
	})
}

func TestValidateNestedStructReferences(t *testing.T) {
	t.Run("event participants valid", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}, "person-2": {}},
			Events: map[string]*Event{
				"event-1": {
					Participants: []EventParticipant{
						{PersonID: "person-1", Role: "bride"},
						{PersonID: "person-2", Role: "groom"},
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
					Participants: []EventParticipant{{PersonID: "person-nonexistent"}},
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
					Participants: []EventParticipant{{PersonID: "person-1", Role: "invalid-role"}},
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
					Participants: []RelationshipParticipant{
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
					Participants: []RelationshipParticipant{{Person: "person-missing"}},
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
					Subject: "event-1",
					Participant: &AssertionParticipant{
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
					Subject: "event-1",
					Participant: &AssertionParticipant{
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
					Persons: []string{"person-1", "person-2"},
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
					Persons: []string{"person-1", "person-missing"},
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
					Subject:   "person-1",
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
					Subject: "person-1",
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
				"assert-1": {Subject: "person-1"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references event", func(t *testing.T) {
		archive := &GLXFile{
			Events: map[string]*Event{"event-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: "event-1"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references relationship", func(t *testing.T) {
		archive := &GLXFile{
			Relationships: map[string]*Relationship{"rel-1": {}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: "rel-1"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject references place", func(t *testing.T) {
		archive := &GLXFile{
			Places: map[string]*Place{"place-1": {Name: "Test"}},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: "place-1"},
			},
		}
		result := archive.Validate()
		assert.Empty(t, result.Errors)
	})

	t.Run("assertion subject invalid", func(t *testing.T) {
		archive := &GLXFile{
			Persons:       map[string]*Person{},
			Events:        map[string]*Event{},
			Relationships: map[string]*Relationship{},
			Places:        map[string]*Place{},
			Assertions: map[string]*Assertion{
				"assert-1": {Subject: "nonexistent"},
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
				"assert-1": {Subject: "event-1", Confidence: "high"},
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

// Helper function for test readability
func boolPtr(b bool) *bool {
	return &b
}
