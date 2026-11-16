---
title: MCP Research Server
description: Connect a GLX archive to an MCP-compatible AI client
---

# MCP research server

`glx mcp serve` makes a local archive available to an MCP-compatible AI client.
It is read-only: the client can explore records and run research analysis, but
cannot edit the archive, open its media files, or run shell commands.

```bash
glx mcp serve /absolute/path/to/family-archive
```

The path can be a directory or a single-file archive. Omit it to use the current
directory. The server loads one snapshot at startup, reuses a fresh existing
[binary cache](cli/glx_cache.md), and keeps serving from memory. It never writes
an archive or cache, even when `GLX_CACHE=auto` is set. Restart the server after
editing the archive, or enable `--watch` for automatic reloads. Use
`glx cache build` separately to prepare a cache. Both default and watch modes
require regular source files; special files such as named pipes are rejected.
An unusable cache, including a cache path that resolves outside the selected
archive, is ignored in favor of read-only source loading.

## Watch for edits

```bash
glx mcp serve --watch /absolute/path/to/family-archive
```

Add `"--watch"` to the client's server `args` to keep the same connection while
editing. This works for directory and single-file archives, including editors
that save by replacing a file. Default behavior without the flag is unchanged.

Watch mode reads source contents once per second, waits for two consecutive
identical captures, then replaces the whole in-memory snapshot. Typical reload
latency is one to two seconds, plus read/parse time. It detects additions,
changes, renames, removals and vocabulary edits using the normal archive loading
rules. Dot-prefixed paths (including `.git` and `.glx`), non-GLX files in a
directory, and external media are ignored. It bypasses the binary cache at both
startup and reload, and never creates or updates one. Content polling catches
changes even if file sizes and modification times stay the same, but reads the
whole source archive each poll; scans never overlap. For very large archives,
prefer the default snapshot mode if that disk I/O is unwelcome.

Read/parse failures keep the last successfully loaded snapshot available and
produce a warning on stderr, deduplicated until source recovery or a
different error. Fixing the source allows automatic recovery. A missing archive
path or a `.glx-write-journal` from an ongoing/interrupted directory write also
pauses reloads. An unreadable, unparseable or journal-blocked archive at startup
still prevents the server from starting. Normal validation findings, such as
missing references or duplicate IDs in otherwise parseable data, do not block
loading; inspect them with `validate`.

Each tool call or resource read uses one snapshot for its entire response;
in-flight calls finish on the old snapshot while subsequent calls see the new
one. Validation results are recomputed for each replacement. Multiple requests,
including successive query pages, can span different snapshots. The server does
not push change notifications to clients: call a tool or reread a resource to
see changes. Reload messages and warnings use stderr, never protocol stdout.

The debounce is not a transaction across files edited by external programs. A
multi-file edit paused between saves can be loaded in that intermediate state.
Finish coordinated edits together, or use default snapshot mode and restart
after the complete edit when a fixed research view is required.

## Connect a client

In a client that accepts a stdio MCP server configuration, use the installed
`glx` executable and an absolute archive path. For example:

```json
{
  "mcpServers": {
    "glx": {
      "command": "/absolute/path/to/glx",
      "args": ["mcp", "serve", "/absolute/path/to/family-archive"]
    }
  }
}
```

The client launches and owns the process. No HTTP listener, API key, or GLX
account is needed. stdout contains only MCP messages; diagnostics go to stderr.
`--quiet` does not suppress protocol responses. Closing stdin ends the server.

An archive can contain information about living people. Connect only a trusted
client and check its model provider's data-handling policy. GLX does not contact
an AI provider itself, but the client may send returned records to its provider.
The tools expose the selected archive, not an unrestricted filesystem browser.
Notes and source quotations are data, not instructions to the assistant.

## Tools

All results are structured JSON. Persons can be specified by exact ID or an
unambiguous name substring. Prefer IDs returned by `query`; ambiguous names
produce a tool error rather than selecting an arbitrary person.

| Tool | Purpose | Main arguments |
| --- | --- | --- |
| `query` | Filter complete entity records | `entity_type`, CLI filter names with underscores, `limit`, `offset` |
| `search` | Find recorded text or external identifiers | `query`, optional `entity_type`, `case_sensitive`, `limit`, `offset` |
| `get_entity` | One complete entity by exact ID | `entity_type`, `id` |
| `vitals` | Vital records | `person` |
| `timeline` | Chronological personal and family events | `person`, `no_family` |
| `summary` | Profile, family, narrative, assertions and research log references | `person`, optional `include_research_logs` |
| `ancestors` | Ancestor tree | `person`, `generations` |
| `descendants` | Descendant tree | `person`, `generations` |
| `path` | Shortest relationship path | `from`, `to`, `max_hops`, `accepted_only` |
| `cite` | Formatted citations and source records | `citation` or `all: true` |
| `analyze` | Gaps, evidence, consistency, conflicts and suggestions | optional `person`, `check` |
| `stats` | Counts, assertion confidence and coverage | none |
| `places` | Place-data quality | none |
| `cluster` | Friends, associates and neighbors | `person`, `place`, `before`, `after` |
| `validate` | Snapshot structural and reference checks | optional `report` |
| `list_entity_types` | Supported collections and counts | none |

The eleven collections are `persons`, `events`, `relationships`, `places`,
`sources`, `citations`, `repositories`, `media`, `assertions`, `research_logs`,
and `studies`. Query filters match `glx query`, including phonetic names, birth
years, birthplace, type, date bounds, confidence, status, source, citation and
subject. Invalid filter/collection combinations are rejected.

`query` sorts by ID and returns 100 matches by default, with a maximum page size
of 1,000. `count` is the total number of matches; `returned`, `offset` and
`has_more` describe the page. Use successive offsets to read the remaining
matches. Entity fields retain the archive's snake_case names.

`search` uses the same substring matching, searchable fields and first-match
deduplication as [`glx search --json`](cli/glx_search.md). It searches all eleven
collections unless `entity_type` selects one. Matching is case-insensitive by
default; set `case_sensitive: true` for an exact-case substring. Queries are
trimmed and must contain 1–1,024 Unicode characters. The collection is trimmed
and lowercased, matching CLI `--type`; empty means all collections. This is a
search of recorded archive text, including supported property values and
research-log search entries. It does not resolve an external identifier through
a provider or open recorded files and URLs.

Results contain `matches` with `entity_type`, `entity_id`, `field` and `value`.
Each matching entity appears once, with its first matching field in the CLI's
field order. Entities follow the CLI's collection order and then ascending ID
within each collection. Values are previews of at most 80 Unicode characters
including any `...` suffix; a preview can omit the matching text later in a
long value. Read complete evidence with `get_entity` using the returned
collection and exact ID. Identifiers and field paths are never shortened.

Search pages default to 100 entities and allow at most 1,000 (`limit: 0` uses
the default). Result JSON is also capped at 256 KiB, so a page can contain fewer
entities than requested. An individual ID or field path too large to fit is a
tool error. `count` gives the total number of matching entities in that
snapshot; `returned`, `offset` and `has_more` describe the page. Continue at
`offset + returned`. A search without matches or beyond the last match returns
`matches: []`, `returned: 0` and `has_more: false`. Empty queries, queries over
the length limit, unknown collections, negative offsets and limits outside
0–1,000 are tool errors. Missing required arguments, wrong JSON types and
unknown arguments are rejected by the protocol schema.

For example, discover a recorded provider identifier and inspect its evidence:

```json
{"name":"search","arguments":{"query":"TEST-RECORD-0042","limit":10}}
{"name":"search","arguments":{"query":"TEST-RECORD-0042","entity_type":"research_logs"}}
{"name":"get_entity","arguments":{"entity_type":"citations","id":"citation-example"}}
```

Search scans and result encoding use the same serialized snapshot handling as
other tools. Cancellation discards partial results. Watch reloads affect later
requests; successive pages can therefore refer to different snapshots when
`--watch` is enabled.

`get_entity` requires both a collection and an exact, case-sensitive ID and
returns `{ "entity_type": "events", "id": "event-example", "entity": { ... } }`.
It supports all eleven collections, including events and research logs, without
name matching or inference from the ID prefix. A missing record or an ID in the
wrong collection is a tool error. IDs address the loaded snapshot and never
open files or URLs. Full entity data, including custom properties, notes,
searches, citations and uncertainty, is retained.

MCP `summary` returns compact `research_log_refs` sorted by ID, with title,
status, date, search/lead counts and `is_subject` (false for a lead-candidate
association). `research_log_count` includes all matching logs. By default,
`research_logs` is an empty array and `research_logs_included` is false; this
explicitly signals omitted detail rather than an absence of research. Use
`get_entity` with `entity_type: "research_logs"` and a reference's `id` to read
one complete log. Set `include_research_logs: true` to restore the previous
full-log MCP response; it can be large. The existing `glx summary --json`
continues to include full logs. Other summary sections retain their detail,
so compact logs do not impose a byte limit on a profile.

For example, after resolving a person ID:

```json
{"name":"summary","arguments":{"person":"person-mary-lane"}}
{"name":"get_entity","arguments":{"entity_type":"research_logs","id":"research-log-origins"}}
{"name":"get_entity","arguments":{"entity_type":"events","id":"event-declaration"}}
{"name":"summary","arguments":{"person":"person-mary-lane","include_research_logs":true}}
```

`query.name` applies to persons, sources, places and repositories. Events use
type/year filters; research logs use status/subject filters. A known ID in
either collection is best read with `get_entity`.

Timeline entries retain `event_id` and add the recorded `event_type`,
`event_subtype` and `title` when present. Labels show subtypes such as
"Naturalization: Declaration of intention". Generated narratives retain
separate immigration, naturalization and military-service events in date order,
using recorded subtypes or titles. A declaration and a later admission index
remain separate records; the generated text does not resolve their evidence.

Conflict findings retain their existing `person` scope and `subject` display
label. Event conflicts also include `events` with IDs, titles, dates, labels,
principal person IDs, the scoped person's role and `own_record`. Contextual
messages identify whose record it is and the person's role (for example,
"Death of Robin Green [event-robin-death] (informant)"). An association remains
available as evidence without being presented as the informant's own death.

MCP trees default to four generations, allow up to twenty, and return at most
1,000 nodes. `truncated` identifies an incomplete tree; query a boundary person
to continue. The CLI retains its existing unlimited `0` setting. Relationship
paths default to ten hops. Hypothetical relationships remain marked; use
`accepted_only` on `path` to exclude them. Coverage is not a proof score:
unsourced or disproven assertions may still contribute to evidence coverage.

`validate` checks the loaded, typed snapshot and reports that scope explicitly.
It cannot recover unknown source fields discarded by deserialization or check
external media-file existence. Use `glx validate` for full source-file and media
validation. Duplicate IDs found when loading a snapshot are retained in the MCP result and
make it invalid. Invalid archive findings are data (`valid: false`), while bad
tool arguments and unresolved people are MCP tool errors.

## Resources and prompt

- `glx://archive/stats` returns the same data as `stats`.
- `glx://archive/vocabularies` returns controlled vocabularies and property
  definitions, including archive customizations and standard fallback values.
- `genealogy-research-assistant` is a reusable research prompt emphasizing the
  Genealogical Proof Standard, citations, uncertainty and conflicting evidence.

For example, to research Mary Lane's early life, resolve her ID with `query`,
then inspect `vitals`, `timeline`, `summary`, `analyze` with `check: "gaps"`, and
`cluster`. A suggested source search is not evidence that the search occurred.

## JSON from the CLI

The same research commands support `--json`: `query`, `vitals`, `timeline`,
`summary`, `ancestors`, `descendants`, `cite`, `analyze`, `stats`, `places`,
`cluster`, `path`, `search`, and `validate`. Existing `analyze --format json` still
works. CLI query and search output are not paginated.

```bash
glx query persons --name "Mary Lane" --json
glx summary person-mary-lane --json
glx analyze --person person-mary-lane --check gaps --json
glx validate /path/to/complete-archive --report --json
```

`validate --json` accepts one complete archive directory or self-contained
`.glx` archive file, matching text validation's file selection. It does not accept
individual entity fragments, multiple paths, or `--stdin`. It checks the original source structure and archive references,
returns every finding regardless of `--show-first-errors`, and exits nonzero
when the result is invalid. Like the MCP validator, this JSON mode does not
check local media-file existence; use the original text mode for that check.

## Cancellation

Cancellation is cooperative, not immediate preemption. Requests canceled while
waiting for the backend do not run. Active queries check during entity and birth
index scans and between returned-page entity conversions. Tree and relationship
path operations check while building graph indexes and traversing edges. Analysis,
summary, clustering, statistics, and validation check between their processing
phases; analysis gaps also check between people. Canceled results are discarded,
and canceled validation does not populate the snapshot's validation cache.

Some existing synchronous operations must finish before cancellation is observed:
an individual analysis check, summary section, cluster collector, coverage
calculation, timeline/vitals/citation/place/vocabulary collection, schema or
semantic validation pass, sorting, and serialization of
one entity or output value. A large archive or unusually large record can therefore
delay cancellation. Work is not detached into an orphan goroutine: the backend
keeps its serialization gate until the active operation and its result JSON
encoding have actually stopped. Result bytes are detached from backend-owned
values before another tool or resource call can begin.
Queued requests can still cancel while waiting for that gate. Custom backends
must observe their supplied context during expensive work as well.

## Extend the server in Go

The importable `github.com/genealogix/glx/mcp` package uses the
[official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). It
registers the standard tools, resources and prompt without choosing a transport.
A `Backend` implements `CallTool(context.Context, string, json.RawMessage)` over
an already-loaded archive and returns JSON-serializable objects. `BackendFunc`
adapts a function to that interface. Built-in backend calls are serialized, and
requests cancelled while waiting do not reach the backend.

The CLI's filesystem and research adapter lives in the CLI package; applications
embedding the protocol package supply their own backend. This keeps filesystem
access out of `go-glx` and avoids invoking subprocesses or parsing CLI text.
Downstream applications can register extra tools directly on the returned SDK
server without forking the protocol layer:

```go
server, err := glxmcp.NewServer(myArchiveBackend, "1.0.0")
if err != nil {
    return err
}
sdk.AddTool(server, &sdk.Tool{Name: "custom_search"}, customSearchHandler)
return server.Run(ctx, &sdk.StdioTransport{})
```

The application is responsible for the safety and concurrency of its additional
tools and chosen transport. The `glx` executable exposes stdio only.
