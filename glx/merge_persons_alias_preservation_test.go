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

func TestMergePersonsPreservesAliasesToChangedAndDeletedAnchors(t *testing.T) {
	for _, directory := range []bool{false, true} {
		for _, anchored := range []string{"keep", "drop"} {
			name := "single/" + anchored
			if directory {
				name = "directory/" + anchored
			}
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "archive.glx")
				keep, drop := "keep", "drop"
				if anchored == "keep" {
					keep = "&shared ['keep']"
				} else {
					drop = "&shared ['drop']"
				}
				original := []byte("persons:\n  person-keep:\n    notes: " + keep + "\n  person-drop:\n    notes: " + drop + `
  person-third:
    # alias usage provenance
    notes: *shared # original alias provenance
  person-stable:
    notes: &stable 'stable'
  person-stable-alias:
    notes: *stable # unaffected alias provenance
`)
				require.NoError(t, os.WriteFile(path, original, 0o600))
				target := path
				if directory {
					target = dir
				}
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, true))
				afterDryRun, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, original, afterDryRun, "dry run must preserve input bytes")
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				for _, retained := range []string{"# alias usage provenance", "# original alias provenance", "notes: &stable 'stable'", "notes: *stable # unaffected alias provenance"} {
					assert.Contains(t, string(data), retained)
				}
				files, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				archive, err := loadMergeFiles(target, files, directory)
				require.NoError(t, err)
				assert.Equal(t, glxlib.NoteList{"keep", "drop"}, archive.Persons["person-keep"].Notes)
				assert.Equal(t, glxlib.NoteList{anchored}, archive.Persons["person-third"].Notes, "aliases must retain their original value rather than inherit the merged notes")
				assert.Equal(t, glxlib.NoteList{"stable"}, archive.Persons["person-stable-alias"].Notes)
				assert.NotContains(t, archive.Persons, "person-drop")
			})
		}
	}
}
