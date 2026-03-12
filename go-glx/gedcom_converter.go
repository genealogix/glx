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
	"strings"
)

// Convert performs the main GEDCOM to GLX conversion with dependency-ordered processing
//
//nolint:gocognit,gocyclo
func (conv *ConversionContext) Convert(records []*GEDCOMRecord) error {
	conv.Logger.LogInfo("Starting conversion")

	// Group records by type for dependency-ordered processing
	// This ensures records are processed in the correct order regardless of file order:
	//   1. Notes, Repositories, Schemas (no dependencies)
	//   2. Sources, Media (depend on repositories)
	//   3. Individuals (depend on sources, media, notes)
	//   4. Families (depend on individuals)
	grouped := groupRecordsByType(records)

	// Pass 1: Process records with no dependencies
	// Notes (may be referenced by any record)
	for _, record := range grouped[GedcomTagNote] {
		conv.Logger.LogInfo("Processing NOTE " + record.XRef)
		if err := convertSharedNote551(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagNote, err.Error())
		}
	}
	for _, record := range grouped[GedcomTagSnote] {
		conv.Logger.LogInfo("Processing SNOTE " + record.XRef)
		if err := convertSharedNote(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagSnote, err.Error())
		}
	}

	// Repositories (referenced by sources)
	for _, record := range grouped[GedcomTagRepo] {
		conv.Logger.LogInfo("Processing REPO " + record.XRef)
		if err := convertRepository(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagRepo, err.Error())
		}
	}

	// Schemas (GEDCOM 7.0 extension definitions)
	for _, record := range grouped[GedcomTagSchma] {
		conv.Logger.LogInfo("Processing SCHMA " + record.XRef)
		if err := convertExtensionSchema(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagSchma, err.Error())
		}
	}

	conv.Logger.LogInfof("Pass 1 complete: %d notes, %d repositories", len(conv.SharedNotes), len(conv.RepositoryIDMap))

	// Pass 2: Process records that depend on pass 1
	// Sources (depend on repositories)
	for _, record := range grouped[GedcomTagSour] {
		conv.Logger.LogInfo("Processing SOUR " + record.XRef)
		if err := convertSource(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagSour, err.Error())
		}
	}

	// Media objects (may reference repositories in some GEDCOM variants)
	for _, record := range grouped[GedcomTagObje] {
		conv.Logger.LogInfo("Processing OBJE " + record.XRef)
		if err := convertMedia(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagObje, err.Error())
		}
	}

	conv.Logger.LogInfof("Pass 2 complete: %d sources, %d media", conv.Stats.SourcesCreated, conv.Stats.MediaCreated)

	// Pass 3: Process individuals (depend on sources, media, notes for citations)
	for _, record := range grouped[GedcomTagIndi] {
		conv.Logger.LogInfo("Processing INDI " + record.XRef)
		if err := convertIndividual(record, conv); err != nil {
			conv.addError(record.Line, GedcomTagIndi, err.Error())
		}
	}

	// Process header and submitter (no strict dependency, but process after main entities)
	for _, record := range grouped[GedcomTagHead] {
		conv.Logger.LogInfo("Processing HEAD")
		convertHeader(record, conv)
	}
	for _, record := range grouped[GedcomTagSubm] {
		conv.Logger.LogInfo("Processing SUBM " + record.XRef)
		convertSubmitter(record, conv)
	}

	// Process extension tags
	for _, record := range grouped["_EXT"] {
		extData := convertExtensionData(record.Tag, record.Value, record.SubRecords)
		if len(extData) > 0 {
			conv.Logger.LogInfof("Processed extension tag %s: %+v", record.Tag, extData)
		}
	}

	// Log unknown tags
	for _, record := range grouped["_UNKNOWN"] {
		conv.addWarning(record.Line, record.Tag, "Unknown top-level tag: "+record.Tag)
	}

	// Store families for pass 4
	conv.DeferredFamilies = grouped[GedcomTagFam]

	conv.Logger.LogInfof("Pass 3 complete: %d persons", conv.Stats.PersonsCreated)

	// Second pass: Process families now that all individuals exist
	conv.Logger.LogInfof("Processing %d deferred families", len(conv.DeferredFamilies))
	for _, famRecord := range conv.DeferredFamilies {
		if err := convertFamily(famRecord, conv); err != nil {
			conv.addError(famRecord.Line, GedcomTagFam, err.Error())
		}
	}

	conv.Logger.LogInfof("Second pass complete: %d relationships created", conv.Stats.RelationshipsCreated)

	// Third pass: Create parent-child relationships with PEDI-based types
	conv.Logger.LogInfof("Processing %d deferred family links (FAMC)", len(conv.DeferredFamilyLinks))
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
					Type: relType,
					Participants: []Participant{
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

	conv.Logger.LogInfof("Third pass complete: total %d relationships", conv.Stats.RelationshipsCreated)

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

// convertHeader extracts metadata from HEAD record and stores it on the GLXFile.
func convertHeader(headRecord *GEDCOMRecord, conv *ConversionContext) {
	meta := &Metadata{}
	var submitterXRef string

	for _, sub := range headRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagDate:
			meta.ExportDate = DateString(sub.Value)
		case GedcomTagFile:
			meta.SourceFile = sub.Value
		case GedcomTagCopr:
			meta.Copyright = sub.Value
		case GedcomTagLang:
			meta.Language = sub.Value
		case GedcomTagSour:
			for _, sourSub := range sub.SubRecords {
				switch sourSub.Tag {
				case GedcomTagName:
					meta.SourceSystem = sourSub.Value
				case GedcomTagVers:
					meta.SourceVersion = sourSub.Value
				case GedcomTagCorp:
					meta.SourceCorporation = sourSub.Value
				}
			}
		case GedcomTagSubm:
			submitterXRef = sub.Value
		case GedcomTagGedc:
			for _, gedcSub := range sub.SubRecords {
				if gedcSub.Tag == GedcomTagVers {
					meta.GEDCOMVersion = gedcSub.Value
				}
			}
		case GedcomTagChar:
			meta.CharacterSet = sub.Value
		case GedcomTagNote:
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				if meta.Notes == "" {
					meta.Notes = noteText
				} else {
					meta.Notes = meta.Notes + "\n" + noteText
				}
			}
		}
	}

	// Always store the SUBM cross-reference so convertSubmitter can filter
	conv.SubmitterXRef = submitterXRef

	if meta.hasContent() {
		conv.GLX.ImportMetadata = meta
		conv.Logger.LogInfo("HEAD metadata stored")
	}
}

// convertSubmitter extracts submitter info from SUBM record and stores it on the GLXFile.
// Only attaches the submitter referenced by HEAD's SUBM pointer.
func convertSubmitter(submRecord *GEDCOMRecord, conv *ConversionContext) {
	// Only process the submitter referenced by HEAD's SUBM pointer.
	// If no SUBM pointer was found in HEAD, skip all SUBM records.
	if conv.SubmitterXRef == "" || submRecord.XRef != conv.SubmitterXRef {
		return
	}

	subm := &Submitter{}

	for _, sub := range submRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagName:
			subm.Name = sub.Value
		case GedcomTagAddr:
			subm.Address = extractAddress(sub)
		case GedcomTagPhon:
			subm.Phone = sub.Value
		case GedcomTagEmail:
			subm.Email = sub.Value
		case GedcomTagWww:
			subm.Website = sub.Value
		case GedcomTagObje:
			resolveOBJE(sub, conv)
		}
	}

	if subm.hasContent() {
		if conv.GLX.ImportMetadata == nil {
			conv.GLX.ImportMetadata = &Metadata{}
		}
		conv.GLX.ImportMetadata.Submitter = subm
		conv.Logger.LogInfo("SUBM submitter stored")
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
		case GedcomTagAdr1, GedcomTagAdr2, GedcomTagAdr3:
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case GedcomTagCity:
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case GedcomTagStae:
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case GedcomTagPost:
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		case GedcomTagCtry:
			if sub.Value != "" {
				parts = append(parts, sub.Value)
			}
		}
	}

	return strings.Join(parts, ", ")
}

// Converter functions implemented in separate files:
// convertSharedNote and convertExtensionSchema are implemented in gedcom_7_0.go

// groupRecordsByType groups GEDCOM records by their tag type for dependency-ordered processing
func groupRecordsByType(records []*GEDCOMRecord) map[string][]*GEDCOMRecord {
	grouped := make(map[string][]*GEDCOMRecord)

	for _, record := range records {
		switch record.Tag {
		case GedcomTagHead, GedcomTagTrlr, GedcomTagNote, GedcomTagSnote,
			GedcomTagRepo, GedcomTagSour, GedcomTagObje, GedcomTagIndi,
			GedcomTagFam, GedcomTagSubm, GedcomTagSchma:
			grouped[record.Tag] = append(grouped[record.Tag], record)
		default:
			if isExtensionTag(record.Tag) {
				grouped["_EXT"] = append(grouped["_EXT"], record)
			} else if record.Tag != GedcomTagTrlr {
				grouped["_UNKNOWN"] = append(grouped["_UNKNOWN"], record)
			}
		}
	}

	return grouped
}

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
