# File Structure

This section describes the recommended layout of a GENEALOGIX archive.

## Repository Layout

See the root `README.md` for a high-level overview and the `examples/` for working archives.

## Naming Conventions

- Person IDs: `person-` followed by 8 lowercase hex characters
- Relationship IDs: `rel-` followed by 8 lowercase hex characters
- Schema version: `{major}.{minor}` (e.g., `1.0`)

## File Organization Patterns

- Place persons under `persons/` as `{person-id}.glx`
- Place relationships under `relationships/` as `{relationship-id}.glx`
- Keep assertions grouped by domain within `assertions/`


