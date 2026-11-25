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

	glxlib "github.com/genealogix/glx/go-glx"
)

// findDuplicates loads an archive and scans for potential duplicate persons.
// errPersonNotInArchive reports a person filter that names no person.
var errPersonNotInArchive = errors.New("not found in archive")

func findDuplicates(archivePath string, threshold float64, personFilter string, jsonOutput bool) error {
	if threshold < 0.0 || threshold > 1.0 {
		return fmt.Errorf("--threshold must be between 0.0 and 1.0, got %.2f", threshold)
	}

	archive, err := loadArchiveForRead(SystemIOStreams(), archivePath)
	if err != nil {
		return err
	}

	if personFilter != "" {
		if _, ok := archive.Persons[personFilter]; !ok {
			return fmt.Errorf("person %q %w%s", personFilter, errPersonNotInArchive, archivePathHint("duplicates", personFilter))
		}
	}

	opts := glxlib.DuplicateOptions{
		Threshold:    threshold,
		PersonFilter: personFilter,
	}

	result, err2 := glxlib.FindDuplicates(archive, opts)
	if err2 != nil {
		return err2
	}

	if jsonOutput {
		return printDuplicatesJSON(result)
	}

	printDuplicatesText(result, archive)

	return nil
}

// printDuplicatesText prints duplicate pairs in a human-readable format.
func printDuplicatesText(result *glxlib.DuplicateResult, archive *glxlib.GLXFile) {
	if len(result.Pairs) == 0 {
		termPrintf("No potential duplicates found (threshold: %.2f).\n", result.Threshold)

		return
	}

	termPrintf("Potential duplicates (threshold: %.2f):\n", result.Threshold)

	for _, pair := range result.Pairs {
		termPrintln()

		nameA := personLabel(archive, pair.PersonA)
		nameB := personLabel(archive, pair.PersonB)

		termPrintf("  %-38s  %s\n", pair.PersonA, nameA)
		termPrintf("  %-38s  %s\n", pair.PersonB, nameB)
		termPrintf("  Score: %.2f\n", pair.Score)

		for _, sig := range pair.Signals {
			termPrintf("    %-26s  %.2f  (%s)\n", sig.Name+":", sig.Score, sig.Detail)
		}
	}

	termPrintf("\nFound %d potential duplicate pair(s).\n", len(result.Pairs))
}

// personLabel returns "Name (b. YEAR)" for display.
func personLabel(archive *glxlib.GLXFile, personID string) string {
	person, ok := archive.Persons[personID]
	if !ok || person == nil {
		return personID
	}

	name := glxlib.PersonDisplayName(person)
	if name == "" {
		return personID
	}

	if _, birthEvent := glxlib.FindPersonEvent(archive, personID, glxlib.EventTypeBirth); birthEvent != nil && birthEvent.Date != "" {
		return fmt.Sprintf("%s (b. %s)", name, string(birthEvent.Date))
	}

	return name
}

// printDuplicatesJSON outputs the result as JSON.
func printDuplicatesJSON(result *glxlib.DuplicateResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	machinePrintln(string(data))

	return nil
}
