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

	// Get schema file path
	schemaPath := getSchemaPath(entityType)
	if schemaPath == "" {
		// Fallback to basic validation if schema not found
		return basicValidateEntity(entityType, entity, vocabs)
	}

	// Load and validate against JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)

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

func getSchemaPath(entityType string) string {
	schemaMap := map[string]string{
		"person":       "specification/schema/v1/person.schema.json",
		"relationship": "specification/schema/v1/relationship.schema.json",
		"event":        "specification/schema/v1/event.schema.json",
		"place":        "specification/schema/v1/place.schema.json",
		"source":       "specification/schema/v1/source.schema.json",
		"citation":     "specification/schema/v1/citation.schema.json",
		"repository":   "specification/schema/v1/repository.schema.json",
		"assertion":    "specification/schema/v1/assertion.schema.json",
		"media":        "specification/schema/v1/media.schema.json",
	}

	if schemaFile, ok := schemaMap[entityType]; ok {
		// Try to find schema relative to current directory or absolute path
		absPath, err := filepath.Abs(schemaFile)
		if err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
		// Try relative to working directory
		if _, err := os.Stat(schemaFile); err == nil {
			abs, _ := filepath.Abs(schemaFile)
			return abs
		}
	}

	return ""
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
					issues = append(issues, fmt.Sprintf("unknown relationship type '%s' - add to vocabularies/relationship-types.glx", relType))
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
					issues = append(issues, fmt.Sprintf("unknown event type '%s' - add to vocabularies/event-types.glx", eventType))
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
					issues = append(issues, fmt.Sprintf("unknown place type '%s' - add to vocabularies/place-types.glx", placeType))
				}
			}
		}
	case "source":
		if _, hasTitle := entity["title"]; !hasTitle {
			issues = append(issues, "title is required")
		}
	case "citation":
		if _, hasSource := entity["source_id"]; !hasSource {
			if _, hasSourceAlt := entity["source"]; !hasSourceAlt {
				issues = append(issues, "source_id or source is required")
			}
		}
	case "repository":
		if _, hasName := entity["name"]; !hasName {
			issues = append(issues, "name is required")
		}
		// Validate against vocabularies if available
		if repoType, ok := entity["type"].(string); ok {
			if vocabs != nil && len(vocabs.RepositoryTypes) > 0 {
				if _, exists := vocabs.RepositoryTypes[repoType]; !exists {
					issues = append(issues, fmt.Sprintf("unknown repository type '%s' - add to vocabularies/repository-types.glx", repoType))
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
					issues = append(issues, fmt.Sprintf("unknown confidence level '%s' - add to vocabularies/confidence-levels.glx", confidence))
				}
			}
		}
		if quality, ok := entity["quality"].(float64); ok {
			qualityStr := fmt.Sprintf("%.0f", quality)
			if vocabs != nil && len(vocabs.QualityRatings) > 0 {
				if _, exists := vocabs.QualityRatings[qualityStr]; !exists {
					issues = append(issues, fmt.Sprintf("unknown quality rating '%s' - add to vocabularies/quality-ratings.glx", qualityStr))
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
					issues = append(issues, fmt.Sprintf("unknown media type '%s' - add to vocabularies/media-types.glx", mediaType))
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

	// Get schema path for this vocabulary type
	schemaPath := getVocabSchemaPath(vocabType)
	if schemaPath == "" {
		return []string{fmt.Sprintf("no schema found for vocabulary type: %s", vocabType)}
	}

	// Validate against JSON schema
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
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

func getVocabSchemaPath(vocabType string) string {
	schemaMap := map[string]string{
		"relationship_types": "specification/schema/v1/vocabularies/relationship-types.schema.json",
		"event_types":        "specification/schema/v1/vocabularies/event-types.schema.json",
		"place_types":        "specification/schema/v1/vocabularies/place-types.schema.json",
		"repository_types":   "specification/schema/v1/vocabularies/repository-types.schema.json",
		"participant_roles":  "specification/schema/v1/vocabularies/participant-roles.schema.json",
		"media_types":        "specification/schema/v1/vocabularies/media-types.schema.json",
		"confidence_levels":  "specification/schema/v1/vocabularies/confidence-levels.schema.json",
		"quality_ratings":    "specification/schema/v1/vocabularies/quality-ratings.schema.json",
	}

	if schemaFile, ok := schemaMap[vocabType]; ok {
		absPath, err := filepath.Abs(schemaFile)
		if err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath
			}
		}
	}

	return ""
}

type ArchiveVocabularies struct {
	RelationshipTypes map[string]VocabEntry
	EventTypes        map[string]VocabEntry
	PlaceTypes        map[string]VocabEntry
	RepositoryTypes   map[string]VocabEntry
	ParticipantRoles  map[string]VocabEntry
	MediaTypes        map[string]VocabEntry
	ConfidenceLevels  map[string]VocabEntry
	QualityRatings    map[string]VocabEntry
}

type VocabEntry struct {
	Label       string
	Description string
	GEDCOM      string
	Category    string
	MimeType    string
	Custom      bool
}

func LoadArchiveVocabularies(rootPath string) (*ArchiveVocabularies, error) {
	vocabs := &ArchiveVocabularies{
		RelationshipTypes: make(map[string]VocabEntry),
		EventTypes:        make(map[string]VocabEntry),
		PlaceTypes:        make(map[string]VocabEntry),
		RepositoryTypes:   make(map[string]VocabEntry),
		ParticipantRoles:  make(map[string]VocabEntry),
		MediaTypes:        make(map[string]VocabEntry),
		ConfidenceLevels:  make(map[string]VocabEntry),
		QualityRatings:    make(map[string]VocabEntry),
	}

	// Load vocabulary files from vocabularies/ directory
	vocabDir := filepath.Join(rootPath, "vocabularies")
	if _, err := os.Stat(vocabDir); os.IsNotExist(err) {
		// No vocabularies directory - use permissive mode
		return vocabs, nil
	}

	// Load each vocabulary file
	vocabFiles := map[string]func(map[string]interface{}) error{
		"relationship-types.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "relationship_types", &vocabs.RelationshipTypes)
		},
		"event-types.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "event_types", &vocabs.EventTypes)
		},
		"place-types.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "place_types", &vocabs.PlaceTypes)
		},
		"repository-types.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "repository_types", &vocabs.RepositoryTypes)
		},
		"participant-roles.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "participant_roles", &vocabs.ParticipantRoles)
		},
		"media-types.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "media_types", &vocabs.MediaTypes)
		},
		"confidence-levels.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "confidence_levels", &vocabs.ConfidenceLevels)
		},
		"quality-ratings.glx": func(data map[string]interface{}) error {
			return loadVocabData(data, "quality_ratings", &vocabs.QualityRatings)
		},
	}

	for file, loader := range vocabFiles {
		path := filepath.Join(vocabDir, file)
		if data, err := os.ReadFile(path); err == nil {
			var doc map[string]interface{}
			if err := yaml.Unmarshal(data, &doc); err == nil {
				if err := loader(doc); err != nil {
					// Log error but continue
					continue
				}
			}
		}
	}

	return vocabs, nil
}

func loadVocabData(data map[string]interface{}, key string, target *map[string]VocabEntry) error {
	if entities, ok := data[key].(map[string]interface{}); ok {
		for entityID, entityData := range entities {
			if entity, ok := entityData.(map[string]interface{}); ok {
				entry := VocabEntry{
					Label:       getString(entity, "label"),
					Description: getString(entity, "description"),
					GEDCOM:      getString(entity, "gedcom"),
					Category:    getString(entity, "category"),
					MimeType:    getString(entity, "mime_type"),
					Custom:      getBool(entity, "custom"),
				}
				(*target)[entityID] = entry
			}
		}
	}
	return nil
}

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
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
					if placeID, ok := event["place_id"].(string); ok {
						if !allEntities["places"][placeID] {
							issues = append(issues, fmt.Sprintf("%s: events[%s].place_id references non-existent place: %s", path, eventID, placeID))
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
								if personID, ok := participant["person_id"].(string); ok {
									if !allEntities["persons"][personID] {
										issues = append(issues, fmt.Sprintf("%s: events[%s].participants[%d].person_id references non-existent person: %s", path, eventID, i, personID))
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
					if parentID, ok := place["parent_id"].(string); ok {
						if !allEntities["places"][parentID] {
							issues = append(issues, fmt.Sprintf("%s: places[%s].parent_id references non-existent place: %s", path, placeID, parentID))
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
					if sourceID, ok := citation["source_id"].(string); ok {
						if !allEntities["sources"][sourceID] {
							issues = append(issues, fmt.Sprintf("%s: citations[%s].source_id references non-existent source: %s", path, citationID, sourceID))
						}
					}
					// Check repository references
					if repoID, ok := citation["repository"].(string); ok {
						if !allEntities["repositories"][repoID] {
							issues = append(issues, fmt.Sprintf("%s: citations[%s].repository references non-existent repository: %s", path, citationID, repoID))
						}
					}
					if repoID, ok := citation["repository_id"].(string); ok {
						if !allEntities["repositories"][repoID] {
							issues = append(issues, fmt.Sprintf("%s: citations[%s].repository_id references non-existent repository: %s", path, citationID, repoID))
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
					if repoID, ok := source["repository_id"].(string); ok {
						if !allEntities["repositories"][repoID] {
							issues = append(issues, fmt.Sprintf("%s: sources[%s].repository_id references non-existent repository: %s", path, sourceID, repoID))
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
