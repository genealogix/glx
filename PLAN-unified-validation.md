# Implementation Plan: Unified Validation Architecture

## Overview

Refactor validation into a unified, reflection-based system that uses `refType` struct tags as the single source of truth. Move validation logic into the `lib` package as methods on `GLXFile`, eliminating duplicate validation pathways and ensuring comprehensive foreign key validation.

## Current Problems

### 1. Dual Validation Pathways
- **File-level validation** (`ValidateGLXFile`) works on `map[string]interface{}`
- **Archive-level validation** (`ValidateArchive`) works on Go structs
- Results in incomplete coverage and maintenance burden

### 2. Missing Validations
Currently **NOT validated** in struct-based validation:
- ❌ `event.type` → `event_types` vocabulary
- ❌ `relationship.type` → `relationship_types` vocabulary
- ❌ `relationship.start_event` → `events`
- ❌ `relationship.end_event` → `events`
- ❌ `place.type` → `place_types` vocabulary
- ❌ `source.type` → `source_types` vocabulary
- ❌ `source.media[]` → `media` entities
- ❌ `citation.media[]` → `media` entities
- ❌ `citation.quality` → `quality_ratings` vocabulary
- ❌ `assertion.confidence` → `confidence_levels` vocabulary
- ❌ `participant.role` → `participant_roles` vocabulary (in events, relationships, assertions)

### 3. No Systematic Approach
- Each entity type has hand-written validation logic
- Easy to miss references when adding new fields
- `refType` tags exist but aren't used systematically
- Property `reference_type` validation exists but isn't unified

## Design Goals

1. **Single Source of Truth**: `refType` struct tags drive all reference validation
2. **Complete Coverage**: All foreign keys validated automatically
3. **Clear Policy**: Warnings for unknowns, errors for broken references
4. **Maintainable**: Adding new entities/fields requires no validation code changes
5. **Type-Safe**: Compile-time safety where possible, runtime reflection where needed
6. **Clean API**: Simple `archive.Validate()` method
7. **Rich Errors**: Full context (source type, id, field, target type, id)

## Validation Policy

### Warnings (Soft - Allow but notify)
- ❗ Unknown property on entity (not in property vocabulary)
- ❗ Unknown assertion claim (not in property vocabulary)
- ❗ Missing property vocabulary for entity type

### Errors (Hard - Fail validation)
- ❌ **All `refType` references must exist**:
  - Entity references: `event.place`, `citation.source`, `relationship.participants[].person`
  - Vocabulary references: `event.type`, `relationship.type`, `citation.quality`
  - Participant roles: `participant.role` in events, relationships, and assertions
- ❌ **Properties with `reference_type` defined must reference valid entities**:
  - Example: If `born_at` has `reference_type: places`, then the value must exist
- ❌ **Structural validation**:
  - Duplicate entity IDs
  - Invalid entity ID format
  - Required fields missing (type, name, etc.)
  - Mutual exclusivity (e.g., assertion `participant` vs `value`)

## Proposed Architecture

### Core Types

```go
// lib/types.go

type GLXFile struct {
    // ... existing entity maps ...
    Persons       map[string]*Person
    Events        map[string]*Event
    Relationships map[string]*Relationship
    Places        map[string]*Place
    Sources       map[string]*Source
    Citations     map[string]*Citation
    Repositories  map[string]*Repository
    Assertions    map[string]*Assertion
    Media         map[string]*Media
    
    // ... existing vocabulary maps ...
    EventTypes        map[string]*EventType
    RelationshipTypes map[string]*RelationshipType
    // ... etc ...
    
    // Validation state (built on demand, cached)
    validation *ValidationResult
}

// ValidationResult holds the complete validation state
type ValidationResult struct {
    // What exists in the archive
    Entities      map[string]map[string]struct{} // "persons" -> {"person-1": {}}
    Vocabularies  map[string]map[string]struct{} // "event_types" -> {"birth": {}}
    
    // Property vocabularies (for reference_type validation)
    PropertyVocabs map[string]map[string]*PropertyDefinition
    
    // Results
    Errors        []ValidationError
    Warnings      []ValidationWarning
    
    validated     bool // Has validation been run?
}

// ValidationError represents a hard validation failure
type ValidationError struct {
    SourceType  string // "events"
    SourceID    string // "event-123"
    SourceField string // "place" or "participants[0].role" or "properties.born_at"
    TargetType  string // "places" or "participant_roles"
    TargetID    string // "place-nonexistent"
    Message     string // Human-readable error
}

// ValidationWarning represents a soft validation issue
type ValidationWarning struct {
    SourceType  string // "persons"
    SourceID    string // "person-123"
    Field       string // "properties.unknown_prop"
    Message     string // Human-readable warning
}
```

### Main Validation Method

```go
// Validate performs comprehensive validation of the archive
// Returns cached results if validation has already been run
func (glx *GLXFile) Validate() *ValidationResult {
    if glx.validation != nil && glx.validation.validated {
        return glx.validation
    }
    
    result := &ValidationResult{
        Entities:       make(map[string]map[string]struct{}),
        Vocabularies:   make(map[string]map[string]struct{}),
        PropertyVocabs: make(map[string]map[string]*PropertyDefinition),
    }
    
    // Phase 1: Build existence maps
    glx.buildEntityMaps(result)
    glx.buildVocabularyMaps(result)
    glx.buildPropertyVocabMaps(result)
    
    // Phase 2: Validate all references using reflection
    glx.validateAllReferences(result)
    
    // Phase 3: Validate entity properties
    glx.validateAllProperties(result)
    
    result.validated = true
    glx.validation = result
    return result
}

// InvalidateCache clears cached validation results
// Call this after modifying the archive
func (glx *GLXFile) InvalidateCache() {
    glx.validation = nil
}
```

### Reflection-Based Validation

```go
// validateAllReferences uses reflection to validate all refType tags
func (glx *GLXFile) validateAllReferences(result *ValidationResult) {
    // Validate each entity type
    glx.validateEntityTypeReferences("persons", glx.Persons, result)
    glx.validateEntityTypeReferences("events", glx.Events, result)
    glx.validateEntityTypeReferences("relationships", glx.Relationships, result)
    glx.validateEntityTypeReferences("places", glx.Places, result)
    glx.validateEntityTypeReferences("sources", glx.Sources, result)
    glx.validateEntityTypeReferences("citations", glx.Citations, result)
    glx.validateEntityTypeReferences("repositories", glx.Repositories, result)
    glx.validateEntityTypeReferences("assertions", glx.Assertions, result)
    glx.validateEntityTypeReferences("media", glx.Media, result)
}

// validateEntityTypeReferences validates all entities of a given type
func (glx *GLXFile) validateEntityTypeReferences(
    entityType string,
    entities interface{},
    result *ValidationResult,
) {
    // Use reflection to iterate through entity map
    entitiesVal := reflect.ValueOf(entities)
    if entitiesVal.Kind() != reflect.Map {
        return
    }
    
    for _, key := range entitiesVal.MapKeys() {
        entityID := key.String()
        entity := entitiesVal.MapIndex(key)
        
        // Validate this entity's references
        glx.validateStructReferences(entityType, entityID, entity, result)
    }
}

// validateStructReferences recursively validates all refType tags in a struct
func (glx *GLXFile) validateStructReferences(
    entityType, entityID string,
    entityVal reflect.Value,
    result *ValidationResult,
) {
    // Dereference pointer if needed
    if entityVal.Kind() == reflect.Ptr {
        if entityVal.IsNil() {
            return
        }
        entityVal = entityVal.Elem()
    }
    
    if entityVal.Kind() != reflect.Struct {
        return
    }
    
    entityTypeVal := entityVal.Type()
    
    // Iterate through struct fields
    for i := 0; i < entityVal.NumField(); i++ {
        field := entityTypeVal.Field(i)
        fieldVal := entityVal.Field(i)
        
        // Check for refType tag
        refType := field.Tag.Get("refType")
        if refType == "" {
            // No refType - check if it's a nested struct/slice to recurse into
            glx.validateNestedStructs(entityType, entityID, field.Name, fieldVal, result)
            continue
        }
        
        // Validate the reference(s)
        glx.validateFieldReference(entityType, entityID, field.Name, fieldVal, refType, result)
    }
}

// validateFieldReference validates a field with a refType tag
func (glx *GLXFile) validateFieldReference(
    entityType, entityID, fieldName string,
    fieldVal reflect.Value,
    refType string,
    result *ValidationResult,
) {
    switch fieldVal.Kind() {
    case reflect.String:
        // Single reference
        refID := fieldVal.String()
        if refID != "" {
            glx.checkReference(entityType, entityID, fieldName, refType, refID, result)
        }
        
    case reflect.Slice:
        // Array of references
        for i := 0; i < fieldVal.Len(); i++ {
            itemVal := fieldVal.Index(i)
            if itemVal.Kind() == reflect.String {
                refID := itemVal.String()
                fieldPath := fmt.Sprintf("%s[%d]", fieldName, i)
                glx.checkReference(entityType, entityID, fieldPath, refType, refID, result)
            }
        }
        
    case reflect.Ptr:
        // Pointer to value (e.g., *int for quality ratings)
        if !fieldVal.IsNil() {
            if fieldVal.Elem().Kind() == reflect.Int {
                refID := fmt.Sprintf("%d", fieldVal.Elem().Int())
                glx.checkReference(entityType, entityID, fieldName, refType, refID, result)
            }
        }
    }
}

// checkReference validates a single reference exists
func (glx *GLXFile) checkReference(
    entityType, entityID, fieldName string,
    refType string,
    refID string,
    result *ValidationResult,
) {
    // refType can be comma-separated (e.g., "persons,events,relationships,places")
    targetTypes := strings.Split(refType, ",")
    
    found := false
    for _, targetType := range targetTypes {
        targetType = strings.TrimSpace(targetType)
        
        // Check if it's a vocabulary or entity reference
        if isVocabularyType(targetType) {
            if _, exists := result.Vocabularies[targetType][refID]; exists {
                found = true
                break
            }
        } else {
            if _, exists := result.Entities[targetType][refID]; exists {
                found = true
                break
            }
        }
    }
    
    if !found {
        result.Errors = append(result.Errors, ValidationError{
            SourceType:  entityType,
            SourceID:    entityID,
            SourceField: fieldName,
            TargetType:  refType,
            TargetID:    refID,
            Message: fmt.Sprintf("%s[%s].%s references non-existent %s: %s",
                entityType, entityID, fieldName, refType, refID),
        })
    }
}

// isVocabularyType determines if a type is a vocabulary vs entity
func isVocabularyType(typeName string) bool {
    return strings.HasSuffix(typeName, "_types") ||
           strings.HasSuffix(typeName, "_levels") ||
           strings.HasSuffix(typeName, "_roles") ||
           strings.HasSuffix(typeName, "_ratings") ||
           strings.HasSuffix(typeName, "_properties")
}
```

### Property Validation

```go
// validateAllProperties validates properties on all entities
func (glx *GLXFile) validateAllProperties(result *ValidationResult) {
    glx.validateEntityProperties("persons", glx.Persons, 
        result.PropertyVocabs["persons"], result)
    glx.validateEntityProperties("events", glx.Events, 
        result.PropertyVocabs["events"], result)
    glx.validateEntityProperties("relationships", glx.Relationships, 
        result.PropertyVocabs["relationships"], result)
    glx.validateEntityProperties("places", glx.Places, 
        result.PropertyVocabs["places"], result)
}

// validateEntityProperties validates properties map on entities
func (glx *GLXFile) validateEntityProperties(
    entityType string,
    entities interface{},
    propVocab map[string]*PropertyDefinition,
    result *ValidationResult,
) {
    entitiesVal := reflect.ValueOf(entities)
    if entitiesVal.Kind() != reflect.Map {
        return
    }
    
    for _, key := range entitiesVal.MapKeys() {
        entityID := key.String()
        entity := entitiesVal.MapIndex(key)
        
        // Get properties field
        if entity.Kind() == reflect.Ptr {
            entity = entity.Elem()
        }
        if entity.Kind() != reflect.Struct {
            continue
        }
        
        propsField := entity.FieldByName("Properties")
        if !propsField.IsValid() || propsField.IsNil() {
            continue
        }
        
        properties, ok := propsField.Interface().(map[string]interface{})
        if !ok {
            continue
        }
        
        // Validate each property
        glx.validateProperties(entityType, entityID, properties, propVocab, result)
    }
}

// validateProperties validates a properties map
func (glx *GLXFile) validateProperties(
    entityType, entityID string,
    properties map[string]interface{},
    propVocab map[string]*PropertyDefinition,
    result *ValidationResult,
) {
    if len(properties) == 0 {
        return
    }
    
    // If no vocabulary exists, warn
    if len(propVocab) == 0 {
        result.Warnings = append(result.Warnings, ValidationWarning{
            SourceType: entityType,
            SourceID:   entityID,
            Field:      "properties",
            Message: fmt.Sprintf("%s[%s]: no property vocabulary found for %s",
                entityType, entityID, entityType),
        })
        return
    }
    
    for propName, propValue := range properties {
        propDef, exists := propVocab[propName]
        if !exists {
            // Unknown property - WARNING
            result.Warnings = append(result.Warnings, ValidationWarning{
                SourceType: entityType,
                SourceID:   entityID,
                Field:      fmt.Sprintf("properties.%s", propName),
                Message: fmt.Sprintf("%s[%s]: unknown property '%s'",
                    entityType, entityID, propName),
            })
            continue
        }
        
        // If it has a reference_type, validate the reference
        if propDef.ReferenceType != "" {
            glx.validatePropertyReference(
                entityType, entityID, propName, propValue,
                propDef.ReferenceType, result,
            )
        }
    }
}

// validatePropertyReference validates a property that references an entity
func (glx *GLXFile) validatePropertyReference(
    entityType, entityID, propName string,
    propValue interface{},
    referenceType string,
    result *ValidationResult,
) {
    // Handle single value
    if refID, ok := propValue.(string); ok {
        if _, exists := result.Entities[referenceType][refID]; !exists {
            result.Errors = append(result.Errors, ValidationError{
                SourceType:  entityType,
                SourceID:    entityID,
                SourceField: fmt.Sprintf("properties.%s", propName),
                TargetType:  referenceType,
                TargetID:    refID,
                Message: fmt.Sprintf(
                    "%s[%s].properties.%s references non-existent %s: %s",
                    entityType, entityID, propName, referenceType, refID,
                ),
            })
        }
        return
    }
    
    // Handle temporal property (list of {value, date} objects)
    if valueList, ok := propValue.([]interface{}); ok {
        for i, item := range valueList {
            if itemMap, ok := item.(map[string]interface{}); ok {
                if refID, ok := itemMap["value"].(string); ok {
                    if _, exists := result.Entities[referenceType][refID]; !exists {
                        result.Errors = append(result.Errors, ValidationError{
                            SourceType:  entityType,
                            SourceID:    entityID,
                            SourceField: fmt.Sprintf("properties.%s[%d].value", propName, i),
                            TargetType:  referenceType,
                            TargetID:    refID,
                            Message: fmt.Sprintf(
                                "%s[%s].properties.%s[%d].value references non-existent %s: %s",
                                entityType, entityID, propName, i, referenceType, refID,
                            ),
                        })
                    }
                }
            }
        }
    }
}
```

### Map Building Functions

```go
// buildEntityMaps builds maps of all entity IDs
func (glx *GLXFile) buildEntityMaps(result *ValidationResult) {
    result.Entities["persons"] = buildIDSet(glx.Persons)
    result.Entities["events"] = buildIDSet(glx.Events)
    result.Entities["relationships"] = buildIDSet(glx.Relationships)
    result.Entities["places"] = buildIDSet(glx.Places)
    result.Entities["sources"] = buildIDSet(glx.Sources)
    result.Entities["citations"] = buildIDSet(glx.Citations)
    result.Entities["repositories"] = buildIDSet(glx.Repositories)
    result.Entities["assertions"] = buildIDSet(glx.Assertions)
    result.Entities["media"] = buildIDSet(glx.Media)
}

// buildVocabularyMaps builds maps of all vocabulary values
func (glx *GLXFile) buildVocabularyMaps(result *ValidationResult) {
    result.Vocabularies["event_types"] = buildIDSet(glx.EventTypes)
    result.Vocabularies["relationship_types"] = buildIDSet(glx.RelationshipTypes)
    result.Vocabularies["place_types"] = buildIDSet(glx.PlaceTypes)
    result.Vocabularies["repository_types"] = buildIDSet(glx.RepositoryTypes)
    result.Vocabularies["participant_roles"] = buildIDSet(glx.ParticipantRoles)
    result.Vocabularies["media_types"] = buildIDSet(glx.MediaTypes)
    result.Vocabularies["confidence_levels"] = buildIDSet(glx.ConfidenceLevels)
    result.Vocabularies["quality_ratings"] = buildIDSet(glx.QualityRatings)
    result.Vocabularies["source_types"] = buildIDSet(glx.SourceTypes)
}

// buildPropertyVocabMaps builds maps of property vocabularies
func (glx *GLXFile) buildPropertyVocabMaps(result *ValidationResult) {
    result.PropertyVocabs["persons"] = glx.PersonProperties
    result.PropertyVocabs["events"] = glx.EventProperties
    result.PropertyVocabs["relationships"] = glx.RelationshipProperties
    result.PropertyVocabs["places"] = glx.PlaceProperties
}

// buildIDSet creates a set of IDs from a map
func buildIDSet(m interface{}) map[string]struct{} {
    set := make(map[string]struct{})
    
    val := reflect.ValueOf(m)
    if val.Kind() != reflect.Map {
        return set
    }
    
    for _, key := range val.MapKeys() {
        set[key.String()] = struct{}{}
    }
    
    return set
}
```

## Implementation Steps

### Phase 0: Schema Refactoring (Foundation)

#### 0.1 Create Master GLX Schema
- [ ] Create `specification/schema/v1/glx-file.schema.json` (master schema)
- [ ] Include all entity keys as optional properties (persons, events, etc.)
- [ ] Include all vocabulary keys as optional properties (event_types, etc.)
- [ ] Include all property vocab keys as optional properties (person_properties, etc.)
- [ ] Use `$ref` to reference individual entity/vocab schemas for readability
- [ ] Set `additionalProperties: false` at top level
- [ ] Make ALL keys optional (files can contain any combination)

#### 0.2 Refactor Existing Schemas
- [ ] Keep individual entity schemas (person.schema.json, event.schema.json, etc.)
- [ ] Keep individual vocabulary schemas  
- [ ] These are referenced by master schema via `$ref`
- [ ] Update `specification/schema/v1/embed.go` to include master schema

#### 0.3 Update File Validation Function
- [ ] Delete `ValidateGLXFile()` and `ValidateVocabularyFile()`
- [ ] Create single `ValidateGLXFileStructure()` that uses master schema
- [ ] Validate entity ID format for all entity sections
- [ ] Return structural errors before attempting to load/merge

### Phase 1: Core Infrastructure

#### 1.1 Add Validation Types to lib/types.go
- [ ] Add `ValidationResult` struct
- [ ] Add `ValidationError` struct
- [ ] Add `ValidationWarning` struct
- [ ] Add `validation *ValidationResult` field to `GLXFile`

#### 1.2 Implement Map Building Functions
- [ ] Implement `buildEntityMaps()`
- [ ] Implement `buildVocabularyMaps()`
- [ ] Implement `buildPropertyVocabMaps()`
- [ ] Implement helper `buildIDSet()`

#### 1.3 Implement Main Validation Method
- [ ] Implement `GLXFile.Validate()` method
- [ ] Implement `GLXFile.InvalidateCache()` method
- [ ] Wire up the three validation phases

### Phase 2: Reflection-Based Reference Validation

#### 2.1 Implement Core Reflection Logic
- [ ] Implement `validateAllReferences()`
- [ ] Implement `validateEntityTypeReferences()`
- [ ] Implement `validateStructReferences()` (recursive field walker)
- [ ] Implement `validateFieldReference()` (handles string, slice, pointer)
- [ ] Implement `checkReference()` (validates single reference)
- [ ] Implement `isVocabularyType()` helper

#### 2.2 Handle Nested Structures
- [ ] Implement `validateNestedStructs()` for participants, dates, etc.
- [ ] Handle `[]EventParticipant` validation
- [ ] Handle `[]RelationshipParticipant` validation
- [ ] Handle `*AssertionParticipant` validation

### Phase 3: Property Validation

#### 3.1 Implement Property Validation
- [ ] Implement `validateAllProperties()`
- [ ] Implement `validateEntityProperties()`
- [ ] Implement `validateProperties()`
- [ ] Implement `validatePropertyReference()`
- [ ] Handle temporal property structures (list of {value, date})

#### 3.2 Generate Warnings
- [ ] Warn for unknown properties
- [ ] Warn for missing property vocabularies
- [ ] Don't fail validation on warnings

### Phase 4: Update glx Command

#### 4.1 Refactor glx validate Command
- [ ] Update `runValidate()` to use `archive.Validate()`
- [ ] File-level validation still uses unified structural validator
- [ ] Archive-level validation uses new `archive.Validate()`
- [ ] Format and display `ValidationError` objects
- [ ] Format and display `ValidationWarning` objects
- [ ] Return exit code 1 if errors exist
- [ ] Return exit code 0 if only warnings

Validation flow:
```go
// 1. Structural validation (per-file, before merge)
//    Uses single master schema - handles entities, vocabs, and properties in same file
for each .glx file {
    doc := parseYAML(file)
    issues := ValidateGLXFileStructure(doc)  // Against master schema
    if len(issues) > 0 {
        return errors  // Structural errors prevent loading
    }
}

// 2. Load and merge (all files merged into single GLXFile)
archive, duplicates, err := LoadArchive(path)

// 3. Cross-reference validation (new reflection-based system)
result := archive.Validate()

// 4. Report errors and warnings
for error := range result.Errors {
    fmt.Printf("❌ Error: %s\n", error.Message)
}
for warning := range result.Warnings {
    fmt.Printf("⚠️  Warning: %s\n", warning.Message)
}

// Exit code
if len(result.Errors) > 0 {
    return exit(1)
}
```

#### 4.2 Improve Output Formatting
- [ ] Group errors by source entity
- [ ] Use color coding (red for errors, yellow for warnings)
- [ ] Show summary: "X errors, Y warnings"
- [ ] Show detailed error messages with full context

### Phase 5: Remove Old Validation Code

#### 5.1 Delete Redundant Functions
- [ ] Delete `ValidateArchive()` from glx/validator.go
- [ ] Delete `validateRelationships()`
- [ ] Delete `validateEvents()`
- [ ] Delete `validatePlaces()`
- [ ] Delete `validatePersons()`
- [ ] Delete `validateCitations()`
- [ ] Delete `validateSources()`
- [ ] Delete `validateAssertions()`
- [ ] Delete `buildEntityMaps()` in glx/validator.go
- [ ] Delete `buildVocabularyStruct()` if no longer needed
- [ ] Delete `validateEntityPropertiesFromStruct()` if no longer needed
- [ ] Delete `validateAssertionSemanticsFromStruct()` if no longer needed

#### 5.2 Unify File-Level Validation with Master Schema
- [ ] **Delete both `ValidateGLXFile()` and `ValidateVocabularyFile()`** - they're based on false assumption
- [ ] **Create one master GLX schema** that allows ANY combination of top-level keys:
  - Entity keys: persons, events, relationships, places, sources, citations, repositories, assertions, media
  - Vocabulary keys: event_types, relationship_types, place_types, etc.
  - Property vocab keys: person_properties, event_properties, etc.
  - All are optional - a file can have any combination
- [ ] **Implement single `ValidateGLXFileStructure()`** function:
  - Uses one master schema
  - Validates against all possible top-level keys
  - Validates entity ID format for all entity keys
  - Validates vocabulary structure for all vocab keys
  - No need to "detect" file type - schema handles everything
- [ ] This validates YAML structure before merge, not cross-references

#### 5.3 Update Imports and Exports
- [ ] Remove unused imports from glx/validator.go
- [ ] Update glx package exports if needed
- [ ] Clean up any unused helper functions

### Phase 6: Testing

#### 6.1 Create Unit Tests (In-Memory, No Files!)
- [ ] Create `lib/validation_test.go`
- [ ] Test entity reference validation with programmatically-built GLXFile
- [ ] Test vocabulary reference validation with in-memory archives
- [ ] Test property reference validation
- [ ] Test unknown property warnings
- [ ] Test missing vocabulary warnings
- [ ] Test nested struct validation (participants)
- [ ] Test multi-type references (e.g., assertion.subject can be person/event/relationship/place)
- [ ] Test pointer types (e.g., citation.quality *int)
- [ ] Test slice references (e.g., source.media, assertion.citations)
- [ ] Test temporal property references (list of {value, date})

Example test structure:
```go
func TestValidateEntityReferences(t *testing.T) {
    // Build archive in memory
    archive := &GLXFile{
        Persons: map[string]*Person{
            "person-1": {},
        },
        Events: map[string]*Event{
            "event-1": {
                PlaceID: "place-nonexistent", // Should error
            },
        },
        Places: map[string]*Place{},
    }
    
    result := archive.Validate()
    
    // Assert errors
    require.Len(t, result.Errors, 1)
    assert.Equal(t, "events", result.Errors[0].SourceType)
    assert.Equal(t, "event-1", result.Errors[0].SourceID)
    assert.Equal(t, "place", result.Errors[0].SourceField)
    assert.Equal(t, "places", result.Errors[0].TargetType)
    assert.Equal(t, "place-nonexistent", result.Errors[0].TargetID)
}
```

#### 6.2 Integration Tests (Minimal File-Based)
- [ ] Keep existing `glx/validate_test.go` for CLI integration
- [ ] Update to use new validation API
- [ ] Keep a few testdata files for end-to-end testing
- [ ] Remove testdata files that can be replaced with in-memory tests

#### 6.3 Coverage Tests
- [ ] Add test for each entity type's references
- [ ] Add test for each vocabulary type's references
- [ ] Add test covering all missing validations listed in "Current Problems"
- [ ] Ensure 100% coverage of new validation code

### Phase 7: Documentation & Specification Updates

#### 7.1 Update Code Documentation
- [ ] Add package documentation to `lib/validation.go`
- [ ] Document all public methods with examples
- [ ] Document validation policy in comments
- [ ] Add examples of usage in godoc

#### 7.2 Review and Update Specification
- [ ] **Review all entity specifications** for validation policy statements
- [ ] Update `specification/2-core-concepts.md`:
  - Add clear section on "Validation Policy"
  - Explain warnings vs errors
  - Document that unknown properties/claims are warnings
  - Document that broken references are errors
- [ ] Update `specification/3-archive-organization.md`:
  - Document validation behavior
  - Explain referential integrity enforcement
- [ ] Update each entity specification in `specification/4-entity-types/`:
  - `person.md` - reference validation for properties
  - `event.md` - validation of place, participants, type
  - `relationship.md` - validation of participants, type, start_event, end_event
  - `place.md` - validation of parent, type
  - `source.md` - validation of repository, media, type
  - `citation.md` - validation of source, repository, media, quality
  - `repository.md` - validation of type
  - `assertion.md` - validation of subject, citations, sources, confidence, participant
  - `media.md` - validation of media_type
- [ ] Update `specification/4-entity-types/vocabularies.md`:
  - Document that vocabulary references are validated (errors for missing)
  - Document that property vocabularies are advisory (warnings for unknown)
  - Add section on "Validation Behavior"
- [ ] Search for any contradictions to new validation policy
- [ ] Ensure consistent use of "warning" vs "error" terminology throughout spec

## Technical Specifications

### Reflection Strategy

**Why Reflection?**
- Single source of truth: `refType` tags
- Automatic coverage: new fields work without code changes
- Extensible: works with any entity structure

**Performance Considerations:**
- Validation runs once and caches results
- Reflection overhead is acceptable for one-time validation
- Alternative would be code generation (overkill for this use case)

**Safety:**
- All reflection wrapped in type checks
- Nil pointer checks at every level
- Graceful degradation if unexpected types encountered

### Validation Order

1. **Build Maps** (cheap, fast)
   - Entity IDs
   - Vocabulary values
   - Property vocabularies

2. **Validate References** (expensive, cached)
   - Walk all entity structs with reflection
   - Check every `refType` tag
   - Validate nested structures

3. **Validate Properties** (moderate cost)
   - Check property names against vocabularies
   - Validate reference_type properties
   - Generate warnings for unknowns

### Error Message Format

```
❌ Error: events[event-123].place references non-existent places: place-nonexistent
❌ Error: relationships[rel-456].participants[0].person references non-existent persons: person-999
❌ Error: persons[person-789].properties.born_at references non-existent places: place-invalid

⚠️  Warning: persons[person-123]: unknown property 'shoe_size'
⚠️  Warning: events[event-456]: no property vocabulary found for events
```

### Cache Invalidation

The validation cache must be invalidated when:
- Entities are added/removed/modified
- Vocabularies are added/removed/modified
- Archive is merged with another archive

Call `glx.InvalidateCache()` after any modification.

## Implementation Strategy

This will be implemented by an AI agent in a single comprehensive pass:

0. **Create master GLX schema** - Foundation for structural validation
1. **Add new validation system** to lib package
2. **Update glx command** to use new system
3. **Write comprehensive tests** (primarily in-memory)
4. **Delete old validation code** completely
5. **Update specification** to align with new validation policy
6. **Verify all tests pass**

No gradual migration or deprecation period needed since this is pre-release.

### Key Architectural Insight

**GLX files are compositional** - any file can contain any combination of:
- Entities (persons, events, etc.)
- Vocabularies (event_types, etc.)  
- Property definitions (person_properties, etc.)

This means:
- ✅ One master schema for ALL GLX files
- ✅ All top-level keys are optional
- ✅ No need to "detect" file type
- ❌ No separate "entity file" vs "vocabulary file" validation

## Success Criteria

- [ ] All `refType` tags validated automatically
- [ ] All property `reference_type` values validated
- [ ] Unknown properties generate warnings (not errors)
- [ ] Broken references generate errors
- [ ] Validation runs in < 100ms for typical archive (1000 entities)
- [ ] All existing tests pass
- [ ] No regression in validation coverage
- [ ] New validation catches previously-missed errors
- [ ] Error messages are clear and actionable
- [ ] Cache invalidation works correctly

## Benefits Over Current System

1. **Completeness**: All references validated automatically
2. **Maintainability**: Add fields with `refType`, validation is automatic
3. **Clarity**: Single validation pathway, no confusion
4. **Performance**: Results cached, validation runs once
5. **Testability**: Easy to unit test in isolation
6. **Errors**: Rich context for debugging
7. **Extensibility**: Property vocabularies integrated seamlessly

## Questions & Decisions

### Q: Should we validate value_type (string, date, integer, boolean)?
**A:** Defer to future phase. Focus on reference validation first (harder problem). Value type validation can be added later without architectural changes.

### Q: Should we validate temporal property structure?
**A:** Yes, but only for reference_type properties. Value type validation deferred.

### Q: Should validation be automatic on Load?
**A:** No. Validation should be explicit via `Validate()` call. This allows:
- Loading without validation (for programmatic use)
- Validation on demand (CLI use)
- Multiple validation calls with cache

### Q: Should we support custom validation rules?
**A:** Not in initial implementation. Current policy (warnings for unknowns, errors for broken refs) covers 99% of use cases.

### Q: What about circular references?
**A:** Not applicable. GLX doesn't support circular references (e.g., place can't reference itself as parent). Reference graph is a DAG.

## Testing Philosophy

**Primary: In-Memory Unit Tests**
- Build `GLXFile` structures programmatically in test code
- Fast, no I/O overhead
- Easy to test edge cases
- Clear test cases in code, not YAML files

**Secondary: Integration Tests**
- Keep minimal file-based tests for CLI end-to-end testing
- Use existing testdata where it makes sense
- Focus on command-line interface behavior

**Example:**
```go
// Easy to read, fast to run, no files needed!
func TestMissingPlaceReference(t *testing.T) {
    archive := &GLXFile{
        Events: map[string]*Event{
            "event-1": {PlaceID: "place-999"},
        },
        Places: map[string]*Place{},
    }
    result := archive.Validate()
    assert.Contains(t, result.Errors[0].Message, "place-999")
}
```

## Key Benefits

1. **Testing is code** - clear, maintainable, version controlled
2. **No file overhead** - tests run in milliseconds
3. **Easy edge cases** - construct any scenario programmatically
4. **Better coverage** - test every code path without YAML complexity
5. **Cleaner codebase** - delete old validation code completely

## Next Steps

1. ✅ Review and approve this plan
2. Begin implementation in single comprehensive pass:
   - **Phase 0: Schema refactoring** (master GLX schema)
   - Phase 1: Core infrastructure
   - Phase 2: Reflection validation
   - Phase 3: Property validation
   - Phase 4: CLI updates
   - Phase 5: Delete old code
   - Phase 6: In-memory tests
   - Phase 7: Spec updates

