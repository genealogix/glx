// Package vocabularies provides embedded standard vocabulary files for GENEALOGIX.
package vocabularies

import _ "embed"

// RelationshipTypes contains the embedded relationship-types.glx vocabulary file.
//
//go:embed relationship-types.glx
var RelationshipTypes []byte

// EventTypes contains the embedded event-types.glx vocabulary file.
//
//go:embed event-types.glx
var EventTypes []byte

// PlaceTypes contains the embedded place-types.glx vocabulary file.
//
//go:embed place-types.glx
var PlaceTypes []byte

// RepositoryTypes contains the embedded repository-types.glx vocabulary file.
//
//go:embed repository-types.glx
var RepositoryTypes []byte

// SourceTypes contains the embedded source-types.glx vocabulary file.
//
//go:embed source-types.glx
var SourceTypes []byte

// ParticipantRoles contains the embedded participant-roles.glx vocabulary file.
//
//go:embed participant-roles.glx
var ParticipantRoles []byte

// MediaTypes contains the embedded media-types.glx vocabulary file.
//
//go:embed media-types.glx
var MediaTypes []byte

// ConfidenceLevels contains the embedded confidence-levels.glx vocabulary file.
//
//go:embed confidence-levels.glx
var ConfidenceLevels []byte

// PersonProperties contains the embedded person-properties.glx vocabulary file.
//
//go:embed person-properties.glx
var PersonProperties []byte

// EventProperties contains the embedded event-properties.glx vocabulary file.
//
//go:embed event-properties.glx
var EventProperties []byte

// RelationshipProperties contains the embedded relationship-properties.glx vocabulary file.
//
//go:embed relationship-properties.glx
var RelationshipProperties []byte

// PlaceProperties contains the embedded place-properties.glx vocabulary file.
//
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
	"person-properties.glx":       PersonProperties,
	"event-properties.glx":        EventProperties,
	"relationship-properties.glx": RelationshipProperties,
	"place-properties.glx":        PlaceProperties,
}
