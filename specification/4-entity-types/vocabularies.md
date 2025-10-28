# Vocabularies

[← Back to Entity Types](README.md)

## Overview

GENEALOGIX uses **repository-owned vocabularies** to define controlled lists of types, roles, and classifications used throughout the archive. Vocabularies are stored as YAML files in the `vocabularies/` directory and allow each archive to customize its terminology while maintaining consistency and validation.

## Benefits of Vocabularies

- **Consistency**: Ensures all researchers use the same terminology
- **Validation**: The `glx validate` command checks that all types exist in vocabularies
- **Customization**: Archives can extend standard types with custom definitions
- **Documentation**: Each type can include labels, descriptions, and metadata
- **Interoperability**: Standard types map to GEDCOM and other formats

## Vocabulary Files

All vocabulary files are stored in:
```
vocabularies/
├── event-types.glx
├── relationship-types.glx
├── place-types.glx
├── source-types.glx
├── media-types.glx
├── quality-ratings.glx
├── confidence-levels.glx
├── participant-roles.glx
└── repository-types.glx
```

---

## Event Types Vocabulary

**File**: `vocabularies/event-types.glx`

**Used By**: [Event Entity](event.md#event-types)

**Purpose**: Defines all event and fact types used in the archive (birth, marriage, death, occupation, etc.)

### Structure

```yaml
# vocabularies/event-types.glx
event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
    gedcom: "BIRT"
  
  marriage:
    label: "Marriage"
    description: "Marriage ceremony"
    category: "lifecycle"
    gedcom: "MARR"
  
  occupation:
    label: "Occupation"
    description: "Employment or profession"
    category: "attribute"
    gedcom: "OCCU"
  
  # Custom event types
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `category` | No | Category (lifecycle, attribute, religious, custom) |
| `gedcom` | No | GEDCOM tag mapping |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Event Types

GENEALOGIX provides standardized event type codes for interoperability:

| Type | Label | Category | GEDCOM | Description |
|------|-------|----------|--------|-------------|
| `birth` | Birth | Lifecycle | BIRT | Person's birth |
| `death` | Death | Lifecycle | DEAT | Person's death |
| `marriage` | Marriage | Lifecycle | MARR | Marriage ceremony |
| `divorce` | Divorce | Lifecycle | DIV | Legal dissolution of marriage |
| `engagement` | Engagement | Lifecycle | ENGA | Engagement to be married |
| `adoption` | Adoption | Lifecycle | ADOP | Legal adoption |
| `baptism` | Baptism | Religious | BAPM | Christian baptism ceremony |
| `confirmation` | Confirmation | Religious | CONF | Religious confirmation |
| `bar_mitzvah` | Bar Mitzvah | Religious | BARM | Jewish coming of age ceremony (male) |
| `bat_mitzvah` | Bat Mitzvah | Religious | BATM | Jewish coming of age ceremony (female) |
| `burial` | Burial | Lifecycle | BURI | Burial of remains |
| `cremation` | Cremation | Lifecycle | CREM | Cremation of remains |
| `christening` | Christening | Religious | CHR | Infant christening |
| `residence` | Residence | Attribute | RESI | Place of residence |
| `occupation` | Occupation | Attribute | OCCU | Employment or profession |
| `title` | Title | Attribute | TITL | Nobility or honorific title |
| `nationality` | Nationality | Attribute | NATI | National citizenship |
| `religion` | Religion | Attribute | RELI | Religious affiliation |
| `education` | Education | Attribute | EDUC | Educational achievement |

**Complete Vocabulary Example:**

```yaml
# vocabularies/event-types.glx
event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
    gedcom: "BIRT"
  
  death:
    label: "Death"
    description: "Person's death"
    category: "lifecycle"
    gedcom: "DEAT"
  
  marriage:
    label: "Marriage"
    description: "Marriage ceremony"
    category: "lifecycle"
    gedcom: "MARR"
  
  baptism:
    label: "Baptism"
    description: "Religious baptism ceremony"
    category: "religious"
    gedcom: "BAPM"
  
  occupation:
    label: "Occupation"
    description: "Employment or profession"
    category: "attribute"
    gedcom: "OCCU"
  
  residence:
    label: "Residence"
    description: "Place of residence"
    category: "attribute"
    gedcom: "RESI"
```

### Adding Custom Event Types

Add custom event types for specialized research:

```yaml
# vocabularies/event-types.glx
event_types:
  # ... standard types ...
  
  # Custom types
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"
    custom: true
  
  land-grant:
    label: "Land Grant"
    description: "Receipt of land grant or patent"
    category: "property"
    gedcom: "_LAND"
    custom: true
```

---

## Relationship Types Vocabulary

**File**: `vocabularies/relationship-types.glx`

**Used By**: [Relationship Entity](relationship.md#relationship-types)

**Purpose**: Defines all relationship types between persons (marriage, parent-child, sibling, etc.)

### Structure

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
  
  sibling:
    label: "Sibling"
    description: "Brother or sister relationship"
    gedcom: "SIBL"
  
  # Custom relationship types
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor relationship"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `gedcom` | No | GEDCOM tag mapping |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Relationship Types

GENEALOGIX provides standardized relationship type codes for interoperability:

| Type | Label | GEDCOM | Description |
|------|-------|--------|-------------|
| `marriage` | Marriage | MARR | Legal or religious union of two people |
| `parent-child` | Parent-Child | CHIL/FAMC | Biological, adoptive, or legal parent-child relationship |
| `sibling` | Sibling | SIBL | Brother or sister relationship |
| `adoption` | Adoption | ADOP | Legal adoption relationship |
| `step-parent` | Step-Parent | - | Step-parent through marriage |
| `godparent` | Godparent | - | Spiritual sponsor relationship |
| `guardian` | Guardian | - | Legal guardian relationship |
| `partner` | Partner | - | Domestic partnership or cohabitation |

**Complete Vocabulary Example:**

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
  
  sibling:
    label: "Sibling"
    description: "Brother or sister relationship"
    gedcom: "SIBL"
  
  adoption:
    label: "Adoption"
    description: "Legal adoption relationship"
    gedcom: "ADOP"
  
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor relationship"
```

### Adding Custom Relationship Types

Add custom relationship types for specialized research:

```yaml
# vocabularies/relationship-types.glx
relationship_types:
  # ... standard types ...
  
  # Custom types
  blood-brother:
    label: "Blood Brother"
    description: "Non-biological brotherhood bond through ceremony"
    custom: true
  
  chosen-family:
    label: "Chosen Family"
    description: "Close familial bond without biological or legal tie"
    custom: true
```

---

## Place Types Vocabulary

**File**: `vocabularies/place-types.glx`

**Used By**: [Place Entity](place.md#place-types)

**Purpose**: Defines geographic and administrative place classifications (country, state, city, parish, etc.)

### Structure

```yaml
# vocabularies/place-types.glx
place_types:
  country:
    label: "Country"
    description: "Nation state or country"
    category: "administrative"
  
  county:
    label: "County"
    description: "County or similar administrative division"
    category: "administrative"
  
  city:
    label: "City"
    description: "City or town"
    category: "geographic"
  
  parish:
    label: "Parish"
    description: "Church parish or ecclesiastical division"
    category: "religious"
  
  # Custom place types
  plantation:
    label: "Plantation"
    description: "Agricultural estate or plantation"
    category: "geographic"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `category` | No | Category (administrative, geographic, religious) |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Place Types

GENEALOGIX provides standardized place type codes for geographic and administrative classifications:

| Type | Label | Category | Description |
|------|-------|----------|-------------|
| `country` | Country | Administrative | Nation state or country |
| `state` | State/Province | Administrative | State, province, or region |
| `county` | County | Administrative | County or similar administrative division |
| `city` | City | Geographic | City or town |
| `town` | Town | Geographic | Town or village |
| `parish` | Parish | Religious | Church parish or ecclesiastical division |
| `district` | District | Administrative | Administrative district |
| `region` | Region | Geographic | Geographic region |
| `neighborhood` | Neighborhood | Geographic | Neighborhood or locality |
| `street` | Street | Geographic | Street or road |
| `building` | Building | Geographic | Specific building or structure |

**Complete Vocabulary Example:**

```yaml
# vocabularies/place-types.glx
place_types:
  country:
    label: "Country"
    description: "Nation state or country"
    category: "administrative"
  
  state:
    label: "State/Province"
    description: "State, province, or region"
    category: "administrative"
  
  county:
    label: "County"
    description: "County or similar administrative division"
    category: "administrative"
  
  city:
    label: "City"
    description: "City or town"
    category: "geographic"
  
  parish:
    label: "Parish"
    description: "Church parish or ecclesiastical division"
    category: "religious"
```

### Adding Custom Place Types

Add custom place types for specialized research:

```yaml
# vocabularies/place-types.glx
place_types:
  # ... standard types ...
  
  # Custom types
  plantation:
    label: "Plantation"
    description: "Agricultural estate or plantation"
    category: "geographic"
    custom: true
  
  mission:
    label: "Mission"
    description: "Religious mission station"
    category: "religious"
    custom: true
```

---

## Source Types Vocabulary

**File**: `vocabularies/source-types.glx`

**Used By**: [Source Entity](source.md#source-types)

**Purpose**: Defines categories of sources (vital records, census, church registers, newspapers, etc.)

### Structure

```yaml
# vocabularies/source-types.glx
source_types:
  vital_record:
    label: "Vital Record"
    description: "Birth, marriage, death certificates"
    
  census:
    label: "Census Record"
    description: "Population census enumerations"
    
  church_register:
    label: "Church Register"
    description: "Parish registers of baptisms, marriages, burials"
    
  newspaper:
    label: "Newspaper"
    description: "Newspapers, periodicals, gazettes"
  
  oral_history:
    label: "Oral History"
    description: "Interviews, recorded memories"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Source Types

GENEALOGIX provides standardized source type codes for different categories of genealogical sources:

| Type | Label | Description |
|------|-------|-------------|
| `vital_record` | Vital Record | Birth, marriage, death certificates |
| `census` | Census Record | Census records and population enumerations |
| `church_register` | Church Register | Parish registers of baptisms, marriages, burials |
| `military` | Military Record | Military service records, pension files |
| `newspaper` | Newspaper | Newspapers, periodicals, gazettes |
| `probate` | Probate Record | Wills, probate records, estate files |
| `land` | Land Record | Deeds, land grants, property records |
| `court` | Court Record | Court records, legal proceedings |
| `immigration` | Immigration Record | Passenger lists, naturalization records |
| `directory` | Directory | City directories, telephone books |
| `book` | Published Book | Published genealogies, family histories |
| `database` | Online Database | Online databases, compiled records |
| `oral_history` | Oral History | Interviews, recorded memories |
| `correspondence` | Correspondence | Letters, emails, personal papers |
| `photograph` | Photograph Collection | Photo collections |
| `other` | Other | Other source types |

**Complete Vocabulary Example:**

```yaml
# vocabularies/source-types.glx
source_types:
  vital_record:
    label: "Vital Record"
    description: "Birth, marriage, death certificates"
  
  census:
    label: "Census Record"
    description: "Population census enumerations"
  
  church_register:
    label: "Church Register"
    description: "Parish registers of baptisms, marriages, burials"
  
  military:
    label: "Military Record"
    description: "Service records, pension files"
  
  newspaper:
    label: "Newspaper"
    description: "Newspapers, periodicals, gazettes"
  
  probate:
    label: "Probate Record"
    description: "Wills, probate records, estate files"
```

### Adding Custom Source Types

Add custom source types for specialized research:

```yaml
# vocabularies/source-types.glx
source_types:
  # ... standard types ...
  
  # Custom types
  oral_history:
    label: "Oral History"
    description: "Interviews, recorded memories"
    custom: true
```

---

## Media Types Vocabulary

**File**: `vocabularies/media-types.glx`

**Used By**: [Media Entity](media.md#media-types)

**Purpose**: Defines categories of media objects (photographs, documents, audio, video, etc.)

### Structure

```yaml
# vocabularies/media-types.glx
media_types:
  photograph:
    label: "Photograph"
    description: "Photographic image"
    mime_type: "image/jpeg"
  
  document:
    label: "Document"
    description: "Scanned or digital document"
    mime_type: "application/pdf"
  
  audio:
    label: "Audio Recording"
    description: "Audio interview or recording"
    mime_type: "audio/mpeg"
  
  video:
    label: "Video Recording"
    description: "Video recording or footage"
    mime_type: "video/mp4"
  
  certificate:
    label: "Certificate"
    description: "Official certificate or license"
    mime_type: "image/tiff"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `mime_type` | No | Default MIME type for this media type |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Media Types

GENEALOGIX provides standardized media type codes for different categories of digital and physical media:

| Type | Label | Default MIME Type | Description |
|------|-------|------------------|-------------|
| `photograph` | Photograph | image/jpeg | Photographic images |
| `document` | Document | application/pdf | Scanned or digital documents |
| `audio` | Audio Recording | audio/mpeg | Audio interviews or recordings |
| `video` | Video Recording | video/mp4 | Video recordings or footage |
| `scan` | Scan | image/tiff | High-resolution scans of documents |
| `image` | Image | image/png | General images |
| `certificate` | Certificate | image/tiff | Official certificates or licenses |

**Complete Vocabulary Example:**

```yaml
# vocabularies/media-types.glx
media_types:
  photograph:
    label: "Photograph"
    description: "Photographic image"
    mime_type: "image/jpeg"
  
  document:
    label: "Document"
    description: "Scanned or digital document"
    mime_type: "application/pdf"
  
  audio:
    label: "Audio Recording"
    description: "Audio interview or recording"
    mime_type: "audio/mpeg"
  
  video:
    label: "Video Recording"
    description: "Video recording or footage"
    mime_type: "video/mp4"
```

### Adding Custom Media Types

Add custom media types for specialized collections:

```yaml
# vocabularies/media-types.glx
media_types:
  # ... standard types ...
  
  # Custom types
  certificate:
    label: "Certificate"
    description: "Official certificate or license"
    mime_type: "image/tiff"
    custom: true
  
  artifact:
    label: "Artifact"
    description: "3D scan or photo of physical artifact"
    mime_type: "model/gltf"
    custom: true
```

---

## Quality Ratings Vocabulary

**File**: `vocabularies/quality-ratings.glx`

**Used By**: [Citation Entity](citation.md#evidence-quality), [Assertion Entity](assertion.md)

**Purpose**: Defines the meaning of citation quality ratings (0-3 scale, GEDCOM QUAY compatible)

### Structure

```yaml
# vocabularies/quality-ratings.glx
quality_ratings:
  3:
    label: "Primary source"
    description: "Original document created at time of event"
    examples:
      - "Birth certificate"
      - "Original parish register"
      - "Contemporary diary entry"
  
  2:
    label: "Secondary source"
    description: "Record created after event"
    examples:
      - "Census record"
      - "Death certificate for birth information"
      - "Published vital records index"
  
  1:
    label: "Questionable"
    description: "Conflicting or unreliable evidence"
    examples:
      - "Undocumented oral history"
      - "Conflicting sources"
  
  0:
    label: "Estimated"
    description: "No direct evidence, estimated from other data"
    examples:
      - "Unverified family tradition"
      - "Calculated from age at death"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Short label for this quality level |
| `description` | No | Detailed description |
| `examples` | No | Array of example scenarios |

### Important Notes

- **Archive-defined**: Each archive determines what these ratings mean
- **GEDCOM compatibility**: 0-3 scale maps 1:1 to GEDCOM 5.5.1 QUAY values
- **Optional**: Archives can omit quality ratings entirely
- **Alternative**: Use assertion `confidence` levels instead

See [Core Concepts - Evidence Hierarchy](../2-core-concepts.md#evidence-hierarchy) for details on quality assessment.

---

## Confidence Levels Vocabulary

**File**: `vocabularies/confidence-levels.glx`

**Used By**: [Assertion Entity](assertion.md#confidence)

**Purpose**: Defines confidence levels for assertions (alternative to citation quality ratings)

### Structure

```yaml
# vocabularies/confidence-levels.glx
confidence_levels:
  high:
    label: "High Confidence"
    description: "Multiple high-quality sources agree, minimal uncertainty"
  
  medium:
    label: "Medium Confidence"
    description: "Some evidence supports conclusion, but conflicts or gaps exist"
  
  low:
    label: "Low Confidence"
    description: "Limited evidence, significant uncertainty"
  
  disputed:
    label: "Disputed"
    description: "Multiple sources conflict, resolution unclear"
  
  # Custom confidence levels
  tentative:
    label: "Tentative"
    description: "Working hypothesis pending additional research"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `custom` | No | Mark as custom (non-standard) level |

### Important Notes

- **Alternative to quality ratings**: Use confidence levels on assertions instead of quality ratings on citations
- **Researcher's judgment**: Reflects overall confidence in conclusion, not just source quality
- **Archive-defined**: Each archive can customize the meaning of confidence levels

See [Assertion Entity - Confidence](assertion.md#confidence) for usage details.

---

## Repository Types Vocabulary

**File**: `vocabularies/repository-types.glx`

**Used By**: [Repository Entity](repository.md#repository-types)

**Purpose**: Defines categories of repositories (archives, libraries, churches, online databases, etc.)

### Structure

```yaml
# vocabularies/repository-types.glx
repository_types:
  archive:
    label: "Archive"
    description: "Government or historical archive"
  
  library:
    label: "Library"
    description: "Public, university, or specialty library"
  
  church:
    label: "Church"
    description: "Church or religious organization archives"
  
  database:
    label: "Online Database"
    description: "Online genealogical database service"
  
  museum:
    label: "Museum"
    description: "Museum with genealogical collections"
  
  # Custom repository types
  historical_society:
    label: "Historical Society"
    description: "Local historical society"
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `custom` | No | Mark as custom (non-standard) type |

### Standard Repository Types

See [Repository Entity](repository.md#repository-types) for the complete list of standard repository types.

---

## Participant Roles Vocabulary

**File**: `vocabularies/participant-roles.glx`

**Used By**: [Event Entity](event.md#participant-roles), [Relationship Entity](relationship.md#participant-roles)

**Purpose**: Defines roles that people play in events and relationships (principal, witness, officiant, etc.)

### Structure

```yaml
# vocabularies/participant-roles.glx
participant_roles:
  # Event roles
  principal:
    label: "Principal"
    description: "Primary person in the event"
    applies_to:
      - event
  
  witness:
    label: "Witness"
    description: "Person who witnessed the event"
    applies_to:
      - event
  
  officiant:
    label: "Officiant"
    description: "Person who officiated the ceremony"
    applies_to:
      - event
  
  # Relationship roles
  spouse:
    label: "Spouse"
    description: "Marriage partner"
    applies_to:
      - relationship
  
  parent:
    label: "Parent"
    description: "Parent in parent-child relationship"
    applies_to:
      - relationship
  
  child:
    label: "Child"
    description: "Child in parent-child relationship"
    applies_to:
      - relationship
  
  # Custom roles
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor at baptism"
    applies_to:
      - event
      - relationship
    custom: true
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `applies_to` | No | Array of entity types (event, relationship) |
| `custom` | No | Mark as custom (non-standard) role |

### Standard Participant Roles

Common event roles:
- `principal` - Primary person in the event
- `subject` - Subject of the event (alias for principal)
- `groom`, `bride` - Marriage participants
- `witness` - Event witness
- `officiant` - Ceremony officiant
- `informant` - Person providing information

Common relationship roles:
- `spouse` - Marriage partner
- `parent` - Parent in parent-child relationship
- `child` - Child in parent-child relationship
- `adoptive-parent`, `adopted-child` - Adoption roles
- `sibling` - Brother or sister

---

## Vocabulary Validation

The `glx validate` command checks:

1. **Type existence**: All types used in entities must be defined in vocabularies
2. **Required fields**: Each vocabulary entry has required fields (e.g., `label`)
3. **Format**: Vocabulary files follow proper YAML structure
4. **References**: Custom types are marked with `custom: true`

Example validation:

```bash
$ glx validate

✓ vocabularies/event-types.glx
✓ vocabularies/relationship-types.glx
✓ events/event-birth.glx
  - event type 'birth' found in vocabulary
✓ relationships/rel-marriage.glx
  - relationship type 'marriage' found in vocabulary

❌ events/event-custom.glx
  - ERROR: event type 'unknown-type' not found in vocabularies/event-types.glx
```

---

## Creating Custom Types

### Step 1: Add to Vocabulary File

```yaml
# vocabularies/event-types.glx
event_types:
  # ... standard types ...
  
  # Custom types
  land-grant:
    label: "Land Grant"
    description: "Receipt of land grant or patent"
    category: "property"
    gedcom: "_LAND"  # Non-standard GEDCOM tag
    custom: true
```

### Step 2: Use in Entity

```yaml
# events/event-land-grant.glx
events:
  event-john-land-grant:
    version: "1.0"
    type: land-grant  # Custom type from vocabulary
    date: "1850-03-10"
    place: place-indiana
    value: "160 acres in Howard County"
```

### Step 3: Validate

```bash
$ glx validate events/event-land-grant.glx
✓ events/event-land-grant.glx
  - event type 'land-grant' found in vocabulary (custom)
```

---

## Best Practices

### Use Standard Types First

Before creating custom types, check if standard types meet your needs. Standard types:
- Ensure GEDCOM compatibility
- Work with most genealogy software
- Are understood by other researchers

### Document Custom Types

When adding custom types:
- Provide clear `label` and `description`
- Mark with `custom: true`
- Document why the custom type is needed
- Include GEDCOM mapping if possible (use `_TAG` format for custom GEDCOM tags)

### Keep Vocabularies Consistent

- Use consistent naming conventions (lowercase with hyphens)
- Group related types together
- Add comments to explain complex types
- Version vocabulary files alongside schema updates

### Share Vocabularies

- Custom vocabularies can be shared between archives
- Archives working on similar research can standardize custom types
- Consider submitting useful custom types as proposals for standard types

---

## Vocabulary Extensibility

Vocabularies support various extension mechanisms:

### Additional Fields

Add custom fields to vocabulary entries:

```yaml
event_types:
  baptism:
    label: "Baptism"
    description: "Religious baptism ceremony"
    category: "religious"
    gedcom: "BAPM"
    # Custom fields
    icon: "water-drop"
    color: "#4A90E2"
    requires_place: true
    requires_participants: true
```

### Hierarchical Types

Create type hierarchies:

```yaml
event_types:
  occupation:
    label: "Occupation"
    category: "attribute"
    
  occupation.agricultural:
    label: "Agricultural Occupation"
    parent: "occupation"
    
  occupation.agricultural.farmer:
    label: "Farmer"
    parent: "occupation.agricultural"
```

### Localization

Add translations for labels:

```yaml
place_types:
  county:
    label: "County"
    label_es: "Condado"
    label_fr: "Comté"
    label_de: "Landkreis"
    description: "County or similar administrative division"
```

---

## Schema Reference

Each vocabulary type has a corresponding JSON Schema for validation:

| Vocabulary | Schema File |
|------------|-------------|
| Event Types | [event-types.schema.json](../schema/v1/vocabularies/event-types.schema.json) |
| Relationship Types | [relationship-types.schema.json](../schema/v1/vocabularies/relationship-types.schema.json) |
| Place Types | [place-types.schema.json](../schema/v1/vocabularies/place-types.schema.json) |
| Source Types | (included in source.schema.json) |
| Media Types | [media-types.schema.json](../schema/v1/vocabularies/media-types.schema.json) |
| Quality Ratings | [quality-ratings.schema.json](../schema/v1/vocabularies/quality-ratings.schema.json) |
| Participant Roles | [participant-roles.schema.json](../schema/v1/vocabularies/participant-roles.schema.json) |
| Repository Types | [repository-types.schema.json](../schema/v1/vocabularies/repository-types.schema.json) |
| Confidence Levels | [confidence-levels.schema.json](../schema/v1/vocabularies/confidence-levels.schema.json) |

All vocabulary schemas are located in `specification/schema/v1/vocabularies/` and define:
- Required top-level key (e.g., `event_types`, `relationship_types`)
- Required fields for each entry (typically `label`)
- Optional fields (e.g., `description`, `gedcom`, `custom`)
- Pattern properties for vocabulary keys (alphanumeric with hyphens, 1-64 characters)

Vocabulary files are validated by the `glx validate` command using these schemas.

## See Also

- [Core Concepts - Repository-Owned Vocabularies](../2-core-concepts.md#repository-owned-vocabularies)
- [Archive Organization](../3-archive-organization.md) - Where vocabulary files are stored
- [Event Entity](event.md) - Event types vocabulary
- [Relationship Entity](relationship.md) - Relationship types vocabulary
- [Place Entity](place.md) - Place types vocabulary
- [Source Entity](source.md) - Source types vocabulary
- [Media Entity](media.md) - Media types vocabulary
- [Citation Entity](citation.md) - Quality ratings usage

---

