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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The research-log workflow from issue #1336: create a log, append searches
// one at a time (the common mid-session case), and scope a study. Every step
// prints only the ID on stdout and the archive stays valid throughout.
// complete-family is used because it ships the research-log and study
// vocabularies that `glx validate` checks against.
func TestAddResearch_LogSearchStudyWorkflow(t *testing.T) {
	archive := copyExample(t, "complete-family")
	before := snapshotTree(t, archive)

	logRes := runGLX(t, archive, "add", "research-log", "--id", "rl-death",
		"--subject-person", "person-john-smith-1850",
		"--objective", "Verify the reported death", "--status", "in_progress")
	require.Equal(t, 0, logRes.exitCode, logRes.stderr)
	assert.Equal(t, "rl-death\n", logRes.stdout)
	assert.Contains(t, logRes.stderr, "Adding research-log: rl-death")

	diff := diffTrees(before, snapshotTree(t, archive))
	assert.Equal(t, []string{"research_logs/rl-death.glx"}, diff.created)
	assert.Empty(t, diff.changed)

	afterLog := snapshotTree(t, archive)
	searchRes := runGLX(t, archive, "add", "search", "--log", "rl-death",
		"--source", "source-census-1851", "--query", "John Smith d. 1900-1910",
		"--result", "not_found", "--date", "2026-09-17", "--note", "Checked twice, nothing")
	require.Equal(t, 0, searchRes.exitCode, searchRes.stderr)
	assert.Equal(t, "rl-death\n", searchRes.stdout, "add search echoes the log it appended to")
	assert.Contains(t, searchRes.stderr, "Adding search to research-log: rl-death (search 1")

	second := runGLX(t, archive, "add", "search", "--log", "rl-death",
		"--collection", "Leeds burials", "--result", "found", "--citation", "citation-census-john")
	require.Equal(t, 0, second.exitCode, second.stderr)
	assert.Contains(t, second.stderr, "(search 2")

	diff = diffTrees(afterLog, snapshotTree(t, archive))
	assert.Equal(t, []string{"research_logs/rl-death.glx"}, diff.changed, "only the log's own file is touched")
	assert.Empty(t, diff.created)

	data, err := os.ReadFile(filepath.Join(archive, "research_logs", "rl-death.glx"))
	require.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, "query: John Smith d. 1900-1910")
	assert.Contains(t, body, "result: not_found")
	assert.Contains(t, body, "collection: Leeds burials")
	assert.Equal(t, 2, strings.Count(body, "citation-census-john"), "the search citation is rolled up into the log's citations")

	studyRes := runGLX(t, archive, "add", "study", "--title", "Parents of John Smith",
		"--type", "brick_wall", "--status", "active", "--place", "place-leeds",
		"--date-range", "FROM 1750 TO 1822")
	require.Equal(t, 0, studyRes.exitCode, studyRes.stderr)
	assert.Equal(t, "study-parents-of-john-smith\n", studyRes.stdout)
	assert.FileExists(t, filepath.Join(archive, "studies", "study-parents-of-john-smith.glx"))

	assertArchiveValid(t, archive)
}

// Appending to a hand-written log keeps the rest of the file exactly as it
// was: the diff is the new search lines and nothing else.
func TestAddResearch_SearchPreservesExistingLayout(t *testing.T) {
	archive := copyExample(t, "complete-family")
	path := filepath.Join(archive, "research_logs", "research-log-john-smith-birth.glx")
	original, err := os.ReadFile(path)
	require.NoError(t, err)

	res := runGLX(t, archive, "add", "search", "--log", "research-log-john-smith-birth",
		"--repository", "repository-tna", "--query", "John Smith 1861", "--result", "not_found")
	require.Equal(t, 0, res.exitCode, res.stderr)

	updated, err := os.ReadFile(path)
	require.NoError(t, err)
	normalize := func(b []byte) string { return strings.ReplaceAll(string(b), "\r\n", "\n") }
	want := strings.Replace(normalize(original), "    citations:\n",
		"      - repository: repository-tna\n        query: John Smith 1861\n        result: not_found\n    citations:\n", 1)
	assert.Equal(t, want, normalize(updated))
	assertArchiveValid(t, archive)
}

func TestAddResearch_InvalidInputFailsWithoutWriting(t *testing.T) {
	archive := copyExample(t, "complete-family")
	before := snapshotTree(t, archive)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"bad log status", []string{"research-log", "--title", "X", "--status", "bogus"}, `research_log_status_types="bogus"`},
		{"two subjects", []string{"research-log", "--subject-person", "person-john-smith-1850", "--subject-place", "place-leeds"}, "mutually exclusive"},
		{"unknown log", []string{"search", "--log", "rl-nope", "--query", "x"}, `research_logs "rl-nope"`},
		{"missing log flag", []string{"search", "--query", "x"}, `required flag(s) "log" not set`},
		{"empty search", []string{"search", "--log", "research-log-john-smith-birth", "--result", "not_found"}, "at least one of --source"},
		{"bad search result", []string{"search", "--log", "research-log-john-smith-birth", "--query", "x", "--result", "bogus"}, `search_result_types="bogus"`},
		{"dangling search source", []string{"search", "--log", "research-log-john-smith-birth", "--source", "source-nope"}, `sources "source-nope"`},
		{"missing study title", []string{"study", "--type", "brick_wall"}, `required flag(s) "title" not set`},
		{"bad study type", []string{"study", "--title", "X", "--type", "bogus"}, `study_types="bogus"`},
		{"bad study status", []string{"study", "--title", "X", "--status", "bogus"}, `study_statuses="bogus"`},
		{"dangling study place", []string{"study", "--title", "X", "--place", "place-nope"}, `places "place-nope"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertExitWithStderr(t, runGLX(t, archive, append([]string{"add"}, tc.args...)...), tc.want)
		})
	}
	assertTreeUnchanged(t, before, archive)
}

func TestAddResearch_SearchDryRunWritesNothing(t *testing.T) {
	archive := copyExample(t, "complete-family")
	before := snapshotTree(t, archive)

	res := runGLX(t, archive, "add", "search", "--log", "research-log-john-smith-birth",
		"--query", "x", "--result", "not_searched", "--dry-run")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Equal(t, "research-log-john-smith-birth\n", res.stdout)
	assert.Contains(t, res.stderr, "dry run")
	assertTreeUnchanged(t, before, archive)
}

// The smaller gaps from #1336: source properties, person external IDs, and
// event properties, each written as the vocabulary-defined property.
func TestAddResearch_SourcePersonEventPropertyFlags(t *testing.T) {
	archive := copyExample(t, "basic-family")
	capture := func(args ...string) string {
		t.Helper()
		res := runGLX(t, archive, append([]string{"add"}, args...)...)
		require.Equal(t, 0, res.exitCode, res.stderr)

		return strings.TrimSpace(res.stdout)
	}

	source := capture("source", "--title", "Find a Grave", "--type", "database",
		"--url", "https://www.findagrave.com", "--publication-info", "Find a Grave, 1995-",
		"--call-number", "FG-1")
	person := capture("person", "--given", "Lewis", "--surname", "Little",
		"--external-id", "wikitree:Little-20642")
	event := capture("event", "--type", "death", "--date", "1826", "--principal", person,
		"--property", "cause=fever", "--property", "description=Died intestate")

	read := func(dir, id string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(archive, dir, id+".glx"))
		require.NoError(t, err)

		return string(data)
	}
	assert.Contains(t, read("sources", source), "url: https://www.findagrave.com")
	assert.Contains(t, read("sources", source), "call_number: FG-1")
	assert.Contains(t, read("sources", source), "publication_info: Find a Grave, 1995-")
	assert.Contains(t, read("persons", person), "value: Little-20642")
	assert.Contains(t, read("persons", person), "type: wikitree")
	assert.Contains(t, read("events", event), "cause: fever")
	assert.Contains(t, read("events", event), "description: Died intestate")
	assertArchiveValid(t, archive)

	bad := runGLX(t, archive, "add", "event", "--type", "death", "--property", "bogus=1")
	assertExitWithStderr(t, bad, `event_properties "bogus"`)
}
