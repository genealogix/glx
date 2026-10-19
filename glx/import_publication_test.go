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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestPlainImport_MediaCommitFailurePreservesExistingArchive(t *testing.T) {
	media := []glxlib.MediaFileSource{
		{SourceType: glxlib.MediaSourceBlob, TargetFilename: "same.bin", BlobData: "...."},
		{SourceType: glxlib.MediaSourceBlob, TargetFilename: "same.bin", BlobData: "...."},
	}
	for _, format := range []string{FormatSingle, FormatMulti} {
		t.Run(format, func(t *testing.T) {
			work := t.TempDir()
			output := filepath.Join(work, "archive")
			oldPath := filepath.Join(output, "metadata.glx")
			if format == FormatSingle {
				output += ".glx"
				oldPath = output
			}
			require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), dirPermissions))
			require.NoError(t, os.WriteFile(oldPath, []byte("old archive"), filePermissions))
			glx := &glxlib.GLXFile{}
			var err error
			if format == FormatSingle {
				err = importToSingleFile(glx, output, false, false, 0, media, work, io.Discard)
			} else {
				err = importToMultiFile(glx, output, false, false, 0, media, work, io.Discard)
			}
			require.ErrorIs(t, err, os.ErrExist)
			actual, readErr := os.ReadFile(oldPath)
			require.NoError(t, readErr)
			require.Equal(t, []byte("old archive"), actual)
		})
	}
}

func TestSingleFileImport_ArchivePublicationFailureRollsBackMedia(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		t.Run(input, func(t *testing.T) {
			work := t.TempDir()
			output := filepath.Join(work, "archive.glx")
			require.NoError(t, os.Mkdir(output, dirPermissions))
			sourceDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "photo.jpg"), []byte("new media"), filePermissions))
			media := []glxlib.MediaFileSource{{
				SourceType: glxlib.MediaSourceFile, RelativePath: "photo.jpg",
				MemberPath: "photo.jpg", TargetFilename: "photo.jpg",
			}}
			glx := &glxlib.GLXFile{}
			var err error
			if input == "GEDCOM" {
				err = importToSingleFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
			} else {
				err = importGEDZIPToSingleFile(glx, output, false, false, 0, media, os.DirFS(sourceDir), io.Discard)
			}
			require.Error(t, err)
			require.DirExists(t, output)
			require.NoFileExists(t, filepath.Join(work, glxlib.MediaFilesDir, "photo.jpg"))
		})
	}
}

func TestImport_ReusedMediaChangedBeforePublicationPreservesArchive(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		for _, format := range []string{FormatSingle, FormatMulti} {
			for _, mutation := range []string{"replaced", "deleted"} {
				t.Run(input+"/"+format+"/"+mutation, func(t *testing.T) {
					work := t.TempDir()
					output := filepath.Join(work, "archive")
					archiveDir := output
					if format == FormatSingle {
						output += ".glx"
						archiveDir = work
					}
					mediaDir := filepath.Join(archiveDir, glxlib.MediaFilesDir)
					require.NoError(t, os.MkdirAll(mediaDir, dirPermissions))
					require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "photo.jpg"), []byte("expected"), filePermissions))
					oldPath := filepath.Join(output, "metadata.glx")
					if format == FormatSingle {
						oldPath = output
					}
					oldArchive := []byte("old archive")
					require.NoError(t, os.WriteFile(oldPath, oldArchive, filePermissions))

					sourceDir := t.TempDir()
					require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "photo.jpg"), []byte("expected"), filePermissions))
					require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "new.jpg"), []byte("new media"), filePermissions))
					glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
						"photo": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
						"new":   {URI: glxlib.MediaFilesDir + "/new.jpg"},
					}}
					media := []glxlib.MediaFileSource{
						{MediaID: "photo", SourceType: glxlib.MediaSourceFile, RelativePath: "photo.jpg", MemberPath: "photo.jpg", TargetFilename: "photo.jpg"},
						{MediaID: "new", SourceType: glxlib.MediaSourceFile, RelativePath: "new.jpg", MemberPath: "new.jpg", TargetFilename: "new.jpg"},
					}
					if input == "GEDCOM" {
						require.NoError(t, allocateImportMediaDestinations(glx, media, archiveDir, sourceDir, nil))
					} else {
						require.NoError(t, allocateImportMediaDestinations(glx, media, archiveDir, "", os.DirFS(sourceDir)))
					}
					require.True(t, media[0].ReuseExisting)
					if mutation == "deleted" {
						require.NoError(t, os.Remove(filepath.Join(mediaDir, "photo.jpg")))
					} else {
						require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "photo.jpg"), []byte("changed!"), filePermissions))
					}

					var err error
					switch {
					case input == "GEDCOM" && format == FormatSingle:
						err = importToSingleFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
					case input == "GEDCOM":
						err = importToMultiFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
					case format == FormatSingle:
						err = importGEDZIPToSingleFile(glx, output, false, false, 0, media, os.DirFS(sourceDir), io.Discard)
					default:
						err = importGEDZIPToMultiFile(glx, output, false, false, 0, media, os.DirFS(sourceDir), io.Discard)
					}
					if mutation == "deleted" {
						require.ErrorIs(t, err, os.ErrNotExist)
					} else {
						require.ErrorIs(t, err, os.ErrInvalid)
					}
					actualArchive, readErr := os.ReadFile(oldPath)
					require.NoError(t, readErr)
					require.Equal(t, oldArchive, actualArchive)
					require.NoFileExists(t, filepath.Join(mediaDir, "new.jpg"))
				})
			}
		}
	}
}
