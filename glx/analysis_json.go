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
	"context"
	"encoding/json"
	"fmt"
	"os"

	glxlib "github.com/genealogix/glx/go-glx"
)

type statsJSONResult struct {
	EntityCounts map[string]int    `json:"entity_counts"`
	Confidence   map[string]int    `json:"assertion_confidence"`
	Coverage     statsJSONCoverage `json:"coverage"`
}

type statsJSONCoverage struct {
	Direct   map[string]statsJSONCoverageRow `json:"direct"`
	Evidence map[string]statsJSONCoverageRow `json:"evidence"`
	Note     string                          `json:"note"`
}

type statsJSONCoverageRow struct {
	Covered int `json:"covered"`
	Total   int `json:"total"`
}

func archiveEntityCounts(archive *glxlib.GLXFile) map[string]int {
	return map[string]int{
		"persons": len(archive.Persons), "events": len(archive.Events),
		"relationships": len(archive.Relationships), searchFieldPlaces: len(archive.Places),
		"sources": len(archive.Sources), "citations": len(archive.Citations),
		"repositories": len(archive.Repositories), "media": len(archive.Media),
		"assertions": len(archive.Assertions), "research_logs": len(archive.ResearchLogs),
		"studies": len(archive.Studies),
	}
}

func buildStatsJSON(archive *glxlib.GLXFile) (*statsJSONResult, error) {
	return buildStatsJSONContext(context.Background(), archive)
}

func buildStatsJSONContext(ctx context.Context, archive *glxlib.GLXFile) (*statsJSONResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	confidence := make(map[string]int)
	for _, assertion := range archive.Assertions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if assertion == nil {
			continue
		}
		confidence[assertion.Confidence]++
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	coverage := glxlib.ComputeAssertionCoverage(archive)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &statsJSONResult{
		EntityCounts: archiveEntityCounts(archive), Confidence: confidence,
		Coverage: statsJSONCoverage{
			Direct:   statsCoverageRows(coverage.Direct, archive),
			Evidence: statsCoverageRows(coverage.Evidence, archive),
			Note:     "Evidence coverage includes unsourced or disproven assertions and place ancestors; counts do not measure sourcing or proof. An empty confidence key denotes an unset confidence level.",
		},
	}, nil
}

func statsCoverageRows(coverage glxlib.EntityCoverage, archive *glxlib.GLXFile) map[string]statsJSONCoverageRow {
	return map[string]statsJSONCoverageRow{
		"persons":         {countCovered(coverage.Persons, archive.Persons), len(archive.Persons)},
		"events":          {countCovered(coverage.Events, archive.Events), len(archive.Events)},
		"relationships":   {countCovered(coverage.Relationships, archive.Relationships), len(archive.Relationships)},
		searchFieldPlaces: {countCovered(coverage.Places, archive.Places), len(archive.Places)},
	}
}

func buildPlacesJSON(archive *glxlib.GLXFile) (*placeAnalysis, error) {
	if archive == nil {
		return nil, glxlib.ErrNilArchive
	}
	result := buildPlaceAnalysis(archive)
	if result.MissingCoords == nil {
		result.MissingCoords = []string{}
	}
	if result.MissingType == nil {
		result.MissingType = []string{}
	}
	if result.NoParent == nil {
		result.NoParent = []string{}
	}
	if result.DanglingParent == nil {
		result.DanglingParent = []string{}
	}
	if result.Unreferenced == nil {
		result.Unreferenced = []string{}
	}

	return result, nil
}

func showStatsJSON(path string) error {
	archive, err := loadArchiveForStats(path)
	if err != nil {
		return err
	}
	result, err := buildStatsJSON(archive)
	if err != nil {
		return err
	}

	return printArchiveAnalysisJSON(result)
}

func showPlacesJSON(path string) error {
	archive, err := loadArchiveForPlaces(path)
	if err != nil {
		return err
	}
	result, err := buildPlacesJSON(archive)
	if err != nil {
		return err
	}

	return printArchiveAnalysisJSON(result)
}

func printArchiveAnalysisJSON(result any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encode analysis JSON: %w", err)
	}

	return nil
}
