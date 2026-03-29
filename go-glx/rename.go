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

import "fmt"

// RenameResult holds the outcome of a rename operation.
type RenameResult struct {
	EntityType  string // which entity map contained the ID (e.g., "persons")
	RefsUpdated int    // number of reference fields updated
}

// RenameEntity renames an entity ID throughout the archive, updating all
// cross-references. Returns an error if the old ID is not found or the
// new ID already exists. Invalidates the validation cache after mutation.
func RenameEntity(glx *GLXFile, oldID, newID string) (*RenameResult, error) {
	entityType, err := findEntityType(glx, oldID)
	if err != nil {
		return nil, err
	}

	if err := checkTargetFree(glx, newID); err != nil {
		return nil, err
	}

	// Move the map key
	moveMapKey(glx, entityType, oldID, newID)

	// Update all references
	refs := updateAllRefs(glx, oldID, newID)

	// Invalidate cached validation since maps have been mutated
	glx.validation = nil

	return &RenameResult{
		EntityType:  entityType,
		RefsUpdated: refs + 1, // +1 for the map key itself
	}, nil
}

// findEntityType returns which entity map contains the given ID.
func findEntityType(glx *GLXFile, id string) (string, error) {
	if v, ok := glx.Persons[id]; ok && v != nil {
		return EntityTypePersons, nil
	}
	if v, ok := glx.Events[id]; ok && v != nil {
		return EntityTypeEvents, nil
	}
	if v, ok := glx.Relationships[id]; ok && v != nil {
		return EntityTypeRelationships, nil
	}
	if v, ok := glx.Places[id]; ok && v != nil {
		return EntityTypePlaces, nil
	}
	if v, ok := glx.Sources[id]; ok && v != nil {
		return EntityTypeSources, nil
	}
	if v, ok := glx.Citations[id]; ok && v != nil {
		return EntityTypeCitations, nil
	}
	if v, ok := glx.Repositories[id]; ok && v != nil {
		return EntityTypeRepositories, nil
	}
	if v, ok := glx.Assertions[id]; ok && v != nil {
		return EntityTypeAssertions, nil
	}
	if v, ok := glx.Media[id]; ok && v != nil {
		return EntityTypeMedia, nil
	}
	return "", fmt.Errorf("entity %q not found in archive", id)
}

// checkTargetFree returns an error if newID already exists in any entity map.
func checkTargetFree(glx *GLXFile, id string) error {
	if _, err := findEntityType(glx, id); err == nil {
		return fmt.Errorf("entity %q already exists in archive", id)
	}
	return nil
}

// moveMapKey moves an entity from oldID to newID in its entity map.
func moveMapKey(glx *GLXFile, entityType, oldID, newID string) {
	switch entityType {
	case EntityTypePersons:
		glx.Persons[newID] = glx.Persons[oldID]
		delete(glx.Persons, oldID)
	case EntityTypeEvents:
		glx.Events[newID] = glx.Events[oldID]
		delete(glx.Events, oldID)
	case EntityTypeRelationships:
		glx.Relationships[newID] = glx.Relationships[oldID]
		delete(glx.Relationships, oldID)
	case EntityTypePlaces:
		glx.Places[newID] = glx.Places[oldID]
		delete(glx.Places, oldID)
	case EntityTypeSources:
		glx.Sources[newID] = glx.Sources[oldID]
		delete(glx.Sources, oldID)
	case EntityTypeCitations:
		glx.Citations[newID] = glx.Citations[oldID]
		delete(glx.Citations, oldID)
	case EntityTypeRepositories:
		glx.Repositories[newID] = glx.Repositories[oldID]
		delete(glx.Repositories, oldID)
	case EntityTypeAssertions:
		glx.Assertions[newID] = glx.Assertions[oldID]
		delete(glx.Assertions, oldID)
	case EntityTypeMedia:
		glx.Media[newID] = glx.Media[oldID]
		delete(glx.Media, oldID)
	}
}

// updateAllRefs scans every entity in the archive and replaces oldID with
// newID in all reference fields, including Properties maps and Assertion.Value.
// Returns the count of fields updated.
func updateAllRefs(glx *GLXFile, oldID, newID string) int {
	count := 0

	// Participants in events
	for _, ev := range glx.Events {
		if ev == nil {
			continue
		}
		for i := range ev.Participants {
			if ev.Participants[i].Person == oldID {
				ev.Participants[i].Person = newID
				count++
			}
			count += replaceInProperties(ev.Participants[i].Properties, oldID, newID)
		}
		if ev.PlaceID == oldID {
			ev.PlaceID = newID
			count++
		}
		count += replaceInProperties(ev.Properties, oldID, newID)
	}

	// Participants and event refs in relationships
	for _, rel := range glx.Relationships {
		if rel == nil {
			continue
		}
		for i := range rel.Participants {
			if rel.Participants[i].Person == oldID {
				rel.Participants[i].Person = newID
				count++
			}
			count += replaceInProperties(rel.Participants[i].Properties, oldID, newID)
		}
		if rel.StartEvent == oldID {
			rel.StartEvent = newID
			count++
		}
		if rel.EndEvent == oldID {
			rel.EndEvent = newID
			count++
		}
	}

	// Place parent hierarchy
	for _, place := range glx.Places {
		if place == nil {
			continue
		}
		if place.ParentID == oldID {
			place.ParentID = newID
			count++
		}
		count += replaceInProperties(place.Properties, oldID, newID)
	}

	// Person properties (born_at, died_at, etc. can contain place IDs)
	for _, person := range glx.Persons {
		if person == nil {
			continue
		}
		count += replaceInProperties(person.Properties, oldID, newID)
	}

	// Source refs
	for _, src := range glx.Sources {
		if src == nil {
			continue
		}
		if src.RepositoryID == oldID {
			src.RepositoryID = newID
			count++
		}
		count += replaceInSlice(src.Media, oldID, newID)
		count += replaceInProperties(src.Properties, oldID, newID)
	}

	// Citation refs
	for _, cit := range glx.Citations {
		if cit == nil {
			continue
		}
		if cit.SourceID == oldID {
			cit.SourceID = newID
			count++
		}
		if cit.RepositoryID == oldID {
			cit.RepositoryID = newID
			count++
		}
		count += replaceInSlice(cit.Media, oldID, newID)
		count += replaceInProperties(cit.Properties, oldID, newID)
	}

	// Assertion refs
	for _, a := range glx.Assertions {
		if a == nil {
			continue
		}
		if a.Subject.Person == oldID {
			a.Subject.Person = newID
			count++
		}
		if a.Subject.Event == oldID {
			a.Subject.Event = newID
			count++
		}
		if a.Subject.Relationship == oldID {
			a.Subject.Relationship = newID
			count++
		}
		if a.Subject.Place == oldID {
			a.Subject.Place = newID
			count++
		}
		// Assertion.Value can contain entity IDs for reference-type properties
		if a.Value == oldID {
			a.Value = newID
			count++
		}
		count += replaceInSlice(a.Sources, oldID, newID)
		count += replaceInSlice(a.Citations, oldID, newID)
		count += replaceInSlice(a.Media, oldID, newID)
		if a.Participant != nil {
			if a.Participant.Person == oldID {
				a.Participant.Person = newID
				count++
			}
			count += replaceInProperties(a.Participant.Properties, oldID, newID)
		}
	}

	// Media refs
	for _, m := range glx.Media {
		if m == nil {
			continue
		}
		if m.Source == oldID {
			m.Source = newID
			count++
		}
		count += replaceInProperties(m.Properties, oldID, newID)
	}

	return count
}

// replaceInSlice replaces all occurrences of oldID with newID in a string
// slice, returning the number of replacements made.
func replaceInSlice(s []string, oldID, newID string) int {
	count := 0
	for i := range s {
		if s[i] == oldID {
			s[i] = newID
			count++
		}
	}
	return count
}

// replaceInProperties scans a properties map for string values matching oldID
// and replaces them with newID. Handles simple strings, structured maps with
// "value" keys, and temporal lists.
func replaceInProperties(props map[string]any, oldID, newID string) int {
	count := 0
	for key, val := range props {
		switch v := val.(type) {
		case string:
			if v == oldID {
				props[key] = newID
				count++
			}
		case map[string]any:
			if s, ok := v["value"].(string); ok && s == oldID {
				v["value"] = newID
				count++
			}
		case []any:
			for _, elem := range v {
				if m, ok := elem.(map[string]any); ok {
					if s, ok := m["value"].(string); ok && s == oldID {
						m["value"] = newID
						count++
					}
				}
			}
		}
	}
	return count
}
