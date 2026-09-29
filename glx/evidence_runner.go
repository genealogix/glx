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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// showEvidence loads an archive, resolves the subject, and prints the grouped
// evidence for the requested property in text or JSON form.
func showEvidence(io *IOStreams, archivePath, subjectQuery, property, format string, options ...glxlib.ComparisonOptions) error {
	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	subject, err := findEvidenceSubject(archive, subjectQuery)
	if err != nil {
		return err
	}

	report, err := glxlib.BuildEvidenceReport(archive, subject, property, comparisonOptions(options))
	if err != nil {
		return err
	}

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

// printEvidenceText renders the human-readable report.
func printEvidenceText(io *IOStreams, r *EvidenceReport) {
	if len(r.Groups) == 0 && len(r.Undated) == 0 {
		io.Printf("No assertions found for %s of %s (%s).\n", r.Property, r.SubjectName, r.Subject)

		return
	}

	io.Printf("Evidence for %s of %s (%s):\n", r.Property, r.SubjectName, r.Subject)
	io.Printf("%d %s across %d %s\n\n",
		r.TotalReports, pluralize(r.TotalReports, "report", "reports"),
		len(r.Groups)+len(r.Undated), pluralize(len(r.Groups)+len(r.Undated), "value", "values"))

	if r.Temporal {
		io.Println("Dated history:")
	}
	printEvidenceGroups(io, r.Groups)
	if len(r.Undated) > 0 {
		io.Println("Undated:")
		printEvidenceGroups(io, r.Undated)
	}
	for _, c := range r.Conflicts {
		io.Printf("  %s conflict: %s / %s\n", c.Verdict, c.Values[0].Value, c.Values[1].Value)
	}
	if r.Temporal {
		return
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

func printEvidenceGroups(io *IOStreams, groups []EvidenceGroup) {
	for _, g := range groups {
		if g.Date != "" {
			io.Printf("  %s\n", g.Date)
		}
		io.Printf("  %s — %d %s, best confidence: %s\n", g.Value, g.Reports, pluralize(g.Reports, "report", "reports"), displayOrDash(g.BestConfidence))
		for _, item := range g.Items {
			io.Printf("    %-32s %-30s %s\n", displayOrDash(item.CitationID), item.Source, displayOrDash(item.Confidence))
		}
		if g.BestEvidence != "" {
			io.Printf("    Best evidence among overlapping claims: %s\n", g.BestEvidence)
		}
		io.Println("")
	}
}

type (
	EvidenceItem     = glxlib.EvidenceItem
	EvidenceGroup    = glxlib.EvidenceGroup
	EvidenceReport   = glxlib.EvidenceReport
	EvidenceConflict = glxlib.EvidenceConflict
)
