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

package lib

import (
	"strings"
)

// PlaceHierarchy represents a parsed place hierarchy
type PlaceHierarchy struct {
	Components []string // From specific to general
}

// parseGEDCOMPlace parses a GEDCOM place string
// Format: "City, County, State, Country" (comma-separated, specific to general)
func parseGEDCOMPlace(placeValue string) *PlaceHierarchy {
	if placeValue == "" {
		return nil
	}

	// Split by comma
	parts := strings.Split(placeValue, ",")
	var components []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			components = append(components, trimmed)
		}
	}

	if len(components) == 0 {
		return nil
	}

	return &PlaceHierarchy{
		Components: components,
	}
}

// buildPlaceHierarchy creates GLX Place entities for a place hierarchy
// Returns the ID of the most specific place (leaf)
func buildPlaceHierarchy(hierarchy *PlaceHierarchy, ctx *ConversionContext) (string, error) {
	if hierarchy == nil || len(hierarchy.Components) == 0 {
		return "", nil
	}

	var parentID string
	var leafID string

	// Build from general to specific (reverse order)
	totalLevels := len(hierarchy.Components)
	for i := totalLevels - 1; i >= 0; i-- {
		name := hierarchy.Components[i]
		level := totalLevels - i - 1

		// Check if place already exists
		placeID := createOrGetPlace(name, parentID, level, totalLevels, ctx)

		if i == 0 {
			// This is the most specific place (leaf)
			leafID = placeID
		}

		// Set as parent for next iteration
		parentID = placeID
	}

	return leafID, nil
}

// createOrGetPlace creates a place or returns existing place ID
func createOrGetPlace(name string, parentID string, level int, totalLevels int, ctx *ConversionContext) string {
	// Create a key for deduplication
	key := name
	if parentID != "" {
		key = parentID + ":" + name
	}

	// Check if place already exists
	if existingID, ok := ctx.PlaceIDMap[key]; ok {
		return existingID
	}

	// Create new place
	placeID := generatePlaceID(ctx)

	place := &Place{
		Name:       name,
		Type:       inferPlaceType(name, level, totalLevels),
		Properties: make(map[string]interface{}),
	}

	if parentID != "" {
		place.Parent = parentID
	}

	ctx.GLX.Places[placeID] = place
	ctx.PlaceIDMap[key] = placeID
	ctx.Stats.PlacesCreated++

	return placeID
}

// inferPlaceType infers the place type from name and position in hierarchy
func inferPlaceType(name string, level int, totalLevels int) string {
	nameLower := strings.ToLower(name)

	// Check for keywords
	if strings.Contains(nameLower, "cemetery") || strings.Contains(nameLower, "graveyard") {
		return "cemetery"
	}
	if strings.Contains(nameLower, "church") || strings.Contains(nameLower, "cathedral") {
		return "church"
	}
	if strings.Contains(nameLower, "hospital") {
		return "hospital"
	}
	if strings.Contains(nameLower, "county") {
		return "county"
	}
	if strings.Contains(nameLower, "province") || strings.Contains(nameLower, "state") {
		return "state_province"
	}

	// Infer from position in hierarchy
	// Typical order: City, County, State, Country
	switch level {
	case 0:
		// Most specific - likely city or town
		return "city"
	case 1:
		// Second level - likely county
		return "county"
	case 2:
		// Third level - likely state/province
		return "state_province"
	case 3:
		// Fourth level - likely country
		return "country"
	default:
		return "locality"
	}
}
