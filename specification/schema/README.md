---
title: GENEALOGIX JSON Schemas
description: JSON Schema definitions for validating GENEALOGIX archives
layout: doc
---

# GENEALOGIX JSON Schemas

This directory contains JSON Schema definitions for validating
GENEALOGIX archives.

## Schema Dialect

All schemas declare [JSON Schema 2020-12](https://json-schema.org/draft/2020-12)
(`"$schema": "https://json-schema.org/draft/2020-12/schema"`) — see
[ADR-0007](/decisions/0007-json-schema-2020-12). Subschemas live
under `$defs`, and property co-requirements use `dependentRequired`. Use a
2020-12-capable validator; draft-07 idioms (`definitions`, array-form
`dependencies`) are not accepted by the meta-schema.

## Usage

### With ajv (JavaScript)

```javascript
// The default `ajv` export is draft-07-only; use the 2020-12 dialect class.
const Ajv2020 = require('ajv/dist/2020');
const ajv = new Ajv2020();

const schema = require('./v1/person.schema.json');
const data = require('./persons/person-123.glx');

const valid = ajv.validate(schema, data);
if (!valid) console.log(ajv.errors);
```

### With jsonschema (Python)

```python
from jsonschema import validate
import yaml, json

with open('schema/v1/person.schema.json') as f:
    schema = json.load(f)

with open('persons/person-123.glx') as f:
    data = yaml.safe_load(f)

validate(instance=data, schema=schema)
```

### With glx CLI

```bash
glx validate persons/person-123.glx
glx validate --all
```

## Schema Versioning

Schemas follow the format version of the specification:

- `v1/` - Version 1.x schemas
- `v2/` - Version 2.x schemas (future)

### Breaking vs Non-Breaking Changes

Non-breaking (minor version bump):

- Adding optional fields
- Adding new enum values
- Relaxing validation rules

Breaking (major version bump):

- Removing fields
- Changing required fields
- Changing field types
- Restricting validation rules

## Schema References

All schemas use `$id` URIs that resolve to GitHub raw content. Examples of schema URIs:

```text
https://raw.githubusercontent.com/genealogix/glx/main/specification/schema/v1/person.schema.json
https://raw.githubusercontent.com/genealogix/glx/main/specification/schema/v1/relationship.schema.json
https://raw.githubusercontent.com/genealogix/glx/main/specification/schema/v1/assertion.schema.json
```

These URIs directly serve the schema files from the main branch. Additional schemas follow the same pattern (e.g., `event.schema.json`, `source.schema.json`, `citation.schema.json`, etc.).

## Custom Extensions

Archives can extend the base format using archive-owned vocabularies. See [Core Concepts](../2-core-concepts#archive-owned-vocabularies) for details.
