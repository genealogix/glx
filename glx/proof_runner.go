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
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Output format keys accepted by the proof command.
const (
	proofFormatText     = "text"
	proofFormatJSON     = "json"
	proofFormatMarkdown = "markdown"
	proofFormatMD       = "md"
)

// Static base errors so dynamic context can be wrapped (satisfies err113).
var (
	errUnknownQuestion = glxlib.ErrUnknownResearchQuestion
	errUnknownFormat   = errors.New("unknown format")
)

// showProof loads an archive and prints a structured proof summary for a person
// and research question, following the BCG Genealogical Proof Standard format.
// Human-readable text/markdown is written to io.Out (silenced by --quiet); JSON,
// which is machine-consumable, is written to io.MachineOut so it survives --quiet
// for shell capture.
func showProof(io *IOStreams, archivePath, personQuery, question, format string, options ...glxlib.ComparisonOptions) error {
	topic, ok := glxlib.CanonicalProofQuestion(question)
	if !ok {
		return fmt.Errorf("%w %q (valid: %s)", errUnknownQuestion, question, strings.Join(glxlib.ProofQuestions(), ", "))
	}

	archive, err := loadArchiveForEvidence(io, archivePath)
	if err != nil {
		return err
	}

	personID, _, err := findPersonForCoverage(archive, personQuery)
	if err != nil {
		return err
	}

	result, err := glxlib.BuildProof(archive, personID, topic, glxlib.ProofOptions{Comparison: comparisonOptions(options)})
	if err != nil {
		return err
	}

	switch format {
	case proofFormatJSON:
		return printProofJSON(io, result)
	case proofFormatMarkdown, proofFormatMD:
		printProofMarkdown(io, result)
	case "", proofFormatText:
		printProofText(io, result)
	default:
		return fmt.Errorf("%w %q (valid: %s)", errUnknownFormat, format, strings.Join(proofFormatKeys(), ", "))
	}

	return nil
}

// proofFormatKeys returns the accepted output format tokens, in display order,
// for error messages. Kept in lockstep with the format switch in showProof so
// the valid set advertised to users never drifts from what is actually accepted.
func proofFormatKeys() []string {
	return []string{proofFormatText, proofFormatJSON, proofFormatMarkdown, proofFormatMD}
}

// printProofJSON outputs the proof result as JSON. JSON is machine-consumable, so
// it goes to MachineOut — which, unlike Out, survives --quiet for shell capture.
func printProofJSON(io *IOStreams, result *proofResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Fprintln(io.MachineOut, string(data))

	return nil
}

// printProofText renders the proof result as human-readable terminal output.
func printProofText(io *IOStreams, result *proofResult) {
	io.Printf("Proof Summary: %s (%s)\n\n", result.QuestionText, result.PersonID)
	io.Printf("  Question: %s\n", result.QuestionText)

	io.Printf("\n  Evidence Collected:\n")
	if len(result.Evidence) == 0 {
		io.Println("    (none)")
	}
	for i := range result.Evidence {
		printProofEvidenceText(io, i+1, &result.Evidence[i])
	}

	if len(result.Undated) > 0 {
		io.Printf("\n  Undated:\n")
		for i := range result.Undated {
			printProofEvidenceText(io, i+1, &result.Undated[i])
		}
	}
	io.Printf("\n  Evidence Gaps:\n")
	if len(result.Gaps) == 0 {
		io.Println("    (none identified)")
	}
	for i := range result.Gaps {
		io.Println("    [ ] " + gapLine(&result.Gaps[i], " -- ", "HIGH PRIORITY"))
	}

	if len(result.Searches) > 0 {
		io.Printf("\n  Reasonably Exhaustive Search:\n")
		for i := range result.Searches {
			io.Println("    " + formatSearchLine(&result.Searches[i]))
		}
	}

	io.Printf("\n  Conflicts: ")
	if len(result.Conflicts) == 0 {
		io.Println("None identified")
	} else {
		io.Println("")
		for i := range result.Conflicts {
			printProofConflictText(io, &result.Conflicts[i])
		}
	}

	io.Printf("\n  Conclusion: %s\n", result.Conclusion)
	if result.Summary != "" {
		io.Printf("    %s\n", result.Summary)
	}
}

// printProofEvidenceText prints one numbered evidence entry.
func printProofEvidenceText(io *IOStreams, n int, ev *proofEvidence) {
	io.Printf("    %d. %s\n", n, evidenceHeader(ev))

	if detail := evidenceDetail(ev); detail != "" {
		io.Printf("       -> %s\n", detail)
	}
	if ev.Notes != "" {
		io.Printf("       -> %s\n", ev.Notes)
	}
}

// evidenceHeader builds the first line of an evidence entry: the source title
// (or assertion ID) plus the backing reference.
func evidenceHeader(ev *proofEvidence) string {
	if len(ev.Support) == 0 {
		return "Uncited assertion (" + ev.AssertionID + ")"
	}

	s := ev.Support[0]
	title := s.SourceTitle
	if title == "" {
		title = s.SourceID
	}
	if title == "" {
		title = s.Ref
	}
	header := title + " (" + s.Ref + ")"
	if len(ev.Support) > 1 {
		header += fmt.Sprintf(" +%d more", len(ev.Support)-1)
	}

	return header
}

// evidenceDetail builds the claim line of an evidence entry.
func evidenceDetail(ev *proofEvidence) string {
	var b strings.Builder
	if ev.Subject != "" {
		b.WriteString(ev.Subject)
		b.WriteString(": ")
	}

	if ev.Property == "" && ev.Value == "" {
		// Existential assertion: only the subject's existence is claimed
		// (optionally scoped by date).
		b.WriteString("existence asserted")
		if ev.Date != "" {
			b.WriteString(" (" + ev.Date + ")")
		}
	} else {
		if ev.Property != "" {
			b.WriteString(ev.Property)
			if ev.Value != "" {
				b.WriteString(" = ")
			}
		}
		b.WriteString(ev.Value)
	}

	if tags := claimTags(ev.Confidence, ev.Status); tags != "" {
		b.WriteString(" [")
		b.WriteString(tags)
		b.WriteString("]")
	}

	return strings.TrimSpace(b.String())
}

// claimTags renders the confidence/status annotation for a claim.
func claimTags(confidence, status string) string {
	var tags []string
	if confidence != "" {
		tags = append(tags, "confidence: "+confidence)
	}
	if status != "" {
		tags = append(tags, "status: "+status)
	}

	return strings.Join(tags, ", ")
}

// printProofConflictText prints one conflict entry.
func printProofConflictText(io *IOStreams, c *proofConflict) {
	io.Printf("    ! %s: %s\n", conflictLabel(c), conflictValuesString(c))
	if c.Resolved {
		io.Printf("      RESOLVED: %s\n", c.Resolution)
	} else if !c.Definite && c.Verdict != "" {
		io.Printf("      possible — check\n")
	} else {
		io.Printf("      UNRESOLVED — resolution needed\n")
	}
}

// conflictLabel renders the conflict's subject (when known) and property.
func conflictLabel(c *proofConflict) string {
	if c.Subject != "" {
		return c.Subject + " " + c.Property
	}

	return c.Property
}

// conflictValuesString renders the competing values of a conflict.
func conflictValuesString(c *proofConflict) string {
	values := make([]string, 0, len(c.Values))
	for i := range c.Values {
		v := &c.Values[i]
		entry := v.Value
		var tags []string
		if v.Confidence != "" {
			tags = append(tags, v.Confidence)
		}
		if v.Status != "" {
			tags = append(tags, v.Status)
		}
		if len(tags) > 0 {
			entry += " [" + strings.Join(tags, ", ") + "]"
		}
		values = append(values, entry)
	}

	return strings.Join(values, " vs ")
}

// gapLine renders a single evidence-gap line with a separator and priority label.
func gapLine(g *proofGap, sep, highLabel string) string {
	line := g.Label
	if g.Priority == severityHigh {
		line += sep + highLabel
	}
	if g.Description != "" {
		line += sep + g.Description
	}

	return line
}

// formatSearchLine renders a logged search as a single line.
func formatSearchLine(s *proofSearch) string {
	target := s.Source
	if target == "" {
		target = s.Repository
	}
	if s.Collection != "" {
		if target != "" {
			target += " / "
		}
		target += s.Collection
	}

	var b strings.Builder
	if s.Query != "" {
		b.WriteString(s.Query)
	} else {
		b.WriteString("(search)")
	}
	if target != "" {
		b.WriteString(" @ ")
		b.WriteString(target)
	}
	if s.Result != "" {
		b.WriteString(" -> ")
		b.WriteString(s.Result)
	}

	return b.String()
}

// printProofMarkdown renders the proof result as Markdown.
func printProofMarkdown(io *IOStreams, result *proofResult) {
	io.Printf("# Proof Summary: %s\n\n", result.QuestionText)
	io.Printf("- **Person:** %s (`%s`)\n", result.PersonName, result.PersonID)
	io.Printf("- **Question:** %s\n", result.QuestionText)
	io.Printf("- **Conclusion:** %s\n", result.Conclusion)
	if result.Summary != "" {
		io.Printf("- **Summary:** %s\n", result.Summary)
	}

	printMarkdownEvidence(io, result.Evidence)
	if len(result.Undated) > 0 {
		io.Printf("\n## Undated\n\n")
		printMarkdownEvidenceItems(io, result.Undated)
	}
	printMarkdownGaps(io, result.Gaps)
	printMarkdownSearches(io, result.Searches)
	printMarkdownConflicts(io, result.Conflicts)
}

// printMarkdownEvidence renders the evidence section as Markdown.
func printMarkdownEvidence(io *IOStreams, evidence []proofEvidence) {
	io.Printf("\n## Evidence Collected\n\n")
	if len(evidence) == 0 {
		io.Println("_No evidence collected._")
	}
	printMarkdownEvidenceItems(io, evidence)
}

func printMarkdownEvidenceItems(io *IOStreams, evidence []proofEvidence) {
	for i := range evidence {
		ev := &evidence[i]
		io.Printf("%d. **%s**\n", i+1, evidenceHeader(ev))
		if detail := evidenceDetail(ev); detail != "" {
			io.Printf("   - %s\n", detail)
		}
		if ev.Notes != "" {
			io.Printf("   - %s\n", ev.Notes)
		}
	}
}

// printMarkdownGaps renders the evidence-gaps section as Markdown.
func printMarkdownGaps(io *IOStreams, gaps []proofGap) {
	io.Printf("\n## Evidence Gaps\n\n")
	if len(gaps) == 0 {
		io.Println("_None identified._")
	}
	for i := range gaps {
		io.Println("- [ ] " + gapLine(&gaps[i], " — ", "**HIGH PRIORITY**"))
	}
}

// printMarkdownSearches renders the logged-searches section as Markdown.
func printMarkdownSearches(io *IOStreams, searches []proofSearch) {
	if len(searches) == 0 {
		return
	}
	io.Printf("\n## Reasonably Exhaustive Search\n\n")
	for i := range searches {
		io.Println("- " + formatSearchLine(&searches[i]))
	}
}

// printMarkdownConflicts renders the conflicts section as Markdown.
func printMarkdownConflicts(io *IOStreams, conflicts []proofConflict) {
	io.Printf("\n## Conflicts\n\n")
	if len(conflicts) == 0 {
		io.Println("_None identified._")
	}
	for i := range conflicts {
		c := &conflicts[i]
		io.Printf("- **%s:** %s\n", conflictLabel(c), conflictValuesString(c))
		if c.Resolved {
			io.Printf("  - _Resolved:_ %s\n", c.Resolution)
		} else if !c.Definite && c.Verdict != "" {
			io.Printf("  - _possible — check._\n")
		} else {
			io.Printf("  - _Unresolved — resolution needed._\n")
		}
	}
}

type (
	proofSupport       = glxlib.ProofSupport
	proofEvidence      = glxlib.ProofEvidence
	proofGap           = glxlib.ProofGap
	proofConflictValue = glxlib.ConflictValue
	proofConflict      = glxlib.ConflictGroup
	proofSearch        = glxlib.ProofSearch
	proofResult        = glxlib.ProofResult
)
