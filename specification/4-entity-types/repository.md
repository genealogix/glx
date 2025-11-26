---
title: Repository Entity
description: Institutions and archives that hold genealogical sources
layout: doc
---

# Repository Entity

[← Back to Entity Types](README.md)

## Overview

A Repository entity represents an institution, archive, library, or organization that holds genealogical sources. Repositories provide the physical or virtual location where evidence is housed and can be accessed.

## Core Concepts

### Repository Types

GENEALOGIX supports various repository types:

- **Archive**: Government or historical archive (national, regional, local)
- **Library**: Public, university, or specialty library
- **Museum**: Museum with genealogical collections
- **Registry**: Civil registration office or vital records office
- **Database**: Online genealogical database service
- **Church**: Church or religious organization archives
- **Historical Society**: Local historical society
- **University**: University special collections or archives
- **Government Agency**: Government record-keeping agency
- **Other**: Other institution type

**See [Vocabularies - Repository Types](vocabularies.md#repository-types-vocabulary) for:**
- Complete list of standard repository types
- How to add custom repository types
- Vocabulary file structure and examples

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `name` | string | Name of the repository |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Repository type (see types above) |
| `address` | string | Physical address |
| `city` | string | City/town |
| `state_province` | string | State or province |
| `postal_code` | string | Postal/zip code |
| `country` | string | Country |
| `phone` | string | Telephone number |
| `email` | string | Email address |
| `website` | string | URL to repository website |
| `access_hours` | string | Hours of operation/access |
| `properties` | object | Vocabulary-defined properties |
| `notes` | string | Free-form notes |
| `tags` | array | Tags for categorization |
| `access_restrictions` | string | Any restrictions on access |
| `holding_types` | array | Types of materials held (microfilm, digital, books, etc.) |

## Usage Patterns

### Simple Local Repository

```yaml
# repositories/repository-leeds.glx
repositories:
  repository-leeds-library:
    name: "Leeds Library - Local Studies"
    type: library
    address: "Market Street"
    city: "Leeds"
    state_province: "Yorkshire"
    country: "England"
    phone: "+44 113 247 6000"
    website: "https://www.leeds.gov.uk/libraries"
```

### National Archive

```yaml
# repositories/repository-tna.glx
repositories:
  repository-tna:
    name: "The National Archives"
    type: archive
    address: "Kew, Richmond"
    city: "London"
    country: "England"
    phone: "+44 20 8876 3444"
    email: "enquiry@nationalarchives.gov.uk"
    website: "https://www.nationalarchives.gov.uk"
    access_hours: "Monday-Friday 9am-5pm"
    holding_types:
      - "government records"
      - "microfilm"
      - "digital images"
    access_restrictions: "Some records require appointment"
```

### Online Database

```yaml
# repositories/repository-ancestry.glx
repositories:
  repository-ancestry:
    name: "Ancestry.com"
    type: database
    website: "https://www.ancestry.com"
    access_hours: "24/7"
    holding_types:
      - "census records"
      - "vital records"
      - "military records"
      - "church records"
      - "newspapers"
    access_restrictions: "Subscription required"
```

### Church Archive

```yaml
# repositories/repository-church.glx
repositories:
  repository-stpauls:
    name: "St Paul's Cathedral Archive"
    type: church
    address: "St Paul's Churchyard"
    city: "London"
    country: "England"
    phone: "+44 20 7246 8348"
    email: "archive@stpauls.co.uk"
    holding_types:
      - "parish registers"
      - "marriage records"
      - "baptismal records"
      - "manuscripts"
    access_restrictions: "By appointment only"
```

## Repository in Sources

Repositories are referenced from Source entities:

```yaml
# sources/source-wills.glx
sources:
  source-tna-wills:
    title: "Wills and Probate Records"
    repository: repository-tna
    description: "Wills proved in the Prerogative Court of Canterbury"
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Repository files are typically stored in a `repositories/` directory:

```
repositories/
├── archives/
│   ├── repository-tna.glx
│   ├── repository-spa.glx
│   └── repository-local.glx
├── libraries/
│   ├── repository-bm.glx
│   └── repository-leeds.glx
├── churches/
│   ├── repository-stpauls.glx
│   └── repository-westminster.glx
├── online/
│   ├── repository-ancestry.glx
│   ├── repository-familysearch.glx
│   └── repository-findmypast.glx
└── other/
    └── repository-custom.glx
```

## GEDCOM Mapping

| GLX Property | GEDCOM Element | Notes |
|--------------|----------------|-------|
| Entity ID (map key) | (synthetic) | Not in GEDCOM |
| `name` | REPO.NAME | Repository name |
| `address` | REPO.ADDR | Repository address |
| `phone` | REPO.PHON | Phone number (non-standard) |
| `email` | REPO.EMAIL | Email (non-standard) |
| `website` | REPO.WWW | Website (non-standard) |

## Access Information

Best practices for recording repository access:

- Include both physical address and website URL if applicable
- Note any access restrictions or requirements (appointment, membership, subscription)
- Record hours of operation for physical repositories
- Include phone/email for inquiries
- Document required identification or credentials

## Related Entities

- **Source**: References specific collections within repositories
- **Citation**: May reference repository location information via Source
- **Holdings**: Detailed inventory of materials within repository

## See Also

- [Source Entity](source.md) - Collections held in repositories
- [Citation Entity](citation.md) - References to specific materials in repositories




