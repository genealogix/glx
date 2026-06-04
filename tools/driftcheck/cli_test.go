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
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestParseFlags(t *testing.T) {
	oldArgs, oldCL := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldCL }()

	flag.CommandLine = flag.NewFlagSet("driftcheck", flag.ContinueOnError)
	os.Args = []string{"driftcheck", "-v", "-root", "/somewhere", "-schema-dir", "s"}

	o := parseFlags()
	if !o.verbose {
		t.Error("-v should set verbose")
	}
	if o.root != "/somewhere" || o.schemaDir != "s" {
		t.Errorf("flags not parsed: %+v", o)
	}
}

func TestLoadAllowlistMissingAndPresent(t *testing.T) {
	// Missing file → empty (non-nil) allowlist, no error.
	a, err := loadAllowlist(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil || len(a.entries) != 0 {
		t.Fatalf("missing allowlist should be empty/no-error: %v %+v", err, a)
	}

	// Present file → parsed entries (allowlistYAML defined in allowlist_test.go).
	path := filepath.Join(t.TempDir(), "a.yaml")
	if writeErr := os.WriteFile(path, []byte(allowlistYAML), 0o600); writeErr != nil {
		t.Fatalf("write: %v", writeErr)
	}
	a2, err := loadAllowlist(path)
	if err != nil || len(a2.entries) != 2 {
		t.Fatalf("expected 2 entries, got %v %d", err, len(a2.entries))
	}
}

func TestRunMissingSchemas(t *testing.T) {
	code, err := run(options{
		root:          t.TempDir(), // empty dir: no schema files
		schemaDir:     "specification/schema/v1",
		allowlistFile: ".claude/drift-allowlist.yaml",
	}, io.Discard)
	if code != 2 {
		t.Fatalf("expected exit 2 when schemas are missing, got %d (err=%v)", code, err)
	}
}
