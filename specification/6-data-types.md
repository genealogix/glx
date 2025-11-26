---
title: Data Types
description: Fundamental data types used in GENEALOGIX - primitives, dates, temporal values, and complex structures
layout: doc
---

# Data Types

This document defines the fundamental data types used throughout the GENEALOGIX specification.

## Primitive Types

### String
A sequence of Unicode characters. Strings are the default type when no specific type is specified in property definitions.

**Example:**
```yaml
name:
  value: "John Smith"
  fields:
    given: "John"
    surname: "Smith"
occupation: "blacksmith"
```

### Integer
A whole number (positive, negative, or zero). Used for numeric values like population counts.

**Example:**
```yaml
population: 5000
```

### Boolean
A true/false value.

**Example:**
```yaml
living: true
verified: false
```

### Date
A calendar date or fuzzy date specification. GENEALOGIX uses ISO 8601-style dates (YYYY-MM-DD) combined with FamilySearch-inspired keywords for fuzzy dates.

## Date Format Standard

GENEALOGIX uses a **hybrid date format** combining:
- **ISO 8601-style dates** for precise calendar dates
- **FamilySearch-inspired keywords** for approximate, ranged, and calculated dates

This format supports both precise dates and fuzzy/approximate dates commonly encountered in genealogical research.

### Format Specification

**Simple Dates (ISO 8601-style):**
- `YYYY` - Year only (4 digits required, e.g., `1850`, `2020`, `0047`)
- `YYYY-MM` - Year and month (e.g., `1850-03`, `2020-12`)
- `YYYY-MM-DD` - Full date (e.g., `1850-03-15`, `2020-12-31`)

**Keyword Modifiers (FamilySearch-inspired):**
- **Approximate Dates:**
  - `ABT YYYY` - About/approximately (e.g., `ABT 1850`)
  - `BEF YYYY` - Before (e.g., `BEF 1920`)
  - `AFT YYYY` - After (e.g., `AFT 1880`)
  - `CAL YYYY` - Calculated (e.g., `CAL 1850`)

- **Date Ranges:**
  - `BET YYYY AND YYYY` - Between two dates (e.g., `BET 1880 AND 1890`)
  - `FROM YYYY TO YYYY` - Range with start and end (e.g., `FROM 1900 TO 1950`)
  - `FROM YYYY` - Open-ended range from a start date (e.g., `FROM 1900`)

- **Interpreted Dates:**
  - `INT YYYY-MM-DD (original text)` - Interpreted from original source (e.g., `INT 1850-03-15 (March 15th, 1850)`)

### Important Notes

1. **Year Format:** Years must be exactly 4 digits. Pad with zeros for years before 1000 CE (e.g., `0047` for year 47, `0800` for year 800).

2. **Keywords vs Full Format:** GENEALOGIX uses **keywords inspired by** the FamilySearch Normalized Date Format, but the underlying date representation uses ISO 8601-style dates (YYYY-MM-DD), not the full FamilySearch format.

3. **Keyword Combinations:** Keywords can be combined with any simple date format (e.g., `ABT 1850`, `ABT 1850-03`, `ABT 1850-03-15`).

### Examples

```yaml
# Precise dates
born_on: "1850-03-15"      # Full date
born_on: "1850-03"          # Year and month
born_on: "1850"             # Year only
born_on: "0047"             # Year 47 AD (zero-padded)

# Approximate dates
born_on: "ABT 1850"         # About 1850
death_year: "BEF 1920"      # Before 1920
married_on: "AFT 1880-06"   # After June 1880

# Date ranges
residence_dates:
  - value: "place-leeds"
    date: "FROM 1900 TO 1950"  # Lived in Leeds 1900-1950
  - value: "place-london"
    date: "FROM 1950"           # Lived in London from 1950 onward

# Fuzzy dates
born_on: "BET 1880 AND 1890"   # Born between 1880 and 1890

# Calculated dates
born_on: "CAL 1850"             # Birth year calculated from other evidence

# Interpreted dates
born_on: "INT 1850-03-15 (15th March 1850)"  # Original text preserved
```

### Validation

GENEALOGIX validates date formats at two levels:
1. **Structure:** Dates must follow the format specifications above
2. **Keywords:** Only the defined keywords (FROM, TO, ABT, BEF, AFT, BET, AND, CAL, INT) are recognized

Invalid date formats will generate validation warnings (not errors), allowing archives with imperfect dates to still load while alerting researchers to potential data quality issues.

## Reference Types

Reference types indicate that a property value is a string identifier that must exist as an entity in the archive. References are validated at runtime against the actual entities in the archive.

### Supported Reference Types

- **persons** - Reference to a person entity
- **places** - Reference to a place entity
- **events** - Reference to an event entity
- **relationships** - Reference to a relationship entity
- **sources** - Reference to a source entity
- **citations** - Reference to a citation entity
- **repositories** - Reference to a repository entity
- **media** - Reference to a media entity

### Example

```yaml
properties:
  born_at: "place-leeds"  # Reference to a place
  residence: "place-london"  # Reference to a place
```

## Temporal Values

Properties marked as `temporal: true` in their property definition can represent values that change over time. Such properties support two formats:

### Single Value (Non-Temporal)
For properties that represent a state at a point in time:

```yaml
properties:
  gender: "male"
  born_on: "1850-01-15"
```

### Temporal History (Multiple Values with Dates)
For properties that capture how a value changed over time, represented as a list of objects:

```yaml
properties:
  residence:
    - value: "place-leeds"
      date: "FROM 1900 TO 1920"
    - value: "place-london"
      date: "FROM 1920 TO 1950"
  occupation:
    - value: "blacksmith"
      date: "1880"
    - value: "farmer"
      date: "FROM 1885 TO 1920"
```

Each entry in a temporal property must include:
- `value` - The actual property value, conforming to the property's `value_type` or `reference_type`
- `date` - Optional FamilySearch normalized date string (if omitted, the entry is undated)

### Validation

When validating temporal properties:
1. If the property definition has `temporal: true`, the value can be either:
   - A single value (primitive or reference)
   - A list of temporal value objects
2. Each temporal value object must have a valid `value` field
3. The `date` field is optional but recommended for temporal properties
4. All values are validated according to the property's `value_type` or `reference_type`

## Vocabulary References

In addition to data type definitions, properties can reference controlled vocabularies:

- **confidence_levels** - Confidence in assertion conclusions
- **participant_roles** - Roles people play in events or relationships
- **event_types** - Standard event types
- **relationship_types** - Standard relationship types
- **place_types** - Standard place types
- **source_types** - Standard source types
- **repository_types** - Standard repository types
- **media_types** - Standard media types


