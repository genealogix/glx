---
editLink: false
---

## glx split

Split a single-file GLX archive into multi-file format

### Synopsis

Split a single-file GLX archive into a multi-file directory structure.

The multi-file format organizes entities into separate directories:
- persons/ - One file per person (person-{id}.glx)
- events/ - One file per event (event-{id}.glx)
- relationships/ - One file per relationship (relationship-{id}.glx)
- places/ - One file per place (place-{id}.glx)
- sources/ - One file per source (source-{id}.glx)
- citations/ - One file per citation (citation-{id}.glx)
- repositories/ - One file per repository (repository-{id}.glx)
- media/ - One file per media object (media-{id}.glx)
- assertions/ - One file per assertion (assertion-{id}.glx)
- vocabularies/ - Standard vocabulary definitions

Each entity file uses standard GLX structure with the entity ID as the map key.

```
glx split <input-file> <output-directory> [flags]
```

### Examples

```
  # Split an archive
  glx split family.glx family-archive

  # Split without validation
  glx split family.glx family-archive --no-validate
```

### Options

```
  -h, --help                    help for split
      --no-validate             Skip validation before splitting
      --show-first-errors int   Number of validation errors to show (0 for all) (default 10)
  -v, --verbose                 Verbose output
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

