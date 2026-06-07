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

// Command driftcheck deterministically detects drift between the GLX Go type
// definitions (go-glx/types.go, compared via reflection) and the JSON Schemas
// in specification/schema/v1. It is the deterministic counterpart to the
// LLM-based /check-code-drift slash command: it catches structural drift
// (missing/extra fields, yaml-tag mismatches, required/omitempty mismatches,
// type-family mismatches) on every PR, with no API keys and zero false
// positives, deferring ambiguous/semantic cases to the LLM checker.
//
// Source-of-truth flow: specification (*.md) -> schema (*.schema.json) -> Go
// code. A finding therefore means the Go code needs to change to match the
// schema. Known, by-design exceptions live in .claude/drift-allowlist.yaml.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	errNoSchemaFiles = errors.New("no *.schema.json files found")
	errNoGoMod       = errors.New("could not locate go.mod")
)

func main() {
	opts := parseFlags()
	code, err := run(opts, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "driftcheck:", err)
		os.Exit(2)
	}
	os.Exit(code)
}

type options struct {
	root          string
	schemaDir     string
	typesFile     string
	allowlistFile string
	verbose       bool
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.root, "root", "", "repo root (default: auto-detect by walking up to go.mod)")
	flag.StringVar(&o.schemaDir, "schema-dir", "specification/schema/v1", "schema directory, relative to root")
	flag.StringVar(&o.typesFile, "types-file", "go-glx/types.go", "Go types file (used for allowlist attribution)")
	flag.StringVar(&o.allowlistFile, "allowlist", ".claude/drift-allowlist.yaml", "drift allowlist, relative to root")
	flag.BoolVar(&o.verbose, "v", false, "list allowlist-suppressed findings")
	flag.Parse()

	return o
}

// run performs the check and returns the process exit code (0 = clean,
// 1 = gating drift found). It writes the human-readable report to out.
func run(opts options, out io.Writer) (int, error) {
	root := opts.root
	if root == "" {
		detected, err := findRepoRoot()
		if err != nil {
			return 2, err
		}
		root = detected
	}

	schemaDir := resolveUnderRoot(root, opts.schemaDir)
	files, err := readSchemaFiles(schemaDir)
	if err != nil {
		return 2, fmt.Errorf("reading schemas from %s: %w", schemaDir, err)
	}
	if len(files) == 0 {
		return 2, fmt.Errorf("%w under %s", errNoSchemaFiles, schemaDir)
	}

	schemas, err := newSchemaSet(files)
	if err != nil {
		return 2, err
	}

	allow, err := loadAllowlist(resolveUnderRoot(root, opts.allowlistFile))
	if err != nil {
		return 2, err
	}

	c := newChecker(schemas, filepath.ToSlash(opts.typesFile))
	c.run()

	surviving, suppressed := allow.partition(dedupeFindings(c.findings))
	// Best-effort: attach go-glx/types.go source lines via the AST extractor.
	_ = attachLines(surviving, resolveUnderRoot(root, opts.typesFile))
	byEntityThenSeverity(surviving)

	if _, writeErr := io.WriteString(out, buildReport(surviving, suppressed, len(files), opts.verbose)); writeErr != nil {
		return 2, writeErr
	}

	if hasFailures(surviving) {
		return 1, nil
	}

	return 0, nil
}

// resolveUnderRoot interprets p relative to root, but honors an absolute p as-is
// so callers can point -schema-dir / -allowlist anywhere.
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

// readSchemaFiles loads every *.schema.json under dir, keyed by path relative
// to dir using forward slashes (matching how $refs are written).
func readSchemaFiles(dir string) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".schema.json") {
			return nil
		}
		// #nosec G304 -- path is a repo schema file discovered by WalkDir, not user input.
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		files[filepath.ToSlash(rel)] = data

		return nil
	})

	return files, err
}

// loadAllowlist reads and parses the allowlist; a missing file is treated as an
// empty allowlist so the tool still runs in a checkout without one.
func loadAllowlist(path string) (*allowlist, error) {
	// #nosec G304 -- path is the repo-relative drift allowlist, not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &allowlist{}, nil
		}

		return nil, fmt.Errorf("reading allowlist %s: %w", path, err)
	}

	return parseAllowlist(data)
}
