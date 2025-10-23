# File Structure

This section describes the recommended layout and organization of a GENEALOGIX archive repository.

## Repository Layout

A GENEALOGIX archive is a Git repository with a standardized directory structure:

```
family-archive/
├── .git/
├── .gitignore
├── README.md (auto-generated)
├── persons/           # Individual people
├── relationships/     # Family connections
├── events/           # Life events and facts
├── places/           # Geographic locations
├── sources/          # Original materials
├── citations/        # Specific references
├── repositories/     # Archives and institutions
├── assertions/       # Evidence-based conclusions
└── media/           # Supporting files
```

### Standard Directory Structure

| Directory | Purpose | File Pattern |
|-----------|---------|--------------|
| `persons/` | Individual human beings | `person-{8hex}.glx` |
| `relationships/` | Connections between people | `rel-{8hex}.glx` |
| `events/` | Life events and biographical facts | `event-{8hex}.glx` |
| `places/` | Geographic locations (hierarchical) | `place-{8hex}.glx` |
| `sources/` | Original documents and materials | `source-{8hex}.glx` |
| `citations/` | Specific references within sources | `citation-{8hex}.glx` |
| `repositories/` | Physical/digital archives | `repository-{8hex}.glx` |
| `assertions/` | Evidence-based conclusions | `assertion-{8hex}.glx` |
| `media/` | Supporting photos/documents | `media-{8hex}.{ext}` |

## Naming Conventions

### ID Format Standards
All GENEALOGIX entities use structured ID formats for consistency and validation:

| Entity Type | ID Prefix | Example | Pattern |
|-------------|-----------|---------|---------|
| Person | `person-` | `person-a1b2c3d4` | `person-[a-f0-9]{8}` |
| Relationship | `rel-` | `rel-1a2b3c4d` | `rel-[a-f0-9]{8}` |
| Event | `event-` | `event-2b3c4d5e` | `event-[a-f0-9]{8}` |
| Place | `place-` | `place-3c4d5e6f` | `place-[a-f0-9]{8}` |
| Source | `source-` | `source-4d5e6f7g` | `source-[a-f0-9]{8}` |
| Citation | `citation-` | `citation-5e6f7g8h` | `citation-[a-f0-9]{8}` |
| Repository | `repository-` | `repository-6f7g8h9i` | `repository-[a-f0-9]{8}` |
| Assertion | `assertion-` | `assertion-7g8h9i0j` | `assertion-[a-f0-9]{8}` |
| Media | `media-` | `media-8h9i0j1k.jpg` | `media-[a-f0-9]{8}.*` |

### ID Generation
IDs are generated using the pattern: `{type}-{8hex}` where:
- `{type}` is the entity type prefix (person, event, etc.)
- `{8hex}` is exactly 8 lowercase hexadecimal characters

**Recommended generation methods:**
```bash
# Using CLI tool (preferred)
glx init  # Auto-generates with proper structure

# Using command line tools
echo "person-$(openssl rand -hex 4)"
echo "event-$(xxd -l 4 -p /dev/urandom | tr '[:upper:]' '[:lower:]')"

# Using programming languages
# Python: f"person-{random.randint(0, 0xFFFFFFFF):08x}"
# Node.js: `person-${crypto.randomBytes(4).toString('hex')}`
# Go: fmt.Sprintf("person-%08x", rand.Uint32())
```

### File Naming Rules

#### 1. Entity Files
- **Extension**: All entity files use `.glx` extension
- **Case**: All lowercase for consistency
- **Characters**: Only lowercase letters, numbers, and hyphens
- **Structure**: `{id}.glx` where ID includes the type prefix

**Valid examples:**
```
persons/person-a1b2c3d4.glx
events/event-birth-1850.glx
places/place-leeds-yorkshire.glx
citations/citation-parish-entry.glx
```

**Invalid examples:**
```
persons/John_Smith.glx          # No spaces or underscores
Persons/person-a1b2c3d4.glx     # Directory must be lowercase
person-a1b2c3d4.txt             # Must use .glx extension
a1b2c3d4.glx                    # Must include entity type prefix
```

#### 2. Media Files
- **Extension**: Preserve original file extension (.jpg, .pdf, .doc, etc.)
- **ID Prefix**: Use `media-` prefix in filename
- **Organization**: Can be organized in subdirectories by type

**Valid examples:**
```
media/media-a1b2c3d4.jpg
media/media-b2c3d4e5.pdf
media/documents/birth-certificate-1850.pdf
media/photos/family-portrait-1880.jpg
media/audio/interview-1990.mp3
```

#### 3. Supporting Files
- **README**: Repository documentation (auto-generated)
- **.gitignore**: Git ignore patterns (auto-generated)
- **Archive metadata**: Custom files in root directory

## File Organization Patterns

### Directory Organization

#### 1. Core Data Directories
```bash
family-archive/
├── persons/           # All individual people
│   ├── person-john-smith.glx
│   ├── person-mary-brown.glx
│   └── person-jane-smith.glx
├── relationships/     # Family and social connections
│   ├── rel-john-mary-marriage.glx
│   ├── rel-john-jane-parent.glx
│   └── rel-mary-jane-step.glx
└── events/           # All life events
    ├── event-john-birth.glx
    ├── event-marriage-1875.glx
    └── event-john-death.glx
```

#### 2. Evidence Directories
```bash
family-archive/
├── sources/          # Original documents
│   ├── source-parish-register.glx
│   ├── source-census-1851.glx
│   └── source-family-bible.glx
├── citations/        # Specific references
│   ├── citation-birth-entry.glx
│   ├── citation-census-schedule.glx
│   └── citation-bible-notation.glx
├── repositories/     # Where sources are held
│   ├── repository-leeds-library.glx
│   └── repository-gro-london.glx
└── assertions/       # Conclusions from evidence
    ├── assertion-john-birth.glx
    ├── assertion-marriage.glx
    └── assertion-occupation.glx
```

#### 3. Geographic Organization
```bash
family-archive/
└── places/           # Hierarchical locations
    ├── place-england.glx      # Country level
    ├── place-yorkshire.glx    # County level
    │   └── parent: place-england
    ├── place-leeds.glx        # City level
    │   └── parent: place-yorkshire
    ├── place-liverpool.glx    # Another city
    │   └── parent: place-england
    └── place-london.glx       # Capital city
        └── parent: place-england
```

#### 4. Media Organization
```bash
family-archive/
└── media/           # Supporting files
    ├── photos/
    │   ├── portraits/
    │   │   ├── media-john-smith-1880.jpg
    │   │   └── media-mary-brown-1885.jpg
    │   └── family/
    │       └── media-smith-family-1890.jpg
    ├── documents/
    │   ├── certificates/
    │   │   ├── media-birth-cert-1850.pdf
    │   │   └── media-marriage-cert-1875.pdf
    │   └── letters/
    │       └── media-letter-1860.pdf
    └── audio/
        └── interviews/
            └── media-aunt-mary-interview-1985.mp3
```

### Grouping Strategies

#### 1. Chronological Grouping (Events)
```bash
events/
├── 1800s/
│   ├── event-john-birth-1850.glx
│   └── event-mary-birth-1852.glx
├── 1870s/
│   ├── event-marriage-1875.glx
│   └── event-jane-birth-1876.glx
└── 1920s/
    └── event-john-death-1920.glx
```

#### 2. Source Type Grouping (Evidence)
```bash
sources/
├── vital-records/
│   ├── source-birth-certificates.glx
│   └── source-death-certificates.glx
├── church-records/
│   ├── source-parish-registers.glx
│   └── source-bishop-transcripts.glx
└── census/
    ├── source-1851-census.glx
    ├── source-1861-census.glx
    └── source-1871-census.glx
```

#### 3. Geographic Grouping (Places)
```bash
places/
├── countries/
│   └── place-england.glx
├── counties/
│   ├── place-yorkshire.glx
│   └── place-lancashire.glx
└── cities/
    ├── place-leeds.glx
    ├── place-bradford.glx
    └── place-manchester.glx
```

## Version Control Integration

### Git-Ignored Files
The `.gitignore` file (auto-generated by `glx init`) excludes:

```gitignore
# GENEALOGIX Repository
# Ignore temporary files and build artifacts

*.tmp
*.bak
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Research scratch files
research-notes.txt
temp-*.glx
scratch/
```

### Recommended Git Workflow

#### 1. Repository Initialization
```bash
# Create new repository
mkdir my-family-archive
cd my-family-archive
glx init  # Creates directory structure and .gitignore
git init
git add .
git commit -m "Initial commit: Set up GENEALOGIX archive"
```

#### 2. Research Branching
```bash
# Create research branch for investigation
git checkout -b research/1851-census

# Add census data
# Create sources, citations, person updates
glx validate

# Commit findings
git add .
git commit -m "Add 1851 Census data for Smith family

- Census source and citation added
- Updated John Smith occupation to blacksmith
- Added residence information
- Quality rating: 2 (secondary source)"

# Merge to main when complete
git checkout main
git merge research/1851-census
```

#### 3. Evidence Integration
```bash
# Integrate multiple evidence sources
git checkout -b evidence-integration
git merge research/vital-records
git merge research/census-data
git merge research/church-records

# Validate complete evidence chain
glx validate

# Resolve any conflicts or inconsistencies
# Commit integrated evidence
git commit -m "Integrated evidence for 1850-1900 period

Combined sources:
- Birth certificates (primary evidence)
- Census records (secondary evidence)
- Parish registers (primary evidence)
- All assertions properly cited and validated"
```

## Validation Rules

### File Structure Validation
The `glx validate` command checks:

1. **File Extensions**: All `.glx` files must be in correct directories
2. **ID Formats**: All IDs must follow `{type}-{8hex}` pattern
3. **References**: All referenced IDs must exist in their directories
4. **Schema Compliance**: All files must match JSON Schema definitions

### Directory Permissions
```bash
# Ensure proper permissions for collaboration
find . -type d -exec chmod 755 {} \;
find . -name "*.glx" -exec chmod 644 {} \;

# Git-friendly permissions
git config core.filemode false  # Ignore permission changes
```

## Migration from Other Formats

### From GEDCOM
GEDCOM files can be converted to GENEALOGIX structure:

```
gedcom-import/
├── individuals/ → persons/
├── families/ → relationships/
├── events/ → events/
├── places/ → places/
└── sources/ → sources/
```

### From Legacy Databases
Database records map to GENEALOGIX entities:

| Database Table | GENEALOGIX Directory | Notes |
|----------------|---------------------|-------|
| `individuals` | `persons/` | One record per person |
| `families` | `relationships/` | Family connections |
| `events` | `events/` | All life events |
| `places` | `places/` | Geographic hierarchy |
| `sources` | `sources/` | Original materials |
| `citations` | `citations/` | Evidence references |

This file structure ensures that GENEALOGIX archives are organized, searchable, and maintainable over time.


