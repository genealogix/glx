# GEDCOM 7.0 Test Files

Test files for validating GEDCOM 7.0 format parsing, new features, and specification compliance.

## Expected Format

Files should start with a header like:
```
0 HEAD
1 GEDC
2 VERS 7.0
```

## Test Categories

### Specification Samples
- **[minimal-valid/](minimal-valid/README.md)** - Minimal legal GEDCOM 7.0 file (32 bytes)
- **[comprehensive-spec/](comprehensive-spec/README.md)** - Maximal GEDCOM 7.0 test (15 KB)

### GEDCOM 7.0 New Features
- **[escaping/](escaping/README.md)** - @ character escaping (@@)
- **[extensions/](extensions/README.md)** - Extension tags and custom structures
- **[language/](language/README.md)** - BCP 47 language tags (LANG field)
- **[notes/](notes/README.md)** - NOTE vs SNOTE (7.0 change)
- **[void-pointers/](void-pointers/README.md)** - @VOID@ null references
- **[cross-references/](cross-references/README.md)** - XREF format validation

### Data Format Testing
- **[age-values/](age-values/README.md)** - Age field formats (5.9 KB)
- **[date-formats/](date-formats/README.md)** - Date format validation (348 KB)
- **[long-url/](long-url/README.md)** - Single line longer than the 5.5.1 255-character limit
- **[media-objects/](media-objects/README.md)** - OBJE records, multi-file media, and FILE URI forms

### Family Structures
- **[same-sex-marriage/](same-sex-marriage/README.md)** - Same-sex marriage handling
- **[remarriage/](remarriage/README.md)** - Divorce and remarriage, as one family or as two

## Key GEDCOM 7.0 Changes from 5.5.1

1. **NOTE/SNOTE split** - Separate tags for shared vs embedded notes
2. **@VOID@ pointers** - Explicit null reference representation
3. **BCP 47 language tags** - Standard language codes (en-US, etc.)
4. **@ escaping** - Email addresses use @@ (user@@domain.com)
5. **Extension mechanism** - URI-based schema extensions
6. **Stricter syntax** - More rigorous format requirements

## Where to Find More Test Files

- https://github.com/gedcom7code/test-files/tree/main/7
- https://gedcom.io/tools/ (official test files)
- https://gedcom.io/specifications/FamilySearchGEDCOMv7.html (specification)
