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
	"sort"
	"strings"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// unspecifiedValue labels reports whose assertion carries no value (an
// existential or placeholder assertion) so they still group and count.
const unspecifiedValue = "(unspecified)"

// noCitationLabel marks a report backed by neither a citation nor a source.
const noCitationLabel = "(no citation)"

// EvidenceItem is a single supporting report — one citation, one direct
// source, or a bare assertion — backing a particular value.
type EvidenceItem struct {
	CitationID string `json:"citation_id,omitempty"`
	Source     string `json:"source,omitempty"`
	Confidence string `json:"confidence,omitempty"`
}

// EvidenceGroup holds every report that agrees on a single value, with the
// best (highest-ranked) confidence seen among them.
type EvidenceGroup struct {
	Date           DateString     `json:"date,omitempty"`
	RawValue       string         `json:"raw_value,omitempty"`
	BestEvidence   string         `json:"best_evidence,omitempty"`
	Value          string         `json:"value"`
	Reports        int            `json:"reports"`
	BestConfidence string         `json:"best_confidence,omitempty"`
	Items          []EvidenceItem `json:"items"`
}

// EvidenceConflict is a shared-engine verdict with the original claims.
type EvidenceConflict struct {
	Verdict Verdict     `json:"verdict"`
	Values  []FactValue `json:"values"`
}

// EvidenceReport is a deterministic evidence breakdown. Temporal groups are
// ordered by date; fixed-property groups are ranked by support.
type EvidenceReport struct {
	Temporal  bool               `json:"temporal,omitempty"`
	Undated   []EvidenceGroup    `json:"undated,omitempty"`
	Conflicts []EvidenceConflict `json:"conflicts,omitempty"`
	// Subject is the entity ID the evidence is about; SubjectType is its
	// singular entity type ("person", "event", "place", "relationship") and
	// SubjectName its display label.
	Subject     string `json:"subject"`
	SubjectType string `json:"subject_type"`
	SubjectName string `json:"subject_name"`
	// Person and PersonName repeat Subject and SubjectName for person subjects,
	// keeping `--format json` consumers written before event/place/relationship
	// subjects were accepted working unchanged. They are empty for every other
	// subject type; new consumers should read Subject/SubjectName.
	Person       string          `json:"person,omitempty"`
	PersonName   string          `json:"person_name,omitempty"`
	Property     string          `json:"property"`
	TotalReports int             `json:"total_reports"`
	Groups       []EvidenceGroup `json:"groups"`
	// BestEvidence is the winning value, or "" when the top two groups tie on
	// both report count and confidence — an unresolved conflict surfaced to the
	// researcher rather than papered over.
	BestEvidence string `json:"best_evidence,omitempty"`
}

// subjectLabel returns the display label for an evidence subject: a person's
// name, an event's title (or a title generated from its type), a place's name,
// or a relationship's type. It falls back to the entity ID whenever the entity
// is missing or carries no label of its own.
func subjectLabel(archive *GLXFile, subject EntityRef) string {
	switch subject.Type() {
	case EntityTypePersons:
		return personName(archive, subject.Person)
	case EntityTypeEvents:
		if ev, ok := archive.Events[subject.Event]; ok && ev != nil {
			if ev.Title != "" {
				return ev.Title
			}
			if title := GenerateEventTitle(ev.Type, nil); title != "" {
				return title
			}
		}
	case EntityTypePlaces:
		return resolvePlaceName(subject.Place, archive)
	case EntityTypeRelationships:
		if rel, ok := archive.Relationships[subject.Relationship]; ok && rel != nil && rel.Type != "" {
			return rel.Type
		}
	}

	return subject.ID()
}

// noteConfidence raises the group's best confidence to c when c outranks the
// current best (lower confidenceRank means stronger).
func (g *EvidenceGroup) noteConfidence(c string) {
	if g.BestConfidence == "" || researchConfidenceRank(c) < researchConfidenceRank(g.BestConfidence) {
		g.BestConfidence = c
	}
}

// collectEvidence gathers every assertion for the given subject+property,
// groups the supporting reports by asserted value, and ranks the values by
// report count and confidence. Output is deterministic.
func collectEvidence(archive *GLXFile, subject EntityRef, property string, options ...ComparisonOptions) EvidenceReport {
	assertions, canonicalProperty := matchingAssertions(archive, subject, property)
	report := EvidenceReport{
		Subject:     subject.ID(),
		SubjectType: subject.Type().Singular(),
		SubjectName: subjectLabel(archive, subject),
		// canonicalProperty is the property key as stored in the matched
		// assertions (e.g. "residence" for a "RESIDENCE" query), so JSON and text
		// output reflect the data rather than the query's casing.
		Property: canonicalProperty,
	}
	if subject.Person != "" {
		report.Person = report.Subject
		report.PersonName = report.SubjectName
	}

	def := ConflictProperty(archive, subject, canonicalProperty)
	report.Temporal = IsTemporalProperty(def)
	report.Conflicts = evidenceConflicts(assertions, def, archive, comparisonOptions(options))

	groups := make(map[string]*EvidenceGroup)
	// citationIdx maps value -> citationID -> index of that citation's report in
	// the group's Items. The same record cited for the same value by more than
	// one assertion is a single report — but its confidence is upgraded to the
	// strongest seen across those assertions, rather than whichever was seen first.
	citationIdx := make(map[string]map[string]int)

	for _, a := range assertions {
		if strings.EqualFold(a.Status, "disproven") {
			continue
		}
		// Resolve against the matched assertion's own property key (a.Property),
		// not the raw query string: under the case-insensitive fallback the query
		// casing can differ, and placeRefProperties / PersonProperties lookups are
		// case-sensitive, so resolving by the query would miss place/person/event
		// references.
		value := factDisplay(a.Value, subject, a.Property, archive)
		if value == "" {
			value = unspecifiedValue
		}

		key := evidenceGroupKey(a, groups, def, archive, comparisonOptions(options))
		g, ok := groups[key]
		if !ok {
			g = &EvidenceGroup{Value: value, RawValue: a.Value}
			if report.Temporal {
				g.Date = a.Date
			}
			groups[key] = g
			citationIdx[key] = make(map[string]int)
		}

		for _, item := range assertionItems(a, archive) {
			if item.CitationID != "" {
				if idx, seen := citationIdx[key][item.CitationID]; seen {
					// Same record cited again: keep one report, but raise its
					// confidence (and the group's) when this assertion is stronger.
					if researchConfidenceRank(item.Confidence) < researchConfidenceRank(g.Items[idx].Confidence) {
						g.Items[idx].Confidence = item.Confidence
					}
					g.noteConfidence(item.Confidence)

					continue
				}
				citationIdx[key][item.CitationID] = len(g.Items)
			}

			g.Items = append(g.Items, item)
			g.Reports++
			report.TotalReports++
			g.noteConfidence(item.Confidence)
		}
	}

	// Group order is fully determined by sortEvidenceGroups: values are unique,
	// so its Value tiebreaker makes the ordering total and map iteration order
	// cannot affect the result.
	report.Groups = make([]EvidenceGroup, 0, len(groups))
	for _, g := range groups {
		sortEvidenceItems(g.Items)
		report.Groups = append(report.Groups, *g)
	}
	sortEvidenceGroups(report.Groups)
	report.BestEvidence = bestEvidence(report.Groups)
	if report.Temporal {
		arrangeEvidenceHistory(&report)
	}

	return report
}

// matchingAssertions returns the assertions whose subject is subject and whose
// property matches, plus the canonical property key actually stored on those
// assertions. Exact matches win; only when none exist does it fall back to
// case-insensitive matches, so "born_at" never silently picks up "Born_At" when
// an exact "born_at" is present. On a case-insensitive match the stored key
// (e.g. "residence" for a "RESIDENCE" query) is returned so callers report the
// property as stored rather than as typed; with no matches the query is echoed
// back. Iteration order is deterministic.
func matchingAssertions(archive *GLXFile, subject EntityRef, property string) (matched []*Assertion, canonical string) {
	var exact, insensitive []*Assertion

	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		// Compare by type and ID rather than by struct equality: an assertion
		// whose subject somehow carries more than one field still matches on the
		// one its Type() reports, instead of matching nothing at all.
		if a == nil || a.Subject.Type() != subject.Type() || a.Subject.ID() != subject.ID() {
			continue
		}

		switch {
		case a.Property == property:
			exact = append(exact, a)
		case strings.EqualFold(a.Property, property):
			insensitive = append(insensitive, a)
		}
	}

	if len(exact) > 0 {
		return exact, property
	}
	if len(insensitive) > 0 {
		return insensitive, insensitive[0].Property
	}

	return nil, property
}

// assertionItems expands an assertion into its supporting reports: one per
// citation, else one per direct source, else a single bare report. Each
// inherits the assertion's confidence.
func assertionItems(a *Assertion, archive *GLXFile) []EvidenceItem {
	var items []EvidenceItem

	for _, citID := range a.Citations {
		items = append(items, EvidenceItem{
			CitationID: citID,
			Source:     citationSourceLabel(citID, archive),
			Confidence: a.Confidence,
		})
	}

	if len(items) == 0 {
		for _, srcID := range a.Sources {
			items = append(items, EvidenceItem{
				Source:     sourceLabel(srcID, archive),
				Confidence: a.Confidence,
			})
		}
	}

	if len(items) == 0 {
		items = append(items, EvidenceItem{
			Source:     noCitationLabel,
			Confidence: a.Confidence,
		})
	}

	return items
}

// citationSourceLabel resolves a citation to its source title, falling back to
// the citation ID when the citation, its source, or the source's title is
// missing. Note: this deliberately does NOT delegate to sourceLabel, which
// falls back to the source ID — appropriate where no citation column exists,
// but here the citation ID already occupies its own column and surfacing a
// raw source ID alongside would duplicate identifier noise without adding
// information.
func citationSourceLabel(citID string, archive *GLXFile) string {
	cit, ok := archive.Citations[citID]
	if !ok || cit == nil {
		return citID
	}
	if cit.SourceID != "" {
		if src, ok := archive.Sources[cit.SourceID]; ok && src != nil && src.Title != "" {
			return src.Title
		}
	}

	return citID
}

// sourceLabel returns a source's title, or the source ID when the source is
// missing or untitled. Empty input yields empty output.
func sourceLabel(sourceID string, archive *GLXFile) string {
	if sourceID == "" {
		return ""
	}
	if src, ok := archive.Sources[sourceID]; ok && src != nil && src.Title != "" {
		return src.Title
	}

	return sourceID
}

// resolveAssertionValue turns a raw assertion value into its display form.
// Place-reference properties and any property whose definition declares a
// persons/places/events reference resolve to the referenced entity's name;
// everything else (free text, vocabulary values) passes through unchanged.
// The property definition is looked up in the vocabulary belonging to the
// subject's own entity type, so an event property is never resolved against
// the person vocabulary (or the other way round).
//
// Assertion.Value is always a scalar string, so — unlike person properties —
// there is no temporal-shape map to unwrap here.
func resolveAssertionValue(value, property string, subject EntityRef, archive *GLXFile) string {
	if value == "" {
		return value
	}

	// Resolve standard place-reference properties.
	if placeRefProperties[property] {
		return resolvePlaceName(value, archive)
	}

	// `place` on an event subject is the event's own structural place field
	// (Event.PlaceID), not a vocabulary property, so no definition declares it
	// a reference. `glx migrate` maps it the same way.
	if subject.Event != "" && property == eventFieldPlace {
		return resolvePlaceName(value, archive)
	}

	if def, ok := subjectPropertyDefinitions(archive, subject)[property]; ok && def != nil {
		// PropertyDefinition.ReferenceType is an untyped string from YAML;
		// bridge to EntityType via .String() to match the convention used by
		// isPlaceReferenceProperty in summary_runner.go.
		switch def.ReferenceType {
		case EntityTypePlaces.String():
			return resolvePlaceName(value, archive)
		case EntityTypePersons.String():
			return personName(archive, value)
		case EntityTypeEvents.String():
			if ev, ok := archive.Events[value]; ok && ev != nil && ev.Title != "" {
				return ev.Title
			}
		}
	}

	return value
}

// subjectPropertyDefinitions returns the vocabulary property definitions that
// apply to a subject's entity type. A nil map is a safe read in Go, so an
// unknown type (or a vocabulary that was never loaded) simply resolves nothing
// and values are shown verbatim.
func subjectPropertyDefinitions(archive *GLXFile, subject EntityRef) map[string]*PropertyDefinition {
	switch subject.Type() {
	case EntityTypePersons:
		return archive.PersonProperties
	case EntityTypeEvents:
		return archive.EventProperties
	case EntityTypePlaces:
		return archive.PlaceProperties
	case EntityTypeRelationships:
		return archive.RelationshipProperties
	default:
		return nil
	}
}

// sortEvidenceItems orders reports within a group deterministically:
// citation-backed reports first (by citation ID), then source-only reports.
func sortEvidenceItems(items []EvidenceItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if (a.CitationID == "") != (b.CitationID == "") {
			return a.CitationID != "" // citation-backed reports first
		}
		if a.CitationID != b.CitationID {
			return a.CitationID < b.CitationID
		}

		return a.Source < b.Source
	})
}

// sortEvidenceGroups ranks values by report count (desc), then best confidence
// (highest first), then display value, raw identity and date (asc). Distinct
// references can share a display name, so that label alone is not a total order.
func sortEvidenceGroups(groups []EvidenceGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Reports != groups[j].Reports {
			return groups[i].Reports > groups[j].Reports
		}

		ri, rj := researchConfidenceRank(groups[i].BestConfidence), researchConfidenceRank(groups[j].BestConfidence)
		if ri != rj {
			return ri < rj
		}

		if groups[i].Value != groups[j].Value {
			return groups[i].Value < groups[j].Value
		}
		if groups[i].RawValue != groups[j].RawValue {
			return groups[i].RawValue < groups[j].RawValue
		}

		return groups[i].Date < groups[j].Date
	})
}

// bestEvidence returns the leading value, or "" when the top two groups tie on
// both report count and confidence (an unresolved conflict).
func bestEvidence(groups []EvidenceGroup) string {
	if len(groups) == 0 {
		return ""
	}
	if len(groups) >= 2 {
		top, runnerUp := groups[0], groups[1]
		if top.Reports == runnerUp.Reports &&
			researchConfidenceRank(top.BestConfidence) == researchConfidenceRank(runnerUp.BestConfidence) {
			return ""
		}
	}

	return groups[0].Value
}

func arrangeEvidenceHistory(report *EvidenceReport) {
	report.BestEvidence = ""
	dated := make([]EvidenceGroup, 0, len(report.Groups))
	for _, g := range report.Groups {
		d, _ := g.Date.Parse()
		if !d.Timing().Known {
			report.Undated = append(report.Undated, g)
		} else {
			dated = append(dated, g)
		}
	}
	sort.SliceStable(dated, func(i, j int) bool {
		a, _ := dated[i].Date.Parse()
		b, _ := dated[j].Date.Parse()
		x, y := a.Timing().Outer.Start, b.Timing().Outer.Start
		if x != y {
			return x < y
		}
		if dated[i].Date != dated[j].Date {
			return dated[i].Date < dated[j].Date
		}

		return dated[i].RawValue < dated[j].RawValue
	})
	for i := range dated {
		a, _ := dated[i].Date.Parse()
		candidates := []EvidenceGroup{dated[i]}
		for j := range dated {
			if i == j {
				continue
			}
			b, _ := dated[j].Date.Parse()
			if glxdate.CompareTiming(a, b) != glxdate.NoOverlap {
				candidates = append(candidates, dated[j])
			}
		}
		if len(candidates) > 1 && simultaneousEvidence(candidates) {
			sortEvidenceGroups(candidates)
			dated[i].BestEvidence = bestEvidence(candidates)
		}
	}
	report.Groups = dated
}

func evidenceConflicts(assertions []*Assertion, def *PropertyDefinition, archive *GLXFile, opts ComparisonOptions) []EvidenceConflict {
	var conflicts []EvidenceConflict
	facts := make([]FactValue, len(assertions))
	for i, a := range assertions {
		facts[i] = AssertionFact(a)
	}
	for _, c := range compareFacts(facts, def, archive.Places, opts) {
		if c.IsConflict() {
			conflicts = append(conflicts, EvidenceConflict{Verdict: c.Verdict, Values: []FactValue{facts[c.Left], facts[c.Right]}})
		}
	}

	return conflicts
}

func evidenceGroupKey(a *Assertion, groups map[string]*EvidenceGroup, def *PropertyDefinition, archive *GLXFile, opts ComparisonOptions) string {
	key := a.Value
	if IsTemporalProperty(def) {
		key += "\x00" + string(a.Date)
	}
	for existing, group := range groups {
		if IsTemporalProperty(def) && group.Date != a.Date {
			continue
		}
		values := []FactValue{{Value: group.RawValue}, {Value: a.Value}}
		if compareFacts(values, def, archive.Places, opts)[0].Verdict == VerdictAgree {
			return existing
		}
	}

	return key
}

// A long period must not join two disjoint histories into one ranked contest.
func simultaneousEvidence(groups []EvidenceGroup) bool {
	for i := range groups {
		a, _ := groups[i].Date.Parse()
		for j := i + 1; j < len(groups); j++ {
			b, _ := groups[j].Date.Parse()
			if glxdate.CompareTiming(a, b) == glxdate.NoOverlap {
				return false
			}
		}
	}

	return true
}
