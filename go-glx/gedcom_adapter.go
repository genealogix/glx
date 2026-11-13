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
	"html"
	"regexp"
	"sort"
	"strings"
)

// GEDCOMPlatformAuto selects the adapter from the file's HEAD.SOUR record.
const GEDCOMPlatformAuto = ""

// GEDCOMPlatformGeneric is the name of the adapter that applies no
// platform-specific handling.
const GEDCOMPlatformGeneric = "generic"

// GEDCOMHeaderSource is what a GEDCOM file says about the software that
// exported it: the HEAD.SOUR value (the system ID) and its NAME and CORP
// sub-records.
type GEDCOMHeaderSource struct {
	SystemID    string
	Name        string
	Corporation string
}

// GEDCOMAdapterRewrites counts, per original tag, how many platform-specific
// structures an adapter rewrote into standard GEDCOM.
type GEDCOMAdapterRewrites map[string]int

// GEDCOMAdapter handles one platform's GEDCOM dialect.
//
// An adapter runs after parsing and before conversion. It rewrites the
// platform's custom tags in place into the standard GEDCOM structures the
// converter already maps, so the converter itself stays platform-neutral.
// Anything an adapter leaves alone is converted exactly as it would be
// without it: unknown underscore tags on persons are still preserved under
// gedcom_extensions.
type GEDCOMAdapter interface {
	// Name is the stable key accepted by ImportOptions.Platform.
	Name() string
	// Label is the human-readable platform name used in logs.
	Label() string
	// Detect reports whether the header identifies this platform.
	Detect(head GEDCOMHeaderSource) bool
	// TransformRecord rewrites one top-level record in place, counting each
	// rewrite in rewrites.
	TransformRecord(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites)
}

// gedcomAdapters lists the platform adapters in detection order. Family Tree
// Maker precedes Ancestry because its header can name Ancestry as CORP.
var gedcomAdapters = []GEDCOMAdapter{
	ftmAdapter{},
	ancestryAdapter{},
	uidAdapter{name: "rootsmagic", label: "RootsMagic", match: "rootsmagic"},
	uidAdapter{name: "legacy", label: "Legacy Family Tree", match: "legacy"},
	uidAdapter{name: "myheritage", label: "MyHeritage", match: "myheritage", stripHTML: true},
	uidAdapter{name: "findmypast", label: "Findmypast", match: "findmypast"},
}

// GEDCOMPlatforms returns every selectable adapter, sorted by name, ending
// with the generic adapter.
func GEDCOMPlatforms() []GEDCOMAdapter {
	platforms := make([]GEDCOMAdapter, len(gedcomAdapters), len(gedcomAdapters)+1)
	copy(platforms, gedcomAdapters)
	sort.Slice(platforms, func(i, j int) bool { return platforms[i].Name() < platforms[j].Name() })

	return append(platforms, genericAdapter{})
}

// LookupGEDCOMPlatform returns the adapter with the given name
// (case-insensitive).
//
//nolint:ireturn // adapters are polymorphic; callers need the platform behind the name
func LookupGEDCOMPlatform(name string) (GEDCOMAdapter, error) {
	for _, adapter := range GEDCOMPlatforms() {
		if strings.EqualFold(adapter.Name(), name) {
			return adapter, nil
		}
	}

	names := make([]string, 0, len(gedcomAdapters)+1)
	for _, adapter := range GEDCOMPlatforms() {
		names = append(names, adapter.Name())
	}

	return nil, fmt.Errorf("%w: %q (available: %s)", ErrUnknownGEDCOMPlatform, name, strings.Join(names, ", "))
}

// DetectGEDCOMPlatform returns the adapter whose platform the header names,
// or the generic adapter when none does.
//
//nolint:ireturn // adapters are polymorphic; callers need the platform behind the name
func DetectGEDCOMPlatform(head GEDCOMHeaderSource) GEDCOMAdapter {
	for _, adapter := range gedcomAdapters {
		if adapter.Detect(head) {
			return adapter
		}
	}

	return genericAdapter{}
}

// gedcomHeaderSource extracts HEAD.SOUR from parsed records.
func gedcomHeaderSource(records []*GEDCOMRecord) GEDCOMHeaderSource {
	var head GEDCOMHeaderSource
	for _, record := range records {
		if record.Tag != GedcomTagHead {
			continue
		}
		for _, sub := range record.SubRecords {
			if sub.Tag != GedcomTagSour {
				continue
			}
			head.SystemID = sub.Value
			for _, sourSub := range sub.SubRecords {
				switch sourSub.Tag {
				case GedcomTagName:
					head.Name = sourSub.Value
				case GedcomTagCorp:
					head.Corporation = sourSub.Value
				}
			}
		}

		break
	}

	return head
}

// applyGEDCOMAdapter selects an adapter (forced by platform, or detected from
// HEAD.SOUR) and runs it over every top-level record except HEAD.
//
//nolint:ireturn // adapters are polymorphic; callers need the platform behind the name
func applyGEDCOMAdapter(records []*GEDCOMRecord, platform string, logger *ImportLogger) (GEDCOMAdapter, GEDCOMAdapterRewrites, error) {
	var adapter GEDCOMAdapter
	if platform == GEDCOMPlatformAuto {
		adapter = DetectGEDCOMPlatform(gedcomHeaderSource(records))
		if adapter.Name() != GEDCOMPlatformGeneric {
			logger.LogInfof("Detected %s export — applying %s-specific handling", adapter.Label(), adapter.Label())
		}
	} else {
		var err error
		adapter, err = LookupGEDCOMPlatform(platform)
		if err != nil {
			return nil, nil, err
		}
		logger.LogInfof("Using %s adapter (forced)", adapter.Label())
	}

	rewrites := GEDCOMAdapterRewrites{}
	for _, record := range records {
		if record.Tag == GedcomTagHead {
			continue
		}
		adapter.TransformRecord(record, rewrites)
	}

	for _, tag := range sortedRewriteTags(rewrites) {
		logger.LogInfof("%s adapter: rewrote %d %s", adapter.Label(), rewrites[tag], tag)
	}

	return adapter, rewrites, nil
}

func sortedRewriteTags(rewrites GEDCOMAdapterRewrites) []string {
	tags := make([]string, 0, len(rewrites))
	for tag := range rewrites {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return tags
}

// headerMentions reports whether any HEAD.SOUR field contains needle,
// case-insensitively.
func headerMentions(needle string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}

	return false
}

// genericAdapter applies no platform-specific handling.
type genericAdapter struct{}

func (genericAdapter) Name() string                                         { return GEDCOMPlatformGeneric }
func (genericAdapter) Label() string                                        { return "Generic" }
func (genericAdapter) Detect(GEDCOMHeaderSource) bool                       { return false }
func (genericAdapter) TransformRecord(*GEDCOMRecord, GEDCOMAdapterRewrites) {}

// uidAdapter handles the platforms whose mapped dialect features are the
// _UID universal identifier and, optionally, HTML markup in notes
// (RootsMagic, Legacy, MyHeritage, Findmypast).
type uidAdapter struct {
	name      string
	label     string
	match     string
	stripHTML bool
}

func (a uidAdapter) Name() string  { return a.name }
func (a uidAdapter) Label() string { return a.label }

func (a uidAdapter) Detect(head GEDCOMHeaderSource) bool {
	return headerMentions(a.match, head.SystemID, head.Name)
}

func (a uidAdapter) TransformRecord(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	rewriteUIDs(record, rewrites)
	if a.stripHTML {
		walkGEDCOMRecord(record, func(r *GEDCOMRecord) { stripNoteRecordHTML(r, rewrites) })
	}
}

// GEDCOMUIDTypeURI is the EXID TYPE written for a vendor _UID: the GEDCOM 7
// UID structure that _UID was standardized as.
const GEDCOMUIDTypeURI = "https://gedcom.io/terms/v7/UID"

// rewriteUIDs turns level-1 _UID tags into EXID on the record types whose
// converters keep external IDs (INDI, SOUR, REPO). On any other record _UID
// is left as is, since an EXID there would be dropped.
func rewriteUIDs(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	switch record.Tag {
	case GedcomTagIndi, GedcomTagSour, GedcomTagRepo:
	default:
		return
	}

	for _, sub := range record.SubRecords {
		if sub.Tag != "_UID" || sub.Value == "" {
			continue
		}
		sub.Tag = GedcomTagExid
		sub.SubRecords = append(sub.SubRecords, &GEDCOMRecord{Tag: GedcomTagType, Value: GEDCOMUIDTypeURI, Line: sub.Line})
		rewrites["_UID"]++
	}
}

// htmlTagPattern matches the formatting tags that platforms embed in note
// text. It is deliberately narrow so that a literal "<" in a note (e.g.
// "<unknown>") is never mistaken for markup.
var htmlTagPattern = regexp.MustCompile(`(?i)</?(?:p|br|b|i|u|em|strong|div|span|font|a|ul|ol|li|sup|sub)(?:\s[^<>]*)?\s*/?>`)

// htmlBreakPattern matches the tags that end a line of text.
var htmlBreakPattern = regexp.MustCompile(`(?i)<br\s*/?>|</(?:p|div|li)\s*>`)

var blankLinesPattern = regexp.MustCompile(`\n{3,}`)

// stripNoteHTML removes formatting markup from note text, keeping its text
// and line breaks. Text without recognized markup is returned unchanged, so
// entities and angle brackets in plain notes are left alone.
func stripNoteHTML(text string) (string, bool) {
	if !htmlTagPattern.MatchString(text) {
		return text, false
	}

	text = htmlBreakPattern.ReplaceAllString(text, "\n")
	text = htmlTagPattern.ReplaceAllString(text, "")
	text = html.UnescapeString(text)
	text = strings.ReplaceAll(text, " ", " ")

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = blankLinesPattern.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")

	return strings.TrimSpace(text), true
}

// stripNoteRecordHTML strips markup from an inline NOTE (top-level or
// nested), folding its CONT/CONC lines into the value first so a tag split
// across lines is still recognized. Pointer notes are left alone.
func stripNoteRecordHTML(record *GEDCOMRecord, rewrites GEDCOMAdapterRewrites) {
	if record.Tag != GedcomTagNote || isGEDCOMPointer(record.Value) {
		return
	}

	stripped, changed := stripNoteHTML(extractTextWithContinuation(record))
	if !changed {
		return
	}

	kept := record.SubRecords[:0]
	for _, sub := range record.SubRecords {
		if sub.Tag != GedcomTagCont && sub.Tag != GedcomTagConc {
			kept = append(kept, sub)
		}
	}
	record.Value = stripped
	record.SubRecords = kept
	rewrites["NOTE (HTML)"]++
}

// walkGEDCOMRecord calls fn on record and every record beneath it.
func walkGEDCOMRecord(record *GEDCOMRecord, fn func(*GEDCOMRecord)) {
	fn(record)
	for _, sub := range record.SubRecords {
		walkGEDCOMRecord(sub, fn)
	}
}
