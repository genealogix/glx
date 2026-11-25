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
	"maps"
	"path"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
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

	// Indent specifies the indentation string for YAML output (default: "  ")
	Indent string

	// StandardVocabularyFallback makes DeserializeMultiFileFromMap fill every
	// vocabulary category the archive leaves empty, and every property
	// definition it leaves out, from the embedded standard vocabularies
	// before validating (MergeStandardVocabularies). A category the archive
	// defines stays authoritative. Single-file archives are validated under
	// the same rule, so this gives a directory archive the one a single file
	// already has (#1355).
	StandardVocabularyFallback bool
}

// DefaultSerializerOptions returns the default serializer options.
func DefaultSerializerOptions() *SerializerOptions {
	return &SerializerOptions{
		Validate: true,
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

	// Normalize empty metadata so omitempty suppresses it.
	glx = withoutEmptyMetadata(glx)

	yamlBytes, err := marshalYAML(glx, s.Options.Indent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return yamlBytes, nil
}

func withoutEmptyMetadata(archive *GLXFile) *GLXFile {
	if archive.ImportMetadata == nil || archive.ImportMetadata.hasContent() {
		return archive
	}
	normalized := *archive
	normalized.ImportMetadata = nil

	return &normalized
}

// SerializeMultiFileToMap serializes a GLX archive to a map of relative paths to file contents.
// Map keys are relative paths like "persons/person-001.glx", "vocabularies/event-types.glx".
func (s *DefaultSerializer) SerializeMultiFileToMap(glx *GLXFile) (map[string][]byte, error) {
	files := make(map[string][]byte)
	if err := s.serializeMultiFile(glx, func(filename string, data []byte) error {
		files[filename] = data

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func (s *DefaultSerializer) serializeMultiFile(glx *GLXFile, writeFile func(string, []byte) error) error {
	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	// Guard against nil input when validation is disabled.
	if glx == nil {
		return ErrGLXFileNil
	}

	// Add standard vocabularies.
	// path.Join (forward slashes), not filepath.Join: these are logical
	// archive keys, not disk paths, and must be OS-independent so the same
	// GLXFile produces the same map on Linux and Windows. See issue #900.
	vocabularies := StandardVocabularies()
	for _, filename := range slices.Sorted(maps.Keys(vocabularies)) {
		vocabPath := path.Join("vocabularies", filename)
		if err := writeFile(vocabPath, vocabularies[filename]); err != nil {
			return err
		}
	}

	// Serialize metadata if present and non-empty
	if glx.ImportMetadata != nil && glx.ImportMetadata.hasContent() {
		metaWrapper := map[string]*Metadata{"metadata": glx.ImportMetadata}
		yamlBytes, err := marshalYAML(metaWrapper, s.Options.Indent)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		if err := writeFile("metadata.glx", yamlBytes); err != nil {
			return err
		}
	}

	// Serialize entities by type
	if err := serializeEntitiesWrapped(glx.Persons, "persons", "person", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Events, "events", "event", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Relationships, "relationships", "relationship", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Places, "places", "place", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Sources, "sources", "source", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Citations, "citations", "citation", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Repositories, "repositories", "repository", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Media, "media", "media", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.Assertions, "assertions", "assertion", s.Options.Indent, writeFile); err != nil {
		return err
	}
	if err := serializeEntitiesWrapped(glx.ResearchLogs, "research_logs", "research-log", s.Options.Indent, writeFile); err != nil {
		return err
	}

	return serializeEntitiesWrapped(glx.Studies, "studies", "study", s.Options.Indent, writeFile)
}

// serializeEntitiesWrapped serializes each entity as a standard GLX file:
// {collectionKey: {entityID: entity}}. Derives filenames deterministically from entity IDs.
func serializeEntitiesWrapped[T any](entities map[string]*T, dirName, entityType, indent string, writeFile func(string, []byte) error) error {
	lowercaseToID := make(map[string]string, len(entities))

	for _, entityID := range sortedKeys(entities) {
		filename, err := EntityIDToFilename(entityID)
		if err != nil {
			return fmt.Errorf("entity %s %s: %w", entityType, entityID, err)
		}

		// Detect case-insensitive collision
		if existingID, exists := lowercaseToID[filename]; exists {
			return fmt.Errorf("entity IDs %q and %q: %w", existingID, entityID, ErrCaseInsensitiveCollision)
		}
		lowercaseToID[filename] = entityID

		// Wrap as standard GLX structure: {collectionKey: {entityID: entity}}
		wrapper := map[string]map[string]*T{
			dirName: {entityID: entities[entityID]},
		}

		// Marshal to YAML
		yamlBytes, err := marshalYAML(wrapper, indent)
		if err != nil {
			return fmt.Errorf("failed to marshal %s %s: %w", entityType, entityID, err)
		}

		// path.Join, not filepath.Join — same rationale as vocabPath in
		// SerializeMultiFileToMap above (issue #900).
		filePath := path.Join(dirName, filename)
		if err := writeFile(filePath, yamlBytes); err != nil {
			return err
		}
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
	glx := &GLXFile{}
	glx.initMaps()

	var allConflicts []string
	origins := newKeyOrigins()

	// Each file is a GLXFile fragment — the YAML top-level keys (persons:,
	// events:, event_types:, etc.) determine what entities it contains,
	// regardless of which directory the file lives in.
	//
	// Files are merged in sorted path order. Map iteration order is randomized
	// per process, so when two files define the same entity ID the surviving
	// record used to be chosen at random on every run: consecutive invocations
	// returned different data for the same archive, and `glx cache build`
	// froze one arbitrary winner, letting two developers' caches disagree
	// about the same commit. The duplicate is still reported as a conflict;
	// sorting only makes which record wins reproducible.
	filePaths := make([]string, 0, len(files))
	for filePath := range files {
		filePaths = append(filePaths, filePath)
	}
	sort.Strings(filePaths)

	for _, filePath := range filePaths {
		ext := strings.ToLower(path.Ext(filePath))
		if ext != FileExtGLX && ext != ".yaml" && ext != ".yml" {
			continue
		}
		var partial GLXFile
		if err := yaml.Unmarshal(files[filePath], &partial); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
		}
		conflicts, _ := glx.Merge(&partial)
		allConflicts = append(allConflicts, origins.annotate(conflicts, filePath)...)
		origins.record(&partial, filePath)
	}

	// Fill vocabulary categories the archive does not define from the
	// standard vocabularies before validating, when asked to.
	if s.Options.StandardVocabularyFallback {
		if err := MergeStandardVocabularies(glx); err != nil {
			return nil, nil, fmt.Errorf("loading standard vocabularies: %w", err)
		}
	}

	// Validate if requested
	if s.Options.Validate {
		if err := validateGLXFile(glx); err != nil {
			return nil, nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	return glx, allConflicts, nil
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
	if glx.ResearchLogs == nil {
		glx.ResearchLogs = make(map[string]*ResearchLog)
	}
	if glx.Studies == nil {
		glx.Studies = make(map[string]*Study)
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
