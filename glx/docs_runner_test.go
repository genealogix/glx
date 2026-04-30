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

	"github.com/spf13/cobra"
)

// countAvailableCommands mirrors cobra/doc.GenMarkdownTreeCustom's traversal
// rule (skip hidden, deprecated, additional-help-topic) so the test asserts
// against exactly the file count that runDocsGen should produce.
func countAvailableCommands(cmd *cobra.Command) int {
	count := 1
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		count += countAvailableCommands(sub)
	}

	return count
}

func TestRunDocsGenProducesPerCommandMarkdown(t *testing.T) {
	outDir := t.TempDir()

	if err := runDocsGen(rootCmd, outDir); err != nil {
		t.Fatalf("runDocsGen: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}

	mdFiles := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		mdFiles[e.Name()] = struct{}{}
	}

	if want := countAvailableCommands(rootCmd); len(mdFiles) != want {
		t.Fatalf("got %d Markdown files, want %d (one per available command in the tree)", len(mdFiles), want)
	}

	for _, name := range []string{"glx.md", "glx_init.md", "glx_validate.md", "glx_import.md"} {
		if _, ok := mdFiles[name]; !ok {
			t.Errorf("expected %s in generated output", name)
		}
	}

	if _, ok := mdFiles["glx_docs.md"]; ok {
		t.Errorf("hidden glx docs subcommand leaked into generated output")
	}
}

func TestRunDocsGenWritesFrontmatter(t *testing.T) {
	outDir := t.TempDir()

	if err := runDocsGen(rootCmd, outDir); err != nil {
		t.Fatalf("runDocsGen: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(outDir, "glx_init.md"))
	if err != nil {
		t.Fatalf("read glx_init.md: %v", err)
	}
	head := string(body)
	if len(head) > 100 {
		head = head[:100]
	}
	if !strings.HasPrefix(head, "---\n") || !strings.Contains(head, "editLink: false") {
		t.Errorf("glx_init.md missing expected VitePress frontmatter; head: %q", head)
	}
}

// TestRunDocsGenIsDeterministic guards against drift detection becoming a
// false-positive treadmill: two consecutive runs into clean directories must
// produce byte-identical output, otherwise CI would fail on every PR.
func TestRunDocsGenIsDeterministic(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()

	if err := runDocsGen(rootCmd, a); err != nil {
		t.Fatalf("runDocsGen a: %v", err)
	}
	if err := runDocsGen(rootCmd, b); err != nil {
		t.Fatalf("runDocsGen b: %v", err)
	}

	entriesA, err := os.ReadDir(a)
	if err != nil {
		t.Fatalf("read a: %v", err)
	}
	if len(entriesA) == 0 {
		t.Fatal("runDocsGen produced no files; nothing to compare")
	}

	for _, e := range entriesA {
		if e.IsDir() {
			continue
		}
		ba, err := os.ReadFile(filepath.Join(a, e.Name()))
		if err != nil {
			t.Fatalf("read %s from a: %v", e.Name(), err)
		}
		bb, err := os.ReadFile(filepath.Join(b, e.Name()))
		if err != nil {
			t.Fatalf("read %s from b: %v", e.Name(), err)
		}
		if string(ba) != string(bb) {
			t.Errorf("nondeterministic output for %s", e.Name())
		}
	}
}
