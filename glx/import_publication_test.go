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
