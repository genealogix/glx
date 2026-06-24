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
	"strings"

	"gopkg.in/yaml.v3"
)

// allowEntry mirrors one item of .claude/memory-drift-allowlist.yaml: a
// deliberately-documented reference that the deterministic check would otherwise
// flag (e.g. an illustrative path that is not meant to exist). File and Kind are
// optional narrowing filters; Reason is for humans only.
type allowEntry struct {
	Token  string `yaml:"token"`
	File   string `yaml:"file"`
	Kind   string `yaml:"kind"`
	Reason string `yaml:"reason"`
}

type allowlist struct {
	entries []allowEntry
}

// parseAllowlist decodes the allowlist YAML (a top-level list of entries). An
// empty or whitespace-only document yields an empty, non-nil allowlist.
func parseAllowlist(data []byte) (*allowlist, error) {
	var entries []allowEntry
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := yaml.Unmarshal(data, &entries); err != nil {
			return nil, err
		}
	}

	return &allowlist{entries: entries}, nil
}

type suppression struct {
	finding finding
	entry   allowEntry
}

// partition splits findings into those an allowlist entry suppresses and those
// that survive, returning the matching entry alongside each suppressed finding
// for reporting.
func (a *allowlist) partition(findings []finding) (surviving []finding, suppressed []suppression) {
	surviving = make([]finding, 0, len(findings))
	for i := range findings {
		f := findings[i]
		if entry, ok := a.match(&f); ok {
			suppressed = append(suppressed, suppression{finding: f, entry: entry})

			continue
		}
		surviving = append(surviving, f)
	}

	return surviving, suppressed
}

// match returns the first allowlist entry that suppresses f, if any. An entry
// matches when its Token equals the finding's Token and its optional File and
// Kind filters (when set) also match.
func (a *allowlist) match(f *finding) (allowEntry, bool) {
	for i := range a.entries {
		e := &a.entries[i]
		if e.Token != f.Token {
			continue
		}
		if e.File != "" && e.File != f.File {
			continue
		}
		if e.Kind != "" && e.Kind != string(f.Kind) {
			continue
		}

		return *e, true
	}

	return allowEntry{}, false
}
