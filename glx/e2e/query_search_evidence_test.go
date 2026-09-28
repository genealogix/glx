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

// archiveWithConflictingOccupations adds two person-level assertions that
// disagree on Robert's occupation, which the examples lack.
func archiveWithConflictingOccupations(t *testing.T) string {
	t.Helper()
	archive := copyExample(t, "basic-family")
	for _, v := range []struct{ value, confidence string }{{"Clerk", "high"}, {"Farmer", "low"}} {
		res := runGLX(t, archive, "add", "assertion", "--subject-person", readPerson,
			"--property", "occupation", "--value", v.value,
			"--citation", "citation-robert-birth", "--confidence", v.confidence)
		require.Equal(t, 0, res.exitCode, res.stderr)
	}

	return archive
}

func TestEvidence_GroupsConflictingValues(t *testing.T) {
	archive := archiveWithConflictingOccupations(t)

	res := runGLX(t, archive, "evidence", readPerson, "occupation")

	require.Equal(t, 0, res.exitCode, res.stderr)
	assert.Contains(t, res.stdout, "2 reports across 2 values")
	assert.Contains(t, res.stdout, "Clerk — 1 report, best confidence: high")
	assert.Contains(t, res.stdout, "Farmer — 1 report, best confidence: low")
	assert.Contains(t, res.stdout, "Best evidence: Clerk")

	asJSON := runGLX(t, archive, "evidence", readPerson, "occupation", "--format", "json")
	require.Equal(t, 0, asJSON.exitCode, asJSON.stderr)
	var doc struct {
		TotalReports int `json:"total_reports"`
		Groups       []any
	}
	require.NoError(t, json.Unmarshal([]byte(asJSON.stdout), &doc), asJSON.stdout)
	assert.Equal(t, 2, doc.TotalReports)
	assert.Len(t, doc.Groups, 2)
}

func TestQuery_Filters(t *testing.T) {
	archive := archiveWithConflictingOccupations(t)

	cases := []struct {
		name  string
		args  []string
		want  []string
		count string
	}{
		{"born before", []string{"persons", "--born-before", "1851"}, []string{"person-robert-thompson "}, "1 person(s) found"},
		{"name and born after", []string{"persons", "--name", "thompson", "--born-after", "1870"}, []string{"person-alice-thompson", "person-robert-thompson-jr"}, "2 person(s) found"},
		{"phonetic name", []string{"persons", "--name", "Tompson", "--phonetic"}, []string{"person-mary-thompson"}, "4 person(s) found"},
		{"birthplace", []string{"persons", "--birthplace", "Springfield"}, []string{"person-mary-thompson"}, "4 person(s) found"},
		{"event type and year", []string{"events", "--type", "birth", "--after", "1860"}, []string{"event-birth-alice", "event-birth-robert-jr"}, "2 event(s) found"},
		{"assertion confidence", []string{"assertions", "--confidence", "low"}, []string{"occupation=Farmer"}, "1 assertion(s) found"},
		{"assertion citation", []string{"assertions", "--citation", "citation-robert-birth"}, []string{"occupation=Clerk", "occupation=Farmer"}, "3 assertion(s) found"},
		{"assertion subject", []string{"assertions", "--subject", readPerson}, []string{"occupation=Clerk"}, "2 assertion(s) found"},
		{"relationships by type", []string{"relationships", "--type", "marriage"}, []string{"rel-marriage"}, "1 relationship(s) found"},
		{"sources", []string{"sources"}, []string{"source-sangamon-births"}, "1 source(s) found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runGLX(t, archive, append([]string{"query"}, tc.args...)...)

			require.Equal(t, 0, res.exitCode, res.stderr)
			for _, w := range tc.want {
				assert.Contains(t, res.stdout, w)
			}
			assert.Contains(t, res.stdout, tc.count)
		})
	}
}

func TestSearch_Flags(t *testing.T) {
	archive := copyExample(t, "basic-family")

	byType := runGLX(t, archive, "search", "thompson", "--type", "persons")
	require.Equal(t, 0, byType.exitCode, byType.stderr)
	assert.Contains(t, byType.stdout, "Found 4 match(es)")
	assert.NotContains(t, byType.stdout, "Events")

	insensitive := runGLX(t, archive, "search", "THOMPSON")
	require.Equal(t, 0, insensitive.exitCode, insensitive.stderr)
	assert.Contains(t, insensitive.stdout, "person-robert-thompson")

	sensitive := runGLX(t, archive, "search", "THOMPSON", "--case-sensitive")
	require.Equal(t, 0, sensitive.exitCode, sensitive.stderr)
	assert.Contains(t, sensitive.stdout, `No matches found for "THOMPSON"`)
}
