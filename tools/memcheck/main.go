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

// Command memcheck deterministically detects drift between the repository's
// agent-memory files (every CLAUDE.md, and AGENTS.md if adopted) and reality. It
// is the offline, API-key-free counterpart to the LLM-based check-* drift suite,
// asserting three cheap invariants on every PR:
//
//  1. Every `make <target>` referenced exists in the Makefile.
//  2. Every concrete file/directory path referenced exists on disk.
//  3. Every github.com/<org>/<repo> import path under this module's org matches
//     the go.mod module line.
//
// Stale instructions are worse than missing ones — an agent confidently applies
// dead patterns — so a surviving finding fails the check (exit 1). Known, by-
// design references that are not meant to resolve live in
// .claude/memory-drift-allowlist.yaml.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	errNoGoMod  = errors.New("could not locate go.mod")
	errNoModule = errors.New("no module line in go.mod")
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	opts := parseFlags()
	code, err := run(opts, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	return code
}

type options struct {
	root          string
	allowlistFile string
	verbose       bool
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.root, "root", "", "repo root (default: auto-detect by walking up to go.mod)")
	flag.StringVar(&o.allowlistFile, "allowlist", ".claude/memory-drift-allowlist.yaml",
		"memory-drift allowlist, relative to root")
	flag.BoolVar(&o.verbose, "v", false, "list allowlist-suppressed findings")
	flag.Parse()

	return o
}

// run performs the check and returns the process exit code together with any
// operational error. It writes the human-readable report to out. Codes:
// 0 = clean, 1 = gating drift found, 2 = operational failure — in which case a
// non-nil error is also returned, which realMain prints before exiting 2.
func run(opts options, out io.Writer) (int, error) {
	root := opts.root
	if root == "" {
		detected, err := findRepoRoot()
		if err != nil {
			return 2, err
		}
		root = detected
	}

	r, err := newRepo(root)
	if err != nil {
		return 2, err
	}

	memFiles, err := discoverMemoryFiles(root)
	if err != nil {
		return 2, fmt.Errorf("discovering memory files: %w", err)
	}

	var findings []finding
	for _, mf := range memFiles {
		// #nosec G304 -- mf is a repo memory file discovered by WalkDir under
		// root, not user input.
		data, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(mf)))
		if readErr != nil {
			return 2, fmt.Errorf("reading %s: %w", mf, readErr)
		}
		findings = append(findings, scanFile(mf, string(data), r)...)
	}

	allow, err := loadAllowlist(resolveUnderRoot(root, opts.allowlistFile))
	if err != nil {
		return 2, err
	}

	surviving, suppressed := allow.partition(dedupe(findings))
	byFileLineToken(surviving)

	if _, writeErr := io.WriteString(out, buildReport(len(memFiles), surviving, suppressed, opts.verbose)); writeErr != nil {
		return 2, writeErr
	}

	if hasFailures(surviving) {
		return 1, nil
	}

	return 0, nil
}

// newRepo loads the deterministic ground truth (Makefile targets, go.mod module)
// and wires a filesystem-backed existence check rooted at root.
func newRepo(root string) (repo, error) {
	targets, err := makeTargets(root)
	if err != nil {
		return repo{}, err
	}
	module, err := goModule(root)
	if err != nil {
		return repo{}, err
	}

	exists := func(repoRel string) bool {
		_, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(repoRel)))

		return statErr == nil
	}

	return repo{
		makeTargets: targets,
		modulePath:  module,
		orgPrefix:   moduleOrgPrefix(module),
		exists:      exists,
	}, nil
}

var makeTargetLineRe = regexp.MustCompile(`(?m)^([A-Za-z][A-Za-z0-9_-]*)\s*:`)

// makeTargets parses the Makefile at root and returns the set of declared rule
// targets. A missing Makefile yields an empty set (no make-target checking)
// rather than an error, so the tool still runs in a checkout without one.
func makeTargets(root string) (map[string]bool, error) {
	// #nosec G304 -- fixed repo path, not user input.
	data, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]bool{}, nil
		}

		return nil, fmt.Errorf("reading Makefile: %w", err)
	}

	targets := map[string]bool{}
	for _, m := range makeTargetLineRe.FindAllStringSubmatch(string(data), -1) {
		targets[m[1]] = true
	}

	return targets, nil
}

var goModuleLineRe = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// goModule returns the module path from the go.mod at root.
func goModule(root string) (string, error) {
	// #nosec G304 -- fixed repo path, not user input.
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	m := goModuleLineRe.FindStringSubmatch(string(data))
	if m == nil {
		return "", errNoModule
	}

	return m[1], nil
}

// moduleOrgPrefix returns the host/org prefix of a module path (the first two
// segments plus a trailing slash), e.g. "github.com/genealogix/" for
// "github.com/genealogix/glx". References under this prefix are claims about the
// module's own org; everything else is a third-party import left unchecked.
func moduleOrgPrefix(module string) string {
	parts := strings.SplitN(module, "/", 3)
	if len(parts) >= 2 {
		return parts[0] + "/" + parts[1] + "/"
	}

	return module + "/"
}

// skipDirs are directory names never walked for memory files: VCS internals,
// vendored dependencies, and build output.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
}

// discoverMemoryFiles walks root and returns every CLAUDE.md / AGENTS.md, as
// repo-relative forward-slash paths, sorted. Nested worktrees under
// .claude/worktrees are skipped so a checkout that hosts in-progress worktrees
// does not double-report their copies.
func discoverMemoryFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldSkipDir(root, p, d.Name()) {
				return fs.SkipDir
			}

			return nil
		}
		if d.Name() == "CLAUDE.md" || d.Name() == "AGENTS.md" {
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
		}

		return nil
	})
	sort.Strings(files)

	return files, err
}

func shouldSkipDir(root, p, name string) bool {
	if skipDirs[name] {
		return true
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}

	return filepath.ToSlash(rel) == ".claude/worktrees"
}

// resolveUnderRoot interprets p relative to root, honoring an absolute p as-is.
func resolveUnderRoot(root, p string) string {
	local := filepath.FromSlash(p)
	if filepath.IsAbs(local) {
		return local
	}

	return filepath.Join(root, local)
}

// findRepoRoot walks up from the working directory until it finds a go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w above %q (use -root)", errNoGoMod, dir)
		}
		dir = parent
	}
}

// loadAllowlist reads and parses the allowlist; a missing file is treated as an
// empty allowlist so the tool still runs in a checkout without one.
func loadAllowlist(path string) (*allowlist, error) {
	// #nosec G304 -- path is the repo-relative memory-drift allowlist, not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &allowlist{}, nil
		}

		return nil, fmt.Errorf("reading allowlist %s: %w", path, err)
	}

	return parseAllowlist(data)
}
