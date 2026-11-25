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

	glxlib "github.com/genealogix/glx/go-glx"
)

// mustGLX runs one setup command and fails the test on a non-zero exit.
func mustGLX(t *testing.T, archive string, args ...string) {
	t.Helper()
	res := runGLX(t, archive, args...)
	require.Equal(t, 0, res.exitCode, "%v\n%s%s", args, res.stdout, res.stderr)
}

// The per-candidate questions end to end: a disproven spouse is excluded, two
// namesake parents are told apart, a children question reads the negative
// searches of a log scoped to it, and an occupation names its other holder.
func TestProof_CandidateQuestions(t *testing.T) {
	archive := copyExample(t, "complete-family")
	for _, args := range [][]string{
		{"add", "person", "--id", "p-a", "--given", "Johann", "--surname", "Schepp", "--sex", "male"},
		{"add", "person", "--id", "p-a2", "--given", "Johann", "--surname", "Schepp", "--sex", "male"},
		{"add", "person", "--id", "p-w", "--given", "Catharina", "--surname", "Zoeller", "--sex", "female"},
		{"add", "person", "--id", "p-x", "--given", "Anna", "--surname", "Brueckel", "--sex", "female"},
		{"add", "person", "--id", "p-d", "--given", "Eva", "--surname", "Schepp", "--sex", "female"},
		{"add", "source", "--id", "s-kb", "--title", "Register", "--type", "church_register"},
		{"add", "citation", "--id", "c-kb", "--source", "s-kb", "--locator", "p. 1"},
		{"add", "relationship", "--id", "r1", "--type", "marriage", "--participant", "p-a:spouse", "--participant", "p-w:spouse"},
		{"add", "relationship", "--id", "r2", "--type", "marriage", "--participant", "p-a:spouse", "--participant", "p-x:spouse"},
		{"add", "assertion", "--id", "a1", "--subject-relationship", "r1", "--participant", "p-w:spouse", "--citation", "c-kb", "--confidence", "high", "--status", "proven"},
		{"add", "assertion", "--id", "a2", "--subject-relationship", "r2", "--participant", "p-x:spouse", "--citation", "c-kb", "--confidence", "high", "--status", "disproven", "--note", "not his wife"},
		{"add", "relationship", "--id", "pc-a", "--type", "parent_child", "--participant", "p-a:parent", "--participant", "p-d:child"},
		{"add", "relationship", "--id", "pc-a2", "--type", "parent_child", "--participant", "p-a2:parent", "--participant", "p-d:child"},
		{"add", "assertion", "--id", "a-pc-a", "--subject-relationship", "pc-a", "--participant", "p-d:child", "--citation", "c-kb", "--confidence", "high", "--status", "proven"},
		{"add", "assertion", "--id", "a-pc-a2", "--subject-relationship", "pc-a2", "--participant", "p-d:child", "--citation", "c-kb", "--confidence", "high", "--status", "disproven"},
		{"add", "assertion", "--id", "a-occ", "--subject-person", "p-a", "--property", "occupation", "--value", "church elder", "--date", "1762", "--citation", "c-kb", "--confidence", "high", "--status", "proven"},
		{"add", "assertion", "--id", "a-occ2", "--subject-person", "p-a2", "--property", "occupation", "--value", "church elder", "--date", "1774", "--citation", "c-kb", "--confidence", "high", "--status", "proven"},
		{"add", "research-log", "--id", "rl-kids", "--subject-person", "p-x", "--question", "issue"},
		{"add", "search", "--log", "rl-kids", "--source", "s-kb", "--query", "Baptisms 1700-1720", "--result", "not_found"},
	} {
		mustGLX(t, archive, args...)
	}
	assertArchiveValid(t, archive)
	before := snapshotTree(t, archive)

	marriage := runGLX(t, archive, "proof", "p-a", "--question", "marriage")
	require.Equal(t, 0, marriage.exitCode, marriage.stderr)
	assert.Contains(t, marriage.stdout, "= spouse: Catharina Zoeller [r1] -- PROVEN")
	assert.Contains(t, marriage.stdout, "x Anna Brueckel (r2) -- all assertions disproven")
	assert.Contains(t, marriage.stdout, "assertions: a2; citations: c-kb; not his wife")
	assert.Contains(t, marriage.stdout, "Married: Catharina Zoeller.")
	assert.NotContains(t, marriage.stdout, "Marriage record — Anna Brueckel")

	parentage := runGLX(t, archive, "proof", "p-d", "--question", "parentage", "--format", "markdown")
	require.Equal(t, 0, parentage.exitCode, parentage.stderr)
	assert.Contains(t, parentage.stdout, "child of Johann Schepp (p-a2) [pc-a2]")
	assert.Contains(t, parentage.stdout, "- Johann Schepp (p-a2) (`pc-a2`) — all assertions disproven")
	summary := runGLX(t, archive, "summary", "p-d")
	require.Equal(t, 0, summary.exitCode, summary.stderr)
	assert.Contains(t, summary.stdout, "Johann Schepp (p-a)")
	assert.Contains(t, summary.stdout, "Johann Schepp (p-a2)  (disproven parent)")

	children := runGLX(t, archive, "proof", "p-x", "--question", "children", "--format", "json")
	require.Equal(t, 0, children.exitCode, children.stderr)
	var result glxlib.ProofResult
	require.NoError(t, json.Unmarshal([]byte(children.stdout), &result))
	assert.Equal(t, glxlib.ConclusionProbable, result.Conclusion)
	require.Len(t, result.Searches, 1)
	assert.Equal(t, "rl-kids", result.Searches[0].Log)
	scoped := runGLX(t, archive, "proof", "p-x", "--question", "birth")
	require.Equal(t, 0, scoped.exitCode, scoped.stderr)
	assert.NotContains(t, scoped.stdout, "Baptisms 1700-1720")
	assert.Contains(t, scoped.stdout, "Other research on this person: 1 search in 1 research log")

	occupation := runGLX(t, archive, "proof", "p-a", "--question", "office")
	require.Equal(t, 0, occupation.exitCode, occupation.stderr)
	assert.Contains(t, occupation.stdout, "= church elder (1762) -- PROVEN")
	assert.Contains(t, occupation.stdout, "also recorded for Johann Schepp (p-a2) 1774")

	assertTreeUnchanged(t, before, archive)

	bad := runGLX(t, archive, "add", "research-log", "--id", "rl-bad", "--subject-person", "p-x", "--question", "nonsense")
	assertExitWithStderr(t, bad, `unknown research question "nonsense"`)
	assertTreeUnchanged(t, before, archive)
}
