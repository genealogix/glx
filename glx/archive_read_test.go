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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const singleFileArchiveBody = "persons:\n  person-a:\n    properties:\n      primary_name: \"Anna\"\n"

// writeTestSingleFile writes a one-person single-file archive at path.
func writeTestSingleFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(singleFileArchiveBody), 0o644))
}

func TestLoadArchiveForRead_MissingPath(t *testing.T) {
	streams := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: io.Discard}
	_, err := loadArchiveForRead(streams, filepath.Join(t.TempDir(), "nope"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot access path")
}

func TestLoadArchiveForRead_DirectoryWarnsOnDuplicates(t *testing.T) {
	t.Setenv("GLX_CACHE", "off")
	dir := t.TempDir()
	writePersonFile(t, dir, "person-a", "Anna")
	// A second file defining the same ID produces a duplicate-ID warning.
	dup := "persons:\n  person-a:\n    properties:\n      primary_name: \"Anna Again\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "zz-dup.glx"), []byte(dup), 0o644))

	var errOut bytes.Buffer
	streams := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: &errOut}
	archive, err := loadArchiveForRead(streams, dir)
	require.NoError(t, err)
	require.Contains(t, archive.Persons, "person-a")
	assert.Contains(t, errOut.String(), "Warning: ")
	assert.Contains(t, errOut.String(), "person-a")
}

func TestLoadArchiveForRead_DirectoryLoadError(t *testing.T) {
	t.Setenv("GLX_CACHE", "off")
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "bad.glx"), []byte("persons: [unclosed\n"), 0o644))

	streams := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: io.Discard}
	_, err := loadArchiveForRead(streams, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load archive")
}

func TestLoadArchiveForRead_SingleFileMergesVocabularies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "family.glx")
	writeTestSingleFile(t, path)

	streams := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: io.Discard}
	archive, err := loadArchiveForRead(streams, path)
	require.NoError(t, err)
	require.Contains(t, archive.Persons, "person-a")
	assert.NotEmpty(t, archive.EventTypes, "standard vocabularies should be merged into a single-file archive")
}

func TestLoadArchiveForRead_SingleFileReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.glx")
	require.NoError(t, os.WriteFile(path, []byte("persons: [unclosed\n"), 0o644))

	streams := &IOStreams{Out: io.Discard, MachineOut: io.Discard, ErrOut: io.Discard}
	_, err := loadArchiveForRead(streams, path)
	require.Error(t, err)
}

func TestRefuseSingleFileArchive(t *testing.T) {
	t.Run("missing path is left to the loader", func(t *testing.T) {
		assert.NoError(t, refuseSingleFileArchive("add", filepath.Join(t.TempDir(), "nope")))
	})

	t.Run("glx file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "family.GLX")
		writeTestSingleFile(t, path)
		err := refuseSingleFileArchive("add", path)
		require.ErrorIs(t, err, ErrSingleFileArchiveUnsupported)
		assert.Contains(t, err.Error(), "glx add")
		assert.Contains(t, err.Error(), "glx split "+path)
	})

	t.Run("directory holding one glx file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "family.glx")
		writeTestSingleFile(t, path)
		err := refuseSingleFileArchive("link", dir)
		require.ErrorIs(t, err, ErrSingleFileArchiveUnsupported)
		assert.Contains(t, err.Error(), path)
	})

	t.Run("non-glx file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "notes.txt")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
		assert.NoError(t, refuseSingleFileArchive("add", path))
	})

	t.Run("multi-file archive", func(t *testing.T) {
		dir := t.TempDir()
		writePersonFile(t, dir, "person-a", "Anna")
		assert.NoError(t, refuseSingleFileArchive("add", dir))
	})
}

func TestSingleFileArchiveInDir(t *testing.T) {
	t.Run("vocabularies directory means multi-file layout", func(t *testing.T) {
		dir := t.TempDir()
		writeTestSingleFile(t, filepath.Join(dir, "family.glx"))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "vocabularies"), 0o755))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("two glx files are ambiguous", func(t *testing.T) {
		dir := t.TempDir()
		writeTestSingleFile(t, filepath.Join(dir, "a.glx"))
		writeTestSingleFile(t, filepath.Join(dir, "b.glx"))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("no glx file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.md"), []byte("x"), 0o644))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("a glx directory is not a file", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "odd.glx"), 0o755))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("unreadable directory", func(t *testing.T) {
		assert.Empty(t, singleFileArchiveInDir(filepath.Join(t.TempDir(), "nope")))
	})

	t.Run("one glx file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "family.glx")
		writeTestSingleFile(t, path)
		assert.Equal(t, path, singleFileArchiveInDir(dir))
	})

	t.Run("upper-case layout directory means multi-file layout", func(t *testing.T) {
		dir := t.TempDir()
		writeTestSingleFile(t, filepath.Join(dir, "family.glx"))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "PERSONS"), 0o755))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("media holding only the binary store is still single-file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "family.glx")
		writeTestSingleFile(t, path)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "media", "files"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "media", "files", "photo.jpg"), []byte("x"), 0o644))
		assert.Equal(t, path, singleFileArchiveInDir(dir))
	})

	t.Run("media holding entity files means multi-file layout", func(t *testing.T) {
		dir := t.TempDir()
		writeTestSingleFile(t, filepath.Join(dir, "family.glx"))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "media", "files"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "media", "media-a.glx"), []byte("media: {}\n"), 0o644))
		assert.Empty(t, singleFileArchiveInDir(dir))
	})

	t.Run("dot-prefixed sidecar is not counted", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "family.glx")
		writeTestSingleFile(t, path)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "._family.glx"), []byte{0, 5, 22, 7}, 0o644))
		assert.Equal(t, path, singleFileArchiveInDir(dir))
	})

	t.Run("contained relative symlink is counted", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.Mkdir(filepath.Join(dir, "data"), 0o755))
		writeTestSingleFile(t, filepath.Join(dir, "data", "archive.yaml"))
		link := filepath.Join(dir, "archive.glx")
		if err := os.Symlink(filepath.Join("data", "archive.yaml"), link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		assert.Equal(t, link, singleFileArchiveInDir(dir))
	})

	t.Run("symlink escaping the archive is not counted", func(t *testing.T) {
		outside := filepath.Join(t.TempDir(), "elsewhere.glx")
		writeTestSingleFile(t, outside)
		dir := t.TempDir()
		path := filepath.Join(dir, "family.glx")
		writeTestSingleFile(t, path)
		if err := os.Symlink(outside, filepath.Join(dir, "other.glx")); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		assert.Equal(t, path, singleFileArchiveInDir(dir))
	})
}

func TestArchivePathHint(t *testing.T) {
	t.Run("missing path", func(t *testing.T) {
		assert.Empty(t, archivePathHint("timeline", filepath.Join(t.TempDir(), "nope")))
	})

	t.Run("plain file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "notes.txt")
		require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
		assert.Empty(t, archivePathHint("timeline", path))
	})

	t.Run("directory", func(t *testing.T) {
		dir := t.TempDir()
		hint := archivePathHint("timeline", dir)
		assert.True(t, strings.HasPrefix(hint, "\nHint: "))
		assert.Contains(t, hint, "glx timeline --archive "+dir)
	})

	t.Run("glx file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "family.glx")
		writeTestSingleFile(t, path)
		assert.Contains(t, archivePathHint("summary", path), "glx summary --archive "+path)
	})
}

// census add pointed at the .glx file itself gets the single-file refusal
// with the glx split guidance, not only "not a directory".
func TestLoadArchiveForCensus_SingleFilePathRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "family.glx")
	writeTestSingleFile(t, path)

	_, err := loadArchiveForCensus(path)
	require.ErrorIs(t, err, ErrSingleFileArchiveUnsupported)
	assert.Contains(t, err.Error(), "glx census add")
	assert.Contains(t, err.Error(), "glx split "+path)
}
