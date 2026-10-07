# Multimedia Object Test Cases (GEDCOM 7.0)

## Description
GEDCOM 7.0 files covering `OBJE` multimedia records: multiple files per object, media types, titles, notes, and the range of URI forms a `FILE` payload may take.

## Files
- **`obje-1.ged`** (~425 bytes) — two `OBJE` records with several `FILE`s each (`FORM`, `MEDI`, `TITL`), linked from an `INDI` with a link-level `TITL`
- **`filename-1.ged`** (~1.8 KB) — one `OBJE` whose `FILE`s use every URI shape the spec discusses: `file://` URIs (Unix, Windows drive letter, remote host), relative paths, percent-escaped segments, `https://` URLs with query/fragment and non-ASCII characters, and the discouraged-but-legal `gedcom.ged` / `MANIFEST.MF` / `META-INF/` names

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Encoding**: UTF-8 with BOM
- **Source Software**: gedcom.io (official GEDCOM reference implementation)

## Provenance
- **Source**: gedcom.io v7.0 Official Test Files
- **URL**: https://gedcom.io/tools/
- **Availability**: Public specification test files
- **Purpose**: Media record and file-reference testing

## License & Usage
- **License**: Public domain test files
- **Attribution**: FamilySearch / GEDCOM.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: No restrictions for testing

## Testing Coverage
- `OBJE` records with more than one `FILE`
- `FORM` media types and `MEDI` medium values
- Media linked from an individual, with a link-level title
- `FILE` payloads that are URIs, relative paths, or percent-escaped strings
- None of the referenced files exist: importers must not require the binaries to be present

## Notes
- Useful for import, export, and publish tests that need Media entities without shipping any media binaries
