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
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An import that writes media waits while another holds the same media
// directory, so it cannot reuse a binary the other may still roll back
// (#1443). Two single-file archives side by side share that directory.
func TestImportGEDCOM_WaitsForMediaDirectoryLock(t *testing.T) {
	work := t.TempDir()
	source := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(source, "photo.jpg"), []byte("same bytes"), filePermissions))
	gedcom := filepath.Join(source, "tree.ged")
	const tree = "0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @O1@ OBJE\n1 FILE photo.jpg\n2 FORM jpg\n0 TRLR\n"
	require.NoError(t, os.WriteFile(gedcom, []byte(tree), filePermissions))

	release, err := lockImportMedia(work, &bytes.Buffer{}) // import A, mid-publication
	require.NoError(t, err)

	var out bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- importGEDCOM(gedcom, filepath.Join(work, "b.glx"), FormatSingle, false, false, 0, &out)
	}()

	select {
	case err := <-done:
		t.Fatalf("import B finished while A held the media directory: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	assert.NoFileExists(t, filepath.Join(work, "b.glx"), "B must not publish before A is done")

	release()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(30 * time.Second):
		t.Fatal("import B did not resume after the lock was released")
	}
	assert.Contains(t, out.String(), "Waiting for another import")
	assert.FileExists(t, filepath.Join(work, "b.glx"))
}

// Spellings of one directory share a lock; different directories do not.
func TestImportMediaLockPath(t *testing.T) {
	dir := t.TempDir()
	a, err := importMediaLockPath(dir)
	require.NoError(t, err)
	b, err := importMediaLockPath(filepath.Join(dir, ".", "sub", ".."))
	require.NoError(t, err)
	assert.Equal(t, a, b)
	other, err := importMediaLockPath(t.TempDir())
	require.NoError(t, err)
	assert.NotEqual(t, a, other)
}
