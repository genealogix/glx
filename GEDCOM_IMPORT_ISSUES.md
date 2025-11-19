# GEDCOM to GLX Import - Information Loss Analysis

**Date**: 2025-11-19
**Source**: Shakespeare family GEDCOM (31 persons, 77 events)
**Target**: sksp.glx

## Summary

The current GEDCOM importer loses several categories of information during conversion. This document catalogs each issue with examples and proposed fixes.

---

## Issue 1: Date Qualifiers Lost ✅ RESOLVED

### Problem
GEDCOM date qualifiers (BEF, AFT, ABT, EST, CAL) were stripped during parsing, losing important context about date certainty.

### Resolution
**Fixed on 2025-11-19**: Date qualifiers are now preserved in GLX keyword format as specified in `specification/6-data-types.md`. The `parseGEDCOMDate` function returns strings like `"ABT 1850"`, `"BEF 1920-01-15"`, `"BET 1880 AND 1890"`, matching the GLX specification exactly.

### Examples from Shakespeare GEDCOM (NOW CORRECT)

| GEDCOM Line | Original Date | Converted GLX | Status |
|------------|---------------|---------------|--------|
| Line 19 (William) | `BEF 23 APR 1564` | `BEF 1564-04-23` | ✅ Preserved |
| Line 35 (Mary) | `ABT 1537` | `ABT 1537` | ✅ Preserved |
| Line 46 (John) | `ABT 1531` | `ABT 1531` | ✅ Preserved |
| Line 129 (Joan) | `AFT SEP 1558` | `AFT 1558-09` | ✅ Preserved |
| Line 201 (Richard) | `ABT 1490` | `ABT 1490` | ✅ Preserved |
| Line 203 (Richard) | `BEF 10 FEB 1561` | `BEF 1561-02-10` | ✅ Preserved |

**Total occurrences**: 8 (all now correctly preserved)

### Implementation
`lib/gedcom_date.go:30-85` - The `parseGEDCOMDate` function now:
- Preserves qualifiers (ABT, BEF, AFT, CAL) as keywords: `"ABT 1850"`
- Handles date ranges: `"BET 1880 AND 1890"`, `"FROM 1850 TO 1860"`
- Returns strings in GLX format matching `specification/6-data-types.md`
- Converts GEDCOM month names (JAN, FEB, etc.) to ISO 8601 format

### Changes Made
1. Updated `Event.Date` field in `types.go` from `any` to `string`
2. Removed unused `EventDate` struct
3. Modified `parseGEDCOMDate` to return GLX-compliant string format
4. Updated all tests to verify correct keyword preservation

---

## Issue 2: Invalid Place Names (HIGH PRIORITY)

### Problem
Non-geographic text in PLAC fields is being treated as real places.

### Examples

| Place ID | Name | Type | Actual Meaning |
|----------|------|------|----------------|
| place-4 | "Died in childbirth" | city | Death cause/note |
| place-5 | "Unmarried" | city | Relationship status |

### GEDCOM Source
```
0 @I0029@ INDI
1 NAME Margaret /Wheeler/
1 DEAT
2 PLAC Died in childbirth   <-- This is NOT a place!

0 @F011@ FAM
1 MARR
2 PLAC Unmarried   <-- This is NOT a place!
```

### Current Code
`lib/gedcom_place.go` - Accepts any PLAC value as a valid place

### Proposed Fix
1. Add validation to detect non-geographic place names
2. Common patterns to detect:
   - "Died in..."
   - "Unmarried"
   - "Unknown"
   - "At sea"
3. For invalid places:
   - Store in `notes` or `description` field of event
   - Add to event properties: `cause: "Died in childbirth"`
   - Do NOT create place entity

### Impact
- **Data Quality**: Pollutes place database with invalid entries
- **Validation**: Makes it impossible to validate place hierarchies
- **User Experience**: Confusing to see "Died in childbirth" as a city

---

## Issue 3: TITL Property Storage (MEDIUM PRIORITY)

### Problem
GEDCOM TITL (title/honorific) stored as lowercase `titl` instead of `title`.

### Examples

| Person ID | GEDCOM | GLX Property | Should Be |
|-----------|--------|--------------|-----------|
| person-20 (John Hall) | `1 TITL Dr.` | `titl: Dr.` | `title: Dr.` |
| person-23 (John Barnard) | `1 TITL Sir` | `titl: Sir` | `title: Sir` |

**Total occurrences**: 2

### Current Code
`lib/gedcom_individual.go` - Stores administrative tags as lowercase properties

### Proposed Fix
1. Add explicit TITL handling in `convertIndividual`
2. Store as `title` not `titl`
3. Consider vocabulary: `title_types` (Dr., Sir, Rev., etc.)

### Impact
- **Consistency**: Property names should be clear and consistent
- **Standards**: "title" is more standard than "titl"

---

## Issue 4: Missing Christening Properties (MEDIUM PRIORITY)

### Problem
Christening events are created but dates/places are not added to person properties for quick access.

### Statistics
- **Christening events**: 10
- **Persons with `christened_on` property**: 0
- **Persons with `christened_at` property**: 0

### Examples
```yaml
# Current: Only in events
event-2:
  type: christening
  place: place-1
  date: "1564-04-26"
  participants:
    - person: person-1

# Should ALSO be in person properties:
person-1:
  properties:
    christened_on: "1564-04-26"
    christened_at: place-1
```

### Current Code
`lib/gedcom_individual.go:386-402` - Only handles birth/death, not christening

### Proposed Fix
1. Add christening to the same logic as birth/death
2. Set `person.Properties["christened_on"]` and `christened_at`
3. Still create event AND assertion (triple storage like birth/death)

### Impact
- **Consistency**: Birth and death are in properties, why not christening?
- **Accessibility**: Users can access christening date without querying events
- **Common Use**: Christening often used when birth date unknown

---

## Issue 5: Missing Burial Properties (MEDIUM PRIORITY)

### Problem
Burial events are created but dates/places are not added to person properties.

### Statistics
- **Burial events**: 9
- **Persons with `buried_on` property**: 0
- **Persons with `buried_at` property**: 0

### Examples
```yaml
# Current: Only in events
event-8:
  type: burial
  date: "1601-09-08"
  participants:
    - person: person-3

# Should ALSO be in person properties:
person-3:
  properties:
    buried_on: "1601-09-08"
    buried_at: place-x
```

### Current Code
Same as Issue 4

### Proposed Fix
Same as Issue 4 - add burial alongside birth/death/christening

### Impact
- **Cemetery Research**: Burial location is critical for genealogy
- **Consistency**: Should match birth/death pattern

---

## Issue 6: Date Range Support (LOW PRIORITY - FUTURE)

### Observation
GEDCOM supports date ranges:
- `BET date1 AND date2` (between)
- `FROM date1 TO date2` (period)

Current code handles these (lines 40-66 in gedcom_date.go) but converts to ISO 8601 range notation (`date1/date2`).

GLX should formally support date ranges in schema.

---

## Recommended Priority

1. ✅ ~~**CRITICAL**: Fix date qualifiers (Issue 1)~~ - **RESOLVED 2025-11-19**
2. **HIGH**: Fix invalid places (Issue 2) - Data quality corruption
3. **MEDIUM**: Fix TITL storage (Issue 3) - Standards compliance
4. ~~**MEDIUM**: Add christening/burial properties (Issues 4-5)~~ - **NOT NEEDED** (events are sufficient)
5. **LOW**: Formalize date range schema (Issue 6) - Future enhancement

---

## Testing Plan

After fixes:
1. ✅ Re-import Shakespeare GEDCOM
2. ✅ Verify date qualifiers preserved (Issue 1)
3. ✅ All existing tests pass (2.230s, all tests passing)
4. ✅ Date parsing tests pass (TestParseGEDCOMDate, TestValidateDateFormat)
5. **TODO**: Verify no invalid places created (Issue 2)
6. **TODO**: Verify TITL stored as "title" (Issue 3)
7. **TODO**: Add regression tests for remaining issues
