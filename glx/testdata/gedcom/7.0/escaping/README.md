# @ Character Escaping (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file for validating proper handling of @ character escaping and escape sequences. Tests parser ability to correctly process escaped @ symbols, which are used for cross-references in GEDCOM and require special handling when appearing in text content.

## Test Case Type
- **Escape sequence testing**: @ character escaping validation
- **Special character handling**: @ symbol in text vs. cross-references
- **Parser disambiguation**: Distinguishing references from literal text
- **GEDCOM 7.0 features**: New escaping rules in version 7.0
- **Text processing**: Proper string parsing with escape sequences

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~935 bytes
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: GEDCOM 7.0 feature testing
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **@ symbol escaping**: Literal @ in text content
- **Cross-reference format**: @XREF@ vs escaped @@
- **Escape sequence parsing**: @@  (double @ for single @)
- **Text field handling**: @ in notes, names, and other text
- **Parser ambiguity resolution**: References vs. literal text
- **Email addresses**: Handling @ in email addresses
- **GEDCOM 7.0 rules**: New escaping specification
- **Backward compatibility**: Differences from GEDCOM 5.5.1

## Notes
- Essential for proper text parsing in GEDCOM 7.0
- GEDCOM 7.0 changed @ escaping rules from 5.5.1
- @ is reserved for cross-references (@I1@, @F1@, etc.)
- Literal @ in text must be escaped as @@
- Email addresses require @@ escaping
- Parser must distinguish @XREF@ from @@ correctly
- Critical for data integrity (don't lose @ symbols)
- Important for email and internet data
- Tests GEDCOM 7.0 specification compliance

## GEDCOM 7.0 Escaping Rules
- `@` = Cross-reference delimiter (reserved)
- `@@` = Escaped @ (renders as single @)
- `@VOID@` = Null reference (special case)
- Email: `user@@domain.com` (double @@)
- Cross-ref: `@I1@` (single @)

## Usage Recommendations
1. Validate @ character detection
2. Test escape sequence processing
3. Verify email address handling
4. Ensure cross-references parse correctly
5. Compare with GEDCOM 5.5.1 handling
6. Test round-trip preservation (read/write/read)
7. Validate text output doesn't double-escape
