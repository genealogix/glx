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
	"sort"
)

// severity orders findings for reporting and gating. memcheck only emits major
// findings today (every check is a concrete, deterministic correctness claim),
// but the scale mirrors the shared severity rubric used by the check-* suite so
// the report format and any future advisory checks stay consistent.
type severity string

const (
	severityMajor severity = "major"
	severityMinor severity = "minor"
	severityInfo  severity = "info"
)

// rank orders severities most-to-least serious for stable sorting.
func (s severity) rank() int {
	switch s {
	case severityMajor:
		return 0
	case severityMinor:
		return 1
	default:
		return 2
	}
}

// fails reports whether a finding of this severity should fail the check. A
// stale path, an unknown make target, or an import path that no longer matches
// go.mod is a correctness problem for any agent that follows the instruction,
// so all three are major and gating.
func (s severity) fails() bool {
	return s == severityMajor
}

// kind names the assertion that drifted, for grouping and allowlist matching.
type kind string

const (
	kindPath       kind = "stale-path"
	kindMakeTarget kind = "unknown-make-target"
	kindImportPath kind = "import-path-drift"
)

// finding is one detected mismatch between a memory file's assertion and the
// repository on disk.
type finding struct {
	// File is the memory file the reference lives in, repo-relative with forward
	// slashes (e.g. "CLAUDE.md", "go-glx/CLAUDE.md"). Used for display and
	// allowlist matching.
	File string
	// Line is the 1-based line of the reference in File.
	Line int
	// Kind is the category of drift (path / make target / import path).
	Kind kind
	// Severity gates the exit code; see severity.fails.
	Severity severity
	// Token is the offending reference, verbatim, used for display and allowlist
	// matching (e.g. "testdata/gedcom/shakespeare.ged", "make frobnicate").
	Token string
	// Message is a human-readable explanation of the drift.
	Message string
}

// byFileLineToken sorts findings for stable, readable report output.
func byFileLineToken(findings []finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		if findings[i].Severity.rank() != findings[j].Severity.rank() {
			return findings[i].Severity.rank() < findings[j].Severity.rank()
		}

		return findings[i].Token < findings[j].Token
	})
}

// dedupe drops exact-duplicate findings, preserving first-seen order. The same
// stale path can legitimately appear on several lines (distinct findings), but a
// single line can also yield the same token twice via overlapping scans; this
// keeps the report honest without collapsing genuinely distinct lines.
func dedupe(findings []finding) []finding {
	seen := make(map[finding]struct{}, len(findings))
	out := make([]finding, 0, len(findings))
	for i := range findings {
		if _, ok := seen[findings[i]]; ok {
			continue
		}
		seen[findings[i]] = struct{}{}
		out = append(out, findings[i])
	}

	return out
}

// hasFailures reports whether any finding is at a gating severity.
func hasFailures(findings []finding) bool {
	for i := range findings {
		if findings[i].Severity.fails() {
			return true
		}
	}

	return false
}
