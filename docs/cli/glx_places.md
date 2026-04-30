---
editLink: false
---

## glx places

Analyze places for ambiguity and completeness

### Synopsis

Analyze places in a GENEALOGIX archive for data quality issues.

Reports:
- Duplicate names: places that share the same name (ambiguous without context)
- Missing coordinates: places without latitude/longitude
- Missing type: places without a type classification
- No parent: non-country/region places missing a parent (hierarchy gap)
- Dangling parent: places referencing a parent that doesn't exist in the archive
- Unreferenced: places not used by any event, assertion, or as a parent

Each place is shown with its full canonical hierarchy path.
If no path is given, uses the current directory.

```
glx places [path] [flags]
```

### Examples

```
  # Analyze places in current directory
  glx places

  # Analyze places in a specific archive
  glx places my-family-archive

  # Analyze a single-file archive
  glx places family.glx
```

### Options

```
  -h, --help   help for places
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

