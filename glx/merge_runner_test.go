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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeArchives_NewEntities(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Events:        map[string]*glxlib.Event{},
		Relationships: map[string]*glxlib.Relationship{},
		Places:        map[string]*glxlib.Place{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Repositories:  map[string]*glxlib.Repository{},
		Assertions:    map[string]*glxlib.Assertion{},
		Media:         map[string]*glxlib.Media{},
	}

	src := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {Type: "birth", Date: "1850"},
		},
	}

	result := mergeArchivesInMemory(dest, src)

	assert.Empty(t, result.Duplicates, "no duplicates expected")
	assert.Equal(t, 1, result.NewPersons)
	assert.Equal(t, 1, result.NewEvents)
	assert.Len(t, dest.Persons, 2)
	assert.Contains(t, dest.Persons, "person-b")
}

func TestMergeArchives_Duplicates(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Events: map[string]*glxlib.Event{},
	}

	src := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Different A"}},
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}

	result := mergeArchivesInMemory(dest, src)

	require.Len(t, result.Duplicates, 1)
	assert.Contains(t, result.Duplicates[0], "person-a")
	assert.Equal(t, 1, result.NewPersons)
}

func TestMergeArchives_EmptySource(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}

	src := &glxlib.GLXFile{}

	result := mergeArchivesInMemory(dest, src)

	assert.Empty(t, result.Duplicates)
	assert.Equal(t, 0, result.TotalNew())
	assert.Len(t, dest.Persons, 1)
}

func TestMergeArchives_DiskRoundTrip(t *testing.T) {
	// Create temp directories for source and destination archives
	destDir := t.TempDir()
	srcDir := t.TempDir()

	// Initialize dest with one person
	destSerializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})
	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := destSerializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	// Initialize src with a different person
	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := destSerializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	// Merge via CLI function
	err = mergeArchives(srcDir, destDir, false)
	require.NoError(t, err)

	// Reload and verify no duplicates
	reloaded, dupes, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)
	assert.Empty(t, dupes, "reloaded archive should have no duplicates")
	assert.Len(t, reloaded.Persons, 2, "should have both persons after merge")
	assert.Contains(t, reloaded.Persons, "person-a")
	assert.Contains(t, reloaded.Persons, "person-b")
}

func TestMergeArchives_DotDestination(t *testing.T) {
	// Create dest archive in temp dir
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	// Save original cwd, chdir into dest, merge with "."
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { os.Chdir(origDir) })

	require.NoError(t, os.Chdir(destDir))
	err = mergeArchives(srcDir, ".", false)
	require.NoError(t, err)

	// Verify merge result (use absolute destDir since cwd may have changed)
	reloaded, dupes, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)
	assert.Empty(t, dupes, "reloaded archive should have no duplicates")
	assert.Len(t, reloaded.Persons, 2, "should have both persons after merge")
	assert.Contains(t, reloaded.Persons, "person-a")
	assert.Contains(t, reloaded.Persons, "person-b")
}
