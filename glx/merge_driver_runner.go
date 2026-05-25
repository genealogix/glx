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
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// mergedFilePerm is the permission bits used when overwriting %A. git owns
// the file path and re-applies any per-repo umask after the merge driver
// returns, so the value matches the OS default for regular files.
const mergedFilePerm = 0o644

// maxConflictValueDisplay caps the per-conflict stderr value length so a
// pathologically large structured value doesn't drown the diagnostic block.
const maxConflictValueDisplay = 200

// maxMergeInputBytes caps the per-file size the merge driver will parse as
// structured YAML. Files larger than this are handled by git's text merge
// fallback (which streams rather than holding the file in memory). Caps the
// blast radius of a hostile branch shipping a YAML bomb / multi-GB .glx.
// 64 MiB is ~6500× the size of a realistic single-entity GLX file (~10 KiB)
// and three orders of magnitude above the largest .glx in any of the example
// archives shipped with this repo.
const maxMergeInputBytes = 64 * 1024 * 1024

// asciiDel is the C0 DEL control character. Named so the sanitizer doesn't
// trip the magic-number linter and the intent is obvious at the call site.
const asciiDel = 0x7F

// errMergeInputTooLarge is returned by readCappedFile when a merge input
// exceeds maxMergeInputBytes. The runner reports it via stderr and falls
// back to git's text merge.
var errMergeInputTooLarge = errors.New("merge input exceeds merge-driver cap")

// mergeDriverInputs is what git passes us. Named fields make the runner
// signature self-documenting and let tests construct it without positional
// ambiguity.
type mergeDriverInputs struct {
	BasePath   string // %O — common ancestor (may be an empty file when both branches added)
	OursPath   string // %A — our side; the merge driver writes the result here
	TheirsPath string // %B — their side
	OrigPath   string // %P — original pathname; used in diagnostics only
}

// mergeDriverExitCode is the exit code returned to git. 0 = clean merge,
// nonzero = conflicts remain in OursPath.
type mergeDriverExitCode int

const (
	mergeDriverExitClean    mergeDriverExitCode = 0
	mergeDriverExitConflict mergeDriverExitCode = 1
)

// runMergeDriver is the entry point invoked by the Cobra command. Reads the
// three input files, attempts a structural 3-way merge, and falls back to
// `git merge-file` when conflicts remain. Returns the exit code git should
// see (writes to OursPath and to errOut as side effects).
//
// errOut is used for the conflict summary so the researcher sees both
// sides' values + assertion evidence while resolving. The fall-back also
// writes git's own messages to errOut.
func runMergeDriver(in mergeDriverInputs, errOut io.Writer) mergeDriverExitCode {
	baseBytes, oursBytes, theirsBytes, readErr := readAllInputs(in, errOut)
	if readErr != nil {
		// Size-cap rejection means the files exist and are readable; we
		// just declined to parse them structurally. Hand them off to git's
		// text merge — it streams line-by-line and isn't bounded by our
		// in-memory cap. Other read failures (missing file, permission
		// denied) would just make git merge-file fail too, so exit nonzero.
		if errors.Is(readErr, errMergeInputTooLarge) {
			return execGitMergeFile(in, errOut)
		}

		return mergeDriverExitConflict
	}

	// Deserialize without validation — the merge driver runs per-file, so
	// cross-reference validation against entities in other files would always
	// fail and isn't this layer's job.
	deser := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false})

	base, ours, theirs, parseOK := parseAllInputs(deser, baseBytes, oursBytes, theirsBytes, errOut)
	if !parseOK {
		return execGitMergeFile(in, errOut)
	}

	merged, conflicts := glxlib.ThreeWayMerge(base, ours, theirs)

	if glxlib.HasUnresolvedConflict(conflicts) {
		printConflictSummary(errOut, in.OrigPath, conflicts, ours, theirs)

		return execGitMergeFile(in, errOut)
	}

	// Surface auto-resolved decisions even on the clean path so the
	// researcher knows why their value was picked over theirs (or vice versa).
	if len(conflicts) > 0 {
		printAutoResolvedSummary(errOut, in.OrigPath, conflicts)
	}

	mergedBytes, err := deser.SerializeSingleFileBytes(merged)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] serialize merged: %v — falling back to text merge\n", err)

		return execGitMergeFile(in, errOut)
	}

	// Only touch OursPath now that we know we have a clean structural merge.
	// Up to this point OursPath still holds the original "ours" bytes that
	// git invoked us with, so if anything fails above we hand a pristine
	// %A off to `git merge-file`.
	if err := os.WriteFile(in.OursPath, mergedBytes, mergedFilePerm); err != nil {
		fprintf(errOut, "[glx merge-driver] write ours %q: %v\n", in.OursPath, err)

		return mergeDriverExitConflict
	}

	return mergeDriverExitClean
}

// readAllInputs reads base/ours/theirs from disk, reporting each failure to
// errOut. Returns the first error encountered, with nil byte slices on
// failure. The caller checks for errMergeInputTooLarge specifically to
// decide between hard-exit and text-merge fallback.
func readAllInputs(in mergeDriverInputs, errOut io.Writer) (base, ours, theirs []byte, err error) {
	base, err = readCappedFile(in.BasePath)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] read base %q: %v\n", in.BasePath, err)

		return nil, nil, nil, err
	}
	ours, err = readCappedFile(in.OursPath)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] read ours %q: %v\n", in.OursPath, err)

		return nil, nil, nil, err
	}
	theirs, err = readCappedFile(in.TheirsPath)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] read theirs %q: %v\n", in.TheirsPath, err)

		return nil, nil, nil, err
	}

	return base, ours, theirs, nil
}

// readCappedFile is os.ReadFile with a size ceiling. A file larger than
// maxMergeInputBytes returns errMergeInputTooLarge so the caller hands the
// merge off to git's text merge instead of trying to parse a multi-gigabyte
// input.
//
// The path argument originates from git's merge-driver invocation
// (%O / %A / %B), which git always supplies as a path inside the worktree
// or git's own tempdir. The runner does no path arithmetic on top of it.
// G304 here is a false positive in that contract.
func readCappedFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxMergeInputBytes {
		return nil, fmt.Errorf("%w: %s is %d bytes (cap %d)",
			errMergeInputTooLarge, path, info.Size(), maxMergeInputBytes)
	}

	return os.ReadFile(path) //#nosec G304 -- path is supplied by git per the merge-driver contract; see func doc.
}

// parseAllInputs deserializes the three input byte slices. On any failure it
// writes a "falling back to text merge" note to errOut and returns ok=false,
// so the caller can hand the still-untouched files off to git merge-file.
func parseAllInputs(deser glxlib.Serializer, baseBytes, oursBytes, theirsBytes []byte, errOut io.Writer) (base, ours, theirs *glxlib.GLXFile, ok bool) {
	base, err := parseOrEmpty(deser, baseBytes)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] parse base: %v — falling back to text merge\n", err)

		return nil, nil, nil, false
	}
	ours, err = parseOrEmpty(deser, oursBytes)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] parse ours: %v — falling back to text merge\n", err)

		return nil, nil, nil, false
	}
	theirs, err = parseOrEmpty(deser, theirsBytes)
	if err != nil {
		fprintf(errOut, "[glx merge-driver] parse theirs: %v — falling back to text merge\n", err)

		return nil, nil, nil, false
	}

	return base, ours, theirs, true
}

// parseOrEmpty deserializes GLX YAML bytes, treating empty input as an empty
// GLXFile (git passes an empty %O when both branches added the file fresh).
// Uses bytes.TrimSpace to avoid copying potentially large []byte → string
// when probing for emptiness — the post-cap input may still be up to
// maxMergeInputBytes large, which would be a 64 MiB allocation per side.
func parseOrEmpty(deser glxlib.Serializer, data []byte) (*glxlib.GLXFile, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return &glxlib.GLXFile{}, nil
	}

	return deser.DeserializeSingleFileBytes(data)
}

// execGitMergeFile runs `git merge-file -L ours -L base -L theirs %A %O %B`.
// This is git's standard text-merge implementation; it writes <<<<<<< markers
// into %A on conflict. Returns the exit code (0 = clean merge, 1 = conflicts,
// >1 = error).
//
// The three paths come from git's merge-driver invocation. They're passed as
// argv to exec.CommandContext (not interpolated into a shell command), so
// there's no shell-injection surface; the worst a hostile path could do is
// fail to open. G204 here is a false positive in that contract.
func execGitMergeFile(in mergeDriverInputs, errOut io.Writer) mergeDriverExitCode {
	cmd := exec.CommandContext(context.Background(), "git", "merge-file", //#nosec G204 -- argv (no shell); paths supplied by git per merge-driver contract.
		"-L", "ours", "-L", "base", "-L", "theirs",
		in.OursPath, in.BasePath, in.TheirsPath)
	cmd.Stderr = errOut

	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return mergeDriverExitCode(ee.ExitCode())
		}
		fprintf(errOut, "[glx merge-driver] git merge-file failed: %v\n", err)

		return mergeDriverExitConflict
	}

	return mergeDriverExitClean
}

// printConflictSummary writes a structured block per unresolved conflict to
// errOut. For assertion-property conflicts it pulls Confidence and Citations
// from the original ours/theirs entities so the researcher sees the evidence
// alongside the values they have to choose between.
//
// All user-controlled strings (entity IDs from YAML, property values, citation
// IDs, the original path) are passed through safeForStderr before formatting
// so a malicious branch can't embed ANSI escapes or Unicode bidi marks that
// would mislead the researcher resolving the merge.
func printConflictSummary(errOut io.Writer, origPath string, conflicts []glxlib.Merge3Conflict, ours, theirs *glxlib.GLXFile) {
	fprintf(errOut, "[glx merge-driver] file=%s\n", safeForStderr(origPath))
	for i := range conflicts {
		c := &conflicts[i]
		if c.AutoResolved {
			continue
		}
		oursMeta, theirsMeta := assertionMetaForConflict(c, ours, theirs)
		fprintf(errOut, "  conflict at %s\n", safeForStderr(c.Path))
		fprintf(errOut, "    ours    : %s%s\n", formatValue(c.OursValue), oursMeta)
		fprintf(errOut, "    theirs  : %s%s\n", formatValue(c.TheirsValue), theirsMeta)
	}
}

// printAutoResolvedSummary tells the researcher when the driver picked a
// winner on their behalf — relevant for confidence-based resolution where
// the file lands without conflict markers but a real decision was made.
func printAutoResolvedSummary(errOut io.Writer, origPath string, conflicts []glxlib.Merge3Conflict) {
	hasAuto := false
	for i := range conflicts {
		if conflicts[i].AutoResolved {
			hasAuto = true

			break
		}
	}
	if !hasAuto {
		return
	}

	fprintf(errOut, "[glx merge-driver] file=%s — auto-resolved by the driver:\n", safeForStderr(origPath))
	for i := range conflicts {
		c := &conflicts[i]
		if !c.AutoResolved {
			continue
		}
		fprintf(errOut, "  %s → %s\n    ours   : %s\n    theirs : %s\n",
			safeForStderr(c.Path), safeForStderr(c.Resolution),
			formatValue(c.OursValue), formatValue(c.TheirsValue))
	}
}

// assertionMetaForConflict returns suffix strings like "  conf=high  cites=[a,b]"
// when the conflict's entity is an Assertion present in ours/theirs. Empty
// strings otherwise.
func assertionMetaForConflict(c *glxlib.Merge3Conflict, ours, theirs *glxlib.GLXFile) (oursSuffix, theirsSuffix string) {
	if c.EntityType != glxlib.EntityTypeAssertions || c.EntityID == "" {
		return "", ""
	}
	if a, ok := ours.Assertions[c.EntityID]; ok && a != nil {
		oursSuffix = assertionMetaSuffix(a)
	}
	if a, ok := theirs.Assertions[c.EntityID]; ok && a != nil {
		theirsSuffix = assertionMetaSuffix(a)
	}

	return oursSuffix, theirsSuffix
}

func assertionMetaSuffix(a *glxlib.Assertion) string {
	var parts []string
	if a.Confidence != "" {
		parts = append(parts, "conf="+safeForStderr(a.Confidence))
	}
	if len(a.Citations) > 0 {
		parts = append(parts, "cites=["+safeForStderr(strings.Join(a.Citations, ","))+"]")
	}
	if len(a.Sources) > 0 {
		parts = append(parts, "sources=["+safeForStderr(strings.Join(a.Sources, ","))+"]")
	}
	if len(parts) == 0 {
		return ""
	}

	return "  " + strings.Join(parts, "  ")
}

// formatValue stringifies a conflict's value for stderr display. Designed
// for stderr-readability, not round-trippability — long structured values
// (entire entities, large maps) are still shown but trimmed. The result is
// sanitized of ANSI escapes and Unicode bidi controls so a hostile branch
// can't inject misleading terminal output.
func formatValue(v any) string {
	if v == nil {
		return "<absent>"
	}
	s := fmt.Sprintf("%v", v)
	if len(s) > maxConflictValueDisplay {
		s = s[:maxConflictValueDisplay] + "…"
	}

	return safeForStderr(s)
}

// safeForStderr strips control characters and Unicode bidi marks from a
// string before it lands on stderr. C0 controls (other than tab and newline,
// which are display-safe and present in legitimate values), DEL, and the
// Unicode bidi-control range (LRE/RLE/PDF/LRO/RLO and LRI/RLI/FSI/PDI) are
// replaced with U+FFFD. Without this, a YAML value like
// `"1850\x1b[2J\x1b[Hmerge ok"` would clear the user's terminal and print a
// fake success message when the driver writes its conflict summary.
func safeForStderr(s string) string {
	needsSanitize := false
	for _, r := range s {
		if isUnsafeForStderr(r) {
			needsSanitize = true

			break
		}
	}
	if !needsSanitize {
		return s
	}

	out := make([]rune, 0, len(s))
	for _, r := range s {
		if isUnsafeForStderr(r) {
			out = append(out, '�')

			continue
		}
		out = append(out, r)
	}

	return string(out)
}

func isUnsafeForStderr(r rune) bool {
	// C0 controls except tab and newline.
	if r < 0x20 && r != '\t' && r != '\n' {
		return true
	}
	// DEL.
	if r == asciiDel {
		return true
	}
	// C1 controls (0x80–0x9F).
	if r >= 0x80 && r <= 0x9F {
		return true
	}
	// Unicode bidi controls: explicit-direction (LRE/RLE/PDF/LRO/RLO) and
	// isolate (LRI/RLI/FSI/PDI). Don't strip the implicit LRM/RLM/ALM marks,
	// which are legitimate in mixed-script text.
	if (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
		return true
	}

	return false
}

// fprintf wraps fmt.Fprintf and discards its error return — all callers in
// this file write to stderr where an I/O failure is both unrecoverable and
// the only signal we'd want to surface (the merge driver itself would
// already be exiting nonzero). Keeping the wrapper localized avoids
// scattering `//nolint:errcheck` across every diagnostic line.
func fprintf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
