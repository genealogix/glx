# Complete Family Example

This example demonstrates all 9 GENEALOGIX entity types with proper cross-references and evidence chains.

## Family Structure

**John Smith (person-a1b2c3d4)**
- Born: January 15, 1850 in Leeds
- Died: June 20, 1920
- Occupation: Blacksmith
- Married: Mary Brown (May 10, 1875)

**Mary Brown (person-b2c3d4e5)**
- Born: March 20, 1852 in Leeds
- Died: August 15, 1930
- Occupation: Dressmaker
- Married: John Smith (May 10, 1875)

**Jane Smith (person-c3d4e5f6)**
- Born: September 5, 1876 in Leeds
- Died: December 10, 1955
- Parents: John Smith and Mary Brown
- Occupation: Dressmaker

## Entity Breakdown

### Persons (3 files)
- `person-john-smith.glx` - person-a1b2c3d4
- `person-mary-brown.glx` - person-b2c3d4e5
- `person-jane-smith.glx` - person-c3d4e5f6

### Relationships (2 files)
- `rel-marriage.glx` - rel-12345678 (John & Mary marriage)
- `rel-parent-child.glx` - rel-23456789, rel-34567890 (parent-child relationships)

### Events (3 files)
- `event-births.glx` - 3 birth events
- `event-marriage.glx` - 1 marriage event
- `event-occupations.glx` - 2 occupation events

### Places (3 files)
- `place-england.glx` - place-a1a1a1a1 (country)
- `place-yorkshire.glx` - place-b2b2b2b2 (county, parent: England)
- `place-leeds.glx` - place-d4e5f6a1 (city, parent: Yorkshire)

### Sources (2 files)
- `source-parish-register.glx` - source-12345678
- `source-census.glx` - source-23456789

### Citations (3 files)
- `citation-john-birth.glx` - citation-abc12345 (quality 3, primary)
- `citation-marriage.glx` - citation-def67890 (quality 3, primary)
- `citation-census.glx` - citation-fedcba98 (quality 2, secondary)

### Repositories (2 files)
- `repository-leeds-library.glx` - repository-1a2b3c4d
- `repository-national-archives.glx` - repository-2b3c4d5e

### Assertions (3 files)
- `assertion-john-birth.glx` - Birth date assertion with multiple citations
- `assertion-john-birthplace.glx` - Birth place assertion
- `assertion-marriage.glx` - Marriage date assertion

## Evidence Chain Example

**Claim**: John Smith was born on January 15, 1850

**Evidence Trail**:
1. **Repository**: Leeds Library Local Studies (repository-1a2b3c4d)
2. **Source**: St. Paul's Parish Register (source-12345678)
3. **Citation**: Entry 145, Page 23 (citation-abc12345) - Quality 3 (primary)
4. **Assertion**: Birth date = 1850-01-15 (assertion-1a2b3c4d) - High confidence

## File Format

All files use the unified GENEALOGIX format with entity type keys:

```yaml
# persons/person-john-smith.glx
persons:
  person-a1b2c3d4:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"
```

**Note:** Entity IDs are map keys, NOT `id` fields within entities.

## Validation

```bash
cd examples/complete-family
glx validate
```

Should validate successfully with:
- All required fields present
- All cross-references valid
- Evidence chains complete
- No duplicate entity IDs

## Key Features Demonstrated

✅ **All 9 Entity Types**: Persons, relationships, events, places, sources, citations, repositories, assertions, media  
✅ **Evidence Chains**: Complete provenance from repository to conclusion  
✅ **Quality Ratings**: Primary (3) and secondary (2) evidence  
✅ **Hierarchical Places**: England → Yorkshire → Leeds  
✅ **Cross-References**: All entities properly linked  
✅ **Multi-Generation Family**: Parents and children with relationships  
✅ **Unified Format**: Entity type keys at top level, IDs as map keys
