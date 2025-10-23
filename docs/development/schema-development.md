# Schema Development Guide

This guide explains how to develop and maintain JSON Schemas for GENEALOGIX entity validation.

## Schema Overview

GENEALOGIX uses JSON Schema (Draft 07) to define the structure and validation rules for all entity types.

### Schema Locations

**Schema Files:**
```
schema/
├── meta/
│   └── schema.schema.json     # Schema for schemas
└── v1/
    ├── person.schema.json     # Person entity
    ├── event.schema.json      # Event entity
    ├── place.schema.json      # Place entity
    ├── source.schema.json     # Source entity
    ├── citation.schema.json   # Citation entity
    ├── repository.schema.json # Repository entity
    ├── assertion.schema.json  # Assertion entity
    ├── relationship.schema.json # Relationship entity
    └── media.schema.json      # Media entity
```

**Schema Metadata:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schema.genealogix.org/v1/person",
  "title": "Person",
  "description": "An individual in the family archive",
  "version": "1.0"
}
```

## Schema Structure

### Common Schema Elements

**Every entity schema includes:**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schema.genealogix.org/v1/person",
  "title": "Person",
  "description": "An individual in the family archive",
  "type": "object",
  "required": ["id", "version", "type"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^person-[a-f0-9]{8}$",
      "description": "Unique identifier for this person"
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$",
      "description": "Schema version"
    },
    "type": {
      "type": "string",
      "enum": ["person"],
      "description": "Entity type"
    }
  },
  "additionalProperties": false
}
```

### Schema Components

**1. Core Fields:**
- `id`: Unique identifier with type prefix
- `version`: Schema version (e.g., "1.0")
- `type`: Entity type (e.g., "person")

**2. Entity-Specific Fields:**
- Defined in `properties` object
- Validation rules for each field
- Optional vs required field specification

**3. Validation Rules:**
- `required`: Array of required field names
- `pattern`: Regular expression validation
- `enum`: Allowed values
- `format`: Standard format validation (date, uri, etc.)

## Schema Development Process

### 1. Schema Design

**Start with specification:**
```yaml
# From specification
person:
  id: person-{8hex}
  version: "1.0"
  type: person
  name:
    given: string (required)
    surname: string (required)
    display: string (required)
    nickname: string (optional)
  birth: date (optional)
  death: date (optional)
```

**Convert to JSON Schema:**
```json
{
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^person-[a-f0-9]{8}$"
    },
    "name": {
      "type": "object",
      "properties": {
        "given": {"type": "string"},
        "surname": {"type": "string"},
        "display": {"type": "string"},
        "nickname": {"type": "string"}
      },
      "required": ["given", "surname", "display"],
      "additionalProperties": false
    }
  },
  "required": ["id", "version", "type", "name"]
}
```

### 2. Schema Implementation

**Create new schema file:**
```bash
# Create schema file
vim schema/v1/person.schema.json

# Add standard header
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schema.genealogix.org/v1/person",
  "title": "Person",
  "description": "An individual in the family archive"
}
```

**Define properties:**
```json
{
  "type": "object",
  "required": ["id", "version", "type", "name"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^person-[a-f0-9]{8}$",
      "description": "Unique identifier for this person"
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$",
      "description": "Schema version"
    },
    "type": {
      "type": "string",
      "enum": ["person"],
      "description": "Entity type"
    },
    "name": {
      "type": "object",
      "properties": {
        "given": {
          "type": "string",
          "description": "Given name or first name"
        },
        "surname": {
          "type": "string",
          "description": "Family name or surname"
        },
        "display": {
          "type": "string",
          "description": "Full display name"
        },
        "nickname": {
          "type": "string",
          "description": "Common nickname or familiar name"
        }
      },
      "required": ["given", "surname", "display"],
      "additionalProperties": false
    }
  },
  "additionalProperties": false
}
```

### 3. Schema Testing

**Validate schema syntax:**
```bash
# Test schema compilation
ajv compile -s schema/v1/person.schema.json

# Test against example files
glx validate examples/complete-family/persons/

# Test with minimal example
echo '{
  "id": "person-a1b2c3d4",
  "version": "1.0",
  "type": "person",
  "name": {
    "given": "John",
    "surname": "Smith",
    "display": "John Smith"
  }
}' | ajv validate -s schema/v1/person.schema.json -
```

## Schema Validation Rules

### ID Pattern Validation

**Standard ID patterns:**
```json
{
  "person": "^person-[a-f0-9]{8}$",
  "event": "^event-[a-f0-9]{8}$",
  "place": "^place-[a-f0-9]{8}$",
  "source": "^source-[a-f0-9]{8}$",
  "citation": "^citation-[a-f0-9]{8}$",
  "repository": "^repository-[a-f0-9]{8}$",
  "assertion": "^assertion-[a-f0-9]{8}$",
  "relationship": "^rel-[a-f0-9]{8}$",
  "media": "^media-[a-f0-9]{8}$"
}
```

**Pattern explanation:**
- `^`: Start of string
- `{type}-`: Entity type prefix
- `[a-f0-9]{8}`: Exactly 8 lowercase hex characters
- `$`: End of string

### Date Validation

**Date format patterns:**
```json
{
  "date": {
    "type": "string",
    "pattern": "^\\d{4}(-\\d{2}(-\\d{2})?)?$|^\\d{4}\\?(-\\d{2}(-\\d{2})?)?$|^\\d{4}/\\d{4}$",
    "description": "Date in ISO format or uncertain/between formats"
  }
}
```

**Supported date formats:**
- `1850-01-15` (complete date)
- `1850-01` (year and month)
- `1850` (year only)
- `1850?` (uncertain year)
- `1849/1850` (between years)

### Reference Validation

**Cross-entity references:**
```json
{
  "birth_place": {
    "type": "string",
    "pattern": "^place-[a-f0-9]{8}$",
    "description": "Reference to place entity"
  },
  "citations": {
    "type": "array",
    "items": {
      "type": "string",
      "pattern": "^citation-[a-f0-9]{8}$"
    },
    "description": "References to citation entities"
  }
}
```

## Schema Testing

### 1. Schema Compilation Tests

**Test all schemas compile:**
```bash
# Test meta schema
ajv compile -s schema/meta/schema.schema.json

# Test all entity schemas
find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;

# Test schema references
ajv compile -r schema/v1/*.schema.json -s schema/v1/person.schema.json
```

### 2. Example Validation Tests

**Test schemas against examples:**
```bash
# Test person schema
for file in examples/complete-family/persons/*.glx; do
  echo "Testing $file"
  ajv validate -s schema/v1/person.schema.json "$file"
done

# Test all entity types
for entity in person event place source citation repository assertion relationship; do
  echo "Testing $entity schema"
  find examples -name "*.glx" -exec ajv validate -s "schema/v1/$entity.schema.json" {} \;
done
```

### 3. Edge Case Testing

**Test boundary conditions:**
```bash
# Test minimal valid files
ajv validate -s schema/v1/person.schema.json test-suite/valid/person-minimal.glx

# Test maximum valid files
ajv validate -s schema/v1/person.schema.json test-suite/valid/person-complete.glx

# Test invalid files
ajv validate -s schema/v1/person.schema.json test-suite/invalid/person-missing-id.glx
```

## Schema Versioning

### Version Management

**Schema versioning strategy:**
```json
{
  "version": "1.0",
  "schema_version": "1.0.0",
  "extends": "1.0-base",
  "compatibility": "backwards"
}
```

**Version fields:**
- `version`: Entity format version (e.g., "1.0")
- `schema_version`: Schema specification version (e.g., "1.0.0")
- `extends`: Previous version this extends
- `compatibility`: Compatibility level (backwards, full, breaking)

### Backwards Compatibility

**Adding optional fields:**
```json
{
  "properties": {
    "new_field": {
      "type": "string",
      "description": "New optional field"
    }
  },
  "compatibility": "backwards"
}
```

**Extending enums:**
```json
{
  "properties": {
    "type": {
      "enum": ["person", "person_extended"],  // Added new value
      "default": "person"
    }
  }
}
```

### Breaking Changes

**When breaking changes are needed:**
```json
{
  "version": "2.0",
  "breaking_changes": [
    "Removed deprecated field 'old_name'",
    "Changed ID pattern to include timestamp",
    "Made 'nickname' field required"
  ],
  "migration_guide": "See migration/v1-to-v2.md"
}
```

## Schema Documentation

### Schema Comments

**Document complex validation rules:**
```json
{
  "date": {
    "type": "string",
    "pattern": "^\\d{4}(-\\d{2}(-\\d{2})?)?$",
    "description": "Date in YYYY-MM-DD format, YYYY-MM, or YYYY",
    "examples": ["1850-01-15", "1850-01", "1850"]
  }
}
```

### Field Descriptions

**Provide clear field documentation:**
```json
{
  "name": {
    "type": "object",
    "description": "Person's name information",
    "properties": {
      "given": {
        "type": "string",
        "description": "Given name or first name(s)",
        "examples": ["John", "Mary Anne", "Jean-Pierre"]
      },
      "surname": {
        "type": "string",
        "description": "Family name, surname, or last name",
        "examples": ["Smith", "O'Connor", "García López"]
      },
      "display": {
        "type": "string",
        "description": "Full display name as it should appear",
        "examples": ["John Smith", "Mary Anne O'Connor"]
      }
    }
  }
}
```

## Schema Validation Tools

### 1. AJV CLI

**Install and use AJV:**
```bash
npm install -g ajv-cli

# Test single schema
ajv compile -s schema/v1/person.schema.json

# Test with data
ajv validate -s schema/v1/person.schema.json data.glx

# Test multiple schemas
ajv compile -r schema/v1/*.schema.json -s schema/v1/person.schema.json
```

### 2. JSON Schema Validators

**Online validators:**
- JSON Schema Validator (json-schema-validator.herokuapp.com)
- JSON Schema Lint (jsonschemalint.com)
- Schema Validation (jsonschema.net)

**Command line tools:**
```bash
# Python
pip install jsonschema
jsonschema -i data.json schema.json

# Node.js
npm install -g json-schema-cli
json-schema-cli schema.json data.json
```

### 3. Custom Validation

**GLX CLI validation:**
```bash
# Validate single file
glx validate persons/person-example.glx

# Validate directory
glx validate examples/complete-family/

# Validate with schema checking
glx check-schemas
```

## Schema Best Practices

### 1. Consistent Patterns

**Use consistent validation patterns:**
```json
// ✅ Consistent ID patterns
{
  "person_id": "^person-[a-f0-9]{8}$",
  "event_id": "^event-[a-f0-9]{8}$",
  "place_id": "^place-[a-f0-9]{8}$"
}

// ✅ Consistent date patterns
{
  "date": "^\\d{4}(-\\d{2}(-\\d{2})?)?$",
  "birth_date": "^\\d{4}(-\\d{2}(-\\d{2})?)?$",
  "death_date": "^\\d{4}(-\\d{2}(-\\d{2})?)?$"
}
```

### 2. Clear Error Messages

**Provide helpful validation errors:**
```json
{
  "id": {
    "type": "string",
    "pattern": "^person-[a-f0-9]{8}$",
    "description": "Person ID must be 'person-' followed by 8 lowercase hex characters",
    "errorMessage": "Person ID must match pattern: person-12345678 (8 hex characters)"
  }
}
```

### 3. Performance Optimization

**Optimize for validation speed:**
```json
// ✅ Efficient patterns
{
  "id": {
    "type": "string",
    "pattern": "^person-[a-f0-9]{8}$"  // Simple pattern
  }
}

// ❌ Inefficient patterns
{
  "id": {
    "type": "string",
    "pattern": "^person-(?:[a-f0-9]{8})$"  // Unnecessary groups
  }
}
```

## Schema Maintenance

### 1. Schema Updates

**Process for schema changes:**

1. **Propose change**: Create GitHub issue or RFC
2. **Update specification**: Document new requirements
3. **Modify schema**: Update JSON Schema files
4. **Add tests**: Create validation tests
5. **Update examples**: Modify example files
6. **Version update**: Bump version if needed

**Example update process:**
```bash
# 1. Update schema
vim schema/v1/person.schema.json

# 2. Test schema
ajv compile -s schema/v1/person.schema.json

# 3. Update tests
vim test-suite/valid/person-updated.glx
vim test-suite/invalid/person-old-format.glx

# 4. Test examples
glx validate examples/complete-family/persons/

# 5. Update documentation
vim specification/4-entity-types/person.md
```

### 2. Schema Validation

**Continuous schema validation:**
```bash
# Test all schemas compile
find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;

# Test schema consistency
# Check for duplicate IDs
# Verify reference patterns
# Validate against meta schema
```

### 3. Schema Documentation

**Keep documentation current:**
```bash
# Update schema docs when schemas change
# Document new validation rules
# Update examples to match schema
# Add migration notes for breaking changes
```

## Advanced Schema Features

### 1. Conditional Validation

**Validate fields based on other fields:**
```json
{
  "allOf": [
    {
      "if": {
        "properties": {"relationship_type": {"const": "adoption"}}
      },
      "then": {
        "required": ["adoption_date", "adoption_place"]
      }
    }
  ]
}
```

### 2. Cross-Schema References

**Reference other schemas:**
```json
{
  "person_reference": {
    "$ref": "person.schema.json#/properties/id",
    "description": "Reference to a person entity"
  }
}
```

### 3. Custom Validation Keywords

**Extend validation with custom rules:**
```json
{
  "birth_before_death": {
    "type": "object",
    "properties": {
      "birth": {"type": "string"},
      "death": {"type": "string"}
    },
    "customValidation": "birth_date_before_death_date"
  }
}
```

## Troubleshooting Schemas

### Common Schema Issues

**1. Pattern mismatches:**
```json
// Problem: Pattern too restrictive
"pattern": "^person-[0-9]{8}$"  // Only numbers!

// Solution: Allow hex characters
"pattern": "^person-[a-f0-9]{8}$"
```

**2. Missing required fields:**
```json
// Problem: Schema requires field not in specification
"required": ["id", "version", "type", "missing_field"]

// Solution: Check specification requirements
"required": ["id", "version", "type"]
```

**3. Reference validation:**
```json
// Problem: References don't match patterns
"pattern": "^person-[a-f0-9]{8}$"  // Person schema
"pattern": "^event-[a-f0-9]{8}$"   // Event schema

// Solution: Use correct pattern for each entity type
```

### Schema Debugging

**Debug validation errors:**
```bash
# Test with simple data
echo '{"id": "person-a1b2c3d4", "version": "1.0", "type": "person", "name": {"given": "John", "surname": "Smith", "display": "John Smith"}}' | ajv validate -s schema/v1/person.schema.json -

# Check schema compilation
ajv compile -s schema/v1/person.schema.json

# Validate against examples
glx validate examples/minimal/persons/
```

### Performance Issues

**Optimize schema performance:**
```bash
# Profile schema validation
time ajv validate -s schema/v1/person.schema.json large-person-file.glx

# Check for inefficient patterns
# Use atomic groups where possible
# Avoid unnecessary backtracking
```

This schema development guide ensures that GENEALOGIX schemas are well-designed, thoroughly tested, and properly maintained.
