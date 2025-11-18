# GEDCOM Assessment Test Suite

## Overview

The GEDCOM Assessment test suite is a comprehensive collection of 233 tests across 28 test areas designed to evaluate the GEDCOM 5.5.1 import capability of genealogy software applications. This test suite was created by John Cardinal and is available at [GEDCOM Assessment](https://web.archive.org/web/20211220004240/https://www.gedcomassessment.com/en/index.htm).

## Purpose

This test suite helps genealogists and software developers evaluate GEDCOM transfer capabilities by providing a standardized set of test cases. Unlike testing with exported GEDCOM files from specific programs, using this hand-crafted GEDCOM file focuses the evaluation on the import process of the target application, eliminating uncertainty about whether issues originate from the export or import process.

## Test Structure

### Test Areas (28 total)
Tests are organized by surname, with each surname representing a different test area:

1. **Event Primary** - Primary event handling tests
2. **Date Valid** - Date validation and parsing tests
3. **Exhibit Person** - Person record variations and alternatives
4. **Custom Tags** - Custom GEDCOM tag support
5. **Media Objects** - Multimedia file handling
6. **Source Citations** - Source and citation processing
7. **Place Names** - Geographic location handling
8. **Name Variations** - Different name format handling
9. **Family Structures** - Family relationship testing
10. **Notes and Comments** - Text field processing
11. **Addresses** - Address field handling
12. **Phone Numbers** - Contact information
13. **Email and URLs** - Digital contact methods
14. **Religious Data** - Religious affiliation fields
15. **Military Service** - Military record handling
16. **Education** - Educational background
17. **Occupation** - Employment history
18. **Property** - Real estate and assets
19. **Medical** - Health-related information
20. **DNA** - Genetic testing data
21. **Social Media** - Online presence
22. **Custom Events** - Non-standard event types
23. **Extensions** - GEDCOM extension support
24. **Validation** - Error handling and validation
25. **Performance** - Large dataset handling
26. **Compatibility** - Cross-version compatibility
27. **Edge Cases** - Boundary condition testing
28. **Stress Tests** - Robustness and error recovery

### Test Naming Convention

Each test uses a structured naming system:
- **Surname**: Indicates the test area
- **Given Name**: Format is "XX-Description-Context"
  - Example: "01-Birth-1804" = First test of birth event handling with 1804 date
  - Example: "04-BEF 1811" = Fourth test of "before 1811" date handling

## Test Content Types

### Valid Records
- Demonstrate proper GEDCOM 5.5.1 specification compliance
- Show how target applications should interpret standard data
- Help identify implementation gaps or bugs

### Invalid Records
- Test error handling capabilities
- Identify where applications deviate from standards
- Discover required non-standard records for some applications

### Custom Records
- Test support for common GEDCOM extensions
- Evaluate flexibility in handling non-standard data
- Identify vendor-specific feature support

## Usage Process

### 1. Download and Setup
- Download `assess.ged` from the GEDCOM Assessment website
- Download referenced image files (if available)
- Save files in a convenient location

### 2. Import Testing
- Import `assess.ged` into a new database/file/project in target application
- Ensure no existing data conflicts with test data
- Monitor import process for errors or warnings

### 3. Evaluation
- Review each test person/record in the target application
- Compare results against expected outcomes
- Use test area help pages for detailed evaluation criteria
- Document any discrepancies or failures

### 4. Recording Results
- Use provided data entry utilities to capture results
- Note specific failures, warnings, or unexpected behaviors
- Share results with the GEDCOM Assessment project

## Test Evaluation Examples

### Simple Assessments
- **Date Valid tests**: Compare given name date with birth event date
  - "04-BEF 1811" should show "before 1811" in birth date (ignoring formatting variations)

### Complex Assessments
- **Exhibit Person tests**: Compare multiple variations to find best approach
- **Custom tag tests**: Evaluate support for non-standard GEDCOM tags
- **Media object tests**: Check file reference and display capabilities

## GEDCOM Import Log Analysis

Most applications generate import logs with:
- Line numbers for problematic records
- Error messages and warnings
- Validation feedback
- Processing statistics

Use the Line Numbers utility to correlate log entries with specific tests.

## Validation Notes

**Important**: This test file intentionally contains validation errors and warnings by design. These are used to:
- Probe where target applications deviate from GEDCOM 5.5.1 standards
- Test error handling and recovery capabilities
- Identify different interpretations of the GEDCOM specification
- Discover required non-standard records for some applications

## File Information

- **GEDCOM Version**: 5.5.1
- **Total Tests**: 233
- **Test Areas**: 28
- **File Size**: Approximately 147 KB
- **Encoding**: UTF-8
- **Line Endings**: Standard GEDCOM format

## Copyright and Usage

- **Copyright**: © 2020 by John Cardinal
- **Usage**: Free for testing and evaluation
- **Distribution**: Prohibited (copies may not be distributed)
- **Modification**: Allowed for personal use only

## Contact Information

For more information about the GEDCOM Assessment test suite:
- **Email**: John Cardinal
- **Website**: [GEDCOM Assessment](https://web.archive.org/web/20211220004240/https://www.gedcomassessment.com/en/index.htm)
- **Note**: Email challenges from spam-blocker tools are not responded to

## Test Coverage Benefits

This comprehensive test suite provides:
- **Standardized evaluation** across different genealogy applications
- **Comprehensive coverage** of GEDCOM 5.5.1 features
- **Error handling assessment** for robust applications
- **Extension support testing** for flexible applications
- **Performance evaluation** for large dataset handling
- **Compatibility testing** across different GEDCOM versions

## Related Resources

- **Test Area Help Pages**: Detailed explanations for each test category
- **Image Files**: Multimedia content referenced by test records
- **Data Entry Utilities**: Tools for recording test results
- **Assessment Database**: Results from testing various applications
- **Change Log**: Version history and updates
