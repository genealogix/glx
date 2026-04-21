# specification/ — Claude Guide

## Internal Links

Omit `.md` file extension for VitePress compatibility:

- Good: `[Person Entity](4-entity-types/person)`
- Bad: `[Person Entity](4-entity-types/person.md)`

## Vocabulary Files

Vocabularies are `.glx` YAML files in `5-standard-vocabularies/`. They define controlled terms for event types, relationship types, place types, source types, etc.

## Adding a New Entity Type

1. Define the Go type in `go-glx/types.go`
2. Add the new collection to the `GLXFile` struct in `go-glx/types.go`
3. Update the YAML serializer (`go-glx/serializer.go`) and any entity-aware helpers
4. Add a vocabulary file in `specification/5-standard-vocabularies/` if the type needs controlled values (e.g., `<entity>-types.glx`, `<entity>-properties.glx`)
5. Add the JSON schema in `specification/schema/v1/<entity>.schema.json` and register it in `specification/schema/v1/glx-file.schema.json`
6. Add a new entity spec at `specification/4-entity-types/<entity>.md`, add a card to `specification/4-entity-types/README.md`, and link from `specification/6-glossary.md`
7. Update `CHANGELOG.md` under the Unreleased → `### Added` section with an issue/PR reference
