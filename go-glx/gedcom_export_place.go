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

package glx

import (
	"fmt"
	"strings"
)

// resolvePlaceStrings pre-resolves all place IDs to GEDCOM place strings.
// Walks the parent chain for each place: "City, County, State, Country"
func resolvePlaceStrings(expCtx *ExportContext) {
	for placeID := range expCtx.GLX.Places {
		if _, ok := expCtx.PlaceStrings[placeID]; !ok {
			expCtx.PlaceStrings[placeID] = resolvePlaceString(placeID, expCtx)
			expCtx.Stats.PlacesResolved++
		}
	}
}

// resolvePlaceString builds a single GEDCOM place string by walking the parent chain.
// Returns "City, County, State, Country" format.
// Uses a visited set to prevent infinite loops from circular references.
func resolvePlaceString(placeID string, expCtx *ExportContext) string {
	// Check cache first
	if cached, ok := expCtx.PlaceStrings[placeID]; ok {
		return cached
	}

	place, ok := expCtx.GLX.Places[placeID]
	if !ok {
		return ""
	}

	// Walk parent chain, collecting names from specific to general
	var parts []string
	visited := make(map[string]struct{})
	currentID := placeID

	for currentID != "" {
		// Circular reference protection
		if _, seen := visited[currentID]; seen {
			expCtx.addExportWarning(EntityTypePlaces, placeID, "circular place reference detected")

			break
		}
		visited[currentID] = struct{}{}

		current, exists := expCtx.GLX.Places[currentID]
		if !exists {
			break
		}

		parts = append(parts, current.Name)
		currentID = current.ParentID
	}

	result := strings.Join(parts, ", ")

	// Cache intermediate results too (the starting place)
	_ = place // suppress unused warning; we've already confirmed it exists

	return result
}

// exportPlaceSubrecords creates PLAC and optional MAP/LATI/LONG subrecords
// for a given place ID.
func exportPlaceSubrecords(placeID string, expCtx *ExportContext) []*GEDCOMRecord {
	if placeID == "" {
		return nil
	}

	placeStr, ok := expCtx.PlaceStrings[placeID]
	if !ok || placeStr == "" {
		return nil
	}

	placRecord := &GEDCOMRecord{
		Tag:        GedcomTagPlac,
		Value:      placeStr,
		SubRecords: []*GEDCOMRecord{},
	}

	// Add MAP/LATI/LONG if coordinates are available
	place, exists := expCtx.GLX.Places[placeID]
	if exists && (place.Latitude != nil || place.Longitude != nil) {
		mapRecord := &GEDCOMRecord{
			Tag:        GedcomTagMap,
			SubRecords: []*GEDCOMRecord{},
		}

		if place.Latitude != nil {
			direction := "N"
			lat := *place.Latitude
			if lat < 0 {
				direction = "S"
				lat = -lat
			}
			mapRecord.SubRecords = append(mapRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagLati,
				Value: fmt.Sprintf("%s%f", direction, lat),
			})
		}

		if place.Longitude != nil {
			direction := "E"
			lon := *place.Longitude
			if lon < 0 {
				direction = "W"
				lon = -lon
			}
			mapRecord.SubRecords = append(mapRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagLong,
				Value: fmt.Sprintf("%s%f", direction, lon),
			})
		}

		placRecord.SubRecords = append(placRecord.SubRecords, mapRecord)
	}

	return []*GEDCOMRecord{placRecord}
}
