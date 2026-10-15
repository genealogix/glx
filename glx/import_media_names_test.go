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

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestReserveImportMediaDestinations_PreservesExistingFileAndReferences(t *testing.T) {
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "Photo.JPG"), []byte("old bytes"), filePermissions))
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
		"file": {URI: "media/files/photo.jpg"},
		"blob": {URI: "media/files/photo.jpg"},
	}}
	files := []glxlib.MediaFileSource{
		{MediaID: "file", SourceType: glxlib.MediaSourceFile, RelativePath: "sub/photo.jpg", TargetFilename: "photo.jpg"},
		{MediaID: "blob", SourceType: glxlib.MediaSourceBlob, TargetFilename: "photo.jpg"},
	}

	require.NoError(t, reserveImportMediaDestinations(glx, files, archiveDir))
	require.Equal(t, "photo-2.jpg", files[0].TargetFilename)
	require.Equal(t, "photo-3.jpg", files[1].TargetFilename)
	require.Equal(t, "media/files/photo-2.jpg", glx.Media["file"].URI)
	require.Equal(t, "photo.jpg", glx.Media["file"].Properties[glxlib.MediaPropertyOriginalFilename])
	require.Equal(t, "media/files/photo-3.jpg", glx.Media["blob"].URI)
	oldBytes, err := os.ReadFile(filepath.Join(filesDir, "Photo.JPG"))
	require.NoError(t, err)
	require.Equal(t, []byte("old bytes"), oldBytes)
}

func TestReserveImportMediaDestinations_UnoccupiedAndEmpty(t *testing.T) {
	archiveDir := t.TempDir()
	require.NoError(t, reserveImportMediaDestinations(nil, nil, archiveDir))
	media := &glxlib.Media{URI: "media/files/unique.jpg"}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
	files := []glxlib.MediaFileSource{{MediaID: "file", TargetFilename: "unique.jpg"}}
	require.NoError(t, reserveImportMediaDestinations(glx, files, archiveDir))
	require.Equal(t, "unique.jpg", files[0].TargetFilename)
	require.Equal(t, "media/files/unique.jpg", media.URI)
}

func TestReserveImportMediaDestinations_RejectsNonDirectoryMediaPath(t *testing.T) {
	archiveDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(archiveDir, "media"), dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(archiveDir, glxlib.MediaFilesDir), []byte("occupied"), filePermissions))
	files := []glxlib.MediaFileSource{{MediaID: "file", TargetFilename: "photo.jpg"}}
	require.Error(t, reserveImportMediaDestinations(&glxlib.GLXFile{}, files, archiveDir))
}
