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

// mergePersons performs the merge: load archive, fold drop into keep, save.
// Conflicts are reported on stderr and do not abort the merge — the safe
// default keeps keep's value, and the user explicitly named which person to
// retain.
func mergePersons(archivePath, keepID, dropID string, opts glxlib.MergePersonsOptions, dryRun bool) error {
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

	result, err := glxlib.MergePersons(archive, keepID, dropID, opts)
	if err != nil {
		return err
	}

	fmt.Printf("Merging %s ← %s\n", keepID, dropID)
	fmt.Printf("  Properties merged:    %d\n", result.PropertiesMerged)
	fmt.Printf("  Notes merged:         %d\n", result.NotesMerged)
	fmt.Printf("  References rewritten: %d\n", result.RefsUpdated)

	for _, c := range result.Conflicts {
		fmt.Fprintf(os.Stderr, "  Conflict on %q: keep=%v drop=%v (%s)\n",
			c.Property, c.KeepValue, c.DropValue, c.Resolution)
	}

	if dryRun {
		fmt.Println("\n(dry run — no files written)")

		return nil
	}

	if isDir {
		return safeWriteMultiFileArchive(archivePath, archive)
	}

	return writeSingleFileArchive(archivePath, archive, false)
}
