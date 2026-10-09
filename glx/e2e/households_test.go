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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// censusHeadOnly is a head-only schedule: Josiah Thompson is named, Robert
// (an infant in 1850 here) is counted as a tick mark, and the page records two
// neighboring households (#1332, #180).
const censusHeadOnly = `census:
  year: 1850
  location:
    place_id: place-springfield
  household:
    members:
      - name: "Josiah Thompson"
        sex: male
      - name: "Robert Thompson"
        person_id: person-robert-thompson
        named: false
    tally:
      - sex: male
        age_from: 30
        age_to: 39
        count: 1
      - sex: male
        age_to: 4
        count: 1
    neighbors:
      - name: "Henry Jeffries"
        position: previous_household
        page: "12"
      - name: "James M Clark"
        position: next_household
`

func TestCensusAddHeadOnly_ThenHouseholds(t *testing.T) {
	archive := copyExample(t, "basic-family")
	tpl := filepath.Join(t.TempDir(), "census-1850.yaml")
	require.NoError(t, os.WriteFile(tpl, []byte(censusHeadOnly), 0o644))

	add := runGLX(t, archive, "census", "add", "--from", tpl)
	require.Equal(t, 0, add.exitCode, add.stderr)
	assertArchiveValid(t, archive)

	text := runGLX(t, archive, "households", "person-robert-thompson", "--neighbors")
	require.Equal(t, 0, text.exitCode, text.stderr)
	assert.Contains(t, text.stdout, "Households for Robert Thompson (person-robert-thompson):")
	assert.Contains(t, text.stdout, "Josiah Thompson (person-josiah-thompson) — head")
	assert.Contains(t, text.stdout, "Robert Thompson (person-robert-thompson) — household member, counted, not named")
	assert.Contains(t, text.stdout, "1 male under 5")
	assert.Contains(t, text.stdout, "Henry Jeffries — previous household, page 12")

	js := runGLX(t, t.TempDir(), "households", "--place", "place-springfield", "--year", "1850", "--format", "json", "--archive", archive)
	require.Equal(t, 0, js.exitCode, js.stderr)
	var out struct {
		Households []struct {
			Members []struct {
				PersonID string `json:"person_id"`
				Named    bool   `json:"named"`
			} `json:"members"`
			Tally []struct {
				Count int `json:"count"`
			} `json:"tally"`
		} `json:"households"`
	}
	require.NoError(t, json.Unmarshal([]byte(js.stdout), &out))
	require.Len(t, out.Households, 1)
	require.Len(t, out.Households[0].Members, 2)
	assert.False(t, out.Households[0].Members[1].Named)
	assert.Len(t, out.Households[0].Tally, 2)
}

func TestHouseholds_ArgumentErrors(t *testing.T) {
	archive := copyExample(t, "basic-family")

	assertExitWithStderr(t, runGLX(t, archive, "households"), "give a person, or --place")
	assertExitWithStderr(t, runGLX(t, archive, "households", "a", "b"), "accepts at most 1 arg")
	assertExitWithStderr(t, runGLX(t, archive, "households", "person-robert-thompson", "--format", "xml"), "unknown output format")
	assertExitWithStderr(t, runGLX(t, archive, "households", "--year", "x", "--place", "place-springfield"), "invalid argument")
}
