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

// mergeResult holds statistics from a merge operation.
type mergeResult struct {
	Duplicates       []string
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

	duplicates := dest.Merge(src)

	// Compute new entities added
	after := entityCounts(dest)

	return &mergeResult{
		Duplicates:       duplicates,
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
	persons, events, relationships, places    int
	sources, citations, repositories          int
	assertions, media                         int
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
func mergeArchives(srcPath, destPath string, dryRun bool) error {
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

	// Merge
	result := mergeArchivesInMemory(dest, src)

	// Report
	fmt.Printf("Merging %s into %s\n\n", srcPath, destPath)

	if result.TotalNew() > 0 {
		fmt.Println("  New entities:")
		if result.NewPersons > 0 {
			fmt.Printf("    %d persons\n", result.NewPersons)
		}
		if result.NewEvents > 0 {
			fmt.Printf("    %d events\n", result.NewEvents)
		}
		if result.NewRelationships > 0 {
			fmt.Printf("    %d relationships\n", result.NewRelationships)
		}
		if result.NewPlaces > 0 {
			fmt.Printf("    %d places\n", result.NewPlaces)
		}
		if result.NewSources > 0 {
			fmt.Printf("    %d sources\n", result.NewSources)
		}
		if result.NewCitations > 0 {
			fmt.Printf("    %d citations\n", result.NewCitations)
		}
		if result.NewRepositories > 0 {
			fmt.Printf("    %d repositories\n", result.NewRepositories)
		}
		if result.NewAssertions > 0 {
			fmt.Printf("    %d assertions\n", result.NewAssertions)
		}
		if result.NewMedia > 0 {
			fmt.Printf("    %d media\n", result.NewMedia)
		}
	} else {
		fmt.Println("  No new entities to merge.")
	}

	if len(result.Duplicates) > 0 {
		fmt.Printf("\n  Duplicates (%d — skipped, destination kept):\n", len(result.Duplicates))
		for _, d := range result.Duplicates {
			fmt.Printf("    %s\n", d)
		}
	}

	if dryRun {
		fmt.Println("\n(dry run — no files written)")
		return nil
	}

	// Save — multi-file archives use crash-safe temp+swap to prevent partial writes.
	// Single-file archives use os.WriteFile directly (not atomic; see #595).
	if destIsDir {
		return safeWriteMultiFileArchive(destPath, dest)
	}
	return writeSingleFileArchive(destPath, dest, false)
}

