---
title: Archive Organization
description: GLX file format, directory structure, validation, and organization strategies
layout: doc
---

# Archive Organization

This section describes the GENEALOGIX file format and recommended organization strategies for archives.

## GLX File Format

Every GENEALOGIX file uses the same universal structure:

### Structure Requirements

1. **Top-level keys are entity type plurals**: `persons`, `relationships`, `events`, `places`, `sources`, `citations`, `repositories`, `assertions`, `media`
2. **Each key contains a map** where:
   - Keys are entity IDs (e.g., `person-abc12345`)
   - Values are entity objects
3. **Files may contain any combination** of entity types
4. **Empty sections** can be omitted or left as `{}`

### Basic Example

```yaml
# Any .glx or .yaml file
persons:
  person-abc12345:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"

  person-def67890:
    properties:
      name:
        value: "Mary Brown"
        fields:
          given: "Mary"
          surname: "Brown"

sources:
  source-xyz11111:
    title: "Birth Certificate"
    type: vital_record

# Other entity types can be empty or omitted
events: {}
relationships: {}
```

### Minimal Valid File

```yaml
persons:
  person-abc12345:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
```

## Validation Levels

GENEALOGIX validation operates at two levels:

### 1. File-Level Validation

Each `.glx` file must:
- Be valid YAML with proper structure
- Have at least one top-level entity type key (persons, events, relationships, etc.)
- Pass JSON schema validation for structural correctness
- Contain properly formatted entity IDs (alphanumeric with hyphens, 1-64 characters)

### 2. Archive-Level Validation

Across all files in an archive, the validator checks:

**Errors (Hard Failures):**
- Entity IDs must be unique (no duplicates)
- All entity cross-references must point to existing entities
- All vocabulary type references must be defined (event_types, relationship_types, etc.)
- All property `reference_type` values must point to existing entities

**Warnings (Soft Failures):**
- Unknown properties (not defined in property vocabularies) generate warnings
- Unknown assertion properties (not defined in property vocabularies) generate warnings
- Temporal consistency issues generate warnings:
  - Death year before birth year
  - Parent born after child (in parent-child relationships)
  - Marriage event before a participant's birth year

> **Note:** Temporal checks are warnings rather than errors because dates in genealogical records are often estimates (e.g., `ABT 1850`). A flagged inconsistency may indicate a data entry error or simply imprecise dating.

See [Vocabularies - Vocabulary Validation](4-entity-types/vocabularies#vocabulary-validation) for complete validation policy.

## Organization Strategies

The folder structure and file organization is a **recommended practice**, not a requirement. Choose the strategy that best fits your workflow.

### One Entity Per File (Recommended for Collaboration)

**Structure:**
```
family-archive/
├── persons/
│   ├── person-abc12345.glx
│   └── person-def67890.glx
├── sources/
│   ├── source-xyz11111.glx
│   └── source-mno22222.glx
├── events/
│   └── event-birth-abc.glx
├── relationships/
│   └── rel-marriage-001.glx
├── media/
│   └── files/                   # Local media files (images, PDFs, etc.)
│       └── birth-certificate.jpg
└── vocabularies/
    ├── relationship-types.glx
    ├── event-types.glx
    └── place-types.glx
```

**File Contents:**
```yaml
# persons/person-abc12345.glx
persons:
  person-abc12345:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
```

**Benefits:**
- Fine-grained Git diffs (see exactly what changed)
- Parallel editing without conflicts
- Easy merge conflict resolution
- Clear file organization

**Best for:**
- Team research projects
- Large archives (100+ entities)
- Active collaboration workflows
- Long-term maintenance

### Single File Archive

**Structure:**
```
family-archive/
├── family.glx
└── media/
    └── files/                   # Local media files (images, PDFs, etc.)
        └── birth-certificate.jpg
```

**File Contents:**
```yaml
# family.glx
persons:
  person-abc12345:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
  person-def67890:
    properties:
      name:
        value: "Mary Brown"
        fields:
          given: "Mary"
          surname: "Brown"

relationships:
  rel-marriage-001:
    type: marriage
    participants:
      - person: person-abc12345
        role: spouse
      - person: person-def67890
        role: spouse

events:
  event-birth-abc:
    type: birth
    date: "1850-01-15"
    place: place-leeds-uk
    participants:
      - person: person-abc12345
        role: subject

places:
  place-leeds-uk:
    name: "Leeds"
    type: city

sources:
  source-xyz11111:
    title: "Birth Certificate"
    type: vital_record

citations: {}
repositories: {}
assertions: {}
media: {}
```

**Benefits:**
- Simple structure (one file to manage)
- Easy backup and sharing
- Quick overview of entire archive
- Good for GEDCOM-style workflows

**Best for:**
- Personal research
- Small family trees (<50 entities)
- Quick exports/backups
- Simple sharing scenarios

### Hybrid Approach

Mix and match as needed:

**Structure:**
```
family-archive/
├── core-family.glx          # Main family members and relationships
├── sources/
│   ├── vital-records.glx    # Multiple vital record sources
│   └── census/
│       ├── census-1850.glx
│       └── census-1860.glx
├── places/                  # Individual place files
│   ├── place-leeds.glx
│   └── place-yorkshire.glx
├── vocabularies/            # Controlled vocabularies
│   ├── relationship-types.glx
│   ├── event-types.glx
│   └── place-types.glx
└── media/
    ├── photos.glx           # References to photo files
    └── files/               # Actual media files (binary content)
        ├── photo-001.jpg
        └── photo-002.jpg
```

**Benefits:**
- Flexibility to organize by logical groupings
- Keep related entities together
- Balance between organization and simplicity

**Best for:**
- Medium-sized archives
- Mixed collaboration patterns
- Gradual migration from single-file format

## Media File Storage

The standard location for local media files (images, documents, audio, video) within any GLX archive is **`media/files/`** at the archive root. Media entity metadata (`.glx` files) references these files via the `uri` field using paths relative to the archive root (e.g., `media/files/portrait.jpg`).

This convention applies to all organization strategies — single-file, multi-file, and hybrid. The `glx import` command automatically populates `media/files/` when importing from GEDCOM. See [Media Entity - File Storage](4-entity-types/media#file-storage) for details.

## ID Format Standards

Entity IDs can be any unique identifier you choose, with the following constraints:

**Requirements:**
- 1-64 characters in length
- Alphanumeric characters (a-z, A-Z, 0-9) and hyphens only
- Must be unique across the entire archive

> **Note:** Examples in this documentation use prefixes (e.g., `person-abc123`) for readability. Prefixes are not required—any format meeting the requirements above is valid.

**Example Formats:**
- Random hex: `a1b2c3d4`, `12345678`
- Prefixed: `person-a1b2c3d4`, `event-12345678`
- Descriptive: `john-smith-1850`, `leeds-yorkshire`
- Sequential: `001`, `002`, `person-001`
- UUID-style: `550e8400-e29b-41d4-a716`

### ID Generation Examples

**Random hex:**
```bash
# Bash
echo "$(openssl rand -hex 4)"

# Python
import secrets
secrets.token_hex(4)

# JavaScript
const crypto = require('crypto');
crypto.randomBytes(4).toString('hex')

# Go
import "crypto/rand"
b := make([]byte, 4)
rand.Read(b)
fmt.Sprintf("%x", b)
```

**Descriptive:**
- `john-smith`
- `birth-john-1850`
- `leeds-uk`
- `parish-register-leeds`

**Note:** Descriptive IDs are fine for personal use but may cause conflicts when merging archives. Random IDs reduce collision risk in collaborative projects.

## Vocabulary Files

Every GENEALOGIX archive should include vocabulary definitions. These files define valid types and properties for entities. Like all `.glx` files, vocabulary files can live anywhere in the archive — the parser identifies them by their top-level keys, not by location.

By convention, the CLI places vocabulary files in a `vocabularies/` directory (via `glx init` and `glx import`), but you're free to organize them however you like (alongside entity files, in a custom directory, etc.).

### Format

Vocabulary files use the same GLX format with vocabulary-specific top-level keys:

```yaml
# relationship-types.glx
relationship_types:
  marriage:
    label: "Marriage"
    description: "Legal or religious union of two people"
    gedcom: "MARR"
  parent_child:
    label: "Parent-Child"
    description: "Biological, adoptive, or legal parent-child relationship"
    gedcom: "CHIL/FAMC"
  # Add custom types as needed
```

### Initialization

When you run `glx init` or `glx import`, the CLI copies the standard vocabulary templates from [Standard Vocabularies](5-standard-vocabularies/) into a `vocabularies/` directory. You can then customize these files to add archive-specific types, or move them to a different location.

See [Core Concepts](2-core-concepts#archive-owned-vocabularies) for details on defining custom vocabulary entries and [Standard Vocabularies](5-standard-vocabularies/) for the complete set of standard vocabulary files.

## Important Notes

- **Folder names are conventions**, not requirements
- **Parser must scan ALL** `.glx` and `.yaml` files in the archive
- **Duplicate entity IDs** across files is an error
- **Entity type keys are required** at the top level of every file
- **Cross-references are validated** at archive level
- **Vocabularies define valid types** - entities must reference types from vocabulary files

## Git Workflow Integration

### .gitignore Recommendations

```gitignore
# GENEALOGIX Archive
*.tmp
*.bak
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp

# Build artifacts
bin/
build/
```

### Recommended Git Practices

**For multi-file archives:**
```bash
# Commit by entity type
git add persons/
git commit -m "Add Smith family members"

git add sources/census/
git commit -m "Add 1850 census sources"
```

**For single-file archives:**
```bash
# Commit with descriptive messages
git add family.glx
git commit -m "Add John Smith and birth event with sources"
```

## Migration Between Formats

### Converting Multi-File to Single-File

```bash
glx join path/to/archive/ family.glx
```

### Converting Single-File to Multi-File

```bash
glx split family.glx path/to/archive/
```

## Validation

The `glx validate` command performs comprehensive validation:

```bash
# Validate entire archive (recommended - checks all cross-references)
glx validate

# Validate individual file (structural validation only, limited cross-reference checking)
glx validate family.glx

# Validate specific directory
glx validate persons/
```

**Validation Output:**
- ✓ **Pass**: File/entity is valid
- ⚠ **Warning**: Soft validation issue (unknown property)
- ❌ **Error**: Hard validation failure (missing reference, invalid structure)

**Exit Codes:**
- `0`: All files valid (warnings allowed)
- `1`: One or more validation errors

See [Validation Levels](#validation-levels) above for details on what is validated.

## Best Practices

1. **Choose one primary strategy** and stick with it for consistency
2. **Use meaningful file names** that relate to content (e.g., `smith-family.glx`, `vital-records.glx`)
3. **Group related entities** when using multi-file format
4. **Commit frequently** with descriptive messages
5. **Validate often** to catch errors early
6. **Document your organization** in the archive README

## Examples

See the `docs/examples/` directory for complete working examples:
- `docs/examples/complete-family/` - Multi-file organization with all entity types
- `docs/examples/single-file/` - Single-file archive
- `docs/examples/basic-family/` - Basic family structure
- `docs/examples/minimal/` - Minimal archive example
- `docs/examples/temporal-properties/` - Temporal property examples
- `docs/examples/participant-assertions/` - Participant assertion examples
