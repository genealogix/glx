# Cross-Reference Format Testing (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file for validating cross-reference (XREF) format handling, pointer syntax, and reference resolution. Tests parser's ability to correctly process various cross-reference formats and ensure referential integrity.

## Test Case Type
- **Cross-reference testing**: XREF format validation
- **Pointer syntax**: @ID@ format parsing
- **Reference resolution**: Resolving pointers to records
- **Referential integrity**: Validating reference targets exist
- **GEDCOM 7.0 features**: Cross-reference format rules

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~405 bytes
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: Cross-reference format testing
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **XREF syntax**: @ID@ format validation
- **Reference types**: Individual (@I1@), Family (@F1@), etc.
- **ID formats**: Various valid ID patterns
- **Case sensitivity**: XREF case handling
- **ID length**: Short and long identifiers
- **Special characters**: Allowed characters in IDs
- **Reference resolution**: Pointer target lookup
- **Referential integrity**: Validating all pointers resolve
- **Dangling references**: Handling missing target records
- **Circular references**: Detecting reference cycles

## Cross-Reference Format

### Syntax
```
@<ID>@
```

### Valid Characters
- Alphanumeric: `A-Z`, `a-z`, `0-9`
- Special: `_` (underscore)
- No spaces or other punctuation

### Examples
```
@I1@        # Individual 1
@F23@       # Family 23
@S_SOURCE@  # Source with underscore
@NOTE42@    # Note 42
@VOID@      # Special void pointer
```

## GEDCOM 7.0 XREF Rules
- Must start and end with `@`
- ID must be 1-22 characters
- Alphanumeric plus underscore only
- Case-sensitive comparisons
- Must be unique within file
- @VOID@ is reserved for null pointers
- All references must resolve to valid records

## Notes
- Critical for data integrity
- Must resolve all cross-references
- Invalid references indicate data corruption
- Case sensitivity matters in GEDCOM 7.0
- Essential for relationship building
- Parser should validate referential integrity
- Should detect and report dangling references
- Circular references should be handled gracefully

## Usage Recommendations
1. Validate XREF syntax format
2. Test various ID patterns and lengths
3. Verify case-sensitive matching
4. Test reference resolution
5. Validate referential integrity checking
6. Test error handling for missing targets
7. Detect circular reference loops
8. Ensure unique ID enforcement
