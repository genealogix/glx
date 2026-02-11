# GLX TODO

> **Note**: Organized by impact. 🔴 = bugs or data loss that should be fixed before release, 🟡 = meaningful improvements, 🟢 = nice-to-have or open questions.

---

## 🔴 Bugs & Data Integrity

Issues that silently lose or corrupt data during import.

- **Residence property overwrite on PLAC-without-DATE**: In `convertResidence` and `convertCensus`, when PLAC exists but DATE does not, the residence property is set as a bare place ID which overwrites any existing temporal residence list. Should append instead of overwrite. Affects [gedcom_individual.go](glx/lib/gedcom_individual.go) `convertResidence` (line ~454) and `applyCensusData`.
- **FAM CENS/event processing depends on HUSB/WIFE tag order**: In `convertFamily`, CENS and other family events (ENGA, MARB, etc.) are processed inline during the same loop that extracts `husbandID` and `wifeID`. If these event tags appear before HUSB/WIFE in the GEDCOM file, spouse IDs will be empty strings. GEDCOM does not guarantee tag order. Marriage/divorce handle this by deferring to after the loop, but CENS and other events do not. [gedcom_family.go:64-74](glx/lib/gedcom_family.go#L64-L74)
- **`LoadStandardVocabulariesIntoGLX` silently swallows YAML parse errors**: If a vocabulary file has a parse error, it is silently ignored and that vocabulary will be nil. This could cause subtle validation failures downstream where the validator thinks there are no valid vocabulary entries. Should return or log the error. [vocabularies.go:48-55](glx/lib/vocabularies.go#L48-L55)
- **`appendMediaID` unchecked type assertion**: Uses `props[PropertyMedia].([]string)` which panics if the value isn't `[]string` (e.g., from YAML deserialization producing `[]any`). Should use comma-ok assertion. [gedcom_media.go:127](glx/lib/gedcom_media.go#L127)
- **Migration guide uses stale assertion format**: Uses old format with `subject` as plain string and `claim` instead of `property`. Will mislead users about the correct assertion format. [migration-from-gedcom.md:101-106](docs/guides/migration-from-gedcom.md#L101-L106)

---

## 🟡 Import Gaps

Data that is parsed but silently dropped or not stored.

- **Census NOTE discarded when SOUR exists**: In `convertCensus`, NOTE text from CENS records is only attached to synthetic citations. When SOUR sub-records exist, the NOTE is silently discarded. Should store on the person or pass through to citations.
- **HEAD Metadata** ([gedcom_converter.go:220-221](glx/lib/gedcom_converter.go#L220-L221)): Store HEAD metadata (export_date, source_file, copyright, language, source_system).
- **SUBM Metadata** ([gedcom_converter.go:246-247](glx/lib/gedcom_converter.go#L246-L247)): Store SUBM (submitter) metadata.
- **NCHI Tag** ([gedcom_family.go](glx/lib/gedcom_family.go)): Store NCHI (number of children) — can differ from actual CHIL count.
- **NAME TYPE** ([gedcom_name.go](glx/lib/gedcom_name.go)): Store NAME TYPE subfield (birth, married, aka).
- **LANG Tag Normalization**: GEDCOM 5.5.x uses free-form text (`English`, `French`) while 7.0 uses ISO format (`en-US`). Should normalize 5.5.x values to ISO codes on import.
- **PLAC Validation** ([gedcom_place.go](glx/lib/gedcom_place.go)): Reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown".

---

## 🟡 Data Model & Design

Design decisions that affect the spec and should be resolved before 1.0.

- **Source `description` GEDCOM mapping ambiguity**: The `description` field on Source maps to both GEDCOM SOUR.TEXT (text from source) and SOUR.NOTE (general note). These are semantically different — TEXT is an excerpt from the original, NOTE is researcher commentary. Consider splitting into separate fields or documenting the merge.
- **Event participant requirement**: Consider relaxing — historical events (wars, famines, natural disasters) may be relevant to genealogy without specific participants.
- **Gender/sex controlled vocabularies**: Should these be formalized?
- **Property field data types**: Should property fields carry type information?
- **Repository address fields**: Consider moving `address`, `city`, `state_province`, `postal_code`, `country` from direct entity fields into the `repository_properties` vocabulary for consistency with other entity types.

---

## 🟡 Tooling & Infrastructure

- **Markdown link validation in CI**: Add CI check to validate all internal markdown links in specification and documentation files.
- **Review standard vocabularies**: Audit all standard vocabulary files (.glx) in [5-standard-vocabularies/](specification/5-standard-vocabularies/) for consistency and completeness.
- **Schema embed.go missing individual variables**: [specification/schema/v1/embed.go](specification/schema/v1/embed.go) is missing individual `[]byte` variables for `CitationPropertiesSchema` and `SourcePropertiesSchema`. The JSON files exist and are captured by the glob FS embed, but the individual variables break the pattern established for all other vocabulary schemas.

---

## 🟢 Documentation

- **Add validation rule sections**: Each entity type doc should include a consolidated "Validation Rules" section.
- **Git Workflow Guide**: Create documentation covering Git workflows, branching strategies, collaboration patterns, and branch-based research methodologies for GLX archives.
- **Broken links in migration-from-gedcom.md**: Two references to deleted `../development/gedcom-import.md` at lines 63 and 265. [migration-from-gedcom.md](docs/guides/migration-from-gedcom.md)
- **Duplicate `media/` directory in file storage diagram**: The media spec shows two separate `media/` directories instead of a single tree with both entity `.glx` files and a `files/` subdirectory. [media.md:337-351](specification/4-entity-types/media.md#L337-L351)
- **Stale conversion flow comment in doc.go**: Describes old single-pass approach but actual flow is now dependency-ordered (Notes/Repos → Sources/Media → Individuals → Families). [doc.go:14](glx/lib/doc.go#L14)

---

## 🟢 Code Quality

- **Require participant roles**: Should events, relationships, and assertions require participant roles?
- **Add validator tags to GLX structs**: Use struct tags for validation.
- **Move Loggers to their own package**: Better separation of concerns.
- **Add make command for goreleaser**.
- **`copyFile` discards `dstFile.Close()` error**: On NFS/network mounts, write errors surface at `Close()`. Should check the close error for the destination file. [media_copy.go:119-132](glx/media_copy.go#L119-L132)
- **`decodeGEDCOMBlob` no input validation on byte range**: Characters outside valid range (0x2E-0x6D) produce garbage silently rather than returning an error. BLOB is deprecated and rare. [media_copy.go:143-155](glx/media_copy.go#L143-L155)
- **`extensionFromMimeType` non-deterministic**: Iterates a map, so MIME types with multiple extensions (`.jpg`/`.jpeg`) return random results. [gedcom_media.go:311-318](glx/lib/gedcom_media.go#L311-L318)
- **No path traversal check on GEDCOM FILE references**: `../../etc/passwd` would be resolved by `filepath.Join`. Impact limited since destination uses basename only and user provides the GEDCOM file. [media_copy.go:102-114](glx/media_copy.go#L102-L114)
- **Relationship schema `additionalProperties: true` on participants**: Inconsistent with assertion schema which uses `additionalProperties: false` on its participant object. [relationship.schema.json:41](specification/schema/v1/relationship.schema.json#L41)

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker
