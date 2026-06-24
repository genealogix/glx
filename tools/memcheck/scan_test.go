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
	"testing"
)

// testRepo builds a repo whose existence check is backed by an in-memory set,
// so scan tests are pure (no filesystem). modulePath/orgPrefix mirror this repo.
func testRepo(existing, targets []string) repo {
	ex := make(map[string]bool, len(existing))
	for _, e := range existing {
		ex[e] = true
	}
	tg := make(map[string]bool, len(targets))
	for _, t := range targets {
		tg[t] = true
	}

	return repo{
		makeTargets: tg,
		modulePath:  "github.com/genealogix/glx",
		orgPrefix:   "github.com/genealogix/",
		exists:      func(p string) bool { return ex[p] },
	}
}

// TestClassifyPathClaim pins the discriminator that separates a genuine repo
// path reference from the many look-alikes (GitHub actions, stdlib imports,
// routes, branch prefixes, globs, basenames). Getting this wrong is the only way
// the gate produces false positives, so the table is deliberately exhaustive.
func TestClassifyPathClaim(t *testing.T) {
	tests := []struct {
		tok       string
		candidate string
		isFile    bool
		isDir     bool
	}{
		// Genuine file claims: slash + recognized extension.
		{"testdata/gedcom/shakespeare.ged", "testdata/gedcom/shakespeare.ged", true, false},
		{"go-glx/types.go", "go-glx/types.go", true, false},
		{".github/workflows/lint-pr-title.yml", ".github/workflows/lint-pr-title.yml", true, false},
		{"specification/schema/v1/glx-file.schema.json", "specification/schema/v1/glx-file.schema.json", true, false},

		// Surrounding quotes are stripped before classification.
		{`"go-glx/types.go"`, "go-glx/types.go", true, false},
		{"'src/main.go", "src/main.go", true, false},

		// Genuine directory claims: trailing slash on a multi-segment path.
		{"docs/cli/", "docs/cli", false, true},
		{"specification/5-standard-vocabularies/", "specification/5-standard-vocabularies", false, true},
		{"testdata/gedcom/", "testdata/gedcom", false, true},

		// Single-segment dirs / branch prefixes: not verified (never drift, or
		// not a path at all).
		{"claude/", "", false, false},
		{"go-glx/", "", false, false},
		{"glx/", "", false, false},

		// Slash but no file extension: GitHub actions, stdlib imports, slugs.
		{"actions/checkout", "", false, false},
		{"actions/setup-go", "", false, false},
		{"os/exec", "", false, false},
		{"ossf/scorecard-action", "", false, false},
		{"sigstore/cosign-installer", "", false, false},
		{"bin/glx", "", false, false},
		{"feat/short-description", "", false, false},

		// Leading slash: absolute paths, routes, slash-commands.
		{"/cli", "", false, false},
		{"/compact-changelog", "", false, false},

		// Home-relative paths are not repo-relative.
		{"~/.claude/settings.json", "", false, false},
		{"~/.config/glx/config.yaml", "", false, false},

		// Single-segment basenames: ambiguous, deliberately unverified.
		{"release.yml", "", false, false},
		{"person-john-smith.glx", "", false, false},
		{"gedcom_test.go", "", false, false},

		// Bare extension, globs, URLs, refs, key:value, assignments.
		{".md", "", false, false},
		{".glx", "", false, false},
		{"glx/cmd_*.go", "", false, false},
		{".github/workflows/*.yml", "", false, false},
		{"https://github.blog/x", "", false, false},
		{"sigstore/cosign-installer@v4.1.2", "", false, false},
		{`yaml:"field,omitempty"`, "", false, false},
		{"VERSION=1.2.3", "", false, false},

		// Import paths are handled separately, never as path claims.
		{"github.com/genealogix/glx/go-glx", "", false, false},
	}

	for _, tt := range tests {
		c, isFile, isDir := classifyPathClaim(tt.tok)
		if c != tt.candidate || isFile != tt.isFile || isDir != tt.isDir {
			t.Errorf("classifyPathClaim(%q) = (%q,%v,%v), want (%q,%v,%v)",
				tt.tok, c, isFile, isDir, tt.candidate, tt.isFile, tt.isDir)
		}
	}
}

// TestScanFileNoFalsePositives feeds a realistic memory file modeled on the
// repo's own CLAUDE.md (every observed look-alike token) and asserts the scan is
// silent once the genuinely-referenced files are present.
func TestScanFileNoFalsePositives(t *testing.T) {
	content := "# Guide\n" +
		"Run `make test` and `make check-code-drift`.\n" +
		"Import path: `glxlib \"github.com/genealogix/glx/go-glx\"`.\n" +
		"Branch naming uses prefixes, NOT `claude/`.\n" +
		"Key file: `go-glx/types.go`. Pin `actions/checkout` and use `os/exec`.\n" +
		"See `release.yml` and the `/cli` sidebar; example `person-john-smith.glx`.\n" +
		"```bash\n" +
		"make build           # make sure this is not parsed as a target\n" +
		"feat/short-description\n" +
		"```\n"
	r := testRepo([]string{"go-glx/types.go"}, []string{"test", "check-code-drift", "build"})

	if got := scanFile("CLAUDE.md", content, r); len(got) != 0 {
		t.Fatalf("expected no findings, got %d: %+v", len(got), got)
	}
}

// TestScanFilePathDrift covers a missing file and a missing directory, with line
// numbers, plus the dual-base (repo-root and file-relative) resolution.
func TestScanFilePathDrift(t *testing.T) {
	content := "line1\n" +
		"stale `testdata/gedcom/shakespeare.ged` here\n" +
		"dir `docs/missing/` there\n" +
		"ok `sub/thing.go` (file-relative)\n"
	// thing.go resolves only relative to the memory file's dir (go-glx/).
	r := testRepo([]string{"go-glx/sub/thing.go"}, nil)

	got := scanFile("go-glx/CLAUDE.md", content, r)
	if len(got) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(got), got)
	}
	if got[0].Line != 2 || got[0].Kind != kindPath || got[0].Token != "testdata/gedcom/shakespeare.ged" {
		t.Errorf("finding[0] = %+v", got[0])
	}
	if got[1].Line != 3 || got[1].Kind != kindPath || got[1].Token != "docs/missing/" {
		t.Errorf("finding[1] = %+v", got[1])
	}
}

// TestScanFileRootRelativeResolution confirms a repo-root-relative reference in a
// nested memory file resolves against the root (not only the file's directory).
func TestScanFileRootRelativeResolution(t *testing.T) {
	content := "see `go-glx/serializer.go`\n"
	r := testRepo([]string{"go-glx/serializer.go"}, nil)
	if got := scanFile("specification/CLAUDE.md", content, r); len(got) != 0 {
		t.Fatalf("root-relative path should resolve, got %+v", got)
	}
}

// TestScanFileMakeTargets covers unknown targets inline and in command position,
// and confirms prose / comments / compound lines do not yield false targets.
func TestScanFileMakeTargets(t *testing.T) {
	content := "Run `make frobnicate` now.\n" + // inline, unknown -> finding
		"Please make sure to rebuild and make it work.\n" + // prose -> ignored
		"```bash\n" +
		"make build\n" + // command position, known -> ok
		"# make sure to run this first\n" + // fenced comment -> ignored
		"```\n"
	r := testRepo(nil, []string{"build"})

	got := scanFile("CLAUDE.md", content, r)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 make finding, got %d: %+v", len(got), got)
	}
	if got[0].Kind != kindMakeTarget || got[0].Token != "make frobnicate" || got[0].Line != 1 {
		t.Errorf("finding = %+v", got[0])
	}
}

// TestMakeTargetGrammar confirms the reference matcher accepts the same target
// grammar the Makefile parser does (underscores, uppercase) so a target like
// `check_schemas` isn't truncated at the underscore into a spurious `make check`.
func TestMakeTargetGrammar(t *testing.T) {
	// Underscore target present -> no finding.
	if got := scanFile("CLAUDE.md", "run `make check_schemas`\n", testRepo(nil, []string{"check_schemas"})); len(got) != 0 {
		t.Fatalf("underscore target should resolve, got %+v", got)
	}
	// Underscore target absent -> one finding carrying the full target token.
	got := scanFile("CLAUDE.md", "run `make check_schemas`\n", testRepo(nil, []string{"build"}))
	if len(got) != 1 || got[0].Token != "make check_schemas" {
		t.Fatalf("expected one finding for the full target, got %+v", got)
	}
}

// TestImportFindings covers the rename-detection contract: a same-org *import
// path* (with a subpackage segment) must be prefixed by the module, while
// third-party imports, web URLs, and bare sibling-repo slugs are left alone.
func TestImportFindings(t *testing.T) {
	clean := testRepo(nil, nil) // module github.com/genealogix/glx
	cases := []struct {
		name    string
		content string
		r       repo
		want    int
	}{
		{"matching subpath", "use `github.com/genealogix/glx/go-glx` here", clean, 0},
		{"matching module", "module `github.com/genealogix/glx`", clean, 0},
		{"third-party import", "pin `github.com/securego/gosec/v2`", clean, 0},
		{"cosign identity", "regexp '^https://github.com/genealogix/glx/.github/workflows/release.yml@x'", clean, 0},
		// Sibling-org references must NOT gate CI: a bare org/repo slug or any
		// web URL to a same-org repo is not a Go import of this module.
		{"sibling repo slug", "see `github.com/genealogix/glx-archive-westeros`", clean, 0},
		{"sibling repo url", "clone https://github.com/genealogix/homebrew-tap here", clean, 0},
		{"sibling repo url subpath", "[site](https://github.com/genealogix/glx-website/tree/main)", clean, 0},
		{
			"go.mod renamed, doc stale",
			"import `github.com/genealogix/glx/go-glx`",
			repo{modulePath: "github.com/genealogix/glx-core", orgPrefix: "github.com/genealogix/", exists: func(string) bool { return false }},
			1,
		},
	}
	for _, tt := range cases {
		got := importFindings("CLAUDE.md", tt.content, tt.r)
		if len(got) != tt.want {
			t.Errorf("%s: got %d findings, want %d: %+v", tt.name, len(got), tt.want, got)
		}
	}
}

// TestPathExt checks extension extraction, including dotfiles and dirless names.
func TestPathExt(t *testing.T) {
	cases := map[string]string{
		"a/b.go":            "go",
		"a/b.SCHEMA.JSON":   "json",
		"a/.github":         "", // dotfile segment -> no extension
		"a/Makefile":        "",
		"x.yml":             "yml",
		"noext":             "",
		"docs/cli/index.md": "md",
	}
	for in, want := range cases {
		if got := pathExt(in); got != want {
			t.Errorf("pathExt(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestFenceTrackingAcrossLines ensures an inline-looking token inside a fenced
// block is not treated as a path claim (the fenced-block-leakage hazard).
func TestFenceTrackingAcrossLines(t *testing.T) {
	content := "```\n" +
		"`testdata/inside/fence.ged`\n" + // backticks inside a fence: not a path claim
		"```\n" +
		"outside `testdata/outside.ged`\n" // real claim
	r := testRepo(nil, nil)
	got := scanFile("CLAUDE.md", content, r)
	if len(got) != 1 || got[0].Line != 4 || got[0].Token != "testdata/outside.ged" {
		t.Fatalf("fence handling wrong, got %+v", got)
	}
}
