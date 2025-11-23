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

// Convert performs the main GEDCOM to GLX conversion with two-pass processing
func (conv *ConversionContext) Convert(records []*GEDCOMRecord) error {
	conv.Logger.LogInfo("Starting conversion")

	// First pass: Process all top-level records in dependency order
	for _, record := range records {
		switch record.Tag {
		case "HEAD":
			// Header - extract metadata
			conv.Logger.LogInfo("Processing HEAD")
			convertHeader(record, conv)

		case "TRLR":
			// Trailer - end of file
			continue

		// GEDCOM 5.5.1: Process shared NOTE records
		case "NOTE":
			conv.Logger.LogInfo("Processing NOTE " + record.XRef)
			if err := convertSharedNote551(record, conv); err != nil {
				conv.addError(record.Line, "NOTE", err.Error())
			}

		// GEDCOM 7.0: Process shared notes (SNOTE)
		case "SNOTE":
			conv.Logger.LogInfo("Processing SNOTE " + record.XRef)
			if err := convertSharedNote(record, conv); err != nil {
				conv.addError(record.Line, "SNOTE", err.Error())
			}

		// GEDCOM 7.0: Process extension schemas
		case "SCHMA":
			conv.Logger.LogInfo("Processing SCHMA " + record.XRef)
			if err := convertExtensionSchema(record, conv); err != nil {
				conv.addError(record.Line, "SCHMA", err.Error())
			}

		// Process repositories before sources (for linking)
		case "REPO":
			conv.Logger.LogInfo("Processing REPO " + record.XRef)
			if err := convertRepository(record, conv); err != nil {
				conv.addError(record.Line, "REPO", err.Error())
			}

		// Process sources before individuals (for evidence)
		case "SOUR":
			conv.Logger.LogInfo("Processing SOUR " + record.XRef)
			if err := convertSource(record, conv); err != nil {
				conv.addError(record.Line, "SOUR", err.Error())
			}

		// Process media objects
		case "OBJE":
			conv.Logger.LogInfo("Processing OBJE " + record.XRef)
			if err := convertMedia(record, conv); err != nil {
				conv.addError(record.Line, "OBJE", err.Error())
			}

		// Process individuals
		case "INDI":
			conv.Logger.LogInfo("Processing INDI " + record.XRef)
			if err := convertIndividual(record, conv); err != nil {
				conv.addError(record.Line, "INDI", err.Error())
			}

		// Defer families until after individuals
		case "FAM":
			conv.Logger.LogInfo("Deferring FAM " + record.XRef)
			conv.DeferredFamilies = append(conv.DeferredFamilies, record)

		// Handle submitter (SUBM)
		case "SUBM":
			conv.Logger.LogInfo("Processing SUBM " + record.XRef)
			convertSubmitter(record, conv)

		default:
			// Unknown or extension tag
			if isExtensionTag(record.Tag) {
				// Process extension data
				extData := convertExtensionData(record.Tag, record.Value, record.SubRecords)
				if len(extData) > 0 {
					// Extension data is processed but not yet stored (see todo.md)
					conv.Logger.LogInfo(fmt.Sprintf("Processed extension tag %s: %+v", record.Tag, extData))
				}
			} else {
				conv.addWarning(record.Line, record.Tag, "Unknown top-level tag: "+record.Tag)
			}
		}
	}

	conv.Logger.LogInfo(fmt.Sprintf("First pass complete: %d persons, %d sources, %d repositories, %d media",
		conv.Stats.PersonsCreated, conv.Stats.SourcesCreated, conv.Stats.RepositoriesCreated, conv.Stats.MediaCreated))

	// Second pass: Process families now that all individuals exist
	conv.Logger.LogInfo(fmt.Sprintf("Processing %d deferred families", len(conv.DeferredFamilies)))
	for _, famRecord := range conv.DeferredFamilies {
		if err := convertFamily(famRecord, conv); err != nil {
			conv.addError(famRecord.Line, "FAM", err.Error())
		}
	}

	conv.Logger.LogInfo(fmt.Sprintf("Second pass complete: %d relationships created", conv.Stats.RelationshipsCreated))

	// Third pass: Create parent-child relationships with PEDI-based types
	conv.Logger.LogInfo(fmt.Sprintf("Processing %d deferred family links (FAMC)", len(conv.DeferredFamilyLinks)))
	for _, link := range conv.DeferredFamilyLinks {
		if link.LinkType == ParticipantRoleChild {
			// Look up parents from family
			parents := conv.FamilyParentsMap[link.FamilyRef]
			if len(parents) == 0 {
				conv.Logger.LogWarning(0, "FAMC", link.FamilyRef, "Family not found or has no parents")

				continue
			}

			// Determine relationship type based on PEDI
			relType := mapPedigreeToRelationshipType(link.PedigreeType)

			// Create relationship for each parent
			for _, parentID := range parents {
				relationshipID := generateRelationshipID(conv)

				relationship := &Relationship{
					Type:    relType,
					Persons: []string{parentID, link.PersonID},
					Participants: []RelationshipParticipant{
						{Person: parentID, Role: ParticipantRoleParent},
						{Person: link.PersonID, Role: ParticipantRoleChild},
					},
					Properties: make(map[string]any),
				}

				conv.GLX.Relationships[relationshipID] = relationship
				conv.Stats.RelationshipsCreated++
			}
		}
	}

	conv.Logger.LogInfo(fmt.Sprintf("Third pass complete: total %d relationships", conv.Stats.RelationshipsCreated))

	return nil
}

// mapPedigreeToRelationshipType maps GEDCOM PEDI values to GLX relationship types
func mapPedigreeToRelationshipType(pediValue string) string {
	switch pediValue {
	case "birth":
		return RelationshipTypeBiologicalParentChild
	case "adopted":
		return RelationshipTypeAdoptiveParentChild
	case "foster":
		return RelationshipTypeFosterParentChild
	case "unknown", "":
		// Unknown or missing PEDI -> use generic parent-child
		return RelationshipTypeParentChild
	default:
		// For any other value (including "sealed" which we're not implementing)
		// use generic parent-child
		return RelationshipTypeParentChild
	}
}

// convertHeader extracts metadata from HEAD record
func convertHeader(headRecord *GEDCOMRecord, conv *ConversionContext) {
	metadata := make(map[string]any)

	for _, sub := range headRecord.SubRecords {
		switch sub.Tag {
		case "DATE":
			metadata["export_date"] = sub.Value
		case "FILE":
			metadata["source_file"] = sub.Value
		case "COPR":
			metadata["copyright"] = sub.Value
		case "LANG":
			metadata["language"] = sub.Value
		case "SOUR":
			// Source system
			for _, sourSub := range sub.SubRecords {
				switch sourSub.Tag {
				case "NAME":
					metadata["source_system"] = sourSub.Value
				case "VERS":
					metadata["source_version"] = sourSub.Value
				case "CORP":
					metadata["source_corporation"] = sourSub.Value
				}
			}
		case "SUBM":
			metadata["submitter_ref"] = sub.Value
		case "GEDC":
			for _, gedcSub := range sub.SubRecords {
				if gedcSub.Tag == "VERS" {
					metadata["gedcom_version"] = gedcSub.Value
				}
			}
		case "CHAR":
			metadata["character_set"] = sub.Value
		case "NOTE":
			metadata["notes"] = extractNoteText(sub, conv)
		}
	}

	// HEAD metadata is processed but not yet stored (see todo.md)
	if len(metadata) > 0 {
		conv.Logger.LogInfo(fmt.Sprintf("HEAD metadata: %+v", metadata))
	}
}

// convertSubmitter converts SUBM record to metadata
func convertSubmitter(submRecord *GEDCOMRecord, conv *ConversionContext) {
	submitter := make(map[string]any)

	for _, sub := range submRecord.SubRecords {
		switch sub.Tag {
		case "NAME":
			submitter["name"] = sub.Value
		case "ADDR":
			submitter["address"] = extractAddress(sub)
		case "PHON":
			submitter["phone"] = sub.Value
		case "EMAIL":
			submitter["email"] = sub.Value
		case "WWW":
			submitter["website"] = sub.Value
		}
	}

	// SUBM metadata is processed but not yet stored (see todo.md)
	if len(submitter) > 0 {
		conv.Logger.LogInfo(fmt.Sprintf("SUBM submitter: %+v", submitter))
	}
}

// extractAddress builds full address from ADDR record and subrecords
func extractAddress(addrRecord *GEDCOMRecord) string {
	var parts []string

	if addrRecord.Value != "" {
		parts = append(parts, addrRecord.Value)
	}

	for _, sub := range addrRecord.SubRecords {
		switch sub.Tag {
		case "ADR1", "ADR2", "ADR3":
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case "CITY":
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case "STAE":
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case "POST":
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case "CTRY":
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		}
	}

	result := ""
	var resultSb288 strings.Builder
	for i, part := range parts {
		if i > 0 {
			resultSb288.WriteString(", ")
		}
		resultSb288.WriteString(part)
	}
	result += resultSb288.String()

	return result
}

// Converter functions implemented in separate files:
// convertSharedNote and convertExtensionSchema are implemented in gedcom_7_0.go

func isExtensionTag(tag string) bool {
	// Extension tags start with underscore
	if len(tag) > 0 && tag[0] == '_' {
		return true
	}

	return false
}

// convertRepository is implemented in gedcom_repository.go
// convertSource is implemented in gedcom_source.go
// convertMedia is implemented in gedcom_media.go
// convertIndividual is implemented in gedcom_individual.go
// convertFamily is implemented in gedcom_family.go
