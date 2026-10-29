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

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestProof_SearchNotesAndResultAnnotations(t *testing.T) {
	archive := copyExample(t, "basic-family")
	logs := filepath.Join(archive, "research_logs")
	require.NoError(t, os.MkdirAll(logs, 0o755))
	const log = `research_logs:
  log-probate-search:
    subject: {person: person-robert-thompson}
    searches:
      - query: Lost probate volume
        result: unavailable
        notes: [Volume not located, courthouse fire 1999]
      - query: Deed Book A
        result: requires_visit
      - query: 1830 census
        result: not_searched
`
	require.NoError(t, os.WriteFile(filepath.Join(logs, "log-probate-search.glx"), []byte(log), 0o644))
	before := snapshotTree(t, archive)

	for _, format := range []string{"text", "markdown", "json"} {
		t.Run(format, func(t *testing.T) {
			res := runGLX(t, archive, "proof", "person-robert-thompson", "--question", "identity", "--format", format)
			require.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
			if format == "json" {
				var result glxlib.ProofResult
				require.NoError(t, json.Unmarshal([]byte(res.stdout), &result))
				require.Len(t, result.Searches, 3)
				assert.Equal(t, glxlib.SearchResultUnavailable, result.Searches[0].Result)
				assert.Equal(t, "Volume not located; courthouse fire 1999", result.Searches[0].Notes)
				assert.Equal(t, glxlib.SearchResultRequiresVisit, result.Searches[1].Result)
				assert.Equal(t, glxlib.SearchResultNotSearched, result.Searches[2].Result)

				return
			}
			assert.Contains(t, res.stdout, "Lost probate volume -> unavailable (documented gap: Volume not located; courthouse fire 1999)")
			assert.Contains(t, res.stdout, "Deed Book A -> requires_visit (outstanding: on-site or by request)")
			assert.Contains(t, res.stdout, "1830 census -> not_searched (outstanding)")
		})
	}
	assert.Empty(t, diffTrees(before, snapshotTree(t, archive)), "proof must not change the archive")
}
