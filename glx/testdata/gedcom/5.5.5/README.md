# GEDCOM 5.5.5 Test Files

Test files for validating GEDCOM 5.5.5 format parsing and specification compliance.

## Expected Format

Files should start with a header like:
```
0 HEAD
1 GEDC
2 VERS 5.5.5
```

## About GEDCOM 5.5.5

GEDCOM 5.5.5 is a refinement of GEDCOM 5.5.1, primarily consisting of bug fixes and clarifications rather than new features. It is backward compatible with 5.5.1 in most cases.

## Test Categories

### Specification Samples
- **[spec-samples/](spec-samples/)** - Official gedcom.org specification samples
  - Standard specification sample (2.0 KB)
  - Minimal valid GEDCOM (132 bytes - smallest possible file)
  - Remarriage scenario
  - Same-sex marriage example

## Key Files

- **minimal.ged** - Essential for testing absolute minimum GEDCOM requirements
- **sample.ged** - Reference implementation of GEDCOM 5.5.5
- **remarriage.ged** - Multiple marriages per individual
- **same-sex-marriage.ged** - Non-traditional family structures

## GEDCOM 5.5.5 vs 5.5.1

- Primarily bug fixes and clarifications
- UTF-8 encoding more standardized
- Some tag clarifications and constraints
- Backward compatible with 5.5.1 in most cases
- No major structural changes

## Where to Find More Test Files

- https://www.gedcom.org/samples.html (official specification samples)
- https://gedcom.io/specifications/ged555.pdf (specification)
