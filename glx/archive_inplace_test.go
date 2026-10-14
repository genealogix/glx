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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// stageInPlaceArchive writes files (slash-separated relative path -> content)
// under a fresh archive directory and returns it.
func stageInPlaceArchive(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "archive")
	for relPath, content := range files {
		writeSkipTestFile(t, filepath.Join(dir, filepath.FromSlash(relPath)), content)
	}

	return dir
}

// loadForInPlace loads the archive the way merge, merge-persons, and migrate do.
func loadForInPlace(t *testing.T, dir string) *glxlib.GLXFile {
	t.Helper()
	archive, _, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)

	return archive
}

func readArchiveFile(t *testing.T, dir, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(relPath)))
	require.NoError(t, err)

	return string(data)
}

const familyGLX = `# Hand-kept family file.
persons:
  person-a:
    properties:
      primary_name: Alice
  person-b:
    properties:
      primary_name: Alicia
`

// The layout the user chose survives a write: a change lands in the file the
// entity came from, no canonical directory appears, and a file the write does
// not touch keeps its bytes, comments included (#1247).
func TestWriteArchiveInPlace_KeepsHybridLayout(t *testing.T) {
	untouched := "# Census notes.\nsources:\n  source-1850:\n    title: 1850 Census\n"
	dir := stageInPlaceArchive(t, map[string]string{
		"people/family.glx":        familyGLX,
		"sources/census/1850.glx":  untouched,
		"vocabularies/events.glx":  "event_types:\n  barmitzvah:\n    label: Bar Mitzvah\n",
		"notes/research-plan.md":   "not archive content\n",
		"media/files/portrait.jpg": "jpeg bytes",
	})
	archive := loadForInPlace(t, dir)
	delete(archive.Persons, "person-b")
	archive.Persons["person-a"].Properties["primary_name"] = "Alice Smith"

	require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

	family := readArchiveFile(t, dir, "people/family.glx")
	assert.Contains(t, family, "Alice Smith")
	assert.NotContains(t, family, "person-b")
	assert.Equal(t, untouched, readArchiveFile(t, dir, "sources/census/1850.glx"))
	assert.Contains(t, readArchiveFile(t, dir, "vocabularies/events.glx"), "barmitzvah", "a custom vocabulary survives")
	assert.NoDirExists(t, filepath.Join(dir, "persons"))
	assert.FileExists(t, filepath.Join(dir, "notes", "research-plan.md"))
	assert.FileExists(t, filepath.Join(dir, "media", "files", "portrait.jpg"))

	reloaded, duplicates, err := LoadArchiveWithOptions(dir, false)
	require.NoError(t, err)
	assert.Empty(t, duplicates)
	assert.Len(t, reloaded.Persons, 1)
}

// An unchanged archive is not written at all.
func TestWriteArchiveInPlace_NoChangeWritesNothing(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})

	require.NoError(t, writeArchiveInPlace(dir, loadForInPlace(t, dir), rewriteChanged))

	assert.Equal(t, familyGLX, readArchiveFile(t, dir, "people/family.glx"))
}

// A file whose last entity is removed is deleted rather than left empty.
func TestWriteArchiveInPlace_DeletesEmptiedFile(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{
		"persons/person-a.glx": "persons:\n  person-a: {}\n",
		"persons/person-b.glx": "persons:\n  person-b: {}\n",
	})
	archive := loadForInPlace(t, dir)
	delete(archive.Persons, "person-b")

	require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

	assert.NoFileExists(t, filepath.Join(dir, "persons", "person-b.glx"))
	assert.FileExists(t, filepath.Join(dir, "persons", "person-a.glx"))
}

// A new entity is written where the multi-file serializer would put it.
func TestWriteArchiveInPlace_NewEntityGoesToCanonicalPath(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive := loadForInPlace(t, dir)
	archive.Events["Event-Birth-A"] = &glxlib.Event{Type: "birth"}

	require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

	assert.Contains(t, readArchiveFile(t, dir, "events/event-birth-a.glx"), "Event-Birth-A")
	assert.Equal(t, familyGLX, readArchiveFile(t, dir, "people/family.glx"))
}

// A file already at a new entity's canonical path, in any letter case, takes
// the entity in rather than being overwritten.
func TestWriteArchiveInPlace_NewEntityJoinsFileAtCanonicalPath(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{
		"Persons/Person-New.glx": "persons:\n  person-other:\n    properties:\n      primary_name: Other\n",
	})
	archive := loadForInPlace(t, dir)
	archive.Persons["person-new"] = &glxlib.Person{Properties: map[string]any{"primary_name": "New"}}

	require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

	joined := readArchiveFile(t, dir, "Persons/Person-New.glx")
	assert.Contains(t, joined, "person-other")
	assert.Contains(t, joined, "person-new")
	reloaded := loadForInPlace(t, dir)
	assert.Len(t, reloaded.Persons, 2)
}

// Two IDs of one type that differ only by case would share a canonical file.
func TestWriteArchiveInPlace_RefusesCaseOnlyCollisionWithNewEntity(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive := loadForInPlace(t, dir)
	archive.Persons["Person-A"] = &glxlib.Person{}

	err := writeArchiveInPlace(dir, archive, rewriteChanged)

	require.ErrorIs(t, err, glxlib.ErrCaseInsensitiveCollision)
	assert.Equal(t, familyGLX, readArchiveFile(t, dir, "people/family.glx"))
}

// When two files define the same ID the loader keeps the first in sorted
// path order. An update goes to that copy only; a removal goes to every copy,
// or the next load would bring the entity back from the other one.
func TestWriteArchiveInPlace_DuplicateDefinitions(t *testing.T) {
	shadow := "persons:\n  person-a:\n    properties:\n      primary_name: Shadow\n"
	files := map[string]string{
		"a/first.glx":  "persons:\n  person-a:\n    properties:\n      primary_name: First\n",
		"b/second.glx": shadow,
	}

	t.Run("update goes to the copy the loader used", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)
		archive := loadForInPlace(t, dir)
		archive.Persons["person-a"].Properties["primary_name"] = "Updated"

		require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, "a/first.glx"), "Updated")
		assert.Equal(t, shadow, readArchiveFile(t, dir, "b/second.glx"))
	})

	t.Run("removal reaches every copy", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)
		archive := loadForInPlace(t, dir)
		delete(archive.Persons, "person-a")

		require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

		assert.NoFileExists(t, filepath.Join(dir, "a", "first.glx"))
		assert.NoFileExists(t, filepath.Join(dir, "b", "second.glx"))
	})
}

// A vocabulary entry added to a vocabulary the archive defines goes into the
// first file that defines it.
func TestWriteArchiveInPlace_NewVocabularyEntryJoinsDefiningFile(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{
		"vocab/events.glx": "event_types:\n  barmitzvah:\n    label: Bar Mitzvah\n",
	})
	archive := loadForInPlace(t, dir)
	archive.EventTypes["confirmation"] = &glxlib.VocabularyEntry{Label: "Confirmation"}

	require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

	events := readArchiveFile(t, dir, "vocab/events.glx")
	assert.Contains(t, events, "barmitzvah")
	assert.Contains(t, events, "confirmation")
	assert.NoDirExists(t, filepath.Join(dir, "vocabularies"))
}

// The loader fills a vocabulary no file defines with the standard entries.
// Leaving it that way writes nothing; adding an entry writes the whole map,
// because a file holding only the new entry would switch that fill off on the
// next load and drop every standard entry.
func TestWriteArchiveInPlace_StandardFilledVocabulary(t *testing.T) {
	t.Run("untouched defaults are not written", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})

		require.NoError(t, writeArchiveInPlace(dir, loadForInPlace(t, dir), rewriteChanged))

		assert.NoDirExists(t, filepath.Join(dir, "vocabularies"))
	})

	t.Run("an added entry writes the full vocabulary", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
		archive := loadForInPlace(t, dir)
		require.Contains(t, archive.EventTypes, "birth", "the loader supplies the standard event types")
		standardCount := len(archive.EventTypes)
		archive.EventTypes["barmitzvah"] = &glxlib.VocabularyEntry{Label: "Bar Mitzvah"}

		require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

		reloaded := loadForInPlace(t, dir)
		assert.Contains(t, reloaded.EventTypes, "barmitzvah")
		assert.Contains(t, reloaded.EventTypes, "birth")
		assert.Len(t, reloaded.EventTypes, standardCount+1)
		assert.FileExists(t, filepath.Join(dir, "vocabularies", "event-types.glx"))
	})
}

// Metadata is updated where it lives, and a block the command adds goes to
// metadata.glx.
func TestWriteArchiveInPlace_Metadata(t *testing.T) {
	t.Run("added", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
		archive := loadForInPlace(t, dir)
		archive.ImportMetadata = &glxlib.Metadata{SourceSystem: "test"}

		require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, archiveMetadataFile), "test")
	})

	t.Run("updated where it lives", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{
			"people/family.glx": "metadata:\n  source_system: old\n" + familyGLX,
		})
		archive := loadForInPlace(t, dir)
		archive.ImportMetadata.SourceSystem = "new"

		require.NoError(t, writeArchiveInPlace(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, "people/family.glx"), "new")
		assert.NoFileExists(t, filepath.Join(dir, archiveMetadataFile))
	})
}

// rewriteNormalized persists a load-time shim's fold, which the typed model
// cannot see as a change; rewriteChanged leaves the file alone.
func TestWriteArchiveInPlace_NormalizedPolicyPersistsShimFolds(t *testing.T) {
	legacy := "media:\n  media-1:\n    uri: files/a.jpg\n    description: A portrait\n"

	dir := stageInPlaceArchive(t, map[string]string{"media/media-1.glx": legacy})
	require.NoError(t, writeArchiveInPlace(dir, loadForInPlace(t, dir), rewriteChanged))
	assert.Equal(t, legacy, readArchiveFile(t, dir, "media/media-1.glx"))

	require.NoError(t, writeArchiveInPlace(dir, loadForInPlace(t, dir), rewriteNormalized))
	migrated := readArchiveFile(t, dir, "media/media-1.glx")
	assert.Contains(t, migrated, "properties:")
	assert.Contains(t, migrated, "A portrait")
}

// A file changed on disk after the command loaded it is refused, not
// overwritten.
func TestWriteArchiveInPlace_RefusesFileChangedSinceLoad(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive := loadForInPlace(t, dir)
	archive.Persons["person-a"].Properties["primary_name"] = "Alice Smith"

	ops, err := planInPlaceWrites(map[string][]byte{
		filepath.Join("people", "family.glx"): []byte(familyGLX),
	}, archive, rewriteChanged)
	require.NoError(t, err)
	edited := familyGLX + "  person-c: {}\n"
	writeSkipTestFile(t, filepath.Join(dir, "people", "family.glx"), edited)

	err = preflightFileOps(dir, orderFileOps(ops))

	require.ErrorIs(t, err, ErrRenameFileChanged)
	assert.Equal(t, edited, readArchiveFile(t, dir, "people/family.glx"))
}
