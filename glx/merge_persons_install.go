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
	"os"
	"path/filepath"
	"reflect"
	"slices"
)

// Install only the files changed by the verified plan. Recursive archives may
// use arbitrary GLX paths; their directories, foreign files and skipped entries
// stay in place. As with the regular writer, each rename is atomic and a durable
// marker protects original files in .bak until installation finishes.
func installMergeFiles(directory string, original, planned map[string][]byte) error {
	return installMergeFilesWithRename(directory, original, planned, robustRename)
}

func installMergeFilesWithRename(directory string, original, planned map[string][]byte, rename func(string, string) error) error {
	destination, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	destination, err = filepath.EvalSymlinks(destination)
	if err != nil {
		return err
	}
	staged, err := os.MkdirTemp(filepath.Dir(destination), ".glx-merge-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staged) }()
	paths, err := stageMergeFiles(destination, staged, original, planned)
	if err != nil {
		return err
	}
	// Recheck after staging too, immediately before moving any original file.
	current, err := collectGLXFilesFromDir(destination)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(current, original) {
		return errMergePreviewChanged
	}
	backup := destination + ".bak"
	if err := removeStaleBackup(backup); err != nil {
		return err
	}
	if err := os.Mkdir(backup, dirPermissions); err != nil {
		return err
	}
	marker := filepath.Join(backup, swapInProgressMarker)
	if err := writeDurableMarker(marker, "glx: interrupted merge in "+destination+"\n"); err != nil {
		_ = os.Remove(marker)
		_ = os.Remove(backup)

		return err
	}
	var backedUp, installed []string
	rollback := func() { rollbackMergeFiles(destination, staged, backup, backedUp, installed, rename) }
	for _, path := range paths {
		target := filepath.Join(backup, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), dirPermissions); err != nil {
			rollback()

			return err
		}
		if err := rename(filepath.Join(destination, filepath.FromSlash(path)), target); err != nil {
			rollback()

			return fmt.Errorf("backing up %s: %w", path, err)
		}
		backedUp = append(backedUp, path)
	}
	for _, path := range paths {
		if _, exists := planned[path]; !exists {
			continue
		}
		if err := rename(filepath.Join(staged, filepath.FromSlash(path)), filepath.Join(destination, filepath.FromSlash(path))); err != nil {
			rollback()

			return fmt.Errorf("installing %s: %w", path, err)
		}
		installed = append(installed, path)
	}
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("removing completed merge backup: %w", err)
	}

	return nil
}

func stageMergeFiles(destination, staged string, original, planned map[string][]byte) ([]string, error) {
	var paths []string
	for path, old := range original {
		next, exists := planned[path]
		if exists && bytes.Equal(old, next) {
			continue
		}
		paths = append(paths, path)
		if !exists {
			continue
		}
		info, err := os.Stat(filepath.Join(destination, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		target := filepath.Join(staged, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), dirPermissions); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, next, info.Mode().Perm()); err != nil {
			return nil, err
		}
	}
	slices.Sort(paths)

	return paths, nil
}

func rollbackMergeFiles(destination, staged, backup string, backedUp, installed []string, rename func(string, string) error) {
	restored := true
	for _, path := range slices.Backward(installed) {
		if err := rename(filepath.Join(destination, filepath.FromSlash(path)), filepath.Join(staged, filepath.FromSlash(path))); err != nil {
			restored = false
		}
	}
	for _, path := range slices.Backward(backedUp) {
		if err := rename(filepath.Join(backup, filepath.FromSlash(path)), filepath.Join(destination, filepath.FromSlash(path))); err != nil {
			restored = false
		}
	}
	if restored {
		_ = os.RemoveAll(backup)
	}
}
