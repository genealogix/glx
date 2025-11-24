package lib

import (
	"fmt"
	"path/filepath"
	"strings"

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

	// DeserializeMultiFileFromMap loads a GLX archive from a map of relative paths to file contents
	DeserializeMultiFileFromMap(files map[string][]byte) (*GLXFile, error)
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

// EntityWithID wraps an entity with its ID for multi-file serialization.
// The _id field is embedded in the YAML to preserve entity IDs when using random filenames.
type EntityWithID[T any] struct {
	ID     string `yaml:"_id"`
	Entity T      `yaml:",inline"`
}

// SerializeSingleFileBytes serializes a GLX archive to YAML bytes (single-file format).
func (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error) {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(glx)
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

	files := make(map[string][]byte)

	// Add standard vocabularies
	for filename, content := range StandardVocabularies() {
		vocabPath := filepath.Join("vocabularies", filename)
		files[vocabPath] = content
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
func (s *DefaultSerializer) serializeEntitiesToMap(entities any, dirName, entityType string, files map[string][]byte) error {
	// Type switch to handle different entity map types
	switch typedEntities := entities.(type) {
	case map[string]*Person:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Event:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Relationship:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Place:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Source:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Citation:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Repository:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Media:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	case map[string]*Assertion:
		return serializeEntitiesWithID(typedEntities, dirName, entityType, files)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedEntityType, entities)
	}
}

// serializeEntitiesWithID serializes entities with embedded ID field to the files map.
// Uses random filenames with collision detection.
func serializeEntitiesWithID[T any](entities map[string]T, dirName, entityType string, files map[string][]byte) error {
	usedFilenames := make(map[string]bool)

	for entityID, entity := range entities {
		// Generate unique random filename
		filename, err := GenerateUniqueFilename(entityType, usedFilenames, 10)
		if err != nil {
			return fmt.Errorf("failed to generate filename for %s: %w", entityID, err)
		}

		// Wrap entity with ID
		wrapper := EntityWithID[T]{
			ID:     entityID,
			Entity: entity,
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
func (s *DefaultSerializer) DeserializeMultiFileFromMap(files map[string][]byte) (*GLXFile, error) {
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
	}

	// Load vocabularies from map
	vocabFiles := make(map[string][]byte)
	for path, content := range files {
		if strings.HasPrefix(path, "vocabularies/") || strings.HasPrefix(path, "vocabularies\\") {
			// Extract filename from path
			filename := filepath.Base(path)
			vocabFiles[filename] = content
		}
	}
	if len(vocabFiles) > 0 {
		if err := LoadVocabulariesFromMap(vocabFiles, glx); err != nil {
			return nil, fmt.Errorf("failed to load vocabularies: %w", err)
		}
	}

	// Load entities from map
	if err := deserializeEntitiesFromMap(files, "persons", glx.Persons); err != nil {
		return nil, fmt.Errorf("failed to load persons: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "events", glx.Events); err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "relationships", glx.Relationships); err != nil {
		return nil, fmt.Errorf("failed to load relationships: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "places", glx.Places); err != nil {
		return nil, fmt.Errorf("failed to load places: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "sources", glx.Sources); err != nil {
		return nil, fmt.Errorf("failed to load sources: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "citations", glx.Citations); err != nil {
		return nil, fmt.Errorf("failed to load citations: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "repositories", glx.Repositories); err != nil {
		return nil, fmt.Errorf("failed to load repositories: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "media", glx.Media); err != nil {
		return nil, fmt.Errorf("failed to load media: %w", err)
	}
	if err := deserializeEntitiesFromMap(files, "assertions", glx.Assertions); err != nil {
		return nil, fmt.Errorf("failed to load assertions: %w", err)
	}

	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return glx, nil
}

// deserializeEntitiesFromMap loads entities with embedded _id field from the files map.
func deserializeEntitiesFromMap[T any](files map[string][]byte, dirName string, entities map[string]T) error {
	for path, data := range files {
		// Check if this file belongs to the specified directory
		dir := filepath.Dir(path)
		if dir != dirName && dir != strings.ReplaceAll(dirName, "/", "\\") {
			continue
		}

		// Check file extension
		if filepath.Ext(path) != FileExtGLX {
			continue
		}

		// Unmarshal entity
		var wrapper EntityWithID[T]
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		entities[wrapper.ID] = wrapper.Entity
	}

	return nil
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
