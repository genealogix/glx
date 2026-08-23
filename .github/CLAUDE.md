# .github/ — Claude Guide

## GitHub Actions pinning: SHA-pin third-party, floats only for first-party

<!-- Last reviewed: 2026-08-23 (#1046) -->

**The repo's actual convention (since the #963/#968 SHA-pin sweep):**

1. **Third-party actions (the default rule): full-commit-SHA pin plus a
   trailing `# vX.Y.Z` comment.** This is the OpenSSF Scorecard
   "pin dependencies" form and what every third-party `uses:` in the repo
   follows today, e.g.

   ```yaml
   uses: golangci/golangci-lint-action@82606bf257cbaff209d206a39f5134f0cfbfd2ee # v9.2.1
   ```

   Do NOT "tidy" a SHA pin down to a bare tag — tags are force-repointable
   upstream (the tj-actions/changed-files incident, CVE-2025-30066). Dependabot
   understands the SHA + comment form and bumps both together.

2. **First-party `actions/*` (checkout, setup-go, setup-node, setup-python,
   upload/download-artifact, cache, attest): floating major tags are the
   current documented choice**, pending the decision in #1022. Don't SHA-pin
   these piecemeal.

3. **Exceptions without a floating major** (they only publish `vX.Y.Z`):
   pin the exact patch tag as the floor, SHA preferred. Each one carries a
   comment so nobody "tidies" it back to `@vN`, which fails with
   `Unable to resolve action <owner>/<repo>@vN`:

   | Action | Pin floor | Issue |
   |---|---|---|
   | `sigstore/cosign-installer` | `@v4.1.2` exact tag | #938 |
   | `ossf/scorecard-action` | SHA-pinned since #1019 | #779 (original lesson) |

**Verify before you push.** Before editing any `uses:` line, check what tags
the action actually publishes:

```bash
gh api repos/<owner>/<repo>/tags --jq '.[].name' | head
```

Caveat: a SHA pin is necessary, not sufficient — it does not pin the action's
transitive `uses:` references, and a pinned action can still fetch a
re-registrable domain at runtime. See `SECURITY-POSTURE.md` for the posture
this supports, and the root `CLAUDE.md` / `go-glx/CLAUDE.md` for sibling
guides.

## Release pipeline: never move a published tag — cut a new patch tag

The `release.yml` workflow triggers on tag push and runs from the workflow
file **at the commit the tag points to**. Re-running a failed release does
NOT pick up a fix you've since merged to main — the re-run uses the same
old YAML.

If a tag's release workflow fails for a workflow-file reason:

1. Land the fix on `main` via a normal hotfix PR.
2. Cut a NEW patch tag from hotfixed `main` (e.g. `v1.2.3` → `v1.2.4`).
   Pushing the new tag triggers a fresh run that loads the corrected YAML.

> Do NOT delete-and-recreate the same tag name. The old
> `git tag -d X && git push --delete origin X && git tag X && git push origin X`
> recipe is broken-by-design under GitHub Immutable Releases (GA 2025-10-28,
> tracked for this repo in #931) and under any `v*` tag ruleset with Restrict
> updates/deletions: recreating the name returns HTTP 422
> `tag_name was used by an immutable release`, burning the tag name
> permanently. There is also no operationally useful "safe window before
> assets upload" — in a normal GoReleaser run that window is seconds.
>
> For a safe pre-publish re-run window, prefer GoReleaser
> `release.draft: true` (assets upload to a still-mutable draft; flip to
> published only when green) instead of any tag surgery.

## Drift gates that will fail your PR

Three CI gates hard-fail PRs for out-of-sync generated or mirrored content —
know which one you're about to trip before you push:

- **Edit a CLI command** → run `make docs-cli` and commit the regenerated
  `docs/cli/**` (`docs-drift.yml`, hard-fail).
- **Change a JSON schema** → `drift-checks.yml` runs two checks with
  different teeth: schema backward-compat is HARD-fail (a breaking change
  invalidates existing archives); spec-schema field parity is WARN-only
  (#309). A green "spec-schema" job does NOT mean parity is enforced.
- **Add or rename an issue Area** → edit `.github/issue-areas.yml` first
  (the single source of truth), create the matching repo label, then update
  the `Area` dropdown in every `ISSUE_TEMPLATE/*.yml` form (each must keep
  its `id: area` key) or `issue-templates-drift.yml` hard-fails. The labeler
  and the drift check read that YAML with deliberately chosen parsers
  (#947 has the history) — don't swap them casually.

## Workflow injection: never interpolate untrusted input into `run:`

The `security-guidance` plugin issues a warning on every `Edit` of a workflow
file. The summary:

- Don't use `${{ github.event.issue.title }}` (or any other attacker-
  controllable field) directly inside a `run:` block — that's command
  injection.
- Stage the value as `env:`, then reference `"$VAR"` inside the script with
  quotes.
- `ref:` of `actions/checkout` must never accept untrusted input. For
  `client_payload.pr_number` from `repository_dispatch`, validate against
  `^[0-9]+$` before interpolation.

Full guide: <https://github.blog/security/vulnerability-research/how-to-catch-github-actions-workflow-injections-before-attackers-do/>

## `security.yml` holds SARIF-uploading jobs and nothing else

GitHub treats `security.yml` as the Code Scanning **setup** for the `gosec` and
`govulncheck` tools, and the tool status page attributes the conclusion of the
whole *workflow run* to every tool that setup configures — not the conclusion of
the job that actually uploaded the SARIF.

So any failing job in `security.yml`, even one that touches no scanning at all,
puts a red error on both Go tools and raises

> Code scanning configuration error: Golang security checks by gosec and
> govulncheck are reporting errors.

on the Security tab — while `code-scanning/analyses` reports `error: ""` for
every upload, because the uploads were fine. That misdirection cost a debugging
session in #1145; the two offenders (`gosec Pin Currency`, `npm Audit`) now live
in `gosec-pin-currency.yml` and `npm-audit.yml`.

**Never add a job to `security.yml` unless it uploads code-scanning results.**
Auxiliary security checks get their own workflow file. Confirm the mechanism at
`/security/code-scanning/tools/<tool>/status` — compare against `Scorecard`,
which reports no scanned-files summary either yet stays green because its
workflow passes.

## Release workflow specifics

- Tag patterns that trigger `release.yml`: `v[0-9]+.[0-9]+.[0-9]+` and
  `v[0-9]+.[0-9]+.[0-9]+-beta*`. Use POSIX-ERE (`grep -E`) for tag listing,
  not fnmatch globs — the latter lets `v1.2.3.4` and `v1.2.3-rc1` slip
  through (see the `/compact-changelog` skill for the canonical pattern).
- Cosign keyless signing uses GitHub OIDC. Verifiers MUST pass
  `--certificate-identity-regexp '^https://github\.com/genealogix/glx/\.github/workflows/release\.yml@refs/tags/'`
  and `--certificate-oidc-issuer 'https://token.actions.githubusercontent.com'`
  — see `SECURITY-POSTURE.md` for the full command.
- Discord announcement is gated on `secrets.DISCORD_RELEASE_WEBHOOK`; the
  step skips cleanly when the secret isn't set.
- **`id-token: write` must never coexist with a cache (or any other
  untrusted-code execution path) reachable from `pull_request`.** The release
  job is privileged — it signs binaries with the project's real OIDC identity —
  so it must not restore a Go build cache that `pull_request`-triggered
  workflows (`security.yml`, `validate-spec.yml`) can populate; a poisoned
  cache entry would taint the build *upstream* of signing, and provenance/
  signatures won't catch it. Hence `cache: false` on `actions/setup-go` in
  `release.yml` (#1051) — do not "tidy" it back to `true`. Wiring zizmor's
  `cache-poisoning` audit into CI (#928) would catch regressions generically.
