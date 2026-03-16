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
	"maps"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
	schema "github.com/genealogix/glx/specification/schema/v1"
)

const (
	// ID validation constants
	MinEntityIDLength = 1
	MaxEntityIDLength = 64
)

// compiledSchema holds the fully compiled JSON schema. Compiled once on first use
// via sync.Once, then reused for all subsequent validations. This avoids re-parsing
// the schema and re-compiling regexp patterns (~2 MB of regexp allocs) per file.
var (
	compiledSchema     *gojsonschema.Schema
	compiledSchemaOnce sync.Once
	compiledSchemaErr  error
)

// ParseYAMLFile parses YAML content into a map
func ParseYAMLFile(data []byte) (map[string]any, error) {
	var doc any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	// Convert to map[string]any and normalize keys
	normalized := normalizeYAMLMap(doc)
	if result, ok := normalized.(map[string]any); ok {
		return result, nil
	}

	return nil, ErrYAMLNotObject
}

// normalizeYAMLMap recursively converts map[any]any to map[string]any
// This handles YAML files with various key types
func normalizeYAMLMap(val any) any {
	switch v := val.(type) {
	case map[any]any:
		result := make(map[string]any)
		for key, value := range v {
			keyStr := fmt.Sprintf("%v", key)
			result[keyStr] = normalizeYAMLMap(value)
		}

		return result
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			result[key] = normalizeYAMLMap(value)
		}

		return result
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = normalizeYAMLMap(item)
		}

		return result
	default:
		return v
	}
}

// loadAndResolveSchema loads a schema and recursively resolves all $ref entries
func loadAndResolveSchema(filename string) (map[string]any, error) {
	// Read the main schema file
	data, err := schema.EntitySchemas.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema %s: %w", filename, err)
	}

	var schemaDoc map[string]any
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
func resolveRefs(obj any) error {
	return resolveRefsWithRoot(obj, nil)
}

func resolveRefsWithRoot(obj any, root map[string]any) error {
	switch v := obj.(type) {
	case map[string]any:
		// Set root to this object if it's the top level
		if root == nil {
			root = v
		}

		// Check if this object has a $ref
		refStr, hasRef := v["$ref"].(string)
		if !hasRef {
			// No $ref, recursively process all values
			return resolveMapValues(v, root)
		}

		// Has $ref - resolve it
		if strings.HasPrefix(refStr, "#/") {
			return resolveJSONPointerRef(v, root, refStr)
		}

		return resolveFileRef(v, refStr)

	case []any:
		// Process array elements
		for _, item := range v {
			if err := resolveRefsWithRoot(item, root); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveMapValues recursively processes all values in a map
func resolveMapValues(m, root map[string]any) error {
	for _, value := range m {
		if err := resolveRefsWithRoot(value, root); err != nil {
			return err
		}
	}

	return nil
}

// resolveJSONPointerRef resolves a JSON Pointer reference like #/definitions/Something
func resolveJSONPointerRef(target, root map[string]any, refStr string) error {
	resolved, err := resolveJSONPointer(root, refStr)
	if err != nil {
		return fmt.Errorf("failed to resolve JSON pointer %s: %w", refStr, err)
	}

	// Replace $ref with resolved content
	delete(target, "$ref")
	for key, value := range resolved {
		if key != "$schema" && key != "$id" {
			target[key] = value
		}
	}

	return nil
}

// resolveFileRef resolves a file reference and merges it into the target map
func resolveFileRef(target map[string]any, refStr string) error {
	// Load from embedded FS
	refData, err := schema.EntitySchemas.ReadFile(refStr)
	if err != nil {
		return fmt.Errorf("failed to read referenced schema %s: %w", refStr, err)
	}

	var refSchema map[string]any
	if err := json.Unmarshal(refData, &refSchema); err != nil {
		return fmt.Errorf("failed to parse referenced schema %s: %w", refStr, err)
	}

	// Resolve any nested refs in the referenced schema
	if err := resolveRefsWithRoot(refSchema, refSchema); err != nil {
		return err
	}

	// Handle vocabulary schemas specially
	if strings.Contains(refStr, "vocabularies/") {
		return resolveVocabularyRef(target, refSchema)
	}

	// For non-vocabulary refs, use the whole schema
	mergeSchemaIntoTarget(target, refSchema)

	return nil
}

// resolveVocabularyRef extracts the pattern/entry definition from vocabulary schemas
func resolveVocabularyRef(target, refSchema map[string]any) error {
	props, ok := refSchema["properties"].(map[string]any)
	if !ok {
		// No properties, use whole schema
		mergeSchemaIntoTarget(target, refSchema)

		return nil
	}

	// Get the first (and only) property
	for _, vocabDef := range props {
		vocabMap, ok := vocabDef.(map[string]any)
		if !ok {
			continue
		}

		// Try patternProperties first (event_types, etc.)
		if extracted, ok := extractPatternProperties(vocabMap); ok {
			delete(target, "$ref")
			maps.Copy(target, extracted)

			return nil
		}

		// Try additionalProperties (person_properties, etc.)
		if extracted, ok := extractAdditionalProperties(vocabMap); ok {
			delete(target, "$ref")
			maps.Copy(target, extracted)

			return nil
		}
	}

	// Fallback to whole schema
	mergeSchemaIntoTarget(target, refSchema)

	return nil
}

// extractPatternProperties extracts the first pattern definition from patternProperties
func extractPatternProperties(vocabMap map[string]any) (map[string]any, bool) {
	pattern, ok := vocabMap["patternProperties"].(map[string]any)
	if !ok {
		return nil, false
	}

	// Extract the first pattern definition
	for _, patternDef := range pattern {
		if entrySchema, ok := patternDef.(map[string]any); ok {
			return entrySchema, true
		}
	}

	return nil, false
}

// extractAdditionalProperties extracts the additionalProperties definition
func extractAdditionalProperties(vocabMap map[string]any) (map[string]any, bool) {
	addlProps, ok := vocabMap["additionalProperties"].(map[string]any)
	if !ok {
		return nil, false
	}

	return addlProps, true
}

// mergeSchemaIntoTarget merges a schema into the target map, excluding $schema and $id
func mergeSchemaIntoTarget(target, refSchema map[string]any) {
	delete(target, "$ref")
	for key, value := range refSchema {
		if key != "$schema" && key != "$id" {
			target[key] = value
		}
	}
}

// resolveJSONPointer resolves a JSON Pointer reference like #/definitions/PropertyDefinition
func resolveJSONPointer(root map[string]any, pointer string) (map[string]any, error) {
	// Remove the leading #/
	path := strings.TrimPrefix(pointer, "#/")
	parts := strings.Split(path, "/")

	current := any(root)
	for _, part := range parts {
		// Unescape JSON Pointer tokens
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")

		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("%w: %s (missing key: %s)", ErrPathNotFound, pointer, part)
			}
		default:
			return nil, fmt.Errorf("%w: %s (not an object at: %s)", ErrInvalidPath, pointer, part)
		}
	}

	result, ok := current.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPointerNotObject, pointer)
	}

	return result, nil
}

// ValidateGLXFileStructure validates a single GLX file against the master schema
func ValidateGLXFileStructure(doc map[string]any) []string {
	var issues []string

	// Compile the schema once (resolves $ref, compiles regexps) and reuse it.
	compiledSchemaOnce.Do(func() {
		resolved, err := loadAndResolveSchema("glx-file.schema.json")
		if err != nil {
			compiledSchemaErr = err
			return
		}
		schemaBytes, err := json.Marshal(resolved)
		if err != nil {
			compiledSchemaErr = fmt.Errorf("failed to marshal resolved schema: %w", err)
			return
		}
		compiledSchema, compiledSchemaErr = gojsonschema.NewSchema(
			gojsonschema.NewBytesLoader(schemaBytes))
	})
	if compiledSchemaErr != nil {
		return []string{fmt.Sprintf("failed to load schema: %v", compiledSchemaErr)}
	}

	// Validate against the pre-compiled schema
	entityJSON, err := json.Marshal(doc)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal entity: %v", err))

		return issues
	}
	result, err := compiledSchema.Validate(gojsonschema.NewBytesLoader(entityJSON))
	if err != nil {
		issues = append(issues, fmt.Sprintf("schema validation failed: %v", err))
	} else if !result.Valid() {
		for _, desc := range result.Errors() {
			issues = append(issues, desc.String())
		}
	}

	// Validate entity ID formats (only for entity types, not vocabularies or property definitions)
	entityTypes := []string{
		glxlib.EntityTypePersons, glxlib.EntityTypeEvents, glxlib.EntityTypeRelationships,
		glxlib.EntityTypePlaces, glxlib.EntityTypeSources, glxlib.EntityTypeCitations,
		glxlib.EntityTypeRepositories, glxlib.EntityTypeAssertions, glxlib.EntityTypeMedia,
	}

	for _, entityType := range entityTypes {
		if entities, ok := doc[entityType].(map[string]any); ok {
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
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '-' {
			return false
		}
	}

	return true
}
