# GLX TODO

> **Note**: Organized by impact. 🔴 = bugs or data loss, 🟡 = meaningful improvements, 🟢 = nice-to-have or open questions.

---

## 🔴 Roundtrip Fidelity Gaps

Known data loss during GED → GLX → GED roundtrip. Steps to reproduce any: `glx import FILE.ged -o rt.glx && glx export rt.glx -o rt.ged`, then compare tag counts.

### Family Reconstruction

- **Extra FAMS references** (queen FAMS +124→-124, british-royalty FAMS +22): Synthetic single-parent families created during reconstruction generate FAMS references that didn't exist in the original.

### Notes

- **Multi-NOTE merging on import** (queen NOTE -7, habsburg NOTE -76): Multiple NOTE records on a person/family are concatenated into one `Notes` string. On export, they become a single NOTE with CONT lines instead of separate NOTE records. Content is preserved but record boundaries are lost. Would require changing Notes from `string` to `[]string`.

### Model Limitations

- **RESI with no PLAC lost** (bullinger RESI 26→19): RESI records with only DATE/TYPE/value but no PLAC sub-record cannot be stored in the place-reference residence model. 7 such records in bullinger are bare markers like `RESI Y` or `RESI\n2 TYPE married`.
- **FACT tag not imported** (bullinger -7): GEDCOM FACT records (e.g., `FACT P908 / TYPE Merged Gramps ID`) are not imported. Most are application-specific metadata (Gramps merge IDs, alias markers), not genealogical facts.

### Minor / By Design

- **Minor SOUR surplus** (bullinger +7, habsburg -108): Small SOUR count differences from assertions on dropped structures (e.g., RESI without PLAC) or citation deduplication during import.
- **CONT line wrapping differences** (queen -167): Multiline text roundtrips with different CONT/CONC splitting than the original. Content is preserved, formatting differs.
- **CHAN dates not exported**: GEDCOM CHAN (change timestamp) records aren't preserved through GLX. Design decision — change metadata, not genealogical data.
- **Extension tags dropped**: Proprietary tags (_MSTAT, _UID, _NSTY, _MARNM, _FREL, _MREL, etc.) are silently dropped on import. By design — these are application-specific.
- **HEAD DEST/SOUR.CORP not roundtripped**: DEST (receiving system) not imported. SOUR.CORP hardcoded to GLX on export (original source system info preserved in ImportMetadata but not re-emitted in HEAD).

## 🟡 Import Gaps

Data that is parsed but silently dropped or not stored.

- **NCHI Tag** ([gedcom_family.go](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_family.go)): Store number of children — can differ from actual CHIL count.
- **LANG Tag Normalization**: GEDCOM 5.5.x uses free-form text (`English`) while 7.0 uses ISO format (`en-US`). Should normalize on import.
- **Media entities with empty URI**: GEDCOM OBJE records with empty `FILE` values produce media entities with empty `uri`. Should skip, populate from extension tags, or set a placeholder.
- **PLAC Validation** ([gedcom_place.go](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_place.go)): Reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown".

---

## 🟡 Data Model & Design

Design decisions to resolve before 1.0.

- **Source `description` GEDCOM mapping ambiguity**: Maps to both SOUR.TEXT (excerpt from original) and SOUR.NOTE (researcher commentary). These are semantically different. Consider splitting or documenting the merge.
- **Fields-only structured properties**: Spec allows `fields` without `value` (e.g., crop coordinates). Ensure validator doesn't warn on this.
- **Property `value_type` for structured-only properties**: Properties like `crop` that only use `fields` (no single value) currently require `value_type: integer`, which is misleading. Consider adding an `object` or `nil` value type, or relaxing the "exactly one of `value_type`/`reference_type`" requirement for fields-only properties.

---

## 🟡 Tooling & Infrastructure

- **Vocabulary audit**: Review all standard vocabulary files in [5-standard-vocabularies/](specification/5-standard-vocabularies/) for consistency and completeness.
- **Validate participant role `applies_to`**: Roles with `applies_to: [event]` should error when used on relationships, and vice versa. Currently `applies_to` is deserialized but never checked.

---

## 🟢 Code Quality

- **Move command logic from `glx/` CLI into `go-glx/` library**: Newer commands (`places_runner.go`, `query_runner.go`, `report_runner.go`, `stats_runner.go`) have non-I/O logic in the CLI package that should live in the library. CLI should only handle I/O and flag parsing; all data transformation, analysis, and formatting logic belongs in `go-glx/`.
- **Migrate from `gopkg.in/yaml.v3` to `github.com/goccy/go-yaml`**: `gopkg.in/yaml.v3` is archived and no longer maintained. Used across 14 files in `go-glx/` and `glx/`.
- **`copyFile` discards `dstFile.Close()` error**: Write errors on NFS/network mounts surface at `Close()`. [media_copy.go:119-132](https://github.com/genealogix/glx/blob/main/glx/media_copy.go#L119-L132)
- **`decodeGEDCOMBlob` no input validation**: Characters outside valid range produce garbage silently. BLOB is deprecated and rare. [media_copy.go:143-155](https://github.com/genealogix/glx/blob/main/glx/media_copy.go#L143-L155)
- **No path traversal check on GEDCOM FILE references**: Impact limited since destination uses basename only. [media_copy.go:102-114](https://github.com/genealogix/glx/blob/main/glx/media_copy.go#L102-L114)
- **Add validator tags to GLX structs**: Use struct tags for validation.
- **Move Loggers to their own package**: Better separation of concerns.

---

## 🟢 Documentation

- **Add validation rule sections**: Each entity type doc should include a consolidated "Validation Rules" section.
- **Git Workflow Guide**: Document branching strategies, collaboration patterns, and branch-based research methodologies for GLX archives.
- **Place schema: enforce latitude/longitude co-dependency**: Schema allows `latitude` without `longitude` (and vice versa). Add a `dependencies` clause so both must be present together.
- **Validate media `hash` format**: Media `hash` field should follow `algorithm:hexstring` format (e.g., `sha256:abc123...`). Currently not validated.

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker
