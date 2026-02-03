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
func convertRepository(repoRecord *GEDCOMRecord, conv *ConversionContext) error {
	if repoRecord.Tag != GedcomTagRepo {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedRepoRecord, GedcomTagRepo, repoRecord.Tag)
	}

	// Generate repository ID
	repositoryID := generateRepositoryID(conv)
	conv.RepositoryIDMap[repoRecord.XRef] = repositoryID

	conv.Logger.LogInfo(fmt.Sprintf("Converting REPO %s -> %s", repoRecord.XRef, repositoryID))

	// Create repository entity
	repository := &Repository{
		Properties: make(map[string]any),
	}

	var phones []string
	var emails []string
	var notes []string

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	exids := extractExternalIDs(repoRecord)
	if len(exids) > 0 {
		repository.Properties[RepositoryPropertyExternalIDs] = exids
	}

	// Process subrecords
	for _, sub := range repoRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagName:
			// Repository name - may have CONT/CONC for long names
			repository.Name = extractTextWithContinuation(sub)

		case GedcomTagAddr:
			// Address - extract components
			for _, addrSub := range sub.SubRecords {
				switch addrSub.Tag {
				case GedcomTagCity:
					repository.City = addrSub.Value
				case GedcomTagStae:
					repository.State = addrSub.Value
				case GedcomTagPost:
					repository.PostalCode = addrSub.Value
				case GedcomTagCtry:
					repository.Country = addrSub.Value
				}
			}
			// Main address value - may have CONT/CONC for long addresses
			addrText := extractTextWithContinuation(sub)
			if addrText != "" {
				repository.Address = addrText
			}

		case GedcomTagPhon:
			// Phone - collect all into properties
			phones = append(phones, sub.Value)

		case GedcomTagEmail:
			// Email - collect all into properties
			emails = append(emails, sub.Value)

		case GedcomTagWww:
			// Website (GEDCOM 7.0)
			repository.Website = sub.Value

		case GedcomTagNote:
			// Notes
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				notes = append(notes, noteText)
			}

		case GedcomTagType:
			// Repository type (GEDCOM 7.0)
			repository.Type = mapRepositoryType(sub.Value)
		}
	}

	// Store phones in properties (multi-value)
	if len(phones) > 0 {
		repository.Properties[RepositoryPropertyPhones] = phones
	}

	// Store emails in properties (multi-value)
	if len(emails) > 0 {
		repository.Properties[RepositoryPropertyEmails] = emails
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
	conv.GLX.Repositories[repositoryID] = repository
	conv.Stats.RepositoriesCreated++

	return nil
}

// mapRepositoryType maps GEDCOM repository type to GLX.
// Uses gedcomRepositoryTypeMapping from constants.go.
func mapRepositoryType(gedcomType string) string {
	typeLower := strings.ToLower(gedcomType)
	if mapped, ok := gedcomRepositoryTypeMapping[typeLower]; ok {
		return mapped
	}
	return "other"
}

// inferRepositoryType infers repository type from name
func inferRepositoryType(name string) string {
	nameLower := strings.ToLower(name)

	if strings.Contains(nameLower, "archive") {
		return RepositoryTypeArchive
	}
	if strings.Contains(nameLower, "library") {
		return RepositoryTypeLibrary
	}
	if strings.Contains(nameLower, "church") {
		return RepositoryTypeChurch
	}
	if strings.Contains(nameLower, "museum") {
		return RepositoryTypeMuseum
	}
	if strings.Contains(nameLower, "university") || strings.Contains(nameLower, "college") {
		return RepositoryTypeUniversity
	}
	if strings.Contains(nameLower, "society") {
		return RepositoryTypeHistoricalSociety
	}
	if strings.Contains(nameLower, "ancestr") || strings.Contains(nameLower, "familysearch") {
		return RepositoryTypeDatabase
	}

	return RepositoryTypeOther
}
