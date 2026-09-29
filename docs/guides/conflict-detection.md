---
title: Conflict Detection
description: Compare evidence consistently while preserving changes over time
---

# Conflict detection

`analyze`, `proof`, `evidence`, and `merge-persons` use the same comparison rules.
Each command selects the facts relevant to its task.

## History and uncertainty

Properties marked `temporal: true` in the vocabulary can change over time.
Leeds in 1851 and London in 1875 describe history. Periods ending and starting
at the same boundary, such as `FROM 1850 TO 1870` and `FROM 1870 TO 1920`, touch
without conflicting.

A period `FROM 1850 TO 1880` certainly covers 1875. Two different values within
that overlap conflict. Two values recorded only as `1851` might both be correct
at different points during the year, so their conflict is only possible.
Approximate dates, open ends, reversed ranges, and `BET … AND` uncertainty never
prove simultaneous validity. An undated temporal value cannot contradict a
history.

Periods recorded in different calendars are possible overlaps until they can be
compared in a common calendar. Their numeric date components alone cannot prove
that the periods are disjoint.

## Agreement and resolution

- A precise date can refine a year: `1850` and `1850-03-02` agree.
- Approximate date values (`ABT`, `EST`, `CAL`) use ±2 years by default. All four
  comparing commands accept `--approximation-years N` (0 through 10000). This
  widens date **values**, not the validity periods of temporal properties.
- A place and its ancestor agree: Hartford County refines Connecticut. Distinct
  places sharing a name remain separate, with IDs added to distinguish them.
- Free text ignores case, punctuation, and extra whitespace. Vocabulary values
  and entity references retain their exact identity.
- `disproven` claims drop out. `disputed` claims remain acknowledged disputes,
  including an explicitly disputed claim with no competing value recorded.
  This also applies to participant and existence assertions.

Agreement is checked pairwise. A broad year does not make two incompatible exact
birth dates agree merely because both fall within that year.

## Command output

`analyze --check conflicts` includes assertions on the person, their events and
relationships. Duplicate birth or death events are compared for their principal
participant. Definite conflicts are HIGH, possible conflicts MEDIUM, and known
disputes LOW. Resolved conflicts and undated temporal claims are omitted.

`proof` uses the same verdicts for facts relevant to the research question.
Only a definite unresolved conflict makes the conclusion CONFLICTED. A possible
conflict appears as “possible — check” and limits support to POSSIBLE. Undated
temporal evidence appears separately.

`evidence` presents a temporal property in date order, followed by an Undated
section. It ranks evidence only against claims whose periods could overlap;
there is no overall winner between values recorded at different times. JSON
includes `temporal`, dated `groups`, `undated`, and `conflicts`.

`merge-persons` keeps non-conflicting history from both people, including mixed
scalar, structured, and list representations. Identical values agree silently.
Conflicting entries retain the kept person's value unless `--keep-newest` or
`--keep-oldest` selects the other entry. Those flags affect collisions; they do
not remove unrelated history. Use `--dry-run` to inspect the report.
Disagreements already recorded within either person's history are preserved;
merging people does not resolve those claims based on their order in a list.

`validate` warns when a range ends before it starts. Conflicting evidence remains
valid archive data; validation does not choose which claim is correct.

See the [analyze](../cli/glx_analyze.md), [proof](../cli/glx_proof.md),
[evidence](../cli/glx_evidence.md), and [merge-persons](../cli/glx_merge-persons.md)
command references.

## Go SDK contract

The `github.com/genealogix/glx/go-glx` package exposes the same domain operations
used by the CLI. It performs no filesystem access or terminal output.

| API | Result |
| --- | --- |
| `CompareFacts` | Pairwise verdicts for explicitly supplied property semantics |
| `EvaluateFacts` | Pairwise verdicts, selected/surviving input indices, and resolution |
| `CollectPersonFacts` | Recorded claims and synthetic duplicate-event facts, with provenance |
| `PersonConflicts` | Grouped person/event/relationship disagreements, including resolved groups |
| `AnalyzeConflicts` | Unresolved findings for the archive or one exact person ID |
| `BuildEvidenceReport` | Ranked evidence and dated/undated history for one typed subject/property |
| `BuildProof` | Evidence, conflicts, searches, coverage gaps, and a conclusion for a research question |
| `BuildCoverage` | The source-coverage checklist used by proof |
| `MergePersons` | An in-memory merge with history preservation and conflict reports |

For example:

```go
import glx "github.com/genealogix/glx/go-glx"

width := 3
opts := glx.ComparisonOptions{ApproximationYears: &width}
report, err := glx.BuildProof(archive, "person-mary", glx.QuestionBirth,
    glx.ProofOptions{Comparison: opts})
if err != nil {
    return err
}
// The application renders or serializes report as needed.
```

Research entry points accept **exact entity IDs**, validate their options, and
return errors for nil archives, missing subjects, or unknown questions. Use
`errors.Is` with exported errors such as `ErrInvalidApproximation`,
`ErrNilArchive`, `ErrPersonNotFound`, and `ErrResearchSubjectNotFound`.
`ComparisonOptions{}` uses ±2 years; a pointer to zero disables widening.
Negative widths and widths above 10000 are rejected by both SDK and CLI.

Read operations supply missing standard property definitions without changing
the archive. Explicit custom definitions take precedence. Nonempty type
vocabularies remain archive-owned sets. `MergeStandardVocabularies` is available
when an application explicitly wants to fill defaults in-place. Low-level
`CompareFacts` and `EvaluateFacts` instead use the provided definition; a nil
definition means a fixed free-text property.

Read results own their slices and can be modified without changing the archive.
`ConflictGroup.Facts` retains raw subject IDs, assertion IDs, values, periods,
and support references; `Evaluation` indexes that fact slice. A singleton
explicit dispute has a self-comparison (`Left == Right`). Compatibility remains
pairwise: a broad date cannot make incompatible precise dates agree transitively.

`CoverageOptions.CensusCountry` supplies a fallback country explicitly; its zero
value makes no geographic assumption. There is no shared mutable SDK run state.
Applications may run read operations concurrently on an unchanged archive.
`MergePersons` mutates the archive after argument validation, so synchronize it
against other readers and writers. These operations tolerate incomplete research
records but do not replace `GLXFile.Validate` for archive conformance checks.
