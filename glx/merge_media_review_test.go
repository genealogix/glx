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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMergeReview_IdenticalMediaRetainsSourceBinary(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	sourceMedia := &glxlib.Media{URI: "media/files/photo.jpg", Title: "Shared", Properties: map[string]any{}, Notes: []string{}}
	destMedia := &glxlib.Media{URI: sourceMedia.URI, Title: sourceMedia.Title}
	src := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": sourceMedia}}
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": destMedia}}
	require.NoError(t, os.MkdirAll(filepath.Join(source, glxlib.MediaFilesDir), dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(source, sourceMedia.URI), []byte("source photo"), filePermissions))
	result := mergeArchivesInMemory(merged, src)
	require.Empty(t, result.Conflicts)
	require.Equal(t, 1, result.IdenticalEntities)
	require.Same(t, destMedia, merged.Media["shared"], "destination metadata must still win")
	plan, err := planSourceMediaCopies(source, dest, src)
	require.NoError(t, err)
	plan = adoptIdenticalMedia(plan, merged, result.IdenticalMedia)

	copied, skipped, rollback, err := executeMediaCopiesWithRollback(plan, dest, merged)
	require.NoError(t, err)
	assert.Equal(t, 1, copied)
	assert.Zero(t, skipped)
	data, err := os.ReadFile(filepath.Join(dest, destMedia.URI))
	require.NoError(t, err)
	assert.Equal(t, "source photo", string(data))
	rollback()
	assert.NoFileExists(t, filepath.Join(dest, destMedia.URI))
}

func TestMergeReview_ReusedIdenticalMediaIsRevalidated(t *testing.T) {
	for _, mutation := range []string{"destination-changed", "destination-deleted", "source-changed"} {
		t.Run(mutation, func(t *testing.T) {
			source, dest := t.TempDir(), t.TempDir()
			src := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": {URI: "media/files/photo.jpg"}, "fresh": {URI: "media/files/new.jpg"}}}
			merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": {URI: "media/files/photo.jpg"}}}
			for _, root := range []string{source, dest} {
				require.NoError(t, os.MkdirAll(filepath.Join(root, glxlib.MediaFilesDir), dirPermissions))
				require.NoError(t, os.WriteFile(filepath.Join(root, glxlib.MediaFilesDir, "photo.jpg"), []byte("expected"), filePermissions))
			}
			require.NoError(t, os.WriteFile(filepath.Join(source, glxlib.MediaFilesDir, "new.jpg"), []byte("new"), filePermissions))
			result := mergeArchivesInMemory(merged, src)
			plan, err := planSourceMediaCopies(source, dest, src)
			require.NoError(t, err)
			plan = adoptIdenticalMedia(plan, merged, result.IdenticalMedia)
			require.NotSame(t, src.Media["shared"], merged.Media["shared"])
			target := filepath.Join(dest, glxlib.MediaFilesDir, "photo.jpg")
			if mutation == "source-changed" {
				target = filepath.Join(source, glxlib.MediaFilesDir, "photo.jpg")
			}
			if mutation == "destination-deleted" {
				require.NoError(t, os.Remove(target))
			} else {
				require.NoError(t, os.WriteFile(target, []byte("changed!"), filePermissions))
			}
			_, _, err = publishMergedMultiFileArchive(dest, nil, merged, plan)
			require.Error(t, err)
			assert.NoFileExists(t, filepath.Join(dest, glxlib.MediaFilesDir, "new.jpg"), "a failed reuse check rolls back new copies")
			assert.NoFileExists(t, filepath.Join(dest, "media/shared.glx"))
		})
	}
}

// The destination already holds a different binary under the name its
// identical metadata names. The planner would place the source's copy under a
// new name; that must neither turn the identical entity into a URI conflict
// nor copy an orphan binary nobody references.
func TestMergeReview_IdenticalMediaWithDifferentDestinationBinary(t *testing.T) {
	source, dest := t.TempDir(), t.TempDir()
	src := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": {URI: "media/files/photo.jpg", Title: "Shared"}}}
	destMedia := &glxlib.Media{URI: "media/files/photo.jpg", Title: "Shared"}
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": destMedia}}
	for root, content := range map[string]string{source: "source photo", dest: "destination photo"} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, glxlib.MediaFilesDir), dirPermissions))
		require.NoError(t, os.WriteFile(filepath.Join(root, glxlib.MediaFilesDir, "photo.jpg"), []byte(content), filePermissions))
	}
	result := mergeArchivesInMemory(merged, src)
	require.Empty(t, result.Conflicts)
	require.Equal(t, []string{"shared"}, result.IdenticalMedia)
	plan, err := planSourceMediaCopies(source, dest, src)
	require.NoError(t, err)
	require.Len(t, plan, 1, "the planner still places the source binary under a new name")
	plan = adoptIdenticalMedia(plan, merged, result.IdenticalMedia)
	assert.Empty(t, plan, "the destination's own binary serves its identical metadata")
	assert.Equal(t, "media/files/photo.jpg", merged.Media["shared"].URI)
	data, err := os.ReadFile(filepath.Join(dest, glxlib.MediaFilesDir, "photo.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "destination photo", string(data))
}

// Two source entities share one binary: one is identical to the destination,
// the other is new. The entry still serves the new one.
func TestMergeReview_AdoptKeepsEntryForNewSharer(t *testing.T) {
	destMedia := &glxlib.Media{URI: "media/files/photo.jpg"}
	newMedia := &glxlib.Media{URI: "media/files/photo-1.jpg"}
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"shared": destMedia, "fresh": newMedia}}
	plan := []plannedMediaCopy{{TargetName: "photo-1.jpg", SrcMedia: map[string]*glxlib.Media{
		"shared": {URI: "media/files/photo-1.jpg"},
		"fresh":  newMedia,
	}}}
	plan = adoptIdenticalMedia(plan, merged, []string{"shared"})
	require.Len(t, plan, 1)
	assert.Equal(t, map[string]*glxlib.Media{"fresh": newMedia}, plan[0].SrcMedia)
	assert.True(t, plannedMediaIsReferenced(&plan[0], merged))
}
