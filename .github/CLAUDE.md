# .github/ — Claude Guide

## This file is never the source of truth

Policies, conventions, and decisions live in project files —
`../SECURITY-POSTURE.md`, `../CONTRIBUTING.md`, `../docs/`,
`../specification/`, `../docs/decisions/`. A `CLAUDE.md` may only summarize
them and point at them, and public docs must never link to a `CLAUDE.md`.
When a convention changes, change the project file first; update the summary
here afterwards.

## GitHub Actions pinning: SHA-pin everything, no exceptions

<!-- Last reviewed: 2026-09-18 (#1022, reversed) -->

**Source of truth: the "Action pinning" section of `../SECURITY-POSTURE.md`.**
Read it before changing any `uses:` line, and record any change to the
convention there — not here. This is a summary for applying the rule, and
nothing in it may contradict that section.

Every `uses:` reference — third-party, `github/*`, and first-party `actions/*`
alike — is a full commit SHA with a trailing version comment:

```yaml
uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
```

The rule is namespace-blind. There is no first-party carve-out: the floating
`actions/*` majors were removed in #1022 (reversing that issue's original
"accept the floats" close), because a major tag is a mutable ref no matter who
publishes it.

Practical consequences:

- Never "tidy" a SHA pin down to a bare tag. Any bare `@vN` in a workflow diff
  is a defect, whatever the namespace.
- Three actions publish no floating major at all — `ossf/scorecard-action`
  (#779), `mszostok/codeowners-validator`, `sigstore/cosign-installer` (#938).
  Their `uses:` lines carry a trailing "do not tidy to @vN" note; "correcting"
  them fails with `Unable to resolve action <owner>/<repo>@vN`.
- Dependabot sustains the pins: it bumps the SHA and rewrites the `# vX.Y.Z`
  comment together, weekly, via the `github-actions` ecosystem entry.
- To find the SHA for a tag (this dereferences annotated tags correctly):

  ```bash
  gh api repos/<owner>/<repo>/commits/<tag> --jq .sha
  ```

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
  (the single source of truth), then update the `Area` dropdown in every
  `ISSUE_TEMPLATE/*.yml` form (each must keep its `id: area` key) or
  `issue-templates-drift.yml` hard-fails. Also create the matching repo
  label: the same workflow queries the repo's labels and hard-fails the PR
  when an Area has no matching label (#946, PR #1183) — without that gate,
  the gap would only surface as `issue-labeler.yml` failing `--add-label`
  on the first issue filed with that Area, leaving it unlabeled. The
  labeler and the drift check read that YAML with deliberately chosen
  parsers (#947 has the history) — don't swap them casually.

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
