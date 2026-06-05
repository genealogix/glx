#!/usr/bin/env python3
"""Tests for gh-api-gate.py (genealogix/glx#912).

Run: python scripts/claude-hooks/gh-api-gate_test.py
 or: python -m unittest -v scripts.claude-hooks.gh-api-gate_test  (won't import — hyphen)

Decision contract for `decide(command)`:
  * None  -> confirmed read-only; hook stays silent and the allow-list rule
             auto-approves the call.
  * "ask" -> a write/mutation or anything not provably read-only; force a prompt.
  * "deny"-> git-ref tampering; hard block.
"""

import importlib.util
import json
import os
import subprocess
import sys
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_GATE_PATH = os.path.join(_HERE, "gh-api-gate.py")

_spec = importlib.util.spec_from_file_location("gh_api_gate", _GATE_PATH)
gate = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(gate)


def verdict(command):
    """Return the bare decision string ('ask'/'deny') or None for read-only."""
    return gate.decide(command)[0]


# (command, expected_decision) — None means read-only/allow.
CASES = [
    # ---- The seven cases enumerated in issue #912 ----
    ("gh api graphql -f query='query { viewer { login } }'", None),
    ("gh api repos/genealogix/glx/issues/871", None),
    ("gh api repos/genealogix/glx/issues/911/comments", None),
    ("gh api graphql -f query='mutation { resolveReviewThread(input:{threadId:$id}) { thread { id } } }'", "ask"),
    ("gh api repos/genealogix/glx/issues/comments/123 -X DELETE", "ask"),
    ("gh api repos/genealogix/glx/git/refs/heads/main -X PATCH -f sha=abc", "deny"),
    ("gh api -X DELETE repos/genealogix/glx/issues/comments/123", "ask"),

    # ---- GraphQL operation-type detection ----
    ("gh api graphql -f query='{ viewer { login } }'", None),            # anonymous query
    ("gh api graphql -f query='query Foo($x:Int){ a }'", None),          # named query
    ("gh api graphql -f query='# comment\nquery { a }'", None),          # leading comment
    ("gh api graphql -f query='\n\n  query { a }'", None),               # leading whitespace
    ("gh api graphql -f query='query { search(query:\"how to mutation\", type:ISSUE){ x } }'", None),  # 'mutation' in a string
    ("gh api graphql -f query='mutation{ a }'", "ask"),                  # no space
    ("gh api graphql -f query='subscription { a }'", "ask"),             # subscription is not a read we allow
    ("gh api graphql -f query='# mutation\nmutation { a }'", "ask"),     # comment can't hide the real op
    ("gh api graphql -f query='fragment F on T { a } query Q { ...F }'", None),  # fragment + one query = read
    ("gh api graphql -f query='queryThing { a }'", "ask"),              # identifier starting with 'query'
    ("gh api graphql -F query='mutation { a }'", "ask"),                 # -F typed field
    ("gh api graphql --field query='mutation { a }'", "ask"),
    ("gh api graphql --raw-field=query='mutation { a }'", "ask"),        # long flag with '='
    ("gh api graphql -fquery='mutation { a }'", "ask"),                  # glued short flag
    ("gh api graphql -f query=@query.graphql", "ask"),                   # body from @file
    ("gh api graphql --input query.json", "ask"),                       # body from file
    ("gh api graphql --input -", "ask"),                                # body from stdin
    ("gh api graphql", "ask"),                                          # no query at all
    ("gh api graphql -f query=''", "ask"),                              # empty query
    ("gh api graphql -f query='query{a}' -f variables='{\"x\":1}'", None),  # variables field ignored
    ("gh api graphql -f query='query{a}' -f query='mutation{b}'", "ask"),   # most-restrictive across fields
    ("gh api /graphql -f query='query { a }'", None),                   # leading-slash graphql endpoint

    # ---- REST reads (allow / None) ----
    ("gh api repos/genealogix/glx", None),
    ("gh api repos/genealogix/glx/pulls/911", None),
    ("gh api repos/genealogix/glx/issues -X GET -f state=open", None),  # explicit GET + query params
    ("gh api --method GET repos/genealogix/glx/x", None),
    ("gh api repos/genealogix/glx/issues/1 -X HEAD", None),
    ("gh api repos/genealogix/glx/issues/1 -q .title", None),           # jq flag is read
    ("gh api repos/genealogix/glx/issues/1 -q.title", None),            # glued jq flag
    ("gh api repos/genealogix/glx/issues/1 -i", None),                  # include headers, read
    ("gh api repos/genealogix/glx/issues --paginate", None),
    ("gh api repos/genealogix/glx/issues/1 -H 'Accept: application/vnd.github+json'", None),

    # ---- REST writes (ask) ----
    ("gh api repos/genealogix/glx/issues/123/comments -f body=hi", "ask"),   # implicit POST (body, no -X)
    ("gh api repos/genealogix/glx/pulls/911/comments/123/replies -f body='Addressed'", "ask"),
    ("gh api --method POST repos/genealogix/glx/issues/1/comments -f body=hi", "ask"),
    ("gh api repos/genealogix/glx/issues/1 -X PATCH -f title=new", "ask"),
    ("gh api repos/genealogix/glx/x -X PUT -f a=b", "ask"),
    ("gh api -XDELETE repos/genealogix/glx/issues/comments/1", "ask"),       # glued
    ("gh api --method=DELETE repos/genealogix/glx/issues/comments/1", "ask"),  # long '='
    ("gh api repos/genealogix/glx/issues/1/labels --input labels.json", "ask"),  # body via file -> POST

    # ---- git-refs tampering (deny) ----
    ("gh api repos/genealogix/glx/git/refs -X POST -f ref=refs/heads/x -f sha=abc", "deny"),
    ("gh api repos/genealogix/glx/git/refs/heads/feature -X DELETE", "deny"),
    ("gh api repos/genealogix/glx/git/refs/heads/main -X PATCH -f sha=abc -f force=true", "deny"),
    ("gh api -X DELETE repos/genealogix/glx/git/refs/tags/v1", "deny"),
    ("gh api repos/genealogix/glx/git/refs/heads/main", None),               # GET a ref is a read

    # ---- compound / piped / redirected commands ----
    ("gh api repos/genealogix/glx/issues/1 && gh api repos/genealogix/glx/issues/2", None),
    ("gh api repos/genealogix/glx/issues/1 && gh api repos/genealogix/glx/issues/comments/2 -X DELETE", "ask"),
    ("gh api repos/genealogix/glx/issues/1 | jq .title", None),
    ("gh api repos/genealogix/glx/issues/1 > out.json", None),
    ("(gh api repos/genealogix/glx/issues/1)", None),
    ("gh api repos/genealogix/glx/issues/1; gh api repos/genealogix/glx/git/refs/heads/x -X DELETE", "deny"),
    ("GH_HOST=github.com gh api repos/genealogix/glx/issues/1", None),
    # a read gh api alongside a non-gh-api write should still be read-only for OUR gate
    ("gh api repos/genealogix/glx/issues/1 && rm -rf /tmp/x", None),

    # ---- robustness / fail-closed ----
    ("gh api repos/genealogix/glx/issues/1 -X", "ask"),                 # dangling method flag, no endpoint clarity
    ("gh api 'graphql' -f query='mutation { a }'", "ask"),             # quoted endpoint

    # ==== Regressions from the #912 red-team (all were confirmed under-gates) ====

    # -- GraphQL multi-operation + operationName decoy (was: none) --
    ("gh api graphql -f query='query Q { viewer { login } } mutation Evil { deleteRef(input:{refId:\"x\"}) { clientMutationId } }' -f operationName=Evil", "ask"),
    ("gh api graphql -f query='{ viewer { login } } mutation Evil { deleteRef(input:{}){__typename} }' -f operationName=Evil", "ask"),
    ("gh api graphql -f query='query Q{x} mutation Evil{deleteRef}' -foperationName=Evil", "ask"),
    # a leading fragment in front of a single real query is still a read
    ("gh api graphql -f query='fragment F on T { a } query Q { ...F }'", None),
    # two queries (no mutation) — still gated: operationName could select either
    ("gh api graphql -f query='query A{x} query B{y}'", "ask"),

    # -- refs-path obfuscation (was: ask, must be deny) --
    ("gh api repos/genealogix/glx/%67it/refs/heads/main -X DELETE", "deny"),     # %67 = 'g'
    ("gh api repos/genealogix/glx/git/%72efs/heads/main -X DELETE", "deny"),     # %72 = 'r'
    ("gh api repos/genealogix/glx/git%2Frefs/heads/main -X DELETE", "deny"),     # %2F = '/'
    ("gh api repos/genealogix/glx/git/./refs/heads/main -X DELETE", "deny"),     # dot-segment
    ("gh api repos/genealogix/glx/git//refs/heads/main -X DELETE", "deny"),      # double slash
    ("gh api repos/genealogix/glx/Git/Refs/heads/main -X DELETE", "deny"),       # case
    ("gh api repos/genealogix/glx/git/Refs/heads/main -X DELETE", "deny"),       # case
    ("gh api REPOS/genealogix/glx/git/refs/heads/main -X DELETE", "deny"),       # case
    ("gh api 'repos/genealogix/glx/git/refs#x' -X DELETE", "deny"),              # fragment
    ("gh api 'repos/genealogix/glx/git/refs?per_page=1' -X DELETE", "deny"),     # query string
    ("gh api https://api.github.com/repos/genealogix/glx/git/refs/heads/main -X DELETE", "deny"),  # absolute URL
    ("gh api repos/genealogix/glx/%2567it/refs/heads/main -X DELETE", "deny"),   # double-encoded 'g'
    # refs GET stays a read even when obfuscated
    ("gh api repos/genealogix/glx/Git/Refs/heads/main", None),

    # -- glued short-flag clusters (was: ask, must be deny/correct) --
    ("gh api -iX DELETE repos/genealogix/glx/git/refs/heads/main", "deny"),      # -iX = -i -X
    ("gh api -iX DELETE repos/genealogix/glx/issues/comments/1", "ask"),         # -iX on non-refs
    ("gh api -i repos/genealogix/glx/issues/1", None),                           # -i alone, read

    # -- shell substitution / dynamic endpoints (must floor at ask) --
    ("gh api repos/genealogix/glx/$P -X DELETE", "ask"),
    ("gh api repos/genealogix/glx/issues/$NUM", "ask"),                          # dynamic read path -> ask (conservative)
    ("gh api `cat f`/git/refs/heads/main -X DELETE", "ask"),
    ("gh api $(echo repos/genealogix/glx/issues/1)", "ask"),
    ("gh api repos/genealogix/glx/git/refs/heads/${BRANCH} -X DELETE", "deny"),  # ref write w/ dynamic operand -> hard block
    ("gh api repos/genealogix/glx/git/refs/heads/${BRANCH}", "deny"),            # dynamic operand could word-split a write onto a ref path
    ("gh api repos/genealogix/glx/issues/${NUM}", "ask"),                        # non-refs dynamic endpoint -> prompt

    # -- deeper GraphQL scanner / normalization probes (round 2) --
    ('gh api graphql -f query=\'query { a } """desc""" mutation { evil }\'', "ask"),   # mutation after closed block-string desc
    ('gh api graphql -f query=\'query { a(x:"} mutation { evil }") }\'', None),        # mutation inside a string is inert
    ("gh api repos/genealogix/glx/git/refs -f ref=refs/heads/x -f sha=abc", "deny"),  # implicit-POST ref creation
    ("gh api -X DELETE -- repos/genealogix/glx/git/refs/heads/main", "deny"),         # path after -- separator
    ("gh api repos/genealogix/glx/issues/1 --method GET --method DELETE", "ask"),     # last method wins
    ("gh api repos/genealogix/glx/git/refs/heads/x --method GET --method DELETE", "deny"),
    ("gh api repos/genealogix/glx/git/refs/heads/x -X delete", "deny"),               # lowercase method
    ("gh api repos/genealogix/glx/git/trees/../refs/heads/main -X DELETE", "deny"),   # path traversal into refs
    ("gh api https://user@api.github.com:443/repos/genealogix/glx/git/refs/heads/x -X DELETE", "deny"),  # URL w/ userinfo+port
    ("gh api repos/genealogix/glx/git/refs/heads/x --input -", "deny"),               # stdin body = implicit POST on refs
    ("gh api repos/genealogix/glx/issues/1&&gh api repos/genealogix/glx/git/refs/heads/x -X DELETE", "deny"),  # glued && 2nd segment
    ("gh api $(printf -- '-X DELETE repos/genealogix/glx/git/refs/heads/x')", "ask"), # substitution injecting flags

    # ==== Round-2 red-team regressions ====

    # -- `#` must NOT be treated as a shell comment (bash: mid-word # is literal;
    #    in an endpoint it's a URL fragment). Was silently dropping the rest. --
    ("gh api repos/genealogix/glx/git/refs/heads/main#x -X DELETE", "deny"),
    ("gh api repos/genealogix/glx/git/refs#heads/main -X DELETE", "deny"),
    ("gh api repos/genealogix/glx/issues/comments/123#x -X DELETE", "ask"),
    ("gh api repos/genealogix/glx/issues/comments/123#x -f body=pwned", "ask"),

    # -- shell substitution / $VAR / brace expansion in ARGUMENT position can
    #    inject flags at runtime; must floor at ask (never silent). --
    ("gh api repos/genealogix/glx/issues $(printf -- '-X DELETE')", "ask"),
    ("gh api repos/genealogix/glx/git/refs/heads/x $(printf -- '-X DELETE')", "ask"),
    ("gh api repos/genealogix/glx/issues $METHOD", "ask"),
    ("gh api repos/genealogix/glx/issues {-X,DELETE}", "ask"),
    ("gh api repos/genealogix/glx/git/refs/heads/x {-X,DELETE}", "ask"),
    ("gh api repos/genealogix/glx/issues `printf -- '-X DELETE'`", "ask"),
    ("gh api repos/genealogix/glx/issues `printf -- '-f title=x'`", "ask"),

    # -- GraphQL block strings (""") are gated: the `\\\"\"\"` escape makes lexing
    #    ambiguous and could hide a trailing mutation. --
    ('gh api graphql -f query=\'query Good { a(x: """ \\""" """) } mutation Evil { deleteRef(input:{refId:"R"}){clientMutationId} }\' -F operationName=Evil', "ask"),
    ('gh api graphql -f query=\'{ a(x: """ \\""" """) } mutation Evil { deleteRef }\' -F operationName=Evil', "ask"),
    ('gh api graphql -f query=\'query G { a(x: """ \\""" """) } subscription S { x }\'', "ask"),
    ("gh api graphql -f query='query { a(desc: \"\"\"hi\"\"\") }'", "ask"),       # even a benign block string -> ask

    # -- regression guards: legit reads must STILL be silent --
    ("gh api graphql -f query='query($id:ID!){ node(id:$id){ id } }'", None),    # single-quoted GraphQL $var
    ("gh api graphql -f query='query($owner:String!){ repositoryOwner(login:$owner){ id } }'", None),

    # ==== Round-3 red-team regressions: bash newline semantics ====
    # backslash-newline is a line continuation -> rejoin before classifying
    ("gh api repos/genealogix/glx/git/refs/heads/main \\\n-X DELETE", "deny"),
    ("gh api repos/genealogix/glx/issues/comments/1 \\\n-X DELETE", "ask"),
    ("gh api repos/genealogix/glx/git/refs/heads/x \\\r\n-X DELETE", "deny"),
    # an unquoted literal newline separates commands -> classify each segment
    ("gh api repos/genealogix/glx/issues/1\ngh api repos/genealogix/glx/git/refs/heads/x -X DELETE", "deny"),
    # newlines INSIDE a single-quoted query body are literal, not separators
    ("gh api graphql -f query='query {\n viewer {\n login\n }\n}'", None),
    ("gh api graphql -f query='mutation {\n deleteRef\n}'", "ask"),

    # the numeric `repositories/{id}/git/refs` endpoint form is refs too
    ("gh api repositories/1300192/git/refs -f ref=refs/heads/x -f sha=abc", "deny"),
    ("gh api repositories/1300192/git/refs/heads/main -X DELETE", "deny"),
    ("gh api repositories/1300192/git/refs/heads/main", None),   # GET a ref = read
    ("gh api repositories/1300192", None),                        # plain GET
]


class GateDecisionTests(unittest.TestCase):
    def test_cases(self):
        failures = []
        for command, expected in CASES:
            got = verdict(command)
            if got != expected:
                failures.append(f"  {command!r}\n    expected={expected!r} got={got!r}")
        if failures:
            self.fail("Mismatched decisions:\n" + "\n".join(failures))


class GateIOContractTests(unittest.TestCase):
    """Exercise the script end-to-end over stdin/stdout, the way the hook runs."""

    def _run(self, command):
        payload = json.dumps({
            "hook_event_name": "PreToolUse",
            "tool_name": "Bash",
            "tool_input": {"command": command},
        })
        proc = subprocess.run(
            [sys.executable, _GATE_PATH],
            input=payload, capture_output=True, text=True,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        out = proc.stdout.strip()
        if not out:
            return None
        return json.loads(out)["hookSpecificOutput"]["permissionDecision"]

    def test_read_emits_nothing(self):
        self.assertIsNone(self._run("gh api repos/genealogix/glx/issues/1"))

    def test_mutation_asks(self):
        self.assertEqual(self._run("gh api graphql -f query='mutation { a }'"), "ask")

    def test_ref_denies(self):
        self.assertEqual(self._run("gh api repos/genealogix/glx/git/refs/heads/x -X DELETE"), "deny")

    def test_malformed_input_fails_closed(self):
        proc = subprocess.run(
            [sys.executable, _GATE_PATH],
            input="not json", capture_output=True, text=True,
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(
            json.loads(proc.stdout)["hookSpecificOutput"]["permissionDecision"], "ask")


if __name__ == "__main__":
    unittest.main(verbosity=2)
