# Same-Sex Marriage Test Case (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file specifically designed to validate same-sex marriage support. This file contains an example of a same-sex marriage relationship, testing proper handling of marriage scenarios where both spouses are the same gender.

## Test Case Type
- **Edge case testing**: Tests non-traditional family structures
- **Specification compliance**: Validates GEDCOM 7.0 same-sex marriage support
- **Relationship handling**: Tests spouse relationships regardless of gender
- **Modern genealogy**: Tests contemporary family scenarios
- **Minimal but complete**: Contains just enough data to test the scenario

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~173 bytes
- **Encoding**: UTF-8
- **Source Software**: gedcom.io (official GEDCOM reference implementation)

## Provenance
- **Source**: gedcom.io v7.0 Official Test Files
- **URL**: https://gedcom.io/tools/
- **Availability**: Public specification test files
- **Purpose**: Edge case and specification compliance testing

## License & Usage
- **License**: Public domain test files
- **Attribution**: FamilySearch / GEDCOM.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: No restrictions for testing

## Testing Coverage
- Same-sex marriage relationships
- Family linking with non-traditional gender combinations
- FAMC (family as child) and FAMS (family as spouse) records
- Gender field values
- Proper relationship handling regardless of gender
- GEDCOM 7.0 compliance for modern family structures

## Notes
- Small, focused test case for specific feature validation
- Excellent for unit testing marriage relationship handling
- Tests parser's ability to handle non-traditional structures
- Demonstrates GEDCOM 7.0's inclusive family relationship modeling
- Essential for modern genealogy software testing
- Should pass without any special-case logic (relationships are gender-neutral)
