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
	Conflicts         []string
	IdenticalSkipped  int
	NewPersons        int
	NewEvents         int
	NewRelationships  int
	NewPlaces         int
	NewSources        int
	NewCitations      int
	NewRepositories   int
	NewAssertions     int
	NewMedia          int
	MediaFilesPlanned int // binaries scheduled to copy from src into dest
	MediaFilesRenamed int // planned binaries that needed a dedup rename
	MediaFilesCopied  int // binaries actually copied after the save
	MediaFilesSkipped int // planned binaries dropped because their entity didn't merge
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

	// Run cross-archive duplicate detection before merge (both archives still intact).
	// Snapshot dest person IDs so labels can attribute each person to the archive
	// it originated in — post-merge, dest contains src persons too, which would
	// collapse the dest/src distinction the label needs.
	var dupResult *glxlib.DuplicateResult
	var origDestPersonIDs map[string]struct{}
	if preview {
		var dupErr error
		dupResult, dupErr = glxlib.FindCrossArchiveDuplicates(dest, src, glxlib.DuplicateOptions{
			Threshold: threshold,
		})
		if dupErr != nil {
			return fmt.Errorf("failed to detect cross-archive duplicates: %w", dupErr)
		}
		origDestPersonIDs = make(map[string]struct{}, len(dest.Persons))
		for id := range dest.Persons {
			origDestPersonIDs[id] = struct{}{}
		}
	}

	// Plan source media binary copies before the merge, so the URI rewrites
	// done for name-collision dedup flow into the merged Media entities. Only
	// multi-file destinations can hold binaries; warn and skip the plan
	// otherwise.
	var mediaPlan []plannedMediaCopy
	if destIsDir {
		plan, planErr := planSourceMediaCopies(srcPath, destPath, src)
		if planErr != nil {
			return fmt.Errorf("planning source media copies: %w", planErr)
		}
		mediaPlan = plan
	} else if srcInfo.IsDir() {
		warnOrphanSourceMedia(srcPath)
	}

	// Merge
	result := mergeArchivesInMemory(dest, src)
	result.MediaFilesPlanned = len(mediaPlan)
	for _, p := range mediaPlan {
		if filepath.Base(p.SrcPath) != p.TargetName {
			result.MediaFilesRenamed++
		}
	}

	// Report
	fmt.Printf("Merging %s into %s\n\n", srcPath, destPath)
	printMergeReport(result)

	// Show cross-archive duplicates in preview mode
	if preview && dupResult != nil {
		printCrossArchiveDuplicates(dupResult, dest, src, origDestPersonIDs)
	}

	if preview {
		if result.MediaFilesPlanned > 0 {
			fmt.Printf("\n  Would copy %d media file(s) into %s", result.MediaFilesPlanned, glxlib.MediaFilesDir)
			if result.MediaFilesRenamed > 0 {
				fmt.Printf(" (%d renamed to avoid name collisions)", result.MediaFilesRenamed)
			}
			fmt.Println("")
		}
		fmt.Println("\n(preview — no files written)")

		return nil
	}

	// Save — multi-file archives use crash-safe temp+swap to prevent partial writes.
	// Single-file archives use os.WriteFile directly (not atomic; see #595).
	if destIsDir {
		if err := safeWriteMultiFileArchive(destPath, dest); err != nil {
			return err
		}
		copied, skipped, copyErr := executeMediaCopies(mediaPlan, destPath, dest)
		result.MediaFilesCopied = copied
		result.MediaFilesSkipped = skipped
		printMediaCopyReport(result)
		if copyErr != nil {
			return fmt.Errorf("copying source media binaries: %w", copyErr)
		}

		return nil
	}

	return writeSingleFileArchive(destPath, dest, false)
}

// warnOrphanSourceMedia prints a warning when the source archive carries media
// binaries that a single-file destination cannot hold. Merged Media URIs will
// dangle in this case; the warning lets the user decide whether to convert the
// destination to multi-file or accept the dangling references.
func warnOrphanSourceMedia(srcPath string) {
	srcFiles := filepath.Join(srcPath, glxlib.MediaFilesDir)
	entries, err := os.ReadDir(srcFiles)
	if err != nil || len(entries) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr,
		"Warning: source has %d file(s) under %s; single-file destination cannot hold binaries, so they will not be copied. Merged Media URIs may dangle.\n",
		len(entries), glxlib.MediaFilesDir,
	)
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

// printMediaCopyReport reports the media-binary copy outcome of a non-preview
// merge. Silent when no source binaries were planned (single-file source or
// archive with no media/files/).
func printMediaCopyReport(result *mergeResult) {
	if result.MediaFilesPlanned == 0 {
		return
	}
	fmt.Printf("  Media files: %d copied", result.MediaFilesCopied)
	if result.MediaFilesRenamed > 0 {
		fmt.Printf(", %d renamed to avoid name collisions", result.MediaFilesRenamed)
	}
	if result.MediaFilesSkipped > 0 {
		fmt.Printf(", %d skipped (entity conflict)", result.MediaFilesSkipped)
	}
	fmt.Println("")
}

// printCrossArchiveDuplicates renders potential duplicates found between two archives.
// origDestPersonIDs is the set of person IDs that originated in dest (captured
// before merge mutated it) so each label can be resolved against the person's
// origin archive.
func printCrossArchiveDuplicates(result *glxlib.DuplicateResult, dest, src *glxlib.GLXFile, origDestPersonIDs map[string]struct{}) {
	if len(result.Pairs) == 0 {
		fmt.Printf("\n  No potential cross-archive duplicates found (threshold: %.2f)\n", result.Threshold)

		return
	}

	fmt.Printf("\n  Potential cross-archive duplicates (%d, threshold: %.2f):\n", len(result.Pairs), result.Threshold)
	for _, pair := range result.Pairs {
		nameA := crossArchivePersonLabel(pair.PersonA, dest, src, origDestPersonIDs)
		nameB := crossArchivePersonLabel(pair.PersonB, dest, src, origDestPersonIDs)
		fmt.Printf("    %s ↔ %s  (score: %.2f)\n", nameA, nameB, pair.Score)
		for _, sig := range pair.Signals {
			if sig.Score > 0 {
				fmt.Printf("      %s: %.2f (%s)\n", sig.Name, sig.Score, sig.Detail)
			}
		}
	}
}

// crossArchivePersonLabel resolves a person to their origin archive for
// labeling. origDestPersonIDs must be the pre-merge set of dest IDs; post-merge
// dest would contain src persons too, collapsing the distinction.
func crossArchivePersonLabel(personID string, dest, src *glxlib.GLXFile, origDestPersonIDs map[string]struct{}) string {
	if _, ok := origDestPersonIDs[personID]; ok {
		return personLabel(dest, personID)
	}

	return personLabel(src, personID)
}
