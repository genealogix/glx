# `tools/fragmentcheck` — changelog fragment issue-reference gate

Asserts that every [changie](https://changie.dev) changelog fragment under
`.changes/unreleased/` carries an issue/PR reference in its top-level
`custom.Issue` field — the field `changeFormat` renders into `CHANGELOG.md`
(#858, implements #321).

`changie new` makes the field required, so this is the backstop for fragments
hand-written without it. It runs from
[`scripts/check-changelog-fragments.sh`](../../scripts/check-changelog-fragments.sh)
(via `make changelog-check` and the `changelog-fragments` CI workflow), after
changie's own parse/render pass.

## Usage

```bash
go run ./tools/fragmentcheck                    # defaults to .changes/unreleased
go run ./tools/fragmentcheck path/to/fragments  # explicit directory
```

Exit code is 1 when any fragment fails, with one
`::error file=…::` GitHub Actions annotation per offender so failures surface
inline on the PR diff.

## Why a YAML parser and not a text pattern

The gate has to satisfy two requirements at once, and text matching can only
get one of them right at a time:

- An `Issue:`-looking line inside the `body: |-` block scalar is body text, not
  a field, and must **not** satisfy the gate.
- Every spelling YAML permits for the real field must be accepted: block map,
  inline flow map (`custom: {Issue: "#123"}`), quoted or unquoted keys, any key
  order.

Parsing the fragment gives both for free. The awk gate this replaced scoped its
search to lines indented under a `custom:` line, which got the first
requirement right but rejected valid inline flow maps.

## Tests

`go test ./tools/fragmentcheck/` covers the accepted and rejected shapes, and
runs the gate over the fragments actually committed in this repository so a
fragment that would fail CI fails the unit test too.
