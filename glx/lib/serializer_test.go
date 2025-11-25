package lib

import (
	"os"
	"path/filepath"
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

	// Check that YAML contains expected content
	yamlStr := string(yamlBytes)
	if !contains(yamlStr, "person-001") {
		t.Error("YAML doesn't contain person-001")
	}
	if !contains(yamlStr, "John") {
		t.Error("YAML doesn't contain John")
	}

	t.Logf("Serialized %d bytes", len(yamlBytes))
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

	// Read and check content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	yamlStr := string(data)
	if !contains(yamlStr, "person-001") {
		t.Error("YAML doesn't contain person-001")
	}

	t.Logf("Wrote %d bytes to %s", len(data), outputPath)
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

	given, _ := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected name.fields.given=John, got %s", given)
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

	// Check person files
	personFiles, err := os.ReadDir(personsDir)
	if err != nil {
		t.Fatalf("Failed to read persons directory: %v", err)
	}
	if len(personFiles) != 2 {
		t.Errorf("Expected 2 person files, got %d", len(personFiles))
	}

	// Check event files
	eventFiles, err := os.ReadDir(eventsDir)
	if err != nil {
		t.Fatalf("Failed to read events directory: %v", err)
	}
	if len(eventFiles) != 1 {
		t.Errorf("Expected 1 event file, got %d", len(eventFiles))
	}

	t.Logf("Created multi-file archive in %s", tmpDir)
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
	loaded, err := s.DeserializeMultiFileFromMap(filesRead)
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

	given, _ := ExtractNameFields(person.Properties["name"])
	if given != "John" {
		t.Errorf("Expected name.fields.given=John, got %s", given)
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

	// Compare
	if len(loaded.Persons) != len(original.Persons) {
		t.Errorf("Person count mismatch: expected %d, got %d", len(original.Persons), len(loaded.Persons))
	}

	loadedPerson := loaded.Persons["person-001"]
	originalPerson := original.Persons["person-001"]

	loadedGiven, _ := ExtractNameFields(loadedPerson.Properties["name"])
	originalGiven, _ := ExtractNameFields(originalPerson.Properties["name"])
	if loadedGiven != originalGiven {
		t.Error("name.fields.given mismatch after round-trip")
	}
}

func TestEntityWithID(t *testing.T) {
	person := Person{
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

	wrapper := EntityWithID[Person]{
		ID:     "person-001",
		Entity: person,
	}

	// Marshal
	yamlBytes, err := yaml.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check YAML contains _id
	yamlStr := string(yamlBytes)
	if !contains(yamlStr, "_id: person-001") {
		t.Error("YAML doesn't contain _id field")
	}

	// Unmarshal
	var restored EntityWithID[Person]
	if err := yaml.Unmarshal(yamlBytes, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Check ID restored
	if restored.ID != wrapper.ID {
		t.Errorf("Expected ID %s, got %s", wrapper.ID, restored.ID)
	}

	// Check entity restored
	given, _ := ExtractNameFields(restored.Entity.Properties["name"])
	if given != "John" {
		t.Error("Entity properties not restored correctly")
	}
}
