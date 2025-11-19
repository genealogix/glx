# GEDCOM Import Implementation

**Purpose**: Developer documentation for the GLX GEDCOM import functionality
**Status**: Production-ready (v0.0.0-beta.2)
**Location**: `glx/lib/gedcom_*.go`

---

## Overview

The GEDCOM import system converts GEDCOM 5.5.1 and GEDCOM 7.0 files into GLX format. It supports comprehensive mapping of individuals, families, events, sources, notes, and places into the GLX entity model.

**Key Features**:
- Two-pass conversion (entities first, then families)
- Evidence chain mapping (SOUR → Citations → Assertions)
- Place hierarchy building
- Support for both GEDCOM 5.5.1 and 7.0
- Extensive gap analysis and coverage

---

## Architecture

### Core Files

| File | Purpose | Lines |
|------|---------|-------|
| `gedcom_converter.go` | Main conversion orchestrator | ~500 |
| `gedcom_individual.go` | Individual/person conversion | ~800 |
| `gedcom_family.go` | Family/relationship conversion | ~600 |
| `gedcom_source.go` | Source/citation conversion | ~400 |
| `gedcom_note.go` | Note conversion | ~200 |
| `gedcom_place.go` | Place hierarchy building | ~300 |
| `gedcom_test.go` | Integration tests | ~500 |

### Conversion Flow

```
GEDCOM File
    ↓
ParseGEDCOM()
    ↓
ConversionContext (tracking entities, errors)
    ↓
Pass 1: Convert Individuals, Sources, Notes, Places
    ↓
Pass 2: Convert Families (requires persons)
    ↓
Pass 3: Create Parent-Child Relationships (with PEDI types)
    ↓
GLXFile (complete archive)
```

**Note**: Pass 3 was added in v0.0.0-beta.2 to support PEDI (pedigree linkage) types, which distinguish biological, adoptive, and foster parent-child relationships.

---

## Entity ID Generation

### Current Implementation (Incremental)

The current implementation uses incremental counters:

```go
func (ctx *ConversionContext) generatePersonID(gedcomID string) string {
    ctx.PersonCounter++
    return fmt.Sprintf("person-%03d", ctx.PersonCounter)
}
```

**Format**: `person-001`, `event-042`, `relationship-017`

**Pros**:
- Deterministic ordering
- Human-readable
- Sequential numbering

**Cons**:
- Stateful (requires tracking counters)
- Not suitable for multi-file serialization

### Future Implementation (Random IDs)

For multi-file serialization, random IDs will be used:

**Format**: `person-a3f8d2c1.glx`, `event-b9e4f7a2.glx`

**Implementation**: See `glx/lib/id_generator.go`

**Benefits**:
- Stateless generation
- No collision tracking needed
- Safe for distributed/parallel processing
- Suitable for file-per-entity storage

**Migration Path**:
1. Current import uses incremental IDs (GEDCOM → GLX single-file)
2. Serializer converts to random IDs when writing multi-file
3. `_id` field preserves original entity ID in multi-file archives

---

## Conversion Details

### Person Conversion

**GEDCOM Tags Mapped**:
- `INDI` → `Person` entity
- `NAME` → `properties.given_name`, `properties.family_name`
- `SEX` → `properties.gender`
- `BIRT`, `DEAT`, `CHR`, etc. → `Event` entities
- `NOTE` → `notes` array
- `SOUR` → `Citation` entities + `Assertion` entities

**Property Mapping**:
```go
// GEDCOM NAME structure
0 @I1@ INDI
1 NAME John /Smith/
  → properties.given_name: "John"
  → properties.family_name: "Smith"

// GEDCOM SEX
1 SEX M
  → properties.gender: "male"
```

**Evidence Chain Construction**:
```go
// GEDCOM source citation
1 BIRT
2 DATE 1850
2 SOUR @S1@
3 PAGE Page 42

// Converts to:
- Event: event-001 (type: birth, date: "1850")
- Citation: citation-001 (source: source-001, page: "Page 42")
- Assertion: assertion-001 (subject: person-001, claim: "born_on", value: "1850")
```

### Family Conversion

**GEDCOM Tags Mapped**:
- `FAM` → `Relationship` entities (marriage, parent-child)
- `HUSB` / `WIFE` → Marriage relationship participants
- `CHIL` → Parent-child relationships
- `MARR`, `DIV`, `ANUL`, etc. → Family event conversion
- `SOUR` → Evidence chains for relationships

**Relationship Types**:
- `marriage`: HUSB + WIFE in FAM record
- `biological-parent-child`: CHIL with `PEDI birth` in INDI record
- `adoptive-parent-child`: CHIL with `PEDI adopted` in INDI record
- `foster-parent-child`: CHIL with `PEDI foster` in INDI record
- `parent-child`: CHIL without PEDI or PEDI unknown/sealed

**PEDI (Pedigree Linkage) Support**:

The PEDI tag in GEDCOM 5.5.1 specifies the type of parent-child relationship:

```go
// GEDCOM Individual with PEDI
0 @I3@ INDI
1 FAMC @F1@
2 PEDI birth    // or: adopted, foster, sealed, unknown

// Maps to relationship type:
PEDI birth   → biological-parent-child
PEDI adopted → adoptive-parent-child
PEDI foster  → foster-parent-child
PEDI unknown → parent-child
(no PEDI)    → parent-child
```

**Example**:
```go
// GEDCOM
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE 1875

0 @I3@ INDI
1 FAMC @F1@
2 PEDI birth

// Converts to:
- Relationship: rel-001 (type: marriage, participants: [person-001, person-002])
- Relationship: rel-002 (type: biological-parent-child, participants: [person-001, person-003])
- Relationship: rel-003 (type: biological-parent-child, participants: [person-002, person-003])
- Event: event-042 (type: marriage, date: "1875", participants: [person-001, person-002])
```

### Place Hierarchy

GEDCOM places are flat strings with comma-separated components:
```
"Springfield, Sangamon County, Illinois, USA"
```

GLX requires hierarchical place entities:
```yaml
places:
  place-usa:
    name: "United States of America"
    type: country

  place-illinois:
    name: "Illinois"
    type: state
    parent: place-usa

  place-sangamon:
    name: "Sangamon County"
    type: county
    parent: place-illinois

  place-springfield:
    name: "Springfield"
    type: city
    parent: place-sangamon
```

**Implementation**: `buildPlaceHierarchy()` in `gedcom_place.go`

**ADDR Subfield Support**:

When the PLAC field is missing, GLX can build a place hierarchy from ADDR subfields:

```go
// GEDCOM event with ADDR but no PLAC
1 BIRT
2 DATE 24 FEB 1875
2 ADDR
3 ADR2 Olnhausen
3 STAE Baden-Wuerrtemberg
3 CTRY Germany

// Converts to:
- Place hierarchy: Germany > Baden-Wuerrtemberg > Olnhausen
- Event property: address = "Olnhausen, Baden-Wuerrtemberg, Germany"
```

**Supported ADDR subfields**:
- `CITY` or `ADR2` → City/locality (most specific)
- `STAE` → State/province
- `CTRY` → Country (most general)
- `ADR1`, `ADR3`, `POST` → Included in concatenated address property

**Implementation**: `buildPlaceHierarchyFromAddress()` in `gedcom_individual.go`

### Source and Citation Conversion

**GEDCOM Source Structure**:
```
0 @S1@ SOUR
1 TITL "1850 US Census"
1 PUBL "National Archives"
1 REPO @R1@
```

**GLX Mapping**:
```yaml
sources:
  source-001:
    title: "1850 US Census"
    publication_info: "National Archives"
    repository: repository-001

citations:
  citation-001:
    source: source-001
    page: "Page 42"
    quality: 2

assertions:
  assertion-001:
    subject: person-001
    claim: "born_on"
    value: "1850-01-15"
    citations: [citation-001]
```

### Note Conversion

**GEDCOM 5.5.1** (shared notes):
```
0 @N1@ NOTE This is a note
1 CONT with continuation
```

**GEDCOM 7.0** (shared notes):
```
0 @N1@ SNOTE This is a note
1 CONT with continuation
```

**GLX Mapping**:
Both map to entity `notes` arrays:
```yaml
persons:
  person-001:
    notes:
      - "This is a note with continuation"
```

---

## Version Differences

### GEDCOM 5.5.1 vs 7.0

| Feature | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Support |
|---------|--------------|------------|-------------|
| **Void pointer** | `@VOID@` | `@VOID@` | ✅ Both |
| **Shared notes** | `NOTE` (level 0) | `SNOTE` | ✅ Both |
| **External IDs** | No | `EXID` | ✅ Mapped to Properties |
| **Coordinates** | `MAP/LATI/LONG` | `MAP/LATI/LONG` | ✅ Place coordinates |
| **GEDCOM version** | `HEAD.GEDC.VERS` | `HEAD.GEDC.VERS` | ✅ Detected automatically |

---

## Testing

### Test Files

**Location**: `glx/testdata/gedcom/`

**Key Test Files**:
- `shakespeare.ged` - GEDCOM 5.5.1 comprehensive test (31 persons, 77 events, 49 relationships)
- `minimal-70.ged` - GEDCOM 7.0 minimal test
- 180+ additional test files from GEDCOM validation suite

### Test Coverage

**Gap Analysis Results** (v0.0.0-beta.2):
- ✅ 100% critical features
- ✅ 94% high-priority features
- ⚠️ 76% medium-priority features
- ⚠️ 45% low-priority features

**Critical Features** (all implemented):
- Individual names, sex, birth, death
- Family structure (HUSB, WIFE, CHIL)
- Basic source citations
- Place names
- Notes

**High-Priority Features** (94% implemented):
- All lifecycle events (birth, death, marriage, etc.)
- Religious events (baptism, bar mitzvah, etc.)
- Occupation, residence, education attributes
- Citation page numbers and quality
- Shared notes (GEDCOM 5.5.1 and 7.0)

### Running Tests

```bash
# Run all GEDCOM tests
go test -v ./glx/lib -run GEDCOM

# Run specific test
go test -v ./glx/lib -run TestImportShakespeare

# Run with coverage
go test -cover ./glx/lib
```

---

## Error Handling

### Error Accumulation

Errors are accumulated in `ConversionContext.Errors` to allow partial conversion:

```go
ctx.AddError(fmt.Sprintf("Unknown event tag: %s", tag))
```

**Philosophy**: Convert as much as possible, report all errors at end.

### Error Types

1. **Critical Errors**: Stop conversion (e.g., malformed GEDCOM)
2. **Warning Errors**: Continue conversion, report issue (e.g., unknown tag)
3. **Info Messages**: Logged for debugging (e.g., skipped optional field)

### Malformed Line Recovery

The parser includes resilience for common real-world GEDCOM issues:

**Malformed Continuation Lines** (MyHeritage HTML notes bug):
- **Problem**: Some exports (esp. MyHeritage 8.0.0.8367) produce NOTE fields with HTML that's missing CONT/CONC prefixes
- **Example**: Line starts with `<div>` instead of `2 CONT <div>`
- **Recovery**: If a line fails to parse and doesn't start with a digit, it's treated as a continuation of the previous line
- **Impact**: Allows importing otherwise-valid genealogy data with formatting issues
- **Test Case**: `queen.ged` (4,683 persons, line 15903 missing CONT prefix)

---

## Future Enhancements

### Property Vocabularies

Map GEDCOM-specific fields to property vocabularies:

```yaml
person_properties:
  gedcom_rin:
    label: "GEDCOM RIN"
    description: "GEDCOM record identifier number"
    value_type: string
    custom: true
```

### Export to GEDCOM

Reverse conversion: GLX → GEDCOM

**Challenges**:
- GLX assertions → GEDCOM SOUR structure
- GLX place hierarchy → GEDCOM flat place strings
- GLX vocabularies → GEDCOM standard tags

### Improved Mapping

- Better name parsing (honorifics, suffixes, nicknames)
- Advanced source citation mapping
- Custom GEDCOM tags (`_TAG`) to GLX properties
- GEDCOM Extensions (MyHeritage, FamilySearch, etc.)

---

## Best Practices

### When Modifying Import Code

1. **Run tests**: Ensure Shakespeare test still passes
2. **Update gap analysis**: Document any new features implemented
3. **Add test cases**: Add GEDCOM examples for new features
4. **Check both versions**: Test with GEDCOM 5.5.1 and 7.0 files
5. **Preserve entity IDs**: Maintain ID mapping for cross-references

### Adding New Tag Support

1. Find appropriate converter file (e.g., `gedcom_individual.go`)
2. Add tag handling in switch statement
3. Extract data and map to GLX entity
4. Add test case in `gedcom_test.go`
5. Update this documentation

### Debugging Import Issues

1. Enable verbose logging: `ctx.Logger.LogInfo(...)`
2. Check `ConversionContext` entity maps
3. Run specific test: `go test -v -run TestImportShakespeare`
4. Review error accumulation in `ctx.Errors`
5. Check GEDCOM spec: `docs/gedcom-spec/`

---

## References

### Specifications

- **GEDCOM 5.5.1**: `docs/gedcom-spec/gedcom-5-5-1.pdf`
- **GEDCOM 7.0**: https://gedcom.io/specifications/
- **GLX Specification**: `specification/`

### Related Documentation

- **User Guide**: `docs/guides/migration-from-gedcom.md`
- **Entity Types**: `specification/4-entity-types/`
- **Vocabularies**: `specification/5-standard-vocabularies/`
- **ID Generation**: `docs/development/gedcom-import.md` (this file)

---

## Changelog

**v0.0.0-beta.2** (2025-11-19):
- ✅ Full GEDCOM 5.5.1 support
- ✅ Full GEDCOM 7.0 support
- ✅ Evidence chain mapping
- ✅ Place hierarchy building
- ✅ **PEDI (pedigree linkage) support**: Biological, adoptive, foster parent-child relationships
- ✅ **ADDR subfield extraction**: Full address preservation and place hierarchy fallback
- ✅ Three-pass conversion for accurate relationship typing
- ✅ 31 persons, 77 events, 49 relationships imported (Shakespeare test)
- ✅ 948 persons with PEDI tags tested (Bullinger family)
- ✅ 514 addresses preserved (Bullinger family)
- ✅ Gap analysis: 100% critical, 94% high-priority coverage
- ✅ **6 issues resolved**: Date qualifiers, date quoting, TITL, date ranges, PEDI, ADDR subfields

---

Last Updated: 2025-11-19
