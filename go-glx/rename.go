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

package glx

import (
	"fmt"
)

// RenameResult holds the outcome of a rename operation.
type RenameResult struct {
	EntityType EntityType // which entity map contained the ID (e.g., EntityTypePersons)
	// RefsUpdated is the total number of changes made, which is the number of
	// reference fields rewritten plus one for the entity's own map key. It is
	// not a count of references alone; present it to users as "changes".
	RefsUpdated int
}

// RenameEntity renames an entity ID throughout the archive, updating all
// cross-references. Returns an error if the old ID is not found or the
// new ID already exists. Invalidates the validation cache after mutation.
func RenameEntity(glx *GLXFile, oldID, newID string) (*RenameResult, error) {
	entityType, err := findEntityType(glx, oldID)
	if err != nil {
		return nil, err
	}

	if err := checkTargetFree(glx, newID); err != nil {
		return nil, err
	}

	changes := RenameInFragment(glx, glx, entityType, oldID, newID)

	return &RenameResult{
		EntityType:  entityType,
		RefsUpdated: changes, // includes the map key itself
	}, nil
}

// RenameInFragment applies a rename to a GLXFile that may hold only part of
// an archive — for example a single file from a multi-file archive. Unlike
// RenameEntity it does not require oldID to be present or newID to be free:
// the fragment's map key for oldID is moved if the fragment holds an entity
// of the given type under it, and every reference to oldID within the
// fragment is rewritten. The type is what RenameEntity reported for the
// whole archive, so an archive that (legitimately) defines the same ID under
// two entity types only has the selected one renamed. Callers are expected
// to have validated the rename against the whole archive.
//
// Only typed references are rewritten (#1440): fields declared as references
// to entityType, and properties whose definition declares that
// reference_type. A free-text property that happens to equal oldID is left
// alone. A fragment rarely carries the archive's property vocabularies, so
// they are read from archive (the whole archive, with the standard
// vocabularies filling any it does not define); pass the fragment itself
// when it is the whole archive.
//
// Returns the number of changes made (a moved map key counts as one), so a
// caller can tell whether the fragment was touched at all. Invalidates the
// validation cache when anything changed.
func RenameInFragment(glx, archive *GLXFile, entityType EntityType, oldID, newID string) int {
	changes := 0

	if probe, ok := entityIDProbes[entityType]; ok && probe(glx, oldID) {
		moveMapKey(glx, entityType, oldID, newID)
		changes++
	}

	changes += rewriteTypedReferences(glx, referenceVocabularies(archive), entityType, oldID, newID)

	if changes > 0 {
		glx.validation = nil
	}

	return changes
}

// HasEntity reports whether any entity map defines an entity with the given
// ID. Vocabulary entries are not entities and are not consulted.
func (g *GLXFile) HasEntity(id string) bool {
	_, err := findEntityType(g, id)

	return err == nil
}

// nonNilEntry reports whether m has a non-nil entry under id.
func nonNilEntry[T any](m map[string]*T, id string) bool {
	v, ok := m[id]

	return ok && v != nil
}

// entityIDProbes maps each EntityType to a closure that reports whether the
// archive's entity map for that type holds a non-nil entry for the given ID.
// Driving lookups through this table keeps findEntityType a single loop and
// localizes the per-type map-access boilerplate to one declaration.
var entityIDProbes = map[EntityType]func(*GLXFile, string) bool{
	EntityTypePersons:       func(g *GLXFile, id string) bool { return nonNilEntry(g.Persons, id) },
	EntityTypeEvents:        func(g *GLXFile, id string) bool { return nonNilEntry(g.Events, id) },
	EntityTypeRelationships: func(g *GLXFile, id string) bool { return nonNilEntry(g.Relationships, id) },
	EntityTypePlaces:        func(g *GLXFile, id string) bool { return nonNilEntry(g.Places, id) },
	EntityTypeSources:       func(g *GLXFile, id string) bool { return nonNilEntry(g.Sources, id) },
	EntityTypeCitations:     func(g *GLXFile, id string) bool { return nonNilEntry(g.Citations, id) },
	EntityTypeRepositories:  func(g *GLXFile, id string) bool { return nonNilEntry(g.Repositories, id) },
	EntityTypeAssertions:    func(g *GLXFile, id string) bool { return nonNilEntry(g.Assertions, id) },
	EntityTypeMedia:         func(g *GLXFile, id string) bool { return nonNilEntry(g.Media, id) },
	EntityTypeResearchLogs:  func(g *GLXFile, id string) bool { return nonNilEntry(g.ResearchLogs, id) },
	EntityTypeStudies:       func(g *GLXFile, id string) bool { return nonNilEntry(g.Studies, id) },
}

// findEntityType returns which entity map contains the given ID. Iterates
// AllEntityTypes in canonical order so a new entity type is picked up
// automatically when added to that list and entityIDProbes.
func findEntityType(glx *GLXFile, id string) (EntityType, error) {
	for _, t := range AllEntityTypes {
		if probe, ok := entityIDProbes[t]; ok && probe(glx, id) {
			return t, nil
		}
	}

	return "", fmt.Errorf("entity %q not found in archive", id)
}

// checkTargetFree returns an error if id already exists in any entity map.
// Case-folded collisions are a storage concern (multi-file filenames derive
// from lowercased IDs) and are left to the caller; see EntityIDIgnoringCase.
func checkTargetFree(glx *GLXFile, id string) error {
	if _, err := findEntityType(glx, id); err == nil {
		return fmt.Errorf("entity %q: %w", id, ErrEntityAlreadyExists)
	}

	return nil
}

// findKeyFold returns a key in m other than id itself whose canonical
// multi-file filename (EntityIDToFilename) equals id's, if any. Comparing
// derived filenames rather than using strings.EqualFold matches what the
// serializer does on disk; the two differ for some Unicode (e.g. "ſ" and "s"
// are EqualFold-equal but lowercase to distinct filenames). An id or key with
// no valid filename cannot collide. The exact key is skipped, not reported,
// so the answer does not depend on map iteration order when id is present.
func findKeyFold[T any](m map[string]*T, id string) (string, bool) {
	want, err := EntityIDToFilename(id)
	if err != nil {
		return "", false
	}
	for key := range m {
		if key == id {
			continue
		}
		if name, err := EntityIDToFilename(key); err == nil && name == want {
			return key, true
		}
	}

	return "", false
}

// entityIDFoldProbes mirrors entityIDProbes but matches IDs case-insensitively,
// returning the existing ID that matched.
var entityIDFoldProbes = map[EntityType]func(*GLXFile, string) (string, bool){
	EntityTypePersons:       func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Persons, id) },
	EntityTypeEvents:        func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Events, id) },
	EntityTypeRelationships: func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Relationships, id) },
	EntityTypePlaces:        func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Places, id) },
	EntityTypeSources:       func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Sources, id) },
	EntityTypeCitations:     func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Citations, id) },
	EntityTypeRepositories:  func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Repositories, id) },
	EntityTypeAssertions:    func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Assertions, id) },
	EntityTypeMedia:         func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Media, id) },
	EntityTypeResearchLogs:  func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.ResearchLogs, id) },
	EntityTypeStudies:       func(g *GLXFile, id string) (string, bool) { return findKeyFold(g.Studies, id) },
}

// EntityIDIgnoringCase returns the existing ID of the given entity type that
// lowercases to the same string as id. Multi-file archive filenames derive
// from lowercased IDs (EntityIDToFilename) and each entity type lives in its
// own directory, so a caller about to introduce an ID into such an archive
// can use this to detect the per-type collision that would otherwise surface
// as ErrCaseInsensitiveCollision at serialize time. The exact id itself is
// never reported (use HasEntity for that), so the check is well-defined
// whether or not id is already present.
func (g *GLXFile) EntityIDIgnoringCase(entityType EntityType, id string) (string, bool) {
	probe, ok := entityIDFoldProbes[entityType]
	if !ok {
		return "", false
	}

	return probe(g, id)
}

// moveMapKey moves an entity from oldID to newID in its entity map.
func moveMapKey(glx *GLXFile, entityType EntityType, oldID, newID string) {
	switch entityType {
	case EntityTypePersons:
		glx.Persons[newID] = glx.Persons[oldID]
		delete(glx.Persons, oldID)
	case EntityTypeEvents:
		glx.Events[newID] = glx.Events[oldID]
		delete(glx.Events, oldID)
	case EntityTypeRelationships:
		glx.Relationships[newID] = glx.Relationships[oldID]
		delete(glx.Relationships, oldID)
	case EntityTypePlaces:
		glx.Places[newID] = glx.Places[oldID]
		delete(glx.Places, oldID)
	case EntityTypeSources:
		glx.Sources[newID] = glx.Sources[oldID]
		delete(glx.Sources, oldID)
	case EntityTypeCitations:
		glx.Citations[newID] = glx.Citations[oldID]
		delete(glx.Citations, oldID)
	case EntityTypeRepositories:
		glx.Repositories[newID] = glx.Repositories[oldID]
		delete(glx.Repositories, oldID)
	case EntityTypeAssertions:
		glx.Assertions[newID] = glx.Assertions[oldID]
		delete(glx.Assertions, oldID)
	case EntityTypeMedia:
		glx.Media[newID] = glx.Media[oldID]
		delete(glx.Media, oldID)
	case EntityTypeResearchLogs:
		glx.ResearchLogs[newID] = glx.ResearchLogs[oldID]
		delete(glx.ResearchLogs, oldID)
	case EntityTypeStudies:
		glx.Studies[newID] = glx.Studies[oldID]
		delete(glx.Studies, oldID)
	}
}

// referenceVocabularies returns archive with the standard vocabularies filling
// any it does not define, which is what decides whether a property holds a
// reference. archive is not modified.
func referenceVocabularies(archive *GLXFile) *GLXFile {
	if archive == nil {
		archive = &GLXFile{}
	}
	prepared, err := researchArchive(archive)
	if err != nil {
		return archive // the embedded vocabularies failed to parse; use what the archive has
	}

	return prepared
}
