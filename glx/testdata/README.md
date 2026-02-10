# Test Data for GLX Validator

This directory contains test data for validating the GLX specification validator.

## Directory Structure

- `valid/` - Valid archives that should pass validation
- `invalid/` - Invalid archives that should fail validation

## Test Cases

### Valid Archives

- `minimal-example/` - Minimal valid archive with one person and required vocabularies

### Invalid Archives

- `missing-vocabularies/` - Archive missing required property vocabularies (should warn about unknown properties)
- `broken-references/` - Archive with invalid entity references (non-existent persons, citations, sources)
- `invalid-properties/` - Archive with properties that don't exist in vocabularies
- `invalid-entity-ids/` - Archive with entity IDs that contain invalid characters (underscores)
- `duplicate-ids/` - Archive with duplicate entity IDs across files
- `invalid-relationship-participants/` - Archive with relationship participants referencing non-existent persons
- `invalid-assertion-properties/` - Archive with assertions using unknown property names

## Existing Test Files

The following files were copied from the original testdata directory and are used by existing unit tests:

- `valid/assertion-with-participant.glx`
- `valid/person-with-properties.glx`
- `invalid/assertion-participant-and-property.glx`
- `invalid/assertion-participant-and-value.glx`
- `invalid/assertion-participant-invalid-person.glx`
- `invalid/assertion-participant-invalid-role.glx`
- `invalid/assertion-unknown-property.glx`

## Usage

These test cases are used by the Go test suite to ensure the validator correctly identifies and reports validation issues.
