# GLX CLI Reference

The official command-line tool for working with [GENEALOGIX (GLX)](/) family archives. Use `glx` to initialize archives, import GEDCOM files, validate data quality, query entities, and analyze relationships.

The per-command pages linked below are auto-generated from the live Cobra command tree by `make docs-cli`. To change a command's documentation, edit its `Use`/`Short`/`Long`/`Example` strings in [`glx/cli_commands.go`](https://github.com/genealogix/glx/blob/main/glx/cli_commands.go) (or its `*_runner.go` file) and re-run the target. CI fails on any drift between the source command tree and the committed pages.

For installation instructions, see [`glx/README.md`](https://github.com/genealogix/glx/blob/main/glx/README.md). For a guided walkthrough, see the [Hands-On CLI Guide](/guides/hands-on-cli-guide).

## Commands

### Archive Management

- [`glx init`](/cli/glx_init) — initialize a new archive
- [`glx validate`](/cli/glx_validate) — validate files and cross-references
- [`glx split`](/cli/glx_split) — convert a single-file archive to multi-file
- [`glx join`](/cli/glx_join) — convert a multi-file archive to single-file
- [`glx merge`](/cli/glx_merge) — combine two archives with duplicate detection
- [`glx migrate`](/cli/glx_migrate) — migrate an archive to the current format
- [`glx rename`](/cli/glx_rename) — rename an entity by ID

### Import & Export

- [`glx import`](/cli/glx_import) — import a GEDCOM file
- [`glx export`](/cli/glx_export) — export to GEDCOM or Schema.org-aligned JSON-LD

### Exploration

- [`glx search`](/cli/glx_search) — full-text search across entities
- [`glx query`](/cli/glx_query) — filter and list entities
- [`glx vitals`](/cli/glx_vitals) — show birth, death, burial for a person
- [`glx timeline`](/cli/glx_timeline) — chronological events for a person
- [`glx summary`](/cli/glx_summary) — full person profile with narrative
- [`glx ancestors`](/cli/glx_ancestors) — ancestor tree
- [`glx descendants`](/cli/glx_descendants) — descendant tree
- [`glx cite`](/cli/glx_cite) — formatted citation text
- [`glx path`](/cli/glx_path) — shortest relationship path between two people

### Data Entry

- [`glx census`](/cli/glx_census) — census tooling (see subcommands)
- [`glx census add`](/cli/glx_census_add) — generate entities from a census template

### Analysis

- [`glx stats`](/cli/glx_stats) — entity-count and confidence dashboard
- [`glx places`](/cli/glx_places) — place data quality issues
- [`glx cluster`](/cli/glx_cluster) — FAN-club analysis
- [`glx analyze`](/cli/glx_analyze) — gap, conflict, and suggestion analysis
- [`glx duplicates`](/cli/glx_duplicates) — detect duplicate entities
- [`glx coverage`](/cli/glx_coverage) — research coverage report
- [`glx diff`](/cli/glx_diff) — diff two archives

### Shell completion

- [`glx completion`](/cli/glx_completion) — generate shell completion scripts (bash, zsh, fish, powershell)
