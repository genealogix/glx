# Minimal Archive Example

The smallest valid GENEALOGIX archive with one person.

## Structure

```
minimal/
├── .glx-archive/
│   └── schema-version.glx
├── persons/
│   └── person-abc123.glx
└── README.md
```

## Files

### persons/person-abc123.glx

```yaml
id: person-abc123
version: 1.0

concluded_identity:
  primary_name: "John Smith"
  
created_at: "2024-01-15T10:30:00Z"
created_by: example-user
```

## Validation

```bash
glx validate .
# ✓ All files valid
```

## What This Demonstrates

- Minimum required file structure
- Simplest valid person entity
- Schema version configuration

## Next Steps

See [basic-family](../basic-family/) for a more complete example
with relationships and assertions.


