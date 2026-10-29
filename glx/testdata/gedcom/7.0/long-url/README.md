# Long URL Test Case (GEDCOM 7.0)

## Description
A GEDCOM 7.0 file whose only substantive content is a submitter record with a `WWW` URL of roughly 800 characters on a single line. It exercises line-length handling: GEDCOM 7.0 dropped the 255-character line limit that 5.5.1 imposed, so the URL must survive import without being truncated or split.

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~1 KB
- **Encoding**: UTF-8
- **Source Software**: gedcom.io (official GEDCOM reference implementation)

## Provenance
- **Source**: gedcom.io v7.0 Official Test Files
- **URL**: https://gedcom.io/tools/
- **Availability**: Public specification test files
- **Purpose**: Line-length edge case testing

## License & Usage
- **License**: Public domain test files
- **Attribution**: FamilySearch / GEDCOM.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: No restrictions for testing

## Testing Coverage
- Lines longer than the GEDCOM 5.5.1 255-character limit
- `SUBM` record with `NAME` and `WWW`
- An archive with no persons, only submitter metadata

## Notes
- Exporting to GEDCOM 5.5.1 needs `CONC` splitting to stay within that version's line limit
