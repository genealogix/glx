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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestBuildStatsJSON(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons:      map[string]*glxlib.Person{"p": {}},
		Studies:      map[string]*glxlib.Study{"study": {}},
		ResearchLogs: map[string]*glxlib.ResearchLog{"log": {}},
		Assertions: map[string]*glxlib.Assertion{
			"a":    {Subject: glxlib.EntityRef{Person: "p"}, Confidence: "high"},
			"b":    {Subject: glxlib.EntityRef{Person: "missing"}},
			"null": nil,
		},
	}
	result, err := buildStatsJSON(archive)
	require.NoError(t, err)
	require.Len(t, result.EntityCounts, 11)
	require.Equal(t, 1, result.EntityCounts["studies"])
	require.Equal(t, 1, result.EntityCounts["research_logs"])
	require.Equal(t, map[string]int{"high": 1, "": 1}, result.Confidence)
	require.Equal(t, statsJSONCoverageRow{Covered: 1, Total: 1}, result.Coverage.Direct["persons"])
	require.Contains(t, result.Coverage.Note, "do not measure sourcing or proof")
	empty, err := buildStatsJSON(&glxlib.GLXFile{})
	require.NoError(t, err)
	require.Empty(t, empty.Confidence)
	require.Equal(t, statsJSONCoverageRow{}, empty.Coverage.Direct["persons"])
	_, err = buildStatsJSON(nil)
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
}

func TestBuildPlacesJSONDeterministic(t *testing.T) {
	archive := &glxlib.GLXFile{Places: map[string]*glxlib.Place{
		"z": {Name: "PARIS", ParentID: "missing"}, "a": {Name: "Paris"}, "null": nil,
	}}
	first, err := buildPlacesJSON(archive)
	require.NoError(t, err)
	require.Equal(t, 2, first.Total)
	require.Equal(t, []string{"a", "z"}, first.Duplicates["paris"])
	require.Equal(t, "Paris", first.DuplicateOriginals["paris"])
	require.Equal(t, []string{"z"}, first.DanglingParent)
	encoded, err := json.Marshal(first)
	require.NoError(t, err)
	for range 30 {
		next, buildErr := buildPlacesJSON(archive)
		require.NoError(t, buildErr)
		data, marshalErr := json.Marshal(next)
		require.NoError(t, marshalErr)
		require.Equal(t, string(encoded), string(data))
	}
	empty, err := buildPlacesJSON(&glxlib.GLXFile{})
	require.NoError(t, err)
	require.NotNil(t, empty.MissingCoords)
	require.NotNil(t, empty.Unreferenced)
	_, err = buildPlacesJSON(nil)
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
}

func TestBuildAnalysisJSON(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"person-a": {Properties: map[string]any{"name": "Alice"}},
		"person-b": {Properties: map[string]any{"name": "Bob"}},
	}}
	all, err := buildAnalysisJSON(archive, "", "gaps")
	require.NoError(t, err)
	require.NotEmpty(t, all.Issues)
	filtered, err := buildAnalysisJSON(archive, "person-a", "gap")
	require.NoError(t, err)
	require.NotEmpty(t, filtered.Issues)
	for _, issue := range filtered.Issues {
		require.Equal(t, "person-a", issue.Person)
	}
	named, err := buildAnalysisJSON(archive, "Alice", "gaps")
	require.NoError(t, err)
	require.Equal(t, filtered, named)
	require.Equal(t, len(filtered.Issues), filtered.Summary["gap"])
	_, err = buildAnalysisJSON(archive, "", "invalid")
	require.ErrorContains(t, err, "unknown check category")
	_, err = buildAnalysisJSON(nil, "", "gaps")
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
	empty, err := buildAnalysisJSON(&glxlib.GLXFile{}, "", "gaps")
	require.NoError(t, err)
	require.NotNil(t, empty.Issues)
}
