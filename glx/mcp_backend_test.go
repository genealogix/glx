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
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Exercise the actual in-memory archive backend, not a mock backend or a
// subprocess binary outside Go's coverage instrumentation.
func TestMCPArchiveBackendAllTools(t *testing.T) {
	t.Setenv("GLX_CACHE", "off")
	backend, err := loadMCPArchive("../docs/examples/complete-family")
	require.NoError(t, err)
	before, err := json.Marshal(backend.archive)
	require.NoError(t, err)
	const person = "person-jane-smith-1876"
	for _, testCase := range []struct{ name, args, key string }{
		{"search", `{"query":"Jane"}`, "matches"},
		{"query", `{"entity_type":"persons","name":"Jane"}`, "entities"},
		{"get_entity", `{"entity_type":"persons","id":"` + person + `"}`, "entity"},
		{"vitals", `{"person":"` + person + `"}`, "vitals"},
		{"timeline", `{"person":"` + person + `","no_family":true}`, "entries"},
		{"summary", `{"person":"` + person + `"}`, "narrative"},
		{"ancestors", `{"person":"` + person + `","generations":2}`, "root"},
		{"descendants", `{"person":"person-john-smith-1850","generations":2}`, "root"},
		{"cite", `{"all":true}`, "citations"},
		{"analyze", `{"person":"Jane Smith","check":"gaps"}`, "issues"},
		{"stats", `{}`, "entity_counts"},
		{"places", `{}`, "missing_coordinates"},
		{"path", `{"from":"` + person + `","to":"person-john-smith-1850","accepted_only":true}`, "path"},
		{"cluster", `{"person":"` + person + `","before":1900}`, "associates"},
		{"validate", `{"report":true}`, "report"},
		{"list_entity_types", `{}`, "entity_counts"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			result, callErr := backend.CallTool(t.Context(), testCase.name, json.RawMessage(testCase.args))
			require.NoError(t, callErr)
			data, marshalErr := json.Marshal(result)
			require.NoError(t, marshalErr)
			var output map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(data, &output))
			require.Contains(t, output, testCase.key, string(data))
			switch testCase.name {
			case "query":
				var persons struct {
					Entities []struct {
						ID string `json:"id"`
					} `json:"entities"`
				}
				require.NoError(t, json.Unmarshal(data, &persons))
				require.NotEmpty(t, persons.Entities)
				require.Equal(t, person, persons.Entities[0].ID)
			case "path":
				var path pathResult
				require.NoError(t, json.Unmarshal(data, &path))
				require.Equal(t, person, path.From)
				require.Equal(t, "person-john-smith-1850", path.To)
				require.Positive(t, path.Hops)
			case "vitals", "timeline", "summary", "cluster":
				require.JSONEq(t, `"`+person+`"`, string(output["person_id"]))
			case "ancestors", "descendants":
				var tree struct {
					NodeCount int    `json:"node_count"`
					Direction string `json:"direction"`
				}
				require.NoError(t, json.Unmarshal(data, &tree))
				require.Positive(t, tree.NodeCount)
				require.Equal(t, testCase.name, tree.Direction)
			}
		})
	}
	after, err := json.Marshal(backend.archive)
	require.NoError(t, err)
	require.JSONEq(t, string(before), string(after), "tools must not modify the loaded archive")
}

func TestMCPArchiveBackendVocabularyResourceExcludesPrivateData(t *testing.T) {
	backend := &mcpArchiveBackend{archive: explorationTestArchive()}
	backend.archive.ImportMetadata = &glxlib.Metadata{SourceFile: "private-import.ged", Submitter: &glxlib.Submitter{Name: "Private Person", Email: "private@example.invalid"}}
	require.NoError(t, mergeStandardVocabularies(backend.archive))
	result, err := backend.CallTool(t.Context(), "vocabularies", json.RawMessage(`{}`))
	require.NoError(t, err)
	vocabularies, ok := result.(map[string]any)
	require.True(t, ok)
	require.Contains(t, vocabularies, "event_types")
	require.Contains(t, vocabularies, "person_properties")
	for _, key := range []string{"persons", "events", "relationships", "places", "sources", "citations", "repositories", "media", "assertions", "research_logs", "studies", "metadata"} {
		require.NotContains(t, vocabularies, key)
	}
	require.NotEmpty(t, backend.archive.Persons, "resource serialization must not clear the source archive")
}

func TestMCPArchiveBackendErrorsAndContinuedUse(t *testing.T) {
	backend := &mcpArchiveBackend{archive: explorationTestArchive()}
	for _, testCase := range []struct{ name, args string }{
		{"unknown", `{}`},
		{"vitals", `{"person":`},
		{"vitals", `{"person":"missing"}`},
		{"analyze", `{"person":"missing","check":"gaps"}`},
		{"analyze", `{"check":"unknown"}`},
		{"cluster", `{"person":"missing"}`},
		{"query", `{"entity_type":"unknown"}`},
		{"path", `{"from":"","to":"person-child"}`},
		{"path", `{"from":"person-father","to":"person-child","max_hops":-1}`},
		{"path", `{"from":"missing","to":"person-child"}`},
		{"path", `{"from":"person-child","to":"missing"}`},
		{"path", `{"from":"person-child","to":"Child Smith"}`},
		{"ancestors", `{"person":"person-child","generations":21}`},
		{"descendants", `{"person":"missing"}`},
	} {
		t.Run(testCase.name+testCase.args, func(t *testing.T) {
			_, err := backend.CallTool(t.Context(), testCase.name, json.RawMessage(testCase.args))
			require.Error(t, err)
			_, err = backend.CallTool(t.Context(), "stats", json.RawMessage(`{}`))
			require.NoError(t, err, "a rejected call must not poison the archive backend")
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := backend.CallTool(ctx, "stats", nil)
	require.ErrorIs(t, err, context.Canceled)
	_, err = (&mcpArchiveBackend{}).CallTool(t.Context(), "stats", nil)
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
	// Out-of-range offsets are valid empty pages, not slice panics.
	result, err := backend.CallTool(t.Context(), "query", json.RawMessage(`{"entity_type":"persons","offset":999}`))
	require.NoError(t, err)
	page, ok := result.(*mcpQueryResult)
	require.True(t, ok)
	require.Zero(t, page.Returned)
	require.False(t, page.HasMore)
	require.NotZero(t, page.Count)
}

func TestLoadMCPArchiveReadOnlyCacheAndSnapshot(t *testing.T) {
	t.Setenv("GLX_CACHE", "auto")
	dir := t.TempDir()
	source := filepath.Join(dir, "person.glx")
	writeMCPWatchSource(t, source, "First")
	backend, err := loadMCPArchive(dir)
	require.NoError(t, err)
	_, err = os.Stat(cachePath(dir))
	require.ErrorIs(t, err, os.ErrNotExist, "MCP must not populate a missing cache")
	require.NoError(t, writeCache(dir, backend.archive, nil))
	cachedBytes, err := os.ReadFile(cachePath(dir))
	require.NoError(t, err)
	cachedInfo, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	cached, err := loadMCPArchive(dir)
	require.NoError(t, err)
	require.Equal(t, "First", extractPersonName(cached.archive.Persons["person-one"]))
	after, err := os.ReadFile(cachePath(dir))
	require.NoError(t, err)
	afterInfo, err := os.Stat(cachePath(dir))
	require.NoError(t, err)
	require.Equal(t, cachedBytes, after)
	require.Equal(t, cachedInfo.ModTime(), afterInfo.ModTime())
	writeMCPWatchSource(t, source, "Updated Name")
	fresh, err := loadMCPArchive(dir)
	require.NoError(t, err)
	require.Equal(t, "Updated Name", extractPersonName(fresh.archive.Persons["person-one"]))
	require.Equal(t, "First", extractPersonName(cached.archive.Persons["person-one"]), "loaded backends own a stable snapshot")
	after, err = os.ReadFile(cachePath(dir))
	require.NoError(t, err)
	require.Equal(t, cachedBytes, after, "stale cache fallback must not rewrite the cache")
	single, err := loadMCPArchive(source)
	require.NoError(t, err)
	require.Equal(t, "Updated Name", extractPersonName(single.archive.Persons["person-one"]))
}

func TestLoadMCPArchiveReportsInvalidSourcesAndDuplicates(t *testing.T) {
	t.Setenv("GLX_CACHE", "off")
	dir := t.TempDir()
	_, err := loadMCPArchive(filepath.Join(dir, "missing.glx"))
	require.ErrorIs(t, err, os.ErrNotExist)
	broken := filepath.Join(dir, "broken.glx")
	require.NoError(t, os.WriteFile(broken, []byte("persons: ["), 0o600))
	_, err = loadMCPArchive(broken)
	require.Error(t, err)
	_, err = loadMCPArchive(dir)
	require.Error(t, err)
	require.NoError(t, os.Remove(broken))
	writeMCPWatchSource(t, filepath.Join(dir, "a.glx"), "First")
	writeMCPWatchSource(t, filepath.Join(dir, "b.glx"), "Second")
	backend, err := loadMCPArchive(dir)
	require.NoError(t, err)
	require.NotEmpty(t, backend.duplicates)
	result, err := backend.CallTool(t.Context(), "validate", json.RawMessage(`{}`))
	require.NoError(t, err)
	validation, ok := result.(*validationJSONResult)
	require.True(t, ok)
	require.False(t, validation.Valid)
	require.Equal(t, backend.duplicates, validation.DuplicateIDs)
}

func TestMCPJSONCommandAdapters(t *testing.T) {
	t.Setenv("GLX_CACHE", "off")
	const archive = "../docs/examples/complete-family"
	for name, run := range map[string]func() error{
		"stats":    func() error { return showStatsJSON(archive) },
		"places":   func() error { return showPlacesJSON(archive) },
		"validate": func() error { return showValidationJSON([]string{archive}, true) },
	} {
		t.Run(name, func(t *testing.T) {
			output := captureStdout(t, func() { require.NoError(t, run()) })
			require.True(t, json.Valid([]byte(output)), output)
		})
	}
	missing := filepath.Join(t.TempDir(), "missing")
	require.Error(t, showStatsJSON(missing))
	require.Error(t, showPlacesJSON(missing))
	require.Error(t, showValidationJSON([]string{missing}, false))
	require.ErrorIs(t, showValidationJSON([]string{"-"}, false), errJSONValidationScope)
	require.ErrorIs(t, showValidationJSON([]string{archive, archive}, false), errJSONValidationScope)
	require.Error(t, printArchiveAnalysisJSON(make(chan int)), "encoding failures must be returned to the caller")
	for _, watch := range []bool{false, true} {
		require.ErrorIs(t, serveMCP(t.Context(), missing, watch), os.ErrNotExist)
	}
}
