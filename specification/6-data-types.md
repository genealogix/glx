# Data Types

This document defines the fundamental data types used throughout the GENEALOGIX specification.

## Primitive Types

### String
A sequence of Unicode characters. Strings are the default type when no specific type is specified in property definitions.

**Example:**
```yaml
primary_name: "John Smith"
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
custom: true
```

### Date
A calendar date or fuzzy date specification. GENEALOGIX uses the FamilySearch Normalized Date Format (with the exclusion of the stillborn modifier) for standardized date representation.

## Date Format Standard

GENEALOGIX uses the **FamilySearch Normalized Date Format** for all date representations. This format supports both precise dates and fuzzy/approximate dates commonly encountered in genealogical research.

### Format Specification

The format consists of the following elements:

- **Simple Date:** `YYYY`, `YYYY-MM`, or `YYYY-MM-DD`
  - `2020` - Year only
  - `2020-03` - Year and month
  - `2020-03-15` - Year, month, and day

- **Approximate Dates:** Prefixed with a modifier
  - `ABT 2020` - About (approximate)
  - `BEF 2020` - Before
  - `AFT 2020` - After
  - `BET 2020 AND 2025` - Between two dates
  - `CAL 2020` - Calculated

- **Date Ranges:**
  - `FROM 2020 TO 2025` - Range of dates
  - `FROM 2020` - Start date with no end

- **Interpreted Dates:**
  - `INT 1999-03-15 (March 15, 1999)` - Interpreted/translated date

### Examples

```yaml
# Precise dates
born_on: "2020-03-15"

# Approximate dates
born_on: "ABT 1850"
death_year: "BEF 1920"

# Date ranges
residence_dates:
  - value: "Leeds"
    date: "FROM 1900 TO 1950"

# Fuzzy dates
born_on: "BET 1880 AND 1890"
```

For detailed specifications, refer to the [FamilySearch Date Standard](https://www.familysearch.org/en/developer/docs/resources/date-standard).

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
- **quality_ratings** - Quality ratings for citations


