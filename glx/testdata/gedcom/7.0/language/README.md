# Language Field Testing (GEDCOM 7.0)

## Description
GEDCOM 7.0 test file for validating LANG tag handling and language code processing. Tests parser's ability to correctly process language specifications using BCP 47 language tags and handle multilingual content.

## Test Case Type
- **Language tag testing**: LANG field validation
- **BCP 47 compliance**: Standard language tag format
- **Multilingual support**: Multiple language handling
- **Language code validation**: Proper language code parsing
- **GEDCOM 7.0 features**: New language handling in version 7.0

## File Information
- **GEDCOM Version**: 7.0
- **Format**: `.ged` (standard GEDCOM text format)
- **Size**: ~2.1 KB
- **Encoding**: UTF-8
- **Source Software**: gedcom.io test suite

## Provenance
- **Source**: FamilySearch GEDCOM 7.0 specification samples
- **URL**: https://gedcom.io/tools/
- **Availability**: Open source specification test files
- **Purpose**: Language field testing and validation
- **Authority**: Official FamilySearch/gedcom.io samples

## License & Usage
- **License**: Open source test files
- **Attribution**: FamilySearch / gedcom.io
- **Usage Rights**: Educational and testing purposes
- **Restrictions**: Subject to FamilySearch GEDCOM 7.0 license

## Testing Coverage
- **LANG tag parsing**: Language field extraction
- **BCP 47 format**: RFC 5646 language tag compliance
- **Language codes**: ISO 639 language code handling
- **Script variants**: Language with script specification
- **Regional variants**: Language with region/country codes
- **Language hierarchies**: Language inheritance and fallback
- **Multilingual text**: Content in multiple languages
- **Default language**: Header-level language specification
- **Record-level language**: Per-record language overrides
- **Field-level language**: Per-field language specification

## BCP 47 Language Tag Examples
- `en` - English
- `en-US` - English (United States)
- `en-GB` - English (United Kingdom)
- `zh-Hans` - Chinese (Simplified script)
- `zh-Hant` - Chinese (Traditional script)
- `pt-BR` - Portuguese (Brazil)
- `es-MX` - Spanish (Mexico)

## GEDCOM 7.0 Language Features
- BCP 47 language tags (IETF RFC 5646)
- Language specification at multiple levels
- Default language in header
- Per-record language overrides
- Language inheritance
- Multilingual content support

## Notes
- GEDCOM 7.0 uses BCP 47 standard language tags
- Replaces older ad-hoc language codes
- Essential for international genealogy
- Supports multilingual family trees
- Parser should validate language tag format
- Important for proper text rendering
- Enables language-specific searching
- Critical for non-English genealogy data

## Usage Recommendations
1. Validate BCP 47 language tag format
2. Test language code extraction
3. Verify language inheritance behavior
4. Test multilingual content handling
5. Validate default language processing
6. Ensure proper language code storage
7. Test round-trip language preservation
