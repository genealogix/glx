# Remarriage Test Cases (GEDCOM 7.0)

## Description
Two GEDCOM 7.0 files that encode the same life history in the two ways the specification allows: John Q Public marries Jane Doe (1911), divorces her (1912), marries Mary Roe (1913), is widowed (1914), and remarries Jane (1914).

## Files
- **`remarriage1.ged`** — the second marriage to Jane is a second `MARR` event on the *same* `FAM` record (`@F1@` holds `MARR`, `DIV`, `MARR`)
- **`remarriage2.ged`** — the second marriage to Jane is a *separate* `FAM` record (`@F3@`), so the couple appears in two families

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~390–450 bytes each
- **Encoding**: UTF-8 with BOM
- **Source Software**: gedcom.io (official GEDCOM reference implementation)

## Provenance
- **Source**: gedcom.io v7.0 Official Test Files
- **URL**: https://gedcom.io/tools/
- **Availability**: Public specification test files
- **Purpose**: Family-structure edge case testing

## License & Usage
- **License**: Public domain test files
- **Attribution**: FamilySearch / GEDCOM.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: No restrictions for testing

## Testing Coverage
- Multiple `FAMS` links on one individual
- `DIV` events between two `MARR` events in one family
- The same couple in two distinct family records
- A spouse's `DEAT` ending a marriage
- Timeline ordering of marriage, divorce, and remarriage events

## Notes
- Both files should import to equivalent people; they differ only in how many relationship records the couple gets
- Useful for relationship-path, timeline, and merge-persons tests that need more than one spouse
