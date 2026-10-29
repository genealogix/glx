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

func TestMediaFreeImportIgnoresUnrelatedMediaPaths(t *testing.T) {
	for _, obstruction := range []string{"media-file", "MEDIA-file", "media-link", "MEDIA-link", "media/files-file", "MEDIA/FILES-file", "media/files-link", "ambiguous"} {
		for _, entry := range []string{"allocation", "GEDCOM/single", "GEDCOM/multi", "GEDZIP/single", "GEDZIP/multi"} {
			t.Run(obstruction+"/"+entry, func(t *testing.T) {
				dir := t.TempDir()
				name := "media"
				switch obstruction {
				case "MEDIA-file", "MEDIA-link":
					name = "MEDIA"
				case "media/files-file", "media/files-link":
					name = "media/files"
				case "MEDIA/FILES-file":
					name = "MEDIA/FILES"
				}
				target := filepath.Join(dir, filepath.FromSlash(name))
				require.NoError(t, os.MkdirAll(filepath.Dir(target), dirPermissions))
				switch obstruction {
				case "ambiguous":
					if !fsCaseSensitive(t, dir) {
						t.Skip("requires distinct directory spellings")
					}
					require.NoError(t, os.MkdirAll(filepath.Join(dir, "media/files"), dirPermissions))
					require.NoError(t, os.MkdirAll(filepath.Join(dir, "MEDIA/files"), dirPermissions))
				case "media-link", "MEDIA-link", "media/files-link":
					external := t.TempDir()
					require.NoError(t, os.WriteFile(filepath.Join(external, "keep"), []byte("external"), filePermissions))
					if err := os.Symlink(external, target); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				default:
					require.NoError(t, os.WriteFile(target, []byte("unrelated"), filePermissions))
				}
				before, err := os.Lstat(target)
				require.NoError(t, err)
				archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"person-test": {}}}
				switch entry {
				case "allocation":
					err = allocateImportMediaDestinations(nil, nil, dir, "", nil)
				case "GEDCOM/single":
					err = importToSingleFile(archive, filepath.Join(dir, "out.glx"), false, false, 0, nil, "", io.Discard)
				case "GEDCOM/multi":
					err = importToMultiFile(archive, dir, false, false, 0, nil, "", io.Discard)
				case "GEDZIP/single":
					err = importGEDZIPToSingleFile(archive, filepath.Join(dir, "out.glx"), false, false, 0, nil, nil, io.Discard)
				case "GEDZIP/multi":
					err = importGEDZIPToMultiFile(archive, dir, false, false, 0, nil, nil, io.Discard)
				}
				require.NoError(t, err)
				after, err := os.Lstat(target)
				require.NoError(t, err)
				require.True(t, os.SameFile(before, after), "unrelated media entry must remain in place")
				if before.Mode().IsRegular() {
					require.Equal(t, []byte("unrelated"), mustReadImportMedia(t, target))
				}
			})
		}
	}
}

func TestMediaFreeImportStillChecksRequiredMetadataDirectory(t *testing.T) {
	for _, gedzip := range []bool{false, true} {
		t.Run(map[bool]string{false: "GEDCOM", true: "GEDZIP"}[gedzip], func(t *testing.T) {
			dir := t.TempDir()
			obstruction := filepath.Join(dir, "media")
			require.NoError(t, os.WriteFile(obstruction, []byte("keep"), filePermissions))
			archive := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"remote": {URI: "https://example.com/photo.jpg"}}}
			var err error
			if gedzip {
				err = importGEDZIPToMultiFile(archive, dir, false, false, 0, nil, nil, io.Discard)
			} else {
				err = importToMultiFile(archive, dir, false, false, 0, nil, "", io.Discard)
			}
			require.Error(t, err)
			require.Equal(t, []byte("keep"), mustReadImportMedia(t, obstruction))
		})
	}
}
