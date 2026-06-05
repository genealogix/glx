#!/usr/bin/env python3
"""PreToolUse gate for `gh api` Bash commands (genealogix/glx#912).

The repo's `.claude/settings.json` allow-list grants two prefix rules:

    Bash(gh api graphql:*)
    Bash(gh api repos/genealogix/glx:*)

A prefix matcher cannot tell a read query from a mutation, so on its own it
would silently auto-approve destructive calls (the bypass found in PR #911):

    gh api graphql -f query='mutation { deleteRef(...) }'      # any mutation
    gh api repos/genealogix/glx/issues/comments/123 -X DELETE  # REST DELETE

This hook is the granular gate the prefix matcher cannot be. It is a *pure
restrictor*: it never emits "allow". For a `gh api` invocation it emits

  * "deny" — for git-ref tampering (force-update/delete of branch/tag
    pointers), the one operation with catastrophic, irreversible blast radius
    and no routine `gh api` use case;
  * "ask"  — for every other write/mutation (GraphQL mutations, REST
    POST/PATCH/PUT/DELETE, including the *implicit* POST that `gh api` performs
    when body fields are present) and for anything it cannot positively verify
    is read-only;
  * nothing (exit 0, no decision) — for confirmed read-only calls, letting the
    allow-list rules auto-approve them without a prompt.

Because the hook only ever restricts, it cannot over-approve: the allow-list
defines what is auto-approved, and this hook subtracts the dangerous subset.
"Ask" preserves the pre-#911 behaviour (the prompt was already the correct gate
for writes); the only thing re-enabled is prompt-free *reads*.

Input : the PreToolUse hook JSON on stdin (uses `tool_input.command`).
Output: a PreToolUse `hookSpecificOutput` JSON decision on stdout, or nothing.

Fail-closed: any unexpected/unparseable condition resolves to "ask", never to
silent approval. (The settings.json wrapper also falls back to "ask" if this
script cannot be executed at all, e.g. no Python on PATH.)
"""

import json
import re
import shlex
import sys
import urllib.parse

# gh api flags that consume the following token as their value. Used to avoid
# mistaking a flag's value for the endpoint path. (gh api --help)
VALUE_FLAGS = {
    "-X", "--method",
    "-F", "--field",
    "-f", "--raw-field",
    "-H", "--header",
    "--hostname",
    "--input",
    "-q", "--jq",
    "-t", "--template",
    "--cache",
    "--preview",
}

# Flags that attach a request body. Their presence makes `gh api <path>` an
# implicit POST when no explicit --method is given.
BODY_FLAGS = {"-F", "--field", "-f", "--raw-field", "--input"}

# Field flags whose value can carry a key=value parameter (e.g. query=...).
FIELD_FLAGS = {"-F", "--field", "-f", "--raw-field"}

# Single-dash short flags: those that consume a value (mapped to their long
# form) and the only bare boolean one. Used to unpack glued clusters like
# `-iX DELETE` (= `-i -X DELETE`).
SHORT_VALUE_FLAGS = {
    "X": "--method", "F": "--field", "f": "--raw-field",
    "H": "--header", "q": "--jq", "t": "--template",
}
SHORT_BOOL_FLAGS = {"i"}  # -i/--include is gh api's only short boolean flag

READ_METHODS = {"GET", "HEAD"}

# Characters in an endpoint token that mean the path is computed by the shell at
# runtime (variable expansion, command substitution) and therefore cannot be
# statically verified — gate such calls.
_SHELL_DYNAMIC = ("$", "`")

# Shell tokens that terminate a simple command (start of a new pipeline stage,
# list element, subshell, or redirection target).
TERMINATORS = {
    "&&", "||", "|", "|&", "&", ";", ";;", "\n",
    "(", ")", "{", "}",
    "<", "<<", "<<<", ">", ">>", ">|", "&>", "&>>",
}

# Classification verdicts, ordered most-restrictive first.
DENY, ASK, READ = "deny", "ask", "read"


def _strip_graphql_noise(s):
    """Return the document with comments and string literals removed (replaced
    by spaces) so brace/keyword structure can be scanned without being fooled by
    text inside strings or `#` comments. Handles block strings (`\"\"\"…\"\"\"`),
    quoted strings with `\\` escapes, and line comments."""
    out = []
    i, n = 0, len(s)
    while i < n:
        c = s[i]
        if c == "#":
            while i < n and s[i] != "\n":
                i += 1
            continue
        if c == '"':
            if s[i:i + 3] == '"""':
                i += 3
                while i < n and s[i:i + 3] != '"""':
                    # `\"""` is the one block-string escape: it is literal
                    # content, not a terminator (GraphQL spec 2.9.4).
                    i += 4 if (s[i] == "\\" and s[i + 1:i + 4] == '"""') else 1
                i += 3
            else:
                i += 1
                while i < n and s[i] != '"':
                    i += 2 if s[i] == "\\" else 1
                i += 1
            out.append(" ")
            continue
        out.append(c)
        i += 1
    return "".join(out)


def _leading_keyword(seg):
    """First identifier in a chunk of top-level GraphQL text, lowercased, or
    None (e.g. the chunk before an anonymous query's `{` is blank)."""
    m = re.search(r"[A-Za-z_]\w*", seg)
    return m.group(0).lower() if m else None


def _graphql_is_readonly(body):
    """True only when the GraphQL document is *provably* a single read operation.

    A document is read-only iff it defines exactly one operation and that
    operation is a query (named or anonymous `{…}`), with no `mutation`/
    `subscription` operation anywhere. Multi-operation documents are gated even
    if every operation is a query, because an attached `operationName` can select
    any of them — the decoy-query-plus-real-mutation bypass. Comments and string
    literals are stripped first, and only brace-depth-zero operations count, so a
    `mutation` keyword inside a string, a field name, or a nested selection set
    cannot fool the scan, and a leading `fragment` definition is tolerated.
    """
    if '"""' in body:
        # Block strings have subtle escape rules (`\"""`) and can carry
        # unbalanced braces; rather than risk a mutation hiding behind a
        # mis-lexed block string, gate any document that uses one. Read queries
        # essentially never need block-string arguments.
        return False
    s = _strip_graphql_noise(body)
    depth = 0
    chunk = ""           # top-level text accumulated since the last block
    num_ops = 0
    saw_write = False
    saw_query = False
    saw_unknown = False
    for c in s:
        if c == "{":
            if depth == 0:
                kw = _leading_keyword(chunk)
                if kw in ("mutation", "subscription"):
                    saw_write = True
                    num_ops += 1
                elif kw == "query":
                    saw_query = True
                    num_ops += 1
                elif kw == "fragment":
                    pass  # a fragment definition, not an operation
                elif kw is None and chunk.strip(" \t\r\n,") == "":
                    saw_query = True   # blank before `{` => anonymous query
                    num_ops += 1
                else:                  # an unrecognised leading token => malformed
                    saw_unknown = True
                    num_ops += 1
                chunk = ""
            depth += 1
        elif c == "}":
            if depth > 0:
                depth -= 1
        elif depth == 0:
            chunk += c
    if saw_write or saw_unknown or num_ops != 1:
        return False
    return saw_query


def _parse_gh_api_args(tokens):
    """Parse the args of a single `gh api ...` command (tokens AFTER `gh api`).

    Returns {endpoint, method, has_body, field_values, has_unparsed,
    endpoint_dynamic}. `has_unparsed` flags constructs we cannot reason about and
    `endpoint_dynamic` flags an endpoint computed by the shell ($VAR/$()/backtick)
    — both make the caller fail closed."""
    state = {
        "endpoint": None, "method": None, "has_body": False,
        "field_values": [], "has_unparsed": False, "endpoint_dynamic": False,
    }
    n = len(tokens)
    i = 0

    def set_endpoint(tok):
        if state["endpoint"] is None and tok != "-":
            state["endpoint"] = tok
            if any(c in tok for c in _SHELL_DYNAMIC):
                state["endpoint_dynamic"] = True

    def apply_flag(name, inline_val, inline_present):
        """Apply one normalised long-form flag; consume the next token as its
        value when the flag needs one and no value was glued on. Returns the
        (possibly advanced) token index."""
        nonlocal i
        val = inline_val
        if name in BODY_FLAGS:
            state["has_body"] = True
        if name in VALUE_FLAGS and not inline_present:
            if i + 1 < n:
                i += 1
                val = tokens[i]
            else:
                state["has_unparsed"] = True
                val = None
        if name in FIELD_FLAGS:
            if val is not None:
                state["field_values"].append(val)
        elif name == "--method":
            if val is not None:
                state["method"] = val.strip("'\"").upper()
        return i

    while i < n:
        tok = tokens[i]

        if tok == "--":  # end of flags; the rest are positionals
            i += 1
            while i < n:
                set_endpoint(tokens[i])
                i += 1
            break

        if tok.startswith("--"):
            if "=" in tok:
                name, _, val = tok.partition("=")
                apply_flag(name, val, inline_present=True)
            else:
                apply_flag(tok, None, inline_present=False)
            i += 1
            continue

        if tok.startswith("-") and tok != "-":
            # Short-flag cluster, e.g. `-i`, `-XDELETE`, `-iX DELETE` (= -i -X …).
            cluster = tok[1:]
            j = 0
            while j < len(cluster):
                ch = cluster[j]
                if ch in SHORT_BOOL_FLAGS:
                    j += 1
                    continue
                if ch in SHORT_VALUE_FLAGS:
                    rest = cluster[j + 1:]
                    if rest:
                        apply_flag(SHORT_VALUE_FLAGS[ch],
                                   rest[1:] if rest.startswith("=") else rest,
                                   inline_present=True)
                    else:
                        apply_flag(SHORT_VALUE_FLAGS[ch], None,
                                   inline_present=False)
                    break  # a value flag consumes the remainder of the cluster
                state["has_unparsed"] = True  # unknown short flag
                break
            i += 1
            continue

        set_endpoint(tok)  # positional (bare "-" = stdin, ignored)
        i += 1

    return state


def _normalize_endpoint(ep):
    """Canonicalise an endpoint the way `gh`/GitHub's router effectively does, so
    obfuscated paths can't dodge `_path_is_refs`: percent-decode (repeatedly, to
    defeat double-encoding), drop any `#fragment`/`?query`, strip an absolute
    `scheme://host/` prefix, and collapse empty/`.`/`..` path segments."""
    if ep is None:
        return None
    for _ in range(3):
        dec = urllib.parse.unquote(ep)
        if dec == ep:
            break
        ep = dec
    ep = ep.split("#", 1)[0].split("?", 1)[0]
    m = re.match(r"^[A-Za-z][A-Za-z0-9+.\-]*://[^/]+/?(.*)$", ep)
    if m:
        ep = m.group(1)
    parts = []
    for seg in ep.split("/"):
        if seg in ("", "."):
            continue
        if seg == "..":
            if parts:
                parts.pop()
            continue
        parts.append(seg)
    return "/".join(parts)


def _path_is_refs(norm):
    """True for the GitHub git-refs endpoint family (branch/tag pointers).
    Case-insensitive: `git`/`refs`/`repos` route case-insensitively on GitHub."""
    if not norm:
        return False
    # Both the by-owner/repo path and the numeric `repositories/{id}` form.
    return re.search(
        r"(?:^|/)(?:repos/[^/]+/[^/]+|repositories/[^/]+)/git/refs(?:/|$)",
        norm, re.IGNORECASE) is not None


def _classify_gh_api(tokens):
    """Classify one `gh api` command (full token list incl. `gh api`)."""
    args = _parse_gh_api_args(tokens[2:])
    endpoint = args["endpoint"]
    if endpoint is None:
        return ASK
    norm = _normalize_endpoint(endpoint)

    if norm == "graphql":
        # GraphQL is always POST under the hood; classify by the operation type
        # of the inline query, never by HTTP method.
        if args["has_unparsed"]:
            return ASK
        queries = [v[len("query="):] for v in args["field_values"]
                   if v.startswith("query=")]
        if not queries:
            # Query supplied via --input/stdin/@file or not at all: can't verify.
            return ASK
        return READ if all(_graphql_is_readonly(q) for q in queries) else ASK

    method = args["method"]
    if method in READ_METHODS:
        is_write = False
    elif method is not None:
        is_write = True
    else:
        # No explicit method: gh defaults to POST when a body is attached.
        is_write = args["has_body"]

    if _path_is_refs(norm):
        # Git-ref tampering is hard-blocked, not merely prompted. Only a
        # provably-static read of a ref is allowed through; a write, an
        # unparsed flag, OR a shell-computed endpoint (which could word-split
        # into `-X DELETE`, e.g. `…/git/refs/heads/$BRANCH -X DELETE`) all deny.
        if is_write or args["endpoint_dynamic"] or args["has_unparsed"]:
            return DENY
        return READ

    if args["endpoint_dynamic"]:
        # Non-refs path built by the shell ($VAR / $(...) / `...`); can't
        # verify it — prompt rather than auto-approve.
        return ASK
    if args["has_unparsed"]:
        return ASK
    return ASK if is_write else READ


def _segments(tokens):
    """Yield the token slice of each `gh api` simple-command in the token list,
    splitting on shell pipeline/list/redirection operators."""
    i, n = 0, len(tokens)
    while i < n:
        if tokens[i] in TERMINATORS:
            i += 1
            continue
        # Start of a simple command: collect until a terminator.
        start = i
        while i < n and tokens[i] not in TERMINATORS:
            i += 1
        seg = tokens[start:i]
        # Is this command a `gh api ...` invocation? (`gh` may be preceded by
        # env assignments / sudo etc., which sit before it in their own command;
        # here `seg` already starts at the command word.)
        for j in range(len(seg) - 1):
            if seg[j] == "gh" and seg[j + 1] == "api":
                yield seg[j:]
                break


def _is_shell_dynamic(command):
    """True if the command contains shell expansion that could inject or rewrite
    `gh api` arguments at runtime — `$`/backtick outside single quotes, or an
    unquoted `{a,b}` brace expansion. The static parse cannot see what these
    expand to, so a dynamic command is never treated as a confirmed read.

    Quote-aware so it does NOT trip on GraphQL `$variables` inside a
    single-quoted `-f query='query($id){…}'`, which the shell leaves literal."""
    in_single = in_double = False
    unquoted = []
    for c in command:
        if in_single:
            in_single = c != "'"
            continue
        if in_double:
            if c == '"':
                in_double = False
            elif c in ("$", "`"):
                return True  # expansion inside double quotes is shell-active
            continue
        if c == "'":
            in_single = True
            continue
        if c == '"':
            in_double = True
            continue
        if c in ("$", "`"):
            return True  # unquoted expansion / command substitution
        unquoted.append(c)
    # Unquoted brace expansion, e.g. `{-X,DELETE}`.
    return re.search(r"\{[^{}]*,[^{}]*\}", "".join(unquoted)) is not None


def _preprocess(command):
    """Apply bash newline semantics that shlex does not, quote-aware:

      * a backslash-newline line-continuation (outside single quotes) is removed,
        so `… main \\<nl>-X DELETE` rejoins to `… main -X DELETE`;
      * an unquoted literal newline separates commands, so it becomes ` ; `
        (a token `_segments` splits on).

    Newlines *inside* quotes (e.g. a multi-line `-f query='…'`) are preserved."""
    out = []
    i, n = 0, len(command)
    in_single = in_double = False
    while i < n:
        c = command[i]
        if in_single:
            out.append(c)
            if c == "'":
                in_single = False
            i += 1
            continue
        if c == "\\" and i + 1 < n and not in_single:
            nxt = command[i + 1]
            if nxt == "\n":            # line continuation
                i += 2
                continue
            if nxt == "\r" and command[i + 2:i + 3] == "\n":
                i += 3
                continue
            out.append(c)              # any other escaped char: keep both
            out.append(nxt)
            i += 2
            continue
        if in_double:
            out.append(c)
            if c == '"':
                in_double = False
            i += 1
            continue
        if c == "'":
            in_single = True
            out.append(c)
        elif c == '"':
            in_double = True
            out.append(c)
        elif c in ("\n", "\r"):
            out.append(" ; ")          # unquoted newline = command separator
        else:
            out.append(c)
        i += 1
    return "".join(out)


def decide(command):
    """Return (decision, reason) where decision is 'deny'/'ask'/None."""
    try:
        lex = shlex.shlex(_preprocess(command), posix=True, punctuation_chars=True)
        lex.whitespace_split = True
        lex.commenters = ""  # `#` is not a mid-word comment in bash (and is a
        # URL fragment in endpoints); never let it truncate the token stream.
        tokens = list(lex)
    except ValueError:
        # Unbalanced quotes etc. — fail closed.
        return ASK, "Could not parse the shell command; gating for manual review."

    verdicts = [_classify_gh_api(seg) for seg in _segments(tokens)]
    if not verdicts:
        return None, None  # No gh api command found; defer to normal flow.

    if DENY in verdicts:
        return DENY, (
            "Blocked: writing to the git-refs endpoint (force-update or delete "
            "of a branch/tag pointer) is irreversible and never part of a "
            "routine gh api workflow. Run it manually if you truly intend to."
        )
    if ASK in verdicts or _is_shell_dynamic(command):
        # Either a statically-detected write, or shell expansion that could be
        # hiding/injecting one — confirm before it runs.
        return ASK, (
            "This gh api call is not provably read-only (a GraphQL mutation; a "
            "REST POST/PATCH/PUT/DELETE; an implicit POST from body fields; or a "
            "body/argument computed by the shell that can't be verified). "
            "Confirm before it runs."
        )
    return None, None  # All segments are confirmed reads.


def main():
    try:
        payload = json.load(sys.stdin)
        command = payload.get("tool_input", {}).get("command", "")
    except (ValueError, AttributeError):
        # Malformed hook input but the matcher fired on a gh api command:
        # fail closed.
        _emit(ASK, "Could not read hook input; gating gh api for manual review.")
        return

    if not isinstance(command, str) or not command.strip():
        return  # Nothing to gate.

    try:
        decision, reason = decide(command)
    except Exception:  # noqa: BLE001 — a gate must never crash open.
        decision, reason = ASK, "gh-api gate hit an internal error; gating for manual review."
    if decision is not None:
        _emit(decision, reason)


def _emit(decision, reason):
    json.dump({
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": decision,
            "permissionDecisionReason": reason,
        }
    }, sys.stdout)
    sys.stdout.write("\n")


if __name__ == "__main__":
    try:
        main()
    except Exception:  # noqa: BLE001 — never let the gate exit non-zero / open.
        _emit(ASK, "gh-api gate hit an internal error; gating for manual review.")
    sys.exit(0)
