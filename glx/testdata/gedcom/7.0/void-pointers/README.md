# Void Pointer Testing (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file for validating @VOID@ null reference handling. Tests parser's ability to process void pointers, which represent explicitly null or undefined cross-references in GEDCOM 7.0.

## Test Case Type
- **Null reference testing**: @VOID@ pointer handling
- **GEDCOM 7.0 features**: New void pointer mechanism
- **Reference validation**: Null vs. missing vs. valid references
- **Data integrity**: Proper null reference representation
- **Specification compliance**: @VOID@ usage rules

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~292 bytes
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: Void pointer testing
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **@VOID@ pointer**: Null reference handling
- **Explicitly null**: Distinction from missing field
- **Reference validation**: Void vs. valid cross-references
- **Family links**: @VOID@ in FAMC/FAMS fields
- **Spouse pointers**: @VOID@ for HUSB/WIFE
- **Parent pointers**: @VOID@ for parent references
- **Optional references**: Null pointer representation
- **Data semantics**: Meaning of void vs. omitted

## @VOID@ Usage

### Purpose
- Explicitly marks a reference as null/undefined
- Distinguished from simply omitting the field
- Provides semantic clarity in data model

### Example
```
1 FAMC @VOID@   # Explicitly no family-child link
1 FAMS @F1@     # Valid family-spouse link
```

### Difference from Omission
- `1 FAMC @VOID@` = "explicitly no family link"
- (no FAMC tag) = "family link not specified"

## GEDCOM 7.0 Void Pointer Rules
- @VOID@ is a reserved cross-reference ID
- Used only for null pointer representation
- Cannot be used as actual record ID
- Semantically different from omitting tag
- Valid in any cross-reference context
- Should not resolve to actual record

## Notes
- @VOID@ is new in GEDCOM 7.0
- Provides explicit null representation
- Parser must recognize @VOID@ as special
- Should not attempt to resolve @VOID@ references
- Important for data model completeness
- Clarifies "unknown" vs "known to be none"
- Essential for proper data semantics
- Not present in GEDCOM 5.5.1

## Usage Recommendations
1. Validate @VOID@ recognition
2. Test that @VOID@ doesn't resolve to record
3. Verify @VOID@ in various reference contexts
4. Test semantic difference from omitted tags
5. Ensure @VOID@ preserved in round-trip
6. Validate error if @VOID@ used as record ID
7. Test @VOID@ in family/spouse/parent pointers
