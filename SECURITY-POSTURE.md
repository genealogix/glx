---
title: Security Posture
description: GLX self-attestation against the OpenSSF OSPS Baseline and notes on EU CRA readiness.
layout: doc
---

# Security Posture

> As of **2026-05-07**, GLX self-attests to the [OpenSSF OSPS Baseline](https://baseline.openssf.org/) version **2026.02.19** at **Level 1 (foundational)**, with most **Level 2 (operationally mature)** controls also met. Outstanding gaps are listed below and tracked as open issues.

This document is the public-facing companion to [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md). SECURITY.md tells you how to *report* a vulnerability; this file tells you how the project handles supply-chain and process risk so adopters can make informed decisions about depending on GLX.

## Why this exists

Two converging frameworks shape the credibility expectations for open-source projects in 2026:

- **OpenSSF OSPS Baseline** — a structured set of controls organized into three maturity levels (L1 foundational → L3 mature) that adopters and procurement teams reference when evaluating dependencies. Published at <https://baseline.openssf.org/>.
- **EU Cyber Resilience Act (CRA)** — partial enforcement begins **2026-09-11**. GLX itself is a CLI tool processing local YAML files and is not a "product with digital elements" under CRA scope, but downstream products that embed GLX may need this posture document to assemble their own CRA evidence pack. The OSPS Baseline was designed in part as a compliance bridge for that mapping.

GLX prioritizes the Baseline because doing so produces verifiable evidence (CI configurations, documented policies, signed releases) that adopters can cite directly without bespoke questionnaires.

## Current status

Status legend: ✓ met · ◐ partial · ☐ not yet met

### Level 1 — Foundational

| Control | Status | Evidence |
|---|---|---|
| Security contacts documented | ✓ | [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md) — GitHub Security Advisories with a 48-hour acknowledgment SLA |
| Contribution process explained | ✓ | [CONTRIBUTING.md](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md) covers setup, branch naming, conventional commits, DCO sign-off, PR template |
| Direct commits to primary branch prevented | ✓ | Maintainer-verified as of 2026-05-07: an active **Main Protection** ruleset on the default branch requires PRs, blocks force-push and deletion, and gates merge on the workflows listed in the row below. The ruleset is org-internal (not in this repo); the workflows it gates are public under [`.github/workflows/`](https://github.com/genealogix/glx/tree/main/.github/workflows) |
| MFA for sensitive resource access | ✓ | Two-factor authentication is enforced organization-wide on the [`genealogix`](https://github.com/genealogix) GitHub organization |

### Level 2 — Operationally Mature

| Control | Status | Evidence |
|---|---|---|
| Coordinated vulnerability disclosure policy | ◐ | [SECURITY.md](https://github.com/genealogix/glx/blob/main/SECURITY.md) defines reporting channel, acknowledgment, severity classification, and fix timelines. Safe-harbor language and embargo policy are tracked in [#424](https://github.com/genealogix/glx/issues/424) |
| Contributors assert legal authorization (DCO/CLA) | ✓ | [Developer Certificate of Origin 1.1](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md#developer-certificate-of-origin-dco) sign-off required on every commit. Verification is currently manual at review time; an automated DCO check is a possible future enhancement |
| Automated test suite runs before merge | ✓ | Six workflows run on every `pull_request` and are expected to be green before merge: [Validate Specification](https://github.com/genealogix/glx/blob/main/.github/workflows/validate-spec.yml), [Lint](https://github.com/genealogix/glx/blob/main/.github/workflows/lint.yml), [Security](https://github.com/genealogix/glx/blob/main/.github/workflows/security.yml), [Lint Markdown](https://github.com/genealogix/glx/blob/main/.github/workflows/lint-markdown.yml), [Dependency Review](https://github.com/genealogix/glx/blob/main/.github/workflows/dependency-review.yml), and [Lint PR Title](https://github.com/genealogix/glx/blob/main/.github/workflows/lint-pr-title.yml). The Main Protection ruleset (see L1 row above) gates merge on these |
| Static analysis of dependencies and code | ✓ | [`security.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/security.yml) runs `govulncheck` (known-vulnerable Go dependencies) and `gosec` (Go static security analysis) on every PR and weekly. [`dependency-review.yml`](https://github.com/genealogix/glx/blob/main/.github/workflows/dependency-review.yml) blocks PRs that introduce known-vulnerable dependencies. **CodeQL** runs via [GitHub Default Setup](https://docs.github.com/en/code-security/code-scanning/enabling-code-scanning/configuring-default-setup-for-code-scanning) across `actions`, `go`, `javascript-typescript`, and `python`; findings surface in Code Scanning |
| Continuous security posture scoring | ✓ | [OpenSSF Scorecard](https://github.com/genealogix/glx/blob/main/.github/workflows/scorecard.yml) runs on push to `main`, weekly on Mondays, and on `workflow_dispatch`. Results are published to the public OSSF dataset and uploaded to GitHub Code Scanning as SARIF |

### Level 3 — Mature

| Control | Status | Evidence |
|---|---|---|
| SBOM with compiled releases | ☐ | Tracked in [#269](https://github.com/genealogix/glx/issues/269) — GoReleaser v2 native SBOM via `sboms:` config |
| Build provenance / SLSA attestations | ☐ | Tracked in [#256](https://github.com/genealogix/glx/issues/256) |

## Outstanding gaps

The following issues are blocking full Level 2 / Level 3 self-attestation:

- [#424](https://github.com/genealogix/glx/issues/424) — Safe-harbor clause and embargo policy in `SECURITY.md` (Level 2)
- [#269](https://github.com/genealogix/glx/issues/269) — SBOM emission alongside compiled releases (Level 3)
- [#256](https://github.com/genealogix/glx/issues/256) — SLSA build provenance attestations (Level 3)

This document is updated when any of these close.

## EU Cyber Resilience Act note

> **This section is informational and is not legal advice.** CRA applicability is nuanced and depends on how a downstream product packages, distributes, and monetises the components it embeds. Adopters should consult their own counsel for product classification under the CRA or any other regulatory regime.

GLX is a local CLI and Go library. There is no service, no telemetry, no auto-update channel, no network listener in normal operation. On its own, GLX is intended for use as a local CLI/library and is generally expected to fall outside the CRA's "product with digital elements" definition — but the project does not assert a definitive legal scope on adopters' behalf.

Adopters who **package or embed** GLX into a CRA-regulated product may cite this attestation as part of their own evidence pack. The OpenSSF maintains guidance on how the OSPS Baseline maps to CRA expectations: <https://openssf.org/public-policy/eu-cyber-resilience-act/>.

If you need an explicit statement for procurement or audit purposes that does not appear here (for example, a specific export-control classification or evidence packaging beyond what this file already cites), open a discussion on <https://github.com/genealogix/glx/discussions> rather than an issue — procurement-shaped questions tend to attract maintainer time better in that forum.

## Maintenance

- **Review cadence**: this document is reviewed at every minor release, when any tracked gap (#424, #269, #256) closes, and when a new OSPS Baseline version is published.
- **Pinned Baseline version**: 2026.02.19. Re-review on each new Baseline release to incorporate added or changed controls.
- **Last reviewed**: 2026-05-07.
