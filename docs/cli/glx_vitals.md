---
editLink: false
---

## glx vitals

Show vital records for a person

### Synopsis

Display vital records for a person in the archive.

Shows: Name, Sex, Birth, Christening, Death, Burial.
Use "glx summary" for a complete view including all life events.

The person argument can be an exact entity ID (e.g., person-robert-webb) or
a name to search for (e.g., "Jane Miller"). If the name matches multiple
persons, all matches are listed for disambiguation.

```
glx vitals <person> [flags]
```

### Examples

```
  # Look up by person ID
  glx vitals person-robert-webb

  # Look up by name
  glx vitals "Jane Miller"

  # Specify archive path
  glx vitals "Jane Miller" --archive my-archive
```

### Options

```
  -a, --archive string   Archive path (directory or single file) (default ".")
  -h, --help             help for vitals
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

