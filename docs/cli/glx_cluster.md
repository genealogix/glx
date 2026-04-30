---
editLink: false
---

## glx cluster

FAN club analysis — find associates of a person

### Synopsis

Identify associates of a person using FAN (Friends, Associates, Neighbors)
club analysis — the primary methodology for breaking genealogical brickwalls.

Cross-references the archive to find people connected to the target through:
- Census households: people enumerated in the same census events
- Shared events: co-participants in marriages, baptisms, land records, etc.
- Place overlap: people associated with the same places in the same time period

Associates are ranked by connection strength: census household links (3 points),
shared event links (2 points), and place overlap links (1 point). Multiple
connections compound for higher scores.

The person argument can be an exact entity ID (e.g., person-d-lane) or a
name to search for (e.g., "Mary Green"). If the name matches multiple
persons, all matches are listed for disambiguation.

```
glx cluster <person> [flags]
```

### Examples

```
  # Show all associates
  glx cluster person-mary-lane

  # Filter to a specific place
  glx cluster person-mary-lane --place place-ironton-sauk-wi

  # Filter to a time range
  glx cluster person-mary-lane --before 1860 --after 1840

  # JSON output for downstream tooling
  glx cluster person-mary-lane --json

  # Use a specific archive
  glx cluster "Mary Green" --archive my-archive
```

### Options

```
      --after int        Only consider dated events after this year (undated events are still included)
  -a, --archive string   Archive path (directory or single file) (default ".")
      --before int       Only consider dated events before this year (undated events are still included)
  -h, --help             help for cluster
      --json             Output as JSON
      --place string     Filter by place ID (includes descendant places)
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

