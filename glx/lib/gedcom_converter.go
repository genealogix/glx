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
)

// Convert performs the main GEDCOM to GLX conversion with two-pass processing
func (ctx *ConversionContext) Convert(records []*GEDCOMRecord) error {
	ctx.Logger.LogInfo("Starting conversion")

	// First pass: Process all top-level records in dependency order
	for _, record := range records {
		switch record.Tag {
		case "HEAD":
			// Header - extract metadata
			ctx.Logger.LogInfo("Processing HEAD")
			convertHeader(record, ctx)

		case "TRLR":
			// Trailer - end of file
			continue

		// GEDCOM 5.5.1: Process shared NOTE records
		case "NOTE":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing NOTE %s", record.XRef))
			if err := convertSharedNote551(record, ctx); err != nil {
				ctx.addError(record.Line, "NOTE", err.Error())
			}

		// GEDCOM 7.0: Process shared notes (SNOTE)
		case "SNOTE":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing SNOTE %s", record.XRef))
			if err := convertSharedNote(record, ctx); err != nil {
				ctx.addError(record.Line, "SNOTE", err.Error())
			}

		// GEDCOM 7.0: Process extension schemas
		case "SCHMA":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing SCHMA %s", record.XRef))
			if err := convertExtensionSchema(record, ctx); err != nil {
				ctx.addError(record.Line, "SCHMA", err.Error())
			}

		// Process repositories before sources (for linking)
		case "REPO":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing REPO %s", record.XRef))
			if err := convertRepository(record, ctx); err != nil {
				ctx.addError(record.Line, "REPO", err.Error())
			}

		// Process sources before individuals (for evidence)
		case "SOUR":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing SOUR %s", record.XRef))
			if err := convertSource(record, ctx); err != nil {
				ctx.addError(record.Line, "SOUR", err.Error())
			}

		// Process media objects
		case "OBJE":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing OBJE %s", record.XRef))
			if err := convertMedia(record, ctx); err != nil {
				ctx.addError(record.Line, "OBJE", err.Error())
			}

		// Process individuals
		case "INDI":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing INDI %s", record.XRef))
			if err := convertIndividual(record, ctx); err != nil {
				ctx.addError(record.Line, "INDI", err.Error())
			}

		// Defer families until after individuals
		case "FAM":
			ctx.Logger.LogInfo(fmt.Sprintf("Deferring FAM %s", record.XRef))
			ctx.DeferredFamilies = append(ctx.DeferredFamilies, record)

		// Handle submitter (SUBM)
		case "SUBM":
			ctx.Logger.LogInfo(fmt.Sprintf("Processing SUBM %s", record.XRef))
			convertSubmitter(record, ctx)

		default:
			// Unknown or extension tag
			if isExtensionTag(record.Tag) {
				ctx.addWarning(record.Line, record.Tag, "Extension tag not fully processed")
			} else {
				ctx.addWarning(record.Line, record.Tag, fmt.Sprintf("Unknown top-level tag: %s", record.Tag))
			}
		}
	}

	ctx.Logger.LogInfo(fmt.Sprintf("First pass complete: %d persons, %d sources, %d repositories, %d media",
		ctx.Stats.PersonsCreated, ctx.Stats.SourcesCreated, ctx.Stats.RepositoriesCreated, ctx.Stats.MediaCreated))

	// Second pass: Process families now that all individuals exist
	ctx.Logger.LogInfo(fmt.Sprintf("Processing %d deferred families", len(ctx.DeferredFamilies)))
	for _, famRecord := range ctx.DeferredFamilies {
		if err := convertFamily(famRecord, ctx); err != nil {
			ctx.addError(famRecord.Line, "FAM", err.Error())
		}
	}

	ctx.Logger.LogInfo(fmt.Sprintf("Second pass complete: %d relationships created", ctx.Stats.RelationshipsCreated))

	// Third pass: Create parent-child relationships with PEDI-based types
	ctx.Logger.LogInfo(fmt.Sprintf("Processing %d deferred family links (FAMC)", len(ctx.DeferredFamilyLinks)))
	for _, link := range ctx.DeferredFamilyLinks {
		if link.LinkType == ParticipantRoleChild {
			// Look up parents from family
			parents := ctx.FamilyParentsMap[link.FamilyRef]
			if len(parents) == 0 {
				ctx.Logger.LogWarning(0, "FAMC", link.FamilyRef, "Family not found or has no parents")
				continue
			}

			// Determine relationship type based on PEDI
			relType := mapPedigreeToRelationshipType(link.PedigreeType)

			// Create relationship for each parent
			for _, parentID := range parents {
				relationshipID := generateRelationshipID(ctx)

				relationship := &Relationship{
					Type:       relType,
					Persons:    []string{parentID, link.PersonID},
					Properties: make(map[string]any),
				}

				ctx.GLX.Relationships[relationshipID] = relationship
				ctx.Stats.RelationshipsCreated++
			}
		}
	}

	ctx.Logger.LogInfo(fmt.Sprintf("Third pass complete: total %d relationships", ctx.Stats.RelationshipsCreated))

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
func convertHeader(headRecord *GEDCOMRecord, ctx *ConversionContext) {
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
			metadata["notes"] = extractNoteText(sub, ctx)
		}
	}

	// TODO: Store metadata somewhere (maybe in properties or external file)
	// For now, just log it
	if len(metadata) > 0 {
		ctx.Logger.LogInfo(fmt.Sprintf("HEAD metadata: %+v", metadata))
	}
}

// convertSubmitter converts SUBM record to metadata
func convertSubmitter(submRecord *GEDCOMRecord, ctx *ConversionContext) {
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

	// TODO: Store submitter metadata somewhere
	// For now, just log it
	if len(submitter) > 0 {
		ctx.Logger.LogInfo(fmt.Sprintf("SUBM submitter: %+v", submitter))
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
	for i, part := range parts {
		if i > 0 {
			result += ", "
		}
		result += part
	}

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
