---
title: "ADR-0001: Use YAML as the archive file format"
description: Why GLX stores genealogical data in YAML rather than JSON, TOML, or a binary format.
layout: doc
---

# ADR-0001: Use YAML as the archive file format

## Status

Accepted

## Context

GLX is a *permanent, human-readable archive format* for genealogical research. The core design goal is that an archive should outlive any specific software application — a researcher should be able to open a GLX file in a plain text editor decades from now and still understand it, without running GLX tooling at all.

That goal rules out binary formats (SQLite, protobuf, custom binary) outright: a binary blob cannot be read by a human and depends on a compatible reader existing forever. It leaves three realistic text-based candidates for the on-disk format:

- **YAML** — indentation-based, widely used for configuration, permissive to comments.
- **JSON** — brace/quote-heavy, strict, ubiquitous in software but punishing to hand-edit.
- **TOML** — key/value-oriented, works well for flat configs but awkward for deeply nested data.

Genealogical records are deeply nested (a person has events, events have participants, participants have roles and notes, and so on). Researchers are the primary editors and many of them are not software developers.

## Decision

Store GLX archives as YAML files with the `.glx` extension (or `.yaml`/`.yml` where appropriate — see the specification for the full rules on file extensions).

As stated in the [Introduction](/specification/1-introduction):

> YAML is a standard text format used across the software industry. GLX chose it for a simple reason: *you can read it*. Indentation shows structure, colons separate labels from values, and there are no cryptic tags or angle brackets to learn.

## Consequences

**Positive**

- A genealogist who has never seen GLX can read a `.glx` file and figure out what it means. That is the single most important property of the format.
- Comments are first-class (`# like this`), so researchers can leave notes alongside their data — something JSON and binary formats cannot offer at all.
- YAML has mature libraries in every mainstream language, keeping the specification implementable outside Go.
- Plain text plays well with Git — see [ADR-0004](0004-git-native-archives).

**Negative**

- YAML parsing is slower than JSON, and libraries vary in spec compliance.
- Indentation sensitivity means a stray tab or misaligned space can change meaning. Tooling (the `glx validate` command) is the mitigation.
- YAML is complex: tags, anchors, and multi-document streams exist in the spec but are rarely needed for GLX. The specification restricts usage to a simple subset; see the JSON schemas in `specification/schema/` for the canonical shape.
- The format is slightly more verbose on the wire than JSON (negligible in practice; archives are small compared to media).
