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

package glx

import (
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestStandardVocabularies(t *testing.T) {
	vocabs := StandardVocabularies()

	// Should have vocabularies
	if len(vocabs) == 0 {
		t.Fatal("StandardVocabularies returned empty map")
	}

	// Check for expected vocabularies
	expected := []string{
		"event-types.glx",
		"relationship-types.glx",
		"place-types.glx",
		"source-types.glx",
		"repository-types.glx",
		"media-types.glx",
		"participant-roles.glx",
		"confidence-levels.glx",
		"person-properties.glx",
		"event-properties.glx",
		"relationship-properties.glx",
		"place-properties.glx",
		"gender-types.glx",
	}

	for _, name := range expected {
		content, ok := vocabs[name]
		if !ok {
			t.Errorf("Missing vocabulary: %s", name)

			continue
		}
		if len(content) == 0 {
			t.Errorf("Empty vocabulary: %s", name)
		}

		// Verify content is valid YAML
		var parsed map[string]any
		if err := yaml.Unmarshal(content, &parsed); err != nil {
			t.Errorf("Vocabulary %s is not valid YAML: %v", name, err)
		}

		// Verify vocabulary has expected structure (should contain vocabulary type data)
		if len(parsed) == 0 {
			t.Errorf("Vocabulary %s has no keys after parsing", name)
		}
	}
}

func TestListStandardVocabularies(t *testing.T) {
	names := ListStandardVocabularies()

	// Should have vocabularies
	if len(names) == 0 {
		t.Fatal("ListStandardVocabularies returned empty list")
	}

	// Names should not have .glx extension
	for _, name := range names {
		if len(name) > 4 && name[len(name)-4:] == ".glx" {
			t.Errorf("Vocabulary name should not have .glx extension: %s", name)
		}
	}

	// Verify expected vocabulary names are present
	expectedNames := []string{
		"event-types",
		"relationship-types",
		"place-types",
		"source-types",
	}

	for _, expected := range expectedNames {
		found := slices.Contains(names, expected)
		if !found {
			t.Errorf("Expected vocabulary %q not found in list", expected)
		}
	}

	// Verify each name can be retrieved with GetStandardVocabulary
	for _, name := range names {
		_, err := GetStandardVocabulary(name)
		if err != nil {
			t.Errorf("Listed vocabulary %q cannot be retrieved: %v", name, err)
		}
	}
}

func TestGetStandardVocabulary(t *testing.T) {
	tests := []struct {
		name        string
		wantErr     bool
		expectKey   string // A key we expect to find in the vocabulary
		errContains string // Expected error message substring
	}{
		{"event-types", false, "event_types", ""},
		{"relationship-types", false, "relationship_types", ""},
		{"place-types", false, "place_types", ""},
		{"source-types", false, "source_types", ""},
		{"nonexistent-vocab", true, "", "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := GetStandardVocabulary(tt.name)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)

				return
			}

			if len(content) == 0 {
				t.Errorf("Empty content for %s", tt.name)
			}

			// Verify content is valid YAML
			var parsed map[string]any
			if err := yaml.Unmarshal(content, &parsed); err != nil {
				t.Errorf("Vocabulary %s content is not valid YAML: %v", tt.name, err)

				return
			}

			// Verify expected key is present
			if tt.expectKey != "" {
				if _, ok := parsed[tt.expectKey]; !ok {
					t.Errorf("Vocabulary %s missing expected key %q, found keys: %v", tt.name, tt.expectKey, keysOf(parsed))
				}
			}
		})
	}
}

// keysOf returns the keys of a map for error messages
func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func TestLoadStandardVocabulariesIntoGLX_ClonedMaps(t *testing.T) {
	// Regression test: verify that mutating one GLXFile's vocabulary map
	// does not affect another GLXFile that loaded the same vocabularies.
	glx1 := &GLXFile{}
	glx2 := &GLXFile{}

	if err := LoadStandardVocabulariesIntoGLX(glx1); err != nil {
		t.Fatalf("LoadStandardVocabulariesIntoGLX(glx1): %v", err)
	}
	if err := LoadStandardVocabulariesIntoGLX(glx2); err != nil {
		t.Fatalf("LoadStandardVocabulariesIntoGLX(glx2): %v", err)
	}

	if len(glx1.EventTypes) != len(glx2.EventTypes) {
		t.Fatalf("expected same EventTypes length, got %d vs %d", len(glx1.EventTypes), len(glx2.EventTypes))
	}

	// Add a key to glx1
	glx1.EventTypes["test-mutation"] = &EventType{Label: "Test"}

	// glx2 should NOT have the new key
	if _, exists := glx2.EventTypes["test-mutation"]; exists {
		t.Error("mutating glx1's EventTypes should not affect glx2")
	}

	// Delete a key from glx2
	var firstKey string
	for k := range glx2.PlaceTypes {
		firstKey = k

		break
	}
	delete(glx2.PlaceTypes, firstKey)

	// glx1 should still have the deleted key
	if _, exists := glx1.PlaceTypes[firstKey]; !exists {
		t.Errorf("deleting from glx2's PlaceTypes should not affect glx1 (missing key %q)", firstKey)
	}
}

// These tests were removed because WriteStandardVocabularies and WriteVocabulariesToFile
// were removed from lib (they violated the no-I/O rule). Vocabulary writing is now
// handled by the CLI commands, and vocabulary serialization is tested in roundtrip tests.
