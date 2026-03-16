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

## Security Measures

- **govulncheck** runs in CI on every push and PR to detect known vulnerabilities in dependencies
- **gosec** performs static security analysis on every push and PR
- Weekly scheduled scans catch newly disclosed vulnerabilities in existing dependencies

## Scope

This policy covers the GLX CLI tool and the go-glx library. GLX archives are YAML files processed locally — there is no network-facing attack surface in normal usage.
