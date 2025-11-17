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
1. **[Minimal](minimal/)** - Start with the basics
   - Smallest valid archive
   - Required fields only
   - Foundation concepts

2. **[Basic Family](basic-family/)** - Simple nuclear family
   - Core relationships
   - Minimal evidence requirements
   - Easy to understand

3. **[Complete Family](complete-family/)** ⭐ **Recommended**
   - All 9 entity types demonstrated
   - Complete evidence chains
   - Real-world family structure
   - Best practices shown

### Advanced Concepts
4. **[Single-File](single-file/)** - Single-file archives
   - All entities in one file
   - Portable format
   - Simple backup/sharing

5. **[Temporal Properties](temporal-properties/)** - Time-changing values
   - Properties that change over time
   - Occupations, residences, names
   - Date-stamped values

6. **[Participant Assertions](participant-assertions/)** - Event participants
   - Assertion-based participant roles
   - Conflicting evidence about participants
   - Evidence for relationships

## Understanding Evidence Chains

> **Learn More:** For a complete explanation of the evidence chain model, see [Core Concepts: Evidence Hierarchy](../../specification/2-core-concepts.md#evidence-hierarchy).

The examples below demonstrate how GENEALOGIX structures evidence from Repository → Source → Citation → Assertion. See the [complete-family](./complete-family/) example for a working implementation.

### Collaboration with Git
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

## Example Descriptions

| Example | Focus | Entity Types | Use Case |
|---------|-------|--------------|----------|
| **[Minimal](minimal/)** | Essentials | Person | Smallest valid archive |
| **[Basic Family](basic-family/)** | Relationships | Person, Relationship | Nuclear family structure |
| **[Complete Family](complete-family/)** ⭐ | All features | All 9 types | Comprehensive demonstration |
| **[Single-File](single-file/)** | Portability | All in one file | Simple sharing/backup |
| **[Temporal Properties](temporal-properties/)** | Time-changing data | Person with temporal values | Changing occupations/names |
| **[Participant Assertions](participant-assertions/)** | Event participation | Assertion-based roles | Evidence-based participants |

> **Tip:** See each example's README for detailed explanations and file structure.

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


