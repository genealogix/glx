# Temporary picomatch security backport

The website toolchain currently resolves picomatch 4.0.7. There is no published
fixed release for [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf).
The local patch applies only the parser changes from
[upstream PR #262](https://github.com/micromatch/picomatch/pull/262), pinned at
`3222059ac27492d9c0147fee5c65f751bbd851ab`, to the published 4.0.7 parser.
It does not pull in unrelated unreleased upstream changes. The upstream MIT
license is preserved in `picomatch-LICENSE`.

`npm ci`/`npm install` applies the backport through `postinstall` to every
installed picomatch copy at 4.0.7. For such a copy the installer checks the
exact original and patched SHA-256 hashes, is idempotent, and fails closed on
any other content. A copy at another version, or one not installed (for example
`npm ci --omit=dev`), is left alone and does not break the install: the audit
gate refuses the exception for any picomatch version other than 4.0.7, so a
version or layout change is reviewed there. Do not use `--ignore-scripts` for the website
without subsequently running `npm run postinstall`. `npm run test:security`
verifies the installed patch, supported ordinary matching, and bounded
adversarial subprocesses across matcher, makeRe and parse/compileRe APIs.
These are finite regression tests, not a proof for arbitrary regular expressions.

This is an unmerged upstream candidate, not an official patched release. Its
unsupported complex patterns and capture forms can fall back to literal matching;
`maxExtglobRecursion: false` explicitly opts out of the protection. The website
uses developer-controlled build patterns, not a public user-supplied glob API.
The patch is a temporary defense for the affected development toolchain.

The package version is deliberately unchanged, so raw npm audit and dependency
scanners still report the advisory, including transitive findings. The CI audit
gate accepts it through `../audit-accepted.json`, for picomatch 4.0.7 only,
until the entry's `ignoreUntil` date and only while `npm run test:security`
passes (see SECURITY-POSTURE.md, "Vulnerability suppression"); any other
advisory still fails the gate (see `../../scripts/npm-audit-gate.mjs`). Extend
`ignoreUntil` only after re-checking that no fixed release exists. Replace this
backport with an upstream fixed release once available and verified, then
remove the installer, patch, temporary regression tests and the
`audit-accepted.json` entry together. This installation currently has one root
picomatch instance; a nested 4.0.7 copy would be patched too.
