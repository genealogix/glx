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

// GenderTypesSchema contains the embedded gender-types vocabulary schema.
//
//go:embed vocabularies/gender-types.schema.json
var GenderTypesSchema []byte

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

// MediaPropertiesSchema contains the embedded media-properties vocabulary schema.
//
//go:embed vocabularies/media-properties.schema.json
var MediaPropertiesSchema []byte

// RepositoryPropertiesSchema contains the embedded repository-properties vocabulary schema.
//
//go:embed vocabularies/repository-properties.schema.json
var RepositoryPropertiesSchema []byte

// CitationPropertiesSchema contains the embedded citation-properties vocabulary schema.
//
//go:embed vocabularies/citation-properties.schema.json
var CitationPropertiesSchema []byte

// SourcePropertiesSchema contains the embedded source-properties vocabulary schema.
//
//go:embed vocabularies/source-properties.schema.json
var SourcePropertiesSchema []byte
