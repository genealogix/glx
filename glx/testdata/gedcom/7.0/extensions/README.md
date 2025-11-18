# GEDCOM 7.0 Extensions

## Description
GEDCOM 7.0 test file for validating handling of extension tags, custom tags, and non-standard structures. Tests parser's ability to process GEDCOM extensions while maintaining backward compatibility and data integrity.

## Test Case Type
- **Extension tag testing**: Custom tag handling
- **GEDCOM 7.0 extensions**: New extension mechanisms
- **Backward compatibility**: Extension handling across versions
- **Data preservation**: Maintain unknown/custom data
- **Specification flexibility**: Non-standard tag support

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~3.4 KB
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: Extension and custom tag testing
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **Custom tags**: Application-specific extensions
- **Extension mechanisms**: GEDCOM 7.0 extension structures
- **Schema extensions**: Registered extensions
- **Unknown tags**: Graceful handling of unrecognized tags
- **Data preservation**: Maintaining extended data
- **Tag prefixes**: Underscore-prefixed tags (_TAG)
- **Extension records**: Non-standard record types
- **Nested extensions**: Custom tags in hierarchies
- **Extension URLs**: Schema URL references

## GEDCOM 7.0 Extension Features
- **Registered extensions**: Official extension schemas
- **URI-based schemas**: Extension identification via URIs
- **Standard extension mechanism**: Formalized extension process
- **Preservation requirement**: Must preserve unknown tags
- **Extension documentation**: Self-documenting extensions

## Notes
- GEDCOM 7.0 has formal extension mechanism
- Extensions use URI-based schema identification
- Parsers must preserve unknown tags for round-trip
- Custom tags traditionally start with underscore (_)
- Essential for cross-application compatibility
- Tests parser flexibility and robustness
- Important for data migration scenarios
- Extensions allow application-specific features
- Must handle gracefully without errors

## Usage Recommendations
1. Validate custom tag preservation
2. Test round-trip with extensions (read/write/read)
3. Ensure unknown tags don't cause errors
4. Verify extension schema recognition
5. Test nested extension structures
6. Validate data integrity with extensions
7. Compare extension handling across versions
