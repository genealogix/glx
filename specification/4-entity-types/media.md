---
title: Media Entity
description: Digital and physical media objects - photographs, documents, audio, and video
layout: doc
---

# Media Entity

[← Back to Entity Types](README)

## Overview

A Media entity represents digital or physical media objects that provide visual, audio, or documentary evidence for genealogical research. Media can include photographs, scanned documents, audio recordings, video, or any other supporting materials that help document and illustrate family history.

Media entities serve several purposes:
- Preserve digital copies of original documents
- Provide visual context for people, places, and events
- Document physical artifacts and heirlooms
- Store audio/video recordings of interviews and oral histories
- Serve as direct evidence in assertions (e.g., gravestone photos, handwritten documents)
- Link supporting evidence to citations and sources

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in media/ directory)
media:
  media-birth-cert-scan:
    uri: "media/files/birth-certificate-john-smith.jpg"
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
| `date` | string | Date the media was created |
| `source` | string | Reference to Source entity this media documents |
| `properties` | object | Vocabulary-defined properties (see Properties section) |
| `notes` | string | Free-form notes |

## Required Fields (Detailed)

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Example formats:
  - Descriptive: `birth-cert`, `john-portrait`
  - Random hex: `a1b2c3d4`
  - Prefixed: `media-a1b2c3d4`
  - Sequential: `001`, `002`

### `uri`

- Type: String
- Required: Yes
- Description: Location of the media file

The URI can be:
- **Local file**: `media/files/john-smith.jpg` (recommended — see [File Storage](#file-storage))
- **URL**: `https://example.com/archives/document.pdf` (for external resources)
- **URN**: `urn:hash:sha256:abc123...` (content-addressable)

Example:
```yaml
uri: "media/files/john-smith-1890.jpg"
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

### `source`

- Type: String
- Required: No
- Description: Reference to Source entity that this media documents

Example:
```yaml
source: source-parish-register
```

## Properties

Media entities support vocabulary-defined properties through the `properties` field. The following are standard properties from the default vocabulary; archives can define additional properties by extending the vocabulary.

### Standard Media Properties

| Property | Type | Description |
|----------|------|-------------|
| `subjects` | reference (persons) | People depicted or referenced in the media |
| `width` | integer | Width in pixels (for images and video) |
| `height` | integer | Height in pixels (for images and video) |
| `duration` | integer | Duration in seconds (for audio and video) |
| `file_size` | integer | File size in bytes |
| `crop` | structured | Crop coordinates (top, left, width, height) |
| `medium` | string | Physical medium type (e.g., photograph, film, document) |
| `photographer` | reference (persons) | Person who created the media |
| `location` | reference (places) | Place where the media was created |
| `original_filename` | string | Original filename of the media file |

### Example with Properties

```yaml
media:
  media-family-portrait:
    uri: "media/files/smith-family-1890.jpg"
    mime_type: "image/jpeg"
    title: "Smith Family Portrait, 1890"
    properties:
      subjects:
        - person-john-smith
        - person-mary-smith
        - person-alice-smith
      width: 3200
      height: 2400
      medium: "photograph"
      location: place-leeds-studio
```

### Cropped Region Example

```yaml
media:
  media-census-detail:
    uri: "media/files/census-1851-page-234.jpg"
    mime_type: "image/jpeg"
    title: "1851 Census - Smith Family Entry"
    properties:
      crop:
        top: 450
        left: 100
        width: 800
        height: 200
```

**See [Vocabularies - Media Properties](vocabularies#media-properties-vocabulary) for the full vocabulary definition.**

## Usage Patterns

### Scanned Document

```yaml
media:
  media-birth-cert-john:
    uri: "media/files/birth-certificate-john-smith-1850.pdf"
    mime_type: "application/pdf"
    title: "Birth Certificate - John Smith"
    description: "Original birth certificate from General Register Office, scanned 2024-01-15"
    hash: "sha256:7d865e959b2466918c9863afca942d0fb89d7c9ac0c99bafc3749504ded97730"
```

### Family Photograph

```yaml
media:
  media-smith-family-1890:
    uri: "media/files/smith-family-portrait-1890.jpg"
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
```

### Audio Recording

```yaml
media:
  media-interview-mary-2020:
    uri: "media/files/interview-mary-smith-2020-03-15.mp3"
    mime_type: "audio/mpeg"
    title: "Oral History Interview - Mary Smith"
    description: |
      Interview with Mary Smith about her memories of growing up
      in Leeds in the 1940s and 1950s. Discusses family traditions,
      local history, and genealogical information.
      Recorded 2020-03-15, duration 60 minutes.
```

### Online Resource

```yaml
media:
  media-census-1851-page:
    uri: "https://ancestry.com/imageviewer/1851-census-yorkshire-page-234"
    mime_type: "image/jpeg"
    title: "1851 Census - Yorkshire, Page 234"
    description: "Census page showing Smith family at Wellington Street, Leeds (1851-04-06)"
```

### Historical Document

```yaml
media:
  media-marriage-register-page:
    uri: "media/files/st-pauls-marriage-register-1875-page-45.tiff"
    mime_type: "image/tiff"
    title: "Marriage Register - St Paul's Church, 1875"
    description: |
      Parish register page showing marriage of John Smith and Mary Brown
      on May 10, 1875 at St Paul's Cathedral, Leeds.
    date: "2024-02-10"  # Date photographed
    hash: "sha256:9f3d4c2e7a8b1f6d5c4e3b2a1f0e9d8c7b6a5f4e3d2c1b0a9f8e7d6c5b4a3f2e"
    notes: "High-resolution scan for archival preservation"
```

## Media Types

Media types are defined in the archive's `vocabularies/media-types.glx` file.

**See [Vocabularies - Media Types](vocabularies#media-types-vocabulary) for:**
- Complete list of standard media types
- How to add custom media types
- Vocabulary file structure and examples
- MIME type conventions

## Relationship to Other Entities

```
Media
    ├── source → Source ID (what source this documents)
    ├── referenced by → Citations (citations can include media array)
    └── referenced by → Assertions (assertions can include media array as direct evidence)

Source
    └── documented by → Media (via source field)

Citation
    └── media → array of Media IDs (scans, photos supporting the citation)

Assertion
    └── media → array of Media IDs (direct visual/documentary evidence)
```

## File Storage

Media entities reference files via the `uri` field. Files can be stored locally within the archive or on external storage.

### Local Files: `media/files/`

The standard location for local media files within a GLX archive is **`media/files/`** at the archive root. This flat directory stores the actual binary content (images, PDFs, audio, video) separately from YAML entity metadata:

```
family-archive/
├── persons/
│   └── person-abc12345.glx
├── media/
│   ├── media-portrait.glx        # Media entity metadata (.glx)
│   ├── media-birth-cert.glx
│   └── files/                    # Actual binary files
│       ├── john-smith-1890.jpg
│       ├── birth-certificate.pdf
│       └── interview-2020.mp3
└── vocabularies/
    └── event-types.glx
```

Media entities reference local files using paths relative to the archive root:

```yaml
media:
  media-portrait:
    uri: "media/files/john-smith-1890.jpg"
    mime_type: "image/jpeg"
    title: "Portrait of John Smith"
```

**Why `media/files/`?**
- Separates binary content from YAML entity data
- Easy to apply Git LFS rules or `.gitignore` to one directory
- Portable — relative paths work regardless of where the archive lives
- Tools like `glx import` can populate it automatically

```yaml
# ✅ Good: Relative path from archive root
uri: "media/files/john-smith.jpg"

# ❌ Bad: Absolute path (not portable)
uri: "/home/user/genealogy/media/files/john-smith.jpg"
```

### External Storage

For large archives or shared projects, media files can be stored externally. The `uri` field accepts any valid URI:

```yaml
# Cloud storage
uri: "https://s3.amazonaws.com/family-archives/photos/john-smith.jpg"

# Content-addressable storage
uri: "ipfs://QmXoypizjW3WknFiJnKLwHCnL72vedxjQkDDP1mXWo6uco"
```

When using external URIs, no local file is stored. The media entity simply references the remote resource directly.

## Validation Rules

- `uri` must be a valid URI or path
- `type` must be from the [media types vocabulary](vocabularies#media-types-vocabulary)
- If `mime_type` is specified, it should follow standard MIME type format
- If `hash` is specified, it should follow `algorithm:hexstring` format
- If `date` is specified, it should follow standard date formats
- If `source` is specified, it must reference an existing Source entity

## GEDCOM Mapping

Media entities map to GEDCOM multimedia objects:

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Entity ID | `@OBJE@` | Media record ID |
| `uri` | `OBJE.FILE` | File path or URL |
| `mime_type` | `OBJE.FORM` | File format |
| `title` | `OBJE.TITL` | Media title |
| `description` | `OBJE.NOTE` | Description/notes |
| `uri` | `OBJE.BLOB` | Decoded to file in `media/files/` |

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
    uri: "media/files/john-smith.jpg"
    mime_type: "image/jpeg"
    title: "Portrait of John Smith"
    description: "Studio portrait, circa 1890"
```

### GEDCOM Import: Media File Handling

When importing from GEDCOM, the `glx import` command automatically populates `media/files/`:

- **Relative FILE paths** — copied from the GEDCOM source directory into `media/files/`, URI rewritten (e.g., `photos/portrait.jpg` → `media/files/portrait.jpg`)
- **URLs and absolute paths** — preserved as-is in the `uri` field, no file copied
- **GEDCOM 5.5.1 BLOB data** — decoded from legacy binary encoding and written to `media/files/` (e.g., `media/files/blob-a1b2c3d4.bin`)
- **Duplicate filenames** — deduplicated with a counter suffix (`photo.jpg`, `photo-2.jpg`, `photo-3.jpg`)
- **Missing source files** — produce warnings, not errors; the media entity is still created

## Media Linking

Media can be linked in two ways:

**Via Citations** (when media supports a citation's evidence):

```yaml
citations:
  citation-birth-record:
    source: source-parish-register
    properties:
      locator: "Page 45"
    media:
      - media-birth-cert-scan
      - media-birth-cert-photo
```

**Directly on Assertions** (when media itself is the primary evidence):

```yaml
assertions:
  assertion-john-death:
    subject:
      person: person-john-smith
    property: died_on
    value: "1920-06-20"
    media:
      - media-gravestone-photo
    confidence: medium
    notes: "Date read from gravestone inscription"
```

## Schema Reference

See [media.schema.json](../schema/v1/media.schema.json) for the complete JSON Schema definition.

## See Also

- [Source Entity](source) - Sources that media documents
- [Citation Entity](citation) - Citations that media supports
- [Assertion Entity](assertion) - Assertions that media can directly evidence
- [Person Entity](person) - People depicted in media
- [Archive Organization](../3-archive-organization#organization-strategies) - Organizing media files
