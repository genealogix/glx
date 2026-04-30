---
editLink: false
---

## glx join

Join a multi-file GLX archive into single-file format

### Synopsis

Join a multi-file GLX archive into a single YAML file.

Reads entity files from a multi-file directory structure and combines them
into a single GLX archive file.

The multi-file structure should contain:
- persons/ - Person entity files
- events/ - Event entity files
- relationships/ - Relationship entity files
- places/ - Place entity files
- sources/ - Source entity files
- citations/ - Citation entity files
- repositories/ - Repository entity files
- media/ - Media entity files
- assertions/ - Assertion entity files

Entity IDs are read from the map key in each file.

```
glx join <input-directory> <output-file> [flags]
```

### Examples

```
  # Join an archive
  glx join family-archive family.glx

  # Join without validation
  glx join family-archive family.glx --no-validate
```

### Options

```
  -h, --help                    help for join
      --no-validate             Skip validation before joining
      --show-first-errors int   Number of validation errors to show (0 for all) (default 10)
  -v, --verbose                 Verbose output
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

