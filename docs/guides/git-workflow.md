---
title: Git Workflow Guide
description: Branching strategies, collaboration patterns, and branch-based research workflows for GLX archives.
layout: doc
---

# Git Workflow Guide

A GLX archive is a Git repository of YAML files. That single design decision gives genealogy four things databases struggle with — history, collaboration, backup, and longevity — but only if you actually use the Git side of the format. This guide shows how: daily commit habits, hypothesis branches, collaborating with other researchers, and reading history as a research log.

> **Background:** For *why* archives are Git repositories, see [ADR-0004](/decisions/0004-git-native-archives) and [Core Concepts — Collaboration](/specification/2-core-concepts#collaboration). For general archive hygiene, see the [Best Practices Guide](/guides/best-practices).

This is not a general Git tutorial — it assumes you know `clone`, `add`, `commit`, `push`, and `pull`. If you don't yet, the [Git book](https://git-scm.com/book/en/v2) chapters 1–3 cover everything this guide uses.

## Setting Up an Archive Repository

`glx init` creates the default multi-file archive layout, a `.gitignore` (which excludes the disposable `.glx/` cache directory), and a `README.md` — but it does not create the Git repository itself. (With `--single-file`, only `archive.glx` is written, so add the `.gitignore` yourself.) Do that immediately after:

```bash
glx init my-family-archive
cd my-family-archive
git init -b main
git add .
git commit -m "Initial archive structure"
```

One setting is worth pinning before the first collaborator arrives:

```bash
# .gitattributes — pin line endings so Windows and Unix collaborators
# don't generate spurious whole-file diffs
echo "*.glx text eol=lf" > .gitattributes
echo "*.md text eol=lf" >> .gitattributes
git add .gitattributes
git commit -m "Pin LF line endings for archive files"
```

Entity filenames are derived from entity IDs and lowercased at serialize time, so avoid IDs that differ only by case — they collide on the case-insensitive filesystems common on Windows and macOS. See [Best Practices — ID Generation](/guides/best-practices#id-generation).

### Remote Backup

Mirroring an archive takes two commands — add the remote, then push:

```bash
git remote add origin git@github.com:yourname/my-family-archive.git
git push -u origin main
```

::: warning Privacy
An archive almost always contains data about **living people**. Keep the remote repository private by default, and treat making it public as a deliberate decision: review person entities for living individuals first. A public Git host is a publication — cloned copies and caches survive even if you later delete the repository.
:::

Any Git host works — GitHub, GitLab, Codeberg, a bare repository on a NAS, or all of them at once (`git remote add` is repeatable). The archive is portable because nothing about it depends on the host.

## The Research Cycle: Edit, Validate, Commit

The unit of work in genealogy is the **research finding**: you examined a source, extracted evidence, and recorded conclusions. Make that the unit of commit, too — one source examined, one commit. Avoid both extremes: a commit per keystroke is noise, and a month of research in one commit destroys the audit trail that makes Git worth using.

Always validate before committing, so every commit in history is a known-good archive state:

```bash
glx validate
git add -A
git commit -m "Add 1851 Census evidence for Smith family

- John Smith: occupation (blacksmith), residence
- Mary Smith: age, birthplace
- Source: HO107, Piece 2319, Yorkshire"
```

`git add -A` stages everything in the archive, which is what you want when the commit *is* the finding; if you have unrelated work in progress, stage those paths explicitly instead.

A good archive commit message answers three questions: **what changed** (which people/events), **on what evidence** (which source), and **why** (what made you conclude this). The subject line states the finding; the body carries the detail. Six months later, `git log` reads as a research journal.

::: tip Automate the validation gate
A pre-commit hook makes "every commit validates" automatic. In your archive repository:

```bash
mkdir -p .git/hooks
printf '#!/bin/sh\nexec glx validate\n' > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

Now `git commit` refuses to proceed while the archive has validation errors.
:::

## Branch-Based Research

Branches are where Git stops being a backup tool and becomes a research method. A genealogical hypothesis — "I think Thomas Smith of Pudsey is John's father" — is exactly what a branch models: a line of work that might be confirmed and merged, or refuted and closed, without contaminating your proven conclusions on `main`.

### Branch Naming

Use prefixes that say what kind of work the branch holds:

```bash
git checkout -b research/thomas-smith-paternity   # a hypothesis under investigation
git checkout -b evidence/1851-census-yorkshire    # transcribing one record set
git checkout -b import/aunt-carol-gedcom          # reviewing an import before accepting it
git checkout -b fix/duplicate-mary-smith          # data cleanup
```

The convention matters more than the exact prefixes — `git branch --list 'research/*'` becomes your list of open research questions.

### A Hypothesis Branch, Start to Finish

On the branch, record the hypothesis as **speculative assertions** — full entities, real citations, honest confidence — so the work is structured data from day one, not notes to be transcribed later:

```bash
git checkout -b research/thomas-smith-paternity
```

```yaml
# relationships/relationship-thomas-john-smith.glx
relationships:
  relationship-thomas-john-smith:
    type: parent_child
    participants:
      - person: person-thomas-smith
        role: parent
      - person: person-john-smith
        role: child

# assertions/assertion-thomas-paternity.glx
assertions:
  assertion-thomas-paternity:
    subject:
      relationship: relationship-thomas-john-smith
    property: type
    value: parent_child
    citations: [citation-1841-census-pudsey]
    confidence: medium
    status: speculative   # the branch IS the hypothesis
```

Commit research on the branch as it accumulates. When the evidence becomes decisive, the branch ends one of two ways:

**Confirmed.** Update `status: speculative` → `proven`, raise `confidence` if warranted, cite the clinching source, and merge:

```bash
glx validate
git checkout main
git merge research/thomas-smith-paternity
git branch -d research/thomas-smith-paternity
```

**Refuted.** Do *not* simply delete the branch. A disproven hypothesis is a research result — the next researcher (possibly you, in five years) needs to know this avenue was tried and why it failed, or they will repeat it. Flip the assertion to `status: disproven`, cite the disproving source in a note, and merge anyway:

```yaml
assertions:
  assertion-thomas-paternity:
    subject:
      relationship: relationship-thomas-john-smith
    property: type
    value: parent_child
    citations: [citation-1841-census-pudsey, citation-thomas-burial-1838]
    confidence: high
    status: disproven
    notes:
      - "Thomas Smith of Pudsey buried 1838-11-02, three years before
        John's birth. Cannot be the father. See burial register citation."
```

Negative findings merged to `main` keep the archive honest about what is known *not* to be true — and `status: disproven` keeps them out of conclusion-oriented views while preserving the audit trail.

### Comparing Branches

Plain `git diff main..research/thomas-smith-paternity` shows the textual changes, and because GLX stores one entity per file, the file list alone tells you which people and events the hypothesis touches. For a genealogy-aware comparison, check out both states and use [`glx diff`](/cli/glx_diff):

```bash
git worktree add /tmp/archive-main main
glx diff /tmp/archive-main . --short
git worktree remove /tmp/archive-main
```

`git worktree` gives you a second checkout of another branch without disturbing your working directory — useful any time a `glx` command needs to see two archive states at once.

## Collaboration Patterns

Which workflow fits depends on how much the collaborators trust each other's research, not how much they trust each other personally.

### Shared Repository

For a small family research team — siblings, cousins, a parent — give everyone push access to one repository. The one rule that keeps it pleasant: **nobody commits directly to `main`**. Research happens on branches; merging to `main` happens through a pull request, even a self-approved one. The PR is where the second pair of eyes happens.

```bash
git checkout -b research/webb-line-virginia
# ...research, commit, push...
git push -u origin research/webb-line-virginia
# open a PR; a collaborator reviews; merge
```

Reduce conflict surface by dividing work along natural research lines — one person per surname line, county, or record set. Because GLX stores one entity per file, two researchers working different family lines almost never touch the same file, and their merges are automatic.

### Fork and Pull Request

For looser collaborations — a one-name study, a local history society, a stranger who emails "I think we share a great-great-grandfather" — use the maintainer model that open-source projects run on. Contributors fork the archive, research on a branch in their fork, and open a PR. One or two maintainers review and merge. Contributors never need push access, so the bar to accepting help from a stranger is low: their work arrives as a reviewable proposal, not as edits to your data.

### Review as Genealogical Peer Review

A pull request review on an archive is a proof-standard check, not a code review. Things a reviewer should ask of every PR:

- Does every new assertion cite a source? (No orphan conclusions — see [Best Practices](/guides/best-practices#complete-evidence-chains).)
- Are `confidence` and `status` honest? `status: proven` should be rare in a PR; most incoming research is `speculative` until corroborated.
- Does new evidence conflict with existing assertions? If so, the PR should *record* the conflict (`status: disputed` on both sides), not silently overwrite the old conclusion.
- Does `glx validate` pass? Don't burn human review time on what a machine checks — automate it:

```yaml
# .github/workflows/validate.yml
name: Validate archive
on: [push, pull_request]
jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install and verify glx
        run: |
          # Verify the published checksum before running the binary — never pipe
          # an unverified download straight into a shell. For full reproducibility,
          # replace "latest/download" with a pinned "download/vX.Y.Z".
          mkdir -p .bin
          base="https://github.com/genealogix/glx/releases/latest/download"
          curl -fsSL -o glx_Linux_x86_64.tar.gz "$base/glx_Linux_x86_64.tar.gz"
          curl -fsSL -o checksums.txt "$base/checksums.txt"
          grep ' glx_Linux_x86_64.tar.gz$' checksums.txt | sha256sum -c -
          tar -xzf glx_Linux_x86_64.tar.gz -C .bin glx
      - run: ./.bin/glx validate
```

For shared archives, also agree on vocabulary governance — who may add custom event or relationship types, and how. See [Best Practices — Vocabulary Governance](/guides/best-practices#vocabulary-governance).

## Merging and Conflict Resolution

First, a distinction that trips people up. There are two different "merges":

| | Use |
|---|---|
| `git merge` | Combining **branches of the same repository** — hypothesis branches, collaborators' PRs. Git matches files line-by-line. |
| [`glx merge`](/cli/glx_merge) | Combining **two separate archives** — a cousin's independently-built archive into yours. Entities are copied by ID; there is no shared Git history to merge. |

If you and your cousin both cloned the same repository, you want `git merge` (via pull requests). If you built archives independently and discovered the overlap later, you want `glx merge` — see the [Hands-On CLI Guide](/guides/hands-on-cli-guide#archive-merging).

### Why Conflicts Are Rare

A Git conflict requires two branches to edit the same lines of the same file. GLX's file-per-entity layout means that only happens when two people edit *the same person, event, or assertion* — different entities never conflict, no matter how much parallel work happened. Most archive merges are clean.

### When They Happen Anyway

A conflict in a `.glx` file looks like any Git conflict:

```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
<<<<<<< HEAD
      occupation: Blacksmith
=======
      occupation: Farrier
>>>>>>> research/1861-census
```

Resolve it by editing the file to the correct YAML, then `git add` and complete the merge. But pause on *why* it conflicted:

- **Same fact, refined reading** — one side corrected a transcription the other also touched. Keep the better reading; done.
- **Genuinely conflicting evidence** — the 1851 census says blacksmith, the 1861 says farrier, and both are right for their year, or the sources actually disagree. Don't let the merge force a choice the evidence doesn't support: keep **both** as assertions with their own citations, mark them `status: disputed` if they're irreconcilable, and record the reasoning. The textual conflict was a symptom; the evidential conflict is the real data. See [Best Practices — Conflicting Evidence](/guides/best-practices#conflicting-evidence).

::: warning
Never resolve a merge by silently discarding another researcher's cited assertion. If you believe their conclusion is wrong, the archive has a vocabulary for that — `status: disputed` or `disproven`, with a note saying why. Deleting cited evidence in a merge is the Git equivalent of tearing a page out of a shared notebook.
:::

After any merge — `git` or `glx` — run the post-merge checks:

```bash
glx validate      # cross-references intact, no duplicate IDs
glx duplicates    # did the merge introduce the same person twice under different IDs?
```

If `glx duplicates` finds that you and your collaborator each created the same ancestor independently, fold one into the other with [`glx merge-persons`](/cli/glx_merge-persons).

## History as a Research Log

Once the habits above are in place, Git's history tools become genealogy tools.

**The biography of a file.** Every conclusion's evolution is replayable:

```bash
# Everything that ever happened to one person
git log --follow -p -- persons/person-john-smith.glx

# When was this conclusion drawn, and on what evidence?
git blame assertions/assertion-john-birth.glx

# What did the archive look like before the GEDCOM import?
git diff import-baseline..main --stat
```

**Tags as research milestones.** Tag states worth returning to — a generation fully proven, the archive as it stood before a big import, the version you shared at the family reunion:

```bash
git tag -a generation-4-proven -m "All 16 great-great-grandparents documented with primary sources"
git tag -a pre-import-2026 -m "State before merging Aunt Carol's GEDCOM"
```

**Correct, don't erase.** When a long-held conclusion turns out wrong, resist rewriting it out of existence. Update the assertion (`status: disproven`, corrected values, a note explaining the new reading) in a *new commit* — the old state stays in history, and `git log` shows exactly when and why the conclusion changed. History rewriting (`rebase`, `push --force`) has no place in a shared archive: it destroys the provenance trail that is half the point of a Git-native format.

## Large Archives and Media Files

Git handles tens of thousands of small YAML files comfortably — text is what it was built for. The pressure points are binaries and very large histories:

- **Media scans** (`media/files/`) are binary JPG/PDF files that Git stores whole on every change. For archives with gigabytes of scans, use [Git LFS](https://git-lfs.com/): `git lfs track "media/files/*"` keeps the repository fast while the scans remain part of the archive. Configure LFS *before* committing the scans — retrofitting it requires history rewriting.
- **`.glx/` cache** is disposable and already ignored by the `glx init` `.gitignore`; never commit it.
- **Very large archives** (hundreds of thousands of entities) can use Git's partial-clone (`git clone --filter=blob:none`) so collaborators fetch file contents on demand, or split along family lines into separate archives combined with `glx merge` when needed.

## See Also

- [ADR-0004: Archives are Git repositories](/decisions/0004-git-native-archives) — the reasoning behind Git-native design
- [Best Practices Guide](/guides/best-practices) — evidence chains, vocabulary governance, validation habits
- [Hands-On CLI Guide](/guides/hands-on-cli-guide) — `glx merge`, `glx diff`, `glx duplicates` walkthroughs
- [Core Concepts — Collaboration](/specification/2-core-concepts#collaboration) — the specification's collaboration model
- [Quickstart](/quickstart) — create your first archive
