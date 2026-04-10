# Gramps Encoding Validation (GEDCOM 5.5.1)

## Description
Collection of GEDCOM 5.5.1 test files from the Gramps open-source genealogy application, specifically focused on character encoding and line ending variations. These files validate parser handling of UTF-8, Windows CP1252 encoding, and various line terminator combinations (CR, LF, CRLF).

## Test Case Type
- **Encoding validation**: UTF-8 and Windows CP1252 character sets
- **Line ending testing**: CR, LF, and CRLF line terminator variations
- **Cross-platform compatibility**: Windows and Unix line endings
- **BOM handling**: UTF-8 with and without byte order marks
- **Real application output**: Actual Gramps genealogy software exports

## File Information
- **GEDCOM Version**: 5.5
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: 3.4 KB to 8.4 KB
- **Encoding**: UTF-8 (no BOM), CP1252 (Windows-1252)
- **Source Software**: Gramps Project / Unknown

## Provenance
- **Source**: Gramps Project official test data
- **URL**: https://github.com/gramps-project/gramps/tree/master/data/tests
- **Availability**: Open source test files
- **Purpose**: Import/export validation and encoding testing
- **Collection**: Official Gramps application test suite

## License & Usage
- **License**: GPL (Gramps Project license)
- **Attribution**: Gramps Project
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to GPL license terms

## Test Files

### utf8-nobom-lf.ged
- UTF-8 encoding without byte order mark (BOM)
- LF (Line Feed) line terminators only
- 8.4 KB file size
- Tests UTF-8 without BOM detection
- Unix/Linux line ending style
- Original: `imp_UTF_8_NOBOM_LF.ged`

### cp1252-crlf.ged
- Windows CP1252 (Windows-1252) character encoding
- CRLF (Carriage Return + Line Feed) line terminators
- 3.6 KB file size
- Tests Windows encoding handling
- DOS/Windows line ending style
- Original: `imp_cp1252_CRLF.ged`

### cp1252-lf.ged
- Windows CP1252 (Windows-1252) character encoding
- LF (Line Feed) line terminators only
- 3.4 KB file size
- Tests Windows encoding with Unix line endings
- Mixed platform line ending style
- Original: `imp_cp1252_LF.ged`

## Testing Coverage
- **UTF-8 encoding**: Modern Unicode encoding without BOM
- **CP1252 encoding**: Windows-1252 character set (Western European)
- **Line ending variations**: LF and CRLF combinations
- **Cross-platform compatibility**: Windows and Unix line endings
- **BOM detection**: UTF-8 without byte order mark
- **Encoding header validation**: CHAR tag processing
- **Special characters**: Extended ASCII and Unicode characters
- **Real-world data**: Actual genealogy application exports

## Notes
- Essential for cross-platform encoding testing
- Validates handling of Windows encoding (CP1252)
- Tests UTF-8 without BOM (common in modern files)
- Important for line ending normalization logic
- Gramps is a widely-used open-source genealogy application
- Files represent real-world export scenarios
- Should handle both Windows and Unix line endings gracefully
- CP1252 is common in older Windows genealogy software
- UTF-8 without BOM is standard in modern applications
- Good for regression testing with actual application output
