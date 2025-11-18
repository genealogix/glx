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
	repository := &Repository{}

	var phones []string
	var emails []string
	var notes []string

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	// Note: Repository doesn't have Properties field, so store in notes for now
	exids := extractExternalIDs(repoRecord)
	if len(exids) > 0 {
		for _, exid := range exids {
			exidType := exid["type"]
			if exidType == "" {
				exidType = "external"
			}
			notes = append(notes, fmt.Sprintf("External ID (%s): %s", exidType, exid["id"]))
		}
	}

	// Process subrecords
	for _, sub := range repoRecord.SubRecords {
		switch sub.Tag {
		case "NAME":
			// Repository name
			repository.Name = sub.Value

		case "ADDR":
			// Address - extract components
			for _, addrSub := range sub.SubRecords {
				switch addrSub.Tag {
				case "CITY":
					repository.City = addrSub.Value
				case "STAE":
					repository.State = addrSub.Value
				case "POST":
					repository.PostalCode = addrSub.Value
				case "CTRY":
					repository.Country = addrSub.Value
				}
			}
			// Main address value
			if sub.Value != "" {
				repository.Address = sub.Value
			}

		case "PHON":
			// Phone - collect all, use first as primary
			phones = append(phones, sub.Value)

		case "EMAIL":
			// Email - collect all, use first as primary
			emails = append(emails, sub.Value)

		case "WWW":
			// Website (GEDCOM 7.0)
			repository.Website = sub.Value

		case "NOTE":
			// Notes
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				notes = append(notes, noteText)
			}

		case "TYPE":
			// Repository type (GEDCOM 7.0)
			repository.Type = mapRepositoryType(sub.Value)
		}
	}

	// Set first phone/email as primary
	if len(phones) > 0 {
		repository.Phone = phones[0]
		if len(phones) > 1 {
			notes = append(notes, "Additional phones: "+strings.Join(phones[1:], ", "))
		}
	}
	if len(emails) > 0 {
		repository.Email = emails[0]
		if len(emails) > 1 {
			notes = append(notes, "Additional emails: "+strings.Join(emails[1:], ", "))
		}
	}

	// Combine notes
	if len(notes) > 0 {
		repository.Notes = strings.Join(notes, "\n")
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
