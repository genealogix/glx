# Edge Case Testing (GEDCOM 5.5.1)

## Description
Collection of GEDCOM 5.5.1 edge case test files that test parser robustness with unusual but valid genealogical scenarios. These files validate handling of non-traditional family structures, gender variations, and other boundary cases that may occur in real-world genealogy data.

## Test Case Type
- **Edge case testing**: Non-traditional family structures and relationships
- **Gender variations**: All gender combinations and unknown genders
- **Family structure testing**: Empty families, unusual parent-child relationships
- **Boundary conditions**: Self-marriage, same-sex marriages, childless families
- **Parser robustness**: Tests handling of valid but unusual GEDCOM structures

## File Information
- **GEDCOM Version**: 5.5.1
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: 539 bytes to 1.1 KB per file
- **Encoding**: UTF-8 (with BOM)
- **Source Software**: Family Tree Maker 17.0

## Provenance
- **Source**: findmypast/gedcom-samples (GEDCOM Permutations)
- **URL**: https://github.com/findmypast/gedcom-samples
- **Availability**: Open source test files
- **Purpose**: Edge case testing and parser validation

## License & Usage
- **License**: Open source test files
- **Attribution**: findmypast
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to repository license

## Test Files

### all-genders.ged
- Tests all possible gender combinations (Male, Female, Unknown)
- Validates handling of different gender values
- Tests gender field parsing

### empty-family.ged
- Tests family structures with no members
- Validates handling of empty family records
- Tests parser resilience to minimal data

### female-female-marriage.ged
- Tests same-sex female marriage
- Validates handling of non-traditional family structures
- Tests gender-neutral relationship parsing

### male-male-marriage.ged
- Tests same-sex male marriage
- Validates handling of non-traditional family structures
- Tests parser flexibility with relationship types

### self-marriage.ged
- Tests individual married to themselves
- Validates handling of unusual relationship structures
- Tests circular reference detection

### unknown-unknown-marriage.ged
- Tests marriage between individuals with unknown gender
- Validates handling of missing gender information
- Tests gender-agnostic relationship parsing

## Testing Coverage
- All gender combinations (M, F, U)
- Empty family structures
- Same-sex marriages (both male and female)
- Self-referential relationships
- Unknown gender handling
- Non-traditional family structures
- Parser error handling and validation
- Relationship type flexibility

## Notes
- Essential for testing parser robustness
- Validates handling of modern family structures
- Tests compliance with GEDCOM 5.5.1 specification edge cases
- Useful for regression testing unusual scenarios
- Should not cause parser errors despite unusual content
- All files are valid GEDCOM 5.5.1 format
- Tests demonstrate flexibility needed for real-world genealogy data
- Important for LGBTQ+ genealogy support
