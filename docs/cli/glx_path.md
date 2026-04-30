---
editLink: false
---

## glx path

Find the shortest relationship path between two people

### Synopsis

Find and display the shortest relationship path between two persons
in the archive using breadth-first search.

Traverses all relationship types (parent-child, marriage, sibling,
godparent, neighbor, etc.) to find the shortest connection.

Each hop shows the relationship type and the destination person's role.
Use --max-hops to limit search depth (default 10).

Person arguments can be exact entity IDs or name substrings.

```
glx path <person-a> <person-b> [flags]
```

### Examples

```
  # Find path between two persons by ID
  glx path person-mary-lane person-louenza-mortimer

  # Find path by name
  glx path "Mary Lane" "Louenza Mortimer"

  # Limit search depth
  glx path "Mary Lane" "John Smith" --max-hops 5

  # JSON output
  glx path "Mary Lane" "John Smith" --json

  # Specify archive path
  glx path "Mary Lane" "John Smith" --archive my-archive
```

### Options

```
  -a, --archive string   Archive path (directory or single file) (default ".")
  -h, --help             help for path
      --json             Output as JSON
      --max-hops int     Maximum number of hops to search (default 10)
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

