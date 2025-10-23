package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectGLXType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"persons/person-123.glx", "person"},
		{"relationships/rel-123.glx", "relationship"},
		{"events/event-123.glx", "event"},
		{"places/place-123.glx", "place"},
		{"sources/source-123.glx", "source"},
		{"citations/citation-123.glx", "citation"},
		{"repositories/repo-123.glx", "repository"},
		{"assertions/assertion-123.glx", "assertion"},
		{"media/photo-123.glx", "media"},
		{".glx-archive/metadata.glx", "archive-metadata"},
		{"unknown/file.glx", "generic"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := DetectGLXType(tt.path)
			if result != tt.expected {
				t.Errorf("DetectGLXType(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestValidateGLXFile_Person_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "person-abc123",
		"version": "1.0",
		"concluded_identity": map[string]interface{}{
			"primary_name": "John Doe",
		},
	}

	issues := ValidateGLXFile("persons/person-abc123.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Person_MissingID(t *testing.T) {
	doc := map[string]interface{}{
		"version": "1.0",
		"concluded_identity": map[string]interface{}{
			"primary_name": "John Doe",
		},
	}

	issues := ValidateGLXFile("persons/person-test.glx", doc)
	if len(issues) == 0 {
		t.Error("expected validation issues for missing id")
	}
	if !contains(issues, "id is required") {
		t.Errorf("expected 'id is required', got %v", issues)
	}
}

func TestValidateGLXFile_Person_MissingConcludedIdentity(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "person-abc123",
		"version": "1.0",
	}

	issues := ValidateGLXFile("persons/person-test.glx", doc)
	// concluded_identity is now optional, so this should pass
	if len(issues) > 0 {
		t.Errorf("expected no issues for optional concluded_identity, got %v", issues)
	}
}

func TestValidateGLXFile_Relationship_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "rel-123",
		"version": "1.0",
		"type":    "parent",
		"persons": []interface{}{"person-1", "person-2"},
	}

	issues := ValidateGLXFile("relationships/rel-123.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Relationship_MissingPersons(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "rel-123",
		"version": "1.0",
		"type":    "parent",
	}

	issues := ValidateGLXFile("relationships/rel-123.glx", doc)
	if !contains(issues, "persons must be an array") {
		t.Errorf("expected persons array error, got %v", issues)
	}
}

func TestValidateGLXFile_Event_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "event-birth-001",
		"version": "1.0",
		"type":    "birth",
	}

	issues := ValidateGLXFile("events/event-birth-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Event_MissingType(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "event-001",
		"version": "1.0",
	}

	issues := ValidateGLXFile("events/event-001.glx", doc)
	if !contains(issues, "type is required") {
		t.Errorf("expected type required error, got %v", issues)
	}
}

func TestValidateGLXFile_Place_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "place-leeds",
		"version": "1.0",
		"name":    "Leeds",
	}

	issues := ValidateGLXFile("places/place-leeds.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Place_MissingName(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "place-001",
		"version": "1.0",
	}

	issues := ValidateGLXFile("places/place-001.glx", doc)
	if !contains(issues, "name is required") {
		t.Errorf("expected name required error, got %v", issues)
	}
}

func TestValidateGLXFile_Citation_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":        "citation-001",
		"version":   "1.0",
		"source_id": "source-register",
	}

	issues := ValidateGLXFile("citations/citation-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Citation_MissingSource(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "citation-001",
		"version": "1.0",
	}

	issues := ValidateGLXFile("citations/citation-001.glx", doc)
	if !contains(issues, "source_id is required") {
		t.Errorf("expected source_id required error, got %v", issues)
	}
}

func TestValidateGLXFile_Repository_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "repo-leeds",
		"version": "1.0",
		"name":    "Leeds Library",
	}

	issues := ValidateGLXFile("repositories/repo-leeds.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Repository_MissingName(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "repo-001",
		"version": "1.0",
	}

	issues := ValidateGLXFile("repositories/repo-001.glx", doc)
	if !contains(issues, "name is required") {
		t.Errorf("expected name required error, got %v", issues)
	}
}

func TestValidateGLXFile_Assertion_WithSubjectID(t *testing.T) {
	doc := map[string]interface{}{
		"id":         "assertion-001",
		"version":    "1.0",
		"subject_id": "person-abc",
		"property":   "birth_date",
	}

	issues := ValidateGLXFile("assertions/assertion-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Assertion_MissingProperty(t *testing.T) {
	doc := map[string]interface{}{
		"id":         "assertion-001",
		"version":    "1.0",
		"subject_id": "person-abc",
	}

	issues := ValidateGLXFile("assertions/assertion-001.glx", doc)
	if !contains(issues, "property is required") {
		t.Errorf("expected property required error, got %v", issues)
	}
}

func TestValidateGLXFile_Assertion_MissingSubject(t *testing.T) {
	doc := map[string]interface{}{
		"id":       "assertion-001",
		"version":  "1.0",
		"property": "birth_date",
	}

	issues := ValidateGLXFile("assertions/assertion-001.glx", doc)
	if !contains(issues, "assertion must have subject or subject_id") {
		t.Errorf("expected subject error, got %v", issues)
	}
}

func TestValidateGLXFile_Source_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "source-001",
		"version": "1.0",
		"title":   "Birth Register",
	}

	issues := ValidateGLXFile("sources/source-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Media_WithFilePath(t *testing.T) {
	doc := map[string]interface{}{
		"id":        "media-001",
		"version":   "1.0",
		"file_path": "photos/john.jpg",
	}

	issues := ValidateGLXFile("media/media-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Media_WithURI(t *testing.T) {
	doc := map[string]interface{}{
		"id":      "media-001",
		"version": "1.0",
		"uri":     "https://example.com/photo.jpg",
	}

	issues := ValidateGLXFile("media/media-001.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidateGLXFile_Config_Minimal(t *testing.T) {
	doc := map[string]interface{}{
		"version": "1.0",
		"schema":  "v1",
	}

	issues := ValidateGLXFile(".glx-archive/metadata.glx", doc)
	if len(issues) > 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestParseYAMLFile_Valid(t *testing.T) {
	yaml := []byte(`
id: test-123
version: "1.0"
name: Test
`)

	doc, err := ParseYAMLFile(yaml)
	if err != nil {
		t.Fatalf("ParseYAMLFile failed: %v", err)
	}

	if doc["id"] != "test-123" {
		t.Errorf("expected id=test-123, got %v", doc["id"])
	}
}

func TestParseYAMLFile_Invalid(t *testing.T) {
	yaml := []byte(`
invalid: [yaml: content:
`)

	_, err := ParseYAMLFile(yaml)
	if err == nil {
		t.Error("expected ParseYAMLFile to fail on invalid YAML")
	}
}

func TestValidateExampleFiles(t *testing.T) {
	examples := []struct {
		path       string
		shouldPass bool
	}{
		{"../test-suite/valid/event-minimal.glx", true},
		{"../test-suite/valid/place-minimal.glx", true},
		{"../test-suite/valid/citation-minimal.glx", true},
		{"../test-suite/valid/repository-minimal.glx", true},
		{"../test-suite/valid/assertion-minimal.glx", true},
		{"../test-suite/valid/person-minimal.glx", true},
		{"../test-suite/invalid/event-missing-type.glx", false},
		{"../test-suite/invalid/place-missing-name.glx", false},
		{"../test-suite/invalid/citation-missing-source.glx", false},
		{"../test-suite/invalid/repository-missing-name.glx", false},
		{"../test-suite/invalid/assertion-missing-property.glx", false},
	}

	for _, example := range examples {
		t.Run(example.path, func(t *testing.T) {
			// Check if file exists first
			if _, err := os.Stat(example.path); err != nil {
				t.Skipf("test file not found: %s", example.path)
			}

			data, err := os.ReadFile(example.path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", example.path, err)
			}

			doc, err := ParseYAMLFile(data)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", example.path, err)
			}

			issues := ValidateGLXFile(example.path, doc)
			hasIssues := len(issues) > 0

			if example.shouldPass && hasIssues {
				t.Errorf("expected %s to pass validation, got issues: %v", example.path, issues)
			}
			if !example.shouldPass && !hasIssues {
				t.Errorf("expected %s to fail validation, but it passed", example.path)
			}
		})
	}
}

func TestValidateCompleteFamily(t *testing.T) {
	baseDir := filepath.Join("..", "examples", "complete-family")

	entries := []struct {
		glob string
	}{
		{"persons/*.glx"},
		{"events/*.glx"},
		{"places/*.glx"},
		{"repositories/*.glx"},
		{"citations/*.glx"},
	}

	for _, entry := range entries {
		pattern := filepath.Join(baseDir, entry.glob)
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatalf("glob failed for %s: %v", pattern, err)
		}

		if len(matches) == 0 {
			t.Logf("no files matched %s", pattern)
			continue
		}

		for _, path := range matches {
			t.Run(path, func(t *testing.T) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read %s: %v", path, err)
				}

				doc, err := ParseYAMLFile(data)
				if err != nil {
					t.Fatalf("failed to parse %s: %v", path, err)
				}

				issues := ValidateGLXFile(path, doc)
				if len(issues) > 0 {
					t.Errorf("%s: %v", path, issues)
				}
			})
		}
	}
}

// Helper function
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
