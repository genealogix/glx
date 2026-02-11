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
	"fmt"
)

// GLXFile represents the top-level structure of a .glx file, which can
// contain maps of different entity types and vocabulary definitions.
type GLXFile struct { //nolint:revive // GLXFile is the established name across the codebase
	// Entity types
	Persons       map[string]*Person       `yaml:"persons,omitempty"`
	Relationships map[string]*Relationship `yaml:"relationships,omitempty"`
	Events        map[string]*Event        `yaml:"events,omitempty"`
	Places        map[string]*Place        `yaml:"places,omitempty"`
	Sources       map[string]*Source       `yaml:"sources,omitempty"`
	Citations     map[string]*Citation     `yaml:"citations,omitempty"`
	Repositories  map[string]*Repository   `yaml:"repositories,omitempty"`
	Assertions    map[string]*Assertion    `yaml:"assertions,omitempty"`
	Media         map[string]*Media        `yaml:"media,omitempty"`

	// Vocabulary definitions
	EventTypes        map[string]*EventType        `yaml:"event_types,omitempty"`
	ParticipantRoles  map[string]*ParticipantRole  `yaml:"participant_roles,omitempty"`
	ConfidenceLevels  map[string]*ConfidenceLevel  `yaml:"confidence_levels,omitempty"`
	RelationshipTypes map[string]*RelationshipType `yaml:"relationship_types,omitempty"`
	PlaceTypes        map[string]*PlaceType        `yaml:"place_types,omitempty"`
	SourceTypes       map[string]*SourceType       `yaml:"source_types,omitempty"`
	RepositoryTypes   map[string]*RepositoryType   `yaml:"repository_types,omitempty"`
	MediaTypes        map[string]*MediaType        `yaml:"media_types,omitempty"`

	// Property vocabularies
	PersonProperties       map[string]*PropertyDefinition `yaml:"person_properties,omitempty"`
	EventProperties        map[string]*PropertyDefinition `yaml:"event_properties,omitempty"`
	RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties,omitempty"`
	PlaceProperties        map[string]*PropertyDefinition `yaml:"place_properties,omitempty"`
	MediaProperties        map[string]*PropertyDefinition `yaml:"media_properties,omitempty"`
	RepositoryProperties   map[string]*PropertyDefinition `yaml:"repository_properties,omitempty"`
	CitationProperties     map[string]*PropertyDefinition `yaml:"citation_properties,omitempty"`
	SourceProperties       map[string]*PropertyDefinition `yaml:"source_properties,omitempty"`

	// Validation state (built on demand, cached)
	validation *ValidationResult
}

// ValidationResult holds the complete validation state of the archive.
type ValidationResult struct {
	// Entities contains maps of all existing entity IDs, keyed by entity type.
	// Example: "persons" -> {"person-1": {}}
	Entities map[string]map[string]struct{}

	// Vocabularies contains maps of all existing vocabulary values, keyed by vocabulary type.
	// Example: "event_types" -> {"birth": {}}
	Vocabularies map[string]map[string]struct{}

	// PropertyVocabs contains the definitions for custom properties, keyed by entity type.
	// Example: "persons" -> {"born_at" -> PropertyDefinition{...}}
	PropertyVocabs map[string]map[string]*PropertyDefinition

	// Errors is a slice of hard validation failures.
	Errors []ValidationError

	// Warnings is a slice of soft validation issues.
	Warnings []ValidationWarning

	validated bool // Internal flag to check if validation has been run.
}

// ValidationError represents a hard validation failure that makes the archive invalid.
type ValidationError struct {
	SourceType  string `json:"source_type"`  // e.g., "events"
	SourceID    string `json:"source_id"`    // e.g., "event-123"
	SourceField string `json:"source_field"` // e.g., "place" or "participants[0].role"
	TargetType  string `json:"target_type"`  // e.g., "places" or "participant_roles"
	TargetID    string `json:"target_id"`    // e.g., "place-nonexistent"
	Message     string `json:"message"`      // Human-readable error message
}

// ValidationWarning represents a soft validation issue that does not invalidate the archive.
type ValidationWarning struct {
	SourceType string `json:"source_type"` // e.g., "persons"
	SourceID   string `json:"source_id"`   // e.g., "person-123"
	Field      string `json:"field"`       // e.g., "properties.unknown_prop"
	Message    string `json:"message"`     // Human-readable warning message
}

// ============================================================================
// Entity Types
// ============================================================================
//
// The following types represent the core genealogical entities in a GLX archive.

// Person represents an individual in the family archive.
type Person struct {
	Properties map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes      string         `yaml:"notes,omitempty"`
}

// Participant defines a person's role in an event, relationship, or assertion.
type Participant struct {
	Person string `refType:"persons"           yaml:"person"`
	Role   string `refType:"participant_roles" yaml:"role,omitempty"`
	Notes  string `yaml:"notes,omitempty"`
}

// Relationship represents a relationship between two or more people.
type Relationship struct {
	Type         string         `refType:"relationship_types" yaml:"type"`
	Participants []Participant  `yaml:"participants"`
	StartEvent   string         `refType:"events"             yaml:"start_event,omitempty"`
	EndEvent     string         `refType:"events"             yaml:"end_event,omitempty"`
	Properties   map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes        string         `yaml:"notes,omitempty"`
}

// Event represents a genealogical event.
type Event struct {
	Type         string         `refType:"event_types"       yaml:"type"`
	PlaceID      string         `refType:"places"            yaml:"place,omitempty"`
	Date         DateString     `yaml:"date,omitempty"` // Date in GLX format: "1850", "ABT 1850", "BEF 1920-01-15", "BET 1880 AND 1890"
	Participants []Participant  `yaml:"participants"`
	Properties   map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes        string         `yaml:"notes,omitempty"`
}

// Place represents a geographical location.
type Place struct {
	Name       string         `yaml:"name"`
	ParentID   string         `refType:"places"            yaml:"parent,omitempty"`
	Type       string         `refType:"place_types"       yaml:"type,omitempty"`
	Latitude   *float64       `yaml:"latitude,omitempty"`
	Longitude  *float64       `yaml:"longitude,omitempty"`
	Properties map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties (jurisdiction, place_format, etc.)
	Notes      string         `yaml:"notes,omitempty"`
}

// Source represents a source of information.
type Source struct {
	Title        string         `yaml:"title"`
	Type         string         `refType:"source_types"       yaml:"type,omitempty"`
	Authors      []string       `yaml:"authors,omitempty"`
	Date         DateString     `yaml:"date,omitempty"`
	Description  string         `yaml:"description,omitempty"`
	RepositoryID string         `refType:"repositories"       yaml:"repository,omitempty"`
	Language     string         `yaml:"language,omitempty"`
	Media        []string       `refType:"media"              yaml:"media,omitempty"`
	Properties   map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes        string         `yaml:"notes,omitempty"`
}

// Citation represents a citation of a source.
type Citation struct {
	SourceID     string         `refType:"sources"           yaml:"source"`
	RepositoryID string         `refType:"repositories"      yaml:"repository,omitempty"`
	Media        []string       `refType:"media"             yaml:"media,omitempty"`
	Properties   map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties (locator, text_from_source, source_date)
	Notes        string         `yaml:"notes,omitempty"`
}

// Repository represents a repository where sources are held.
type Repository struct {
	Name       string         `yaml:"name"`
	Type       string         `refType:"repository_types"      yaml:"type,omitempty"`
	Address    string         `yaml:"address,omitempty"`
	City       string         `yaml:"city,omitempty"`
	State      string         `yaml:"state_province,omitempty"`
	PostalCode string         `yaml:"postal_code,omitempty"`
	Country    string         `yaml:"country,omitempty"`
	Website    string         `yaml:"website,omitempty"`
	Properties map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties (phones, emails, fax, access_hours, access_restrictions, holding_types, external_ids)
	Notes      string         `yaml:"notes,omitempty"`
}

// EntityRef is a typed reference to an entity. Exactly one field must be set.
// This allows assertions to reference different entity types unambiguously.
type EntityRef struct {
	Person       string `yaml:"person,omitempty"`
	Event        string `yaml:"event,omitempty"`
	Relationship string `yaml:"relationship,omitempty"`
	Place        string `yaml:"place,omitempty"`
}

// Type returns the entity type being referenced (using EntityType* constants).
// Returns empty string if no field is set.
func (e *EntityRef) Type() string {
	switch {
	case e.Person != "":
		return EntityTypePersons
	case e.Event != "":
		return EntityTypeEvents
	case e.Relationship != "":
		return EntityTypeRelationships
	case e.Place != "":
		return EntityTypePlaces
	default:
		return ""
	}
}

// ID returns the entity ID being referenced.
// Returns empty string if no field is set.
func (e *EntityRef) ID() string {
	switch {
	case e.Person != "":
		return e.Person
	case e.Event != "":
		return e.Event
	case e.Relationship != "":
		return e.Relationship
	case e.Place != "":
		return e.Place
	default:
		return ""
	}
}

// Assertion represents a conclusion made by a researcher.
type Assertion struct {
	Subject     EntityRef    `yaml:"subject"`
	Property    string       `yaml:"property,omitempty"`    // Optional, not present if participant exists
	Value       string       `yaml:"value,omitempty"`       // Not present if participant exists
	Date        string       `yaml:"date,omitempty"`        // For temporal properties
	Participant *Participant `yaml:"participant,omitempty"` // Not present if property/value exists
	Confidence  string       `refType:"confidence_levels"  yaml:"confidence,omitempty"`
	Sources     []string     `refType:"sources"            yaml:"sources,omitempty"`
	Citations   []string     `refType:"citations"          yaml:"citations,omitempty"`
	Media       []string     `refType:"media"              yaml:"media,omitempty"`
	Notes       string       `yaml:"notes,omitempty"`
}

// Media represents a media object, like a photo or document.
type Media struct {
	URI         string         `yaml:"uri"`
	Type        string         `refType:"media_types"        yaml:"type,omitempty"`
	MimeType    string         `yaml:"mime_type,omitempty"`
	Hash        string         `yaml:"hash,omitempty"`
	Title       string         `yaml:"title,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Date        DateString     `yaml:"date,omitempty"`
	Source      string         `refType:"sources"            yaml:"source,omitempty"`
	Properties  map[string]any `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes       string         `yaml:"notes,omitempty"`
}

// ============================================================================
// Vocabulary Types
// ============================================================================
//
// The following types represent standard vocabulary definitions used in GLX
// archives. These vocabularies define controlled vocabularies for entity
// properties like event types, relationship types, participant roles, etc.

// EventType represents a standard event type vocabulary entry.
type EventType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
	GEDCOM      string `yaml:"gedcom,omitempty"`
}

// ParticipantRole represents a standard participant role vocabulary entry.
type ParticipantRole struct {
	Label       string   `yaml:"label"`
	Description string   `yaml:"description,omitempty"`
	AppliesTo   []string `yaml:"applies_to,omitempty"`
}

// ConfidenceLevel represents a standard confidence level vocabulary entry.
type ConfidenceLevel struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
}

// RelationshipType represents a standard relationship type vocabulary entry.
type RelationshipType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	GEDCOM      string `yaml:"gedcom,omitempty"`
}

// PlaceType represents a standard place type vocabulary entry.
type PlaceType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
}

// SourceType represents a standard source type vocabulary entry.
type SourceType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
}

// RepositoryType represents a standard repository type vocabulary entry.
type RepositoryType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
}

// MediaType represents a standard media type vocabulary entry.
type MediaType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	MimeType    string `yaml:"mime_type,omitempty"`
}

// PropertyDefinition defines a property that can be used on entities.
// value_type and reference_type are mutually exclusive.
type PropertyDefinition struct {
	Label         string                      `yaml:"label"`
	Description   string                      `yaml:"description,omitempty"`
	GEDCOM        string                      `yaml:"gedcom,omitempty"`
	ValueType     string                      `yaml:"value_type,omitempty"`     // string, date, integer, boolean
	ReferenceType string                      `yaml:"reference_type,omitempty"` // persons, places, events, relationships, etc.
	Temporal      *bool                       `yaml:"temporal,omitempty"`       // Can this property change over time?
	MultiValue    *bool                       `yaml:"multi_value,omitempty"`    // Can this property have multiple values (stored as array)?
	Fields        map[string]*FieldDefinition `yaml:"fields,omitempty"`         // Optional structured breakdown of the value
}

// FieldDefinition defines a field within a structured property value.
type FieldDefinition struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
}

// TemporalValue represents a single entry in the history of a temporal property.
// It is used when a temporal property is represented as a list.
type TemporalValue struct {
	Value any    `yaml:"value"`
	Date  string `yaml:"date,omitempty"` // FamilySearch normalized date string
}

// Merge combines another GLXFile into this one, returning duplicate IDs as errors.
// Duplicates are fatal errors for both entities and vocabularies.
func (g *GLXFile) Merge(other *GLXFile) []string {
	duplicates := make([]string, 0, 10)

	// Merge entities (fail on duplicates)
	duplicates = append(duplicates, mergeMap("persons", g.Persons, other.Persons)...)
	duplicates = append(duplicates, mergeMap("relationships", g.Relationships, other.Relationships)...)
	duplicates = append(duplicates, mergeMap("events", g.Events, other.Events)...)
	duplicates = append(duplicates, mergeMap("places", g.Places, other.Places)...)
	duplicates = append(duplicates, mergeMap("sources", g.Sources, other.Sources)...)
	duplicates = append(duplicates, mergeMap("citations", g.Citations, other.Citations)...)
	duplicates = append(duplicates, mergeMap("repositories", g.Repositories, other.Repositories)...)
	duplicates = append(duplicates, mergeMap("assertions", g.Assertions, other.Assertions)...)
	duplicates = append(duplicates, mergeMap("media", g.Media, other.Media)...)

	// Merge vocabularies (ALSO fail on duplicates - treat same as entities)
	duplicates = append(duplicates, mergeMap("event_types", g.EventTypes, other.EventTypes)...)
	duplicates = append(duplicates, mergeMap("relationship_types", g.RelationshipTypes, other.RelationshipTypes)...)
	duplicates = append(duplicates, mergeMap("place_types", g.PlaceTypes, other.PlaceTypes)...)
	duplicates = append(duplicates, mergeMap("source_types", g.SourceTypes, other.SourceTypes)...)
	duplicates = append(duplicates, mergeMap("repository_types", g.RepositoryTypes, other.RepositoryTypes)...)
	duplicates = append(duplicates, mergeMap("media_types", g.MediaTypes, other.MediaTypes)...)
	duplicates = append(duplicates, mergeMap("participant_roles", g.ParticipantRoles, other.ParticipantRoles)...)
	duplicates = append(duplicates, mergeMap("confidence_levels", g.ConfidenceLevels, other.ConfidenceLevels)...)

	// Merge property vocabularies
	duplicates = append(duplicates, mergeMap("person_properties", g.PersonProperties, other.PersonProperties)...)
	duplicates = append(duplicates, mergeMap("event_properties", g.EventProperties, other.EventProperties)...)
	duplicates = append(duplicates, mergeMap("relationship_properties", g.RelationshipProperties, other.RelationshipProperties)...)
	duplicates = append(duplicates, mergeMap("place_properties", g.PlaceProperties, other.PlaceProperties)...)
	duplicates = append(duplicates, mergeMap("media_properties", g.MediaProperties, other.MediaProperties)...)
	duplicates = append(duplicates, mergeMap("repository_properties", g.RepositoryProperties, other.RepositoryProperties)...)
	duplicates = append(duplicates, mergeMap("citation_properties", g.CitationProperties, other.CitationProperties)...)
	duplicates = append(duplicates, mergeMap("source_properties", g.SourceProperties, other.SourceProperties)...)

	return duplicates
}

// mergeMap is used for BOTH entities and vocabularies - duplicates are always errors
func mergeMap[T any](mapType string, dest, src map[string]*T) []string {
	var duplicates []string
	if dest == nil {
		return duplicates
	}
	if src == nil {
		return duplicates
	}
	for k, v := range src {
		if _, exists := dest[k]; exists {
			duplicates = append(duplicates, fmt.Sprintf("duplicate %s ID: %s", mapType, k))
		} else {
			dest[k] = v
		}
	}

	return duplicates
}
