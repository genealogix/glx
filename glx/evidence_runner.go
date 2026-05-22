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

// EvidenceReport is the full evidence breakdown for one person+property,
// ranked so the most-supported value comes first.
type EvidenceReport struct {
	Person       string          `json:"person"`
	PersonName   string          `json:"person_name"`
	Property     string          `json:"property"`
	TotalReports int             `json:"total_reports"`
	Groups       []EvidenceGroup `json:"groups"`
	// BestEvidence is the winning value, or "" when the top two groups tie on
	// both report count and confidence — an unresolved conflict surfaced to the
	// researcher rather than papered over.
	BestEvidence string `json:"best_evidence,omitempty"`
}

// showEvidence loads an archive, resolves the person, and prints the grouped
// evidence for the requested property in text or JSON form.
func showEvidence(io *IOStreams, archivePath, personQuery, property, format string) error {
	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	personID, _, err := findPersonByQuery(archive, personQuery)
	if err != nil {
		return err
	}

	report := collectEvidence(archive, personID, property)

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
		archive, duplicates, err := LoadArchiveWithOptions(path, false)
		if err != nil {
			return nil, fmt.Errorf("failed to load archive: %w", err)
		}
		for _, d := range duplicates {
			io.Errorf("Warning: %s\n", d)
		}

		return archive, nil
	}

	return readSingleFileArchive(path, false)
}

// noteConfidence raises the group's best confidence to c when c outranks the
// current best (lower confidenceRank means stronger).
func (g *EvidenceGroup) noteConfidence(c string) {
	if g.BestConfidence == "" || confidenceRank(c) < confidenceRank(g.BestConfidence) {
		g.BestConfidence = c
	}
}

// collectEvidence gathers every assertion for the given person+property,
// groups the supporting reports by asserted value, and ranks the values by
// report count and confidence. Output is deterministic.
func collectEvidence(archive *glxlib.GLXFile, personID, property string) EvidenceReport {
	report := EvidenceReport{
		Person:     personID,
		PersonName: personName(archive, personID),
		Property:   property,
	}

	groups := make(map[string]*EvidenceGroup)
	// citationIdx maps value -> citationID -> index of that citation's report in
	// the group's Items. The same record cited for the same value by more than
	// one assertion is a single report — but its confidence is upgraded to the
	// strongest seen across those assertions, rather than whichever was seen first.
	citationIdx := make(map[string]map[string]int)

	for _, a := range matchingAssertions(archive, personID, property) {
		// Resolve against the matched assertion's own property key (a.Property),
		// not the raw query string: under the case-insensitive fallback the query
		// casing can differ, and placeRefProperties / PersonProperties lookups are
		// case-sensitive, so resolving by the query would miss place/person/event
		// references.
		value := resolveAssertionValue(a.Value, a.Property, archive)
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

// matchingAssertions returns the assertions whose subject is personID and whose
// property matches. Exact matches win; only when none exist does it fall back
// to case-insensitive matches, so "born_at" never silently picks up "Born_At"
// when an exact "born_at" is present. Iteration order is deterministic.
func matchingAssertions(archive *glxlib.GLXFile, personID, property string) []*glxlib.Assertion {
	var exact, insensitive []*glxlib.Assertion

	for _, id := range sortedKeys(archive.Assertions) {
		a := archive.Assertions[id]
		if a == nil || a.Subject.Person != personID {
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
		return exact
	}

	return insensitive
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
// the citation ID when the citation or its source is missing.
func citationSourceLabel(citID string, archive *glxlib.GLXFile) string {
	cit, ok := archive.Citations[citID]
	if !ok || cit == nil {
		return citID
	}
	if label := sourceLabel(cit.SourceID, archive); label != "" {
		return label
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
//
// Assertion.Value is always a scalar string, so — unlike person properties —
// there is no temporal-shape map to unwrap here.
func resolveAssertionValue(value, property string, archive *glxlib.GLXFile) string {
	if value == "" {
		return value
	}

	// Fast path matching analyze's resolveConflictValue for place properties.
	if placeRefProperties[property] {
		return resolvePlaceName(value, archive)
	}

	if def, ok := archive.PersonProperties[property]; ok && def != nil {
		switch def.ReferenceType {
		case glxlib.EntityTypePlaces:
			return resolvePlaceName(value, archive)
		case glxlib.EntityTypePersons:
			return personName(archive, value)
		case glxlib.EntityTypeEvents:
			if ev, ok := archive.Events[value]; ok && ev != nil && ev.Title != "" {
				return ev.Title
			}
		}
	}

	return value
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
		io.Printf("No assertions found for %s of %s (%s).\n", r.Property, r.PersonName, r.Person)

		return
	}

	io.Printf("Evidence for %s of %s (%s):\n", r.Property, r.PersonName, r.Person)
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
	fmt.Fprintln(io.MachineOut, string(data)) //nolint:errcheck // CLI output

	return nil
}

// pluralize returns singular when n == 1, otherwise plural.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}

	return plural
}
