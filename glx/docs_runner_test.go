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

// TestRunDocsGenPrunesStaleGenerated verifies that a previously-generated
// doc whose command no longer exists is removed. The zombie file carries the
// real generated-doc front-matter so the pruner recognizes it as ours.
// Without pruning, renaming or removing a command would leave an orphan
// glx_<old>.md page on disk and the drift check would silently pass.
func TestRunDocsGenPrunesStaleGenerated(t *testing.T) {
	outDir := t.TempDir()

	zombie := filepath.Join(outDir, "glx_zombie_command.md")
	zombieBody := []byte(generatedDocFrontmatter + "\n## glx zombie_command\n\nstale\n")
	if err := os.WriteFile(zombie, zombieBody, 0o644); err != nil {
		t.Fatalf("seed zombie: %v", err)
	}

	indexPath := filepath.Join(outDir, "index.md")
	indexBody := []byte("hand-written overview")
	if err := os.WriteFile(indexPath, indexBody, 0o644); err != nil {
		t.Fatalf("seed index: %v", err)
	}

	if err := runDocsGen(rootCmd, outDir); err != nil {
		t.Fatalf("runDocsGen: %v", err)
	}

	if _, err := os.Stat(zombie); !os.IsNotExist(err) {
		t.Errorf("expected stale glx_zombie_command.md to be removed; stat err: %v", err)
	}

	got, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read index after run: %v", err)
	}
	if string(got) != string(indexBody) {
		t.Errorf("index.md was modified; got %q want %q", got, indexBody)
	}
}

// TestRunDocsGenPreservesHandAuthoredGlxFiles verifies that the pruner does
// NOT delete a glx-prefixed file lacking the generated-doc front-matter
// signature. This guards two scenarios: a future hand-authored
// docs/cli/glx_tips.md sibling, and a stray glx*.md in some unrelated
// directory if a user accidentally runs `glx docs --output <wrong-path>`.
func TestRunDocsGenPreservesHandAuthoredGlxFiles(t *testing.T) {
	outDir := t.TempDir()

	handAuthored := filepath.Join(outDir, "glx_tips.md")
	body := []byte("# Tips\n\nHand-written, no front-matter.\n")
	if err := os.WriteFile(handAuthored, body, 0o644); err != nil {
		t.Fatalf("seed handAuthored: %v", err)
	}

	if err := runDocsGen(rootCmd, outDir); err != nil {
		t.Fatalf("runDocsGen: %v", err)
	}

	got, err := os.ReadFile(handAuthored)
	if err != nil {
		t.Fatalf("read handAuthored after run: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("hand-authored glx_tips.md was modified; got %q want %q", got, body)
	}
}

// TestRunDocsGenIsDeterministic guards against drift detection becoming a
// false-positive treadmill: two consecutive runs into clean directories must
// produce byte-identical output, otherwise CI would fail on every PR.
//
// The filename sets must match too — comparing only files present in dir a
// would miss a run that emits an extra (or skips a) file in dir b, which is a
// subtler form of nondeterminism that would still cause drift CI to flap.
func TestRunDocsGenIsDeterministic(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()

	if err := runDocsGen(rootCmd, a); err != nil {
		t.Fatalf("runDocsGen a: %v", err)
	}
	if err := runDocsGen(rootCmd, b); err != nil {
		t.Fatalf("runDocsGen b: %v", err)
	}

	namesA := fileNames(t, a)
	namesB := fileNames(t, b)

	if len(namesA) == 0 {
		t.Fatal("runDocsGen produced no files; nothing to compare")
	}
	for name := range namesA {
		if _, ok := namesB[name]; !ok {
			t.Errorf("file %s present in a but missing from b", name)
		}
	}
	for name := range namesB {
		if _, ok := namesA[name]; !ok {
			t.Errorf("file %s present in b but missing from a", name)
		}
	}

	for name := range namesA {
		if _, ok := namesB[name]; !ok {
			continue
		}
		ba, err := os.ReadFile(filepath.Join(a, name))
		if err != nil {
			t.Fatalf("read %s from a: %v", name, err)
		}
		bb, err := os.ReadFile(filepath.Join(b, name))
		if err != nil {
			t.Fatalf("read %s from b: %v", name, err)
		}
		if string(ba) != string(bb) {
			t.Errorf("nondeterministic output for %s", name)
		}
	}
}

// fileNames returns the set of regular-file names directly under dir.
// Subdirectories are skipped because runDocsGen emits a flat layout.
func fileNames(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	names := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names[e.Name()] = struct{}{}
	}

	return names
}
