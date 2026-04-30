---
editLink: false
---

## glx cite

Generate formatted citation text from structured fields

### Synopsis

Generate a formatted citation string from structured citation data.

Assembles citations from the source title, source type, repository name,
URL, and accessed date already stored in the archive. This eliminates
repetitive manual writing of the citation_text property.

If a citation ID is given, prints that single citation. If no ID is given,
prints all citations in the archive.

```
glx cite [citation-id] [flags]
```

### Examples

```
  # Format a specific citation
  glx cite citation-1860-census-webb-household

  # Format all citations in the archive
  glx cite

  # Use a specific archive
  glx cite --archive my-archive
```

### Options

```
  -a, --archive string   Archive path (directory or single file) (default ".")
  -h, --help             help for cite
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

