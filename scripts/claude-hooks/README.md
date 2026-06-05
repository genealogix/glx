# Claude Code hooks

In-repo scripts invoked by the `hooks` block of [`.claude/settings.json`](../../.claude/settings.json).
Keeping them here (rather than inlined in the JSON) makes the non-trivial logic reviewable and testable.

## `gh-api-gate.py` — `gh api` mutation gate (genealogix/glx#912)

### Why

`.claude/settings.json` auto-approves two `gh api` prefixes so routine maintenance
(the PR-review workflow, issue/label queries) doesn't prompt on every call:

```jsonc
"Bash(gh api graphql:*)",
"Bash(gh api repos/genealogix/glx:*)",
```

A prefix matcher can't tell a read query from a mutation. On their own these rules
silently auto-approve destructive calls — the bypass a pre-merge review found in
[#911](https://github.com/genealogix/glx/pull/911):

```bash
gh api graphql -f query='mutation { deleteRef(...) }'      # any GraphQL mutation
gh api repos/genealogix/glx/issues/comments/123 -X DELETE  # REST DELETE
```

so they were dropped, which also made every read-only query prompt again. This hook
is the granular gate the prefix matcher can't be, so the rules can be re-enabled safely.

### What it does

A `PreToolUse` hook (matcher `Bash(gh api*)`) that is a **pure restrictor** — it never
emits `allow`, only subtracts the dangerous subset of what the allow-list grants:

| `gh api` call | Verdict | Effect |
|---|---|---|
| Confirmed read-only (GraphQL `query`/anonymous `{…}`; REST `GET`/`HEAD` or no body) | *(silent)* | allow-list rule auto-approves — **no prompt** |
| Any write/mutation, or anything not provably read-only (GraphQL `mutation`/`subscription`; REST `POST`/`PATCH`/`PUT`/`DELETE`; implicit POST from body fields; unverifiable/`--input`/`@file` body) | `ask` | confirmation prompt (the pre-#911 gate) |
| Write to `…/git/refs/…` (force-update or delete of a branch/tag pointer) | `deny` | hard block |

Because the hook only ever restricts, it cannot over-approve: the allow-list defines
what's auto-approved; the hook removes the dangerous subset. `ask` (not `deny`) is used
for ordinary writes so the documented review workflow — `resolveReviewThread`,
`updateIssueIssueType`, posting reply comments — stays doable *with a prompt*, exactly
as before #911. Only ref-tampering, which is irreversible and never part of a `gh api`
workflow, is hard-blocked.

### Correctness notes

- **Implicit POST.** `gh api <path> -f k=v` (or `-F`/`--field`/`--raw-field`/`--input`)
  sends a POST even with no `-X` flag. The gate treats a REST call with body fields and
  no explicit read method as a write — closing a hole the issue's original spec missed.
- **GraphQL operation type.** A document is read-only only if it defines **exactly one**
  operation and that operation is a query (named or anonymous `{…}`) — multi-operation
  documents are gated because an attached `operationName` can select a hidden mutation
  (the decoy-query bypass). Comments and string literals are stripped and only
  brace-depth-zero operation keywords count, so `mutation` inside a string or field name is
  inert. Any query carrying a `"""` block string is gated outright (its escape rules make
  static lexing unsafe).
- **Endpoint normalization.** Before the refs check the path is percent-decoded (repeatedly),
  has its `#fragment`/`?query` and `scheme://host` stripped, and its `.`/`..`/empty segments
  collapsed; the refs match is case-insensitive — so `%67it/refs`, `git%2Frefs`, `git/./refs`,
  `Git/Refs`, a fragment, or an absolute URL can't dodge the `deny`.
- **Shell parsing.** Tokenized with `shlex` (`commenters=''`, since a mid-word `#` is literal
  in bash and a URL fragment in endpoints); short-flag clusters are unpacked (`-iX DELETE`
  ≡ `-i -X DELETE`). Compound commands are split on shell operators; every `gh api` segment
  is classified and the most-restrictive verdict wins.
- **Shell-dynamic floor.** If the command contains expansion that could inject/rewrite
  arguments at runtime — a `$` or backtick outside single quotes, or an unquoted `{a,b}`
  brace expansion — a would-be read is floored to `ask` (the static parse can't see what it
  expands to). GraphQL `$variables` inside a single-quoted query are *not* treated as dynamic,
  so parameterized read queries stay prompt-free.
- **Fail-closed.** Any parse error, malformed input, or internal exception resolves to
  `ask`, never to silent approval. The [`gh-api-gate.sh`](gh-api-gate.sh) wrapper emits
  `ask` if no Python interpreter is available, and `.claude/settings.json` blocks (`exit 2`)
  if the wrapper itself is missing.

### Scope boundary

The allow-list rules and this hook govern only commands that **begin with `gh api`**. Wrapping
a `gh api` call inside `bash -c '…'`, `sh -c`, `eval`, or `xargs` does not match either
allow rule, so such commands are never auto-approved — they fall through to a normal
permission prompt regardless of this gate.

### Run the tests

```bash
python scripts/claude-hooks/gh-api-gate_test.py
```
