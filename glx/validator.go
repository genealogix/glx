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
	"reflect"
	"strconv"
	"strings"

	"github.com/genealogix/spec/lib"
	schema "github.com/genealogix/spec/specification/schema/v1"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

// ParseYAMLFile parses YAML content into a map
func ParseYAMLFile(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// ValidateGLXFile validates a single GLX file
func ValidateGLXFile(path string, doc map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check if this is a vocabulary file
	vocabKeys := []string{"relationship_types", "event_types", "place_types", "repository_types",
		"participant_roles", "media_types", "confidence_levels", "quality_ratings"}
	isVocabFile := false
	for _, key := range vocabKeys {
		if _, exists := doc[key]; exists {
			isVocabFile = true
			break
		}
	}

	if !isVocabFile {
		// Regular entity file - check for entity type keys
		validKeys := []string{"persons", "relationships", "events", "places",
			"sources", "citations", "repositories", "assertions", "media"}
		hasValidKey := false
		for _, key := range validKeys {
			if _, exists := doc[key]; exists {
				hasValidKey = true
				break
			}
		}

		if !hasValidKey {
			return []string{"file must contain at least one entity type key (persons, relationships, events, places, sources, citations, repositories, assertions, media)"}
		}

		// If this is a vocabulary file, validate it as such
		if isVocabFile {
			return ValidateVocabularyFile(path, doc)
		}
	}

	// Validate each entity type section
	entityTypes := map[string]string{
		"persons":       "person",
		"relationships": "relationship",
		"events":        "event",
		"places":        "place",
		"sources":       "source",
		"citations":     "citation",
		"repositories":  "repository",
		"assertions":    "assertion",
		"media":         "media",
	}

	for pluralKey, singularType := range entityTypes {
		if entities, ok := doc[pluralKey].(map[string]interface{}); ok {
			for entityID, entityData := range entities {
				if entityMap, ok := entityData.(map[string]interface{}); ok {
					// Reject if entity has an "id" field (map key is the ID)
					if _, hasID := entityMap["id"]; hasID {
						issues = append(issues, fmt.Sprintf("%s[%s]: entity must not have 'id' field - the map key is the ID", pluralKey, entityID))
						continue
					}

					// Validate entity ID format (alphanumeric + hyphens, 1-64 chars)
					if !isValidEntityID(entityID) {
						issues = append(issues, fmt.Sprintf("%s[%s]: invalid entity ID (must be alphanumeric/hyphens, 1-64 chars)", pluralKey, entityID))
					}

					// Validate individual entity
					entityIssues := validateEntityByType(singularType, entityMap, vocabs)
					for _, issue := range entityIssues {
						issues = append(issues, fmt.Sprintf("%s[%s]: %s", pluralKey, entityID, issue))
					}
				}
			}
		}
	}

	return issues
}

func isValidEntityID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-') {
			return false
		}
	}
	return true
}

func validateEntityByType(entityType string, entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check for version field (required for all entities)
	if _, hasVersion := entity["version"]; !hasVersion {
		issues = append(issues, "version is required")
		return issues
	}

	// Get schema bytes
	schemaBytes := getSchemaBytes(entityType)
	if schemaBytes == nil {
		// Fallback to basic validation if schema not found
		return basicValidateEntity(entityType, entity, vocabs)
	}

	// Load and validate against JSON schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Convert entity to JSON for validation
	entityJSON, err := json.Marshal(entity)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal entity: %v", err))
		return issues
	}

	documentLoader := gojsonschema.NewBytesLoader(entityJSON)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		// If schema validation fails, fall back to basic validation
		return basicValidateEntity(entityType, entity, vocabs)
	}

	if !result.Valid() {
		for _, desc := range result.Errors() {
			issues = append(issues, desc.String())
		}
	}

	return issues
}

func getSchemaBytes(entityType string) []byte {
	if schemaBytes, ok := schema.EntitySchemas[entityType]; ok {
		return schemaBytes
	}
	return nil
}

func basicValidateEntity(entityType string, entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Type-specific validation
	switch entityType {
	case "person":
		// Persons don't have strict requirements beyond version
	case "relationship":
		if _, hasType := entity["type"]; !hasType {
			issues = append(issues, "type is required")
		}
		if _, hasPersons := entity["persons"]; !hasPersons {
			issues = append(issues, "persons is required")
		}
		// Validate against vocabularies if available
		if relType, ok := entity["type"].(string); ok {
			if vocabs != nil && len(vocabs.RelationshipTypes) > 0 {
				if _, exists := vocabs.RelationshipTypes[relType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown relationship type '%s' - add a relationship_types vocabulary entry", relType))
				}
			}
		}
	case "event":
		if _, hasType := entity["type"]; !hasType {
			issues = append(issues, "type is required")
		}
		// Validate against vocabularies if available
		if eventType, ok := entity["type"].(string); ok {
			if vocabs != nil && len(vocabs.EventTypes) > 0 {
				if _, exists := vocabs.EventTypes[eventType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown event type '%s' - add an event_types vocabulary entry", eventType))
				}
			}
		}
	case "place":
		if _, hasName := entity["name"]; !hasName {
			issues = append(issues, "name is required")
		}
		// Validate against vocabularies if available
		if placeType, ok := entity["type"].(string); ok {
			if vocabs != nil && len(vocabs.PlaceTypes) > 0 {
				if _, exists := vocabs.PlaceTypes[placeType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown place type '%s' - add a place_types vocabulary entry", placeType))
				}
			}
		}
	case "source":
		if _, hasTitle := entity["title"]; !hasTitle {
			issues = append(issues, "title is required")
		}
	case "citation":
		if _, hasSource := entity["source"]; !hasSource {
			issues = append(issues, "source is required")
		}
	case "repository":
		if _, hasName := entity["name"]; !hasName {
			issues = append(issues, "name is required")
		}
		// Validate against vocabularies if available
		if repoType, ok := entity["type"].(string); ok {
			if vocabs != nil && len(vocabs.RepositoryTypes) > 0 {
				if _, exists := vocabs.RepositoryTypes[repoType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown repository type '%s' - add a repository_types vocabulary entry", repoType))
				}
			}
		}
	case "assertion":
		if _, hasSubject := entity["subject"]; !hasSubject {
			issues = append(issues, "subject is required")
		}
		if _, hasClaim := entity["claim"]; !hasClaim {
			issues = append(issues, "claim is required")
		}
		if _, hasSources := entity["sources"]; !hasSources {
			if _, hasCitations := entity["citations"]; !hasCitations {
				issues = append(issues, "sources or citations is required")
			}
		}
		// Validate against vocabularies if available
		if confidence, ok := entity["confidence"].(string); ok {
			if vocabs != nil && len(vocabs.ConfidenceLevels) > 0 {
				if _, exists := vocabs.ConfidenceLevels[confidence]; !exists {
					issues = append(issues, fmt.Sprintf("unknown confidence level '%s' - add a confidence_levels vocabulary entry", confidence))
				}
			}
		}
		if quality, ok := entity["quality"].(float64); ok {
			qualityStr := fmt.Sprintf("%.0f", quality)
			if vocabs != nil && len(vocabs.QualityRatings) > 0 {
				if _, exists := vocabs.QualityRatings[qualityStr]; !exists {
					issues = append(issues, fmt.Sprintf("unknown quality rating '%s' - add a quality_ratings vocabulary entry", qualityStr))
				}
			}
		}
		if quality, ok := entity["quality"].(*int); ok && quality != nil {
			qualityStr := fmt.Sprintf("%d", *quality)
			if vocabs != nil && len(vocabs.QualityRatings) > 0 {
				if _, exists := vocabs.QualityRatings[qualityStr]; !exists {
					issues = append(issues, fmt.Sprintf("unknown quality rating '%s' - add a quality_ratings vocabulary entry", qualityStr))
				}
			}
		}
	case "media":
		if _, hasURI := entity["uri"]; !hasURI {
			if _, hasFilePath := entity["file_path"]; !hasFilePath {
				issues = append(issues, "uri or file_path is required")
			}
		}
		// Validate against vocabularies if available
		if mediaType, ok := entity["media_type"].(string); ok {
			if vocabs != nil && len(vocabs.MediaTypes) > 0 {
				if _, exists := vocabs.MediaTypes[mediaType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown media type '%s' - add a media_types vocabulary entry", mediaType))
				}
			}
		}
	}

	return issues
}

// ValidateVocabularyFile validates a vocabulary file against its schema
func ValidateVocabularyFile(path string, doc map[string]interface{}) []string {
	var issues []string

	// Check for vocabulary type key
	vocabKeys := []string{"relationship_types", "event_types", "place_types", "repository_types",
		"participant_roles", "media_types", "confidence_levels", "quality_ratings"}
	hasVocabKey := false
	var vocabType string
	for _, key := range vocabKeys {
		if _, exists := doc[key]; exists {
			hasVocabKey = true
			vocabType = key
			break
		}
	}

	if !hasVocabKey {
		return []string{"vocabulary file must contain exactly one vocabulary type key"}
	}

	// Get schema bytes for this vocabulary type
	schemaBytes := getVocabSchemaBytes(vocabType)
	if schemaBytes == nil {
		return []string{fmt.Sprintf("no schema found for vocabulary type: %s", vocabType)}
	}

	// Validate against JSON schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
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

	return issues
}

func getVocabSchemaBytes(vocabType string) []byte {
	if schemaBytes, ok := schema.VocabularySchemas[vocabType]; ok {
		return schemaBytes
	}
	return nil
}

type ArchiveVocabularies struct {
	RelationshipTypes map[string]*lib.RelationshipType
	EventTypes        map[string]*lib.EventType
	PlaceTypes        map[string]*lib.PlaceType
	RepositoryTypes   map[string]*lib.RepositoryType
	ParticipantRoles  map[string]*lib.ParticipantRole
	MediaTypes        map[string]*lib.MediaType
	ConfidenceLevels  map[string]*lib.ConfidenceLevel
	QualityRatings    map[string]*lib.QualityRating
}

func LoadArchiveVocabularies(rootPath string) (*ArchiveVocabularies, error) {
	vocabs := &ArchiveVocabularies{
		RelationshipTypes: make(map[string]*lib.RelationshipType),
		EventTypes:        make(map[string]*lib.EventType),
		PlaceTypes:        make(map[string]*lib.PlaceType),
		RepositoryTypes:   make(map[string]*lib.RepositoryType),
		ParticipantRoles:  make(map[string]*lib.ParticipantRole),
		MediaTypes:        make(map[string]*lib.MediaType),
		ConfidenceLevels:  make(map[string]*lib.ConfidenceLevel),
		QualityRatings:    make(map[string]*lib.QualityRating),
	}

	// Walk all GLX files and extract vocabulary definitions from any file
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var glxFile lib.GLXFile
		if err := yaml.Unmarshal(data, &glxFile); err != nil {
			// Skip files that can't be parsed
			return nil
		}

		// Merge vocabulary definitions from this file
		for k, v := range glxFile.RelationshipTypes {
			vocabs.RelationshipTypes[k] = v
		}
		for k, v := range glxFile.EventTypes {
			vocabs.EventTypes[k] = v
		}
		for k, v := range glxFile.PlaceTypes {
			vocabs.PlaceTypes[k] = v
		}
		for k, v := range glxFile.RepositoryTypes {
			vocabs.RepositoryTypes[k] = v
		}
		for k, v := range glxFile.ParticipantRoles {
			vocabs.ParticipantRoles[k] = v
		}
		for k, v := range glxFile.MediaTypes {
			vocabs.MediaTypes[k] = v
		}
		for k, v := range glxFile.ConfidenceLevels {
			vocabs.ConfidenceLevels[k] = v
		}
		for k, v := range glxFile.QualityRatings {
			vocabs.QualityRatings[k] = v
		}

		return nil
	})

	return vocabs, err
}

// CollectAllEntities walks all GLX files and collects entity IDs
func CollectAllEntities(rootPath string) (map[string]map[string]bool, []string, error) {
	allEntities := make(map[string]map[string]bool)
	for _, entityType := range []string{"persons", "relationships", "events", "places",
		"sources", "citations", "repositories", "assertions", "media"} {
		allEntities[entityType] = make(map[string]bool)
	}

	var duplicates []string

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		doc, err := ParseYAMLFile(data)
		if err != nil {
			return nil
		}

		// Collect entity IDs from this file
		for pluralKey, entities := range doc {
			if entityMap, ok := entities.(map[string]interface{}); ok {
				if _, exists := allEntities[pluralKey]; exists {
					for entityID := range entityMap {
						if allEntities[pluralKey][entityID] {
							duplicates = append(duplicates, fmt.Sprintf("duplicate entity ID %s found in %s", entityID, path))
						}
						allEntities[pluralKey][entityID] = true
					}
				}
			}
		}
		return nil
	})

	return allEntities, duplicates, err
}

// ValidateRepositoryReferences validates all cross-references across the entire repository
func ValidateRepositoryReferences(rootPath string, allEntities map[string]map[string]bool) []string {
	var issues []string

	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != ".glx" && ext != ".yaml" && ext != ".yml" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		doc, err := ParseYAMLFile(data)
		if err != nil {
			return nil
		}

		// Validate relationships
		if relationships, ok := doc["relationships"].(map[string]interface{}); ok {
			for relID, relData := range relationships {
				if rel, ok := relData.(map[string]interface{}); ok {
					// Check participants reference valid persons
					if participants, ok := rel["participants"].([]interface{}); ok {
						for i, p := range participants {
							if participant, ok := p.(map[string]interface{}); ok {
								if personID, ok := participant["person"].(string); ok {
									if !allEntities["persons"][personID] {
										issues = append(issues, fmt.Sprintf("%s: relationships[%s].participants[%d].person references non-existent person: %s", path, relID, i, personID))
									}
								}
							}
						}
					}
					// Also check old format with persons array
					if persons, ok := rel["persons"].([]interface{}); ok {
						for i, p := range persons {
							if personID, ok := p.(string); ok {
								if !allEntities["persons"][personID] {
									issues = append(issues, fmt.Sprintf("%s: relationships[%s].persons[%d] references non-existent person: %s", path, relID, i, personID))
								}
							}
						}
					}
				}
			}
		}

		// Validate events
		if events, ok := doc["events"].(map[string]interface{}); ok {
			for eventID, eventData := range events {
				if event, ok := eventData.(map[string]interface{}); ok {
					// Check place references
					if placeID, ok := event["place"].(string); ok {
						if !allEntities["places"][placeID] {
							issues = append(issues, fmt.Sprintf("%s: events[%s].place references non-existent place: %s", path, eventID, placeID))
						}
					}
					// Check participants reference valid persons
					if participants, ok := event["participants"].([]interface{}); ok {
						for i, p := range participants {
							if participant, ok := p.(map[string]interface{}); ok {
								if personID, ok := participant["person"].(string); ok {
									if !allEntities["persons"][personID] {
										issues = append(issues, fmt.Sprintf("%s: events[%s].participants[%d].person references non-existent person: %s", path, eventID, i, personID))
									}
								}
							}
						}
					}
				}
			}
		}

		// Validate places
		if places, ok := doc["places"].(map[string]interface{}); ok {
			for placeID, placeData := range places {
				if place, ok := placeData.(map[string]interface{}); ok {
					// Check parent place references
					if parentID, ok := place["parent"].(string); ok {
						if !allEntities["places"][parentID] {
							issues = append(issues, fmt.Sprintf("%s: places[%s].parent references non-existent place: %s", path, placeID, parentID))
						}
					}
				}
			}
		}

		// Validate citations
		if citations, ok := doc["citations"].(map[string]interface{}); ok {
			for citationID, citationData := range citations {
				if citation, ok := citationData.(map[string]interface{}); ok {
					// Check source references
					if sourceID, ok := citation["source"].(string); ok {
						if !allEntities["sources"][sourceID] {
							issues = append(issues, fmt.Sprintf("%s: citations[%s].source references non-existent source: %s", path, citationID, sourceID))
						}
					}
					// Check repository references
					if repoID, ok := citation["repository"].(string); ok {
						if !allEntities["repositories"][repoID] {
							issues = append(issues, fmt.Sprintf("%s: citations[%s].repository references non-existent repository: %s", path, citationID, repoID))
						}
					}
				}
			}
		}

		// Validate sources
		if sources, ok := doc["sources"].(map[string]interface{}); ok {
			for sourceID, sourceData := range sources {
				if source, ok := sourceData.(map[string]interface{}); ok {
					// Check repository references
					if repoID, ok := source["repository"].(string); ok {
						if !allEntities["repositories"][repoID] {
							issues = append(issues, fmt.Sprintf("%s: sources[%s].repository references non-existent repository: %s", path, sourceID, repoID))
						}
					}
				}
			}
		}

		// Validate assertions
		if assertions, ok := doc["assertions"].(map[string]interface{}); ok {
			for assertionID, assertionData := range assertions {
				if assertion, ok := assertionData.(map[string]interface{}); ok {
					// Check subject references (could be person, event, relationship, place)
					if subjectID, ok := assertion["subject"].(string); ok {
						found := false
						for _, entityType := range []string{"persons", "events", "relationships", "places"} {
							if allEntities[entityType][subjectID] {
								found = true
								break
							}
						}
						if !found {
							issues = append(issues, fmt.Sprintf("%s: assertions[%s].subject references non-existent entity: %s", path, assertionID, subjectID))
						}
					}
					// Check citations
					if citations, ok := assertion["citations"].([]interface{}); ok {
						for i, c := range citations {
							if citationID, ok := c.(string); ok {
								if !allEntities["citations"][citationID] {
									issues = append(issues, fmt.Sprintf("%s: assertions[%s].citations[%d] references non-existent citation: %s", path, assertionID, i, citationID))
								}
							}
						}
					}
					// Also check old format with sources
					if sources, ok := assertion["sources"].([]interface{}); ok {
						for i, s := range sources {
							if sourceID, ok := s.(string); ok {
								if !allEntities["sources"][sourceID] {
									issues = append(issues, fmt.Sprintf("%s: assertions[%s].sources[%d] references non-existent source: %s", path, assertionID, i, sourceID))
								}
							}
						}
					}
				}
			}
		}

		return nil
	})

	return issues
}

// ValidateReferencesWithStructs validates all references using refType struct tags
func ValidateReferencesWithStructs(glxFile *lib.GLXFile) []string {
	var issues []string

	// Build lookup maps for entities
	entities := map[string]map[string]bool{
		"persons":       makeKeySet(glxFile.Persons),
		"relationships": makeKeySet(glxFile.Relationships),
		"events":        makeKeySet(glxFile.Events),
		"places":        makeKeySet(glxFile.Places),
		"sources":       makeKeySet(glxFile.Sources),
		"citations":     makeKeySet(glxFile.Citations),
		"repositories":  makeKeySet(glxFile.Repositories),
		"assertions":    makeKeySet(glxFile.Assertions),
		"media":         makeKeySet(glxFile.Media),
	}

	// Build lookup maps for vocabularies
	vocabs := map[string]map[string]bool{
		"event_types":        makeKeySet(glxFile.EventTypes),
		"relationship_types": makeKeySet(glxFile.RelationshipTypes),
		"place_types":        makeKeySet(glxFile.PlaceTypes),
		"source_types":       makeKeySet(glxFile.SourceTypes),
		"repository_types":   makeKeySet(glxFile.RepositoryTypes),
		"media_types":        makeKeySet(glxFile.MediaTypes),
		"participant_roles":  makeKeySet(glxFile.ParticipantRoles),
		"confidence_levels":  makeKeySet(glxFile.ConfidenceLevels),
		"quality_ratings":    makeKeySet(glxFile.QualityRatings),
	}

	// Validate each entity type
	issues = append(issues, validateEntityMap("persons", glxFile.Persons, entities, vocabs)...)
	issues = append(issues, validateEntityMap("relationships", glxFile.Relationships, entities, vocabs)...)
	issues = append(issues, validateEntityMap("events", glxFile.Events, entities, vocabs)...)
	issues = append(issues, validateEntityMap("places", glxFile.Places, entities, vocabs)...)
	issues = append(issues, validateEntityMap("sources", glxFile.Sources, entities, vocabs)...)
	issues = append(issues, validateEntityMap("citations", glxFile.Citations, entities, vocabs)...)
	issues = append(issues, validateEntityMap("repositories", glxFile.Repositories, entities, vocabs)...)
	issues = append(issues, validateEntityMap("assertions", glxFile.Assertions, entities, vocabs)...)
	issues = append(issues, validateEntityMap("media", glxFile.Media, entities, vocabs)...)

	return issues
}

func makeKeySet[T any](m map[string]*T) map[string]bool {
	result := make(map[string]bool)
	for k := range m {
		result[k] = true
	}
	return result
}

func validateEntityMap[T any](entityType string, entities map[string]*T, allEntities, allVocabs map[string]map[string]bool) []string {
	var issues []string
	for id, entity := range entities {
		if entity == nil {
			continue
		}
		issues = append(issues, validateEntityReferences(entityType, id, entity, allEntities, allVocabs)...)
	}
	return issues
}

func validateEntityReferences(entityType, entityID string, entity interface{}, allEntities, allVocabs map[string]map[string]bool) []string {
	var issues []string

	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return issues
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return issues
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get refType tag
		refType := field.Tag.Get("refType")
		if refType == "" {
			// No refType tag, but check if it's a nested struct or slice that might contain references
			if fieldValue.Kind() == reflect.Struct || (fieldValue.Kind() == reflect.Ptr && fieldValue.Elem().Kind() == reflect.Struct) {
				// Recurse into nested structs
				var nestedEntity interface{}
				if fieldValue.Kind() == reflect.Ptr {
					if fieldValue.IsNil() {
						continue
					}
					nestedEntity = fieldValue.Interface()
				} else {
					nestedEntity = fieldValue.Addr().Interface()
				}
				issues = append(issues, validateEntityReferences(entityType, entityID, nestedEntity, allEntities, allVocabs)...)
			} else if fieldValue.Kind() == reflect.Slice {
				// Recurse into slice elements
				for j := 0; j < fieldValue.Len(); j++ {
					elem := fieldValue.Index(j)
					if elem.Kind() == reflect.Struct || (elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct) {
						var nestedEntity interface{}
						if elem.Kind() == reflect.Ptr {
							if elem.IsNil() {
								continue
							}
							nestedEntity = elem.Interface()
						} else {
							nestedEntity = elem.Addr().Interface()
						}
						issues = append(issues, validateEntityReferences(entityType, entityID, nestedEntity, allEntities, allVocabs)...)
					}
				}
			}
			continue
		}

		// Parse comma-delimited refType values
		refTypes := strings.Split(refType, ",")
		for i := range refTypes {
			refTypes[i] = strings.TrimSpace(refTypes[i])
		}

		// Validate based on field type
		switch fieldValue.Kind() {
		case reflect.String:
			if fieldValue.String() == "" {
				continue
			}
			refValue := fieldValue.String()
			// Check if it's a vocabulary reference or entity reference
			found := false
			for _, rt := range refTypes {
				if allVocabs[rt] != nil && allVocabs[rt][refValue] {
					found = true
					break
				}
				if allEntities[rt] != nil && allEntities[rt][refValue] {
					found = true
					break
				}
			}
			if !found {
				issues = append(issues, fmt.Sprintf("%s[%s].%s references non-existent %s: %s", entityType, entityID, getYAMLTagName(field.Tag), strings.Join(refTypes, " or "), refValue))
			}

		case reflect.Slice:
			if fieldValue.Len() == 0 {
				continue
			}
			// For slices, validate each element
			for j := 0; j < fieldValue.Len(); j++ {
				elem := fieldValue.Index(j)
				if elem.Kind() == reflect.String {
					refValue := elem.String()
					if refValue == "" {
						continue
					}
					found := false
					for _, rt := range refTypes {
						if allVocabs[rt] != nil && allVocabs[rt][refValue] {
							found = true
							break
						}
						if allEntities[rt] != nil && allEntities[rt][refValue] {
							found = true
							break
						}
					}
					if !found {
						issues = append(issues, fmt.Sprintf("%s[%s].%s[%d] references non-existent %s: %s", entityType, entityID, getYAMLTagName(field.Tag), j, strings.Join(refTypes, " or "), refValue))
					}
				} else if elem.Kind() == reflect.Struct || (elem.Kind() == reflect.Ptr && elem.Elem().Kind() == reflect.Struct) {
					// Recurse into nested structs in slices
					var nestedEntity interface{}
					if elem.Kind() == reflect.Ptr {
						if elem.IsNil() {
							continue
						}
						nestedEntity = elem.Interface()
					} else {
						nestedEntity = elem.Addr().Interface()
					}
					issues = append(issues, validateEntityReferences(entityType, entityID, nestedEntity, allEntities, allVocabs)...)
				}
			}

		case reflect.Ptr:
			// Handle pointer types (like *int for Quality)
			if fieldValue.IsNil() {
				continue
			}
			if fieldValue.Elem().Kind() == reflect.Int {
				// Special handling for Quality (int pointer) -> quality_ratings (string map)
				intValue := fieldValue.Elem().Int()
				refValue := strconv.FormatInt(intValue, 10)
				found := false
				for _, rt := range refTypes {
					if allVocabs[rt] != nil && allVocabs[rt][refValue] {
						found = true
						break
					}
				}
				if !found {
					issues = append(issues, fmt.Sprintf("%s[%s].%s references non-existent %s: %s", entityType, entityID, getYAMLTagName(field.Tag), strings.Join(refTypes, " or "), refValue))
				}
			}
		}
	}

	return issues
}

func getYAMLTagName(tag reflect.StructTag) string {
	yamlTag := tag.Get("yaml")
	if yamlTag == "" {
		return ""
	}
	// Extract the first part before comma
	parts := strings.Split(yamlTag, ",")
	return parts[0]
}
