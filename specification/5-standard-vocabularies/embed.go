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

//go:embed source-types.glx
var SourceTypes []byte

//go:embed participant-roles.glx
var ParticipantRoles []byte

//go:embed media-types.glx
var MediaTypes []byte

//go:embed confidence-levels.glx
var ConfidenceLevels []byte

//go:embed quality-ratings.glx
var QualityRatings []byte

//go:embed person-properties.glx
var PersonProperties []byte

//go:embed event-properties.glx
var EventProperties []byte

//go:embed relationship-properties.glx
var RelationshipProperties []byte

//go:embed place-properties.glx
var PlaceProperties []byte

// Files maps output filenames to embedded content
var Files = map[string][]byte{
	"relationship-types.glx":      RelationshipTypes,
	"event-types.glx":             EventTypes,
	"place-types.glx":             PlaceTypes,
	"repository-types.glx":        RepositoryTypes,
	"source-types.glx":            SourceTypes,
	"participant-roles.glx":       ParticipantRoles,
	"media-types.glx":             MediaTypes,
	"confidence-levels.glx":       ConfidenceLevels,
	"quality-ratings.glx":         QualityRatings,
	"person-properties.glx":       PersonProperties,
	"event-properties.glx":        EventProperties,
	"relationship-properties.glx": RelationshipProperties,
	"place-properties.glx":        PlaceProperties,
}
