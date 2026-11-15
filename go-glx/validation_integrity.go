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

// nullEntityFindings reports every entity whose YAML node is empty (`persons:
// {my-person-id: }`), which decodes to a nil pointer. It returns the findings
// and, when there are any, a shallow copy of the archive with those entries
// dropped, so the remaining checks walk only real entities instead of
// dereferencing nil (#1406). The archive itself is left untouched.
func (glx *GLXFile) nullEntityFindings() ([]ValidationError, *GLXFile) {
	view := *glx
	var findings []ValidationError
	drop := func(entityType EntityType, ids []string) {
		for _, id := range ids {
			findings = append(findings, ValidationError{
				SourceType: entityType,
				SourceID:   id,
				Message: fmt.Sprintf("%s[%s]: entity is empty (null); give it its fields or remove the entry",
					entityType, id),
			})
		}
	}
	var ids []string
	view.Persons, ids = withoutNil(glx.Persons)
	drop(EntityTypePersons, ids)
	view.Relationships, ids = withoutNil(glx.Relationships)
	drop(EntityTypeRelationships, ids)
	view.Events, ids = withoutNil(glx.Events)
	drop(EntityTypeEvents, ids)
	view.Places, ids = withoutNil(glx.Places)
	drop(EntityTypePlaces, ids)
	view.Sources, ids = withoutNil(glx.Sources)
	drop(EntityTypeSources, ids)
	view.Citations, ids = withoutNil(glx.Citations)
	drop(EntityTypeCitations, ids)
	view.Repositories, ids = withoutNil(glx.Repositories)
	drop(EntityTypeRepositories, ids)
	view.Assertions, ids = withoutNil(glx.Assertions)
	drop(EntityTypeAssertions, ids)
	view.Media, ids = withoutNil(glx.Media)
	drop(EntityTypeMedia, ids)
	view.ResearchLogs, ids = withoutNil(glx.ResearchLogs)
	drop(EntityTypeResearchLogs, ids)
	view.Studies, ids = withoutNil(glx.Studies)
	drop(EntityTypeStudies, ids)

	if len(findings) == 0 {
		return nil, glx
	}
	view.validation = nil

	return findings, &view
}

// withoutNil returns m without its nil entries, plus the sorted IDs it
// dropped. When nothing is nil it returns m itself, so the common case copies
// nothing.
func withoutNil[T any](m map[string]*T) (map[string]*T, []string) {
	var nilIDs []string
	for id, v := range m {
		if v == nil {
			nilIDs = append(nilIDs, id)
		}
	}
	if len(nilIDs) == 0 {
		return m, nil
	}
	slices.Sort(nilIDs)
	kept := make(map[string]*T, len(m)-len(nilIDs))
	for id, v := range m {
		if v != nil {
			kept[id] = v
		}
	}

	return kept, nilIDs
}

// validateIDCaseCollisions reports two IDs of one entity type that differ only
// by case (`Person-A` and `person-a`). Entity IDs are unique within each entity
// type, ignoring case (#744): in a multi-file archive both would be written to
// the same file, and on a case-insensitive filesystem one silently replaces
// the other. The serializer already refuses to write such an archive; this
// makes `glx validate` say so before anything is written (#696). IDs are
// folded with EntityIDToFilename, the same key the serializer uses.
func (glx *GLXFile) validateIDCaseCollisions(result *ValidationResult) {
	for _, entityType := range AllEntityTypes {
		byKey := make(map[string][]string)
		for id := range result.Entities[entityType] {
			key, err := EntityIDToFilename(id)
			if err != nil {
				continue // an invalid ID is reported by its own check
			}
			byKey[key] = append(byKey[key], id)
		}
		for _, ids := range byKey {
			if len(ids) < 2 {
				continue
			}
			slices.Sort(ids)
			for _, other := range ids[1:] {
				result.Errors = append(result.Errors, ValidationError{
					SourceType: entityType,
					SourceID:   other,
					TargetType: string(entityType),
					TargetID:   ids[0],
					Message: fmt.Sprintf("%s IDs %q and %q differ only by case; "+
						"IDs must be unique within each entity type, ignoring case",
						entityType, ids[0], other),
				})
			}
		}
	}
}

// assertionStructuralProperties lists, per subject type, the entity fields an
// assertion may name as its `property` without a property-vocabulary entry:
// the subject's own top-level single-valued fields, read from its YAML tags
// so a field added to the type is covered without a second list. Collections
// (properties, notes, participants, census neighbors) are not assertion
// targets. Persons have no such fields.
var assertionStructuralProperties = map[EntityType][]string{
	EntityTypeEvents:        structuralFieldNames(reflect.TypeFor[Event]()),
	EntityTypePlaces:        structuralFieldNames(reflect.TypeFor[Place]()),
	EntityTypeRelationships: structuralFieldNames(reflect.TypeFor[Relationship]()),
}

func structuralFieldNames(t reflect.Type) []string {
	var names []string
	for f := range t.Fields() {
		kind := f.Type.Kind()
		if kind == reflect.Pointer {
			kind = f.Type.Elem().Kind()
		}
		switch kind {
		case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
			continue
		default:
		}
		name, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}

	return names
}

// assertionPropertyVocab names the property vocabulary an assertion about a
// subject of the given type is checked against.
var assertionPropertyVocab = map[EntityType]string{
	EntityTypePersons:       PropPersonProperties,
	EntityTypeEvents:        PropEventProperties,
	EntityTypeRelationships: PropRelationshipProperties,
	EntityTypePlaces:        PropPlaceProperties,
}

// validateAssertionPropertyNames warns when an assertion's `property` is
// neither defined in its subject type's property vocabulary nor one of that
// subject's structural fields (#1224). A warning, not an error: emerging
// properties are allowed, but a typo such as `occupaton` should not pass
// silently. An archive with no vocabulary for the subject type is skipped;
// the per-entity property check already warns about that once.
func (glx *GLXFile) validateAssertionPropertyNames(result *ValidationResult) {
	for assertionID, assertion := range glx.Assertions {
		if assertion.Property == "" {
			continue
		}
		subjectType := assertion.Subject.Type()
		vocabKey, ok := assertionPropertyVocab[subjectType]
		if !ok {
			continue
		}
		if slices.Contains(assertionStructuralProperties[subjectType], assertion.Property) {
			continue
		}
		vocab := result.PropertyVocabs[vocabKey]
		if len(vocab) == 0 {
			continue
		}
		if _, defined := vocab[assertion.Property]; defined {
			continue
		}
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: EntityTypeAssertions,
			SourceID:   assertionID,
			Field:      "property",
			Message: fmt.Sprintf("%s[%s]: property %q is not defined in %s",
				EntityTypeAssertions, assertionID, assertion.Property, vocabKey),
		})
	}
}
