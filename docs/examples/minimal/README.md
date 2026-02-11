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
├── persons/
│   └── person-abc123.glx
├── vocabularies/           # Symlinks to standard vocabularies
└── README.md
```

## Files

### persons/person-abc123.glx

```yaml
persons:
  person-abc123:
    properties:
      name:
        value: "Test Person"
        fields:
          given: "Test"
          surname: "Person"
```

## Validation

```bash
glx validate .
# ✓ All files valid
```

## What This Demonstrates

- Minimum required file structure
- Simplest valid person entity

## Next Steps

See [basic-family](../basic-family/) for a more complete example
with relationships and assertions.

