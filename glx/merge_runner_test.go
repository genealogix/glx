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
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

var stdoutCaptureMu sync.Mutex

func TestMergeArchives_NewEntities(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Events:        map[string]*glxlib.Event{},
		Relationships: map[string]*glxlib.Relationship{},
		Places:        map[string]*glxlib.Place{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Repositories:  map[string]*glxlib.Repository{},
		Assertions:    map[string]*glxlib.Assertion{},
		Media:         map[string]*glxlib.Media{},
	}

	src := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {Type: "birth", Date: "1850"},
		},
	}

	result := mergeArchivesInMemory(dest, src)

	assert.Empty(t, result.Conflicts, "no conflicts expected")
	assert.Equal(t, 1, result.NewPersons)
	assert.Equal(t, 1, result.NewEvents)
	assert.Len(t, dest.Persons, 2)
	assert.Contains(t, dest.Persons, "person-b")
}

func TestMergeArchives_Conflicts(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Events: map[string]*glxlib.Event{},
	}

	src := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Different A"}},
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}

	result := mergeArchivesInMemory(dest, src)

	require.Len(t, result.Conflicts, 1)
	assert.Contains(t, result.Conflicts[0], "person-a")
	assert.Equal(t, 1, result.NewPersons)
}

func TestMergeArchives_EmptySource(t *testing.T) {
	dest := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}

	src := &glxlib.GLXFile{}

	result := mergeArchivesInMemory(dest, src)

	assert.Empty(t, result.Conflicts)
	assert.Equal(t, 0, result.TotalNew())
	assert.Len(t, dest.Persons, 1)
}

func TestMergeArchives_DiskRoundTrip(t *testing.T) {
	// Create temp directories for source and destination archives
	destDir := t.TempDir()
	srcDir := t.TempDir()

	// Initialize dest with one person
	destSerializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})
	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := destSerializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	// Initialize src with a different person
	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := destSerializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	// Merge via CLI function
	err = mergeArchives(srcDir, destDir, false, 0.6)
	require.NoError(t, err)

	// Reload and verify no duplicates
	reloaded, dupes, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)
	assert.Empty(t, dupes, "reloaded archive should have no duplicates")
	assert.Len(t, reloaded.Persons, 2, "should have both persons after merge")
	assert.Contains(t, reloaded.Persons, "person-a")
	assert.Contains(t, reloaded.Persons, "person-b")
}

func TestMergeArchives_PreviewLeavesFilesystemUnchanged(t *testing.T) {
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	before := snapshotDir(t, destDir)
	require.NotEmpty(t, before, "destination should have files before merge")

	err = mergeArchives(srcDir, destDir, true, 0.6)
	require.NoError(t, err)

	after := snapshotDir(t, destDir)
	assert.Equal(t, before, after, "preview should not create, modify, or remove any files")
}

// snapshotDir returns a map of relative file paths to file sizes for all files
// under root. Used to detect any filesystem changes after a preview merge.
func snapshotDir(t *testing.T, root string) map[string]int64 {
	t.Helper()
	snapshot := make(map[string]int64)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(root, path)
			snapshot[rel] = info.Size()
		}
		return nil
	})
	require.NoError(t, err)
	return snapshot
}

func TestMergeArchives_DotDestination(t *testing.T) {
	// Create dest archive in temp dir
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	// Save original cwd, chdir into dest, merge with "."
	origDir, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { os.Chdir(origDir) })

	require.NoError(t, os.Chdir(destDir))
	err = mergeArchives(srcDir, ".", false, 0.6)
	require.NoError(t, err)

	// Verify merge result (use absolute destDir since cwd may have changed)
	reloaded, dupes, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)
	assert.Empty(t, dupes, "reloaded archive should have no duplicates")
	assert.Len(t, reloaded.Persons, 2, "should have both persons after merge")
	assert.Contains(t, reloaded.Persons, "person-a")
	assert.Contains(t, reloaded.Persons, "person-b")
}

func TestMergeArchives_PreviewShowsDuplicates(t *testing.T) {
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-dest-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-src-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	before := snapshotDir(t, destDir)

	// Capture stdout to verify duplicate detection output
	stdoutCaptureMu.Lock()
	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = w.Close()
		_ = r.Close()
		stdoutCaptureMu.Unlock()
	})

	err = mergeArchives(srcDir, destDir, true, 0.2)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)

	// Verify duplicate detection output
	assert.Contains(t, output, "Potential cross-archive duplicates")
	assert.Contains(t, output, "John Smith")
	assert.Contains(t, output, "Name similarity")
	assert.Contains(t, output, "(preview — no files written)")

	// Filesystem unchanged
	after := snapshotDir(t, destDir)
	assert.Equal(t, before, after, "preview should not modify any files")
}

func TestMergeArchives_CopiesSourceMediaBinariesAndPreservesDestBinaries(t *testing.T) {
	// Verifies the full fix for genealogix/glx#593:
	//  1. Source media binaries are copied into the destination.
	//  2. The destination's pre-existing binaries survive the safe-write swap.
	//  3. Name collisions with different content get a dedup rename + URI update.
	//  4. Name collisions with identical content reuse the dest binary.
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": "Person A"}},
		},
		Media: map[string]*glxlib.Media{
			"media-dest-photo": {URI: "media/files/photo.jpg", Title: "Dest photo"},
			"media-dest-deed":  {URI: "media/files/deed.pdf", Title: "Dest deed"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	// Pre-existing dest binaries (must survive the merge)
	destMediaFiles := filepath.Join(destDir, "media", "files")
	require.NoError(t, os.MkdirAll(destMediaFiles, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destMediaFiles, "photo.jpg"), []byte("DEST-PHOTO"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(destMediaFiles, "deed.pdf"), []byte("SHARED-DEED"), 0o644))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-b": {Properties: map[string]any{"name": "Person B"}},
		},
		Media: map[string]*glxlib.Media{
			// Collides with dest by name AND content (identical) — should reuse dest binary, no rename.
			"media-src-deed": {URI: "media/files/deed.pdf", Title: "Src deed"},
			// Collides with dest by name, DIFFERENT content — should rename and rewrite URI.
			"media-src-photo": {URI: "media/files/photo.jpg", Title: "Src photo"},
			// No collision — should copy as-is.
			"media-src-letter": {URI: "media/files/letter.txt", Title: "Src letter"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	srcMediaFiles := filepath.Join(srcDir, "media", "files")
	require.NoError(t, os.MkdirAll(srcMediaFiles, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcMediaFiles, "deed.pdf"), []byte("SHARED-DEED"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcMediaFiles, "photo.jpg"), []byte("SRC-PHOTO"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcMediaFiles, "letter.txt"), []byte("SRC-LETTER"), 0o644))

	require.NoError(t, mergeArchives(srcDir, destDir, false, 0.6))

	// Dest's own binaries must still be there
	got, err := os.ReadFile(filepath.Join(destMediaFiles, "photo.jpg"))
	require.NoError(t, err, "dest's photo.jpg must survive the merge")
	assert.Equal(t, "DEST-PHOTO", string(got), "dest's photo.jpg must not be clobbered")

	got, err = os.ReadFile(filepath.Join(destMediaFiles, "deed.pdf"))
	require.NoError(t, err, "dest's deed.pdf must survive the merge")
	assert.Equal(t, "SHARED-DEED", string(got))

	// Different-content collision → renamed to photo-2.jpg
	got, err = os.ReadFile(filepath.Join(destMediaFiles, "photo-2.jpg"))
	require.NoError(t, err, "source photo.jpg should be copied as photo-2.jpg after collision")
	assert.Equal(t, "SRC-PHOTO", string(got))

	// Unique source file copied as-is
	got, err = os.ReadFile(filepath.Join(destMediaFiles, "letter.txt"))
	require.NoError(t, err, "non-colliding source binary should be copied")
	assert.Equal(t, "SRC-LETTER", string(got))

	// Reload merged archive and verify URIs
	reloaded, _, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)

	require.Contains(t, reloaded.Media, "media-src-photo")
	assert.Equal(t, "media/files/photo-2.jpg", reloaded.Media["media-src-photo"].URI,
		"merged src photo URI must reflect the rename")

	require.Contains(t, reloaded.Media, "media-src-deed")
	assert.Equal(t, "media/files/deed.pdf", reloaded.Media["media-src-deed"].URI,
		"identical-content collision should leave URI unchanged")

	require.Contains(t, reloaded.Media, "media-src-letter")
	assert.Equal(t, "media/files/letter.txt", reloaded.Media["media-src-letter"].URI)

	// Dest's Media entities must still point at their original URIs
	assert.Equal(t, "media/files/photo.jpg", reloaded.Media["media-dest-photo"].URI)
	assert.Equal(t, "media/files/deed.pdf", reloaded.Media["media-dest-deed"].URI)
}

func TestMergeArchives_SkipsBinaryWhenMediaEntityConflicts(t *testing.T) {
	// When a source Media entity collides with a destination entity by ID,
	// Merge() keeps the destination version. The source binary then has no
	// referent in the merged archive and copying it would orphan the file.
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {Properties: map[string]any{"name": "A"}}},
		Media: map[string]*glxlib.Media{
			"media-shared": {URI: "media/files/dest-only.jpg", Title: "Dest version"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	files, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, files))
	require.NoError(t, os.MkdirAll(filepath.Join(destDir, "media", "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "media", "files", "dest-only.jpg"), []byte("DEST"), 0o644))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-b": {Properties: map[string]any{"name": "B"}}},
		Media: map[string]*glxlib.Media{
			// Same ID as dest — Merge() will keep dest's. URI here would
			// reference src's binary, which must NOT be copied.
			"media-shared": {URI: "media/files/src-only.jpg", Title: "Src version"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	files, err = serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, files))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "media", "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "media", "files", "src-only.jpg"), []byte("SRC"), 0o644))

	require.NoError(t, mergeArchives(srcDir, destDir, false, 0.6))

	// src-only.jpg must NOT have been copied — its Media entity didn't survive
	_, err = os.Stat(filepath.Join(destDir, "media", "files", "src-only.jpg"))
	assert.True(t, os.IsNotExist(err), "orphan binary must not be copied when its Media entity conflicts")

	// dest-only.jpg is still there
	got, err := os.ReadFile(filepath.Join(destDir, "media", "files", "dest-only.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "DEST", string(got))

	// Surviving Media entity is dest's version
	reloaded, _, err := LoadArchiveWithOptions(destDir, false)
	require.NoError(t, err)
	require.Contains(t, reloaded.Media, "media-shared")
	assert.Equal(t, "media/files/dest-only.jpg", reloaded.Media["media-shared"].URI)
}

func TestMergeArchives_PreviewReportsPlannedMediaCopies(t *testing.T) {
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {Properties: map[string]any{"name": "A"}}},
		Media: map[string]*glxlib.Media{
			"media-dest": {URI: "media/files/photo.jpg"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	files, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, files))

	destMediaFiles := filepath.Join(destDir, "media", "files")
	require.NoError(t, os.MkdirAll(destMediaFiles, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destMediaFiles, "photo.jpg"), []byte("DEST"), 0o644))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-b": {Properties: map[string]any{"name": "B"}}},
		Media: map[string]*glxlib.Media{
			"media-src-1": {URI: "media/files/photo.jpg"},
			"media-src-2": {URI: "media/files/letter.txt"},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	files, err = serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, files))

	srcMediaFiles := filepath.Join(srcDir, "media", "files")
	require.NoError(t, os.MkdirAll(srcMediaFiles, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcMediaFiles, "photo.jpg"), []byte("SRC-DIFFERENT"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcMediaFiles, "letter.txt"), []byte("LETTER"), 0o644))

	before := snapshotDir(t, destDir)

	cmd := exec.Command(os.Args[0], "-test.run=^TestMergeArchives_PreviewReportsPlannedMediaCopies_Helper$")
	cmd.Env = append(os.Environ(),
		"GLX_PREVIEW_HELPER=1",
		"GLX_SRC_DIR="+srcDir,
		"GLX_DEST_DIR="+destDir,
	)
	outputBytes, err := cmd.CombinedOutput()
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	assert.Contains(t, output, "Would copy 2 media file(s)")
	assert.Contains(t, output, "1 renamed to avoid name collisions")

	// Preview must not modify the destination
	after := snapshotDir(t, destDir)
	assert.Equal(t, before, after, "preview must not write any files")
}

func TestMergeArchives_PreviewReportsPlannedMediaCopies_Helper(t *testing.T) {
	if os.Getenv("GLX_PREVIEW_HELPER") != "1" {
		t.Skip("helper test only")
	}
	srcDir := os.Getenv("GLX_SRC_DIR")
	destDir := os.Getenv("GLX_DEST_DIR")
	require.NotEmpty(t, srcDir)
	require.NotEmpty(t, destDir)

	err := mergeArchives(srcDir, destDir, true, 0.6)
	require.NoError(t, err)
}

func TestMergeArchives_PreviewNoDuplicates(t *testing.T) {
	destDir := t.TempDir()
	srcDir := t.TempDir()

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})

	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-alice": {Properties: map[string]any{"name": "Alice Johnson"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	destFiles, err := serializer.SerializeMultiFileToMap(destArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(destDir, destFiles))

	srcArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-bob": {Properties: map[string]any{"name": "Bob Williams"}},
		},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	before := snapshotDir(t, destDir)

	// Capture preview output
	oldStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	os.Stdout = w

	err = mergeArchives(srcDir, destDir, true, 0.8)

	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "No potential cross-archive duplicates found")

	after := snapshotDir(t, destDir)
	assert.Equal(t, before, after, "preview must not write any files")
}
