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
func ValidateGLXFile(path string, doc map[string]interface{}) []string {
	var issues []string

	// Check for at least one entity type key
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
					entityIssues := validateEntityByType(singularType, entityMap)
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

func validateEntityByType(entityType string, entity map[string]interface{}) []string {
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
		return basicValidateEntity(entityType, entity)
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
		return basicValidateEntity(entityType, entity)
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
		"person":       "schema/v1/person.schema.json",
		"relationship": "schema/v1/relationship.schema.json",
		"event":        "schema/v1/event.schema.json",
		"place":        "schema/v1/place.schema.json",
		"source":       "schema/v1/source.schema.json",
		"citation":     "schema/v1/citation.schema.json",
		"repository":   "schema/v1/repository.schema.json",
		"assertion":    "schema/v1/assertion.schema.json",
		"media":        "schema/v1/media.schema.json",
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

func basicValidateEntity(entityType string, entity map[string]interface{}) []string {
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
	case "event":
		if _, hasType := entity["type"]; !hasType {
			issues = append(issues, "type is required")
		}
	case "place":
		if _, hasName := entity["name"]; !hasName {
			issues = append(issues, "name is required")
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
	case "media":
		if _, hasURI := entity["uri"]; !hasURI {
			if _, hasFilePath := entity["file_path"]; !hasFilePath {
				issues = append(issues, "uri or file_path is required")
			}
		}
	}

	return issues
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
