<!--
  PR title must follow Conventional Commits: type: Description
  Examples: feat: Add person search, fix: Handle nil map in merge
  Description should start with an uppercase letter (style convention; not enforced by CI).
  The lint-pr-title check enforces only the allowed types (feat, fix, docs, chore,
  refactor, test, perf, ci) — it is the source of truth for that list.
  On squash-merge this title is normally used as the commit subject, so keep it accurate.
-->

## What and why

<!-- What changed and why? Link to design discussion if relevant.
     Spec, data-model, validation-rule, or file-format change? Link the accepted proposal
     issue (min 7-day discussion) and any ADR — see CONTRIBUTING.md "Proposing Major Changes". -->

## Related issues

<!-- Fixes #123 — or "None" if no linked issue. Use a closing keyword (Closes/Fixes/Resolves)
     so the issue auto-closes on merge; a bare "#123" links but does not close.
     Non-trivial changes should reference an issue created first (CONTRIBUTING.md "Pull Request Process"). -->

## Review focus

<!-- What should reviewers pay attention to? e.g. "API design", "correctness", or "trivial change". -->

## Testing

<!-- What did you run or verify? For website/UI changes, a screenshot or before/after is appreciated. -->

## Breaking changes

<!-- None, or describe the migration path. -->

<!-- Before submitting (none of these are CI-enforced — they're trusted/checked at review):
     - Sign off EVERY commit (git commit -s) per the DCO. Forgot one?
       git rebase --signoff HEAD~N && git push --force-with-lease — see CONTRIBUTING.md "Developer Certificate of Origin".
     - If AI tools substantially helped, disclose via an `Assisted-by:` commit trailer
       (see CONTRIBUTING.md "AI-Assisted Development"). You remain accountable for the change.
     - Update CHANGELOG.md for user-facing changes (Added/Changed/Fixed/Removed); every entry
       needs an issue/PR reference, e.g. (#123) or Fixes #123. -->
