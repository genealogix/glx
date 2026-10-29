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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	mergeReviewPeople        = "persons:\n  person-keep:\n    notes: keep\n  person-drop:\n    notes: drop\n"
	mergeReviewUnknownSource = "sources:\n  source-other:\n    title: Unrelated\n    obsolete: retained # original archival annotation\n"
)

func TestMergePersonsPreservesUnrelatedStructuralErrors(t *testing.T) {
	for _, directory := range []bool{false, true} {
		for _, shared := range []bool{false, true} {
			if !directory && !shared {
				continue
			}
			t.Run(fmt.Sprintf("directory=%t/shared=%t", directory, shared), func(t *testing.T) {
				root := t.TempDir()
				file := filepath.Join(root, "archive.glx")
				data := mergeReviewPeople
				if shared {
					data += mergeReviewUnknownSource
				}
				require.NoError(t, os.WriteFile(file, []byte(data), 0o600))
				if !shared {
					require.NoError(t, os.WriteFile(filepath.Join(root, "unrelated.glx"), []byte(mergeReviewUnknownSource), 0o600))
				}
				target := file
				if directory {
					target = root
				}
				before, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, false))
				preview, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				assert.Equal(t, before, preview)
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				after, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				archive, err := loadMergeFiles(target, after, directory)
				require.NoError(t, err)
				assert.NotContains(t, archive.Persons, "person-drop")
				if shared {
					assert.Contains(t, string(after[mergeReviewFileKey(directory)]), "obsolete: retained # original archival annotation")
				} else {
					assert.Equal(t, before["unrelated.glx"], after["unrelated.glx"])
				}
			})
		}
	}
}

func mergeReviewFileKey(directory bool) string {
	if directory {
		return "archive.glx"
	}

	return ""
}

func TestMergePersonsRefusesUnknownFieldsOnEditedEntities(t *testing.T) {
	for _, id := range []string{"person-keep", "person-drop"} {
		t.Run(id, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "archive.glx")
			original := strings.Replace(mergeReviewPeople, id+":\n", id+":\n    obsolete: preserve this researcher data\n", 1)
			require.NoError(t, os.WriteFile(file, []byte(original), 0o600))
			streams, _, _ := TestIOStreams()
			err := mergePersonsWithIO(streams, strings.NewReader(""), false, file, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true)
			require.ErrorContains(t, err, "cannot safely edit")
			require.ErrorContains(t, err, id)
			after, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, original, string(after), "unknown data may not be silently dropped by typed editing")
		})
	}
}

func TestMergePersonsMaterializesRootInheritance(t *testing.T) {
	fixtures := []struct {
		name, yaml    string
		preserveAlias bool
	}{
		{"external_source", `research_logs:
  snapshot:
    properties:
      original: &seed
        persons:
          person-keep: {notes: keep}
          person-drop: {notes: drop}
      copy: *seed # untouched snapshot consumer
# inherited archival record
<<: *seed
`, true},
		{"inline_source_consumers", `# inherited archival record
<<: &seed
  persons: &people
    person-keep: {notes: keep}
    person-drop: {notes: drop}
research_logs:
  snapshot:
    properties:
      original: *seed
      people: *people
`, false},
		{"first_source_wins", `# inherited archival record
<<:
  - persons:
      person-keep: {notes: first}
      person-drop: {notes: drop}
  - persons:
      person-keep: {notes: second}
      person-drop: {notes: ignored}
`, false},
		{"explicit_alias_key_wins", `research_logs:
  keys:
    properties:
      collection: &collection persons
# inherited archival record
<<:
  persons:
    ignored: {notes: not inherited}
*collection:
  person-keep: {notes: explicit}
  person-drop: {notes: drop}
`, false},
		{"unrelated_root_source", `# inherited archival record
<<: &source_data
  sources:
    source-record:
      title: Unrelated source
persons:
  person-keep: {notes: keep}
  person-drop: {notes: drop}
research_logs:
  snapshot:
    properties:
      original: *source_data # untouched source alias
`, false},
		{"entity_fields", `persons:
  person-keep:
    # inherited archival record
    <<: &person_fields
      notes: 'keep' # inherited note provenance
  person-drop:
    notes: drop
research_logs:
  snapshot:
    properties:
      original: *person_fields
`, false},
	}
	for _, fixture := range fixtures {
		for _, directory := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/directory=%t", fixture.name, directory), func(t *testing.T) {
				root := t.TempDir()
				file := filepath.Join(root, "archive.glx")
				require.NoError(t, os.WriteFile(file, []byte(fixture.yaml), 0o600))
				target := file
				if directory {
					target = root
				}
				before, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				expected, err := loadMergeFiles(target, before, directory)
				require.NoError(t, err)
				require.Empty(t, expected.Validate().Errors)
				_, err = glxlib.MergePersons(expected, "person-keep", "person-drop", glxlib.MergePersonsOptions{})
				require.NoError(t, err)
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, false))
				preview, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				assert.Equal(t, before, preview)
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				after, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				actual, err := loadMergeFiles(target, after, directory)
				require.NoError(t, err)
				require.Empty(t, actual.Validate().Errors)
				expectedYAML, err := yaml.Marshal(expected)
				require.NoError(t, err)
				actualYAML, err := yaml.Marshal(actual)
				require.NoError(t, err)
				assert.YAMLEq(t, string(expectedYAML), string(actualYAML), "unrelated anchor consumers must retain their original data")
				result := string(after[mergeReviewFileKey(directory)])
				assert.Contains(t, result, "# inherited archival record")
				if fixture.preserveAlias {
					assert.Contains(t, result, "copy: *seed # untouched snapshot consumer")
				}
				if fixture.name == "unrelated_root_source" {
					assert.Contains(t, result, "<<: &source_data")
					assert.Contains(t, result, "original: *source_data # untouched source alias")
				}
				if fixture.name == "entity_fields" {
					assert.Contains(t, result, "# inherited note provenance")
				}
			})
		}
	}
}
