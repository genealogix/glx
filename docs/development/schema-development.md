# Schema Development Guide

This guide explains how to develop and maintain JSON Schemas for GENEALOGIX entity validation.

## Schema Overview

GENEALOGIX uses JSON Schema (Draft 07) to define structure and validation rules for all entity types.

### Schema Locations

```
specification/schema/v1/
├── person.schema.json
├── event.schema.json
├── place.schema.json
├── source.schema.json
├── citation.schema.json
├── repository.schema.json
├── assertion.schema.json
├── relationship.schema.json
├── media.schema.json
└── vocabularies/
    ├── event-types.schema.json
    ├── relationship-types.schema.json
    └── ... (other vocabularies)
```

Schemas are embedded in the Go binary via `specification/schema/v1/embed.go`.

## Schema Structure

### Required Elements

Every entity schema must include:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schema.genealogix.io/v1/person",
  "title": "Person",
  "description": "An individual in the family archive",
  "type": "object",
  "required": ["properties"],
  "properties": {
    "properties": {
      "type": "object",
      "description": "Entity-specific properties",
      "additionalProperties": true
    }
  },
  "additionalProperties": false
}
```

**Key Points:**
- `$schema` and `$id` are required (validated by `glx check-schemas`)
- Use `additionalProperties: false` for strict validation
- Entity IDs come from YAML map keys, not from an `id` field

## Converting Specification to Schema

### Process

1. **Read the entity specification** in `specification/4-entity-types/{entity}.md`
2. **Identify required vs optional fields** from the properties table
3. **Define field types and patterns** based on descriptions
4. **Add validation rules** for references, enums, and formats
5. **Test against examples** in `docs/examples/`

### Field Type Mapping

| Specification Type | JSON Schema Type | Notes |
|-------------------|------------------|-------|
| string | `"type": "string"` | Add pattern if format-specific |
| integer | `"type": "integer"` | Add min/max if bounded |
| boolean | `"type": "boolean"` | Use pointer in Go for optional |
| array | `"type": "array"` | Define items schema |
| object | `"type": "object"` | Define nested properties |
| reference | `"type": "string", "pattern": "^[a-zA-Z0-9-]{1,64}$"` | Entity reference |

### Reference Fields

Reference fields follow a consistent pattern:

```json
{
  "source": {
    "type": "string",
    "pattern": "^[a-zA-Z0-9-]{1,64}$",
    "description": "Reference to Source entity"
  }
}
```

**Naming Convention:**
- Use singular entity name (e.g., `source`, not `source_id`)
- Pattern allows alphanumeric and hyphens, 1-64 characters
- Description states what entity type is referenced

### Vocabulary References

Vocabulary references (types, roles, etc.) use the same pattern:

```json
{
  "type": {
    "type": "string",
    "description": "Event type from event_types vocabulary"
  }
}
```

Vocabulary values are validated at runtime, not in schema.

## Schema Validation

### Testing Schemas

```bash
# Validate schema files have required metadata
go test ./glx/... -run TestRunCheckSchemas

# Validate all examples against schemas
go test ./glx/... -run TestExamples

# Run full validation test suite
go test ./glx/...
```

### Common Validation Rules

- **Entity IDs**: Alphanumeric and hyphens, 1-64 chars
- **References**: Must use singular entity names
- **Arrays**: Use `uniqueItems: true` for ID arrays
- **Enums**: Only for closed vocabularies (avoid when possible)

## Schema Updates

### Update Process

1. **Update specification** in `specification/4-entity-types/`
2. **Modify JSON schema** in `specification/schema/v1/`
3. **Update Go structs** in `lib/types.go` (add YAML and refType tags)
4. **Update examples** to match new schema
5. **Run tests** to verify changes
6. **Update documentation** as needed

### Breaking Changes

Breaking changes require:
- Major version bump (e.g., `1.0` → `2.0`)
- Migration guide

## Embedded Schemas

Schemas are embedded in the Go binary:

```go
// specification/schema/v1/embed.go
//go:embed person.schema.json
var PersonSchema []byte

var EntitySchemas = map[string][]byte{
    "person": PersonSchema,
    // ...
}
```

After modifying schemas, rebuild the CLI:
```bash
cd glx
go build
```

## Vocabulary Schemas

Vocabulary schemas define controlled vocabularies:

```json
{
  "type": "object",
  "additionalProperties": {
    "type": "object",
    "required": ["label"],
    "properties": {
      "label": {
        "type": "string",
        "description": "Human-readable label"
      },
      "description": {
        "type": "string",
        "description": "Detailed description"
      },
      "custom": {
        "type": "boolean",
        "description": "Whether this is a custom entry"
      }
    }
  }
}
```

Vocabulary entries can be defined in any `.glx` file in the archive.

## Best Practices

### Schema Design

- Keep schemas simple and focused
- Use clear, descriptive field names
- Provide helpful descriptions
- Avoid overly restrictive patterns
- Allow for extensibility

### Validation Strategy

- JSON Schema validates structure and types
- Go struct validation handles references
- Fail fast on duplicates
- Report all reference errors at once

### Documentation

- Keep schema docs in sync with specification
- Update examples when schemas change
- Document validation rules clearly
- Link to relevant specification sections

## See Also

- [Entity Types Specification](../../specification/4-entity-types/README.md)
- [Archive Organization](../../specification/3-archive-organization.md)
- [Testing Guide](testing-guide.md)
- [CLI Validator](../../glx/validator.go)
