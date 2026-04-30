---
editLink: false
---

## glx stats

Show summary statistics for a GLX archive

### Synopsis

Display a summary dashboard of a GENEALOGIX archive.

Shows entity counts, assertion confidence distribution, and entity coverage
metrics for quick feedback on archive health.

Accepts either a multi-file directory or a single .glx file.
If no path is given, uses the current directory.

```
glx stats [path] [flags]
```

### Examples

```
  # Stats for current directory
  glx stats

  # Stats for a specific archive directory
  glx stats my-family-archive

  # Stats for a single-file archive
  glx stats family.glx
```

### Options

```
  -h, --help   help for stats
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

