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
	"testing"
)

const allowlistYAML = `
- file: go-glx/types.go
  symbol: GLXFile.ImportMetadata
  yaml_tag: metadata
  reason: by design
  permanent: true
  added: "2026-06-01"
- file: go-glx/types.go
  symbol: Source.Foo
  reason: deferred
  tracking_issue: 999
  added: "2026-06-01"
`

func TestAllowlistExactSymbolMatch(t *testing.T) {
	a, err := parseAllowlist([]byte(allowlistYAML))
	if err != nil {
		t.Fatalf("parseAllowlist: %v", err)
	}
	f := finding{File: "go-glx/types.go", Symbol: "Source.Foo", Field: "Foo"}
	if _, ok := a.match(&f); !ok {
		t.Fatal("expected exact symbol match")
	}
}

func TestAllowlistYAMLTagAlias(t *testing.T) {
	a, _ := parseAllowlist([]byte(allowlistYAML))
	// A missing-property finding is keyed by yaml tag, not Go field name.
	f := finding{File: "go-glx/types.go", Symbol: "GLXFile.metadata", YamlTag: "metadata"}
	if _, ok := a.match(&f); !ok {
		t.Fatal("expected yaml-tag alias match for GLXFile.ImportMetadata/metadata")
	}
}

func TestAllowlistDoesNotCrossEntities(t *testing.T) {
	a, _ := parseAllowlist([]byte(allowlistYAML))
	// Same field name "Foo" but a different owning type must NOT be suppressed.
	f := finding{File: "go-glx/types.go", Symbol: "Citation.Foo", Field: "Foo"}
	if _, ok := a.match(&f); ok {
		t.Fatal("Citation.Foo must not match Source.Foo entry")
	}
}

func TestAllowlistFileMustMatch(t *testing.T) {
	a, _ := parseAllowlist([]byte(allowlistYAML))
	f := finding{File: "go-glx/other.go", Symbol: "Source.Foo", Field: "Foo"}
	if _, ok := a.match(&f); ok {
		t.Fatal("entry in a different file must not match")
	}
}

func TestAllowlistPartition(t *testing.T) {
	a, _ := parseAllowlist([]byte(allowlistYAML))
	findings := []finding{
		{File: "go-glx/types.go", Symbol: "Source.Foo", Field: "Foo", Severity: severityCritical},
		{File: "go-glx/types.go", Symbol: "Event.Title", Field: "Title", Severity: severityMajor},
	}
	surviving, suppressed := a.partition(findings)
	if len(surviving) != 1 || surviving[0].Symbol != "Event.Title" {
		t.Fatalf("unexpected surviving: %+v", surviving)
	}
	if len(suppressed) != 1 || suppressed[0].entry.TrackingIssue != 999 {
		t.Fatalf("unexpected suppressed: %+v", suppressed)
	}
}

func TestEmptyAllowlist(t *testing.T) {
	a, err := parseAllowlist([]byte("   \n"))
	if err != nil {
		t.Fatalf("parseAllowlist empty: %v", err)
	}
	if len(a.entries) != 0 {
		t.Fatalf("expected no entries, got %d", len(a.entries))
	}
}
