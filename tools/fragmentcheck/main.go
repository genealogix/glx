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

// Command fragmentcheck asserts that every changie changelog fragment carries
// an issue/PR reference in its top-level `custom.Issue` field — the field
// `changeFormat` renders into CHANGELOG.md (#858, implements #321).
//
// `changie new` makes the field required, so this is the backstop for
// fragments hand-written without it. It runs from
// scripts/check-changelog-fragments.sh, after changie's own parse/render pass.
//
// The check reads the fragment with a YAML parser rather than pattern-matching
// the file text. Both properties that matter fall out of that for free: an
// `Issue:`-looking line inside the `body: |-` block scalar is body text, not a
// field, and so cannot satisfy the gate; and every spelling YAML permits for
// the real field — block map, inline flow map (`custom: {Issue: "#123"}`),
// quoted or unquoted, any key order — is accepted.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

// issueRefPattern is the reference shape CONTRIBUTING requires: at least one
// `#NNN`. It deliberately matches anywhere in the value so multi-ref strings
// ("#41, #775") and prefixed ones ("PR #456") pass.
var issueRefPattern = regexp.MustCompile(`#[0-9]+`)

// fragment is the subset of changie's fragment schema this check reads.
// Custom is map[string]any rather than map[string]string so a non-string
// value (`Issue: 123`) yields our own diagnostic instead of a decode error.
type fragment struct {
	Custom map[string]any `yaml:"custom"`
}

// defaultFragmentDir is where changie writes fragments (unreleasedDir in
// .changie.yaml), relative to the repository root.
const defaultFragmentDir = ".changes/unreleased"

func main() {
	dir := defaultFragmentDir
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	os.Exit(run(dir, os.Stdout))
}

// run validates every fragment in dir, writing findings to out, and returns
// the process exit code: 0 when every fragment carries a reference, 1
// otherwise.
func run(dir string, out io.Writer) int {
	files, err := fragmentFiles(dir)
	if err != nil {
		fmt.Fprintf(out, "ERROR: %v\n", err)

		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(out, "No unreleased changelog fragments to validate.")

		return 0
	}

	failed := 0
	for _, f := range files {
		if problem := checkFile(f); problem != "" {
			// GitHub Actions annotation: surfaces inline on the PR diff.
			fmt.Fprintf(out, "::error file=%s::%s\n", filepath.ToSlash(f), problem)
			failed++
		}
	}
	if failed > 0 {
		return 1
	}

	fmt.Fprintf(out, "All %d changelog fragment(s) valid.\n", len(files))

	return 0
}

// fragmentFiles returns the fragment paths in dir, sorted so output order is
// deterministic across platforms.
func fragmentFiles(dir string) ([]string, error) {
	var out []string
	for _, ext := range []string{"*.yaml", "*.yml"} {
		matches, err := filepath.Glob(filepath.Join(dir, ext))
		if err != nil {
			return nil, fmt.Errorf("globbing %s: %w", dir, err)
		}
		out = append(out, matches...)
	}
	sort.Strings(out)

	return out, nil
}

// checkFile returns a human-readable problem description, or "" when the
// fragment carries a usable issue reference.
func checkFile(path string) string {
	// #nosec G304 -- the path is a glob hit under the caller-named fragment
	// directory, and reading those files is precisely this tool's job. The
	// directory does come from argv, but this is a developer CLI run by the
	// person who supplied it: there is no privilege boundary to cross, and
	// nothing is done with the contents beyond reporting on them.
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("cannot read changelog fragment: %v", err)
	}

	var frag fragment
	if err := yaml.Unmarshal(data, &frag); err != nil {
		return fmt.Sprintf("changelog fragment is not valid YAML: %v", err)
	}

	raw, ok := frag.Custom["Issue"]
	if !ok {
		return "changelog fragment is missing an issue/PR reference " +
			"(expected an 'Issue' field with #NNN under the top-level 'custom' key)"
	}

	// A YAML-quoted "#123" decodes to a string; an unquoted one is a comment
	// and decodes to nil, which is worth calling out specifically.
	if raw == nil {
		return "changelog fragment has an empty 'custom.Issue' " +
			"(an unquoted #NNN is a YAML comment — quote it, e.g. Issue: \"#123\")"
	}

	value := fmt.Sprint(raw)
	if !issueRefPattern.MatchString(value) {
		return fmt.Sprintf("changelog fragment's 'custom.Issue' (%q) carries no #NNN issue/PR reference", value)
	}

	return ""
}
