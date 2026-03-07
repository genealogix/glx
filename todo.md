# GLX TODO

> **Note**: Organized by impact. 🔴 = bugs or data loss, 🟡 = meaningful improvements, 🟢 = nice-to-have or open questions.

---

## 🔴 Bugs & Data Integrity

Issues that silently lose or corrupt data during import.

- ~~**Multiple GEDCOM NAME records silently dropped** ([#29](https://github.com/genealogix/glx/issues/29))~~ ✅ Multiple NAMEs now stored as temporal list via `appendNameProperty`.
- ~~**FAM event processing depends on HUSB/WIFE tag order** ([#15](https://github.com/genealogix/glx/issues/15))~~ ✅ Two-pass processing: collect all sub-records first, process events after HUSB/WIFE extraction.
- ~~**GEDCOM marriage/divorce events stored as properties instead of `start_event`/`end_event`**~~ ✅ Now uses `relationship.StartEvent`/`EndEvent` instead of properties.
- ~~**Census NOTE discarded when SOUR exists** ([#30](https://github.com/genealogix/glx/issues/30))~~ ✅ NOTE text now appended to existing citations when SOUR sub-records exist.

---

## 🟡 Import Gaps

Data that is parsed but silently dropped or not stored.

- **HEAD Metadata** ([gedcom_converter.go:220-221](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_converter.go#L220-L221)): Store export_date, source_file, copyright, language, source_system.
- **SUBM Metadata** ([gedcom_converter.go:246-247](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_converter.go#L246-L247)): Store submitter information.
- **NCHI Tag** ([gedcom_family.go](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_family.go)): Store number of children — can differ from actual CHIL count.
- **LANG Tag Normalization**: GEDCOM 5.5.x uses free-form text (`English`) while 7.0 uses ISO format (`en-US`). Should normalize on import.
- **Media entities with empty URI**: GEDCOM OBJE records with empty `FILE` values produce media entities with empty `uri`. Should skip, populate from extension tags, or set a placeholder.
- **PLAC Validation** ([gedcom_place.go](https://github.com/genealogix/glx/blob/main/go-glx/gedcom_place.go)): Reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown".

---

## 🟡 Data Model & Design

Design decisions to resolve before 1.0.

- **Source `description` GEDCOM mapping ambiguity**: Maps to both SOUR.TEXT (excerpt from original) and SOUR.NOTE (researcher commentary). These are semantically different. Consider splitting or documenting the merge.
- ~~**Event participant requirement**~~ ✅ Decision: keep participants required (min 1). GLX is genealogical — every event connects to at least one person. Documented in spec.
- ~~**Repository address fields**~~ ✅ Decision: keep as direct entity fields. Core identifying attributes, not extensible metadata. Documented in spec.
- **Fields-only structured properties**: Spec allows `fields` without `value` (e.g., crop coordinates). Ensure validator doesn't warn on this.
- ~~**Gender/sex controlled vocabularies**~~ ✅ Decision: keep as free-form string. Terminology varies across periods/cultures; recommended values documented in spec, not enforced.
- ~~**Participant role requirement**~~ ✅ Decision: roles remain optional on participants (schema does not require `role`, only `person`). This accommodates cases where the role is unknown.

---

## 🟡 Tooling & Infrastructure

- **Markdown link validation in CI**: Validate all internal links in specification and documentation.
- **Vocabulary audit**: Review all standard vocabulary files in [5-standard-vocabularies/](specification/5-standard-vocabularies/) for consistency and completeness.
- ~~**Add make command for goreleaser**~~ ✅ Added `make release-snapshot` for local builds; updated release workflow to latest action versions.

---

## 🟢 Code Quality

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
