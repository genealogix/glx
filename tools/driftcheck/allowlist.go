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

// allowlistEntry mirrors one item of .claude/drift-allowlist.yaml. Only the
// fields the checker needs for matching are modeled; the file is the single
// source of truth for triaged, known drift (see its header for the contract).
type allowlistEntry struct {
	File          string `yaml:"file"`
	Symbol        string `yaml:"symbol"`
	Reason        string `yaml:"reason"`
	Added         string `yaml:"added"`
	Permanent     bool   `yaml:"permanent"`
	TrackingIssue int    `yaml:"tracking_issue"`
	GoType        string `yaml:"go_type"`
	SchemaType    string `yaml:"schema_type"`
	YamlTag       string `yaml:"yaml_tag"`
	SchemaPath    string `yaml:"schema_path"`
}

type allowlist struct {
	entries []allowlistEntry
}

// parseAllowlist decodes the allowlist YAML. An empty/whitespace document
// yields an empty (non-nil) allowlist.
func parseAllowlist(data []byte) (*allowlist, error) {
	var entries []allowlistEntry
	if len(strings.TrimSpace(string(data))) > 0 {
		if err := yaml.Unmarshal(data, &entries); err != nil {
			return nil, err
		}
	}

	return &allowlist{entries: entries}, nil
}

// partition splits findings into those an allowlist entry suppresses and those
// that survive. The matching entry for each suppressed finding is returned in
// parallel for reporting.
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

type suppression struct {
	finding finding
	entry   allowlistEntry
}

// match returns the first allowlist entry that suppresses f, if any.
func (a *allowlist) match(f *finding) (allowlistEntry, bool) {
	for i := range a.entries {
		e := &a.entries[i]
		if e.File == f.File && symbolMatches(e, f) {
			return *e, true
		}
	}

	return allowlistEntry{}, false
}

// symbolMatches implements per-symbol identity matching. An exact qualified
// `Type.Symbol` match always wins. Otherwise, within the SAME owning type, a
// match is allowed when:
//   - the entry's symbol leaf equals the finding's Go field name (f.Field), or
//   - the entry's symbol leaf equals the finding's yaml tag (f.YamlTag), or
//   - the entry's yaml_tag equals the finding's yaml tag.
//
// The aliasing between a Go field name and a yaml tag is therefore NOT
// automatic: a "missing Go field" finding (a schema property with no Go
// counterpart) carries only a yaml tag and no Go field name, so to suppress one
// whose property name differs from the intended Go field name the entry must
// either name the symbol by that property/yaml name or set `yaml_tag` (this is
// why the GLXFile.ImportMetadata allowlist entry sets `yaml_tag: metadata`).
func symbolMatches(e *allowlistEntry, f *finding) bool {
	if e.Symbol == f.Symbol {
		return true
	}
	eType, eName := splitSymbol(e.Symbol)
	fType, _ := splitSymbol(f.Symbol)
	if eType != fType {
		return false
	}
	if f.Field != "" && eName == f.Field {
		return true
	}
	if f.YamlTag != "" && (eName == f.YamlTag || e.YamlTag == f.YamlTag) {
		return true
	}

	return false
}

// splitSymbol divides "Type.Field" into ("Type", "Field"). A bare symbol
// returns ("", symbol).
func splitSymbol(symbol string) (owner, name string) {
	if i := strings.LastIndex(symbol, "."); i >= 0 {
		return symbol[:i], symbol[i+1:]
	}

	return "", symbol
}
