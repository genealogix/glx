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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Research-log leads (#660, #183) surface through query --subject and a
// Research section in summary, and are validated against the lead_statuses
// vocabulary and the archive's persons.
func TestResearchLeads_QuerySummaryValidate(t *testing.T) {
	archive := copyExample(t, "complete-family")
	assertArchiveValid(t, archive)

	q := runGLX(t, archive, "query", "research_logs", "--subject", "person-mary-brown-1852")
	require.Equal(t, 0, q.exitCode, q.stdout+q.stderr)
	assert.Contains(t, q.stdout, "research-log-mary-brown-parents")
	assert.NotContains(t, q.stdout, "research-log-john-smith-birth")
	assert.Contains(t, q.stdout, "(2 leads)")
	assert.Contains(t, q.stdout, "1 research logs found")

	q = runGLX(t, archive, "query", "research_logs", "--status", "complete")
	require.Equal(t, 0, q.exitCode, q.stdout+q.stderr)
	assert.Contains(t, q.stdout, "research-log-john-smith-birth")
	assert.NotContains(t, q.stdout, "research-log-mary-brown-parents")

	s := runGLX(t, archive, "summary", "person-mary-brown-1852")
	require.Equal(t, 0, s.exitCode, s.stdout+s.stderr)
	assert.Contains(t, s.stdout, "── Research")
	assert.Contains(t, s.stdout, "1 planned search not yet performed")
	assert.Contains(t, s.stdout, "1 active, 1 eliminated")

	bad := []byte(`research_logs:
  research-log-bad-lead:
    leads:
      - description: "Unknown candidate"
        persons: [person-does-not-exist]
        status: probable
`)
	require.NoError(t, os.WriteFile(filepath.Join(archive, "research_logs", "research-log-bad-lead.glx"), bad, 0o600))
	v := runGLX(t, archive, "validate", ".")
	assert.NotEqual(t, 0, v.exitCode)
	out := v.stdout + v.stderr
	assert.Contains(t, out, "person-does-not-exist")
	assert.Contains(t, out, "lead_statuses: probable")
}

// A self-contained single-file archive validates standard research-log
// vocabulary values (log status, search result, lead status) just as a
// directory archive does.
func TestResearchLeads_SelfContainedSingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "archive.glx")
	doc := []byte(`persons:
  person-mary:
    properties:
      name: "Mary Green"
  person-john:
    properties:
      name: "John Green"
confidence_levels:
  medium:
    label: "Medium"
research_logs:
  research-log-mary-parents:
    subject:
      person: person-mary
    status: in_progress
    searches:
      - collection: "1850 census"
        result: not_searched
    leads:
      - description: "John Green"
        persons: [person-john]
        status: active
        confidence: medium
`)
	require.NoError(t, os.WriteFile(file, doc, 0o600))
	v := runGLX(t, dir, "validate", "archive.glx")
	assert.Equal(t, 0, v.exitCode, v.stdout+v.stderr)
	assert.Contains(t, v.stdout, "Archive is valid")
}
