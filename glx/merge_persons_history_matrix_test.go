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

func TestMergePersonsTemporalHistoryCommentsFollowEntries(t *testing.T) {
	for _, test := range []struct {
		name       string
		keep, drop []string
		want       []string
	}{
		{"interleaved_same_values", []string{"1800", "1820"}, []string{"1810", "1830"}, []string{"keep-1800", "drop-1810", "keep-1820", "drop-1830"}},
		{"earlier_drop", []string{"1820", "1840"}, []string{"1800", "1810"}, []string{"drop-1800", "drop-1810", "keep-1820", "keep-1840"}},
		{"later_drop", []string{"1800", "1810"}, []string{"1820", "1830"}, []string{"keep-1800", "keep-1810", "drop-1820", "drop-1830"}},
		{"deduplicated_overlap", []string{"1800", "1820"}, []string{"1800", "1810", "1830"}, []string{"keep-1800", "drop-1810", "keep-1820", "drop-1830"}},
		{"unsorted_keep", []string{"1820", "1800"}, []string{"1830", "1810"}, []string{"keep-1800", "drop-1810", "keep-1820", "drop-1830"}},
		{"no_contribution_retains_keep_order", []string{"1820", "1800"}, []string{"1800"}, []string{"keep-1820", "keep-1800"}},
		{"narrower_date_history_retains_both", []string{"BET 1800 AND 1810", "1820"}, []string{"1805", "1830"}, []string{"keep-BET 1800 AND 1810", "drop-1805", "keep-1820", "drop-1830"}},
	} {
		for _, directory := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/directory=%t", test.name, directory), func(t *testing.T) {
				var fixture strings.Builder
				fixture.WriteString("persons:\n")
				for _, person := range []struct {
					id    string
					years []string
				}{{"keep", test.keep}, {"drop", test.drop}} {
					fmt.Fprintf(&fixture, "  person-%s:\n    properties:\n      occupation:\n", person.id)
					for _, year := range person.years {
						fmt.Fprintf(&fixture, "        - value: 'Farmer' # %s-%s\n          date: '%s'\n", person.id, year, year)
					}
				}
				dir := t.TempDir()
				path := filepath.Join(dir, "archive.glx")
				require.NoError(t, os.WriteFile(path, []byte(fixture.String()), 0o600))
				target := path
				if directory {
					target = dir
				}
				streams, _, _ := TestIOStreams()
				require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				var doc yaml.Node
				require.NoError(t, yaml.Unmarshal(data, &doc))
				person := mergeMappingValue(mergeMappingValue(doc.Content[0], "persons"), "person-keep")
				history := mergeMappingValue(mergeMappingValue(person, "properties"), "occupation")
				require.Len(t, history.Content, len(test.want))
				for i, want := range test.want {
					value := mergeMappingValue(history.Content[i], "value")
					assert.Equal(t, "# "+want, value.LineComment, "entry %d date=%s", i, mergeMappingValue(history.Content[i], "date").Value)
				}
			})
		}
	}
}
