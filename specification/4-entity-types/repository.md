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

## Properties

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier (format: `repository-[hex8]`) |
| `version` | string | Schema version (e.g., "1.0") |
| `name` | string | Name of the repository |

### Optional Properties

| Property | Type | Description |
|----------|------|-------------|
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
| `access_restrictions` | string | Any restrictions on access |
| `holding_types` | array | Types of materials held (microfilm, digital, books, etc.) |
| `created_at` | datetime | Creation timestamp |
| `created_by` | string | User who created this record |
| `modified_at` | datetime | Last modification timestamp |
| `modified_by` | string | User who last modified this record |
| `notes` | string | Free-form notes about repository |

## Usage Patterns

### Simple Local Repository

```yaml
---
id: repository-leeds001
version: "1.0"
name: "Leeds Library - Local Studies"
type: "library"
address: "Market Street"
city: "Leeds"
state_province: "Yorkshire"
country: "England"
phone: "+44 113 247 6000"
website: "https://www.leeds.gov.uk/libraries"
created_at: "2025-01-15T10:30:00Z"
created_by: "researcher@example.com"
```

### National Archive

```yaml
---
id: repository-tna001
version: "1.0"
name: "The National Archives"
type: "archive"
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
holding_types: ["documents", "microfilm", "digital"]
access_restrictions: "Some records require appointment"
created_at: "2025-01-15T10:30:00Z"
created_by: "researcher@example.com"
```

### Online Database

```yaml
---
id: repository-ancestry001
version: "1.0"
name: "Ancestry.com"
type: "database"
website: "https://www.ancestry.com"
access_hours: "24/7"
holding_types:
  - "census records"
  - "vital records"
  - "military records"
  - "church records"
  - "newspapers"
access_restrictions: "Subscription required"
created_at: "2025-01-15T10:30:00Z"
created_by: "researcher@example.com"
```

### Church Archive

```yaml
---
id: repository-church001
version: "1.0"
name: "St Paul's Cathedral Archive"
type: "church"
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
created_at: "2025-01-15T10:30:00Z"
created_by: "researcher@example.com"
```

## Repository in Sources

Repositories are referenced from Source entities:

```yaml
# In Source entity
---
id: source-tna-wills
version: "1.0"
title: "Wills and Probate Records"
repository_id: "repository-tna001"
collection: "PROB 11"
description: "Wills proved in the Prerogative Court of Canterbury"
```

## File Organization

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
| `id` | (synthetic) | Not in GEDCOM |
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




