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

func TestMergePersonsAliasValuedCollections(t *testing.T) {
	const persons = "persons:\n  person-keep: {notes: keep}\n  person-drop: {notes: drop}\n  person-other: {notes: other}\n"
	const candidate = "relationships:\n  candidate:\n    type: possibly_same_person\n    participants: [{person: person-keep}, {person: person-drop}]\n"
	cases := []struct {
		name, kind, entries, extra string
		removed                    bool
	}{
		{name: "persons", kind: "persons", entries: "person-keep: {notes: keep}\nperson-drop: {notes: drop}\nperson-other: {notes: other}\n"},
		{name: "removed-relationships", kind: "relationships", entries: "candidate:\n  type: possibly_same_person\n  participants: [{person: person-keep}, {person: person-drop}]\n", extra: persons, removed: true},
		{name: "changed-relationships", kind: "relationships", entries: "family:\n  type: parent_child\n  participants: [{person: person-drop}, {person: person-other}]\n", extra: persons},
		{name: "events", kind: "events", entries: "census:\n  type: census\n  participants: [{person: person-drop, role: principal}]\n", extra: persons},
		{name: "assertions", kind: "assertions", entries: "identity:\n  subject: {relationship: candidate}\n  sources: [source]\n  notes: Original claim\n", extra: persons + candidate + "sources:\n  source: {type: book, title: Register}\n", removed: true},
		{name: "research-logs", kind: "research_logs", entries: "log:\n  title: Research\n  subject: {person: person-drop}\n", extra: persons},
	}
	for _, tc := range cases {
		for _, aliasKey := range []bool{false, true} {
			for _, directory := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/keyalias=%t/directory=%t", tc.name, aliasKey, directory), func(t *testing.T) {
					// Source snapshots and a second alias consumer are outside the edited collection.
					source := "research_logs:\n  snapshot:\n    title: Snapshot\n    properties:\n"
					if tc.kind == "research_logs" {
						source = "sources:\n  snapshot:\n    type: book\n    title: Snapshot\n    properties:\n"
					}
					source += "      collection_name: &kind " + tc.kind + "\n      original: &collection\n        # original snapshot entry\n"
					var entries strings.Builder
					for line := range strings.SplitSeq(strings.TrimSuffix(tc.entries, "\n"), "\n") {
						entries.WriteString("        " + line + "\n")
					}
					source += entries.String()
					source += "      second_consumer: *collection # untouched snapshot alias\n"
					key := tc.kind
					if aliasKey {
						key = "*kind"
					}
					data := []byte(source + tc.extra + "# collection head provenance\n" + key + ": *collection # collection alias usage\n\n# document footer provenance\n")
					dir := t.TempDir()
					path := filepath.Join(dir, "archive.glx")
					target := path
					name := ""
					if directory {
						target = dir
						name = "archive.glx"
					}
					require.NoError(t, os.WriteFile(path, data, 0o600))
					original := map[string][]byte{name: data}
					before, err := loadMergeFiles(target, original, directory)
					require.NoError(t, err, "valid input must load")
					expected, err := loadMergeFiles(target, original, directory)
					require.NoError(t, err)
					_, err = glxlib.MergePersons(expected, "person-keep", "person-drop", glxlib.MergePersonsOptions{})
					require.NoError(t, err)
					streams, _, _ := TestIOStreams()
					require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, true, true))
					dry, err := readMergeFiles(target, directory)
					require.NoError(t, err)
					assert.Equal(t, original, dry)
					require.NoError(t, mergePersonsWithIO(streams, strings.NewReader(""), false, target, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false, true))
					files, err := readMergeFiles(target, directory)
					require.NoError(t, err)
					after, err := loadMergeFiles(target, files, directory)
					require.NoError(t, err)
					assert.NotContains(t, after.Persons, "person-drop")
					expectedYAML, err := yaml.Marshal(expected)
					require.NoError(t, err)
					actualYAML, err := yaml.Marshal(after)
					require.NoError(t, err)
					assert.YAMLEq(t, string(expectedYAML), string(actualYAML), "saved archive must match the complete independent SDK merge")
					if tc.kind == "research_logs" {
						assert.Equal(t, before.Sources, after.Sources)
					} else {
						assert.Equal(t, before.ResearchLogs, after.ResearchLogs)
					}
					saved := string(files[name])
					assert.Contains(t, saved, "second_consumer: *collection # untouched snapshot alias")
					assert.Contains(t, saved, "# original snapshot entry")
					assert.Contains(t, saved, "# document footer provenance")
					if !tc.removed {
						assert.Contains(t, saved, "# collection alias usage")
						assert.Contains(t, saved, "# collection head provenance")
					}
				})
			}
		}
	}
}
