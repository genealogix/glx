/*
Package lib provides core GLX functionality including GEDCOM import,
validation, and serialization.

# GEDCOM Import

The GEDCOM import converts GEDCOM 5.5.1 and 7.0 files into GLX format.
Implementation files: gedcom_*.go

## Conversion Flow

	GEDCOM File → ParseGEDCOM() → ConversionContext
	    ↓
	Pass 1: Individuals, Sources, Notes, Places
	Pass 2: Families (marriage relationships)
	Pass 3: Parent-Child Relationships (with PEDI types)
	    ↓
	GLXFile

## Core Files

  - gedcom_converter.go: Main conversion orchestrator
  - gedcom_individual.go: Person and event conversion
  - gedcom_family.go: Family/relationship conversion
  - gedcom_source.go: Source/citation conversion
  - gedcom_place.go: Place hierarchy building

## Key Mappings

Person properties:
  - NAME → properties.name.value + optional fields
  - SEX → properties.gender
  - BIRT/DEAT/CHR → Event entities
  - SOUR → Citation + Assertion entities

Name fields are only populated from explicit GEDCOM substructure tags
(GIVN, SURN, NPFX, NICK, SPFX, NSFX) - never inferred from parsing.

Relationships:
  - HUSB + WIFE → marriage
  - CHIL + PEDI birth → biological-parent-child
  - CHIL + PEDI adopted → adoptive-parent-child
  - CHIL + PEDI foster → foster-parent-child
  - CHIL (no PEDI) → parent-child

Places: GEDCOM flat strings become hierarchical GLX entities linked by
parent references. When PLAC is missing, hierarchy is built from ADDR
subfields (CITY/ADR2, STAE, CTRY).

## Version Differences

  - Shared notes: 5.5.1 uses NOTE, 7.0 uses SNOTE
  - External IDs: Only in 7.0 (EXID)

Both versions are fully supported.

## Error Handling

Errors accumulate in ConversionContext.Errors for partial conversion.
Philosophy: convert as much as possible, report all errors at end.

Malformed line recovery: Lines missing CONT/CONC prefixes (common in
MyHeritage exports) are treated as continuations of the previous line.

## Adding New Tag Support

 1. Find the appropriate converter file
 2. Add tag handling in the switch statement
 3. Map to GLX entity
 4. Add test case
 5. Run make test

## References

  - GEDCOM specs: docs/gedcom-spec/ (use split PDFs, not full specs)
  - GLX spec: specification/
  - User guide: docs/guides/migration-from-gedcom.md
*/
package lib
