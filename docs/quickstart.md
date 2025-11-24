---
title: Quickstart Guide
description: Get started with GENEALOGIX in 5 minutes - create your first family archive
layout: doc
---

# Quickstart Guide

Get started with GENEALOGIX in 5 minutes! This guide walks you through creating your first family archive.

## What You'll Learn

- Installing the `glx` CLI tool
- Creating a new genealogy repository
- **Customizing vocabularies for your research domain**
- Adding your first person, event, and relationship
- Validating your archive
- Using Git for version control

## Prerequisites

- **Go 1.25+** (for installing the CLI tool)
- **Git** (for version control)
- **Text editor** (any editor that can edit YAML files)

## Step 1: Install the CLI Tool

Install the `glx` command-line tool:

```bash
# Install the latest version
go install github.com/genealogix/glx/glx@latest

# Verify installation
glx --help
```

**Expected output:**
```
genealogix CLI
Usage:
  glx init                Initialize a new genealogix repository
  glx validate [paths]    Validate .glx files (defaults to current directory)
  glx check-schemas       Validate schema files for required metadata
```

## Step 2: Create Your First Repository

Navigate to where you want your family archive and initialize it:

```bash
# Create a new directory for your family archive
mkdir my-family-archive
cd my-family-archive

# Initialize with multi-file structure (recommended for collaboration)
glx init

# OR initialize as a single file (recommended for personal use)
glx init --single-file
```

**What `glx init` creates:**
- Directory structure for all entity types (persons, events, places, etc.)
- `.gitignore` file for Git
- `README.md` with repository documentation

## Step 3: Add Your First Person

Create your first person file. All files use the `.glx` extension and are written in YAML format.

**Create `persons/person-john-smith.glx`:**
```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith:
    
    # Basic information
    name:
      given: John
      surname: Smith
      display: John Smith
    
    # Optional: Add birth information
    birth:
      date: "1850-01-15"
      place: place-leeds
    
    # Optional: Add biographical notes
    notes: |
      John was a blacksmith in Leeds during the Industrial Revolution.
      He worked at the ironworks on Wellington Street.
```

## Step 4: Add a Place

Create the place referenced in the birth information:

**Create `places/place-leeds.glx`:**
```yaml
# places/place-leeds.glx
places:
  place-leeds:
    
    name: Leeds
    type: city
    
    # Hierarchical location (optional)
    parent: place-yorkshire
    
    # Geographic coordinates (optional)
    coordinates:
      latitude: 53.7960
      longitude: -1.5479
    
    # Alternative names (optional)
    alternative_names:
  - West Riding

notes: |
  Major industrial city in Yorkshire, England.
  Known for textile manufacturing and ironworks in the 19th century.
```

## Step 5: Add a Birth Event

Create a structured birth event with evidence:

**Create `events/event-john-birth.glx`:**
```yaml
# events/event-john-birth.glx
id: event-john-birth
type: event

# Event type and date
event_type: birth
date: "1850-01-15"
place: place-leeds

# Participants in the event
participants:
  - person: person-john-smith
    role: subject

# Optional: Add context
description: |
  Birth of John Smith, first child of Thomas and Mary Smith.
  Born at 23 Wellington Street, Leeds.
```

## Step 6: Validate Your Archive

Validate that all your files follow the correct format:

```bash
# Validate the entire archive
glx validate

# Or validate specific files
glx validate persons/
glx validate places/
```

**Expected output:**
```
✓ persons/person-john-smith.glx
✓ places/place-leeds.glx
✓ events/event-john-birth.glx
Validated 3 file(s)
```

## Step 7: Add Evidence with Citations

Make your research more credible by adding source citations:

**Create `sources/source-parish-register.glx`:**
```yaml
# sources/source-parish-register.glx
id: source-parish-register
type: source

title: St. Paul's Parish Register
type: church_register
creator: Church of England, St. Paul's Parish
date: "1849-1855"

# Where to find this source
repository: repository-leeds-library

# Publication details
publication_info:
  publisher: St. Paul's Church
  publication_date: "1850"
```

**Create `citations/citation-birth-entry.glx`:**
```yaml
# citations/citation-birth-entry.glx
id: citation-birth-entry
type: citation

source: source-parish-register

# Specific location within the source
locator: "Entry 145, page 23"

# Optional: Transcription
transcription: |
  "January 15th, 1850. John, son of Thomas Smith, blacksmith,
  and Mary Smith, of 23 Wellington Street. Baptized January 20th."
```

**Update the person file to reference the citation:**
```yaml
# persons/person-john-smith.glx
id: person-john-smith
type: person

name:
  given: John
  surname: Smith
  display: John Smith

birth:
  date: "1850-01-15"
  place: place-leeds
  citations:
    - citation-birth-entry  # Reference to citation
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

GLX isn't limited to traditional genealogy! Customize the vocabularies to match your research domain.

### Example: Maritime History Research

**Edit `vocabularies/event-types.glx`:**
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

**Edit `vocabularies/relationship-types.glx`:**
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
        role: passenger
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

## Troubleshooting

**"Command not found: glx"**
- Ensure Go is installed: `go version`
- Reinstall: `go install github.com/genealogix/glx/glx@latest`

**Validation errors:**
- Check YAML syntax (proper indentation, quotes)
- Ensure all referenced IDs exist (person, place, citation IDs)
- Verify required fields are present (id, version, type)

**Schema validation fails:**
- Use the latest CLI: `go install github.com/genealogix/glx/glx@latest`
- Check file extensions (.glx) and directory structure

## Next Steps

🎉 **Congratulations!** You have a working GENEALOGIX archive.

**Continue learning:**
- [Complete Examples](examples/) - See all entity types in action
- [Specification](../../specification/) - Detailed format documentation
- [Best Practices](../../docs/guides/best-practices.md) - Recommended workflows

**Get help:**
- [GitHub Issues](https://github.com/genealogix/glx/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Community Q&A
- [Contributing Guide](../../CONTRIBUTING.md) - Help improve GENEALOGIX

## Quick Reference

**Entity Types:**
- `persons/` - Individual people
- `relationships/` - Family connections
- `events/` - Life events and facts
- `places/` - Geographic locations
- `sources/` - Books, records, websites
- `citations/` - Specific references
- `repositories/` - Archives and libraries
- `assertions/` - Evidence-based conclusions
- `media/` - Photos and documents

**ID Format:**
- `person-` + 8 hex chars (e.g., `person-a1b2c3d4`)
- `event-` + 8 hex chars
- `place-` + 8 hex chars
- etc.

**Common Commands:**
```bash
glx init                    # Create new repository
glx validate                # Check all files
glx validate persons/       # Check specific directory
glx validate file.glx       # Check specific file
```
