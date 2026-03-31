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

	"github.com/spf13/cobra"

	glxlib "github.com/genealogix/glx/go-glx"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [archive]",
	Short: "Migrate an archive to the current format",
	Long: `Converts deprecated person properties (born_on, born_at, died_on, died_at) to birth/death events.

For each person with deprecated properties:
- Creates a birth or death event if none exists
- Merges date/place into existing events if fields are empty
- Never overwrites existing event data
- Converts assertions to reference the event instead of the person property`,
	Example: `  # Migrate a multi-file archive
  glx migrate ./my-archive

  # Migrate a single-file archive
  glx migrate archive.glx`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrate,
}

func runMigrate(_ *cobra.Command, args []string) error {
	return migrateArchive(args[0])
}

// migrateArchive loads, migrates, saves, and prints a report for the given archive path.
func migrateArchive(archivePath string) error {
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

	report, err := migrateBirthDeathProperties(archive)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if report.EventsCreated == 0 && report.EventsMerged == 0 &&
		report.PropertiesRemoved == 0 && report.AssertionsMigrated == 0 &&
		report.VocabEntriesRemoved == 0 {
		fmt.Println("No deprecated properties found. Archive is already up to date.")
		return nil
	}

	// Write the migrated archive back.
	if isDir {
		if err := clearEntityFiles(archivePath); err != nil {
			return fmt.Errorf("failed to clear old entity files: %w", err)
		}
		if err := writeMultiFileArchive(archivePath, archive, false); err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}
	} else {
		if err := writeSingleFileArchive(archivePath, archive, false); err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}
	}

	fmt.Println("Migration complete:")
	fmt.Printf("  Events created:       %d\n", report.EventsCreated)
	fmt.Printf("  Events merged:        %d\n", report.EventsMerged)
	fmt.Printf("  Properties removed:   %d\n", report.PropertiesRemoved)
	fmt.Printf("  Assertions migrated:  %d\n", report.AssertionsMigrated)
	fmt.Printf("  Vocab entries removed: %d\n", report.VocabEntriesRemoved)

	return nil
}
