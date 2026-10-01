// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// NotesStrategy controls how MergePersons combines notes from the drop person
// into the keep person.
type NotesStrategy string

const (
	// NotesStrategyAppend concatenates drop's notes after keep's notes (default).
	NotesStrategyAppend NotesStrategy = "append"
	// NotesStrategyPreferKeep ignores drop's notes entirely.
	NotesStrategyPreferKeep NotesStrategy = "prefer-keep"
	// NotesStrategyPreferDrop replaces keep's notes with drop's (only if drop has any).
	NotesStrategyPreferDrop NotesStrategy = "prefer-drop"
)

// MergePersonsOptions controls behavior of the MergePersons operation.
type MergePersonsOptions struct {
	Comparison    ComparisonOptions
	NotesStrategy NotesStrategy
	KeepNewest    bool // resolve same-property conflicts by date, latest wins
	KeepOldest    bool // resolve same-property conflicts by date, earliest wins
}

// MergeResolution describes how a property collision was resolved.
type MergeResolution string

// Recognized MergeResolution values reported in PersonMergeConflict.Resolution.
const (
	ResolutionKeptKeep   MergeResolution = "kept-keep"
	ResolutionKeptNewest MergeResolution = "kept-newest"
	ResolutionKeptOldest MergeResolution = "kept-oldest"
)

// PersonMergeConflict records a property collision encountered during a merge.
type PersonMergeConflict struct {
	Verdict    Verdict
	Property   string
	KeepValue  any
	DropValue  any
	Resolution MergeResolution
}

// MergePersonsResult holds the outcome of a MergePersons operation.
type MergePersonsResult struct {
	RefsUpdated      int
	PropertiesMerged int
	NotesMerged      int
	Conflicts        []PersonMergeConflict
}

// MergePersons consolidates two person entities, retaining keepID and folding
// dropID's properties, notes, and references into it. The dropID person is
// removed from the archive after its data has been merged. All cross-references
// to dropID elsewhere in the archive are rewritten to point at keepID.
//
// Property-merge rules:
//   - Missing properties are copied, and identical values agree silently.
//   - TemporalValue, []TemporalValue and mixed lists are normalized to GLX
//     value/date objects before comparison; pointer forms are also accepted.
//   - Fixed-property refinements retain the narrower date or descendant place.
//     Overlapping ranges with no containment are reported for collision selection.
//   - Disproven fixed values never replace surviving claims. Compatible structured
//     fields combine; differing values for the same field are reported as collisions.
//   - Vocabulary-defined temporal properties combine non-conflicting dated and
//     undated history, whether the inputs are scalars, objects, or lists.
//   - Genuine conflicts use keep's value by default. KeepNewest/KeepOldest
//     resolve each collision by its embedded date, preserving unrelated history.
//   - Other multi-value lists are unioned with deep-equal duplicates removed.
//
// Missing standard property definitions are supplied without changing the
// archive vocabulary maps; explicit custom definitions win. Options and person
// IDs are validated before any mutation. Successful calls mutate the archive,
// so callers must synchronize against concurrent readers/writers.
//
// Notes are combined per opts.NotesStrategy.
//
// Returns an error if either ID is missing, the IDs are equal, either ID is
// not a person, both KeepNewest and KeepOldest are set, or NotesStrategy is
// invalid, or if the archive is nil or comparison options are invalid.
func MergePersons(glx *GLXFile, keepID, dropID string, opts MergePersonsOptions) (*MergePersonsResult, error) {
	if err := opts.Comparison.Validate(); err != nil {
		return nil, err
	}
	prepared, err := researchArchive(glx)
	if err != nil {
		return nil, err
	}

	if keepID == dropID {
		return nil, fmt.Errorf("%w: %q", ErrMergeSelfReferential, keepID)
	}
	if opts.KeepNewest && opts.KeepOldest {
		return nil, ErrMergeConflictingFlags
	}
	switch opts.NotesStrategy {
	case "":
		opts.NotesStrategy = NotesStrategyAppend
	case NotesStrategyAppend, NotesStrategyPreferKeep, NotesStrategyPreferDrop:
		// ok
	default:
		return nil, fmt.Errorf("%w: %q (want append | prefer-keep | prefer-drop)",
			ErrMergeInvalidNotesStrat, opts.NotesStrategy)
	}

	if err := requirePerson(glx, keepID, "keep-id"); err != nil {
		return nil, err
	}
	if err := requirePerson(glx, dropID, "drop-id"); err != nil {
		return nil, err
	}

	keep := glx.Persons[keepID]
	drop := glx.Persons[dropID]

	propsAdded, conflicts := mergePersonProperties(prepared, keep, drop, opts)
	notesAdded := mergePersonNotes(keep, drop, opts.NotesStrategy)

	delete(glx.Persons, dropID)
	refsUpdated := updateAllRefs(glx, dropID, keepID)
	glx.validation = nil

	return &MergePersonsResult{
		RefsUpdated:      refsUpdated,
		PropertiesMerged: propsAdded,
		NotesMerged:      notesAdded,
		Conflicts:        conflicts,
	}, nil
}

// requirePerson validates that id exists in glx.Persons. If id exists as a
// non-person entity, the error message reports that explicitly.
func requirePerson(glx *GLXFile, id, role string) error {
	if v, ok := glx.Persons[id]; ok && v != nil {
		return nil
	}
	if entityType, err := findEntityType(glx, id); err == nil {
		return fmt.Errorf("%w: %s %q exists in %s, not persons", ErrMergeNotAPerson, role, id, entityType)
	}

	return fmt.Errorf("%w: %s %q", ErrPersonNotFound, role, id)
}

// mergePersonProperties folds drop.Properties into keep.Properties per the
// rules documented on MergePersons. Returns the count of new/replaced property
// entries and any property collisions encountered.
func mergePersonProperties(archive *GLXFile, keep, drop *Person, opts MergePersonsOptions) (added int, conflicts []PersonMergeConflict) {
	for property, value := range keep.Properties {
		keep.Properties[property] = normalizeTemporalProperty(value)
	}
	if len(drop.Properties) == 0 {
		return 0, nil
	}
	if keep.Properties == nil {
		keep.Properties = map[string]any{}
	}
	properties := make([]string, 0, len(drop.Properties))
	for prop := range drop.Properties {
		properties = append(properties, prop)
	}
	sort.Strings(properties)
	for _, prop := range properties {
		dropVal := normalizeTemporalProperty(drop.Properties[prop])
		keepVal, exists := keep.Properties[prop]
		if !exists {
			keep.Properties[prop] = dropVal
			added++

			continue
		}

		if reflect.DeepEqual(keepVal, dropVal) {
			continue
		}
		def := ConflictProperty(archive, EntityRef{Person: "keep"}, prop)
		if IsTemporalProperty(def) {
			merged, count, collisions := mergeTemporalProperty(prop, keepVal, dropVal, def, archive.Places, opts)
			keep.Properties[prop] = merged
			added += count
			conflicts = append(conflicts, collisions...)

			continue
		}
		keepList, keepIsList := keepVal.([]any)
		dropList, dropIsList := dropVal.([]any)
		if keepIsList && dropIsList {
			merged, count := unionPropertyList(keepList, dropList)
			keep.Properties[prop] = merged
			added += count

			continue
		}
		merged, collision := mergeFixedProperty(prop, keepVal, dropVal, def, archive.Places, opts)
		if !reflect.DeepEqual(keepVal, merged) {
			keep.Properties[prop] = merged
			added++
		}
		if collision != nil {
			conflicts = append(conflicts, *collision)
		}
	}

	return added, conflicts
}

// mergeFixedProperty preserves the researcher's resolution before selecting a
// compatible value. Structured breakdowns are part of the claim: disjoint fields
// combine, while differing values of a shared field require collision selection.
func mergeFixedProperty(property string, keep, drop any, def *PropertyDefinition, places map[string]*Place, opts MergePersonsOptions) (any, *PersonMergeConflict) {
	kept, incoming := propertyFact(keep), propertyFact(drop)
	if equalStatus(incoming.Status, statusDisproven) {
		return keep, nil
	}
	if equalStatus(kept.Status, statusDisproven) {
		return drop, nil
	}
	comparison := compareFactPair(kept, incoming, def, places, opts.Comparison)
	chosen, compatible := keep, comparison.Verdict == VerdictAgree
	if comparison.Verdict == VerdictRefinement {
		var useDrop bool
		useDrop, compatible = refinementPreference(kept, incoming, def, places, opts.Comparison)
		if useDrop {
			chosen = drop
		}
	}
	if compatible {
		if merged, fieldsAgree := combinePropertyFields(keep, drop, chosen); fieldsAgree {
			return merged, nil
		}
		comparison.Verdict = VerdictDefinite
	} else if !comparison.IsConflict() && comparison.Verdict != VerdictRefinement {
		return keep, nil
	}
	useDrop, label := resolveConflict(keep, drop, opts)
	if useDrop {
		chosen = drop
	} else {
		chosen = keep
	}

	return chosen, &PersonMergeConflict{Verdict: comparison.Verdict, Property: property, KeepValue: keep, DropValue: drop, Resolution: label}
}

const propertyFieldsField = "fields"

func propertyFields(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		fields, _ := object[propertyFieldsField].(map[string]any)

		return fields
	}

	return nil
}

func combinePropertyFields(keep, drop, chosen any) (any, bool) {
	kept, incoming := propertyFields(keep), propertyFields(drop)
	if len(kept)+len(incoming) == 0 {
		return chosen, true
	}
	fields := make(map[string]any, len(kept)+len(incoming))
	maps.Copy(fields, kept)
	for key, value := range incoming {
		if previous, exists := fields[key]; exists && !reflect.DeepEqual(previous, value) {
			return chosen, false
		}
		fields[key] = value
	}
	object, structured := chosen.(map[string]any)
	if structured {
		object = maps.Clone(object)
	} else {
		object = map[string]any{temporalValueField: chosen}
	}
	object[propertyFieldsField] = fields

	return object, true
}

// refinementPreference selects a strictly narrower compatible claim. Intersecting
// dates need not contain one another, so they cannot always be ranked by precision.
func refinementPreference(keep, drop FactValue, def *PropertyDefinition, places map[string]*Place, opts ComparisonOptions) (useDrop, ordered bool) {
	if equalStatus(drop.Status, statusDisproven) {
		return false, true
	}
	if equalStatus(keep.Status, statusDisproven) {
		return true, true
	}
	if equalStatus(keep.Status, statusDisputed) || equalStatus(drop.Status, statusDisputed) {
		return false, false
	}
	if def.ReferenceType == EntityTypePlaces.String() {
		keepAncestor, dropAncestor := placeAncestor(keep.Value, drop.Value, places), placeAncestor(drop.Value, keep.Value, places)

		return keepAncestor, keepAncestor != dropAncestor
	}
	a, _ := glxdate.Parse(keep.Value)
	b, _ := glxdate.Parse(drop.Value)
	x, y := a.ValueBounds(opts.approximationWidth()), b.ValueBounds(opts.approximationWidth())
	if x.Outer == y.Outer {
		return x.Uncertain && !y.Uncertain, true
	}
	if x.Outer.Contains(y.Outer) {
		return true, true
	}

	return false, y.Outer.Contains(x.Outer)
}

// resolveConflict decides whether to replace keep's value with drop's. Returns
// true to use drop, false to keep keep, plus a label recording which rule
// applied.
//
// Default (no flag) keeps keep. KeepNewest/KeepOldest only apply when both
// values are maps with a `date` key from which a year can be extracted; if
// either value lacks a usable date, the rule falls back to keeping keep.
func resolveConflict(keepVal, dropVal any, opts MergePersonsOptions) (useDrop bool, label MergeResolution) {
	if !opts.KeepNewest && !opts.KeepOldest {
		return false, ResolutionKeptKeep
	}
	keepYear := propertyYear(keepVal)
	dropYear := propertyYear(dropVal)
	if keepYear == 0 || dropYear == 0 {
		return false, ResolutionKeptKeep
	}
	if opts.KeepNewest {
		return dropYear > keepYear, ResolutionKeptNewest
	}

	return dropYear < keepYear, ResolutionKeptOldest
}

// propertyYear extracts a year from a property value's `date` field. Returns 0
// for plain strings, list values, or maps without a parseable date.
func propertyYear(v any) int {
	m, ok := v.(map[string]any)
	if !ok {
		return 0
	}
	switch d := m[temporalDateField].(type) {
	case string:
		return ExtractFirstYear(d)
	case DateString:
		return d.Year()
	default:
		return 0
	}
}

// unionPropertyList returns the union of two property lists, deduplicating by
// deep-equal of each entry. Returns the merged list and the count of new
// entries added from drop. Duplicates within dropList itself are also
// collapsed (each distinct value is appended at most once).
func unionPropertyList(keepList, dropList []any) ([]any, int) {
	out := make([]any, len(keepList), len(keepList)+len(dropList))
	copy(out, keepList)
	added := 0
	for _, dropEntry := range dropList {
		if containsEntry(out, dropEntry) {
			continue
		}
		out = append(out, dropEntry)
		added++
	}

	return out, added
}

// containsEntry returns true if list contains an entry deep-equal to target.
func containsEntry(list []any, target any) bool {
	for _, entry := range list {
		if reflect.DeepEqual(entry, target) {
			return true
		}
	}

	return false
}

// mergePersonNotes combines drop.Notes into keep.Notes per strategy. Returns
// the number of drop notes incorporated into keep (0 for prefer-keep, full
// drop length for prefer-drop, append delta for append).
func mergePersonNotes(keep, drop *Person, strategy NotesStrategy) int {
	if drop.Notes.IsEmpty() {
		return 0
	}
	switch strategy {
	case NotesStrategyAppend:
		before := len(keep.Notes)
		keep.Notes = append(keep.Notes, drop.Notes...)

		return len(keep.Notes) - before
	case NotesStrategyPreferDrop:
		// drop is about to be deleted, so we can take its slice directly.
		count := len(drop.Notes)
		keep.Notes = drop.Notes

		return count
	case NotesStrategyPreferKeep:
		return 0
	}

	return 0
}

// normalizeTemporalProperty converts the SDK's typed history forms to the
// canonical map/list representation used by deserialization and validation.
// Maps and slices are copied when rewritten; caller-held typed values are not
// modified. Unrelated structured fields remain intact.
func normalizeTemporalProperty(value any) any {
	switch v := value.(type) {
	case TemporalValue:
		entry := map[string]any{temporalValueField: v.Value}
		if v.Date != "" {
			entry[temporalDateField] = string(v.Date)
		}

		return entry
	case *TemporalValue:
		if v == nil {
			return nil
		}

		return normalizeTemporalProperty(*v)
	case []TemporalValue:
		entries := make([]any, len(v))
		for i, entry := range v {
			entries[i] = normalizeTemporalProperty(entry)
		}

		return entries
	case []*TemporalValue:
		entries := make([]any, len(v))
		for i, entry := range v {
			entries[i] = normalizeTemporalProperty(entry)
		}

		return entries
	case []any:
		entries := make([]any, len(v))
		for i, entry := range v {
			entries[i] = normalizeTemporalProperty(entry)
		}

		return entries
	case map[string]any:
		if date, ok := v[temporalDateField].(DateString); ok {
			entry := maps.Clone(v)
			entry[temporalDateField] = string(date)

			return entry
		}
	}

	return value
}

func propertyEntries(v any) []any {
	v = normalizeTemporalProperty(v)
	if list, ok := v.([]any); ok {
		return list
	}

	return []any{v}
}

const (
	temporalValueField = "value"
	temporalDateField  = "date"
)

func propertyFact(v any) FactValue {
	v = normalizeTemporalProperty(v)
	f := FactValue{}
	if m, ok := v.(map[string]any); ok {
		f.Value = propertyText(m[temporalValueField])
		f.Date = DateString(propertyText(m[temporalDateField]))
		f.Status = propertyText(m["status"])
		f.Confidence = propertyText(m["confidence"])
	} else {
		f.Value = propertyText(v)
	}

	return f
}

func propertyText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if s, ok := v.(DateString); ok {
		return string(s)
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}

	return string(data)
}

// FormatPropertyValue renders scalar, structured and temporal property values
// readably for merge reports, retaining dates and structured field details.
func FormatPropertyValue(v any) string {
	v = normalizeTemporalProperty(v)
	if entries, ok := v.([]any); ok {
		parts := make([]string, len(entries))
		for i, entry := range entries {
			parts[i] = FormatPropertyValue(entry)
		}

		return strings.Join(parts, "; ")
	}
	f := propertyFact(v)
	if f.Value == "" {
		return propertyText(v)
	}
	text := f.Value
	if f.Date != "" {
		text += " (" + string(f.Date) + ")"
	}
	if fields := propertyFields(v); len(fields) > 0 {
		parts := make([]string, 0, len(fields))
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			parts = append(parts, key+": "+propertyText(fields[key]))
		}
		text += " [" + strings.Join(parts, ", ") + "]"
	}

	return text
}

func mergeTemporalProperty(property string, keep, drop any, def *PropertyDefinition, places map[string]*Place, opts MergePersonsOptions) (any, int, []PersonMergeConflict) {
	// Compare against the original kept history, never previously accepted drop
	// entries. A person's pre-existing disagreements are not ours to resolve.
	kept := temporalEntries(keep)
	removed := make([]bool, len(kept))
	var accepted []any
	var conflicts []PersonMergeConflict
	for _, incoming := range temporalEntries(drop) {
		if containsEntry(accepted, incoming) {
			continue
		}
		if containsEntry(kept, incoming) {
			// An identical claim already belongs to both histories. Do not use
			// its duplicate to adjudicate an existing keep-side disagreement.
			accepted = append(accepted, incoming)

			continue
		}
		accept, collisions, reports := compareTemporalMergeEntry(property, kept, incoming, def, places, opts)
		conflicts = append(conflicts, reports...)
		if !accept {
			continue
		}
		for _, idx := range collisions {
			removed[idx] = true
		}
		accepted = append(accepted, incoming)
	}
	entries := make([]any, 0, len(kept)+len(accepted))
	for i, entry := range kept {
		if !removed[i] {
			entries = append(entries, entry)
		}
	}
	entries, added := unionPropertyList(entries, accepted)
	if added == 0 {
		return keep, 0, conflicts
	}

	sort.SliceStable(entries, func(i, j int) bool {
		a, _ := propertyFact(entries[i]).Date.Parse()
		b, _ := propertyFact(entries[j]).Date.Parse()
		if a.Timing().Known != b.Timing().Known {
			return a.Timing().Known
		}

		return a.Timing().Outer.Start < b.Timing().Outer.Start
	})

	return entries, added, conflicts
}

// temporalEntries returns history entries as value/date objects. Temporal lists
// require objects, and wrapping undated scalars before deduplication keeps
// "Farmer" and {value: Farmer} from surviving as two identical entries.
func temporalEntries(v any) []any {
	entries := propertyEntries(v)
	for i, entry := range entries {
		if _, structured := entry.(map[string]any); !structured {
			entries[i] = map[string]any{temporalValueField: entry}
		}
	}

	return entries
}

// Each decision uses the same original keep-side values, so processing one
// incoming claim cannot change how a later incoming claim is evaluated.
func compareTemporalMergeEntry(property string, kept []any, incoming any, def *PropertyDefinition, places map[string]*Place, opts MergePersonsOptions) (bool, []int, []PersonMergeConflict) {
	accept := true
	var collisions []int
	var reports []PersonMergeConflict
	for i, existing := range kept {
		c := compareFactPair(propertyFact(existing), propertyFact(incoming), def, places, opts.Comparison)
		if !c.IsConflict() {
			continue
		}
		useDrop, label := resolveConflict(existing, incoming, opts)
		reports = append(reports, PersonMergeConflict{Verdict: c.Verdict, Property: property, KeepValue: existing, DropValue: incoming, Resolution: label})
		if useDrop {
			collisions = append(collisions, i)
		} else {
			accept = false
		}
	}

	return accept, collisions, reports
}
