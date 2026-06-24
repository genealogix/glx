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

func TestParseAllowlistEmptyAndPresent(t *testing.T) {
	for _, in := range []string{"", "   \n", "# just a comment\n"} {
		a, err := parseAllowlist([]byte(in))
		if err != nil || len(a.entries) != 0 {
			t.Fatalf("empty/whitespace/comment should yield 0 entries: %q -> %v %+v", in, err, a)
		}
	}

	a, err := parseAllowlist([]byte("- token: a/b.go\n  reason: example\n- token: make foo\n  kind: unknown-make-target\n"))
	if err != nil || len(a.entries) != 2 {
		t.Fatalf("expected 2 entries: %v %+v", err, a)
	}
}

func TestAllowlistPartition(t *testing.T) {
	a := &allowlist{entries: []allowEntry{
		{Token: "a/b.go", Reason: "any-file/kind"},
		{Token: "x/y.go", File: "go-glx/CLAUDE.md", Reason: "scoped to one file"},
		{Token: "make foo", Kind: "unknown-make-target", Reason: "scoped to kind"},
	}}

	findings := []finding{
		{File: "CLAUDE.md", Token: "a/b.go", Kind: kindPath},            // suppressed (token-only)
		{File: "CLAUDE.md", Token: "x/y.go", Kind: kindPath},            // NOT suppressed (wrong file)
		{File: "go-glx/CLAUDE.md", Token: "x/y.go", Kind: kindPath},     // suppressed (file matches)
		{File: "CLAUDE.md", Token: "make foo", Kind: kindMakeTarget},    // suppressed (kind matches)
		{File: "CLAUDE.md", Token: "make foo", Kind: kindPath},          // NOT suppressed (wrong kind)
		{File: "CLAUDE.md", Token: "untouched.go/x.go", Kind: kindPath}, // NOT suppressed (no entry)
	}

	surviving, suppressed := a.partition(findings)
	if len(suppressed) != 3 {
		t.Fatalf("expected 3 suppressed, got %d: %+v", len(suppressed), suppressed)
	}
	if len(surviving) != 3 {
		t.Fatalf("expected 3 surviving, got %d: %+v", len(surviving), surviving)
	}
}
