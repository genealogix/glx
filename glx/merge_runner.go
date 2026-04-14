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
	"path/filepath"

	glxlib "github.com/genealogix/glx/go-glx"
)

const defaultMergeThreshold = 0.6

// mergeResult holds statistics from a merge operation.
type mergeResult struct {
	Conflicts        []string
	IdenticalSkipped int
	NewPersons       int
	NewEvents        int
	NewRelationships int
	NewPlaces        int
	NewSources       int
	NewCitations     int
	NewRepositories  int
	NewAssertions    int
	NewMedia         int
}

// TotalNew returns the total number of new entities merged.
func (r *mergeResult) TotalNew() int {
	return r.NewPersons + r.NewEvents + r.NewRelationships + r.NewPlaces +
		r.NewSources + r.NewCitations + r.NewRepositories + r.NewAssertions + r.NewMedia
}

// mergeArchivesInMemory merges src into dest, returning statistics.
func mergeArchivesInMemory(dest, src *glxlib.GLXFile) *mergeResult {
	// Snapshot counts before merge
	before := entityCounts(dest)

	conflicts, identicalSkipped := dest.Merge(src)

	// Compute new entities added
	after := entityCounts(dest)

	return &mergeResult{
		Conflicts:        conflicts,
		IdenticalSkipped: identicalSkipped,
		NewPersons:       after.persons - before.persons,
		NewEvents:        after.events - before.events,
		NewRelationships: after.relationships - before.relationships,
		NewPlaces:        after.places - before.places,
		NewSources:       after.sources - before.sources,
		NewCitations:     after.citations - before.citations,
		NewRepositories:  after.repositories - before.repositories,
		NewAssertions:    after.assertions - before.assertions,
		NewMedia:         after.media - before.media,
	}
}

type counts struct {
	persons, events, relationships, places int
	sources, citations, repositories       int
	assertions, media                      int
}

func entityCounts(g *glxlib.GLXFile) counts {
	return counts{
		persons:       len(g.Persons),
		events:        len(g.Events),
		relationships: len(g.Relationships),
		places:        len(g.Places),
		sources:       len(g.Sources),
		citations:     len(g.Citations),
		repositories:  len(g.Repositories),
		assertions:    len(g.Assertions),
		media:         len(g.Media),
	}
}

// mergeArchives loads two archives, merges src into dest, and saves.
func mergeArchives(srcPath, destPath string, preview bool, threshold float64) error {
	// Resolve to absolute paths so "." becomes a real path that os.Rename can
	// operate on. POSIX forbids renaming "." (EINVAL) and Windows rejects it
	// with ERROR_SHARING_VIOLATION.
	var err error
	destPath, err = filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}
	srcPath, err = filepath.Abs(srcPath)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

	// Load destination
	destInfo, err := os.Stat(destPath)
	if err != nil {
		return fmt.Errorf("cannot access destination: %w", err)
	}

	var dest *glxlib.GLXFile
	destIsDir := destInfo.IsDir()

	if destIsDir {
		loaded, dupes, loadErr := LoadArchiveWithOptions(destPath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load destination archive: %w", loadErr)
		}
		for _, d := range dupes {
			fmt.Fprintf(os.Stderr, "Warning (dest): %s\n", d)
		}
		dest = loaded
	} else {
		loaded, loadErr := readSingleFileArchive(destPath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load destination archive: %w", loadErr)
		}
		dest = loaded
	}

	// Load source
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("cannot access source: %w", err)
	}

	var src *glxlib.GLXFile
	if srcInfo.IsDir() {
		loaded, dupes, loadErr := LoadArchiveWithOptions(srcPath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load source archive: %w", loadErr)
		}
		for _, d := range dupes {
			fmt.Fprintf(os.Stderr, "Warning (src): %s\n", d)
		}
		src = loaded
	} else {
		loaded, loadErr := readSingleFileArchive(srcPath, false)
		if loadErr != nil {
			return fmt.Errorf("failed to load source archive: %w", loadErr)
		}
		src = loaded
	}

	// Run cross-archive duplicate detection before merge (both archives still intact)
	var dupResult *glxlib.DuplicateResult
	if preview {
		var dupErr error
		dupResult, dupErr = glxlib.FindCrossArchiveDuplicates(dest, src, glxlib.DuplicateOptions{
			Threshold: threshold,
		})
		if dupErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: duplicate detection failed: %v\n", dupErr)
		}
	}

	// Merge
	result := mergeArchivesInMemory(dest, src)

	// Report
	fmt.Printf("Merging %s into %s\n\n", srcPath, destPath)
	printMergeReport(result)

	// Show cross-archive duplicates in preview mode
	if preview && dupResult != nil {
		printCrossArchiveDuplicates(dupResult, dest, src)
	}

	if preview {
		fmt.Println("\n(preview — no files written)")

		return nil
	}

	// Save — multi-file archives use crash-safe temp+swap to prevent partial writes.
	// Single-file archives use os.WriteFile directly (not atomic; see #595).
	if destIsDir {
		return safeWriteMultiFileArchive(destPath, dest)
	}

	return writeSingleFileArchive(destPath, dest, false)
}

func printMergeReport(result *mergeResult) {
	if result.TotalNew() > 0 {
		fmt.Println("  New entities:")
		printEntityCount("persons", result.NewPersons)
		printEntityCount("events", result.NewEvents)
		printEntityCount("relationships", result.NewRelationships)
		printEntityCount("places", result.NewPlaces)
		printEntityCount("sources", result.NewSources)
		printEntityCount("citations", result.NewCitations)
		printEntityCount("repositories", result.NewRepositories)
		printEntityCount("assertions", result.NewAssertions)
		printEntityCount("media", result.NewMedia)
	} else {
		fmt.Println("  No new entities to merge.")
	}

	if len(result.Conflicts) > 0 {
		fmt.Printf("\n  Conflicts (%d — skipped, destination kept):\n", len(result.Conflicts))
		for _, d := range result.Conflicts {
			fmt.Printf("    %s\n", d)
		}
	}

	if result.IdenticalSkipped > 0 {
		fmt.Printf("\n  %d identical vocabulary/property entries skipped\n", result.IdenticalSkipped)
	}
}

func printEntityCount(name string, count int) {
	if count > 0 {
		fmt.Printf("    %d %s\n", count, name)
	}
}

// printCrossArchiveDuplicates renders potential duplicates found between two archives.
func printCrossArchiveDuplicates(result *glxlib.DuplicateResult, dest, src *glxlib.GLXFile) {
	if len(result.Pairs) == 0 {
		fmt.Printf("\n  No potential cross-archive duplicates found (threshold: %.2f)\n", result.Threshold)

		return
	}

	fmt.Printf("\n  Potential cross-archive duplicates (%d, threshold: %.2f):\n", len(result.Pairs), result.Threshold)
	for _, pair := range result.Pairs {
		nameA := crossArchivePersonLabel(pair.PersonA, dest, src)
		nameB := crossArchivePersonLabel(pair.PersonB, dest, src)
		fmt.Printf("    %s ↔ %s  (score: %.2f)\n", nameA, nameB, pair.Score)
		for _, sig := range pair.Signals {
			if sig.Score > 0 {
				fmt.Printf("      %s: %.2f (%s)\n", sig.Name, sig.Score, sig.Detail)
			}
		}
	}
}

// crossArchivePersonLabel looks up a person in dest first, then src.
func crossArchivePersonLabel(personID string, dest, src *glxlib.GLXFile) string {
	if _, ok := dest.Persons[personID]; ok {
		return personLabel(dest, personID)
	}

	return personLabel(src, personID)
}
