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

package e2e

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Robert Thompson has a birth, a marriage, and two children in basic-family,
// which gives every person-scoped read command something to report.
const readPerson = "person-robert-thompson"

// readCase describes one read-only command run against basic-family.
type readCase struct {
	name string
	// args runs the command from inside the archive root with the default
	// archive path.
	args []string
	// outside builds the equivalent invocation naming the archive
	// explicitly, for a cwd outside it. Most commands take --archive; the
	// rest take the path as a positional argument.
	outside func(archive string) []string
	// want is a line fragment that proves the command read the archive.
	want string
	// jsonArgs, if set, are extra flags that switch the output to JSON.
	jsonArgs []string
}

// withArchiveFlag is readCase.outside for commands with an --archive flag.
func withArchiveFlag(args ...string) func(string) []string {
	return func(archive string) []string {
		return append(append([]string{}, args...), "--archive", archive)
	}
}

// withPathArg is readCase.outside for commands taking the path positionally.
func withPathArg(cmd string) func(string) []string {
	return func(archive string) []string { return []string{cmd, archive} }
}

var readCases = []readCase{
	{"analyze", []string{"analyze"}, withArchiveFlag("analyze"), "Research Gap Analysis", []string{"--format", "json"}},
	{"analyze person", []string{"analyze", readPerson}, withArchiveFlag("analyze", readPerson), "Robert Thompson — no death date or place", nil},
	{"ancestors", []string{"ancestors", "person-alice-thompson"}, withArchiveFlag("ancestors", "person-alice-thompson"), "├── Mary Thompson", nil},
	{"descendants", []string{"descendants", readPerson}, withArchiveFlag("descendants", readPerson), "└── Robert Thompson Jr.", nil},
	{"cite all", []string{"cite"}, withArchiveFlag("cite"), "Volume 1, Page 47, Entry 312", nil},
	{"cite one", []string{"cite", "citation-robert-birth"}, withArchiveFlag("cite", "citation-robert-birth"), `"Sangamon County, Illinois - Register of Births"`, nil},
	{"cluster", []string{"cluster", readPerson}, withArchiveFlag("cluster", readPerson), "3 associate(s) found", []string{"--json"}},
	{"coverage", []string{"coverage", readPerson}, withArchiveFlag("coverage", readPerson), "[x] Birth record (via event-birth-robert)", []string{"--json"}},
	{"duplicates", []string{"duplicates"}, withArchiveFlag("duplicates"), "No potential duplicates found", []string{"--json"}},
	{"evidence", []string{"evidence", readPerson, "born_at"}, withArchiveFlag("evidence", readPerson, "born_at"), "No assertions found for born_at of Robert Thompson", []string{"--format", "json"}},
	{"migrations", []string{"migrations", readPerson}, withArchiveFlag("migrations", readPerson), "Springfield, Illinois, United States", []string{"--format", "json"}},
	{"path", []string{"path", "person-alice-thompson", "person-robert-thompson-jr"}, withArchiveFlag("path", "person-alice-thompson", "person-robert-thompson-jr"), "(2 hop(s))", []string{"--json"}},
	{"places", []string{"places"}, withPathArg("places"), "Place analysis: 3 places", nil},
	{"proof", []string{"proof", readPerson, "--question", "birth"}, withArchiveFlag("proof", readPerson, "--question", "birth"), "When and where was Robert Thompson born?", []string{"--format", "json"}},
	{"query", []string{"query", "persons"}, withArchiveFlag("query", "persons"), "4 person(s) found", nil},
	{"search", []string{"search", "Springfield"}, withArchiveFlag("search", "Springfield"), `Found 8 match(es) for "Springfield"`, []string{"--json"}},
	{"stats", []string{"stats"}, withPathArg("stats"), "Persons:       4", nil},
	{"summary", []string{"summary", readPerson}, withArchiveFlag("summary", readPerson), "Mary Thompson  (m. 1875-06-15, Springfield)", nil},
	{"timeline", []string{"timeline", readPerson}, withArchiveFlag("timeline", readPerson), "5 event(s) shown", nil},
	{"vitals", []string{"vitals", readPerson}, withArchiveFlag("vitals", readPerson), "April 12, 1850, Springfield", nil},
	{"validate", []string{"validate"}, withPathArg("validate"), "Archive is valid", nil},
}

// Every read command, run the way a user runs it, succeeds, reports what is
// in the archive, and writes nothing — from inside the archive with defaults
// and from outside with the path named.
func TestReadCommands_InsideAndOutsideArchive(t *testing.T) {
	archive := copyExample(t, "basic-family")
	before := snapshotTree(t, archive)
	outside := t.TempDir()

	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			inside := runGLX(t, archive, tc.args...)
			require.Equal(t, 0, inside.exitCode, inside.stderr)
			assert.Contains(t, inside.stdout, tc.want)

			named := runGLX(t, outside, tc.outside(archive)...)
			require.Equal(t, 0, named.exitCode, named.stderr)
			assert.Contains(t, named.stdout, tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestReadCommands_JSONOutputParses(t *testing.T) {
	archive := copyExample(t, "basic-family")

	for _, tc := range readCases {
		if tc.jsonArgs == nil {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			res := runGLX(t, archive, append(append([]string{}, tc.args...), tc.jsonArgs...)...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			// The whole of stdout must be one JSON document: a progress line or
			// banner before or after it breaks `| jq` and fails to decode here.
			var decoded any
			require.NoError(t, json.Unmarshal([]byte(res.stdout), &decoded), "stdout is not JSON:\n%s", res.stdout)
			assert.NotEmpty(t, decoded)
		})
	}
}

// Every read command fails, naming the path, when the archive is missing.
func TestReadCommands_MissingArchiveFails(t *testing.T) {
	work := t.TempDir()

	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, work, tc.outside("does-not-exist")...), "does-not-exist")
		})
	}
}

func TestReadCommands_ArgumentCountIsEnforced(t *testing.T) {
	archive := copyExample(t, "basic-family")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"ancestors"}, "accepts 1 arg(s), received 0"},
		{[]string{"descendants", "a", "b"}, "accepts 1 arg(s), received 2"},
		{[]string{"cluster"}, "accepts 1 arg(s), received 0"},
		{[]string{"coverage"}, "accepts 1 arg(s), received 0"},
		{[]string{"summary"}, "accepts 1 arg(s), received 0"},
		{[]string{"timeline"}, "accepts 1 arg(s), received 0"},
		{[]string{"vitals", "a", "b"}, "accepts 1 arg(s), received 2"},
		{[]string{"proof"}, "accepts 1 arg(s), received 0"},
		{[]string{"evidence", readPerson}, "accepts 2 arg(s), received 1"},
		{[]string{"path", readPerson}, "accepts 2 arg(s), received 1"},
		{[]string{"search"}, "accepts 1 arg(s), received 0"},
		{[]string{"query"}, "accepts 1 arg(s), received 0"},
		{[]string{"query", "bogus"}, `invalid argument "bogus" for "glx query"`},
		{[]string{"analyze", "a", "b"}, "accepts at most 1 arg(s), received 2"},
		{[]string{"duplicates", "a", "b"}, "accepts at most 1 arg(s), received 2"},
		{[]string{"cite", "a", "b"}, "accepts at most 1 arg(s), received 2"},
		{[]string{"migrations"}, "a person argument or --pattern is required"},
		{[]string{"proof", readPerson}, `required flag(s) "question" not set`},
	}
	for _, tc := range cases {
		t.Run(tc.args[0], func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, tc.args...), tc.want)
		})
	}
}

func TestReadCommands_UnknownPersonFails(t *testing.T) {
	archive := copyExample(t, "basic-family")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"ancestors", "person-nobody"}, `person "person-nobody" not found`},
		{[]string{"descendants", "person-nobody"}, `person "person-nobody" not found`},
		{[]string{"cluster", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"coverage", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"summary", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"timeline", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"vitals", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"migrations", "person-nobody"}, `no person found matching "person-nobody"`},
		{[]string{"evidence", "person-nobody", "born_at"}, `found matching "person-nobody"`},
		{[]string{"path", "person-nobody", readPerson}, `no person found matching "person-nobody"`},
		{[]string{"cite", "citation-nope"}, `citation "citation-nope" not found`},
	}
	for _, tc := range cases {
		t.Run(tc.args[0], func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, tc.args...), tc.want)
		})
	}
}

// The person-scoped commands that search by name accept a display name as
// well as an ID, which is how their help examples use them.
func TestReadCommands_PersonByName(t *testing.T) {
	archive := copyExample(t, "basic-family")

	for _, cmd := range []string{"summary", "timeline", "vitals", "coverage", "cluster", "migrations"} {
		t.Run(cmd, func(t *testing.T) {
			res := runGLX(t, archive, cmd, "Robert Thompson Jr.")

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Contains(t, res.stdout, "person-robert-thompson-jr")
		})
	}
}

// Commands that support --quiet print nothing on success.
func TestReadCommands_QuietSilencesOutput(t *testing.T) {
	archive := copyExample(t, "basic-family")

	for _, args := range [][]string{
		{"validate", "."},
		{"evidence", readPerson, "born_at"},
		{"migrations", readPerson},
		{"proof", readPerson, "--question", "birth"},
	} {
		t.Run(args[0], func(t *testing.T) {
			res := runGLX(t, archive, append(args, "--quiet")...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.Empty(t, res.stdout)
		})
	}
}

// Read commands work on a single-file archive named by its file.
func TestReadCommands_SingleFileArchive(t *testing.T) {
	archive := copyExample(t, "single-file")

	for _, args := range [][]string{
		{"stats", "archive.glx"},
		{"places", "archive.glx"},
		{"query", "persons", "--archive", "archive.glx"},
		{"analyze", "--archive", "archive.glx"},
	} {
		t.Run(args[0], func(t *testing.T) {
			res := runGLX(t, archive, args...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			assert.NotEmpty(t, res.stdout)
		})
	}
}
