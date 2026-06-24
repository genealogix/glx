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
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRealTreeHasNoDrift is the regression guard: the committed memory files
// must match the committed repository. It runs the full pipeline against the
// real checkout, so any future stale path, unknown make target, or renamed
// module that a memory file fails to track fails here under `make test` / CI,
// not only under the dedicated `make check-memory-drift` target.
func TestRealTreeHasNoDrift(t *testing.T) {
	var buf bytes.Buffer
	code, err := run(options{allowlistFile: ".claude/memory-drift-allowlist.yaml"}, &buf)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != 0 {
		t.Fatalf("memory drift detected (exit %d):\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "No memory drift detected") {
		t.Fatalf("expected clean report, got:\n%s", buf.String())
	}
	// Guard against a discovery bug silently turning the run into a no-op.
	if strings.Contains(buf.String(), "checked 0 memory file(s)") {
		t.Fatal("no memory files discovered — discovery is broken")
	}
}

func TestMakeTargets(t *testing.T) {
	dir := t.TempDir()
	mk := ".PHONY: build test\n\n" +
		"build: ## build it\n\tgo build ./...\n\n" +
		"test:\n\tgo test ./...\n\n" +
		"VERSION = 1.0\n" // assignment, not a target
	mustWrite(t, filepath.Join(dir, "Makefile"), mk)

	targets, err := makeTargets(dir)
	if err != nil {
		t.Fatalf("makeTargets: %v", err)
	}
	for _, want := range []string{"build", "test"} {
		if !targets[want] {
			t.Errorf("missing target %q in %v", want, targets)
		}
	}
	if targets["VERSION"] || targets[".PHONY"] || targets["PHONY"] {
		t.Errorf("parsed a non-target as a target: %v", targets)
	}

	// Missing Makefile -> empty set, no error.
	empty, err := makeTargets(t.TempDir())
	if err != nil || len(empty) != 0 {
		t.Fatalf("missing Makefile should yield empty/no-error: %v %v", err, empty)
	}
}

func TestGoModule(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "go.mod"), "module github.com/genealogix/glx\n\ngo 1.26.0\n")
	mod, err := goModule(dir)
	if err != nil || mod != "github.com/genealogix/glx" {
		t.Fatalf("goModule = %q, %v", mod, err)
	}

	// No module line -> errNoModule.
	bad := t.TempDir()
	mustWrite(t, filepath.Join(bad, "go.mod"), "go 1.26.0\n")
	if _, err := goModule(bad); err == nil {
		t.Fatal("expected error for go.mod without a module line")
	}
}

func TestModuleOrgPrefix(t *testing.T) {
	cases := map[string]string{
		"github.com/genealogix/glx": "github.com/genealogix/",
		"example.com/foo/bar/v2":    "example.com/foo/",
		"single":                    "single/",
	}
	for in, want := range cases {
		if got := moduleOrgPrefix(in); got != want {
			t.Errorf("moduleOrgPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestDiscoverMemoryFiles confirms CLAUDE.md/AGENTS.md are found recursively and
// that vendored, VCS, and nested-worktree directories are skipped.
func TestDiscoverMemoryFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "x")
	mustWrite(t, filepath.Join(root, "pkg", "CLAUDE.md"), "x")
	mustWrite(t, filepath.Join(root, "pkg", "AGENTS.md"), "x")
	mustWrite(t, filepath.Join(root, "node_modules", "dep", "CLAUDE.md"), "x")
	mustWrite(t, filepath.Join(root, ".git", "CLAUDE.md"), "x")
	mustWrite(t, filepath.Join(root, ".claude", "worktrees", "wt", "CLAUDE.md"), "x")
	mustWrite(t, filepath.Join(root, "README.md"), "x") // not a memory file

	files, err := discoverMemoryFiles(root)
	if err != nil {
		t.Fatalf("discoverMemoryFiles: %v", err)
	}
	want := []string{"CLAUDE.md", "pkg/AGENTS.md", "pkg/CLAUDE.md"}
	if strings.Join(files, ",") != strings.Join(want, ",") {
		t.Errorf("discovered %v, want %v", files, want)
	}
}

func TestResolveUnderRoot(t *testing.T) {
	root := filepath.FromSlash("/repo")
	if got := resolveUnderRoot(root, ".claude/x.yaml"); got != filepath.Join(root, ".claude", "x.yaml") {
		t.Errorf("relative not joined: %q", got)
	}
	abs := filepath.Join(t.TempDir(), "a.yaml")
	if got := resolveUnderRoot(root, abs); got != abs {
		t.Errorf("absolute should pass through: %q", got)
	}
}

// TestRunExitCodes drives the full pipeline against a synthetic repo: drift -> 1,
// then clean -> 0 once the referenced file exists.
func TestRunExitCodes(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module github.com/genealogix/glx\n")
	mustWrite(t, filepath.Join(root, "Makefile"), "build:\n\tgo build ./...\n")
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "see `data/fixture.ged` and run `make build`\n")

	opts := options{root: root, allowlistFile: ".claude/memory-drift-allowlist.yaml"}

	var buf bytes.Buffer
	code, err := run(opts, &buf)
	if err != nil || code != 1 {
		t.Fatalf("expected drift exit 1, got %d (err=%v)\n%s", code, err, buf.String())
	}

	// Create the referenced file; now the tree is clean.
	mustWrite(t, filepath.Join(root, "data", "fixture.ged"), "0 HEAD\n")
	buf.Reset()
	code, err = run(opts, &buf)
	if err != nil || code != 0 {
		t.Fatalf("expected clean exit 0, got %d (err=%v)\n%s", code, err, buf.String())
	}
}

// TestRunAllowlistSuppression confirms an allowlist entry clears an otherwise
// gating finding.
func TestRunAllowlistSuppression(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "go.mod"), "module github.com/genealogix/glx\n")
	mustWrite(t, filepath.Join(root, "Makefile"), "build:\n\tgo build ./...\n")
	mustWrite(t, filepath.Join(root, "CLAUDE.md"), "illustrative `path/to/example.go`\n")
	mustWrite(t, filepath.Join(root, ".claude", "memory-drift-allowlist.yaml"),
		"- token: path/to/example.go\n  reason: documentation example\n")

	var buf bytes.Buffer
	code, err := run(options{root: root, allowlistFile: ".claude/memory-drift-allowlist.yaml"}, &buf)
	if err != nil || code != 0 {
		t.Fatalf("allowlisted reference should pass, got %d (err=%v)\n%s", code, err, buf.String())
	}
}

func TestParseFlags(t *testing.T) {
	oldArgs, oldCL := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldCL }()
	flag.CommandLine = flag.NewFlagSet("memcheck", flag.ContinueOnError)
	os.Args = []string{"memcheck", "-v", "-root", "/somewhere", "-allowlist", "a.yaml"}

	o := parseFlags()
	if !o.verbose || o.root != "/somewhere" || o.allowlistFile != "a.yaml" {
		t.Errorf("flags not parsed: %+v", o)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
