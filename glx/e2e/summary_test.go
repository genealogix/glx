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
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A marriage added exactly as `glx add event --help` documents it — an event
// with bride and groom participants and no relationship entity — must appear
// in summary's Family block, not just in timeline (#1275). This is the issue's
// own reproduction, run end to end.
func TestSummary_EventOnlyMarriageShowsSpouse(t *testing.T) {
	root := t.TempDir()
	res := runGLX(t, root, "init", "spousetest")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	archive := filepath.Join(root, "spousetest")

	for _, args := range [][]string{
		{"add", "person", "--id", "person-a", "--given", "Anna", "--surname", "Test", "--sex", "female"},
		{"add", "person", "--id", "person-b", "--given", "Bernd", "--surname", "Test", "--sex", "male"},
		{
			"add", "event", "--id", "event-m", "--type", "marriage", "--date", "1800-01-01",
			"--participant", "person-a:bride", "--participant", "person-b:groom",
		},
	} {
		res := runGLX(t, archive, args...)
		require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	}

	timeline := runGLX(t, archive, "timeline", "person-b")
	require.Equal(t, 0, timeline.exitCode, timeline.stdout+timeline.stderr)
	require.Contains(t, timeline.stdout, "Marriage", "timeline should show the marriage event")

	summary := runGLX(t, archive, "summary", "person-b")
	require.Equal(t, 0, summary.exitCode, summary.stdout+summary.stderr)
	assert.Contains(t, summary.stdout, "Anna Test", "summary should name the spouse timeline already shows")
	assert.Contains(t, summary.stdout, "1800-01-01")
	assert.NotContains(t, summary.stdout, "Spouse:           (none)")

	// Adding the relationship for the same couple must not list Anna twice.
	res = runGLX(t, archive, "add", "relationship", "--type", "marriage",
		"--participant", "person-a:spouse", "--participant", "person-b:spouse")
	require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)

	both := runGLX(t, archive, "summary", "person-b")
	require.Equal(t, 0, both.exitCode, both.stdout+both.stderr)
	assert.Equal(t, 1, strings.Count(both.stdout, "Spouse:"), "spouse should be listed once, not once per model")
	assert.Contains(t, both.stdout, "Anna Test")
}
