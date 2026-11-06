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

func TestMergePersonsPreservesAliasMappingKeys(t *testing.T) {
	for _, fixture := range []struct{ name, yaml string }{
		{"entity", "persons:\n  person-keep:\n    notes: &entity_id person-drop\n  *entity_id:\n    notes: dropped\n"},
		{"collection", "person_properties:\n  custom_label:\n    label: &collection persons\n    value_type: string\n*collection:\n  person-keep:\n    notes: keep\n  person-drop:\n    notes: dropped\n"},
		{"property_changed_anchor", "persons:\n  person-keep:\n    notes: &property occupation\n  person-drop:\n    properties:\n      # transferred property key provenance\n      *property: 'Farmer' # property value provenance\n"},
		{"property_unchanged_anchor", "persons:\n  person-other:\n    notes: &property occupation\n  person-keep:\n    notes: keep\n  person-drop:\n    properties:\n      # transferred property key provenance\n      *property: 'Farmer' # property value provenance\n"},
	} {
		for _, directory := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/directory=%t", fixture.name, directory), func(t *testing.T) {
				dir := t.TempDir()
				path := filepath.Join(dir, "archive.glx")
				original := []byte(fixture.yaml + "  person-stable:\n    notes: &stable 'stable'\n  person-stable-user:\n    notes: *stable # untouched alias provenance\n")
				require.NoError(t, os.WriteFile(path, original, 0o600))
				target := path
				if directory {
					target = dir
				}
				files, err := readMergeFiles(target, directory)
				require.NoError(t, err)
				expected, err := loadMergeFiles(target, files, directory)
				require.NoError(t, err)
				require.Empty(t, expected.Validate().Errors)
				_, err = glxlib.MergePersons(expected, "person-keep", "person-drop", glxlib.MergePersonsOptions{})
				require.NoError(t, err)
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, true))
				dryRun, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, original, dryRun, "preview must preserve every byte")
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				files, err = readMergeFiles(target, directory)
				require.NoError(t, err)
				actual, err := loadMergeFiles(target, files, directory)
				require.NoError(t, err)
				require.Empty(t, actual.Validate().Errors)
				expectedYAML, err := yaml.Marshal(expected)
				require.NoError(t, err)
				actualYAML, err := yaml.Marshal(actual)
				require.NoError(t, err)
				assert.YAMLEq(t, string(expectedYAML), string(actualYAML), "runner must install the SDK's complete semantic outcome")
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				for _, retained := range []string{"notes: &stable 'stable'", "notes: *stable # untouched alias provenance"} {
					assert.Contains(t, string(data), retained)
				}
				if strings.HasPrefix(fixture.name, "property_") {
					assert.Contains(t, string(data), "# transferred property key provenance")
					assert.Contains(t, string(data), "'Farmer' # property value provenance")
				}
			})
		}
	}
}
