package v1

import (
	"embed"
	_ "embed"
)

// EntitySchemas contains all schema files including vocabularies
//
//go:embed *.schema.json
//go:embed vocabularies/*.schema.json
var EntitySchemas embed.FS

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

//go:embed vocabularies/person-properties.schema.json
var PersonPropertiesSchema []byte

//go:embed vocabularies/event-properties.schema.json
var EventPropertiesSchema []byte

//go:embed vocabularies/relationship-properties.schema.json
var RelationshipPropertiesSchema []byte

//go:embed vocabularies/place-properties.schema.json
var PlacePropertiesSchema []byte
