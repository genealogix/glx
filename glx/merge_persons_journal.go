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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const mergeJournalPermissions os.FileMode = 0o600

// Use the general writer's recovery format so every archive writer recognizes
// an interrupted merge. Exclusive creation preserves any existing journal;
// callers still require exclusive archive access throughout the operation.
func prepareMergeJournal(archive *os.Root, paths []string, original map[string][]byte, syncHandle func(*os.File) error) (func() error, error) {
	entries := make([]inPlaceJournalEntry, 0, len(paths))
	for _, path := range paths {
		entries = append(entries, inPlaceJournalEntry{Path: filepath.ToSlash(path), Existed: true, Data: original[path]})
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	file, err := createMergeFile(archive, inPlaceJournalFile, mergeJournalPermissions, nil, "")
	if err != nil {
		return nil, fmt.Errorf("creating merge recovery journal: %w", err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspecting merge recovery journal; original files unchanged: %w", err)
	}
	finish := func() error { return removeMergeJournal(archive, info, data, syncHandle) }
	if err := persistMergeJournal(archive, file, data, syncHandle); err != nil {
		// No live original has moved. Capture the local cleanup closure rather
		// than a named result that an error return would replace with nil.
		return nil, errors.Join(err, finish())
	}

	return finish, nil
}

func persistMergeJournal(archive *os.Root, file *os.File, data []byte, syncHandle func(*os.File) error) error {
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("writing merge recovery journal: %w", err)
	}
	if err := syncHandle(file); err != nil {
		return fmt.Errorf("syncing merge recovery journal: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("closing merge recovery journal: %w", err)
	}
	if err := syncMergeDirectories(archive, []string{inPlaceJournalFile}, syncHandle); err != nil {
		return fmt.Errorf("syncing merge recovery journal directory: %w", err)
	}

	return nil
}

func removeMergeJournal(archive *os.Root, owned os.FileInfo, data []byte, syncHandle func(*os.File) error) error {
	current, err := archive.Lstat(inPlaceJournalFile)
	if err != nil {
		return fmt.Errorf("checking merge recovery journal: %w", err)
	}
	if !os.SameFile(current, owned) {
		return fmt.Errorf("%w: merge recovery journal changed; inspect it before retrying", errMergePreviewChanged)
	}
	if err := archive.Remove(inPlaceJournalFile); err != nil {
		return fmt.Errorf("removing merge recovery journal: %w", err)
	}
	if err := syncMergeDirectories(archive, []string{inPlaceJournalFile}, syncHandle); err != nil {
		// An unsuccessful directory sync leaves deletion durability unknown.
		// Restore the recovery signal, using exclusive creation again so a
		// replacement belonging to another writer is never overwritten.
		return errors.Join(fmt.Errorf("syncing merge recovery journal removal: %w", err),
			restoreMergeJournal(archive, data, syncHandle))
	}

	return nil
}

func restoreMergeJournal(archive *os.Root, data []byte, syncHandle func(*os.File) error) error {
	file, err := createMergeFile(archive, inPlaceJournalFile, mergeJournalPermissions, nil, "")
	if err != nil {
		return fmt.Errorf("restoring merge recovery journal: %w", err)
	}
	defer func() { _ = file.Close() }()

	return persistMergeJournal(archive, file, data, syncHandle)
}
