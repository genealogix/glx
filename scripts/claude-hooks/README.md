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

A `PreToolUse` hook — `matcher: "Bash"` with the handler gated by `if: "Bash(gh api:*)"`
so it spawns only for `gh api` commands — that is a **pure restrictor**: it never
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
  so parameterized read queries stay prompt-free. **Consequence:** a read whose *output* is
  captured via command substitution — e.g. `ISSUE_ID=$(gh api graphql -f query='…')`, the
  capture idiom the PR-review workflow uses — contains a `$(` and therefore prompts, even
  though the inner query is read-only. This is deliberate conservatism, and the prompt is a
  one-keystroke approval; pipe to a file or run the bare query if you want it prompt-free.
- **Fail-closed.** Any parse error, malformed input, or internal exception resolves to
  `ask`, never to silent approval. The [`gh-api-gate.sh`](gh-api-gate.sh) wrapper emits
  `ask` if no Python interpreter is available, and `.claude/settings.json` blocks (`exit 2`)
  if the wrapper itself is missing. The gate restricts an otherwise auto-approved call by
  emitting a `PreToolUse` `permissionDecision` of `ask`/`deny`, which relies on a hook
  decision taking precedence over a matching allow rule — the documented purpose of
  `PreToolUse` decisions, and verified live against the `Bash(gh api …:*)` allow rules.

### Scope boundary

The allow-list rules and this hook govern only commands that **begin with `gh api`**. Wrapping
a `gh api` call inside `bash -c '…'`, `sh -c`, `eval`, or `xargs` does not match either
allow rule, so such commands are never auto-approved — they fall through to a normal
permission prompt regardless of this gate.

### Run the tests

```bash
python scripts/claude-hooks/gh-api-gate_test.py
```

## `pre-commit-golangci.sh` — pre-commit lint gate (genealogix/glx#869)

### Why

`#655` added a `PreToolUse` hook intended to run `golangci-lint` before a Claude
Code session creates a `git commit`. Its matcher was `"Bash(git commit*)"`, which —
because it contains `(`, `)`, `*` — is compiled as a JavaScript regex and tested
against the **tool name** (`"Bash"`), not the command. `"Bash"` never matches that
regex, so the hook was dead code and the Claude-side lint pass never ran (#869).

A matcher filters on tool name only; the command pattern belongs in the per-handler
`if` field, which uses [permission-rule syntax](https://code.claude.com/docs/en/permissions):

```jsonc
{
  "matcher": "Bash",                      // tool name (exact match)
  "hooks": [{
    "type": "command",
    "if": "Bash(git commit:*)",           // command pattern (permission-rule syntax)
    "command": "bash -c 'bash \"${CLAUDE_PROJECT_DIR:-.}/scripts/claude-hooks/pre-commit-golangci.sh\"'"
  }]
}
```

`Bash(git commit:*)` is a literal-prefix match, so it also matches the rarely-used
`git commit-tree` / `git commit-graph`. That over-fire is harmless here: the hook is
advisory and the script no-ops unless the index stages a `*.go` file.

### What it does

When a `git commit` stages Go source, it runs `golangci-lint` scoped to the new
code (`--new-from-rev=$(git merge-base HEAD main)`, falling back to `HEAD~1`) and
prints any findings. It is **advisory only**: findings surface via golangci-lint's
exit code, which Claude Code treats as a *non-blocking* error for `PreToolUse`
(only `exit 2` blocks a tool call), so the commit still proceeds. `lefthook`
(`#280`) remains the effective local gate. Whether to promote this to a hard,
blocking gate — and how it should relate to the lefthook and CI lint passes — is
the strategy decision tracked in `#870`; this script deliberately stays advisory.

### Run it standalone

```bash
bash scripts/claude-hooks/pre-commit-golangci.sh
```

It no-ops (exit 0) unless the index stages a `*.go` file, so it is safe to run any
time. A regression test that exercises the hook end-to-end (matcher firing, exit
semantics) depends on the eval-harness infrastructure tracked in `#796`.
