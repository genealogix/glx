package vocabularies

import _ "embed"

//go:embed relationship-types.glx
var RelationshipTypes []byte

//go:embed event-types.glx
var EventTypes []byte

//go:embed place-types.glx
var PlaceTypes []byte

//go:embed repository-types.glx
var RepositoryTypes []byte

//go:embed participant-roles.glx
var ParticipantRoles []byte

//go:embed media-types.glx
var MediaTypes []byte

//go:embed confidence-levels.glx
var ConfidenceLevels []byte

//go:embed quality-ratings.glx
var QualityRatings []byte

// Files maps output filenames to embedded content
var Files = map[string][]byte{
	"relationship-types.glx": RelationshipTypes,
	"event-types.glx":        EventTypes,
	"place-types.glx":        PlaceTypes,
	"repository-types.glx":   RepositoryTypes,
	"participant-roles.glx":  ParticipantRoles,
	"media-types.glx":        MediaTypes,
	"confidence-levels.glx":  ConfidenceLevels,
	"quality-ratings.glx":    QualityRatings,
}
