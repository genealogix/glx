# Minimal Valid GEDCOM 7.0 File

## Description
A minimal but completely valid GEDCOM 7.0 file that represents the smallest possible legal GEDCOM 7.0 document. This file contains only the essential required elements for a valid GEDCOM 7.0 transmission, making it ideal for testing baseline parser functionality and validation.

## Test Case Type
- **Minimal valid test**: Tests parser's ability to handle minimal input
- **Specification baseline**: Validates baseline GEDCOM 7.0 structure requirements
- **Regression testing**: Quick sanity check for parser functionality
- **Unit testing**: Small, focused test for parser core logic
- **Edge case**: Tests parser with minimal data

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~32 bytes (one of the smallest valid GEDCOM files possible)
- **Encoding**: UTF-8
- **Source Software**: gedcom.io (official GEDCOM reference implementation)

## Provenance
- **Source**: gedcom.io v7.0 Official Test Files
- **URL**: https://gedcom.io/tools/
- **Availability**: Public specification test files
- **Purpose**: Baseline validation and minimal file testing

## License & Usage
- **License**: Public domain test files
- **Attribution**: FamilySearch / GEDCOM.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: No restrictions for testing

## Testing Coverage
- HEAD record (required)
- GEDC tag with version (required for GEDCOM 7.0)
- TRLR record (required)
- Line format and structure
- Cross-references (if any)
- Minimal required fields
- Valid but sparse GEDCOM structure

## Notes
- Smallest possible valid GEDCOM 7.0 file
- Perfect for quick parser sanity checks
- Tests essential parser functionality
- Good for regression testing in CI/CD pipelines
- Should parse without errors
- Useful for testing with minimal resource constraints
- Contains only the absolute minimum required elements
- Excellent baseline for error handling and validation testing
