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

// buildReport renders the human-readable drift report. It is pure (returns a
// string) so callers do a single write and tests can assert on the text.
func buildReport(surviving []finding, suppressed []suppression, schemaFiles int, verbose bool) string {
	parts := []string{
		fmt.Sprintf("driftcheck: compared go-glx types against %d schema files\n", schemaFiles),
	}

	if verbose && len(suppressed) > 0 {
		parts = append(parts, fmt.Sprintf("\nSuppressed by .claude/drift-allowlist.yaml (%d):\n", len(suppressed)))
		for i := range suppressed {
			s := &suppressed[i]
			parts = append(parts, fmt.Sprintf("  - [%s] %s — %s\n", s.finding.Severity, s.finding.Symbol, s.entry.Reason))
		}
	}

	if len(surviving) == 0 {
		parts = append(parts, fmt.Sprintf("\n✅ No drift detected (%d finding(s) suppressed by allowlist).\n", len(suppressed)))

		return strings.Join(parts, "")
	}

	parts = append(parts, fmt.Sprintf("\n⚠️  Drift detected: %d finding(s)\n", len(surviving)))

	counts := map[severity]int{}
	var currentEntity string
	for i := range surviving {
		f := &surviving[i]
		counts[f.Severity]++
		if f.Entity != currentEntity {
			currentEntity = f.Entity
			parts = append(parts, fmt.Sprintf("\n## %s\n", currentEntity))
		}
		loc := ""
		if f.Line > 0 {
			loc = fmt.Sprintf(" (%s:%d)", f.File, f.Line)
		}
		parts = append(parts, fmt.Sprintf("- **%s** [%s] %s%s\n", f.Severity, f.Category, f.Message, loc))
	}

	summary := fmt.Sprintf("\nSummary: %d critical, %d major, %d minor, %d info",
		counts[severityCritical], counts[severityMajor], counts[severityMinor], counts[severityInfo])
	if len(suppressed) > 0 {
		summary += fmt.Sprintf(" (%d suppressed by allowlist)", len(suppressed))
	}
	parts = append(parts, summary,
		"\n\nSchemas are the source of truth — update go-glx/types.go to match,\n",
		"or add a triaged entry to .claude/drift-allowlist.yaml if the drift is by design.\n")

	return strings.Join(parts, "")
}
