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
	"strings"
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

// writeArchiveInPlaceFresh writes archive over the files dir holds now, for a
// test whose archive was built in memory or whose files have not changed
// since it was loaded.
func writeArchiveInPlaceFresh(dir string, archive *glxlib.GLXFile, policy rewritePolicy) error {
	files, err := collectGLXFilesFromDir(dir)
	if err != nil {
		return err
	}

	return writeArchiveInPlace(dir, files, archive, policy)
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

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

	require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteChanged))

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

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

	assert.NoFileExists(t, filepath.Join(dir, "persons", "person-b.glx"))
	assert.FileExists(t, filepath.Join(dir, "persons", "person-a.glx"))
}

// A new entity is written where the multi-file serializer would put it.
func TestWriteArchiveInPlace_NewEntityGoesToCanonicalPath(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive := loadForInPlace(t, dir)
	archive.Events["Event-Birth-A"] = &glxlib.Event{Type: "birth"}

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

	err := writeArchiveInPlaceFresh(dir, archive, rewriteChanged)

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

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, "a/first.glx"), "Updated")
		assert.Equal(t, shadow, readArchiveFile(t, dir, "b/second.glx"))
	})

	t.Run("removal reaches every copy", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)
		archive := loadForInPlace(t, dir)
		delete(archive.Persons, "person-a")

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

		require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteChanged))

		assert.NoDirExists(t, filepath.Join(dir, "vocabularies"))
	})

	t.Run("an added entry writes the full vocabulary", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
		archive := loadForInPlace(t, dir)
		require.Contains(t, archive.EventTypes, "birth", "the loader supplies the standard event types")
		standardCount := len(archive.EventTypes)
		archive.EventTypes["barmitzvah"] = &glxlib.VocabularyEntry{Label: "Bar Mitzvah"}

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

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

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, archiveMetadataFile), "test")
	})

	t.Run("updated where it lives", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{
			"people/family.glx": "metadata:\n  source_system: old\n" + familyGLX,
		})
		archive := loadForInPlace(t, dir)
		archive.ImportMetadata.SourceSystem = "new"

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		assert.Contains(t, readArchiveFile(t, dir, "people/family.glx"), "new")
		assert.NoFileExists(t, filepath.Join(dir, archiveMetadataFile))
	})
}

// rewriteNormalized persists a load-time shim's fold, which the typed model
// cannot see as a change; rewriteChanged leaves the file alone.
func TestWriteArchiveInPlace_NormalizedPolicyPersistsShimFolds(t *testing.T) {
	legacy := "media:\n  media-1:\n    uri: files/a.jpg\n    description: A portrait\n"

	dir := stageInPlaceArchive(t, map[string]string{"media/media-1.glx": legacy})
	require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteChanged))
	assert.Equal(t, legacy, readArchiveFile(t, dir, "media/media-1.glx"))

	require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteNormalized))
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

// requireValidArchive runs the validating load glx validate uses, then the
// full validation pass, and fails on any error either reports.
func requireValidArchive(t *testing.T, dir string) *glxlib.GLXFile {
	t.Helper()
	archive, duplicates, err := LoadArchive(dir)
	require.NoError(t, err)
	assert.Empty(t, duplicates)
	result := archive.Validate()
	for _, issue := range result.Errors {
		t.Errorf("validation error: %s", issue.Message)
	}

	return archive
}

// stageDirSymlink creates relLink inside dir as a symlink to relTarget (both
// slash-separated, relative to dir), creating the target directory.
func stageDirSymlink(t *testing.T, dir, relLink, relTarget string) {
	t.Helper()
	if runtime.GOOS == goosWindows {
		t.Skip("symlink creation requires elevation on Windows")
	}
	link := filepath.Join(dir, filepath.FromSlash(relLink))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, filepath.FromSlash(relTarget)), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(link), 0o755))
	rel, err := filepath.Rel(filepath.Dir(link), filepath.Join(dir, filepath.FromSlash(relTarget)))
	require.NoError(t, err)
	require.NoError(t, os.Symlink(rel, link))
}

const newPersonSource = "persons:\n  person-new:\n    properties:\n      primary_name: Newcomer\n"

// A new entity's canonical directory that is a symlink into a dot directory
// would put the entity where the loader never looks. The write is refused and
// nothing is written; the merge does not report a person the archive lost.
func TestMerge_RefusesNewEntityThroughSymlinkIntoDotDirectory(t *testing.T) {
	dest := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	stageDirSymlink(t, dest, "persons", ".drafts/persons")
	src := stageInPlaceArchive(t, map[string]string{"persons/person-new.glx": newPersonSource})

	err := mergeArchives(src, dest, false, 0.6)

	require.ErrorIs(t, err, ErrPathNotArchiveContent)
	assert.NoFileExists(t, filepath.Join(dest, ".drafts", "persons", "person-new.glx"))
	assert.Equal(t, familyGLX, readArchiveFile(t, dest, "people/family.glx"))
}

// A symlinked directory that resolves to visible archive content is still
// fine: the new entity is written through it and the next load sees it.
func TestMerge_NewEntityThroughSymlinkToVisibleDirectoryStaysVisible(t *testing.T) {
	dest := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	stageDirSymlink(t, dest, "persons", "people/more")
	src := stageInPlaceArchive(t, map[string]string{"persons/person-new.glx": newPersonSource})

	require.NoError(t, mergeArchives(src, dest, false, 0.6))

	reloaded := requireValidArchive(t, dest)
	assert.Contains(t, reloaded.Persons, "person-new")
	assert.Len(t, reloaded.Persons, 3)
}

// The migrate variant: replacing born_on with a birth event must not move
// the fact somewhere the loader skips.
func TestMigrate_RefusesNewEventThroughSymlinkIntoDotDirectory(t *testing.T) {
	person := "persons:\n  person-a:\n    properties:\n      primary_name: Alice\n      born_on: \"1850\"\n"
	dir := stageInPlaceArchive(t, map[string]string{"persons/person-a.glx": person})
	stageDirSymlink(t, dir, "events", ".drafts/events")

	err := migrateArchive(dir)

	require.ErrorIs(t, err, ErrPathNotArchiveContent)
	assert.Equal(t, person, readArchiveFile(t, dir, "persons/person-a.glx"))
	entries, readErr := os.ReadDir(filepath.Join(dir, ".drafts", "events"))
	require.NoError(t, readErr)
	assert.Empty(t, entries)
}

// The validating load checks references before it fills the standard
// vocabularies. Merging a source that names birth/subject into a destination
// that carries no vocabulary files must write the vocabularies the result
// now relies on, or glx validate rejects an archive built from two valid ones.
func TestMerge_TwoValidArchivesStayValid(t *testing.T) {
	dest := stageInPlaceArchive(t, map[string]string{
		"persons/person-a.glx": "persons:\n  person-a:\n    properties:\n      primary_name: Alice\n",
	})
	standard := glxlib.StandardVocabularies()
	src := stageInPlaceArchive(t, map[string]string{
		"persons/person-b.glx": "persons:\n  person-b:\n    properties:\n      primary_name: Bob\n",
		"events/event-b.glx": "events:\n  event-b:\n    type: birth\n    date: \"1850\"\n" +
			"    participants:\n      - person: person-b\n        role: subject\n",
		"vocabularies/event-types.glx":       string(standard["event-types.glx"]),
		"vocabularies/participant-roles.glx": string(standard["participant-roles.glx"]),
	})
	requireValidArchive(t, dest)
	requireValidArchive(t, src)

	require.NoError(t, mergeArchives(src, dest, false, 0.6))

	merged := requireValidArchive(t, dest)
	assert.Contains(t, merged.Events, "event-b")
	assert.FileExists(t, filepath.Join(dest, "vocabularies", "event-types.glx"))
	assert.FileExists(t, filepath.Join(dest, "vocabularies", "participant-roles.glx"))
	assert.NoFileExists(t, filepath.Join(dest, "vocabularies", "place-types.glx"),
		"a vocabulary nothing references stays implicit")
}

// The loader takes metadata from the first block with content and skips an
// empty one, so an earlier `metadata: {}` (family/ sorts before metadata.glx)
// must not be treated as the owner.
func TestWriteArchiveInPlace_EmptyMetadataBlockDoesNotOwnMetadata(t *testing.T) {
	emptyMetaFirst := "metadata: {}\npersons:\n  person-c:\n    properties:\n      primary_name: Carol\n"
	metadata := "metadata:\n  source_system: My Tree\n"
	files := map[string]string{
		"family/person-c.glx":     emptyMetaFirst,
		"metadata.glx":            metadata,
		"persons/person-keep.glx": "persons:\n  person-keep:\n    properties:\n      primary_name: John Smith\n",
		"persons/person-drop.glx": "persons:\n  person-drop:\n    properties:\n      primary_name: John Smith\n",
	}

	t.Run("no-op leaves both files alone", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)

		require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteChanged))

		assert.Equal(t, emptyMetaFirst, readArchiveFile(t, dir, "family/person-c.glx"))
		assert.Equal(t, metadata, readArchiveFile(t, dir, "metadata.glx"))
	})

	t.Run("an update goes to the populated block", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)
		archive := loadForInPlace(t, dir)
		archive.ImportMetadata.SourceSystem = "Other Tree"

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		assert.Equal(t, emptyMetaFirst, readArchiveFile(t, dir, "family/person-c.glx"))
		assert.Contains(t, readArchiveFile(t, dir, "metadata.glx"), "Other Tree")
	})

	t.Run("merge-persons keeps one metadata owner", func(t *testing.T) {
		dir := stageInPlaceArchive(t, files)
		requireValidArchive(t, dir)

		require.NoError(t, mergePersons(dir, "person-keep", "person-drop", glxlib.MergePersonsOptions{}, false))

		merged := requireValidArchive(t, dir)
		assert.Equal(t, "My Tree", merged.ImportMetadata.SourceSystem)
		assert.NotContains(t, readArchiveFile(t, dir, "family/person-c.glx"), "My Tree")
	})
}

// The write compares the disk with the bytes the command loaded, not a fresh
// read: an entity added after the load is neither deleted as though the
// command removed it nor overwritten (#1247 review).
func TestWriteArchiveInPlace_RefusesEditMadeAfterLoad(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive, _, loaded, err := loadArchiveForWrite(dir)
	require.NoError(t, err)
	delete(archive.Persons, "person-b")

	edited := familyGLX + "  person-c:\n    properties:\n      primary_name: Carol\n"
	writeSkipTestFile(t, filepath.Join(dir, "people", "family.glx"), edited)

	err = writeArchiveInPlace(dir, loaded, archive, rewriteChanged)

	require.ErrorIs(t, err, ErrRenameFileChanged)
	assert.Equal(t, edited, readArchiveFile(t, dir, "people/family.glx"))
	assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
}

// A change to an entry the loader filled from the standard set reaches disk:
// the type vocabulary is written whole, so the next load keeps the change
// and every other standard entry.
func TestWriteArchiveInPlace_ChangedStandardEntry(t *testing.T) {
	t.Run("modified", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
		archive := loadForInPlace(t, dir)
		standardCount := len(archive.EventTypes)
		archive.EventTypes["birth"] = &glxlib.VocabularyEntry{Label: "Birth (custom)"}

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		reloaded := loadForInPlace(t, dir)
		assert.Equal(t, "Birth (custom)", reloaded.EventTypes["birth"].Label)
		assert.Len(t, reloaded.EventTypes, standardCount)
	})

	t.Run("removed", func(t *testing.T) {
		dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
		archive := loadForInPlace(t, dir)
		standardCount := len(archive.EventTypes)
		delete(archive.EventTypes, "birth")

		require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

		reloaded := loadForInPlace(t, dir)
		assert.NotContains(t, reloaded.EventTypes, "birth")
		assert.Len(t, reloaded.EventTypes, standardCount-1)
	})
}

// A property vocabulary is always unioned with the standard set, so adding a
// custom property writes that property alone: copying the standard ones in
// would override them for good.
func TestWriteArchiveInPlace_PropertyVocabularyWritesOnlyChanges(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	archive := loadForInPlace(t, dir)
	require.NotEmpty(t, archive.PersonProperties, "the loader supplies the standard person properties")
	standardCount := len(archive.PersonProperties)
	archive.PersonProperties["clan"] = &glxlib.PropertyDefinition{Label: "Clan", ValueType: "string"}

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

	files, err := collectGLXFilesFromDir(dir)
	require.NoError(t, err)
	var written *glxlib.GLXFile
	for relPath, data := range files {
		if strings.HasPrefix(filepath.ToSlash(relPath), "vocabularies/") {
			written, err = createSerializer(false, false, "").DeserializeSingleFileBytes(data)
			require.NoError(t, err)
		}
	}
	require.NotNil(t, written, "the new property is written to a vocabulary file")
	assert.Len(t, written.PersonProperties, 1)
	assert.Contains(t, written.PersonProperties, "clan")
	assert.Len(t, loadForInPlace(t, dir).PersonProperties, standardCount+1)
}

// rewriteNormalized rewrites only files holding a shimmed field: a file that
// is merely formatted differently from the canonical form keeps its bytes.
func TestWriteArchiveInPlace_NormalizedPolicyLeavesFormattingAlone(t *testing.T) {
	handWritten := "# Notes on the census.\nsources:\n  source-1850:   {title: \"1850 Census\"}\n"
	dir := stageInPlaceArchive(t, map[string]string{"sources/census.glx": handWritten})

	require.NoError(t, writeArchiveInPlaceFresh(dir, loadForInPlace(t, dir), rewriteNormalized))

	assert.Equal(t, handWritten, readArchiveFile(t, dir, "sources/census.glx"))
}

// A file deleted because it was left empty takes any directory it leaves
// empty with it.
func TestWriteArchiveInPlace_DeletingLastFileRemovesItsDirectory(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{
		"persons/person-a.glx":    "persons:\n  person-a: {}\n",
		"people/old/person-b.glx": "persons:\n  person-b: {}\n",
	})
	archive := loadForInPlace(t, dir)
	delete(archive.Persons, "person-b")

	require.NoError(t, writeArchiveInPlaceFresh(dir, archive, rewriteChanged))

	assert.NoDirExists(t, filepath.Join(dir, "people"))
	assert.DirExists(t, filepath.Join(dir, "persons"))
	assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
}

// A journal left by an interrupted write refuses the next write until the
// user has dealt with it.
func TestWriteArchiveInPlace_RefusesWhileJournalPresent(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"people/family.glx": familyGLX})
	writeSkipTestFile(t, filepath.Join(dir, inPlaceJournalFile), "[]\n")
	archive := loadForInPlace(t, dir)
	delete(archive.Persons, "person-b")

	err := writeArchiveInPlaceFresh(dir, archive, rewriteChanged)

	require.ErrorIs(t, err, ErrInterruptedInPlaceWrite)
	assert.Equal(t, familyGLX, readArchiveFile(t, dir, "people/family.glx"))
}

// A failed write that rolls back cleanly removes the directories it created
// and its journal.
func TestExecuteInPlaceOps_RolledBackFailureTidiesUp(t *testing.T) {
	dir := stageInPlaceArchive(t, map[string]string{"persons/person-a.glx": "persons:\n  person-a: {}\n"})
	// A directory where a file is to be rewritten makes that write fail.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "blocked.glx"), 0o755))
	ops := []fileOp{
		{relPath: filepath.Join("events", "new", "event-1.glx"), newData: []byte("events:\n  event-1: {}\n")},
		{relPath: "blocked.glx", oldData: []byte("x"), newData: []byte("y")},
	}

	err := executeInPlaceOps(dir, ops)

	require.Error(t, err)
	require.NotErrorIs(t, err, errRollbackIncomplete)
	assert.NoDirExists(t, filepath.Join(dir, "events"))
	assert.NoFileExists(t, filepath.Join(dir, inPlaceJournalFile))
}
