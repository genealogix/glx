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
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// repo is the deterministic ground truth a scan checks each memory file against.
type repo struct {
	// makeTargets is the set of phony/rule targets declared in the Makefile.
	makeTargets map[string]bool
	// modulePath is the module line from go.mod, e.g. "github.com/genealogix/glx".
	modulePath string
	// orgPrefix is the host/org prefix of modulePath, e.g.
	// "github.com/genealogix/". Only references under this prefix are treated as
	// claims about this module (third-party imports are left alone).
	orgPrefix string
	// exists reports whether a repo-relative path (forward slashes) resolves on
	// disk, and if so whether it is a directory. Returning the kind lets a
	// directory claim (`docs/cli/`) reject a same-named regular file and vice
	// versa. Injected so the scan core stays pure/testable.
	exists func(repoRel string) (found, isDir bool)
}

// pathMeta are glob / regex / shell metacharacters. A token containing any of
// them is a pattern or command fragment (e.g. `glx/cmd_*.go`, `docs/**`), not a
// literal path to verify.
const pathMeta = "*?[]{}<>|$()!"

// knownExts is the set of file extensions that mark a slash-bearing token as a
// concrete file reference. Requiring a recognized extension (together with a
// slash) is what separates a real path like `testdata/gedcom/shakespeare.ged`
// from look-alikes such as the GitHub action `actions/checkout` or the stdlib
// import `os/exec`, which carry a slash but no file extension.
var knownExts = map[string]bool{
	"go": true, "mod": true, "sum": true,
	"md": true, "txt": true, "pdf": true,
	"json": true, "yaml": true, "yml": true, "toml": true, "xml": true, "csv": true,
	"mjs": true, "js": true, "ts": true, "tsx": true, "jsx": true,
	"sh": true, "glx": true, "ged": true,
	"html": true, "css": true, "lock": true, "cfg": true, "ini": true,
}

var (
	inlineSpanRe = regexp.MustCompile("`([^`]+)`")
	// makeRe matches a `make <target>` invocation anywhere in an inline code span
	// (spans are short and unambiguous, e.g. `make test`). The target class
	// matches the Makefile target grammar parsed by makeTargetLineRe (letters,
	// digits, `_`, `-`) so a target like `check_schemas` isn't truncated at the
	// underscore into a spurious `make check`. The >= 3-character floor keeps
	// prose words like "make it"/"make an" from masquerading as targets; every
	// real target in this repo is >= 3 chars.
	makeRe = regexp.MustCompile(`\bmake\s+([A-Za-z][A-Za-z0-9_-]{2,})`)
	// makeLineRe matches a `make <target>` invocation in command position inside a
	// fenced block: at the start of the line (after optional indentation and a
	// `$`/`>` prompt) OR right after a shell command separator (`&&`, `||`, `;`,
	// `|`), so a chained command like `cd glx && make build` is checked too.
	// Requiring a line-start/separator boundary keeps a shell comment such as
	// `# make sure to rebuild` (preceded by `# `, not a separator) from being read
	// as the target "sure". FindAllStringSubmatch returns every match on the line.
	makeLineRe = regexp.MustCompile(`(?:^|[;&|])\s*(?:[$>]\s+)?make\s+([A-Za-z][A-Za-z0-9_-]{2,})`)
	// importRe matches a github.com import/module path, captured in group 1. The
	// host is anchored on its left by `(?:^|[^A-Za-z0-9.-])` — start-of-line or a
	// non-hostname character — so a longer hostname can never match on its
	// `github.com/...` suffix: neither a prefix label (`notgithub.com/...`) nor a
	// subdomain (`evil.github.com/...`) matches, since both put a hostname char
	// immediately before `github`. (`https://github.com/...` still matches: the
	// preceding `/` is a non-hostname char, and the `://` web-URL guard in
	// importFindings sorts those out.) The trailing class excludes '@' and
	// backslash so version suffixes and regex-escaped doc strings terminate
	// cleanly.
	importRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9.-])(github\.com/[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+)`)
)

// scanFile runs every check over one memory file's content and returns the
// findings. memPath is the repo-relative path of the file (forward slashes),
// used both to label findings and to resolve paths relative to the file's own
// directory (so a reference in go-glx/CLAUDE.md may be written relative to
// go-glx/ as well as to the repo root).
func scanFile(memPath, content string, r repo) []finding {
	var out []finding
	memDir := slashDir(memPath)

	lines := strings.Split(content, "\n")
	inFence := false
	for idx, line := range lines {
		lineNo := idx + 1
		if isFenceDelimiter(line) {
			inFence = !inFence

			continue
		}

		if inFence {
			// Fenced code blocks are shell/Go examples: check `make` invocations
			// in command position, but do not treat their contents as path claims
			// (too many code fragments look path-shaped).
			out = append(out, makeTargetFindings(memPath, lineNo, line, makeLineRe, r)...)

			continue
		}

		// Outside fences, only inline code spans carry path/make claims. Scanning
		// spans rather than raw prose is what keeps "make sure" and ordinary
		// slashes in sentences from being mistaken for references.
		for _, span := range inlineSpans(line) {
			out = append(out, makeTargetFindings(memPath, lineNo, span, makeRe, r)...)
			out = append(out, pathFindings(memPath, memDir, lineNo, span, r)...)
		}
	}

	// Import paths are matched anywhere (the regex is specific enough that
	// backtick/fence context is irrelevant) so a module rename is caught even
	// when the path is quoted in prose. Reuse the already-split lines.
	out = append(out, importFindings(memPath, lines, r)...)

	return out
}

// makeTargetFindings flags `make <target>` references (matched by re) whose
// target is not declared in the Makefile.
func makeTargetFindings(file string, line int, text string, re *regexp.Regexp, r repo) []finding {
	var out []finding
	for _, m := range re.FindAllStringSubmatch(text, -1) {
		target := m[1]
		if r.makeTargets[target] {
			continue
		}
		out = append(out, finding{
			File:     file,
			Line:     line,
			Kind:     kindMakeTarget,
			Severity: severityMajor,
			Token:    "make " + target,
			Message:  fmt.Sprintf("references `make %s`, which is not a target in the Makefile", target),
		})
	}

	return out
}

// pathFindings flags an inline-code token that names a concrete repo file or
// directory which does not exist (relative to the repo root or the memory
// file's own directory).
func pathFindings(file, memDir string, line int, span string, r repo) []finding {
	candidate, isFile, isDir := classifyPathClaim(span)
	if !isFile && !isDir {
		return nil
	}
	if pathExists(candidate, memDir, isDir, r) {
		return nil
	}

	noun := "file"
	if isDir {
		noun = "directory"
	}
	checkedAgainst := "the repo root"
	if memDir != "" {
		checkedAgainst = "the repo root and " + memDir + "/"
	}
	// Report (and match the allowlist on) the reference with surrounding
	// whitespace/quotes stripped — the same normalization classifyPathClaim
	// applied — so `"go-glx/types.go"` is suppressed by the natural allowlist
	// entry `token: go-glx/types.go`, not only by a quote-wrapped form that an
	// exact-match allowlist would otherwise demand. The trailing slash on a
	// directory claim is kept, since it is shown verbatim and signals the kind.
	token := strings.Trim(strings.TrimSpace(span), `"'`)

	return []finding{{
		File:     file,
		Line:     line,
		Kind:     kindPath,
		Severity: severityMajor,
		Token:    token,
		Message: fmt.Sprintf(
			"references %s `%s`, which does not exist (checked relative to %s)",
			noun, token, checkedAgainst),
	}}
}

// importFindings flags any github.com path under the module's own org whose
// prefix no longer matches the go.mod module line — the signal that go.mod was
// renamed but the memory file was not updated.
func importFindings(file string, lines []string, r repo) []finding {
	var out []finding
	for idx, line := range lines {
		for _, loc := range importRe.FindAllStringSubmatchIndex(line, -1) {
			start := loc[2] // group 1: the github.com path, sans any boundary char
			tok := strings.TrimRight(line[start:loc[3]], "./-")
			if !strings.HasPrefix(tok, r.orgPrefix) {
				continue // third-party (or different-org) import: not our concern
			}
			// A URL to a same-org repo is a link, not a Go import — leave it
			// alone. This covers not just `https://github.com/genealogix/...` but
			// scheme forms where the match is not immediately preceded by `://`,
			// e.g. `ssh://git@github.com/genealogix/...` (preceded by `git@`).
			if withinSchemeToken(line, start) {
				continue
			}
			// A bare `host/org/repo` slug (two slashes) is a sibling-repo
			// reference, not a subpackage import of this module. Only a path with
			// a subpackage segment (>= three slashes, e.g. `…/glx/go-glx`) is an
			// import claim whose prefix must track go.mod.
			if strings.Count(tok, "/") < 3 {
				continue
			}
			// A GitHub *web* link to a same-org repo (e.g.
			// `github.com/genealogix/glx-website/tree/main`) also has >= three
			// slashes but is not a Go import. Such links are not always written
			// with a scheme, so withinSchemeToken alone misses the bare form;
			// recognize them structurally by the route verb (`tree`, `blob`, …)
			// that follows the repo segment. A Go subpackage is never named after
			// one of these GitHub routes.
			if isGitHubWebPath(tok, r.orgPrefix) {
				continue
			}
			if tok == r.modulePath || strings.HasPrefix(tok, r.modulePath+"/") {
				continue
			}
			out = append(out, finding{
				File:     file,
				Line:     idx + 1,
				Kind:     kindImportPath,
				Severity: severityMajor,
				Token:    tok,
				Message:  fmt.Sprintf("import path `%s` does not match the go.mod module `%s`", tok, r.modulePath),
			})
		}
	}

	return out
}

// gitHubWebRoutes are the path segments that mark a `github.com/<org>/<repo>/…`
// token as a link into the GitHub web UI rather than a Go import path: in an
// import the segment after the repo is a package directory, whereas in a web
// link it is one of these route verbs. The set is GitHub's (stable) route
// vocabulary, not an open-ended per-repo allowlist.
var gitHubWebRoutes = map[string]bool{
	"tree": true, "blob": true, "raw": true, "blame": true,
	"commit": true, "commits": true, "compare": true, "actions": true,
	"pull": true, "pulls": true, "issues": true, "discussions": true,
	"releases": true, "tags": true, "branches": true, "wiki": true,
	"security": true, "archive": true, "milestone": true, "milestones": true,
}

// isGitHubWebPath reports whether tok is a github.com web link (under orgPrefix)
// whose segment immediately after the repo is a known GitHub route verb.
func isGitHubWebPath(tok, orgPrefix string) bool {
	rest := strings.TrimPrefix(tok, orgPrefix) // "<repo>/<route>/…"
	segs := strings.Split(rest, "/")

	return len(segs) >= 2 && gitHubWebRoutes[segs[1]]
}

// withinSchemeToken reports whether the github.com match at index start is the
// host of a `scheme://…` URL — i.e. a `://` precedes it with only optional
// userinfo (`user[:pass]@`, no `/`) in between. This recognizes the host of
// `https://github.com/…` and `ssh://git@github.com/…` (where the match is
// preceded by `git@`, not `://`), while NOT skipping a stale import merely glued
// after an unrelated link such as a markdown "](https://ex.com/x)" immediately
// followed by an import in a code span — there the nearest `://` belongs to the
// other URL and is followed by that URL's own `/path`, so it does not govern
// this match. Tying the scheme to the host (rather than to any `://` anywhere in
// the whitespace token) is what avoids that false negative.
func withinSchemeToken(line string, start int) bool {
	prefix := line[:start]
	if i := strings.LastIndexAny(prefix, " \t"); i >= 0 {
		prefix = prefix[i+1:]
	}
	scheme := strings.LastIndex(prefix, "://")
	if scheme < 0 {
		return false
	}
	// Only userinfo may sit between `://` and the host; a `/` there means the
	// preceding URL already had its own path and this match is a separate token.
	return !strings.Contains(prefix[scheme+len("://"):], "/")
}

// classifyPathClaim decides whether an inline-code token asserts a concrete repo
// path, and if so whether it is a file or directory claim. The intent is
// encoded syntactically: a trailing slash means "directory", and a slash plus a
// recognized file extension means "file". Everything else — bare basenames,
// extensionless slugs (`actions/checkout`, `os/exec`), routes (`/cli`), globs,
// commands, URLs, and import paths — is deliberately left unverified to keep the
// gate free of false positives.
func classifyPathClaim(tok string) (candidate string, isFile, isDir bool) {
	tok = strings.TrimSpace(tok)
	// Strip surrounding quotes so a quoted reference (`"go-glx/types.go"`, a
	// stray-apostrophe `'src/main.go`) normalizes to the bare path before any
	// guard or extension check runs.
	tok = strings.Trim(tok, `"'`)
	if tok == "" {
		return "", false, false
	}
	if strings.ContainsAny(tok, pathMeta) || strings.ContainsAny(tok, " \t") {
		return "", false, false
	}
	if strings.HasPrefix(tok, "-") || strings.HasPrefix(tok, "/") || strings.HasPrefix(tok, "~") {
		return "", false, false // flag, absolute path / route, or home-relative path
	}
	// Any ':' already covers a URL's `://`, so one ContainsAny suffices.
	if strings.ContainsAny(tok, "@:=") {
		return "", false, false // URL, ref@version, key:value, assignment
	}
	if strings.HasPrefix(tok, "github.com/") {
		return "", false, false // import path, handled by importFindings
	}
	// Reject any token carrying a `..` path segment. memcheck only ever needs to
	// confirm paths *inside* the checkout; a parent traversal resolved via
	// filepath.Join(root, …) could stat a file outside the repository, so treat
	// it as a non-claim rather than follow it.
	if slices.Contains(strings.Split(tok, "/"), "..") {
		return "", false, false
	}

	// Directory claim: an explicit trailing slash on a multi-segment path. The
	// multi-segment requirement (an internal slash, not just the trailing one)
	// drops single-word branch prefixes like `claude/` and top-level dirs like
	// `go-glx/` — the former is not a path at all, the latter never drifts — and
	// keeps the references that actually rot, e.g. `specification/5-standard-
	// vocabularies/` or `docs/cli/`.
	if strings.HasSuffix(tok, "/") {
		c := strings.TrimRight(tok, "/")
		if !strings.Contains(c, "/") {
			return "", false, false
		}

		return c, false, true
	}

	// File claim: must contain a slash and end in a recognized extension.
	if !strings.Contains(tok, "/") {
		return "", false, false
	}
	if ext := pathExt(tok); ext == "" || !knownExts[ext] {
		return "", false, false
	}

	return tok, true, false
}

// pathExt returns the lower-cased extension of the final path segment, or ""
// when the segment has none or is a dotfile (e.g. ".github").
func pathExt(tok string) string {
	base := tok
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	j := strings.LastIndex(base, ".")
	if j <= 0 { // no dot, or leading dot (dotfile) → no extension
		return ""
	}

	return strings.ToLower(base[j+1:])
}

// pathExists resolves candidate against the repo root first, then against the
// memory file's own directory, so both repo-root-relative and file-relative
// reference styles are accepted. It is satisfied only when the on-disk kind
// matches the claim: wantDir requires a directory, otherwise a regular file.
func pathExists(candidate, memDir string, wantDir bool, r repo) bool {
	if found, isDir := r.exists(candidate); found && isDir == wantDir {
		return true
	}
	if memDir != "" {
		if found, isDir := r.exists(path.Join(memDir, candidate)); found && isDir == wantDir {
			return true
		}
	}

	return false
}

// inlineSpans returns the contents of every single-backtick code span on a line.
func inlineSpans(line string) []string {
	matches := inlineSpanRe.FindAllStringSubmatch(line, -1)
	spans := make([]string, 0, len(matches))
	for _, m := range matches {
		spans = append(spans, m[1])
	}

	return spans
}

// isFenceDelimiter reports whether a line opens or closes a fenced code block.
func isFenceDelimiter(line string) bool {
	t := strings.TrimSpace(line)

	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// slashDir returns the directory portion of a forward-slash path, or "" when the
// path has no directory (a top-level file).
func slashDir(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}

	return ""
}
