# GLX-aware Git merge driver

Two researchers working on the same archive in parallel branches will
eventually edit the same `.glx` file — adding a citation here, refining a
date there. Git's default text merge ignores YAML structure, so what should
be a safe additive merge gets flagged as a conflict.

`glx merge-driver` is a [custom git merge
driver](https://git-scm.com/docs/gitattributes#_defining_a_custom_merge_driver)
that performs a structural 3-way merge of GLX YAML, resolving the safe
cases automatically and falling back to git's standard text merge only when
two branches truly disagree.

This driver is invoked by git, not by humans. You wire it up once per
clone via two `git config` lines, then it runs transparently on every
merge that touches a `.glx` file.

## One-time setup

```sh
git config merge.glx.name "GLX-aware merge"
git config merge.glx.driver "glx merge-driver %O %A %B %P"
```

`glx` must be on `PATH`. The repository's `.gitattributes` already maps
`*.glx merge=glx`; without your `git config` lines, git silently falls
back to its default text merge, so this setup is safe to skip on clones
where you don't have `glx` installed.

To revert:

```sh
git config --unset merge.glx.name
git config --unset merge.glx.driver
```

## What gets auto-resolved

Given a common ancestor (base), your branch (ours), and the incoming
branch (theirs):

| Change shape | Resolution |
| --- | --- |
| Only one side modified an entity / property | Take that side's change. |
| Both sides made the same change | Take it; no conflict. |
| Both sides added different entries to an assertion's `citations`, `sources`, `media`, or `notes` | Union them. |
| Both sides changed an assertion's `value` to different things, but one side has a strictly higher `confidence` from the standard `confidence_levels` vocabulary | Take the higher-confidence side. The other side's value is shown on stderr. |
| Both sides modified the same scalar property differently | Conflict — falls back to text merge with standard `<<<<<<<` markers. |
| One side deleted, other side modified | Conflict — falls back to text merge. |

The auto-resolution stops at the file the driver was invoked on. If two
branches change a person's `birthDate` and the confidence for that
property lives in a separate `assertions/` file, the merge driver running
on `persons/person-foo.glx` can't see that context — those land as
conflicts. A cross-file resolver is on the roadmap.

The `confidence_levels` ranking used for auto-resolution comes from the
standard `high > medium > low` order. Custom confidence levels are
treated as unknown and never win the tiebreaker — equal or unknown
confidence on both sides always falls back to text merge, so a custom
vocabulary cannot silently override a researcher's choice.

## What you see when the driver runs

**Clean structural merge** — silent; git reports a successful merge and
your file contains both sides' changes.

**Auto-resolved confidence tiebreaker** — git reports a clean merge, and
on stderr you'll see:

```
[glx merge-driver] file=assertions/assertion-john-birth-date.glx — auto-resolved by the driver:
  assertions[assertion-john-birth-date].value → ours (higher confidence)
    ours   : 1850-04-12
    theirs : 1850-04-15
```

**Fallback to text merge** — your file contains standard `<<<<<<<` markers,
and on stderr you'll see a per-conflict summary listing both values with
any assertion-level evidence available:

```
[glx merge-driver] file=assertions/assertion-john-birth-date.glx
  conflict at assertions[assertion-john-birth-date].value
    ours    : 1850-04-12  conf=medium  cites=[citation-parish-register]
    theirs  : 1850-04-15  conf=medium  cites=[citation-1851-census]
```

Resolve as you would any other text merge, then `git add` and continue.

## Caveats

**YAML comments and key ordering are not preserved.** The structural
merge re-emits the file via `yaml.Marshal`. Before this driver, that
round-trip happened only when you ran a `glx` write command — you knew
you were editing. After enabling this driver, the same normalization
happens automatically on every `git merge` that touches a `.glx` file. If
you keep important comments in your archive YAML, you have three
options:

1. Move the commentary to sibling `.md` notes that aren't merge-driven.
2. Disable the merge driver for selected paths via a more specific
   `.gitattributes` rule (e.g. `commented-files/*.glx -merge`).
3. Skip the one-time `git config` setup until upstream
   comment-preservation lands.

**Cross-file confidence is not yet handled.** As noted above, the driver
runs per file. A conflict on a `Person` property where the confidence
lives in a separate `Assertion` file lands as a text-merge conflict.

**"More evidence" is not a tiebreaker.** Issue
[#293](https://github.com/genealogix/glx/issues/293) mentions "auto-resolve
when one side has higher confidence or more evidence." The current
driver only auto-resolves on strictly higher confidence. Counting
citations or sources as a tiebreaker is intentionally not implemented —
quantity of evidence doesn't compare cleanly across assertions, and we
prefer to surface the conflict to a researcher rather than silently
prefer the side with more refs.

## Troubleshooting

**`error: glx: command not found`** — Add `glx` to your `PATH`, or use an
absolute path in `git config merge.glx.driver`.

**Driver runs but you see no auto-resolution** — Check that the
`.gitattributes` line is in your branch (it ships in this repo's
default branch), and that your `git config merge.glx.driver` is set.
Verify with `git check-attr -a path/to/file.glx`.

**You want to skip the driver on a particular merge** — Pass
`-Xtheirs`, `-Xours`, or `--no-renormalize`, or remove `merge.glx.driver`
from your config temporarily.
