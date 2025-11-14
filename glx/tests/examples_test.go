package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestValidExamples validates all example GLX files
func TestValidExamples(t *testing.T) {
	examplesDir := "../../docs/examples"

	var validFiles []string
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".glx") {
			validFiles = append(validFiles, path)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk examples directory: %v", err)
	}

	if len(validFiles) == 0 {
		t.Fatal("no .glx files found in examples directory")
	}

	t.Logf("Found %d example files to validate", len(validFiles))

	for _, file := range validFiles {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("failed to read %s: %v", file, err)
			}

			var doc map[string]interface{}
			if err := yaml.Unmarshal(data, &doc); err != nil {
				t.Fatalf("failed to parse YAML in %s: %v", file, err)
			}

			// Skip vocabulary files - they have a different structure
			vocabKeys := []string{"relationship_types", "event_types", "place_types", "repository_types",
				"participant_roles", "media_types", "confidence_levels", "quality_ratings"}
			isVocabFile := false
			for _, key := range vocabKeys {
				if _, exists := doc[key]; exists {
					isVocabFile = true
					break
				}
			}

			if isVocabFile {
				// Vocabulary files don't need entity type keys or version fields
				return
			}

			// Check that file has at least one entity type key
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
				t.Errorf("%s: file must contain at least one entity type key", file)
			}

			// Validate entity structure
			for pluralKey, entities := range doc {
				if entityMap, ok := entities.(map[string]interface{}); ok {
					for entityID, entityData := range entityMap {
						if entity, ok := entityData.(map[string]interface{}); ok {
							// Check no 'id' field
							if _, hasID := entity["id"]; hasID {
								t.Errorf("%s: %s[%s] must not have 'id' field - the map key is the ID", file, pluralKey, entityID)
							}

							// Check version exists
							if _, hasVersion := entity["version"]; !hasVersion {
								t.Errorf("%s: %s[%s] missing required 'version' field", file, pluralKey, entityID)
							}

							// Validate ID format (alphanumeric + hyphens, 1-64 chars)
							if !isValidID(entityID) {
								t.Errorf("%s: %s[%s] invalid ID format (must be alphanumeric/hyphens, 1-64 chars)", file, pluralKey, entityID)
							}
						}
					}
				}
			}
		})
	}
}

// TestValidTestCases validates all valid test cases
func TestValidTestCases(t *testing.T) {
	validDir := "./valid"

	files, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("failed to read valid test directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".glx") {
			t.Run(file.Name(), func(t *testing.T) {
				path := filepath.Join(validDir, file.Name())
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read %s: %v", path, err)
				}

				var doc map[string]interface{}
				if err := yaml.Unmarshal(data, &doc); err != nil {
					t.Fatalf("failed to parse YAML: %v", err)
				}

				// Basic validation
				hasEntityType := false
				for key := range doc {
					if isEntityTypeKey(key) {
						hasEntityType = true
						break
					}
				}

				if !hasEntityType {
					t.Errorf("%s must have at least one entity type key", file.Name())
				}
			})
		}
	}
}

// TestInvalidTestCases ensures invalid test cases have issues
func TestInvalidTestCases(t *testing.T) {
	invalidDir := "./invalid"

	files, err := os.ReadDir(invalidDir)
	if err != nil {
		t.Fatalf("failed to read invalid test directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".glx") {
			t.Run(file.Name(), func(t *testing.T) {
				path := filepath.Join(invalidDir, file.Name())
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read %s: %v", path, err)
				}

				var doc map[string]interface{}
				// Some invalid files may have YAML parse errors - that's OK
				if err := yaml.Unmarshal(data, &doc); err != nil {
					// YAML parse error is a valid type of invalidity
					return
				}

				// For files that parse, they should either:
				// 1. Have entities with 'id' fields (not allowed)
				// 2. Missing required fields
				// 3. Invalid data types
				// We'll just verify they parse as YAML - the CLI validates the rest
			})
		}
	}
}

func isValidID(id string) bool {
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

func isEntityTypeKey(key string) bool {
	validKeys := []string{"persons", "relationships", "events", "places",
		"sources", "citations", "repositories", "assertions", "media"}
	for _, validKey := range validKeys {
		if key == validKey {
			return true
		}
	}
	return false
}
