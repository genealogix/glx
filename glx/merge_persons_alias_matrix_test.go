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

func TestMergePersonsAliasWriterMatrix(t *testing.T) {
	const keep = "  person-keep:\n    notes: keep\n"
	const drop = "  person-drop:\n    notes: drop\n"
	const stable = `  person-stable:
    notes: &stable 'stable'
  person-stable-user:
    notes: *stable # untouched alias
`
	const candidate = `  candidate:
    type: possibly_same_person
    participants:
      - person: person-keep
      - person: person-drop
`
	const snapshot = `research_logs:
  log:
    title: Original snapshot
    properties:
      snapshot: *snapshot # snapshot usage
`
	tests := []struct {
		name, persons, extra string
		dropFile             string
		occupation           bool
	}{
		{name: "removed-collection-key", persons: keep + drop + stable, extra: "&snapshot relationships:\n" + candidate + snapshot},
		{name: "removed-collection", persons: keep + drop + stable, extra: "relationships: &snapshot\n" + candidate + snapshot},
		{name: "changed-collection", persons: keep + drop + stable, extra: snapshot},
		{name: "forward-scalar-transfer", persons: keep + "  person-other:\n    properties:\n      occupation: &job farmer\n" + "  person-drop:\n    properties:\n      occupation: *job # occupation usage\n" + stable, occupation: true},
		{name: "backward-scalar-transfer", persons: "  person-other:\n    properties:\n      occupation: &job farmer\n" + keep + "  person-drop:\n    properties:\n      occupation: *job # occupation usage\n" + stable, occupation: true},
		{name: "nested-forward-transfer", persons: keep + "  person-other:\n    properties:\n      occupation: {value: &job farmer}\n" + "  person-drop:\n    properties:\n      occupation: {value: *job} # occupation usage\n" + stable, occupation: true},
		{name: "retained-collection", persons: keep + drop + stable, extra: "relationships: &snapshot\n" + candidate + "  other:\n    type: parent_child\n    participants:\n      - person: person-keep\n      - person: person-stable\n" + snapshot},
		{name: "nested-alias-transfer", persons: keep + "  person-other:\n    properties:\n      occupation: &job farmer\n  person-middle:\n    properties:\n      occupation: &container {value: *job}\n  person-drop:\n    properties:\n      occupation: *container # occupation usage\n" + stable, occupation: true},
		{name: "rebound-anchor-transfer", persons: keep + "  person-other:\n    properties:\n      occupation: &job farmer\n  person-drop:\n    properties:\n      occupation: *job # occupation usage\n  person-later:\n    properties:\n      occupation: &job carpenter\n  person-later-user:\n    properties:\n      occupation: *job # rebound usage\n" + stable, occupation: true},
		{name: "cross-file-transfer", persons: keep + stable, dropFile: "persons:\n  person-other:\n    properties:\n      occupation: &job farmer\n  person-drop:\n    properties:\n      occupation: *job # occupation usage\n", occupation: true},
	}
	for _, tc := range tests {
		for _, directory := range []bool{false, true} {
			if tc.dropFile != "" && !directory {
				continue
			}
			layout := "single"
			if directory {
				layout = "directory"
			}
			t.Run(tc.name+"/"+layout, func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "archive.glx")
				target := path
				header := "persons:\n"
				if tc.name == "changed-collection" {
					header = "persons: &snapshot\n"
				}
				original := map[string][]byte{"": []byte(header + tc.persons + tc.extra)}
				if directory {
					target = dir
					original = map[string][]byte{"archive.glx": original[""]}
					if tc.dropFile != "" {
						original["drop.glx"] = []byte(tc.dropFile)
					}
				}
				for name, data := range original {
					file := path
					if directory {
						file = filepath.Join(dir, name)
					}
					require.NoError(t, os.WriteFile(file, data, 0o600))
				}
				before, err := loadMergeFiles(target, original, directory)
				require.NoError(t, err, "fixture must be valid before merge")
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, true))
				dry, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				assert.Equal(t, original, dry, "preview must not write")
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				files, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				after, err := loadMergeFiles(target, files, directory)
				require.NoError(t, err)
				assert.NotContains(t, after.Persons, "person-drop")
				if tc.name == "rebound-anchor-transfer" {
					assert.Equal(t, before.Persons["person-later-user"], after.Persons["person-later-user"], "unchanged aliases preserve their original anchor binding")
				}
				assert.Equal(t, before.ResearchLogs, after.ResearchLogs, "unrelated snapshot aliases retain pre-merge values")
				assert.Equal(t, glxlib.NoteList{"stable"}, after.Persons["person-stable-user"].Notes)
				var combined strings.Builder
				for _, data := range files {
					combined.Write(data)
				}
				assert.Contains(t, combined.String(), "notes: *stable # untouched alias")
				if tc.occupation {
					assert.Equal(t, before.Persons["person-drop"].Properties["occupation"], after.Persons["person-keep"].Properties["occupation"])
					assert.Contains(t, combined.String(), "# occupation usage")
				} else {
					assert.Contains(t, combined.String(), "# snapshot usage")
				}
			})
		}
	}
}
