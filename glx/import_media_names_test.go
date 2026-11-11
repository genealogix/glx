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
	"io"
	"io/fs"
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
		{MediaID: "blob", SourceType: glxlib.MediaSourceBlob, BlobData: "....", TargetFilename: "photo.jpg"},
	}
	sourceDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "sub"), dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "sub", "photo.jpg"), []byte("new bytes"), filePermissions))

	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
	require.Equal(t, "photo-2.jpg", files[0].TargetFilename)
	require.Equal(t, "photo-3.jpg", files[1].TargetFilename)
	require.Equal(t, "media/files/photo-2.jpg", glx.Media["file"].URI)
	require.Equal(t, "photo.jpg", glx.Media["file"].Properties[glxlib.MediaPropertyOriginalFilename])
	require.Equal(t, "media/files/photo-3.jpg", glx.Media["blob"].URI)
	oldBytes, err := os.ReadFile(filepath.Join(filesDir, "Photo.JPG"))
	require.NoError(t, err)
	require.Equal(t, []byte("old bytes"), oldBytes)
}

// original_filename names the source file a GEDCOM 7.0 reference actually
// opened (#1153). The importer reads CharlotteBront%C3%AB.jpg as the decoded
// CharlotteBrontë.jpg, and the archive file is named that way (#1152), but a
// source directory holding only the literal %-named file is what the copy
// reads, so that is the name recorded.
func TestReserveImportMediaDestinations_RecordsTheSourceSpellingThatOpened(t *testing.T) {
	const encoded = "CharlotteBront%C3%AB.jpg"
	const decoded = "CharlotteBrontë.jpg"
	for _, tc := range []struct {
		name, onDisk, wantOriginal string
	}{
		{"decoded source", decoded, decoded},
		{"literal percent source", encoded, encoded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
			require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(filesDir, decoded), []byte("old bytes"), filePermissions))
			media := &glxlib.Media{URI: glxlib.MediaFilesDir + "/" + decoded}
			glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
			files := []glxlib.MediaFileSource{{
				MediaID: "file", SourceType: glxlib.MediaSourceFile, RelativePath: "media/files/" + encoded,
				OriginalFilename: decoded, RequestedFilename: decoded, TargetFilename: decoded,
			}}
			sourceDir := t.TempDir()
			require.NoError(t, os.MkdirAll(filepath.Join(sourceDir, "media", "files"), dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "media", "files", tc.onDisk), []byte("new bytes"), filePermissions))

			require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
			require.Equal(t, "CharlotteBrontë-2.jpg", files[0].TargetFilename)
			require.Equal(t, glxlib.MediaFilesDir+"/CharlotteBrontë-2.jpg", media.URI)
			require.Equal(t, tc.wantOriginal, media.Properties[glxlib.MediaPropertyOriginalFilename])
		})
	}
}

func TestReserveImportMediaDestinations_UnoccupiedAndEmpty(t *testing.T) {
	archiveDir := t.TempDir()
	require.NoError(t, allocateImportMediaDestinations(nil, nil, archiveDir, "", nil))
	media := &glxlib.Media{URI: "media/files/unique.jpg"}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
	files := []glxlib.MediaFileSource{{MediaID: "file", TargetFilename: "unique.jpg"}}
	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, "", nil))
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
			require.Error(t, allocateImportMediaDestinations(&glxlib.GLXFile{}, files, archiveDir, "", nil))
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
	require.True(t, same, "allocation trusts a digest whose size and mtime are unchanged")

	files := []glxlib.MediaFileSource{{
		MediaID: "file", TargetFilename: "photo.jpg", ReuseExisting: true, ReuseDigest: m.sourceHash,
	}}
	require.ErrorIs(t, validateReusedMedia(archiveDir, files, "", nil), os.ErrInvalid,
		"publication must reread the destination, so the stale digest cannot be published")
}

// countingFS counts opens per member, to pin how often a source is read.
type countingFS struct {
	fs.FS
	opens map[string]int
}

func (c countingFS) Open(name string) (fs.File, error) {
	c.opens[name]++

	return c.FS.Open(name)
}

func TestReusedMedia_ReadsSourceOnceAcrossAllocationAndPublication(t *testing.T) {
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("same"), filePermissions))
	sourceDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "photo.jpg"), []byte("same"), filePermissions))
	bundle := countingFS{FS: os.DirFS(sourceDir), opens: make(map[string]int)}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": {URI: glxlib.MediaFilesDir + "/photo.jpg"}}}
	files := []glxlib.MediaFileSource{{
		MediaID: "file", SourceType: glxlib.MediaSourceFile, MemberPath: "photo.jpg", TargetFilename: "photo.jpg",
	}}

	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, "", bundle))
	require.True(t, files[0].ReuseExisting)
	require.NoError(t, validateReusedMedia(archiveDir, files, "", bundle))
	require.Equal(t, 1, bundle.opens["photo.jpg"])
}

func TestImport_UnreadableSourceKeepsWorkingReference(t *testing.T) {
	for _, format := range []string{FormatSingle, FormatMulti} {
		t.Run(format, func(t *testing.T) {
			work := t.TempDir()
			output := filepath.Join(work, "archive")
			archiveDir := output
			if format == FormatSingle {
				output += ".glx"
				archiveDir = work
			}
			filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
			require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
			existing := filepath.Join(filesDir, "photo.jpg")
			require.NoError(t, os.WriteFile(existing, []byte("first import"), filePermissions))
			media := &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
			glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"file": media}}
			files := []glxlib.MediaFileSource{{
				MediaID: "file", SourceType: glxlib.MediaSourceFile, RelativePath: "photo.jpg", TargetFilename: "photo.jpg",
			}}
			sourceDir := t.TempDir() // the images have moved away

			require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, sourceDir, nil))
			var err error
			if format == FormatSingle {
				err = importToSingleFile(glx, output, false, false, 0, files, sourceDir, io.Discard)
			} else {
				err = importToMultiFile(glx, output, false, false, 0, files, sourceDir, io.Discard)
			}
			require.NoError(t, err)

			require.Equal(t, glxlib.MediaFilesDir+"/photo.jpg", media.URI)
			require.NoFileExists(t, filepath.Join(filesDir, "photo-2.jpg"))
			kept, err := os.ReadFile(existing)
			require.NoError(t, err)
			require.Equal(t, []byte("first import"), kept)
		})
	}
}

func TestCommitPreparedMedia_RejectsDirectoryChangedSinceURIPreparation(t *testing.T) {
	targetDir := t.TempDir()
	stageDir := filepath.Join(targetDir, ".glx-stage")
	stagedFiles := filepath.Join(stageDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(stagedFiles, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(stagedFiles, "photo.jpg"), []byte("new"), filePermissions))
	current := filepath.Join(targetDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(current, dirPermissions))

	_, err := commitPreparedMedia(stageDir, targetDir, filepath.Join(targetDir, "MEDIA", "files"))
	require.ErrorIs(t, err, os.ErrInvalid)
	require.NoFileExists(t, filepath.Join(current, "photo.jpg"))

	rollback, err := commitPreparedMedia(stageDir, targetDir, current)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(current, "photo.jpg"))
	rollback()
	require.NoFileExists(t, filepath.Join(current, "photo.jpg"))
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

func TestAllocateImportMediaDestinations_UnreadableSourceAvoidsOccupiedVariantClaimedEarlier(t *testing.T) {
	archiveDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(filesDir, "photo.jpg"), []byte("old"), filePermissions))
	first := &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
	second := &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
	glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"first": first, "second": second}}
	files := []glxlib.MediaFileSource{
		{MediaID: "first", SourceType: glxlib.MediaSourceFile, MemberPath: "gone.jpg", TargetFilename: "photo.jpg"},
		{MediaID: "second", SourceType: glxlib.MediaSourceFile, MemberPath: "gone-too.jpg", TargetFilename: "photo.jpg"},
	}

	require.NoError(t, allocateImportMediaDestinations(glx, files, archiveDir, "", os.DirFS(t.TempDir())))
	require.True(t, files[0].KeepExisting)
	require.Equal(t, "photo.jpg", files[0].TargetFilename)
	require.False(t, files[1].KeepExisting, "one on-disk file cannot back two records")
	require.Equal(t, "photo-2.jpg", files[1].TargetFilename)
}

func TestAllocateImportMediaDestinations_KeepsExistingReferenceForUnreadableSource(t *testing.T) {
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
			require.False(t, files[0].ReuseExisting, "unverifiable bytes must not count as a content match")
			require.True(t, files[0].KeepExisting)
			require.Equal(t, "photo.jpg", files[0].TargetFilename)
			require.Equal(t, glxlib.MediaFilesDir+"/photo.jpg", media.URI)
		})
	}
}

// allocationFixture writes on-disk media (name -> content) and one source per
// entry of incoming (content), all requesting photo.jpg, and returns the
// allocated sources.
func allocationFixture(t *testing.T, onDisk map[string]string, incoming []string) []glxlib.MediaFileSource {
	t.Helper()
	archiveDir := t.TempDir()
	sourceDir := t.TempDir()
	filesDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
	require.NoError(t, os.MkdirAll(filesDir, dirPermissions))
	for name, content := range onDisk {
		require.NoError(t, os.WriteFile(filepath.Join(filesDir, name), []byte(content), filePermissions))
	}
	archive := &glxlib.GLXFile{Media: make(map[string]*glxlib.Media)}
	files := make([]glxlib.MediaFileSource, 0, len(incoming))
	for i, content := range incoming {
		id := fmt.Sprintf("incoming-%d", i)
		require.NoError(t, os.WriteFile(filepath.Join(sourceDir, id), []byte(content), filePermissions))
		archive.Media[id] = &glxlib.Media{URI: glxlib.MediaFilesDir + "/photo.jpg"}
		files = append(files, glxlib.MediaFileSource{
			MediaID: id, SourceType: glxlib.MediaSourceFile, RelativePath: id,
			RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg",
		})
	}
	require.NoError(t, allocateImportMediaDestinations(archive, files, archiveDir, sourceDir, nil))

	return files
}

func allocatedNames(files []glxlib.MediaFileSource) []string {
	names := make([]string, len(files))
	for i := range files {
		names[i] = files[i].TargetFilename
		if files[i].ReuseExisting {
			names[i] += " (reused)"
		}
	}

	return names
}

// One allocation path for every family size (#1442): these cases used to run
// through FilenameAllocator.AllocateMatching or the content index depending on
// how many numbered names a family had.
func TestAllocateImportMediaDestinations_OnePath(t *testing.T) {
	t.Run("lowest ordinal match first, then the next free name", func(t *testing.T) {
		files := allocationFixture(t, map[string]string{"photo.jpg": "a", "photo-2.jpg": "a"}, []string{"a", "a", "a"})
		require.Equal(t, []string{"photo.jpg (reused)", "photo-2.jpg (reused)", "photo-3.jpg"}, allocatedNames(files))
	})

	t.Run("a free gap is used before a later match", func(t *testing.T) {
		files := allocationFixture(t, map[string]string{"photo.jpg": "x", "photo-3.jpg": "c"}, []string{"new", "new2", "c"})
		require.Equal(t, []string{"photo-2.jpg", "photo-4.jpg", "photo-3.jpg (reused)"}, allocatedNames(files))
	})

	t.Run("a sparse high name does not slow the gap below it", func(t *testing.T) {
		incoming := make([]string, 1000)
		want := make([]string, 1000)
		for i := range incoming {
			incoming[i] = fmt.Sprintf("new %d", i)
			want[i] = fmt.Sprintf("photo-%d.jpg", i+2)
		}
		files := allocationFixture(t, map[string]string{"photo.jpg": "x", "photo-1000000000.jpg": "y"}, incoming)
		require.Equal(t, want, allocatedNames(files))
	})

	t.Run("a dense refresh reuses every file in order", func(t *testing.T) {
		const count = 500
		onDisk := make(map[string]string, count)
		incoming := make([]string, 0, count+1)
		want := make([]string, 0, count+1)
		for n := 1; n <= count; n++ {
			name := "photo.jpg"
			if n > 1 {
				name = fmt.Sprintf("photo-%d.jpg", n)
			}
			onDisk[name] = fmt.Sprintf("content %d", n)
			incoming = append(incoming, fmt.Sprintf("content %d", n))
			want = append(want, name+" (reused)")
		}
		incoming = append(incoming, "something new")
		want = append(want, fmt.Sprintf("photo-%d.jpg", count+1))
		files := allocationFixture(t, onDisk, incoming)
		require.Equal(t, want, allocatedNames(files))
	})
}
