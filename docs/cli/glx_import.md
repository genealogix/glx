---
editLink: false
---

## glx import

Import a GEDCOM file to GLX format

### Synopsis

Import a GEDCOM file and convert it to GLX format.

Supports both GEDCOM 5.5.1 and GEDCOM 7.0 formats.

The imported archive will include:
- All individuals (persons)
- All events (births, deaths, marriages, etc.)
- All relationships (parent-child, spouse, etc.)
- All places with hierarchical structure
- All sources and citations
- All repositories and media
- Evidence-based assertions

Output formats:
- multi: Multi-file directory structure (default, one file per entity)
- single: Single YAML file

```
glx import <gedcom-file> [flags]
```

### Examples

```
  # Import to multi-file directory (default)
  glx import family.ged -o family-archive

  # Import to single file
  glx import family.ged -o family.glx --format single

  # Import without validation
  glx import family.ged -o family-archive --no-validate
```

### Options

```
  -f, --format string           Output format: multi or single (default "multi")
  -h, --help                    help for import
      --no-validate             Skip validation before saving
  -o, --output string           Output file or directory (required)
      --show-first-errors int   Number of validation errors to show (0 for all) (default 10)
  -v, --verbose                 Verbose output
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

