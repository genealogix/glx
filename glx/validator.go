package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationIssue represents a single validation problem
type ValidationIssue struct {
	Path    string
	Message string
}

// ValidateGLXFile validates a single GLX file and returns any issues found
func ValidateGLXFile(path string, data map[string]interface{}) []string {
	typ := detectGLXType(path)
	var issues []string

	switch typ {
	case "config":
		issues = append(issues, validateStringField(data, "version")...)
		issues = append(issues, validateStringField(data, "schema")...)
	case "schema-version":
		issues = append(issues, validateStringField(data, "schema")...)
	default:
		// Only require id for actual entity types (not config/generic)
		if typ != "generic" {
			issues = append(issues, validateStringField(data, "id")...)
		}
		issues = append(issues, validateStringField(data, "version")...)

		if typ == "person" {
			ci, ok := data["concluded_identity"].(map[string]interface{})
			if ok {
				issues = append(issues, validateStringField(ci, "primary_name")...)
			}
		} else if typ == "relationship" {
			issues = append(issues, validateStringField(data, "type")...)
			persons, ok := data["persons"].([]interface{})
			if !ok {
				issues = append(issues, "persons must be an array")
			} else {
				if len(persons) < 2 {
					issues = append(issues, "persons must contain at least two entries")
				}
				for idx, entry := range persons {
					if _, ok := entry.(string); !ok {
						issues = append(issues, fmt.Sprintf("persons[%d] must be a string", idx))
					}
				}
			}
		} else if typ == "event" {
			issues = append(issues, validateStringField(data, "type")...)
		} else if typ == "place" {
			issues = append(issues, validateStringField(data, "name")...)
		} else if typ == "source" {
			issues = append(issues, validateStringField(data, "title")...)
		} else if typ == "citation" {
			issues = append(issues, validateStringField(data, "source_id")...)
		} else if typ == "repository" {
			issues = append(issues, validateStringField(data, "name")...)
		} else if typ == "assertion" {
			// Assertions require subject reference
			if _, ok := data["subject"]; !ok {
				if _, ok := data["subject_id"]; !ok {
					issues = append(issues, "assertion must have subject or subject_id")
				}
			}
			issues = append(issues, validateStringField(data, "property")...)
		} else if typ == "media" {
			// Media validation - uri is optional but title or file_path is recommended
			if _, ok := data["file_path"]; !ok {
				if _, ok := data["uri"]; !ok {
					issues = append(issues, "media should have file_path or uri")
				}
			}
		}
	}

	return issues
}

// DetectGLXType determines the entity type based on file path
func DetectGLXType(path string) string {
	return detectGLXType(path)
}

// ParseYAMLFile parses a YAML file into a map
func ParseYAMLFile(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func detectGLXType(path string) string {
	n := filepath.ToSlash(path)
	switch {
	case strings.Contains(n, "/persons/"):
		return "person"
	case strings.Contains(n, "persons/"):
		return "person"
	case strings.Contains(n, "/relationships/"):
		return "relationship"
	case strings.Contains(n, "relationships/"):
		return "relationship"
	case strings.Contains(n, "/events/"):
		return "event"
	case strings.Contains(n, "events/"):
		return "event"
	case strings.Contains(n, "/places/"):
		return "place"
	case strings.Contains(n, "places/"):
		return "place"
	case strings.Contains(n, "/sources/"):
		return "source"
	case strings.Contains(n, "sources/"):
		return "source"
	case strings.Contains(n, "/citations/"):
		return "citation"
	case strings.Contains(n, "citations/"):
		return "citation"
	case strings.Contains(n, "/repositories/"):
		return "repository"
	case strings.Contains(n, "repositories/"):
		return "repository"
	case strings.Contains(n, "/assertions/"):
		return "assertion"
	case strings.Contains(n, "assertions/"):
		return "assertion"
	case strings.Contains(n, "/media/"):
		return "media"
	case strings.Contains(n, "media/"):
		return "media"
	case strings.Contains(n, "/.glx-archive/"):
		base := filepath.Base(n)
		if base == "metadata.glx" {
			return "archive-metadata"
		}
	case strings.Contains(n, ".glx-archive/"):
		base := filepath.Base(n)
		if base == "metadata.glx" {
			return "archive-metadata"
		}
		if base == "schema-version.glx" {
			return "schema-version"
		}
	}
	return "generic"
}

func validateStringField(doc map[string]interface{}, key string) []string {
	val, ok := doc[key]
	if !ok {
		return []string{fmt.Sprintf("%s is required", key)}
	}
	str, ok := val.(string)
	if !ok {
		return []string{fmt.Sprintf("%s must be a string", key)}
	}
	if strings.TrimSpace(str) == "" {
		return []string{fmt.Sprintf("%s cannot be empty", key)}
	}
	return nil
}
