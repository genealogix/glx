# GLX Entity ID Generation Strategy

**Date**: 2025-11-18
**Status**: Implementation Plan
**Decision**: Use random IDs for multi-file entity filenames

---

## Overview

When serializing to multi-file format, each entity needs a unique filename. We'll use short random IDs to ensure no naming conflicts while keeping filenames simple and safe.

---

## Requirements

1. **Unique**: No collisions across all entities
2. **Short**: Reasonably short filenames (not UUID-length)
3. **Safe**: Valid across all filesystems (no special characters)
4. **Fast**: Quick to generate (thousands per second)
5. **Readable**: Somewhat recognizable as entity type

---

## Format

```
{entity-type}-{random-id}.glx

Examples:
person-a3f8d2c1.glx
event-b9e4f7a2.glx
relationship-c1d8e9f3.glx
place-d4a7b2e8.glx
```

**Components**:
- `{entity-type}`: person, event, relationship, place, source, repository, media, citation, assertion
- `{random-id}`: 8-character hexadecimal (32 bits of randomness)
- `.glx`: Standard GLX file extension

**Collision Probability**:
- 32 bits = 4,294,967,296 possible values
- Even with 10,000 entities, collision probability is ~1 in 400,000
- Acceptable for typical genealogy archives

---

## Implementation

### ID Generator

```go
// lib/id_generator.go

package lib

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateRandomID generates a random 8-character hex ID
// Returns lowercase hex string like "a3f8d2c1"
func GenerateRandomID() (string, error) {
	// Generate 4 random bytes (32 bits)
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random ID: %w", err)
	}

	// Convert to hex (8 characters)
	return hex.EncodeToString(bytes), nil
}

// GenerateEntityFilename generates a random filename for an entity
// Format: {entity-type}-{random-id}.glx
// Example: person-a3f8d2c1.glx
func GenerateEntityFilename(entityType string) (string, error) {
	id, err := GenerateRandomID()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s.glx", entityType, id), nil
}

// MustGenerateRandomID is like GenerateRandomID but panics on error
// Use only in tests or when random source is guaranteed available
func MustGenerateRandomID() string {
	id, err := GenerateRandomID()
	if err != nil {
		panic(err)
	}
	return id
}
```

### Usage in Serializer

```go
// lib/serializer.go

// writeEntities writes a map of entities to individual files with random IDs
func writeEntities[T any](entities map[string]T, dir string, entityType string) error {
	for entityID, entity := range entities {
		// Generate random filename
		filename, err := GenerateEntityFilename(entityType)
		if err != nil {
			return fmt.Errorf("failed to generate filename for %s: %w", entityID, err)
		}

		filepath := filepath.Join(dir, filename)

		// Marshal entity
		yamlBytes, err := yaml.Marshal(entity)
		if err != nil {
			return fmt.Errorf("failed to marshal %s %s: %w", entityType, entityID, err)
		}

		// Write file
		if err := os.WriteFile(filepath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}
	}

	return nil
}
```

---

## Why Not UUIDs?

UUIDs (Universally Unique Identifiers) are commonly used but have drawbacks for our use case:

**UUID v4 Format**: `550e8400-e29b-41d4-a716-446655440000`
- **Too long**: 36 characters (with hyphens) or 32 (without)
- **Overkill**: 128 bits of randomness is excessive for genealogy archives
- **Less readable**: Long hex strings are hard to scan

**Our Format**: `person-a3f8d2c1.glx`
- **Shorter**: 22 characters total (including extension)
- **Sufficient**: 32 bits is enough for typical archives
- **Clearer**: Entity type prefix makes it obvious what the file contains

**Trade-off**: Slightly higher collision risk, but acceptable for use case.

---

## Alternative: Content-Based IDs

Another approach is to generate IDs based on entity content:

```go
// GenerateContentBasedID generates ID from entity content hash
func GenerateContentBasedID(entity any) (string, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(entity)
	if err != nil {
		return "", err
	}

	// Hash content
	hash := sha256.Sum256(jsonBytes)

	// Take first 4 bytes (32 bits)
	return hex.EncodeToString(hash[:4]), nil
}
```

**Pros**:
- Deterministic (same entity always gets same ID)
- Deduplication (identical entities get same ID)

**Cons**:
- Slower (hashing is more expensive)
- Still have collisions (birthday paradox)
- Content changes → ID changes → harder to track

**Decision**: Use random IDs - simpler, faster, good enough.

---

## Collision Detection

Even with low probability, we should detect collisions:

```go
// writeEntitiesWithCollisionDetection writes entities with collision checking
func writeEntitiesWithCollisionDetection[T any](entities map[string]T, dir string, entityType string) error {
	// Track used filenames
	usedFilenames := make(map[string]bool)

	for entityID, entity := range entities {
		// Generate filename with collision retry
		var filename string
		var filepath string
		maxRetries := 10

		for i := 0; i < maxRetries; i++ {
			var err error
			filename, err = GenerateEntityFilename(entityType)
			if err != nil {
				return fmt.Errorf("failed to generate filename: %w", err)
			}

			// Check if already used
			if !usedFilenames[filename] {
				usedFilenames[filename] = true
				break
			}

			// Collision detected, retry
			if i == maxRetries-1 {
				return fmt.Errorf("failed to generate unique filename after %d retries", maxRetries)
			}
		}

		filepath = filepath.Join(dir, filename)

		// Marshal and write entity
		yamlBytes, err := yaml.Marshal(entity)
		if err != nil {
			return fmt.Errorf("failed to marshal %s %s: %w", entityType, entityID, err)
		}

		if err := os.WriteFile(filepath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}
	}

	return nil
}
```

**Recommendation**: Add collision detection for robustness.

---

## Reading Multi-File Archives

When reading multi-file archives, we need to map filenames back to entity IDs:

```go
// LoadMultiFileArchive loads entities from a multi-file archive
func LoadMultiFileArchive(archiveDir string) (*GLXFile, error) {
	glx := &GLXFile{
		Persons:       make(map[string]*Person),
		Events:        make(map[string]*Event),
		Relationships: make(map[string]*Relationship),
		// ... other entity types ...
	}

	// Load persons
	personsDir := filepath.Join(archiveDir, "persons")
	files, err := os.ReadDir(personsDir)
	if err == nil { // Directory might not exist if no persons
		for _, file := range files {
			if filepath.Ext(file.Name()) != ".glx" {
				continue
			}

			// Read file
			path := filepath.Join(personsDir, file.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", path, err)
			}

			// Unmarshal person
			var person Person
			if err := yaml.Unmarshal(data, &person); err != nil {
				return nil, fmt.Errorf("failed to unmarshal person from %s: %w", path, err)
			}

			// Generate entity ID from person data
			// Could use person properties, or create new ID
			personID := generatePersonIDFromData(&person)

			glx.Persons[personID] = &person
		}
	}

	// ... load other entity types ...

	return glx, nil
}

// generatePersonIDFromData generates a consistent ID from person data
// This ensures references between entities work correctly
func generatePersonIDFromData(person *Person) string {
	// Use name + properties to create ID
	// This should be deterministic so references resolve correctly

	// Simple approach: hash key properties
	givenName, _ := person.Properties["given_name"].(string)
	surname, _ := person.Properties["surname"].(string)

	if givenName == "" && surname == "" {
		// Fallback to random ID
		return "person-" + MustGenerateRandomID()
	}

	// Create ID from name
	return fmt.Sprintf("person-%s-%s",
		sanitizeForID(givenName),
		sanitizeForID(surname))
}
```

**Issue**: Random filenames break references!

**Problem**: If `person-a3f8d2c1.glx` references `event-b9e4f7a2.glx`, but we generate new IDs when loading, references break.

**Solution**: Store entity ID inside the file:

```yaml
# person-a3f8d2c1.glx
_id: person-john-smith-001  # Store original entity ID
properties:
  given_name: "John"
  surname: "Smith"
```

Then when loading, use `_id` field to restore entity ID.

---

## Revised Implementation

### Writing with ID Metadata

```go
// EntityWithID wraps an entity with its ID for serialization
type EntityWithID[T any] struct {
	ID     string `yaml:"_id"`
	Entity T      `yaml:",inline"`
}

// writeEntitiesWithID writes entities with embedded ID field
func writeEntitiesWithID[T any](entities map[string]T, dir string, entityType string) error {
	usedFilenames := make(map[string]bool)

	for entityID, entity := range entities {
		// Generate random filename
		filename, err := generateUniqueFilename(entityType, usedFilenames)
		if err != nil {
			return err
		}

		// Wrap entity with ID
		wrapper := EntityWithID[T]{
			ID:     entityID,
			Entity: entity,
		}

		// Marshal
		yamlBytes, err := yaml.Marshal(wrapper)
		if err != nil {
			return fmt.Errorf("failed to marshal %s %s: %w", entityType, entityID, err)
		}

		// Write
		filepath := filepath.Join(dir, filename)
		if err := os.WriteFile(filepath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}
	}

	return nil
}
```

### Reading with ID Metadata

```go
// loadEntitiesWithID loads entities that have embedded _id field
func loadEntitiesWithID[T any](dir string) (map[string]T, error) {
	entities := make(map[string]T)

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return entities, nil // Empty directory is OK
		}
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".glx" {
			continue
		}

		path := filepath.Join(dir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}

		var wrapper EntityWithID[T]
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s: %w", path, err)
		}

		entities[wrapper.ID] = wrapper.Entity
	}

	return entities, nil
}
```

**Result**: Files have random names, but entity IDs are preserved in content.

---

## Testing

```go
// lib/id_generator_test.go

func TestGenerateRandomID(t *testing.T) {
	// Generate many IDs
	ids := make(map[string]bool)
	for i := 0; i < 10000; i++ {
		id, err := GenerateRandomID()
		if err != nil {
			t.Fatalf("Failed to generate ID: %v", err)
		}

		// Check format
		if len(id) != 8 {
			t.Errorf("Expected 8 character ID, got %d: %s", len(id), id)
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateEntityFilename(t *testing.T) {
	tests := []struct {
		entityType string
		wantPrefix string
	}{
		{"person", "person-"},
		{"event", "event-"},
		{"relationship", "relationship-"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			filename, err := GenerateEntityFilename(tt.entityType)
			if err != nil {
				t.Fatalf("Failed to generate filename: %v", err)
			}

			// Check prefix
			if !strings.HasPrefix(filename, tt.wantPrefix) {
				t.Errorf("Expected prefix %s, got %s", tt.wantPrefix, filename)
			}

			// Check suffix
			if !strings.HasSuffix(filename, ".glx") {
				t.Errorf("Expected .glx suffix, got %s", filename)
			}

			// Check total format
			expected := tt.wantPrefix + "[a-f0-9]{8}.glx"
			matched, _ := regexp.MatchString(expected, filename)
			if !matched {
				t.Errorf("Filename %s doesn't match pattern %s", filename, expected)
			}
		})
	}
}

func TestEntityWithID(t *testing.T) {
	person := Person{
		Properties: map[string]interface{}{
			"given_name": "John",
			"surname":    "Smith",
		},
	}

	wrapper := EntityWithID[Person]{
		ID:     "person-john-smith-001",
		Entity: person,
	}

	// Marshal
	yamlBytes, err := yaml.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Check YAML contains _id
	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "_id: person-john-smith-001") {
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
	if restored.Entity.Properties["given_name"] != "John" {
		t.Error("Entity properties not restored correctly")
	}
}
```

---

## Summary

**Format**: `{entity-type}-{8-char-hex}.glx`
**Generator**: `crypto/rand` for 32 bits of randomness
**Collision handling**: Retry with detection
**ID preservation**: Embed `_id` field in entity YAML
**Benefits**: Simple, safe, fast, preserves references

**Implementation**: ~150 lines of code + tests
