---
title: "ADR-NNNN: Short descriptive title"
description: Template for new Architecture Decision Records.
layout: doc
---

<!--
How to use this template:

1. Copy this file to docs/decisions/NNNN-kebab-case-title.md, where NNNN is
   the next unused 4-digit number (0001, 0002, ...).
2. Update the frontmatter `title` and `description`, and the main heading.
3. Fill in each section. Keep it short — one or two paragraphs per section
   is usually enough.
4. Open a PR. The initial Status is "Proposed".
5. When the maintainers accept the PR, flip Status to "Accepted" in the
   same commit that merges, or as a follow-up edit.
6. Once an ADR is Accepted, do not rewrite it. If the decision changes,
   write a new ADR that supersedes this one and set this Status to
   "Superseded by ADR-XXXX".
7. Delete this comment block before submitting.
-->

# ADR-NNNN: Short descriptive title

## Status

Proposed

<!--
One of:
- Proposed — under review
- Accepted — merged; reflects current practice
- Deprecated — no longer followed, but not replaced
- Superseded by ADR-XXXX — replaced by a later ADR; link to it and give its number
-->

## Context

What prompted this decision? Describe the problem, the forces at play, and any constraints. Link to related issues, discussions, or external references. A new contributor should be able to read this section and understand *why* a choice had to be made, without needing to know the outcome.

## Decision

What was decided. State it plainly, in one or two sentences if possible. If the decision has scope limits ("applies only to X") or carve-outs, call them out here.

## Consequences

What follows from this decision — both the positive and the negative. Include:

- Benefits the decision unlocks.
- Costs, trade-offs, or future work it creates.
- Constraints it places on contributors (e.g., "new CLI commands must also do X").
- Any compatibility implications (breaking change, deprecation window).

This section is where a future contributor reading the ADR learns what they are allowed to change and what they should not.
