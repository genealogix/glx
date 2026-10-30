# Test Data for GLX Validator

This directory contains test data for validating the GLX specification validator.

## Directory Structure

- `valid/` - Archives that should pass validation (some with expected warnings)
- `invalid/` - Archives that should fail validation, each for one stated reason
- `gedcom/` - GEDCOM files for import and export tests
- `merge-driver/` - Inputs for the git merge-driver tests

Validate an archive directory as a directory (`glx validate testdata/invalid/<name>`), not by its `archive.glx`: validating a single file skips the cross-reference checks, so a broken reference is not reported.

## Invalid Archives

`TestInvalidFixtures` (`glx/testdata_fixtures_test.go`) validates every directory here. It checks that each one fails with the errors listed for it in `invalidFixtures`, and that every error reported matches one of them, so a fixture cannot pass by failing for an unrelated reason. A new directory without an entry fails the test.

Each fixture defines the vocabulary entries it uses, so it does not depend on the standard vocabularies being filled in (genealogix/glx#1355).

- `assertion-participant-and-property/` - An assertion with both `participant` and `property`
- `assertion-participant-and-value/` - An assertion with both `participant` and `value`
- `assertion-participant-invalid-person/` - An assertion participant naming a person that does not exist
- `assertion-participant-invalid-role/` - An assertion participant with a role not in `participant_roles`
- `broken-references/` - An assertion referencing an event, citation and source that do not exist
- `comprehensive-broken-references/` - Broken references from citations, events, places and sources (its vocabularies are deliberately customized)
- `duplicate-ids/` - The same person ID defined in two files
- `invalid-assertion-properties/` - An assertion with a property name not in any vocabulary. Skipped until the validator checks assertion property names (genealogix/glx#1224)
- `invalid-entity-ids/` - An entity ID containing underscores
- `invalid-relationship-participants/` - A relationship participant naming a person that does not exist
- `missing-required-fields/` - One entity per type, each missing a required field
- `place-coords-half-set/` - Places with only latitude or only longitude
- `temporal-properties-malformed/` - A temporal property item without `value`

## Valid Archives

- `minimal-example/` - Minimal valid archive with one person and required vocabularies
- `assertion-unknown-property/` - An assertion whose property is not in the vocabulary must still validate. It should also warn once genealogix/glx#1224 lands
- `missing-vocabularies/` - No vocabularies at all: the standard vocabularies fill in
- `unknown-person-properties/` - Person properties not in the vocabulary validate, with an "unknown property" warning each
- `place-coords-both-set/`, `place-coords-neither-set/` - Places with both coordinates and with none
- `temporal-properties-simple/`, `temporal-properties-list/` - Temporal properties in both forms
- `assertion-with-participant.glx`, `person-with-properties.glx`, `overlapping-parent-families.glx` - Single-file archives

`TestValidFixturesWithWarnings` checks the three warning cases above.
