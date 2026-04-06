# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.x.x   | :white_check_mark: |

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

- **govulncheck** runs in CI on pushes to main, pull requests, and weekly to detect known vulnerabilities in dependencies
- **gosec** performs static security analysis on pushes to main, pull requests, and weekly
- Weekly scheduled scans catch newly disclosed vulnerabilities in existing dependencies

## Scope

This policy covers the GLX CLI tool and the go-glx library. GLX archives are YAML files processed locally — there is no network-facing attack surface in normal usage.
