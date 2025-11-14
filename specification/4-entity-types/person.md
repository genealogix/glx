# Person Entity

[← Back to Entity Types](README.md)

## Overview

The Person entity represents an individual in the family archive. Person entities can be stored in any `.glx` file in the repository under the `persons` key.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in persons/ directory)
persons:
  person-john-smith-1850:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"
```

**Key Points:**
- Entity ID is the map key (`person-john-smith-1850`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Required Fields

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Recommended formats:
  - Descriptive: `person-john-smith-1850`, `person-mary-jones`
  - Random hex: `person-a1b2c3d4` (for collaboration)
  - Sequential: `person-001`, `person-002`

### `version`

- Type: String
- Format: `{major}.{minor}`
- Required: Yes
- Description: Schema version

Example:
```yaml
persons:
  person-john-smith:
    version: "1.0"
```

## Optional Fields

### `concluded_identity`

- Type: Object
- Required: No
- Description: The researcher's conclusion about this person's identity

Structure:
```yaml
concluded_identity:
  primary_name: String
  gender: String
  living: Boolean
```

Example:
```yaml
concluded_identity:
  primary_name: "Margaret Eleanor Smith"
  gender: female
  living: false
```

### `assertions`

- Type: Array of Strings
- Required: No
- Description: References to assertion files about this person

Example:
```yaml
assertions:
  - biographical/birth/assert-birth-123
  - biographical/death/assert-death-456
```

Validation Rules:
- Each reference MUST point to a valid assertion file
- Paths are relative to `assertions/` directory
- File extension `.glx` is implied

## Complete Example

```yaml
# persons/person-margaret-smith.glx
persons:
  person-margaret-smith-1825:
    version: "1.0"
    concluded_identity:
      primary_name: "Margaret Eleanor Smith"
      gender: female
      living: false
    assertions:
      - assert-birth-margaret
      - assert-death-margaret
    relationships:
      - rel-parent-child-margaret
      - rel-marriage-margaret
    notes: |
      Family tradition says she was named after her grandmother.
      Need to verify with census records.
    tags:
      - maternal-line
      - smith-family
      - ohio-branch
```

## Schema Reference

See [person.schema.json](../schema/v1/person.schema.json) for the
complete JSON Schema definition.

## See Also

- [Assertion Entity](assertion.md)
- [Relationship Entity](relationship.md)
- [Provenance Tracking](../2-core-concepts.md#provenance-and-confidence)


