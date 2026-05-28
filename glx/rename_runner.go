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
	"fmt"
	"os"

	glxlib "github.com/genealogix/glx/go-glx"
)

var renameDryRun bool

// renameEntities performs the rename operation: load archive, rename, save.
// For multi-file archives, the entire archive is re-serialized to ensure
// filenames and entity IDs stay consistent (old entity files are replaced).
func renameEntities(archivePath, oldID, newID string, dryRun bool) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("cannot access path: %w", err)
	}

	var archive *glxlib.GLXFile
	isDir := info.IsDir()

	if isDir {
		loaded, duplicates, loadErr := LoadArchiveWithOptions(archivePath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		archive = loaded
	} else {
		loaded, loadErr := readSingleFileArchive(archivePath, false)
		if loadErr != nil {
			return loadErr
		}
		archive = loaded
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

	if isDir {
		return safeWriteMultiFileArchive(archivePath, archive)
	}
	return writeSingleFileArchive(archivePath, archive, false)
}
