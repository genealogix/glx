---
title: Quickstart Guide
description: Get started with GENEALOGIX in 5 minutes - create your first family archive
layout: doc
---

# Quickstart Guide

Get started with GENEALOGIX in 5 minutes! This guide walks you through creating your first family archive from scratch. If you'd like to understand the concepts behind GLX first, read the [Introduction](/specification/1-introduction) and [Core Concepts](/specification/2-core-concepts).

::: tip Already have a GEDCOM file?
See the [Migration from GEDCOM](/guides/migration-from-gedcom) guide to import your existing data.
:::

## What You'll Learn

- Setting up the `glx` CLI tool
- Creating a new genealogy repository
- Adding your first person, place, and event
- Validating your archive
- Documenting evidence with sources, citations, and assertions
- Using Git for version control
- Customizing vocabularies for your research domain

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

This creates a directory structure with folders for all [entity types](/specification/4-entity-types/), standard [vocabularies](/specification/4-entity-types/vocabularies), a `.gitignore`, and a `README.md`. See the [CLI README](https://github.com/genealogix/glx/blob/main/glx/README.md#glx-init) for the full layout and [Archive Organization](/specification/3-archive-organization) for how archives are structured.

## Step 3: Add Your First Person

Create your first person file. All GLX files use the `.glx` extension and are written in [YAML](https://yaml.org/), a plain text format where indentation shows structure and colons separate labels from values.

Here's the basic pattern: each file starts with the **entity type** (`persons`), then an **entity ID** (`person-john-smith`), then the entity's **[properties](/specification/2-core-concepts#properties-recording-conclusions)** — the facts you know about this person. See the full [Person specification](/specification/4-entity-types/person) for all available fields.

**Create `persons/person-john-smith.glx`:**
```yaml
persons:
  person-john-smith:             # Unique ID for this person
    properties:
      name:
        value: "John Smith"      # The display name
        fields:                  # Structured name parts
          given: "John"
          surname: "Smith"
      gender: "male"
    notes: |
      John was a blacksmith in Leeds during the Industrial Revolution.
      He worked at the ironworks on Wellington Street.
```

::: details What's the difference between `value` and `fields`?
Properties in GLX can have a simple `value` (the human-readable form) and optional `fields` that break it into structured parts. For a name, `value` is what you'd display ("John Smith") while `fields` lets software know which part is the given name and which is the surname. See [Properties](/specification/2-core-concepts#properties-recording-conclusions) in the specification for the full details.
:::


## Step 4: Add a Place

Places are their own [entities](/specification/4-entity-types/place) in GLX, so they can be referenced by multiple events and shared across your archive.

**Create `places/place-leeds.glx`:**
```yaml
places:
  place-leeds:
    name: "Leeds"
    type: city                   # Defined in your vocabularies
    latitude: 53.7960
    longitude: -1.5479
    notes: |
      Major industrial city in Yorkshire, England.
      Known for textile manufacturing and ironworks in the 19th century.
```

## Step 5: Add a Birth Event

[Events](/specification/4-entity-types/event) connect people to places and dates. Notice how `place` and `person` refer to the IDs you created in the previous steps — this is how GLX [entities link together](/specification/2-core-concepts#entity-relationships).

**Create `events/event-john-birth.glx`:**
```yaml
events:
  event-john-birth:
    type: birth                        # Defined in your vocabularies
    date: "1850-01-15"
    place: place-leeds                 # References the place you created
    participants:
      - person: person-john-smith      # References the person you created
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

## Step 7: Add a Source and Citation

Good research tracks where information comes from. GLX models this as an **[evidence chain](/specification/2-core-concepts#evidence-chain)**: a **[Source](/specification/4-entity-types/source)** describes a document or record, and a **[Citation](/specification/4-entity-types/citation)** points to a specific detail within that source.

First, create the source — the parish register where you found the birth record:

**Create `sources/source-parish-register.glx`:**
```yaml
sources:
  source-parish-register:
    title: "St. Paul's Parish Register"
    type: church_register
    authors:
      - "Church of England, St. Paul's Parish"
    date: "FROM 1849 TO 1855"
    properties:
      publication_info: "St. Paul's Church, Leeds, Yorkshire"
```

Now create a citation — the specific entry you found in that register:

**Create `citations/citation-birth-entry.glx`:**
```yaml
citations:
  citation-birth-entry:
    source: source-parish-register       # References the source above
    properties:
      locator: "Entry 145, page 23"      # Where in the source
      text_from_source: |                 # What the source actually says
        "January 15th, 1850. John, son of Thomas Smith, blacksmith,
        and Mary Smith, of 23 Wellington Street. Baptized January 20th."
```

## Step 8: Record Your Conclusion

You've documented *where* the information comes from (source and citation). Now record *what you conclude* from it using an **[Assertion](/specification/4-entity-types/assertion)** — a formal statement that links your evidence to a claim about a person.

**Create `assertions/assertion-john-birth.glx`:**
```yaml
assertions:
  assertion-john-birth:
    subject:
      person: person-john-smith          # Who this is about
    property: born_on                    # What fact you're asserting
    value: "1850-01-15"                  # Your conclusion
    citations:
      - citation-birth-entry             # The evidence supporting it
    confidence: high                     # How confident you are
```

This is the complete **evidence chain**: Source → Citation → Assertion. It traces your conclusion all the way back to the original document. See [Evidence Chain](/specification/2-core-concepts#evidence-chain) in the specification for more on this model.

## Step 9: Version Control with Git

GLX is designed to work naturally with Git for version control and [collaboration](/specification/2-core-concepts#collaboration). Track your research:

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

## Step 10: Customize Vocabularies for Your Research

GLX isn't limited to traditional genealogy! Each archive defines its own [controlled vocabularies](/specification/2-core-concepts#archive-owned-vocabularies) — the types that matter to your research. Vocabulary files can live anywhere in your archive — the examples below use the default `vocabularies/` directory created by `glx init`. See the [Vocabularies specification](/specification/4-entity-types/vocabularies) and [Standard Vocabularies](/specification/5-standard-vocabularies/) for the full reference.

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

Your archive has a person, place, event, source, citation, and assertion — the core building blocks of GLX. Here's what to try next:

- **Add more family members** — create more person files in `persons/` and link them with events
- **Add [relationships](/specification/4-entity-types/relationship)** — create files in `relationships/` to record marriages, parent-child connections, and other relationships (see the [Basic Family](/examples/basic-family/) example)
- **Explore all entity types** — the [Complete Family](/examples/complete-family/) example shows every entity type working together, or browse the [Entity Types](/specification/4-entity-types/) reference
- **Read the specification** — the [Introduction](/specification/1-introduction) and [Core Concepts](/specification/2-core-concepts) explain the architecture behind what you just built
- **Read the Best Practices** — [recommended workflows](/guides/best-practices) for evidence documentation, Git usage, and file organization

**Get help:**
- [GitHub Issues](https://github.com/genealogix/glx/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Community Q&A
