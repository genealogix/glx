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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

var errTestLaterMetadataWrite = errors.New("later metadata write failed")

func TestMultiFileImport_PartialMetadataWriteKeepsReferencedMedia(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		t.Run(input, func(t *testing.T) {
			work := t.TempDir()
			output := filepath.Join(work, "archive")
			variant := filepath.Join(output, "MEDIA", "files")
			require.NoError(t, os.MkdirAll(variant, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(variant, "photo.jpg"), []byte("old"), filePermissions))
			sourceDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "new.jpg"), []byte("new"), filePermissions))
			glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
				"photo": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
			}}
			media := []glxlib.MediaFileSource{{
				MediaID: "photo", SourceType: glxlib.MediaSourceFile, RelativePath: "new.jpg",
				MemberPath: "new.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg",
			}}
			bundle := os.DirFS(sourceDir)
			if input == "GEDCOM" {
				require.NoError(t, allocateImportMediaDestinations(glx, media, output, sourceDir, nil))
			} else {
				require.NoError(t, allocateImportMediaDestinations(glx, media, output, "", bundle))
			}
			require.Equal(t, "photo-2.jpg", media[0].TargetFilename)

			// writeFilesToDir writes map entries one at a time. Publish the
			// Media entity first, then fail as a later metadata entry could.
			// This makes the partial-write boundary deterministic.
			var published string
			writePartial := func(root string, files map[string][]byte) error {
				for rel, content := range files {
					if strings.HasPrefix(rel, "media/") && strings.HasSuffix(rel, ".glx") {
						published = rel
						if err := writeFilesToDir(root, map[string][]byte{rel: content}); err != nil {
							return err
						}

						return errTestLaterMetadataWrite
					}
				}

				return fmt.Errorf("serialized archive has no Media entity: %w", os.ErrNotExist)
			}
			var err error
			if input == "GEDCOM" {
				err = importToMultiFileUsing(glx, output, false, false, 0, media, sourceDir, io.Discard, writePartial)
			} else {
				err = importGEDZIPToMultiFileUsing(glx, output, false, false, 0, media, bundle, io.Discard, writePartial)
			}
			require.ErrorIs(t, err, errTestLaterMetadataWrite)
			require.NotEmpty(t, published)
			require.Contains(t, string(mustReadImportMedia(t, filepath.Join(output, published))), "MEDIA/files/photo-2.jpg")
			require.Equal(t, []byte("new"), mustReadImportMedia(t, filepath.Join(variant, "photo-2.jpg")))
			require.Equal(t, []byte("old"), mustReadImportMedia(t, filepath.Join(variant, "photo.jpg")))
		})
	}
}

func TestImport_CaseVariantMediaDirectory(t *testing.T) {
	for _, variant := range []string{"MEDIA/files", "media/FILES", "MEDIA/FILES"} {
		for _, input := range []string{"GEDCOM", "GEDZIP"} {
			for _, format := range []string{FormatSingle, FormatMulti} {
				t.Run(variant+"/"+input+"/"+format, func(t *testing.T) {
					work := t.TempDir()
					if !fsCaseSensitive(t, work) {
						t.Skip("requires a case-sensitive filesystem")
					}
					output := filepath.Join(work, "archive")
					archiveDir := output
					if format == FormatSingle {
						archiveDir = work
					}
					mediaDir := filepath.Join(archiveDir, variant)
					require.NoError(t, os.MkdirAll(mediaDir, dirPermissions))
					require.NoError(t, os.WriteFile(filepath.Join(mediaDir, "photo.jpg"), []byte("same"), filePermissions))
					sourceDir := t.TempDir()
					require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "reuse.jpg"), []byte("same"), filePermissions))
					require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "fresh.jpg"), []byte("different"), filePermissions))
					glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
						"reuse": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
						"fresh": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
					}}
					media := []glxlib.MediaFileSource{
						{MediaID: "reuse", SourceType: glxlib.MediaSourceFile, RelativePath: "reuse.jpg", MemberPath: "reuse.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"},
						{MediaID: "fresh", SourceType: glxlib.MediaSourceFile, RelativePath: "fresh.jpg", MemberPath: "fresh.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"},
					}
					bundle := os.DirFS(sourceDir)
					if input == "GEDCOM" {
						require.NoError(t, allocateImportMediaDestinations(glx, media, archiveDir, sourceDir, nil))
					} else {
						require.NoError(t, allocateImportMediaDestinations(glx, media, archiveDir, "", bundle))
					}
					require.True(t, media[0].ReuseExisting)
					require.Equal(t, "photo.jpg", media[0].TargetFilename)
					require.Equal(t, "photo-2.jpg", media[1].TargetFilename)
					var err error
					switch {
					case input == "GEDCOM" && format == FormatSingle:
						err = importToSingleFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
					case input == "GEDCOM":
						err = importToMultiFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
					case format == FormatSingle:
						err = importGEDZIPToSingleFile(glx, output, false, false, 0, media, bundle, io.Discard)
					default:
						err = importGEDZIPToMultiFile(glx, output, false, false, 0, media, bundle, io.Discard)
					}
					require.NoError(t, err)
					actualDir := filepath.Join(archiveDir, variant)
					require.Equal(t, []byte("same"), mustReadImportMedia(t, filepath.Join(actualDir, "photo.jpg")))
					require.Equal(t, []byte("different"), mustReadImportMedia(t, filepath.Join(actualDir, "photo-2.jpg")))
					dirs, err := mediaFilesDirsIn(archiveDir)
					require.NoError(t, err)
					require.Equal(t, []string{actualDir}, dirs)
					require.Equal(t, variant+"/photo.jpg", glx.Media["reuse"].URI)
					require.Equal(t, variant+"/photo-2.jpg", glx.Media["fresh"].URI)
				})
			}
		}
	}
}

func TestImport_AmbiguousMediaDirectoriesRejectBeforeWrites(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		for _, format := range []string{FormatSingle, FormatMulti} {
			t.Run(input+"/"+format, func(t *testing.T) {
				work := t.TempDir()
				if !fsCaseSensitive(t, work) {
					t.Skip("requires a case-sensitive filesystem")
				}
				output := filepath.Join(work, "archive")
				archiveDir := output
				oldPath := filepath.Join(output, "metadata.glx")
				if format == FormatSingle {
					archiveDir = work
					oldPath = output + ".glx"
				}
				for _, variant := range []string{glxlib.MediaFilesDir, "MEDIA/files"} {
					dir := filepath.Join(archiveDir, variant)
					require.NoError(t, os.MkdirAll(dir, dirPermissions))
					require.NoError(t, os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte(variant), filePermissions))
				}
				require.NoError(t, os.WriteFile(oldPath, []byte("old archive"), filePermissions))
				sourceDir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "photo.jpg"), []byte("new"), filePermissions))
				media := []glxlib.MediaFileSource{{MediaID: "photo", SourceType: glxlib.MediaSourceFile, RelativePath: "photo.jpg", MemberPath: "photo.jpg", TargetFilename: "photo.jpg"}}
				glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"photo": {URI: glxlib.MediaFilesDir + "/photo.jpg"}}}
				var err error
				if input == "GEDCOM" {
					err = allocateImportMediaDestinations(glx, media, archiveDir, sourceDir, nil)
				} else {
					err = allocateImportMediaDestinations(glx, media, archiveDir, "", os.DirFS(sourceDir))
				}
				require.ErrorIs(t, err, ErrAmbiguousMediaFilesDirs)
				// The publication helpers also refuse an ambiguous archive when
				// invoked directly, before staging or touching existing metadata.
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
				require.ErrorIs(t, err, ErrAmbiguousMediaFilesDirs)
				for _, variant := range []string{glxlib.MediaFilesDir, "MEDIA/files"} {
					require.Equal(t, []byte(variant), mustReadImportMedia(t, filepath.Join(archiveDir, variant, "photo.jpg")))
				}
				require.Equal(t, []byte("old archive"), mustReadImportMedia(t, oldPath))
				require.False(t, media[0].ReuseExisting)
				stages, err := filepath.Glob(filepath.Join(archiveDir, ".glx-stage-*"))
				require.NoError(t, err)
				require.Empty(t, stages)
			})
		}
	}
}

func TestSingleFileImport_CaseVariantPublicationFailureRestoresMedia(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		t.Run(input, func(t *testing.T) {
			work := t.TempDir()
			if !fsCaseSensitive(t, work) {
				t.Skip("requires a case-sensitive filesystem")
			}
			output := filepath.Join(work, "archive.glx")
			require.NoError(t, os.Mkdir(output, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(output, "marker"), []byte("original"), filePermissions))
			variant := filepath.Join(work, "MEDIA", "files")
			require.NoError(t, os.MkdirAll(variant, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(variant, "photo.jpg"), []byte("same"), filePermissions))
			sourceDir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "reuse.jpg"), []byte("same"), filePermissions))
			require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "fresh.jpg"), []byte("different"), filePermissions))
			glx := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
				"reuse": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
				"fresh": {URI: glxlib.MediaFilesDir + "/photo.jpg"},
			}}
			media := []glxlib.MediaFileSource{
				{MediaID: "reuse", SourceType: glxlib.MediaSourceFile, RelativePath: "reuse.jpg", MemberPath: "reuse.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"},
				{MediaID: "fresh", SourceType: glxlib.MediaSourceFile, RelativePath: "fresh.jpg", MemberPath: "fresh.jpg", RequestedFilename: "photo.jpg", TargetFilename: "photo.jpg"},
			}
			bundle := os.DirFS(sourceDir)
			var err error
			if input == "GEDCOM" {
				require.NoError(t, allocateImportMediaDestinations(glx, media, work, sourceDir, nil))
				err = importToSingleFile(glx, output, false, false, 0, media, sourceDir, io.Discard)
			} else {
				require.NoError(t, allocateImportMediaDestinations(glx, media, work, "", bundle))
				err = importGEDZIPToSingleFile(glx, output, false, false, 0, media, bundle, io.Discard)
			}
			require.Error(t, err)
			require.Equal(t, []byte("original"), mustReadImportMedia(t, filepath.Join(output, "marker")))
			require.Equal(t, []byte("same"), mustReadImportMedia(t, filepath.Join(variant, "photo.jpg")))
			require.NoFileExists(t, filepath.Join(variant, "photo-2.jpg"))
			require.NoDirExists(t, filepath.Join(work, glxlib.MediaFilesDir))
		})
	}
}

func mustReadImportMedia(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	return data
}

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
			for _, mutation := range []string{"replaced", "deleted", "source-changed"} {
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
					switch mutation {
					case "deleted":
						require.NoError(t, os.Remove(filepath.Join(mediaDir, "photo.jpg")))
					case "source-changed":
						source := filepath.Join(sourceDir, "photo.jpg")
						info, statErr := os.Stat(source)
						require.NoError(t, statErr)
						require.NoError(t, os.WriteFile(source, []byte("changed!"), filePermissions))
						require.NoError(t, os.Chtimes(source, info.ModTime(), info.ModTime()))
					default:
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

func TestMultiFileImport_NoMetadataPublishedRollsBackMedia(t *testing.T) {
	for _, input := range []string{"GEDCOM", "GEDZIP"} {
		t.Run(input, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "archive")
			variant := filepath.Join(output, "Media", "Files")
			require.NoError(t, os.MkdirAll(variant, dirPermissions))
			require.NoError(t, os.WriteFile(filepath.Join(variant, "old.jpg"), []byte("old"), filePermissions))
			source := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(source, "new.jpg"), []byte("new"), filePermissions))
			archive := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"new": {URI: "media/files/new.jpg"}}}
			media := []glxlib.MediaFileSource{{MediaID: "new", SourceType: glxlib.MediaSourceFile, RelativePath: "new.jpg", MemberPath: "new.jpg", TargetFilename: "new.jpg"}}
			fail := func(string, map[string][]byte) error { return errTestLaterMetadataWrite }
			var err error
			stdout := captureStdout(t, func() {
				if input == "GEDCOM" {
					err = importToMultiFileUsing(archive, output, false, false, 0, media, source, io.Discard, fail)
				} else {
					err = importGEDZIPToMultiFileUsing(archive, output, false, false, 0, media, os.DirFS(source), io.Discard, fail)
				}
			})
			require.NotContains(t, stdout, "Media files:")
			require.ErrorIs(t, err, errTestLaterMetadataWrite)
			require.Equal(t, []byte("old"), mustReadImportMedia(t, filepath.Join(variant, "old.jpg")))
			require.NoFileExists(t, filepath.Join(variant, "new.jpg"))
			require.NoFileExists(t, filepath.Join(output, "media", "files", "new.jpg"))
		})
	}
}

func TestImportMetadataSnapshotDetectsPartialOverwrite(t *testing.T) {
	root := t.TempDir()
	name := "metadata.glx"
	require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("old bytes"), filePermissions))
	snapshot := snapshotImportMetadata(root, map[string][]byte{name: []byte("new bytes"), "persons/new.glx": []byte("new")})
	require.True(t, snapshot.unchanged(root))
	require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("new"), filePermissions))
	require.False(t, snapshot.unchanged(root), "partial writes must retain potentially referenced binaries")
	require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("old bytes"), filePermissions))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "persons"), dirPermissions))
	require.NoError(t, os.WriteFile(filepath.Join(root, "persons/new.glx"), []byte("partial"), filePermissions))
	require.False(t, snapshot.unchanged(root))
}
