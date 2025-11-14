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

import "time"

// GLXFile represents the top-level structure of a .glx file, which can
// contain maps of different entity types.
type GLXFile struct {
	Persons       map[string]*Person       `yaml:"persons,omitempty"`
	Relationships map[string]*Relationship `yaml:"relationships,omitempty"`
	Events        map[string]*Event        `yaml:"events,omitempty"`
	Places        map[string]*Place        `yaml:"places,omitempty"`
	Sources       map[string]*Source       `yaml:"sources,omitempty"`
	Citations     map[string]*Citation     `yaml:"citations,omitempty"`
	Repositories  map[string]*Repository   `yaml:"repositories,omitempty"`
	Assertions    map[string]*Assertion    `yaml:"assertions,omitempty"`
	Media         map[string]*Media        `yaml:"media,omitempty"`
}

// Person represents an individual in the family archive.
type Person struct {
	Version           string             `yaml:"version"`
	ConcludedIdentity *ConcludedIdentity `yaml:"concluded_identity,omitempty"`
	Relationships     []string           `yaml:"relationships,omitempty"`
	CreatedAt         *time.Time         `yaml:"created_at,omitempty"`
	CreatedBy         string             `yaml:"created_by,omitempty"`
	ModifiedAt        *time.Time         `yaml:"modified_at,omitempty"`
	ModifiedBy        string             `yaml:"modified_by,omitempty"`
	Notes             string             `yaml:"notes,omitempty"`
	Tags              []string           `yaml:"tags,omitempty"`
}

// ConcludedIdentity represents a researcher's conclusion about an individual's identity.
type ConcludedIdentity struct {
	PrimaryName string `yaml:"primary_name,omitempty"`
	Gender      string `yaml:"gender,omitempty"`
	Living      *bool  `yaml:"living,omitempty"`
}

// Relationship represents a relationship between two or more people.
type Relationship struct {
	Version      string                    `yaml:"version"`
	Type         string                    `yaml:"type"`
	Persons      []string                  `yaml:"persons,omitempty"`
	Participants []RelationshipParticipant `yaml:"participants,omitempty"`
	StartEvent   string                    `yaml:"start_event,omitempty"`
	EndEvent     string                    `yaml:"end_event,omitempty"`
	Description  string                    `yaml:"description,omitempty"`
	Notes        string                    `yaml:"notes,omitempty"`
	Assertions   []string                  `yaml:"assertions,omitempty"`
	CreatedAt    *time.Time                `yaml:"created_at,omitempty"`
	CreatedBy    string                    `yaml:"created_by,omitempty"`
	ModifiedAt   *time.Time                `yaml:"modified_at,omitempty"`
	ModifiedBy   string                    `yaml:"modified_by,omitempty"`
	Tags         []string                  `yaml:"tags,omitempty"`
}

// RelationshipParticipant defines a person's role in a relationship.
type RelationshipParticipant struct {
	Person string `yaml:"person"`
	Role   string `yaml:"role,omitempty"`
}

// Event represents a genealogical event.
type Event struct {
	Version      string             `yaml:"version"`
	Type         string             `yaml:"type"`
	PlaceID      string             `yaml:"place_id,omitempty"`
	Place        string             `yaml:"place,omitempty"`
	Value        string             `yaml:"value,omitempty"`
	Date         *EventDate         `yaml:"date,omitempty"`
	Participants []EventParticipant `yaml:"participants,omitempty"`
	Description  string             `yaml:"description,omitempty"`
	CreatedAt    *time.Time         `yaml:"created_at,omitempty"`
	CreatedBy    string             `yaml:"created_by,omitempty"`
	ModifiedAt   *time.Time         `yaml:"modified_at,omitempty"`
	ModifiedBy   string             `yaml:"modified_by,omitempty"`
	Notes        string             `yaml:"notes,omitempty"`
	Tags         []string           `yaml:"tags,omitempty"`
}

// EventDate can be a simple string or a date range.
type EventDate struct {
	Value      string `yaml:"value,omitempty"`
	RangeStart string `yaml:"range_start,omitempty"`
	RangeEnd   string `yaml:"range_end,omitempty"`
}

// EventParticipant defines a person's role in an event.
type EventParticipant struct {
	PersonID string `yaml:"person_id,omitempty"`
	Person   string `yaml:"person,omitempty"`
	Role     string `yaml:"role,omitempty"`
	Notes    string `yaml:"notes,omitempty"`
}

// Place represents a geographical location.
type Place struct {
	Version          string            `yaml:"version"`
	Name             string            `yaml:"name"`
	ParentID         string            `yaml:"parent_id,omitempty"`
	Type             string            `yaml:"type,omitempty"`
	AlternativeNames []AlternativeName `yaml:"alternative_names,omitempty"`
	Latitude         *float64          `yaml:"latitude,omitempty"`
	Longitude        *float64          `yaml:"longitude,omitempty"`
	CreatedAt        *time.Time        `yaml:"created_at,omitempty"`
	CreatedBy        string            `yaml:"created_by,omitempty"`
	ModifiedAt       *time.Time        `yaml:"modified_at,omitempty"`
	ModifiedBy       string            `yaml:"modified_by,omitempty"`
	Notes            string            `yaml:"notes,omitempty"`
	Tags             []string          `yaml:"tags,omitempty"`
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
	Version         string     `yaml:"version"`
	Title           string     `yaml:"title"`
	Type            string     `yaml:"type,omitempty"`
	Authors         []string   `yaml:"authors,omitempty"`
	Date            string     `yaml:"date,omitempty"`
	Description     string     `yaml:"description,omitempty"`
	RepositoryID    string     `yaml:"repository_id,omitempty"`
	PublicationInfo string     `yaml:"publication_info,omitempty"`
	Notes           string     `yaml:"notes,omitempty"`
	Media           []string   `yaml:"media,omitempty"`
	CreatedAt       *time.Time `yaml:"created_at,omitempty"`
	CreatedBy       string     `yaml:"created_by,omitempty"`
	ModifiedAt      *time.Time `yaml:"modified_at,omitempty"`
	ModifiedBy      string     `yaml:"modified_by,omitempty"`
	Tags            []string   `yaml:"tags,omitempty"`
}

// Citation represents a citation of a source.
type Citation struct {
	Version        string     `yaml:"version"`
	SourceID       string     `yaml:"source_id,omitempty"`
	Source         string     `yaml:"source,omitempty"`
	Page           string     `yaml:"page,omitempty"`
	TextFromSource string     `yaml:"text_from_source,omitempty"`
	Transcription  string     `yaml:"transcription,omitempty"`
	Quality        *int       `yaml:"quality,omitempty"`
	Locator        *Locator   `yaml:"locator,omitempty"`
	RepositoryID   string     `yaml:"repository_id,omitempty"`
	Repository     string     `yaml:"repository,omitempty"`
	CreatedAt      *time.Time `yaml:"created_at,omitempty"`
	CreatedBy      string     `yaml:"created_by,omitempty"`
	ModifiedAt     *time.Time `yaml:"modified_at,omitempty"`
	ModifiedBy     string     `yaml:"modified_by,omitempty"`
	Notes          string     `yaml:"notes,omitempty"`
	Tags           []string   `yaml:"tags,omitempty"`
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
	Version    string     `yaml:"version"`
	Name       string     `yaml:"name"`
	Type       string     `yaml:"type,omitempty"`
	Address    string     `yaml:"address,omitempty"`
	City       string     `yaml:"city,omitempty"`
	State      string     `yaml:"state_province,omitempty"`
	PostalCode string     `yaml:"postal_code,omitempty"`
	Country    string     `yaml:"country,omitempty"`
	Phone      string     `yaml:"phone,omitempty"`
	Email      string     `yaml:"email,omitempty"`
	Website    string     `yaml:"website,omitempty"`
	CreatedAt  *time.Time `yaml:"created_at,omitempty"`
	CreatedBy  string     `yaml:"created_by,omitempty"`
	ModifiedAt *time.Time `yaml:"modified_at,omitempty"`
	ModifiedBy string     `yaml:"modified_by,omitempty"`
	Notes      string     `yaml:"notes,omitempty"`
	Tags       []string   `yaml:"tags,omitempty"`
}

// Assertion represents a conclusion made by a researcher.
type Assertion struct {
	Version    string     `yaml:"version"`
	Subject    string     `yaml:"subject"`
	Claim      string     `yaml:"claim"`
	Value      string     `yaml:"value,omitempty"`
	Confidence string     `yaml:"confidence,omitempty"`
	Sources    []string   `yaml:"sources,omitempty"`
	Citations  []string   `yaml:"citations,omitempty"`
	CreatedAt  *time.Time `yaml:"created_at,omitempty"`
	CreatedBy  string     `yaml:"created_by,omitempty"`
	ModifiedAt *time.Time `yaml:"modified_at,omitempty"`
	ModifiedBy string     `yaml:"modified_by,omitempty"`
	Notes      string     `yaml:"notes,omitempty"`
	Tags       []string   `yaml:"tags,omitempty"`
}

// Media represents a media object, like a photo or document.
type Media struct {
	Version    string     `yaml:"version"`
	URI        string     `yaml:"uri"`
	MimeType   string     `yaml:"mime_type,omitempty"`
	Hash       string     `yaml:"hash,omitempty"`
	CreatedAt  *time.Time `yaml:"created_at,omitempty"`
	CreatedBy  string     `yaml:"created_by,omitempty"`
	ModifiedAt *time.Time `yaml:"modified_at,omitempty"`
	ModifiedBy string     `yaml:"modified_by,omitempty"`
}
