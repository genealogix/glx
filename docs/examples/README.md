---
title: GENEALOGIX Examples
description: Complete working examples demonstrating GENEALOGIX features and use cases
layout: doc
---

# GENEALOGIX Examples

[![Examples Status](https://img.shields.io/badge/examples-complete-green.svg)](./)
[![Validation](https://img.shields.io/badge/validation-passing-brightgreen.svg)](../../glx/tests/)
[![Coverage](https://img.shields.io/badge/coverage-9%2F9%20entities-blue.svg)](#entity-types-demonstrated)

This directory contains complete, working GENEALOGIX archives demonstrating various features and use cases. Each example is designed to teach specific concepts and provide practical templates for real genealogy research.

## 🎯 Learning Path

### For Beginners
1. **[Complete Family](complete-family/)** ⭐ - **Start Here**
   - All 9 entity types demonstrated
   - Complete evidence chains
   - Real-world family structure
   - Best practices shown

2. **[Basic Family](basic-family/)** - Simple nuclear family
   - Core relationships only
   - Minimal evidence requirements
   - Easy to understand

3. **[Minimal](minimal/)** - Smallest valid archives
   - Required fields only
   - Foundation concepts
   - Quick validation testing


## Quick Links

- [Minimal](minimal/) - Smallest valid archive (essentials only)
- [Basic Family](basic-family/) - Simple nuclear family with basic relationships
- [Complete Family](complete-family/) - **ALL entity types demonstrated** ⭐ START HERE
  - Shows hierarchical places, events, citations, repositories
  - Complete evidence chain from source to assertions
  - Quality ratings and structured locators

## GEDCOM vs GENEALOGIX Comparison

### Evidence Model Comparison

**GEDCOM Approach:**
```
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Leeds, Yorkshire, England
2 SOUR @S1@
3 QUAY 2
3 PAGE Page 23
0 @S1@ SOUR
1 TITL Parish Register
1 REPO @R1@
```

**GENEALOGIX Approach:**
```yaml
# repositories/repository-leeds.glx
repositories:
  repository-a1b2c3d4:
    name: Leeds Library Local Studies
    email: local.studies@leeds.gov.uk

# sources/parish-register.glx
sources:
  source-12345678:
    title: St. Paul's Parish Register
    repository: repository-a1b2c3d4

# citations/birth-entry.glx
citations:
  citation-abc12345:
    source: source-12345678
    locator: "Entry 145, page 23"
    transcription: "John, son of Thomas Smith, born January 15, 1850"

# assertions/birth-assertion.glx
assertions:
  assertion-1a2b3c4d:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations: [citation-abc12345]
    confidence: high  # Express certainty at assertion level
```

### Collaboration Comparison

**GEDCOM Collaboration:**
- Email file attachments
- Manual merge conflicts
- No change tracking
- Version confusion

**GENEALOGIX Collaboration:**
- Git pull requests and reviews
- Automatic conflict resolution
- Complete change history
- Branch-based research

## Entity Types Demonstrated

### Complete Family Example ⭐ Recommended Starting Point

The **complete-family** example demonstrates all 9 GENEALOGIX entity types:

| Entity Type | Files | Features |
|-------------|-------|----------|
| Person | 3 files | Individual family members |
| Relationship | 2 files | Marriage and parent-child |
| Event | 3 files | Births, marriages, occupations |
| Place | 3 files | Hierarchical locations (England → Yorkshire → Leeds) |
| Source | 2 files | Parish registers, census records |
| Citation | 4 files | Evidence references with locators and transcriptions |
| Repository | 2 files | Archive institutions and access info |
| Assertion | 6 files | Conclusions backed by evidence |
| Media | - | Linked photos/documents |

### Other Examples

- **Minimal**: Foundation - only required fields
- **Basic Family**: Real-world small family tree

## Running Examples

Each example includes documentation:

```bash
cd examples/complete-family
cat README.md  # Learn about this example
```

Validation is centralized in the test suite:

```bash
# Run centralized test suite for all entity types
cd glx/tests
./run-tests.sh

# Or validate examples manually with glx
glx validate examples/complete-family/
```

## Understanding the Complete Family Example

### Data Structure

```
Complete Family Example (John Smith & family, 1850-1920)
├── persons/
│   ├── person-john-smith.glx        (1850-1920)
│   ├── person-mary-brown.glx        (1852-1930)
│   └── person-jane-smith.glx        (daughter, 1875-1955)
├── relationships/
│   ├── rel-john-mary-marriage.glx   (married 1875)
│   └── rel-john-jane-parent.glx     (father-daughter)
├── events/
│   ├── event-john-birth.glx         (Jan 15, 1850, Leeds)
│   ├── event-marriage.glx           (May 10, 1875, Leeds)
│   └── event-occupations.glx        (blacksmith, dressmaker)
├── places/
│   ├── place-england.glx            (root)
│   ├── place-yorkshire.glx          (parent: England)
│   └── place-leeds.glx              (parent: Yorkshire)
├── sources/
│   ├── source-parish-leeds.glx      (St Paul's registers)
│   └── source-census-1851.glx       (1851 Census)
├── citations/
│   ├── citation-birth-register.glx  (with transcription)
│   └── citation-census.glx          (with locator)
├── repositories/
│   ├── repository-leeds-library.glx (Local studies)
│   └── repository-tna.glx           (The National Archives)
└── assertions/
    ├── assertion-birth-date.glx     (supported by citation)
    ├── assertion-birth-place.glx    (supported by citation)
    └── assertion-occupation.glx     (supported by citation)
```

### Evidence Chain Example

How evidence flows from source to conclusion:

1. **Repository**: Leeds Library Local Studies
2. **Source**: St Paul's Church Parish Registers  
3. **Citation**: "Birth entry 145, page 23" with transcription
4. **Assertion**: "John Smith born January 15, 1850" (confidence: high)

## Best Practices Demonstrated

✅ **Hierarchical Places**  
- England (country) → Yorkshire (county) → Leeds (city)
- Alternative names (West Riding for Yorkshire)
- WGS84 coordinates for all places

✅ **Complete Event Information**
- Dates with fuzzy date support
- Place references
- Multiple participants with defined roles
- Event descriptions

✅ **Evidence Documentation**  
- Assertion confidence levels (high, medium, low, disputed)
- Structured locators (film numbers, page ranges, URLs)
- Text transcriptions from sources
- Complete evidence chains

✅ **Repository Information**
- Contact details and hours
- Material types held
- Website and access information
- Archive call numbers

## Validation

All examples pass validation:
```bash
glx validate  # Validates current directory
```

Expected output for complete-family:
```
✓ citations/citation-birth-register.glx
✓ events/event-john-birth.glx
✓ events/event-marriage.glx
✓ places/place-england.glx
✓ places/place-leeds.glx
✓ places/place-yorkshire.glx
✓ repositories/repository-leeds-library.glx
Validated 7 file(s)
```

## Contributing Examples

New examples should be:

- **Complete**: Include all necessary files for the use case
- **Valid**: Pass `glx validate` without errors
- **Documented**: README explaining what it demonstrates
- **Educational**: Teach important GENEALOGIX concepts

### Adding a New Example

1. Create new directory: `docs/examples/my-example/`
2. Add subdirectories: `persons/`, `events/`, `places/`, etc.
3. Create example .glx files for key entities
4. Write detailed README.md
5. Add conformance tests to `glx/tests/` for new entity types
6. Update this README with link and description

## CLI Tool Support

The genealogix CLI (`glx`) supports:

```bash
glx init                    # Initialize new repository with all directories
glx validate [path]         # Validate .glx files in path
glx validate                # Validate current directory
glx validate persons/       # Validate specific directory
glx validate persons/*.glx  # Validate specific files
glx validate examples/      # Validate any example
```

All examples work with the latest CLI tool, which recognizes all entity type directories:
- `persons/`, `relationships/`, `events/`, `places/`
- `sources/`, `citations/`, `repositories/`, `assertions/`, `media/`

## Next Steps

1. **Start with Complete Family**: Run the test script, explore files
2. **Try the CLI**: Run `glx validate` on any example
3. **Initialize Your Own**: `glx init` to start your family archive
4. **Read Specification**: See `specification/` for detailed entity formats
5. **Share Your Example**: Contribute back with new use cases!

## References

- [GENEALOGIX Specification](../../specification/)
- [CLI Tool](../../glx/)
- [JSON Schemas](../../specification/schema/v1/)
- [Contributing Guide](../../CONTRIBUTING.md)


