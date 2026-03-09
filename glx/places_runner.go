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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// analyzePlaces loads an archive and reports place quality issues.
func analyzePlaces(path string) error {
	archive, err := loadArchiveForPlaces(path)
	if err != nil {
		return err
	}

	if len(archive.Places) == 0 {
		fmt.Println("No places found in archive.")

		return nil
	}

	analysis := buildPlaceAnalysis(archive)
	printPlaceAnalysis(analysis)

	return nil
}

// loadArchiveForPlaces loads an archive from a path (directory or single file).
func loadArchiveForPlaces(path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		for _, d := range duplicates {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", d)
		}

		return archive, nil
	}

	archive, err := readSingleFileArchive(path, false)
	if err != nil {
		return nil, err
	}

	return archive, nil
}

// placeAnalysis holds the results of place quality analysis.
type placeAnalysis struct {
	Total              int
	Canonical          map[string]string   // placeID -> canonical path
	Duplicates         map[string][]string // normalized name -> list of placeIDs with that name
	DuplicateOriginals map[string]string   // normalized name -> first original-cased name seen
	MissingCoords      []string            // placeIDs without coordinates
	MissingType        []string            // placeIDs without a type
	NoParent           []string            // non-country/region placeIDs without a parent
	DanglingParent     []string            // placeIDs whose parent doesn't exist
	DanglingParentIDs  map[string]string   // placeID -> missing parent ID
	Unreferenced       []string            // placeIDs not referenced by any event, assertion, or parent
}

// topLevelTypes are place types that don't require a parent.
var topLevelTypes = map[string]bool{
	"country": true,
	"region":  true,
}

// buildPlaceAnalysis analyzes all places in the archive.
func buildPlaceAnalysis(archive *glxlib.GLXFile) *placeAnalysis {
	a := &placeAnalysis{
		Canonical:          make(map[string]string),
		Duplicates:         make(map[string][]string),
		DuplicateOriginals: make(map[string]string),
		DanglingParentIDs:  make(map[string]string),
	}

	// Build canonical paths and detect issues
	for id, place := range archive.Places {
		if place == nil {
			continue
		}
		a.Total++
		a.Canonical[id] = buildCanonicalPath(id, archive.Places)

		// Track names for duplicate detection (skip empty names)
		rawName := strings.TrimSpace(place.Name)
		if rawName != "" {
			name := strings.ToLower(rawName)
			a.Duplicates[name] = append(a.Duplicates[name], id)
			if _, ok := a.DuplicateOriginals[name]; !ok {
				a.DuplicateOriginals[name] = rawName
			}
		}

		// Missing coordinates
		if place.Latitude == nil || place.Longitude == nil {
			a.MissingCoords = append(a.MissingCoords, id)
		}

		// Missing type
		if place.Type == "" {
			a.MissingType = append(a.MissingType, id)
		}

		// No parent (exclude top-level types)
		if place.ParentID == "" && !topLevelTypes[place.Type] {
			a.NoParent = append(a.NoParent, id)
		}

		// Dangling parent (references a parent that doesn't exist or is nil)
		if place.ParentID != "" {
			if parent, ok := archive.Places[place.ParentID]; !ok || parent == nil {
				a.DanglingParent = append(a.DanglingParent, id)
				a.DanglingParentIDs[id] = place.ParentID
			}
		}
	}

	// Find unreferenced places (skip nil entries)
	referenced := collectReferencedPlaces(archive)
	for id, place := range archive.Places {
		if place == nil {
			continue
		}
		if _, ok := referenced[id]; !ok {
			a.Unreferenced = append(a.Unreferenced, id)
		}
	}

	// Remove non-duplicate names
	for name, ids := range a.Duplicates {
		if len(ids) < 2 {
			delete(a.Duplicates, name)
			delete(a.DuplicateOriginals, name)
		}
	}

	// Sort all slices for deterministic output
	sort.Strings(a.MissingCoords)
	sort.Strings(a.MissingType)
	sort.Strings(a.NoParent)
	sort.Strings(a.DanglingParent)
	sort.Strings(a.Unreferenced)

	return a
}

// buildCanonicalPath builds a full hierarchy path for a place (e.g., "Leeds, Yorkshire, England").
func buildCanonicalPath(placeID string, places map[string]*glxlib.Place) string {
	var parts []string
	visited := make(map[string]bool)
	current := placeID

	for current != "" {
		if visited[current] {
			break // prevent cycles
		}
		visited[current] = true

		place, ok := places[current]
		if !ok || place == nil {
			break
		}
		name := strings.TrimSpace(place.Name)
		if name != "" {
			parts = append(parts, name)
		}
		current = place.ParentID
	}

	return strings.Join(parts, ", ")
}

// collectReferencedPlaces returns the set of place IDs referenced by events,
// assertions, or as parents.
func collectReferencedPlaces(archive *glxlib.GLXFile) map[string]struct{} {
	referenced := make(map[string]struct{})

	for _, event := range archive.Events {
		if event != nil && event.PlaceID != "" {
			referenced[event.PlaceID] = struct{}{}
		}
	}

	// Places referenced as assertion subjects
	for _, a := range archive.Assertions {
		if a != nil && a.Subject.Place != "" {
			referenced[a.Subject.Place] = struct{}{}
		}
	}

	// Places referenced as parents (only if the parent exists and is non-nil)
	for _, place := range archive.Places {
		if place != nil && place.ParentID != "" {
			if parent, ok := archive.Places[place.ParentID]; ok && parent != nil {
				referenced[place.ParentID] = struct{}{}
			}
		}
	}

	return referenced
}

// printPlaceAnalysis prints the analysis results.
func printPlaceAnalysis(a *placeAnalysis) {
	fmt.Printf("Place analysis: %d places\n", a.Total)

	issues := 0

	// Duplicate names
	if len(a.Duplicates) > 0 {
		issues += len(a.Duplicates)
		fmt.Println("\nDuplicate names (ambiguous):")

		names := make([]string, 0, len(a.Duplicates))
		for name := range a.Duplicates {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			ids := a.Duplicates[name]
			sort.Strings(ids)
			displayName := a.DuplicateOriginals[name]
			fmt.Printf("  \"%s\" appears %d times:\n", displayName, len(ids))
			for _, id := range ids {
				fmt.Printf("    %s  %s\n", id, a.Canonical[id])
			}
		}
	}

	// Missing coordinates
	if len(a.MissingCoords) > 0 {
		issues += len(a.MissingCoords)
		fmt.Printf("\nMissing coordinates (%d of %d):\n", len(a.MissingCoords), a.Total)
		for _, id := range a.MissingCoords {
			fmt.Printf("  %s  %s\n", id, a.Canonical[id])
		}
	}

	// Missing type
	if len(a.MissingType) > 0 {
		issues += len(a.MissingType)
		fmt.Printf("\nMissing type (%d of %d):\n", len(a.MissingType), a.Total)
		for _, id := range a.MissingType {
			fmt.Printf("  %s  %s\n", id, a.Canonical[id])
		}
	}

	// No parent
	if len(a.NoParent) > 0 {
		issues += len(a.NoParent)
		fmt.Printf("\nNo parent (hierarchy gap):\n")
		for _, id := range a.NoParent {
			fmt.Printf("  %s  %s\n", id, a.Canonical[id])
		}
	}

	// Dangling parent
	if len(a.DanglingParent) > 0 {
		issues += len(a.DanglingParent)
		fmt.Printf("\nDangling parent (references missing place):\n")
		for _, id := range a.DanglingParent {
			fmt.Printf("  %s  %s  (parent: %s)\n", id, a.Canonical[id], a.DanglingParentIDs[id])
		}
	}

	// Unreferenced
	if len(a.Unreferenced) > 0 {
		issues += len(a.Unreferenced)
		fmt.Printf("\nUnreferenced (not used by any event, assertion, or as parent):\n")
		for _, id := range a.Unreferenced {
			fmt.Printf("  %s  %s\n", id, a.Canonical[id])
		}
	}

	if issues == 0 {
		fmt.Println("\nNo issues found.")
	}
}
