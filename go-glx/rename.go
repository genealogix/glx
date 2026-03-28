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
// new ID already exists.
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

	return &RenameResult{
		EntityType:  entityType,
		RefsUpdated: refs + 1, // +1 for the map key itself
	}, nil
}

// findEntityType returns which entity map contains the given ID.
func findEntityType(glx *GLXFile, id string) (string, error) {
	if _, ok := glx.Persons[id]; ok {
		return "persons", nil
	}
	if _, ok := glx.Events[id]; ok {
		return "events", nil
	}
	if _, ok := glx.Relationships[id]; ok {
		return "relationships", nil
	}
	if _, ok := glx.Places[id]; ok {
		return "places", nil
	}
	if _, ok := glx.Sources[id]; ok {
		return "sources", nil
	}
	if _, ok := glx.Citations[id]; ok {
		return "citations", nil
	}
	if _, ok := glx.Repositories[id]; ok {
		return "repositories", nil
	}
	if _, ok := glx.Assertions[id]; ok {
		return "assertions", nil
	}
	if _, ok := glx.Media[id]; ok {
		return "media", nil
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
	case "persons":
		glx.Persons[newID] = glx.Persons[oldID]
		delete(glx.Persons, oldID)
	case "events":
		glx.Events[newID] = glx.Events[oldID]
		delete(glx.Events, oldID)
	case "relationships":
		glx.Relationships[newID] = glx.Relationships[oldID]
		delete(glx.Relationships, oldID)
	case "places":
		glx.Places[newID] = glx.Places[oldID]
		delete(glx.Places, oldID)
	case "sources":
		glx.Sources[newID] = glx.Sources[oldID]
		delete(glx.Sources, oldID)
	case "citations":
		glx.Citations[newID] = glx.Citations[oldID]
		delete(glx.Citations, oldID)
	case "repositories":
		glx.Repositories[newID] = glx.Repositories[oldID]
		delete(glx.Repositories, oldID)
	case "assertions":
		glx.Assertions[newID] = glx.Assertions[oldID]
		delete(glx.Assertions, oldID)
	case "media":
		glx.Media[newID] = glx.Media[oldID]
		delete(glx.Media, oldID)
	}
}

// updateAllRefs scans every entity in the archive and replaces oldID with
// newID in all reference fields. Returns the count of fields updated.
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
		}
		if ev.PlaceID == oldID {
			ev.PlaceID = newID
			count++
		}
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
		count += replaceInSlice(a.Sources, oldID, newID)
		count += replaceInSlice(a.Citations, oldID, newID)
		count += replaceInSlice(a.Media, oldID, newID)
		if a.Participant != nil && a.Participant.Person == oldID {
			a.Participant.Person = newID
			count++
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
