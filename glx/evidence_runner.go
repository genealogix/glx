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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
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
	Value          string         `json:"value"`
	Reports        int            `json:"reports"`
	BestConfidence string         `json:"best_confidence,omitempty"`
	Items          []EvidenceItem `json:"items"`
}

// EvidenceReport is the full evidence breakdown for one subject+property,
// ranked so the most-supported value comes first.
type EvidenceReport struct {
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

// showEvidence loads an archive, resolves the subject, and prints the grouped
// evidence for the requested property in text or JSON form.
func showEvidence(io *IOStreams, archivePath, subjectQuery, property, format string) error {
	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	subject, err := findEvidenceSubject(archive, subjectQuery)
	if err != nil {
		return err
	}

	report := collectEvidence(archive, subject, property)

	switch format {
	case "", "text":
		printEvidenceText(io, &report)

		return nil
	case "json":
		return printEvidenceJSON(io, &report)
	default:
		return fmt.Errorf("%w: %q", ErrEvidenceUnknownFormat, format)
	}
}

// loadArchiveForEvidence loads an archive from a path (directory or single
// file), mirroring loadArchiveForSummary. Duplicate-ID warnings go to ErrOut.
func loadArchiveForEvidence(io *IOStreams, path string) (*glxlib.GLXFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access path: %w", err)
	}

	if info.IsDir() {
		archive, duplicates, err := LoadArchiveCached(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		for _, d := range duplicates {
			io.Errorf("Warning: %s\n", d)
		}

		return archive, nil
	}

	archive, err := readSingleFileArchive(path, false)
	if err != nil {
		return nil, err
	}

	// Single-file archives don't merge standard vocabularies during load (unlike
	// directory archives via LoadArchiveCached), so populate them here.
	// evidence is read-only and mergeStandardVocabularies only fills empty
	// vocabulary maps, so any archive-defined vocabulary is preserved. This
	// lets reference-type properties (e.g. residence) resolve to place/person/
	// event names regardless of whether the archive is a directory or a single
	// file — matching loadArchiveForSummary's behavior.
	if err := mergeStandardVocabularies(archive); err != nil {
		return nil, fmt.Errorf("failed to load standard vocabularies: %w", err)
	}

	return archive, nil
}

// findEvidenceSubject resolves the subject argument of `glx evidence` to any
// assertable subject, matching what `glx add assertion` already accepts via
// --subject-person/--subject-event/--subject-place/--subject-relationship. In
// an evidence-first archive the two properties most likely to have conflicting
// answers — date and place — are asserted on the event, not on the person, so
// a person-only lookup cannot be pointed at an archive's contested facts.
//
// Exact entity IDs are tried first, across all four subject types, and only
// then the person name search. An exact ID is the more specific match, so
// checking it first can never shadow a name the caller meant; doing it the
// other way round would let a substring name match swallow an ID that names a
// different entity outright. Entity IDs are unique archive-wide, but a
// hand-edited archive can break that, so a query matching entities of more
// than one type is reported rather than silently resolved by map order.
func findEvidenceSubject(archive *glxlib.GLXFile, query string) (glxlib.EntityRef, error) {
	var matches []glxlib.EntityRef

	if p, ok := archive.Persons[query]; ok && p != nil {
		matches = append(matches, glxlib.EntityRef{Person: query})
	}
	if ev, ok := archive.Events[query]; ok && ev != nil {
		matches = append(matches, glxlib.EntityRef{Event: query})
	}
	if pl, ok := archive.Places[query]; ok && pl != nil {
		matches = append(matches, glxlib.EntityRef{Place: query})
	}
	if rel, ok := archive.Relationships[query]; ok && rel != nil {
		matches = append(matches, glxlib.EntityRef{Relationship: query})
	}

	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		// Fall through to the person name search below.
	default:
		var lines []string
		for _, m := range matches {
			lines = append(lines, fmt.Sprintf("  %s (%s)", m.ID(), m.Type().Singular()))
		}

		return glxlib.EntityRef{}, fmt.Errorf("%w — %q names:\n%s\nRepair the archive so entity IDs are unique",
			ErrEvidenceSubjectAmbiguous, query, strings.Join(lines, "\n"))
	}

	// No exact ID: fall back to the person name search, which also produces the
	// multi-person disambiguation listing. Its ID lookup is a no-op here.
	personID, _, err := findPersonByQuery(archive, query)
	if err == nil {
		return glxlib.EntityRef{Person: personID}, nil
	}
	if !errors.Is(err, ErrNoPersonMatch) {
		// An ambiguous name: keep findPersonByQuery's listing of the candidates.
		return glxlib.EntityRef{}, err
	}

	return glxlib.EntityRef{}, fmt.Errorf(
		"%w %q (evidence takes a person ID or name, or the exact ID of an event, place, or relationship)",
		ErrEvidenceNoSubject, query,
	)
}

// subjectLabel returns the display label for an evidence subject: a person's
// name, an event's title (or a title generated from its type), a place's name,
// or a relationship's type. It falls back to the entity ID whenever the entity
// is missing or carries no label of its own.
func subjectLabel(archive *glxlib.GLXFile, subject glxlib.EntityRef) string {
	switch subject.Type() {
	case glxlib.EntityTypePersons:
		return personName(archive, subject.Person)
	case glxlib.EntityTypeEvents:
		if ev, ok := archive.Events[subject.Event]; ok && ev != nil {
			if ev.Title != "" {
				return ev.Title
			}
			if title := glxlib.GenerateEventTitle(ev.Type, nil); title != "" {
				return title
			}
		}
	case glxlib.EntityTypePlaces:
		return resolvePlaceName(subject.Place, archive)
	case glxlib.EntityTypeRelationships:
		if rel, ok := archive.Relationships[subject.Relationship]; ok && rel != nil && rel.Type != "" {
			return rel.Type
		}
	}

	return subject.ID()
}

// noteConfidence raises the group's best confidence to c when c outranks the
// current best (lower confidenceRank means stronger).
func (g *EvidenceGroup) noteConfidence(c string) {
	if g.BestConfidence == "" || confidenceRank(c) < confidenceRank(g.BestConfidence) {
		g.BestConfidence = c
	}
}

// collectEvidence gathers every assertion for the given subject+property,
// groups the supporting reports by asserted value, and ranks the values by
// report count and confidence. Output is deterministic.
func collectEvidence(archive *glxlib.GLXFile, subject glxlib.EntityRef, property string) EvidenceReport {
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

	groups := make(map[string]*EvidenceGroup)
	// citationIdx maps value -> citationID -> index of that citation's report in
	// the group's Items. The same record cited for the same value by more than
	// one assertion is a single report — but its confidence is upgraded to the
	// strongest seen across those assertions, rather than whichever was seen first.
	citationIdx := make(map[string]map[string]int)

	for _, a := range assertions {
		// Resolve against the matched assertion's own property key (a.Property),
		// not the raw query string: under the case-insensitive fallback the query
		// casing can differ, and placeRefProperties / PersonProperties lookups are
		// case-sensitive, so resolving by the query would miss place/person/event
		// references.
		value := resolveAssertionValue(a.Value, a.Property, subject, archive)
		if value == "" {
			value = unspecifiedValue
		}

		g, ok := groups[value]
		if !ok {
			g = &EvidenceGroup{Value: value}
			groups[value] = g
			citationIdx[value] = make(map[string]int)
		}

		for _, item := range assertionItems(a, archive) {
			if item.CitationID != "" {
				if idx, seen := citationIdx[value][item.CitationID]; seen {
					// Same record cited again: keep one report, but raise its
					// confidence (and the group's) when this assertion is stronger.
					if confidenceRank(item.Confidence) < confidenceRank(g.Items[idx].Confidence) {
						g.Items[idx].Confidence = item.Confidence
					}
					g.noteConfidence(item.Confidence)

					continue
				}
				citationIdx[value][item.CitationID] = len(g.Items)
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
func matchingAssertions(archive *glxlib.GLXFile, subject glxlib.EntityRef, property string) (matched []*glxlib.Assertion, canonical string) {
	var exact, insensitive []*glxlib.Assertion

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
func assertionItems(a *glxlib.Assertion, archive *glxlib.GLXFile) []EvidenceItem {
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
func citationSourceLabel(citID string, archive *glxlib.GLXFile) string {
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
func sourceLabel(sourceID string, archive *glxlib.GLXFile) string {
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
func resolveAssertionValue(value, property string, subject glxlib.EntityRef, archive *glxlib.GLXFile) string {
	if value == "" {
		return value
	}

	// Fast path matching analyze's resolveConflictValue for place properties.
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
		case glxlib.EntityTypePlaces.String():
			return resolvePlaceName(value, archive)
		case glxlib.EntityTypePersons.String():
			return personName(archive, value)
		case glxlib.EntityTypeEvents.String():
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
func subjectPropertyDefinitions(archive *glxlib.GLXFile, subject glxlib.EntityRef) map[string]*glxlib.PropertyDefinition {
	switch subject.Type() {
	case glxlib.EntityTypePersons:
		return archive.PersonProperties
	case glxlib.EntityTypeEvents:
		return archive.EventProperties
	case glxlib.EntityTypePlaces:
		return archive.PlaceProperties
	case glxlib.EntityTypeRelationships:
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
// (highest first), then value (asc) for stable, predictable output.
func sortEvidenceGroups(groups []EvidenceGroup) {
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Reports != groups[j].Reports {
			return groups[i].Reports > groups[j].Reports
		}

		ri, rj := confidenceRank(groups[i].BestConfidence), confidenceRank(groups[j].BestConfidence)
		if ri != rj {
			return ri < rj
		}

		return groups[i].Value < groups[j].Value
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
			confidenceRank(top.BestConfidence) == confidenceRank(runnerUp.BestConfidence) {
			return ""
		}
	}

	return groups[0].Value
}

// printEvidenceText renders the human-readable report.
func printEvidenceText(io *IOStreams, r *EvidenceReport) {
	if len(r.Groups) == 0 {
		io.Printf("No assertions found for %s of %s (%s).\n", r.Property, r.SubjectName, r.Subject)

		return
	}

	io.Printf("Evidence for %s of %s (%s):\n", r.Property, r.SubjectName, r.Subject)
	io.Printf("%d %s across %d %s\n\n",
		r.TotalReports, pluralize(r.TotalReports, "report", "reports"),
		len(r.Groups), pluralize(len(r.Groups), "value", "values"))

	for _, g := range r.Groups {
		io.Printf("  %s — %d %s, best confidence: %s\n",
			g.Value, g.Reports, pluralize(g.Reports, "report", "reports"), displayOrDash(g.BestConfidence))
		for _, item := range g.Items {
			io.Printf("    %-32s %-30s %s\n",
				displayOrDash(item.CitationID), item.Source, displayOrDash(item.Confidence))
		}
		io.Println("")
	}

	printBestEvidence(io, r)
}

// printBestEvidence prints the closing best-evidence line (or a tie notice).
// bestEvidence guarantees the winner is Groups[0] (or "" for a tie), so there
// is no need to search for the matching group.
func printBestEvidence(io *IOStreams, r *EvidenceReport) {
	if r.BestEvidence == "" {
		io.Println("  Best evidence: inconclusive — leading values tie on report count and confidence")

		return
	}

	g := r.Groups[0]
	io.Printf("  Best evidence: %s (%d %s, %s)\n",
		g.Value, g.Reports, pluralize(g.Reports, "report", "reports"), displayOrDash(g.BestConfidence))
}

// printEvidenceJSON renders the report as indented JSON. JSON is machine-
// consumable output, so it goes to MachineOut — which, unlike Out, survives
// --quiet — keeping `glx --quiet evidence ... --format json` scriptable.
func printEvidenceJSON(io *IOStreams, r *EvidenceReport) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal evidence report: %w", err)
	}
	fmt.Fprintln(io.MachineOut, string(data))

	return nil
}

// pluralize returns singular when n == 1, otherwise plural.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}

	return plural
}
