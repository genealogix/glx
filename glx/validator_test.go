package main

import (
	"testing"
)

func TestValidEntityID(t *testing.T) {
	tests := []struct {
		id         string
		entityType string
		valid      bool
	}{
		{"person-12345678", "person", true},
		{"person-abcdef12", "person", true},
		{"person-12345", "person", false},     // too short
		{"person-123456789", "person", false}, // too long
		{"person-ABCDEF12", "person", false},  // uppercase
		{"event-a1b2c3d4", "event", true},
		{"place-12ab34cd", "place", true},
		{"rel-xyz12345", "relationship", false}, // 'xyz' not hex
		{"source-12345678", "source", true},
		{"wrong-12345678", "person", false}, // wrong prefix
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := isValidEntityID(tt.id, tt.entityType)
			if got != tt.valid {
				t.Errorf("isValidEntityID(%s, %s) = %v, want %v", tt.id, tt.entityType, got, tt.valid)
			}
		})
	}
}

func TestValidateGLXFile_Person_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"persons": map[string]interface{}{
			"person-abc12345": map[string]interface{}{
				"version": "1.0",
				"concluded_identity": map[string]interface{}{
					"primary_name": "John Doe",
				},
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Person_WithIDField(t *testing.T) {
	doc := map[string]interface{}{
		"persons": map[string]interface{}{
			"person-abc12345": map[string]interface{}{
				"id":      "person-abc12345", // Should be rejected
				"version": "1.0",
				"concluded_identity": map[string]interface{}{
					"primary_name": "John Doe",
				},
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) == 0 {
		t.Error("expected validation issues for entity with 'id' field, got none")
	}
}

func TestValidateGLXFile_NoEntityTypeKeys(t *testing.T) {
	doc := map[string]interface{}{
		"something": "invalid",
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) == 0 {
		t.Error("expected validation issues for file without entity type keys, got none")
	}
}

func TestValidateGLXFile_Event_MissingType(t *testing.T) {
	doc := map[string]interface{}{
		"events": map[string]interface{}{
			"event-12345678": map[string]interface{}{
				"version": "1.0",
				// missing 'type' field
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) == 0 {
		t.Error("expected validation issues for event without type, got none")
	}
}

func TestValidateGLXFile_Place_MissingName(t *testing.T) {
	doc := map[string]interface{}{
		"places": map[string]interface{}{
			"place-12345678": map[string]interface{}{
				"version": "1.0",
				// missing 'name' field
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) == 0 {
		t.Error("expected validation issues for place without name, got none")
	}
}

func TestValidateGLXFile_MultipleEntityTypes(t *testing.T) {
	doc := map[string]interface{}{
		"persons": map[string]interface{}{
			"person-a1b2c3d4": map[string]interface{}{
				"version": "1.0",
				"concluded_identity": map[string]interface{}{
					"primary_name": "John Smith",
				},
			},
		},
		"places": map[string]interface{}{
			"place-12345678": map[string]interface{}{
				"version": "1.0",
				"name":    "Leeds",
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues for file with multiple entity types, got %v", issues)
	}
}

func TestParseYAMLFile_Valid(t *testing.T) {
	yaml := []byte("persons:\n  person-12345678:\n    version: \"1.0\"")
	doc, err := ParseYAMLFile(yaml)
	if err != nil {
		t.Errorf("ParseYAMLFile() error = %v", err)
	}
	if doc == nil {
		t.Error("ParseYAMLFile() returned nil")
	}
}

func TestParseYAMLFile_Invalid(t *testing.T) {
	yaml := []byte("invalid: yaml: syntax")
	_, err := ParseYAMLFile(yaml)
	if err == nil {
		t.Error("ParseYAMLFile() expected error for invalid YAML, got none")
	}
}
