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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// stageMergeFixture copies a fixture set (base, ours, theirs) from testdata
// into a fresh temp dir and returns the staged inputs. The runner writes to
// OursPath, so we need a writable copy per test invocation.
func stageMergeFixture(t *testing.T, fixture string) mergeDriverInputs {
	t.Helper()
	src := filepath.Join("testdata", "merge-driver", fixture)
	dst := t.TempDir()

	for _, name := range []string{"base.glx", "ours.glx", "theirs.glx"} {
		data, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			t.Fatalf("read fixture %s/%s: %v", fixture, name, err)
		}
		if err := os.WriteFile(filepath.Join(dst, name), data, 0o644); err != nil {
			t.Fatalf("stage fixture %s/%s: %v", fixture, name, err)
		}
	}

	return mergeDriverInputs{
		BasePath:   filepath.Join(dst, "base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   fixture + "/ours.glx",
	}
}

// requireGit skips when git is not available — the conflict path shells out
// to `git merge-file`. CI always has git; this keeps offline `go test` happy
// on contributor laptops without git.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available; skipping merge driver test that depends on `git merge-file`")
	}
}

func readMerged(t *testing.T, in mergeDriverInputs) string {
	t.Helper()
	data, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read merged %q: %v", in.OursPath, err)
	}

	return string(data)
}

func TestMergeDriver_CleanAdditive_UnionsCitations(t *testing.T) {
	in := stageMergeFixture(t, "clean-additive")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code != mergeDriverExitClean {
		t.Fatalf("expected clean exit, got %d (stderr: %s)", code, errBuf.String())
	}

	merged := readMerged(t, in)
	for _, want := range []string{"citation-parish-register", "citation-1851-census", "citation-family-bible"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged output missing citation %q\n---\n%s", want, merged)
		}
	}
	if strings.Contains(merged, "<<<<<<<") {
		t.Errorf("merged output should not contain conflict markers\n---\n%s", merged)
	}
}

func TestMergeDriver_CleanOneSided_KeepsOursChange(t *testing.T) {
	in := stageMergeFixture(t, "clean-one-sided")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code != mergeDriverExitClean {
		t.Fatalf("expected clean exit, got %d (stderr: %s)", code, errBuf.String())
	}

	merged := readMerged(t, in)
	if !strings.Contains(merged, "birth_date") || !strings.Contains(merged, "1850") {
		t.Errorf("expected ours' birth_date change to survive merge\n---\n%s", merged)
	}
}

func TestMergeDriver_ConflictFallback_WritesGitMarkers(t *testing.T) {
	requireGit(t)
	in := stageMergeFixture(t, "conflict-fallback")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code == mergeDriverExitClean {
		t.Fatalf("expected nonzero exit for unresolved conflict, got %d", code)
	}

	merged := readMerged(t, in)
	if !strings.Contains(merged, "<<<<<<<") {
		t.Errorf("expected git merge-file <<<<<<< markers in ours, got:\n%s", merged)
	}
	// The structured stderr summary should mention the conflicting property
	// path so the researcher knows where to look.
	if !strings.Contains(errBuf.String(), "persons[person-john-smith].properties.birth_date") {
		t.Errorf("expected conflict summary to mention property path, got stderr:\n%s", errBuf.String())
	}
}

func TestMergeDriver_ConfidenceResolves_PicksHigherConfidence(t *testing.T) {
	in := stageMergeFixture(t, "confidence-resolves")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code != mergeDriverExitClean {
		t.Fatalf("expected clean exit (confidence auto-resolves), got %d (stderr: %s)", code, errBuf.String())
	}

	merged := readMerged(t, in)
	if !strings.Contains(merged, `"1850-04-12"`) {
		t.Errorf("expected ours' high-confidence value 1850-04-12 to win\n---\n%s", merged)
	}
	if strings.Contains(merged, `1850-04-15`) {
		t.Errorf("theirs' medium-confidence value 1850-04-15 should not appear in merged\n---\n%s", merged)
	}
	// Both branches added different citations on top of the shared base
	// citation — both should land in the merged result.
	for _, want := range []string{"citation-parish-register", "citation-1851-census", "citation-family-tradition"} {
		if !strings.Contains(merged, want) {
			t.Errorf("merged output missing citation %q\n---\n%s", want, merged)
		}
	}
	// The stderr should explain that the driver picked ours.
	if !strings.Contains(errBuf.String(), "auto-resolved") {
		t.Errorf("expected stderr to surface auto-resolution, got:\n%s", errBuf.String())
	}
}

func TestMergeDriver_EmptyBase_HandlesAddOnBothSides(t *testing.T) {
	in := stageMergeFixture(t, "empty-base")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code != mergeDriverExitClean {
		t.Fatalf("expected clean exit for identical add-on-both-sides, got %d (stderr: %s)", code, errBuf.String())
	}

	merged := readMerged(t, in)
	if !strings.Contains(merged, "person-john-smith") {
		t.Errorf("expected person-john-smith in merged, got:\n%s", merged)
	}
}

func TestMergeDriver_MalformedYAML_FallsBackToTextMerge(t *testing.T) {
	requireGit(t)
	in := stageMergeFixture(t, "malformed-yaml")
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	// Structurally unparseable input forces the text-merge fallback. The
	// text merge of this fixture has no overlapping changes (only theirs
	// changes "1852 → broken-unterminated-string), so `git merge-file` may
	// or may not report a conflict depending on git's heuristics. We just
	// assert the driver didn't crash and stderr mentions the fallback.
	_ = code
	if !strings.Contains(errBuf.String(), "falling back to text merge") {
		t.Errorf("expected stderr to mention text-merge fallback, got:\n%s", errBuf.String())
	}
}

func TestMergeDriver_MissingBaseFile_ReportsAndExitsNonzero(t *testing.T) {
	dst := t.TempDir()
	// Only ours and theirs exist; base is missing.
	if err := os.WriteFile(filepath.Join(dst, "ours.glx"), []byte("persons:\n"), 0o644); err != nil {
		t.Fatalf("write ours: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "theirs.glx"), []byte("persons:\n"), 0o644); err != nil {
		t.Fatalf("write theirs: %v", err)
	}
	in := mergeDriverInputs{
		BasePath:   filepath.Join(dst, "missing-base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   "missing-base.glx",
	}
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code == mergeDriverExitClean {
		t.Errorf("expected nonzero exit when base file missing, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "read base") {
		t.Errorf("expected stderr to mention base-read failure, got:\n%s", errBuf.String())
	}
}

// TestMergeDriver_OursPathUntouchedOnFallback verifies the order-of-operations
// invariant from the plan: when the structural merge can't resolve, the
// driver must not have written to OursPath before invoking git merge-file.
// We assert this indirectly: the conflict-fallback fixture's ours.glx has
// distinctive content; before the runner is called, that's what's on disk;
// if the driver had written its own partial merged YAML before falling
// back, git merge-file would have been given corrupted input and the final
// content would lack the conflict markers we expect.
func TestMergeDriver_OursPathUntouchedOnFallback(t *testing.T) {
	requireGit(t)
	in := stageMergeFixture(t, "conflict-fallback")
	originalOurs, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read original ours: %v", err)
	}

	var errBuf bytes.Buffer
	_ = runMergeDriver(in, &errBuf)

	merged, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read merged: %v", err)
	}
	// After the fallback runs, the file should contain conflict markers AND
	// the original ours value 1851 (since git merge-file would have written
	// that side of the marker).
	if !strings.Contains(string(merged), "1851") {
		t.Errorf("merged output should preserve ours' content side of the conflict marker, got:\n%s", merged)
	}
	// Sanity: we definitely did not just leave the original ours untouched.
	if bytes.Equal(originalOurs, merged) {
		t.Errorf("merged output equals original ours — fallback did not run")
	}
}

func TestMergeDriver_OversizedInputFallsBackToTextMerge(t *testing.T) {
	requireGit(t)
	dst := t.TempDir()
	// 65 MiB of zeros — past the maxMergeInputBytes cap of 64 MiB.
	big := make([]byte, maxMergeInputBytes+1024*1024)
	if err := os.WriteFile(filepath.Join(dst, "base.glx"), []byte("persons:\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "ours.glx"), big, 0o644); err != nil {
		t.Fatalf("write big ours: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "theirs.glx"), []byte("persons:\n"), 0o644); err != nil {
		t.Fatalf("write theirs: %v", err)
	}
	in := mergeDriverInputs{
		BasePath:   filepath.Join(dst, "base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   "oversized.glx",
	}
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code == mergeDriverExitClean {
		t.Errorf("expected nonzero exit for oversized input, got %d", code)
	}
	if !strings.Contains(errBuf.String(), "exceeds merge-driver cap") {
		t.Errorf("expected stderr to mention the size cap, got:\n%s", errBuf.String())
	}
}

func TestSafeForStderr(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"tab and newline kept", "a\tb\nc", "a\tb\nc"},
		{"ANSI escape stripped", "1850\x1b[2J\x1b[Hmerge ok", "1850�[2J�[Hmerge ok"},
		{"DEL stripped", "a\u007fb", "a\uFFFDb"},
		{"C1 control stripped", "a\u009bb", "a\uFFFDb"},
		{"bidi RLO stripped", "alice\u202eevil", "alice\uFFFDevil"},
		{"bidi PDI stripped", "a\u2069b", "a\uFFFDb"},
		{"implicit LRM kept", "a\u200eb", "a\u200eb"},
		{"unicode letters kept", "Märchen", "Märchen"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := safeForStderr(c.in); got != c.want {
				t.Errorf("safeForStderr(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestMergeDriver_ConflictWithAnsiEscapeInValue_StderrSanitized(t *testing.T) {
	requireGit(t)
	dst := t.TempDir()
	// Both branches modify a property to different values, one containing an
	// ANSI clear-screen + fake-success sequence. The driver should print the
	// hostile value to stderr without honoring the escape.
	base := "persons:\n  person-john-smith:\n    properties:\n      note: \"original\"\n"
	ours := "persons:\n  person-john-smith:\n    properties:\n      note: \"ours-version\"\n"
	theirs := "persons:\n  person-john-smith:\n    properties:\n      note: \"\\u001b[2J\\u001b[Hmerge ok\"\n"
	if err := os.WriteFile(filepath.Join(dst, "base.glx"), []byte(base), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "ours.glx"), []byte(ours), 0o644); err != nil {
		t.Fatalf("write ours: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "theirs.glx"), []byte(theirs), 0o644); err != nil {
		t.Fatalf("write theirs: %v", err)
	}
	in := mergeDriverInputs{
		BasePath:   filepath.Join(dst, "base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   "person-john-smith.glx",
	}
	var errBuf bytes.Buffer
	_ = runMergeDriver(in, &errBuf)

	if strings.ContainsRune(errBuf.String(), 0x1B) {
		t.Errorf("stderr must not contain literal ESC (0x1B) after sanitization, got:\n%q", errBuf.String())
	}
	// The replacement glyph confirms sanitization actually triggered.
	if !strings.ContainsRune(errBuf.String(), '�') {
		t.Errorf("stderr should contain U+FFFD replacement after stripping ANSI, got:\n%q", errBuf.String())
	}
}

func TestMergeDriver_ParseOrEmpty_TreatsEmptyInputAsEmptyGLXFile(t *testing.T) {
	deser := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false})

	g, err := parseOrEmpty(deser, []byte(""))
	if err != nil {
		t.Fatalf("parseOrEmpty empty: %v", err)
	}
	if g == nil {
		t.Fatalf("parseOrEmpty empty returned nil GLXFile")
	}

	g2, err := parseOrEmpty(deser, []byte("   \n\t  "))
	if err != nil {
		t.Fatalf("parseOrEmpty whitespace: %v", err)
	}
	if g2 == nil {
		t.Fatalf("parseOrEmpty whitespace returned nil GLXFile")
	}
}
