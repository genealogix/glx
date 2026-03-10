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
	"fmt"
	"os"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// diffArchives loads two archives and prints their diff.
func diffArchives(path1, path2, person string, verbose, short, jsonOutput bool) error {
	old, err := loadArchiveForDiff(path1)
	if err != nil {
		return fmt.Errorf("loading %s: %w", path1, err)
	}

	new, err := loadArchiveForDiff(path2)
	if err != nil {
		return fmt.Errorf("loading %s: %w", path2, err)
	}

	result := glxlib.DiffArchives(old, new, person)

	switch {
	case jsonOutput:
		return printDiffJSON(result)
	case short:
		printDiffShort(result)
	case verbose:
		printDiffVerbose(result)
	default:
		printDiffSummary(result)
	}

	return nil
}

// loadArchiveForDiff loads an archive from a path (directory or single file).
func loadArchiveForDiff(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, loadErr := LoadArchiveWithOptions(path, false)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load archive: %w", loadErr)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}
		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// printDiffSummary prints the default summary view.
func printDiffSummary(result *glxlib.DiffResult) {
	if len(result.Changes) == 0 {
		fmt.Println("No changes.")
		return
	}

	// Group by entity type
	groups := groupByEntityType(result.Changes)
	entityTypes := sortedEntityTypes(groups)

	for _, entityType := range entityTypes {
		changes := groups[entityType]
		fmt.Printf("\n%s\n", strings.ToUpper(entityType))

		for _, c := range changes {
			prefix := changePrefix(c.Kind)
			fmt.Printf("  %s %-38s  %s\n", prefix, c.ID, c.Summary)

			// For modified entities, show field changes inline (up to 3).
			// Skip when only 1 field changed — the summary already shows it.
			if c.Kind == glxlib.ChangeModified && len(c.Fields) > 1 {
				limit := 3
				if len(c.Fields) < limit {
					limit = len(c.Fields)
				}
				for _, f := range c.Fields[:limit] {
					fmt.Printf("    %-36s  %s → %s\n", f.Path+":", f.OldValue, f.NewValue)
				}
				if len(c.Fields) > limit {
					fmt.Printf("    ... and %d more field(s)\n", len(c.Fields)-limit)
				}
			}
		}
	}

	fmt.Println()
	printDiffStatsLine(result)
}

// printDiffVerbose prints all field-level details for modified entities.
func printDiffVerbose(result *glxlib.DiffResult) {
	if len(result.Changes) == 0 {
		fmt.Println("No changes.")
		return
	}

	groups := groupByEntityType(result.Changes)
	entityTypes := sortedEntityTypes(groups)

	for _, entityType := range entityTypes {
		changes := groups[entityType]
		fmt.Printf("\n%s\n", strings.ToUpper(entityType))

		for _, c := range changes {
			prefix := changePrefix(c.Kind)
			fmt.Printf("  %s %s\n", prefix, c.ID)

			if c.Kind == glxlib.ChangeModified {
				for _, f := range c.Fields {
					fmt.Printf("      %-30s  %s → %s\n", f.Path+":", f.OldValue, f.NewValue)
				}
			} else {
				fmt.Printf("      %s\n", c.Summary)
			}
		}
	}

	fmt.Println()
	printDiffStatsLine(result)
}

// printDiffShort prints a single-line compact summary.
func printDiffShort(result *glxlib.DiffResult) {
	s := result.Stats
	parts := []string{
		fmt.Sprintf("+%d ~%d -%d", s.Added, s.Modified, s.Removed),
	}

	if s.ConfidenceUpgrades > 0 {
		parts = append(parts, fmt.Sprintf("%d confidence upgrade(s)", s.ConfidenceUpgrades))
	}
	if s.ConfidenceDowngrades > 0 {
		parts = append(parts, fmt.Sprintf("%d confidence downgrade(s)", s.ConfidenceDowngrades))
	}
	if s.NewSources > 0 {
		parts = append(parts, fmt.Sprintf("%d new source(s)", s.NewSources))
	}
	if s.NewCitations > 0 {
		parts = append(parts, fmt.Sprintf("%d new citation(s)", s.NewCitations))
	}

	fmt.Println(strings.Join(parts, " | "))
}

// printDiffJSON outputs the result as JSON.
func printDiffJSON(result *glxlib.DiffResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// printDiffStatsLine prints the summary stats footer.
func printDiffStatsLine(result *glxlib.DiffResult) {
	s := result.Stats
	fmt.Printf("Summary: +%d added, ~%d modified, -%d removed\n", s.Added, s.Modified, s.Removed)

	extras := make([]string, 0, 4)
	if s.ConfidenceUpgrades > 0 {
		extras = append(extras, fmt.Sprintf("%d confidence upgrade(s)", s.ConfidenceUpgrades))
	}
	if s.ConfidenceDowngrades > 0 {
		extras = append(extras, fmt.Sprintf("%d confidence downgrade(s)", s.ConfidenceDowngrades))
	}
	if s.NewSources > 0 {
		extras = append(extras, fmt.Sprintf("%d new source(s)", s.NewSources))
	}
	if s.NewCitations > 0 {
		extras = append(extras, fmt.Sprintf("%d new citation(s)", s.NewCitations))
	}

	if len(extras) > 0 {
		fmt.Printf("         %s\n", strings.Join(extras, ", "))
	}
}

// changePrefix returns the display prefix for a change kind.
func changePrefix(kind glxlib.ChangeKind) string {
	switch kind {
	case glxlib.ChangeAdded:
		return "+"
	case glxlib.ChangeModified:
		return "~"
	case glxlib.ChangeRemoved:
		return "-"
	default:
		return " "
	}
}

// groupByEntityType groups changes by entity type, preserving order within each group.
func groupByEntityType(changes []glxlib.EntityChange) map[string][]glxlib.EntityChange {
	groups := make(map[string][]glxlib.EntityChange)
	for _, c := range changes {
		groups[c.EntityType] = append(groups[c.EntityType], c)
	}
	return groups
}

// sortedEntityTypes returns entity types sorted in display order.
func sortedEntityTypes(groups map[string][]glxlib.EntityChange) []string {
	order := []string{
		glxlib.EntityTypePersons,
		glxlib.EntityTypeEvents,
		glxlib.EntityTypeRelationships,
		glxlib.EntityTypePlaces,
		glxlib.EntityTypeAssertions,
		glxlib.EntityTypeSources,
		glxlib.EntityTypeCitations,
		glxlib.EntityTypeRepositories,
		glxlib.EntityTypeMedia,
	}
	var result []string
	for _, t := range order {
		if _, ok := groups[t]; ok {
			result = append(result, t)
		}
	}
	return result
}
