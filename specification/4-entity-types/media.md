---
title: Media Entity
description: Digital and physical media objects - photographs, documents, audio, and video
layout: doc
---

# Media Entity

[← Back to Entity Types](README.md)

## Overview

A Media entity represents digital or physical media objects that provide visual, audio, or documentary evidence for genealogical research. Media can include photographs, scanned documents, audio recordings, video, or any other supporting materials that help document and illustrate family history.

Media entities serve several purposes:
- Preserve digital copies of original documents
- Provide visual context for people, places, and events
- Document physical artifacts and heirlooms
- Store audio/video recordings of interviews and oral histories
- Link supporting evidence to assertions and entities

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in media/ directory)
media:
  media-birth-cert-scan:
    uri: "media/documents/birth-certificate-john-smith.jpg"
    mime_type: "image/jpeg"
    title: "Birth Certificate - John Smith"
    description: "Scan of original birth certificate"
```

**Key Points:**
- Entity ID is the map key (`media-birth-cert-scan`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `uri` | string | Location of the media file |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Media type from `vocabularies/media-types.glx` |
| `mime_type` | string | MIME type of the media |
| `hash` | string | Content hash for verification |
| `title` | string | Title of the media |
| `description` | string | Description of the media |
| `notes` | string | Free-form notes |
| `tags` | array | Tags for categorization |

## Required Fields (Detailed)

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Recommended formats:
  - Descriptive: `media-birth-cert`, `media-john-portrait`
  - Random hex: `media-a1b2c3d4` (for collaboration)
  - Sequential: `media-001`, `media-002`

### `uri`

- Type: String
- Required: Yes
- Description: Location of the media file

The URI can be:
- **Relative path**: `media/photos/john-smith.jpg` (recommended, relative to repository root)
- **Absolute path**: `/home/user/genealogy/media/photo.jpg` (not portable)
- **URL**: `https://example.com/archives/document.pdf` (for external resources)
- **URN**: `urn:hash:sha256:abc123...` (content-addressable)

Example:
```yaml
uri: "media/photos/john-smith-1890.jpg"
```

## Optional Fields

### `title`

- Type: String
- Required: No (but recommended)
- Description: Descriptive title for the media

Example:
```yaml
title: "Portrait of John Smith, circa 1890"
```

### `mime_type`

- Type: String
- Required: No (but recommended)
- Description: MIME type of the media file

Common MIME types:
- Images: `image/jpeg`, `image/png`, `image/tiff`, `image/gif`
- Documents: `application/pdf`, `image/tiff`
- Audio: `audio/mpeg`, `audio/wav`, `audio/ogg`
- Video: `video/mp4`, `video/mpeg`, `video/quicktime`

Example:
```yaml
mime_type: "image/jpeg"
```

### `description`

- Type: String
- Required: No
- Description: Detailed description of the media content

Example:
```yaml
description: |
  Original birth certificate for John Smith, born January 15, 1850
  in Leeds, Yorkshire, England. Issued by the General Register Office.
  Certificate shows parents as Thomas Smith and Elizabeth Brown.
```

### `hash`

- Type: String
- Required: No
- Description: Cryptographic hash of the media file for integrity verification

Common formats:
- SHA-256: `sha256:a1b2c3d4...`
- MD5: `md5:xyz123...`

Example:
```yaml
hash: "sha256:7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730"
```

### `date`

- Type: String
- Required: No
- Description: Date the media was created (photograph taken, document scanned, etc.)

Example:
```yaml
date: "1890-06-15"
```

### `subjects`

- Type: Array of Strings
- Required: No
- Description: Entity IDs of people, places, or events depicted in the media

Example:
```yaml
subjects:
  - person-john-smith
  - person-mary-smith
  - place-leeds
```

### `source`

- Type: String
- Required: No
- Description: Source entity that this media documents

Example:
```yaml
source: source-birth-register
```

### `citation`

- Type: String
- Required: No
- Description: Citation entity that this media supports

Example:
```yaml
citation: citation-birth-entry
```

### `width` and `height`

- Type: Integer
- Required: No
- Description: Dimensions in pixels (for images and video)

Example:
```yaml
width: 3000
height: 2400
```

### `duration`

- Type: Integer
- Required: No
- Description: Duration in seconds (for audio and video)

Example:
```yaml
duration: 3600
```

### `file_size`

- Type: Integer
- Required: No
- Description: File size in bytes

Example:
```yaml
file_size: 2458624
```

### Other Fields

| Field | Type | Description |
|-------|------|-------------|
| `notes` | string | Research notes about the media |
| `tags` | array | Tags for categorization |

Example:
```yaml
tags:
  - original-document
  - high-quality-scan
  - verified
```

## Usage Patterns

### Scanned Document

```yaml
# media/media-birth-certificate.glx
media:
  media-birth-cert-john:
    uri: "media/documents/birth-certificate-john-smith-1850.pdf"
    mime_type: "application/pdf"
    title: "Birth Certificate - John Smith"
    description: "Original birth certificate from General Register Office"
    date: "2024-01-15"  # Date scanned
    hash: "sha256:7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730"
    source: source-gro-register
    citation: citation-birth-entry
    subjects:
      - person-john-smith
    file_size: 2458624
    tags:
      - original-document
      - vital-record
```

### Family Photograph

```yaml
# media/media-family-portrait.glx
media:
  media-smith-family-1890:
    uri: "media/photos/smith-family-portrait-1890.jpg"
    mime_type: "image/jpeg"
    title: "Smith Family Portrait, 1890"
    description: |
      Studio portrait of John and Mary Smith with their children.
      Taken at Leeds Portrait Studio, June 1890.
      
      People in photo (left to right):
      - Alice Smith (daughter)
      - John Smith (father)
      - Mary Smith (mother)
      - Thomas Smith (son)
    date: "1890-06-15"
    subjects:
      - person-john-smith
      - person-mary-smith
      - person-alice-smith
      - person-thomas-smith
    width: 3000
    height: 2400
    file_size: 4567890
    tags:
      - family-photo
      - studio-portrait
      - victorian-era
```

### Audio Recording

```yaml
# media/media-interview.glx
media:
  media-interview-mary-2020:
    uri: "media/audio/interview-mary-smith-2020-03-15.mp3"
    mime_type: "audio/mpeg"
    title: "Oral History Interview - Mary Smith"
    description: |
      Interview with Mary Smith about her memories of growing up
      in Leeds in the 1940s and 1950s. Discusses family traditions,
      local history, and genealogical information.
    date: "2020-03-15"
    duration: 3600
    subjects:
      - person-mary-smith
    file_size: 86400000
    tags:
      - oral-history
      - interview
      - audio-recording
```

### Online Resource

```yaml
# media/media-census-image.glx
media:
  media-census-1851-page:
    uri: "https://ancestry.com/imageviewer/1851-census-yorkshire-page-234"
    mime_type: "image/jpeg"
    title: "1851 Census - Yorkshire, Page 234"
    description: "Census page showing Smith family at Wellington Street, Leeds"
    date: "1851-04-06"
    source: source-1851-census
    citation: citation-census-smith-entry
    subjects:
      - person-john-smith
      - person-mary-smith
      - place-leeds
    tags:
      - census-image
      - online-resource
      - subscription-required
```

### Historical Document

```yaml
# media/media-marriage-register.glx
media:
  media-marriage-register-page:
    uri: "media/documents/st-pauls-marriage-register-1875-page-45.tiff"
    mime_type: "image/tiff"
    title: "Marriage Register - St Paul's Church, 1875"
    description: |
      Parish register page showing marriage of John Smith and Mary Brown
      on May 10, 1875 at St Paul's Cathedral, Leeds.
    date: "2024-02-10"  # Date photographed
    hash: "sha256:9f3d4c2e7a8b1f6d5c4e3b2a1f0e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
    source: source-st-pauls-register
    citation: citation-marriage-entry
    subjects:
      - person-john-smith
      - person-mary-brown
    width: 4000
    height: 3000
    file_size: 12000000
    notes: "High-resolution scan for archival preservation"
    tags:
      - parish-register
      - marriage-record
      - original-document
```

## Media Types

Media types are defined in the archive's `vocabularies/media-types.glx` file.

**See [Vocabularies - Media Types](vocabularies.md#media-types-vocabulary) for:**
- Complete list of standard media types
- How to add custom media types
- Vocabulary file structure and examples
- MIME type conventions

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Media files and their metadata are typically organized by type:

```
media/
├── photos/
│   ├── media-john-portrait.glx
│   ├── media-family-photo.glx
│   └── images/
│       ├── john-smith-1890.jpg
│       └── smith-family-1890.jpg
├── documents/
│   ├── media-birth-cert.glx
│   ├── media-marriage-cert.glx
│   └── scans/
│       ├── birth-certificate.pdf
│       └── marriage-certificate.pdf
├── audio/
│   ├── media-interview-mary.glx
│   └── recordings/
│       └── interview-2020-03-15.mp3
└── video/
    ├── media-family-reunion.glx
    └── clips/
        └── reunion-2015.mp4
```

## Relationship to Other Entities

```
Media
    ├── subjects → array of Person/Place/Event IDs (what's in the media)
    ├── source → Source ID (what source this documents)
    └── citation → Citation ID (what citation this supports)

Person/Event/Relationship
    └── referenced by → Media (via subjects array)

Source
    └── documented by → Media (via source field)

Citation
    └── supported by → Media (via citation field)
```

## File Storage Best Practices

### Relative Paths

Use paths relative to repository root for portability:
```yaml
# ✅ Good: Relative path
uri: "media/photos/john-smith.jpg"

# ❌ Bad: Absolute path (not portable)
uri: "/home/user/genealogy/media/photos/john-smith.jpg"
```

### File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The examples below show organization patterns, but you can use any structure that fits your workflow.

Organize media files alongside `.glx` metadata:
```
media/
├── photos/
│   ├── john-smith.jpg          # Actual file
│   └── media-john.glx          # Metadata
```

Or separate data from metadata:
```
media/
└── media-photos.glx            # All photo metadata

images/
└── john-smith.jpg              # Actual files
```

### External Storage

For large archives, media files can be stored externally:
```yaml
# Reference external storage
uri: "https://s3.amazonaws.com/family-archives/photos/john-smith.jpg"

# Or use content-addressable storage
uri: "ipfs://QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"
```

## Validation Rules

- `uri` must be a valid URI or path
- If `mime_type` is specified, it should follow standard MIME type format
- If `hash` is specified, it should follow `algorithm:hexstring` format
- All entity references in `subjects`, `source`, `citation` must point to existing entities
- Dimensions (`width`, `height`) must be positive integers
- `duration` and `file_size` must be positive integers

## GEDCOM Mapping

Media entities map to GEDCOM multimedia objects:

| GLX Property | GEDCOM Element | Notes |
|--------------|----------------|-------|
| Entity ID | `@OBJE@` | Media record ID |
| `uri` | `OBJE.FILE` | File path or URL |
| `mime_type` | `OBJE.FORM` | File format |
| `title` | `OBJE.TITL` | Media title |
| `description` | `OBJE.NOTE` | Description/notes |
| - | `OBJE.BLOB` | Embedded media (deprecated) |

GEDCOM Example:
```
0 @M1@ OBJE
1 FILE photos/john-smith.jpg
1 FORM jpeg
1 TITL Portrait of John Smith
1 NOTE Studio portrait, circa 1890
```

GENEALOGIX Equivalent:
```yaml
media:
  media-john-portrait:
    uri: "media/photos/john-smith.jpg"
    mime_type: "image/jpeg"
    title: "Portrait of John Smith"
    description: "Studio portrait, circa 1890"
```

## Media Linking

Media can be linked to entities in multiple ways:

### 1. Via Subjects Array (in Media)
```yaml
media:
  media-photo:
    subjects:
      - person-john-smith
```

### 2. Via Source Reference
```yaml
media:
  media-document:
    source: source-birth-register
```

### 3. Via Citation Reference
```yaml
media:
  media-scan:
    citation: citation-birth-entry
```

### 4. Direct Reference from Entity (if schema supports)
```yaml
persons:
  person-john-smith:
    media:
      - media-portrait
      - media-birth-cert
```

## Schema Reference

See [media.schema.json](../schema/v1/media.schema.json) for the complete JSON Schema definition.

## See Also

- [Source Entity](source.md) - Sources that media documents
- [Citation Entity](citation.md) - Citations that media supports
- [Person Entity](person.md) - People depicted in media
- [Archive Organization](../3-archive-organization.md#file-organization-patterns) - Organizing media files
