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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzePlaces_CompleteFamily(t *testing.T) {
	err := analyzePlaces("../docs/examples/complete-family")
	require.NoError(t, err)
}

func TestAnalyzePlaces_EmptyArchive(t *testing.T) {
	err := analyzePlaces("../docs/examples/basic-family")
	// basic-family has no places directory, but LoadArchive should still work
	require.NoError(t, err)
}

func TestAnalyzePlaces_SingleFile(t *testing.T) {
	content := `places:
  place-paris:
    name: "Paris"
    type: city
    parent: place-france
  place-france:
    name: "France"
    type: country
events:
  event-1:
    place: place-paris
`
	dir := t.TempDir()
	path := filepath.Join(dir, "archive.glx")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	err := analyzePlaces(path)
	require.NoError(t, err)
}

func TestAnalyzePlaces_NonexistentPath(t *testing.T) {
	err := analyzePlaces(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
}

func TestBuildCanonicalPath(t *testing.T) {
	lat := 53.8
	lng := -1.5
	places := map[string]*glxlib.Place{
		"place-leeds":      {Name: "Leeds", ParentID: "place-yorkshire", Type: "city", Latitude: &lat, Longitude: &lng},
		"place-yorkshire":  {Name: "Yorkshire", ParentID: "place-england", Type: "county"},
		"place-england":    {Name: "England", Type: "country"},
	}

	path := buildCanonicalPath("place-leeds", places)
	assert.Equal(t, "Leeds, Yorkshire, England", path)

	path = buildCanonicalPath("place-england", places)
	assert.Equal(t, "England", path)
}

func TestBuildCanonicalPath_CycleProtection(t *testing.T) {
	places := map[string]*glxlib.Place{
		"a": {Name: "A", ParentID: "b"},
		"b": {Name: "B", ParentID: "a"},
	}
	path := buildCanonicalPath("a", places)
	assert.Equal(t, "A, B", path)
}

func TestBuildPlaceAnalysis_DuplicateNames(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-1": {Name: "Springfield", Type: "city", ParentID: "place-il"},
			"place-2": {Name: "Springfield", Type: "city", ParentID: "place-mo"},
			"place-il": {Name: "Illinois", Type: "state", ParentID: "place-us"},
			"place-mo": {Name: "Missouri", Type: "state", ParentID: "place-us"},
			"place-us": {Name: "USA", Type: "country"},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {PlaceID: "place-1"},
		},
	}

	a := buildPlaceAnalysis(archive)

	// Should detect "springfield" as duplicate
	assert.Contains(t, a.Duplicates, "springfield")
	assert.Len(t, a.Duplicates["springfield"], 2)
}

func TestBuildPlaceAnalysis_MissingCoords(t *testing.T) {
	lat := 40.0
	lng := -89.0
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-1": {Name: "With Coords", Type: "city", Latitude: &lat, Longitude: &lng},
			"place-2": {Name: "No Coords", Type: "city"},
		},
		Events: map[string]*glxlib.Event{},
	}

	a := buildPlaceAnalysis(archive)
	assert.Contains(t, a.MissingCoords, "place-2")
	assert.NotContains(t, a.MissingCoords, "place-1")
}

func TestBuildPlaceAnalysis_NoParent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-us":   {Name: "USA", Type: "country"},
			"place-city": {Name: "Sometown", Type: "city"}, // no parent, not a country
		},
		Events: map[string]*glxlib.Event{},
	}

	a := buildPlaceAnalysis(archive)
	assert.Contains(t, a.NoParent, "place-city")
	assert.NotContains(t, a.NoParent, "place-us") // country is OK without parent
}

func TestBuildPlaceAnalysis_DanglingParent(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-city": {Name: "Sometown", Type: "city", ParentID: "place-missing"},
			"place-us":   {Name: "USA", Type: "country"},
		},
		Events: map[string]*glxlib.Event{},
	}

	a := buildPlaceAnalysis(archive)
	assert.Contains(t, a.DanglingParent, "place-city")
	assert.Equal(t, "place-missing", a.DanglingParentIDs["place-city"])
	assert.NotContains(t, a.DanglingParent, "place-us")
}

func TestBuildPlaceAnalysis_Unreferenced(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-used":   {Name: "Used Place", Type: "country"},
			"place-unused": {Name: "Unused Place", Type: "country"},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {PlaceID: "place-used"},
		},
	}

	a := buildPlaceAnalysis(archive)
	assert.Contains(t, a.Unreferenced, "place-unused")
	assert.NotContains(t, a.Unreferenced, "place-used")
}

func TestBuildPlaceAnalysis_MissingType(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-typed":   {Name: "USA", Type: "country"},
			"place-untyped": {Name: "Somewhere"},
		},
		Events: map[string]*glxlib.Event{},
	}

	a := buildPlaceAnalysis(archive)
	assert.Contains(t, a.MissingType, "place-untyped")
	assert.NotContains(t, a.MissingType, "place-typed")
}

func TestBuildPlaceAnalysis_AssertionReferencedPlace(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-only-assertion": {Name: "Assertion Place", Type: "city"},
			"place-unreferenced":   {Name: "Nowhere", Type: "city"},
		},
		Events:     map[string]*glxlib.Event{},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-1": {Subject: glxlib.EntityRef{Place: "place-only-assertion"}},
		},
	}

	a := buildPlaceAnalysis(archive)
	assert.NotContains(t, a.Unreferenced, "place-only-assertion")
	assert.Contains(t, a.Unreferenced, "place-unreferenced")
}

func TestBuildPlaceAnalysis_ParentReferenceCounts(t *testing.T) {
	// A place referenced only as a parent should not be unreferenced
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-child":  {Name: "Child", Type: "city", ParentID: "place-parent"},
			"place-parent": {Name: "Parent", Type: "state"},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {PlaceID: "place-child"},
		},
	}

	a := buildPlaceAnalysis(archive)
	assert.NotContains(t, a.Unreferenced, "place-parent")
}

func TestBuildPlaceAnalysis_NilParentIsDangling(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-child":  {Name: "Child", Type: "city", ParentID: "place-nil"},
			"place-nil":    nil, // present in map but nil
			"place-normal": {Name: "Normal", Type: "country"},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {PlaceID: "place-child"},
		},
	}

	a := buildPlaceAnalysis(archive)

	// A nil parent should be treated as dangling
	assert.Contains(t, a.DanglingParent, "place-child")
	assert.Equal(t, "place-nil", a.DanglingParentIDs["place-child"])

	// A nil place entry is skipped entirely (not included in Unreferenced)
	assert.NotContains(t, a.Unreferenced, "place-nil")
}
