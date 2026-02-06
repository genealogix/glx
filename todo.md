# GLX TODO

> **Note**: Organized by impact. 🔴 = bugs or data loss that should be fixed before release, 🟡 = meaningful improvements, 🟢 = nice-to-have or open questions.

---

## 🔴 Bugs & Data Integrity

Issues that silently lose or corrupt data during import.

- **Media/OBJE Import**: Only 2 of 32 multimedia references imported in torture test (94% loss). Inline OBJE tags in events, URL-type multimedia, and BLOB data are not imported.
- **Residence property overwrite on PLAC-without-DATE**: In `convertResidence` and `convertCensus`, when PLAC exists but DATE does not, the residence property is set as a bare place ID which overwrites any existing temporal residence list. Should append instead of overwrite. Affects [gedcom_individual.go](glx/lib/gedcom_individual.go) `convertResidence` (line ~454) and `applyCensusData`.

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

- **QUAY ratings**: Currently preserved in citation notes as "GEDCOM QUAY: X". Consider mapping to GLX confidence levels or storing as structured property.
- **Event participant requirement**: Consider relaxing — historical events (wars, famines, natural disasters) may be relevant to genealogy without specific participants.
- **Gender/sex controlled vocabularies**: Should these be formalized?
- **Property field data types**: Should property fields carry type information?

---

## 🟡 Tooling & Infrastructure

- **Markdown link validation in CI**: Add CI check to validate all internal markdown links in specification and documentation files.
- **Review standard vocabularies**: Audit all standard vocabulary files (.glx) in [5-standard-vocabularies/](specification/5-standard-vocabularies/) for consistency and completeness.

---

## 🟢 Documentation

- **Add validation rule sections**: Each entity type doc should include a consolidated "Validation Rules" section.
- **Git Workflow Guide**: Create documentation covering Git workflows, branching strategies, collaboration patterns, and branch-based research methodologies for GLX archives.

---

## 🟢 Code Quality

- **Require participant roles**: Should events, relationships, and assertions require participant roles?
- **Add validator tags to GLX structs**: Use struct tags for validation.
- **Move Loggers to their own package**: Better separation of concerns.
- **Add make command for goreleaser**.

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker
