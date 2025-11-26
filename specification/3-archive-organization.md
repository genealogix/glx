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
      born_on: "1850-01-15"

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

### 2. Repository-Level Validation

Across all files in an archive, the validator checks:

**Errors (Hard Failures):**
- Entity IDs must be unique (no duplicates)
- All entity cross-references must point to existing entities
- All vocabulary type references must be defined (event_types, relationship_types, etc.)
- All property `reference_type` values must point to existing entities
- Entity ID patterns must match their type (e.g., `person-` prefix for persons)

**Warnings (Soft Failures):**
- Unknown properties (not defined in property vocabularies) generate warnings
- Unknown assertion claims (not defined in property vocabularies) generate warnings

See [Entity Types - Validation](4-entity-types/README.md#validation) and [Vocabularies - Vocabulary Validation](4-entity-types/vocabularies.md#vocabulary-validation) for complete validation policy.

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
└── family.glx
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
    └── files/
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

## ID Format Standards

Entity IDs can be any unique identifier you choose, with the following constraints:

**Requirements:**
- 1-64 characters in length
- Alphanumeric characters (a-z, A-Z, 0-9) and hyphens only
- Must be unique across the entire repository

**Recommended Format** (for collaboration):
- Prefix with entity type for clarity: `person-`, `event-`, `place-`, etc.
- Use random hex for uniqueness: `person-a1b2c3d4`, `event-12345678`
- Keeps IDs short and collision-resistant

**Alternative Formats** (also valid):
- Descriptive: `person-john-smith-1850`, `place-leeds-yorkshire`
- Sequential: `person-001`, `person-002`, `event-001`
- UUID-style: `person-550e8400-e29b-41d4-a716`
- Custom: Any format meeting the requirements above

### ID Generation Examples

**Random hex (recommended for collaboration):**
```bash
# Bash
echo "person-$(openssl rand -hex 4)"

# Python
import secrets
f"person-{secrets.token_hex(4)}"

# JavaScript
const crypto = require('crypto');
`person-${crypto.randomBytes(4).toString('hex')}`

# Go
import "crypto/rand"
b := make([]byte, 4)
rand.Read(b)
fmt.Sprintf("person-%x", b)
```

**Descriptive (easier for humans):**
- `person-john-smith`
- `event-birth-john-1850`
- `place-leeds-uk`
- `source-parish-register-leeds`

**Note:** Descriptive IDs are fine for personal use but may cause conflicts when merging archives. Use random IDs for collaborative projects.

## Vocabularies Directory

Every GENEALOGIX archive should include a `vocabularies/` directory containing controlled vocabulary definitions. These files define valid types for entities:

```
vocabularies/
├── relationship-types.glx    # Marriage, parent-child, adoption, etc.
├── event-types.glx           # Birth, death, baptism, occupation, etc.
├── place-types.glx           # Country, city, parish, etc.
├── repository-types.glx      # Archive, library, church, etc.
├── participant-roles.glx     # Principal, witness, officiant, etc.
├── media-types.glx           # Photo, document, audio, etc.
└── confidence-levels.glx     # High, medium, low, disputed
```

### Vocabulary Files

Vocabulary files use the same GLX format with vocabulary-specific top-level keys:

```yaml
# vocabularies/relationship-types.glx
relationship_types:
  marriage:
    label: "Marriage"
    description: "Legal or religious union of two people"
    gedcom: "MARR"
  parent-child:
    label: "Parent-Child"
    description: "Biological, adoptive, or legal parent-child relationship"
    gedcom: "CHIL/FAMC"
  # Add custom types as needed
```

### Initialization

When you run `glx init`, the CLI automatically creates the `vocabularies/` directory by copying the standard vocabulary templates from [Standard Vocabularies](5-standard-vocabularies/). You can then customize these files to add archive-specific types.

See [Core Concepts](2-core-concepts.md#repository-owned-vocabularies) for details on defining custom vocabulary entries and [Standard Vocabularies](5-standard-vocabularies/) for the complete set of standard vocabulary files.

## Important Notes

- **Folder names are conventions**, not requirements
- **Parser must scan ALL** `.glx` and `.yaml` files in the repository
- **Duplicate entity IDs** across files is an error
- **Entity type keys are required** at the top level of every file
- **Cross-references are validated** at repository level
- **Vocabularies define valid types** - entities must reference types from vocabulary files

## Git Workflow Integration

### .gitignore Recommendations

```gitignore
# GENEALOGIX Repository
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
# Using glx tool (future feature)
glx convert --output family.glx persons/ events/ sources/
```

Manual approach: Copy all entity type sections from individual files into one file.

### Converting Single-File to Multi-File

```bash
# Using glx tool (future feature)
glx split family.glx --output-dir .
```

Manual approach: Extract each entity into its own file with the appropriate entity type key.

## Validation

The `glx validate` command performs comprehensive validation:

```bash
# Validate entire repository (recommended - checks all cross-references)
glx validate

# Validate individual file (structural validation only, limited cross-reference checking)
glx validate family.glx

# Validate specific directory
glx validate persons/
```

**Validation Output:**
- ✓ **Pass**: File/entity is valid
- ⚠ **Warning**: Soft validation issue (unknown property, unknown claim)
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
6. **Document your organization** in the repository README

## Examples

See the `docs/examples/` directory for complete working examples:
- `docs/examples/complete-family/` - Multi-file organization
- `docs/examples/single-file/` - Single-file archive
- `docs/examples/mixed-format/` - Hybrid approach
