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
	"strconv"
	"strings"
)

// PlaceHierarchy represents a parsed place hierarchy
type PlaceHierarchy struct {
	Components []string // From specific to general
	Latitude   *float64 // Latitude coordinate (if provided)
	Longitude  *float64 // Longitude coordinate (if provided)
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

		// Only the most specific place gets coordinates
		var lat, lon *float64
		if i == 0 {
			lat = hierarchy.Latitude
			lon = hierarchy.Longitude
		}

		// Check if place already exists
		placeID := createOrGetPlace(name, parentID, level, totalLevels, lat, lon, ctx)

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
func createOrGetPlace(name string, parentID string, level int, totalLevels int, latitude, longitude *float64, ctx *ConversionContext) string {
	// Create a key for deduplication
	key := name
	if parentID != "" {
		key = parentID + ":" + name
	}

	// Check if place already exists
	if existingID, ok := ctx.PlaceIDMap[key]; ok {
		// If we have new coordinates and the existing place doesn't, update it
		if (latitude != nil || longitude != nil) && ctx.GLX.Places[existingID] != nil {
			existingPlace := ctx.GLX.Places[existingID]
			if existingPlace.Latitude == nil && latitude != nil {
				existingPlace.Latitude = latitude
			}
			if existingPlace.Longitude == nil && longitude != nil {
				existingPlace.Longitude = longitude
			}
		}
		return existingID
	}

	// Create new place
	placeID := generatePlaceID(ctx)

	place := &Place{
		Name:       name,
		Type:       inferPlaceType(name, level, totalLevels),
		Latitude:   latitude,
		Longitude:  longitude,
		Properties: make(map[string]interface{}),
	}

	if parentID != "" {
		place.ParentID = parentID
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

// extractPlaceCoordinates extracts latitude and longitude from PLAC subrecords
// GEDCOM format:
//   2 MAP
//   3 LATI N12.345678  (or S12.345678)
//   3 LONG W123.456789 (or E123.456789)
func extractPlaceCoordinates(placeRecord *GEDCOMRecord) (latitude, longitude *float64) {
	for _, sub := range placeRecord.SubRecords {
		if sub.Tag == "MAP" {
			// MAP contains LATI and LONG
			for _, mapSub := range sub.SubRecords {
				switch mapSub.Tag {
				case "LATI":
					if lat := parseCoordinate(mapSub.Value); lat != nil {
						latitude = lat
					}
				case "LONG":
					if lon := parseCoordinate(mapSub.Value); lon != nil {
						longitude = lon
					}
				}
			}
		}
	}
	return latitude, longitude
}

// parseCoordinate parses GEDCOM coordinate format
// Format: N12.345678 or S12.345678 (latitude)
//         E123.456789 or W123.456789 (longitude)
func parseCoordinate(value string) *float64 {
	if value == "" {
		return nil
	}

	// Get direction (first character)
	direction := value[0]
	numStr := value[1:]

	// Parse the number
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil
	}

	// Apply sign based on direction
	// N (north) and E (east) are positive
	// S (south) and W (west) are negative
	if direction == 'S' || direction == 'W' {
		num = -num
	}

	return &num
}
