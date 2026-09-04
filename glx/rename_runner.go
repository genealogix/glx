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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

var renameDryRun bool

// renameEntities performs the rename operation: load archive, rename, save.
//
// A multi-file archive is loaded in full so the rename can be validated
// against every entity (old ID present, new ID free), but only the files
// that actually reference the old ID are rewritten — in place, keeping the
// archive's file layout (multi-entity files stay multi-entity, unrelated
// files are untouched byte-for-byte, the directory is never swapped). If a
// single-entity file is named after the old ID it is renamed to match the
// new one. See genealogix/glx#1192 and genealogix/glx#1197.
func renameEntities(archivePath, oldID, newID string, dryRun bool) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	if !info.IsDir() {
		return renameInSingleFile(archivePath, oldID, newID, dryRun)
	}

	files, err := collectGLXFilesFromDir(archivePath)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}

	// Validate against the whole archive. The merged archive is discarded
	// afterwards; the per-file fragments below are what get written.
	whole, duplicates, err := createSerializer(false, false, "").DeserializeMultiFileFromMap(files)
	if err != nil {
		return fmt.Errorf("failed to load archive: %w", err)
	}
	for _, d := range duplicates {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
	}

	result, err := glxlib.RenameEntity(whole, oldID, newID)
	if err != nil {
		return err
	}

	ops, err := planRenameWrites(files, oldID, newID)
	if err != nil {
		return err
	}

	fmt.Printf("Renaming %s → %s (%s)\n", oldID, newID, result.EntityType)
	fmt.Printf("  Updated %d reference(s) in %d file(s)\n", result.RefsUpdated, countTouchedFiles(ops))

	if dryRun {
		fmt.Println("\n(dry run — no files written)")

		return nil
	}

	return applyFileOps(archivePath, ops)
}

// renameInSingleFile handles the single-file archive form of glx rename.
func renameInSingleFile(archivePath, oldID, newID string, dryRun bool) error {
	archive, err := readSingleFileArchive(archivePath, false)
	if err != nil {
		return err
	}

	result, err := glxlib.RenameEntity(archive, oldID, newID)
	if err != nil {
		return err
	}

	fmt.Printf("Renaming %s → %s (%s)\n", oldID, newID, result.EntityType)
	fmt.Printf("  Updated %d reference(s)\n", result.RefsUpdated)

	if dryRun {
		fmt.Println("\n(dry run — no files written)")

		return nil
	}

	return writeSingleFileArchive(archivePath, archive, false)
}

// fileOp is one planned change to a file in the archive. oldData is the
// content on disk before the operation (nil when the file did not exist) and
// newData the content afterwards (nil when the file is to be removed). Keeping
// both lets applyFileOps undo every completed operation if a later one fails.
type fileOp struct {
	relPath string
	oldData []byte
	newData []byte
}

// planRenameWrites parses every archive file as an independent fragment,
// applies the rename to it, and returns the file operations needed to persist
// the fragments that changed. Files the rename does not touch produce no
// operation. A single-entity file named after oldID is renamed to newID by
// emitting a delete of the old path and a create of the new one.
func planRenameWrites(files map[string][]byte, oldID, newID string) ([]fileOp, error) {
	serializer := createSerializer(false, true, "  ")

	// Walk in sorted order so the plan (and therefore any rollback) is
	// deterministic regardless of map iteration order.
	relPaths := make([]string, 0, len(files))
	for relPath := range files {
		relPaths = append(relPaths, relPath)
	}
	sort.Strings(relPaths)

	var ops []fileOp
	for _, relPath := range relPaths {
		data := files[relPath]
		fragment, err := serializer.DeserializeSingleFileBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}

		holdsEntity := fragment.HasEntity(oldID)
		if glxlib.RenameInFragment(fragment, oldID, newID) == 0 {
			continue
		}

		newData, err := serializer.SerializeSingleFileBytes(fragment)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}

		if !holdsEntity || !fileNamedAfterEntity(relPath, oldID) || !fragmentIsSingleEntity(fragment) {
			ops = append(ops, fileOp{relPath: relPath, oldData: data, newData: newData})

			continue
		}

		// Single-entity file named after the entity: move the file with it.
		newName, err := glxlib.EntityIDToFilename(newID)
		if err != nil {
			return nil, err
		}
		targetPath := filepath.Join(filepath.Dir(relPath), newName)
		if _, exists := files[targetPath]; exists {
			return nil, fmt.Errorf("cannot rename %s to %s: %w", relPath, targetPath, ErrRenameTargetFileExists)
		}
		ops = append(ops,
			fileOp{relPath: relPath, oldData: data},
			fileOp{relPath: targetPath, newData: newData},
		)
	}

	return ops, nil
}

// fileNamedAfterEntity reports whether the file's basename (without the
// .glx extension) is the entity ID, which is the layout `glx split` and the
// full-archive writer produce.
func fileNamedAfterEntity(relPath, id string) bool {
	base := filepath.Base(relPath)

	return strings.TrimSuffix(base, FileExtGLX) == id && strings.HasSuffix(base, FileExtGLX)
}

// countTouchedFiles counts distinct paths across the planned operations.
func countTouchedFiles(ops []fileOp) int {
	seen := make(map[string]struct{}, len(ops))
	for _, op := range ops {
		seen[op.relPath] = struct{}{}
	}

	return len(seen)
}

// applyFileOps executes the planned operations in order. Creates and
// rewrites happen before deletes so a rename's new file exists before its old
// one goes away. If any operation fails, every operation already applied is
// reverted from the bytes held in memory and the first error is returned
// wrapped with the rollback outcome.
func applyFileOps(rootDir string, ops []fileOp) error {
	ordered := make([]fileOp, 0, len(ops))
	for _, op := range ops {
		if op.newData != nil {
			ordered = append(ordered, op)
		}
	}
	for _, op := range ops {
		if op.newData == nil {
			ordered = append(ordered, op)
		}
	}

	for i, op := range ordered {
		if err := executeFileOp(rootDir, op); err != nil {
			if rbErr := rollbackFileOps(rootDir, ordered[:i]); rbErr != nil {
				return fmt.Errorf("%w; rollback failed, archive may be inconsistent: %w", err, rbErr)
			}

			return fmt.Errorf("%w (all changes rolled back)", err)
		}
	}

	return nil
}

// executeFileOp writes op.newData to the file, or removes the file when
// newData is nil. Writes go through the same path in place so the file keeps
// its identity for editors and shells that have it open.
func executeFileOp(rootDir string, op fileOp) error {
	absPath := filepath.Join(rootDir, op.relPath)

	if op.newData == nil {
		if err := os.Remove(absPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", op.relPath, err)
		}

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", op.relPath, err)
	}
	if err := os.WriteFile(absPath, op.newData, filePermissions); err != nil {
		return fmt.Errorf("failed to write %s: %w", op.relPath, err)
	}

	return nil
}

// rollbackFileOps restores the pre-operation state of every op, in reverse
// order. Continues past individual failures and returns them joined.
func rollbackFileOps(rootDir string, applied []fileOp) error {
	var errs []error
	for i := len(applied) - 1; i >= 0; i-- {
		op := applied[i]
		absPath := filepath.Join(rootDir, op.relPath)

		var err error
		if op.oldData == nil {
			err = os.Remove(absPath)
		} else {
			err = os.WriteFile(absPath, op.oldData, filePermissions)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("restore %s: %w", op.relPath, err))
		}
	}

	return errors.Join(errs...)
}

// fragmentIsSingleEntity reports whether the fragment defines exactly one
// entity across all entity types.
func fragmentIsSingleEntity(fragment *glxlib.GLXFile) bool {
	counts := countEntities(fragment)

	return counts.Total() == 1
}
