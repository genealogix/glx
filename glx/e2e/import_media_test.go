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
