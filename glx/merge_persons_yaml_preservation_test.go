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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMergePersonsPreservesTransferredKeysAndCensusNeighbors(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "single"
		if directory {
			name = "directory"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			require.NoError(t, os.WriteFile(path, []byte(`persons:
  person-keep:
    properties:
      sex: male
    notes: keep
  person-drop:
    properties:
      # transferred occupation provenance
      occupation: 'blacksmith' # occupation spelling
    notes: drop
events:
  census:
    type: census
    participants:
      - person: person-keep
        role: principal
    neighbors:
      # preceding household provenance
      - name: 'Henry Jeffries' # neighbor name spelling
        person: person-drop # neighbor identity
        position: previous_household
        page: '012' # original page spelling
      # following household provenance
      - name: 'James Clark'
        person: person-keep # unchanged identity
        position: next_household
`), 0o600))
			target := path
			if directory {
				target = dir
			}
			streams, _, _ := TestIOStreams()
			require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			for _, retained := range []string{
				"# transferred occupation provenance",
				"occupation: 'blacksmith' # occupation spelling",
				"# preceding household provenance",
				"name: 'Henry Jeffries' # neighbor name spelling",
				"person: person-keep # neighbor identity",
				"page: '012' # original page spelling",
				"# following household provenance",
				"name: 'James Clark'",
				"person: person-keep # unchanged identity",
			} {
				assert.Contains(t, string(data), retained)
			}
			files, err := readMergeFiles(target, directory)
			require.NoError(t, err)
			archive, err := loadMergeFiles(target, files, directory)
			require.NoError(t, err)
			assert.Equal(t, "person-keep", archive.Events["census"].Neighbors[0].Person)
			assert.Len(t, archive.Events["census"].Neighbors, 2)
		})
	}
}
