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

properties:
  given_name: "John"
  family_name: "Smith"
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

