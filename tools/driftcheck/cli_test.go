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

// TestRealMain exercises the thin main → realMain wrapper that centralizes exit
// codes: it must thread parseFlags into run and surface run's code, mapping an
// operational error to exit 2. It mutates the global flag set and os streams, so
// it follows the same save/restore discipline as TestParseFlags and does not run
// in parallel.
func TestRealMain(t *testing.T) {
	oldArgs, oldCL := os.Args, flag.CommandLine
	oldStdout, oldStderr := os.Stdout, os.Stderr
	defer func() {
		os.Args, flag.CommandLine = oldArgs, oldCL
		os.Stdout, os.Stderr = oldStdout, oldStderr
	}()

	// realMain writes its report to os.Stdout and diagnostics to os.Stderr;
	// redirect both to the null device so test output stays quiet.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer func() { _ = devNull.Close() }()
	os.Stdout, os.Stderr = devNull, devNull

	// Happy path: with no -root, run auto-detects the repo root and checks the
	// real tree. Whatever the verdict (0 = clean, 1 = drift), realMain returns
	// run's code unchanged; only an operational failure maps to 2. (Drift itself
	// is guarded by TestRealTreeHasNoDrift.)
	flag.CommandLine = flag.NewFlagSet("driftcheck", flag.ContinueOnError)
	os.Args = []string{"driftcheck"}
	if code := realMain(); code == 2 {
		t.Error("realMain on the real tree returned operational error code 2")
	}

	// Error path: an empty -root has no schema files, so run returns an error
	// that realMain maps to exit code 2.
	flag.CommandLine = flag.NewFlagSet("driftcheck", flag.ContinueOnError)
	os.Args = []string{"driftcheck", "-root", t.TempDir()}
	if code := realMain(); code != 2 {
		t.Errorf("realMain with empty -root: got exit %d, want 2", code)
	}
}
