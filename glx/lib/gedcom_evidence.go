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
func createCitationFromSOUR(subjectID string, sourRecord *GEDCOMRecord, conv *ConversionContext) (string, error) {
	var sourceID string

	// Check if it's a reference or embedded source
	if sourRecord.Value != "" {
		// Reference to existing source
		sourceID = conv.SourceIDMap[sourRecord.Value]
		if sourceID == "" {
			// Source doesn't exist yet, log warning but continue
			conv.Logger.LogWarning(sourRecord.Line, GedcomTagSour, sourRecord.Value, "Referenced source not found")

			return "", fmt.Errorf("%w: %s", ErrSourceNotFound, sourRecord.Value)
		}
	} else {
		// Embedded citation (citation details without full source)
		// Not yet implemented (see todo.md)
		return "", nil
	}

	// Create citation
	citationID := generateCitationID(conv)

	citation := &Citation{
		SourceID: sourceID,
	}

	// Extract citation details from SOUR subrecords
	for _, sub := range sourRecord.SubRecords {
		switch sub.Tag {
		case GedcomTagPage:
			// Page/location within source
			citation.Page = sub.Value

		case GedcomTagData:
			// Data from source
			for _, dataSub := range sub.SubRecords {
				switch dataSub.Tag {
				case GedcomTagDate:
					// Source date stored in notes (should be a field - see todo.md)
					dateStr := parseGEDCOMDate(dataSub.Value)
					if dateStr != "" {
						if citation.Notes != "" {
							citation.Notes += "\nSource date: " + string(dateStr)
						} else {
							citation.Notes = "Source date: " + string(dateStr)
						}
					}
				case GedcomTagText:
					citation.TextFromSource = dataSub.Value
				}
			}

		case GedcomTagText:
			// Text from source (GEDCOM 5.5.1)
			citation.TextFromSource = sub.Value

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
			// Media linked to citation (not commonly used, but supported)
			if sub.Value != "" {
				mediaID := conv.MediaIDMap[sub.Value]
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
	conv.GLX.Citations[citationID] = citation
	conv.Stats.CitationsCreated++

	return citationID, nil
}

// createPropertyAssertion creates an assertion for a property
func createPropertyAssertion(subjectID string, claim string, value any, sourceRecord *GEDCOMRecord, conv *ConversionContext) {
	if claim == "" || value == nil {
		return
	}

	// Extract citations from SOUR subrecords
	citationIDs := extractCitations(subjectID, sourceRecord, conv)

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
		Subject:   subjectID,
		Claim:     claim,
		Value:     valueStr,
		Citations: citationIDs,
	}

	// Store assertion
	conv.GLX.Assertions[assertionID] = assertion
	conv.Stats.AssertionsCreated++
}

// extractCitations extracts all citations from a record's SOUR subrecords
func extractCitations(subjectID string, record *GEDCOMRecord, conv *ConversionContext) []string {
	var citationIDs []string

	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagSour {
			citationID, err := createCitationFromSOUR(subjectID, sub, conv)
			if err == nil && citationID != "" {
				citationIDs = append(citationIDs, citationID)
			}
		}
	}

	return citationIDs
}

// extractNoteText extracts note text from NOTE record
func extractNoteText(noteRecord *GEDCOMRecord, conv *ConversionContext) string {
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

	// Check if it's a reference to shared note (GEDCOM 7.0)
	if conv.Version == GEDCOM70 && noteRecord.Value != "" {
		if sharedNote, exists := conv.SharedNotes[noteRecord.Value]; exists {
			return sharedNote
		}
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
