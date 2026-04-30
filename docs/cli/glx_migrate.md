---
editLink: false
---

## glx migrate

Migrate an archive to the current format

### Synopsis

Converts deprecated person properties (born_on, born_at, died_on, died_at, buried_on, buried_at) to birth/death/burial events.

For each person with deprecated properties:
- Creates a birth, death, or burial event if none exists
- Merges date/place into existing events if fields are empty
- Never overwrites existing event data
- Converts assertions to reference the event instead of the person property

With --rename-gender-to-sex, also renames the legacy `gender` person
property (and any related assertions and inlined vocabulary entries) to
`sex`, completing the two-field-model split introduced in #528.

```
glx migrate [archive] [flags]
```

### Examples

```
  # Migrate a multi-file archive
  glx migrate ./my-archive

  # Migrate a single-file archive
  glx migrate archive.glx

  # Also rename legacy 'gender' person properties to 'sex'
  glx migrate ./my-archive --rename-gender-to-sex
```

### Options

```
  -h, --help                   help for migrate
      --rename-gender-to-sex   Rename the legacy 'gender' person property to 'sex' (two-field-model split, #528)
```

### SEE ALSO

* [glx](/cli/glx)	 - GENEALOGIX CLI - Manage and validate genealogy archives

