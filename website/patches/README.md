# Temporary picomatch security backport

The website toolchain currently resolves picomatch 4.0.7. There is no published
fixed release for [GHSA-fc5q-39xc-vjpf](https://github.com/advisories/GHSA-fc5q-39xc-vjpf).
The local patch applies only the parser changes from
[upstream PR #262](https://github.com/micromatch/picomatch/pull/262), pinned at
`3222059ac27492d9c0147fee5c65f751bbd851ab`, to the published 4.0.7 parser.
It does not pull in unrelated unreleased upstream changes. The upstream MIT
license is preserved in `picomatch-LICENSE`.

`npm ci`/`npm install` applies the backport through `postinstall`. The installer
checks the exact original and patched SHA-256 hashes, is idempotent, and fails
closed if the dependency changes. Do not use `--ignore-scripts` for the website
without subsequently running `npm run postinstall`. `npm run test:security`
verifies the installed patch, supported ordinary matching, and bounded
adversarial subprocesses across matcher, makeRe and parse/compileRe APIs.
These are finite regression tests, not a proof for arbitrary regular expressions.

This is an unmerged upstream candidate, not an official patched release. Its
unsupported complex patterns and capture forms can fall back to literal matching;
`maxExtglobRecursion: false` explicitly opts out of the protection. The website
uses developer-controlled build patterns, not a public user-supplied glob API.
The patch is a temporary defense for the affected development toolchain.

The package version is deliberately unchanged. Raw npm audit and dependency
scanners still report the advisory, including transitive findings. No audit
exception or severity downgrade is included. Replace this backport with an
upstream fixed release once available and verified, then remove the installer,
patch, and temporary regression tests together. Review dependency-tree changes
for nested copies: this installation currently has one root picomatch instance.
