# GLX Serializer Implementation Plan

**Date**: 2025-11-18
**Status**: Architectural Plan
**Purpose**: Define single-file and multi-file YAML serializers for GLX archives

---

## Overview

This document defines the implementation plan for GLX archive serialization to YAML format. The serializer must support:

1. **Single-file format** - Entire archive in one YAML file (good for small archives, version control)
2. **Multi-file format** - Distributed across multiple files (recommended format, better for large archives)
3. **Split command** - Convert single-file to multi-file
4. **Join command** - Convert multi-file to single-file

---

## Requirements

### Functional Requirements

1. **Serialization**
   - Serialize GLXFile struct to YAML bytes
   - Handle all entity types (Person, Event, Relationship, Place, Source, Repository, Media, Citation, Assertion)
   - Include standard vocabularies
   - Preserve all fields including new Properties fields
   - Use YAML best practices (anchors, references where appropriate)

2. **Single-File Format**
   - One YAML file with all entities
   - Vocabularies included at top
   - Entities grouped by type
   - File size limit: recommend <10MB (warn if larger)

3. **Multi-File Format**
   - One file per entity (person-001.glx, event-001.glx, etc.)
   - One file per vocabulary type (event-types.glx, relationship-types.glx, etc.)
   - All properties of a type in one file (person-properties.glx)
   - Directory structure:
     ```
     archive/
     ├── persons/
     │   ├── person-001.glx
     │   ├── person-002.glx
     │   └── ...
     ├── events/
     │   ├── event-001.glx
     │   └── ...
     ├── relationships/
     ├── places/
     ├── sources/
     ├── repositories/
     ├── media/
     ├── citations/
     ├── assertions/
     ├── vocabularies/
     │   ├── event-types.glx
     │   ├── relationship-types.glx
     │   ├── place-types.glx
     │   └── ...
     └── properties/
         ├── person-properties.glx
         ├── event-properties.glx
         └── ...
     ```

4. **Standard Vocabularies**
   - Always write full standard vocabularies to new archives
   - Include all vocabulary types from glx/standard-vocabularies/
   - Vocabularies go in vocabularies/ directory
   - Properties go in properties/ directory

5. **CLI Integration**
   - Used by `glx import gedcom` to write imported archives
   - Used by `glx split` to convert single → multi
   - Used by `glx join` to convert multi → single
   - NEW ONLY archives (fail if archive exists)

### Non-Functional Requirements

1. **Performance**
   - Handle 10,000+ entities efficiently
   - Stream large files (don't load all in memory)
   - Concurrent file writes for multi-file format

2. **Error Handling**
   - Validate GLXFile before serialization
   - Return errors (don't panic)
   - Clear error messages

3. **Go Best Practices**
   - Use `any` instead of `interface{}`
   - Return errors, don't panic
   - Use context for cancellation
   - Proper resource cleanup (defer file.Close())

---

## API Design

### Core Serializer Interface

```go
// Serializer interface for GLX archives
type Serializer interface {
	// SerializeSingleFile serializes entire archive to single YAML file
	SerializeSingleFile(glx *GLXFile, outputPath string) error

	// SerializeMultiFile serializes archive to multi-file format
	SerializeMultiFile(glx *GLXFile, outputDir string) error

	// SerializeSingleFileBytes returns YAML bytes for single file
	SerializeSingleFileBytes(glx *GLXFile) ([]byte, error)
}

// DefaultSerializer implements the Serializer interface
type DefaultSerializer struct {
	IncludeVocabularies bool // Whether to include standard vocabularies
	IncludeProperties   bool // Whether to include property definitions
	ValidateBeforeSave  bool // Whether to validate before serializing
}
```

### Usage Examples

```go
// Example 1: GEDCOM import to single file
glx, _, err := ImportGEDCOMFromFile("family.ged", "import.log")
if err != nil {
	return err
}

serializer := &DefaultSerializer{
	IncludeVocabularies: true,
	IncludeProperties:   false, // Let validation add defaults
	ValidateBeforeSave:  true,
}

err = serializer.SerializeSingleFile(glx, "family.glx")

// Example 2: GEDCOM import to multi-file
err = serializer.SerializeMultiFile(glx, "family-archive/")

// Example 3: Get YAML bytes
yamlBytes, err := serializer.SerializeSingleFileBytes(glx)

// Example 4: Split existing archive
glx, err := LoadGLXFile("family.glx")
if err != nil {
	return err
}

err = serializer.SerializeMultiFile(glx, "family-archive/")

// Example 5: Join archive
glx, err := LoadGLXDirectory("family-archive/")
if err != nil {
	return err
}

err = serializer.SerializeSingleFile(glx, "family-combined.glx")
```

---

## Implementation Details

### Single-File Serialization

```go
func (s *DefaultSerializer) SerializeSingleFile(glx *GLXFile, outputPath string) error {
	// 1. Validate if requested
	if s.ValidateBeforeSave {
		result := glx.Validate()
		if len(result.Errors) > 0 {
			return fmt.Errorf("validation failed: %d errors", len(result.Errors))
		}
	}

	// 2. Check if file exists (NEW ONLY)
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("archive already exists: %s (use --force to overwrite)", outputPath)
	}

	// 3. Add standard vocabularies if requested
	if s.IncludeVocabularies {
		if err := addStandardVocabularies(glx); err != nil {
			return fmt.Errorf("failed to add vocabularies: %w", err)
		}
	}

	// 4. Serialize to YAML
	yamlBytes, err := s.SerializeSingleFileBytes(glx)
	if err != nil {
		return err
	}

	// 5. Write to file
	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error) {
	// Use gopkg.in/yaml.v3 for serialization
	yamlBytes, err := yaml.Marshal(glx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Check size and warn if large
	sizeMB := float64(len(yamlBytes)) / 1024 / 1024
	if sizeMB > 10 {
		// Log warning (use logger if available)
		fmt.Fprintf(os.Stderr, "Warning: Large archive (%.1f MB). Consider using multi-file format.\n", sizeMB)
	}

	return yamlBytes, nil
}
```

### Multi-File Serialization

```go
func (s *DefaultSerializer) SerializeMultiFile(glx *GLXFile, outputDir string) error {
	// 1. Validate if requested
	if s.ValidateBeforeSave {
		result := glx.Validate()
		if len(result.Errors) > 0 {
			return fmt.Errorf("validation failed: %d errors", len(result.Errors))
		}
	}

	// 2. Check if directory exists (NEW ONLY)
	if _, err := os.Stat(outputDir); err == nil {
		return fmt.Errorf("archive directory already exists: %s", outputDir)
	}

	// 3. Create directory structure
	dirs := []string{
		filepath.Join(outputDir, "persons"),
		filepath.Join(outputDir, "events"),
		filepath.Join(outputDir, "relationships"),
		filepath.Join(outputDir, "places"),
		filepath.Join(outputDir, "sources"),
		filepath.Join(outputDir, "repositories"),
		filepath.Join(outputDir, "media"),
		filepath.Join(outputDir, "citations"),
		filepath.Join(outputDir, "assertions"),
		filepath.Join(outputDir, "vocabularies"),
		filepath.Join(outputDir, "properties"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 4. Write entities (concurrent)
	errChan := make(chan error, 10)

	// Write persons
	go func() {
		errChan <- writeEntities(glx.Persons, filepath.Join(outputDir, "persons"), "person")
	}()

	// Write events
	go func() {
		errChan <- writeEntities(glx.Events, filepath.Join(outputDir, "events"), "event")
	}()

	// Write relationships
	go func() {
		errChan <- writeEntities(glx.Relationships, filepath.Join(outputDir, "relationships"), "relationship")
	}()

	// ... other entity types ...

	// Wait for all writes to complete
	for i := 0; i < 9; i++ { // Number of entity types
		if err := <-errChan; err != nil {
			return err
		}
	}

	// 5. Write vocabularies
	if s.IncludeVocabularies {
		if err := writeStandardVocabularies(outputDir); err != nil {
			return err
		}
	}

	return nil
}

// writeEntities writes a map of entities to individual files
func writeEntities[T any](entities map[string]T, dir string, prefix string) error {
	for id, entity := range entities {
		// Sanitize ID for filename
		filename := sanitizeFilename(id) + ".glx"
		filepath := filepath.Join(dir, filename)

		// Marshal entity
		yamlBytes, err := yaml.Marshal(entity)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", id, err)
		}

		// Write file
		if err := os.WriteFile(filepath, yamlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filepath, err)
		}
	}

	return nil
}
```

### Standard Vocabularies

```go
// addStandardVocabularies adds standard vocabularies to a GLXFile
func addStandardVocabularies(glx *GLXFile) error {
	vocabDir := "glx/standard-vocabularies"

	// Load each vocabulary type
	vocabTypes := []string{
		"event-types.glx",
		"relationship-types.glx",
		"place-types.glx",
		"source-types.glx",
		"repository-types.glx",
		"media-types.glx",
		"participant-roles.glx",
		"confidence-levels.glx",
		"quality-ratings.glx",
	}

	for _, vocabFile := range vocabTypes {
		path := filepath.Join(vocabDir, vocabFile)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read vocabulary %s: %w", vocabFile, err)
		}

		// Parse and add to GLXFile
		// (Implementation depends on GLXFile structure)
	}

	return nil
}

// writeStandardVocabularies writes standard vocabularies to directory
func writeStandardVocabularies(outputDir string) error {
	vocabDir := filepath.Join(outputDir, "vocabularies")
	sourceDir := "glx/standard-vocabularies"

	// Copy all vocabulary files
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to read vocabulary directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".glx") {
			source := filepath.Join(sourceDir, file.Name())
			dest := filepath.Join(vocabDir, file.Name())

			data, err := os.ReadFile(source)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", source, err)
			}

			if err := os.WriteFile(dest, data, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dest, err)
			}
		}
	}

	return nil
}
```

---

## CLI Commands

### glx import

```bash
# Import GEDCOM to single file
glx import gedcom family.ged -o family.glx

# Import GEDCOM to multi-file
glx import gedcom family.ged -o family-archive/ --multi-file

# Import with logging
glx import gedcom family.ged -o family.glx --log import.log

# Import without vocabularies
glx import gedcom family.ged -o family.glx --no-vocabularies
```

### glx split

```bash
# Split single file to multi-file
glx split family.glx -o family-archive/

# With force overwrite
glx split family.glx -o family-archive/ --force
```

### glx join

```bash
# Join multi-file to single file
glx join family-archive/ -o family.glx

# With force overwrite
glx join family-archive/ -o family.glx --force
```

---

## File Format Examples

### Single-File Format

```yaml
# family.glx

# Vocabularies (optional, can be omitted if using standard)
event_types:
  birth:
    label: "Birth"
    description: "The birth of an individual"
  death:
    label: "Death"
    description: "The death of an individual"
  # ... more types ...

# Entities
persons:
  person-001:
    properties:
      given_name: "John"
      surname: "Smith"
      gender: "male"
      external_ids:
        - id: "12345"
          type: "wikitree"
    notes: "Additional information..."
    tags: ["primary-line"]

events:
  event-001:
    type: "birth"
    date: "1850-03-15"
    place: "place-001"
    participants:
      - person: "person-001"
        role: "principal"
    properties:
      age_at_event: "0"

# ... more entities ...
```

### Multi-File Format

```yaml
# persons/person-001.glx
properties:
  given_name: "John"
  surname: "Smith"
  gender: "male"
  external_ids:
    - id: "12345"
      type: "wikitree"
notes: "Additional information..."
tags: ["primary-line"]
```

```yaml
# events/event-001.glx
type: "birth"
date: "1850-03-15"
place: "place-001"
participants:
  - person: "person-001"
    role: "principal"
properties:
  age_at_event: "0"
```

```yaml
# vocabularies/event-types.glx
birth:
  label: "Birth"
  description: "The birth of an individual"
death:
  label: "Death"
  description: "The death of an individual"
# ... more types ...
```

---

## Testing Strategy

### Unit Tests

1. **TestSerializeSingleFile**
   - Small GLXFile with few entities
   - Verify YAML output structure
   - Verify file created
   - Verify round-trip (serialize → deserialize → compare)

2. **TestSerializeMultiFile**
   - Same GLXFile
   - Verify directory structure
   - Verify individual entity files
   - Verify round-trip

3. **TestStandardVocabularies**
   - Verify all vocabulary files included
   - Verify format

4. **TestLargeArchive**
   - 10,000+ entities
   - Performance benchmarks
   - Memory usage

### Integration Tests

1. **TestGEDCOMImportToSingleFile**
   - Import Shakespeare GEDCOM
   - Serialize to single file
   - Validate output

2. **TestGEDCOMImportToMultiFile**
   - Import Shakespeare GEDCOM
   - Serialize to multi-file
   - Validate output

3. **TestSplitJoinRoundTrip**
   - Create archive
   - Split to multi-file
   - Join back to single
   - Compare

---

## Error Handling

### Common Errors

1. **Archive Already Exists**
   ```
   Error: Archive already exists: family.glx
   Use --force to overwrite, or choose a different output path
   ```

2. **Validation Failed**
   ```
   Error: Validation failed before serialization
   Found 3 errors:
   - person-001: Missing required field 'properties.surname'
   - event-001: Invalid type 'birt' (should be 'birth')
   - relationship-001: Person 'person-999' not found

   Run 'glx validate family.glx' for details
   ```

3. **File Write Error**
   ```
   Error: Failed to write file: family.glx
   Cause: Permission denied
   ```

4. **YAML Marshal Error**
   ```
   Error: Failed to serialize entity person-001
   Cause: Invalid character in properties field
   ```

---

## Dependencies

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)
```

---

## Implementation Checklist

- [ ] Create `lib/serializer.go` with Serializer interface
- [ ] Implement DefaultSerializer struct
- [ ] Implement SerializeSingleFileBytes()
- [ ] Implement SerializeSingleFile()
- [ ] Implement SerializeMultiFile()
- [ ] Implement writeEntities() helper
- [ ] Implement addStandardVocabularies()
- [ ] Implement writeStandardVocabularies()
- [ ] Add unit tests for single-file serialization
- [ ] Add unit tests for multi-file serialization
- [ ] Add integration tests with GEDCOM import
- [ ] Update cmd/glx/import.go to use serializer
- [ ] Implement cmd/glx/split.go
- [ ] Implement cmd/glx/join.go
- [ ] Add CLI tests
- [ ] Update documentation
- [ ] Add examples to website

---

## Architectural Questions for Review

### 1. Vocabulary Handling

**Question**: Should we embed standard vocabularies in the binary or read from files?

**Option A**: Embed using go:embed
- Pros: Self-contained binary, always available
- Cons: Larger binary, harder to customize

**Option B**: Read from files
- Pros: Easy to customize, smaller binary
- Cons: Requires vocabulary files at runtime

**Recommendation**: Option A - embed for default, allow override from files
**✅ DECISION**: Embed vocabularies in binary using go:embed, write with serializer

### 2. Entity File Naming

**Question**: What naming convention for entity files?

**Option A**: Use entity ID directly
- Example: `person-john-smith-001.glx`
- Pros: Human-readable
- Cons: ID might have special characters

**Option B**: Sanitize ID for filename
- Example: `person-john-smith-001.glx` (sanitized)
- Pros: Safe filenames
- Cons: Might lose information

**Option C**: Use random IDs
- Example: `person-123hgadf18.glx`
- Pros: Simple, safe, no collisions
- Cons: Not human-readable without opening file

**Recommendation**: Option B - sanitize but keep readable
**✅ DECISION**: Use random IDs (Option C) - simpler implementation, no naming conflicts

### 3. Concurrent Writes

**Question**: Should we write entity files concurrently?

**Pros**: Faster for large archives
**Cons**: More complex, potential for resource exhaustion

**Recommendation**: Yes, with worker pool (limit concurrent goroutines)
**✅ DECISION**: No - keep it simple with sequential writes for initial implementation

### 4. Validation Before Save

**Question**: Should we always validate before saving?

**Option A**: Always validate
- Pros: Prevents saving invalid archives
- Cons: Slower, might reject valid-but-unusual archives

**Option B**: Optional validation (flag)
- Pros: Flexible, faster
- Cons: Might save invalid archives

**Recommendation**: Option B - optional, default to true
**✅ DECISION**: Default to true with --no-validate flag to override

---

## Next Steps

1. Get approval on architectural decisions
2. Implement serializer interface and basic structure
3. Implement single-file serialization
4. Implement multi-file serialization
5. Add vocabulary handling
6. Integrate with GEDCOM import command
7. Implement split/join commands
8. Add comprehensive tests
9. Update documentation
