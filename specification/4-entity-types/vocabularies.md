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

All vocabulary files are stored in the `vocabularies/` directory of each archive:
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

When initializing a new archive with `glx init`, these files are automatically copied from the [Standard Vocabularies](../5-standard-vocabularies/) templates.

---

## Event Types Vocabulary

**File**: `vocabularies/event-types.glx`

**Used By**: [Event Entity](event.md#event-types)

**Purpose**: Defines all event and fact types used in the archive (birth, marriage, death, occupation, etc.)

**Standard Templates**: See [Standard Vocabularies - Event Types](/specification/5-standard-vocabularies/#event-types) for the complete default vocabulary with all standard event types.

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

**Standard Event Types**: GENEALOGIX provides 19 standardized event type codes including lifecycle events (birth, death, marriage), religious events (baptism, confirmation, bar/bat mitzvah), and attribute facts (occupation, residence, education, title, nationality, religion).

**Complete List**: See [Standard Vocabularies - Event Types](/specification/5-standard-vocabularies/#event-types) for the complete default vocabulary file with all standard types.

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

**Standard Templates**: See [Standard Vocabularies - Relationship Types](/specification/5-standard-vocabularies/#relationship-types) for the complete default vocabulary with all standard relationship types.

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

**Standard Relationship Types**: GENEALOGIX provides 8 standardized relationship type codes including marriage, parent-child, sibling, adoption, step-parent, godparent, guardian, and partner relationships.

**Complete List**: See [Standard Vocabularies - Relationship Types](/specification/5-standard-vocabularies/#relationship-types) for the complete default vocabulary file with all standard types.

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

**Standard Templates**: See [Standard Vocabularies - Place Types](/specification/5-standard-vocabularies/#place-types) for the complete default vocabulary with all standard place types.

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

**Standard Place Types**: GENEALOGIX provides 11 standardized place type codes including administrative divisions (country, state, county, district), geographic features (city, town, region, neighborhood, street, building), and religious divisions (parish).

**Complete List**: See [Standard Vocabularies - Place Types](/specification/5-standard-vocabularies/#place-types) for the complete default vocabulary file with all standard types.

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

**Standard Templates**: See [Standard Vocabularies - Source Types](/specification/5-standard-vocabularies/#source-types) for the complete default vocabulary with all standard source types.

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

**Standard Source Types**: GENEALOGIX provides 16 standardized source type codes including vital records, census, church registers, military records, newspapers, probate, land records, court records, immigration records, directories, books, databases, oral history, correspondence, photograph collections, and other.

**Complete List**: See [Standard Vocabularies - Source Types](/specification/5-standard-vocabularies/#source-types) for the complete default vocabulary file with all standard types.

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

**Standard Templates**: See [Standard Vocabularies - Media Types](/specification/5-standard-vocabularies/#media-types) for the complete default vocabulary with all standard media types.

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

**Standard Media Types**: GENEALOGIX provides 7 standardized media type codes including photograph, document, audio, video, scan, image, and certificate, each with default MIME types.

**Complete List**: See [Standard Vocabularies - Media Types](/specification/5-standard-vocabularies/#media-types) for the complete default vocabulary file with all standard types.

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

**Standard Templates**: See [Standard Vocabularies - Quality Ratings](/specification/5-standard-vocabularies/#quality-ratings) for the complete default vocabulary with all standard ratings.

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

**Standard Templates**: See [Standard Vocabularies - Confidence Levels](/specification/5-standard-vocabularies/#confidence-levels) for the complete default vocabulary with all standard confidence levels.

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

**Standard Templates**: See [Standard Vocabularies - Repository Types](/specification/5-standard-vocabularies/#repository-types) for the complete default vocabulary with all standard repository types.

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

**Standard Templates**: See [Standard Vocabularies - Participant Roles](/specification/5-standard-vocabularies/#participant-roles) for the complete default vocabulary with all standard participant roles.

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

## Property Vocabularies

Property vocabularies define the custom properties available for each entity type. These properties represent "concluded" or "accepted" values and support flexible, extensible data modeling beyond the standard entity fields.

### Overview

Property vocabularies enable archives to:

- Define custom properties for any entity type
- Control property value types (string, date, integer, boolean)
- Validate entity references for reference-type properties
- Support temporal properties that change over time
- Extend GENEALOGIX for domain-specific needs

### Files

Property vocabulary files are included in the `vocabularies/` directory:

```
vocabularies/
├── person-properties.glx
├── event-properties.glx
├── relationship-properties.glx
└── place-properties.glx
```

### Person Properties Vocabulary

**File**: `vocabularies/person-properties.glx`

**Used By**: [Person Entity](person.md#properties)

**Purpose**: Defines properties that can be set on person entities (birth date, occupation, residence, etc.)

### Standard Properties

GENEALOGIX provides standard person properties:

| Property | Type | Temporal | Description |
|----------|------|----------|-------------|
| `primary_name` | string | Yes | Person's primary or preferred name |
| `given_name` | string | Yes | Given name(s) |
| `family_name` | string | Yes | Family or surname |
| `gender` | string | Yes | Gender identity |
| `born_on` | date | No | Date of birth |
| `born_at` | places | No | Place of birth |
| `died_on` | date | No | Date of death |
| `died_at` | places | No | Place of death |
| `occupation` | string | Yes | Profession or trade |
| `residence` | places | Yes | Place of residence |
| `religion` | string | Yes | Religious affiliation |
| `education` | string | Yes | Educational attainment |
| `ethnicity` | string | Yes | Ethnic background |
| `nationality` | string | Yes | National citizenship |

### Event Properties Vocabulary

**File**: `vocabularies/event-properties.glx`

**Used By**: [Event Entity](event.md#properties)

**Purpose**: Defines properties that can be set on event entities

Event properties are generally less common than person properties, since most event data is structural (type, date, place, participants). Standard properties include:

- `occurred_on` - When the event occurred
- `occurred_at` - Where the event occurred
- `description` - Event description
- `notes` - Additional notes

### Relationship Properties Vocabulary

**File**: `vocabularies/relationship-properties.glx`

**Used By**: [Relationship Entity](relationship.md#properties)

**Purpose**: Defines properties that can be set on relationship entities

Standard properties include:

- `started_on` - When the relationship began
- `ended_on` - When the relationship ended
- `location` - Location of the relationship
- `description` - Relationship description
- `notes` - Additional notes

### Place Properties Vocabulary

**File**: `vocabularies/place-properties.glx`

**Used By**: [Place Entity](place.md#properties)

**Purpose**: Defines properties that can be set on place entities

Standard properties include:

- `existed_from` - When the place came into existence
- `existed_to` - When the place ceased to exist
- `population` - Population count (temporal)
- `description` - Place description
- `notes` - Additional notes

### Property Definition Structure

Each property in a property vocabulary is defined with the following fields:

```yaml
person_properties:
  birth_date:
    label: "Date of Birth"
    description: "Person's date of birth"
    value_type: date
    temporal: false
    custom: false
  
  residence:
    label: "Residence"
    description: "Place where person lived"
    reference_type: places
    temporal: true
    custom: false
```

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label for the property |
| `description` | No | Detailed description of the property |
| `value_type` | No* | Data type: `string`, `date`, `integer`, or `boolean` |
| `reference_type` | No* | Entity type for references: `persons`, `places`, `events`, `relationships`, `sources`, `citations`, `repositories`, `media` |
| `temporal` | No | Whether property can change over time (default: false) |
| `custom` | No | Mark as custom property (non-standard) |

*Exactly one of `value_type` or `reference_type` should be specified (neither defaults to `string`)

### Temporal Properties

Properties marked with `temporal: true` support capturing how values change over time:

```yaml
properties:
  occupation:
    - value: "blacksmith"
      date: "1880"
    - value: "farmer"
      date: "FROM 1885 TO 1920"
  residence:
    - value: "place-leeds"
      date: "1900"
    - value: "place-london"
      date: "FROM 1920 TO 1950"
```

See [Data Types - Temporal Values](../6-data-types.md#temporal-values) for complete documentation.

### Adding Custom Properties

Add custom properties for archive-specific needs:

```yaml
# vocabularies/person-properties.glx
person_properties:
  # Standard properties...
  
  # Custom properties
  militia_service:
    label: "Militia Service"
    description: "Service in local militia"
    value_type: string
    temporal: true
    custom: true
  
  land_holdings:
    label: "Land Holdings"
    description: "Land owned by person"
    reference_type: places
    temporal: true
    custom: true
```

### Context-Aware Validation

Assertions use property vocabularies for context-aware claim validation:

```yaml
assertions:
  assertion-john-birth:
    subject: person-john
    claim: born_on  # Validated against person_properties
    value: "1850-01-15"
    citations: [citation-birth]
    confidence: high
```

The validator:
1. Determines the subject's entity type (person, event, relationship, or place)
2. Looks up the appropriate property vocabulary for that type
3. Validates the `claim` against the vocabulary
4. **Emits warnings for unknown claims** (allows flexibility for emerging properties)
5. Validates the value according to the property's `value_type` or `reference_type`
6. **Emits errors for broken references** when a property is defined with `reference_type` but the referenced entity doesn't exist

---

## Vocabulary Validation

The `glx validate` command performs comprehensive validation with different severity levels:

### Validation Errors (Hard Failures)

The following issues cause validation to fail:

1. **Missing vocabulary types**: All types used in entities must be defined in vocabularies
   - Event types (`event_types`)
   - Relationship types (`relationship_types`)
   - Place types (`place_types`)
   - Source types (`source_types`)
   - Repository types (`repository_types`)
   - Media types (`media_types`)
   - Participant roles (`participant_roles`)
   - Quality ratings (`quality_ratings`)
   - Confidence levels (`confidence_levels`)

2. **Broken entity references**: All entity references must point to existing entities
   - Person references
   - Event references
   - Place references
   - Source references
   - Citation references
   - Repository references
   - Media references
   - Relationship references

3. **Broken property references**: Properties defined with `reference_type` must reference existing entities
   ```yaml
   # Error: place-nonexistent doesn't exist
   persons:
     person-john:
       properties:
         born_at: place-nonexistent  # ERROR if born_at has reference_type: places
   ```

4. **Structural validation**: Files must follow proper YAML/JSON structure and schema

### Validation Warnings (Soft Failures)

The following issues generate warnings but don't fail validation:

1. **Unknown property definitions**: Properties used but not defined in property vocabularies
   ```yaml
   # Warning: custom_field not in person_properties vocabulary
   persons:
     person-john:
       properties:
         custom_field: "some value"  # WARNING: unknown property
   ```

2. **Unknown assertion claims**: Claims used but not defined in property vocabularies
   ```yaml
   # Warning: custom_claim not in person_properties vocabulary
   assertions:
     assertion-custom:
       subject: person-john
       claim: custom_claim  # WARNING: unknown claim
       value: "some value"
   ```

Warnings allow flexibility for emerging properties and rapid data entry while still notifying researchers of potential issues.

### Example Validation Output

```bash
$ glx validate

✓ vocabularies/event-types.glx
✓ vocabularies/relationship-types.glx
✓ events/event-birth.glx
  - event type 'birth' found in vocabulary
✓ relationships/rel-marriage.glx
  - relationship type 'marriage' found in vocabulary

⚠ persons/person-john.glx
  - WARNING: property 'custom_field' not defined in person_properties vocabulary

❌ events/event-custom.glx
  - ERROR: event type 'unknown-type' not found in vocabularies/event-types.glx
  - ERROR: place reference 'place-nonexistent' not found
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

- **[Standard Vocabularies](/specification/5-standard-vocabularies/)** - Complete default vocabulary files with all standard types
- [Core Concepts - Repository-Owned Vocabularies](../2-core-concepts.md#repository-owned-vocabularies)
- [Archive Organization](../3-archive-organization.md) - Where vocabulary files are stored
- [Event Entity](event.md) - Event types vocabulary
- [Relationship Entity](relationship.md) - Relationship types vocabulary
- [Place Entity](place.md) - Place types vocabulary
- [Source Entity](source.md) - Source types vocabulary
- [Media Entity](media.md) - Media types vocabulary
- [Citation Entity](citation.md) - Quality ratings usage

---

