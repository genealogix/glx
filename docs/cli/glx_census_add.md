---
editLink: false
---

## glx census add

Import a census template into the archive

### Synopsis

Generate GLX entities from a structured census template file.

Reads a YAML census template and generates:
- Person records (new or matched to existing)
- A census event with participants
- A source (new or matched to existing)
- A citation with locator, URL, transcription
- Assertions for birth year, birthplace, sex, occupation, residence

The template format uses a simple YAML structure describing the census
year, location, household members, and citation details. Members can
reference existing persons by ID or by name (matched against the archive).

Use --dry-run to preview what would be generated without writing files.

```
glx census add [flags]
```

### Examples

```
  # Import a census template
  glx census add --from 1860-census-lane.yaml --archive my-archive

  # Preview without writing
  glx census add --from 1860-census-lane.yaml --archive my-archive --dry-run

  # Verbose output
  glx census add --from 1860-census-lane.yaml --archive my-archive --verbose
```

### Options

```
  -a, --archive string   Archive path (directory) (default ".")
      --dry-run          Preview generated entities without writing files
      --from string      Path to census template YAML file (required)
  -h, --help             help for add
  -v, --verbose          Show detailed summary of generated entities
```

### SEE ALSO

* [glx census](/cli/glx_census)	 - Bulk census record tools

