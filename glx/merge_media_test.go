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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestMediaURIBasename(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{"flat media path", "media/files/photo.jpg", "photo.jpg"},
		{"empty URI", "", ""},
		{"http URL", "https://example.com/photo.jpg", ""},
		{"file scheme", "file:///abs/photo.jpg", ""},
		{"absolute path", "/abs/photo.jpg", ""},
		{"other dir", "other/dir/photo.jpg", ""},
		{"prefix-only", "media/files/", ""},
		{"nested under media/files", "media/files/2024/photo.jpg", ""},
		{"different prefix", "media/photo.jpg", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, mediaURIBasename(tt.uri))
		})
	}
}

func TestIndexMediaByBasename(t *testing.T) {
	media := map[string]*glxlib.Media{
		"m1":  {URI: "media/files/photo.jpg"},
		"m2":  {URI: "media/files/photo.jpg"}, // shared basename → both IDs
		"m3":  {URI: "media/files/letter.txt"},
		"m4":  {URI: "https://example.com/elsewhere.jpg"}, // external, skipped
		"m5":  {URI: ""},                                  // empty, skipped
		"nil": nil,                                        // nil entry, skipped
	}
	got := indexMediaByBasename(media)

	require.Len(t, got, 2, "should only index relative media/files entries")
	assert.ElementsMatch(t, []string{"m1", "m2"}, got["photo.jpg"])
	assert.Equal(t, []string{"m3"}, got["letter.txt"])
}

func TestNextAvailableMediaName(t *testing.T) {
	taken := map[string]struct{}{
		"photo.jpg":   {},
		"photo-2.jpg": {},
		"photo-3.jpg": {},
	}
	assert.Equal(t, "photo-4.jpg", nextAvailableMediaName("photo.jpg", taken))

	noExt := map[string]struct{}{"data": {}}
	assert.Equal(t, "data-2", nextAvailableMediaName("data", noExt))
}

func TestFilesIdentical(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.bin")
	b := filepath.Join(dir, "b.bin")
	c := filepath.Join(dir, "c.bin")

	require.NoError(t, os.WriteFile(a, []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(b, []byte("hello"), 0o644)) // identical content
	require.NoError(t, os.WriteFile(c, []byte("HELLO"), 0o644)) // same size, different bytes

	t.Run("identical content", func(t *testing.T) {
		same, err := filesIdentical(a, b)
		require.NoError(t, err)
		assert.True(t, same)
	})
	t.Run("different size short-circuits", func(t *testing.T) {
		short := filepath.Join(dir, "short.bin")
		require.NoError(t, os.WriteFile(short, []byte("hi"), 0o644))
		same, err := filesIdentical(a, short)
		require.NoError(t, err)
		assert.False(t, same)
	})
	t.Run("same size different bytes", func(t *testing.T) {
		same, err := filesIdentical(a, c)
		require.NoError(t, err)
		assert.False(t, same)
	})
	t.Run("missing left file returns error", func(t *testing.T) {
		_, err := filesIdentical(filepath.Join(dir, "nope.bin"), a)
		assert.Error(t, err)
	})
	t.Run("missing right file returns error", func(t *testing.T) {
		_, err := filesIdentical(a, filepath.Join(dir, "nope.bin"))
		assert.Error(t, err)
	})
}

func TestListMediaFilenames(t *testing.T) {
	t.Run("missing directory is empty", func(t *testing.T) {
		got, err := listMediaFilenames(filepath.Join(t.TempDir(), "does-not-exist"))
		require.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("returns regular files and skips subdirs", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.jpg"), []byte("a"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.pdf"), []byte("b"), 0o644))
		require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0o755))

		got, err := listMediaFilenames(dir)
		require.NoError(t, err)
		assert.Len(t, got, 2)
		assert.Contains(t, got, "a.jpg")
		assert.Contains(t, got, "b.pdf")
		assert.NotContains(t, got, "nested")
	})
}

func TestPlanSourceMediaCopies_NoMediaFilesDir(t *testing.T) {
	// Source with no media/files/ directory at all — common case for archives
	// that have Media entities referencing URLs but no local binaries.
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{
			"m1": {URI: "https://example.com/photo.jpg"},
		},
	}
	plan, err := planSourceMediaCopies(srcDir, destDir, src)
	require.NoError(t, err)
	assert.Empty(t, plan)
}

func TestPlanSourceMediaCopies_OrphanBinaryIsSkipped(t *testing.T) {
	// A binary on disk with no Media entity referencing it must not be copied.
	srcDir := t.TempDir()
	destDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "media", "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "media", "files", "orphan.jpg"), []byte("x"), 0o644))

	src := &glxlib.GLXFile{Media: map[string]*glxlib.Media{}}
	plan, err := planSourceMediaCopies(srcDir, destDir, src)
	require.NoError(t, err)
	assert.Empty(t, plan)
}

func TestPlanSourceMediaCopies_NestedSubdirSkipped(t *testing.T) {
	// Nested subdirectories under media/files/ aren't part of the GLX
	// convention; they must not show up in the plan.
	srcDir := t.TempDir()
	destDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "media", "files", "year-2024"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "media", "files", "year-2024", "photo.jpg"), []byte("x"), 0o644))

	src := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{
			// URI in nested form — also skipped by mediaURIBasename.
			"m1": {URI: "media/files/year-2024/photo.jpg"},
		},
	}
	plan, err := planSourceMediaCopies(srcDir, destDir, src)
	require.NoError(t, err)
	assert.Empty(t, plan)
}

func TestPlanSourceMediaCopies_NonDirAtMediaFilesPath(t *testing.T) {
	// If a regular file sits where media/files/ should be, the planner must
	// silently return nothing rather than try to walk it.
	srcDir := t.TempDir()
	destDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "media"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "media", "files"), []byte("not a dir"), 0o644))

	src := &glxlib.GLXFile{Media: map[string]*glxlib.Media{"m1": {URI: "media/files/photo.jpg"}}}
	plan, err := planSourceMediaCopies(srcDir, destDir, src)
	require.NoError(t, err)
	assert.Empty(t, plan)
}

func TestExecuteMediaCopies_ZeroLengthPlan(t *testing.T) {
	copied, skipped, err := executeMediaCopies(nil, t.TempDir(), &glxlib.GLXFile{})
	require.NoError(t, err)
	assert.Zero(t, copied)
	assert.Zero(t, skipped)
}

func TestExecuteMediaCopies_SkipsWhenAllReferencesDropped(t *testing.T) {
	// A planned binary whose every referencing src Media entity was dropped
	// by the merge (no pointer in merged.Media matches) must be skipped.
	destDir := t.TempDir()
	srcMediaDir := t.TempDir()
	srcBinary := filepath.Join(srcMediaDir, "ghost.jpg")
	require.NoError(t, os.WriteFile(srcBinary, []byte("GHOST"), 0o644))

	srcEntity := &glxlib.Media{URI: "media/files/ghost.jpg"}
	plan := []plannedMediaCopy{{
		SrcPath:    srcBinary,
		TargetName: "ghost.jpg",
		SrcMedia:   map[string]*glxlib.Media{"media-dropped": srcEntity},
	}}
	// merged has no surviving media-dropped entity.
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{}}

	copied, skipped, err := executeMediaCopies(plan, destDir, merged)
	require.NoError(t, err)
	assert.Zero(t, copied)
	assert.Equal(t, 1, skipped)

	// Binary must not have been copied
	_, statErr := os.Stat(filepath.Join(destDir, "media", "files", "ghost.jpg"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestExecuteMediaCopies_CopiesWhenAnyReferenceSurvives(t *testing.T) {
	// When several src Media entities point at the same binary and at least
	// one survives the merge (same pointer ends up in merged.Media), the
	// binary is copied once.
	destDir := t.TempDir()
	srcMediaDir := t.TempDir()
	srcBinary := filepath.Join(srcMediaDir, "shared.jpg")
	require.NoError(t, os.WriteFile(srcBinary, []byte("SHARED"), 0o644))

	droppedSrc := &glxlib.Media{URI: "media/files/shared.jpg"}
	survivorSrc := &glxlib.Media{URI: "media/files/shared.jpg"}
	plan := []plannedMediaCopy{{
		SrcPath:    srcBinary,
		TargetName: "shared.jpg",
		SrcMedia: map[string]*glxlib.Media{
			"media-dropped":  droppedSrc,
			"media-survivor": survivorSrc,
		},
	}}
	// Survivor's src pointer is what merged.Media holds.
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
		"media-survivor": survivorSrc,
	}}

	copied, skipped, err := executeMediaCopies(plan, destDir, merged)
	require.NoError(t, err)
	assert.Equal(t, 1, copied)
	assert.Zero(t, skipped)

	got, err := os.ReadFile(filepath.Join(destDir, "media", "files", "shared.jpg"))
	require.NoError(t, err)
	assert.Equal(t, "SHARED", string(got))
}

func TestExecuteMediaCopies_SkipsWhenSameIDButDestEntityKept(t *testing.T) {
	// Conflict scenario: src and dest both define media-shared. Merge keeps
	// dest's. merged.Media[media-shared] is dest's pointer, not src's — so
	// pointer-check correctly skips the copy even though the URI happens to
	// match the planned target. Copying would import src's binary under
	// dest's preserved metadata, which would misattribute it.
	destDir := t.TempDir()
	srcMediaDir := t.TempDir()
	srcBinary := filepath.Join(srcMediaDir, "photo.jpg")
	require.NoError(t, os.WriteFile(srcBinary, []byte("SRC"), 0o644))

	srcEntity := &glxlib.Media{URI: "media/files/photo.jpg"}
	destEntity := &glxlib.Media{URI: "media/files/photo.jpg"} // same URI by coincidence

	plan := []plannedMediaCopy{{
		SrcPath:    srcBinary,
		TargetName: "photo.jpg",
		SrcMedia:   map[string]*glxlib.Media{"media-shared": srcEntity},
	}}
	// merged kept dest's entity (different pointer).
	merged := &glxlib.GLXFile{Media: map[string]*glxlib.Media{
		"media-shared": destEntity,
	}}

	copied, skipped, err := executeMediaCopies(plan, destDir, merged)
	require.NoError(t, err)
	assert.Zero(t, copied)
	assert.Equal(t, 1, skipped)

	_, statErr := os.Stat(filepath.Join(destDir, "media", "files", "photo.jpg"))
	assert.True(t, os.IsNotExist(statErr))
}

func TestMergeArchives_SingleFileDestWarnsAboutSourceBinaries(t *testing.T) {
	// Single-file destinations can't hold media/files binaries. The merge
	// must still complete, and the runner must print a stderr warning so the
	// user knows their merged Media URIs will dangle.
	srcDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "media", "files"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "media", "files", "photo.jpg"), []byte("SRC"), 0o644))

	serializer := glxlib.NewSerializer(&glxlib.SerializerOptions{Validate: false, Pretty: true})
	srcArchive := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{"media-src": {URI: "media/files/photo.jpg"}},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(srcArchive)
	srcFiles, err := serializer.SerializeMultiFileToMap(srcArchive)
	require.NoError(t, err)
	require.NoError(t, writeFilesToDir(srcDir, srcFiles))

	// Single-file destination archive.
	destDir := t.TempDir()
	destFile := filepath.Join(destDir, "dest.glx")
	destArchive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {Properties: map[string]any{"name": "A"}}},
	}
	glxlib.LoadStandardVocabulariesIntoGLX(destArchive)
	data, err := serializer.SerializeSingleFileBytes(destArchive)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(destFile, data, 0o644))

	// Capture stderr to verify the warning.
	oldStderr := os.Stderr
	r, w, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	os.Stderr = w

	mergeErr := mergeArchives(srcDir, destFile, false, 0.6)

	_ = w.Close()
	os.Stderr = oldStderr
	stderrBytes, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	stderr := string(stderrBytes)

	require.NoError(t, mergeErr)
	assert.Contains(t, stderr, "single-file destination", "expected dangling-binary warning")
	assert.Contains(t, stderr, "media/files")
}
