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
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

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

// writeLargeTextFile streams `size` bytes of filler text to path using a small
// reusable buffer, so tests can create files larger than the merge-driver cap
// without allocating the whole (64+ MiB) payload in memory. The content is
// plain 'x' bytes so git merge-file treats the file as text.
func writeLargeTextFile(t *testing.T, path string, size int) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("create %q: %v", path, err)
	}

	chunk := bytes.Repeat([]byte("x"), 64*1024)
	for remaining := size; remaining > 0; {
		n := min(len(chunk), remaining)
		if _, werr := f.Write(chunk[:n]); werr != nil {
			f.Close()
			t.Fatalf("write %q: %v", path, werr)
		}
		remaining -= n
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %q: %v", path, err)
	}
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

	// theirs is over the cap; ours and base are small with diverging content
	// so git merge-file's line-based merge has something to chew on. 1 KiB
	// beyond the cap is enough — the threshold is exercised regardless of how
	// much we exceed it by. The oversized file is streamed to disk rather than
	// built in memory (see writeLargeTextFile) to keep the test cheap.
	const overage = 1024

	if err := os.WriteFile(filepath.Join(dst, "base.glx"), []byte("base content\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dst, "ours.glx"), []byte("ours content\n"), 0o644); err != nil {
		t.Fatalf("write ours: %v", err)
	}
	writeLargeTextFile(t, filepath.Join(dst, "theirs.glx"), maxMergeInputBytes+overage)

	originalOurs, err := os.ReadFile(filepath.Join(dst, "ours.glx"))
	if err != nil {
		t.Fatalf("read original ours: %v", err)
	}

	in := mergeDriverInputs{
		BasePath:   filepath.Join(dst, "base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   "oversized.glx",
	}
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	// Either clean (rare — git might auto-merge despite divergence) or
	// nonzero (typical — conflict markers written). Either way, the
	// fallback must have run.
	_ = code

	if !strings.Contains(errBuf.String(), "exceeds merge-driver cap") {
		t.Errorf("expected stderr to mention the size cap, got:\n%s", errBuf.String())
	}

	// Verify the fallback actually ran: %A must have been touched by
	// git merge-file, not left as the original "ours content\n".
	merged, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read merged: %v", err)
	}
	if bytes.Equal(originalOurs, merged) {
		t.Errorf("oursPath unchanged — fallback did not run; size-cap path likely exited early.\nstderr was:\n%s", errBuf.String())
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

// TestMergeDriver_ConflictWithNewlineInValue_StderrEscaped pins the
// log-injection guard added in response to Copilot review on PR #906: a
// hostile YAML value can contain a newline that, if rendered literally,
// would forge a "  conflict at <smuggled-path>" line in the conflict
// summary, fooling a researcher into thinking there was a real conflict at
// a path of the attacker's choosing. formatValue must escape the newline
// so the value stays on one physical line — the smuggled token must only
// appear as a substring of theirs's escaped value, never as a stand-alone
// summary line.
func TestMergeDriver_ConflictWithNewlineInValue_StderrEscaped(t *testing.T) {
	requireGit(t)
	dst := t.TempDir()
	// Both branches modify the same property to different values; theirs
	// embeds a newline followed by a fake summary line. Pre-fix, that line
	// would survive verbatim in stderr; post-fix it appears only as the
	// escaped substring `\n  conflict at /etc/secret`.
	base := "persons:\n  person-john-smith:\n    properties:\n      note: \"original\"\n"
	ours := "persons:\n  person-john-smith:\n    properties:\n      note: \"ours-version\"\n"
	theirs := "persons:\n  person-john-smith:\n    properties:\n      note: \"theirs\\n  conflict at /etc/secret\"\n"
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
	out := errBuf.String()

	// Positive: the escaping must have fired, so the literal two-character
	// sequence `\n  conflict at /etc/secret` is present.
	if !strings.Contains(out, `\n  conflict at /etc/secret`) {
		t.Errorf("expected newline in theirs's value to be escaped to literal \\n, got stderr:\n%s", out)
	}

	// Anti-injection: the smuggled string must never appear at the start of
	// a physical line (which is the shape of a real conflict-summary entry).
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "  conflict at /etc/secret") {
			t.Errorf("smuggled '  conflict at /etc/secret' appeared as a standalone line — escaping failed:\n%s", out)

			break
		}
	}
}

// TestFormatValue_TruncatesOnRuneBoundary pins the UTF-8 fix from Copilot
// review on PR #906: the display cap used to slice by byte index, so a value
// of accented or non-Latin characters — routine in genealogical data — got
// cut mid-rune and emitted invalid UTF-8 into the conflict summary.
func TestFormatValue_TruncatesOnRuneBoundary(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"2-byte runes", strings.Repeat("ü", 500)},
		{"3-byte runes", strings.Repeat("漢", 500)},
		{"4-byte runes", strings.Repeat("𐌰", 500)},
		{"mixed", strings.Repeat("Müller-漢字-", 100)},
		// Lands the cut exactly one byte into a multi-byte rune under the
		// old byte-slicing implementation.
		{"boundary straddle", strings.Repeat("a", maxConflictValueDisplay-1) + "ü" + "tail"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatValue(c.in)
			if !utf8.ValidString(got) {
				t.Errorf("formatValue produced invalid UTF-8: %q", got)
			}
			if !strings.HasSuffix(got, "…") {
				t.Errorf("expected truncation ellipsis, got %q", got)
			}
			if n := utf8.RuneCountInString(got); n != maxConflictValueDisplay+1 {
				t.Errorf("expected %d runes + ellipsis, got %d: %q", maxConflictValueDisplay, n, got)
			}
			if strings.ContainsRune(got, '�') {
				t.Errorf("truncation split a rune (U+FFFD in output): %q", got)
			}
		})
	}
}

func TestFormatValue_ShortValueNotTruncated(t *testing.T) {
	in := "Märchen für Großmutter"
	if got := formatValue(in); got != in {
		t.Errorf("formatValue(%q) = %q, want unchanged", in, got)
	}
}

func TestSafeInlineForStderr(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "hello", "hello"},
		{"newline escaped", "a\nb", `a\nb`},
		{"tab escaped", "a\tb", `a\tb`},
		{"forged summary line", "ok\n  conflict at /etc/secret", `ok\n  conflict at /etc/secret`},
		{"ANSI still stripped", "1850\x1b[2Jok", "1850\uFFFD[2Jok"},
		{"bidi RLO still stripped", "alice\u202eevil", "alice\uFFFDevil"},
		{"unicode letters kept", "Märchen", "Märchen"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := safeInlineForStderr(c.in); got != c.want {
				t.Errorf("safeInlineForStderr(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestMergeDriver_NewlineInPathAndEntityID_StderrEscaped covers the second
// half of the log-injection guard raised in Copilot review on PR #906. The
// value call sites were already escaped; the *heading* call sites were not,
// so both git's %P (an arbitrary repository pathname) and an entity ID out of
// YAML the driver parses with validation disabled could forge extra
// "[glx merge-driver] file=…" / "  conflict at …" lines. Neither smuggled
// line may appear standalone in the summary.
func TestMergeDriver_NewlineInPathAndEntityID_StderrEscaped(t *testing.T) {
	requireGit(t)
	dst := t.TempDir()
	// The entity ID carries a newline plus a forged conflict line; %P carries
	// a newline plus a forged driver header.
	base := "persons:\n  \"person-john\\n  conflict at /etc/passwd\":\n    properties:\n      note: \"original\"\n"
	ours := "persons:\n  \"person-john\\n  conflict at /etc/passwd\":\n    properties:\n      note: \"ours-version\"\n"
	theirs := "persons:\n  \"person-john\\n  conflict at /etc/passwd\":\n    properties:\n      note: \"theirs-version\"\n"
	for name, content := range map[string]string{"base.glx": base, "ours.glx": ours, "theirs.glx": theirs} {
		if err := os.WriteFile(filepath.Join(dst, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	in := mergeDriverInputs{
		BasePath:   filepath.Join(dst, "base.glx"),
		OursPath:   filepath.Join(dst, "ours.glx"),
		TheirsPath: filepath.Join(dst, "theirs.glx"),
		OrigPath:   "innocent.glx\n[glx merge-driver] file=decoy.glx — auto-resolved by the driver:",
	}
	var errBuf bytes.Buffer
	_ = runMergeDriver(in, &errBuf)
	out := errBuf.String()

	// Positive: both newlines must have been escaped to a literal \n.
	if !strings.Contains(out, `innocent.glx\n[glx merge-driver] file=decoy.glx`) {
		t.Errorf("expected newline in origPath to be escaped to literal \\n, got stderr:\n%s", out)
	}
	if !strings.Contains(out, `person-john\n  conflict at /etc/passwd`) {
		t.Errorf("expected newline in entity ID to be escaped to literal \\n, got stderr:\n%s", out)
	}

	// Anti-injection: exactly one driver header, and no forged conflict line.
	headers := 0
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(line, "[glx merge-driver]") {
			headers++
		}
		if strings.HasPrefix(line, "  conflict at /etc/passwd") {
			t.Errorf("smuggled '  conflict at /etc/passwd' appeared as a standalone line — escaping failed:\n%s", out)
		}
	}
	if headers != 1 {
		t.Errorf("expected exactly 1 '[glx merge-driver]' header line, got %d:\n%s", headers, out)
	}
}

// TestMergeDriver_ConflictOnAssertion_StderrIncludesEvidence drives an
// assertion-value conflict with equal confidence on both sides (so neither
// wins) and citations set on both sides. The stderr summary should include
// the per-side confidence and citation refs — that's the whole point of
// special-casing assertions in assertionMetaForConflict.
func TestMergeDriver_ConflictOnAssertion_StderrIncludesEvidence(t *testing.T) {
	requireGit(t)
	dst := t.TempDir()
	base := `assertions:
  assertion-john-birth:
    subject:
      event: event-birth
    property: date
    value: "1850"
    confidence: medium
    citations:
      - citation-base
`
	ours := `assertions:
  assertion-john-birth:
    subject:
      event: event-birth
    property: date
    value: "1850-04-12"
    confidence: medium
    citations:
      - citation-base
      - citation-parish-register
`
	theirs := `assertions:
  assertion-john-birth:
    subject:
      event: event-birth
    property: date
    value: "1850-04-15"
    confidence: medium
    citations:
      - citation-base
      - citation-1851-census
`
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
		OrigPath:   "assertion-john-birth.glx",
	}
	var errBuf bytes.Buffer

	code := runMergeDriver(in, &errBuf)
	if code == mergeDriverExitClean {
		t.Fatalf("expected nonzero exit (equal confidence falls back), got %d", code)
	}
	out := errBuf.String()
	// The runner should mention the conflict path under assertions[…].value.
	if !strings.Contains(out, "assertions[assertion-john-birth].value") {
		t.Errorf("expected stderr to name the conflict path, got:\n%s", out)
	}
	// Both sides' confidence is medium — that string should appear.
	if !strings.Contains(out, "conf=medium") {
		t.Errorf("expected stderr to show conf=medium, got:\n%s", out)
	}
	// Both sides' newly-added citations should be visible in the evidence
	// suffix so the researcher can pick.
	if !strings.Contains(out, "citation-parish-register") {
		t.Errorf("expected stderr to include ours' citation, got:\n%s", out)
	}
	if !strings.Contains(out, "citation-1851-census") {
		t.Errorf("expected stderr to include theirs' citation, got:\n%s", out)
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

// TestMergeDriver_RefusesToWriteThroughSymlink pins the symlink guard added
// in response to Copilot review on PR #906: if %A (ours) is a symlink — the
// realistic shape of which is a hostile branch placing a *.glx symlink at a
// tracked path so a merge would clobber a file outside the worktree — the
// driver must refuse the write, return a nonzero exit code, leave the
// symlink target untouched, and surface the refusal to stderr.
func TestMergeDriver_RefusesToWriteThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		// os.Symlink on Windows needs SeCreateSymbolicLinkPrivilege (admin
		// or Developer Mode), which CI test envs don't reliably grant. The
		// Lstat guard itself is portable; the integration check skips here.
		t.Skip("symlink creation requires elevated permissions on Windows")
	}

	in := stageMergeFixture(t, "clean-one-sided")

	// Replace OursPath with a symlink to an attacker-controlled file living
	// outside the staged tempdir, simulating "ours -> /etc/passwd".
	victim := filepath.Join(t.TempDir(), "victim.txt")
	victimContent := []byte("untouched\n")
	if err := os.WriteFile(victim, victimContent, 0o644); err != nil {
		t.Fatalf("write victim: %v", err)
	}
	if err := os.Remove(in.OursPath); err != nil {
		t.Fatalf("remove staged ours: %v", err)
	}
	if err := os.Symlink(victim, in.OursPath); err != nil {
		t.Fatalf("create ours -> victim symlink: %v", err)
	}

	var errBuf bytes.Buffer
	code := runMergeDriver(in, &errBuf)

	if code == mergeDriverExitClean {
		t.Fatalf("expected nonzero exit when ours is a symlink, got clean")
	}

	// The victim must be byte-for-byte unchanged — the write must not have
	// been followed through the symlink.
	after, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("read victim after merge: %v", err)
	}
	if !bytes.Equal(after, victimContent) {
		t.Errorf("victim file modified through symlink:\nbefore: %q\nafter:  %q",
			victimContent, after)
	}

	// The symlink itself must still be a symlink — we must not have replaced
	// it with a regular file via WriteFile.
	fi, err := os.Lstat(in.OursPath)
	if err != nil {
		t.Fatalf("lstat ours after merge: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("ours is no longer a symlink — guard should preserve it")
	}

	if !strings.Contains(errBuf.String(), "refusing to write through symlink") {
		t.Errorf("expected symlink-refusal diagnostic in stderr, got:\n%s", errBuf.String())
	}
}

// requireUnprivilegedPosix skips tests that depend on directory permission
// bits actually denying an operation. Windows doesn't model them the same way,
// and root bypasses them entirely — CI containers frequently run as root, where
// a "read-only" directory is still writable and the test would fail spuriously.
func requireUnprivilegedPosix(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX directory permission bits are not modeled on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root; directory permission bits do not deny writes")
	}
}

// TestMergeDriver_CleanMerge_ReplacesOursByRename pins the atomic-write
// behavior added in response to Copilot review on PR #906.
//
// It is the direct check of the property Copilot asked for: %A must be
// replaced by renaming a fully-written temp file over it, never truncated and
// rewritten in place. A truncating writer (os.WriteFile) leaves a window where
// %A holds a partial file, so a crash, full disk, or killed process mid-write
// destroys the researcher's working-tree content even though the driver
// reports failure and git falls back to the text merge.
//
// The observable difference is file identity: os.WriteFile keeps the same
// inode, a rename installs a new one. Comparing identity discriminates between
// the two implementations without having to induce a real mid-write failure —
// which can't be done portably.
func TestMergeDriver_CleanMerge_ReplacesOursByRename(t *testing.T) {
	if runtime.GOOS == "windows" {
		// os.Stat on Windows doesn't capture the file index up front; SameFile
		// resolves it lazily by reopening the path at comparison time. Both
		// FileInfos here come from the same path, so after the rename they'd
		// both resolve to the new file and compare equal no matter which
		// implementation ran. The check is meaningful only where Stat captures
		// the inode eagerly.
		t.Skip("os.SameFile resolves identity lazily by path on Windows; cannot observe replacement")
	}

	in := stageMergeFixture(t, "clean-one-sided")

	beforeInfo, err := os.Stat(in.OursPath)
	if err != nil {
		t.Fatalf("stat staged ours: %v", err)
	}

	var errBuf bytes.Buffer
	if code := runMergeDriver(in, &errBuf); code != mergeDriverExitClean {
		t.Fatalf("expected clean exit, got %d (stderr: %s)", code, errBuf.String())
	}

	afterInfo, err := os.Stat(in.OursPath)
	if err != nil {
		t.Fatalf("stat merged ours: %v", err)
	}

	if os.SameFile(beforeInfo, afterInfo) {
		t.Error("merged result was written in place — %A must be replaced by renaming a " +
			"complete temp file over it, so a failed write can never leave a partial file")
	}
}

// TestMergeDriver_WriteFailure_LeavesOursIntact covers the other half of the
// atomic-write contract: when the write fails outright, the driver must report
// nonzero and leave %A holding the original "ours" content for git's text-merge
// fallback to work from.
//
// The failure is induced by making the staging directory unwritable, which
// blocks the temp file's creation. Note this is not a truncation scenario —
// a 0555 directory still permits writing to an existing file, so os.WriteFile
// would have succeeded here. Truncation safety is covered by
// TestMergeDriver_CleanMerge_ReplacesOursByRename; this test pins the
// error-reporting and no-partial-content behavior of the failure path.
func TestMergeDriver_WriteFailure_LeavesOursIntact(t *testing.T) {
	requireUnprivilegedPosix(t)

	// clean-one-sided merges cleanly, so the run reaches the write and fails
	// there rather than diverting to the conflict fallback earlier.
	in := stageMergeFixture(t, "clean-one-sided")

	before, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read staged ours: %v", err)
	}

	dir := filepath.Dir(in.OursPath)
	// r-xr-xr-x: base/ours/theirs stay readable so the merge itself proceeds,
	// but no new entry (the temp file) can be created.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod staging dir read-only: %v", err)
	}
	// Restore before t.TempDir cleanup, which needs to remove the contents.
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	var errBuf bytes.Buffer
	code := runMergeDriver(in, &errBuf)

	if code == mergeDriverExitClean {
		t.Fatalf("expected nonzero exit when the merged write fails, got clean")
	}
	if !strings.Contains(errBuf.String(), "write ours") {
		t.Errorf("expected a write-failure diagnostic in stderr, got:\n%s", errBuf.String())
	}

	after, err := os.ReadFile(in.OursPath)
	if err != nil {
		t.Fatalf("read ours after failed write: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("ours was modified by a failed write — must be all-or-nothing\nbefore (%d bytes):\n%s\nafter (%d bytes):\n%s",
			len(before), before, len(after), after)
	}
}

// TestMergeDriver_CleanMerge_PreservesFileMode guards a regression the switch
// to atomicWriteFile could introduce: it creates a new file and renames it over
// the target, so unlike os.WriteFile it does not inherit the replaced file's
// mode. A worktree checked out under a restrictive umask (a 0600 .glx) must not
// come back world-readable after a structural merge.
func TestMergeDriver_CleanMerge_PreservesFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not modeled on Windows")
	}

	in := stageMergeFixture(t, "clean-one-sided")

	const restrictive os.FileMode = 0o600
	if err := os.Chmod(in.OursPath, restrictive); err != nil {
		t.Fatalf("chmod ours: %v", err)
	}

	var errBuf bytes.Buffer
	if code := runMergeDriver(in, &errBuf); code != mergeDriverExitClean {
		t.Fatalf("expected clean exit, got %d (stderr: %s)", code, errBuf.String())
	}

	fi, err := os.Stat(in.OursPath)
	if err != nil {
		t.Fatalf("stat merged ours: %v", err)
	}
	if got := fi.Mode().Perm(); got != restrictive {
		t.Errorf("merged file mode = %04o, want %04o (atomic write must not widen permissions)", got, restrictive)
	}
}

// TestMergeDriver_CleanMerge_LeavesNoTempFiles checks that the atomic write's
// temp file is renamed into place rather than left beside %A. A stray
// .glx-tmp-* in the worktree would show up as an untracked file in the
// researcher's `git status` right after every merge.
func TestMergeDriver_CleanMerge_LeavesNoTempFiles(t *testing.T) {
	in := stageMergeFixture(t, "clean-one-sided")

	var errBuf bytes.Buffer
	if code := runMergeDriver(in, &errBuf); code != mergeDriverExitClean {
		t.Fatalf("expected clean exit, got %d (stderr: %s)", code, errBuf.String())
	}

	entries, err := os.ReadDir(filepath.Dir(in.OursPath))
	if err != nil {
		t.Fatalf("read staging dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".glx-tmp-") {
			t.Errorf("atomic write left a temp file behind: %s", e.Name())
		}
	}
}
