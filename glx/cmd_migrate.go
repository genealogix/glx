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

var migrateRenameGenderToSex bool

var migrateCmd = &cobra.Command{
	Use:   "migrate [archive]",
	Short: "Migrate an archive to the current format",
	Long: `Converts deprecated person properties (born_on, born_at, died_on, died_at, buried_on, buried_at) to birth/death/burial events.

For each person with deprecated properties:
- Creates a birth, death, or burial event if none exists
- Merges date/place into existing events if fields are empty
- Never overwrites existing event data
- Converts assertions to reference the event instead of the person property

With --rename-gender-to-sex, also renames the legacy ` + "`gender`" + ` person
property (and any related assertions and inlined vocabulary entries) to
` + "`sex`" + `, completing the two-field-model split introduced in #528.`,
	Example: `  # Migrate a multi-file archive
  glx migrate ./my-archive

  # Migrate a single-file archive
  glx migrate archive.glx

  # Also rename legacy 'gender' person properties to 'sex'
  glx migrate ./my-archive --rename-gender-to-sex`,
	Args: cobra.ExactArgs(1),
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateRenameGenderToSex, "rename-gender-to-sex", false,
		"Rename the legacy 'gender' person property to 'sex' (two-field-model split, #528)")
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

	report, err := migrateVitalEventProperties(archive)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	if migrateRenameGenderToSex {
		genderReport := migrateGenderToSex(archive, os.Stderr)
		report.PropertiesRenamed += genderReport.PropertiesRenamed
		report.AssertionsRenamed += genderReport.AssertionsRenamed
		report.VocabEntriesRenamed += genderReport.VocabEntriesRenamed
		report.GenderRenameSkipped = genderReport.GenderRenameSkipped
	}

	// If the gender→sex rename was skipped, count any remaining legacy
	// `gender:` person properties so the user knows whether the skip was
	// benign (post-migration re-run, no legacy left) or worrying (manual
	// partial migration leaves legacy data unmigrated).
	//
	// Only values that would plausibly be pre-split recorded sex are
	// counted — identity values like `nonbinary` are legitimate post-split
	// data and must not be misclassified as "needs manual migration".
	legacyGenderRemaining := 0
	if report.GenderRenameSkipped {
		for _, person := range archive.Persons {
			if person == nil {
				continue
			}
			val, ok := person.Properties[glxlib.PersonPropertyGender]
			if !ok {
				continue
			}
			if s, ok := val.(string); ok && isLegacySexValue(s) {
				legacyGenderRemaining++
			}
		}
	}

	if report.EventsCreated == 0 && report.EventsMerged == 0 &&
		report.PropertiesRemoved == 0 && report.AssertionsMigrated == 0 &&
		report.VocabEntriesRemoved == 0 &&
		report.PropertiesRenamed == 0 && report.AssertionsRenamed == 0 &&
		report.VocabEntriesRenamed == 0 {
		if report.GenderRenameSkipped {
			if legacyGenderRemaining > 0 {
				noun, verb := "properties", "remain"
				if legacyGenderRemaining == 1 {
					noun, verb = "property", "remains"
				}
				fmt.Printf("Gender→sex rename skipped (archive already post-split) but %d legacy `gender` %s %s unmigrated; rename manually to preserve intent.\n",
					legacyGenderRemaining, noun, verb)
			} else {
				fmt.Println("Gender→sex rename skipped (archive already post-split); no legacy `gender` properties remain.")
			}

			return nil
		}
		fmt.Println("No deprecated properties found. Archive is already up to date.")
		return nil
	}

	// Write the migrated archive back.
	if isDir {
		if err := safeWriteMultiFileArchive(archivePath, archive); err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}
	} else {
		if err := writeSingleFileArchive(archivePath, archive, false); err != nil {
			return fmt.Errorf("failed to write archive: %w", err)
		}
	}

	fmt.Println("Migration complete:")
	fmt.Printf("  %-27s%d\n", "Events created:", report.EventsCreated)
	fmt.Printf("  %-27s%d\n", "Events merged:", report.EventsMerged)
	fmt.Printf("  %-27s%d\n", "Properties removed:", report.PropertiesRemoved)
	fmt.Printf("  %-27s%d\n", "Assertions migrated:", report.AssertionsMigrated)
	fmt.Printf("  %-27s%d\n", "Vocab entries removed:", report.VocabEntriesRemoved)
	if migrateRenameGenderToSex {
		if report.GenderRenameSkipped {
			if legacyGenderRemaining > 0 {
				noun, verb := "properties", "remain"
				if legacyGenderRemaining == 1 {
					noun, verb = "property", "remains"
				}
				fmt.Printf("  Gender→sex rename:         skipped (archive post-split; %d legacy `gender` %s %s)\n",
					legacyGenderRemaining, noun, verb)
			} else {
				fmt.Println("  Gender→sex rename:         skipped (archive already post-split; no legacy data)")
			}
		} else {
			fmt.Printf("  %-27s%d\n", "Gender→sex properties:", report.PropertiesRenamed)
			fmt.Printf("  %-27s%d\n", "Gender→sex assertions:", report.AssertionsRenamed)
			fmt.Printf("  %-27s%d\n", "Gender→sex vocab entries:", report.VocabEntriesRenamed)
		}
	}

	return nil
}
