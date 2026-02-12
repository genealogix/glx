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
	"strconv"
	"strings"
)

// sourResult holds the result of processing a GEDCOM SOUR subrecord.
// If the SOUR has citation-level detail (PAGE, DATA, TEXT, QUAY, NOTE, OBJE),
// a Citation entity is created and CitationID is set.
// If it only references a source with no additional detail, SourceID is set instead
// so the caller can reference the source directly without a meaningless citation.
type sourResult struct {
	CitationID string
	SourceID   string
}

// createCitationFromSOUR processes a GEDCOM SOUR subrecord.
// Returns a sourResult: either a citation ID (if the SOUR has detail) or
// a bare source ID (if it only references a source with no added value).
//
//nolint:gocognit,gocyclo
func createCitationFromSOUR(sourRecord *GEDCOMRecord, conv *ConversionContext) (sourResult, error) {
	var sourceID string

	// Check if it's a reference or embedded source
	// GEDCOM has two SOURCE_CITATION forms:
	// 1. Pointer to source record: "SOUR @XREF@" - value starts and ends with @
	// 2. Embedded source description: "SOUR description text" - value is plain text
	if sourRecord.Value != "" && isGEDCOMPointer(sourRecord.Value) {
		// Reference to existing source (form 1)
		sourceID = conv.SourceIDMap[sourRecord.Value]
		if sourceID == "" {
			// Source doesn't exist yet, log warning but continue
			conv.Logger.LogWarning(sourRecord.Line, GedcomTagSour, sourRecord.Value, "Referenced source not found")

			return sourResult{}, fmt.Errorf("%w: %s", ErrSourceNotFound, sourRecord.Value)
		}
	} else {
		// Embedded citation (form 2) - create a synthetic source
		// Per GEDCOM spec: "systems need to create a SOURCE_RECORD format and store
		// the source description information found in the non-structured source citation
		// in the title area for the new source record."
		sourceID = createSyntheticSourceFromEmbeddedCitation(sourRecord, conv)
	}

	// Build citation from SOUR subrecords
	citation := &Citation{
		SourceID: sourceID,
	}

	// Extract citation details from SOUR subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagPage:
			// Page/location within source - may have CONT/CONC for long locators
			if propertyKey, ok := conv.GEDCOMIndex.CitationProperties[sub.Tag]; ok {
				if citation.Properties == nil {
					citation.Properties = make(map[string]any)
				}
				citation.Properties[propertyKey] = extractTextWithContinuation(sub)
			}

		case GedcomTagData:
			// Data from source
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case GedcomTagDate:
					// Source date - when the source recorded the information
					if propertyKey, ok := conv.GEDCOMIndex.CitationProperties[dataSub.Tag]; ok {
						dateStr := parseGEDCOMDate(dataSub.Value)
						if dateStr != "" {
							if citation.Properties == nil {
								citation.Properties = make(map[string]any)
							}
							citation.Properties[propertyKey] = string(dateStr)
						}
					}
				case GedcomTagText:
					// Text from source - may have CONT/CONC for long text
					if propertyKey, ok := conv.GEDCOMIndex.CitationProperties[dataSub.Tag]; ok {
						if citation.Properties == nil {
							citation.Properties = make(map[string]any)
						}
						citation.Properties[propertyKey] = extractTextWithContinuation(dataSub)
					}
				}
			}

		case GedcomTagText:
			// Text from source (GEDCOM 5.5.1) - may have CONT/CONC for long text
			if propertyKey, ok := conv.GEDCOMIndex.CitationProperties[sub.Tag]; ok {
				if citation.Properties == nil {
					citation.Properties = make(map[string]any)
				}
				citation.Properties[propertyKey] = extractTextWithContinuation(sub)
			}

		case GedcomTagQuay:
			// GEDCOM quality assessment (0-3) - preserve in notes
			if citation.Notes != "" {
				citation.Notes += "\nGEDCOM QUAY: " + sub.Value
			} else {
				citation.Notes = "GEDCOM QUAY: " + sub.Value
			}

		case GedcomTagNote:
			// Notes about the citation
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				if citation.Notes != "" {
					citation.Notes += "\n" + noteText
				} else {
					citation.Notes = noteText
				}
			}

		case GedcomTagObje:
			// Media linked to citation (reference or embedded)
			if mediaID := resolveOBJE(sub, conv); mediaID != "" {
				citation.Media = append(citation.Media, mediaID)
			}
		}
	}

	// If the citation adds no value beyond referencing the source, skip creating it
	// and return the source ID directly so the caller can reference the source.
	if !citationHasDetail(citation) {
		return sourResult{SourceID: sourceID}, nil
	}

	// Store citation
	citationID := generateCitationID(conv)
	conv.GLX.Citations[citationID] = citation
	conv.Stats.CitationsCreated++

	return sourResult{CitationID: citationID}, nil
}

// citationHasDetail reports whether a citation contains any data beyond its source reference.
func citationHasDetail(c *Citation) bool {
	return c.RepositoryID != "" || len(c.Properties) > 0 || c.Notes != "" || len(c.Media) > 0
}

// createPropertyAssertion creates an assertion for a property, but only if there is evidence.
// Assertions without evidence are not meaningful - the property value is already stored on the entity.
func createPropertyAssertion(subjectID, property string, value any, sourceRecord *GEDCOMRecord, conv *ConversionContext) {
	if property == "" || value == nil {
		return
	}

	// Extract evidence from SOUR subrecords
	refs := extractEvidence(sourceRecord, conv)

	createPropertyAssertionWithEvidence(subjectID, property, value, refs, conv)
}

// createPropertyAssertionWithEvidence creates an assertion for a property using pre-extracted evidence.
// This is used when evidence has already been extracted or synthetically created (e.g., census records).
func createPropertyAssertionWithEvidence(subjectID, property string, value any, refs evidenceRefs, conv *ConversionContext) {
	if property == "" || value == nil {
		return
	}

	// Only create assertion if there is evidence to back it up
	// The property value is already stored on the entity; assertions add evidence
	if !refs.hasEvidence() {
		return
	}

	// Generate assertion ID
	assertionID := generateAssertionID(conv)

	// Convert value to string
	var valueStr string
	switch v := value.(type) {
	case string:
		valueStr = v
	case int:
		valueStr = strconv.Itoa(v)
	case float64:
		valueStr = fmt.Sprintf("%f", v)
	default:
		valueStr = fmt.Sprintf("%v", v)
	}

	// Create assertion
	assertion := &Assertion{
		Subject:   EntityRef{Person: subjectID},
		Property:  property,
		Value:     valueStr,
		Sources:   refs.SourceIDs,
		Citations: refs.CitationIDs,
	}

	// Store assertion
	conv.GLX.Assertions[assertionID] = assertion
	conv.Stats.AssertionsCreated++
}

// evidenceRefs holds citation IDs and bare source IDs extracted from SOUR subrecords.
type evidenceRefs struct {
	CitationIDs []string
	SourceIDs   []string
}

// extractEvidence extracts all evidence references from a record's SOUR subrecords.
// SOUR records that contain citation-level detail (PAGE, DATA, TEXT, QUAY, NOTE, OBJE)
// produce citation IDs. SOUR records that only reference a source with no additional
// detail produce bare source IDs, avoiding meaningless citation entities.
func extractEvidence(record *GEDCOMRecord, conv *ConversionContext) evidenceRefs {
	var refs evidenceRefs

	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagSour {
			result, err := createCitationFromSOUR(sub, conv)
			if err != nil {
				// Error already logged in createCitationFromSOUR, skip
				continue
			}
			if result.CitationID != "" {
				refs.CitationIDs = append(refs.CitationIDs, result.CitationID)
			} else if result.SourceID != "" {
				refs.SourceIDs = append(refs.SourceIDs, result.SourceID)
			}
		}
	}

	return refs
}

// hasEvidence reports whether the refs contain any citations or source references.
func (r evidenceRefs) hasEvidence() bool {
	return len(r.CitationIDs) > 0 || len(r.SourceIDs) > 0
}

// extractNoteText extracts note text from NOTE record
func extractNoteText(noteRecord *GEDCOMRecord, conv *ConversionContext) string {
	// Check if it's a reference to a shared note (works for both GEDCOM 5.5.1 and 7.0)
	if noteRecord.Value != "" {
		if sharedNote, exists := conv.SharedNotes[noteRecord.Value]; exists {
			return sharedNote
		}
	}

	if noteRecord.Value != "" {
		// Inline note
		var text strings.Builder
		text.WriteString(noteRecord.Value)

		// Check for CONT/CONC subrecords
		for _, sub := range noteRecord.SubRecords {
			switch sub.Tag {
			case GedcomTagCont:
				// Continuation on new line
				text.WriteString("\n" + sub.Value)
			case GedcomTagConc:
				// Concatenation (continues same line)
				text.WriteString(sub.Value)
			}
		}

		return text.String()
	}

	// Build from CONT/CONC subrecords only
	var textBuilder strings.Builder
	for _, sub := range noteRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagCont:
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			textBuilder.WriteString(sub.Value)
		case GedcomTagConc:
			textBuilder.WriteString(sub.Value)
		}
	}

	return textBuilder.String()
}

// extractTextWithContinuation extracts text from a GEDCOM record that may have
// CONT (continuation on new line) and CONC (concatenation on same line) subrecords.
// This should be used for any text field that might be split across multiple lines.
func extractTextWithContinuation(record *GEDCOMRecord) string {
	if record.Value == "" && len(record.SubRecords) == 0 {
		return ""
	}

	var text strings.Builder
	text.WriteString(record.Value)

	for _, sub := range record.SubRecords {
		switch sub.Tag {
		case GedcomTagCont:
			// Continuation on new line
			text.WriteString("\n" + sub.Value)
		case GedcomTagConc:
			// Concatenation (continues same line, no space added)
			text.WriteString(sub.Value)
		}
	}

	return text.String()
}

// isGEDCOMPointer checks if a value is a GEDCOM cross-reference pointer (e.g., "@I1@", "@S23@")
func isGEDCOMPointer(value string) bool {
	return len(value) >= 3 && value[0] == '@' && value[len(value)-1] == '@'
}

// createSyntheticSourceFromEmbeddedCitation creates a Source entity from an embedded citation
// This follows the GEDCOM spec recommendation: when encountering embedded citations,
// create a SOURCE_RECORD with the description as the title.
func createSyntheticSourceFromEmbeddedCitation(sourRecord *GEDCOMRecord, conv *ConversionContext) string {
	sourceID := generateSourceID(conv)

	// Determine the source title
	var title string
	if sourRecord.Value != "" {
		// Use the embedded source description as the title (may have CONT/CONC)
		title = extractTextWithContinuation(sourRecord)
	} else {
		// No description provided - try to extract from TEXT subrecord
		for _, sub := range sourRecord.SubRecords {
			if sub.Tag == GedcomTagText {
				title = extractTextWithContinuation(sub)

				break
			}
		}
		// Fallback to generic title
		if title == "" {
			title = "Embedded Citation (No Source Description)"
		}
	}

	// Create the synthetic source
	source := &Source{
		Title: title,
	}

	// Check for TEXT at the SOUR level (may contain additional source text)
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagText:
			// TEXT directly under SOUR provides source text
			if source.Description == "" {
				source.Description = extractTextWithContinuation(sub)
			}
		case GedcomTagNote:
			// Notes about the source
			noteText := extractNoteText(sub, conv)
			if noteText != "" {
				if source.Notes != "" {
					source.Notes += "\n" + noteText
				} else {
					source.Notes = noteText
				}
			}
		}
	}

	// Add note indicating this is a synthetic source from embedded citation
	syntheticNote := "Source created from embedded GEDCOM citation"
	if source.Notes != "" {
		source.Notes = syntheticNote + "\n" + source.Notes
	} else {
		source.Notes = syntheticNote
	}

	// Store the source
	conv.GLX.Sources[sourceID] = source
	conv.Stats.SourcesCreated++

	conv.Logger.LogInfo(fmt.Sprintf("Line %d: Created synthetic source from embedded citation: %s", sourRecord.Line, title))

	return sourceID
}
