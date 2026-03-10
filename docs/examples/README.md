---
title: GENEALOGIX Examples
description: Complete working examples demonstrating GENEALOGIX features and use cases
layout: doc
---

# GENEALOGIX Examples

This directory contains complete, working GENEALOGIX archives demonstrating various features and use cases. Each example is designed to teach specific concepts and provide practical templates for real genealogy research.

## Learning Path

### For Beginners
1. **[Minimal](minimal/)** - Start with the basics
   - Smallest valid archive
   - Required fields only

2. **[Basic Family](basic-family/)** - Simple nuclear family
   - Core relationships
   - Easy to understand

3. **[Complete Family](complete-family/)** - Recommended starting point
   - Most entity types demonstrated
   - Complete evidence chains
   - Real-world family structure

### Real-World Scale
4. **[Westeros: A Song of Ice and Fire](westeros/)** - Large-scale archive
   - 790+ persons across 70+ houses
   - Full evidence chains with 1,800+ assertions
   - 200+ custom vocabulary types
   - Demonstrates every GLX feature at scale

### Advanced Concepts
5. **[Single-File](single-file/)** - Single-file archives
   - All entities in one file
   - Portable format

6. **[Assertion Workflow](assertion-workflow/)** - Evidence documentation
   - Direct properties vs assertion-backed properties
   - Complete evidence chain pattern
   - Iterative research workflow

7. **[Temporal Properties](temporal-properties/)** - Time-changing values
   - Properties that change over time
   - Occupations, residences, names
   - Date-stamped values

8. **[Participant Assertions](participant-assertions/)** - Event participants
   - Assertion-based participant roles
   - Evidence for event participation

## Example Descriptions

| Example | Focus | Use Case |
|---------|-------|----------|
| **[Minimal](minimal/)** | Essentials | Smallest valid archive |
| **[Basic Family](basic-family/)** | Relationships | Nuclear family structure |
| **[Complete Family](complete-family/)** | All features | Comprehensive demonstration |
| **[Single-File](single-file/)** | Portability | Simple sharing/backup |
| **[Assertion Workflow](assertion-workflow/)** | Evidence chains | Direct vs assertion-backed properties |
| **[Temporal Properties](temporal-properties/)** | Time-changing data | Changing occupations/names |
| **[Participant Assertions](participant-assertions/)** | Event participation | Evidence-based participants |
| **[Westeros](westeros/)** | Scale & custom vocabularies | 790+ persons, 200+ custom types, full evidence chains |

## Validation

```bash
glx validate examples/complete-family/
# All examples pass validation
```

## Contributing

To add new examples, see the [Contributing Guide](/development/contributing).

## References

- [GENEALOGIX Specification](/specification/)
- [CLI Tool](/cli)
- [JSON Schemas](/specification/schema/)
- [Contributing Guide](/development/contributing)


