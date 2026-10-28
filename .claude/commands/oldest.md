---
description: Show which tracked files in a repository have gone the longest without a change
allowed-tools:
  - Bash(git -C:*)
  - Bash(git clone:*)
  - Bash(mktemp -d /tmp/oldest-*:*)
  - Bash(rm -rf /tmp/oldest-*:*)
  - Bash(awk:*)
  - Bash(sort:*)
  - Bash(head:*)
model: claude-sonnet-5-5
disable-model-invocation: true
argument-hint: "[repo-path | owner/repo | git-url] [count]"
---

Report the tracked files in a repository whose most recent change is the oldest — the files nobody has touched for the longest time.

## Arguments

`$ARGUMENTS` holds up to two optional tokens, in either order:

- **Repository** — a local path, a GitHub `owner/repo`, or any git URL. Omitted, it is the current repository (`.`).
- **Count** — a positive integer, the number of files to list. Omitted, it is `10`.

A token that is all digits is the count. A token that names an existing local directory is a path. Anything else of the form `owner/repo` becomes `https://github.com/owner/repo`; anything containing `://` or starting with `git@` is used as given.

## Step 1: Resolve the repository

**Local path:** confirm it is a git work tree or bare repo with `git -C <path> rev-parse --git-dir`. If it is not, stop and say so. Use `<path>` as `$REPO` below.

**Remote:** make a temporary directory with `mktemp -d /tmp/oldest-XXXXXX` and clone into it:

```bash
git clone --bare --filter=blob:none --quiet <url> <tmpdir>/repo.git
```

`--filter=blob:none` downloads commits and trees but no file contents, which is all this needs, so even a large repository clones quickly. Use `<tmpdir>/repo.git` as `$REPO`. If the clone fails (private repo, typo, no network), report the git error and stop.

Then check `git -C $REPO rev-parse --is-shallow-repository`. If it prints `true`, the history is truncated and every date is a lower bound — keep going, but say so at the top of the report.

## Step 2: Find each file's last change

Run this exactly, substituting `$REPO` and `$COUNT`:

```bash
{ git -C "$REPO" -c core.quotePath=false ls-tree -r --name-only HEAD; echo '@@END'; git -C "$REPO" -c core.quotePath=false log --no-renames --format='@%cs %h' --name-only HEAD; } | awk '
!end { if ($0 == "@@END") end = 1; else tracked[$0] = 1; next }
/^@/ { split(substr($0, 2), a, " "); d = a[1]; h = a[2]; next }
$0 != "" && ($0 in tracked) && !($0 in seen) { seen[$0] = 1; print d " " h " " $0 }
' | sort -k1,1 -k3 | head -n "$COUNT"
```

How it works: `ls-tree` lists every file at `HEAD` (it reads the tree, so it works on a bare clone with no checkout). `git log` walks history newest-first; the first commit in which a file appears is its most recent change, and `awk` keeps only that one line per file. Sorting by date puts the longest-untouched files first. Files deleted before `HEAD` are ignored because they are not in the `ls-tree` list.

Notes on what counts as a "change":

- Dates are **committer dates** (`%cs`), i.e. when the change landed on the branch, not when it was first written.
- `--no-renames` treats a rename or move as a change to the new path. This keeps the walk from fetching file contents in a blobless clone, and a file that was moved has, in a real sense, been touched.
- Merge commits contribute no file lists (git's default), so a file changed only by a conflict resolution in a merge is dated by the last ordinary commit that touched it.

If the listed files all share one date, run the same pipeline again with `| awk 'NR == 1 { d = $1 } $1 == d { n++ } END { print n }'` in place of `| head -n "$COUNT"`, to learn how many files are tied at that oldest date.

## Step 3: Report

Lead with the single oldest file and how long ago it changed (compute the age from today's date, in years and months or days). Then a table:

| Last changed | Age | Commit | File |
| --- | --- | --- | --- |

If many files tie on the oldest commit (common: the initial import), say how many files share that date and commit rather than implying the first one listed is uniquely oldest. Mention the shallow-clone caveat if it applied.

## Step 4: Clean up

If you cloned into a temporary directory, remove it with `rm -rf /tmp/oldest-XXXXXX` (the exact directory `mktemp` returned) whether or not the earlier steps succeeded.
