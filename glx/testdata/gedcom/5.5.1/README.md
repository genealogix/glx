# GEDCOM 5.5.1 Test Files

Test files for validating GEDCOM 5.5.1 format parsing, conversion, and feature support.

## Expected Format

Files should start with a header like:
```
0 HEAD
1 GEDC
2 VERS 5.5.1
```

## Test Categories

### Real-World Genealogy Files
- **[british-royalty/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/british-royalty)** - British monarchy genealogy (77 KB)
- **[bullinger-family/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/bullinger-family)** - Bullinger family genealogy (306 KB)
- **[kennedy-family/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/kennedy-family)** - Kennedy family genealogy (35 KB)
- **[shakespeare-family/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/shakespeare-family)** - Shakespeare family tree (6.6 KB)

### Edge Cases & Boundary Testing
- **[edge-cases/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/edge-cases)** - Non-traditional family structures
  - All gender combinations
  - Empty families
  - Same-sex marriages
  - Self-marriage scenarios
  - Unknown gender handling

### Character Encoding Tests
- **[character-encoding/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/character-encoding)** - ASCII baseline encoding tests
- **[gramps-encoding/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/gramps-encoding)** - UTF-8 and CP1252 encoding validation
  - UTF-8 without BOM
  - Windows CP1252 with various line endings

### Famous People & Historical Data
- **[famous-people/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/famous-people)** - Historical figure genealogies
  - Brontë literary family (3.1 KB)
  - European royalty (458 KB)

### Comprehensive Testing
- **[gedcom-assessment/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/gedcom-assessment)** - 233 tests across 28 areas
  - Complete GEDCOM 5.5.1 specification coverage
  - Import capability evaluation

### Performance & Stress Testing
- **[large-files/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/large-files)** - Large genealogy databases
  - Habsburg family (10 MB - largest test file)
  - Queen family (2.5 MB)

### Torture Testing
- **[torture-test-551/](https://github.com/genealogix/glx/tree/main/glx/testdata/gedcom/5.5.1/torture-test-551)** - Comprehensive tag coverage
  - Every GEDCOM 5.5 tag in every location
  - Parser stress testing

## Where to Find More Test Files

- https://github.com/gedcom7code/test-files/tree/main/5
- https://github.com/findmypast/gedcom-samples
- https://github.com/gramps-project/gramps/tree/master/data/tests
- https://gedcom.io/specifications/ged551.pdf (specification with samples)
