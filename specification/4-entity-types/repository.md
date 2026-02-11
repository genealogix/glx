---
title: Repository Entity
description: Institutions and archives that hold genealogical sources
layout: doc
---

# Repository Entity

[← Back to Entity Types](README)

## Overview

A Repository entity represents an institution, archive, library, or organization that holds genealogical sources. Repositories provide the physical or virtual location where evidence is housed and can be accessed.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in repositories/ directory)
repositories:
  repository-national-archives:
    name: "The National Archives"
    type: archive
    website: "https://www.nationalarchives.gov.uk"
```

**Key Points:**
- Entity ID is the map key (`repository-national-archives`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

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

**See [Vocabularies - Repository Types](vocabularies#repository-types-vocabulary) for:**
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
| `website` | string | URL to repository website |
| `properties` | object | Vocabulary-defined properties (see below) |
| `notes` | string | Free-form notes |

### Properties

Contact information and access details are stored in the `properties` field. The following are standard properties from the default vocabulary; archives can define additional properties by extending the vocabulary:

| Property | Type | Description |
|----------|------|-------------|
| `phones` | string[] | Phone number(s) for the repository |
| `emails` | string[] | Email address(es) for the repository |
| `fax` | string | Fax number |
| `access_hours` | string | Hours of operation/access |
| `access_restrictions` | string | Any restrictions on access |
| `holding_types` | string[] | Types of materials held |
| `external_ids` | string[] | External identifiers (FamilySearch, WikiTree, etc.) |

**See [Vocabularies - Repository Properties](vocabularies#repository-properties-vocabulary) for the full vocabulary definition.**

## Usage Patterns

### Simple Local Repository

```yaml
repositories:
  repository-leeds-library:
    name: "Leeds Library - Local Studies"
    type: library
    address: "Market Street"
    city: "Leeds"
    state_province: "Yorkshire"
    country: "England"
    website: "https://www.leeds.gov.uk/libraries"
    properties:
      phones: "+44 113 247 6000"
```

### National Archive

```yaml
repositories:
  repository-tna:
    name: "The National Archives"
    type: archive
    address: "Kew, Richmond"
    city: "London"
    country: "England"
    website: "https://www.nationalarchives.gov.uk"
    properties:
      phones: "+44 20 8876 3444"
      emails: "enquiry@nationalarchives.gov.uk"
      access_hours: "Monday-Friday 9am-5pm"
      holding_types:
        - "government records"
        - "microfilm"
        - "digital images"
      access_restrictions: "Some records require appointment"
```

### Online Database

```yaml
repositories:
  repository-ancestry:
    name: "Ancestry.com"
    type: database
    website: "https://www.ancestry.com"
    properties:
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
repositories:
  repository-stpauls:
    name: "St Paul's Cathedral Archive"
    type: church
    address: "St Paul's Churchyard"
    city: "London"
    country: "England"
    properties:
      phones: "+44 20 7246 8348"
      emails: "archive@stpauls.co.uk"
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

## Validation Rules

- `name` must be present and non-empty
- `type` must be from the [repository types vocabulary](vocabularies#repository-types-vocabulary)
- If `website` is specified, it should be a valid URL
- Properties must be defined in the repository properties vocabulary

## GEDCOM Mapping

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Entity ID (map key) | (synthetic) | Not in GEDCOM |
| `name` | REPO.NAME | Repository name |
| `address` | REPO.ADDR | Repository address |
| `website` | REPO.WWW | Website (non-standard) |
| `properties.phones` | REPO.PHON | Phone number(s) |
| `properties.emails` | REPO.EMAIL | Email address(es) |
| `properties.external_ids` | REPO.EXID | External identifiers (GEDCOM 7.0) |

## Access Information

Best practices for recording repository access:

- Include both physical address and website URL if applicable
- Note any access restrictions in `properties.access_restrictions`
- Record hours of operation in `properties.access_hours`
- Include contact details in `properties.phones` and `properties.emails`
- Document types of materials held in `properties.holding_types`

## Related Entities

- **Source**: References specific collections within repositories
- **Citation**: May reference repository location information via Source
- **Holdings**: Detailed inventory of materials within repository

## Schema Reference

See [repository.schema.json](../schema/v1/repository.schema.json) for the complete JSON Schema definition.

## See Also

- [Source Entity](source) - Collections held in repositories
- [Citation Entity](citation) - References to specific materials in repositories




