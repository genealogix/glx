---
title: Working with Conflicting Evidence
description: Find disagreements in your sources and record your research conclusions
layout: doc
---

# Working with Conflicting Evidence

A census, a parish register, and a family recollection may give different answers
about the same person. GLX helps you find those disagreements, compare the
sources, and keep a record of how you reached your conclusion.

## Find disagreements

Start with a conflict check:

```bash
glx analyze --check conflicts
```

Use the report to choose what to investigate next:

- **HIGH:** recorded claims contradict each other.
- **MEDIUM:** the claims might conflict, but their timing is uncertain.
- **LOW:** you have already marked the evidence as disputed.

Some differences describe a person's changing life. A residence in Leeds in 1851
and London in 1875 can both be correct. Names and occupations can change too.
Recording when a claim was true helps GLX distinguish that history from a
contradiction.

## Compare the sources

For a birth-date question, look at the evidence for the birth event. Replace the
example ID with the event in your archive:

```bash
glx evidence event-jane-birth date
```

Review the supporting records and consider who supplied the information, how
close they were to the event, and whether the sources are independent. Two
records may agree even when one gives more detail: “1850” and “2 March 1850” can
describe the same birth.

For information that changes over time, such as residence, the evidence report
shows a dated history. Undated claims appear separately, ready for further
research.

## Record your conclusion

Keep the competing assertions and their citations so another researcher can
follow your reasoning. When you rule out a claim, mark it `disproven` and explain
why in its notes. If the disagreement remains unresolved, use `disputed` to
acknowledge it.

Then review the proof summary for your question:

```bash
glx proof person-jane --question birth
```

The summary brings together the evidence, remaining conflicts, and research gaps.
Resolved disagreements disappear from the conflict check; the assertions remain
in your archive as part of the research record.

See [Assertion status](../../specification/4-entity-types/assertion.md#status) for
recording your decision, or the [analyze](../cli/glx_analyze.md),
[evidence](../cli/glx_evidence.md), and [proof](../cli/glx_proof.md) command
references for more options.
