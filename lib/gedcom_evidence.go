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
	"strconv"
	"strings"
)

// createCitationFromSOUR creates a citation from a GEDCOM SOUR subrecord
func createCitationFromSOUR(subjectID string, sourRecord *GEDCOMRecord, ctx *ConversionContext) (string, error) {
	var sourceID string

	// Check if it's a reference or embedded source
	if sourRecord.Value != "" {
		// Reference to existing source
		sourceID = ctx.SourceIDMap[sourRecord.Value]
		if sourceID == "" {
			// Source doesn't exist yet, log warning but continue
			ctx.Logger.LogWarning(sourRecord.Line, "SOUR", sourRecord.Value, "Referenced source not found")
			return "", fmt.Errorf("source not found: %s", sourRecord.Value)
		}
	} else {
		// Embedded citation (citation details without full source)
		// For now, skip embedded citations
		return "", nil
	}

	// Create citation
	citationID := generateCitationID(ctx)

	citation := &Citation{
		Source:     sourceID,
		Properties: make(map[string]interface{}),
	}

	// Extract citation details from SOUR subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case "PAGE":
			// Page/location within source
			citation.Page = sub.Value

		case "DATA":
			// Data from source
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case "DATE":
					citation.Properties["source_date"] = parseGEDCOMDate(dataSub.Value)
				case "TEXT":
					citation.TextFromSource = dataSub.Value
				}
			}

		case "TEXT":
			// Text from source (GEDCOM 5.5.1)
			citation.TextFromSource = sub.Value

		case "QUAY":
			// Quality assessment (0-3)
			if quay, err := strconv.Atoi(sub.Value); err == nil {
				citation.Properties["quay"] = quay
			}

		case "NOTE":
			// Notes about the citation
			noteText := extractNoteText(sub, ctx)
			if noteText != "" {
				citation.Properties["notes"] = noteText
			}

		case "OBJE":
			// Media linked to citation (not commonly used, but supported)
			if sub.Value != "" {
				mediaID := ctx.MediaIDMap[sub.Value]
				if mediaID != "" {
					if citation.Media == nil {
						citation.Media = []string{}
					}
					citation.Media = append(citation.Media, mediaID)
				}
			}
		}
	}

	// Store citation
	ctx.GLX.Citations[citationID] = citation
	ctx.Stats.CitationsCreated++

	return citationID, nil
}

// createPropertyAssertion creates an assertion for a property
func createPropertyAssertion(subjectID string, claim string, value interface{}, sourceRecord *GEDCOMRecord, ctx *ConversionContext) error {
	if claim == "" || value == nil {
		return nil
	}

	// Extract citations from SOUR subrecords
	citationIDs := extractCitations(subjectID, sourceRecord, ctx)

	// Generate assertion ID
	assertionID := generateAssertionID(ctx)

	// Derive confidence
	confidence := deriveConfidence(citationIDs, ctx)

	// Create assertion
	assertion := &Assertion{
		Subject:    subjectID,
		Claim:      claim,
		Value:      value,
		Confidence: confidence,
		Citations:  citationIDs,
	}

	// Store assertion
	ctx.GLX.Assertions[assertionID] = assertion
	ctx.Stats.AssertionsCreated++

	return nil
}

// extractCitations extracts all citations from a record's SOUR subrecords
func extractCitations(subjectID string, record *GEDCOMRecord, ctx *ConversionContext) []string {
	var citationIDs []string

	for _, sub := range record.SubRecords {
		if sub.Tag == "SOUR" {
			citationID, err := createCitationFromSOUR(subjectID, sub, ctx)
			if err == nil && citationID != "" {
				citationIDs = append(citationIDs, citationID)
			}
		}
	}

	return citationIDs
}

// deriveConfidence derives confidence level from citations
func deriveConfidence(citationIDs []string, ctx *ConversionContext) string {
	if len(citationIDs) == 0 {
		return "medium" // Default when no citations
	}

	// Check QUAY values
	highestQuality := -1
	for _, citationID := range citationIDs {
		citation := ctx.GLX.Citations[citationID]
		if citation != nil {
			if quay, ok := citation.Properties["quay"].(int); ok {
				if quay > highestQuality {
					highestQuality = quay
				}
			}
		}
	}

	// Map QUAY to confidence
	return mapQUAYtoConfidence(highestQuality)
}

// mapQUAYtoConfidence maps GEDCOM QUAY values (0-3) to GLX confidence levels
func mapQUAYtoConfidence(quay int) string {
	switch quay {
	case 0:
		return "very_low"
	case 1:
		return "low"
	case 2:
		return "medium"
	case 3:
		return "high"
	default:
		return "medium" // Default
	}
}

// extractNoteText extracts note text from NOTE record
func extractNoteText(noteRecord *GEDCOMRecord, ctx *ConversionContext) string {
	if noteRecord.Value != "" {
		// Inline note
		text := noteRecord.Value

		// Check for CONT/CONC subrecords
		for _, sub := range noteRecord.SubRecords {
			switch sub.Tag {
			case "CONT":
				// Continuation on new line
				text += "\n" + sub.Value
			case "CONC":
				// Concatenation (continues same line)
				text += sub.Value
			}
		}

		return text
	}

	// Check if it's a reference to shared note (GEDCOM 7.0)
	if ctx.Version == GEDCOM70 && noteRecord.Value != "" {
		if sharedNote, exists := ctx.SharedNotes[noteRecord.Value]; exists {
			return sharedNote
		}
	}

	// Build from CONT/CONC subrecords only
	var textBuilder strings.Builder
	for _, sub := range noteRecord.SubRecords {
		switch sub.Tag {
		case "CONT":
			if textBuilder.Len() > 0 {
				textBuilder.WriteString("\n")
			}
			textBuilder.WriteString(sub.Value)
		case "CONC":
			textBuilder.WriteString(sub.Value)
		}
	}

	return textBuilder.String()
}
