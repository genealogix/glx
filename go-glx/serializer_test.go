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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNewSerializer(t *testing.T) {
	// Test with nil options (should use defaults)
	s := NewSerializer(nil)
	if s == nil {
		t.Fatal("NewSerializer returned nil")
	}
	if s.Options == nil {
		t.Fatal("Serializer options are nil")
	}
	if !s.Options.Validate {
		t.Error("Default options should validate")
	}

	// Test with custom options
	opts := &SerializerOptions{
		Validate: false,
		Pretty:   false,
	}
	s = NewSerializer(opts)
	if s.Options.Validate {
		t.Error("Custom options not applied: Validate")
	}
}

func TestSerializeSingleFileBytes(t *testing.T) {
	// Create a minimal GLX file
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Doe",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Doe",
						},
					},
				},
			},
		},
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	// Serialize
	s := NewSerializer(nil)
	yamlBytes, err := s.SerializeSingleFileBytes(glx)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Error("Serialized bytes are empty")
	}

	// Verify output is valid YAML by parsing it back
	var parsed GLXFile
	if err := yaml.Unmarshal(yamlBytes, &parsed); err != nil {
		t.Fatalf("Serialized output is not valid YAML: %v", err)
	}

	// Verify the parsed structure matches the original
	if len(parsed.Persons) != 1 {
		t.Errorf("Expected 1 person, got %d", len(parsed.Persons))
	}

	person, ok := parsed.Persons["person-001"]
	if !ok {
		t.Fatal("Person person-001 not found in parsed output")
	}

	// Verify the name property was preserved
	given, surname := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected given name 'John', got %q", given)
	}
	if surname != "Doe" {
		t.Errorf("Expected surname 'Doe', got %q", surname)
	}
}

func TestSerializeSingleFile(t *testing.T) {
	// Create a minimal GLX file
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Doe",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Doe",
						},
					},
				},
			},
		},
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	// Create temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.glx")

	// Serialize to bytes
	s := NewSerializer(nil)
	yamlBytes, err := s.SerializeSingleFileBytes(glx)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Write to file (test does I/O, not lib)
	if err := os.WriteFile(outputPath, yamlBytes, 0o644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Check file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("Output file not created")
	}

	// Read and parse the output file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Verify output is valid YAML by parsing it
	var parsed GLXFile
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Output file is not valid YAML: %v", err)
	}

	// Verify the structure
	if len(parsed.Persons) != 1 {
		t.Errorf("Expected 1 person in file, got %d", len(parsed.Persons))
	}

	person, ok := parsed.Persons["person-001"]
	if !ok {
		t.Fatal("Person person-001 not found in file")
	}

	given, _ := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected given name 'John', got %q", given)
	}
}

func TestDeserializeSingleFileBytes(t *testing.T) {
	// Create YAML data
	yamlData := `persons:
  person-001:
    properties:
      name:
        value: John Doe
        fields:
          given: John
          surname: Doe
events: {}
relationships: {}
places: {}
sources: {}
citations: {}
repositories: {}
media: {}
assertions: {}
`

	// Load
	s := NewSerializer(nil)
	glx, err := s.DeserializeSingleFileBytes([]byte(yamlData))
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Check content
	if len(glx.Persons) != 1 {
		t.Errorf("Expected 1 person, got %d", len(glx.Persons))
	}

	person, ok := glx.Persons["person-001"]
	if !ok {
		t.Fatal("Person person-001 not found")
	}

	given, surname := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected name.fields.given=John, got %s", given)
	}
	if surname != "Doe" {
		t.Errorf("Expected name.fields.surname=Doe, got %s", surname)
	}

	// Verify all entity maps are initialized (not nil)
	if glx.Events == nil {
		t.Error("Events map should not be nil")
	}
	if glx.Relationships == nil {
		t.Error("Relationships map should not be nil")
	}
	if glx.Places == nil {
		t.Error("Places map should not be nil")
	}
}

func TestLoadSingleFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.glx")

	// Create YAML file
	yamlData := `persons:
  person-001:
    properties:
      name:
        value: John Doe
        fields:
          given: John
          surname: Doe
events: {}
relationships: {}
places: {}
sources: {}
citations: {}
repositories: {}
media: {}
assertions: {}
`
	if err := os.WriteFile(inputPath, []byte(yamlData), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Read file (test does I/O, not lib)
	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Load
	s := NewSerializer(nil)
	glx, err := s.DeserializeSingleFileBytes(data)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Check content
	if len(glx.Persons) != 1 {
		t.Errorf("Expected 1 person, got %d", len(glx.Persons))
	}
}

func TestSerializeMultiFile(t *testing.T) {
	// Create a minimal GLX file
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Doe",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Doe",
						},
					},
				},
			},
			"person-002": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "Jane Smith",
						"fields": map[string]any{
							"given":   "Jane",
							"surname": "Smith",
						},
					},
				},
			},
		},
		Events: map[string]*Event{
			"event-001": {
				Type: "birth",
			},
		},
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	// Create temp directory
	tmpDir := t.TempDir()

	// Serialize (disable validation for unit test - we're testing serialization, not data validity)
	s := NewSerializer(&SerializerOptions{
		Validate: false, // Disable validation for unit test
		Pretty:   true,
	})
	files, err := s.SerializeMultiFileToMap(glx)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Write files (test does I/O, not lib)
	for relPath, content := range files {
		absPath := filepath.Join(tmpDir, relPath)
		parentDir := filepath.Dir(absPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", parentDir, err)
		}
		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", absPath, err)
		}
	}

	// Check directories exist
	personsDir := filepath.Join(tmpDir, "persons")
	if _, err := os.Stat(personsDir); os.IsNotExist(err) {
		t.Error("Persons directory not created")
	}

	eventsDir := filepath.Join(tmpDir, "events")
	if _, err := os.Stat(eventsDir); os.IsNotExist(err) {
		t.Error("Events directory not created")
	}

	vocabDir := filepath.Join(tmpDir, "vocabularies")
	if _, err := os.Stat(vocabDir); os.IsNotExist(err) {
		t.Error("Vocabularies directory not created")
	}

	// Check deterministic filenames exist in the map
	personFile1 := filepath.Join("persons", "person-001.glx")
	personFile2 := filepath.Join("persons", "person-002.glx")
	eventFile1 := filepath.Join("events", "event-001.glx")

	if _, ok := files[personFile1]; !ok {
		t.Errorf("Expected file %q in map, got keys: %v", personFile1, mapKeys(files))
	}
	if _, ok := files[personFile2]; !ok {
		t.Errorf("Expected file %q in map, got keys: %v", personFile2, mapKeys(files))
	}
	if _, ok := files[eventFile1]; !ok {
		t.Errorf("Expected file %q in map, got keys: %v", eventFile1, mapKeys(files))
	}

	// Verify person file content
	personData := files[personFile1]
	var parsedPerson GLXFile
	if err := yaml.Unmarshal(personData, &parsedPerson); err != nil {
		t.Fatalf("Person file is not valid YAML: %v", err)
	}
	if len(parsedPerson.Persons) != 1 {
		t.Errorf("Expected 1 person in file, got %d", len(parsedPerson.Persons))
	}
	if p, ok := parsedPerson.Persons["person-001"]; !ok {
		t.Error("Expected person-001 in parsed file")
	} else if p.Properties == nil {
		t.Error("Person properties should not be nil")
	}

	// Verify event file content
	eventData := files[eventFile1]
	var parsedEvent GLXFile
	if err := yaml.Unmarshal(eventData, &parsedEvent); err != nil {
		t.Fatalf("Event file is not valid YAML: %v", err)
	}
	if len(parsedEvent.Events) != 1 {
		t.Errorf("Expected 1 event in file, got %d", len(parsedEvent.Events))
	}
	if e, ok := parsedEvent.Events["event-001"]; !ok {
		t.Error("Expected event-001 in parsed file")
	} else if e.Type != "birth" {
		t.Errorf("Expected event type 'birth', got %q", e.Type)
	}
}

// mapKeys returns sorted keys from a map for stable test output.
func mapKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func TestSerializeMultiFileDeterministic(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {Properties: map[string]any{"primary_name": "Test"}},
			"person-002": {Properties: map[string]any{"primary_name": "Test2"}},
		},
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	s := NewSerializer(&SerializerOptions{Validate: false})

	files1, err := s.SerializeMultiFileToMap(glx)
	if err != nil {
		t.Fatalf("First serialization failed: %v", err)
	}

	files2, err := s.SerializeMultiFileToMap(glx)
	if err != nil {
		t.Fatalf("Second serialization failed: %v", err)
	}

	if len(files1) != len(files2) {
		t.Fatalf("File count mismatch: %d vs %d", len(files1), len(files2))
	}
	for key, content1 := range files1 {
		content2, ok := files2[key]
		if !ok {
			t.Errorf("Key %q present in first but not second serialization", key)

			continue
		}
		if !bytes.Equal(content1, content2) {
			t.Errorf("Content for %q differs between serializations:\n--- first ---\n%s\n--- second ---\n%s", key, content1, content2)
		}
	}
}

func TestSerializeMultiFileCaseCollision(t *testing.T) {
	tests := []struct {
		name    string
		persons map[string]*Person
		events  map[string]*Event
	}{
		{
			name: "two-way collision in same type",
			persons: map[string]*Person{
				"Person-A": {Properties: map[string]any{"primary_name": "Alice"}},
				"person-a": {Properties: map[string]any{"primary_name": "Bob"}},
			},
		},
		{
			name: "three-way collision in same type",
			persons: map[string]*Person{
				"PERSON-A": {Properties: map[string]any{"primary_name": "Alice"}},
				"Person-A": {Properties: map[string]any{"primary_name": "Bob"}},
				"person-a": {Properties: map[string]any{"primary_name": "Carol"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glx := &GLXFile{
				Persons:       tt.persons,
				Events:        tt.events,
				Relationships: make(map[string]*Relationship),
				Places:        make(map[string]*Place),
				Sources:       make(map[string]*Source),
				Citations:     make(map[string]*Citation),
				Repositories:  make(map[string]*Repository),
				Media:         make(map[string]*Media),
				Assertions:    make(map[string]*Assertion),
			}
			if glx.Persons == nil {
				glx.Persons = make(map[string]*Person)
			}
			if glx.Events == nil {
				glx.Events = make(map[string]*Event)
			}

			s := NewSerializer(&SerializerOptions{Validate: false})
			_, err := s.SerializeMultiFileToMap(glx)
			if err == nil {
				t.Fatal("Expected case-insensitive collision error, got nil")
			}
			if !errors.Is(err, ErrCaseInsensitiveCollision) {
				t.Errorf("Expected ErrCaseInsensitiveCollision, got: %v", err)
			}
		})
	}
}

// Cross-type "collisions" are not collisions at all — entities live in
// per-type subdirectories (persons/, events/), so two entities of different
// types may share an ID without conflict.
func TestSerializeMultiFileCrossTypeNoCollision(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"shared-id": {Properties: map[string]any{"primary_name": "Alice"}},
		},
		Events: map[string]*Event{
			"shared-id": {Type: "birth", Participants: []Participant{{Person: "shared-id", Role: "principal"}}},
		},
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	s := NewSerializer(&SerializerOptions{Validate: false})
	files, err := s.SerializeMultiFileToMap(glx)
	if err != nil {
		t.Fatalf("Cross-type same-ID should not collide: %v", err)
	}
	if _, ok := files["persons/shared-id.glx"]; !ok {
		t.Error("Expected persons/shared-id.glx in output")
	}
	if _, ok := files["events/shared-id.glx"]; !ok {
		t.Error("Expected events/shared-id.glx in output")
	}
}

func TestLoadMultiFile(t *testing.T) {
	// First create a multi-file archive
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Doe",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Doe",
						},
					},
				},
			},
		},
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	tmpDir := t.TempDir()
	s := NewSerializer(&SerializerOptions{
		Validate: false,
	})

	// Serialize to map
	files, err := s.SerializeMultiFileToMap(glx)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Write files (test does I/O, not lib)
	for relPath, content := range files {
		absPath := filepath.Join(tmpDir, relPath)
		parentDir := filepath.Dir(absPath)
		if err := os.MkdirAll(parentDir, 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(absPath, content, 0o644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	// Read files back (test does I/O, not lib)
	filesRead := make(map[string][]byte)
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(tmpDir, path)
		filesRead[relPath] = data

		return nil
	})
	if err != nil {
		t.Fatalf("Failed to read files: %v", err)
	}

	// Load from map
	loaded, _, err := s.DeserializeMultiFileFromMap(filesRead)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Check content
	if len(loaded.Persons) != 1 {
		t.Errorf("Expected 1 person, got %d", len(loaded.Persons))
	}

	person, ok := loaded.Persons["person-001"]
	if !ok {
		t.Fatal("Person person-001 not found")
	}

	given, surname := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected name.fields.given=John, got %s", given)
	}
	if surname != "Doe" {
		t.Errorf("Expected name.fields.surname=Doe, got %s", surname)
	}

	// Verify entity maps are initialized
	if loaded.Events == nil {
		t.Error("Events map should not be nil after loading")
	}
	if loaded.Relationships == nil {
		t.Error("Relationships map should not be nil after loading")
	}
}

func TestRoundTripSingleFile(t *testing.T) {
	// Create original GLX file
	original := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Doe",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Doe",
						},
					},
				},
			},
		},
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	s := NewSerializer(nil)

	// Serialize to bytes
	yamlBytes, err := s.SerializeSingleFileBytes(original)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Load back from bytes
	loaded, err := s.DeserializeSingleFileBytes(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Compare person counts
	if len(loaded.Persons) != len(original.Persons) {
		t.Errorf("Person count mismatch: expected %d, got %d", len(original.Persons), len(loaded.Persons))
	}

	loadedPerson := loaded.Persons["person-001"]
	if loadedPerson == nil {
		t.Fatal("person-001 not found after round-trip")
	}

	originalPerson := original.Persons["person-001"]

	loadedGiven, loadedSurname := ExtractNameFields(loadedPerson.Properties["name"])
	originalGiven, originalSurname := ExtractNameFields(originalPerson.Properties["name"])
	if loadedGiven != originalGiven {
		t.Errorf("name.fields.given mismatch: expected %q, got %q", originalGiven, loadedGiven)
	}
	if loadedSurname != originalSurname {
		t.Errorf("name.fields.surname mismatch: expected %q, got %q", originalSurname, loadedSurname)
	}

	// Verify all entity maps are preserved (even if empty)
	if len(loaded.Events) != len(original.Events) {
		t.Errorf("Events count mismatch: expected %d, got %d", len(original.Events), len(loaded.Events))
	}
	if len(loaded.Relationships) != len(original.Relationships) {
		t.Errorf("Relationships count mismatch: expected %d, got %d", len(original.Relationships), len(loaded.Relationships))
	}
}

func TestEventTitleRoundTrip(t *testing.T) {
	original := &GLXFile{
		Persons: map[string]*Person{
			"person-001": {
				Properties: map[string]any{
					"name": map[string]any{"value": "R. Webb"},
				},
			},
		},
		Events: map[string]*Event{
			"event-census-1860": {
				Title: "1860 Census — Webb Household",
				Type:  "census",
				Date:  "1860",
				Participants: []Participant{
					{Person: "person-001", Role: "subject"},
				},
			},
			"event-birth-001": {
				Type: "birth",
				Date: "1815",
				Participants: []Participant{
					{Person: "person-001", Role: "subject"},
				},
			},
		},
		Relationships: make(map[string]*Relationship),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Media:         make(map[string]*Media),
		Assertions:    make(map[string]*Assertion),
	}

	s := NewSerializer(&SerializerOptions{Validate: false})

	// Single-file roundtrip
	yamlBytes, err := s.SerializeSingleFileBytes(original)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	loaded, err := s.DeserializeSingleFileBytes(yamlBytes)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// Event with title should preserve it
	census := loaded.Events["event-census-1860"]
	if census == nil {
		t.Fatal("event-census-1860 not found after round-trip")
	}
	if census.Title != "1860 Census — Webb Household" {
		t.Errorf("Title mismatch: expected %q, got %q", "1860 Census — Webb Household", census.Title)
	}

	// Event without title should have empty string
	birth := loaded.Events["event-birth-001"]
	if birth == nil {
		t.Fatal("event-birth-001 not found after round-trip")
	}
	if birth.Title != "" {
		t.Errorf("Expected empty title for birth event, got %q", birth.Title)
	}

	// Verify title appears only for census event in serialized YAML and omitempty works
	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		t.Fatalf("Failed to unmarshal serialized YAML for inspection: %v", err)
	}

	eventsVal, ok := raw["events"]
	if !ok {
		t.Fatalf("Serialized YAML missing top-level 'events' key")
	}

	eventsMap, ok := eventsVal.(map[string]any)
	if !ok {
		t.Fatalf("Serialized 'events' value has unexpected type %T", eventsVal)
	}

	censusVal, ok := eventsMap["event-census-1860"]
	if !ok {
		t.Fatalf("Serialized YAML missing 'event-census-1860' entry")
	}
	censusMap, ok := censusVal.(map[string]any)
	if !ok {
		t.Fatalf("Serialized 'event-census-1860' has unexpected type %T", censusVal)
	}
	if _, ok := censusMap["title"]; !ok {
		t.Error("Expected 'title' field for census event in serialized YAML")
	}

	birthVal, ok := eventsMap["event-birth-001"]
	if !ok {
		t.Fatalf("Serialized YAML missing 'event-birth-001' entry")
	}
	birthMap, ok := birthVal.(map[string]any)
	if !ok {
		t.Fatalf("Serialized 'event-birth-001' has unexpected type %T", birthVal)
	}
	if _, ok := birthMap["title"]; ok {
		t.Error("Did not expect 'title' field for birth event in serialized YAML (omitempty should omit empty titles)")
	}
}

func TestMultiFileEntityFormat(t *testing.T) {
	// Verify the multi-file serializer produces standard GLX structure
	person := &Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value": "John Doe",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Doe",
				},
			},
		},
	}

	wrapper := map[string]map[string]*Person{
		"persons": {"person-001": person},
	}

	// Marshal
	yamlBytes, err := yaml.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Fatal("Marshaled YAML is empty")
	}

	// Unmarshal as GLXFile and verify round-trip
	var restored GLXFile
	if err := yaml.Unmarshal(yamlBytes, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check entity was restored correctly
	restoredPerson, ok := restored.Persons["person-001"]
	if !ok {
		t.Fatal("person-001 not found after round-trip")
	}

	given, surname := ExtractNameFields(restoredPerson.Properties["name"])
	if given != "John" {
		t.Errorf("Expected given name 'John', got %q", given)
	}
	if surname != "Doe" {
		t.Errorf("Expected surname 'Doe', got %q", surname)
	}
}
