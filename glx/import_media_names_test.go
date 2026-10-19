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
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestAllocateImportMediaDestinations_DenseNonmatchingFamilyKeepsReuse(t *testing.T) {
	const count = 64
	archiveDir := t.TempDir()
	sourceDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	for n := 1; n <= count; n++ {
		name := "photo.jpg"
		if n > 1 {
			name = fmt.Sprintf("photo-%d.jpg", n)
		}
		require.NoError(t, os.WriteFile(filepath.Join(filesDir, name), []byte(fmt.Sprintf("existing %d", n)), filePermissions))
	}
	archive := &glxlib.GLXFile{Media: make(map[string]*glxlib.Media)}
	files := make([]glxlib.MediaFileSource, 0, count+1)
	for n := 1; n <= count+1; n++ {
		id := fmt.Sprintf("incoming-%d", n)
		name := id + ".jpg"
		content := []byte(fmt.Sprintf("new %d", n))
		if n == count+1 {
			content = []byte("existing 2")
		}
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, name), content, filePermissions))
		archive.Media[id] = &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
		files = append(files, glxlib.MediaFileSource{
			MediaID: id, SourceType: glxlib.MediaSourceFile, RelativePath: name,
			RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg",
		})
	}

	require.NoError(t, allocateImportMediaDestinations(archive, files, archiveDir, sourceDir, nil))
	for n := 1; n <= count; n++ {
		require.Equal(t, fmt.Sprintf("photo-%d.jpg", count+n), files[n-1].TargetFilename)
		require.False(t, files[n-1].ReuseExisting)
	}
	require.Equal(t, "photo-2.jpg", files[count].TargetFilename)
	require.True(t, files[count].ReuseExisting)
}

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

func TestReserveImportMediaDestinations_UsesVersionAwareOriginalName(t *testing.T) {
	const encoded = "CharlotteBront%C3%AB.jpg"
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, encoded), []byte("old bytes"), filePermissions))
	media := &glxlib.Media{URI: glxlib.MediaFilesDir + "/" + encoded}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
	files := []glxlib.MediaFileSource{{
		MediaID: "file", SourceType: glxlib.MediaSourceFile, RelativePath: "media/files/" + encoded,
		OriginalFilename: "CharlotteBrontë.jpg", TargetFilename: encoded,
	}}

	require.NoError(t, reserveImportMediaDestinations(glx, files, archiveDir))
	require.Equal(t, "CharlotteBront%C3%AB-2.jpg", files[0].TargetFilename)
	require.Equal(t, glxlib.MediaFilesDir+"/CharlotteBront%C3%AB-2.jpg", media.URI)
	require.Equal(t, "CharlotteBrontë.jpg", media.Properties[glxlib.MediaPropertyOriginalFilename])
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
	for _, occupied := range []string{"media", glxlib.MediaFilesDir} {
		t.Run(occupied, func(t *testing.T) {
			archiveDir := t.TempDir()
			occupiedPath := filepath.Join(archiveDir, occupied)
			require.NoError(t, os.MkdirAll(filepath.Dir(occupiedPath), dirPermissions))
			require.NoError(t, os.WriteFile(occupiedPath, []byte("occupied"), filePermissions))
			files := []glxlib.MediaFileSource{{MediaID: "file", TargetFilename: "photo.jpg"}}
			require.Error(t, reserveImportMediaDestinations(&glxlib.GLXFile{}, files, archiveDir))
		})
	}
}

func TestAllocateImportMediaDestinations_ReusesExactAndNumberedContent(t *testing.T) {
	archiveDir := t.TempDir()
	sourceDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "photo.jpg"), []byte("current"), filePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "Photo.JPG"), []byte("old"), filePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("current"), filePermissions))
	entries, err := os.ReadDir(filesDir)
	require.NoError(t, err)
	exactAvailable := len(entries) == 2 // default macOS volumes collapse case variants
	media := &glxlib.Media{URI: "media/files/photo.jpg"}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
	files := []glxlib.MediaFileSource{{MediaID: "file", RelativePath: "photo.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"}}
	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
	if exactAvailable {
		require.Equal(t, "photo.jpg", files[0].TargetFilename)
	} else {
		require.Equal(t, "Photo.JPG", files[0].TargetFilename)
	}
	require.True(t, files[0].ReuseExisting)
	require.Equal(t, "media/files/"+files[0].TargetFilename, media.URI)

	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("old exact"), filePermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo-2.jpg"), []byte("current"), filePermissions))
	files[0].TargetFilename = "photo.jpg"
	media.URI = "media/files/photo.jpg"
	files[0].ReuseExisting = false
	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
	require.Equal(t, "photo-2.jpg", files[0].TargetFilename)
	require.True(t, files[0].ReuseExisting)
	require.Equal(t, "media/files/photo-2.jpg", media.URI)
}

func TestAllocateImportMediaDestinations_NumberedWithoutNestedSuffix(t *testing.T) {
	archiveDir := t.TempDir()
	sourceDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("occupied"), filePermissions))
	for _, sub := range []string{"a", "b"} {
		require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, sub), dirPermissions))
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, sub, "photo.jpg"), []byte(sub), filePermissions))
	}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
		"a": {URI: "media/files/photo.jpg"},
		"b": {URI: "media/files/photo-2.jpg"},
	}}
	files := []glxlib.MediaFileSource{
		{MediaID: "a", RelativePath: "a/photo.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"},
		{MediaID: "b", RelativePath: "b/photo.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo-2.jpg"},
	}
	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
	require.Equal(t, "photo-2.jpg", files[0].TargetFilename)
	require.Equal(t, "photo-3.jpg", files[1].TargetFilename)
	require.Equal(t, "media/files/photo-3.jpg", glx.Media["b"].URI)
}

func TestImportMediaMatcher_RechecksChangedDestination(t *testing.T) {
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	path := filepath.Join(filesDir, "photo.jpg")
	require.NoError(t, os.WriteFile(path, []byte("older"), filePermissions))
	m := &importMediaMatcher{
		archiveDir:        archiveDir,
		filesDir:          filesDir,
		destinationHashes: make(map[string]mediaHashCache),
		sourceHash:        sha256.Sum256([]byte("fresh")),
	}
	same, err := m.matchesFile("photo.jpg")
	require.NoError(t, err)
	require.False(t, same)
	require.Len(t, m.destinationHashes, 1)

	replacement := filepath.Join(filesDir, "replacement.tmp")
	require.NoError(t, os.WriteFile(replacement, []byte("fresh"), filePermissions))
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.Rename(replacement, path))
	same, err = m.matchesFile("photo.jpg")
	require.NoError(t, err)
	require.True(t, same)

	info := m.destinationHashes["photo.jpg"].info
	require.NoError(t, os.WriteFile(path, []byte("other"), filePermissions))
	require.NoError(t, os.Chtimes(path, info.ModTime(), info.ModTime()))
	same, err = m.matchesFile("photo.jpg")
	require.NoError(t, err)
	require.False(t, same, "a cached match must be verified before reusing the file")
}

func TestAllocateImportMediaDestinations_ReusesIdenticalBlob(t *testing.T) {
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "blob-media-1.bin"), []byte{0, 0, 0}, filePermissions))
	media := &glxlib.Media{URI: glxlib.MediaFilesDir + "/blob-media-1.bin"}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"blob": media}}
	files := []glxlib.MediaFileSource{{
		MediaID: "blob", SourceType: glxlib.MediaSourceBlob,
		BlobData: "....", TargetFilename: "blob-media-1.bin",
	}}

	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, "", nil))
	require.True(t, files[0].ReuseExisting)
	require.Equal(t, "blob-media-1.bin", files[0].TargetFilename)
	require.Equal(t, glxlib.MediaFilesDir+"/blob-media-1.bin", media.URI)
}

func TestAllocateImportMediaDestinations_DoesNotReuseUnreadableBundleMember(t *testing.T) {
	for _, member := range []string{"", "missing.jpg", "."} {
		t.Run("member="+member, func(t *testing.T) {
			archiveDir := t.TempDir()
			filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
			require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("old"), filePermissions))
			media := &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
			glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
			files := []glxlib.MediaFileSource{{
				MediaID: "file", SourceType: glxlib.MediaSourceFile,
				MemberPath: member, TargetFilename: "photo.jpg",
			}}

			require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, "", os.DirFS(t.TempDir())))
			require.False(t, files[0].ReuseExisting)
			require.Equal(t, "photo-2.jpg", files[0].TargetFilename)
			require.Equal(t, glxlib.MediaFilesDir+"/photo-2.jpg", media.URI)
		})
	}
}
