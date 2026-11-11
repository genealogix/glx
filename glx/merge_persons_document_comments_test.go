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
	"bytes"
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

func TestMergePersonsPreservesDocumentCommentsWithoutAliases(t *testing.T) {
	const original = `# document preamble first
# document preamble second

# persons collection heading
persons:
  # first entity heading
  person-keep:
    notes: keep
  person-drop:
    notes: drop
  person-other:
    notes: unchanged

# document trailer first
# document trailer second
`
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "archive.glx")
			require.NoError(t, os.WriteFile(path, []byte(original), 0o600))
			target := path
			if directory {
				target = dir
			}
			files, err := readMergeFiles(target, directory)
			require.NoError(t, err)
			expected, err := loadMergeFiles(target, files, directory)
			require.NoError(t, err)
			unaffected := *expected.Persons["person-other"]
			_, err = glxlib.MergePersons(expected, "person-keep", "person-drop", glxlib.MergePersonsOptions{})
			require.NoError(t, err)
			streams, _, _ := TestIOStreams()
			require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, true))
			dryRun, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, original, string(dryRun))
			require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
			saved, err := os.ReadFile(path)
			require.NoError(t, err)
			previous := -1
			for _, fragment := range []string{"# document preamble first", "# document preamble second", "# persons collection heading", "persons:", "# first entity heading", "person-keep:", "person-other:", "# document trailer first", "# document trailer second"} {
				assert.Equal(t, 1, strings.Count(string(saved), fragment), "each source comment and node must occur once")
				position := bytes.Index(saved, []byte(fragment))
				assert.Greater(t, position, previous, "source ordering must survive for %s", fragment)
				previous = position
			}
			files, err = readMergeFiles(target, directory)
			require.NoError(t, err)
			actual, err := loadMergeFiles(target, files, directory)
			require.NoError(t, err)
			require.Empty(t, actual.Validate().Errors)
			assert.Equal(t, &unaffected, actual.Persons["person-other"])
			expectedYAML, err := yaml.Marshal(expected)
			require.NoError(t, err)
			actualYAML, err := yaml.Marshal(actual)
			require.NoError(t, err)
			assert.YAMLEq(t, string(expectedYAML), string(actualYAML))
		})
	}
}
