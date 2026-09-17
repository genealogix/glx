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
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func preserveTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"primary_name": "Alice"}},
		},
	}
}

// A dot entry inside media/files/ must not break the safe-write swap. The
// dot-entry preservation walk used to run first and recreate media/files/ in
// the destination to hold .keep, so the whole-directory rename of media/files/
// that followed failed on an existing target and the write errored out.
func TestSafeWrite_PreservesDotEntryUnderMediaFiles(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	writeSkipTestFile(t, filepath.Join(archiveDir, "media", "files", ".keep"), "")
	writeSkipTestFile(t, filepath.Join(archiveDir, "media", "files", "portrait.jpg"), "jpeg bytes")

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, filepath.Join(archiveDir, "media", "files", ".keep"))
	assert.FileExists(t, filepath.Join(archiveDir, "media", "files", "portrait.jpg"))
	assert.FileExists(t, filepath.Join(archiveDir, "persons", "person-1.glx"))
}

// Dot-prefixed entries nested inside a managed directory are skipped by the
// loader, so the serializer never re-emits them; the swap must carry them over.
func TestSafeWrite_PreservesNestedDotEntries(t *testing.T) {
	archiveDir := filepath.Join(t.TempDir(), "archive")
	require.NoError(t, os.MkdirAll(archiveDir, 0o755))
	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))
	draft := filepath.Join(archiveDir, "persons", ".drafts", "draft.glx")
	sidecar := filepath.Join(archiveDir, "persons", "._person-1.glx")
	writeSkipTestFile(t, draft, "persons: {}")
	writeSkipTestFile(t, sidecar, "not glx")

	require.NoError(t, safeWriteMultiFileArchive(archiveDir, preserveTestArchive()))

	assert.FileExists(t, draft)
	assert.FileExists(t, sidecar)
	assert.FileExists(t, filepath.Join(archiveDir, "persons", "person-1.glx"))
}

func TestFirstNestedDotEntry(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "person-1.glx"), "persons: {}")

	found, err := firstNestedDotEntry(dir)
	require.NoError(t, err)
	assert.Empty(t, found)

	writeSkipTestFile(t, filepath.Join(dir, "persons", ".drafts", "x.glx"), "persons: {}")

	found, err = firstNestedDotEntry(dir)
	require.NoError(t, err)
	assert.Equal(t, "persons/.drafts", found)
}

// The loader follows an ordinary symlink and reads its target, so the cache
// fingerprint has to describe the target too. Recording the link's own
// metadata let edits to the target leave the fingerprint unchanged and the
// cache serve stale entities.
func TestComputeFSFingerprint_FollowsSymlinkTarget(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	root := t.TempDir()
	target := filepath.Join(root, "data", "current.txt")
	writeSkipTestFile(t, target, "persons: {}")
	link := filepath.Join(root, "persons", "current.glx")
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	require.NoError(t, os.Symlink(filepath.Join("..", "data", "current.txt"), link))

	before, err := computeFSFingerprint(root)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(target, []byte("persons: {}\n# edited through the target\n"), 0o644))

	after, err := computeFSFingerprint(root)
	require.NoError(t, err)
	assert.NotEqual(t, before, after, "editing a symlink's target must change the fingerprint")
}

func TestValidateSingleFileSemantics_SkipsDotEntries(t *testing.T) {
	dir := t.TempDir()
	writeSkipTestFile(t, filepath.Join(dir, "persons", "good.glx"), "persons: {}\n")
	hidden := filepath.Join(dir, "persons", ".hidden.glx")
	writeSkipTestFile(t, hidden, "persons: [\n")
	writeSkipTestFile(t, filepath.Join(dir, "persons", ".drafts", "bad.glx"), "persons: [\n")

	errs, _ := validateSingleFileSemantics([]string{dir})
	assert.Empty(t, errs, "dot-prefixed entries discovered by the walk are skipped")

	errs, _ = validateSingleFileSemantics([]string{hidden})
	assert.NotEmpty(t, errs, "a dot-prefixed file named explicitly is still validated")
}

func TestValidateMediaFileExistence_DotComponentWarns(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, ".stash", "x.jpg"), "jpeg")
	archive := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{
			"media-1": {URI: ".stash/x.jpg"},
		},
	}

	warnings := validateMediaFileExistence(archive, root)

	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "dot-prefixed path")
	assert.Contains(t, warnings[0], "media-1")
}

func TestValidateAndReport(t *testing.T) {
	t.Run("rejects more than one path", func(t *testing.T) {
		streams, _, _ := newTestStreams()
		err := validateAndReport(streams, []string{"a", "b"})
		assert.ErrorIs(t, err, errReportTooManyArgs)
	})

	t.Run("validation gates the report", func(t *testing.T) {
		dir := t.TempDir()
		writeSkipTestFile(t, filepath.Join(dir, "persons", "bad.glx"), "persons: [\n")
		streams, _, _ := newTestStreams()
		err := validateAndReport(streams, []string{dir})
		assert.Error(t, err, "an archive that fails validation must not produce a report")
	})
}
