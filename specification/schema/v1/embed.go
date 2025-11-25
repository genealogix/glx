// Package v1 provides embedded JSON schema files for GENEALOGIX v1 specification.
package v1

import (
	"embed"
)

// EntitySchemas contains all schema files including vocabularies
//
//go:embed *.schema.json
//go:embed vocabularies/*.schema.json
var EntitySchemas embed.FS

// RelationshipTypesSchema contains the embedded relationship-types vocabulary schema.
//
//go:embed vocabularies/relationship-types.schema.json
var RelationshipTypesSchema []byte

// EventTypesSchema contains the embedded event-types vocabulary schema.
//
//go:embed vocabularies/event-types.schema.json
var EventTypesSchema []byte

// PlaceTypesSchema contains the embedded place-types vocabulary schema.
//
//go:embed vocabularies/place-types.schema.json
var PlaceTypesSchema []byte

// RepositoryTypesSchema contains the embedded repository-types vocabulary schema.
//
//go:embed vocabularies/repository-types.schema.json
var RepositoryTypesSchema []byte

// ParticipantRolesSchema contains the embedded participant-roles vocabulary schema.
//
//go:embed vocabularies/participant-roles.schema.json
var ParticipantRolesSchema []byte

// MediaTypesSchema contains the embedded media-types vocabulary schema.
//
//go:embed vocabularies/media-types.schema.json
var MediaTypesSchema []byte

// ConfidenceLevelsSchema contains the embedded confidence-levels vocabulary schema.
//
//go:embed vocabularies/confidence-levels.schema.json
var ConfidenceLevelsSchema []byte

// PersonPropertiesSchema contains the embedded person-properties vocabulary schema.
//
//go:embed vocabularies/person-properties.schema.json
var PersonPropertiesSchema []byte

// EventPropertiesSchema contains the embedded event-properties vocabulary schema.
//
//go:embed vocabularies/event-properties.schema.json
var EventPropertiesSchema []byte

// RelationshipPropertiesSchema contains the embedded relationship-properties vocabulary schema.
//
//go:embed vocabularies/relationship-properties.schema.json
var RelationshipPropertiesSchema []byte

// PlacePropertiesSchema contains the embedded place-properties vocabulary schema.
//
//go:embed vocabularies/place-properties.schema.json
var PlacePropertiesSchema []byte
