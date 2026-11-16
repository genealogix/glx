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
	plan, err := planSourceMediaCopies(source, dest, src)
	require.NoError(t, err)
	result := mergeArchivesInMemory(merged, src)
	require.Empty(t, result.Conflicts)
	require.Equal(t, 1, result.IdenticalEntities)
	require.Same(t, destMedia, merged.Media["shared"], "destination metadata must still win")

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
			plan, err := planSourceMediaCopies(source, dest, src)
			require.NoError(t, err)
			mergeArchivesInMemory(merged, src)
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
