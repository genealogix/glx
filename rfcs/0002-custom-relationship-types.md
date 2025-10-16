# RFC-0002: Custom Relationship Types

- **Start Date**: 2025-03-10
- **RFC PR**: #2
- **Tracking Issue**: #42
- **Status**: Discussion

## Summary

Introduce configurable relationship types to support chosen family, guardianship, and cultural kinship models.

## Motivation

Default relationship catalog is insufficient for diverse family structures.

## Guide-level Explanation

Archives can register custom relationship types via configuration referenced by relationship entities.

## Reference-level Explanation

Adds `schema/v1/config/relationship-types.schema.json` and updates relationship entity validation to reference config entries.

## Drawbacks

Increases configuration complexity.

## Rationale and Alternatives

Evaluated embedding relationship metadata directly in entity; rejected for reuse and validation clarity.

## Prior Art

GEDCOM custom tags; various genealogy tools allow custom relationships.

## Unresolved Questions

How to manage clashes between shared custom catalogs.

## Future Possibilities

Schema registry for sharing relationship catalogs across archives.


