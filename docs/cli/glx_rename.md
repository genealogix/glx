---
editLink: false
---

## glx rename

Rename an entity ID and update all references

### Synopsis

Rename an entity ID throughout the archive, atomically updating all
cross-references in events, relationships, assertions, citations, and other entities.

Works with any entity type: persons, events, relationships, places, sources,
citations, repositories, assertions, and media.

```
glx rename <old-id> <new-id> [flags]
```

### Examples

```
  # Rename a person
  glx rename person-a3f8d2c1 person-jane-miller --archive ./archive

  # Rename a place
  glx rename place-b7e2f1a0 place-millbrook-hartford --archive ./archive

  # Preview changes without writing
  glx rename person-old person-new --archive ./archive --dry-run
```

### Options

```
  -a, --archive string   Path to GLX archive (default ".")
      --dry-run          Show what would change without writing
  -h, --help             help for rename
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

