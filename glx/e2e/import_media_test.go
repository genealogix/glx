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

package e2e

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestImport_MediaFilenameCollisionsPreserveBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"natural before generated", []string{"photo.jpg", "photo-2.jpg", "photo.jpg"}, []string{"photo.jpg", "photo-2.jpg", "photo-3.jpg"}},
		{"natural first", []string{"photo-2.jpg", "photo.jpg", "photo.jpg"}, []string{"photo-2.jpg", "photo.jpg", "photo-3.jpg"}},
		{"generated before natural", []string{"photo.jpg", "photo.jpg", "photo-2.jpg"}, []string{"photo.jpg", "photo-2.jpg", "photo-2-2.jpg"}},
		{"multiple occupied suffixes", []string{"photo.jpg", "photo-2.jpg", "photo-3.jpg", "photo.jpg"}, []string{"photo.jpg", "photo-2.jpg", "photo-3.jpg", "photo-4.jpg"}},
		{"case equivalent", []string{"Photo.jpg", "photo.jpg"}, []string{"Photo.jpg", "photo-2.jpg"}},
		{"extension case", []string{"photo.JPG", "photo.jpg"}, []string{"photo.JPG", "photo-2.jpg"}},
		{"unicode normalization", []string{"Caf\u00e9.jpg", "Cafe\u0301.jpg"}, []string{"Caf\u00e9.jpg", "Cafe\u0301-2.jpg"}},
	}
	for _, tt := range tests {
		for _, extension := range []string{"ged", "gdz"} {
			for _, format := range []string{"multi", "single"} {
				t.Run(tt.name+"/"+extension+"/"+format, func(t *testing.T) {
					work := t.TempDir()
					var gedcom strings.Builder
					version := "5.5.1"
					if extension == "gdz" {
						version = "7.0"
					}
					fmt.Fprintf(&gedcom, "0 HEAD\n1 GEDC\n2 VERS %s\n", version)
					var bundle bytes.Buffer
					writer := zip.NewWriter(&bundle)
					payloads := make([][]byte, len(tt.input))
					colors := []color.RGBA{{R: 240, A: 255}, {G: 240, A: 255}, {B: 240, A: 255}, {R: 240, G: 240, A: 255}}
					for i, name := range tt.input {
						relPath := fmt.Sprintf("source-%d/%s", i+1, name)
						img := image.NewRGBA(image.Rect(0, 0, 16, 16))
						for y := range 16 {
							for x := range 16 {
								img.SetRGBA(x, y, colors[i])
							}
						}
						var encoded bytes.Buffer
						require.NoError(t, jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}))
						payloads[i] = encoded.Bytes()
						path := filepath.Join(work, filepath.FromSlash(relPath))
						require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
						require.NoError(t, os.WriteFile(path, payloads[i], 0o600))
						entry, err := writer.Create(relPath)
						require.NoError(t, err)
						_, err = entry.Write(payloads[i])
						require.NoError(t, err)
						fmt.Fprintf(&gedcom, "0 @M%d@ OBJE\n1 FILE %s\n2 FORM image/jpeg\n2 TITL Image %d\n", i+1, relPath, i+1)
					}
					gedcom.WriteString("0 TRLR\n")
					entry, err := writer.Create("gedcom.ged")
					require.NoError(t, err)
					_, err = entry.Write([]byte(gedcom.String()))
					require.NoError(t, err)
					require.NoError(t, writer.Close())
					input := []byte(gedcom.String())
					if extension == "gdz" {
						input = bundle.Bytes()
					}
					src := filepath.Join(work, "collision."+extension)
					require.NoError(t, os.WriteFile(src, input, 0o600))
					output := filepath.Join(work, "archive")
					mediaRoot := output
					if format == "single" {
						output += ".glx"
						mediaRoot = work
					}
					res := runGLX(t, work, "import", src, "-o", output, "--format", format)
					require.Equal(t, 0, res.exitCode, res.stderr)
					assert.NotContains(t, res.stdout, "Warning:")
					files := []string{output}
					if format == "multi" {
						files, err = filepath.Glob(filepath.Join(output, "media", "*.glx"))
						require.NoError(t, err)
					}
					media := make(map[string]*glxlib.Media)
					for _, file := range files {
						data, readErr := os.ReadFile(file)
						require.NoError(t, readErr)
						var fragment glxlib.GLXFile
						require.NoError(t, yaml.Unmarshal(data, &fragment))
						maps.Copy(media, fragment.Media)
					}
					require.Len(t, media, len(tt.input))
					seenHashes := make(map[[32]byte]bool)
					for i, want := range tt.want {
						id := fmt.Sprintf("media-%d", i+1)
						entity := media[id]
						require.NotNil(t, entity, id)
						assert.Equal(t, "media/files/"+want, entity.URI, id)
						actual, readErr := os.ReadFile(filepath.Join(mediaRoot, filepath.FromSlash(entity.URI)))
						require.NoError(t, readErr)
						assert.Equal(t, payloads[i], actual, "media reference %s must retain its own image", id)
						seenHashes[sha256.Sum256(actual)] = true
					}
					assert.Len(t, seenHashes, len(tt.input), "every distinct source image must survive")
					copied, err := os.ReadDir(filepath.Join(mediaRoot, "media", "files"))
					require.NoError(t, err)
					assert.Len(t, copied, len(tt.input))
					validation := runGLX(t, work, "validate", output)
					require.Equal(t, 0, validation.exitCode, validation.stderr)
				})
			}
		}
	}
}

func TestImport_PreservesExistingMediaFiles(t *testing.T) {
	for _, extension := range []string{"ged", "gdz"} {
		for _, format := range []string{"multi", "single"} {
			t.Run(extension+"/"+format, func(t *testing.T) {
				work := t.TempDir()
				firstBytes := mediaTestJPEG(t, color.RGBA{R: 240, A: 255})
				secondBytes := mediaTestJPEG(t, color.RGBA{B: 240, A: 255})
				first := writeMediaTestInput(t, work, "first", extension, "Photo.JPG", firstBytes)
				second := writeMediaTestInput(t, work, "second", extension, "photo.jpg", secondBytes)
				outputDir := filepath.Join(work, "output")
				require.NoError(t, os.MkdirAll(outputDir, 0o755))
				firstOutput := filepath.Join(outputDir, "archive")
				secondOutput := firstOutput
				mediaRoot := firstOutput
				if format == "single" {
					firstOutput = filepath.Join(outputDir, "first.glx")
					secondOutput = filepath.Join(outputDir, "second.glx")
					mediaRoot = outputDir
				}

				res := runGLX(t, work, "import", first, "-o", firstOutput, "--format", format)
				require.Equal(t, 0, res.exitCode, res.stderr)
				res = runGLX(t, work, "import", second, "-o", secondOutput, "--format", format)
				require.Equal(t, 0, res.exitCode, res.stderr)
				media := importedMediaEntity(t, secondOutput, format, "media-1")
				require.Equal(t, "media/files/photo-2.jpg", media.URI)
				actual, err := os.ReadFile(filepath.Join(mediaRoot, filepath.FromSlash(media.URI)))
				require.NoError(t, err)
				require.Equal(t, secondBytes, actual)
				original, err := os.ReadFile(filepath.Join(mediaRoot, "media", "files", "Photo.JPG"))
				require.NoError(t, err)
				require.Equal(t, firstBytes, original, "prior import's image must survive")
				if format == "single" {
					firstMedia := importedMediaEntity(t, firstOutput, format, "media-1")
					require.Equal(t, "media/files/Photo.JPG", firstMedia.URI)
				}
			})
		}
	}
}

func TestImport_GEDCOM7DestinationCollisionKeepsDecodedOriginalName(t *testing.T) {
	const encoded = "CharlotteBront%C3%AB.jpg"
	const decoded = "CharlotteBrontë.jpg"
	for _, extension := range []string{"ged", "gdz"} {
		for _, format := range []string{"multi", "single"} {
			t.Run(extension+"/"+format, func(t *testing.T) {
				work := t.TempDir()
				gedcom := "0 HEAD\n1 GEDC\n2 VERS 7.0\n0 @M1@ OBJE\n1 FILE media/files/" + encoded + "\n2 FORM image/jpeg\n0 TRLR\n"
				sourceBytes := mediaTestJPEG(t, color.RGBA{G: 240, A: 255})
				src := writeMediaTestArchive(t, filepath.Join(work, "source"), extension, gedcom, "media/files/"+decoded, sourceBytes)
				outputRoot := filepath.Join(work, "output")
				output := outputRoot
				if format == "single" {
					output = filepath.Join(outputRoot, "archive.glx")
				}
				filesDir := filepath.Join(outputRoot, "media", "files")
				require.NoError(t, os.MkdirAll(filesDir, 0o755))
				originalBytes := []byte("already present")
				require.NoError(t, os.WriteFile(filepath.Join(filesDir, encoded), originalBytes, 0o600))

				res := runGLX(t, work, "import", src, "-o", output, "--format", format)
				require.Equal(t, 0, res.exitCode, res.stderr)
				media := importedMediaEntity(t, output, format, "media-1")
				require.Equal(t, "media/files/CharlotteBront%C3%AB-2.jpg", media.URI)
				require.Equal(t, decoded, media.Properties[glxlib.MediaPropertyOriginalFilename])
				actual, err := os.ReadFile(filepath.Join(outputRoot, filepath.FromSlash(media.URI)))
				require.NoError(t, err)
				require.Equal(t, sourceBytes, actual)
				prior, err := os.ReadFile(filepath.Join(filesDir, encoded))
				require.NoError(t, err)
				require.Equal(t, originalBytes, prior)
			})
		}
	}
}

func TestImport_MixedFileAndBLOBDestinations(t *testing.T) {
	for _, extension := range []string{"ged", "gdz"} {
		for _, format := range []string{"multi", "single"} {
			t.Run(extension+"/"+format, func(t *testing.T) {
				work := t.TempDir()
				fileBytes := []byte{1, 2, 3, 4}
				gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
					"0 @M1@ OBJE\n1 FILE blob-media-2.bin\n" +
					"0 @M2@ OBJE\n1 FORM PICT\n1 BLOB\n2 CONT ....\n0 TRLR\n"
				src := writeMediaTestArchive(t, work, extension, gedcom, "blob-media-2.bin", fileBytes)
				output := filepath.Join(work, "archive")
				mediaRoot := output
				if format == "single" {
					output += ".glx"
					mediaRoot = work
				}
				res := runGLX(t, work, "import", src, "-o", output, "--format", format)
				require.Equal(t, 0, res.exitCode, res.stderr)
				fileMedia := importedMediaEntity(t, output, format, "media-1")
				blobMedia := importedMediaEntity(t, output, format, "media-2")
				require.Equal(t, "media/files/blob-media-2.bin", fileMedia.URI)
				require.Equal(t, "media/files/blob-media-2-2.bin", blobMedia.URI)
				fileActual, err := os.ReadFile(filepath.Join(mediaRoot, filepath.FromSlash(fileMedia.URI)))
				require.NoError(t, err)
				require.Equal(t, fileBytes, fileActual)
				blobActual, err := os.ReadFile(filepath.Join(mediaRoot, filepath.FromSlash(blobMedia.URI)))
				require.NoError(t, err)
				require.Equal(t, []byte{0, 0, 0}, blobActual)
			})
		}
	}
}

func mediaTestJPEG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetRGBA(x, y, c)
		}
	}
	var encoded bytes.Buffer
	require.NoError(t, jpeg.Encode(&encoded, img, &jpeg.Options{Quality: 90}))

	return encoded.Bytes()
}

func writeMediaTestInput(t *testing.T, work, label, extension, name string, payload []byte) string {
	t.Helper()
	gedcom := "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n2 FORM LINEAGE-LINKED\n1 CHAR UTF-8\n" +
		"0 @M1@ OBJE\n1 FILE " + name + "\n0 TRLR\n"

	return writeMediaTestArchive(t, filepath.Join(work, label), extension, gedcom, name, payload)
}

func writeMediaTestArchive(t *testing.T, dir, extension, gedcom, name string, payload []byte) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	src := filepath.Join(dir, "input."+extension)
	if extension == "ged" {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), payload, 0o600))
		require.NoError(t, os.WriteFile(src, []byte(gedcom), 0o600))

		return src
	}
	var bundle bytes.Buffer
	writer := zip.NewWriter(&bundle)
	for _, item := range []struct {
		path     string
		contents []byte
	}{{"gedcom.ged", []byte(gedcom)}, {name, payload}} {
		entry, err := writer.Create(item.path)
		require.NoError(t, err)
		_, err = entry.Write(item.contents)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(src, bundle.Bytes(), 0o600))

	return src
}

func importedMediaEntity(t *testing.T, output, format, id string) *glxlib.Media {
	t.Helper()
	files := []string{output}
	if format == "multi" {
		var err error
		files, err = filepath.Glob(filepath.Join(output, "media", "*.glx"))
		require.NoError(t, err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		var fragment glxlib.GLXFile
		require.NoError(t, yaml.Unmarshal(data, &fragment))
		if media := fragment.Media[id]; media != nil {
			return media
		}
	}
	t.Fatalf("media %s missing from %s", id, output)

	return nil
}
