# Implementation Plan: Entity-Specific Property Vocabularies

## Overview

Implement entity-specific property vocabularies to enable controlled, context-aware validation of assertion claims. This includes support for participant assertions and allows properties to be set directly on entities as "accepted" values.

## Key Design Decisions

### 1. Assertion Structure

**Simple property claim:**
```yaml
assertions:
  assertion-birth-date:
    subject: person-john
    claim: born_on
    value: "1850-01-15"
    citations: [citation-birth-cert]
    confidence: high
```

**Participant claim:**
```yaml
assertions:
  assertion-witness:
    subject: event-marriage
    participant:
      person: person-thomas
      role: witness
    citations: [citation-marriage-cert]
    confidence: high
```

**Rules:**
- `claim` field is NOT present when `participant` is present (mutually exclusive)
- When `participant` exists, claim is implicitly "participant"
- `value` and `participant` are mutually exclusive

### 2. Properties on Entities

Entities can have properties set in a dedicated `properties` section to represent the "accepted" or "concluded" value:

**Current (nested concluded_identity):**
```yaml
persons:
  person-john:
    concluded_identity:
      primary_name: "John Smith"
      birth_date: "1850-01-15"
      death_date: "1920-06-20"
```

**New (segregated properties):**
```yaml
persons:
  person-john:
    # Polymorphic properties from vocabulary
    properties:
      primary_name: "John Smith"
      gender: "male"
      born_on: "1850-01-15"
      born_at: place-leeds
      died_on: "1920-06-20"
      occupation: "blacksmith"
      residence: place-leeds
    
    notes: "Research notes"
    tags: [verified]
```

**Benefits:**
- Clear separation of spec fields vs vocabulary-defined properties
- Properties match assertion claim names
- Fully extensible - archives can add any custom properties
- No special "concluded_identity" nesting
- Future-proof - new spec fields won't conflict with properties
- Living status implied by presence/absence of `died_on`

### 3. Remove All Backlinking

**Current (with backlinks):**
```yaml
persons:
  person-john:
    assertions: [assertion-birth, assertion-occupation]  # Remove this
    relationships: [rel-marriage, rel-parent-child]      # Remove this too
```

**New (no backlinks):**
```yaml
persons:
  person-john:
    properties:
      primary_name: "John Smith"
      gender: "male"
    # No assertions or relationships arrays - referential integrity via forward references
```

**Rationale:**
- Assertions reference their subjects via `assertion.subject`
- Relationships reference persons via `relationship.persons` or `relationship.participants`
- Events reference persons via `event.participants`
- No need for reverse references - they create maintenance burden and duplication

### 4. Validation Strategy

**Warnings (soft validation):**
- Unknown property claims (not in vocabulary)
- Value type mismatches
- Missing property vocabulary

**Errors (hard validation):**
- Invalid entity references in `participant.person`
- Invalid role references in `participant.role`
- Both `value` and `participant` present
- Both `claim` and `participant` present
- Missing required fields

### 5. Property Vocabulary Structure

```yaml
# person-properties.glx
person_properties:
  primary_name:
    label: "Primary Name"
    description: "Person's primary or preferred name"
    value_type: string
    temporal: true  # Names can change
  
  gender:
    label: "Gender"
    description: "Gender identity"
    value_type: string
    temporal: true  # Gender identity can change
  
  born_on:
    label: "Birth Date"
    description: "Date of birth"
    value_type: date
  
  born_at:
    label: "Birth Place"
    description: "Place of birth"
    reference_type: places
  
  died_on:
    label: "Death Date"
    description: "Date of death"
    value_type: date
  
  died_at:
    label: "Death Place"
    description: "Place of death"
    reference_type: places
  
  occupation:
    label: "Occupation"
    description: "Profession or trade"
    value_type: string
    temporal: true
  
  residence:
    label: "Residence"
    reference_type: places
    temporal: true
  
  religion:
    label: "Religion"
    description: "Religious affiliation"
    value_type: string
    temporal: true
  
  education:
    label: "Education"
    description: "Educational attainment"
    value_type: string
    temporal: true

# event-properties.glx
event_properties:
  occurred_on:
    label: "Event Date"
    description: "When the event occurred"
    value_type: date
  
  occurred_at:
    label: "Event Location"
    description: "Where the event occurred"
    reference_type: places
  
  # Note: Participants are NOT properties - they're handled by:
  # 1. event.participants array (structural)
  # 2. assertion.participant field (evidential)

# relationship-properties.glx
relationship_properties:
  started_on:
    label: "Start Date"
    description: "When the relationship began"
    value_type: date
  
  ended_on:
    label: "End Date"
    description: "When the relationship ended"
    value_type: date
  
  # Note: Participants are NOT properties - they're handled by:
  # 1. relationship.participants array (structural)
  # 2. assertion.participant field (evidential)

# place-properties.glx
place_properties:
  existed_from:
    label: "Existence Start"
    description: "When the place came into existence"
    value_type: date
  
  existed_to:
    label: "Existence End"
    description: "When the place ceased to exist"
    value_type: date
  
  population:
    label: "Population"
    description: "Population count"
    value_type: integer
```

## Implementation Steps

**Note on Scope:** The changes outlined in this plan, particularly the adoption of the FamilySearch Date Standard and the flattened entity structure, have repository-wide implications. All relevant specification documents, examples, and schemas must be updated to reflect these new standards consistently.

### Phase 1: Core Infrastructure

#### 1.1 Create Property Vocabulary Files
- [ ] Create `specification/5-standard-vocabularies/person-properties.glx`
- [ ] Create `specification/5-standard-vocabularies/event-properties.glx`
- [ ] Create `specification/5-standard-vocabularies/relationship-properties.glx`
- [ ] Create `specification/5-standard-vocabularies/place-properties.glx`
- [ ] Include standard properties for each entity type
- [ ] Add to `specification/5-standard-vocabularies/embed.go`

#### 1.2 Create Property Vocabulary Schemas
- [ ] Create `specification/schema/v1/vocabularies/person-properties.schema.json`
- [ ] Create `specification/schema/v1/vocabularies/event-properties.schema.json`
- [ ] Create `specification/schema/v1/vocabularies/relationship-properties.schema.json`
- [ ] Create `specification/schema/v1/vocabularies/place-properties.schema.json`
- [ ] Schema should validate:
  - Required `label` field
  - Optional `description`, `value_type`, `reference_type`, `temporal`, `applies_to`, `custom`
  - `value_type` and `reference_type` are mutually exclusive
  - Top-level key matches `{entity}_properties`
- [ ] Update `specification/schema/v1/embed.go` to include new schemas

#### 1.3 Update Go Structs in `lib/types.go`
- [ ] Add `PropertyDefinition` struct:
  ```go
  type PropertyDefinition struct {
      Label       string   `yaml:"label"`
      Description string   `yaml:"description,omitempty"`
      ValueType   string   `yaml:"value_type,omitempty"`
      Temporal    *bool    `yaml:"temporal,omitempty"`
      Custom      *bool    `yaml:"custom,omitempty"`
  }
  ```
- [ ] Add property vocabulary maps to `GLXFile` struct:
  ```go
  PersonProperties       map[string]*PropertyDefinition `yaml:"person_properties,omitempty"`
  EventProperties        map[string]*PropertyDefinition `yaml:"event_properties,omitempty"`
  RelationshipProperties map[string]*PropertyDefinition `yaml:"relationship_properties,omitempty"`
  PlaceProperties        map[string]*PropertyDefinition `yaml:"place_properties,omitempty"`
  ```
- [ ] Update `Assertion` struct:
  ```go
  type Assertion struct {
      Subject     string                `yaml:"subject" refType:"persons,events,relationships,places"`
      Claim       string                `yaml:"claim,omitempty"`  // Optional, not present if participant exists
      Value       string                `yaml:"value,omitempty"`  // Not present if participant exists
      Participant *AssertionParticipant `yaml:"participant,omitempty"`  // Not present if value exists
      Confidence  string                `yaml:"confidence,omitempty" refType:"confidence_levels"`
      Sources     []string              `yaml:"sources,omitempty" refType:"sources"`
      Citations   []string              `yaml:"citations,omitempty" refType:"citations"`
      Notes       string                `yaml:"notes,omitempty"`
      Tags        []string              `yaml:"tags,omitempty"`
  }
  
  type AssertionParticipant struct {
      Person string `yaml:"person" refType:"persons"`
      Role   string `yaml:"role,omitempty" refType:"participant_roles"`
      Notes  string `yaml:"notes,omitempty"`
  }
  ```

### Phase 2: Entity Schema Updates

#### 2.1 Flatten Person Entity
- [ ] Update `specification/schema/v1/person.schema.json`:
  - Remove `concluded_identity` nesting
  - Add properties directly at top level: `primary_name`, `born_on`, `died_on`, etc.
  - Make properties optional (they're asserted, not required)
  - Add `additionalProperties: true` to allow custom properties
- [ ] Update `lib/types.go` Person struct:
  ```go
  type Person struct {
      Properties map[string]interface{} `yaml:"properties,omitempty"`  // All properties including gender
      Notes      string                 `yaml:"notes,omitempty"`
      Tags       []string               `yaml:"tags,omitempty"`
  }
  ```
- [ ] Remove `ConcludedIdentity` struct (no longer needed)
- [ ] Remove `Gender` and `Living` fields (moved to properties)
- [ ] Properties are validated dynamically against `person_properties` vocabulary
- [ ] References within properties validated at runtime based on `value_type`

#### 2.2 Update Other Entity Schemas
- [ ] Update Event, Relationship, Place schemas:
  - Add optional `properties` object with `additionalProperties: true`
  - Keep existing spec fields outside properties section
- [ ] Update corresponding structs in `lib/types.go`:
  ```go
  type Event struct {
      Type         string                 `yaml:"type" refType:"event_types"`
      PlaceID      string                 `yaml:"place,omitempty" refType:"places"`
      Date         *EventDate             `yaml:"date,omitempty"`
      Participants []EventParticipant     `yaml:"participants,omitempty"`
      Properties   map[string]interface{} `yaml:"properties,omitempty"`
      Description  string                 `yaml:"description,omitempty"`
      Notes        string                 `yaml:"notes,omitempty"`
      Tags         []string               `yaml:"tags,omitempty"`
  }
  
  type Relationship struct {
      Type         string                    `yaml:"type" refType:"relationship_types"`
      Participants []RelationshipParticipant `yaml:"participants"` // Required, must have at least 2
      StartEvent   string                    `yaml:"start_event,omitempty" refType:"events"`
      EndEvent     string                    `yaml:"end_event,omitempty" refType:"events"`
      Properties   map[string]interface{}    `yaml:"properties,omitempty"`
      Description  string                    `yaml:"description,omitempty"`
      Notes        string                    `yaml:"notes,omitempty"`
      Tags         []string                  `yaml:"tags,omitempty"`
  }
  
  type Place struct {
      Name             string                 `yaml:"name"`
      ParentID         string                 `yaml:"parent,omitempty" refType:"places"`
      Type             string                 `yaml:"type,omitempty" refType:"place_types"`
      AlternativeNames []AlternativeName      `yaml:"alternative_names,omitempty"`
      Latitude         *float64               `yaml:"latitude,omitempty"`
      Longitude        *float64               `yaml:"longitude,omitempty"`
      Properties       map[string]interface{} `yaml:"properties,omitempty"`
      Notes            string                 `yaml:"notes,omitempty"`
      Tags             []string               `yaml:"tags,omitempty"`
  }
  ```

#### 2.3 Remove All Backlinking
- [ ] Remove `assertions` array from Person schema
- [ ] Remove `assertions` array from Event schema
- [ ] Remove `assertions` array from Relationship schema
- [ ] Remove `relationships` array from Person schema
- [ ] Remove `assertions` field from all entity structs in `lib/types.go`
- [ ] Remove `relationships` field from Person struct in `lib/types.go`

### Phase 3: Assertion Schema Updates

#### 3.1 Update Assertion Schema
- [ ] Update `specification/schema/v1/assertion.schema.json`:
  - Add optional `participant` object:
    ```json
    "participant": {
      "type": "object",
      "required": ["person"],
      "properties": {
        "person": {
          "type": "string",
          "pattern": "^[a-zA-Z0-9-]{1,64}$"
        },
        "role": {
          "type": "string"
        },
        "notes": {
          "type": "string"
        }
      }
    }
    ```
  - Add validation: if `participant` exists, `claim` and `value` must not exist
  - Make `claim` optional (not required if `participant` present)
  - Update `anyOf` for evidence requirement

### Phase 4: Validation Logic

#### 4.1 Context-Aware Claim Validation
- [ ] Add `getEntityType()` function to determine subject's entity type:
  ```go
  func getEntityType(subjectID string, glxFile *GLXFile) string {
      if _, exists := glxFile.Persons[subjectID]; exists {
          return "person"
      }
      if _, exists := glxFile.Events[subjectID]; exists {
          return "event"
      }
      if _, exists := glxFile.Relationships[subjectID]; exists {
          return "relationship"
      }
      if _, exists := glxFile.Places[subjectID]; exists {
          return "place"
      }
      return ""
  }
  ```
- [ ] Add property vocabulary lookup based on entity type
- [ ] Validate claim against appropriate property vocabulary
- [ ] Emit **warnings** (not errors) for unknown claims

#### 4.4 Entity Properties Validation
- [ ] Add validation for `properties` map on entities:
  - Validate property names against vocabulary (warn for unknown properties).
  - Based on the property's definition in the vocabulary:
    - If **non-temporal**, validate the single value against its `value_type` or `reference_type`.
    - If **temporal**, validate the value's structure. It must be either:
      a) A single value consistent with `value_type`/`reference_type`.
      b) A list where each item is a map containing a `value` and a `date`, and the inner `value` is validated.
  - Check entity references for `reference_type` properties.
  - Emit warnings for unknown properties.
- [ ] Use same validation logic as assertion values

#### 4.2 Participant Validation
- [ ] Validate participant structure when present:
  - **Error** if `participant.person` doesn't exist in persons
  - **Error** if `participant.role` doesn't exist in participant_roles vocabulary
  - **Error** if both `participant` and `value` are present
  - **Error** if both `participant` and `claim` are present
- [ ] When `participant` exists, treat claim as implicitly "participant"

#### 4.3 Property Value Validation
- [ ] Implement `validatePropertyValue()` function:
  - If `reference_type` is set, validate entity reference exists
  - If `value_type` is set, validate value format (future phase - start with references)
  - If neither set, treat as `value_type: string`
- [ ] Validate references for all supported reference types:
  - `persons`, `places`, `events`, `relationships`
  - `sources`, `citations`, `repositories`, `media`
- [ ] Emit errors for invalid references
- [ ] Defer format validation (date, integer, boolean) to future phase

#### 4.4 Update Merge Logic
- [ ] Update property vocabulary merging in `GLXFile.Merge()`
- [ ] Add duplicate detection for property vocabularies

### Phase 5: Update Specification (Source of Truth)

#### 5.0 Create Data Types Specification
- [ ] Create `specification/6-data-types.md` as a central source of truth for all data types.
- [ ] In the new file, document:
  - **Primitive Types:** Define `string`, `integer`, and `boolean`.
  - **Date Format Standard:** Move the detailed explanation of the FamilySearch Normalized Date Format here. Specify the standard except the stillborn modifier, and link to the original.
  - **Temporal Values:** Document the dual structure for temporal properties (single value or list of date-stamped objects).
  - **Reference Types:** Explain that these are string identifiers that are validated against existing entity IDs.
- [ ] Update all other specification documents (`person.md`, `vocabularies.md`, etc.) to remove duplicated data type definitions and link to this new central document instead.

#### 5.1 Update Person Entity Specification
- [ ] Update `specification/4-entity-types/person.md`:
  - **Remove entire `concluded_identity` section**
  - **Add new `properties` section:**
    - Explain properties are vocabulary-defined
    - Document that properties represent "accepted" or "concluded" values
    - Show properties come from `person_properties` vocabulary
    - Explain properties can be set without assertions (for quick recording)
    - Note that assertions provide evidence for properties
  - **Update Properties table:**
    - Remove `concluded_identity` fields
    - Remove `relationships` array
    - Remove `assertions` array
    - Remove `gender` field (moved to properties)
    - Remove `living` field (implied by died_on property)
    - Add `properties` as optional object
    - Document spec fields: `properties`, `notes`, `tags`
  - **Update all examples** to use new structure:
    ```yaml
    persons:
      person-john:
        properties:
          primary_name: "John Smith"
          gender: "male"
          born_on: "1850-01-15"
          born_at: place-leeds
    ```
  - **Add section on property validation:**
    - Properties validated against `person_properties` vocabulary
    - Unknown properties generate warnings
    - Reference validation based on `reference_type` (see Data Types spec)
    - Date format standard is documented in the new Data Types specification.
  - **Document name structure:**
    - Use `given_name` and `family_name` as the standard, separate temporal properties for names.
    - This provides essential structure for searching and sorting while remaining simple.
  - **Update all examples** to show new structure:
    - Person with properties section
    - Assertions with participant field
  - **Remove references** to entity assertion backlinking

#### 5.2 Update Event Entity Specification
- [ ] Update `specification/4-entity-types/event.md`:
  - **Remove `assertions` array** from properties table and examples
  - **Add `properties` section documentation:**
    - Explain event properties (less common than person properties)
    - Show examples: custom event metadata, calculated fields
  - **Update Properties table:**
    - Add `properties` as optional object
  - **Update examples** to remove assertion references
  - **Add note:** Event properties are rare; most event data is structural (type, date, place, participants)

#### 5.3 Update Relationship Entity Specification
- [ ] Update `specification/4-entity-types/relationship.md`:
  - **Align with Event structure:** Replace the `persons` array with a `participants` array.
  - **Explain Participants:** Document that `participants` is now the sole field for defining members of the relationship. Each participant object contains a `person` reference and an optional `role`.
  - **Make `role` optional** to simplify defining basic relationships while allowing for detailed roles where needed.
  - **Update Properties table:**
    - Remove `persons` array.
    - Update `participants` to be the primary member field.
    - Add `properties` as optional object
  - **Update all examples** to use the new `participants` structure, showing both simple (e.g., marriage) and complex (e.g., parent-child with roles) scenarios.

#### 5.4 Update Place Entity Specification
- [ ] Update `specification/4-entity-types/place.md`:
  - **Add `properties` section documentation:**
    - Explain place properties
    - Show examples: historical population, existence dates
  - **Update Properties table:**
    - Add `properties` as optional object
  - **Add examples** showing place properties

#### 5.5 Update Assertion Entity Specification
- [ ] Update `specification/4-entity-types/assertion.md`:
  - **Add `participant` field documentation:**
    - Required fields: `participant.person`
    - Optional fields: `participant.role`, `participant.notes`
    - Explain mutual exclusivity with `claim` and `value`
  - **Update Properties table:**
    - Make `claim` optional (not present if `participant` exists)
    - Make `value` optional (not present if `participant` exists)
    - Add `participant` as optional object
  - **Add "Participant Assertions" section:**
    - Explain when to use participant assertions
    - Show examples of conflicting participant evidence
    - Demonstrate multiple assertions about same event participant
  - **Update all examples:**
    - Show both simple property claims and participant claims
    - Remove references to entity backlinks
  - **Add validation rules:**
    - `participant` and `value` are mutually exclusive
    - `participant` and `claim` are mutually exclusive
    - When `participant` exists, claim is implicitly "participant"

#### 5.6 Update Vocabularies Specification
- [ ] Update `specification/4-entity-types/vocabularies.md`:
  - **Add new section: "Property Vocabularies"**
  - Document all four property vocabularies:
    - `person_properties` - Properties of persons
    - `event_properties` - Properties of events
    - `relationship_properties` - Properties of relationships
    - `place_properties` - Properties of places
  - **Document PropertyDefinition structure:**
    - Required: `label`
    - Optional: `description`, `value_type`, `reference_type`, `temporal`, `custom`
    - `value_type` and `reference_type` are mutually exclusive (cannot both be present)
    - Exactly one of `value_type` or `reference_type` should be specified (or neither, defaults to string)
  - **Explain `temporal` flag:**
    - Explain what the temporal flag means and link to the Data Types spec for the detailed structure of temporal values.
  - **Update Data Type Documentation:**
    - Remove detailed explanations of value types, reference types, and date formats.
    - Replace them with a single reference to the new `specification/6-data-types.md` document.
  - **Note on participants:**
    - Participants are NOT properties
    - Handled by `assertion.participant` field (special case)
    - Not included in property vocabularies
  - **Add examples** for each property vocabulary
  - **Explain context-aware validation:**
    - Claims validated against appropriate vocabulary based on subject type
    - Unknown claims generate warnings
    - Custom properties supported

#### 5.7 Update Core Concepts
- [ ] Update `specification/2-core-concepts.md`:
  - **Update "Assertion-Aware Data Model" section:**
    - Explain properties on entities vs assertions
    - Show that properties can be set without assertions (quick recording)
    - Explain assertions provide evidence for properties
  - **Add property vocabularies** to "Repository-Owned Vocabularies" section
  - **Update all examples** to show new structure:
    - Person with properties section
    - Assertions with participant field
  - **Remove references** to entity assertion backlinking

#### 5.8 Update Archive Organization
- [ ] Update `specification/3-archive-organization.md`:
  - **Update entity structure examples** to show properties section
  - **Add property vocabularies** to vocabulary files list
  - **Update file organization examples**

#### 5.9 Update Introduction
- [ ] Update `specification/1-introduction.md`:
  - **Update examples** to show new person structure with properties
  - **Update feature list** if it mentions concluded_identity or assertion backlinking

#### 5.2 Update Assertion Specification
- [ ] Update `specification/4-entity-types/assertion.md`:
  - Document `participant` field
  - Add examples of participant assertions
  - Explain mutual exclusivity with `claim` and `value`
  - Document implicit claim when participant is present
  - Show examples of conflicting participant evidence
  - Remove references to entity backlinks

#### 5.10 Create Property Vocabulary Specification Files
- [ ] Create `specification/5-standard-vocabularies/person-properties.glx`:
  - Include all standard person properties
  - Add descriptions and value types
  - Mark standard vs custom
- [ ] Create `specification/5-standard-vocabularies/event-properties.glx`:
  - Include standard event properties
  - Include `participant` property with `value_type: participant`
- [ ] Create `specification/5-standard-vocabularies/relationship-properties.glx`:
  - Include standard relationship properties
  - Include `participant` property
- [ ] Create `specification/5-standard-vocabularies/place-properties.glx`:
  - Include standard place properties
- [ ] Update `specification/5-standard-vocabularies/README.md`:
  - Add property vocabularies to the list
  - Link to new vocabulary files
  - Explain property vocabulary system
- [ ] Update `specification/5-standard-vocabularies/embed.go`:
  - Add embed directives for new vocabulary files

### Phase 6: Example Updates

#### 6.1 Update Example Files
- [ ] Update all person files in `docs/examples/`:
  - Remove `concluded_identity` nesting
  - Move `gender` into `properties` section
  - Remove `living` field (implied by died_on)
  - Add `properties` section with: `primary_name`, `gender`, `born_on`, `born_at`, etc.
  - Remove `relationships` arrays
  - Remove `assertions` arrays
- [ ] Update assertion examples:
  - Add participant assertion examples
  - Show both simple and participant claims
  - Demonstrate conflicting evidence scenarios
- [ ] Create a new example file `docs/examples/complex-scenarios/surrogacy.glx`:
  - This file will demonstrate modeling nuanced family structures like surrogacy.
  - It will require adding custom `participant_roles` (e.g., `birth-mother`, `legal-father`) to its embedded vocabulary to properly describe the scenario.
- [ ] Update event examples:
  - Remove `assertions` arrays

#### 6.2 Update Example Documentation
- [ ] Update `docs/examples/README.md` with new structure
- [ ] Update `docs/examples/complete-family/README.md`
- [ ] Update `docs/examples/basic-family/README.md`

### Phase 7: Testing

#### 7.1 Add Test Files
- [ ] Create `glx/testdata/valid/`:
  - `assertion-with-participant.glx`
  - `assertion-simple-claim.glx`
  - `person-with-properties.glx`
  - `assertion-participant-conflicting.glx`
- [ ] Create `glx/testdata/invalid/`:
  - `assertion-participant-and-value.glx` (mutually exclusive - error)
  - `assertion-participant-and-claim.glx` (mutually exclusive - error)
  - `assertion-participant-invalid-person.glx` (broken reference - error)
  - `assertion-participant-invalid-role.glx` (broken reference - error)
  - `assertion-unknown-claim.glx` (should warn, not error)

#### 7.2 Update Validation Tests
- [ ] Update `glx/validate_test.go`:
  - Test context-aware claim validation
  - Test participant validation (errors for invalid references)
  - Test mutual exclusivity (participant vs value/claim)
  - Test warning generation for unknown claims
  - Test property vocabulary loading

#### 7.3 Update Data Generation
- [ ] Update `lib/datagen.go`:
  - Generate property vocabularies
  - Use flat person structure (no concluded_identity)
  - Generate participant assertions
  - Remove assertion backlinking

### Phase 8: CLI Updates

#### 8.1 Update Vocabulary Loading
- [ ] Update `glx/validator.go` `LoadArchiveVocabularies()`:
  - Load property vocabularies from any .glx file
  - Merge property vocabularies like other vocabularies

#### 8.2 Update Validation Output
- [ ] Distinguish between warnings and errors in output
- [ ] Show warnings for unknown claims but don't fail validation
- [ ] Show errors for invalid participant references

## Technical Specifications

### Struct Definitions

```go
// Property vocabulary definition
type PropertyDefinition struct {
    Label         string   `yaml:"label"`
    Description   string   `yaml:"description,omitempty"`
    ValueType     string   `yaml:"value_type,omitempty"`      // string, date, integer, boolean (mutually exclusive with reference_type)
    ReferenceType string   `yaml:"reference_type,omitempty"`  // persons, places, events, relationships, etc. (mutually exclusive with value_type)
    Temporal      *bool    `yaml:"temporal,omitempty"`        // Property can change over time
    Custom        *bool    `yaml:"custom,omitempty"`
}

// TemporalValue represents a single entry in the history of a temporal property.
// It is used when a temporal property is represented as a list.
type TemporalValue struct {
    Value interface{} `yaml:"value"`          // The actual property value, conforming to value_type or reference_type
    Date  string      `yaml:"date,omitempty"` // FamilySearch normalized date string
}

// Updated Assertion
type Assertion struct {
    Subject     string                `yaml:"subject" refType:"persons,events,relationships,places"`
    Claim       string                `yaml:"claim,omitempty"`  // Not present if participant exists
    Value       string                `yaml:"value,omitempty"`  // Not present if participant exists
    Participant *AssertionParticipant `yaml:"participant,omitempty"`  // Not present if value exists
    Confidence  string                `yaml:"confidence,omitempty" refType:"confidence_levels"`
    Sources     []string              `yaml:"sources,omitempty" refType:"sources"`
    Citations   []string              `yaml:"citations,omitempty" refType:"citations"`
    Notes       string                `yaml:"notes,omitempty"`
    Tags        []string              `yaml:"tags,omitempty"`
}

type AssertionParticipant struct {
    Person string `yaml:"person" refType:"persons"`
    Role   string `yaml:"role,omitempty" refType:"participant_roles"`
    Notes  string `yaml:"notes,omitempty"`
}

// Person with segregated properties (no backlinks, minimal spec fields)
type Person struct {
    // Properties contains all concluded data for the person.
    // The keys are defined in the person_properties vocabulary.
    // For properties marked as temporal, the value can be a single primitive
    // OR a list of TemporalValue objects to capture its history.
    Properties map[string]interface{} `yaml:"properties,omitempty"`
    Notes      string                 `yaml:"notes,omitempty"`
    Tags       []string               `yaml:"tags,omitempty"`
}

// Event with segregated properties
type Event struct {
    Type         string                 `yaml:"type" refType:"event_types"`
    PlaceID      string                 `yaml:"place,omitempty" refType:"places"`
    Date         *EventDate             `yaml:"date,omitempty"`
    Participants []EventParticipant     `yaml:"participants,omitempty"`
    // Properties contains all concluded data for the event.
    // For properties marked as temporal, the value can be a single primitive
    // OR a list of TemporalValue objects to capture its history.
    Properties   map[string]interface{} `yaml:"properties,omitempty"`
    Description  string                 `yaml:"description,omitempty"`
    Notes        string                 `yaml:"notes,omitempty"`
    Tags         []string               `yaml:"tags,omitempty"`
}

// Relationship with segregated properties
type Relationship struct {
    Type         string                    `yaml:"type" refType:"relationship_types"`
    Participants []RelationshipParticipant `yaml:"participants"`
    StartEvent   string                    `yaml:"start_event,omitempty" refType:"events"`
    EndEvent     string                    `yaml:"end_event,omitempty" refType:"events"`
    Properties   map[string]interface{}    `yaml:"properties,omitempty"`
    Description  string                    `yaml:"description,omitempty"`
    Notes        string                    `yaml:"notes,omitempty"`
    Tags         []string                  `yaml:"tags,omitempty"`
}

// Place with segregated properties
type Place struct {
    Name             string                 `yaml:"name"`
    ParentID         string                 `yaml:"parent,omitempty" refType:"places"`
    Type             string                 `yaml:"type,omitempty" refType:"place_types"`
    AlternativeNames []AlternativeName      `yaml:"alternative_names,omitempty"`
    Latitude         *float64               `yaml:"latitude,omitempty"`
    Longitude        *float64               `yaml:"longitude,omitempty"`
    // Properties contains all concluded data for the place.
    // For properties marked as temporal, the value can be a single primitive
    // OR a list of TemporalValue objects to capture its history.
    Properties       map[string]interface{} `yaml:"properties,omitempty"`
    Notes            string                 `yaml:"notes,omitempty"`
    Tags             []string               `yaml:"tags,omitempty"`
}
```

### Validation Rules

**Hard Errors:**
- `participant.person` references non-existent person
- `participant.role` references non-existent role (if role is present)
- Both `participant` and `value` present
- Both `participant` and `claim` present
- Missing required fields (`version`, `subject`)
- Missing evidence (`citations` or `sources`)
- Invalid entity references in properties (based on `value_type`)
- Duplicate IDs

**Soft Warnings:**
- Unknown claim (not in property vocabulary)
- Unknown property on entity (not in property vocabulary) - allows on-the-fly property addition
- Property vocabulary doesn't exist for entity type
- Value type mismatch (future phase)

### Naming Conventions

**Go Struct Fields:**
- Use descriptive names: `PrimaryName`, `BornOn`, `DiedOn`
- Use `ID` suffix for references: `PersonID`, `PlaceID`

**YAML Tags:**
- Snake case: `primary_name`, `born_on`, `died_on`
- Singular entity names for references: `person`, `place`

**Vocabulary Keys:**
- Snake case with entity type: `person_properties`, `event_properties`
- Match property names: `born_on`, `occurred_on`

**refType Tags:**
- Plural entity types: `persons`, `places`, `events`
- Vocabulary names: `participant_roles`, `confidence_levels`

## Value Types and Reference Types

Property definitions use two mutually exclusive fields:

### Value Types (Primitive Data)
- `string` - Free text (default if neither specified)
- `date` - ISO date or fuzzy date string
- `integer` - Numeric value
- `boolean` - True/false value

### Reference Types (Entity References)
- `persons` - Reference to person entity
- `places` - Reference to place entity
- `events` - Reference to event entity
- `relationships` - Reference to relationship entity
- `sources` - Reference to source entity
- `citations` - Reference to citation entity
- `repositories` - Reference to repository entity
- `media` - Reference to media entity

### Special Case: Participants
Participants are NOT properties. They are handled by:
1. **Structural**: `event.participants` or `relationship.participants` arrays
2. **Evidential**: `assertion.participant` field (special assertion structure)

Participant assertions do not use property vocabularies.

## Migration Notes

Since no one is using the format yet:
- No backwards compatibility concerns
- Breaking changes are acceptable
- Focus on getting the design right

## Success Criteria

- [ ] Property vocabularies load and validate correctly
- [ ] Participant assertions work as designed
- [ ] Context-aware claim validation works
- [ ] Warnings generated for unknown claims
- [ ] Errors generated for invalid participant references
- [ ] Person entities use flat structure (no concluded_identity)
- [ ] Assertion backlinking removed from all entities
- [ ] All tests pass
- [ ] Documentation is complete
- [ ] Examples demonstrate new features


## Decisions Made (Updated)

1. **Claim field when participant present:** NOT present (mutually exclusive)
2. **Validation strictness:** Warnings for unknown claims, errors for invalid participant references
3. **Properties on entities:** Segregated in `properties` section (Option A)
4. **Backwards compatibility:** Not a concern - breaking changes acceptable
5. **Value type validation:** Defer to future phase, focus on references for now
6. **Gender field:** Moved to properties (vocabulary-defined)
7. **Living field:** Removed - implied by presence/absence of `died_on` property
8. **Temporal properties:** Add `temporal` flag to PropertyDefinition for time-varying properties
9. **Relationship backlinking:** Removed from Person entity
10. **Version field:** Removed from all entities - versioning handled at archive level, not per-entity
11. **Property vocabulary validation:** Validate like other vocabularies (require `label`)
12. **Standard properties scope:** Include common genealogical properties for 100% GEDCOM coverage before release
13. **Properties map validation:** Validate against vocabulary using same logic as assertion values

## Next Steps

1. Review and approve this plan
2. Begin Phase 1: Create property vocabulary files
3. Implement Phase 2: Update entity schemas and structs
4. Implement Phase 3: Update assertion schema and struct
5. Implement Phase 4: Add validation logic
6. Continue through remaining phases
