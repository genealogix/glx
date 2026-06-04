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
	"path/filepath"
	"strings"
	"testing"
)

// TestRealTreeHasNoDrift is the regression guard: the committed go-glx types
// must match the committed schemas (modulo the allowlist). It runs the full
// pipeline against the real repository, so any future schema/type change that
// introduces drift fails here under `make test` / CI, not just under the
// dedicated `make check-code-drift` target.
func TestRealTreeHasNoDrift(t *testing.T) {
	var buf bytes.Buffer
	code, err := run(defaultOptions(), &buf)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if code != 0 {
		t.Fatalf("schema/code drift detected (exit %d):\n%s", code, buf.String())
	}
	if !strings.Contains(buf.String(), "No drift detected") {
		t.Fatalf("expected clean report, got:\n%s", buf.String())
	}
}

// TestRealTreeCoversAllEntities guards against the checker silently skipping
// entities (e.g. a broken $ref turning the run into a no-op): every binding
// must resolve and compare at least its entity types.
func TestRealTreeCoversAllEntities(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	schemas := loadRealSchemas(t, root)
	c := newChecker(schemas, "go-glx/types.go")
	c.run()
	// Any "Internal"-category finding means a binding failed to resolve.
	for _, f := range c.findings {
		if f.Category == "Internal" {
			t.Fatalf("internal checker failure (broken binding/$ref): %s", f.Message)
		}
	}
}

func TestResolveUnderRoot(t *testing.T) {
	root := filepath.FromSlash("/repo")
	rel := resolveUnderRoot(root, "specification/schema/v1")
	if rel != filepath.Join(root, "specification", "schema", "v1") {
		t.Errorf("relative path not joined to root: %q", rel)
	}

	abs := filepath.Join(t.TempDir(), "allow.yaml")
	if got := resolveUnderRoot(root, abs); got != abs {
		t.Errorf("absolute path should be honored as-is: got %q want %q", got, abs)
	}
}

func defaultOptions() options {
	return options{
		schemaDir:     "specification/schema/v1",
		typesFile:     "go-glx/types.go",
		allowlistFile: ".claude/drift-allowlist.yaml",
	}
}

func loadRealSchemas(t *testing.T, root string) *schemaSet {
	t.Helper()
	files, err := readSchemaFiles(filepath.Join(root, "specification", "schema", "v1"))
	if err != nil {
		t.Fatalf("readSchemaFiles: %v", err)
	}
	set, err := newSchemaSet(files)
	if err != nil {
		t.Fatalf("newSchemaSet: %v", err)
	}

	return set
}
