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

// Finding categories. catInternal marks findings about the checker itself
// (e.g. an unresolved schema $ref) rather than genuine schema/code drift.
const (
	catInternal      = "Internal"
	catFieldPresence = "Field Presence"
	catFieldTypes    = "Field Types"
	catRequiredOpt   = "Required vs Optional"
)

// severity mirrors the Severity Rubric in .claude/skills/check-code-drift/SKILL.md.
type severity string

const (
	severityCritical severity = "critical"
	severityMajor    severity = "major"
	severityMinor    severity = "minor"
	severityInfo     severity = "info"
)

// rank orders severities most-to-least serious for sorting and gating.
func (s severity) rank() int {
	switch s {
	case severityCritical:
		return 0
	case severityMajor:
		return 1
	case severityMinor:
		return 2
	default:
		return 3
	}
}

// fails reports whether a finding of this severity should fail the check.
// Structural drift (critical/major) breaks marshaling or correctness and is
// gated; minor/info are reported but advisory.
func (s severity) fails() bool {
	return s == severityCritical || s == severityMajor
}

// finding is one detected discrepancy between a Go type and a schema.
type finding struct {
	Severity severity
	Entity   string // schema title / Go type group, e.g. "Event"
	Category string // e.g. "Field Presence", "YAML Tags"
	// File is the source file the drift lives in (always go-glx/types.go for
	// this checker) — used for allowlist matching.
	File string
	// Symbol is the qualified Go symbol, e.g. "Event.Title". Used for both
	// display and allowlist matching.
	Symbol string
	// Field is the bare Go field name; YamlTag is its yaml key. Both feed
	// allowlist identity matching (an entry may name a field by either).
	Field   string
	YamlTag string
	Message string
	// Line is the 1-based source line of the field in File, attached after the
	// reflection-based comparison by the AST extractor (structdump, #795).
	// Reflection cannot give source positions; 0 means unknown.
	Line int
}

// byEntityThenSeverity sorts findings for stable, readable report output.
func byEntityThenSeverity(findings []finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Entity != findings[j].Entity {
			return findings[i].Entity < findings[j].Entity
		}
		if findings[i].Severity.rank() != findings[j].Severity.rank() {
			return findings[i].Severity.rank() < findings[j].Severity.rank()
		}

		return findings[i].Symbol < findings[j].Symbol
	})
}

// dedupeFindings drops exact-duplicate findings, preserving first-seen order. A
// nested type reachable from several parents (e.g. Participant via Event,
// Relationship, and Assertion) is compared once per parent and — now that
// findings are keyed by the owning Go type rather than the parent — yields
// identical findings each time; collapsing them keeps the report honest.
func dedupeFindings(findings []finding) []finding {
	seen := make(map[finding]struct{}, len(findings))
	out := make([]finding, 0, len(findings))
	for i := range findings {
		if _, dup := seen[findings[i]]; dup {
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
