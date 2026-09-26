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
- **[minimal-valid/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/minimal-valid)** - Minimal legal GEDCOM 7.0 file (32 bytes)
- **[comprehensive-spec/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/comprehensive-spec)** - Maximal GEDCOM 7.0 test (15 KB)

### GEDCOM 7.0 New Features
- **[escaping/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/escaping)** - @ character escaping (@@)
- **[extensions/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/extensions)** - Extension tags and custom structures
- **[language/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/language)** - BCP 47 language tags (LANG field)
- **[notes/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/notes)** - NOTE vs SNOTE (7.0 change)
- **[void-pointers/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/void-pointers)** - @VOID@ null references
- **[cross-references/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/cross-references)** - XREF format validation

### Data Format Testing
- **[age-values/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/age-values)** - Age field formats (5.9 KB)
- **[date-formats/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/date-formats)** - Date format validation (348 KB)
- **[long-url/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/long-url)** - Single line longer than the 5.5.1 255-character limit
- **[media-objects/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/media-objects)** - OBJE records, multi-file media, and FILE URI forms

### Family Structures
- **[same-sex-marriage/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/same-sex-marriage)** - Same-sex marriage handling
- **[remarriage/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/7.0/remarriage)** - Divorce and remarriage, as one family or as two

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
