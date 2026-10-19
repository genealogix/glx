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
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
)

const mergeTransactionPermissions os.FileMode = 0o700

// Install only changed files, using archive-scoped renames throughout. Original
// inodes stay in an archive-local rollback area until success. Durable copies
// in the sibling .bak preserve the regular writer's recovery contract.
func installMergeFiles(directory string, original, planned map[string][]byte) error {
	return installMergeFilesWithRename(directory, original, planned, robustRenameIn)
}

func installMergeFilesWithRename(directory string, original, planned map[string][]byte, rename func(*os.Root, string, string) error) error {
	return installMergeFilesWithSync(directory, original, planned, rename, syncMergeHandle)
}

func installMergeFilesWithSync(directory string, original, planned map[string][]byte, rename func(*os.Root, string, string) error, syncHandle func(*os.File) error) error {
	destination, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	destination, err = filepath.EvalSymlinks(destination)
	if err != nil {
		return err
	}
	parent, err := os.OpenRoot(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	archive, err := parent.OpenRoot(filepath.Base(destination))
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	transaction := ".glx-merge-" + rand.Text()
	if err := archive.Mkdir(transaction, mergeTransactionPermissions); err != nil {
		return err
	}
	removeTransaction := true
	defer func() {
		if removeTransaction {
			_ = archive.RemoveAll(transaction)
		}
	}()
	paths, err := stageMergeFilesWithSync(archive, transaction, original, planned, syncHandle)
	if err != nil {
		return err
	}
	if err := verifyMergeSnapshot(archive, original); err != nil {
		return err
	}
	backupName := filepath.Base(destination) + ".bak"
	if err := removeStaleMergeBackup(parent, backupName); err != nil {
		return err
	}
	if err := parent.Mkdir(backupName, mergeTransactionPermissions); err != nil {
		return err
	}
	backup, err := parent.OpenRoot(backupName)
	if err != nil {
		_ = parent.Remove(backupName)

		return err
	}
	defer func() { _ = backup.Close() }()
	if err := copyMergeBackupWithSync(archive, backup, transaction, paths, original, syncHandle); err != nil {
		_ = parent.RemoveAll(backupName)

		return err
	}
	if err := syncPreparedMergeBackup(backup, parent, backupName, paths, syncHandle); err != nil {
		return err
	}
	if err := verifyMergeSnapshot(archive, original); err != nil {
		_ = parent.RemoveAll(backupName)

		return err
	}
	// All backup copies are durable before any live original moves. An
	// unsuccessful rollback leaves both copies and the original inodes intact.
	removeTransaction = false
	var backedUp, installed []string
	rollback := func() {
		if restoreMergeFilesDurably(archive, transaction, backedUp, installed, paths, planned, rename, syncHandle) {
			removeTransaction = true
			_ = parent.RemoveAll(backupName)
		}
	}
	backedUp, err = moveMergeOriginals(archive, transaction, paths, original, rename)
	if err != nil {
		rollback()

		return err
	}
	installed, err = placeMergeFiles(archive, transaction, paths, planned, rename)
	if err != nil {
		rollback()

		return err
	}
	if err := syncInstalledMergeDirectories(archive, transaction, paths, planned, syncHandle); err != nil {
		return fmt.Errorf("syncing installed merge; recovery data retained: %w", err)
	}
	removeTransaction = true
	if err := parent.RemoveAll(backupName); err != nil {
		return fmt.Errorf("removing completed merge backup: %w", err)
	}

	return nil
}

func moveMergeOriginals(archive *os.Root, transaction string, paths []string, original map[string][]byte, rename func(*os.Root, string, string) error) ([]string, error) {
	var moved []string
	for _, path := range paths {
		target := filepath.Join(transaction, "original", path)
		if err := archive.MkdirAll(filepath.Dir(target), dirPermissions); err != nil {
			return moved, err
		}
		if err := rename(archive, path, target); err != nil {
			return moved, fmt.Errorf("backing up %s: %w", path, err)
		}
		moved = append(moved, path)
		data, err := archive.ReadFile(target)
		if err != nil {
			return moved, fmt.Errorf("verifying moved original %s: %w", path, err)
		}
		if !bytes.Equal(data, original[path]) {
			return moved, errMergePreviewChanged
		}
	}

	return moved, nil
}

func placeMergeFiles(archive *os.Root, transaction string, paths []string, planned map[string][]byte, rename func(*os.Root, string, string) error) ([]string, error) {
	var installed []string
	for _, path := range paths {
		if _, exists := planned[path]; !exists {
			continue
		}
		if err := rename(archive, filepath.Join(transaction, "new", path), path); err != nil {
			return installed, fmt.Errorf("installing %s: %w", path, err)
		}
		installed = append(installed, path)
	}

	return installed, nil
}

func verifyMergeSnapshot(archive *os.Root, original map[string][]byte) error {
	current, err := collectMergeSnapshot(archive)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, original) {
		return errMergePreviewChanged
	}

	return nil
}

func collectMergeSnapshot(archive *os.Root) (map[string][]byte, error) {
	files := make(map[string][]byte)
	err := walkGLXFilesUnderRoot(archive, archive.Name(), ".", func(path string, data []byte, err error) error {
		if err == nil {
			files[path] = data
		}

		return err
	})

	return files, err
}

// A merge backup only owns regular GLX files. Preserve marker-bearing backups
// and all foreign/skipped entries rather than deleting potential recovery data.
func removeStaleMergeBackup(parent *os.Root, name string) error {
	info, err := parent.Lstat(name)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%w: %s is not a directory", ErrStaleBackupForeignFile, name)
	}
	backup, err := parent.OpenRoot(name)
	if err != nil {
		return err
	}
	defer func() { _ = backup.Close() }()
	if _, err := backup.Lstat(swapInProgressMarker); err == nil {
		return fmt.Errorf("%w: %s holds original files; restore any missing files before removing it", ErrInterruptedSwap, name)
	} else if !os.IsNotExist(err) {
		return err
	}
	err = fs.WalkDir(backup.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || path == "." {
			return walkErr
		}
		parts := strings.Split(path, "/")
		foreign := isDotName(entry.Name()) || entry.Type()&fs.ModeSymlink != 0
		if len(parts) == 1 {
			foreign = foreign || !isManagedTopLevel(entry)
		}
		if !entry.IsDir() {
			foreign = foreign || !entry.Type().IsRegular() || !isGLXFile(entry.Name())
		}
		if len(parts) > 2 && strings.EqualFold(parts[0], "media") && strings.EqualFold(parts[1], "files") {
			foreign = true
		}
		if foreign {
			return fmt.Errorf("%w: %s contains %q", ErrStaleBackupForeignFile, name, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return parent.RemoveAll(name)
}

func stageMergeFilesWithSync(archive *os.Root, transaction string, original, planned map[string][]byte, syncHandle func(*os.File) error) ([]string, error) {
	var paths []string
	for path, old := range original {
		next, exists := planned[path]
		if exists && bytes.Equal(old, next) {
			continue
		}
		paths = append(paths, path)
		info, err := regularMergeFileInfo(archive, path)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		target := filepath.Join(transaction, "new", path)
		if err := archive.MkdirAll(filepath.Dir(target), dirPermissions); err != nil {
			return nil, err
		}
		if err := writeDurableMergeFileWithSync(archive, target, next, info.Mode().Perm(), syncHandle); err != nil {
			return nil, err
		}
	}
	slices.Sort(paths)

	return paths, nil
}

func validateMergeFileLinks(directory string, original, planned map[string][]byte) error {
	archive, err := os.OpenRoot(directory)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	for path, data := range original {
		if next, exists := planned[path]; exists && bytes.Equal(data, next) {
			continue
		}
		if _, err := regularMergeFileInfo(archive, path); err != nil {
			return err
		}
	}

	return nil
}

func regularMergeFileInfo(archive *os.Root, path string) (os.FileInfo, error) {
	info, err := archive.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("%w: %s", errMergeLinkedFile, path)
	}
	if runtime.GOOS == "windows" {
		data, err := archive.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if target, ok := placeholderTarget(filepath.ToSlash(path), data); ok {
			if _, err := archive.ReadFile(target); err == nil && !pathHasDotComponent(target) {
				return nil, fmt.Errorf("%w: %s", errMergeLinkedFile, path)
			}
		}
	}

	return info, nil
}

func copyMergeBackupWithSync(archive, backup *os.Root, transaction string, paths []string, original map[string][]byte, syncHandle func(*os.File) error) error {
	marker := "glx: interrupted merge; original files are copied here; original inodes may remain in " + transaction + "/original\n"
	if err := writeDurableMergeFileWithSync(backup, swapInProgressMarker, []byte(marker), filePermissions, syncHandle); err != nil {
		return err
	}
	for _, path := range paths {
		info, err := archive.Stat(path)
		if err != nil {
			return err
		}
		if err := backup.MkdirAll(filepath.Dir(path), dirPermissions); err != nil {
			return err
		}
		if err := writeDurableMergeFileWithSync(backup, path, original[path], info.Mode().Perm(), syncHandle); err != nil {
			return err
		}
	}

	return nil
}

func writeDurableMergeFileWithSync(root *os.Root, path string, data []byte, mode os.FileMode, syncHandle func(*os.File) error) error {
	file, err := root.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()

		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()

		return err
	}
	if err := syncHandle(file); err != nil {
		_ = file.Close()

		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func rollbackMergeFiles(archive *os.Root, transaction string, backedUp, installed []string, rename func(*os.Root, string, string) error) bool {
	restored := true
	for _, path := range slices.Backward(installed) {
		if err := rename(archive, path, filepath.Join(transaction, "new", path)); err != nil {
			restored = false
		}
	}
	for _, path := range slices.Backward(backedUp) {
		if err := rename(archive, filepath.Join(transaction, "original", path), path); err != nil {
			restored = false
		}
	}

	return restored
}

// Sync contents on every platform. Windows read-only directory handles cannot
// be flushed by FlushFileBuffers (which requires GENERIC_WRITE); directory
// entry persistence there remains subject to the filesystem's guarantees.
// Other sync failures are never ignored, including regular-file failures.
func syncMergeHandle(file *os.File) error {
	err := file.Sync()
	if runtime.GOOS == "windows" && errors.Is(err, os.ErrPermission) {
		info, statErr := file.Stat()
		if statErr == nil && info.IsDir() {
			return nil
		}
	}

	return err
}

func syncInstalledMergeDirectories(archive *os.Root, transaction string, paths []string, planned map[string][]byte, syncHandle func(*os.File) error) error {
	entries := make([]string, 0, len(paths)*3)
	for _, path := range paths {
		entries = append(entries, path, filepath.Join(transaction, "original", path))
		if _, exists := planned[path]; exists {
			entries = append(entries, filepath.Join(transaction, "new", path))
		}
	}

	return syncMergeDirectories(archive, entries, syncHandle)
}

// Flush deepest directories first, including every ancestor through the root,
// so newly created nested recovery directories and renamed entries persist.
func syncMergeDirectories(root *os.Root, entries []string, syncHandle func(*os.File) error) error {
	directories := map[string]bool{".": true}
	for _, entry := range entries {
		for dir := filepath.Dir(entry); dir != "."; dir = filepath.Dir(dir) {
			directories[dir] = true
		}
	}
	ordered := make([]string, 0, len(directories))
	for dir := range directories {
		ordered = append(ordered, dir)
	}
	slices.SortFunc(ordered, func(a, b string) int { return strings.Compare(b, a) })
	for _, dir := range ordered {
		file, err := root.Open(dir)
		if os.IsNotExist(err) {
			continue
		} // A failed move may not have created this transaction branch.
		if err != nil {
			return err
		}
		err = syncHandle(file)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

func syncPreparedMergeBackup(backup, parent *os.Root, backupName string, paths []string, syncHandle func(*os.File) error) error {
	if err := syncMergeDirectories(backup, append(slices.Clone(paths), swapInProgressMarker), syncHandle); err != nil {
		return fmt.Errorf("syncing merge backup: %w", err)
	}
	if err := syncMergeDirectories(parent, []string{backupName}, syncHandle); err != nil {
		return fmt.Errorf("syncing merge backup parent: %w", err)
	}

	return nil
}

func restoreMergeFilesDurably(archive *os.Root, transaction string, backedUp, installed, paths []string, planned map[string][]byte, rename func(*os.Root, string, string) error, syncHandle func(*os.File) error) bool {
	return rollbackMergeFiles(archive, transaction, backedUp, installed, rename) && syncInstalledMergeDirectories(archive, transaction, paths, planned, syncHandle) == nil
}
