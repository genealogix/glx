# .github/ — Claude Guide

## Third-party GitHub Actions: always pin to `vX.Y.Z`, never `@vN`

**Verify before you push.** Most third-party actions in this repo bumped via
`@vN` floating tags have broken at some point because the upstream maintainer
doesn't publish a floating major-version tag. Symptoms:

- `Unable to resolve action <owner>/<repo>@vN, unable to find version vN`
  (action only ships `vX.Y.Z`, no floating major)
- A release run that pulls a stale binary path because the floating tag never
  moved when a new minor shipped

Before editing any `uses:` line in `.github/workflows/*.yml`, **check what tags
the action actually publishes**:

```bash
gh api repos/<owner>/<repo>/tags --jq '.[].name' | head
```

If you only see `v4.1.2`, `v4.1.1`, etc. and no bare `v4`, pin to the latest
patch (`@v4.1.2`). Add a comment explaining why the pin is patch-level so the
next person (or AI assistant) doesn't "tidy up" the pin back to `@v4`.

**Known instances in this repo** — each one cost a failed release pipeline:

| Action | Required pin form | Issue / commit |
|---|---|---|
| `ossf/scorecard-action` | `@v2.4.3` (no floating `@v2`) | #779 |
| `sigstore/cosign-installer` | `@v4.1.2` (no floating `@v4`) | #938 |

The whole project also follows OpenSSF Scorecard "pin dependencies" guidance
(see `SECURITY-POSTURE.md`), so pinning to a tag is the floor — pinning to a
commit SHA is acceptable too and arguably preferred.

## Release pipeline: don't push a tag without verifying its workflow YAML

The `release.yml` workflow triggers on tag push and runs from the workflow
file **at the commit the tag points to**. Re-running a failed release does
NOT pick up a fix you've since merged to main — the re-run uses the same
old YAML.

If a tag's release workflow fails for a workflow-file reason:

1. Land the fix on `main` via a normal hotfix PR.
2. Move the release tag forward to the new `main` HEAD that includes the fix
   (`git tag -d X && git push --delete origin X && git tag X && git push origin X`).
3. Pushing the moved tag triggers a fresh run that loads the corrected YAML.

This is only safe **before the goreleaser run has uploaded assets**. If a
release already has published binaries, moving the tag would orphan them and
confuse `cosign verify-blob` against the new artifact set — investigate
before touching the tag.

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
