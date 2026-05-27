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
	"reflect"
	"sort"
)

// Merge3Conflict records a single point of disagreement found during a 3-way
// merge. Conflicts come in two flavors: ones the merger could not resolve
// (AutoResolved == false), and ones it resolved using a stronger signal such
// as a higher confidence level (AutoResolved == true). The latter exist so
// callers can surface the resolution to the researcher.
type Merge3Conflict struct {
	// Path identifies the locus of disagreement, e.g.
	// "persons[person-foo].properties.birthDate" or
	// "assertions[assertion-bar].value".
	Path string

	// EntityType is the GLX entity type ("persons", "events", …). Empty for
	// non-entity conflicts (vocabularies, top-level metadata).
	EntityType string

	// EntityID is the entity ID being merged. Empty for non-entity conflicts.
	EntityID string

	// Property is the conflicting property name within the entity (e.g.
	// "birthDate"). Empty when the conflict is over the entity as a whole
	// (e.g. add/add of incompatible entities sharing an ID).
	Property string

	// BaseValue, OursValue, TheirsValue capture the conflicting values. Nil
	// means "absent on that side." Concrete types vary by field (string,
	// []string, map[string]any, *Person, …).
	BaseValue   any
	OursValue   any
	TheirsValue any

	// AutoResolved is true when the merger chose a winner (e.g. by confidence
	// rank). The chosen side is recorded in Resolution.
	AutoResolved bool

	// Resolution describes how the conflict was resolved when AutoResolved is
	// true. Empty otherwise. Examples: "ours (higher confidence)", "theirs
	// (higher confidence)".
	Resolution string
}

// ThreeWayMerge performs a structural 3-way merge of three GLXFile snapshots:
// the common ancestor (base), the current branch (ours), and the incoming
// branch (theirs). Returns the merged file and a slice of conflicts. A
// non-empty slice with all entries AutoResolved == true still represents a
// clean merge (callers should check for any conflict with AutoResolved ==
// false before declaring success).
//
// The merge is value-based: entities and properties present on only one side
// (relative to base) flow through; concurrent identical changes flow through;
// concurrent diverging changes produce conflicts. Reference list fields on
// assertions (Sources, Citations, Media) and free-form Notes use additive
// set-merge semantics — an addition on one side never conflicts with an
// unrelated addition on the other.
//
// Inputs are not mutated. Nil inputs are treated as empty GLXFiles.
func ThreeWayMerge(base, ours, theirs *GLXFile) (*GLXFile, []Merge3Conflict) {
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	merged := &GLXFile{}
	var conflicts []Merge3Conflict

	// Top-level metadata: 3-way scalar over the whole pointer.
	mergedMeta, metaConflict := merge3Metadata(base.ImportMetadata, ours.ImportMetadata, theirs.ImportMetadata)
	merged.ImportMetadata = mergedMeta
	if metaConflict != nil {
		conflicts = append(conflicts, *metaConflict)
	}

	// Entity maps.
	merged.Persons, conflicts = merge3EntityMap(EntityTypePersons, base.Persons, ours.Persons, theirs.Persons, mergeOnePerson, conflicts)
	merged.Relationships, conflicts = merge3EntityMap(EntityTypeRelationships, base.Relationships, ours.Relationships, theirs.Relationships, mergeOneRelationship, conflicts)
	merged.Events, conflicts = merge3EntityMap(EntityTypeEvents, base.Events, ours.Events, theirs.Events, mergeOneEvent, conflicts)
	merged.Places, conflicts = merge3EntityMap(EntityTypePlaces, base.Places, ours.Places, theirs.Places, mergeOnePlace, conflicts)
	merged.Sources, conflicts = merge3EntityMap(EntityTypeSources, base.Sources, ours.Sources, theirs.Sources, mergeOneSource, conflicts)
	merged.Citations, conflicts = merge3EntityMap(EntityTypeCitations, base.Citations, ours.Citations, theirs.Citations, mergeOneCitation, conflicts)
	merged.Repositories, conflicts = merge3EntityMap(EntityTypeRepositories, base.Repositories, ours.Repositories, theirs.Repositories, mergeOneRepository, conflicts)
	merged.Assertions, conflicts = merge3EntityMap(EntityTypeAssertions, base.Assertions, ours.Assertions, theirs.Assertions, mergeOneAssertion, conflicts)
	merged.Media, conflicts = merge3EntityMap(EntityTypeMedia, base.Media, ours.Media, theirs.Media, mergeOneMedia, conflicts)

	// Vocabulary maps and property-definition maps: opaque whole-value 3-way.
	merged.EventTypes, conflicts = merge3OpaqueMap("event_types", base.EventTypes, ours.EventTypes, theirs.EventTypes, conflicts)
	merged.ParticipantRoles, conflicts = merge3OpaqueMap("participant_roles", base.ParticipantRoles, ours.ParticipantRoles, theirs.ParticipantRoles, conflicts)
	merged.ConfidenceLevels, conflicts = merge3OpaqueMap("confidence_levels", base.ConfidenceLevels, ours.ConfidenceLevels, theirs.ConfidenceLevels, conflicts)
	merged.RelationshipTypes, conflicts = merge3OpaqueMap("relationship_types", base.RelationshipTypes, ours.RelationshipTypes, theirs.RelationshipTypes, conflicts)
	merged.PlaceTypes, conflicts = merge3OpaqueMap("place_types", base.PlaceTypes, ours.PlaceTypes, theirs.PlaceTypes, conflicts)
	merged.SourceTypes, conflicts = merge3OpaqueMap("source_types", base.SourceTypes, ours.SourceTypes, theirs.SourceTypes, conflicts)
	merged.RepositoryTypes, conflicts = merge3OpaqueMap("repository_types", base.RepositoryTypes, ours.RepositoryTypes, theirs.RepositoryTypes, conflicts)
	merged.MediaTypes, conflicts = merge3OpaqueMap("media_types", base.MediaTypes, ours.MediaTypes, theirs.MediaTypes, conflicts)
	merged.SexTypes, conflicts = merge3OpaqueMap("sex_types", base.SexTypes, ours.SexTypes, theirs.SexTypes, conflicts)
	merged.GenderTypes, conflicts = merge3OpaqueMap("gender_types", base.GenderTypes, ours.GenderTypes, theirs.GenderTypes, conflicts)
	merged.LegalStatuses, conflicts = merge3OpaqueMap("legal_statuses", base.LegalStatuses, ours.LegalStatuses, theirs.LegalStatuses, conflicts)

	merged.PersonProperties, conflicts = merge3OpaqueMap("person_properties", base.PersonProperties, ours.PersonProperties, theirs.PersonProperties, conflicts)
	merged.EventProperties, conflicts = merge3OpaqueMap("event_properties", base.EventProperties, ours.EventProperties, theirs.EventProperties, conflicts)
	merged.RelationshipProperties, conflicts = merge3OpaqueMap("relationship_properties", base.RelationshipProperties, ours.RelationshipProperties, theirs.RelationshipProperties, conflicts)
	merged.PlaceProperties, conflicts = merge3OpaqueMap("place_properties", base.PlaceProperties, ours.PlaceProperties, theirs.PlaceProperties, conflicts)
	merged.MediaProperties, conflicts = merge3OpaqueMap("media_properties", base.MediaProperties, ours.MediaProperties, theirs.MediaProperties, conflicts)
	merged.RepositoryProperties, conflicts = merge3OpaqueMap("repository_properties", base.RepositoryProperties, ours.RepositoryProperties, theirs.RepositoryProperties, conflicts)
	merged.CitationProperties, conflicts = merge3OpaqueMap("citation_properties", base.CitationProperties, ours.CitationProperties, theirs.CitationProperties, conflicts)
	merged.SourceProperties, conflicts = merge3OpaqueMap("source_properties", base.SourceProperties, ours.SourceProperties, theirs.SourceProperties, conflicts)

	return merged, conflicts
}

// HasUnresolvedConflict reports whether any conflict in the slice was not
// auto-resolved. The driver uses this to decide whether to write the merged
// output or fall back to git's text merge.
func HasUnresolvedConflict(conflicts []Merge3Conflict) bool {
	for i := range conflicts {
		if !conflicts[i].AutoResolved {
			return true
		}
	}

	return false
}

// ============================================================================
// Entity-map orchestration
// ============================================================================

// merge3EntityMap walks a single typed entity map (e.g. Persons), applying
// 3-way logic per ID. mergeOne handles the per-entity property merge when an
// ID exists in more than one side.
func merge3EntityMap[V any](
	entityType string,
	base, ours, theirs map[string]*V,
	mergeOne func(entityType, id string, base, ours, theirs *V) (*V, []Merge3Conflict),
	conflicts []Merge3Conflict,
) (map[string]*V, []Merge3Conflict) {
	ids := unionMapKeys(base, ours, theirs)
	out := make(map[string]*V, len(ids))

	for _, id := range ids {
		merged, sub, keep := merge3EntityID(entityType, id, base, ours, theirs, mergeOne)
		if keep {
			out[id] = merged
		}
		conflicts = append(conflicts, tagConflicts(entityType, id, sub)...)
	}

	if len(out) == 0 {
		return nil, conflicts
	}

	return out, conflicts
}

// tagConflicts fills in EntityType and EntityID on any conflict that doesn't
// already carry them. The per-entity mergeOne functions and the field-level
// helpers (scalarOrConflict, merge3Properties, …) only know the Path; they
// don't carry the entity type and ID through every call site. The runner
// uses EntityType / EntityID to look up assertion-level context (confidence,
// citations) for stderr conflict summaries, so we backfill them here, where
// we know both.
func tagConflicts(entityType, id string, conflicts []Merge3Conflict) []Merge3Conflict {
	for i := range conflicts {
		if conflicts[i].EntityType == "" {
			conflicts[i].EntityType = entityType
		}
		if conflicts[i].EntityID == "" {
			conflicts[i].EntityID = id
		}
	}

	return conflicts
}

// merge3EntityID dispatches one ID's base/ours/theirs presence pattern to the
// right rule. Returns (merged value, conflicts produced by this ID, whether
// the merged value should be present in the output map).
//
// A nil pointer stored at the ID is treated as "absent" rather than "present
// with a null value" — git's merge can land us with a map literal that lists
// an ID but no body (e.g. an empty `persons:` entry under YAML serialization
// quirks), and we never want to emit `persons: {p1: null}` back out.
//
//nolint:gocyclo // 7-case finite presence enumeration; splitting further hurts readability.
func merge3EntityID[V any](
	entityType, id string,
	base, ours, theirs map[string]*V,
	mergeOne func(entityType, id string, base, ours, theirs *V) (*V, []Merge3Conflict),
) (*V, []Merge3Conflict, bool) {
	b, bOK := lookupNonNil(base, id)
	o, oOK := lookupNonNil(ours, id)
	t, tOK := lookupNonNil(theirs, id)

	switch {
	case !bOK && oOK && !tOK:
		return o, nil, true
	case !bOK && !oOK && tOK:
		return t, nil, true
	case bOK && oOK && !tOK:
		return mergeDeleteVsKeep(entityType, id, b, o, nil)
	case bOK && !oOK && tOK:
		return mergeDeleteVsKeep(entityType, id, b, nil, t)
	case !bOK && oOK && tOK:
		if reflect.DeepEqual(o, t) {
			return o, nil, true
		}
		// Concurrent add of the same ID with different shape — recurse with a
		// synthetic empty base so compatible fields can still merge.
		var zero V
		m, sub := mergeOne(entityType, id, &zero, o, t)

		return m, sub, true
	case bOK && !oOK && !tOK:
		// Triple delete (defensive — unionMapKeys filters this).
		return nil, nil, false
	default:
		m, sub := mergeOne(entityType, id, b, o, t)

		return m, sub, true
	}
}

// mergeDeleteVsKeep handles the asymmetric "one side deleted, the other side
// still has it" case. The surviving side is passed via either `ours` or
// `theirs` (the deleted side is nil). When the surviving side equals base
// (i.e. the only change was the delete), the delete proceeds quietly;
// otherwise we record a delete-vs-modify conflict and keep the surviving
// side in the merged output so the diagnostic file remains well-formed.
func mergeDeleteVsKeep[V any](entityType, id string, b, ours, theirs *V) (*V, []Merge3Conflict, bool) {
	surviving := ours
	if surviving == nil {
		surviving = theirs
	}
	if reflect.DeepEqual(b, surviving) {
		return nil, nil, false
	}

	return surviving, []Merge3Conflict{{
		Path:        entityPath(entityType, id),
		EntityType:  entityType,
		EntityID:    id,
		BaseValue:   b,
		OursValue:   ours,
		TheirsValue: theirs,
	}}, true
}

// merge3OpaqueID handles one ID's worth of opaque-value 3-way merge. Returns
// the value to place in the merged map, conflicts produced, and whether to
// emit the entry into the output map. Same nil-as-absent rule as
// merge3EntityID applies.
//
//nolint:gocyclo // 7-case finite presence enumeration; splitting further hurts readability.
func merge3OpaqueID[V any](vocabType, id string, base, ours, theirs map[string]*V) (*V, []Merge3Conflict, bool) {
	b, bOK := lookupNonNil(base, id)
	o, oOK := lookupNonNil(ours, id)
	t, tOK := lookupNonNil(theirs, id)

	path := vocabType + "[" + id + "]"

	switch {
	case !bOK && oOK && !tOK:
		return o, nil, true
	case !bOK && !oOK && tOK:
		return t, nil, true
	case !bOK && oOK && tOK:
		if reflect.DeepEqual(o, t) {
			return o, nil, true
		}

		return o, []Merge3Conflict{{Path: path, EntityType: vocabType, EntityID: id, OursValue: o, TheirsValue: t}}, true
	case bOK && oOK && !tOK:
		if reflect.DeepEqual(b, o) {
			return nil, nil, false
		}

		return o, []Merge3Conflict{{Path: path, EntityType: vocabType, EntityID: id, BaseValue: b, OursValue: o}}, true
	case bOK && !oOK && tOK:
		if reflect.DeepEqual(b, t) {
			return nil, nil, false
		}

		return t, []Merge3Conflict{{Path: path, EntityType: vocabType, EntityID: id, BaseValue: b, TheirsValue: t}}, true
	case bOK && !oOK && !tOK:
		return nil, nil, false
	default:
		val, sub := opaqueAllPresent(path, vocabType, id, b, o, t)

		return val, sub, true
	}
}

// opaqueAllPresent resolves the all-three-present case for opaque-value
// merge: take whichever side diverged from base, or report a conflict if
// both diverged differently. The boolean "emit this" return is unconditional
// at this point (the entry is present everywhere), so callers append true.
func opaqueAllPresent[V any](path, vocabType, id string, b, o, t *V) (*V, []Merge3Conflict) {
	switch {
	case reflect.DeepEqual(b, o):
		return t, nil
	case reflect.DeepEqual(b, t):
		return o, nil
	case reflect.DeepEqual(o, t):
		return o, nil
	default:
		return o, []Merge3Conflict{{
			Path:        path,
			EntityType:  vocabType,
			EntityID:    id,
			BaseValue:   b,
			OursValue:   o,
			TheirsValue: t,
		}}
	}
}

// merge3OpaqueMap merges a map whose values are compared with reflect.DeepEqual
// as a whole — used for vocabulary and property-definition maps where field-
// level merging adds little value and risks subtle semantic drift.
func merge3OpaqueMap[V any](
	vocabType string,
	base, ours, theirs map[string]*V,
	conflicts []Merge3Conflict,
) (map[string]*V, []Merge3Conflict) {
	ids := unionMapKeys(base, ours, theirs)
	out := make(map[string]*V, len(ids))

	for _, id := range ids {
		merged, sub, keep := merge3OpaqueID(vocabType, id, base, ours, theirs)
		if keep {
			out[id] = merged
		}
		conflicts = append(conflicts, sub...)
	}

	if len(out) == 0 {
		return nil, conflicts
	}

	return out, conflicts
}

// merge3Metadata 3-way merges *Metadata as a single opaque pointer value. The
// metadata block is small enough that field-level merging buys little.
func merge3Metadata(base, ours, theirs *Metadata) (*Metadata, *Merge3Conflict) {
	switch {
	case reflect.DeepEqual(base, ours):
		return theirs, nil
	case reflect.DeepEqual(base, theirs):
		return ours, nil
	case reflect.DeepEqual(ours, theirs):
		return ours, nil
	default:
		c := Merge3Conflict{
			Path:        "metadata",
			BaseValue:   base,
			OursValue:   ours,
			TheirsValue: theirs,
		}

		return ours, &c
	}
}

// ============================================================================
// Per-entity merge functions
// ============================================================================

// shortCircuitEntity handles the three pre-field-merge "two sides agree" cases
// shared by every mergeOne* function. When any two of base/ours/theirs are
// deep-equal, the third is returned (cloned). Returns (clone, true) when a
// short-circuit fired; (nil, false) otherwise, signaling that the caller
// should proceed to field-level merge.
func shortCircuitEntity[V any](base, ours, theirs *V) (*V, bool) {
	switch {
	case reflect.DeepEqual(base, ours):
		return cloneOrNil(theirs), true
	case reflect.DeepEqual(base, theirs):
		return cloneOrNil(ours), true
	case reflect.DeepEqual(ours, theirs):
		return cloneOrNil(ours), true
	}

	return nil, false
}

// orZero substitutes a fresh zero-valued struct for a nil pointer.
//
// ThreeWayMerge uses this on its three top-level inputs so the rest of the
// merge can dereference unconditionally.
//
// Each mergeOne* uses it on its base/ours/theirs after the DeepEqual
// short-circuit, since by that point the three sides are known to differ
// and field-level access (e.g. `base.Properties`) needs a non-nil receiver.
// The substitution can't move before the short-circuit without flipping
// `DeepEqual(nil, &V{})` from false to true, changing observable behavior on
// the nil-entry-in-map edge.
func orZero[V any](p *V) *V {
	if p == nil {
		return new(V)
	}

	return p
}

func mergeOnePerson(entityType, id string, base, ours, theirs *Person) (*Person, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	merged := &Person{}
	var conflicts []Merge3Conflict

	prefix := entityPath(entityType, id)

	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneRelationship(entityType, id string, base, ours, theirs *Relationship) (*Relationship, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Relationship{}
	var conflicts []Merge3Conflict

	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.StartEvent, conflicts = scalarOrConflict(prefix+".start_event", base.StartEvent, ours.StartEvent, theirs.StartEvent, conflicts)
	merged.EndEvent, conflicts = scalarOrConflict(prefix+".end_event", base.EndEvent, ours.EndEvent, theirs.EndEvent, conflicts)

	// Participants list — opaque (order-sensitive list of structs).
	merged.Participants, conflicts = participantsOrConflict(prefix+".participants", base.Participants, ours.Participants, theirs.Participants, conflicts)

	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneEvent(entityType, id string, base, ours, theirs *Event) (*Event, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Event{}
	var conflicts []Merge3Conflict

	merged.Title, conflicts = scalarOrConflict(prefix+".title", base.Title, ours.Title, theirs.Title, conflicts)
	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.PlaceID, conflicts = scalarOrConflict(prefix+".place", base.PlaceID, ours.PlaceID, theirs.PlaceID, conflicts)
	merged.Date, conflicts = dateOrConflict(prefix+".date", base.Date, ours.Date, theirs.Date, conflicts)

	merged.Participants, conflicts = participantsOrConflict(prefix+".participants", base.Participants, ours.Participants, theirs.Participants, conflicts)

	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOnePlace(entityType, id string, base, ours, theirs *Place) (*Place, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Place{}
	var conflicts []Merge3Conflict

	merged.Name, conflicts = scalarOrConflict(prefix+".name", base.Name, ours.Name, theirs.Name, conflicts)
	merged.ParentID, conflicts = scalarOrConflict(prefix+".parent", base.ParentID, ours.ParentID, theirs.ParentID, conflicts)
	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.Latitude, conflicts = float64PtrOrConflict(prefix+".latitude", base.Latitude, ours.Latitude, theirs.Latitude, conflicts)
	merged.Longitude, conflicts = float64PtrOrConflict(prefix+".longitude", base.Longitude, ours.Longitude, theirs.Longitude, conflicts)
	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneSource(entityType, id string, base, ours, theirs *Source) (*Source, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Source{}
	var conflicts []Merge3Conflict

	merged.Title, conflicts = scalarOrConflict(prefix+".title", base.Title, ours.Title, theirs.Title, conflicts)
	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.Date, conflicts = dateOrConflict(prefix+".date", base.Date, ours.Date, theirs.Date, conflicts)
	// Source.description was demoted to properties.description in #893 (#667);
	// it now flows through merge3Properties below as sources[id].properties.description.
	merged.RepositoryID, conflicts = scalarOrConflict(prefix+".repository", base.RepositoryID, ours.RepositoryID, theirs.RepositoryID, conflicts)
	merged.Language, conflicts = scalarOrConflict(prefix+".language", base.Language, ours.Language, theirs.Language, conflicts)

	// Authors — ordered, treat opaquely.
	merged.Authors, conflicts = orderedStringListOrConflict(prefix+".authors", base.Authors, ours.Authors, theirs.Authors, conflicts)

	// Media on Source is a reference list — additive 3-way set merge.
	merged.Media = merge3StringSet(base.Media, ours.Media, theirs.Media)

	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneCitation(entityType, id string, base, ours, theirs *Citation) (*Citation, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Citation{}
	var conflicts []Merge3Conflict

	merged.SourceID, conflicts = scalarOrConflict(prefix+".source", base.SourceID, ours.SourceID, theirs.SourceID, conflicts)
	merged.RepositoryID, conflicts = scalarOrConflict(prefix+".repository", base.RepositoryID, ours.RepositoryID, theirs.RepositoryID, conflicts)

	merged.Media = merge3StringSet(base.Media, ours.Media, theirs.Media)

	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneRepository(entityType, id string, base, ours, theirs *Repository) (*Repository, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Repository{}
	var conflicts []Merge3Conflict

	merged.Name, conflicts = scalarOrConflict(prefix+".name", base.Name, ours.Name, theirs.Name, conflicts)
	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.Address, conflicts = scalarOrConflict(prefix+".address", base.Address, ours.Address, theirs.Address, conflicts)
	merged.City, conflicts = scalarOrConflict(prefix+".city", base.City, ours.City, theirs.City, conflicts)
	merged.State, conflicts = scalarOrConflict(prefix+".state_province", base.State, ours.State, theirs.State, conflicts)
	merged.PostalCode, conflicts = scalarOrConflict(prefix+".postal_code", base.PostalCode, ours.PostalCode, theirs.PostalCode, conflicts)
	merged.Country, conflicts = scalarOrConflict(prefix+".country", base.Country, ours.Country, theirs.Country, conflicts)
	merged.Website, conflicts = scalarOrConflict(prefix+".website", base.Website, ours.Website, theirs.Website, conflicts)
	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneAssertion(entityType, id string, base, ours, theirs *Assertion) (*Assertion, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Assertion{}
	var conflicts []Merge3Conflict

	// Subject and Participant are typed structs/pointers — opaque 3-way.
	merged.Subject, conflicts = entityRefOrConflict(prefix+".subject", base.Subject, ours.Subject, theirs.Subject, conflicts)
	merged.Participant, conflicts = participantPtrOrConflict(prefix+".participant", base.Participant, ours.Participant, theirs.Participant, conflicts)
	merged.Property, conflicts = scalarOrConflict(prefix+".property", base.Property, ours.Property, theirs.Property, conflicts)
	merged.Date, conflicts = dateOrConflict(prefix+".date", base.Date, ours.Date, theirs.Date, conflicts)
	merged.Status, conflicts = scalarOrConflict(prefix+".status", base.Status, ours.Status, theirs.Status, conflicts)

	// Value + Confidence: confidence-aware resolution when both sides diverged.
	merged.Value, merged.Confidence, conflicts = mergeAssertionValue(prefix, base, ours, theirs, conflicts)

	// Reference lists: additive set merge.
	merged.Sources = merge3StringSet(base.Sources, ours.Sources, theirs.Sources)
	merged.Citations = merge3StringSet(base.Citations, ours.Citations, theirs.Citations)
	merged.Media = merge3StringSet(base.Media, ours.Media, theirs.Media)

	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

func mergeOneMedia(entityType, id string, base, ours, theirs *Media) (*Media, []Merge3Conflict) {
	if m, ok := shortCircuitEntity(base, ours, theirs); ok {
		return m, nil
	}
	base = orZero(base)
	ours = orZero(ours)
	theirs = orZero(theirs)

	prefix := entityPath(entityType, id)
	merged := &Media{}
	var conflicts []Merge3Conflict

	merged.URI, conflicts = scalarOrConflict(prefix+".uri", base.URI, ours.URI, theirs.URI, conflicts)
	merged.Type, conflicts = scalarOrConflict(prefix+".type", base.Type, ours.Type, theirs.Type, conflicts)
	merged.MimeType, conflicts = scalarOrConflict(prefix+".mime_type", base.MimeType, ours.MimeType, theirs.MimeType, conflicts)
	merged.Hash, conflicts = scalarOrConflict(prefix+".hash", base.Hash, ours.Hash, theirs.Hash, conflicts)
	merged.Title, conflicts = scalarOrConflict(prefix+".title", base.Title, ours.Title, theirs.Title, conflicts)
	merged.Description, conflicts = scalarOrConflict(prefix+".description", base.Description, ours.Description, theirs.Description, conflicts)
	merged.Date, conflicts = dateOrConflict(prefix+".date", base.Date, ours.Date, theirs.Date, conflicts)
	merged.Source, conflicts = scalarOrConflict(prefix+".source", base.Source, ours.Source, theirs.Source, conflicts)
	merged.Properties, conflicts = merge3Properties(prefix+".properties", base.Properties, ours.Properties, theirs.Properties, conflicts)
	merged.Notes = merge3NoteList(base.Notes, ours.Notes, theirs.Notes)

	return merged, conflicts
}

// mergeAssertionValue merges Value + Confidence as a coupled pair. When both
// sides change Value differently, the side with strictly higher Confidence
// wins (auto-resolved). Equal or unknown confidence → unresolved conflict.
//
// Confidence itself, when only one side changed it, follows the standard
// 3-way scalar rule. The coupled handling activates only when Value diverges.
func mergeAssertionValue(prefix string, base, ours, theirs *Assertion, conflicts []Merge3Conflict) (string, string, []Merge3Conflict) {
	bv, bc := base.Value, base.Confidence
	ov, oc := ours.Value, ours.Confidence
	tv, tc := theirs.Value, theirs.Confidence

	switch {
	case ov == bv && tv == bv:
		// neither side changed value — confidence can still differ; standard scalar.
		conf, mergedConflicts := scalarOrConflict(prefix+".confidence", bc, oc, tc, conflicts)

		return ov, conf, mergedConflicts
	case ov == bv:
		// only theirs changed value — take theirs (value + confidence move together).
		return tv, tc, conflicts
	case tv == bv:
		return ov, oc, conflicts
	case ov == tv:
		// both sides made the same value change — pick that, confidence falls out by scalar rule.
		conf, mergedConflicts := scalarOrConflict(prefix+".confidence", bc, oc, tc, conflicts)

		return ov, conf, mergedConflicts
	}

	// Diverging value changes — confidence-based auto-resolve.
	or := confidenceRank[oc]
	tr := confidenceRank[tc]
	oKnown := isKnownConfidence(oc)
	tKnown := isKnownConfidence(tc)

	switch {
	case oKnown && tKnown && or > tr:
		conflicts = append(conflicts, Merge3Conflict{
			Path:         prefix + ".value",
			BaseValue:    bv,
			OursValue:    ov,
			TheirsValue:  tv,
			AutoResolved: true,
			Resolution:   "ours (higher confidence)",
		})

		return ov, oc, conflicts
	case oKnown && tKnown && tr > or:
		conflicts = append(conflicts, Merge3Conflict{
			Path:         prefix + ".value",
			BaseValue:    bv,
			OursValue:    ov,
			TheirsValue:  tv,
			AutoResolved: true,
			Resolution:   "theirs (higher confidence)",
		})

		return tv, tc, conflicts
	default:
		conflicts = append(conflicts, Merge3Conflict{
			Path:        prefix + ".value",
			BaseValue:   bv,
			OursValue:   ov,
			TheirsValue: tv,
		})

		return ov, oc, conflicts
	}
}

// isKnownConfidence reports whether c is one of the standard confidence
// levels we can rank. Custom or empty confidence is treated as unknown and
// never wins the auto-resolution tiebreaker.
func isKnownConfidence(c string) bool {
	_, ok := confidenceRank[c]

	return ok
}

// ============================================================================
// Primitive 3-way helpers
// ============================================================================

// scalarOrConflict 3-way merges a single comparable value. Appends a conflict
// when ours and theirs diverged from base and from each other, and returns
// ours unchanged. Otherwise returns the natural 3-way winner.
//
//nolint:ireturn // T is a type parameter constrained to `comparable`, not a returned interface.
func scalarOrConflict[T comparable](path string, base, ours, theirs T, conflicts []Merge3Conflict) (T, []Merge3Conflict) {
	switch {
	case ours == base:
		return theirs, conflicts
	case theirs == base:
		return ours, conflicts
	case ours == theirs:
		return ours, conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return ours, conflicts
}

// dateOrConflict mirrors scalarOrConflict for DateString (which is a typed
// string but a Go generic clause on `comparable` would need DateString to be
// declared comparable, which it is by virtue of being a string alias).
func dateOrConflict(path string, base, ours, theirs DateString, conflicts []Merge3Conflict) (DateString, []Merge3Conflict) {
	return scalarOrConflict(path, base, ours, theirs, conflicts)
}

// float64PtrOrConflict handles *float64 (used for Place lat/lng).
func float64PtrOrConflict(path string, base, ours, theirs *float64, conflicts []Merge3Conflict) (*float64, []Merge3Conflict) {
	switch {
	case float64PtrEq(base, ours):
		return theirs, conflicts
	case float64PtrEq(base, theirs):
		return ours, conflicts
	case float64PtrEq(ours, theirs):
		return ours, conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return ours, conflicts
}

func float64PtrEq(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return *a == *b
}

// orderedStringListOrConflict handles []string where order is significant
// (e.g. Source.Authors). Treated as a single opaque value.
func orderedStringListOrConflict(path string, base, ours, theirs []string, conflicts []Merge3Conflict) ([]string, []Merge3Conflict) {
	switch {
	case stringSliceEq(base, ours):
		return cloneStrings(theirs), conflicts
	case stringSliceEq(base, theirs):
		return cloneStrings(ours), conflicts
	case stringSliceEq(ours, theirs):
		return cloneStrings(ours), conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return cloneStrings(ours), conflicts
}

func stringSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func cloneStrings(s []string) []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s))
	copy(out, s)

	return out
}

// participantsOrConflict handles []Participant (Events, Relationships) as an
// opaque ordered list of structs. Field-level participant merging is out of
// scope for the MVP.
func participantsOrConflict(path string, base, ours, theirs []Participant, conflicts []Merge3Conflict) ([]Participant, []Merge3Conflict) {
	switch {
	case reflect.DeepEqual(base, ours):
		return cloneParticipants(theirs), conflicts
	case reflect.DeepEqual(base, theirs):
		return cloneParticipants(ours), conflicts
	case reflect.DeepEqual(ours, theirs):
		return cloneParticipants(ours), conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return cloneParticipants(ours), conflicts
}

func cloneParticipants(p []Participant) []Participant {
	if p == nil {
		return nil
	}
	out := make([]Participant, len(p))
	copy(out, p)

	return out
}

func participantPtrOrConflict(path string, base, ours, theirs *Participant, conflicts []Merge3Conflict) (*Participant, []Merge3Conflict) {
	switch {
	case reflect.DeepEqual(base, ours):
		return clonePtrParticipant(theirs), conflicts
	case reflect.DeepEqual(base, theirs):
		return clonePtrParticipant(ours), conflicts
	case reflect.DeepEqual(ours, theirs):
		return clonePtrParticipant(ours), conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return clonePtrParticipant(ours), conflicts
}

func clonePtrParticipant(p *Participant) *Participant {
	if p == nil {
		return nil
	}
	v := *p

	return &v
}

func entityRefOrConflict(path string, base, ours, theirs EntityRef, conflicts []Merge3Conflict) (EntityRef, []Merge3Conflict) {
	switch {
	case base == ours:
		return theirs, conflicts
	case base == theirs:
		return ours, conflicts
	case ours == theirs:
		return ours, conflicts
	}

	conflicts = append(conflicts, Merge3Conflict{
		Path:        path,
		BaseValue:   base,
		OursValue:   ours,
		TheirsValue: theirs,
	})

	return ours, conflicts
}

// merge3StringSet performs an additive 3-way merge of a string list treated
// as a set. Entries removed from base on either side are removed; entries
// added on either side are added. Order: preserved from base for entries that
// survive, then ours-additions in their original order, then theirs-additions
// in their original order. Duplicates are deduped throughout.
func merge3StringSet(base, ours, theirs []string) []string {
	baseSet := stringSet(base)
	oursSet := stringSet(ours)
	theirsSet := stringSet(theirs)

	// A base entry survives unless removed by either side.
	survivors := make(map[string]struct{}, len(baseSet))
	for v := range baseSet {
		_, inOurs := oursSet[v]
		_, inTheirs := theirsSet[v]
		if inOurs && inTheirs {
			survivors[v] = struct{}{}
		}
	}

	out := make([]string, 0, len(base)+len(ours)+len(theirs))
	emitted := make(map[string]struct{}, len(out))

	// Step 1: preserve base order for survivors.
	for _, v := range base {
		if _, ok := survivors[v]; !ok {
			continue
		}
		if _, done := emitted[v]; done {
			continue
		}
		out = append(out, v)
		emitted[v] = struct{}{}
	}

	// Step 2: ours-additions (not already emitted).
	for _, v := range ours {
		if _, ok := baseSet[v]; ok {
			continue
		}
		if _, done := emitted[v]; done {
			continue
		}
		out = append(out, v)
		emitted[v] = struct{}{}
	}

	// Step 3: theirs-additions.
	for _, v := range theirs {
		if _, ok := baseSet[v]; ok {
			continue
		}
		if _, done := emitted[v]; done {
			continue
		}
		out = append(out, v)
		emitted[v] = struct{}{}
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func stringSet(s []string) map[string]struct{} {
	out := make(map[string]struct{}, len(s))
	for _, v := range s {
		out[v] = struct{}{}
	}

	return out
}

// merge3NoteList applies the same set-merge semantics to NoteList values.
func merge3NoteList(base, ours, theirs NoteList) NoteList {
	merged := merge3StringSet([]string(base), []string(ours), []string(theirs))
	if merged == nil {
		return nil
	}

	return NoteList(merged)
}

// merge3Properties performs per-key 3-way merge of the Properties map[string]any.
// Per-key concurrent diverging changes record a conflict; the function picks
// ours so the diagnostic output stays well-formed.
func merge3Properties(prefix string, base, ours, theirs map[string]any, conflicts []Merge3Conflict) (map[string]any, []Merge3Conflict) {
	if base == nil && ours == nil && theirs == nil {
		return nil, conflicts
	}

	keys := make(map[string]struct{})
	for k := range base {
		keys[k] = struct{}{}
	}
	for k := range ours {
		keys[k] = struct{}{}
	}
	for k := range theirs {
		keys[k] = struct{}{}
	}

	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	out := make(map[string]any, len(sorted))
	for _, k := range sorted {
		val, sub, keep := merge3PropertyKey(prefix, k, base, ours, theirs)
		if keep {
			out[k] = val
		}
		conflicts = append(conflicts, sub...)
	}

	if len(out) == 0 {
		return nil, conflicts
	}

	return out, conflicts
}

// merge3PropertyKey resolves a single key in a Properties map. The presence
// matrix has the same six effective cases as entity-map merging; we delegate
// the all-three-present case to a small helper to keep this function under
// the cyclomatic threshold.
//
//nolint:gocyclo // 7-case finite presence enumeration; splitting further hurts readability.
func merge3PropertyKey(prefix, k string, base, ours, theirs map[string]any) (any, []Merge3Conflict, bool) {
	b, bOK := base[k]
	o, oOK := ours[k]
	t, tOK := theirs[k]

	path := prefix + "." + k

	switch {
	case !bOK && oOK && !tOK:
		return o, nil, true
	case !bOK && !oOK && tOK:
		return t, nil, true
	case !bOK && oOK && tOK:
		if reflect.DeepEqual(o, t) {
			return o, nil, true
		}

		return o, []Merge3Conflict{{Path: path, Property: k, OursValue: o, TheirsValue: t}}, true
	case bOK && oOK && !tOK:
		if reflect.DeepEqual(b, o) {
			return nil, nil, false
		}

		return o, []Merge3Conflict{{Path: path, Property: k, BaseValue: b, OursValue: o}}, true
	case bOK && !oOK && tOK:
		if reflect.DeepEqual(b, t) {
			return nil, nil, false
		}

		return t, []Merge3Conflict{{Path: path, Property: k, BaseValue: b, TheirsValue: t}}, true
	case bOK && !oOK && !tOK:
		return nil, nil, false
	default:
		return propertyAllPresent(path, k, b, o, t)
	}
}

func propertyAllPresent(path, k string, b, o, t any) (any, []Merge3Conflict, bool) {
	switch {
	case reflect.DeepEqual(b, o):
		return t, nil, true
	case reflect.DeepEqual(b, t):
		return o, nil, true
	case reflect.DeepEqual(o, t):
		return o, nil, true
	default:
		return o, []Merge3Conflict{{
			Path:        path,
			Property:    k,
			BaseValue:   b,
			OursValue:   o,
			TheirsValue: t,
		}}, true
	}
}

// ============================================================================
// Utility
// ============================================================================

// unionMapKeys returns the sorted union of keys across the input maps,
// excluding keys whose value is a nil pointer. A nil-valued map entry is
// treated as "key absent" so the merger doesn't carry forward (or emit)
// entries like `persons: {p1: null}` in the merged YAML output.
func unionMapKeys[V any](maps ...map[string]*V) []string {
	seen := make(map[string]struct{})
	for _, m := range maps {
		for k, v := range m {
			if v == nil {
				continue
			}
			seen[k] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)

	return out
}

func entityPath(entityType, id string) string {
	return entityType + "[" + id + "]"
}

// lookupNonNil reads m[id] and reports presence only when the stored pointer
// is non-nil. A `map[string]*Person{"p1": nil}` returns (nil, false), not
// (nil, true), so downstream logic treats it as "absent" — see merge3EntityID
// for why this matters.
func lookupNonNil[V any](m map[string]*V, id string) (*V, bool) {
	v, ok := m[id]
	if !ok || v == nil {
		return nil, false
	}

	return v, true
}

func cloneOrNil[V any](v *V) *V {
	if v == nil {
		return nil
	}
	clone := *v

	return &clone
}
