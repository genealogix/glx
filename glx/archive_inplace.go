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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// rewritePolicy decides which archive files writeArchiveInPlace rewrites.
type rewritePolicy int

const (
	// rewriteChanged rewrites a file only when the entities or vocabulary
	// entries it holds changed. A file the command did not touch keeps its
	// bytes, comments and formatting.
	rewriteChanged rewritePolicy = iota
	// rewriteNormalized additionally rewrites a file that still holds a field
	// a load-time shim folds away (a legacy top-level Source or Media
	// description, #667 and #894). The typed model cannot see that fold as a
	// change, so this is how it reaches disk. glx migrate uses it. A file
	// with no such field is judged as under rewriteChanged, so formatting
	// alone never causes a rewrite.
	rewriteNormalized
)

// emptyFragment is what the serializer writes for a fragment that holds
// nothing.
const emptyFragment = "{}\n"

// inPlaceJournalFile is written to the archive root while an in-place write
// runs and removed when it ends. It holds the prior content of every file the
// write touches, so a write cut short by a crash is detected by the next one
// and can be undone by hand. The leading dot keeps the loader from reading it.
const inPlaceJournalFile = ".glx-write-journal"

// writeArchiveInPlace persists archive, the complete result of a command
// that loaded the multi-file archive at archivePath from loaded (relative
// path -> contents, as collectGLXFilesFromDir returned them) and modified it
// in memory, by rewriting only the files that need it, where they are:
//
//   - an entity or vocabulary entry that changed is rewritten in the file
//     that defined it, which keeps that file's other contents and its place
//     in the user's layout (multi-entity files, sources/census/1850.glx, a
//     top-level family.glx);
//   - one the command removed is removed from every file defining it, and a
//     file left with nothing in it is deleted, along with any directory that
//     deletion leaves empty;
//   - a new entity gets a file of its own at the canonical
//     <type>/<id>.glx (joining the file already at that path, if any);
//   - everything else — untouched files, media binaries, dot entries,
//     symlinks, foreign files — is never opened for writing.
//
// loaded must be the bytes the command loaded, not a fresh read: preflight
// compares each file it rewrites against them, so an edit made on disk since
// the load is refused rather than overwritten, and an entity added since is
// never mistaken for one the command removed.
//
// This is the model glx rename adopted in #1204, generalized to commands
// that may change any part of the archive; see genealogix/glx#1247. Writes go
// through the same preflight and per-file atomic write with in-memory
// rollback as rename (preflightFileOps, executeFileOps), with a journal on
// disk for the length of the write (inPlaceJournalFile).
func writeArchiveInPlace(archivePath string, loaded map[string][]byte, archive *glxlib.GLXFile, policy rewritePolicy) error {
	if archive == nil {
		return glxlib.ErrGLXFileNil
	}
	if err := checkInPlaceJournal(archivePath); err != nil {
		return err
	}

	ops, err := planInPlaceWrites(loaded, archive, policy)
	if err != nil || len(ops) == 0 {
		return err
	}

	ordered := orderFileOps(ops)
	if err := preflightFileOps(archivePath, ordered); err != nil {
		return err
	}

	return executeInPlaceOps(archivePath, ordered)
}

// inPlaceJournalEntry records one file an in-place write touches: its path
// and, when it existed, the content it held before the write.
type inPlaceJournalEntry struct {
	Path    string `json:"path"`
	Existed bool   `json:"existed"`
	Data    []byte `json:"data,omitempty"`
}

// checkInPlaceJournal refuses a write while an earlier write's journal is
// still in the archive root: that write stopped part way, and writing over a
// half-applied plan would bury the evidence of what it changed.
func checkInPlaceJournal(archivePath string) error {
	journal := filepath.Join(archivePath, inPlaceJournalFile)
	_, err := os.Lstat(journal)
	switch {
	case err == nil:
		return fmt.Errorf("%w: %s lists each file it touched with its prior content; "+
			"restore those files (or the archive, from version control), then delete the journal", ErrInterruptedInPlaceWrite, journal)
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("checking %s: %w", journal, err)
	}
}

// executeInPlaceOps writes the journal, runs the preflighted operations, and
// tidies up after them: on success it removes directories the deletions left
// empty, on a rolled-back failure the directories the creations made. The
// journal is removed unless rollback itself failed, in which case it is the
// record of what the archive held before.
func executeInPlaceOps(archivePath string, ordered []fileOp) error {
	journal := filepath.Join(archivePath, inPlaceJournalFile)
	entries := make([]inPlaceJournalEntry, 0, len(ordered))
	for _, op := range ordered {
		entries = append(entries, inPlaceJournalEntry{Path: filepath.ToSlash(op.relPath), Existed: op.oldData != nil, Data: op.oldData})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding write journal: %w", err)
	}
	if err := atomicWriteFile(journal, data, filePermissions); err != nil {
		return fmt.Errorf("writing %s: %w", journal, err)
	}

	createdDirs := missingParentDirs(archivePath, ordered)
	if err := executeFileOps(archivePath, ordered); err != nil {
		if errors.Is(err, errRollbackIncomplete) {
			return fmt.Errorf("%w; prior contents are recorded in %s", err, journal)
		}
		removeEmptyDirs(createdDirs)
		if rmErr := os.Remove(journal); rmErr != nil {
			return fmt.Errorf("%w; removing %s: %w", err, journal, rmErr)
		}

		return err
	}

	removeEmptyDirs(emptiedParentDirs(archivePath, ordered))
	if err := os.Remove(journal); err != nil {
		return fmt.Errorf("archive written, but removing %s failed (delete it by hand): %w", journal, err)
	}

	return nil
}

// missingParentDirs returns the directories below root that creating the
// ops' new files will make, deepest first.
func missingParentDirs(root string, ops []fileOp) []string {
	var dirs []string
	for _, op := range ops {
		if op.oldData != nil || op.newData == nil {
			continue
		}
		for dir := filepath.Dir(op.relPath); dir != "." && !slices.Contains(dirs, dir); dir = filepath.Dir(dir) {
			if _, err := os.Lstat(filepath.Join(root, dir)); err == nil {
				break
			}
			dirs = append(dirs, dir)
		}
	}

	return deepestFirst(root, dirs)
}

// emptiedParentDirs returns the directories below root that held a file the
// ops delete, deepest first. Any of them may now be empty.
func emptiedParentDirs(root string, ops []fileOp) []string {
	var dirs []string
	for _, op := range ops {
		if op.newData != nil {
			continue
		}
		for dir := filepath.Dir(op.relPath); dir != "." && !slices.Contains(dirs, dir); dir = filepath.Dir(dir) {
			dirs = append(dirs, dir)
		}
	}

	return deepestFirst(root, dirs)
}

// deepestFirst joins each relative directory to root and orders the result
// so a directory comes before its parent.
func deepestFirst(root string, dirs []string) []string {
	abs := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		abs = append(abs, filepath.Join(root, dir))
	}
	slices.SortFunc(abs, func(a, b string) int { return len(b) - len(a) })

	return abs
}

// removeEmptyDirs removes each directory that is empty. One that is not, or
// that cannot be removed, is left as it is: tidying never fails a write.
func removeEmptyDirs(dirs []string) {
	for _, dir := range dirs {
		_ = os.Remove(dir)
	}
}

// glxCollection is one map-valued field of GLXFile: an entity type, a
// vocabulary, or a property vocabulary, named by its YAML key.
type glxCollection struct {
	key      string
	field    int
	isEntity bool
	// unionDefaults is set for the property vocabularies, which the loader
	// always unions with the standard set (an archive entry overriding a
	// standard one). The type vocabularies are filled with the standard set
	// only when the archive defines none of their entries.
	unionDefaults bool
}

// glxCollections lists every map field of GLXFile. Deriving it by reflection
// means a newly added entity type or vocabulary is written in place without
// this file changing, as archiveManagedTopLevel derives from AllEntityTypes.
var glxCollections = func() []glxCollection {
	entityKeys := make(map[string]bool, len(glxlib.AllEntityTypes))
	for _, entityType := range glxlib.AllEntityTypes {
		entityKeys[entityType.String()] = true
	}
	propertyDefinition := reflect.TypeFor[*glxlib.PropertyDefinition]()

	t := reflect.TypeFor[glxlib.GLXFile]()
	var collections []glxCollection
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() || field.Type.Kind() != reflect.Map {
			continue
		}
		key, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		collections = append(collections, glxCollection{
			key:           key,
			field:         i,
			isEntity:      entityKeys[key],
			unionDefaults: field.Type.Elem() == propertyDefinition,
		})
	}

	return collections
}()

// mapOf returns the collection's map in g, settable.
func (c glxCollection) mapOf(g *glxlib.GLXFile) reflect.Value {
	return reflect.ValueOf(g).Elem().Field(c.field)
}

// sortedMapKeys returns the string keys of a map value in sorted order.
func sortedMapKeys(m reflect.Value) []string {
	keys := make([]string, 0, m.Len())
	for _, k := range m.MapKeys() {
		keys = append(keys, k.String())
	}
	slices.Sort(keys)

	return keys
}

// setMapEntry stores value under key in the collection's map in g, creating
// the map if the fragment had none.
func (c glxCollection) setMapEntry(g *glxlib.GLXFile, key string, value reflect.Value) {
	m := c.mapOf(g)
	if m.IsNil() {
		m.Set(reflect.MakeMap(m.Type()))
	}
	m.SetMapIndex(reflect.ValueOf(key), value)
}

// standardVocabularyPaths maps each vocabulary collection key to the
// archive path of the standard vocabulary file that defines it, e.g.
// event_types -> vocabularies/event-types.glx. A standard file that does not
// parse is an error rather than a gap, since a gap would send its
// collections to a guessed filename.
var standardVocabularyPaths = sync.OnceValues(func() (map[string]string, error) {
	serializer := createSerializer(false, false, "")
	paths := make(map[string]string)
	for filename, data := range glxlib.StandardVocabularies() {
		fragment, err := serializer.DeserializeSingleFileBytes(data)
		if err != nil {
			return nil, fmt.Errorf("parsing standard vocabulary %s: %w", filename, err)
		}
		for _, c := range glxCollections {
			if c.mapOf(fragment).Len() > 0 {
				paths[c.key] = filepath.Join(glxlib.ArchiveDirVocabularies, filename)
			}
		}
	}

	return paths, nil
})

// vocabularyPath is where a vocabulary collection that no archive file
// defines yet is written: the standard file for it, or, for a collection no
// standard file covers, vocabularies/<key with hyphens>.glx.
func vocabularyPath(key string) (string, error) {
	paths, err := standardVocabularyPaths()
	if err != nil {
		return "", err
	}
	if p, ok := paths[key]; ok {
		return p, nil
	}

	return filepath.Join(glxlib.ArchiveDirVocabularies, strings.ReplaceAll(key, "_", "-")+FileExtGLX), nil
}

// inPlacePlan accumulates the fragments a write changes: the archive's
// existing files, parsed, plus any files the write creates.
type inPlacePlan struct {
	files     map[string][]byte
	relPaths  []string
	fragments map[string]*glxlib.GLXFile
	created   map[string]*glxlib.GLXFile
	// dirty holds the existing files whose fragment the projection changed.
	// Only these (and, under rewriteNormalized, files holding a shimmed
	// field) are serialized again.
	dirty map[string]bool
	// legacy holds the existing files with a field a load-time shim folds
	// away; filled only under rewriteNormalized.
	legacy map[string]bool
	// canonical caches the typed round-trip of an existing file's original
	// content, against which its rewrite is compared.
	canonical map[string][]byte
	// existingFold maps each existing path, slash-separated and lowercased,
	// to its key in files: a file created at persons/person-a.glx would
	// overwrite Persons/Person-A.glx on a case-insensitive filesystem, so
	// a target that folds to an existing file joins that file instead.
	existingFold map[string]string
	// implicit holds the vocabulary collections no archive file defines and
	// the write leaves that way, keyed by YAML key: the loader fills them
	// with the standard entries, but only after the validating load has
	// checked references (loadArchiveFromFiles).
	implicit map[string]glxCollection
}

// fragmentFor returns the fragment that a write targeting relPath adds to:
// the existing file it folds to (marked dirty), an already-planned new file,
// or a new one.
func (p *inPlacePlan) fragmentFor(relPath string) *glxlib.GLXFile {
	if existing, ok := p.existingFold[strings.ToLower(filepath.ToSlash(relPath))]; ok {
		p.dirty[existing] = true

		return p.fragments[existing]
	}
	if fragment, ok := p.created[relPath]; ok {
		return fragment
	}
	fragment := &glxlib.GLXFile{}
	p.created[relPath] = fragment

	return fragment
}

// planInPlaceWrites computes the file operations that make the archive files
// (relative path -> contents, as collectGLXFilesFromDir returns them) hold
// final. See writeArchiveInPlace for the rules.
func planInPlaceWrites(files map[string][]byte, final *glxlib.GLXFile, policy rewritePolicy) ([]fileOp, error) {
	serializer := createSerializer(false, true, "  ")

	plan := &inPlacePlan{
		files:        files,
		relPaths:     slices.Sorted(maps.Keys(files)),
		fragments:    make(map[string]*glxlib.GLXFile, len(files)),
		created:      make(map[string]*glxlib.GLXFile),
		dirty:        make(map[string]bool),
		legacy:       make(map[string]bool),
		canonical:    make(map[string][]byte),
		existingFold: make(map[string]string, len(files)),
		implicit:     make(map[string]glxCollection),
	}

	// What the command started from, as the loader built it: the files merged
	// in sorted order, then the standard vocabularies filled in. A vocabulary
	// entry no file defines is either one of those defaults (not the
	// archive's to write unless the command changed it) or new. Merge copies
	// map entries, so projecting into the fragments below leaves it intact.
	baseline := &glxlib.GLXFile{}
	for _, relPath := range plan.relPaths {
		fragment, err := serializer.DeserializeSingleFileBytes(files[relPath])
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}
		plan.fragments[relPath] = fragment
		plan.existingFold[strings.ToLower(filepath.ToSlash(relPath))] = relPath
		if policy == rewriteNormalized && hasShimmedFields(files[relPath]) {
			plan.legacy[relPath] = true
		}
		baseline.Merge(fragment)
	}
	if err := mergeStandardVocabularies(baseline); err != nil {
		return nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}

	for _, c := range glxCollections {
		if err := plan.projectCollection(c, final, baseline); err != nil {
			return nil, err
		}
	}
	plan.projectMetadata(serializer, final)

	// Writing one implicit vocabulary cannot make another one referenced, but
	// loop to a fixed point anyway: each pass removes a collection from
	// implicit, so it ends.
	for {
		ops, err := plan.fileOps(serializer)
		if err != nil || len(ops) == 0 || len(plan.implicit) == 0 {
			return ops, err
		}
		materialized, err := plan.materializeReferencedVocabularies(ops, final)
		if err != nil {
			return nil, err
		}
		if !materialized {
			return ops, nil
		}
	}
}

// hasShimmedFields reports whether a file holds a field a load-time shim
// folds into the typed model: a top-level description on a source (#667) or
// a media entity (#894).
func hasShimmedFields(data []byte) bool {
	var doc struct {
		Sources map[string]map[string]any `yaml:"sources"`
		Media   map[string]map[string]any `yaml:"media"`
	}
	if yaml.Unmarshal(data, &doc) != nil {
		return false
	}
	for _, entities := range []map[string]map[string]any{doc.Sources, doc.Media} {
		for _, fields := range entities {
			if _, ok := fields["description"]; ok {
				return true
			}
		}
	}

	return false
}

// materializeReferencedVocabularies writes out each implicit vocabulary that
// the archive as written would reference, and reports whether it wrote any.
//
// The validating load checks references before it fills the standard
// vocabularies, so an archive with no event-types file is valid only while
// nothing names an event type. A write that adds the first birth event (a
// merge from an archive that did carry the file) would otherwise produce an
// archive glx validate rejects. Validating the files as the write leaves
// them, before any fill, finds exactly the vocabularies the result needs.
// That pass collects every issue rather than stopping at the first failure,
// so an unrelated problem in the archive does not hide a missing vocabulary.
func (p *inPlacePlan) materializeReferencedVocabularies(ops []fileOp, final *glxlib.GLXFile) (bool, error) {
	written := make(map[string][]byte, len(p.files)+len(ops))
	maps.Copy(written, p.files)
	for _, op := range ops {
		if op.newData == nil {
			delete(written, op.relPath)
		} else {
			written[op.relPath] = op.newData
		}
	}

	archive, _, err := createSerializer(false, false, "").DeserializeMultiFileFromMap(written)
	if err != nil {
		return false, fmt.Errorf("failed to read back the planned archive: %w", err)
	}

	materialized := false
	for _, issue := range archive.Validate().Errors {
		c, ok := p.implicit[issue.TargetType]
		if !ok || !c.mapOf(final).MapIndex(reflect.ValueOf(issue.TargetID)).IsValid() {
			continue
		}
		path, err := vocabularyPath(c.key)
		if err != nil {
			return false, err
		}
		finalMap := c.mapOf(final)
		fragment := p.fragmentFor(path)
		for _, key := range sortedMapKeys(finalMap) {
			c.setMapEntry(fragment, key, finalMap.MapIndex(reflect.ValueOf(key)))
		}
		delete(p.implicit, c.key)
		materialized = true
	}

	return materialized, nil
}

// projectCollection brings collection c of every fragment in line with final.
func (p *inPlacePlan) projectCollection(c glxCollection, final, baseline *glxlib.GLXFile) error {
	finalMap := c.mapOf(final)

	// The loader keeps the first definition of a key in sorted path order
	// (later ones are reported as duplicates), so that file owns the key and
	// receives its update. A shadowed copy is left as it was, but a removal
	// applies to every copy, or the next load would bring the key back.
	owners := make(map[string]string)
	home := ""
	for _, relPath := range p.relPaths {
		m := c.mapOf(p.fragments[relPath])
		if m.Len() > 0 && home == "" {
			home = relPath
		}
		for _, key := range sortedMapKeys(m) {
			k := reflect.ValueOf(key)
			value := finalMap.MapIndex(k)
			switch _, owned := owners[key]; {
			case !value.IsValid():
				m.SetMapIndex(k, reflect.Value{})
				p.dirty[relPath] = true
			case !owned:
				owners[key] = relPath
				if !reflect.DeepEqual(m.MapIndex(k).Interface(), value.Interface()) {
					m.SetMapIndex(k, value)
					p.dirty[relPath] = true
				}
			}
		}
	}
	if finalMap.Len() == 0 {
		return nil
	}

	finalKeys := sortedMapKeys(finalMap)
	if c.isEntity {
		return p.placeNewEntities(c, finalMap, finalKeys, owners)
	}

	return p.projectVocabularyDefaults(c, finalMap, finalKeys, owners, home, c.mapOf(baseline))
}

// projectVocabularyDefaults handles the entries of vocabulary c that no file
// owns: ones the command added, and standard-filled ones it changed or
// removed. home is the first file defining c, or "" when none does.
func (p *inPlacePlan) projectVocabularyDefaults(c glxCollection, finalMap reflect.Value, finalKeys []string, owners map[string]string, home string, baselineMap reflect.Value) error {
	var changed []string
	for _, key := range finalKeys {
		if _, owned := owners[key]; owned {
			continue
		}
		k := reflect.ValueOf(key)
		if before := baselineMap.MapIndex(k); !before.IsValid() || !reflect.DeepEqual(before.Interface(), finalMap.MapIndex(k).Interface()) {
			changed = append(changed, key)
		}
	}

	// A type vocabulary no file defines holds the standard fill, so dropping
	// one of its entries is a change too. A property vocabulary's standard
	// entries come back on every load whatever the files say, so dropping one
	// cannot be written and is not looked for.
	fillReduced := false
	if home == "" && !c.unionDefaults {
		for _, key := range sortedMapKeys(baselineMap) {
			if !finalMap.MapIndex(reflect.ValueOf(key)).IsValid() {
				fillReduced = true

				break
			}
		}
	}

	if len(changed) == 0 && !fillReduced {
		if home == "" && !c.unionDefaults {
			p.implicit[c.key] = c
		}

		return nil
	}

	target := home
	if target == "" {
		path, err := vocabularyPath(c.key)
		if err != nil {
			return err
		}
		target = path
	}
	fragment := p.fragmentFor(target)

	// Writing only the changed entries of a filled type vocabulary would make
	// its map non-empty on the next load, which turns the fill off and drops
	// every standard entry the archive relies on; write the whole map. A
	// property vocabulary keeps its fill, so it gets the changed entries
	// only, and the standard ones stay the standard set's to update.
	keys := changed
	if home == "" && !c.unionDefaults {
		keys = finalKeys
	}
	for _, key := range keys {
		c.setMapEntry(fragment, key, finalMap.MapIndex(reflect.ValueOf(key)))
	}

	return nil
}

// placeNewEntities gives each entity in final that no file defines a file at
// its canonical path, as the multi-file serializer would name it.
func (p *inPlacePlan) placeNewEntities(c glxCollection, finalMap reflect.Value, finalKeys []string, owners map[string]string) error {
	// Filenames are derived from lowercased IDs, so two IDs of one type that
	// differ only by case would share a file. Refuse, as rename and the
	// serializer do.
	folded := make(map[string]string, len(finalKeys))
	for _, key := range finalKeys {
		if other, clash := folded[strings.ToLower(key)]; clash {
			_, keyOwned := owners[key]
			_, otherOwned := owners[other]
			if !keyOwned || !otherOwned {
				return fmt.Errorf("entity IDs %q and %q: %w", other, key, glxlib.ErrCaseInsensitiveCollision)
			}
		}
		folded[strings.ToLower(key)] = key
	}

	for _, key := range finalKeys {
		if _, owned := owners[key]; owned {
			continue
		}
		filename, err := glxlib.EntityIDToFilename(key)
		if err != nil {
			return fmt.Errorf("%s %s: %w", c.key, key, err)
		}
		c.setMapEntry(p.fragmentFor(filepath.Join(c.key, filename)), key, finalMap.MapIndex(reflect.ValueOf(key)))
	}

	return nil
}

// projectMetadata updates the archive metadata block in the file the loader
// took it from, or writes it to metadata.glx when the command added one.
//
// The loader takes the first block with content in sorted path order and
// skips empty ones (GLXFile.Merge), so an earlier `metadata: {}` does not own
// the block. Projecting into it would leave two populated blocks, which the
// loader then reports as a conflict.
func (p *inPlacePlan) projectMetadata(serializer *glxlib.DefaultSerializer, final *glxlib.GLXFile) {
	for _, relPath := range p.relPaths {
		if fragment := p.fragments[relPath]; metadataHasContent(serializer, fragment.ImportMetadata) {
			if !reflect.DeepEqual(fragment.ImportMetadata, final.ImportMetadata) {
				fragment.ImportMetadata = final.ImportMetadata
				p.dirty[relPath] = true
			}

			return
		}
	}

	if metadataHasContent(serializer, final.ImportMetadata) {
		p.fragmentFor(archiveMetadataFile).ImportMetadata = final.ImportMetadata
	}
}

// metadataHasContent reports whether a metadata block has content, as the
// loader's Merge judges it. The serializer drops a block with no content, so
// an empty one serializes to an empty fragment.
func metadataHasContent(serializer *glxlib.DefaultSerializer, metadata *glxlib.Metadata) bool {
	if metadata == nil {
		return false
	}
	data, err := serializer.SerializeSingleFileBytes(&glxlib.GLXFile{ImportMetadata: metadata})

	return err == nil && string(data) != emptyFragment
}

// canonicalOf returns the typed round-trip of an existing file's original
// content: what it would hold if rewritten with nothing changed.
func (p *inPlacePlan) canonicalOf(serializer *glxlib.DefaultSerializer, relPath string) ([]byte, error) {
	if data, ok := p.canonical[relPath]; ok {
		return data, nil
	}
	fragment, err := serializer.DeserializeSingleFileBytes(p.files[relPath])
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
	}
	data, err := serializer.SerializeSingleFileBytes(fragment)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
	}
	p.canonical[relPath] = data

	return data, nil
}

// fileOps serializes the planned fragments and returns an operation for each
// file whose content changes: rewritten, deleted when nothing is left in it,
// or created. An existing file the projection did not touch is skipped
// without being serialized, unless it holds a shimmed field.
func (p *inPlacePlan) fileOps(serializer *glxlib.DefaultSerializer) ([]fileOp, error) {
	var ops []fileOp
	for _, relPath := range p.relPaths {
		if !p.dirty[relPath] && !p.legacy[relPath] {
			continue
		}
		after, err := serializer.SerializeSingleFileBytes(p.fragments[relPath])
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}
		before, err := p.canonicalOf(serializer, relPath)
		if err != nil {
			return nil, err
		}
		if string(after) == emptyFragment {
			if string(before) != emptyFragment {
				ops = append(ops, fileOp{relPath: relPath, oldData: p.files[relPath]})
			}

			continue
		}
		if p.legacy[relPath] || !bytes.Equal(before, after) {
			ops = append(ops, fileOp{relPath: relPath, oldData: p.files[relPath], newData: after})
		}
	}

	for _, relPath := range slices.Sorted(maps.Keys(p.created)) {
		data, err := serializer.SerializeSingleFileBytes(p.created[relPath])
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}
		ops = append(ops, fileOp{relPath: relPath, newData: data})
	}

	return ops, nil
}
