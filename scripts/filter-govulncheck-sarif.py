#!/usr/bin/env python3
"""Drop govulncheck SARIF findings that osv-scanner.toml already suppresses.

osv-scanner.toml is the repository's single reviewed list of advisories that do
not apply to GLX (see its header and SECURITY-POSTURE.md, "Vulnerability
suppression"). osv-scanner and the Scorecard Vulnerabilities check read it, but
govulncheck has no ignore mechanism, so without this filter every suppressed
advisory still surfaces as a Code Scanning alert from the govulncheck upload.

Only module-level findings are dropped: govulncheck reports an advisory at
level "note" when a vulnerable module is required but none of its vulnerable
packages are imported. A suppressed advisory reported at "warning" (package
imported) or "error" (symbol called) is kept, because that means the reasoning
recorded in osv-scanner.toml no longer holds.

Usage:
    python3 scripts/filter-govulncheck-sarif.py osv-scanner.toml < in.sarif > out.sarif
"""

import json
import sys
import tomllib

# The only SARIF level this filter may drop; see the module docstring.
DROPPABLE_LEVEL = "note"


def suppressed_ids(config_path):
    with open(config_path, "rb") as f:
        config = tomllib.load(f)
    return {entry["id"] for entry in config.get("IgnoredVulns", [])}


def main():
    if len(sys.argv) != 2:
        sys.exit(__doc__)
    ignored = suppressed_ids(sys.argv[1])
    sarif = json.load(sys.stdin)

    for run in sarif.get("runs", []):
        kept = []
        for result in run.get("results", []):
            rule = result.get("ruleId")
            if rule in ignored and result.get("level") == DROPPABLE_LEVEL:
                print(f"filtered {rule} (module-level; suppressed in {sys.argv[1]})", file=sys.stderr)
                continue
            if rule in ignored:
                print(f"kept {rule}: level {result.get('level')!r} means the suppression no longer holds", file=sys.stderr)
            kept.append(result)
        # The rules table is left as is: a rule with no result raises no alert,
        # and pruning it could invalidate a result's ruleIndex.
        run["results"] = kept

    json.dump(sarif, sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
