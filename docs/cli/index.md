# GLX CLI Reference

The official command-line tool for working with [GENEALOGIX (GLX)](../../README.md) family archives. Use `glx` to initialize archives, import GEDCOM files, validate data quality, query entities, and analyze relationships.

The per-command pages linked below are auto-generated from the live Cobra command tree by `make docs-cli`. To change a command's documentation, edit its `Use`/`Short`/`Long`/`Example` strings in [`glx/cli_commands.go`](https://github.com/genealogix/glx/blob/main/glx/cli_commands.go) (or its `*_runner.go` file) and re-run the target. CI fails on any drift between the source command tree and the committed pages.

For installation instructions, see [`glx/README.md`](https://github.com/genealogix/glx/blob/main/glx/README.md). For a guided walkthrough, see the [Hands-On CLI Guide](../guides/hands-on-cli-guide.md).

## Commands

### Archive Management

- [`glx init`](glx_init.md) — initialize a new archive
- [`glx validate`](glx_validate.md) — validate files and cross-references
- [`glx split`](glx_split.md) — convert a single-file archive to multi-file
- [`glx join`](glx_join.md) — convert a multi-file archive to single-file
- [`glx merge`](glx_merge.md) — combine two archives with duplicate detection
- [`glx migrate`](glx_migrate.md) — migrate an archive to the current format
- [`glx rename`](glx_rename.md) — rename an entity by ID

### Import & Export

- [`glx import`](glx_import.md) — import a GEDCOM file
- [`glx export`](glx_export.md) — export to GEDCOM or Schema.org-aligned JSON-LD

### Exploration

- [`glx search`](glx_search.md) — full-text search across entities
- [`glx query`](glx_query.md) — filter and list entities
- [`glx vitals`](glx_vitals.md) — show birth, death, burial for a person
- [`glx timeline`](glx_timeline.md) — chronological events for a person
- [`glx summary`](glx_summary.md) — full person profile with narrative
- [`glx ancestors`](glx_ancestors.md) — ancestor tree
- [`glx descendants`](glx_descendants.md) — descendant tree
- [`glx cite`](glx_cite.md) — formatted citation text
- [`glx path`](glx_path.md) — shortest relationship path between two people
- [`glx serve`](glx_serve.md) — local browser-based read-only archive viewer

### Data Entry

- [`glx census`](glx_census.md) — census tooling (see subcommands)
- [`glx census add`](glx_census_add.md) — generate entities from a census template

### Analysis

- [`glx stats`](glx_stats.md) — entity-count and confidence dashboard
- [`glx places`](glx_places.md) — place data quality issues
- [`glx cluster`](glx_cluster.md) — FAN-club analysis
- [`glx analyze`](glx_analyze.md) — gap, conflict, and suggestion analysis
- [`glx duplicates`](glx_duplicates.md) — detect duplicate entities
- [`glx coverage`](glx_coverage.md) — research coverage report
- [`glx diff`](glx_diff.md) — diff two archives

### Shell completion

- [`glx completion`](glx_completion.md) — generate shell completion scripts (bash, zsh, fish, powershell)
