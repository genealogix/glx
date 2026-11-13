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

// AncestryAPIDTypeURI is the EXID TYPE written for an Ancestry _APID record
// pointer.
const AncestryAPIDTypeURI = "https://www.ancestry.com/apid"

// ancestryAdapter handles Ancestry.com member-tree exports.
type ancestryAdapter struct{}

func (ancestryAdapter) Name() string  { return "ancestry" }
func (ancestryAdapter) Label() string { return "Ancestry" }

func (ancestryAdapter) Detect(head GEDCOMHeaderSource) bool {
	return headerMentions("ancestry", head.SystemID, head.Name)
}

func (ancestryAdapter) TransformRecord(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	transformAncestryRecord(record, rewrites)
}

// ftmAdapter handles Family Tree Maker exports. FTM is Ancestry's desktop
// sibling and writes the same custom tags.
type ftmAdapter struct{}

func (ftmAdapter) Name() string  { return "ftm" }
func (ftmAdapter) Label() string { return "Family Tree Maker" }

func (ftmAdapter) Detect(head GEDCOMHeaderSource) bool {
	return strings.EqualFold(strings.TrimSpace(head.SystemID), "FTM") ||
		headerMentions("family tree maker", head.SystemID, head.Name)
}

func (ftmAdapter) TransformRecord(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	transformAncestryRecord(record, rewrites)
}

// transformAncestryRecord applies the rewrites shared by Ancestry and FTM:
//
//   - INDI _MILT becomes EVEN with TYPE "Military Service", the value kept as
//     the subtype ("Military Service: US Army"), so the event keeps its date,
//     place, notes and citations as a military_service event.
//   - INDI _DCAUSE becomes CAUS on the person's death event, creating DEAT
//     when there is none.
//   - _APID on a citation or source becomes EXID, and on a citation also a
//     WWW link to the Ancestry record it points at.
//   - Inline notes have their HTML formatting stripped.
//   - _UID becomes EXID, as on the other platforms.
func transformAncestryRecord(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	if record.Tag == GedcomTagIndi {
		rewriteMilitaryService(record, rewrites)
		rewriteDeathCause(record, rewrites)
	}

	rewriteUIDs(record, rewrites)

	walkGEDCOMRecord(record, func(r *GEDCOMRecord) {
		stripNoteRecordHTML(r, rewrites)
		if r.Tag == GedcomTagSour {
			rewriteAPID(r, r == record, rewrites)
		}
	})
}

// militaryServiceLabel is the label of the military_service event type, which
// EVEN + TYPE resolves back to that type.
const militaryServiceLabel = "Military Service"

func rewriteMilitaryService(indi *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	for _, sub := range indi.SubRecords {
		if sub.Tag != "_MILT" {
			continue
		}

		var details []string
		if value := strings.TrimSpace(sub.Value); value != "" {
			details = append(details, value)
		}
		kept := make([]*GEDCOMRecord, 0, len(sub.SubRecords)+1)
		for _, child := range sub.SubRecords {
			if child.Tag == GedcomTagType {
				if value := strings.TrimSpace(child.Value); value != "" {
					details = append([]string{value}, details...)
				}

				continue
			}
			kept = append(kept, child)
		}

		typeValue := militaryServiceLabel
		if len(details) > 0 {
			typeValue += genericEventSubtypeSeparator + strings.Join(details, ", ")
		}

		sub.Tag = GedcomTagEven
		sub.Value = ""
		sub.SubRecords = append([]*GEDCOMRecord{{Tag: GedcomTagType, Value: typeValue, Line: sub.Line}}, kept...)
		rewrites["_MILT"]++
	}
}

// rewriteDeathCause moves each _DCAUSE onto the first DEAT as CAUS. A
// _DCAUSE carrying anything but continuation lines, or one the death event
// already has a cause for, is left in place and preserved as an extension.
func rewriteDeathCause(indi *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	var death *GEDCOMRecord
	for _, sub := range indi.SubRecords {
		if sub.Tag == GedcomTagDeat {
			death = sub

			break
		}
	}

	kept := make([]*GEDCOMRecord, 0, len(indi.SubRecords))
	for _, sub := range indi.SubRecords {
		if sub.Tag != "_DCAUSE" || !onlyContinuation(sub) {
			kept = append(kept, sub)

			continue
		}
		cause := strings.TrimSpace(extractTextWithContinuation(sub))
		if cause == "" || (death != nil && hasSubTag(death, GedcomTagCaus)) {
			kept = append(kept, sub)

			continue
		}
		if death == nil {
			death = &GEDCOMRecord{Tag: GedcomTagDeat, Line: sub.Line}
			kept = append(kept, death)
		}
		death.SubRecords = append(death.SubRecords, &GEDCOMRecord{Tag: GedcomTagCaus, Value: cause, Line: sub.Line})
		rewrites["_DCAUSE"]++
	}
	indi.SubRecords = kept
}

// rewriteAPID turns _APID under a SOUR into EXID. On a citation (a SOUR that
// is not itself the top-level source record) it also adds a WWW link when the
// pointer names a specific record.
func rewriteAPID(sour *GEDCOMRecord, isSourceRecord bool, rewrites GEDCOMAdapterRewrites) {
	var links []*GEDCOMRecord
	for _, sub := range sour.SubRecords {
		if sub.Tag != "_APID" || strings.TrimSpace(sub.Value) == "" {
			continue
		}
		sub.Tag = GedcomTagExid
		sub.Value = strings.TrimSpace(sub.Value)
		sub.SubRecords = append(sub.SubRecords, &GEDCOMRecord{Tag: GedcomTagType, Value: AncestryAPIDTypeURI, Line: sub.Line})
		rewrites["_APID"]++

		if url := AncestryRecordURL(sub.Value); url != "" && !isSourceRecord && !hasSubTag(sour, GedcomTagWww) {
			links = append(links, &GEDCOMRecord{Tag: GedcomTagWww, Value: url, Line: sub.Line})
		}
	}
	if len(links) > 0 {
		sour.SubRecords = append(sour.SubRecords, links[0])
	}
}

// AncestryRecordURL returns the Ancestry record page an _APID points at, or ""
// when it names no specific record. An _APID has the form
// "<site>,<database>::<record>", e.g. "1,7602::12345678"; a record of 0 points
// at the database as a whole.
func AncestryRecordURL(apid string) string {
	_, ids, found := strings.Cut(strings.TrimSpace(apid), ",")
	if !found {
		return ""
	}
	database, record, found := strings.Cut(ids, "::")
	if !found || !allDigits(database) || !allDigits(record) || strings.Trim(record, "0") == "" {
		return ""
	}

	return "https://www.ancestry.com/discoveryui-content/view/" + record + ":" + database
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for i := range len(value) {
		if !isDigit(value[i]) {
			return false
		}
	}

	return true
}

func hasSubTag(record *GEDCOMRecord, tag string) bool {
	for _, sub := range record.SubRecords {
		if sub.Tag == tag {
			return true
		}
	}

	return false
}

func onlyContinuation(record *GEDCOMRecord) bool {
	for _, sub := range record.SubRecords {
		if sub.Tag != GedcomTagCont && sub.Tag != GedcomTagConc {
			return false
		}
	}

	return true
}
