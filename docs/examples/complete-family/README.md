# Complete Family Example

This example demonstrates all 9 GENEALOGIX entity types with proper cross-references, evidence chains, and descriptive entity IDs.

## Family Structure

**John Smith (person-john-smith-1850)**
- Born: January 15, 1850 in Leeds
- Died: June 20, 1920
- Occupation: Blacksmith
- Married: Mary Brown (May 10, 1875)

**Mary Brown (person-mary-brown-1852)**
- Born: March 20, 1852 in Leeds
- Died: August 15, 1930
- Occupation: Dressmaker
- Married: John Smith (May 10, 1875)

**Jane Smith (person-jane-smith-1876)**
- Born: September 5, 1876 in Leeds
- Died: December 10, 1955
- Parents: John Smith and Mary Brown
- Occupation: Dressmaker

## Entity IDs

This example uses **descriptive IDs** to make the archive more human-readable:
- `person-john-smith-1850` (person with birth year)
- `place-leeds` (descriptive place name)
- `event-birth-john` (event type + person)
- `citation-john-birth` (what it cites)

**Note:** You can use any ID format you prefer. For collaborative projects, random hex IDs (like `person-a1b2c3d4`) are recommended to avoid conflicts.

## File Organization

### Persons (3 files)
- `persons/person-john-smith.glx` - John Smith (b. 1850)
- `persons/person-mary-brown.glx` - Mary Brown (b. 1852)
- `persons/person-jane-smith.glx` - Jane Smith (b. 1876)

### Relationships (2 files)
- `relationships/rel-marriage.glx` - Marriage relationship
- `relationships/rel-parent-child.glx` - Parent-child relationships (2 relationships in one file)

### Events (3 files)
- `events/event-births.glx` - 3 birth events (all in one file)
- `events/event-marriage.glx` - 1 marriage event
- `events/event-occupations.glx` - 2 occupation events

### Places (3 files)
- `places/place-england.glx` - England (country)
- `places/place-yorkshire.glx` - Yorkshire (county, parent: England)
- `places/place-leeds.glx` - Leeds (city, parent: Yorkshire)

### Sources (2 files)
- `sources/source-parish-register.glx` - St. Paul's Parish Register
- `sources/source-census.glx` - 1851 Census

### Citations (3 files)
- `citations/citation-john-birth.glx` - Birth citation (quality 3, primary)
- `citations/citation-marriage.glx` - Marriage citation (quality 3, primary)
- `citations/citation-census.glx` - Census citation (quality 2, secondary)

### Repositories (2 files)
- `repositories/repository-leeds-library.glx` - Leeds Library
- `repositories/repository-national-archives.glx` - The National Archives

### Assertions (3 files)
- `assertions/assertion-john-birth.glx` - Birth date with multiple citations
- `assertions/assertion-john-birthplace.glx` - Birth place
- `assertions/assertion-marriage.glx` - Marriage date

## Evidence Chain Example

**Claim**: John Smith was born on January 15, 1850

**Evidence Trail**:
1. **Repository**: Leeds Library Local Studies (`repository-leeds-library`)
2. **Source**: St. Paul's Parish Register (`source-parish-leeds`)
3. **Citation**: Entry 145, Page 23 (`citation-john-birth`) - Quality 3 (primary)
4. **Assertion**: Birth date = 1850-01-15 (`assertion-john-birth-date`) - High confidence
5. **Corroboration**: 1851 Census (`citation-census-john`) - Quality 2 (secondary)

## File Format

All files use the unified GENEALOGIX format:

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith-1850:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"
      birth_date: "1850-01-15"
```

**Key Points:**
- Entity ID is the map key (`person-john-smith-1850`)
- Entity type plural at top level (`persons`)
- Files can contain multiple entities of the same type (see `events/event-births.glx`)

## Validation

```bash
cd examples/complete-family
glx validate
```

Should validate successfully with:
- ✓ All 21 files pass validation
- ✓ All cross-references valid
- ✓ Evidence chains complete
- ✓ No duplicate entity IDs

## Key Features Demonstrated

✅ **All 9 Entity Types**: Persons, relationships, events, places, sources, citations, repositories, assertions, media  
✅ **Descriptive IDs**: Human-readable entity identifiers  
✅ **Evidence Chains**: Complete provenance from repository to conclusion  
✅ **Quality Ratings**: Primary (3) and secondary (2) evidence  
✅ **Hierarchical Places**: England → Yorkshire → Leeds  
✅ **Cross-References**: All entities properly linked and validated  
✅ **Multi-Generation Family**: Parents and children with relationships  
✅ **Flexible Files**: Some files have one entity, some have multiple  
✅ **Unified Format**: All files use entity type keys at top level
