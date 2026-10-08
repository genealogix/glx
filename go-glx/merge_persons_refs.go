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
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// MergeReferenceChange describes a typed reference affected by a person merge.
// Action is rewritten, cleared, pruned, or removed-assertion. OldValue retains
// whole pruned structured/temporal entries; NewValue is nil for deletions.
// Values are detached, transient preview data, not archive history records.
type MergeReferenceChange struct {
	EntityType EntityType
	ID         string
	Path       string
	TargetType EntityType
	Action     string
	OldValue   any
	NewValue   any
}

type mergeEntity struct {
	kind        EntityType
	id          string
	value       reflect.Value
	definitions map[string]*PropertyDefinition
}

func cloneMergeArchive(archive *GLXFile) *GLXFile {
	view := *archive
	view.validation = nil
	cloned, _ := reflect.TypeAssert[*GLXFile](cloneMergeValue(reflect.ValueOf(&view)))

	return cloned
}

// Preserve concrete SDK value types, including TemporalValue and integer
// scalars. A serialization roundtrip would retype custom YAML values.
func cloneMergeValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.New(value.Type()).Elem()
		if value.Kind() == reflect.Pointer {
			out.Set(reflect.New(value.Type().Elem()))
			out.Elem().Set(cloneMergeValue(value.Elem()))
		} else {
			out.Set(cloneMergeValue(value.Elem()))
		}

		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		for key, item := range value.Seq2() {
			out.SetMapIndex(key, cloneMergeValue(item))
		}

		return out
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := range value.Len() {
			out.Index(i).Set(cloneMergeValue(value.Index(i)))
		}

		return out
	case reflect.Struct:
		out := reflect.New(value.Type()).Elem()
		out.Set(value)
		for i := range value.NumField() {
			if value.Type().Field(i).IsExported() {
				out.Field(i).Set(cloneMergeValue(value.Field(i)))
			}
		}

		return out
	default:

		return value
	}
}

func mergePropertyVocabulary(archive *GLXFile, kind EntityType) map[string]*PropertyDefinition {
	switch kind {
	case EntityTypePersons:

		return archive.PersonProperties
	case EntityTypeEvents:

		return archive.EventProperties
	case EntityTypeRelationships:

		return archive.RelationshipProperties
	case EntityTypePlaces:

		return archive.PlaceProperties
	case EntityTypeSources:

		return archive.SourceProperties
	case EntityTypeCitations:

		return archive.CitationProperties
	case EntityTypeRepositories:

		return archive.RepositoryProperties
	case EntityTypeMedia:

		return archive.MediaProperties
	default:

		return nil // ResearchLog and Study properties are opaque metadata.
	}
}

func mergeEntityCollection(archive *GLXFile, kind EntityType) reflect.Value {
	root := reflect.ValueOf(archive).Elem()
	for i := range root.NumField() {
		if strings.Split(root.Type().Field(i).Tag.Get("yaml"), ",")[0] == string(kind) {
			return root.Field(i)
		}
	}

	return reflect.Value{}
}

func forMergeEntities(archive, prepared *GLXFile, visit func(mergeEntity)) {
	for _, kind := range []EntityType{
		EntityTypePersons, EntityTypeEvents, EntityTypeRelationships,
		EntityTypePlaces, EntityTypeSources, EntityTypeCitations, EntityTypeRepositories,
		EntityTypeAssertions, EntityTypeMedia, EntityTypeResearchLogs, EntityTypeStudies,
	} {
		collection := mergeEntityCollection(archive, kind)
		keys := collection.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
		for _, key := range keys {
			value := collection.MapIndex(key)
			if value.IsNil() {
				continue
			}
			definitions := mergePropertyVocabulary(prepared, kind)
			if kind == EntityTypeAssertions {
				a, _ := reflect.TypeAssert[*Assertion](value)
				definitions = mergePropertyVocabulary(prepared, a.Subject.Type())
			}
			visit(mergeEntity{kind, key.String(), value.Elem(), definitions})
		}
	}
}

// Walk structural reference tags and EntityRef fields. Property maps and
// ordinary strings are handled separately with vocabulary semantics.
func walkMergeStructural(value reflect.Value, path string, visit func(EntityType, string, reflect.Value)) {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return
		}
		walkMergeStructural(value.Elem(), path, visit)

		return
	}
	if value.Kind() == reflect.Slice {
		for i := range value.Len() {
			walkMergeStructural(value.Index(i), fmt.Sprintf("%s[%d]", path, i), visit)
		}

		return
	}
	if value.Kind() != reflect.Struct {
		return
	}
	for i := range value.NumField() {
		field := value.Type().Field(i)
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}
		fieldValue := value.Field(i)
		target := EntityType(field.Tag.Get("refType"))
		if value.Type() == reflect.TypeFor[EntityRef]() {
			target = EntityType(name + "s")
		}
		if target == "" {
			walkMergeStructural(fieldValue, fieldPath, visit)

			continue
		}
		if fieldValue.Kind() == reflect.String {
			if fieldValue.String() != "" {
				visit(target, fieldPath, fieldValue)
			}
		} else if fieldValue.Kind() == reflect.Slice {
			for j := range fieldValue.Len() {
				visit(target, fmt.Sprintf("%s[%d]", fieldPath, j), fieldValue.Index(j))
			}
		}
	}
}

func mergePropertyMap(value reflect.Value) map[string]any {
	properties, _ := reflect.TypeAssert[map[string]any](value)

	return properties
}

func walkMergeProperties(entity mergeEntity, visit func(map[string]any, string, map[string]*PropertyDefinition)) {
	value := entity.value
	if properties := value.FieldByName("Properties"); properties.IsValid() {
		visit(mergePropertyMap(properties), "properties", entity.definitions)
	}
	if participants := value.FieldByName("Participants"); participants.IsValid() {
		for i := range participants.Len() {
			visit(mergePropertyMap(participants.Index(i).FieldByName("Properties")),
				fmt.Sprintf("participants[%d].properties", i), entity.definitions)
		}
	}
	if participant := value.FieldByName("Participant"); participant.IsValid() && !participant.IsNil() {
		visit(mergePropertyMap(participant.Elem().FieldByName("Properties")), "participant.properties", entity.definitions)
	}
}

// Edit only the reference-bearing value of an occurrence. Pruning its value
// removes the complete occurrence, including dates/fields, not just one key.
func editMergeProperty(value reflect.Value, path string, edit func(string, string, any) (string, bool)) (reflect.Value, bool) {
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return value, false
		}
		next, remove := editMergeProperty(value.Elem(), path, edit)
		if remove {
			return value, true
		}
		out := reflect.New(value.Type()).Elem()
		if value.Kind() == reflect.Pointer {
			out.Set(reflect.New(value.Type().Elem()))
			out.Elem().Set(next)
		} else {
			out.Set(next)
		}

		return out, false
	}
	if value.Kind() == reflect.Slice {
		out := reflect.MakeSlice(value.Type(), 0, value.Len())
		for i := range value.Len() {
			next, remove := editMergeProperty(value.Index(i), fmt.Sprintf("%s[%d]", path, i), edit)
			if !remove {
				out = reflect.Append(out, next)
			}
		}

		return out, value.Len() > 0 && out.Len() == 0
	}
	reference := mergeReferenceValue(value)
	if !reference.IsValid() || reference.Kind() != reflect.String {
		return value, false
	}
	replacement, remove := edit(reference.String(), path, value.Interface())
	if remove {
		return value, true
	}
	if replacement == reference.String() {
		return value, false
	}
	out := cloneMergeValue(value)
	switch out.Kind() {
	case reflect.Map:
		out.SetMapIndex(reflect.ValueOf("value").Convert(out.Type().Key()), reflect.ValueOf(replacement))
	case reflect.Struct:
		out.FieldByName("Value").Set(reflect.ValueOf(replacement))
	default:
		out = reflect.New(value.Type()).Elem()
		out.SetString(replacement)
	}

	return out, false
}

func mergeReferenceValue(value reflect.Value) reflect.Value {
	reference := value
	if value.Kind() == reflect.Map && value.Type().Key().Kind() == reflect.String {
		reference = value.MapIndex(reflect.ValueOf("value").Convert(value.Type().Key()))
	} else if value.Type() == reflect.TypeFor[TemporalValue]() {
		reference = value.FieldByName("Value")
	}
	for reference.IsValid() && (reference.Kind() == reflect.Interface || reference.Kind() == reflect.Pointer) {
		if reference.IsNil() {
			return reflect.Value{}
		}
		reference = reference.Elem()
	}

	return reference
}

func editMergeProperties(entity mergeEntity, edit func(EntityType, string, string, any) (string, bool)) {
	walkMergeProperties(entity, func(properties map[string]any, prefix string, definitions map[string]*PropertyDefinition) {
		keys := make([]string, 0, len(properties))
		for key := range properties {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			definition := definitions[key]
			if definition == nil || definition.ReferenceType == "" || properties[key] == nil {
				continue
			}
			next, remove := editMergeProperty(reflect.ValueOf(properties[key]), prefix+"."+key,
				func(id, path string, original any) (string, bool) {
					return edit(EntityType(definition.ReferenceType), id, path, original)
				})
			if remove {
				delete(properties, key)
			} else {
				properties[key] = next.Interface()
			}
		}
	})
}

func cleanupResolvedPersonRelationships(archive, prepared *GLXFile, keepID, dropID string, result *MergePersonsResult) {
	result.RemovedRelationships = resolvedPersonRelationships(archive, keepID, dropID)
	removed := make(map[string]bool, len(result.RemovedRelationships))
	for _, id := range result.RemovedRelationships {
		removed[id] = true
	}
	removeMergeDependentAssertions(archive, prepared, removed, result)
	for _, id := range result.RemovedRelationships {
		delete(archive.Relationships, id)
	}
	forMergeEntities(archive, prepared, func(entity mergeEntity) {
		if entity.kind == EntityTypeResearchLogs {
			log, _ := reflect.TypeAssert[*ResearchLog](entity.value.Addr())
			if log.Subject != nil && removed[log.Subject.Relationship] {
				result.ReferenceChanges = append(result.ReferenceChanges, MergeReferenceChange{entity.kind, entity.id, "subject", EntityTypeRelationships, "cleared", *log.Subject, nil})
				log.Subject = nil
			}
		}
		editMergeProperties(entity, func(kind EntityType, id, path string, original any) (string, bool) {
			if kind == EntityTypeRelationships && removed[id] {
				result.ReferenceChanges = append(result.ReferenceChanges, MergeReferenceChange{entity.kind, entity.id, path, kind, "pruned", cloneMergeValue(reflect.ValueOf(original)).Interface(), nil})

				return id, true
			}

			return id, false
		})
	})
}

func removeMergeDependentAssertions(archive, prepared *GLXFile, removed map[string]bool, result *MergePersonsResult) {
	forMergeEntities(archive, prepared, func(entity mergeEntity) {
		if entity.kind != EntityTypeAssertions {
			return
		}
		a, _ := reflect.TypeAssert[*Assertion](entity.value.Addr())
		dependent := false
		recordDependency := func(path string, original any) {
			dependent = true
			result.ReferenceChanges = append(result.ReferenceChanges, MergeReferenceChange{
				entity.kind, entity.id, path, EntityTypeRelationships, "removed-assertion",
				cloneMergeValue(reflect.ValueOf(original)).Interface(), nil,
			})
		}
		if removed[a.Subject.Relationship] {
			recordDependency("subject.relationship", a.Subject.Relationship)
		}
		if def := ConflictProperty(prepared, a.Subject, a.Property); def != nil && def.ReferenceType == string(EntityTypeRelationships) {
			if removed[a.Value] {
				recordDependency("value", a.Value)
			}
		}
		walkMergeProperties(entity, func(properties map[string]any, prefix string, definitions map[string]*PropertyDefinition) {
			for key, value := range properties {
				def := definitions[key]
				if def == nil || def.ReferenceType != string(EntityTypeRelationships) || value == nil {
					continue
				}
				_, _ = editMergeProperty(reflect.ValueOf(value), prefix+"."+key, func(id, path string, original any) (string, bool) {
					if removed[id] {
						recordDependency(path, original)
					}

					return id, false
				})
			}
		})
		if dependent {
			result.RemovedAssertions = append(result.RemovedAssertions, entity.id)
			delete(archive.Assertions, entity.id)
		}
	})
}

func rewriteMergedPersonReferences(archive, prepared *GLXFile, oldID, newID string, result *MergePersonsResult) int {
	count := 0
	forMergeEntities(archive, prepared, func(entity mergeEntity) {
		record := func(path string, original, replacement any) {
			count++
			result.ReferenceChanges = append(result.ReferenceChanges, MergeReferenceChange{entity.kind, entity.id, path, EntityTypePersons, "rewritten", original, replacement})
		}
		walkMergeStructural(entity.value, "", func(kind EntityType, path string, value reflect.Value) {
			if kind == EntityTypePersons && value.String() == oldID {
				record(path, oldID, newID)
				value.SetString(newID)
			}
		})
		if entity.kind == EntityTypeAssertions {
			a, _ := reflect.TypeAssert[*Assertion](entity.value.Addr())
			if def := ConflictProperty(prepared, a.Subject, a.Property); def != nil && def.ReferenceType == string(EntityTypePersons) && a.Value == oldID {
				record("value", oldID, newID)
				a.Value = newID
			}
		}
		editMergeProperties(entity, func(kind EntityType, id, path string, original any) (string, bool) {
			if kind == EntityTypePersons && id == oldID {
				next, _ := editMergeProperty(cloneMergeValue(reflect.ValueOf(original)), path,
					func(_, _ string, _ any) (string, bool) { return newID, false })
				record(path, cloneMergeValue(reflect.ValueOf(original)).Interface(), next.Interface())

				return newID, false
			}

			return id, false
		})
	})

	return count
}

// The SDK accepts sparse archives, so validate entity references independently
// of vocabulary/format validation. Existing missing targets are tolerated;
// valid inputs cannot gain a dangling target, including structured values.
func missingMergeTargets(archive *GLXFile) (map[string]bool, error) {
	prepared, err := researchArchive(archive)
	if err != nil {
		return nil, err
	}
	missing := make(map[string]bool)
	check := func(kind EntityType, id string) {
		collection := mergeEntityCollection(archive, kind)
		if !collection.IsValid() {
			return
		} // Vocabulary values are not entity references.
		value := collection.MapIndex(reflect.ValueOf(id))
		if !value.IsValid() || value.IsNil() {
			missing[string(kind)+":"+id] = true
		}
	}
	forMergeEntities(archive, prepared, func(entity mergeEntity) {
		walkMergeStructural(entity.value, "", func(kind EntityType, _ string, value reflect.Value) { check(kind, value.String()) })
		if entity.kind == EntityTypeAssertions {
			a, _ := reflect.TypeAssert[*Assertion](entity.value.Addr())
			if def := ConflictProperty(prepared, a.Subject, a.Property); def != nil && def.ReferenceType != "" {
				check(EntityType(def.ReferenceType), a.Value)
			}
		}
		editMergeProperties(entity, func(kind EntityType, id, _ string, _ any) (string, bool) {
			check(kind, id)

			return id, false
		})
	})

	return missing, nil
}

func validateMergeReferences(before, after *GLXFile) error {
	oldMissing, err := missingMergeTargets(cloneMergeArchive(before))
	if err != nil {
		return err
	}
	newMissing, err := missingMergeTargets(after)
	if err != nil {
		return err
	}
	var introduced []string
	for target := range newMissing {
		if !oldMissing[target] {
			introduced = append(introduced, target)
		}
	}
	slices.Sort(introduced)
	if len(introduced) != 0 {
		return fmt.Errorf("%w: merge introduces missing targets: %s", ErrValidationFailed, strings.Join(introduced, ", "))
	}

	return nil
}
