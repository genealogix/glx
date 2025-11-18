package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Serializer defines the interface for GLX archive serialization.
type Serializer interface {
	// SerializeSingleFile serializes a GLX archive to a single YAML file
	SerializeSingleFile(glx *GLXFile, outputPath string) error

	// SerializeSingleFileBytes serializes a GLX archive to YAML bytes (single-file format)
	SerializeSingleFileBytes(glx *GLXFile) ([]byte, error)

	// SerializeMultiFile serializes a GLX archive to a multi-file directory structure
	SerializeMultiFile(glx *GLXFile, outputDir string) error

	// LoadSingleFile loads a GLX archive from a single YAML file
	LoadSingleFile(inputPath string) (*GLXFile, error)

	// LoadSingleFileBytes loads a GLX archive from YAML bytes (single-file format)
	LoadSingleFileBytes(data []byte) (*GLXFile, error)

	// LoadMultiFile loads a GLX archive from a multi-file directory structure
	LoadMultiFile(inputDir string) (*GLXFile, error)
}

// SerializerOptions configures the serializer behavior.
type SerializerOptions struct {
	// IncludeVocabularies determines whether to write standard vocabularies to the archive
	IncludeVocabularies bool

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
		IncludeVocabularies: true,
		Validate:            true,
		Pretty:              true,
		Indent:              "  ",
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

// SerializeSingleFile serializes a GLX archive to a single YAML file.
func (s *DefaultSerializer) SerializeSingleFile(glx *GLXFile, outputPath string) error {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Serialize to bytes
	yamlBytes, err := s.SerializeSingleFileBytes(glx)
	if err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
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

// SerializeMultiFile serializes a GLX archive to a multi-file directory structure.
func (s *DefaultSerializer) SerializeMultiFile(glx *GLXFile, outputDir string) error {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write vocabularies if requested
	if s.Options.IncludeVocabularies {
		if err := WriteStandardVocabularies(outputDir); err != nil {
			return fmt.Errorf("failed to write vocabularies: %w", err)
		}
	}

	// Write entities by type
	if err := s.writeEntities(glx.Persons, outputDir, "persons", "person"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Events, outputDir, "events", "event"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Relationships, outputDir, "relationships", "relationship"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Places, outputDir, "places", "place"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Sources, outputDir, "sources", "source"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Citations, outputDir, "citations", "citation"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Repositories, outputDir, "repositories", "repository"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Media, outputDir, "media", "media"); err != nil {
		return err
	}
	if err := s.writeEntities(glx.Assertions, outputDir, "assertions", "assertion"); err != nil {
		return err
	}

	return nil
}

// writeEntities writes a map of entities to individual files with random IDs.
func (s *DefaultSerializer) writeEntities(entities any, outputDir, dirName, entityType string) error {
	// Create entity directory
	entityDir := filepath.Join(outputDir, dirName)
	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", dirName, err)
	}

	// Type switch to handle different entity map types
	switch typedEntities := entities.(type) {
	case map[string]*Person:
		return s.writePersonEntities(typedEntities, entityDir, entityType)
	case map[string]*Event:
		return s.writeEventEntities(typedEntities, entityDir, entityType)
	case map[string]*Relationship:
		return s.writeRelationshipEntities(typedEntities, entityDir, entityType)
	case map[string]*Place:
		return s.writePlaceEntities(typedEntities, entityDir, entityType)
	case map[string]*Source:
		return s.writeSourceEntities(typedEntities, entityDir, entityType)
	case map[string]*Citation:
		return s.writeCitationEntities(typedEntities, entityDir, entityType)
	case map[string]*Repository:
		return s.writeRepositoryEntities(typedEntities, entityDir, entityType)
	case map[string]*Media:
		return s.writeMediaEntities(typedEntities, entityDir, entityType)
	case map[string]*Assertion:
		return s.writeAssertionEntities(typedEntities, entityDir, entityType)
	default:
		return fmt.Errorf("unsupported entity type: %T", entities)
	}
}

// writePersonEntities writes person entities to individual files.
func (s *DefaultSerializer) writePersonEntities(entities map[string]*Person, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeEventEntities writes event entities to individual files.
func (s *DefaultSerializer) writeEventEntities(entities map[string]*Event, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeRelationshipEntities writes relationship entities to individual files.
func (s *DefaultSerializer) writeRelationshipEntities(entities map[string]*Relationship, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writePlaceEntities writes place entities to individual files.
func (s *DefaultSerializer) writePlaceEntities(entities map[string]*Place, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeSourceEntities writes source entities to individual files.
func (s *DefaultSerializer) writeSourceEntities(entities map[string]*Source, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeCitationEntities writes citation entities to individual files.
func (s *DefaultSerializer) writeCitationEntities(entities map[string]*Citation, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeRepositoryEntities writes repository entities to individual files.
func (s *DefaultSerializer) writeRepositoryEntities(entities map[string]*Repository, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeMediaEntities writes media entities to individual files.
func (s *DefaultSerializer) writeMediaEntities(entities map[string]*Media, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeAssertionEntities writes assertion entities to individual files.
func (s *DefaultSerializer) writeAssertionEntities(entities map[string]*Assertion, dir, entityType string) error {
	return writeEntitiesWithID(entities, dir, entityType)
}

// writeEntitiesWithID writes entities with embedded ID field.
// Uses random filenames with collision detection.
func writeEntitiesWithID[T any](entities map[string]T, dir, entityType string) error {
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

		// Write file
		filepath := filepath.Join(dir, filename)
		if err := os.WriteFile(filepath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}
	}

	return nil
}

// LoadSingleFile loads a GLX archive from a single YAML file.
func (s *DefaultSerializer) LoadSingleFile(inputPath string) (*GLXFile, error) {
	// Read file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return s.LoadSingleFileBytes(data)
}

// LoadSingleFileBytes loads a GLX archive from YAML bytes (single-file format).
func (s *DefaultSerializer) LoadSingleFileBytes(data []byte) (*GLXFile, error) {
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

// LoadMultiFile loads a GLX archive from a multi-file directory structure.
func (s *DefaultSerializer) LoadMultiFile(inputDir string) (*GLXFile, error) {
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

	// Load each entity type
	if err := loadEntitiesWithID(filepath.Join(inputDir, "persons"), glx.Persons); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load persons: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "events"), glx.Events); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "relationships"), glx.Relationships); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load relationships: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "places"), glx.Places); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load places: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "sources"), glx.Sources); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load sources: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "citations"), glx.Citations); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load citations: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "repositories"), glx.Repositories); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load repositories: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "media"), glx.Media); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load media: %w", err)
	}
	if err := loadEntitiesWithID(filepath.Join(inputDir, "assertions"), glx.Assertions); err != nil && !os.IsNotExist(err) {
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

// loadEntitiesWithID loads entities with embedded _id field from a directory.
func loadEntitiesWithID[T any](dir string, entities map[string]T) error {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err // Return the error so caller can check os.IsNotExist
	}

	// Read directory
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Load each file
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".glx" {
			continue
		}

		path := filepath.Join(dir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var wrapper EntityWithID[T]
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		entities[wrapper.ID] = wrapper.Entity
	}

	return nil
}

// validateGLXFile performs basic validation on a GLX archive.
// Returns error if validation fails.
func validateGLXFile(glx *GLXFile) error {
	// Basic validation - just check that the file is not nil
	if glx == nil {
		return fmt.Errorf("GLX file is nil")
	}

	// Initialize maps if nil
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

	// TODO: Add more comprehensive validation
	// - Check for dangling references
	// - Validate vocabulary types
	// - Check required fields
	// - Validate date formats
	// etc.

	return nil
}
