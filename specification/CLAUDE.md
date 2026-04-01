# specification/ — Claude Guide

## Internal Links

Omit `.md` file extension for VitePress compatibility:
- Good: `[Person Entity](4-entity-types/person)`
- Bad: `[Person Entity](4-entity-types/person.md)`

## Vocabulary Files

Vocabularies are `.glx` YAML files in `5-standard-vocabularies/`. They define controlled terms for event types, relationship types, place types, source types, etc.

## Adding a New Entity Type

1. Define type in `go-glx/types.go`
2. Add to `GLXFile` struct
3. Update serializer
4. Add vocabulary if needed
5. Add JSON schema in `schema/v1/`
6. Update documentation
