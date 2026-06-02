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
	"testing"
)

func TestSeverityRankAndFails(t *testing.T) {
	order := []severity{severityCritical, severityMajor, severityMinor, severityInfo}
	for i := 1; i < len(order); i++ {
		if order[i-1].rank() >= order[i].rank() {
			t.Errorf("rank not increasing: %s !< %s", order[i-1], order[i])
		}
	}
	if !severityCritical.fails() || !severityMajor.fails() {
		t.Error("critical/major must gate")
	}
	if severityMinor.fails() || severityInfo.fails() {
		t.Error("minor/info must not gate")
	}
}

func TestHasFailures(t *testing.T) {
	if hasFailures([]finding{{Severity: severityMinor}, {Severity: severityInfo}}) {
		t.Error("minor+info should not fail")
	}
	if !hasFailures([]finding{{Severity: severityInfo}, {Severity: severityMajor}}) {
		t.Error("a major should fail")
	}
}

func TestByEntityThenSeverity(t *testing.T) {
	findings := []finding{
		{Entity: "Beta", Severity: severityMajor, Symbol: "Beta.b"},
		{Entity: "Beta", Severity: severityCritical, Symbol: "Beta.a"},
		{Entity: "Alpha", Severity: severityInfo, Symbol: "Alpha.z"},
	}
	byEntityThenSeverity(findings)
	if findings[0].Entity != "Alpha" {
		t.Fatalf("Alpha should sort first, got %s", findings[0].Entity)
	}
	if findings[1].Severity != severityCritical || findings[2].Severity != severityMajor {
		t.Fatalf("within an entity, critical should precede major: %+v", findings[1:])
	}
}

func TestBuildReportDriftPath(t *testing.T) {
	findings := []finding{
		{Severity: severityCritical, Entity: "Beta", Category: catFieldPresence, Message: "m1"},
		{Severity: severityMajor, Entity: "Beta", Category: catFieldTypes, Message: "m2"},
		{Severity: severityMinor, Entity: "Gamma", Category: "X", Message: "m3"},
		{Severity: severityInfo, Entity: "Gamma", Category: "Y", Message: "m4"},
	}
	supp := []suppression{{
		finding: finding{Severity: severityMajor, Symbol: "A.b"},
		entry:   allowlistEntry{Reason: "by design"},
	}}
	out := buildReport(findings, supp, 35, true)
	for _, want := range []string{
		"Drift detected: 4", "## Beta", "## Gamma", "m1", "m4",
		"Summary: 1 critical, 1 major, 1 minor, 1 info",
		"1 suppressed", "Suppressed by .claude/drift-allowlist.yaml", "by design",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}

func TestBuildReportCleanPath(t *testing.T) {
	out := buildReport(nil, nil, 35, false)
	if !strings.Contains(out, "No drift detected") {
		t.Fatalf("expected clean report, got %s", out)
	}
}

func TestEffectiveSchemaType(t *testing.T) {
	cases := []struct {
		name string
		node *schemaNode
		want string
	}{
		{"explicit", &schemaNode{Type: schemaTypeString}, schemaTypeString},
		{"oneof", &schemaNode{OneOf: []*schemaNode{{Type: schemaTypeString}}}, schemaTypeOneOf},
		{"object-by-props", &schemaNode{Properties: map[string]*schemaNode{"x": {}}}, schemaTypeObject},
		{"array-by-items", &schemaNode{Items: &schemaNode{Type: schemaTypeString}}, schemaTypeArray},
		{"empty", &schemaNode{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectiveSchemaType(tc.node); got != tc.want {
				t.Errorf("effectiveSchemaType = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseYAMLTag(t *testing.T) {
	cases := []struct {
		tag                       string
		name                      string
		omitempty, inline, skipIt bool
	}{
		{"title", "title", false, false, false},
		{"date,omitempty", "date", true, false, false},
		{",inline", "", false, true, false},
		{"-", "-", false, false, true},
		{"", "", false, false, false},
	}
	for _, tc := range cases {
		name, omitempty, inline, skip := parseYAMLTag(tc.tag)
		if name != tc.name || omitempty != tc.omitempty || inline != tc.inline || skip != tc.skipIt {
			t.Errorf("parseYAMLTag(%q) = (%q,%v,%v,%v), want (%q,%v,%v,%v)",
				tc.tag, name, omitempty, inline, skip, tc.name, tc.omitempty, tc.inline, tc.skipIt)
		}
	}
}

func TestSchemaDesc(t *testing.T) {
	if got := schemaDesc(ref{file: "a.json"}); got != "a.json" {
		t.Errorf("schemaDesc root = %q", got)
	}
	if got := schemaDesc(ref{file: "a.json", pointer: "/definitions/X"}); got != "a.json#/definitions/X" {
		t.Errorf("schemaDesc pointer = %q", got)
	}
}

func TestDerefNodeBrokenRef(t *testing.T) {
	set, err := newSchemaSet(map[string][]byte{"a.schema.json": []byte(`{"type":"object"}`)})
	if err != nil {
		t.Fatalf("newSchemaSet: %v", err)
	}
	c := newChecker(set, "go-glx/types.go")
	node, _ := c.derefNode("Sym", &schemaNode{Ref: "missing.schema.json"}, ref{file: "a.schema.json"})
	if node != nil {
		t.Fatal("expected nil node for broken ref")
	}
	if !hasFinding(c.findings, severityCritical, "unresolved schema reference") {
		t.Fatalf("expected internal finding for broken ref, got %+v", c.findings)
	}
}
