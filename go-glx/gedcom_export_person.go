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
	"sort"
	"strings"
)

// Properties that are handled specially and should not be exported as generic tags.
var skipPersonProperties = map[string]bool{
	PersonPropertyName:      true,
	PersonPropertyGender:    true,
	DeprecatedPropertyBornOn: true,
	DeprecatedPropertyBornAt: true,
	DeprecatedPropertyDiedOn: true,
	DeprecatedPropertyDiedAt: true,
	PersonPropertyResidence: true,
	PropertyNotes:           true,
	PropertyMedia:           true,
	PropertySources:         true,
	PropertyCitations:       true,
}

// buildPersonEventsIndex scans all events and builds a map from person ID
// to the event IDs where that person participates as principal.
// This avoids scanning all events for each person during export.
func buildPersonEventsIndex(expCtx *ExportContext) {
	expCtx.PersonEvents = make(map[string][]string)

	// Sort event IDs for deterministic output
	eventIDs := sortedKeys(expCtx.GLX.Events)
	for _, eventID := range eventIDs {
		event := expCtx.GLX.Events[eventID]
		if event == nil {
			continue
		}
		for _, participant := range event.Participants {
			if participant.Role == ParticipantRolePrincipal && participant.Person != "" {
				expCtx.PersonEvents[participant.Person] = append(
					expCtx.PersonEvents[participant.Person], eventID)
			}
		}
	}
}

// buildPersonPropertyAssertionsIndex builds a lookup from (personID, property) to
// assertions. This is used to export SOUR on NAME, OCCU, RESI, and other records
// that have assertion-based evidence.
func buildPersonPropertyAssertionsIndex(expCtx *ExportContext) {
	expCtx.PersonPropertyAssertions = make(map[string]map[string][]*Assertion)

	for _, assertion := range expCtx.GLX.Assertions {
		personID := assertion.Subject.Person
		if personID == "" || assertion.Property == "" {
			continue
		}
		if len(assertion.Sources) == 0 && len(assertion.Citations) == 0 {
			continue
		}

		if _, ok := expCtx.PersonPropertyAssertions[personID]; !ok {
			expCtx.PersonPropertyAssertions[personID] = make(map[string][]*Assertion)
		}
		expCtx.PersonPropertyAssertions[personID][assertion.Property] = append(
			expCtx.PersonPropertyAssertions[personID][assertion.Property], assertion)
	}
}

// exportAssertionSourceRefs adds SOUR subrecords from assertion sources and citations.
func exportAssertionSourceRefs(assertions []*Assertion, expCtx *ExportContext, record *GEDCOMRecord) {
	for _, assertion := range assertions {
		// Direct sources
		for _, sourceID := range assertion.Sources {
			if sourceXRef := expCtx.SourceXRefMap[sourceID]; sourceXRef != "" {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagSour,
					Value: sourceXRef,
				})
			}
		}

		// Citations
		for _, citationID := range assertion.Citations {
			citation, exists := expCtx.GLX.Citations[citationID]
			if !exists {
				continue
			}

			if sourceXRef := expCtx.SourceXRefMap[citation.SourceID]; sourceXRef != "" {
				record.SubRecords = append(record.SubRecords, exportCitationAsSOUR(citation, sourceXRef, expCtx))
			}
		}
	}
}

// exportPerson converts a GLX Person to a GEDCOM INDI record.
func exportPerson(personID string, person *Person, expCtx *ExportContext) *GEDCOMRecord {
	xref := expCtx.PersonXRefMap[personID]

	record := &GEDCOMRecord{
		XRef:       xref,
		Tag:        GedcomTagIndi,
		SubRecords: []*GEDCOMRecord{},
	}

	// NAME records
	if nameVal, ok := person.Properties[PersonPropertyName]; ok {
		nameRecords := exportNameRecords(personID, nameVal, expCtx)
		record.SubRecords = append(record.SubRecords, nameRecords...)
	}

	// SEX
	if gender, ok := getStringProperty(person.Properties, PersonPropertyGender); ok {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagSex,
			Value: mapGenderToSex(gender, expCtx),
		})
	}

	// Person events (BIRT, DEAT, BAPM, etc.)
	if eventIDs, ok := expCtx.PersonEvents[personID]; ok {
		for _, eventID := range eventIDs {
			event := expCtx.GLX.Events[eventID]
			if event == nil {
				continue
			}
			eventRecord := exportPersonEvent(event, expCtx)
			if eventRecord != nil {
				record.SubRecords = append(record.SubRecords, eventRecord)
				expCtx.Stats.EventsProcessed++
			}
		}
	}

	// RESI (residence) records from temporal property
	record.SubRecords = append(record.SubRecords, exportResidenceRecords(personID, person, expCtx)...)

	// Person properties with GEDCOM tag mappings (OCCU, RELI, EDUC, etc.)
	record.SubRecords = append(record.SubRecords, exportMappedPersonProperties(personID, person, expCtx)...)

	// NOTE - check both struct field and Properties map
	noteText := person.Notes
	if noteText == "" {
		if propNotes, ok := person.Properties[PropertyNotes].(string); ok {
			noteText = propNotes
		}
	}
	if noteText != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: noteText,
		})
	}

	// OBJE references from media property
	exportPersonMediaRefs(personID, person, expCtx, record)

	// SOUR references from sources and citations properties
	exportPersonSourceRefs(personID, person, expCtx, record)

	// FAMS back-references (families where this person is a spouse)
	if famsXRefs, ok := expCtx.PersonSpouseFamilies[personID]; ok {
		for _, famsXRef := range famsXRefs {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagFams,
				Value: famsXRef,
			})
		}
	}

	// FAMC back-references (families where this person is a child)
	if famcRefs, ok := expCtx.PersonChildFamilies[personID]; ok {
		for _, famcRef := range famcRefs {
			famcRecord := &GEDCOMRecord{
				Tag:   GedcomTagFamc,
				Value: famcRef.FamilyXRef,
			}
			if famcRef.Pedigree != "" {
				famcRecord.SubRecords = []*GEDCOMRecord{
					{Tag: GedcomTagPedi, Value: famcRef.Pedigree},
				}
			}
			record.SubRecords = append(record.SubRecords, famcRecord)
		}
	}

	return record
}

// exportNameRecords handles the name property which can be a single map or a list.
// Returns one or more NAME GEDCOMRecords.
func exportNameRecords(personID string, nameVal any, expCtx *ExportContext) []*GEDCOMRecord {
	// Look up all name assertions for this person
	var nameAssertions []*Assertion
	if personAssertions, ok := expCtx.PersonPropertyAssertions[personID]; ok {
		nameAssertions = personAssertions[PersonPropertyName]
	}

	switch v := nameVal.(type) {
	case map[string]any:
		rec := exportNameRecord(v, nameAssertions, expCtx)
		if rec != nil {
			return []*GEDCOMRecord{rec}
		}
	case []any:
		var records []*GEDCOMRecord
		for _, item := range v {
			if nameMap, ok := item.(map[string]any); ok {
				rec := exportNameRecord(nameMap, nameAssertions, expCtx)
				if rec != nil {
					records = append(records, rec)
				}
			}
		}
		return records
	}

	return nil
}

// exportNameRecord converts a single name property value (map with "value" and "fields")
// to a GEDCOM NAME record with substructure.
func exportNameRecord(nameMap map[string]any, nameAssertions []*Assertion, expCtx *ExportContext) *GEDCOMRecord {
	value, _ := nameMap["value"].(string)
	if value == "" {
		return nil
	}

	// Extract fields if present
	var fields map[string]any
	if fieldsVal, ok := nameMap["fields"]; ok {
		fields, _ = fieldsVal.(map[string]any)
	}

	// Build the GEDCOM NAME value: "Given /Surname/ Suffix"
	nameValue := formatGEDCOMNameValue(value, fields)

	record := &GEDCOMRecord{
		Tag:        GedcomTagName,
		Value:      nameValue,
		SubRecords: []*GEDCOMRecord{},
	}

	// Add substructure tags from fields
	if fields != nil {
		// TYPE (birth, married, aka, etc.)
		if nameType, ok := fields[NameFieldType].(string); ok && nameType != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagType,
				Value: nameType,
			})
		}

		// NPFX (name prefix)
		if prefix, ok := fields[NameFieldPrefix].(string); ok && prefix != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagNpfx,
				Value: prefix,
			})
		}

		// GIVN (given name)
		if given, ok := fields[NameFieldGiven].(string); ok && given != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagGivn,
				Value: given,
			})
		}

		// NICK (nickname)
		if nick, ok := fields[NameFieldNickname].(string); ok && nick != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagNick,
				Value: nick,
			})
		}

		// SPFX (surname prefix)
		if spfx, ok := fields[NameFieldSurnamePrefix].(string); ok && spfx != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagSpfx,
				Value: spfx,
			})
		}

		// SURN (surname)
		if surn, ok := fields[NameFieldSurname].(string); ok && surn != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagSurn,
				Value: surn,
			})
		}

		// NSFX (name suffix)
		if nsfx, ok := fields[NameFieldSuffix].(string); ok && nsfx != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagNsfx,
				Value: nsfx,
			})
		}
	}

	// SOUR from assertions matching this name value
	for _, assertion := range nameAssertions {
		if assertion.Value == value {
			exportAssertionSourceRefs([]*Assertion{assertion}, expCtx, record)
		}
	}

	return record
}

// formatGEDCOMNameValue constructs a GEDCOM-formatted name string.
// If fields with given/surname are available, format as "Given /Surname/ Suffix".
// Otherwise, parse the value string to extract surname for slash notation.
func formatGEDCOMNameValue(value string, fields map[string]any) string {
	if fields != nil {
		given, _ := fields[NameFieldGiven].(string)
		surname, _ := fields[NameFieldSurname].(string)
		surnamePrefix, _ := fields[NameFieldSurnamePrefix].(string)
		suffix, _ := fields[NameFieldSuffix].(string)

		// Build "Given /SurnamePrefix Surname/ Suffix"
		var parts []string

		if given != "" {
			parts = append(parts, given)
		}

		// Construct the surname part with slashes
		fullSurname := strings.TrimSpace(surnamePrefix + " " + surname)
		fullSurname = strings.TrimSpace(fullSurname)
		if fullSurname != "" {
			parts = append(parts, "/"+fullSurname+"/")
		} else {
			// Even with no surname, GEDCOM expects //
			parts = append(parts, "//")
		}

		if suffix != "" {
			parts = append(parts, suffix)
		}

		result := strings.Join(parts, " ")
		return strings.TrimSpace(result)
	}

	// No fields available — try to parse the value string
	// Look for a surname that could be the last word
	return parseValueToGEDCOMName(value)
}

// parseValueToGEDCOMName attempts to convert a simple "Given Surname" string
// to GEDCOM "Given /Surname/" format by treating the last word as surname.
func parseValueToGEDCOMName(value string) string {
	// If already contains slashes, return as-is
	if strings.Contains(value, "/") {
		return value
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return value
	}

	if len(words) == 1 {
		// Single name — could be given or surname, use as given with empty surname
		return words[0] + " //"
	}

	// Treat last word as surname, rest as given
	given := strings.Join(words[:len(words)-1], " ")
	surname := words[len(words)-1]

	return given + " /" + surname + "/"
}

// mapGenderToSex converts a GLX gender value to a GEDCOM SEX value.
// Uses the gender_types vocabulary for lookup; falls back to hardcoded
// mapping for standard values when the vocabulary is not loaded.
func mapGenderToSex(gender string, expCtx *ExportContext) string {
	// Try vocabulary lookup first
	if expCtx != nil && expCtx.GLX != nil && expCtx.GLX.GenderTypes != nil {
		if genderType, ok := expCtx.GLX.GenderTypes[gender]; ok && genderType != nil && genderType.GEDCOM != "" {
			return genderType.GEDCOM
		}
	}

	// Fallback for standard values
	switch gender {
	case GenderMale:
		return "M"
	case GenderFemale:
		return "F"
	case GenderOther:
		return "X"
	default:
		return "U"
	}
}

// exportPersonEvent converts a GLX Event to a GEDCOM event subrecord of INDI.
// Returns nil if the event type has no GEDCOM mapping.
func exportPersonEvent(event *Event, expCtx *ExportContext) *GEDCOMRecord {
	gedcomTag, ok := expCtx.ExportIndex.EventTypes[event.Type]
	if !ok || gedcomTag == "" {
		return nil
	}

	record := &GEDCOMRecord{
		Tag:        gedcomTag,
		SubRecords: []*GEDCOMRecord{},
	}

	// DATE
	if event.Date != "" {
		gedcomDate := formatGEDCOMDate(event.Date)
		if gedcomDate != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagDate,
				Value: gedcomDate,
			})
		}
	}

	// PLAC (from PlaceStrings cache)
	placRecords := exportPlaceSubrecords(event.PlaceID, expCtx)
	if placRecords != nil {
		record.SubRecords = append(record.SubRecords, placRecords...)
	}

	// Event properties (AGE, CAUS, TYPE)
	record.SubRecords = append(record.SubRecords, exportEventPropertySubrecords(event, expCtx)...)

	// NOTE - check both struct field and Properties map
	eventNoteText := event.Notes
	if eventNoteText == "" {
		if propNotes, ok := event.Properties[PropertyNotes].(string); ok {
			eventNoteText = propNotes
		}
	}
	if eventNoteText != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: eventNoteText,
		})
	}

	// SOUR references from event sources and citations
	exportEventSourceRefs(event, expCtx, record)

	return record
}

// exportEventPropertySubrecords exports event properties that have GEDCOM tag mappings.
func exportEventPropertySubrecords(event *Event, expCtx *ExportContext) []*GEDCOMRecord {
	if len(event.Properties) == 0 {
		return nil
	}

	var records []*GEDCOMRecord

	// Sort property keys for deterministic output
	keys := make([]string, 0, len(event.Properties))
	for k := range event.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		gedcomTag, ok := expCtx.ExportIndex.EventProperties[key]
		if !ok || gedcomTag == "" {
			continue
		}

		val := event.Properties[key]
		if s, ok := val.(string); ok && s != "" {
			records = append(records, &GEDCOMRecord{
				Tag:   gedcomTag,
				Value: s,
			})
		}
	}

	return records
}

// exportResidenceRecords exports RESI records from the person's residence temporal property.
func exportResidenceRecords(personID string, person *Person, expCtx *ExportContext) []*GEDCOMRecord {
	resiVal, ok := person.Properties[PersonPropertyResidence]
	if !ok {
		return nil
	}

	var resiList []any
	switch v := resiVal.(type) {
	case []any:
		resiList = v
	case string:
		resiList = []any{v}
	case map[string]any:
		resiList = []any{v}
	default:
		return nil
	}

	// Look up assertions for residence citations
	var assertions []*Assertion
	if personAssertions, ok := expCtx.PersonPropertyAssertions[personID]; ok {
		assertions = personAssertions[PersonPropertyResidence]
	}

	// Build a map from placeID -> assertions for matching
	assertionsByPlace := make(map[string][]*Assertion)
	for _, a := range assertions {
		assertionsByPlace[a.Value] = append(assertionsByPlace[a.Value], a)
	}

	var records []*GEDCOMRecord

	for _, entry := range resiList {
		record := &GEDCOMRecord{
			Tag:        GedcomTagResi,
			SubRecords: []*GEDCOMRecord{},
		}

		var placeID, dateStr string

		switch v := entry.(type) {
		case string:
			placeID = v
		case map[string]any:
			if val, ok := v["value"].(string); ok {
				placeID = val
			}
			if d, ok := v["date"].(string); ok {
				dateStr = d
			}
		default:
			continue
		}

		if dateStr != "" {
			gedcomDate := formatGEDCOMDate(DateString(dateStr))
			if gedcomDate != "" {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagDate,
					Value: gedcomDate,
				})
			}
		}

		if placeID != "" {
			placRecords := exportPlaceSubrecords(placeID, expCtx)
			if placRecords != nil {
				record.SubRecords = append(record.SubRecords, placRecords...)
			} else {
				// Not a known place ID — emit as RESI value text
				record.Value = placeID
			}

			// Add SOUR from assertions matching this place
			if placeAssertions, ok := assertionsByPlace[placeID]; ok {
				exportAssertionSourceRefs(placeAssertions, expCtx, record)
			}
		}

		records = append(records, record)
	}

	return records
}

// exportMappedPersonProperties exports person properties that have GEDCOM tag mappings
// (e.g., occupation -> OCCU, religion -> RELI), skipping properties handled elsewhere.
func exportMappedPersonProperties(personID string, person *Person, expCtx *ExportContext) []*GEDCOMRecord {
	if len(person.Properties) == 0 {
		return nil
	}

	var records []*GEDCOMRecord

	// Sort property keys for deterministic output
	keys := make([]string, 0, len(person.Properties))
	for k := range person.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		// Skip properties handled elsewhere
		if skipPersonProperties[key] {
			continue
		}

		gedcomTag, ok := expCtx.ExportIndex.PersonProperties[key]
		if !ok || gedcomTag == "" {
			continue
		}

		val := person.Properties[key]

		// Collect property items — single string, or list of strings/{value:, date:} maps
		type propItem struct {
			value string
			date  string
			place string
		}
		var items []propItem
		switch v := val.(type) {
		case string:
			if v != "" {
				items = []propItem{{value: v}}
			}
		case map[string]any:
			if s, ok := v["value"].(string); ok && s != "" {
				pi := propItem{value: s}
				if d, ok := v["date"].(string); ok {
					pi.date = d
				}
				if p, ok := v["place"].(string); ok {
					pi.place = p
				}
				items = []propItem{pi}
			}
		case []any:
			for _, item := range v {
				switch it := item.(type) {
				case string:
					if it != "" {
						items = append(items, propItem{value: it})
					}
				case map[string]any:
					if s, ok := it["value"].(string); ok && s != "" {
						pi := propItem{value: s}
						if d, ok := it["date"].(string); ok {
							pi.date = d
						}
						if p, ok := it["place"].(string); ok {
							pi.place = p
						}
						items = append(items, pi)
					}
				}
			}
		}

		for _, item := range items {
			propRecord := &GEDCOMRecord{
				Tag:        gedcomTag,
				Value:      item.value,
				SubRecords: []*GEDCOMRecord{},
			}

			// Emit DATE sub-record from temporal list item
			if item.date != "" {
				propRecord.SubRecords = append(propRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagDate,
					Value: formatGEDCOMDate(DateString(item.date)),
				})
			}

			// Emit PLAC sub-record from temporal list item
			if item.place != "" {
				propRecord.SubRecords = append(propRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagPlac,
					Value: item.place,
				})
			}

			// Add SOUR from assertions matching this specific value
			if personAssertions, ok := expCtx.PersonPropertyAssertions[personID]; ok {
				if assertions, ok := personAssertions[key]; ok {
					var matching []*Assertion
					for _, a := range assertions {
						if a.Value == item.value {
							matching = append(matching, a)
						}
					}
					if len(matching) > 0 {
						exportAssertionSourceRefs(matching, expCtx, propRecord)
					}
				}
			}

			records = append(records, propRecord)
		}
	}

	return records
}

// exportPersonMediaRefs adds OBJE references for media attached to a person.
func exportPersonMediaRefs(personID string, person *Person, expCtx *ExportContext, record *GEDCOMRecord) {
	mediaVal, ok := person.Properties[PropertyMedia]
	if !ok {
		return
	}

	mediaIDs := extractStringList(mediaVal)
	for _, mediaID := range mediaIDs {
		mediaXRef := expCtx.MediaXRefMap[mediaID]
		if mediaXRef != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagObje,
				Value: mediaXRef,
			})
		} else {
			expCtx.addExportWarning(EntityTypePersons, personID,
				fmt.Sprintf("media %s has no XREF mapping", mediaID))
		}
	}
}

// exportPersonSourceRefs adds SOUR references for sources and citations attached to a person.
func exportPersonSourceRefs(personID string, person *Person, expCtx *ExportContext, record *GEDCOMRecord) {
	// Sources
	if sourcesVal, ok := person.Properties[PropertySources]; ok {
		sourceIDs := extractStringList(sourcesVal)
		for _, sourceID := range sourceIDs {
			sourceXRef := expCtx.SourceXRefMap[sourceID]
			if sourceXRef != "" {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagSour,
					Value: sourceXRef,
				})
			} else {
				expCtx.addExportWarning(EntityTypePersons, personID,
					fmt.Sprintf("source %s has no XREF mapping", sourceID))
			}
		}
	}

	// Citations (resolve to source references)
	if citationsVal, ok := person.Properties[PropertyCitations]; ok {
		citationIDs := extractStringList(citationsVal)
		for _, citationID := range citationIDs {
			citation, exists := expCtx.GLX.Citations[citationID]
			if !exists {
				expCtx.addExportWarning(EntityTypePersons, personID,
					fmt.Sprintf("citation %s not found", citationID))
				continue
			}

			sourceXRef := expCtx.SourceXRefMap[citation.SourceID]
			if sourceXRef != "" {
				sourRecord := &GEDCOMRecord{
					Tag:        GedcomTagSour,
					Value:      sourceXRef,
					SubRecords: []*GEDCOMRecord{},
				}

				// PAGE from locator
				if locator, ok := getStringProperty(citation.Properties, "locator"); ok {
					sourRecord.SubRecords = append(sourRecord.SubRecords, &GEDCOMRecord{
						Tag:   GedcomTagPage,
						Value: locator,
					})
				}

				record.SubRecords = append(record.SubRecords, sourRecord)
			} else {
				expCtx.addExportWarning(EntityTypePersons, personID,
					fmt.Sprintf("citation %s source %s has no XREF mapping", citationID, citation.SourceID))
			}
		}
	}
}

// exportEventSourceRefs adds SOUR references for sources and citations attached to an event.
func exportEventSourceRefs(event *Event, expCtx *ExportContext, record *GEDCOMRecord) {
	// Direct sources
	if sourcesVal, ok := event.Properties[PropertySources]; ok {
		sourceIDs := extractStringList(sourcesVal)
		for _, sourceID := range sourceIDs {
			if sourceXRef := expCtx.SourceXRefMap[sourceID]; sourceXRef != "" {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagSour,
					Value: sourceXRef,
				})
			}
		}
	}

	// Citations (resolve to source references)
	if citationsVal, ok := event.Properties[PropertyCitations]; ok {
		citationIDs := extractStringList(citationsVal)
		for _, citationID := range citationIDs {
			citation, exists := expCtx.GLX.Citations[citationID]
			if !exists {
				continue
			}

			if sourceXRef := expCtx.SourceXRefMap[citation.SourceID]; sourceXRef != "" {
				record.SubRecords = append(record.SubRecords, exportCitationAsSOUR(citation, sourceXRef, expCtx))
			}
		}
	}
}

// exportCitationAsSOUR creates a GEDCOM SOUR sub-record from a GLX Citation,
// including PAGE, NOTE, DATA (DATE/TEXT), and OBJE sub-records.
func exportCitationAsSOUR(citation *Citation, sourceXRef string, expCtx *ExportContext) *GEDCOMRecord {
	sourRecord := &GEDCOMRecord{
		Tag:        GedcomTagSour,
		Value:      sourceXRef,
		SubRecords: []*GEDCOMRecord{},
	}

	// PAGE (locator)
	if locator, ok := getStringProperty(citation.Properties, "locator"); ok {
		sourRecord.SubRecords = append(sourRecord.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagPage,
			Value: locator,
		})
	}

	// DATA sub-record (date and text)
	var dataSubRecords []*GEDCOMRecord
	if dateStr, ok := getStringProperty(citation.Properties, "date"); ok {
		dataSubRecords = append(dataSubRecords, &GEDCOMRecord{
			Tag:   GedcomTagDate,
			Value: formatGEDCOMDate(DateString(dateStr)),
		})
	}
	if text, ok := getStringProperty(citation.Properties, "description"); ok {
		dataSubRecords = append(dataSubRecords, &GEDCOMRecord{
			Tag:   GedcomTagText,
			Value: text,
		})
	}
	if len(dataSubRecords) > 0 {
		sourRecord.SubRecords = append(sourRecord.SubRecords, &GEDCOMRecord{
			Tag:        GedcomTagData,
			SubRecords: dataSubRecords,
		})
	}

	// NOTE
	if citation.Notes != "" {
		sourRecord.SubRecords = append(sourRecord.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: citation.Notes,
		})
	}

	// OBJE (media references)
	for _, mediaID := range citation.Media {
		if mediaXRef := expCtx.MediaXRefMap[mediaID]; mediaXRef != "" {
			sourRecord.SubRecords = append(sourRecord.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagObje,
				Value: mediaXRef,
			})
		}
	}

	return sourRecord
}

// extractStringList converts a property value to a list of strings.
// Handles both []string and []any types.
func extractStringList(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		return []string{v}
	default:
		return nil
	}
}
