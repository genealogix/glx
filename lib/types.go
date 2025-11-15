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

package lib

import "fmt"

// GLXFile represents the top-level structure of a .glx file, which can
// contain maps of different entity types and vocabulary definitions.
type GLXFile struct {
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
	QualityRatings    map[string]*QualityRating    `yaml:"quality_ratings,omitempty"`

	// Property vocabularies
	PersonProperties       map[string]*PropertyDefinition `yaml:"person_properties,omitempty"`
	EventProperties        map[string]*PropertyDefinition `yaml:"event_properties,omitempty"`
	RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties,omitempty"`
	PlaceProperties        map[string]*PropertyDefinition `yaml:"place_properties,omitempty"`
}

// ============================================================================
// Entity Types
// ============================================================================
//
// The following types represent the core genealogical entities in a GLX archive.

// Person represents an individual in the family archive.
type Person struct {
	Properties map[string]interface{} `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Notes      string                 `yaml:"notes,omitempty"`
	Tags       []string               `yaml:"tags,omitempty"`
}

// Relationship represents a relationship between two or more people.
type Relationship struct {
	Version      string                    `yaml:"version"`
	Type         string                    `yaml:"type" refType:"relationship_types"`
	Persons      []string                  `yaml:"persons,omitempty" refType:"persons"`
	Participants []RelationshipParticipant `yaml:"participants,omitempty"`
	StartEvent   string                    `yaml:"start_event,omitempty" refType:"events"`
	EndEvent     string                    `yaml:"end_event,omitempty" refType:"events"`
	Properties   map[string]interface{}    `yaml:"properties,omitempty"` // Vocabulary-defined properties
	Description  string                    `yaml:"description,omitempty"`
	Notes        string                    `yaml:"notes,omitempty"`
	Tags         []string                  `yaml:"tags,omitempty"`
}

// RelationshipParticipant defines a person's role in a relationship.
type RelationshipParticipant struct {
	Person string `yaml:"person" refType:"persons"`
	Role   string `yaml:"role,omitempty" refType:"participant_roles"`
}

// Event represents a genealogical event.
type Event struct {
	Type         string                 `yaml:"type" refType:"event_types"`
	PlaceID      string                 `yaml:"place,omitempty" refType:"places"`
	Value        string                 `yaml:"value,omitempty"`
	Date         *EventDate             `yaml:"date,omitempty"`
	Participants []EventParticipant     `yaml:"participants,omitempty"`
	Properties   map[string]interface{} `yaml:"properties,omitempty"`  // Vocabulary-defined properties
	Description  string                 `yaml:"description,omitempty"`
	Notes        string                 `yaml:"notes,omitempty"`
	Tags         []string               `yaml:"tags,omitempty"`
}

// EventDate can be a simple string or a date range.
type EventDate struct {
	Value      string `yaml:"value,omitempty"`
	RangeStart string `yaml:"range_start,omitempty"`
	RangeEnd   string `yaml:"range_end,omitempty"`
}

// EventParticipant defines a person's role in an event.
type EventParticipant struct {
	PersonID string `yaml:"person,omitempty" refType:"persons"`
	Role     string `yaml:"role,omitempty" refType:"participant_roles"`
	Notes    string `yaml:"notes,omitempty"`
}

// Place represents a geographical location.
type Place struct {
	Name             string                 `yaml:"name"`
	ParentID         string                 `yaml:"parent,omitempty" refType:"places"`
	Type             string                 `yaml:"type,omitempty" refType:"place_types"`
	AlternativeNames []AlternativeName      `yaml:"alternative_names,omitempty"`
	Latitude         *float64               `yaml:"latitude,omitempty"`
	Longitude        *float64               `yaml:"longitude,omitempty"`
	Properties       map[string]interface{} `yaml:"properties,omitempty"`  // Vocabulary-defined properties
	Notes            string                 `yaml:"notes,omitempty"`
	Tags             []string               `yaml:"tags,omitempty"`
}

// AlternativeName is a historical or alternative name for a place.
type AlternativeName struct {
	Name      string     `yaml:"name"`
	Type      string     `yaml:"type,omitempty"`
	Language  string     `yaml:"language,omitempty"`
	DateRange *DateRange `yaml:"date_range,omitempty"`
}

// DateRange represents a period of time.
type DateRange struct {
	Start string `yaml:"start,omitempty"`
	End   string `yaml:"end,omitempty"`
}

// Source represents a source of information.
type Source struct {
	Version         string   `yaml:"version"`
	Title           string   `yaml:"title"`
	Type            string   `yaml:"type,omitempty" refType:"source_types"`
	Authors         []string `yaml:"authors,omitempty"`
	Date            string   `yaml:"date,omitempty"`
	Description     string   `yaml:"description,omitempty"`
	RepositoryID    string   `yaml:"repository,omitempty" refType:"repositories"`
	PublicationInfo string   `yaml:"publication_info,omitempty"`
	Notes           string   `yaml:"notes,omitempty"`
	Media           []string `yaml:"media,omitempty" refType:"media"`
	Tags            []string `yaml:"tags,omitempty"`
}

// Citation represents a citation of a source.
type Citation struct {
	Version        string   `yaml:"version"`
	SourceID       string   `yaml:"source,omitempty" refType:"sources"`
	Page           string   `yaml:"page,omitempty"`
	TextFromSource string   `yaml:"text_from_source,omitempty"`
	Transcription  string   `yaml:"transcription,omitempty"`
	Quality        *int     `yaml:"quality,omitempty" refType:"quality_ratings"`
	Locator        *Locator `yaml:"locator,omitempty"`
	RepositoryID   string   `yaml:"repository,omitempty" refType:"repositories"`
	Media          []string `yaml:"media,omitempty" refType:"media"`
	Notes          string   `yaml:"notes,omitempty"`
	Tags           []string `yaml:"tags,omitempty"`
}

// Locator provides structured information for finding a source.
type Locator struct {
	FilmNumber         string `yaml:"film_number,omitempty"`
	ItemNumber         string `yaml:"item_number,omitempty"`
	ImageNumber        string `yaml:"image_number,omitempty"`
	URL                string `yaml:"url,omitempty"`
	RecordID           string `yaml:"record_id,omitempty"`
	DatabaseCollection string `yaml:"database_collection,omitempty"`
}

// Repository represents a repository where sources are held.
type Repository struct {
	Version    string   `yaml:"version"`
	Name       string   `yaml:"name"`
	Type       string   `yaml:"type,omitempty" refType:"repository_types"`
	Address    string   `yaml:"address,omitempty"`
	City       string   `yaml:"city,omitempty"`
	State      string   `yaml:"state_province,omitempty"`
	PostalCode string   `yaml:"postal_code,omitempty"`
	Country    string   `yaml:"country,omitempty"`
	Phone      string   `yaml:"phone,omitempty"`
	Email      string   `yaml:"email,omitempty"`
	Website    string   `yaml:"website,omitempty"`
	Notes      string   `yaml:"notes,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

// Assertion represents a conclusion made by a researcher.
type Assertion struct {
	Subject     string                `yaml:"subject" refType:"persons,events,relationships,places"`
	Claim       string                `yaml:"claim,omitempty"`  // Optional, not present if participant exists
	Value       string                `yaml:"value,omitempty"`  // Not present if participant exists
	Participant *AssertionParticipant `yaml:"participant,omitempty"`  // Not present if value exists
	Confidence  string                `yaml:"confidence,omitempty" refType:"confidence_levels"`
	Sources     []string              `yaml:"sources,omitempty" refType:"sources"`
	Citations   []string              `yaml:"citations,omitempty" refType:"citations"`
	Notes       string                `yaml:"notes,omitempty"`
	Tags        []string              `yaml:"tags,omitempty"`
}

// AssertionParticipant represents a participant in an assertion (used for participant-based claims).
type AssertionParticipant struct {
	Person string `yaml:"person" refType:"persons"`
	Role   string `yaml:"role,omitempty" refType:"participant_roles"`
	Notes  string `yaml:"notes,omitempty"`
}

// Media represents a media object, like a photo or document.
type Media struct {
	Version  string `yaml:"version"`
	URI      string `yaml:"uri"`
	MimeType string `yaml:"mime_type,omitempty"`
	Hash     string `yaml:"hash,omitempty"`
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
	Custom      *bool  `yaml:"custom,omitempty"`
}

// ParticipantRole represents a standard participant role vocabulary entry.
type ParticipantRole struct {
	Label       string   `yaml:"label"`
	Description string   `yaml:"description,omitempty"`
	AppliesTo   []string `yaml:"applies_to,omitempty"`
	Custom      *bool    `yaml:"custom,omitempty"`
}

// ConfidenceLevel represents a standard confidence level vocabulary entry.
type ConfidenceLevel struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// RelationshipType represents a standard relationship type vocabulary entry.
type RelationshipType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	GEDCOM      string `yaml:"gedcom,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// PlaceType represents a standard place type vocabulary entry.
type PlaceType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Category    string `yaml:"category,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// SourceType represents a standard source type vocabulary entry.
type SourceType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// RepositoryType represents a standard repository type vocabulary entry.
type RepositoryType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// MediaType represents a standard media type vocabulary entry.
type MediaType struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description,omitempty"`
	MimeType    string `yaml:"mime_type,omitempty"`
	Custom      *bool  `yaml:"custom,omitempty"`
}

// QualityRating represents a standard quality rating vocabulary entry.
// Note: The key in the map is a string representation of the integer rating (0-3).
type QualityRating struct {
	Label       string   `yaml:"label"`
	Description string   `yaml:"description,omitempty"`
	Examples    []string `yaml:"examples,omitempty"`
	Custom      *bool    `yaml:"custom,omitempty"`
}

// PropertyDefinition defines a property that can be used on entities.
// value_type and reference_type are mutually exclusive.
type PropertyDefinition struct {
	Label         string `yaml:"label"`
	Description   string `yaml:"description,omitempty"`
	ValueType     string `yaml:"value_type,omitempty"`      // string, date, integer, boolean
	ReferenceType string `yaml:"reference_type,omitempty"`  // persons, places, events, relationships, etc.
	Temporal      *bool  `yaml:"temporal,omitempty"`        // Can this property change over time?
	Custom        *bool  `yaml:"custom,omitempty"`
}

// TemporalValue represents a single entry in the history of a temporal property.
type TemporalValue struct {
	Value interface{} `yaml:"value"`
	Date  string      `yaml:"date,omitempty"`  // FamilySearch normalized date string
}

// Merge combines another GLXFile into this one, returning duplicate IDs as errors.
// Duplicates are fatal errors for both entities and vocabularies.
func (g *GLXFile) Merge(other *GLXFile) []string {
	var duplicates []string

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
	duplicates = append(duplicates, mergeMap("quality_ratings", g.QualityRatings, other.QualityRatings)...)

	// Merge property vocabularies
	duplicates = append(duplicates, mergeMap("person_properties", g.PersonProperties, other.PersonProperties)...)
	duplicates = append(duplicates, mergeMap("event_properties", g.EventProperties, other.EventProperties)...)
	duplicates = append(duplicates, mergeMap("relationship_properties", g.RelationshipProperties, other.RelationshipProperties)...)
	duplicates = append(duplicates, mergeMap("place_properties", g.PlaceProperties, other.PlaceProperties)...)

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
