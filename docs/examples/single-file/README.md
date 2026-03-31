---
title: Single-File Archive Example
description: Complete GENEALOGIX archive in a single file for simple sharing and backup
layout: doc
---

# Single-File GENEALOGIX Example

This example demonstrates a complete GENEALOGIX archive in a single file.

## Structure

All entities are in one file: `archive.glx`

```
single-file/
├── archive.glx    # All entities in one file
└── README.md
```

## File Contents

The archive.glx file contains all entity types:
- persons (2): John Smith, Mary Brown
- relationships (1): Marriage
- events (4): John's birth, marriage, two deaths
- places (3): England, Yorkshire, Leeds (hierarchical)
- sources (1): Parish register
- citations (1): Birth certificate citation
- repositories (1): Leeds Library
- assertions (1): Birth date assertion

## Benefits of Single-File Format

- **Simple**: One file to manage and backup
- **Portable**: Easy to share via email or file transfer
- **Quick**: See entire archive structure at a glance
- **Good for**: Personal research, small families, quick exports

## When to Use

Use single-file format when:
- Working solo on personal research
- Archive has fewer than 50-100 entities
- Don't need fine-grained version control
- Want simple backup/sharing

## Validation

```bash
glx validate archive.glx
```

Should validate successfully with all cross-references intact.

## Migration to Multi-File

To split this into multiple files:
1. Create entity-type directories (persons/, sources/, etc.)
2. Extract each entity into its own file
3. Keep the same entity type key structure

Example:
```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
```

