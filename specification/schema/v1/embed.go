package v1

import _ "embed"

// Entity schemas
//
//go:embed person.schema.json
var PersonSchema []byte

//go:embed relationship.schema.json
var RelationshipSchema []byte

//go:embed event.schema.json
var EventSchema []byte

//go:embed place.schema.json
var PlaceSchema []byte

//go:embed source.schema.json
var SourceSchema []byte

//go:embed citation.schema.json
var CitationSchema []byte

//go:embed repository.schema.json
var RepositorySchema []byte

//go:embed assertion.schema.json
var AssertionSchema []byte

//go:embed media.schema.json
var MediaSchema []byte

// Vocabulary schemas
//
//go:embed vocabularies/relationship-types.schema.json
var RelationshipTypesSchema []byte

//go:embed vocabularies/event-types.schema.json
var EventTypesSchema []byte

//go:embed vocabularies/place-types.schema.json
var PlaceTypesSchema []byte

//go:embed vocabularies/repository-types.schema.json
var RepositoryTypesSchema []byte

//go:embed vocabularies/participant-roles.schema.json
var ParticipantRolesSchema []byte

//go:embed vocabularies/media-types.schema.json
var MediaTypesSchema []byte

//go:embed vocabularies/confidence-levels.schema.json
var ConfidenceLevelsSchema []byte

//go:embed vocabularies/quality-ratings.schema.json
var QualityRatingsSchema []byte

// EntitySchemas maps entity type names to their embedded schema content
var EntitySchemas = map[string][]byte{
	"person":       PersonSchema,
	"relationship": RelationshipSchema,
	"event":        EventSchema,
	"place":        PlaceSchema,
	"source":       SourceSchema,
	"citation":     CitationSchema,
	"repository":   RepositorySchema,
	"assertion":    AssertionSchema,
	"media":        MediaSchema,
}

// VocabularySchemas maps vocabulary type names to their embedded schema content
var VocabularySchemas = map[string][]byte{
	"relationship_types": RelationshipTypesSchema,
	"event_types":        EventTypesSchema,
	"place_types":        PlaceTypesSchema,
	"repository_types":   RepositoryTypesSchema,
	"participant_roles":  ParticipantRolesSchema,
	"media_types":        MediaTypesSchema,
	"confidence_levels":  ConfidenceLevelsSchema,
	"quality_ratings":    QualityRatingsSchema,
}
