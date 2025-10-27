# Custom Types

GENEALOGIX uses repository-owned controlled vocabularies to define entity types. Each archive contains its own vocabulary definitions in the `vocabularies/` directory, allowing for both standardization and customization.

## Overview

Controlled vocabularies define valid values for:
- Relationship types (marriage, parent-child, adoption, etc.)
- Event types (birth, death, baptism, occupation, etc.)
- Place types (country, city, parish, etc.)
- Repository types (archive, library, church, etc.)
- Participant roles (principal, witness, godparent, etc.)
- Media types (photo, document, audio, etc.)
- Confidence levels (high, medium, low, disputed)
- Quality ratings (0-3, unreliable to primary evidence)

## Vocabulary Files

When you initialize a new archive with `glx init`, standard vocabulary files are created in the `vocabularies/` directory:

```
vocabularies/
  relationship-types.glx
  event-types.glx
  place-types.glx
  repository-types.glx
  participant-roles.glx
  media-types.glx
  confidence-levels.glx
  quality-ratings.glx
```

Each vocabulary file defines a set of types using a simple YAML structure.

## Vocabulary File Structure

### Relationship Types

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
  # ... more standard types
```

### Event Types

```yaml
# vocabularies/event-types.glx
event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
    gedcom: "BIRT"
  baptism:
    label: "Baptism"
    description: "Religious baptism ceremony"
    category: "religious"
    gedcom: "BAPM"
  # ... more standard types
```

### Place Types

```yaml
# vocabularies/place-types.glx
place_types:
  country:
    label: "Country"
    description: "Nation state or country"
    category: "administrative"
  parish:
    label: "Parish"
    description: "Church parish or ecclesiastical division"
    category: "religious"
  # ... more standard types
```

## Adding Custom Types

To add custom types to your archive, edit the appropriate vocabulary file and add your type definition:

### Example: Custom Relationship Type

```yaml
# vocabularies/relationship-types.glx
relationship_types:
  # ... standard types ...
  
  # Custom types
  blood-brother:
    label: "Blood Brother"
    description: "Non-biological brotherhood bond through ceremony"
    custom: true
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor relationship"
    custom: true
```

### Example: Custom Event Type

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
    custom: true
```

### Example: Custom Place Type

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

## Vocabulary Entry Fields

Each vocabulary entry can include:

- **label** (required): Human-readable display name
- **description** (optional): Detailed explanation of the type
- **category** (optional): Grouping category (for event types, place types)
- **gedcom** (optional): Corresponding GEDCOM tag for interoperability
- **mime_type** (optional): MIME type for media types
- **custom** (optional): Boolean flag indicating this is a custom type

## Using Custom Types

Once defined in your vocabulary files, custom types can be used throughout your archive:

```yaml
# relationships/rel-blood-brothers.glx
relationships:
  rel-john-james-blood:
    version: "1.0"
    type: blood-brother  # Custom type from vocabulary
    persons:
      - person-john-smith
      - person-james-brown
    start_event: event-ceremony-1845
```

```yaml
# events/event-apprenticeship.glx
events:
  event-john-apprentice:
    version: "1.0"
    type: apprenticeship  # Custom type from vocabulary
    date: "1850-03-15"
    participants:
      - person: person-john-smith
        role: principal
      - person: person-master-carpenter
        role: master
```

## Validation

The `glx validate` command checks that all entity types used in your archive are defined in the vocabulary files:

```bash
$ glx validate
✓ relationships/rel-blood-brothers.glx
✗ relationships/rel-unknown.glx
  - relationships[rel-xyz]: type 'unknown-type' not found in relationship_types vocabulary
```

## Best Practices

### 1. Document Custom Types

Always include clear descriptions for custom types to help collaborators understand their meaning and usage.

### 2. Use Consistent Naming

Follow the naming convention of standard types:
- Use lowercase with hyphens: `blood-brother`, `land-grant`
- Be descriptive but concise
- Avoid abbreviations unless widely understood

### 3. Mark Custom Types

Set `custom: true` for types you add to distinguish them from standard types:

```yaml
apprenticeship:
  label: "Apprenticeship"
  description: "Beginning of apprenticeship training"
  category: "occupation"
  custom: true
```

### 4. Consider GEDCOM Mapping

If your custom type has a GEDCOM equivalent, include it for better interoperability:

```yaml
blood-brother:
  label: "Blood Brother"
  description: "Non-biological brotherhood bond"
  gedcom: "ASSO"  # Generic association in GEDCOM
  custom: true
```

### 5. Group Related Types

Use categories to organize related types:

```yaml
event_types:
  apprenticeship:
    label: "Apprenticeship"
    category: "occupation"
    custom: true
  journeyman:
    label: "Journeyman"
    category: "occupation"
    custom: true
  master-craftsman:
    label: "Master Craftsman"
    category: "occupation"
    custom: true
```

## Collaboration

When collaborating on an archive:

1. **Review vocabularies first**: Check `vocabularies/` to understand what types are available
2. **Discuss new types**: Before adding custom types, discuss with collaborators
3. **Version control**: Commit vocabulary changes separately with clear descriptions
4. **Document decisions**: Add comments in vocabulary files explaining why custom types were added

## Migration from Other Formats

When importing data from other genealogy formats:

1. **Map standard types**: Most GEDCOM types have GENEALOGIX equivalents
2. **Define custom types**: For unmapped types, create custom vocabulary entries
3. **Preserve original codes**: Use the `gedcom` field to maintain traceability

Example:

```yaml
# Mapping GEDCOM _MILT (military service) to custom type
event_types:
  military-service:
    label: "Military Service"
    description: "Period of military service"
    category: "occupation"
    gedcom: "_MILT"
    custom: true
```

## Schema Validation

Vocabulary files are validated against their JSON schemas in `specification/schema/v1/vocabularies/`. The schemas ensure:

- Required fields are present
- Type keys follow naming conventions (alphanumeric with hyphens, 1-64 characters)
- Structure is consistent

## Why Repository-Owned Vocabularies?

GENEALOGIX uses repository-owned vocabularies rather than a centralized registry because:

1. **Flexibility**: Each archive can define types specific to its research context
2. **Autonomy**: No dependency on external services or registries
3. **Versioning**: Vocabulary changes are tracked with the archive in version control
4. **Offline work**: No internet connection required for validation
5. **Collaboration**: Teams can discuss and agree on types within their repository
6. **Standards + Custom**: Provides standard types while allowing customization

This approach balances standardization (common types work everywhere) with flexibility (archives can extend as needed).
