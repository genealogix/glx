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

func TestBuildReportClean(t *testing.T) {
	out := buildReport(5, nil, nil, false)
	if !strings.Contains(out, "checked 5 memory file(s)") || !strings.Contains(out, "No memory drift detected") {
		t.Errorf("unexpected clean report:\n%s", out)
	}
}

func TestBuildReportCleanWithSuppressed(t *testing.T) {
	supp := []suppression{{finding: finding{File: "CLAUDE.md", Line: 3, Token: "a/b.go", Kind: kindPath}, entry: allowEntry{Reason: "example"}}}
	out := buildReport(5, nil, supp, true)
	if !strings.Contains(out, "1 finding(s) suppressed") {
		t.Errorf("missing suppressed count:\n%s", out)
	}
	if !strings.Contains(out, "Suppressed by") || !strings.Contains(out, "example") {
		t.Errorf("verbose suppressed list missing:\n%s", out)
	}
}

func TestBuildReportFindings(t *testing.T) {
	findings := []finding{
		{File: "CLAUDE.md", Line: 10, Kind: kindPath, Severity: severityMajor, Token: "x/y.ged", Message: "references file `x/y.ged`, which does not exist"},
		{File: "go-glx/CLAUDE.md", Line: 4, Kind: kindMakeTarget, Severity: severityMajor, Token: "make foo", Message: "references `make foo`, which is not a target in the Makefile"},
	}
	out := buildReport(5, findings, nil, false)
	for _, want := range []string{
		"Memory drift detected: 2 finding(s)",
		"## CLAUDE.md",
		"## go-glx/CLAUDE.md",
		"Summary: 2 major, 0 minor, 0 info",
		"memory-drift-allowlist.yaml",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
}
