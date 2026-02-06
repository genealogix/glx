# GLX TODO

> **Note**: Todos are organized by category and priority. Items marked with 🔴 are high priority, 🟡 are medium priority, and 🟢 are low priority or nice-to-have.

---

## 📥 GEDCOM Import

### Critical Data Loss

- 🔴 **Media/OBJE Import**: Only 2 of 32 multimedia references imported in torture test (94% loss). Inline OBJE tags in events, URL-type multimedia, and BLOB data are not imported.

### Missing Data Storage

**Issue**: Data is being processed but not stored/exposed after import

- 🟡 **Extension Tags** ([gedcom_converter.go:102-103](glx/lib/gedcom_converter.go#L102-L103)): Store extension tag data (tags starting with `_`) - vendor-specific metadata like _MSTAT, _UID, _NSTY
- 🟢 **HEAD Metadata** ([gedcom_converter.go:220-221](glx/lib/gedcom_converter.go#L220-L221)): Store HEAD metadata (export_date, source_file, copyright, language, source_system)
- 🟢 **SUBM Metadata** ([gedcom_converter.go:246-247](glx/lib/gedcom_converter.go#L246-L247)): Store SUBM (submitter) metadata
- 🟢 **NCHI Tag** ([gedcom_family.go](glx/lib/gedcom_family.go)): Store NCHI (number of children) - can differ from actual CHIL count
- 🟢 **NAME TYPE** ([gedcom_name.go](glx/lib/gedcom_name.go)): Store NAME TYPE subfield (birth, married, aka)

### Data Quality

- 🟡 **Residence property overwrite on PLAC-without-DATE**: In `convertResidence` and `convertCensus`, when PLAC exists but DATE does not, the residence property is set as a bare place ID which overwrites any existing temporal residence list. Should append instead of overwrite. Affects [gedcom_individual.go](glx/lib/gedcom_individual.go) `convertResidence` (line ~454) and `applyCensusData`.
- 🟢 **Census NOTE discarded when SOUR exists**: In `convertCensus`, NOTE text from CENS records is only attached to synthetic citations. When SOUR sub-records exist, the NOTE is silently discarded. Should store on the person or pass through to citations.
- 🟢 **LANG Tag Normalization**: Normalize language tags on import. GEDCOM 7.0 uses ISO format (e.g., `en-US`), but GEDCOM 5.5.x uses free-form text (e.g., `English`, `French`). Should convert 5.5.x values to ISO codes for consistency.
- 🟢 **PLAC Validation** ([gedcom_place.go](glx/lib/gedcom_place.go)): Validate PLAC fields - reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown"
- 🟢 **Place Type Vocab**: Fix places[place-31].Type references non-existent place_types: locality

---

## ✅ GLX Validation
- 🟢 Require participant roles in events, relationships, assertions?
- 🟢 Add validator tags to GLX structs

---

## 🏗️ Type System & Data Model

### Vocabulary & Mapping

- 🟢 Gender/sex controlled vocabularies?
- 🟢 Should property fields have data types?

### Evidence & Assertions

- 🟡 Consider adding `media` as a third evidence option for assertions (alongside `citations` and `sources`) - useful for direct visual evidence like gravestone photos. More broadly, solidify the purpose of media entities as more than just window dressing or additional non-critical data.
- 🟢 **QUAY ratings**: Currently preserved in citation notes as "GEDCOM QUAY: X". Consider mapping to GLX confidence levels or storing as structured property instead.
- 🟢 Consider relaxing event participant requirement - the spec says "At least one participant is required (events without participants are not meaningful)" but historical events (wars, famines, natural disasters) may be relevant to genealogy without specific participants

---

## 🚀 Infrastructure & Deployment

- 🟡 **Markdown link validation in CI**: Add CI check to validate all internal markdown links in specification and documentation files. Catch broken links before merge.
- 🟢 Add make command for goreleaser

---

## 📖 Documentation

### Specification

- 🟡 **Review standard vocabularies**: Audit all standard vocabulary files (.glx) in [5-standard-vocabularies/](specification/5-standard-vocabularies/) to ensure consistency and completeness.
- 🟢 **Add validation rule sections**: Each entity type documentation should include a consolidated "Validation Rules" section summarizing all validation requirements for that entity.

### Guides

- 🟢 **Git Workflow Guide**: Create separate documentation covering Git workflows, branching strategies, collaboration patterns, merge conflict resolution, and branch-based research methodologies for GLX archives

---

## 🧹 Code Organization & Quality

- 🟢 Move Loggers to their own package?

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker
