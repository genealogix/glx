---
title: Minimal Archive Example
description: The smallest valid GENEALOGIX archive with one person
layout: doc
---

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
persons:
  person-abc123:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
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

