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

// writeMultiPathArchive lays out an archive whose entities cross-reference
// each other across four directories without needing any vocabulary file:
// assertion -> person and assertion -> citation -> source.
func writeMultiPathArchive(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "persons", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	writeSkipTestFile(t, filepath.Join(root, "sources", "source-1.glx"),
		"sources:\n  source-1:\n    title: Parish register\n")
	writeSkipTestFile(t, filepath.Join(root, "citations", "citation-1.glx"),
		"citations:\n  citation-1:\n    source: source-1\n")
	writeSkipTestFile(t, filepath.Join(root, "assertions", "assertion-1.glx"),
		"assertions:\n  assertion-1:\n    subject:\n      person: person-1\n    property: occupation\n    value: smith\n    citations:\n      - citation-1\n")

	return root
}

func TestValidatePaths_MultipleDirectoriesFormOneArchive(t *testing.T) {
	root := writeMultiPathArchive(t)
	dirs := []string{
		filepath.Join(root, "persons"),
		filepath.Join(root, "sources"),
		filepath.Join(root, "citations"),
		filepath.Join(root, "assertions"),
	}

	t.Run("all directories together resolve every cross-reference", func(t *testing.T) {
		streams, out, _ := newTestStreams()
		require.NoError(t, validatePaths(streams, dirs))
		assert.Contains(t, out.String(), "Validated 4 files.")
	})

	t.Run("a directory after the first is really validated", func(t *testing.T) {
		// Leave out citations/ and sources/: the assertion in the last
		// argument now points at a citation that is not part of the load.
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{dirs[0], dirs[3]})
		require.Error(t, err)
		assert.Contains(t, errOut.String(), "citation-1")
	})

	t.Run("a broken file in a later directory fails structurally", func(t *testing.T) {
		writeSkipTestFile(t, filepath.Join(root, "sources", "broken.glx"), "sources: [\n")
		streams, _, _ := newTestStreams()
		err := validatePaths(streams, dirs)
		assert.ErrorIs(t, err, ErrStructuralValidationFailed)
	})
}

func TestValidatePaths_DirectoryAndFileArguments(t *testing.T) {
	root := writeMultiPathArchive(t)
	streams, out, _ := newTestStreams()

	err := validatePaths(streams, []string{
		filepath.Join(root, "persons"),
		filepath.Join(root, "sources"),
		filepath.Join(root, "citations"),
		filepath.Join(root, "assertions", "assertion-1.glx"),
	})

	require.NoError(t, err)
	assert.Contains(t, out.String(), "Validated 4 files.")
}

func TestCommonArchiveRoot(t *testing.T) {
	root := writeMultiPathArchive(t)

	got, err := commonArchiveRoot([]string{
		filepath.Join(root, "persons"),
		filepath.Join(root, "assertions", "assertion-1.glx"),
	})
	require.NoError(t, err)
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolvedGot, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, resolvedRoot, resolvedGot)

	if runtime.GOOS == goosWindows {
		return
	}
	_, err = commonArchiveRoot([]string{"/one/persons", "/two/events"})
	assert.ErrorIs(t, err, errNoCommonArchiveRoot, "only the filesystem root in common is not an archive")
}

func TestValidatePaths_SubsetIncludesArchiveVocabularies(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "persons", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n  person-2:\n    properties:\n      primary_name: Bob\n")
	writeSkipTestFile(t, filepath.Join(root, "relationships", "rel-1.glx"),
		"relationships:\n  rel-1:\n    type: marriage\n    participants:\n      - person: person-1\n      - person: person-2\n")
	persons := filepath.Join(root, "persons")
	relationships := filepath.Join(root, "relationships")

	t.Run("without vocabularies the type is undefined", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{persons, relationships})
		require.Error(t, err)
		assert.Contains(t, errOut.String(), "relationship_types")
	})

	writeSkipTestFile(t, filepath.Join(root, "vocabularies", "relationship-types.glx"),
		"relationship_types:\n  marriage:\n    label: Marriage\n")

	t.Run("vocabularies under the shared root are included automatically", func(t *testing.T) {
		streams, out, _ := newTestStreams()
		require.NoError(t, validatePaths(streams, []string{persons, relationships}))
		assert.Contains(t, out.String(), "Validated 3 files.")
	})
}

func TestValidatePaths_FileArgumentsFormOneArchive(t *testing.T) {
	root := writeMultiPathArchive(t)
	files := []string{
		filepath.Join(root, "persons", "person-1.glx"),
		filepath.Join(root, "sources", "source-1.glx"),
		filepath.Join(root, "citations", "citation-1.glx"),
		filepath.Join(root, "assertions", "assertion-1.glx"),
	}

	t.Run("cross-references between named files resolve", func(t *testing.T) {
		streams, out, _ := newTestStreams()
		require.NoError(t, validatePaths(streams, files))
		assert.Contains(t, out.String(), "Validated 4 files.")
	})

	t.Run("a broken reference between named files is reported", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{files[0], files[3]})
		require.Error(t, err)
		assert.Contains(t, errOut.String(), "citation-1")
	})

	t.Run("duplicate IDs across named files are reported", func(t *testing.T) {
		dup := filepath.Join(root, "persons", "person-1-again.glx")
		writeSkipTestFile(t, dup, "persons:\n  person-1:\n    properties:\n      primary_name: Alice again\n")
		streams, _, _ := newTestStreams()
		err := validatePaths(streams, []string{files[0], dup})
		assert.Error(t, err, "the same ID in two named files must be a duplicate")
	})
}

func TestValidatePaths_MultiPathErrors(t *testing.T) {
	root := writeMultiPathArchive(t)

	t.Run("a missing path fails structurally", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{filepath.Join(root, "persons"), filepath.Join(root, "nope.glx")})
		require.ErrorIs(t, err, ErrStructuralValidationFailed)
		assert.Contains(t, errOut.String(), "nope.glx")
	})

	if runtime.GOOS != goosWindows {
		t.Run("paths with only the filesystem root in common are rejected", func(t *testing.T) {
			streams, _, errOut := newTestStreams()
			err := validatePaths(streams, []string{filepath.Join(root, "persons"), "/"})
			require.ErrorIs(t, err, ErrStructuralValidationFailed)
			assert.Contains(t, errOut.String(), "do not share an archive root")
		})
	}
}

func TestWithArchiveVocabularies(t *testing.T) {
	root := t.TempDir()
	vocab := filepath.Join(root, "vocabularies")
	persons := filepath.Join(root, "persons")
	require.NoError(t, os.MkdirAll(vocab, 0o755))
	require.NoError(t, os.MkdirAll(persons, 0o755))

	assert.Equal(t, []string{persons, vocab}, withArchiveVocabularies(root, []string{persons}), "added when absent")
	assert.Equal(t, []string{persons, vocab}, withArchiveVocabularies(root, []string{persons, vocab}), "not duplicated when named")
	assert.Equal(t, []string{root}, withArchiveVocabularies(root, []string{root}), "not duplicated when a parent is named")
	assert.Equal(t, []string{persons}, withArchiveVocabularies(filepath.Join(root, "persons"), []string{persons}), "untouched when the root has no vocabularies/")
}

// Files named from inside one entity directory still belong to the archive
// above it: keys and the vocabulary lookup must be rooted there.
func TestValidatePaths_SameDirectoryFilesUseArchiveRoot(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "vocabularies", "event-types.glx"),
		"event_types:\n  land_grant:\n    label: Land Grant\n")
	writeSkipTestFile(t, filepath.Join(root, "persons", "person-1.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Alice\n")
	for _, name := range []string{"a", "b"} {
		writeSkipTestFile(t, filepath.Join(root, "events", name+".glx"),
			"events:\n  event-"+name+":\n    type: land_grant\n    participants:\n      - person: person-1\n")
	}

	got, err := commonArchiveRoot([]string{filepath.Join(root, "events", "a.glx"), filepath.Join(root, "events", "b.glx")})
	require.NoError(t, err)
	resolvedRoot, _ := filepath.EvalSymlinks(root)
	resolvedGot, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, resolvedRoot, resolvedGot, "the archive root is the parent of the entity directory")

	streams, out, _ := newTestStreams()
	require.NoError(t, validatePaths(streams, []string{
		filepath.Join(root, "events", "a.glx"),
		filepath.Join(root, "events", "b.glx"),
		filepath.Join(root, "persons", "person-1.glx"),
	}))
	assert.Contains(t, out.String(), "Validated 4 files.", "two events, the person, and the auto-included vocabulary")
}

func TestValidatePaths_IgnoresNonGLXFileArguments(t *testing.T) {
	root := writeMultiPathArchive(t)
	writeSkipTestFile(t, filepath.Join(root, "README.md"), "# notes\n")
	streams, out, _ := newTestStreams()

	require.NoError(t, validatePaths(streams, []string{
		filepath.Join(root, "persons"),
		filepath.Join(root, "sources"),
		filepath.Join(root, "citations"),
		filepath.Join(root, "assertions"),
		filepath.Join(root, "README.md"),
	}))
	assert.Contains(t, out.String(), "Validated 4 files.")
}

// The climb out of an entity directory is exactly one level: an archive whose
// own root is named events/ must not be climbed past.
func TestCommonArchiveRoot_ClimbsOneLevelOnly(t *testing.T) {
	data := t.TempDir()
	archive := filepath.Join(data, "events")
	writeSkipTestFile(t, filepath.Join(archive, "events", "a.glx"), "events: {}\n")
	writeSkipTestFile(t, filepath.Join(archive, "events", "b.glx"), "events: {}\n")

	got, err := commonArchiveRoot([]string{filepath.Join(archive, "events", "a.glx"), filepath.Join(archive, "events", "b.glx")})
	require.NoError(t, err)
	resolvedArchive, _ := filepath.EvalSymlinks(archive)
	resolvedGot, _ := filepath.EvalSymlinks(got)
	assert.Equal(t, resolvedArchive, resolvedGot)
}

// The full archive walk never descends into a directory symlink, so the
// automatic vocabularies/ inclusion must not follow one either.
func TestWithArchiveVocabularies_IgnoresSymlinkedVocabularies(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	root := t.TempDir()
	persons := filepath.Join(root, "persons")
	require.NoError(t, os.MkdirAll(persons, 0o755))
	writeSkipTestFile(t, filepath.Join(root, ".drafts", "vocabularies", "event-types.glx"), "event_types: {}\n")
	require.NoError(t, os.Symlink(filepath.Join(".drafts", "vocabularies"), filepath.Join(root, "vocabularies")))

	assert.Equal(t, []string{persons}, withArchiveVocabularies(root, []string{persons}))
}

func TestValidateMediaFileExistence_NormalizesNavigationSegments(t *testing.T) {
	root := t.TempDir()
	writeSkipTestFile(t, filepath.Join(root, "media", "files", "photo.jpg"), "jpeg")
	archive := &glxlib.GLXFile{
		Media: map[string]*glxlib.Media{
			"media-1": {URI: "media/files/../files/photo.jpg"},
		},
	}

	assert.Empty(t, validateMediaFileExistence(archive, root), "a URI that normalizes inside the archive is not a dot-prefixed path")
}

// Explicitly named files follow the archive-membership rules too: a link out
// of the archive is refused, and a link into a dot-prefixed directory is not
// archive content.
func TestValidatePaths_ExplicitFileArgumentsRespectContainment(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	outside := t.TempDir()
	writeSkipTestFile(t, filepath.Join(outside, "secret.glx"), "persons: {}\n")
	root := writeMultiPathArchive(t)
	escape := filepath.Join(root, "persons", "escape.glx")
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.glx"), escape))
	writeSkipTestFile(t, filepath.Join(root, ".worktrees", "copy", "persons", "dup.glx"),
		"persons:\n  person-1:\n    properties:\n      primary_name: Duplicate\n")
	excluded := filepath.Join(root, "persons", "alias.glx")
	require.NoError(t, os.Symlink(filepath.Join("..", ".worktrees", "copy", "persons", "dup.glx"), excluded))
	person := filepath.Join(root, "persons", "person-1.glx")

	t.Run("a named link out of the archive is refused", func(t *testing.T) {
		streams, _, errOut := newTestStreams()
		err := validatePaths(streams, []string{person, escape})
		require.ErrorIs(t, err, ErrStructuralValidationFailed)
		assert.Contains(t, errOut.String(), "escape.glx")
	})

	t.Run("a named link into a dot directory is not archive content", func(t *testing.T) {
		streams, out, _ := newTestStreams()
		require.NoError(t, validatePaths(streams, []string{person, excluded}), "the duplicate behind the link must not be loaded")
		assert.Contains(t, out.String(), "Validated 1 files.")
	})
}
