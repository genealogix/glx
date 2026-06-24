// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"strings"
)

// buildReport renders the human-readable memory-drift report. It is pure
// (returns a string) so callers do a single write and tests can assert on text.
// findings are expected pre-sorted by byFileLineToken. allowlistPath is the
// effective allowlist file (the -allowlist value, default
// .claude/memory-drift-allowlist.yaml) so the report points at the file actually
// in use rather than a hard-coded default.
func buildReport(memFiles int, allowlistPath string, findings []finding, suppressed []suppression, verbose bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "memcheck: checked %d memory file(s) (CLAUDE.md/AGENTS.md) against the repository\n", memFiles)

	if verbose && len(suppressed) > 0 {
		fmt.Fprintf(&b, "\nSuppressed by %s (%d):\n", allowlistPath, len(suppressed))
		for i := range suppressed {
			s := &suppressed[i]
			fmt.Fprintf(&b, "  - %s:%d [%s] %s — %s\n",
				s.finding.File, s.finding.Line, s.finding.Kind, s.finding.Token, s.entry.Reason)
		}
	}

	if len(findings) == 0 {
		suffix := ""
		if len(suppressed) > 0 {
			suffix = fmt.Sprintf(" (%d finding(s) suppressed by allowlist)", len(suppressed))
		}
		fmt.Fprintf(&b, "\n✅ No memory drift detected%s.\n", suffix)

		return b.String()
	}

	fmt.Fprintf(&b, "\n⚠️  Memory drift detected: %d finding(s)\n", len(findings))

	counts := map[severity]int{}
	currentFile := ""
	for i := range findings {
		f := &findings[i]
		counts[f.Severity]++
		if f.File != currentFile {
			currentFile = f.File
			fmt.Fprintf(&b, "\n## %s\n", currentFile)
		}
		fmt.Fprintf(&b, "- **%s** [%s] line %d: %s\n", f.Severity, f.Kind, f.Line, f.Message)
	}

	fmt.Fprintf(&b, "\nSummary: %d major, %d minor, %d info",
		counts[severityMajor], counts[severityMinor], counts[severityInfo])
	if len(suppressed) > 0 {
		fmt.Fprintf(&b, " (%d suppressed by allowlist)", len(suppressed))
	}
	fmt.Fprintf(&b, "\n\nThe repository is the source of truth — update the memory file to match,\n"+
		"or add a triaged entry to %s if the\n"+
		"reference is an intentional, non-existent example.\n", allowlistPath)

	return b.String()
}
