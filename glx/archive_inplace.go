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
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// rewritePolicy decides which archive files writeArchiveInPlace rewrites.
type rewritePolicy int

const (
	// rewriteChanged rewrites a file only when the entities or vocabulary
	// entries it holds changed. A file the command did not touch keeps its
	// bytes, comments and formatting.
	rewriteChanged rewritePolicy = iota
	// rewriteNormalized additionally rewrites a file whose on-disk content
	// differs from what the typed model writes back, which is how a
	// load-time shim's fold (a legacy Source or Media description, #667 and
	// #894) reaches disk. glx migrate uses it.
	rewriteNormalized
)

// emptyFragment is what the serializer writes for a fragment that holds
// nothing.
const emptyFragment = "{}\n"

// writeArchiveInPlace persists archive, the complete result of a command
// that loaded the multi-file archive at archivePath and modified it in
// memory, by rewriting only the files that need it, where they are:
//
//   - an entity or vocabulary entry that changed is rewritten in the file
//     that defined it, which keeps that file's other contents and its place
//     in the user's layout (multi-entity files, sources/census/1850.glx, a
//     top-level family.glx);
//   - one the command removed is removed from every file defining it, and a
//     file left with nothing in it is deleted;
//   - a new entity gets a file of its own at the canonical
//     <type>/<id>.glx (joining the file already at that path, if any);
//   - everything else — untouched files, media binaries, dot entries,
//     symlinks, foreign files — is never opened for writing.
//
// This is the model glx rename adopted in #1204, generalized to commands
// that may change any part of the archive; see genealogix/glx#1247. Writes go
// through the same preflight and per-file atomic write with in-memory
// rollback as rename (preflightFileOps, executeFileOps).
func writeArchiveInPlace(archivePath string, archive *glxlib.GLXFile, policy rewritePolicy) error {
	if archive == nil {
		return glxlib.ErrGLXFileNil
	}

	// The files are read again rather than threaded from the command's load:
	// preflight compares each file it rewrites against these bytes, so a
	// change made since the load is still refused rather than overwritten
	// whenever it lands in a file this write touches.
	files, err := collectGLXFilesFromDir(archivePath)
	if err != nil {
		return fmt.Errorf("failed to read archive: %w", err)
	}

	ops, err := planInPlaceWrites(files, archive, policy)
	if err != nil {
		return err
	}

	ordered := orderFileOps(ops)
	if err := preflightFileOps(archivePath, ordered); err != nil {
		return err
	}

	return executeFileOps(archivePath, ordered)
}

// glxCollection is one map-valued field of GLXFile: an entity type, a
// vocabulary, or a property vocabulary, named by its YAML key.
type glxCollection struct {
	key      string
	field    int
	isEntity bool
}

// glxCollections lists every map field of GLXFile. Deriving it by reflection
// means a newly added entity type or vocabulary is written in place without
// this file changing, as archiveManagedTopLevel derives from AllEntityTypes.
var glxCollections = func() []glxCollection {
	entityKeys := make(map[string]bool, len(glxlib.AllEntityTypes))
	for _, entityType := range glxlib.AllEntityTypes {
		entityKeys[entityType.String()] = true
	}

	t := reflect.TypeFor[glxlib.GLXFile]()
	var collections []glxCollection
	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() || field.Type.Kind() != reflect.Map {
			continue
		}
		key, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		collections = append(collections, glxCollection{key: key, field: i, isEntity: entityKeys[key]})
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
// event_types -> vocabularies/event-types.glx.
var standardVocabularyPaths = sync.OnceValue(func() map[string]string {
	serializer := createSerializer(false, false, "")
	paths := make(map[string]string)
	for filename, data := range glxlib.StandardVocabularies() {
		fragment, err := serializer.DeserializeSingleFileBytes(data)
		if err != nil {
			continue
		}
		for _, c := range glxCollections {
			if c.mapOf(fragment).Len() > 0 {
				paths[c.key] = filepath.Join(glxlib.ArchiveDirVocabularies, filename)
			}
		}
	}

	return paths
})

// vocabularyPath is where a vocabulary collection that no archive file
// defines yet is written.
func vocabularyPath(key string) string {
	if p, ok := standardVocabularyPaths()[key]; ok {
		return p
	}

	return filepath.Join(glxlib.ArchiveDirVocabularies, strings.ReplaceAll(key, "_", "-")+FileExtGLX)
}

// inPlacePlan accumulates the fragments a write changes: the archive's
// existing files, parsed, plus any files the write creates.
type inPlacePlan struct {
	files     map[string][]byte
	relPaths  []string
	fragments map[string]*glxlib.GLXFile
	created   map[string]*glxlib.GLXFile
	// existingFold maps each existing path, slash-separated and lowercased,
	// to its key in files: a file created at persons/person-a.glx would
	// overwrite Persons/Person-A.glx on a case-insensitive filesystem, so
	// a target that folds to an existing file joins that file instead.
	existingFold map[string]string
}

// fragmentFor returns the fragment that a write targeting relPath adds to:
// the existing file it folds to, an already-planned new file, or a new one.
func (p *inPlacePlan) fragmentFor(relPath string) *glxlib.GLXFile {
	if existing, ok := p.existingFold[strings.ToLower(filepath.ToSlash(relPath))]; ok {
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
		existingFold: make(map[string]string, len(files)),
	}
	before := make(map[string][]byte, len(files))
	for _, relPath := range plan.relPaths {
		fragment, err := serializer.DeserializeSingleFileBytes(files[relPath])
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}
		serialized, err := serializer.SerializeSingleFileBytes(fragment)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}
		plan.fragments[relPath] = fragment
		before[relPath] = serialized
		plan.existingFold[strings.ToLower(filepath.ToSlash(relPath))] = relPath
	}

	// What the command started from, as the loader built it: the files merged
	// in sorted order, then the standard vocabularies filled into every
	// vocabulary map the files left empty. A vocabulary entry no file defines
	// is either one of those defaults (not the archive's to write) or new.
	baseline, _, err := serializer.DeserializeMultiFileFromMap(files)
	if err != nil {
		return nil, fmt.Errorf("failed to load archive: %w", err)
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

	return plan.fileOps(serializer, before, policy)
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
			value := finalMap.MapIndex(reflect.ValueOf(key))
			switch _, owned := owners[key]; {
			case !value.IsValid():
				m.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
			case !owned:
				owners[key] = relPath
				m.SetMapIndex(reflect.ValueOf(key), value)
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

	baselineMap := c.mapOf(baseline)
	var added []string
	for _, key := range finalKeys {
		if _, owned := owners[key]; !owned && !baselineMap.MapIndex(reflect.ValueOf(key)).IsValid() {
			added = append(added, key)
		}
	}
	if len(added) == 0 {
		return nil
	}

	if home != "" {
		for _, key := range added {
			c.setMapEntry(p.fragments[home], key, finalMap.MapIndex(reflect.ValueOf(key)))
		}

		return nil
	}

	// No file defines this vocabulary, so the loader filled it with the
	// standard entries. Writing only the new ones would make the map
	// non-empty on the next load, which turns that fill off and drops every
	// standard entry the archive relies on; write the whole map instead.
	fragment := p.fragmentFor(vocabularyPath(c.key))
	for _, key := range finalKeys {
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
func (p *inPlacePlan) projectMetadata(serializer *glxlib.DefaultSerializer, final *glxlib.GLXFile) {
	for _, relPath := range p.relPaths {
		if fragment := p.fragments[relPath]; fragment.ImportMetadata != nil {
			fragment.ImportMetadata = final.ImportMetadata

			return
		}
	}

	// The serializer drops a metadata block with no content, so an empty
	// one serializes to an empty fragment and needs no file.
	data, err := serializer.SerializeSingleFileBytes(&glxlib.GLXFile{ImportMetadata: final.ImportMetadata})
	if err != nil || string(data) == emptyFragment {
		return
	}
	p.fragmentFor(archiveMetadataFile).ImportMetadata = final.ImportMetadata
}

// fileOps serializes the planned fragments and returns an operation for each
// file whose content changes: rewritten, deleted when nothing is left in it,
// or created.
func (p *inPlacePlan) fileOps(serializer *glxlib.DefaultSerializer, before map[string][]byte, policy rewritePolicy) ([]fileOp, error) {
	var ops []fileOp
	for _, relPath := range p.relPaths {
		after, err := serializer.SerializeSingleFileBytes(p.fragments[relPath])
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}
		if string(after) == emptyFragment {
			if string(before[relPath]) != emptyFragment {
				ops = append(ops, fileOp{relPath: relPath, oldData: p.files[relPath]})
			}

			continue
		}
		unchanged := bytes.Equal(before[relPath], after)
		if policy == rewriteNormalized {
			unchanged = sameYAMLContent(p.files[relPath], after)
		}
		if !unchanged {
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

// sameYAMLContent reports whether two YAML documents decode to the same
// data, ignoring formatting and comments.
func sameYAMLContent(a, b []byte) bool {
	var docA, docB any
	if yaml.Unmarshal(a, &docA) != nil || yaml.Unmarshal(b, &docB) != nil {
		return false
	}

	return reflect.DeepEqual(docA, docB)
}
