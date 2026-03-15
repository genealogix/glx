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
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Serializer defines the interface for GLX archive serialization.
// All methods work with bytes only - no filesystem I/O.
type Serializer interface {
	// SerializeSingleFileBytes serializes a GLX archive to YAML bytes (single-file format)
	SerializeSingleFileBytes(glx *GLXFile) ([]byte, error)

	// DeserializeSingleFileBytes loads a GLX archive from YAML bytes (single-file format)
	DeserializeSingleFileBytes(data []byte) (*GLXFile, error)

	// SerializeMultiFileToMap serializes a GLX archive to a map of relative paths to file contents
	// Map keys are relative paths like "persons/person-abc123.glx", "vocabularies/event-types.glx"
	SerializeMultiFileToMap(glx *GLXFile) (map[string][]byte, error)

	// DeserializeMultiFileFromMap loads a GLX archive from a map of relative paths to file contents.
	// Returns the loaded archive, a list of duplicate entity IDs, and any error.
	DeserializeMultiFileFromMap(files map[string][]byte) (*GLXFile, []string, error)
}

// SerializerOptions configures the serializer behavior.
type SerializerOptions struct {
	// Validate determines whether to validate the archive before serialization
	Validate bool

	// Pretty determines whether to use pretty-print formatting (indentation, etc.)
	Pretty bool

	// Indent specifies the indentation string for YAML output (default: "  ")
	Indent string
}

// DefaultSerializerOptions returns the default serializer options.
func DefaultSerializerOptions() *SerializerOptions {
	return &SerializerOptions{
		Validate: true,
		Pretty:   true,
		Indent:   "  ",
	}
}

// DefaultSerializer is the standard GLX serializer implementation.
type DefaultSerializer struct {
	Options *SerializerOptions
}

// NewSerializer creates a new serializer with the given options.
// If options is nil, uses DefaultSerializerOptions().
func NewSerializer(options *SerializerOptions) *DefaultSerializer {
	if options == nil {
		options = DefaultSerializerOptions()
	}

	return &DefaultSerializer{
		Options: options,
	}
}

// SerializeSingleFileBytes serializes a GLX archive to YAML bytes (single-file format).
func (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error) {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Guard against nil input when validation is disabled.
	if glx == nil {
		return nil, ErrGLXFileNil
	}

	// Normalize empty metadata to nil so omitempty suppresses it.
	// Save and restore the original value to avoid mutating the caller's data.
	origMetadata := glx.ImportMetadata
	if glx.ImportMetadata != nil && !glx.ImportMetadata.hasContent() {
		glx.ImportMetadata = nil
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(glx)

	glx.ImportMetadata = origMetadata

	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return yamlBytes, nil
}

// SerializeMultiFileToMap serializes a GLX archive to a map of relative paths to file contents.
// Map keys are relative paths like "persons/person-abc123.glx", "vocabularies/event-types.glx".
func (s *DefaultSerializer) SerializeMultiFileToMap(glx *GLXFile) (map[string][]byte, error) {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Guard against nil input when validation is disabled.
	if glx == nil {
		return nil, ErrGLXFileNil
	}

	files := make(map[string][]byte)

	// Add standard vocabularies
	for filename, content := range StandardVocabularies() {
		vocabPath := filepath.Join("vocabularies", filename)
		files[vocabPath] = content
	}

	// Serialize metadata if present and non-empty
	if glx.ImportMetadata != nil && glx.ImportMetadata.hasContent() {
		metaWrapper := map[string]*Metadata{"metadata": glx.ImportMetadata}
		yamlBytes, err := yaml.Marshal(metaWrapper)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		files["metadata.glx"] = yamlBytes
	}

	// Serialize entities by type
	if err := s.serializeEntitiesToMap(glx.Persons, "persons", "person", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Events, "events", "event", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Relationships, "relationships", "relationship", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Places, "places", "place", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Sources, "sources", "source", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Citations, "citations", "citation", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Repositories, "repositories", "repository", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Media, "media", "media", files); err != nil {
		return nil, err
	}
	if err := s.serializeEntitiesToMap(glx.Assertions, "assertions", "assertion", files); err != nil {
		return nil, err
	}

	return files, nil
}

// serializeEntitiesToMap serializes entities to the files map with random filenames.
// Each entity file uses the standard GLX structure: {collectionKey: {entityID: entity}}
func (s *DefaultSerializer) serializeEntitiesToMap(entities any, dirName, entityType string, files map[string][]byte) error {
	// Type switch to handle different entity map types
	switch typedEntities := entities.(type) {
	case map[string]*Person:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Event:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Relationship:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Place:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Source:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Citation:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Repository:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Media:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	case map[string]*Assertion:
		return serializeEntitiesWrapped(typedEntities, dirName, entityType, files)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedEntityType, entities)
	}
}

// serializeEntitiesWrapped serializes each entity as a standard GLX file:
// {collectionKey: {entityID: entity}}. Uses random filenames with collision detection.
func serializeEntitiesWrapped[T any](entities map[string]T, dirName, entityType string, files map[string][]byte) error {
	usedFilenames := make(map[string]bool)

	for entityID, entity := range entities {
		// Generate unique random filename
		filename, err := GenerateUniqueFilename(entityType, usedFilenames, 10)
		if err != nil {
			return fmt.Errorf("failed to generate filename for %s: %w", entityID, err)
		}

		// Wrap as standard GLX structure: {collectionKey: {entityID: entity}}
		wrapper := map[string]map[string]T{
			dirName: {entityID: entity},
		}

		// Marshal to YAML
		yamlBytes, err := yaml.Marshal(wrapper)
		if err != nil {
			return fmt.Errorf("failed to marshal %s %s: %w", entityType, entityID, err)
		}

		// Add to files map
		filePath := filepath.Join(dirName, filename)
		files[filePath] = yamlBytes
	}

	return nil
}

// DeserializeSingleFileBytes loads a GLX archive from YAML bytes (single-file format).
func (s *DefaultSerializer) DeserializeSingleFileBytes(data []byte) (*GLXFile, error) {
	var glx GLXFile

	// Unmarshal YAML
	if err := yaml.Unmarshal(data, &glx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(&glx); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return &glx, nil
}

// DeserializeMultiFileFromMap loads a GLX archive from a map of relative paths to file contents.
// Returns the loaded archive, a list of duplicate entity IDs, and any error.
func (s *DefaultSerializer) DeserializeMultiFileFromMap(files map[string][]byte) (*GLXFile, []string, error) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),

		EventTypes:        make(map[string]*EventType),
		ParticipantRoles:  make(map[string]*ParticipantRole),
		ConfidenceLevels:  make(map[string]*ConfidenceLevel),
		RelationshipTypes: make(map[string]*RelationshipType),
		PlaceTypes:        make(map[string]*PlaceType),
		SourceTypes:       make(map[string]*SourceType),
		RepositoryTypes:   make(map[string]*RepositoryType),
		MediaTypes:        make(map[string]*MediaType),
		GenderTypes:       make(map[string]*GenderType),

		PersonProperties:       make(map[string]*PropertyDefinition),
		EventProperties:        make(map[string]*PropertyDefinition),
		RelationshipProperties: make(map[string]*PropertyDefinition),
		PlaceProperties:        make(map[string]*PropertyDefinition),
		MediaProperties:        make(map[string]*PropertyDefinition),
		RepositoryProperties:   make(map[string]*PropertyDefinition),
		CitationProperties:     make(map[string]*PropertyDefinition),
		SourceProperties:       make(map[string]*PropertyDefinition),
	}

	var allDuplicates []string

	// Each file is a GLXFile fragment — the YAML top-level keys (persons:,
	// events:, event_types:, etc.) determine what entities it contains,
	// regardless of which directory the file lives in.
	for path, data := range files {
		ext := filepath.Ext(path)
		if ext != FileExtGLX && ext != ".yaml" && ext != ".yml" {
			continue
		}
		var partial GLXFile
		if err := yaml.Unmarshal(data, &partial); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}
		duplicates := glx.Merge(&partial)
		allDuplicates = append(allDuplicates, duplicates...)
	}

	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return glx, allDuplicates, nil
}

// validateGLXFile validates a GLX archive using the built-in validation system.
// Returns StructuredValidationError if validation fails with hard errors.
func validateGLXFile(glx *GLXFile) error {
	if glx == nil {
		return ErrGLXFileNil
	}

	// Initialize maps if nil (prevents validation from failing on nil maps)
	if glx.Persons == nil {
		glx.Persons = make(map[string]*Person)
	}
	if glx.Events == nil {
		glx.Events = make(map[string]*Event)
	}
	if glx.Relationships == nil {
		glx.Relationships = make(map[string]*Relationship)
	}
	if glx.Places == nil {
		glx.Places = make(map[string]*Place)
	}
	if glx.Sources == nil {
		glx.Sources = make(map[string]*Source)
	}
	if glx.Citations == nil {
		glx.Citations = make(map[string]*Citation)
	}
	if glx.Repositories == nil {
		glx.Repositories = make(map[string]*Repository)
	}
	if glx.Media == nil {
		glx.Media = make(map[string]*Media)
	}
	if glx.Assertions == nil {
		glx.Assertions = make(map[string]*Assertion)
	}

	// Run full validation
	result := glx.Validate()

	// Check for hard errors
	if len(result.Errors) > 0 {
		// Return structured validation error with all errors
		return &StructuredValidationError{
			Errors: result.Errors,
		}
	}

	return nil
}
