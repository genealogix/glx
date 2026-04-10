# Note Field Testing (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file for validating NOTE and SNOTE tag handling. Tests parser's ability to process both shared notes (NOTE record references) and embedded notes (SNOTE inline text), which changed between GEDCOM 5.5.1 and 7.0.

## Test Case Type
- **Note handling**: NOTE and SNOTE tag processing
- **Shared notes**: NOTE record references
- **Embedded notes**: Inline SNOTE text
- **GEDCOM 7.0 changes**: New note structure in version 7.0
- **Backward compatibility**: Differences from GEDCOM 5.5.1

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~390 bytes
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: Note structure testing
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **NOTE tag**: Shared note references
- **SNOTE tag**: Embedded inline notes
- **Note records**: Separate NOTE record structures
- **Cross-references**: NOTE record pointers (@N1@, etc.)
- **Inline text**: Direct SNOTE text content
- **CONC/CONT**: Note text continuation
- **Note sharing**: Multiple references to same note
- **Note inheritance**: Note placement in record hierarchy

## GEDCOM 7.0 vs 5.5.1 Note Differences

### GEDCOM 5.5.1
- `NOTE` with text = embedded note
- `NOTE @N1@` = shared note reference
- Single tag for both uses

### GEDCOM 7.0
- `SNOTE` with text = embedded note (new tag)
- `NOTE @N1@` = shared note reference only
- Separate tags for clarity and type safety

## Example Structures

### Shared Note (GEDCOM 7.0)
```
1 NOTE @N1@
0 @N1@ NOTE This is a shared note.
```

### Embedded Note (GEDCOM 7.0)
```
1 SNOTE This is an inline embedded note.
```

## Notes
- GEDCOM 7.0 splits NOTE into NOTE and SNOTE
- Clarifies distinction between shared and embedded notes
- Improves data model clarity
- Parser must handle both NOTE and SNOTE
- Important for note deduplication
- Essential for proper data migration from 5.5.1
- SNOTE is new in GEDCOM 7.0
- NOTE usage changed from 5.5.1

## Usage Recommendations
1. Validate NOTE reference resolution
2. Test SNOTE inline text extraction
3. Verify shared note deduplication
4. Test note record creation
5. Validate CONC/CONT in both NOTE and SNOTE
6. Test migration from 5.5.1 NOTE to 7.0 SNOTE
7. Ensure backward compatibility handling
