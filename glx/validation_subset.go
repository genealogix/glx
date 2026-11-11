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
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// validateArchiveSubset validates selected — the files named on the command
// line, keyed relative to archiveRoot — as part of the archive at archiveRoot
// (#1206). The selected files are schema-checked themselves; the whole archive
// is then loaded so their references, types and properties resolve the way
// they do when the archive is validated whole, and only findings about the
// selected entities are reported. Before this, `glx validate events` loaded
// events/ alone and reported every person, place and event type it pointed at
// as missing.
func validateArchiveSubset(streams *IOStreams, archiveRoot string, selected map[string][]byte, showFirstErrors int) error {
	// The archive walk keys files with forward slashes; an explicitly named
	// file is keyed with the OS separator. Use one spelling, or the same file
	// would be loaded twice and reported as a duplicate of itself.
	selected = slashKeys(selected)
	if err := schemaValidateFiles(archiveRoot, selected); err != nil {
		reportArchiveLoadError(streams, err, showFirstErrors)

		return ErrStructuralValidationFailed
	}

	contextFiles, err := collectGLXFilesFromDir(archiveRoot)
	if err != nil {
		return validateSubsetAlone(streams, archiveRoot, selected, err, showFirstErrors)
	}
	contextFiles = slashKeys(contextFiles)
	// The selection is read through the same root, but an explicitly named
	// file can be one the archive walk skips; make sure every selected file
	// is part of the context it is checked against.
	maps.Copy(contextFiles, selected)

	archive, duplicates, err := loadArchiveFromFiles(archiveRoot, contextFiles, false)
	if err != nil {
		return validateSubsetAlone(streams, archiveRoot, selected, err, showFirstErrors)
	}
	selection, _, err := loadArchiveFromFiles(archiveRoot, selected, false)
	if err != nil {
		reportArchiveLoadError(streams, err, showFirstErrors)

		return ErrStructuralValidationFailed
	}

	inSelection := selectedEntityIDs(selection)
	selectedPaths := make([]string, 0, len(selected))
	for rel := range selected {
		selectedPaths = append(selectedPaths, rel)
	}

	var allErrors, allWarnings []string
	for _, d := range duplicates {
		if mentionsAnyPath(d, selectedPaths) {
			allErrors = append(allErrors, d)
		}
	}
	slices.Sort(allErrors)

	result := archive.Validate()
	for _, e := range result.Errors {
		if inSelection.covers(e.SourceType, e.SourceID) {
			allErrors = append(allErrors, e.Message)
		}
	}
	for _, w := range result.Warnings {
		if inSelection.covers(w.SourceType, w.SourceID) {
			allWarnings = append(allWarnings, w.Message)
		}
	}
	allWarnings = append(allWarnings, validateMediaFileExistence(selection, archiveRoot)...)

	streams.Printf("References and vocabularies resolved against the archive at %s.\n", archiveRoot)

	return reportArchiveValidation(streams, len(selected), allErrors, allWarnings, showFirstErrors)
}

func slashKeys(files map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(files))
	for rel, data := range files {
		out[filepath.ToSlash(rel)] = data
	}

	return out
}

// validateSubsetAlone validates the selected files as an archive of their own
// — what subset validation did before #1206 — when the archive around them
// cannot be read: a file elsewhere in it that is malformed, or a symlink that
// leaves it. Failing the named files for a file nobody named would make the
// subset unvalidatable until that file is fixed, so the reason is reported as
// a warning and the selection is still checked; references outside it are
// then errors. Validating the whole archive reports the unreadable file
// itself.
func validateSubsetAlone(streams *IOStreams, archiveRoot string, selected map[string][]byte, contextErr error, showFirstErrors int) error {
	streams.Errorf("⚠️  Could not read the rest of the archive at %s, so the named files are validated on their own "+
		"(references outside them are errors): %s\n", archiveRoot, sanitizeForTerminal(contextErr.Error()))

	return validateFilesAsArchive(streams, archiveRoot, selected, showFirstErrors)
}

// entityIDSet is the set of entities a selection defines, by entity type.
type entityIDSet map[glxlib.EntityType]map[string]bool

func selectedEntityIDs(selection *glxlib.GLXFile) entityIDSet {
	set := entityIDSet{}
	add := func(entityType glxlib.EntityType, ids []string) {
		set[entityType] = map[string]bool{}
		for _, id := range ids {
			set[entityType][id] = true
		}
	}
	add(glxlib.EntityTypePersons, mapKeys(selection.Persons))
	add(glxlib.EntityTypeRelationships, mapKeys(selection.Relationships))
	add(glxlib.EntityTypeEvents, mapKeys(selection.Events))
	add(glxlib.EntityTypePlaces, mapKeys(selection.Places))
	add(glxlib.EntityTypeSources, mapKeys(selection.Sources))
	add(glxlib.EntityTypeCitations, mapKeys(selection.Citations))
	add(glxlib.EntityTypeRepositories, mapKeys(selection.Repositories))
	add(glxlib.EntityTypeAssertions, mapKeys(selection.Assertions))
	add(glxlib.EntityTypeMedia, mapKeys(selection.Media))
	add(glxlib.EntityTypeResearchLogs, mapKeys(selection.ResearchLogs))
	add(glxlib.EntityTypeStudies, mapKeys(selection.Studies))

	return set
}

// covers reports whether a finding about sourceID of sourceType concerns the
// selection. Findings about an entity's participants carry a suffix after
// the ID ("event-1 participants[0]"), so only the leading ID is compared. A
// finding that is not about an entity — a vocabulary definition, say — is
// always reported: the selection's vocabularies are part of what was asked
// for.
func (s entityIDSet) covers(sourceType glxlib.EntityType, sourceID string) bool {
	ids, isEntity := s[sourceType]
	if !isEntity {
		return true
	}
	id, _, _ := strings.Cut(sourceID, " ")

	return ids[id]
}

func mapKeys[T any](m map[string]*T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

// mentionsAnyPath reports whether a duplicate-ID conflict, which names its two
// files as "(a vs b)", involves one of paths.
func mentionsAnyPath(conflict string, paths []string) bool {
	for _, p := range paths {
		if strings.Contains(conflict, "("+p+" vs ") || strings.Contains(conflict, " vs "+p+")") {
			return true
		}
	}

	return false
}

// sameDirectory reports whether a and b name the same directory once made
// absolute. Windows paths compare case-insensitively, as commonArchiveRoot
// compares them.
func sameDirectory(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return a == b
	}
	if runtime.GOOS == goosWindows {
		return strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
	}
	if filepath.Clean(absA) == filepath.Clean(absB) {
		return true
	}
	infoA, errA := os.Stat(absA)
	infoB, errB := os.Stat(absB)

	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}
