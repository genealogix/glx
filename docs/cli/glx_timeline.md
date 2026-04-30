---
editLink: false
---

## glx timeline

Show chronological timeline of events for a person

### Synopsis

Display a chronological timeline of all events in a person's life.

Shows direct events (where the person is a participant) and family events
(spouse births/deaths, children's births/deaths, parent deaths) discovered
through relationship traversal.

The person argument can be an exact entity ID (e.g., person-john-smith)
or a name to search for (e.g., "John Smith"). If the name matches
multiple persons, all matches are listed for disambiguation.

Use --no-family to exclude family events and show only direct events.

```
glx timeline <person> [flags]
```

### Examples

```
  # Timeline by person ID
  glx timeline person-john-smith

  # Timeline by name
  glx timeline "John Smith"

  # Direct events only (no family events)
  glx timeline "John Smith" --no-family

  # Specify archive path
  glx timeline "John Smith" --archive my-archive
```

### Options

```
  -a, --archive string   Archive path (directory or single file) (default ".")
  -h, --help             help for timeline
      --no-family        Exclude family events (show only direct events)
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

