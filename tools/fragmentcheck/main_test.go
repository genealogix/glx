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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFragment(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fragment.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fragment: %v", err)
	}

	return path
}

func TestCheckFile_Accepts(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{
			"block map",
			"kind: Added\nbody: something\ncustom:\n  Issue: \"#123\"\n",
		},
		{
			// The shape the awk gate this replaced rejected outright: valid
			// YAML, but the field never appeared on its own indented line.
			"inline flow map",
			"kind: Added\nbody: something\ncustom: { Issue: \"#123\" }\n",
		},
		{
			"inline flow map with sibling keys",
			"kind: Added\nbody: something\ncustom: {Kind: chore, Issue: '#123'}\n",
		},
		{
			"multi-ref value",
			"kind: Added\nbody: something\ncustom:\n  Issue: '#41, #775'\n",
		},
		{
			"prefixed ref",
			"kind: Added\nbody: something\ncustom:\n  Issue: \"PR #456\"\n",
		},
		{
			"quoted key",
			"kind: Added\nbody: something\n\"custom\":\n  \"Issue\": \"#123\"\n",
		},
		{
			"deep-indented block map",
			"kind: Added\nbody: something\ncustom:\n    Issue: \"#123\"\n",
		},
		{
			"issue key last after multi-line body",
			"kind: Added\nbody: |-\n  line one\n  line two\ntime: 2026-06-05T17:30:00Z\ncustom:\n  Issue: \"#123\"\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if problem := checkFile(writeFragment(t, c.content)); problem != "" {
				t.Errorf("expected fragment to pass, got problem: %s", problem)
			}
		})
	}
}

func TestCheckFile_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantSub string
	}{
		{
			"no custom block",
			"kind: Added\nbody: something\n",
			"missing an issue/PR reference",
		},
		{
			"custom without Issue",
			"kind: Added\nbody: something\ncustom:\n  Other: \"#123\"\n",
			"missing an issue/PR reference",
		},
		{
			"Issue without a number",
			"kind: Added\nbody: something\ncustom:\n  Issue: see the tracker\n",
			"carries no #NNN",
		},
		{
			// Unquoted `#123` is a YAML comment, so the value decodes to nil.
			"unquoted ref is a comment",
			"kind: Added\nbody: something\ncustom:\n  Issue: #123\n",
			"unquoted #NNN is a YAML comment",
		},
		{
			// The property the awk gate was written to have, preserved here
			// structurally: body text can never satisfy the gate.
			"Issue-looking line inside the body block scalar",
			"kind: Added\nbody: |-\n  Fixes the thing.\n  Issue: \"#123\"\n",
			"missing an issue/PR reference",
		},
		{
			"malformed YAML",
			"kind: Added\nbody: [unterminated\n",
			"not valid YAML",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			problem := checkFile(writeFragment(t, c.content))
			if problem == "" {
				t.Fatalf("expected fragment to be rejected, got no problem")
			}
			if !strings.Contains(problem, c.wantSub) {
				t.Errorf("problem = %q, want it to contain %q", problem, c.wantSub)
			}
		})
	}
}

// TestCheckFile_NonStringIssue pins that a numeric value is reported as a
// missing reference rather than blowing up the YAML decode for the whole file.
func TestCheckFile_NonStringIssue(t *testing.T) {
	problem := checkFile(writeFragment(t, "kind: Added\nbody: something\ncustom:\n  Issue: 123\n"))
	if !strings.Contains(problem, "carries no #NNN") {
		t.Errorf("problem = %q, want it to report a missing #NNN reference", problem)
	}
}

func TestFragmentFiles_SortedAndBothExtensions(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"b.yaml", "a.yml", "c.yaml", "ignored.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("kind: Added\n"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	got, err := fragmentFiles(dir)
	if err != nil {
		t.Fatalf("fragmentFiles: %v", err)
	}
	names := make([]string, 0, len(got))
	for _, p := range got {
		names = append(names, filepath.Base(p))
	}
	want := []string{"a.yml", "b.yaml", "c.yaml"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("fragmentFiles = %v, want %v", names, want)
	}
}

// TestCheckFile_RepoFragments runs the gate over the fragments actually
// committed in this repository, so a fragment that would fail CI fails here.
func TestCheckFile_RepoFragments(t *testing.T) {
	files, err := fragmentFiles(filepath.Join("..", "..", ".changes", "unreleased"))
	if err != nil {
		t.Fatalf("fragmentFiles: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no unreleased fragments committed")
	}
	for _, f := range files {
		if problem := checkFile(f); problem != "" {
			t.Errorf("committed fragment %s fails the gate: %s", f, problem)
		}
	}
}
