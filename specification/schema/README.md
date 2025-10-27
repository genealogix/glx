# GENEALOGIX JSON Schemas

This directory contains JSON Schema definitions for validating
GENEALOGIX archives.

## Usage

### With ajv (JavaScript)

```javascript
const Ajv = require('ajv');
const ajv = new Ajv();

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

All schemas use `$id` URIs that resolve to canonical versions:

```
https://schema.genealogix.org/v1/person
https://schema.genealogix.org/v1/relationship
https://schema.genealogix.org/v1/assertion
```

These URIs serve the actual schema files via HTTP.

## Custom Extensions

Archives can extend the base format using repository-owned vocabularies. See [Core Concepts](../2-core-concepts.md#repository-owned-vocabularies) for details.


