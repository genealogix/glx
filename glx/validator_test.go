package main

import (
	"strings"
	"testing"
)

func TestValidEntityID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		// Standard format
		{"person-12345678", true},
		{"person-abcdef12", true},
		{"person-ABCDEF12", true},
		{"event-a1b2c3d4", true},
		{"place-12ab34cd", true},
		{"source-12345678", true},

		// Descriptive IDs
		{"person-john-smith", true},
		{"place-leeds-uk", true},
		{"event-birth-john-1850", true},
		{"source-parish-register", true},

		// Edge cases
		{"a", true},                      // single char OK
		{strings.Repeat("x", 64), true},  // 64 chars OK
		{strings.Repeat("x", 65), false}, // 65 chars too long
		{"", false},                      // empty not OK

		// Invalid characters
		{"person_12345", false}, // underscore not allowed
		{"person.12345", false}, // dot not allowed
		{"person 12345", false}, // space not allowed
		{"person@12345", false}, // special char not allowed
		{"person/12345", false}, // slash not allowed
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got := isValidEntityID(tt.id)
			if got != tt.valid {
				t.Errorf("isValidEntityID(%q) = %v, want %v", tt.id, got, tt.valid)
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

func TestValidateGLXFile_DescriptiveIDs(t *testing.T) {
	doc := map[string]interface{}{
		"persons": map[string]interface{}{
			"person-john-smith": map[string]interface{}{
				"version": "1.0",
				"concluded_identity": map[string]interface{}{
					"primary_name": "John Smith",
				},
			},
		},
		"places": map[string]interface{}{
			"place-leeds-yorkshire": map[string]interface{}{
				"version": "1.0",
				"name":    "Leeds",
			},
		},
	}

	issues := ValidateGLXFile("test.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues for descriptive IDs, got %v", issues)
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
	yaml := []byte("invalid: yaml: syntax: error:")
	_, err := ParseYAMLFile(yaml)
	if err == nil {
		t.Error("ParseYAMLFile() expected error for invalid YAML, got none")
	}
}
