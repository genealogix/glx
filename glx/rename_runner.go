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
	"slices"
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

	// In a multi-file archive the new ID must be usable as a filename even
	// when the entity currently lives in a multi-entity file (and so is not
	// moved today): a later split or full rewrite would otherwise fail on an
	// ID we accepted here. Single-file archives never derive filenames.
	newName, err := glxlib.EntityIDToFilename(newID)
	if err != nil {
		return err
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

	// Multi-file filenames derive from lowercased IDs, so a new ID that
	// differs only by case from another entity's would collide on disk (the
	// serializer would refuse it as ErrCaseInsensitiveCollision on the next
	// full write). A case variant of the entity's own ID is fine.
	if existing, ok := whole.EntityIDIgnoringCase(newID); ok && existing != oldID && existing != newID {
		return fmt.Errorf("entity %q conflicts with existing %q: %w", newID, existing, glxlib.ErrCaseInsensitiveCollision)
	}

	result, err := glxlib.RenameEntity(whole, oldID, newID)
	if err != nil {
		return err
	}

	ops, err := planRenameWrites(files, result.EntityType, oldID, newID, newName)
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
//
// mode is the permission bits to write with (valid when hasMode is set),
// filled in by preflightFileOps from the existing file so a rewrite or
// rollback never widens a private file's permissions; modeFrom names the
// file a created file inherits its mode from (the source of a move). Without
// a captured mode, writes use filePermissions. hasMode is tracked separately
// so a legitimate mode of 000 is preserved rather than read as "unset".
type fileOp struct {
	relPath  string
	oldData  []byte
	newData  []byte
	mode     os.FileMode
	hasMode  bool
	modeFrom string
}

// perm returns the permission bits to write the file with.
func (op *fileOp) perm() os.FileMode {
	if op.hasMode {
		return op.mode
	}

	return filePermissions
}

// planRenameWrites parses every archive file as an independent fragment,
// applies the rename to it, and returns the file operations needed to persist
// the fragments that changed. Files the rename does not touch produce no
// operation. A single-entity file named after oldID is renamed to newName
// (the canonical filename for newID) by emitting a delete of the old path and
// a create of the new one.
func planRenameWrites(files map[string][]byte, entityType glxlib.EntityType, oldID, newID, newName string) ([]fileOp, error) {
	serializer := createSerializer(false, true, "  ")

	// Walk in sorted order so the plan (and therefore any rollback) is
	// deterministic regardless of map iteration order. Track existing paths
	// case-folded: on a case-insensitive filesystem a create onto
	// persons/person-new.glx would silently overwrite persons/Person-New.glx.
	relPaths := make([]string, 0, len(files))
	existingFold := make(map[string]string, len(files))
	for relPath := range files {
		relPaths = append(relPaths, relPath)
		existingFold[strings.ToLower(relPath)] = relPath
	}
	sort.Strings(relPaths)

	var ops []fileOp
	for _, relPath := range relPaths {
		data := files[relPath]
		fragment, err := serializer.DeserializeSingleFileBytes(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", relPath, err)
		}

		if glxlib.RenameInFragment(fragment, entityType, oldID, newID) == 0 {
			continue
		}
		// The fragment held the selected entity (not merely a reference, and
		// not a same-ID entity of another type) iff the key moved.
		movedKey := fragment.HasEntity(newID) && !fragment.HasEntity(oldID)

		newData, err := serializer.SerializeSingleFileBytes(fragment)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize %s: %w", relPath, err)
		}

		targetPath := filepath.Join(filepath.Dir(relPath), newName)
		moveFile := movedKey && fileNamedAfterEntity(relPath, oldID) && fragmentIsSingleEntity(fragment) &&
			// A case-only change of the ID maps to the same canonical filename;
			// keep the on-disk name rather than delete-and-create, which on a
			// case-insensitive filesystem would remove the file just written.
			// The basename is lowercased the way EntityIDToFilename lowercases
			// IDs (not EqualFold, which differs for some Unicode).
			strings.ToLower(filepath.Base(relPath)) != newName
		if !moveFile {
			ops = append(ops, fileOp{relPath: relPath, oldData: data, newData: newData})

			continue
		}

		// Single-entity file named after the entity: move the file with it.
		if clash, exists := existingFold[strings.ToLower(targetPath)]; exists {
			return nil, fmt.Errorf("cannot rename %s to %s (%s): %w", relPath, targetPath, clash, ErrRenameTargetFileExists)
		}
		ops = append(ops,
			fileOp{relPath: relPath, oldData: data},
			fileOp{relPath: targetPath, newData: newData, modeFrom: relPath},
		)
	}

	return ops, nil
}

// fileNamedAfterEntity reports whether the file is named after the entity
// ID the way `glx split` and the full-archive writer name files: the
// canonical (lowercased) filename derived by EntityIDToFilename. The
// basename is lowercased the same way so a hand-named Person-A.glx still
// counts; strings.ToLower rather than EqualFold because that is the exact
// transformation the filename derivation applies.
func fileNamedAfterEntity(relPath, id string) bool {
	canonical, err := glxlib.EntityIDToFilename(id)
	if err != nil {
		return false
	}

	return strings.ToLower(filepath.Base(relPath)) == canonical
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
// one goes away. Every path is checked up front (inside the archive, not a
// symlink, creates don't clobber) so nothing is written unless the whole
// plan is executable. Each write is atomic (temp file + rename), so a failed
// write leaves its target untouched. If any operation fails, every operation
// already applied is reverted from the bytes held in memory and the first
// error is returned wrapped with the rollback outcome.
//
// Rollback state lives only in memory: a crash mid-plan can leave some
// files updated and others not. That is a deliberate trade against the
// directory-swap approach (#1192); per-file writes are atomic and the
// archive is expected to be under version control for anything worse.
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

	if err := preflightFileOps(rootDir, ordered); err != nil {
		return err
	}

	for i := range ordered {
		if err := executeFileOp(rootDir, &ordered[i]); err != nil {
			if rbErr := rollbackFileOps(rootDir, ordered[:i]); rbErr != nil {
				return fmt.Errorf("%w; rollback failed, archive may be inconsistent: %w", err, rbErr)
			}

			return fmt.Errorf("%w (all changes rolled back)", err)
		}
	}

	return nil
}

// preflightFileOps rejects a plan before any write happens if a path would
// escape rootDir, an existing file to be rewritten or removed is a symlink
// (writing through it could clobber a file outside the archive; the loader
// follows symlinks when reading, so a reference-only fragment can be one), or
// a file to be created already exists on disk (catches case-insensitive
// filesystem collisions the plan's map lookup could not see).
func preflightFileOps(rootDir string, ops []fileOp) error {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolving archive root: %w", err)
	}

	modes := make(map[string]os.FileMode, len(ops))
	for i := range ops {
		op := &ops[i]
		absPath := filepath.Join(absRoot, op.relPath)
		if rel, err := filepath.Rel(absRoot, absPath); err != nil ||
			rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("%s: %w", op.relPath, ErrRenamePathEscapesArchive)
		}

		info, err := os.Lstat(absPath)
		switch {
		case err != nil && errors.Is(err, os.ErrNotExist):
			if op.oldData != nil {
				return fmt.Errorf("%s: %w", op.relPath, os.ErrNotExist)
			}
		case err != nil:
			return fmt.Errorf("checking %s: %w", op.relPath, err)
		case info.Mode()&os.ModeSymlink != 0:
			return fmt.Errorf("%s: %w", op.relPath, ErrRenameThroughSymlink)
		case op.oldData == nil:
			return fmt.Errorf("cannot create %s: %w", op.relPath, ErrRenameTargetFileExists)
		default:
			op.mode = info.Mode().Perm()
			op.hasMode = true
			modes[op.relPath] = op.mode
		}
	}

	// A created file (the target of a move) inherits its source's mode.
	for i := range ops {
		if ops[i].hasMode || ops[i].modeFrom == "" {
			continue
		}
		if mode, ok := modes[ops[i].modeFrom]; ok {
			ops[i].mode = mode
			ops[i].hasMode = true
		}
	}

	return nil
}

// executeFileOp writes op.newData to the file at the same path, or removes
// the file when newData is nil. Writes are atomic (temp file in the same
// directory, then rename) so a failure mid-write, e.g. a full disk, leaves
// the original content intact rather than a truncated file.
func executeFileOp(rootDir string, op *fileOp) error {
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
	if err := atomicWriteFile(absPath, op.newData, op.perm()); err != nil {
		return fmt.Errorf("failed to write %s: %w", op.relPath, err)
	}

	return nil
}

// rollbackFileOps restores the pre-operation state of every op, in reverse
// order. A created file that is already gone counts as restored. Continues
// past individual failures and returns them joined.
func rollbackFileOps(rootDir string, applied []fileOp) error {
	var errs []error
	for i := range slices.Backward(applied) {
		op := &applied[i]
		absPath := filepath.Join(rootDir, op.relPath)

		var err error
		if op.oldData == nil {
			if err = os.Remove(absPath); errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		} else {
			err = atomicWriteFile(absPath, op.oldData, op.perm())
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
