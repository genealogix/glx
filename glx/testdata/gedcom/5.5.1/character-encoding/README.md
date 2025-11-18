# Character Encoding Testing (GEDCOM 5.5.1)

## Description
Collection of GEDCOM 5.5.1 test files with various character encodings to validate parser handling of ASCII, ANSEL, and UNICODE character sets. These files test the parser's ability to correctly detect, decode, and process different character encodings specified in the GEDCOM header.

## Test Case Type
- **Character encoding validation**: Tests various encoding formats
- **ASCII baseline**: Simple ASCII-only GEDCOM file
- **Encoding detection**: Tests automatic encoding detection
- **Special character handling**: Validates proper character set processing
- **Cross-platform compatibility**: Tests line endings and byte order marks

## File Information
- **GEDCOM Version**: 5.5.1 / 5.5
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: 742 bytes to 32 KB
- **Encoding**: US-ASCII, ANSEL, UNICODE (various byte orders)
- **Source Software**: Various (Heiner Eichmann test suite)

## Provenance
- **Source**: Heiner Eichmann GEDCOM 5.5 Samples
- **URL**: http://heiner-eichmann.de/gedcom/gedcom.htm
- **Availability**: Public test files
- **Purpose**: Character set and encoding testing

## License & Usage
- **License**: Public test files
- **Attribution**: Heiner Eichmann
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: None specified

## Test Files

### simple-ascii.ged
- Most simple GEDCOM transmission
- Pure US-ASCII encoding
- Baseline test for minimal encoding requirements
- 742 bytes
- Tests basic ASCII character handling

## Testing Coverage
- **ASCII encoding**: Standard US-ASCII character set
- **Encoding detection**: Header CHAR tag parsing
- **Basic GEDCOM structure**: Minimal but valid GEDCOM file
- **Line terminators**: Standard line ending handling
- **Character set validation**: Ensures ASCII-only processing
- **Cross-platform compatibility**: Tests portability of simple encoding

## Notes
- Essential baseline for encoding tests
- All genealogy software should handle ASCII encoding
- Simple ASCII is the most portable GEDCOM encoding
- Good starting point for encoding validation logic
- Tests minimum encoding requirements
- Should parse without any encoding-related errors
- Useful for quick encoding sanity checks
- Can serve as reference for comparing other encodings
