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
	"fmt"
	"strings"
)

// convertRepository converts a GEDCOM REPO record to a GLX Repository
func convertRepository(repoRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if repoRecord.Tag != "REPO" {
		return fmt.Errorf("expected REPO record, got %s", repoRecord.Tag)
	}

	// Generate repository ID
	repositoryID := generateRepositoryID(ctx)
	ctx.RepositoryIDMap[repoRecord.XRef] = repositoryID

	ctx.Logger.LogInfo(fmt.Sprintf("Converting REPO %s -> %s", repoRecord.XRef, repositoryID))

	// Create repository entity
	repository := &Repository{
		Properties: make(map[string]interface{}),
	}

	// Process subrecords
	for _, sub := range repoRecord.SubRecords {
		switch sub.Tag {
		case "NAME":
			// Repository name
			repository.Name = sub.Value

		case "ADDR":
			// Address - build from components
			address := extractAddress(sub)
			if address != "" {
				repository.Properties["address"] = address
			}

		case "PHON":
			// Phone
			phones, ok := repository.Properties["phone"].([]string)
			if !ok {
				phones = []string{}
			}
			repository.Properties["phone"] = append(phones, sub.Value)

		case "EMAIL":
			// Email
			emails, ok := repository.Properties["email"].([]string)
			if !ok {
				emails = []string{}
			}
			repository.Properties["email"] = append(emails, sub.Value)

		case "WWW":
			// Website (GEDCOM 7.0)
			repository.Properties["website"] = sub.Value

		case "NOTE":
			// Notes
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				repository.Properties["notes"] = noteText
			}

		case "TYPE":
			// Repository type (GEDCOM 7.0)
			repository.Type = mapRepositoryType(sub.Value)
		}
	}

	// Default type if not set
	if repository.Type == "" {
		repository.Type = inferRepositoryType(repository.Name)
	}

	// Store repository
	ctx.GLX.Repositories[repositoryID] = repository
	ctx.Stats.RepositoriesCreated++

	return nil
}

// mapRepositoryType maps GEDCOM repository type to GLX
func mapRepositoryType(gedcomType string) string {
	mapping := map[string]string{
		"archive":    "archive",
		"library":    "library",
		"church":     "church",
		"government": "government_agency",
		"museum":     "museum",
		"online":     "database",
		"registry":   "registry",
		"society":    "historical_society",
		"university": "university",
	}

	typeLower := strings.ToLower(gedcomType)
	if mapped, ok := mapping[typeLower]; ok {
		return mapped
	}

	return "other"
}

// inferRepositoryType infers repository type from name
func inferRepositoryType(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "archive") {
		return "archive"
	}
	if strings.Contains(nameLower, "library") {
		return "library"
	}
	if strings.Contains(nameLower, "church") {
		return "church"
	}
	if strings.Contains(nameLower, "museum") {
		return "museum"
	}
	if strings.Contains(nameLower, "university") || strings.Contains(nameLower, "college") {
		return "university"
	}
	if strings.Contains(nameLower, "society") {
		return "historical_society"
	}
	if strings.Contains(nameLower, "ancestr") || strings.Contains(nameLower, "familysearch") {
		return "database"
	}

	return "other"
}
