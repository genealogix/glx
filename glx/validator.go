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

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/genealogix/spec/lib"
	schema "github.com/genealogix/spec/specification/schema/v1"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

const (
	// File extensions
	FileExtGLX  = ".glx"
	FileExtYAML = ".yaml"
	FileExtYML  = ".yml"

	// ID validation constants
	MinEntityIDLength = 1
	MaxEntityIDLength = 64
)

// ParseYAMLFile parses YAML content into a map
func ParseYAMLFile(data []byte) (map[string]interface{}, error) {
	var doc interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	// Convert to map[string]interface{} and normalize keys
	normalized := normalizeYAMLMap(doc)
	if result, ok := normalized.(map[string]interface{}); ok {
		return result, nil
	}
	return nil, fmt.Errorf("YAML document is not an object")
}

// normalizeYAMLMap recursively converts map[interface{}]interface{} to map[string]interface{}
// This handles YAML files with numeric keys like quality_ratings
func normalizeYAMLMap(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			keyStr := fmt.Sprintf("%v", key)
			result[keyStr] = normalizeYAMLMap(value)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = normalizeYAMLMap(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = normalizeYAMLMap(item)
		}
		return result
	default:
		return v
	}
}

// loadAndResolveSchema loads a schema and recursively resolves all $ref entries
func loadAndResolveSchema(filename string) (map[string]interface{}, error) {
	// Read the main schema file
	data, err := schema.EntitySchemas.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema %s: %w", filename, err)
	}

	var schemaDoc map[string]interface{}
	if err := json.Unmarshal(data, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to parse schema %s: %w", filename, err)
	}

	// Recursively resolve all $ref entries
	if err := resolveRefs(schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to resolve refs in %s: %w", filename, err)
	}

	return schemaDoc, nil
}

// resolveRefs recursively walks a schema and replaces $ref with the actual schema content
// root is the top-level schema document for resolving JSON Pointer references (#/...)
func resolveRefs(obj interface{}) error {
	return resolveRefsWithRoot(obj, nil)
}

func resolveRefsWithRoot(obj interface{}, root map[string]interface{}) error {
	switch v := obj.(type) {
	case map[string]interface{}:
		// Set root to this object if it's the top level
		if root == nil {
			root = v
		}

		// Check if this object has a $ref
		if refStr, ok := v["$ref"].(string); ok {
			if strings.HasPrefix(refStr, "#/") {
				// JSON Pointer reference (internal to schema)
				resolved, err := resolveJSONPointer(root, refStr)
				if err != nil {
					return fmt.Errorf("failed to resolve JSON pointer %s: %w", refStr, err)
				}
				
				// Replace $ref with resolved content
				delete(v, "$ref")
				for key, value := range resolved {
					if key != "$schema" && key != "$id" {
						v[key] = value
					}
				}
			} else {
				// File reference - load from embedded FS
				refData, err := schema.EntitySchemas.ReadFile(refStr)
				if err != nil {
					return fmt.Errorf("failed to read referenced schema %s: %w", refStr, err)
				}

				var refSchema map[string]interface{}
				if err := json.Unmarshal(refData, &refSchema); err != nil {
					return fmt.Errorf("failed to parse referenced schema %s: %w", refStr, err)
				}

				// Resolve any nested refs in the referenced schema
				if err := resolveRefsWithRoot(refSchema, refSchema); err != nil {
					return err
				}

				// For vocabulary schemas, extract the actual pattern/entry definition
				// They have structure: {"properties": {"vocab_name": {"patternProperties" or "additionalProperties": {...}}}}
				if strings.Contains(refStr, "vocabularies/") {
					if props, ok := refSchema["properties"].(map[string]interface{}); ok {
						// Get the first (and only) property key
						for _, vocabDef := range props {
							if vocabMap, ok := vocabDef.(map[string]interface{}); ok {
								// Try patternProperties first (event_types, etc.)
								if pattern, ok := vocabMap["patternProperties"].(map[string]interface{}); ok {
									// Extract the pattern definition (first pattern)
									for _, patternDef := range pattern {
										// This is the individual entry schema - use it directly
										if entrySchema, ok := patternDef.(map[string]interface{}); ok {
											delete(v, "$ref")
											for key, value := range entrySchema {
												v[key] = value
											}
											return nil
										}
									}
								}
								// Try additionalProperties (person_properties, etc.)
								if addlProps, ok := vocabMap["additionalProperties"].(map[string]interface{}); ok {
									// This might be a $ref to #/definitions/PropertyDefinition
									// The ref has already been resolved, so use it directly
									delete(v, "$ref")
									for key, value := range addlProps {
										v[key] = value
									}
									return nil
								}
							}
						}
					}
				}

				// For non-vocabulary refs, use the whole schema
				delete(v, "$ref")
				for key, value := range refSchema {
					if key != "$schema" && key != "$id" {
						v[key] = value
					}
				}
			}
		} else {
			// No $ref, but recursively process all values
			for _, value := range v {
				if err := resolveRefsWithRoot(value, root); err != nil {
					return err
				}
			}
		}

	case []interface{}:
		// Process array elements
		for _, item := range v {
			if err := resolveRefsWithRoot(item, root); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveJSONPointer resolves a JSON Pointer reference like #/definitions/PropertyDefinition
func resolveJSONPointer(root map[string]interface{}, pointer string) (map[string]interface{}, error) {
	// Remove the leading #/
	path := strings.TrimPrefix(pointer, "#/")
	parts := strings.Split(path, "/")
	
	current := interface{}(root)
	for _, part := range parts {
		// Unescape JSON Pointer tokens
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")
		
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("path not found: %s (missing key: %s)", pointer, part)
			}
		default:
			return nil, fmt.Errorf("invalid path: %s (not an object at: %s)", pointer, part)
		}
	}
	
	result, ok := current.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("pointer does not reference an object: %s", pointer)
	}
	
	return result, nil
}

// ValidateGLXFileStructure validates a single GLX file against the master schema
func ValidateGLXFileStructure(doc map[string]interface{}) []string {
	var issues []string

	// Load and resolve master schema
	resolvedSchema, err := loadAndResolveSchema("glx-file.schema.json")
	if err != nil {
		return []string{fmt.Sprintf("failed to load schema: %v", err)}
	}

	// Convert resolved schema to bytes
	schemaBytes, err := json.Marshal(resolvedSchema)
	if err != nil {
		return []string{fmt.Sprintf("failed to marshal resolved schema: %v", err)}
	}
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Validate against JSON schema
	entityJSON, err := json.Marshal(doc)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal entity: %v", err))
		return issues
	}
	documentLoader := gojsonschema.NewBytesLoader(entityJSON)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		issues = append(issues, fmt.Sprintf("schema validation failed: %v", err))
	} else if !result.Valid() {
		for _, desc := range result.Errors() {
			issues = append(issues, desc.String())
		}
	}

	// Validate entity ID formats (only for entity types, not vocabularies or property definitions)
	entityTypes := []string{
		lib.EntityTypePersons, lib.EntityTypeEvents, lib.EntityTypeRelationships,
		lib.EntityTypePlaces, lib.EntityTypeSources, lib.EntityTypeCitations,
		lib.EntityTypeRepositories, lib.EntityTypeAssertions, lib.EntityTypeMedia,
	}
	
	for _, entityType := range entityTypes {
		if entities, ok := doc[entityType].(map[string]interface{}); ok {
			for entityID := range entities {
				if !isValidEntityID(entityID) {
					issues = append(issues, fmt.Sprintf("%s[%s]: invalid entity ID (must be alphanumeric/hyphens, 1-64 chars)", entityType, entityID))
				}
			}
		}
	}

	return issues
}

func isValidEntityID(id string) bool {
	if len(id) < MinEntityIDLength || len(id) > MaxEntityIDLength {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-') {
			return false
		}
	}
	return true
}

// LoadArchive loads and merges all GLX files from a directory into a single GLXFile struct
func LoadArchive(rootPath string) (*lib.GLXFile, []string, error) {
	merged := &lib.GLXFile{
		Persons:       make(map[string]*lib.Person),
		Relationships: make(map[string]*lib.Relationship),
		Events:        make(map[string]*lib.Event),
		Places:        make(map[string]*lib.Place),
		Sources:       make(map[string]*lib.Source),
		Citations:     make(map[string]*lib.Citation),
		Repositories:  make(map[string]*lib.Repository),
		Assertions:    make(map[string]*lib.Assertion),
		Media:         make(map[string]*lib.Media),

		EventTypes:        make(map[string]*lib.EventType),
		ParticipantRoles:  make(map[string]*lib.ParticipantRole),
		ConfidenceLevels:  make(map[string]*lib.ConfidenceLevel),
		RelationshipTypes: make(map[string]*lib.RelationshipType),
		PlaceTypes:        make(map[string]*lib.PlaceType),
		SourceTypes:       make(map[string]*lib.SourceType),
		RepositoryTypes:   make(map[string]*lib.RepositoryType),
		MediaTypes:        make(map[string]*lib.MediaType),
		QualityRatings:    make(map[string]*lib.QualityRating),

		PersonProperties:       make(map[string]*lib.PropertyDefinition),
		EventProperties:        make(map[string]*lib.PropertyDefinition),
		RelationshipProperties: make(map[string]*lib.PropertyDefinition),
		PlaceProperties:        make(map[string]*lib.PropertyDefinition),
	}

	var allDuplicates []string

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != FileExtGLX && ext != FileExtYAML && ext != FileExtYML {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// YAML parsing
		doc, err := ParseYAMLFile(data)
		if err != nil {
			// This check happens before loading, so we can just return a generic error
			return fmt.Errorf("failed to parse YAML file %s: %v", path, err)
		}

		// Structural validation against master schema
		issues := ValidateGLXFileStructure(doc)
		if len(issues) > 0 {
			// This is not ideal as it returns on first file with errors.
			// The CLI will handle collecting errors from all files.
			// For now, returning an error is sufficient to stop the process.
			errorMessages := make([]string, len(issues))
			for i, issue := range issues {
				errorMessages[i] = issue
			}
			return fmt.Errorf("validation of file %s failed:\n- %s", path, strings.Join(errorMessages, "\n- "))
		}

		var glxFile lib.GLXFile
		err = yaml.Unmarshal(data, &glxFile)
		if err != nil {
			// This should not happen if parsing and structural validation passed
			return nil
		}

		duplicates := merged.Merge(&glxFile)
		allDuplicates = append(allDuplicates, duplicates...)

		return nil
	})

	return merged, allDuplicates, err
}

// isVocabularyType checks if a key is a known vocabulary type.
func isVocabularyType(key string) bool {
	vocabKeys := []string{
		lib.VocabRelationshipTypes, lib.VocabEventTypes, lib.VocabPlaceTypes,
		lib.VocabRepositoryTypes, lib.VocabParticipantRoles, lib.VocabMediaTypes,
		lib.VocabConfidenceLevels, lib.VocabQualityRatings, lib.VocabSourceTypes,
		lib.PropPersonProperties, lib.PropEventProperties,
		lib.PropRelationshipProperties, lib.PropPlaceProperties,
	}
	for _, v := range vocabKeys {
		if key == v {
			return true
		}
	}
	return false
}
