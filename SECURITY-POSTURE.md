---
title: Security Posture
description: GLX self-attestation against the OpenSSF OSPS Baseline and notes on EU CRA readiness.
layout: doc
---

# Security Posture

> As of **2026-09-18**, GLX self-attests to the [OpenSSF OSPS Baseline](https://baseline.openssf.org/) version **2026.02.19** at **Level 3 (mature)**: every control tracked below is met, and no gap remains open.
>
> **One control is configured but not yet demonstrated on a published artifact.** SBOM emission takes effect on releases cut from the next tag onward; `v0.0.0-beta.12` (2026-09-17) and every earlier release predate it and publish no `.sbom.json` assets. Read the [SBOM verification details](#sbom-verification-details) against a release tagged after that date.

This document is the public-facing companion to [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md). SECURITY.md tells you how to *report* a vulnerability; this file tells you how the project handles supply-chain and process risk so adopters can make informed decisions about depending on GLX.

## Why this exists

Two converging frameworks shape the credibility expectations for open-source projects in 2026:

- **OpenSSF OSPS Baseline** — a structured set of controls organized into three maturity levels (L1 foundational → L3 mature) that adopters and procurement teams reference when evaluating dependencies. Published at <https://baseline.openssf.org/>.
- **EU Cyber Resilience Act (CRA)** — partial enforcement begins **2026-09-11**. GLX itself is a CLI tool processing local YAML files and is generally expected to fall outside the CRA's "product with digital elements" definition, though the project does not assert a definitive legal scope on adopters' behalf — see the [EU CRA section below](#eu-cyber-resilience-act-note) for the non-legal-advice posture. Downstream products that embed GLX may need this posture document to assemble their own CRA evidence pack; the OSPS Baseline was designed in part as a compliance bridge for that mapping.

GLX prioritizes the Baseline because doing so produces verifiable evidence (CI configurations, documented policies, signed releases) that adopters can cite directly without bespoke questionnaires.

## Current status

Status legend: ✓ met · ◐ partial · ☐ not yet met

### Level 1 — Foundational

| Control | Status | Evidence |
|---|---|---|
| Security contacts documented | ✓ | [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md) — GitHub Security Advisories with a 48-hour acknowledgment SLA |
| Contribution process explained (OSPS-GV-03.01) | ✓ | [CONTRIBUTING.md](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md) covers setup, branch naming, conventional commits, DCO sign-off, PR template |
| Direct commits to primary branch prevented | ✓ | Maintainer-verified as of 2026-05-07: an active **Main Protection** ruleset on the default branch requires PRs, blocks force-push and deletion, and gates merge on the workflows listed in the row below. The ruleset is org-internal (not in this repo); the workflows it gates are public under [`.github/workflows/`](https://github.com/genealogix/glx/tree/main/.github/workflows) |
| MFA for sensitive resource access | ✓ | Maintainer-verified as of 2026-05-07: two-factor authentication is enforced organization-wide on the [`genealogix`](https://github.com/genealogix) GitHub organization. Like the branch-protection row, this is a GitHub organization setting and is not visible from the repo contents |

#### Documentation (OSPS-DO)

| Control | Status | Evidence |
|---|---|---|
| OSPS-DO-01.01 — user guides for basic functionality | ✓ | [README.md](https://github.com/genealogix/glx/blob/main/README.md) (installation, quick start, CLI overview) and the [specification](https://github.com/genealogix/glx/tree/main/specification) |
| OSPS-DO-02.01 — guide for reporting defects | ✓ | [.github/SUPPORT.md](https://github.com/genealogix/glx/blob/main/.github/SUPPORT.md) (bug/feature/question routing) and [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md) (vulnerabilities) |

#### Governance (OSPS-GV)

| Control | Status | Evidence |
|---|---|---|
| OSPS-GV-02.01 — public discussion mechanism | ✓ | [GitHub Discussions](https://github.com/genealogix/glx/discussions) |

### Level 2 — Operationally Mature

| Control | Status | Evidence |
|---|---|---|
| Coordinated vulnerability disclosure policy | ✓ | [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md) defines the reporting channel, acknowledgment SLA, severity classification, and fix timelines, plus a [safe-harbor clause](https://github.com/genealogix/glx/blob/main/SECURITY.md#safe-harbor) protecting good-faith researchers and a [coordinated-disclosure embargo policy](https://github.com/genealogix/glx/blob/main/SECURITY.md#coordinated-disclosure-and-embargo) (90-day disclosure backstop, embargo-conduct expectations, and downstream notification via a published GitHub Security Advisory + CVE). Completed in [#424](https://github.com/genealogix/glx/issues/424) |
| Contributors assert legal authorization (DCO/CLA) | ✓ | [Developer Certificate of Origin 1.1](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md#developer-certificate-of-origin-dco) sign-off is required on every commit and is verified for every PR during maintainer review (with direct commits to the default branch blocked by protection rules). An automated DCO check is tracked as a future enhancement to reduce reviewer toil, not as a prerequisite for enforcing this control |
| Automated test suite runs before merge | ✓ | Five workflows run unconditionally on every `pull_request` — [CI](https://github.com/genealogix/glx/blob/main/.github/workflows/validate-spec.yml), [Security](https://github.com/genealogix/glx/blob/main/.github/workflows/security.yml), [npm Audit](https://github.com/genealogix/glx/blob/main/.github/workflows/npm-audit.yml), [Dependency Review](https://github.com/genealogix/glx/blob/main/.github/workflows/dependency-review.yml), and [Lint PR Title](https://github.com/genealogix/glx/blob/main/.github/workflows/lint-pr-title.yml). Two are path-filtered to PRs that touch the relevant files: [Lint](https://github.com/genealogix/glx/blob/main/.github/workflows/lint.yml) (`**.go`, `go.mod`, `go.sum`, `.golangci.yml`) and [Lint Markdown](https://github.com/genealogix/glx/blob/main/.github/workflows/lint-markdown.yml) (`specification/**/*.md`, `docs/**/*.md`, `*.md`, `.markdownlint-cli2.jsonc`). Whichever workflows are triggered are expected to be green before merge; the Main Protection ruleset (see L1 row above) gates this |
| Static analysis of dependencies and code | ✓ | [`security.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/security.yml) runs `govulncheck` (known-vulnerable Go dependencies) and `gosec` (Go static security analysis) on every PR and weekly. gosec is version-pinned via the repo's [`.gosec-version`](https://github.com/genealogix/glx/blob/main/.gosec-version) file, and a weekly [`gosec-pin-currency.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/gosec-pin-currency.yml) workflow fails when that pin falls behind the latest gosec release, so the analyzer set cannot silently go stale. This matters because gosec's rule set maps directly onto GLX's threat surface as a parser of untrusted GEDCOM/GEDZIP archives: G122 (symlink TOCTOU in directory walks) and G120 (unbounded stream parsing) cover the archive-extraction zip-slip/symlink and decompression-bomb classes — pattern-based coverage complementary to CodeQL's dataflow analysis. [`dependency-review.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/dependency-review.yml) blocks PRs that introduce known-vulnerable dependencies, and since [#340](https://github.com/genealogix/glx/issues/340) also fails PRs that add a dependency under a resolved license outside the configured allow-list (a license the action cannot resolve is reported, not failed). **CodeQL** is configured via [GitHub Default Setup](https://docs.github.com/en/code-security/code-scanning/enabling-code-scanning/configuring-default-setup-for-code-scanning) across `actions`, `go`, `javascript-typescript`, and `python` (maintainer-verified as of 2026-05-07; Default Setup is a repository setting in the Security tab and is not represented by a workflow file in this repo). The per-PR `Analyze (…)` jobs are visible on every PR's checks list, and findings surface in Code Scanning |
| Continuous security posture scoring | ✓ | [OpenSSF Scorecard](https://github.com/genealogix/glx/blob/main/.github/workflows/scorecard.yml) runs on push to `main`, weekly on Mondays, and on `workflow_dispatch`. Results are published to the public OSSF dataset and uploaded to GitHub Code Scanning as SARIF. The workflow deliberately has **no** `pull_request` trigger: publishing requires OIDC (`id-token: write`), which fork PRs cannot safely hold, so per-PR Scorecard coverage would break the publish model rather than harden it. Advisories the *Vulnerabilities* check reports that do not apply to GLX are handled per [Vulnerability suppression](#vulnerability-suppression), not left to drift |
| GitHub Actions pinned per a documented convention | ✓ | **Every** `uses:` reference in every workflow is pinned to a full commit SHA with a trailing version comment — third-party (including `github/*`) and first-party `actions/*` alike. No floating major tags and no exact-patch-tag references remain. See [Action pinning](#action-pinning) for the rationale and the mutable-ref threat it closes |

#### Action pinning

This section is the project's canonical statement of how GitHub Actions references are pinned. There is a single rule, applied to every `uses:` line in the repository.

**The rule — every action: full commit SHA plus a trailing version comment.** It admits no exceptions, and it is the form the OpenSSF Scorecard *Pinned-Dependencies* check expects:

```yaml
uses: golangci/golangci-lint-action@82606bf257cbaff209d206a39f5134f0cfbfd2ee # v9.2.1
```

A SHA pin is never "tidied" back down to a bare tag: upstream tags are force-repointable, which is how the `tj-actions/changed-files` compromise (CVE-2025-30066) reached its consumers. Dependabot understands the SHA-plus-comment form and bumps both together, so pinning does not freeze the action. The rule is namespace-blind: `github/codeql-action`, `actions/checkout` and `golangci/golangci-lint-action` are pinned identically. Publisher identity does not change the pin form, because the threat the pin closes — a mutable ref silently repointing — does not depend on who owns the ref.

**First-party `actions/*` are pinned too.** `checkout`, `setup-go`, `setup-node`, `setup-python`, `upload-artifact`, `download-artifact`, `cache`, `create-github-app-token`, `dependency-review-action` and `attest` all carry SHA pins. They were previously referenced by floating major tag on the reasoning that GitHub publishes them and also operates the runners, so the publisher is trusted end-to-end. That reasoning was reconsidered in [#1022](https://github.com/genealogix/glx/issues/1022) and reversed: trust in the publisher is not the question a pin answers. A pin answers whether the ref can change under the workflow, and a major tag is a mutable ref regardless of who owns it — a re-pointed one would run with the workflow's permissions, which is precisely the `tj-actions/changed-files` primitive (CVE-2025-30066). Uniformity also removes a standing review hazard: with one rule, any bare `@vN` in a diff is a defect, and no reviewer has to decide which namespace a given action belongs to.

**Actions that publish no floating major tag** — [`ossf/scorecard-action`](https://github.com/ossf/scorecard-action) ([#779](https://github.com/genealogix/glx/issues/779)), [`mszostok/codeowners-validator`](https://github.com/mszostok/codeowners-validator) and [`sigstore/cosign-installer`](https://github.com/sigstore/cosign-installer) ([#938](https://github.com/genealogix/glx/issues/938)) — need no special handling under a universal SHA rule, since a SHA is reachable whether or not a major tag exists. They previously needed a carve-out and `cosign-installer` in particular sat on an exact patch tag, which is itself a mutable ref; it is now SHA-pinned like everything else. Their `uses:` lines keep a trailing note recording that no floating major exists, so that a future edit does not "correct" them to `@vN` — which fails outright with `Unable to resolve action`.

**Accepted consequence.** Dependabot raises vulnerability *alerts* only for actions referenced by semantic-version tag, and [never for SHA-pinned references](https://docs.github.com/en/actions/reference/security/secure-use). Pinning everything therefore gives up that alerting channel across all workflows. It is accepted because the replacement path is already in place and is the one that actually lands a fix: Dependabot's weekly `github-actions` version updates in [`.github/dependabot.yml`](https://github.com/genealogix/glx/blob/main/.github/dependabot.yml) bump the SHA and rewrite the trailing `# vX.Y.Z` comment together, so a patched release arrives as a normal PR rather than as an alert a maintainer must act on by hand. An alert that is not acted on is not a control; a merged bump is.

**Known limit of this control.** A SHA pin is necessary but not sufficient. It does not pin the pinned action's own transitive `uses:` references, and a pinned action can still resolve a re-registrable domain at runtime. Pinning is one layer, not a complete supply-chain boundary.

**Named instance of that limit — the syft installer.** `anchore/sbom-action/download-syft` in [`release.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/release.yml) is SHA-pinned per the rule above, and the action at that SHA pins syft to an exact version. It nonetheless downloads `https://raw.githubusercontent.com/anchore/syft/<version>/install.sh` and executes it with `sh` at run time. That URL resolves a **git tag**, which is a mutable ref — the same repointing primitive behind CVE-2025-30066 — and the script runs inside the release job, which holds `id-token: write` and sits upstream of cosign signing and the SLSA attestation. The SHA pin on the `uses:` line does not close that path; only upstream shipping a pinned-by-digest installer would. The risk is accepted rather than dismissed: the alternative is vendoring an installer this project would then have to keep current, and the exposure window is limited to a tagged release run. It is recorded here so the SHA pin is not read as implying more than it delivers.

#### Vulnerability suppression

<!-- Last reviewed: 2026-09-18 -->

`govulncheck` decides applicability by reachability: it reports a dependency advisory only when the vulnerable symbol is actually called. The OpenSSF Scorecard *Vulnerabilities* check does not — it runs [osv-scanner](https://github.com/google/osv-scanner) over the manifests and reports every advisory that matches a resolved module version, whether or not the affected package is in the build graph. The two tools therefore disagree on advisories that are scoped to a package GLX does not import, and that disagreement is resolved in [`osv-scanner.toml`](https://github.com/genealogix/glx/blob/main/osv-scanner.toml) at the repository root, the file osv-scanner and Scorecard both read.

The rule for that file: an advisory is suppressed only when it cannot be cleared by upgrading **and** the affected package is provably outside the build graph, and every entry states the reason and the command that re-verifies it. An advisory that applies is fixed by bumping the dependency, never by an entry here — which is the same principle as [Action pinning](#action-pinning) above, where the control’s accepted consequence and its known limits are written down rather than papered over.

One entry exists today. `GO-2026-5932` marks `golang.org/x/crypto/openpgp` unmaintained; it has no fixed version, so no upgrade clears it. `golang.org/x/crypto` is reached only through `github.com/go-git/go-git/v5`, which uses `golang.org/x/crypto/hkdf` and gets its OpenPGP support from `github.com/ProtonMail/go-crypto/openpgp` — the maintained replacement. `go list -deps ./...` returns no `golang.org/x/crypto/openpgp` package, and `govulncheck` agrees, reporting it as present in a required module but uncalled.

### Level 3 — Mature

| Control | Status | Evidence |
|---|---|---|
| Release artifact signing | ✓ | [`.goreleaser.yml`](https://github.com/genealogix/glx/blob/main/.goreleaser.yml) signs `checksums.txt` with cosign keyless (Sigstore / OIDC), publishing `checksums.txt.sigstore.json`. Signing the checksum manifest transitively covers every release artifact via SHA-256. See [Release signing verification details](#release-signing-verification-details). ([#387](https://github.com/genealogix/glx/issues/387)) |
| SBOM with compiled releases | ✓ | [`.goreleaser.yml`](https://github.com/genealogix/glx/blob/main/.goreleaser.yml) emits one SPDX-JSON SBOM per release archive via the GoReleaser v2 `sboms:` stanza (syft, installed in [`release.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/release.yml) by a SHA-pinned `anchore/sbom-action/download-syft`). GoReleaser lists each SBOM in `checksums.txt` alongside the archives, so the SBOMs are covered by both the cosign keyless signature and the SLSA provenance attestation over that manifest. Effective for releases cut from the next tag onward; `v0.0.0-beta.12` and earlier predate the stanza. See [SBOM verification details](#sbom-verification-details). ([#269](https://github.com/genealogix/glx/issues/269)) |
| Build provenance / SLSA attestations | ✓ | [`release.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/release.yml) runs `actions/attest` (SHA-pinned with a trailing version comment) after GoReleaser with `subject-checksums: ./dist/checksums.txt`, producing a keyless-signed SLSA provenance attestation that covers every release artifact via SHA-256. See [Build-provenance verification details](#build-provenance-verification-details). ([#256](https://github.com/genealogix/glx/issues/256)) |
| OSPS-DO-04.01 — support scope/duration per release | ✓ | GLX is pre-1.0: security fixes target the **latest 0.x.x release** (the most recent tag) only, per the [Supported Versions table in SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md#supported-versions). No fixed support duration is promised for any individual release; a release stops receiving security fixes when the next release supersedes it ([#1060](https://github.com/genealogix/glx/issues/1060)) |
| OSPS-DO-05.01 — end of security updates stated | ✓ | Security fixes target the latest 0.x.x release only; an older release stops receiving security updates the moment the next release ships. Stated in [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md#supported-versions) ([#1060](https://github.com/genealogix/glx/issues/1060)) |

#### Release signing verification details

Verification command:

`cosign verify-blob --bundle checksums.txt.sigstore.json --certificate-identity-regexp '^https://github\.com/genealogix/glx/\.github/workflows/release\.yml@refs/tags/' --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' checksums.txt`

The `--certificate-*` flags constrain which OIDC identity the Fulcio certificate was issued to and which OIDC provider issued the underlying token, binding verification to this repository's release workflow. Without these constraints, `verify-blob` would still validate the Fulcio chain, signature, and Rekor inclusion proof, but it would accept any valid keyless signature from any workflow.

#### Build-provenance verification details

Verification command (run against any downloaded release archive):

`gh attestation verify --owner genealogix <downloaded-file>`

This checks the artifact's SHA-256 against the SLSA provenance attestation GitHub stores for this repository, confirming the binary was produced by `release.yml` running in GitHub Actions. The attestation is signed keyless via the workflow's OIDC token (`id-token: write`) and stored with the `attestations: write` permission. It is complementary to the cosign signature above: cosign proves the integrity of the `checksums.txt` manifest, while the attestation proves the build provenance of each artifact per SLSA.

#### SBOM verification details

Every release cut from the next tag onward publishes one SPDX-JSON SBOM per archive, named after the archive it describes — for example `glx_Linux_x86_64.tar.gz.sbom.json` accompanies `glx_Linux_x86_64.tar.gz`. Releases up to and including **`v0.0.0-beta.12`** predate the `sboms:` stanza and carry no `.sbom.json` assets, so the commands below match nothing against them: `sha256sum -c -` exits non-zero with `no properly formatted checksum lines found`. Because the SBOMs are listed in `checksums.txt`, verifying one is the same two-step flow as verifying an archive: check the manifest's signature, then check the file against the manifest.

```bash
# 1. Verify the checksum manifest's cosign signature (see the command above).
# 2. Check the downloaded SBOM against the verified manifest. Select its line
#    rather than running the whole manifest: a bare `-c checksums.txt` fails on
#    every artifact you did not download, and `--ignore-missing` would let the
#    command succeed even if the SBOM itself was never fetched.
grep ' glx_Linux_x86_64.tar.gz.sbom.json$' checksums.txt | sha256sum -c -
```

Inspect the contents with any SPDX-aware tool, for example:

`syft convert glx_Linux_x86_64.tar.gz.sbom.json -o table`

No separate `.sigstore.json` is published per SBOM: like the archives, the SBOMs inherit their integrity guarantee from the signed and attested `checksums.txt` manifest.

**Known limit — SBOMs are not byte-reproducible.** The release binaries and archives are reproducible from the tagged commit (`CGO_ENABLED=0`, `-trimpath`, and `.CommitDate`/`.CommitTimestamp`-pinned timestamps, [#1049](https://github.com/genealogix/glx/issues/1049)), but the SBOMs are not: syft's SPDX output embeds a randomly generated `documentNamespace` UUID and a `created` wall-clock timestamp, so rebuilding the same tag produces different SBOM bytes. Because the SBOMs are listed in `checksums.txt`, that manifest's `.sbom.json` lines differ on every build of the same tag even though its archive lines do not. A rebuild-and-compare audit must therefore compare the **archive** lines of the manifest, not the manifest as a whole. The signature and attestation are unaffected: each covers the manifest that the build it came from actually produced.

## Outstanding gaps

None. Every Level 1, Level 2, and Level 3 control tracked above is met as of 2026-09-18, when SBOM emission ([#269](https://github.com/genealogix/glx/issues/269)) closed the last gap. That control is configured rather than yet evidenced on a published release: it first applies to the next tag, and the assets of `v0.0.0-beta.12` and earlier do not include SBOMs.

This document is updated whenever a new gap opens — for example when a new OSPS Baseline version adds controls (see [Maintenance](#maintenance)).

## EU Cyber Resilience Act note

> **This section is informational and is not legal advice.** CRA applicability is nuanced and depends on how a downstream product packages, distributes, and monetises the components it embeds. Adopters should consult their own counsel for product classification under the CRA or any other regulatory regime.

GLX is a local CLI and Go library. There is no service, no telemetry, no auto-update channel, no network listener in normal operation. On its own, GLX is intended for use as a local CLI/library and is generally expected to fall outside the CRA's "product with digital elements" definition — but the project does not assert a definitive legal scope on adopters' behalf.

Adopters who **package or embed** GLX into a CRA-regulated product may cite this attestation as part of their own evidence pack. The OpenSSF maintains guidance on how the OSPS Baseline maps to CRA expectations: <https://openssf.org/public-policy/eu-cyber-resilience-act/>.

If you need an explicit statement for procurement or audit purposes that does not appear here (for example, a specific export-control classification or evidence packaging beyond what this file already cites), open a discussion on <https://github.com/genealogix/glx/discussions> rather than an issue — procurement-shaped questions tend to attract maintainer time better in that forum.

## Maintenance

- **Review cadence**: this document is reviewed at every minor release, whenever a tracked gap closes, and when a new OSPS Baseline version is published.
- **Pinned Baseline version**: 2026.02.19. Re-review on each new Baseline release to incorporate added or changed controls.
- **Last reviewed**: 2026-09-18.
