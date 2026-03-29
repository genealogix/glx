# Remove born_on/born_at/died_on/died_at Person Properties

**Date**: 2026-03-29
**Status**: Draft
**Type**: Breaking change

## Summary

Remove the four person properties `born_on`, `born_at`, `died_on`, and `died_at` from the GLX specification and codebase. These are redundant with birth/death Event entities, which already carry the same date and place information with richer structure (participants, citations, notes). Events become the single source of truth for birth and death data.

A `glx migrate` CLI command will convert existing archives.

## Motivation

Birth and death information currently exists in two parallel representations:

1. **Person properties** — `born_on` (date string), `born_at` (place ID), `died_on` (date string), `died_at` (place ID)
2. **Event entities** — Full event with type `birth`/`death`, carrying date, place, participants, properties, and notes

The GEDCOM importer populates both from the same source data. CLI tools have fallback logic that checks properties first, then events. This dual representation adds complexity with no benefit — the event representation is strictly more capable.

No other person properties are redundant. Properties like `occupation`, `residence`, `religion`, etc. are temporal attributes of a person, not discrete events, and are correctly modeled as properties.

## Design

### 1. Vocabulary & Schema Changes

**Remove from `specification/5-standard-vocabularies/person-properties.glx`:**
- `born_on` (value_type: date)
- `born_at` (reference_type: places)
- `died_on` (value_type: date)
- `died_at` (reference_type: places)

**Remove from JSON schemas** in `specification/schema/v1/` if referenced.

**Update `specification/4-entity-types/person.md`** to remove documentation of these properties and note that birth/death data lives on Event entities.

### 2. Validation

Currently, unknown properties generate warnings (not errors). After removing these four from the vocabulary, archives using them will get "unknown property" warnings.

**Change**: Upgrade unknown property detection from warning to error for these specific properties, with a clear message directing users to the `glx migrate` command. For example:

```
person[person-1]: property 'born_on' has been removed — use birth/death events instead. Run 'glx migrate' to convert.
```

General unknown properties remain warnings (no change to existing behavior for other properties).

### 3. Constants

**Rename in `go-glx/constants.go`** (not remove — validation needs them):
- `PersonPropertyBornOn` → `DeprecatedPropertyBornOn`
- `PersonPropertyBornAt` → `DeprecatedPropertyBornAt`
- `PersonPropertyDiedOn` → `DeprecatedPropertyDiedOn`
- `PersonPropertyDiedAt` → `DeprecatedPropertyDiedAt`

These are used by the validation banned-property check (section 2) and the migration tool (section 8). No magic strings.

### 4. GEDCOM Importer

**File**: `go-glx/gedcom_individual.go`

Remove the block (~lines 297-311) that sets person properties for birth/death.

**Evidence chain**: The importer currently creates property assertions with `Subject: {Person: personID}` and `Property: "born_on"` etc. These must be converted to event assertions instead — `Subject: {Event: eventID}` pointing at the birth/death event's `date` or `place` property. The same source/citation evidence is preserved, just attached to the event rather than the removed person property.

This is not optional — the evidence chain must not be lost.

### 5. CLI Tool Updates

All CLI tools that currently read `born_on`/`born_at`/`died_on`/`died_at` from person properties must be updated to query Event entities instead. This requires finding events where the person is a participant with a birth/death event type.

**Helper function** (in `go-glx/`): Add a function to find a person's birth/death event from the archive, since events link to persons through the `Participants` array (there's no reverse index):

```go
func FindPersonEvent(archive *GLXFile, personID string, eventType string) *Event
```

Returns the first event of the given type where the person is a **principal/subject participant**. Must filter by participant role — a person who was a witness to someone else's birth is not the subject of that birth event. Returns nil if not found.

**Files to update:**

| File | Current usage | New behavior |
|------|--------------|-------------|
| `glx/timeline_runner.go` | Synthesizes birth/death timeline entries from properties when no event exists | Use `FindPersonEvent` — no fallback needed since events are now the only source |
| `glx/vitals_runner.go` | `formatPropertyDatePlace()` reads born_on/born_at and died_on/died_at | Read date/place directly from birth/death Event |
| `glx/summary_runner.go` | Skips these properties in output (they're handled in vitals section) | Remove skip list entries; vitals section reads from events |
| `glx/analyze_gaps.go` | `checkMissingBirth()`/`checkMissingDeath()` read properties | Check for existence of birth/death events instead |
| `glx/analyze_suggestions.go` | Extracts birth/death year from properties | Extract from event date |
| `glx/query_runner.go` | `--birthplace` filter matches against `born_at` property | Match against birth event's place ID |
| `glx/duplicates_runner.go` | Reads `born_on` for person disambiguation | Read from birth event date |

### 6. Rename/Reference Tracking

**File**: `go-glx/rename.go`

Place ID references in `born_at`/`died_at` are currently updated when places are renamed via `replaceInProperties()`. After removal, these properties won't exist, so no rename logic is needed for them. The rename system should still handle place references in Event `PlaceID` fields (which it presumably already does).

### 7. Example Archives

Update all example archives to remove these properties. Ensure corresponding birth/death events exist:

- `docs/examples/single-file/archive.glx`
- `docs/examples/complete-family/persons/person-john-smith.glx`
- `docs/examples/complete-family/persons/person-mary-brown.glx`
- `docs/examples/complete-family/persons/person-jane-smith.glx`
- `docs/examples/complete-family/assertions/assertion-john-birth.glx`
- `docs/examples/complete-family/assertions/assertion-john-birthplace.glx`
- `docs/examples/temporal-properties/archive.glx`
- `docs/examples/assertion-workflow/archive.glx`
- `docs/examples/participant-assertions/archive.glx`

Assertions that reference `born_on`/`born_at` as the `property` field need to be reworked or removed, since the property no longer exists.

### 8. Migration Tool

**New CLI command**: `glx migrate`

Reads an archive, performs the following for each person. **This must be zero data loss — removing redundancy, not discarding information.**

For each person with `born_on`/`born_at`/`died_on`/`died_at` properties:

1. **No existing birth/death event**: Create a birth/death Event entity with the date and/or place from the properties, with the person as principal participant
2. **Existing birth/death event**: Merge property data into the event — if the event is missing a date or place that the property has, add it to the event. Do not overwrite existing event data.
3. Remove `born_on`, `born_at`, `died_on`, `died_at` from the person's properties
4. Convert property assertions (`Subject: {Person: id}, Property: "born_on"` etc.) to event assertions (`Subject: {Event: id}`) pointing at the corresponding event property. Preserve all source/citation references.

**Output**: Writes the migrated archive. Reports what was changed.

**Scope**: This is a simple, single-purpose command. No flags beyond input/output paths.

### 9. Tests

- Update existing tests that set or assert on these properties
- Add test for `FindPersonEvent` helper
- Add test for `glx migrate` command
- Update GEDCOM import tests to verify properties are NOT set
- Update validation tests for the new error on removed properties

### 10. Documentation & Changelog

- Update `CHANGELOG.md` with breaking change entry under "Removed"
- Update `docs/quickstart.md` if it references these properties
- Update `docs/guides/hands-on-cli-guide.md` if affected
- Add `glx migrate` to CLI documentation (README, website sidebar, hands-on guide)

## Out of Scope

- Changing how other person properties work
- Adding computed/derived fields on Person
- Changing the Event entity structure
- Deprecation period — this is a hard break
