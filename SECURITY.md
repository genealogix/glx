# Security Policy

> For supply-chain controls and OSPS Baseline self-attestation, see [SECURITY-POSTURE](/development/security-posture).

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

GLX is pre-1.0: security fixes target the **latest 0.x release only**. An older
release stops receiving security updates the moment the next release ships —
upgrade to the newest release to stay covered.

## Reporting a Vulnerability

If you discover a security vulnerability in GLX, please report it responsibly:

1. **Do not** open a public GitHub issue for security vulnerabilities
2. Report via [GitHub Security Advisories](https://github.com/genealogix/glx/security/advisories/new)
3. Include a description of the vulnerability, steps to reproduce, and potential impact

## What to expect

- **Acknowledgment** within 48 hours of your report
- **Assessment** within 1 week — we'll confirm the vulnerability and its severity
- **Fix timeline** depends on severity:
  - Critical: patch release within 72 hours
  - High: patch release within 1 week
  - Medium/Low: included in next scheduled release

## Safe Harbor

We want security researchers to feel safe reporting vulnerabilities in GLX. We will not pursue or support legal action against anyone who discovers and reports a vulnerability in good faith and substantially in accordance with this policy — and that protection still applies to accidental, good-faith deviations from it. For such research, we consider it:

- **Authorized** under applicable anti-hacking laws (such as the US Computer Fraud and Abuse Act and equivalents elsewhere); and
- **Lawful**, helpful to the overall security of the project and its users, and conducted in good faith.

If a third party initiates legal action against you for activity conducted in good faith and substantially in accordance with this policy — including accidental, good-faith deviations from it — we will, to the extent we reasonably can, make it publicly known that your actions were authorized under it.

To stay within this safe harbor, please:

- act in good faith, and avoid privacy violations, data destruction, and disruption to others;
- only interact with archives, files, and systems you own or are explicitly authorized to test — GLX is a local tool, so in practice this is your own machine and your own data;
- use a vulnerability only to the extent necessary to confirm it, and never to access, modify, or exfiltrate data that is not yours; and
- report promptly through the channel above and allow a reasonable opportunity to fix the issue before public disclosure (see [Coordinated Disclosure and Embargo](#coordinated-disclosure-and-embargo) below).

This language is adapted from the [disclose.io](https://disclose.io/) Core Terms, a widely used open-source safe-harbor framework. Many major programs — including GitHub, GitLab, and Cloudflare — extend comparable good-faith safe-harbor protections through their own policies or platform frameworks (for example, HackerOne's Gold Standard Safe Harbor) rather than identical text. If you are unsure whether a particular action is authorized, ask first in a private report before proceeding.

## Coordinated Disclosure and Embargo

GLX follows coordinated (formerly "responsible") disclosure: we ask that you keep vulnerability details private until a fix is available and an advisory is published, and in return we commit to fixing reported issues promptly, keeping you informed, and crediting your work.

**Disclosure timeline.** We publish the advisory when the fix ships, which — given the [fix timelines above](#what-to-expect) — is normally within 72 hours for Critical issues and within a week for High-severity ones. We treat **90 days** from the initial report as a backstop, not a hard auto-publish date: it is the widely used industry default (e.g. [Google Project Zero](https://googleprojectzero.blogspot.com/p/vulnerability-disclosure-policy.html)) beyond which a vulnerability should not stay embargoed indefinitely. If a fix legitimately needs longer than 90 days, we will agree on a revised disclosure date with you rather than either publishing without a remedy or letting the embargo lapse silently.

**Who is informed during the embargo.** GLX is a small project. While an issue is embargoed, the details are shared only with:

- the project maintainers, via the private [GitHub Security Advisory](https://github.com/genealogix/glx/security/advisories) draft; and
- you, the reporter, with whom we coordinate on the draft, the fix, and the disclosure date.

We do not operate a standing pre-notification list for downstream consumers. If a specific vulnerability warrants early warning to a known integrator, we will arrange that privately on a case-by-case basis.

**Conduct during the embargo.** While an advisory is embargoed, we ask that you:

- do not publicly disclose the vulnerability — including in blog posts, social media, conference talks, or public issues and pull requests — until the advisory is published;
- do not share the details with third parties outside the coordination described above; and
- do not exploit the vulnerability beyond what is necessary to demonstrate it, and never against data or systems that are not your own.

In return, the maintainers commit to acknowledging your report within the SLA above, keeping you updated on remediation progress, declining to pursue legal action consistent with the [Safe Harbor](#safe-harbor) section, and crediting you in the advisory and release notes unless you ask to remain anonymous.

**How downstream consumers are notified.** When the fix ships, we publish the GitHub Security Advisory and request a CVE ID through GitHub (a CVE Numbering Authority). The published advisory is listed in the [GitHub Advisory Database](https://github.com/advisories) and, for the `go-glx` module, becomes discoverable through the [Go vulnerability database](https://pkg.go.dev/vuln/) — so downstream consumers of the library and the `glx` CLI are alerted by `govulncheck`, `dependency-review`, and Dependabot, the same tooling [GLX runs on itself](#vulnerability-scanning). Each advisory names the affected versions and the first fixed version, and the fix is also called out in [`CHANGELOG.md`](https://github.com/genealogix/glx/blob/main/CHANGELOG.md).

## Severity Classification

We use severity levels (Critical / High / Medium / Low) to prioritize fix timelines, but we do not use CVSS scores to determine them.

**Why not CVSS?** The Go security team [explicitly rejects CVSS scoring](https://go.dev/security/policy) because the formula doesn't map well to real-world exploitability in many Go projects. The curl project [takes the same position](https://raw.githubusercontent.com/curl/curl/master/docs/VULN-DISCLOSURE-POLICY.md). We agree with this assessment.

Instead, severity is determined by a practical reading of:

- **Exploitability** — how easy is it to trigger the vulnerability? Does it require local access, specific file inputs, or a contrived scenario?
- **Impact** — what's the realistic worst case? Data loss, corruption, information disclosure, or denial of service?
- **Affected surface** — GLX is a local CLI tool processing YAML files. There is no server, no network listener, no multi-user context. Most vulnerabilities are limited to the trust boundary of the files a user chooses to process.

**Severity definitions:**

| Level    | Typical criteria |
|----------|-----------------|
| Critical | Arbitrary file write or complete data corruption via crafted archive, or complete bypass of file integrity checks |
| High     | Significant data corruption or exfiltration possible with a malicious archive file |
| Medium   | Denial of service, path traversal with limited impact, or logic errors affecting correctness |
| Low      | Edge-case bugs with minimal real-world impact, or issues requiring unusual preconditions |

These are guidelines, not a formula. We'll explain our severity reasoning in each advisory.

## Bug Bounty

GLX does not offer a bug bounty program. There is no financial reward for vulnerability reports.

We appreciate responsible disclosure and will credit reporters in release notes when a fix ships, but we cannot commit to monetary compensation. This policy exists to avoid ambiguity — silence on the topic is not an implicit promise of payment.

If you're evaluating whether to report: please do. The project benefits from security research regardless of bounty.

## Security Measures

### Vulnerability Scanning

- **govulncheck** — scans Go dependencies for known CVEs (CI + weekly)
- **gosec** — static analysis for common Go security issues (CI + weekly)
- **npm audit** — checks website dependencies for vulnerabilities (CI + weekly)

### Dependency Management

- **Dependabot** — daily automated dependency updates for Go, npm, and GitHub Actions
- **Dependency review** — blocks PRs that introduce dependencies with moderate+ vulnerabilities

### Code Scanning

- **GitHub Code Scanning** — receives govulncheck SARIF uploads for triage

## Scope

This policy covers the GLX CLI tool and the go-glx library. GLX archives are YAML files processed locally — there is no network-facing attack surface in normal usage.
