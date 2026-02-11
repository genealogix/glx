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

// convertRepository converts a GEDCOM REPO record to a GLX Repository
//
//nolint:gocyclo
func convertRepository(repoRecord *GEDCOMRecord, conv *ConversionContext) error {
	if repoRecord.Tag != GedcomTagRepo {
		return fmt.Errorf("%w: expected %s, got %s", ErrUnexpectedRepoRecord, GedcomTagRepo, repoRecord.Tag)
	}

	// Create repository entity (we parse first to get the name for deduplication)
	repository := &Repository{
		Properties: make(map[string]any),
	}

	var phones []string
	var emails []string
	var notes []string

	// Extract external IDs (GEDCOM 7.0 EXID tags)
	exids := extractExternalIDs(repoRecord)
	if len(exids) > 0 {
		if propertyKey, ok := conv.GEDCOMIndex.RepositoryProperties[GedcomTagExid]; ok {
			repository.Properties[propertyKey] = exids
		}
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
		if propertyKey, ok := conv.GEDCOMIndex.RepositoryProperties[GedcomTagPhon]; ok {
			repository.Properties[propertyKey] = phones
		}
	}

	// Store emails in properties (multi-value)
	if len(emails) > 0 {
		if propertyKey, ok := conv.GEDCOMIndex.RepositoryProperties[GedcomTagEmail]; ok {
			repository.Properties[propertyKey] = emails
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

	// Check for duplicate repository by name
	dedupeKey := buildRepositoryDedupeKey(repository)
	if existingID, ok := conv.RepositoryNameMap[dedupeKey]; ok {
		// Reuse existing repository - just map the XRef to the existing ID
		conv.RepositoryIDMap[repoRecord.XRef] = existingID
		conv.Logger.LogInfo(fmt.Sprintf("Reusing existing repository %s for REPO %s (name: %s)", existingID, repoRecord.XRef, repository.Name))
		conv.Stats.RepositoriesDeduplicated++

		return nil
	}

	// Generate new repository ID
	repositoryID := generateRepositoryID(conv)
	conv.RepositoryIDMap[repoRecord.XRef] = repositoryID
	conv.RepositoryNameMap[dedupeKey] = repositoryID

	conv.Logger.LogInfo(fmt.Sprintf("Converting REPO %s -> %s", repoRecord.XRef, repositoryID))

	// Store repository
	conv.GLX.Repositories[repositoryID] = repository
	conv.Stats.RepositoriesCreated++

	return nil
}

// buildRepositoryDedupeKey creates a deduplication key for a repository.
// Uses name and city/country when available for more precise matching.
func buildRepositoryDedupeKey(repo *Repository) string {
	key := strings.ToLower(strings.TrimSpace(repo.Name))

	// Add location qualifiers if available to distinguish same-named repositories
	if repo.City != "" {
		key += "|" + strings.ToLower(strings.TrimSpace(repo.City))
	}
	if repo.Country != "" {
		key += "|" + strings.ToLower(strings.TrimSpace(repo.Country))
	}

	return key
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
