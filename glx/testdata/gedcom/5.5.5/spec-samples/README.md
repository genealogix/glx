# GEDCOM 5.5.5 Specification Samples

## Description
Official GEDCOM 5.5.5 specification sample files from gedcom.org, providing reference implementations of the GEDCOM 5.5.5 standard. These files serve as authoritative examples of proper GEDCOM 5.5.5 format and demonstrate various genealogical scenarios specified in the standard.

## Test Case Type
- **Specification compliance**: Official reference implementations
- **Standard validation**: Authoritative GEDCOM 5.5.5 examples
- **Format reference**: Canonical GEDCOM structure examples
- **Minimal testing**: Smallest valid GEDCOM files
- **Scenario testing**: Common genealogical use cases

## File Information
- **GEDCOM Version**: 5.5.5
- **Format**: `.GED` (standard GEDCOM text format)
- **Size**: 132 bytes to 2.0 KB
- **Encoding**: UTF-8
- **Source Software**: GS (GEDCOM Specification reference software)

## Provenance
- **Source**: gedcom.org official specification samples
- **URL**: https://www.gedcom.org/samples.html
- **Availability**: Public specification samples
- **Purpose**: Reference implementation and specification compliance
- **Authority**: Official GEDCOM organization samples

## License & Usage
- **License**: Public specification samples
- **Attribution**: gedcom.org
- **Usage Rights**: Educational, testing, and reference purposes
- **Restrictions**: None specified for specification samples

## Test Files

### sample.ged
- **Original Name**: `555SAMPLE.GED`
- **Description**: Standard GEDCOM 5.5.5 specification sample
- **Size**: 2.0 KB
- **Encoding**: UTF-8
- **Purpose**: Reference implementation of GEDCOM 5.5.5
- **Coverage**: Standard genealogical data structures
- **Use Case**: General GEDCOM format validation

### minimal.ged
- **Original Name**: `MINIMAL555.GED`
- **Description**: Minimal valid GEDCOM 5.5.5 file
- **Size**: 132 bytes
- **Encoding**: UTF-8
- **Purpose**: Smallest possible valid GEDCOM file
- **Coverage**: Absolute minimum required structures
- **Use Case**: Minimal format validation and parser baseline
- **Note**: Essential for testing minimum requirements

### remarriage.ged
- **Original Name**: `REMARR.GED`
- **Description**: Remarriage scenario example
- **Size**: 1.3 KB
- **Encoding**: UTF-8
- **Purpose**: Demonstrates remarriage handling
- **Coverage**: Multiple marriages for same individual
- **Use Case**: Tests family relationship complexity
- **Scenario**: Individual married multiple times

### same-sex-marriage.ged
- **Original Name**: `SSMARR.GED`
- **Description**: Same-sex marriage example
- **Size**: 864 bytes
- **Encoding**: UTF-8
- **Purpose**: Demonstrates same-sex marriage handling
- **Coverage**: Non-traditional family structures
- **Use Case**: Tests gender-neutral relationship parsing
- **Scenario**: Same-sex marriage with GEDCOM 5.5.5
- **Note**: Important for modern genealogy applications

## Testing Coverage
- **Specification compliance**: Official GEDCOM 5.5.5 examples
- **Minimal format**: Absolute minimum valid structure
- **Standard scenarios**: Common genealogical use cases
- **Remarriage handling**: Multiple marriages per person
- **Same-sex relationships**: Modern family structures
- **UTF-8 encoding**: Modern character encoding
- **Header structures**: Proper GEDCOM headers
- **Family structures**: Various family configurations
- **Individual records**: Person data formatting
- **Format validation**: Canonical structure examples

## Notes
- **Official specification samples**: Authoritative reference
- **Minimal.ged is essential**: Tests absolute minimum requirements
- **All files are reference implementations**: Should parse perfectly
- **UTF-8 encoding standard**: Modern GEDCOM files use UTF-8
- **Specification version 5.5.5**: Later than 5.5.1, more refined
- **Same-sex marriage support**: Shows GEDCOM 5.5.5 capabilities
- **Remarriage testing**: Common real-world scenario
- **Parser validation baseline**: Use for initial parser testing
- **Zero tolerance for errors**: These should never fail to parse
- **Specification compliance check**: Compare parser output to spec

## Usage Recommendations
1. Start with minimal.ged for basic parser validation
2. Use sample.ged as reference implementation
3. Test remarriage handling with remarriage.ged
4. Validate same-sex relationship support with same-sex-marriage.ged
5. Compare parser output to specification requirements
6. Use as baseline for GEDCOM 5.5.5 compliance testing
7. Reference when implementing new features
8. Regression testing after parser changes

## Differences from GEDCOM 5.5.1
- GEDCOM 5.5.5 is a refinement of 5.5.1
- Primarily bug fixes and clarifications
- UTF-8 encoding is more standardized
- Some tag clarifications and constraints
- Backward compatible with 5.5.1 in most cases
