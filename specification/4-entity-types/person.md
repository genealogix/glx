# Person Entity

[← Back to Entity Types](README.md)

## Overview

The Person entity represents an individual in the family archive. Each
person is stored in a separate `.glx` file within the `persons/` directory.

## File Location

```
persons/
└── {person-id}.glx
```

## Required Fields

The following fields MUST be present in every Person file:

### `id`

- Type: String
- Format: `person-{uuid}`
- Required: Yes
- Description: Unique identifier for this person

Example:
```yaml
id: person-a1b2c3d4
```

Validation Rules:
- MUST start with `person-`
- MUST be followed by 8 hexadecimal characters
- MUST be unique within the archive

### `version`

- Type: String
- Format: `{major}.{minor}`
- Required: Yes
- Description: Schema version for this file

Example:
```yaml
version: 1.0
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
# persons/person-a1b2c3d4.glx
id: person-a1b2c3d4
version: 1.0

concluded_identity:
  primary_name: "Margaret Eleanor Smith"
  gender: female
  living: false

assertions:
  - biographical/birth/assert-birth-001
  - biographical/death/assert-death-001

relationships:
  - rel-parent-child-001
  - rel-marriage-002

created_at: "2024-01-15T10:30:00Z"
created_by: user-xyz
modified_at: "2024-03-20T14:15:00Z"
modified_by: user-abc

notes: |
  Family tradition says she was named after her grandmother.
  Need to verify with census records.

tags:
  - maternal-line
  - smith-family
  - ohio-branch
```

## Schema Reference

See [person.schema.json](../../schema/v1/person.schema.json) for the
complete JSON Schema definition.

## See Also

- [Assertion Entity](assertion.md)
- [Relationship Entity](relationship.md)
- [Provenance Tracking](../5-data-model/provenance-tracking.md)


