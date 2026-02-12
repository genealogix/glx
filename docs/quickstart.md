---
title: Quickstart Guide
description: Get started with GENEALOGIX in 5 minutes - create your first family archive
layout: doc
---

# Quickstart Guide

Get started with GENEALOGIX in 5 minutes! This guide walks you through creating your first family archive from scratch.

::: tip Already have a GEDCOM file?
See the [Migration from GEDCOM](/guides/migration-from-gedcom) guide to import your existing data.
:::

## What You'll Learn

- Setting up the `glx` CLI tool
- Creating a new genealogy repository
- Adding your first person, event, and relationship
- Validating your archive
- Adding evidence with source citations
- **Customizing vocabularies for your research domain**
- Using Git for version control

## Prerequisites

- **Git** (for version control)
- **Text editor** (any editor that can edit YAML files)

## Step 1: Install the CLI Tool

Follow the [installation instructions](https://github.com/genealogix/glx/blob/main/glx/README.md#installation) to download the latest `glx` binary for your platform.

Verify it works:

```bash
glx --help
```

## Step 2: Create Your First Repository

Navigate to where you want your family archive and initialize it:

```bash
# Create a new directory for your family archive
mkdir my-family-archive
cd my-family-archive

# Initialize with multi-file structure (recommended for collaboration)
glx init

# OR initialize as a single file
glx init --single-file
```

This creates a directory structure with folders for all entity types, standard vocabularies, a `.gitignore`, and a `README.md`. See the [CLI README](https://github.com/genealogix/glx/blob/main/glx/README.md#glx-init) for the full layout.

## Step 3: Add Your First Person

Create your first person file. All files use the `.glx` extension and are written in YAML format.

**Create `persons/person-john-smith.glx`:**
```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"
    notes: |
      John was a blacksmith in Leeds during the Industrial Revolution.
      He worked at the ironworks on Wellington Street.
```

## Step 4: Add a Place

Create the place referenced in the birth information:

**Create `places/place-leeds.glx`:**
```yaml
places:
  place-leeds:
    name: "Leeds"
    type: city
    latitude: 53.7960
    longitude: -1.5479
    notes: |
      Major industrial city in Yorkshire, England.
      Known for textile manufacturing and ironworks in the 19th century.
```

## Step 5: Add a Birth Event

Create a structured birth event with evidence:

**Create `events/event-john-birth.glx`:**
```yaml
events:
  event-john-birth:
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: subject
    properties:
      description: |
        Birth of John Smith, first child of Thomas and Mary Smith.
        Born at 23 Wellington Street, Leeds.
```

## Step 6: Validate Your Archive

Validate that all your files follow the correct format:

```bash
glx validate
```

**Expected output:**
```
Validated 19 files.
✅ Archive is valid.
```

You can also validate specific directories or single files. See the [CLI README](https://github.com/genealogix/glx/blob/main/glx/README.md#glx-validate) for all validation options.

## Step 7: Add Evidence with Citations

Make your research more credible by adding source citations:

**Create `sources/source-parish-register.glx`:**
```yaml
sources:
  source-parish-register:
    title: "St. Paul's Parish Register"
    type: church_register
    authors:
      - "Church of England, St. Paul's Parish"
    date: "1849-1855"
    properties:
      publication_info: "St. Paul's Church, Leeds, Yorkshire"
```

**Create `citations/citation-birth-entry.glx`:**
```yaml
citations:
  citation-birth-entry:
    source: source-parish-register
    properties:
      locator: "Entry 145, page 23"
      text_from_source: |
        "January 15th, 1850. John, son of Thomas Smith, blacksmith,
        and Mary Smith, of 23 Wellington Street. Baptized January 20th."
```

**Create an assertion to link the citation to the person's birth:**
```yaml
assertions:
  assertion-john-birth:
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-entry
    confidence: high
```

## Step 8: Version Control with Git

Track your research with Git:

```bash
# Initialize git repository (if not already done)
git init

# Check what files you've created
git status

# Stage your files
git add .

# Make your first commit
git commit -m "Initial commit: Add John Smith family data

- Added John Smith person record with birth information
- Created Leeds place record with coordinates
- Added birth event with participants
- Created parish register source and citation
- All files validated successfully"
```

## Step 9: Explore More Features

Your basic archive is complete! Here are some next steps:

```bash
# Add family relationships
glx init  # Already done - relationships/ directory exists

# Add more family members
# Create persons/person-mary-smith.glx, etc.

# Add marriage events
# Create events/event-marriage.glx

# Add family relationships
# Create relationships/rel-marriage.glx
# Create relationships/rel-parent-child.glx

# Validate everything
glx validate

# See all validation examples
glx validate examples/complete-family/
```

## Step 10: Customize Vocabularies for Your Research

GLX isn't limited to traditional genealogy! Customize the vocabularies to match your research domain. Vocabulary files can live anywhere in your archive — the examples below use the default `vocabularies/` directory created by `glx init`.

### Example: Maritime History Research

**Edit `vocabularies/event-types.glx` (or wherever you keep your event types):**
```yaml
# vocabularies/event-types.glx
event_types:
  # Standard types (already present)
  birth:
    label: "Birth"
    description: "Birth of a person"
    gedcom: "BIRT"

  # ADD YOUR CUSTOM TYPES
  ship_departure:
    label: "Ship Departure"
    description: "Departure on a sea voyage"

  port_arrival:
    label: "Port Arrival"
    description: "Arrival at a port"
```

### Example: Academic Biography

**Edit `vocabularies/relationship-types.glx` (or wherever you keep your relationship types):**
```yaml
# vocabularies/relationship-types.glx
relationship_types:
  # Standard types
  marriage:
    label: "Marriage"
    description: "Legal or religious union"
    gedcom: "MARR"

  # ADD ACADEMIC RELATIONSHIPS
  doctoral_advisor:
    label: "Doctoral Advisor"
    description: "PhD thesis advisor"

  collaborator:
    label: "Research Collaborator"
    description: "Co-author or research partner"
```

**Then use your custom types:**
```yaml
# events/event-voyage.glx
events:
  event-voyage-1850:
    type: ship_departure  # Your custom type!
    date: "1850-06-15"
    place: place-liverpool
    participants:
      - person: person-john-smith
        role: subject
```

**Validate your custom vocabulary:**
```bash
glx validate
# ✓ Confirms your custom types are properly defined
```

### The Power of Custom Vocabularies

GLX adapts to YOUR research domain:
- **Genealogy**: Use standard family history types
- **Biography**: Add professional relationships and achievements
- **Local History**: Track community roles and civic events
- **Maritime History**: Document voyages and naval careers
- **Religious Studies**: Record ordinations, pilgrimages, and church roles
- **And more**: Any domain with people, events, and relationships

## Next Steps

🎉 **Congratulations!** You have a working GENEALOGIX archive.

**Continue learning:**
- [Complete Examples](/examples/) - See all entity types in action
- [Specification](/specification/) - Detailed format documentation
- [Best Practices](/guides/best-practices) - Recommended workflows

**Get help:**
- [GitHub Issues](https://github.com/genealogix/glx/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Community Q&A
- [Contributing Guide](/development/contributing) - Help improve GENEALOGIX

## Quick Reference

For the full list of entity types, ID format, file format details, and CLI commands, see the [GLX CLI README](https://github.com/genealogix/glx/blob/main/glx/README.md).
